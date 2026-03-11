package feed

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
	timelinePosts  []*bsky.FeedDefs_FeedViewPost
	timelineCursor string
	timelineErr    error
}

func (m *mockBlueskyClient) GetTimeline(ctx context.Context, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	if m.timelineErr != nil {
		return nil, "", m.timelineErr
	}
	return m.timelinePosts, m.timelineCursor, nil
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
	return "", "", nil
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

// Helper functions to create test posts
func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }

func createTestPost(text, handle, displayName, uri, cid string) *bsky.FeedDefs_FeedViewPost {
	return &bsky.FeedDefs_FeedViewPost{
		Post: &bsky.FeedDefs_PostView{
			Uri:         uri,
			Cid:         cid,
			IndexedAt:   time.Now().Format(time.RFC3339),
			LikeCount:   int64Ptr(5),
			RepostCount: int64Ptr(2),
			ReplyCount:  int64Ptr(3),
			Author: &bsky.ActorDefs_ProfileViewBasic{
				Did:         "did:plc:" + handle,
				Handle:      handle,
				DisplayName: strPtr(displayName),
			},
			Record: &util.LexiconTypeDecoder{
				Val: &bsky.FeedPost{
					LexiconTypeID: "app.bsky.feed.post",
					Text:          text,
					CreatedAt:     time.Now().Format(time.RFC3339),
				},
			},
		},
	}
}

func createTestPostWithFacets(text, handle string, facets []*bsky.RichtextFacet) *bsky.FeedDefs_FeedViewPost {
	post := createTestPost(text, handle, "Test User", "at://did:plc:test/app.bsky.feed.post/123", "testcid")
	post.Post.Record.Val.(*bsky.FeedPost).Facets = facets
	return post
}

// Test 1: TestFeedRenderSinglePost
func TestFeedRenderSinglePost(t *testing.T) {
	client := &mockBlueskyClient{
		timelinePosts: []*bsky.FeedDefs_FeedViewPost{
			createTestPost("Hello world!", "alice.bsky.social", "Alice", "at://uri1", "cid1"),
		},
	}

	m := NewFeedModel(client, 80, 24)
	m.posts = client.timelinePosts
	m.loading = false

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Hello world!") {
		t.Errorf("Expected content to contain 'Hello world!', got %q", content)
	}
	if !strings.Contains(content, "Alice") {
		t.Errorf("Expected content to contain author name 'Alice', got %q", content)
	}
	if !strings.Contains(content, "@alice.bsky.social") {
		t.Errorf("Expected content to contain handle, got %q", content)
	}
}

// Test 2: TestFeedRenderMultiplePosts
func TestFeedRenderMultiplePosts(t *testing.T) {
	client := &mockBlueskyClient{
		timelinePosts: []*bsky.FeedDefs_FeedViewPost{
			createTestPost("First post", "alice.bsky.social", "Alice", "at://uri1", "cid1"),
			createTestPost("Second post", "bob.bsky.social", "Bob", "at://uri2", "cid2"),
			createTestPost("Third post", "carol.bsky.social", "Carol", "at://uri3", "cid3"),
		},
	}

	m := NewFeedModel(client, 80, 24)
	m.posts = client.timelinePosts
	m.loading = false

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "First post") {
		t.Errorf("Expected content to contain 'First post', got %q", content)
	}
	if !strings.Contains(content, "Second post") {
		t.Errorf("Expected content to contain 'Second post', got %q", content)
	}
	if !strings.Contains(content, "Bob") {
		t.Errorf("Expected content to contain 'Bob', got %q", content)
	}
}

// Test 3: TestPostWidgetAuthorLine
func TestPostWidgetAuthorLine(t *testing.T) {
	post := createTestPost("Test content", "testhandle.bsky.social", "Test Display Name", "at://uri", "cid")

	rendered := RenderPost(post, 80, false)

	if !strings.Contains(rendered, "Test Display Name") {
		t.Errorf("Expected author line to contain display name 'Test Display Name', got %q", rendered)
	}
	if !strings.Contains(rendered, "@testhandle.bsky.social") {
		t.Errorf("Expected author line to contain handle, got %q", rendered)
	}
}

// Test 4: TestPostWidgetTimestamp (FormatRelativeTime)
func TestPostWidgetTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"30 seconds ago", time.Now().Add(-30 * time.Second), "30s"},
		{"5 minutes ago", time.Now().Add(-5 * time.Minute), "5m"},
		{"2 hours ago", time.Now().Add(-2 * time.Hour), "2h"},
		{"3 days ago", time.Now().Add(-3 * 24 * time.Hour), "3d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatRelativeTime(tt.time)
			if result != tt.expected {
				t.Errorf("FormatRelativeTime(%v) = %q, want %q", tt.time, result, tt.expected)
			}
		})
	}
}

// Test 5: TestFacetMentionHighlight
func TestFacetMentionHighlight(t *testing.T) {
	text := "Hello @alice.bsky.social!"
	facets := []*bsky.RichtextFacet{
		{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: 6,
				ByteEnd:   26,
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Mention: &bsky.RichtextFacet_Mention{
					LexiconTypeID: "app.bsky.richtext.facet#mention",
					Did:           "did:plc:alice",
				},
			}},
		},
	}

	post := createTestPostWithFacets(text, "test.bsky.social", facets)
	rendered := RenderPost(post, 80, false)

	// The mention should be rendered (content is present)
	if !strings.Contains(rendered, "@alice.bsky.social") {
		t.Errorf("Expected rendered post to contain mention text, got %q", rendered)
	}
}

// Test 6: TestFacetLinkUnderline
func TestFacetLinkUnderline(t *testing.T) {
	text := "Check out https://example.com"
	facets := []*bsky.RichtextFacet{
		{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: 10,
				ByteEnd:   29,
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Link: &bsky.RichtextFacet_Link{
					LexiconTypeID: "app.bsky.richtext.facet#link",
					Uri:           "https://example.com",
				},
			}},
		},
	}

	post := createTestPostWithFacets(text, "test.bsky.social", facets)
	rendered := RenderPost(post, 80, false)
	rendered = stripAnsi(rendered)

	if !strings.Contains(rendered, "https://example.com") {
		t.Errorf("Expected rendered post to contain link text, got %q", rendered)
	}
}

// Test 7: TestFacetWithEmoji (byte offset correctness)
func TestFacetWithEmoji(t *testing.T) {
	text := "Hello 👋 world"
	// "Hello " = 6 bytes, "👋" = 4 bytes, " " = 1 byte, "world" starts at byte 11
	facets := []*bsky.RichtextFacet{
		{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: 11,
				ByteEnd:   16,
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Tag: &bsky.RichtextFacet_Tag{
					LexiconTypeID: "app.bsky.richtext.facet#tag",
					Tag:           "world",
				},
			}},
		},
	}

	post := createTestPostWithFacets(text, "test.bsky.social", facets)
	rendered := RenderPost(post, 80, false)
	rendered = stripAnsi(rendered)

	if !strings.Contains(rendered, "👋") {
		t.Errorf("Expected rendered post to contain emoji, got %q", rendered)
	}
	if !strings.Contains(rendered, "world") {
		t.Errorf("Expected rendered post to contain 'world', got %q", rendered)
	}
}

// Test 8: TestScrollDown
func TestScrollDown(t *testing.T) {
	client := &mockBlueskyClient{
		timelinePosts: []*bsky.FeedDefs_FeedViewPost{
			createTestPost("Post 1", "a.bsky.social", "A", "at://1", "c1"),
			createTestPost("Post 2", "b.bsky.social", "B", "at://2", "c2"),
			createTestPost("Post 3", "c.bsky.social", "C", "at://3", "c3"),
		},
	}

	m := NewFeedModel(client, 80, 24)
	m.posts = client.timelinePosts
	m.loading = false

	if m.selectedIndex != 0 {
		t.Errorf("Expected initial selectedIndex to be 0, got %d", m.selectedIndex)
	}

	// Press 'j' to scroll down
	updated, _ := m.Update(tea.KeyPressMsg{Text: "j"})
	m = updated.(FeedModel)

	if m.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1 after j, got %d", m.selectedIndex)
	}

	// Press 'down' arrow
	updated, _ = m.Update(tea.KeyPressMsg{Text: "down"})
	m = updated.(FeedModel)

	if m.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex to be 2 after down, got %d", m.selectedIndex)
	}
}

// Test 9: TestScrollUp
func TestScrollUp(t *testing.T) {
	client := &mockBlueskyClient{
		timelinePosts: []*bsky.FeedDefs_FeedViewPost{
			createTestPost("Post 1", "a.bsky.social", "A", "at://1", "c1"),
			createTestPost("Post 2", "b.bsky.social", "B", "at://2", "c2"),
			createTestPost("Post 3", "c.bsky.social", "C", "at://3", "c3"),
		},
	}

	m := NewFeedModel(client, 80, 24)
	m.posts = client.timelinePosts
	m.selectedIndex = 2
	m.loading = false

	// Press 'k' to scroll up
	updated, _ := m.Update(tea.KeyPressMsg{Text: "k"})
	m = updated.(FeedModel)

	if m.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1 after k, got %d", m.selectedIndex)
	}

	// Press 'up' arrow
	updated, _ = m.Update(tea.KeyPressMsg{Text: "up"})
	m = updated.(FeedModel)

	if m.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0 after up, got %d", m.selectedIndex)
	}

	// Try to scroll up at top - should stay at 0
	updated, _ = m.Update(tea.KeyPressMsg{Text: "k"})
	m = updated.(FeedModel)

	if m.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to stay at 0 at top, got %d", m.selectedIndex)
	}
}

// Test 10: TestScrollPagination
func TestScrollPagination(t *testing.T) {
	// Create 25 posts
	posts := make([]*bsky.FeedDefs_FeedViewPost, 25)
	for i := 0; i < 25; i++ {
		posts[i] = createTestPost("Post "+string(rune('A'+i%26)), "user.bsky.social", "User", "at://uri", "cid")
	}

	client := &mockBlueskyClient{
		timelinePosts:  posts[:20],
		timelineCursor: "next-page-cursor",
	}

	m := NewFeedModel(client, 80, 24)
	m.posts = posts[:20]
	m.cursor = "next-page-cursor"
	m.loading = false

	// Scroll to near bottom (position 17, which is 20-3)
	m.selectedIndex = 17

	// Press down to trigger pagination (selectedIndex becomes 18, which is >= 20-3)
	updated, _ := m.Update(tea.KeyPressMsg{Text: "j"})
	m = updated.(FeedModel)

	// Should trigger loading
	if !m.loading {
		t.Errorf("Expected loading to be true when near bottom with cursor, got %v", m.loading)
	}
}

// Test 11: TestEmptyFeed
func TestEmptyFeed(t *testing.T) {
	client := &mockBlueskyClient{
		timelinePosts: []*bsky.FeedDefs_FeedViewPost{},
	}

	m := NewFeedModel(client, 80, 24)
	m.posts = nil
	m.loading = false

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "No posts yet") {
		t.Errorf("Expected empty feed to show 'No posts yet', got %q", content)
	}
}

// Test 12: TestFeedLoading
func TestFeedLoading(t *testing.T) {
	client := &mockBlueskyClient{}

	m := NewFeedModel(client, 80, 24)
	// loading is set to true in NewFeedModel

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Loading") {
		t.Errorf("Expected loading state to show 'Loading', got %q", content)
	}
}

// Test 13: TestPostSelection
func TestPostSelection(t *testing.T) {
	client := &mockBlueskyClient{
		timelinePosts: []*bsky.FeedDefs_FeedViewPost{
			createTestPost("Post 1", "a.bsky.social", "A", "at://1", "c1"),
			createTestPost("Post 2", "b.bsky.social", "B", "at://2", "c2"),
		},
	}

	m := NewFeedModel(client, 80, 24)
	m.posts = client.timelinePosts
	m.loading = false

	// First post should be selected
	v := m.View()
	content := v.Content
	if !strings.Contains(content, "▶") {
		t.Errorf("Expected selected post to have marker '▶', got %q", content)
	}

	// Move selection down
	updated, _ := m.Update(tea.KeyPressMsg{Text: "j"})
	m = updated.(FeedModel)

	if m.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1, got %d", m.selectedIndex)
	}

	// Move selection up
	updated, _ = m.Update(tea.KeyPressMsg{Text: "k"})
	m = updated.(FeedModel)

	if m.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0, got %d", m.selectedIndex)
	}
}

// Test 14: TestPostActionKeybinds
func TestPostActionKeybinds(t *testing.T) {
	client := &mockBlueskyClient{
		timelinePosts: []*bsky.FeedDefs_FeedViewPost{
			createTestPost("Test post", "test.bsky.social", "Test", "at://test/123", "testcid"),
		},
	}

	m := NewFeedModel(client, 80, 24)
	m.posts = client.timelinePosts
	m.loading = false

	// Test 'enter' - ViewThreadMsg
	_, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Error("Expected 'enter' to return a command")
	} else {
		msg := cmd()
		if vtMsg, ok := msg.(ViewThreadMsg); !ok {
			t.Errorf("Expected ViewThreadMsg, got %T", msg)
		} else if vtMsg.URI != "at://test/123" {
			t.Errorf("Expected URI 'at://test/123', got %q", vtMsg.URI)
		}
	}

	// Test 'l' - LikePostMsg
	_, cmd = m.Update(tea.KeyPressMsg{Text: "l"})
	if cmd == nil {
		t.Error("Expected 'l' to return a command")
	} else {
		msg := cmd()
		if likeMsg, ok := msg.(LikePostMsg); !ok {
			t.Errorf("Expected LikePostMsg, got %T", msg)
		} else {
			if likeMsg.URI != "at://test/123" {
				t.Errorf("Expected URI 'at://test/123', got %q", likeMsg.URI)
			}
			if likeMsg.CID != "testcid" {
				t.Errorf("Expected CID 'testcid', got %q", likeMsg.CID)
			}
		}
	}

	// Test 't' - RepostMsg
	_, cmd = m.Update(tea.KeyPressMsg{Text: "t"})
	if cmd == nil {
		t.Error("Expected 't' to return a command")
	} else {
		msg := cmd()
		if repostMsg, ok := msg.(RepostMsg); !ok {
			t.Errorf("Expected RepostMsg, got %T", msg)
		} else {
			if repostMsg.URI != "at://test/123" {
				t.Errorf("Expected URI 'at://test/123', got %q", repostMsg.URI)
			}
			if repostMsg.CID != "testcid" {
				t.Errorf("Expected CID 'testcid', got %q", repostMsg.CID)
			}
		}
	}

	// Test 'c' - ComposeMsg
	_, cmd = m.Update(tea.KeyPressMsg{Text: "c"})
	if cmd == nil {
		t.Error("Expected 'c' to return a command")
	} else {
		msg := cmd()
		if _, ok := msg.(ComposeMsg); !ok {
			t.Errorf("Expected ComposeMsg, got %T", msg)
		}
	}
}

// Test 15: TestFeedRefresh
func TestFeedRefresh(t *testing.T) {
	client := &mockBlueskyClient{
		timelinePosts: []*bsky.FeedDefs_FeedViewPost{
			createTestPost("New post", "new.bsky.social", "New", "at://new", "newcid"),
		},
	}

	m := NewFeedModel(client, 80, 24)
	m.posts = []*bsky.FeedDefs_FeedViewPost{
		createTestPost("Old post", "old.bsky.social", "Old", "at://old", "oldcid"),
	}
	m.selectedIndex = 5
	m.offset = 3
	m.loading = false

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "r"})
	m = updated.(FeedModel)

	if cmd == nil {
		t.Fatal("Expected 'r' to return a command")
	}
	msg := cmd()
	if _, ok := msg.(FeedRefreshMsg); !ok {
		t.Fatalf("Expected FeedRefreshMsg, got %T", msg)
	}

	updated, _ = m.Update(FeedRefreshMsg{})
	m = updated.(FeedModel)

	if m.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be reset to 0, got %d", m.selectedIndex)
	}
	if m.offset != 0 {
		t.Errorf("Expected offset to be reset to 0, got %d", m.offset)
	}
	if !m.loading {
		t.Error("Expected loading to be true during refresh")
	}
}

// Test 16: TestFeedError
func TestFeedError(t *testing.T) {
	client := &mockBlueskyClient{
		timelineErr: errors.New("network error"),
	}

	m := NewFeedModel(client, 80, 24)

	// Simulate receiving an error message
	updated, _ := m.Update(FeedErrorMsg{Err: errors.New("network error")})
	m = updated.(FeedModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Error") {
		t.Errorf("Expected error state to show 'Error', got %q", content)
	}
	if !strings.Contains(content, "network error") {
		t.Errorf("Expected error state to show error message, got %q", content)
	}
	if !strings.Contains(content, "'r' to retry") {
		t.Errorf("Expected error state to show retry hint, got %q", content)
	}
}

// Test 17: TestRepostIndicator
func TestRepostIndicator(t *testing.T) {
	post := createTestPost("Reposted content", "original.bsky.social", "Original", "at://orig", "origcid")
	post.Reason = &bsky.FeedDefs_FeedViewPost_Reason{
		FeedDefs_ReasonRepost: &bsky.FeedDefs_ReasonRepost{
			LexiconTypeID: "app.bsky.feed.defs#reasonRepost",
			By: &bsky.ActorDefs_ProfileViewBasic{
				Did:    "did:plc:reposter",
				Handle: "reposter.bsky.social",
			},
		},
	}

	rendered := RenderPost(post, 80, false)

	if !strings.Contains(rendered, "Reposted by") {
		t.Errorf("Expected repost indicator, got %q", rendered)
	}
	if !strings.Contains(rendered, "@reposter.bsky.social") {
		t.Errorf("Expected reposter handle in indicator, got %q", rendered)
	}
}

// Test 18: TestReplyIndicator
func TestReplyIndicator(t *testing.T) {
	post := createTestPost("This is a reply", "replier.bsky.social", "Replier", "at://reply", "replycid")
	post.Reply = &bsky.FeedDefs_ReplyRef{
		Parent: &bsky.FeedDefs_ReplyRef_Parent{
			FeedDefs_PostView: &bsky.FeedDefs_PostView{
				Uri: "at://parent",
				Author: &bsky.ActorDefs_ProfileViewBasic{
					Did:    "did:plc:parent",
					Handle: "parent.bsky.social",
				},
			},
		},
		Root: &bsky.FeedDefs_ReplyRef_Root{
			FeedDefs_PostView: &bsky.FeedDefs_PostView{
				Uri: "at://root",
			},
		},
	}

	rendered := RenderPost(post, 80, false)

	if !strings.Contains(rendered, "Replying to") {
		t.Errorf("Expected reply indicator, got %q", rendered)
	}
	if !strings.Contains(rendered, "@parent.bsky.social") {
		t.Errorf("Expected parent handle in indicator, got %q", rendered)
	}
}

// Test 19: TestFeedModelInit
func TestFeedModelInit(t *testing.T) {
	client := &mockBlueskyClient{
		timelinePosts: []*bsky.FeedDefs_FeedViewPost{
			createTestPost("Test", "test.bsky.social", "Test", "at://test", "cid"),
		},
	}

	m := NewFeedModel(client, 80, 24)

	if !m.loading {
		t.Error("Expected initial loading state to be true")
	}

	cmd := m.Init()
	if cmd == nil {
		t.Error("Expected Init to return a command")
	}
}

// Test 20: TestWindowSizeMsg
func TestWindowSizeMsg(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewFeedModel(client, 80, 24)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = updated.(FeedModel)

	if m.width != 100 {
		t.Errorf("Expected width to be 100, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("Expected height to be 50, got %d", m.height)
	}
}

// Test 21: TestFeedLoadedMsg
func TestFeedLoadedMsg(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewFeedModel(client, 80, 24)
	m.loading = true

	posts := []*bsky.FeedDefs_FeedViewPost{
		createTestPost("Test", "test.bsky.social", "Test", "at://test", "cid"),
	}

	updated, _ := m.Update(FeedLoadedMsg{Posts: posts, Cursor: "next"})
	m = updated.(FeedModel)

	if m.loading {
		t.Error("Expected loading to be false after FeedLoadedMsg")
	}
	if len(m.posts) != 1 {
		t.Errorf("Expected 1 post, got %d", len(m.posts))
	}
	if m.cursor != "next" {
		t.Errorf("Expected cursor 'next', got %q", m.cursor)
	}
}

// Test 22: TestFeedLoadedMsgAppend
func TestFeedLoadedMsgAppend(t *testing.T) {
	client := &mockBlueskyClient{}
	m := NewFeedModel(client, 80, 24)
	m.posts = []*bsky.FeedDefs_FeedViewPost{
		createTestPost("First", "a.bsky.social", "A", "at://1", "c1"),
	}
	m.cursor = "existing"

	newPosts := []*bsky.FeedDefs_FeedViewPost{
		createTestPost("Second", "b.bsky.social", "B", "at://2", "c2"),
	}

	updated, _ := m.Update(FeedLoadedMsg{Posts: newPosts, Cursor: "next"})
	m = updated.(FeedModel)

	if len(m.posts) != 2 {
		t.Errorf("Expected 2 posts after append, got %d", len(m.posts))
	}
}

// Test 23: TestEngagementLine
func TestEngagementLine(t *testing.T) {
	post := createTestPost("Test", "test.bsky.social", "Test", "at://test", "cid")
	post.Post.LikeCount = int64Ptr(42)
	post.Post.RepostCount = int64Ptr(10)
	post.Post.ReplyCount = int64Ptr(5)

	rendered := RenderPost(post, 80, false)

	if !strings.Contains(rendered, "42") {
		t.Errorf("Expected like count 42 in engagement line, got %q", rendered)
	}
	if !strings.Contains(rendered, "10") {
		t.Errorf("Expected repost count 10 in engagement line, got %q", rendered)
	}
	if !strings.Contains(rendered, "5") {
		t.Errorf("Expected reply count 5 in engagement line, got %q", rendered)
	}
}
