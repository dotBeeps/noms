// Package vnotifications uses whitebox tests so that unexported functions
// (formatNotification, renderNotification) can be exercised directly.
package vnotifications

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	voresky "github.com/dotBeeps/noms/internal/api/voresky"
)

var ansiEscapeRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsi(s string) string {
	return ansiEscapeRe.ReplaceAllString(s, "")
}

// ─── stub ImageRenderer ───────────────────────────────────────────────────────

type stubImageRenderer struct {
	cached bool
	img    string
}

func (s *stubImageRenderer) Enabled() bool                                 { return true }
func (s *stubImageRenderer) IsCached(url string) bool                      { return s.cached }
func (s *stubImageRenderer) RenderImage(url string, cols, rows int) string { return s.img }
func (s *stubImageRenderer) FetchAvatar(url string) tea.Cmd                { return nil }

// ─── helpers ─────────────────────────────────────────────────────────────────

func makeNotif(
	typ voresky.NotificationType,
	payload json.RawMessage,
	src, tgt *voresky.NotificationCharacterInfo,
) voresky.Notification {
	return voresky.Notification{
		ID:              "test-id",
		Type:            typ,
		Payload:         payload,
		CreatedAt:       time.Time{},
		SourceCharacter: src,
		TargetCharacter: tgt,
	}
}

func makeChar(name, avatarURL string) *voresky.NotificationCharacterInfo {
	return &voresky.NotificationCharacterInfo{
		ID:   "char-" + name,
		Name: name,
		Avatar: voresky.CharacterAvatar{
			URL: avatarURL,
		},
	}
}

// ─── formatNotification tests ────────────────────────────────────────────────

func TestFormatNotification_Nil(t *testing.T) {
	got := formatNotification(nil)
	if got != "Notification" {
		t.Errorf("nil notification: want %q, got %q", "Notification", got)
	}
}

func TestFormatNotification_Poke(t *testing.T) {
	notif := makeNotif(
		voresky.NotifPoke,
		json.RawMessage(`{"universe":"Verdant"}`),
		makeChar("Sable", ""),
		makeChar("Pip", ""),
	)
	got := formatNotification(&notif)
	if !strings.Contains(got, "poked") {
		t.Errorf("POKE: want 'poked' in output, got %q", got)
	}
	if !strings.Contains(got, "Sable") {
		t.Errorf("POKE: want source name 'Sable' in output, got %q", got)
	}
	if !strings.Contains(got, "Pip") {
		t.Errorf("POKE: want target name 'Pip' in output, got %q", got)
	}
	t.Logf("POKE format: %q", got)
}

func TestFormatNotification_InteractionProposal(t *testing.T) {
	payload := json.RawMessage(`{
		"proposalId":"p1",
		"predatorCharacterName":"Rex",
		"preyCharacterName":"Pip",
		"pathName":"Forest Hunt",
		"estimatedDurationSeconds":3600,
		"initiatedBy":"predator"
	}`)
	notif := makeNotif(
		voresky.NotifInteractionProposal,
		payload,
		makeChar("Rex", ""),
		makeChar("Pip", ""),
	)
	got := formatNotification(&notif)
	if !strings.Contains(got, "Forest Hunt") {
		t.Errorf("INTERACTION_PROPOSAL: want path 'Forest Hunt' in output, got %q", got)
	}
	if !strings.Contains(got, "Rex") {
		t.Errorf("INTERACTION_PROPOSAL: want predator 'Rex' in output, got %q", got)
	}
	t.Logf("INTERACTION_PROPOSAL format: %q", got)
}

func TestFormatNotification_InteractionNodeChanged(t *testing.T) {
	payload := json.RawMessage(`{"interactionId":"i1","newNodeVerbPast":"swallowed","universe":"Verdant"}`)
	notif := makeNotif(
		voresky.NotifInteractionNodeChanged,
		payload,
		makeChar("Rex", ""),
		makeChar("Pip", ""),
	)
	got := formatNotification(&notif)
	if !strings.Contains(got, "swallowed") {
		t.Errorf("INTERACTION_NODE_CHANGED: want 'swallowed' in output, got %q", got)
	}
	t.Logf("INTERACTION_NODE_CHANGED format: %q", got)
}

func TestFormatNotification_InteractionCompleted_VerbPast(t *testing.T) {
	payload := json.RawMessage(`{
		"interactionId":"i1",
		"predatorCharacterName":"Rex",
		"preyCharacterName":"Pip",
		"pathName":"Deep Forest",
		"hasPointOfNoReturn":true,
		"finalNodeName":"The End",
		"verbPast":"devoured"
	}`)
	notif := makeNotif(
		voresky.NotifInteractionCompleted,
		payload,
		makeChar("Rex", ""),
		makeChar("Pip", ""),
	)
	got := formatNotification(&notif)
	if !strings.Contains(got, "devoured") {
		t.Errorf("INTERACTION_COMPLETED: want 'devoured' in output, got %q", got)
	}
	if !strings.Contains(got, "Pip") {
		t.Errorf("INTERACTION_COMPLETED: want prey 'Pip' in output, got %q", got)
	}
	t.Logf("INTERACTION_COMPLETED (verbPast) format: %q", got)
}

func TestFormatNotification_NilPayload(t *testing.T) {
	// nil Payload bytes — ParsePayload returns (nil, nil), formatNotification
	// falls back to character names from the Notification itself.
	notif := makeNotif(
		voresky.NotifInteractionProposal,
		nil,
		makeChar("Alpha", ""),
		makeChar("Beta", ""),
	)
	got := formatNotification(&notif)
	if got == "" {
		t.Error("nil payload: expected non-empty rendered string")
	}
	t.Logf("nil payload fallback: %q", got)
}

// ─── renderNotification tests ────────────────────────────────────────────────

// TestRenderNotification_DualAvatars verifies that when both source and target
// characters have cached avatars, both avatar images appear in the rendered row.
func TestRenderNotification_DualAvatars(t *testing.T) {
	src := makeChar("Sable", "https://example.com/sable.jpg")
	tgt := makeChar("Pip", "https://example.com/pip.jpg")
	notif := makeNotif(voresky.NotifPoke, json.RawMessage(`{}`), src, tgt)

	stub := &stubImageRenderer{cached: true, img: "AVATAR"}
	m := VNotificationsModel{
		notifications: []voresky.Notification{notif},
		imageCache:    stub,
		width:         80,
	}

	result := m.renderNotification(0, false)
	count := strings.Count(result, "AVATAR")
	if count < 2 {
		t.Errorf("dual avatars: want AVATAR marker at least 2 times in output, got %d\noutput: %q",
			count, result)
	}
	t.Logf("dual avatar render: found AVATAR x%d", count)
}

// TestRenderNotification_SingleAvatar verifies rendering with only a source avatar
// (no target character) does not crash and includes the avatar.
func TestRenderNotification_SingleAvatar(t *testing.T) {
	src := makeChar("Sable", "https://example.com/sable.jpg")
	notif := makeNotif(voresky.NotifPoke, json.RawMessage(`{}`), src, nil)

	stub := &stubImageRenderer{cached: true, img: "AVATAR"}
	m := VNotificationsModel{
		notifications: []voresky.Notification{notif},
		imageCache:    stub,
		width:         80,
	}

	result := m.renderNotification(0, false)
	if !strings.Contains(result, "AVATAR") {
		t.Errorf("single avatar: expected AVATAR in rendered output\noutput: %q", result)
	}
	t.Logf("single avatar render OK")
}

// TestRenderNotification_NoAvatars verifies rendering without an imageCache does
// not crash and still produces non-empty output.
func TestRenderNotification_NoAvatars(t *testing.T) {
	src := makeChar("Sable", "")
	notif := makeNotif(voresky.NotifPoke, json.RawMessage(`{}`), src, nil)

	m := VNotificationsModel{
		notifications: []voresky.Notification{notif},
		imageCache:    nil,
		width:         80,
	}

	result := m.renderNotification(0, false)
	if result == "" {
		t.Error("no avatars: expected non-empty render")
	}
	t.Logf("no avatar render OK: len=%d", len(result))
}

// TestRenderNotification_UncachedAvatars verifies that when avatar URLs are set
// but not yet cached, placeholder text is rendered (not the actual image).
func TestRenderNotification_UncachedAvatars(t *testing.T) {
	src := makeChar("Sable", "https://example.com/sable.jpg")
	tgt := makeChar("Pip", "https://example.com/pip.jpg")
	notif := makeNotif(voresky.NotifPoke, json.RawMessage(`{}`), src, tgt)

	// cached=false: IsCached returns false → falls back to RenderPlaceholder
	stub := &stubImageRenderer{cached: false, img: "SHOULD_NOT_APPEAR"}
	m := VNotificationsModel{
		notifications: []voresky.Notification{notif},
		imageCache:    stub,
		width:         80,
	}

	result := m.renderNotification(0, false)
	if strings.Contains(result, "SHOULD_NOT_APPEAR") {
		t.Error("uncached avatars: RenderImage should not be called when not cached")
	}
	if result == "" {
		t.Error("uncached avatars: expected non-empty render")
	}
	t.Logf("uncached avatar render OK (placeholder used)")
}

func TestDualAvatarGutterWidth13(t *testing.T) {
	t.Parallel()
	src := makeChar("Sable", "https://example.com/sable.jpg")
	tgt := makeChar("Pip", "https://example.com/pip.jpg")
	notif := makeNotif(voresky.NotifPoke, json.RawMessage(`{"universe":"TestU"}`), src, tgt)
	notif.Universe = "A-very-long-universe-name-to-force-extra-wrap"

	stub := &stubImageRenderer{cached: true, img: "AAAAAA\nBBBBBB\nCCCCCC"}
	m := VNotificationsModel{
		notifications: []voresky.Notification{notif},
		imageCache:    stub,
		width:         30,
	}

	out := stripAnsi(m.renderNotification(0, false))
	lines := strings.Split(out, "\n")

	foundIndented := false
	for _, line := range lines {
		if !strings.Contains(line, "Jan") {
			continue
		}
		line = strings.TrimPrefix(line, "▎  ")
		idx := strings.Index(line, "Jan")
		if idx == -1 {
			continue
		}
		if strings.HasSuffix(line[:idx], strings.Repeat(" ", 16)) {
			foundIndented = true
			break
		}
	}
	if !foundIndented {
		t.Fatalf("expected >=13 gutter spaces before time line for dual avatars; output: %q", out)
	}
}

func TestSingleAvatarGutterWidth6(t *testing.T) {
	t.Parallel()
	src := makeChar("Sable", "https://example.com/sable.jpg")
	notif := makeNotif(voresky.NotifPoke, json.RawMessage(`{"universe":"TestU"}`), src, nil)
	notif.Universe = "Long-universe-wrap"

	stub := &stubImageRenderer{cached: true, img: "AAAAAA\nBBBBBB\nCCCCCC"}
	m := VNotificationsModel{
		notifications: []voresky.Notification{notif},
		imageCache:    stub,
		width:         24,
	}

	out := stripAnsi(m.renderNotification(0, false))
	lines := strings.Split(out, "\n")

	foundIndented := false
	for _, line := range lines {
		if !strings.Contains(line, "Jan") {
			continue
		}
		line = strings.TrimPrefix(line, "▎  ")
		idx := strings.Index(line, "Jan")
		if idx == -1 {
			continue
		}
		if strings.HasSuffix(line[:idx], strings.Repeat(" ", 9)) {
			foundIndented = true
			break
		}
	}
	if !foundIndented {
		t.Fatalf("expected >=6 gutter spaces before time line for single avatar; output: %q", out)
	}
}

func TestNarrowTerminalFallsBackToSingleAvatar(t *testing.T) {
	t.Parallel()
	src := makeChar("Sable", "https://example.com/sable.jpg")
	tgt := makeChar("Pip", "https://example.com/pip.jpg")
	notif := makeNotif(voresky.NotifPoke, json.RawMessage(`{"universe":"TestU"}`), src, tgt)

	stub := &stubImageRenderer{cached: true, img: "AAAAAA\nBBBBBB\nCCCCCC"}
	m := VNotificationsModel{
		notifications: []voresky.Notification{notif},
		imageCache:    stub,
		width:         20,
	}

	out := m.renderNotification(0, false)
	if strings.Count(out, "AAAAAA") > 1 {
		t.Fatalf("expected fallback to single avatar at narrow width, got dual avatar block: %q", out)
	}
	if strings.Contains(out, strings.Repeat(" ", 14)) {
		t.Fatalf("expected no 13-width dual gutter at narrow width; output: %q", out)
	}
}

func TestVnotificationAvatarPresentInOutput(t *testing.T) {
	t.Parallel()
	src := makeChar("Sable", "https://example.com/sable.jpg")
	notif := makeNotif(voresky.NotifPoke, json.RawMessage(`{}`), src, nil)

	stub := &stubImageRenderer{cached: true, img: "VN_AVATAR"}
	m := VNotificationsModel{
		notifications: []voresky.Notification{notif},
		imageCache:    stub,
		width:         80,
	}

	out := m.renderNotification(0, false)
	if !strings.Contains(out, "VN_AVATAR") {
		t.Fatalf("expected avatar marker in output, got: %q", out)
	}
}

func TestDualAvatarRenderContainsBothWhenWide(t *testing.T) {
	t.Parallel()
	src := makeChar("Sable", "https://example.com/sable.jpg")
	tgt := makeChar("Pip", "https://example.com/pip.jpg")
	notif := makeNotif(voresky.NotifPoke, json.RawMessage(`{}`), src, tgt)

	stub := &stubImageRenderer{cached: true, img: "DUALAV"}
	m := VNotificationsModel{
		notifications: []voresky.Notification{notif},
		imageCache:    stub,
		width:         80,
	}

	out := m.renderNotification(0, false)
	if strings.Count(out, "DUALAV") < 2 {
		t.Fatalf("expected two avatars to render on wide terminal; output: %q", out)
	}
}
