# Authentication

noms uses the full atproto OAuth 2.0 flow with DPoP (Demonstrating Proof of Possession) and PKCE. No app passwords.

---

## OAuth Flow (step by step)

### 1. Resolve handle → PDS

The user's handle (e.g. `you.bsky.social`) is resolved via `identity.DefaultDirectory().LookupHandle()` to find their Personal Data Server (PDS) URL. This uses the atproto identity resolution protocol — DNS TXT records and/or DID documents.

### 2. Discover authorization server

noms fetches `{PDS}/.well-known/oauth-protected-resource` and extracts the authorization server URL from `authorization_servers[0]`.

### 3. Fetch authorization server metadata

noms fetches `{auth_server}/.well-known/oauth-authorization-server` to get endpoints: `authorization_endpoint`, `token_endpoint`, `pushed_authorization_request_endpoint`, and supported algorithms.

### 4. Generate PKCE + state

- PKCE verifier: 32 random bytes, base64url-encoded
- PKCE challenge: `BASE64URL(SHA256(verifier))` (S256 method)
- State: 16 random bytes, base64url-encoded (CSRF protection)

### 5. Send PAR (Pushed Authorization Request)

If the server provides `pushed_authorization_request_endpoint`, noms sends the authorization parameters (client_id, scopes, redirect_uri, PKCE challenge, state, login_hint) as a DPoP-signed POST. The server returns a `request_uri`. On `use_dpop_nonce` error, noms retries once with the server-issued nonce.

If PAR is not available, the parameters are embedded directly in the authorization URL.

### 6. Open browser

noms calls `open`/`xdg-open` with the authorization URL. The URL either references the PAR `request_uri` or contains all parameters inline.

### 7. Loopback callback

A local HTTP server binds to a random port on `127.0.0.1`. The redirect URI points to it. When the browser redirects after user consent, the server captures `code` and `state` from the query parameters. State is verified against the value from step 4.

### 8. Token exchange

noms POSTs to the `token_endpoint` with `grant_type=authorization_code`, the code, PKCE verifier, and a DPoP proof. On `use_dpop_nonce` error, retries once. The response contains `access_token`, `refresh_token`, `expires_in`, and `sub` (the DID).

---

## DPoP Signer

`internal/auth/dpop.go` — `DPoPSigner` wraps an EC P-256 private key.

- Key is loaded from disk (PEM format) or generated fresh and saved to `{data_dir}/dpop.pem`
- Each request gets a unique proof JWT signed with `ES256` (`typ: dpop+jwt`)
- JWT claims: `jti` (UUID), `htm` (HTTP method), `htu` (URL without query), `iat` (timestamp), `ath` (SHA-256 of access token, when present), `nonce` (if server issued one)
- Nonces are tracked per-host in a map; updated from `DPoP-Nonce` response headers

The `dpopTransport` wraps Go's `http.RoundTripper` and automatically handles:

- Injecting `Authorization: DPoP {token}` and `DPoP: {proof}` headers
- Nonce rotation on `use_dpop_nonce` (one retry per request)
- Token refresh on `invalid_token` (calls `TokenManager.Refresh`, then retries)

---

## Token Refresh

`internal/auth/token.go` — `TokenManager` holds the current `TokenSet` and handles proactive expiry.

- `IsExpired()` returns true if the token expires within 1 minute (1-minute buffer)
- `Refresh()` uses `grant_type=refresh_token` with a DPoP-signed POST
- Thread-safe: a mutex prevents double-refreshes if multiple goroutines call `Refresh()` concurrently
- On success, fires `OnTokenRefresh` callback (used to persist the new tokens)

---

## Token Storage

`internal/config/keyring.go` — `TokenStore` interface with two implementations:

### FileStore (default)

Tokens are stored at `~/.local/share/noms/tokens.enc` (AES-256-GCM encrypted, with atomic rename on save).

Key derivation:

- **v2 format** (current): 32-byte random salt prepended to ciphertext; key derived via `scrypt(keyMaterial, salt, N=32768, r=8, p=1)` where `keyMaterial` defaults to `SHA256("noms:" + home_dir)` or the `NOMS_TOKEN_KEY` env var
- **v1 format** (legacy): fixed salt + hardcoded passphrase (transparently decoded for existing files)

### MemoryStore (fallback)

Used if `FileStore` creation fails. Data does not persist across restarts.

---

## Session Persistence

`internal/auth/persist.go` — `SaveSession` and `RestoreSession` serialize/deserialize a `Session` (DID, handle, PDS URL, tokens, token endpoint, client ID) as JSON through the `SessionStore` interface.

On startup, if `default_account` is set in config, noms calls `RestoreSession` to rebuild the session without requiring a browser login.
