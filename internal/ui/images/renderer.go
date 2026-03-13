package images

import tea "charm.land/bubbletea/v2"

type ImageRenderer interface {
	Enabled() bool
	IsCached(url string) bool
	RenderImage(url string, cols, rows int) string
	FetchAvatar(url string) tea.Cmd
}
