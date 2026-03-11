package search

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/dotBeeps/noms/internal/ui/feed"
)

// mockBlueskyClient implements bluesky.BlueskyClient for testing
type mockBlueskyClient struct {
	// Search results
	searchPosts        []*bsky.FeedDefs_PostView
	searchPostsCursor  string
	searchPostsErr     error
	searchActors       []*bsky.ActorDefs_ProfileView
	searchActorsCursor string
	searchActorsErr    error

	// Track calls
	searchPostsCalls []struct {
		query, cursor string
		limit         int
	}
	searchActorsCalls []struct {
		query, cursor string
		limit         int
	}
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
	m.searchPostsCalls = append(m.searchPostsCalls, struct {
		query, cursor string
		limit         int
	}{query, cursor, limit})
	if m.searchPostsErr != nil {
		return nil, "", m.searchPostsErr
	}
	return m.searchPosts, m.searchPostsCursor, nil
}

func (m *mockBlueskyClient) SearchActors(ctx context.Context, query string, cursor string, limit int) ([]*bsky.ActorDefs_ProfileView, string, error) {
	m.searchActorsCalls = append(m.searchActorsCalls, struct {
		query, cursor string
		limit         int
	}{query, cursor, limit})
	if m.searchActorsErr != nil {
		return nil, "", m.searchActorsErr
	}
	return m.searchActors, m.searchActorsCursor, nil
}

// Helper functions to create test data
func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }

func createTestPostView(text, handle, displayName, uri, cid string) *bsky.FeedDefs_PostView {
	return &bsky.FeedDefs_PostView{
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
	}
}

func createTestActor(handle, displayName, did, description string) *bsky.ActorDefs_ProfileView {
	return &bsky.ActorDefs_ProfileView{
		Did:         did,
		Handle:      handle,
		DisplayName: strPtr(displayName),
		Description: strPtr(description),
	}
}

// Test 1: TestSearchInput — renders input field, accepts text
func TestSearchInput(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)

	// Check input is focused
	if !m.input.Focused() {
		t.Error("Expected input to be focused initially")
	}

	// Check placeholder
	if m.input.Placeholder != "Type to search..." {
		t.Errorf("Expected placeholder 'Type to search...', got %q", m.input.Placeholder)
	}

	// Type into input
	updated, _ := m.Update(tea.KeyPressMsg{Text: "h"})
	m = updated.(SearchModel)
	updated, _ = m.Update(tea.KeyPressMsg{Text: "e"})
	m = updated.(SearchModel)
	updated, _ = m.Update(tea.KeyPressMsg{Text: "l"})
	m = updated.(SearchModel)
	updated, _ = m.Update(tea.KeyPressMsg{Text: "l"})
	m = updated.(SearchModel)
	updated, _ = m.Update(tea.KeyPressMsg{Text: "o"})
	m = updated.(SearchModel)

	if m.input.Value() != "hello" {
		t.Errorf("Expected input value 'hello', got %q", m.input.Value())
	}

	// View should contain the input
	v := m.View()
	if !strings.Contains(v.Content, "🔍") {
		t.Error("Expected view to contain search icon")
	}
}

// Test 2: TestSearchDebounce — search fires only after 300ms pause
func TestSearchDebounce(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{
		searchPosts: []*bsky.FeedDefs_PostView{
			createTestPostView("Test post", "test.bsky.social", "Test User", "at://test/1", "cid1"),
		},
	}
	m := NewSearchModel(client, 80, 24)

	// Type a character
	updated, cmd := m.Update(tea.KeyPressMsg{Text: "t"})
	m = updated.(SearchModel)

	// Should schedule a debounce tick
	if cmd == nil {
		t.Error("Expected debounce command to be scheduled")
	}

	// Verify no search has been made yet
	if len(client.searchPostsCalls) > 0 {
		t.Error("Expected no search calls immediately after typing")
	}

	// Simulate the debounce message arriving
	now := m.lastKeystroke
	debounceMsg := debounceMsg{ts: now}
	updated, _ = m.Update(debounceMsg)
	m = updated.(SearchModel)

	// After debounce, query should be set
	if m.query != "t" {
		t.Errorf("Expected query 't' after debounce, got %q", m.query)
	}
}

// Test 3: TestSearchPostsResults — displays post results correctly
func TestSearchPostsResults(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{
		searchPosts: []*bsky.FeedDefs_PostView{
			createTestPostView("Hello world", "alice.bsky.social", "Alice", "at://test/1", "cid1"),
			createTestPostView("Another post", "bob.bsky.social", "Bob", "at://test/2", "cid2"),
		},
	}
	m := NewSearchModel(client, 80, 24)
	m.mode = ModePosts
	m.query = "test"

	// Simulate receiving search results
	results := SearchResultsMsg{
		Posts:  client.searchPosts,
		Cursor: "",
		Mode:   ModePosts,
		Append: false,
	}
	updated, _ := m.Update(results)
	m = updated.(SearchModel)

	if len(m.postResults) != 2 {
		t.Errorf("Expected 2 post results, got %d", len(m.postResults))
	}

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Hello world") {
		t.Errorf("Expected view to contain 'Hello world', got %q", content)
	}
	if !strings.Contains(content, "Another post") {
		t.Errorf("Expected view to contain 'Another post', got %q", content)
	}
}

// Test 4: TestSearchPeopleResults — displays actor results correctly
func TestSearchPeopleResults(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{
		searchActors: []*bsky.ActorDefs_ProfileView{
			createTestActor("alice.bsky.social", "Alice", "did:plc:alice", "Hello I'm Alice"),
			createTestActor("bob.bsky.social", "Bob", "did:plc:bob", "Bob here"),
		},
	}
	m := NewSearchModel(client, 80, 24)
	m.mode = ModePeople
	m.query = "test"

	// Simulate receiving search results
	results := SearchResultsMsg{
		Actors: client.searchActors,
		Cursor: "",
		Mode:   ModePeople,
		Append: false,
	}
	updated, _ := m.Update(results)
	m = updated.(SearchModel)

	if len(m.actorResults) != 2 {
		t.Errorf("Expected 2 actor results, got %d", len(m.actorResults))
	}

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "alice.bsky.social") {
		t.Errorf("Expected view to contain 'alice.bsky.social', got %q", content)
	}
	if !strings.Contains(content, "Alice") {
		t.Errorf("Expected view to contain 'Alice', got %q", content)
	}
}

// Test 5: TestSearchToggleMode — Tab switches Posts/People
func TestSearchToggleMode(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)

	// Initially Posts mode
	if m.mode != ModePosts {
		t.Errorf("Expected initial mode to be ModePosts, got %v", m.mode)
	}

	// Press Tab to toggle
	updated, _ := m.Update(tea.KeyPressMsg{Text: "tab"})
	m = updated.(SearchModel)

	if m.mode != ModePeople {
		t.Errorf("Expected mode to be ModePeople after Tab, got %v", m.mode)
	}

	// Press Tab again
	updated, _ = m.Update(tea.KeyPressMsg{Text: "tab"})
	m = updated.(SearchModel)

	if m.mode != ModePosts {
		t.Errorf("Expected mode to be ModePosts after second Tab, got %v", m.mode)
	}

	// Verify UI shows correct mode
	v := m.View()
	content := v.Content

	// When in Posts mode, Posts tab should be active
	if !strings.Contains(content, "[Posts]") {
		t.Errorf("Expected view to contain '[Posts]' tab, got %q", content)
	}
}

// Test 6: TestSearchResultNavigation — j/k moves through results
func TestSearchResultNavigation(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{
		searchPosts: []*bsky.FeedDefs_PostView{
			createTestPostView("Post 1", "a.bsky.social", "A", "at://1", "c1"),
			createTestPostView("Post 2", "b.bsky.social", "B", "at://2", "c2"),
			createTestPostView("Post 3", "c.bsky.social", "C", "at://3", "c3"),
		},
	}
	m := NewSearchModel(client, 80, 24)
	m.mode = ModePosts
	m.postResults = client.searchPosts
	m.input.Blur() // Unfocus input to enable navigation

	// Initially at index 0
	if m.selectedIndex != 0 {
		t.Errorf("Expected initial selectedIndex to be 0, got %d", m.selectedIndex)
	}

	// Press 'j' to move down
	updated, _ := m.Update(tea.KeyPressMsg{Text: "j"})
	m = updated.(SearchModel)

	if m.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1 after 'j', got %d", m.selectedIndex)
	}

	// Press 'down' arrow
	updated, _ = m.Update(tea.KeyPressMsg{Text: "down"})
	m = updated.(SearchModel)

	if m.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex to be 2 after 'down', got %d", m.selectedIndex)
	}

	// Press 'k' to move up
	updated, _ = m.Update(tea.KeyPressMsg{Text: "k"})
	m = updated.(SearchModel)

	if m.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1 after 'k', got %d", m.selectedIndex)
	}

	// Press 'up' arrow
	updated, _ = m.Update(tea.KeyPressMsg{Text: "up"})
	m = updated.(SearchModel)

	if m.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0 after 'up', got %d", m.selectedIndex)
	}

	// Try to go above top - should stay at 0
	updated, _ = m.Update(tea.KeyPressMsg{Text: "k"})
	m = updated.(SearchModel)

	if m.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to stay at 0 at top, got %d", m.selectedIndex)
	}
}

// Test 7: TestSearchEmpty — shows placeholder when no query
func TestSearchEmpty(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)

	// Empty query
	if m.query != "" {
		t.Errorf("Expected initial query to be empty, got %q", m.query)
	}

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Type to search...") {
		t.Errorf("Expected placeholder 'Type to search...', got %q", content)
	}
}

// Test 8: TestSearchNoResults — shows "No results" message
func TestSearchNoResults(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{
		searchPosts: []*bsky.FeedDefs_PostView{}, // Empty results
	}
	m := NewSearchModel(client, 80, 24)
	m.mode = ModePosts
	m.query = "nonexistent"

	// Simulate receiving empty results
	results := SearchResultsMsg{
		Posts:  []*bsky.FeedDefs_PostView{},
		Cursor: "",
		Mode:   ModePosts,
		Append: false,
	}
	updated, _ := m.Update(results)
	m = updated.(SearchModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "No results") {
		t.Errorf("Expected 'No results' message, got %q", content)
	}
}

// Test 9: TestSearchPagination — cursor-based load more
func TestSearchPagination(t *testing.T) {
	t.Parallel()
	// First page of results
	firstPage := []*bsky.FeedDefs_PostView{
		createTestPostView("Post 1", "a.bsky.social", "A", "at://1", "c1"),
		createTestPostView("Post 2", "b.bsky.social", "B", "at://2", "c2"),
	}

	// Second page of results
	secondPage := []*bsky.FeedDefs_PostView{
		createTestPostView("Post 3", "c.bsky.social", "C", "at://3", "c3"),
		createTestPostView("Post 4", "d.bsky.social", "D", "at://4", "c4"),
	}

	client := &mockBlueskyClient{
		searchPosts:       firstPage,
		searchPostsCursor: "next-page-cursor",
	}
	m := NewSearchModel(client, 80, 24)
	m.mode = ModePosts
	m.query = "test"
	m.postResults = firstPage
	m.cursor = "next-page-cursor"
	m.input.Blur()

	// Move to the last item to trigger pagination
	m.selectedIndex = len(firstPage) - 1

	// Update mock for second page
	client.searchPosts = secondPage
	client.searchPostsCursor = ""

	// Press down at the end to trigger load more
	updated, _ := m.Update(tea.KeyPressMsg{Text: "down"})
	m = updated.(SearchModel)

	// Should be loading
	if !m.loading {
		t.Error("Expected loading to be true when triggering pagination")
	}

	// Simulate receiving second page with append
	results := SearchResultsMsg{
		Posts:  secondPage,
		Cursor: "",
		Mode:   ModePosts,
		Append: true,
	}
	updated, _ = m.Update(results)
	m = updated.(SearchModel)

	if len(m.postResults) != 4 {
		t.Errorf("Expected 4 total results after pagination, got %d", len(m.postResults))
	}
}

// Test 10: TestSearchEnterNavigation — enter navigates to thread/profile
func TestSearchEnterNavigation(t *testing.T) {
	t.Parallel()
	t.Run("Posts mode navigates to thread", func(t *testing.T) {
		client := &mockBlueskyClient{
			searchPosts: []*bsky.FeedDefs_PostView{
				createTestPostView("Test post", "test.bsky.social", "Test", "at://test/123", "cid123"),
			},
		}
		m := NewSearchModel(client, 80, 24)
		m.mode = ModePosts
		m.postResults = client.searchPosts
		m.input.Blur()

		updated, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
		m = updated.(SearchModel)

		if cmd == nil {
			t.Fatal("Expected command to be returned on enter")
		}

		msg := cmd()
		threadMsg, ok := msg.(feed.ViewThreadMsg)
		if !ok {
			t.Errorf("Expected ViewThreadMsg, got %T", msg)
		} else if threadMsg.URI != "at://test/123" {
			t.Errorf("Expected URI 'at://test/123', got %q", threadMsg.URI)
		}
	})

	t.Run("People mode navigates to profile", func(t *testing.T) {
		client := &mockBlueskyClient{
			searchActors: []*bsky.ActorDefs_ProfileView{
				createTestActor("test.bsky.social", "Test User", "did:plc:test", "Bio"),
			},
		}
		m := NewSearchModel(client, 80, 24)
		m.mode = ModePeople
		m.actorResults = client.searchActors
		m.input.Blur()

		updated, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
		m = updated.(SearchModel)

		if cmd == nil {
			t.Fatal("Expected command to be returned on enter")
		}

		msg := cmd()
		profileMsg, ok := msg.(ViewProfileMsg)
		if !ok {
			t.Errorf("Expected ViewProfileMsg, got %T", msg)
		} else if profileMsg.DID != "did:plc:test" {
			t.Errorf("Expected DID 'did:plc:test', got %q", profileMsg.DID)
		}
	})
}

// Test 11: TestSearchError — handles errors gracefully
func TestSearchError(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{
		searchPostsErr: errors.New("network error"),
	}
	m := NewSearchModel(client, 80, 24)
	m.mode = ModePosts
	m.query = "test"

	// Simulate receiving error
	errMsg := SearchErrorMsg{Err: errors.New("network error")}
	updated, _ := m.Update(errMsg)
	m = updated.(SearchModel)

	if m.err == nil {
		t.Error("Expected error to be set")
	}

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Error") {
		t.Errorf("Expected error message in view, got %q", content)
	}
}

// Test 12: TestSearchEscape — clears search on escape when input is blurred
func TestSearchEscape(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)
	m.query = "test"
	m.postResults = []*bsky.FeedDefs_PostView{
		createTestPostView("Test", "test.bsky.social", "Test", "at://1", "c1"),
	}
	m.input.Blur() // Blur input first so escape clears search

	// Press escape
	updated, _ := m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updated.(SearchModel)

	if m.query != "" {
		t.Errorf("Expected query to be cleared, got %q", m.query)
	}
	if len(m.postResults) != 0 {
		t.Errorf("Expected results to be cleared, got %d", len(m.postResults))
	}
}

// Test 12b: TestSearchEscapeBlurs — escape blurs input when focused
func TestSearchEscapeBlurs(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)
	m.query = "test"
	m.postResults = []*bsky.FeedDefs_PostView{
		createTestPostView("Test", "test.bsky.social", "Test", "at://1", "c1"),
	}
	// Input is focused by default

	// Press escape - should blur but not clear
	updated, _ := m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updated.(SearchModel)

	if m.input.Focused() {
		t.Error("Expected input to be blurred after escape")
	}
	// Query and results should still exist
	if m.query != "test" {
		t.Errorf("Expected query to still be 'test', got %q", m.query)
	}
}

// Test 13: TestSearchFocusSlash — slash key focuses input
func TestSearchFocusSlash(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)
	m.input.Blur()

	if m.input.Focused() {
		t.Error("Input should not be focused initially")
	}

	// Press '/' to focus
	updated, cmd := m.Update(tea.KeyPressMsg{Text: "/"})
	m = updated.(SearchModel)

	if !m.input.Focused() {
		t.Error("Expected input to be focused after '/'")
	}
	if cmd == nil {
		t.Error("Expected blink command after focusing input")
	}
}

// Test 14: TestSearchInit — Init returns blink command
func TestSearchInit(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)

	cmd := m.Init()
	if cmd == nil {
		t.Error("Expected Init to return blink command")
	}
}

// Test 15: TestSearchWindowSizeMsg — handles window resize
func TestSearchWindowSizeMsg(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = updated.(SearchModel)

	if m.width != 100 {
		t.Errorf("Expected width to be 100, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("Expected height to be 50, got %d", m.height)
	}
}

// Test 16: TestSearchStatusBar — shows correct status
func TestSearchStatusBar(t *testing.T) {
	t.Parallel()
	t.Run("Shows Searching during load", func(t *testing.T) {
		client := &mockBlueskyClient{}
		m := NewSearchModel(client, 80, 24)
		m.query = "test"
		m.loading = true

		v := m.View()
		if !strings.Contains(v.Content, "Searching") {
			t.Errorf("Expected status to show 'Searching', got %q", v.Content)
		}
	})

	t.Run("Shows result count", func(t *testing.T) {
		client := &mockBlueskyClient{}
		m := NewSearchModel(client, 80, 24)
		m.query = "test"
		m.postResults = []*bsky.FeedDefs_PostView{
			createTestPostView("Test", "test.bsky.social", "Test", "at://1", "c1"),
			createTestPostView("Test2", "test2.bsky.social", "Test2", "at://2", "c2"),
		}

		v := m.View()
		if !strings.Contains(v.Content, "2 results") {
			t.Errorf("Expected status to show '2 results', got %q", v.Content)
		}
	})
}

func TestSearchMouseWheelDownScrolls(t *testing.T) {
	t.Parallel()
	posts := make([]*bsky.FeedDefs_PostView, 10)
	for i := range 10 {
		posts[i] = createTestPostView("Post", "user.bsky.social", "User", "at://uri"+string(rune('0'+i)), "cid")
	}
	client := &mockBlueskyClient{searchPosts: posts}
	m := NewSearchModel(client, 80, 24)
	m.mode = ModePosts
	m.postResults = posts
	m.input.Blur()

	updated, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = updated.(SearchModel)

	if m.selectedIndex != 3 {
		t.Errorf("Expected selectedIndex=3 after mouse wheel down, got %d", m.selectedIndex)
	}
}

func TestSearchMouseWheelUpScrolls(t *testing.T) {
	t.Parallel()
	posts := make([]*bsky.FeedDefs_PostView, 10)
	for i := range 10 {
		posts[i] = createTestPostView("Post", "user.bsky.social", "User", "at://uri"+string(rune('0'+i)), "cid")
	}
	client := &mockBlueskyClient{searchPosts: posts}
	m := NewSearchModel(client, 80, 24)
	m.mode = ModePosts
	m.postResults = posts
	m.input.Blur()
	m.selectedIndex = 5

	updated, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = updated.(SearchModel)

	if m.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex=2 after mouse wheel up from 5, got %d", m.selectedIndex)
	}
}

func TestSearchMouseWheelBoundsCheck(t *testing.T) {
	t.Parallel()
	posts := []*bsky.FeedDefs_PostView{
		createTestPostView("Post 1", "a.bsky.social", "A", "at://1", "c1"),
		createTestPostView("Post 2", "b.bsky.social", "B", "at://2", "c2"),
	}
	client := &mockBlueskyClient{searchPosts: posts}
	m := NewSearchModel(client, 80, 24)
	m.mode = ModePosts
	m.postResults = posts
	m.input.Blur()

	updated, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = updated.(SearchModel)
	if m.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex capped at 1, got %d", m.selectedIndex)
	}

	m.selectedIndex = 0
	updated, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = updated.(SearchModel)
	if m.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to stay at 0, got %d", m.selectedIndex)
	}
}

func TestSearchSpinnerTickDuringLoad(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)
	m.loading = true

	_, cmd := m.Update(m.spinner.Tick())
	if cmd == nil {
		t.Error("Expected spinner tick to return a command when loading")
	}
}

func TestSearchSpinnerTickIgnoredWhenNotLoading(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)
	m.loading = false

	_, cmd := m.Update(m.spinner.Tick())
	if cmd != nil {
		t.Error("Expected spinner tick to return nil command when not loading")
	}
}

func TestSearchLoadingViewShowsSpinner(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{}
	m := NewSearchModel(client, 80, 24)
	m.query = "test"
	m.loading = true

	v := m.View()
	if !strings.Contains(v.Content, "Searching") {
		t.Error("Expected searching text in view while loading")
	}
}

// Test 17: TestSearchModeResetsResults — switching mode clears results
func TestSearchModeResetsResults(t *testing.T) {
	t.Parallel()
	client := &mockBlueskyClient{
		searchPosts: []*bsky.FeedDefs_PostView{
			createTestPostView("Post", "test.bsky.social", "Test", "at://1", "c1"),
		},
		searchActors: []*bsky.ActorDefs_ProfileView{
			createTestActor("test.bsky.social", "Test", "did:plc:test", "Bio"),
		},
	}
	m := NewSearchModel(client, 80, 24)
	m.query = "test"
	m.postResults = []*bsky.FeedDefs_PostView{
		createTestPostView("Old post", "old.bsky.social", "Old", "at://old", "cold"),
	}

	// Toggle mode to People
	updated, _ := m.Update(tea.KeyPressMsg{Text: "tab"})
	m = updated.(SearchModel)

	if m.mode != ModePeople {
		t.Error("Expected mode to be People")
	}
	if len(m.postResults) != 0 {
		t.Error("Expected post results to be cleared after mode switch")
	}
}
