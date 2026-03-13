package vtab

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	voresky "github.com/dotBeeps/noms/internal/api/voresky"
	"github.com/dotBeeps/noms/internal/ui/images"
)

type stubImageRenderer struct {
	enabled bool
	cached  bool
	img     string
}

func (s *stubImageRenderer) Enabled() bool                                 { return s.enabled }
func (s *stubImageRenderer) IsCached(url string) bool                      { return s.cached }
func (s *stubImageRenderer) RenderImage(url string, cols, rows int) string { return s.img }
func (s *stubImageRenderer) FetchAvatar(url string) tea.Cmd                { return nil }

var _ images.ImageRenderer = (*stubImageRenderer)(nil)

func makeTestCharacter(name, avatar string) voresky.Character {
	return voresky.Character{
		ID:     "char-" + name,
		Name:   name,
		Avatar: avatar,
		Status: "active",
	}
}

func TestVoreskyAvatarRenderedWhenCached(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: true, img: "CHAR_AVATAR"}

	m := NewVoreskyModel(nil, 80, 24, stub)
	updated, _ := m.Update(CharactersLoadedMsg{
		Characters:      []voresky.Character{makeTestCharacter("Foxy", "https://example.com/avatar.png")},
		MainCharacterID: "",
	})
	m = updated.(VoreskyModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "CHAR_AVATAR") {
		t.Errorf("Expected cached avatar 'CHAR_AVATAR' in character view, got: %s", content)
	}
}

func TestVoreskyAvatarPlaceholderWhenUncached(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: false, img: ""}

	m := NewVoreskyModel(nil, 80, 24, stub)
	updated, _ := m.Update(CharactersLoadedMsg{
		Characters:      []voresky.Character{makeTestCharacter("Foxy", "https://example.com/avatar.png")},
		MainCharacterID: "",
	})
	m = updated.(VoreskyModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "[··]") {
		t.Errorf("Expected placeholder '[··]' for uncached avatar, got: %s", content)
	}
}

func TestVoreskyNoAvatarWhenEmptyURL(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: true, img: "SHOULD_NOT_APPEAR"}

	m := NewVoreskyModel(nil, 80, 24, stub)
	updated, _ := m.Update(CharactersLoadedMsg{
		Characters:      []voresky.Character{makeTestCharacter("Foxy", "")},
		MainCharacterID: "",
	})
	m = updated.(VoreskyModel)

	v := m.View()
	content := v.Content

	if strings.Contains(content, "SHOULD_NOT_APPEAR") {
		t.Error("Expected no avatar image when character Avatar URL is empty")
	}
	if !strings.Contains(content, "Foxy") {
		t.Error("Expected character name in view even without avatar")
	}
}

func TestVoreskyNilImageCache(t *testing.T) {
	t.Parallel()

	m := NewVoreskyModel(nil, 80, 24, nil)
	updated, _ := m.Update(CharactersLoadedMsg{
		Characters:      []voresky.Character{makeTestCharacter("Foxy", "https://example.com/avatar.png")},
		MainCharacterID: "",
	})
	m = updated.(VoreskyModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Foxy") {
		t.Errorf("Expected character name with nil imageCache, got: %s", content)
	}
}
