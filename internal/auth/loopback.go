package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
	return openBrowser(authURL)
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
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Authentication failed: %s", errStr)))
		return
	}

	if code == "" || state == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing code or state parameter"))
		return
	}

	select {
	case l.codeCh <- code:
		l.stateCh <- state
	default:
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<html><body><h1>Authentication successful!</h1><p>You may now close this window and return to noms.</p></body></html>`))
}

// overrideable for testing
var openBrowser = func(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
