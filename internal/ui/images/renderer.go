package images

import tea "charm.land/bubbletea/v2"

type ImageRenderer interface {
	Enabled() bool
	IsCached(url string) bool
	RenderImage(url string, cols, rows int) string
	FetchAvatar(url string) tea.Cmd
	InvalidateTransmissions()
	Dimensions(url string) (int, int, bool)
}

// LazyRenderer wraps an ImageRenderer to avoid Kitty image transmissions for
// off-screen items. When NearVisible is true, delegates normally. When false,
// uses RenderImageNoTransmit on *Cache (skips new transmissions), falling back
// to normal rendering for non-Cache implementations (test stubs).
type LazyRenderer struct {
	Inner       ImageRenderer
	NearVisible bool
}

func (lr *LazyRenderer) Enabled() bool {
	return lr.Inner != nil && lr.Inner.Enabled()
}

func (lr *LazyRenderer) IsCached(url string) bool {
	return lr.Inner != nil && lr.Inner.IsCached(url)
}

func (lr *LazyRenderer) FetchAvatar(url string) tea.Cmd {
	if lr.Inner == nil {
		return nil
	}
	return lr.Inner.FetchAvatar(url)
}

func (lr *LazyRenderer) InvalidateTransmissions() {
	if lr.Inner != nil {
		lr.Inner.InvalidateTransmissions()
	}
}

func (lr *LazyRenderer) Dimensions(url string) (int, int, bool) {
	if lr.Inner == nil {
		return 0, 0, false
	}
	return lr.Inner.Dimensions(url)
}

func (lr *LazyRenderer) RenderImage(url string, cols, rows int) string {
	if lr.Inner == nil {
		return ""
	}
	if lr.NearVisible {
		return lr.Inner.RenderImage(url, cols, rows)
	}
	if c, ok := lr.Inner.(*Cache); ok {
		return c.RenderImageNoTransmit(url, cols, rows)
	}
	return lr.Inner.RenderImage(url, cols, rows)
}
