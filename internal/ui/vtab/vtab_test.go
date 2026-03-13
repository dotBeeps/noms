package vtab

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	voresky "github.com/dotBeeps/noms/internal/api/voresky"
	"github.com/dotBeeps/noms/internal/ui/images"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsi(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

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

	if !strings.Contains(content, "[····]") {
		t.Errorf("Expected placeholder '[····]' for uncached avatar, got: %s", content)
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

func TestVoreskyAvatarContentInsideBorder(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: true, img: "VTAB_AVATAR"}

	m := NewVoreskyModel(nil, 80, 24, stub)
	updated, _ := m.Update(CharactersLoadedMsg{
		Characters:      []voresky.Character{makeTestCharacter("Foxy", "https://example.com/avatar.png")},
		MainCharacterID: "",
	})
	m = updated.(VoreskyModel)
	m.characters[0].Description = "inside border test"

	content := stripAnsi(m.View().Content)

	if !strings.Contains(content, "VTAB_AVATAR") {
		t.Fatalf("expected avatar marker in rendered output, got: %q", content)
	}
	if !strings.Contains(content, "▎") {
		t.Fatalf("expected border character ▎ in rendered output")
	}
}

func TestVoreskyNoAvatarFullWidth(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: true, img: "SHOULD_NOT_APPEAR"}

	m := NewVoreskyModel(nil, 80, 24, stub)
	desc := "This is a fairly long description that should remain visible without avatar gutter reduction."
	updated, _ := m.Update(CharactersLoadedMsg{
		Characters: []voresky.Character{{
			ID:          "char-foxy",
			Name:        "Foxy",
			Avatar:      "",
			Status:      "active",
			Description: desc,
		}},
		MainCharacterID: "",
	})
	m = updated.(VoreskyModel)

	content := stripAnsi(m.View().Content)
	if strings.Contains(content, "SHOULD_NOT_APPEAR") {
		t.Fatalf("did not expect avatar render when avatar URL is empty")
	}
	if !strings.Contains(content, "This is a fairly long description") {
		t.Fatalf("expected long description to render without avatar truncation artifact; got %q", content)
	}
}

func TestVoreskyDescriptionTruncatedToContentWidth(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: true, img: "AV6\nAV6\nAV6"}

	m := NewVoreskyModel(nil, 80, 24, stub)
	updated, _ := m.Update(CharactersLoadedMsg{
		Characters: []voresky.Character{{
			ID:          "char-lore",
			Name:        "Lore",
			Avatar:      "https://example.com/lore.png",
			Status:      "active",
			Description: strings.Repeat("x", 200),
		}},
		MainCharacterID: "",
	})
	m = updated.(VoreskyModel)

	content := stripAnsi(m.View().Content)

	found := false
	for _, line := range strings.Split(content, "\n") {
		idx := strings.Index(line, "xxxx")
		if idx == -1 {
			continue
		}
		found = true
		descPart := strings.TrimRight(line[idx:], " ")
		if got := len([]rune(descPart)); got > 71+3 {
			t.Fatalf("expected truncated description to fit content width (<=74 incl ellipsis), got=%d line=%q", got, line)
		}
		if !strings.Contains(descPart, "...") {
			t.Fatalf("expected truncated description to include ellipsis, got %q", descPart)
		}
		break
	}

	if !found {
		t.Fatal("expected truncated description line in output")
	}
}
