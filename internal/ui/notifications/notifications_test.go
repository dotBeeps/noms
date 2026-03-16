package notifications

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/dotBeeps/noms/internal/ui/shared"
)

var ansiEscapeRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsi(s string) string {
	return ansiEscapeRe.ReplaceAllString(s, "")
}

// mockBlueskyClient implements bluesky.BlueskyClient for testing
type mockBlueskyClient struct {
	notifications    []*bsky.NotificationListNotifications_Notification
	cursor           string
	unreadCount      int
	err              error
	markReadCalled   bool
	markReadCalledAt time.Time
	followErr        error
	unfollowErr      error
	likeErr          error
	unlikeErr        error
	repostErr        error
	unrepostErr      error
	deletePostErr    error
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
	return m.followErr
}

func (m *mockBlueskyClient) UnfollowActor(ctx context.Context, did string) error {
	return m.unfollowErr
}

func (m *mockBlueskyClient) ListNotifications(ctx context.Context, cursor string, limit int) ([]*bsky.NotificationListNotifications_Notification, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return m.notifications, m.cursor, nil
}

func (m *mockBlueskyClient) GetUnreadCount(ctx context.Context) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.unreadCount, nil
}

func (m *mockBlueskyClient) MarkNotificationsRead(ctx context.Context, seenAt time.Time) error {
	m.markReadCalled = true
	m.markReadCalledAt = seenAt
	return nil
}

func (m *mockBlueskyClient) CreatePost(ctx context.Context, text string, facets []*bsky.RichtextFacet, reply *bsky.FeedPost_ReplyRef, embed *bsky.FeedPost_Embed) (string, string, error) {
	return "", "", nil
}

func (m *mockBlueskyClient) DeletePost(ctx context.Context, uri string) error {
	return m.deletePostErr
}

func (m *mockBlueskyClient) Like(ctx context.Context, uri, cid string) (string, error) {
	return "", m.likeErr
}

func (m *mockBlueskyClient) Unlike(ctx context.Context, likeURI string) error {
	return m.unlikeErr
}

func (m *mockBlueskyClient) Repost(ctx context.Context, uri, cid string) (string, error) {
	return "", m.repostErr
}

func (m *mockBlueskyClient) UnRepost(ctx context.Context, repostURI string) error {
	return m.unrepostErr
}

func (m *mockBlueskyClient) SearchPosts(ctx context.Context, query string, cursor string, limit int) ([]*bsky.FeedDefs_PostView, string, error) {
	return nil, "", nil
}

func (m *mockBlueskyClient) SearchActors(ctx context.Context, query string, cursor string, limit int) ([]*bsky.ActorDefs_ProfileView, string, error) {
	return nil, "", nil
}

// Helper functions to create test notifications
func strPtr(s string) *string {
	return &s
}

func createTestNotification(reason, handle, did string, isRead bool, indexedAt string) *bsky.NotificationListNotifications_Notification {
	return &bsky.NotificationListNotifications_Notification{
		Reason:    reason,
		IsRead:    isRead,
		IndexedAt: indexedAt,
		Author: &bsky.ActorDefs_ProfileView{
			Handle:      handle,
			Did:         did,
			DisplayName: strPtr(handle + " Display Name"),
		},
		Uri:           "at://did:plc:test/app.bsky.feed.post/test123",
		ReasonSubject: strPtr("at://did:plc:test/app.bsky.feed.post/subject123"),
	}
}

func createTestNotificationWithRecord(reason, handle, did string, isRead bool, indexedAt string, recordText string) *bsky.NotificationListNotifications_Notification {
	notif := createTestNotification(reason, handle, did, isRead, indexedAt)
	notif.Record = &lexutil.LexiconTypeDecoder{
		Val: &bsky.FeedPost{
			Text:      recordText,
			CreatedAt: indexedAt,
		},
	}
	return notif
}

func TestRenderLikeNotification(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("like", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-5*time.Minute).Format(time.RFC3339)),
		},
		unreadCount: 1,
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "♡") {
		t.Error("Expected like icon ♡ in notification view")
	}
	if !strings.Contains(content, "liked your post") {
		t.Error("Expected 'liked your post' text in notification view")
	}
	if !strings.Contains(content, "alice.bsky.social") {
		t.Error("Expected author handle in notification view")
	}
}

func TestRenderRepostNotification(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("repost", "bob.bsky.social", "did:plc:bob", true, time.Now().Add(-1*time.Hour).Format(time.RFC3339)),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "⟲") {
		t.Error("Expected repost icon ⟲ in notification view")
	}
	if !strings.Contains(content, "reposted your post") {
		t.Error("Expected 'reposted your post' text in notification view")
	}
}

func TestRenderFollowNotification(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("follow", "charlie.bsky.social", "did:plc:charlie", false, time.Now().Add(-2*time.Hour).Format(time.RFC3339)),
		},
		unreadCount: 1,
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "+") {
		t.Error("Expected follow icon + in notification view")
	}
	if !strings.Contains(content, "followed you") {
		t.Error("Expected 'followed you' text in notification view")
	}
}

func TestRenderMentionNotification(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("mention", "dave.bsky.social", "did:plc:dave", false, time.Now().Add(-30*time.Minute).Format(time.RFC3339)),
		},
		unreadCount: 1,
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "@") {
		t.Error("Expected mention icon @ in notification view")
	}
	if !strings.Contains(content, "mentioned you") {
		t.Error("Expected 'mentioned you' text in notification view")
	}
}

func TestRenderReplyNotification(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotificationWithRecord("reply", "eve.bsky.social", "did:plc:eve", false, time.Now().Add(-15*time.Minute).Format(time.RFC3339), "This is a reply to your post"),
		},
		unreadCount: 1,
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "💬") {
		t.Error("Expected reply icon 💬 in notification view")
	}
	if !strings.Contains(content, "replied to your post") {
		t.Error("Expected 'replied to your post' text in notification view")
	}
}

func TestRenderQuoteNotification(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("quote", "frank.bsky.social", "did:plc:frank", true, time.Now().Add(-3*time.Hour).Format(time.RFC3339)),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "❝") {
		t.Error("Expected quote icon ❝ in notification view")
	}
	if !strings.Contains(content, "quoted your post") {
		t.Error("Expected 'quoted your post' text in notification view")
	}
}

func TestUnreadIndicator(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("like", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-5*time.Minute).Format(time.RFC3339)),
			createTestNotification("like", "bob.bsky.social", "did:plc:bob", true, time.Now().Add(-10*time.Minute).Format(time.RFC3339)),
		},
		unreadCount: 1,
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	// Should have unread dot for first notification
	if !strings.Contains(content, "●") {
		t.Error("Expected unread indicator ● in notification view for unread notification")
	}
}

func TestMarkAsRead(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("like", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-5*time.Minute).Format(time.RFC3339)),
		},
		unreadCount: 1,
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	// Press 'r' to refresh and mark as read
	updated, cmd := m.Update(tea.KeyPressMsg{})
	// Need to properly create key press for 'r'
	_ = cmd
	_ = updated

	// Verify markReadCalled would be set by executing the command
	// In a real scenario, the command would be executed
}

func TestNotificationNavigationToPost(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("like", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-5*time.Minute).Format(time.RFC3339)),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	// Press enter to navigate
	updated, cmd := m.Update(tea.KeyPressMsg{})
	m = updated.(NotificationsModel)

	_ = m
	_ = cmd
	// The cmd should emit NavigateToPostMsg
	// In the actual test, we would execute the cmd and check the message
}

func TestNotificationNavigationToProfile(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("follow", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-5*time.Minute).Format(time.RFC3339)),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	// Press enter on follow notification
	updated, cmd := m.Update(tea.KeyPressMsg{})
	m = updated.(NotificationsModel)

	_ = m
	_ = cmd
	// The cmd should emit NavigateToProfileMsg
}

func TestEmptyNotifications(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{},
		unreadCount:   0,
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "No notifications") {
		t.Errorf("Expected empty state message, got: %s", content)
	}
}

func TestNotificationPagination(t *testing.T) {
	t.Parallel()
	// Create 5 notifications for first page
	notifs1 := make([]*bsky.NotificationListNotifications_Notification, 5)
	for i := 0; i < 5; i++ {
		notifs1[i] = createTestNotification("like", "user.bsky.social", "did:plc:user", false, time.Now().Add(-time.Duration(i)*time.Minute).Format(time.RFC3339))
	}

	mockClient := &mockBlueskyClient{
		notifications: notifs1,
		cursor:        "nextpage",
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	m.cursor = "nextpage"

	// Load first batch
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: notifs1,
		Cursor:        "nextpage",
	})
	m = updated.(NotificationsModel)

	if len(m.notifications) != 5 {
		t.Errorf("Expected 5 notifications, got %d", len(m.notifications))
	}

	if m.cursor != "nextpage" {
		t.Errorf("Expected cursor 'nextpage', got %s", m.cursor)
	}
}

func TestNotificationSelection(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("like", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-5*time.Minute).Format(time.RFC3339)),
			createTestNotification("like", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-10*time.Minute).Format(time.RFC3339)),
			createTestNotification("like", "charlie.bsky.social", "did:plc:charlie", false, time.Now().Add(-15*time.Minute).Format(time.RFC3339)),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	if m.viewport.SelectedIndex() != 0 {
		t.Errorf("Expected initial selection at 0, got %d", m.viewport.SelectedIndex())
	}

	// Press 'j' to move down
	updated, _ = m.Update(tea.KeyPressMsg{})
	m = updated.(NotificationsModel)
	// This won't actually change selection because we need to properly simulate key press
	// The key handling is done via msg.String() which needs proper key

	_ = m
}

func TestNotificationsLoadedMsg(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("like", "test.bsky.social", "did:plc:test", false, time.Now().Format(time.RFC3339)),
		},
		cursor: "testcursor",
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)

	// Simulate receiving NotificationsLoadedMsg
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
		Cursor:        "testcursor",
	})
	m = updated.(NotificationsModel)

	if len(m.notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(m.notifications))
	}

	if m.cursor != "testcursor" {
		t.Errorf("Expected cursor 'testcursor', got %s", m.cursor)
	}
}

func TestNotificationsErrorMsg(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		err: errors.New("network error"),
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)

	// Simulate receiving NotificationsErrorMsg
	testErr := errors.New("failed to load notifications")
	updated, _ := m.Update(NotificationsErrorMsg{Err: testErr})
	m = updated.(NotificationsModel)

	if m.err == nil {
		t.Error("Expected error to be set")
	}

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "Error:") {
		t.Errorf("Expected error message in view, got: %s", content)
	}
}

func TestUnreadCountMsg(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		unreadCount: 5,
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)

	// Simulate receiving UnreadCountMsg
	updated, _ := m.Update(UnreadCountMsg{Count: 5})
	m = updated.(NotificationsModel)

	if m.unreadCount != 5 {
		t.Errorf("Expected unreadCount 5, got %d", m.unreadCount)
	}

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "5 unread") {
		t.Errorf("Expected '5 unread' in header, got: %s", content)
	}
}

func TestWindowSizeMsg(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(NotificationsModel)

	if m.width != 100 {
		t.Errorf("Expected width 100, got %d", m.width)
	}

	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}

func TestContentPreview(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotificationWithRecord("reply", "test.bsky.social", "did:plc:test", false, time.Now().Format(time.RFC3339), "This is a very long reply that should be truncated to approximately fifty characters"),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: mockClient.notifications,
	})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	// Should contain the preview (possibly truncated)
	if !strings.Contains(content, "This is") {
		t.Error("Expected content preview in notification view")
	}
}

func TestTruncateText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 50, "short"},
		{"this is a very long text that should be truncated", 20, "this is a very long …"},
		{"exactly50chars_exactly50chars_exactly50char", 50, "exactly50chars_exactly50chars_exactly50char"},
	}

	for _, tc := range tests {
		result := shared.TruncateStr(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("shared.TruncateStr(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
		}
	}
}

func TestGetNotificationStyle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		reason       string
		expectedIcon string
		expectedText string
	}{
		{"like", "♡", "liked your post"},
		{"repost", "⟲", "reposted your post"},
		{"follow", "+", "followed you"},
		{"mention", "@", "mentioned you"},
		{"reply", "💬", "replied to your post"},
		{"quote", "❝", "quoted your post"},
		{"like-via-repost", "♡", "liked your repost"},
		{"repost-via-repost", "⟲", "reposted your repost"},
		{"starterpack-joined", "+", "joined your starter pack"},
		{"verified", "✓", "verified you"},
		{"unverified", "✗", "removed your verification"},
		{"subscribed-post", "🔔", "subscribed post was updated"},
	}

	for _, tc := range tests {
		icon, text, _ := getNotificationStyle(tc.reason)
		if icon != tc.expectedIcon {
			t.Errorf("getNotificationStyle(%q) icon = %q, want %q", tc.reason, icon, tc.expectedIcon)
		}
		if text != tc.expectedText {
			t.Errorf("getNotificationStyle(%q) text = %q, want %q", tc.reason, text, tc.expectedText)
		}
	}
}

func TestNotificationMouseWheelDownScrolls(t *testing.T) {
	t.Parallel()
	notifs := make([]*bsky.NotificationListNotifications_Notification, 10)
	for i := range 10 {
		notifs[i] = createTestNotification("like", "user.bsky.social", "did:plc:user", false, time.Now().Add(-time.Duration(i)*time.Minute).Format(time.RFC3339))
		notifs[i].ReasonSubject = strPtr("at://subject" + string(rune('0'+i)))
	}

	mockClient := &mockBlueskyClient{notifications: notifs}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	updated, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = updated.(NotificationsModel)

	if m.viewport.SelectedIndex() != 3 {
		t.Errorf("Expected selected=3 after mouse wheel down, got %d", m.viewport.SelectedIndex())
	}
}

func TestNotificationMouseWheelUpScrolls(t *testing.T) {
	t.Parallel()
	notifs := make([]*bsky.NotificationListNotifications_Notification, 10)
	for i := range 10 {
		notifs[i] = createTestNotification("like", "user.bsky.social", "did:plc:user", false, time.Now().Add(-time.Duration(i)*time.Minute).Format(time.RFC3339))
		notifs[i].ReasonSubject = strPtr("at://subject" + string(rune('0'+i)))
	}

	mockClient := &mockBlueskyClient{notifications: notifs}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)
	m.viewport.SetSelectedIndex(6)

	updated, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = updated.(NotificationsModel)

	if m.viewport.SelectedIndex() != 3 {
		t.Errorf("Expected selected=3 after mouse wheel up from 6, got %d", m.viewport.SelectedIndex())
	}
}

func TestNotificationMouseWheelBoundsCheck(t *testing.T) {
	t.Parallel()
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("like", "a.bsky.social", "did:plc:a", false, time.Now().Format(time.RFC3339)),
		createTestNotification("follow", "b.bsky.social", "did:plc:b", false, time.Now().Format(time.RFC3339)),
	}

	mockClient := &mockBlueskyClient{notifications: notifs}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	updated, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = updated.(NotificationsModel)
	if m.viewport.SelectedIndex() != 1 {
		t.Errorf("Expected selected capped at 1, got %d", m.viewport.SelectedIndex())
	}

	m.viewport.SetSelectedIndex(0)
	updated, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = updated.(NotificationsModel)
	if m.viewport.SelectedIndex() != 0 {
		t.Errorf("Expected selected to stay at 0, got %d", m.viewport.SelectedIndex())
	}
}

func TestNotificationSpinnerTickDuringLoad(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)

	_, cmd := m.Update(m.spinner.Tick())
	if cmd == nil {
		t.Error("Expected spinner tick to return a command when loading")
	}
}

func TestNotificationSpinnerTickIgnoredWhenNotLoading(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	m.loading = false

	_, cmd := m.Update(m.spinner.Tick())
	if cmd != nil {
		t.Error("Expected spinner tick to return nil command when not loading")
	}
}

func TestNotificationLoadingViewShowsSpinner(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)

	v := m.View()
	if !strings.Contains(v.Content, "Loading") {
		t.Error("Expected loading text in view while loading")
	}
}

func TestInitReturnsCommands(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{},
		unreadCount:   0,
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	cmd := m.Init()

	if cmd == nil {
		t.Error("Expected Init() to return non-nil command")
	}
}

func TestGroupingLikesSamePost(t *testing.T) {
	t.Parallel()
	subject := "at://did:plc:author/app.bsky.feed.post/target"
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("like", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("like", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
		createTestNotification("like", "charlie.bsky.social", "did:plc:charlie", false, time.Now().Add(-3*time.Minute).Format(time.RFC3339)),
	}
	for _, n := range notifs {
		n.ReasonSubject = &subject
	}

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	if len(m.groups) != 1 {
		t.Fatalf("Expected 1 group for 3 likes on same post, got %d", len(m.groups))
	}
	if m.groups[0].count != 3 {
		t.Errorf("Expected group count 3, got %d", m.groups[0].count)
	}
	if len(m.groups[0].authors) != 3 {
		t.Errorf("Expected 3 authors in group, got %d", len(m.groups[0].authors))
	}
}

func TestGroupingRepostsSamePost(t *testing.T) {
	t.Parallel()
	subject := "at://did:plc:author/app.bsky.feed.post/target"
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("repost", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("repost", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
	}
	for _, n := range notifs {
		n.ReasonSubject = &subject
	}

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	if len(m.groups) != 1 {
		t.Fatalf("Expected 1 group for 2 reposts on same post, got %d", len(m.groups))
	}
	if m.groups[0].count != 2 {
		t.Errorf("Expected group count 2, got %d", m.groups[0].count)
	}
	if m.groups[0].reason != ReasonRepost {
		t.Errorf("Expected reason 'repost', got %q", m.groups[0].reason)
	}
}

func TestGroupingMixedTypes(t *testing.T) {
	t.Parallel()
	subject := "at://did:plc:author/app.bsky.feed.post/target"
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("like", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("like", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
		createTestNotification("follow", "charlie.bsky.social", "did:plc:charlie", false, time.Now().Add(-3*time.Minute).Format(time.RFC3339)),
	}
	notifs[0].ReasonSubject = &subject
	notifs[1].ReasonSubject = &subject

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	if len(m.groups) != 2 {
		t.Fatalf("Expected 2 groups (likes grouped + follow separate), got %d", len(m.groups))
	}
	if m.groups[0].reason != ReasonLike {
		t.Errorf("Expected first group reason 'like', got %q", m.groups[0].reason)
	}
	if m.groups[0].count != 2 {
		t.Errorf("Expected like group count 2, got %d", m.groups[0].count)
	}
	if m.groups[1].reason != ReasonFollow {
		t.Errorf("Expected second group reason 'follow', got %q", m.groups[1].reason)
	}
	if m.groups[1].count != 1 {
		t.Errorf("Expected follow group count 1, got %d", m.groups[1].count)
	}
}

func TestGroupingIndividualNotifications(t *testing.T) {
	t.Parallel()
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("follow", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("mention", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
		createTestNotification("reply", "charlie.bsky.social", "did:plc:charlie", false, time.Now().Add(-3*time.Minute).Format(time.RFC3339)),
	}

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	if len(m.groups) != 3 {
		t.Errorf("Expected 3 individual groups for follow/mention/reply, got %d", len(m.groups))
	}
	for i, g := range m.groups {
		if g.count != 1 {
			t.Errorf("Group %d: expected count 1, got %d", i, g.count)
		}
	}
}

func TestGroupingPreservesOrder(t *testing.T) {
	t.Parallel()
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("follow", "first.bsky.social", "did:plc:first", false, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("mention", "second.bsky.social", "did:plc:second", false, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
		createTestNotification("reply", "third.bsky.social", "did:plc:third", false, time.Now().Add(-3*time.Minute).Format(time.RFC3339)),
	}

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	if len(m.groups) != 3 {
		t.Fatalf("Expected 3 groups, got %d", len(m.groups))
	}
	if m.groups[0].reason != ReasonFollow {
		t.Errorf("Expected first group to be follow, got %q", m.groups[0].reason)
	}
	if m.groups[1].reason != ReasonMention {
		t.Errorf("Expected second group to be mention, got %q", m.groups[1].reason)
	}
	if m.groups[2].reason != ReasonReply {
		t.Errorf("Expected third group to be reply, got %q", m.groups[2].reason)
	}
}

func TestGroupingAllRead(t *testing.T) {
	t.Parallel()
	subject := "at://did:plc:author/app.bsky.feed.post/target"
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("like", "alice.bsky.social", "did:plc:alice", true, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("like", "bob.bsky.social", "did:plc:bob", true, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
	}
	for _, n := range notifs {
		n.ReasonSubject = &subject
	}

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	if len(m.groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(m.groups))
	}
	if !m.groups[0].allRead {
		t.Error("Expected group allRead to be true when all notifications are read")
	}
}

func TestGroupingPartiallyRead(t *testing.T) {
	t.Parallel()
	subject := "at://did:plc:author/app.bsky.feed.post/target"
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("like", "alice.bsky.social", "did:plc:alice", true, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("like", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
	}
	for _, n := range notifs {
		n.ReasonSubject = &subject
	}

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	if len(m.groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(m.groups))
	}
	if m.groups[0].allRead {
		t.Error("Expected group allRead to be false when some notifications are unread")
	}
}

func TestGroupingEmptyList(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{
		Notifications: []*bsky.NotificationListNotifications_Notification{},
	})
	m = updated.(NotificationsModel)

	if len(m.groups) != 0 {
		t.Errorf("Expected 0 groups for empty notification list, got %d", len(m.groups))
	}
}

func TestGroupRenderOutput(t *testing.T) {
	t.Parallel()
	subject := "at://did:plc:author/app.bsky.feed.post/target"
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("like", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("like", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
		createTestNotification("like", "charlie.bsky.social", "did:plc:charlie", false, time.Now().Add(-3*time.Minute).Format(time.RFC3339)),
	}
	for _, n := range notifs {
		n.ReasonSubject = &subject
	}

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "and 2 others") {
		t.Errorf("Expected 'and 2 others' in grouped notification view, got %q", content)
	}
	if !strings.Contains(content, "liked your post") {
		t.Error("Expected 'liked your post' in grouped notification view")
	}
}

func TestRenderLikeViaRepostNotification(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("like-via-repost", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-5*time.Minute).Format(time.RFC3339)),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: mockClient.notifications})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "♡") {
		t.Error("Expected like icon ♡ in like-via-repost notification view")
	}
	if !strings.Contains(content, "liked your repost") {
		t.Error("Expected 'liked your repost' text in notification view")
	}
}

func TestRenderRepostViaRepostNotification(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("repost-via-repost", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-5*time.Minute).Format(time.RFC3339)),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: mockClient.notifications})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "⟲") {
		t.Error("Expected repost icon ⟲ in repost-via-repost notification view")
	}
	if !strings.Contains(content, "reposted your repost") {
		t.Error("Expected 'reposted your repost' text in notification view")
	}
}

func TestGroupingLikeViaRepostSamePost(t *testing.T) {
	t.Parallel()
	subject := "at://did:plc:author/app.bsky.feed.post/target"
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("like-via-repost", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("like-via-repost", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
	}
	for _, n := range notifs {
		n.ReasonSubject = &subject
	}

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	if len(m.groups) != 1 {
		t.Fatalf("Expected 1 group for 2 like-via-reposts on same post, got %d", len(m.groups))
	}
	if m.groups[0].count != 2 {
		t.Errorf("Expected group count 2, got %d", m.groups[0].count)
	}
	if m.groups[0].reason != ReasonLikeViaRepost {
		t.Errorf("Expected reason 'like-via-repost', got %q", m.groups[0].reason)
	}
}

func TestGroupingRepostViaRepostSamePost(t *testing.T) {
	t.Parallel()
	subject := "at://did:plc:author/app.bsky.feed.post/target"
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("repost-via-repost", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("repost-via-repost", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
		createTestNotification("repost-via-repost", "charlie.bsky.social", "did:plc:charlie", false, time.Now().Add(-3*time.Minute).Format(time.RFC3339)),
	}
	for _, n := range notifs {
		n.ReasonSubject = &subject
	}

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	if len(m.groups) != 1 {
		t.Fatalf("Expected 1 group for 3 repost-via-reposts on same post, got %d", len(m.groups))
	}
	if m.groups[0].count != 3 {
		t.Errorf("Expected group count 3, got %d", m.groups[0].count)
	}
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
func (s *stubImageRenderer) InvalidateTransmissions()                      {}

func createTestNotificationWithAvatar(reason, handle, did string, isRead bool, indexedAt string, avatarURL string) *bsky.NotificationListNotifications_Notification {
	notif := createTestNotification(reason, handle, did, isRead, indexedAt)
	notif.Author.Avatar = strPtr(avatarURL)
	return notif
}

func TestAvatarRenderedWhenCachedAndEnabled(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: true, img: "AVATAR_IMG"}
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotificationWithAvatar("like", "alice.bsky.social", "did:plc:alice", false,
				time.Now().Add(-5*time.Minute).Format(time.RFC3339),
				"https://cdn.bsky.app/img/avatar/alice.jpg"),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, stub)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: mockClient.notifications})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "AVATAR_IMG") {
		t.Errorf("Expected cached avatar 'AVATAR_IMG' in notification view, got: %s", content)
	}
}

func TestAvatarPlaceholderWhenUncached(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: false, img: ""}
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotificationWithAvatar("like", "alice.bsky.social", "did:plc:alice", false,
				time.Now().Add(-5*time.Minute).Format(time.RFC3339),
				"https://cdn.bsky.app/img/avatar/alice.jpg"),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, stub)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: mockClient.notifications})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "[····]") {
		t.Errorf("Expected placeholder '[····]' for uncached avatar, got: %s", content)
	}
}

func TestNoAvatarWhenNilAvatarURL(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: true, img: "SHOULD_NOT_APPEAR"}
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("like", "alice.bsky.social", "did:plc:alice", false,
				time.Now().Add(-5*time.Minute).Format(time.RFC3339)),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, stub)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: mockClient.notifications})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if strings.Contains(content, "SHOULD_NOT_APPEAR") {
		t.Error("Expected no avatar image when Author.Avatar is nil")
	}
	if !strings.Contains(content, "liked your post") {
		t.Error("Expected notification action text even without an avatar")
	}
}

func TestNoAvatarWhenImageCacheNil(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotificationWithAvatar("like", "alice.bsky.social", "did:plc:alice", false,
				time.Now().Add(-5*time.Minute).Format(time.RFC3339),
				"https://cdn.bsky.app/img/avatar/alice.jpg"),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: mockClient.notifications})
	m = updated.(NotificationsModel)

	v := m.View()
	content := v.Content

	if !strings.Contains(content, "liked your post") {
		t.Errorf("Expected notification text with nil imageCache, got: %s", content)
	}
}

func TestGroupingNavigationBounds(t *testing.T) {
	t.Parallel()
	notifs := []*bsky.NotificationListNotifications_Notification{
		createTestNotification("follow", "alice.bsky.social", "did:plc:alice", false, time.Now().Add(-1*time.Minute).Format(time.RFC3339)),
		createTestNotification("mention", "bob.bsky.social", "did:plc:bob", false, time.Now().Add(-2*time.Minute).Format(time.RFC3339)),
	}

	mockClient := &mockBlueskyClient{}
	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: notifs})
	m = updated.(NotificationsModel)

	if len(m.groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(m.groups))
	}

	updated, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	m = updated.(NotificationsModel)
	if m.viewport.SelectedIndex() != 1 {
		t.Errorf("Expected selected to be 1 after j, got %d", m.viewport.SelectedIndex())
	}

	updated, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	m = updated.(NotificationsModel)
	if m.viewport.SelectedIndex() != 1 {
		t.Errorf("Expected selected to stay at 1 at end, got %d", m.viewport.SelectedIndex())
	}

	updated, _ = m.Update(tea.KeyPressMsg{Text: "k"})
	m = updated.(NotificationsModel)
	if m.viewport.SelectedIndex() != 0 {
		t.Errorf("Expected selected to be 0 after k, got %d", m.viewport.SelectedIndex())
	}

	updated, _ = m.Update(tea.KeyPressMsg{Text: "k"})
	m = updated.(NotificationsModel)
	if m.viewport.SelectedIndex() != 0 {
		t.Errorf("Expected selected to stay at 0 at start, got %d", m.viewport.SelectedIndex())
	}
}

func TestNotificationAvatarGutterIndentation(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: true, img: "AAAAAA\nBBBBBB\nCCCCCC"}
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotificationWithRecord("mention", "alice.bsky.social", "did:plc:alice", false,
				time.Now().Add(-5*time.Minute).Format(time.RFC3339),
				strings.Repeat("very long wrapped preview text ", 8)),
		},
	}
	mockClient.notifications[0].Author.Avatar = strPtr("https://cdn.bsky.app/img/avatar/alice.jpg")

	m := NewNotificationsModel(mockClient, 36, 24, stub)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: mockClient.notifications})
	m = updated.(NotificationsModel)

	rendered := stripAnsi(m.View().Content)
	lines := strings.Split(rendered, "\n")

	foundIndented := false
	for _, line := range lines {
		if !(strings.Contains(line, "5m") || strings.Contains(line, "very long")) {
			continue
		}
		idx := strings.Index(line, "5m")
		if idx == -1 {
			idx = strings.Index(line, "very long")
		}
		if idx == -1 {
			continue
		}
		if strings.Contains(line[:idx], strings.Repeat(" ", shared.AvatarCols+1)) {
			foundIndented = true
			break
		}
	}
	if !foundIndented {
		t.Fatalf("expected gutter indentation on lines beyond avatar rows; output: %q", rendered)
	}
}

func TestNotificationNoAvatarGutterAbsent(t *testing.T) {
	t.Parallel()
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotification("mention", "alice.bsky.social", "did:plc:alice", false,
				time.Now().Add(-5*time.Minute).Format(time.RFC3339)),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, nil)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: mockClient.notifications})
	m = updated.(NotificationsModel)

	rendered := stripAnsi(m.View().Content)
	for _, line := range strings.Split(rendered, "\n") {
		if !strings.Contains(line, "mentioned you") {
			continue
		}
		if strings.Contains(line, strings.Repeat(" ", shared.AvatarCols+1)+"@") || strings.Contains(line, strings.Repeat(" ", shared.AvatarCols+1)+"●") {
			t.Fatalf("unexpected avatar gutter indentation without avatar: %q", line)
		}
	}
}

func TestNotificationAvatarPresentInOutput(t *testing.T) {
	t.Parallel()
	stub := &stubImageRenderer{enabled: true, cached: true, img: "NOTIF_AVATAR"}
	mockClient := &mockBlueskyClient{
		notifications: []*bsky.NotificationListNotifications_Notification{
			createTestNotificationWithAvatar("like", "alice.bsky.social", "did:plc:alice", false,
				time.Now().Add(-5*time.Minute).Format(time.RFC3339),
				"https://cdn.bsky.app/img/avatar/alice.jpg"),
		},
	}

	m := NewNotificationsModel(mockClient, 80, 24, stub)
	updated, _ := m.Update(NotificationsLoadedMsg{Notifications: mockClient.notifications})
	m = updated.(NotificationsModel)

	content := stripAnsi(m.View().Content)
	if !strings.Contains(content, "NOTIF_AVATAR") {
		t.Fatalf("expected avatar marker in notification output, got: %q", content)
	}
}
