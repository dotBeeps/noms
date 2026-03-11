package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type DPoPSigner struct {
	keyPath string
	privKey *ecdsa.PrivateKey

	mu     sync.RWMutex
	nonces map[string]string
}

// NewDPoPSigner creates a new DPoP signer. It loads the key from keyPath or generates
// a new one and saves it there. If keyPath is empty, it only keeps the key in memory.
func NewDPoPSigner(keyPath string) (*DPoPSigner, error) {
	s := &DPoPSigner{
		keyPath: keyPath,
		nonces:  make(map[string]string),
	}

	if err := s.loadOrGenerateKey(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *DPoPSigner) loadOrGenerateKey() error {
	if s.keyPath != "" {
		data, err := os.ReadFile(s.keyPath)
		if err == nil {
			block, _ := pem.Decode(data)
			if block != nil && block.Type == "EC PRIVATE KEY" {
				parsed, err := x509.ParseECPrivateKey(block.Bytes)
				if err == nil {
					s.privKey = parsed
					return nil
				}
			}
		}
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate DPoP key: %w", err)
	}
	s.privKey = privKey

	if s.keyPath != "" {
		bytes, err := x509.MarshalECPrivateKey(privKey)
		if err != nil {
			return err
		}
		block := &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: bytes,
		}
		if err := os.WriteFile(s.keyPath, pem.EncodeToMemory(block), 0600); err != nil {
			return fmt.Errorf("persisting DPoP key to %s: %w", s.keyPath, err)
		}
	}

	return nil
}

func (s *DPoPSigner) GetPublicJWK() map[string]interface{} {
	pub := s.privKey.PublicKey
	return map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(pub.X.Bytes()),
		"y":   base64.RawURLEncoding.EncodeToString(pub.Y.Bytes()),
	}
}

func (s *DPoPSigner) UpdateNonce(server, nonce string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nonces[server] = nonce
}

func (s *DPoPSigner) getNonce(server string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nonces[server]
}

func (s *DPoPSigner) Sign(method, reqUrl, accessToken string) (string, error) {
	claims := jwt.MapClaims{
		"jti": uuid.New().String(),
		"htm": method,
		"htu": reqUrl,
		"iat": time.Now().Unix(),
	}

	if accessToken != "" {
		hash := sha256.Sum256([]byte(accessToken))
		claims["ath"] = base64.RawURLEncoding.EncodeToString(hash[:])
	}

	server := extractHost(reqUrl)
	nonce := s.getNonce(server)
	if nonce != "" {
		claims["nonce"] = nonce
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	token.Header["typ"] = "dpop+jwt"
	token.Header["jwk"] = s.GetPublicJWK()

	return token.SignedString(s.privKey)
}

func extractHost(fullUrl string) string {
	parsed, err := url.Parse(fullUrl)
	if err == nil && parsed.Host != "" {
		return parsed.Host
	}
	return fullUrl
}
