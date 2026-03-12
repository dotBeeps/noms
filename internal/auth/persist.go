package auth

import (
	"encoding/json"
	"fmt"
	"time"
)

// SessionStore is the interface for persisting session credentials.
// Defined here to avoid a circular import with config; config.TokenStore
// satisfies this interface via Go structural typing.
type SessionStore interface {
	Store(did string, data []byte) error
	Retrieve(did string) ([]byte, error)
}

// storedSession is the JSON-serializable form of a session persisted to disk.
type storedSession struct {
	DID           string    `json:"did"`
	Handle        string    `json:"handle"`
	PDS           string    `json:"pds"`
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	ExpiresAt     time.Time `json:"expires_at"`
	TokenType     string    `json:"token_type"`
	Sub           string    `json:"sub"`
	TokenEndpoint string    `json:"token_endpoint"`
	ClientID      string    `json:"client_id"`
}

// SaveSession persists the session's tokens and metadata to the store,
// keyed by the session's DID.
func SaveSession(store SessionStore, session *Session) error {
	s := storedSession{
		DID:    session.DID,
		Handle: session.Handle,
		PDS:    session.PDS,
	}

	if session.Tokens != nil {
		s.AccessToken = session.Tokens.AccessToken
		s.RefreshToken = session.Tokens.RefreshToken
		s.ExpiresAt = session.Tokens.ExpiresAt
		s.TokenType = session.Tokens.TokenType
		s.Sub = session.Tokens.Sub
	}

	if session.TokenManager != nil {
		s.TokenEndpoint = session.TokenManager.TokenEndpoint
		s.ClientID = session.TokenManager.ClientID
	}

	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	return store.Store(session.DID, data)
}

// RestoreSession loads a session from the store and rebuilds a fully
// functional Session with a working TokenManager and DPoP signer.
func RestoreSession(store SessionStore, did, dpopKeyPath string) (*Session, error) {
	data, err := store.Retrieve(did)
	if err != nil {
		return nil, fmt.Errorf("retrieving stored session: %w", err)
	}

	var s storedSession
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshaling session: %w", err)
	}

	if s.RefreshToken == "" {
		return nil, fmt.Errorf("stored session has no refresh token")
	}

	dpop, err := NewDPoPSigner(dpopKeyPath)
	if err != nil {
		return nil, fmt.Errorf("loading DPoP key: %w", err)
	}

	tokens := &TokenSet{
		AccessToken:  s.AccessToken,
		RefreshToken: s.RefreshToken,
		ExpiresAt:    s.ExpiresAt,
		TokenType:    s.TokenType,
		Sub:          s.Sub,
	}

	session := NewSession(s.DID, s.Handle, s.PDS, tokens, dpop)

	if s.TokenEndpoint != "" && s.ClientID != "" {
		tm := NewTokenManager(s.TokenEndpoint, s.ClientID, tokens, dpop)
		session.TokenManager = tm
	}

	return session, nil
}
