package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"
)

type OAuthFlow interface {
	Start(ctx context.Context, authURL string) error
	WaitForCallback(ctx context.Context) (code string, state string, err error)
}

type LoopbackFlow struct {
	port    int
	server  *http.Server
	codeCh  chan string
	stateCh chan string
	errCh   chan error
}

func NewLoopbackFlow() *LoopbackFlow {
	return &LoopbackFlow{
		codeCh:  make(chan string, 1),
		stateCh: make(chan string, 1),
		errCh:   make(chan error, 1),
	}
}

// RedirectURI returns the callback URL for this loopback flow.
// It starts the listener to reserve the port.
func (l *LoopbackFlow) RedirectURI() (string, error) {
	if l.port == 0 {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", err
		}
		l.port = listener.Addr().(*net.TCPAddr).Port

		mux := http.NewServeMux()
		mux.HandleFunc("/callback", l.handleCallback)

		l.server = &http.Server{
			Handler: mux,
		}

		go func() {
			if err := l.server.Serve(listener); err != nil && err != http.ErrServerClosed {
				select {
				case l.errCh <- err:
				default:
				}
			}
		}()
	}
	return fmt.Sprintf("http://127.0.0.1:%d/callback", l.port), nil
}

func (l *LoopbackFlow) Start(ctx context.Context, authURL string) error {
	// If server isn't started yet, start it.
	if l.port == 0 {
		_, err := l.RedirectURI()
		if err != nil {
			return err
		}
	}
	return openBrowser(ctx, authURL)
}

func (l *LoopbackFlow) WaitForCallback(ctx context.Context) (string, string, error) {
	defer func() {
		if l.server != nil {
			// shutdown gracefully
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = l.server.Shutdown(shutdownCtx)
		}
	}()

	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case err := <-l.errCh:
		return "", "", err
	case code := <-l.codeCh:
		state := <-l.stateCh
		return code, state, nil
	}
}

func (l *LoopbackFlow) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errStr := r.URL.Query().Get("error")

	if errStr != "" {
		errDesc := r.URL.Query().Get("error_description")
		select {
		case l.errCh <- fmt.Errorf("oauth error: %s - %s", errStr, errDesc):
		default:
		}
		l.writeCallbackResponse(w, http.StatusBadRequest, fmt.Sprintf("Authentication failed: %s", errStr))
		return
	}

	if code == "" || state == "" {
		l.writeCallbackResponse(w, http.StatusBadRequest, "Missing code or state parameter")
		return
	}

	select {
	case l.codeCh <- code:
		l.stateCh <- state
	default:
	}

	l.writeCallbackResponse(w, http.StatusOK, `<html><body><h1>Authentication successful!</h1><p>You may now close this window and return to noms.</p></body></html>`)
}

func (l *LoopbackFlow) writeCallbackResponse(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		select {
		case l.errCh <- fmt.Errorf("writing callback response: %w", err):
		default:
		}
	}
}

// overrideable for testing
var openBrowser = func(ctx context.Context, rawURL string) error {
	if err := validateBrowserURL(rawURL); err != nil {
		return err
	}

	launchCtx := ctx
	if launchCtx == nil {
		launchCtx = context.Background()
	}
	cmdCtx, cancel := context.WithTimeout(launchCtx, 10*time.Second)
	defer cancel()

	type browserCommand struct {
		name string
		args []string
	}

	var candidates []browserCommand
	switch runtime.GOOS {
	case "windows":
		candidates = []browserCommand{{name: "cmd", args: []string{"/c", "start", "", rawURL}}}
	case "darwin":
		candidates = []browserCommand{{name: "open", args: []string{rawURL}}}
	default:
		candidates = []browserCommand{
			{name: "xdg-open", args: []string{rawURL}},
			{name: "gio", args: []string{"open", rawURL}},
			{name: "sensible-browser", args: []string{rawURL}},
		}
	}

	var lastErr error
	for _, candidate := range candidates {
		if err := startAndCheckBrowserCommand(cmdCtx, candidate.name, candidate.args...); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no browser launch command candidates configured")
	}

	return fmt.Errorf("opening browser: %w", lastErr)

}

func startAndCheckBrowserCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	select {
	case err := <-waitCh:
		return err
	case <-time.After(1500 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func validateBrowserURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid browser URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid browser URL scheme: %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("invalid browser URL host")
	}
	return nil
}
