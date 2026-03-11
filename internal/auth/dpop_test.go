package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"path/filepath"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestDPoPJWTStructure(t *testing.T) {
	t.Parallel()
	s, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	tokenStr, err := s.Sign("GET", "https://example.com/api", "")
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if token.Header["typ"] != "dpop+jwt" {
		t.Errorf("Expected typ dpop+jwt, got %v", token.Header["typ"])
	}
	if token.Header["alg"] != "ES256" {
		t.Errorf("Expected alg ES256, got %v", token.Header["alg"])
	}

	jwk, ok := token.Header["jwk"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected jwk to be map, got %T", token.Header["jwk"])
	}
	if jwk["kty"] != "EC" || jwk["crv"] != "P-256" {
		t.Errorf("Invalid jwk: %v", jwk)
	}
}

func TestDPoPJWTFields(t *testing.T) {
	t.Parallel()
	s, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	accessToken := "my-access-token"
	tokenStr, err := s.Sign("POST", "https://example.com/api", accessToken)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims := token.Claims.(jwt.MapClaims)

	if claims["htm"] != "POST" {
		t.Errorf("Expected htm POST, got %v", claims["htm"])
	}
	if claims["htu"] != "https://example.com/api" {
		t.Errorf("Expected htu, got %v", claims["htu"])
	}
	if _, ok := claims["jti"].(string); !ok {
		t.Errorf("Expected jti to be string")
	}
	if _, ok := claims["iat"].(float64); !ok {
		t.Errorf("Expected iat to be number")
	}

	hash := sha256.Sum256([]byte(accessToken))
	expectedAth := base64.RawURLEncoding.EncodeToString(hash[:])
	if claims["ath"] != expectedAth {
		t.Errorf("Expected ath %s, got %v", expectedAth, claims["ath"])
	}
}

func TestDPoPNonceRotation(t *testing.T) {
	t.Parallel()
	s, err := NewDPoPSigner("")
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	s.UpdateNonce("example.com", "nonce-123")

	tokenStr, err := s.Sign("GET", "https://example.com/api", "")
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims := token.Claims.(jwt.MapClaims)
	if claims["nonce"] != "nonce-123" {
		t.Errorf("Expected nonce nonce-123, got %v", claims["nonce"])
	}

	// Update nonce
	s.UpdateNonce("example.com", "nonce-456")
	tokenStr2, _ := s.Sign("GET", "https://example.com/api", "")
	token2, _, _ := new(jwt.Parser).ParseUnverified(tokenStr2, jwt.MapClaims{})
	claims2 := token2.Claims.(jwt.MapClaims)

	if claims2["nonce"] != "nonce-456" {
		t.Errorf("Expected nonce nonce-456, got %v", claims2["nonce"])
	}
}

func TestDPoPKeyPersistence(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "dpop.key")

	s1, err := NewDPoPSigner(keyPath)
	if err != nil {
		t.Fatalf("Failed to create signer 1: %v", err)
	}

	s2, err := NewDPoPSigner(keyPath)
	if err != nil {
		t.Fatalf("Failed to create signer 2: %v", err)
	}

	jwk1 := s1.GetPublicJWK()
	jwk2 := s2.GetPublicJWK()

	if jwk1["x"] != jwk2["x"] || jwk1["y"] != jwk2["y"] {
		t.Errorf("Keys did not persist correctly, x or y mismatch")
	}
}
