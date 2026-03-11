package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Session struct {
	DID          string
	Handle       string
	PDS          string
	Tokens       *TokenSet
	DPoP         *DPoPSigner
	TokenManager *TokenManager
}

func NewSession(did, handle, pds string, tokens *TokenSet, dpop *DPoPSigner) *Session {
	return &Session{
		DID:    did,
		Handle: handle,
		PDS:    pds,
		Tokens: tokens,
		DPoP:   dpop,
	}
}

func (s *Session) AuthenticatedHTTPClient() *http.Client {
	base := http.DefaultTransport
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &dpopTransport{
			base:    base,
			session: s,
		},
	}
}

type dpopTransport struct {
	base    http.RoundTripper
	session *Session
}

func (t *dpopTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	accessToken := t.currentAccessToken()

	dpopJWT, err := t.session.DPoP.Sign(req.Method, stripQueryFragment(req.URL.String()), accessToken)
	if err != nil {
		return nil, fmt.Errorf("DPoP signing failed: %w", err)
	}

	reqCopy := req.Clone(req.Context())
	reqCopy.Header.Set("Authorization", "DPoP "+accessToken)
	reqCopy.Header.Set("DPoP", dpopJWT)

	resp, err := t.base.RoundTrip(reqCopy)
	if err != nil {
		return nil, err
	}

	if nonce := resp.Header.Get("DPoP-Nonce"); nonce != "" {
		host := extractHost(req.URL.String())
		t.session.DPoP.UpdateNonce(host, nonce)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	authHdr := resp.Header.Get("WWW-Authenticate")
	if authHdr == "" {
		return resp, nil
	}

	if strings.Contains(authHdr, `error="use_dpop_nonce"`) {
		dpopNonce := resp.Header.Get("DPoP-Nonce")
		if dpopNonce == "" {
			return resp, nil
		}

		host := extractHost(req.URL.String())
		t.session.DPoP.UpdateNonce(host, dpopNonce)

		retryReq, err := cloneRequest(req)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("cloning request for DPoP nonce retry: %w", err)
		}
		resp.Body.Close()

		dpopJWT2, err := t.session.DPoP.Sign(retryReq.Method, stripQueryFragment(retryReq.URL.String()), accessToken)
		if err != nil {
			return nil, fmt.Errorf("DPoP signing failed on retry: %w", err)
		}
		retryReq.Header.Set("Authorization", "DPoP "+accessToken)
		retryReq.Header.Set("DPoP", dpopJWT2)

		return t.base.RoundTrip(retryReq)
	}

	if strings.Contains(authHdr, `error="invalid_token"`) && t.session.TokenManager != nil {
		newTokens, err := t.session.TokenManager.Refresh(req.Context())
		if err != nil {
			return resp, nil
		}

		retryReq, err := cloneRequest(req)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("cloning request for token refresh retry: %w", err)
		}
		resp.Body.Close()

		dpopJWT2, err := t.session.DPoP.Sign(retryReq.Method, stripQueryFragment(retryReq.URL.String()), newTokens.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("DPoP signing failed on retry: %w", err)
		}
		retryReq.Header.Set("Authorization", "DPoP "+newTokens.AccessToken)
		retryReq.Header.Set("DPoP", dpopJWT2)

		return t.base.RoundTrip(retryReq)
	}

	return resp, nil
}

func (t *dpopTransport) currentAccessToken() string {
	if t.session.TokenManager != nil {
		return t.session.TokenManager.AccessToken()
	}
	if t.session.Tokens != nil {
		return t.session.Tokens.AccessToken
	}
	return ""
}

func cloneRequest(req *http.Request) (*http.Request, error) {
	retry := req.Clone(req.Context())
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("failed to get request body for retry: %w", err)
		}
		retry.Body = body
	}
	return retry, nil
}

func stripQueryFragment(rawURL string) string {
	if idx := strings.IndexByte(rawURL, '?'); idx != -1 {
		return rawURL[:idx]
	}
	if idx := strings.IndexByte(rawURL, '#'); idx != -1 {
		return rawURL[:idx]
	}
	return rawURL
}
