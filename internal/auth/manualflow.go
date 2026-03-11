package auth

import "context"

// ManualFlow implements OAuthFlow by capturing the auth URL for display in the TUI
// instead of opening a browser. Uses LoopbackFlow's callback server for receiving
// the OAuth redirect.
type ManualFlow struct {
	*LoopbackFlow
	authURL chan string
}

func NewManualFlow() *ManualFlow {
	return &ManualFlow{
		LoopbackFlow: NewLoopbackFlow(),
		authURL:      make(chan string, 1),
	}
}

// Start captures the auth URL instead of opening a browser.
func (f *ManualFlow) Start(ctx context.Context, authURL string) error {
	// Ensure the callback server is running (delegates to LoopbackFlow).
	if f.LoopbackFlow.port == 0 {
		_, err := f.LoopbackFlow.RedirectURI()
		if err != nil {
			return err
		}
	}
	f.authURL <- authURL
	return nil
}

// AuthURL blocks until the auth URL is ready, then returns it.
func (f *ManualFlow) AuthURL() string {
	return <-f.authURL
}
