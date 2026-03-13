// Package vnotifications uses whitebox tests so that unexported functions
// (formatNotification, renderNotification) can be exercised directly.
package vnotifications

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	voresky "github.com/dotBeeps/noms/internal/api/voresky"
)

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
