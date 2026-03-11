package auth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

type PasteCodeFlow struct {
	In  io.Reader
	Out io.Writer
}

func NewPasteCodeFlow() *PasteCodeFlow {
	return &PasteCodeFlow{
		In:  os.Stdin,
		Out: os.Stdout,
	}
}

// RedirectURI returns the special urn:ietf:wg:oauth:2.0:oob for manual paste code.
// For atproto/Bluesky, it is usually "urn:ietf:wg:oauth:2.0:oob"
func (p *PasteCodeFlow) RedirectURI() (string, error) {
	return "urn:ietf:wg:oauth:2.0:oob", nil
}

func (p *PasteCodeFlow) Start(ctx context.Context, authURL string) error {
	_, err := fmt.Fprintf(p.Out, "Please visit the following URL to authenticate:\n\n%s\n\n", authURL)
	return err
}

func (p *PasteCodeFlow) WaitForCallback(ctx context.Context) (string, string, error) {
	_, err := fmt.Fprint(p.Out, "Enter the authorization code: ")
	if err != nil {
		return "", "", err
	}

	type result struct {
		code string
		err  error
	}

	ch := make(chan result, 1)

	go func() {
		reader := bufio.NewReader(p.In)
		text, err := reader.ReadString('\n')
		if err != nil {
			ch <- result{"", err}
			return
		}
		
		code := strings.TrimSpace(text)
		if code == "" {
			ch <- result{"", fmt.Errorf("authorization code cannot be empty")}
			return
		}

		ch <- result{code, nil}
	}()

	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case res := <-ch:
		// Paste code flow does not return a state via the user, 
		// the state is usually maintained by the client locally. 
		// We return empty state, so the caller knows it wasn't provided.
		return res.code, "", res.err
	}
}
