package compose

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsi(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// mockBlueskyClient implements bluesky.BlueskyClient for testing
type mockBlueskyClient struct {
	createPostCalled bool
	lastText         string
	lastFacets       []*bsky.RichtextFacet
	lastReply        *bsky.FeedPost_ReplyRef
	lastEmbed        *bsky.FeedPost_Embed
	createPostErr    error
	createPostURI    string
	createPostCID    string
}

func (m *mockBlueskyClient) GetTimeline(ctx context.Context, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	return nil, "", nil
}

func (m *mockBlueskyClient) GetPost(ctx context.Context, uri string) (*bsky.FeedDefs_PostView, error) {
	return nil, nil
}

func (m *mockBlueskyClient) GetPostThread(ctx context.Context, uri string, depth int) (*bsky.FeedGetPostThread_Output, error) {
	return nil, nil
}

func (m *mockBlueskyClient) GetProfile(ctx context.Context, actor string) (*bsky.ActorDefs_ProfileViewDetailed, error) {
	return nil, nil
}

func (m *mockBlueskyClient) GetAuthorFeed(ctx context.Context, actor string, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	return nil, "", nil
}

func (m *mockBlueskyClient) FollowActor(ctx context.Context, did string) error {
	return nil
}

func (m *mockBlueskyClient) UnfollowActor(ctx context.Context, did string) error {
	return nil
}

func (m *mockBlueskyClient) ListNotifications(ctx context.Context, cursor string, limit int) ([]*bsky.NotificationListNotifications_Notification, string, error) {
	return nil, "", nil
}

func (m *mockBlueskyClient) GetUnreadCount(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *mockBlueskyClient) MarkNotificationsRead(ctx context.Context, seenAt time.Time) error {
	return nil
}

func (m *mockBlueskyClient) CreatePost(ctx context.Context, text string, facets []*bsky.RichtextFacet, reply *bsky.FeedPost_ReplyRef, embed *bsky.FeedPost_Embed) (string, string, error) {
	m.createPostCalled = true
	m.lastText = text
	m.lastFacets = facets
	m.lastReply = reply
	m.lastEmbed = embed
	if m.createPostErr != nil {
		return "", "", m.createPostErr
	}
	return m.createPostURI, m.createPostCID, nil
}

func (m *mockBlueskyClient) DeletePost(ctx context.Context, uri string) error {
	return nil
}

func (m *mockBlueskyClient) Like(ctx context.Context, uri, cid string) (string, error) {
	return "", nil
}

func (m *mockBlueskyClient) Unlike(ctx context.Context, likeURI string) error {
	return nil
}

func (m *mockBlueskyClient) Repost(ctx context.Context, uri, cid string) (string, error) {
	return "", nil
}

func (m *mockBlueskyClient) UnRepost(ctx context.Context, repostURI string) error {
	return nil
}

func (m *mockBlueskyClient) SearchPosts(ctx context.Context, query string, cursor string, limit int) ([]*bsky.FeedDefs_PostView, string, error) {
	return nil, "", nil
}

func (m *mockBlueskyClient) SearchActors(ctx context.Context, query string, cursor string, limit int) ([]*bsky.ActorDefs_ProfileView, string, error) {
	return nil, "", nil
}

// Helper functions
func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }

func createTestPostView(uri, cid, handle, text string) *bsky.FeedDefs_PostView {
	return &bsky.FeedDefs_PostView{
		Uri:       uri,
		Cid:       cid,
		IndexedAt: time.Now().Format(time.RFC3339),
		Author: &bsky.ActorDefs_ProfileViewBasic{
			Did:         "did:plc:" + handle,
			Handle:      handle,
			DisplayName: strPtr("Test User"),
		},
		Record: &util.LexiconTypeDecoder{
			Val: &bsky.FeedPost{
				LexiconTypeID: "app.bsky.feed.post",
				Text:          text,
				CreatedAt:     time.Now().Format(time.RFC3339),
			},
		},
	}
}

// Test 1: TestComposeNewPost
func TestComposeNewPost(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "New Post") {
		t.Errorf("Expected header 'New Post', got %q", content)
	}
	if !strings.Contains(content, "Ctrl+Enter") {
		t.Errorf("Expected key hint for Ctrl+Enter, got %q", content)
	}
	if !strings.Contains(content, "Esc") {
		t.Errorf("Expected key hint for Esc, got %q", content)
	}
}

// Test 2: TestComposeReply
func TestComposeReply(t *testing.T) {
	client := &mockBlueskyClient{}
	parentPost := createTestPostView("at://did:plc:test/app.bsky.feed.post/123", "parentcid", "alice.bsky.social", "Original post content")

	m := NewComposeModel(client, ModeReply, parentPost, 80, 24)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Reply") {
		t.Errorf("Expected header 'Reply', got %q", content)
	}
	if !strings.Contains(content, "Replying to:") {
		t.Errorf("Expected 'Replying to:' label, got %q", content)
	}
	if !strings.Contains(content, "alice.bsky.social") {
		t.Errorf("Expected parent author handle, got %q", content)
	}
}

// Test 3: TestComposeQuote
func TestComposeQuote(t *testing.T) {
	client := &mockBlueskyClient{}
	quotedPost := createTestPostView("at://did:plc:test/app.bsky.feed.post/456", "quotecid", "bob.bsky.social", "Post to be quoted")

	m := NewComposeModel(client, ModeQuote, quotedPost, 80, 24)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Quote Post") {
		t.Errorf("Expected header 'Quote Post', got %q", content)
	}
	if !strings.Contains(content, "Quoting:") {
		t.Errorf("Expected 'Quoting:' label, got %q", content)
	}
	if !strings.Contains(content, "bob.bsky.social") {
		t.Errorf("Expected quoted author handle, got %q", content)
	}
}

// Test 4: TestCharCounter
func TestCharCounter(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	// Initial state - empty
	v := m.View()
	content := v.Content
	if !strings.Contains(content, "0/300") {
		t.Errorf("Expected char counter '0/300', got %q", content)
	}

	// Set text and check counter
	m.SetText("Hello world!")
	if m.CharCount() != 12 {
		t.Errorf("Expected CharCount 12, got %d", m.CharCount())
	}

	v = m.View()
	content = v.Content
	if !strings.Contains(content, "12/300") {
		t.Errorf("Expected char counter '12/300', got %q", content)
	}
}

// Test 5: TestCharCounterLimit
func TestCharCounterLimit(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	// Set text that exceeds limit
	longText := strings.Repeat("x", 350)
	m.SetText(longText)

	if m.CharCount() != 350 {
		t.Errorf("Expected CharCount 350, got %d", m.CharCount())
	}

	v := m.View()
	content := v.Content

	// Should show over limit indicator (350/300)
	if !strings.Contains(content, "350/300") {
		t.Errorf("Expected char counter '350/300', got %q", content)
	}
}

// Test 6: TestMentionAutoDetect
func TestMentionAutoDetect(t *testing.T) {
	client := &mockBlueskyClient{
		createPostURI: "at://did:plc:test/app.bsky.feed.post/new",
		createPostCID: "newcid",
	}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	// Set text with mention
	m.SetText("Hello @alice.bsky.social!")

	// Simulate submit
	_, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+enter"})
	if cmd == nil {
		t.Fatal("Expected command from ctrl+enter")
	}

	// Execute the command
	msg := cmd()
	if _, ok := msg.(postSuccessMsg); !ok && !isPostSuccessMsg(msg) {
		// It might be a batch command, execute again
		if batchMsg, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batchMsg {
				if c != nil {
					innerMsg := c()
					if successMsg, ok := innerMsg.(postSuccessMsg); ok {
						msg = successMsg
						break
					}
				}
			}
		}
	}

	// Check that CreatePost was called with facets
	if !client.createPostCalled {
		t.Error("Expected CreatePost to be called")
	}
	if len(client.lastFacets) == 0 {
		t.Error("Expected facets to be detected for @mention")
	}

	// Check facet content
	foundMention := false
	for _, facet := range client.lastFacets {
		if len(facet.Features) > 0 && facet.Features[0].RichtextFacet_Mention != nil {
			foundMention = true
			if !strings.Contains(client.lastText, "@alice.bsky.social") {
				t.Error("Expected mention text in post")
			}
		}
	}
	if !foundMention {
		t.Error("Expected mention facet to be detected")
	}
}

// Helper to check if msg is a postSuccessMsg
func isPostSuccessMsg(msg tea.Msg) bool {
	_, ok := msg.(postSuccessMsg)
	return ok
}

// Test 7: TestLinkAutoDetect
func TestLinkAutoDetect(t *testing.T) {
	client := &mockBlueskyClient{
		createPostURI: "at://did:plc:test/app.bsky.feed.post/new",
		createPostCID: "newcid",
	}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	// Set text with URL
	m.SetText("Check out https://example.com for more info")

	// Simulate submit
	_, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+enter"})
	if cmd == nil {
		t.Fatal("Expected command from ctrl+enter")
	}

	// Execute the command
	msg := cmd()
	_ = msg // We just need to trigger the create

	// Check that CreatePost was called with facets
	if !client.createPostCalled {
		t.Error("Expected CreatePost to be called")
	}
	if len(client.lastFacets) == 0 {
		t.Error("Expected facets to be detected for URL")
	}

	// Check facet content
	foundLink := false
	for _, facet := range client.lastFacets {
		if len(facet.Features) > 0 && facet.Features[0].RichtextFacet_Link != nil {
			foundLink = true
			if facet.Features[0].RichtextFacet_Link.Uri != "https://example.com" {
				t.Errorf("Expected link URI 'https://example.com', got %q", facet.Features[0].RichtextFacet_Link.Uri)
			}
		}
	}
	if !foundLink {
		t.Error("Expected link facet to be detected")
	}
}

// Test 8: TestSubmitPost
func TestSubmitPost(t *testing.T) {
	client := &mockBlueskyClient{
		createPostURI: "at://did:plc:test/app.bsky.feed.post/new",
		createPostCID: "newcid",
	}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	// Set text
	m.SetText("This is a test post")

	// Simulate Ctrl+Enter
	_, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+enter"})
	if cmd == nil {
		t.Fatal("Expected command from ctrl+enter")
	}

	// Execute the submit command
	_ = cmd()

	// Verify CreatePost was called correctly
	if !client.createPostCalled {
		t.Error("Expected CreatePost to be called")
	}
	if client.lastText != "This is a test post" {
		t.Errorf("Expected text 'This is a test post', got %q", client.lastText)
	}
	if client.lastReply != nil {
		t.Error("Expected no reply ref for new post")
	}
	if client.lastEmbed != nil {
		t.Error("Expected no embed for new post")
	}
}

// Test 9: TestSubmitReply
func TestSubmitReply(t *testing.T) {
	client := &mockBlueskyClient{
		createPostURI: "at://did:plc:test/app.bsky.feed.post/reply",
		createPostCID: "replycid",
	}
	parentPost := createTestPostView("at://did:plc:parent/app.bsky.feed.post/123", "parentcid", "parent.bsky.social", "Original post")

	m := NewComposeModel(client, ModeReply, parentPost, 80, 24)
	m.SetText("This is a reply")

	// Simulate Ctrl+Enter
	_, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+enter"})
	if cmd == nil {
		t.Fatal("Expected command from ctrl+enter")
	}

	// Execute the submit command
	_ = cmd()

	// Verify CreatePost was called with ReplyRef
	if !client.createPostCalled {
		t.Fatal("Expected CreatePost to be called")
	}
	if client.lastReply == nil {
		t.Fatal("Expected reply ref for reply post")
	}
	if client.lastReply.Root == nil {
		t.Fatal("Expected root in reply ref")
	}
	if client.lastReply.Parent == nil {
		t.Fatal("Expected parent in reply ref")
	}
	if client.lastReply.Root.Uri != "at://did:plc:parent/app.bsky.feed.post/123" {
		t.Errorf("Expected root URI, got %q", client.lastReply.Root.Uri)
	}
	if client.lastReply.Parent.Cid != "parentcid" {
		t.Errorf("Expected parent CID, got %q", client.lastReply.Parent.Cid)
	}
}

// Test 10: TestCancelCompose
func TestCancelCompose(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	// Simulate Esc
	_, cmd := m.Update(tea.KeyPressMsg{Text: "esc"})
	if cmd == nil {
		t.Fatal("Expected command from esc")
	}

	msg := cmd()
	cancelMsg, ok := msg.(CancelComposeMsg)
	if !ok {
		t.Errorf("Expected CancelComposeMsg, got %T", msg)
	}
	_ = cancelMsg // Just verify it's the right type
}

// Test 11: TestComposeLoadingState
func TestComposeLoadingState(t *testing.T) {
	client := &mockBlueskyClient{
		createPostURI: "at://did:plc:test/app.bsky.feed.post/new",
		createPostCID: "newcid",
	}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)
	m.SetText("Test post")

	// Before loading
	if m.loading {
		t.Error("Expected loading to be false initially")
	}

	// Set loading
	m.SetLoading(true)

	v := m.View()
	content := v.Content

	// Loading indicator should be present
	if !strings.Contains(content, "Posting") {
		t.Errorf("Expected loading indicator 'Posting', got %q", content)
	}
}

// Test 12: TestComposeError
func TestComposeError(t *testing.T) {
	client := &mockBlueskyClient{
		createPostErr: errors.New("network error"),
	}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)
	m.SetText("Test post")

	// Simulate Ctrl+Enter (will fail)
	_, cmd := m.Update(tea.KeyPressMsg{Text: "ctrl+enter"})
	if cmd == nil {
		t.Fatal("Expected command from ctrl+enter")
	}

	// Execute the command
	msg := cmd()
	_ = msg

	// Simulate receiving error
	updated, _ := m.Update(ComposeErrorMsg{Err: errors.New("network error")})
	m = updated.(ComposeModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Error") {
		t.Errorf("Expected error display, got %q", content)
	}
	if !strings.Contains(content, "network error") {
		t.Errorf("Expected error message, got %q", content)
	}
	if !strings.Contains(content, "Esc to clear") {
		t.Errorf("Expected hint to clear error, got %q", content)
	}
}

// Test 13: TestComposeInit
func TestComposeInit(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	cmd := m.Init()
	if cmd == nil {
		t.Error("Expected Init to return a command (for cursor blink)")
	}
}

// Test 14: TestCharCounterUnicode
func TestCharCounterUnicode(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	// Set text with Unicode characters (emoji, etc.)
	// "Hello 👋 World 🌍" should be 15 runes, not bytes
	m.SetText("Hello 👋 World 🌍")

	charCount := m.CharCount()
	if charCount != 15 {
		t.Errorf("Expected CharCount 15 for Unicode text, got %d", charCount)
	}

	v := m.View()
	content := v.Content
	if !strings.Contains(content, "15/300") {
		t.Errorf("Expected char counter '15/300', got %q", content)
	}
}

// Test 15: TestModeAccessor
func TestModeAccessor(t *testing.T) {
	client := &mockBlueskyClient{}

	// Test NewPost mode
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)
	if m.Mode() != ModeNewPost {
		t.Errorf("Expected ModeNewPost, got %v", m.Mode())
	}

	// Test Reply mode
	parentPost := createTestPostView("at://test", "cid", "user", "text")
	m = NewComposeModel(client, ModeReply, parentPost, 80, 24)
	if m.Mode() != ModeReply {
		t.Errorf("Expected ModeReply, got %v", m.Mode())
	}

	// Test Quote mode
	m = NewComposeModel(client, ModeQuote, parentPost, 80, 24)
	if m.Mode() != ModeQuote {
		t.Errorf("Expected ModeQuote, got %v", m.Mode())
	}
}

// Test 16: TestTextAccessor
func TestTextAccessor(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	// Initial text should be empty
	if m.Text() != "" {
		t.Errorf("Expected empty text initially, got %q", m.Text())
	}

	// Set text
	m.SetText("Test content")
	if m.Text() != "Test content" {
		t.Errorf("Expected 'Test content', got %q", m.Text())
	}
}

// Test 17: TestWindowSizeResize
func TestWindowSizeResize(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewComposeModel(client, ModeNewPost, nil, 80, 24)

	// Simulate window resize
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(ComposeModel)

	if m.width != 100 {
		t.Errorf("Expected width 100, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}
