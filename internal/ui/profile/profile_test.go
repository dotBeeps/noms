package profile

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

// mockClient implements bluesky.BlueskyClient for testing.
type mockClient struct {
	getProfileErr      error
	profile            *bsky.ActorDefs_ProfileViewDetailed
	getAuthorFeedErr   error
	authorFeed         []*bsky.FeedDefs_FeedViewPost
	feedCursor         string
	followErr          error
	unfollowErr        error
	followCalled       bool
	unfollowCalled     bool
	followCalledWith   string
	unfollowCalledWith string
}

func (m *mockClient) GetTimeline(ctx context.Context, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	return nil, "", nil
}

func (m *mockClient) GetPost(ctx context.Context, uri string) (*bsky.FeedDefs_PostView, error) {
	return nil, nil
}

func (m *mockClient) GetPostThread(ctx context.Context, uri string, depth int) (*bsky.FeedGetPostThread_Output, error) {
	return nil, nil
}

func (m *mockClient) GetProfile(ctx context.Context, actor string) (*bsky.ActorDefs_ProfileViewDetailed, error) {
	if m.getProfileErr != nil {
		return nil, m.getProfileErr
	}
	return m.profile, nil
}

func (m *mockClient) GetAuthorFeed(ctx context.Context, actor string, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	if m.getAuthorFeedErr != nil {
		return nil, "", m.getAuthorFeedErr
	}
	return m.authorFeed, m.feedCursor, nil
}

func (m *mockClient) FollowActor(ctx context.Context, did string) error {
	m.followCalled = true
	m.followCalledWith = did
	return m.followErr
}

func (m *mockClient) UnfollowActor(ctx context.Context, did string) error {
	m.unfollowCalled = true
	m.unfollowCalledWith = did
	return m.unfollowErr
}

func (m *mockClient) ListNotifications(ctx context.Context, cursor string, limit int) ([]*bsky.NotificationListNotifications_Notification, string, error) {
	return nil, "", nil
}

func (m *mockClient) GetUnreadCount(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *mockClient) MarkNotificationsRead(ctx context.Context, seenAt time.Time) error {
	return nil
}

func (m *mockClient) CreatePost(ctx context.Context, text string, facets []*bsky.RichtextFacet, reply *bsky.FeedPost_ReplyRef, embed *bsky.FeedPost_Embed) (string, string, error) {
	return "", "", nil
}

func (m *mockClient) DeletePost(ctx context.Context, uri string) error {
	return nil
}

func (m *mockClient) Like(ctx context.Context, uri, cid string) (string, error) {
	return "", nil
}

func (m *mockClient) Unlike(ctx context.Context, likeURI string) error {
	return nil
}

func (m *mockClient) Repost(ctx context.Context, uri, cid string) (string, error) {
	return "", nil
}

func (m *mockClient) UnRepost(ctx context.Context, repostURI string) error {
	return nil
}

func (m *mockClient) SearchPosts(ctx context.Context, query string, cursor string, limit int) ([]*bsky.FeedDefs_PostView, string, error) {
	return nil, "", nil
}

func (m *mockClient) SearchActors(ctx context.Context, query string, cursor string, limit int) ([]*bsky.ActorDefs_ProfileView, string, error) {
	return nil, "", nil
}

// Helper functions to create test data

func strPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func makeTestProfile(handle, displayName, bio string, followers, following, posts int64, followingURI *string) *bsky.ActorDefs_ProfileViewDetailed {
	return &bsky.ActorDefs_ProfileViewDetailed{
		Handle:         handle,
		DisplayName:    strPtr(displayName),
		Description:    strPtr(bio),
		FollowersCount: int64Ptr(followers),
		FollowsCount:   int64Ptr(following),
		PostsCount:     int64Ptr(posts),
		Viewer: &bsky.ActorDefs_ViewerState{
			Following: followingURI,
		},
	}
}

func makeTestPost(handle, displayName, text, uri string, likes, reposts, replies int64) *bsky.FeedDefs_FeedViewPost {
	return &bsky.FeedDefs_FeedViewPost{
		Post: &bsky.FeedDefs_PostView{
			Author: &bsky.ActorDefs_ProfileViewBasic{
				Handle:      handle,
				DisplayName: strPtr(displayName),
			},
			Uri:         uri,
			Cid:         "test-cid",
			LikeCount:   int64Ptr(likes),
			RepostCount: int64Ptr(reposts),
			ReplyCount:  int64Ptr(replies),
			Record: &util.LexiconTypeDecoder{
				Val: &bsky.FeedPost{
					LexiconTypeID: "app.bsky.feed.post",
					Text:          text,
				},
			},
		},
	}
}

// Tests

func TestProfileHeaderRender(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Hello world!", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)

	view := model.View()
	content := view.Content

	if !strings.Contains(content, "Alice") {
		t.Errorf("Expected profile to contain display name 'Alice', got: %s", content)
	}

	if !strings.Contains(content, "@alice.bsky.social") {
		t.Errorf("Expected profile to contain handle '@alice.bsky.social', got: %s", content)
	}

	if !strings.Contains(content, "Hello world!") {
		t.Errorf("Expected profile to contain bio 'Hello world!', got: %s", content)
	}
}

func TestProfileBioWithFacets(t *testing.T) {
	t.Parallel()
	bio := "Check out my website https://example.com and follow @bob.bsky.social!"
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", bio, 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)

	view := model.View()
	content := view.Content

	if !strings.Contains(content, "https://example.com") {
		t.Errorf("Expected bio to contain 'https://example.com', got: %s", content)
	}

	if !strings.Contains(content, "@bob.bsky.social") {
		t.Errorf("Expected bio to contain '@bob.bsky.social', got: %s", content)
	}
}

func TestFollowerCountFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		count    int64
		expected string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1K"},
		{1500, "1.5K"},
		{10000, "10K"},
		{12345, "12.3K"},
		{100000, "100K"},
		{999999, "999K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
	}

	for _, tt := range tests {
		result := formatCount(tt.count)
		if result != tt.expected {
			t.Errorf("formatCount(%d) = %q, want %q", tt.count, result, tt.expected)
		}
	}
}

func TestFollowAction(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)

	// Press 'f' to follow
	updated, cmd := model.Update(tea.KeyPressMsg{Text: "f"})
	model = updated.(ProfileModel)

	// Simulate the command execution (it should call FollowActor)
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(FollowToggledMsg); !ok {
			t.Errorf("Expected FollowToggledMsg, got %T", msg)
		}
	}

	// Check that follow was called
	if !mock.followCalled {
		t.Error("Expected FollowActor to be called")
	}

	if mock.followCalledWith != "did:plc:alice" {
		t.Errorf("Expected FollowActor called with 'did:plc:alice', got %q", mock.followCalledWith)
	}
}

func TestUnfollowAction(t *testing.T) {
	t.Parallel()
	followURI := "at://did:plc:bob/app.bsky.graph.follow/123"
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, &followURI),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)

	// Check initial state shows Following
	view := model.View()
	if !strings.Contains(view.Content, "Following") {
		t.Errorf("Expected profile to show 'Following' status, got: %s", view.Content)
	}

	// Press 'f' to unfollow
	updated, cmd := model.Update(tea.KeyPressMsg{Text: "f"})
	model = updated.(ProfileModel)

	// Simulate the command execution
	if cmd != nil {
		msg := cmd()
		if ftMsg, ok := msg.(FollowToggledMsg); !ok {
			t.Errorf("Expected FollowToggledMsg, got %T", msg)
		} else if ftMsg.Following {
			t.Error("Expected Following to be false in FollowToggledMsg")
		}
	}

	// Check that unfollow was called with the follow URI
	if !mock.unfollowCalled {
		t.Error("Expected UnfollowActor to be called")
	}

	if mock.unfollowCalledWith != followURI {
		t.Errorf("Expected UnfollowActor called with %q, got %q", followURI, mock.unfollowCalledWith)
	}
}

func TestOwnProfileNoFollowButton(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}

	// Create model where viewDID == ownDID (own profile)
	model := NewProfileModel(mock, "did:plc:alice", "did:plc:alice", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)

	view := model.View()
	content := view.Content

	// Should not show follow button for own profile
	if strings.Contains(content, "[Follow]") || strings.Contains(content, "[Following") {
		t.Errorf("Expected no follow button for own profile, got: %s", content)
	}

	// Press 'f' - should be ignored
	updated, cmd := model.Update(tea.KeyPressMsg{Text: "f"})
	model = updated.(ProfileModel)

	if cmd != nil {
		t.Error("Expected no command when pressing 'f' on own profile")
	}

	if mock.followCalled {
		t.Error("Expected FollowActor NOT to be called for own profile")
	}
}

func TestProfileNotFound(t *testing.T) {
	t.Parallel()
	testErr := errors.New("profile not found")
	mock := &mockClient{
		getProfileErr: testErr,
	}

	model := NewProfileModel(mock, "did:plc:nonexistent", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileErrorMsg{Err: testErr})
	model = updated.(ProfileModel)

	view := model.View()
	content := view.Content

	if !strings.Contains(content, "Error:") {
		t.Errorf("Expected error message to contain 'Error:', got: %s", content)
	}

	if !strings.Contains(content, "profile not found") {
		t.Errorf("Expected error message to contain 'profile not found', got: %s", content)
	}
}

func TestProfileAuthorFeed(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{
			makeTestPost("alice.bsky.social", "Alice", "First post!", "at://post1", 10, 5, 2),
			makeTestPost("alice.bsky.social", "Alice", "Second post!", "at://post2", 20, 8, 3),
		},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	updated, _ = model.Update(AuthorFeedLoadedMsg{Posts: mock.authorFeed, Cursor: ""})
	model = updated.(ProfileModel)

	view := model.View()
	content := view.Content

	if !strings.Contains(content, "First post!") {
		t.Errorf("Expected feed to contain 'First post!', got: %s", content)
	}

	if !strings.Contains(content, "Second post!") {
		t.Errorf("Expected feed to contain 'Second post!', got: %s", content)
	}

	// Check engagement counts are rendered
	if !strings.Contains(content, "10") {
		t.Errorf("Expected feed to contain like count '10', got: %s", content)
	}
}

func TestProfileAuthorFeedPagination(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{
			makeTestPost("alice.bsky.social", "Alice", "Post 1", "at://post1", 0, 0, 0),
		},
		feedCursor: "next-cursor",
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	updated, _ = model.Update(AuthorFeedLoadedMsg{Posts: mock.authorFeed, Cursor: "next-cursor"})
	model = updated.(ProfileModel)

	// Verify initial feed is loaded
	if len(model.authorFeed) != 1 {
		t.Errorf("Expected 1 post, got %d", len(model.authorFeed))
	}

	if model.cursor != "next-cursor" {
		t.Errorf("Expected cursor 'next-cursor', got %q", model.cursor)
	}
}

func TestBackNavigation(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)

	// Press 'esc'
	updated, cmd := model.Update(tea.KeyPressMsg{Text: "esc"})
	model = updated.(ProfileModel)

	if cmd == nil {
		t.Error("Expected command from 'esc' key press")
	} else {
		msg := cmd()
		if _, ok := msg.(BackMsg); !ok {
			t.Errorf("Expected BackMsg, got %T", msg)
		}
	}

	// Test 'backspace' as well
	updated, cmd = model.Update(tea.KeyPressMsg{Text: "backspace"})
	model = updated.(ProfileModel)

	if cmd == nil {
		t.Error("Expected command from 'backspace' key press")
	} else {
		msg := cmd()
		if _, ok := msg.(BackMsg); !ok {
			t.Errorf("Expected BackMsg, got %T", msg)
		}
	}
}

func TestViewThreadNavigation(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{
			makeTestPost("alice.bsky.social", "Alice", "Test post", "at://did:plc:alice/app.bsky.feed.post/123", 5, 2, 1),
		},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	updated, _ = model.Update(AuthorFeedLoadedMsg{Posts: mock.authorFeed, Cursor: ""})
	model = updated.(ProfileModel)

	// Press 'enter' on selected post
	updated, cmd := model.Update(tea.KeyPressMsg{Text: "enter"})
	model = updated.(ProfileModel)

	if cmd == nil {
		t.Error("Expected command from 'enter' key press")
	} else {
		msg := cmd()
		if vtMsg, ok := msg.(ViewThreadMsg); !ok {
			t.Errorf("Expected ViewThreadMsg, got %T", msg)
		} else if vtMsg.URI != "at://did:plc:alice/app.bsky.feed.post/123" {
			t.Errorf("Expected URI 'at://did:plc:alice/app.bsky.feed.post/123', got %q", vtMsg.URI)
		}
	}
}

func TestRefreshProfile(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{
			makeTestPost("alice.bsky.social", "Alice", "Old post", "at://post1", 0, 0, 0),
		},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	updated, _ = model.Update(AuthorFeedLoadedMsg{Posts: mock.authorFeed, Cursor: ""})
	model = updated.(ProfileModel)

	// Verify initial state
	if len(model.authorFeed) != 1 {
		t.Errorf("Expected 1 post initially, got %d", len(model.authorFeed))
	}

	// Press 'r' to refresh
	updated, cmd := model.Update(tea.KeyPressMsg{Text: "r"})
	model = updated.(ProfileModel)

	if cmd == nil {
		t.Error("Expected command from 'r' key press")
	}

	// After refresh, the model should be reset
	if !model.loading {
		t.Error("Expected loading to be true after refresh")
	}

	if model.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0 after refresh, got %d", model.selectedIndex)
	}

	if model.cursor != "" {
		t.Errorf("Expected cursor to be empty after refresh, got %q", model.cursor)
	}
}

func TestNavigateFeed(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{
			makeTestPost("alice.bsky.social", "Alice", "Post 1", "at://post1", 0, 0, 0),
			makeTestPost("alice.bsky.social", "Alice", "Post 2", "at://post2", 0, 0, 0),
			makeTestPost("alice.bsky.social", "Alice", "Post 3", "at://post3", 0, 0, 0),
		},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	updated, _ = model.Update(AuthorFeedLoadedMsg{Posts: mock.authorFeed, Cursor: ""})
	model = updated.(ProfileModel)

	// Initial selection should be 0
	if model.selectedIndex != 0 {
		t.Errorf("Expected initial selectedIndex to be 0, got %d", model.selectedIndex)
	}

	// Press 'j' to move down
	updated, _ = model.Update(tea.KeyPressMsg{Text: "j"})
	model = updated.(ProfileModel)
	if model.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1 after 'j', got %d", model.selectedIndex)
	}

	// Press 'down' to move down again
	updated, _ = model.Update(tea.KeyPressMsg{Text: "down"})
	model = updated.(ProfileModel)
	if model.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex to be 2 after 'down', got %d", model.selectedIndex)
	}

	// Press 'j' at end - should stay at end
	updated, _ = model.Update(tea.KeyPressMsg{Text: "j"})
	model = updated.(ProfileModel)
	if model.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex to stay at 2 at end, got %d", model.selectedIndex)
	}

	// Press 'k' to move up
	updated, _ = model.Update(tea.KeyPressMsg{Text: "k"})
	model = updated.(ProfileModel)
	if model.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1 after 'k', got %d", model.selectedIndex)
	}

	// Press 'up' to move up again
	updated, _ = model.Update(tea.KeyPressMsg{Text: "up"})
	model = updated.(ProfileModel)
	if model.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to be 0 after 'up', got %d", model.selectedIndex)
	}

	// Press 'k' at start - should stay at start
	updated, _ = model.Update(tea.KeyPressMsg{Text: "k"})
	model = updated.(ProfileModel)
	if model.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex to stay at 0 at start, got %d", model.selectedIndex)
	}
}

func TestProfileDeleteFirstDPress(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{
			makeTestPost("alice.bsky.social", "Alice", "My post", "at://post1", 0, 0, 0),
		},
	}
	mock.authorFeed[0].Post.Author.Did = "did:plc:alice"

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:alice", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	updated, _ = model.Update(AuthorFeedLoadedMsg{Posts: mock.authorFeed, Cursor: ""})
	model = updated.(ProfileModel)

	updated, cmd := model.Update(tea.KeyPressMsg{Text: "d"})
	model = updated.(ProfileModel)

	if model.confirmDelete != 0 {
		t.Errorf("Expected confirmDelete=0, got %d", model.confirmDelete)
	}
	if cmd != nil {
		t.Error("Expected no command on first 'd' press")
	}

	v := model.View()
	if !strings.Contains(v.Content, "Press d to confirm delete") {
		t.Error("Expected confirmation prompt in view")
	}
}

func TestProfileDeleteSecondDPress(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{
			makeTestPost("alice.bsky.social", "Alice", "My post", "at://post1", 0, 0, 0),
		},
	}
	mock.authorFeed[0].Post.Author.Did = "did:plc:alice"

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:alice", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	updated, _ = model.Update(AuthorFeedLoadedMsg{Posts: mock.authorFeed, Cursor: ""})
	model = updated.(ProfileModel)

	updated, _ = model.Update(tea.KeyPressMsg{Text: "d"})
	model = updated.(ProfileModel)

	updated, cmd := model.Update(tea.KeyPressMsg{Text: "d"})
	model = updated.(ProfileModel)

	if cmd == nil {
		t.Fatal("Expected command on second 'd' press")
	}
	msg := cmd()
	deleteMsg, ok := msg.(feed.DeletePostMsg)
	if !ok {
		t.Fatalf("Expected DeletePostMsg, got %T", msg)
	}
	if deleteMsg.URI != "at://post1" {
		t.Errorf("Expected URI 'at://post1', got %q", deleteMsg.URI)
	}
}

func TestProfileDeleteNotOwnPost(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{
			makeTestPost("alice.bsky.social", "Alice", "Not mine", "at://post1", 0, 0, 0),
		},
	}
	mock.authorFeed[0].Post.Author.Did = "did:plc:other"

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:me", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	updated, _ = model.Update(AuthorFeedLoadedMsg{Posts: mock.authorFeed, Cursor: ""})
	model = updated.(ProfileModel)

	updated, cmd := model.Update(tea.KeyPressMsg{Text: "d"})
	model = updated.(ProfileModel)

	if model.confirmDelete != -1 {
		t.Errorf("Expected confirmDelete=-1 for non-own post, got %d", model.confirmDelete)
	}
	if cmd != nil {
		t.Error("Expected no command for non-own post delete")
	}
}

func TestProfileDeletePostResultRemovesPost(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:alice", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	model.authorFeed = []*bsky.FeedDefs_FeedViewPost{
		makeTestPost("alice.bsky.social", "Alice", "Post 1", "at://post1", 0, 0, 0),
		makeTestPost("alice.bsky.social", "Alice", "Post 2", "at://post2", 0, 0, 0),
	}
	model.selectedIndex = 1

	updated, _ = model.Update(feed.DeletePostResultMsg{URI: "at://post2", Err: nil})
	model = updated.(ProfileModel)

	if len(model.authorFeed) != 1 {
		t.Errorf("Expected 1 post after delete, got %d", len(model.authorFeed))
	}
	if model.selectedIndex != 0 {
		t.Errorf("Expected selectedIndex adjusted to 0, got %d", model.selectedIndex)
	}
}

func TestProfileMouseWheelDownScrolls(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
	}
	posts := make([]*bsky.FeedDefs_FeedViewPost, 10)
	for i := range 10 {
		posts[i] = makeTestPost("alice.bsky.social", "Alice", "Post", "at://post"+string(rune('0'+i)), 0, 0, 0)
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	updated, _ = model.Update(AuthorFeedLoadedMsg{Posts: posts, Cursor: ""})
	model = updated.(ProfileModel)

	updated, _ = model.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	model = updated.(ProfileModel)

	if model.selectedIndex != 3 {
		t.Errorf("Expected selectedIndex=3 after mouse wheel down, got %d", model.selectedIndex)
	}
}

func TestProfileMouseWheelUpScrolls(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile: makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
	}
	posts := make([]*bsky.FeedDefs_FeedViewPost, 10)
	for i := range 10 {
		posts[i] = makeTestPost("alice.bsky.social", "Alice", "Post", "at://post"+string(rune('0'+i)), 0, 0, 0)
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)
	updated, _ = model.Update(AuthorFeedLoadedMsg{Posts: posts, Cursor: ""})
	model = updated.(ProfileModel)
	model.selectedIndex = 5

	updated, _ = model.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	model = updated.(ProfileModel)

	if model.selectedIndex != 2 {
		t.Errorf("Expected selectedIndex=2 after mouse wheel up from 5, got %d", model.selectedIndex)
	}
}

func TestProfileSpinnerTickDuringLoad(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}
	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)

	_, cmd := model.Update(model.spinner.Tick())
	if cmd == nil {
		t.Error("Expected spinner tick to return a command when loading")
	}
}

func TestProfileSpinnerTickIgnoredWhenNotLoading(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}
	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	model.loading = false
	model.loadingFeed = false

	_, cmd := model.Update(model.spinner.Tick())
	if cmd != nil {
		t.Error("Expected spinner tick to return nil command when not loading")
	}
}

func TestStatsLineFormat(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Bio", 1234, 5678, 90, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)

	view := model.View()
	content := view.Content

	// Check formatted numbers appear in stats
	if !strings.Contains(content, "1.2K") {
		t.Errorf("Expected stats to contain '1.2K' for followers, got: %s", content)
	}

	if !strings.Contains(content, "5.7K") {
		t.Errorf("Expected stats to contain '5.7K' for following, got: %s", content)
	}

	if !strings.Contains(content, "90") {
		t.Errorf("Expected stats to contain '90' for posts, got: %s", content)
	}
}

func TestFollowStateAfterToggle(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	updated, _ := model.Update(ProfileLoadedMsg{Profile: mock.profile})
	model = updated.(ProfileModel)

	// Check initial state shows Follow button
	view := model.View()
	if !strings.Contains(view.Content, "[Follow]") {
		t.Errorf("Expected profile to show '[Follow]' button, got: %s", view.Content)
	}

	// Simulate follow toggle message
	updated, _ = model.Update(FollowToggledMsg{Following: true})
	model = updated.(ProfileModel)

	// Check state changed to Following
	view = model.View()
	if !strings.Contains(view.Content, "[Following") {
		t.Errorf("Expected profile to show '[Following' after toggle, got: %s", view.Content)
	}

	// Simulate unfollow toggle message
	updated, _ = model.Update(FollowToggledMsg{Following: false})
	model = updated.(ProfileModel)

	// Check state changed back to Follow
	view = model.View()
	if !strings.Contains(view.Content, "[Follow]") {
		t.Errorf("Expected profile to show '[Follow]' after unfollow toggle, got: %s", view.Content)
	}
}

func TestInitReturnsBatchCommands(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)
	cmd := model.Init()

	if cmd == nil {
		t.Error("Expected Init to return a batch command")
	}
}

func TestWindowSizeUpdate(t *testing.T) {
	t.Parallel()
	mock := &mockClient{
		profile:    makeTestProfile("alice.bsky.social", "Alice", "Bio", 100, 50, 25, nil),
		authorFeed: []*bsky.FeedDefs_FeedViewPost{},
	}

	model := NewProfileModel(mock, "did:plc:alice", "did:plc:bob", 80, 24)

	// Update window size
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model = updated.(ProfileModel)

	if model.width != 100 {
		t.Errorf("Expected width to be 100, got %d", model.width)
	}

	if model.height != 40 {
		t.Errorf("Expected height to be 40, got %d", model.height)
	}
}
