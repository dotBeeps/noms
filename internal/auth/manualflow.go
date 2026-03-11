package auth

import "context"

// ManualFlow implements OAuthFlow by capturing the auth URL for display in the TUI
// instead of opening a browser. Uses LoopbackFlow's callback server for receiving
// the OAuth redirect.
type ManualFlow struct {
	*LoopbackFlow
	result chan manualFlowResult
}

type manualFlowResult struct {
	URL string
	Err error
}

func NewManualFlow() *ManualFlow {
	return &ManualFlow{
		LoopbackFlow: NewLoopbackFlow(),
		result:       make(chan manualFlowResult, 1),
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
	f.result <- manualFlowResult{URL: authURL}
	return nil
}

// SignalError unblocks AuthURL when Authenticate fails before Start is called.
func (f *ManualFlow) SignalError(err error) {
	select {
	case f.result <- manualFlowResult{Err: err}:
	default:
	}
}

// AuthURL blocks until the auth URL is ready or an error occurs.
func (f *ManualFlow) AuthURL() (string, error) {
	r := <-f.result
	return r.URL, r.Err
}
