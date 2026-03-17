package main

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"

	"github.com/dotBeeps/noms/internal/auth"
	"github.com/dotBeeps/noms/internal/ui"
	"github.com/dotBeeps/noms/internal/ui/compose"
	"github.com/dotBeeps/noms/internal/ui/feed"
	"github.com/dotBeeps/noms/internal/ui/notifications"
	"github.com/dotBeeps/noms/internal/ui/profile"
	"github.com/dotBeeps/noms/internal/ui/search"
	"github.com/dotBeeps/noms/internal/ui/thread"
)

var integTestAnsiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripAnsiInteg(s string) string { return integTestAnsiRe.ReplaceAllString(s, "") }

type mockClient struct {
	timelinePosts []*bsky.FeedDefs_FeedViewPost
	profileData   *bsky.ActorDefs_ProfileViewDetailed
	unreadCount   int
}

func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }

func newMockClient() *mockClient {
	return &mockClient{
		timelinePosts: []*bsky.FeedDefs_FeedViewPost{
			{
				Post: &bsky.FeedDefs_PostView{
					Uri: "at://did:plc:test123/app.bsky.feed.post/abc1",
					Cid: "cid1",
					Author: &bsky.ActorDefs_ProfileViewBasic{
						Did: "did:plc:author1", Handle: "alice.bsky.social",
						DisplayName: strPtr("Alice"),
					},
					Record:      &lexutil.LexiconTypeDecoder{Val: &bsky.FeedPost{Text: "Hello, world!"}},
					LikeCount:   int64Ptr(5),
					RepostCount: int64Ptr(2),
					ReplyCount:  int64Ptr(1),
				},
			},
			{
				Post: &bsky.FeedDefs_PostView{
					Uri: "at://did:plc:test123/app.bsky.feed.post/abc2",
					Cid: "cid2",
					Author: &bsky.ActorDefs_ProfileViewBasic{
						Did: "did:plc:author2", Handle: "bob.bsky.social",
						DisplayName: strPtr("Bob"),
					},
					Record:      &lexutil.LexiconTypeDecoder{Val: &bsky.FeedPost{Text: "Second post"}},
					LikeCount:   int64Ptr(3),
					RepostCount: int64Ptr(0),
					ReplyCount:  int64Ptr(0),
				},
			},
		},
		profileData: &bsky.ActorDefs_ProfileViewDetailed{
			Did: "did:plc:test123", Handle: "test.bsky.social",
			DisplayName:    strPtr("Test User"),
			Description:    strPtr("A test profile"),
			FollowersCount: int64Ptr(100),
			FollowsCount:   int64Ptr(50),
			PostsCount:     int64Ptr(200),
		},
		unreadCount: 3,
	}
}

func (m *mockClient) GetTimeline(_ context.Context, _ string, _ int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	return m.timelinePosts, "", nil
}

func (m *mockClient) GetPost(_ context.Context, uri string) (*bsky.FeedDefs_PostView, error) {
	for _, p := range m.timelinePosts {
		if p.Post != nil && p.Post.Uri == uri {
			return p.Post, nil
		}
	}
	return &bsky.FeedDefs_PostView{
		Uri: uri, Cid: "cid-fetched",
		Author: &bsky.ActorDefs_ProfileViewBasic{Did: "did:plc:unknown", Handle: "unknown.bsky.social"},
		Record: &lexutil.LexiconTypeDecoder{Val: &bsky.FeedPost{Text: "fetched post"}},
	}, nil
}

func (m *mockClient) GetPostThread(_ context.Context, uri string, _ int) (*bsky.FeedGetPostThread_Output, error) {
	return &bsky.FeedGetPostThread_Output{
		Thread: &bsky.FeedGetPostThread_Output_Thread{
			FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
				Post: &bsky.FeedDefs_PostView{
					Uri: uri, Cid: "thread-cid",
					Author: &bsky.ActorDefs_ProfileViewBasic{Did: "did:plc:author1", Handle: "alice.bsky.social", DisplayName: strPtr("Alice")},
					Record: &lexutil.LexiconTypeDecoder{Val: &bsky.FeedPost{Text: "Thread root post"}},
				},
			},
		},
	}, nil
}

func (m *mockClient) GetProfile(_ context.Context, _ string) (*bsky.ActorDefs_ProfileViewDetailed, error) {
	return m.profileData, nil
}

func (m *mockClient) GetAuthorFeed(_ context.Context, _ string, _ string, _ int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	return m.timelinePosts, "", nil
}

func (m *mockClient) FollowActor(_ context.Context, _ string) error   { return nil }
func (m *mockClient) UnfollowActor(_ context.Context, _ string) error { return nil }

func (m *mockClient) ListNotifications(_ context.Context, _ string, _ int) ([]*bsky.NotificationListNotifications_Notification, string, error) {
	return nil, "", nil
}

func (m *mockClient) GetUnreadCount(_ context.Context) (int, error) { return m.unreadCount, nil }
func (m *mockClient) MarkNotificationsRead(_ context.Context, _ time.Time) error {
	return nil
}

func (m *mockClient) CreatePost(_ context.Context, _ string, _ []*bsky.RichtextFacet, _ *bsky.FeedPost_ReplyRef, _ *bsky.FeedPost_Embed) (string, string, error) {
	return "at://did:plc:test123/app.bsky.feed.post/new1", "new-cid", nil
}

func (m *mockClient) DeletePost(_ context.Context, _ string) error          { return nil }
func (m *mockClient) Like(_ context.Context, _, _ string) (string, error)   { return "like-uri", nil }
func (m *mockClient) Unlike(_ context.Context, _ string) error              { return nil }
func (m *mockClient) Repost(_ context.Context, _, _ string) (string, error) { return "rp-uri", nil }
func (m *mockClient) UnRepost(_ context.Context, _ string) error            { return nil }

func (m *mockClient) SearchPosts(_ context.Context, _ string, _ string, _ int) ([]*bsky.FeedDefs_PostView, string, error) {
	var results []*bsky.FeedDefs_PostView
	for _, p := range m.timelinePosts {
		results = append(results, p.Post)
	}
	return results, "", nil
}

func (m *mockClient) SearchActors(_ context.Context, _ string, _ string, _ int) ([]*bsky.ActorDefs_ProfileView, string, error) {
	return []*bsky.ActorDefs_ProfileView{
		{Did: "did:plc:test123", Handle: "test.bsky.social", DisplayName: strPtr("Test User")},
	}, "", nil
}

func newTestApp(client *mockClient) ui.App {
	app := ui.NewApp()
	updated, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = updated.(ui.App)

	session := &auth.Session{
		DID: "did:plc:test123", Handle: "test.bsky.social", PDS: "https://bsky.social",
	}
	updated, _ = app.Update(ui.LoginSuccessMsg{Session: session})
	app = updated.(ui.App)

	updated, _ = app.Update(ui.VoreskySkipMsg{})
	app = updated.(ui.App)

	app.SetClient(client)
	h := app.Height() - 2
	if h < 1 {
		h = 1
	}
	app.SetFeedModel(feed.NewFeedModel(client, "", app.Width(), h, nil))
	app.SetNotifModel(notifications.NewNotificationsModel(client, app.Width(), h, nil))
	app.SetSearchModel(search.NewSearchModel(client, app.Width(), h, nil))
	return app
}

func keyPress(key rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: key, Text: string(key)}
}

func TestLoginFlowToFeed(t *testing.T) {
	t.Parallel()
	app := ui.NewApp()
	if app.Screen() != ui.ScreenLogin {
		t.Fatalf("expected ScreenLogin, got %d", app.Screen())
	}

	session := &auth.Session{DID: "did:plc:test123", Handle: "test.bsky.social", PDS: "https://bsky.social"}
	updated, cmd := app.Update(ui.LoginSuccessMsg{Session: session})
	app = updated.(ui.App)

	if app.Screen() != ui.ScreenVoreskySetup {
		t.Fatalf("expected ScreenVoreskySetup after login, got %d", app.Screen())
	}
	if !app.IsLoggedIn() {
		t.Fatal("expected IsLoggedIn true")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from login (feed init + vsetup)")
	}

	updated, _ = app.Update(ui.VoreskySkipMsg{})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenFeed {
		t.Fatalf("expected ScreenFeed after voresky skip, got %d", app.Screen())
	}
}

func TestLoginAndFeedLoad(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(feed.FeedLoadedMsg{Posts: client.timelinePosts, Cursor: ""})
	app = updated.(ui.App)

	v := app.View()
	if !strings.Contains(stripAnsiInteg(v.Content), "[1]") {
		t.Errorf("expected tab bar with [1] in view")
	}
}

func TestTabSwitching(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	tests := []struct {
		key    rune
		screen ui.Screen
	}{
		{'2', ui.ScreenNotifications},
		{'1', ui.ScreenFeed},
		{'3', ui.ScreenProfile},
		{'4', ui.ScreenSearch},
		{'1', ui.ScreenFeed},
	}

	for _, tt := range tests {
		updated, _ := app.Update(keyPress(tt.key))
		app = updated.(ui.App)
		if app.Screen() != tt.screen {
			t.Errorf("key %c: expected screen %d, got %d", tt.key, tt.screen, app.Screen())
		}
	}
}

func TestFeedToThreadNavigation(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(feed.FeedLoadedMsg{Posts: client.timelinePosts, Cursor: ""})
	app = updated.(ui.App)

	updated, cmd := app.Update(feed.ViewThreadMsg{URI: client.timelinePosts[0].Post.Uri})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenThread {
		t.Fatalf("expected ScreenThread, got %d", app.Screen())
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from thread navigation")
	}

	updated, _ = app.Update(thread.BackMsg{})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenFeed {
		t.Errorf("expected ScreenFeed after back, got %d", app.Screen())
	}
}

func TestThreadBackPreservesPrevScreen(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(keyPress('2'))
	app = updated.(ui.App)

	updated, _ = app.Update(notifications.NavigateToPostMsg{URI: "at://test/post/1"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenThread {
		t.Fatalf("expected ScreenThread, got %d", app.Screen())
	}

	updated, _ = app.Update(thread.BackMsg{})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenNotifications {
		t.Errorf("expected ScreenNotifications after back, got %d", app.Screen())
	}
}

func TestComposeNewPost(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, cmd := app.Update(feed.ComposeMsg{})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenCompose {
		t.Fatalf("expected ScreenCompose, got %d", app.Screen())
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from compose init")
	}

	updated, _ = app.Update(compose.PostCreatedMsg{URI: "at://test/post/new", CID: "newcid"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenFeed {
		t.Errorf("expected ScreenFeed after post created, got %d", app.Screen())
	}
}

func TestComposeCancel(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(feed.ComposeMsg{})
	app = updated.(ui.App)

	updated, _ = app.Update(compose.CancelComposeMsg{})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenFeed {
		t.Errorf("expected ScreenFeed after cancel, got %d", app.Screen())
	}
}

func TestNotificationNavigateToPost(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, cmd := app.Update(notifications.NavigateToPostMsg{URI: "at://test/post/notif"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenThread {
		t.Fatalf("expected ScreenThread, got %d", app.Screen())
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
}

func TestNotificationNavigateToProfile(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, cmd := app.Update(notifications.NavigateToProfileMsg{DID: "did:plc:author1"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenProfile {
		t.Fatalf("expected ScreenProfile, got %d", app.Screen())
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
}

func TestSearchViewThread(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(keyPress('4'))
	app = updated.(ui.App)

	updated, cmd := app.Update(feed.ViewThreadMsg{URI: "at://test/post/search1"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenThread {
		t.Fatalf("expected ScreenThread, got %d", app.Screen())
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	updated, _ = app.Update(thread.BackMsg{})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenSearch {
		t.Errorf("expected ScreenSearch after back, got %d", app.Screen())
	}
}

func TestSearchProfileNavigation(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, cmd := app.Update(search.ViewProfileMsg{DID: "did:plc:author1"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenProfile {
		t.Fatalf("expected ScreenProfile, got %d", app.Screen())
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
}

func TestProfileBack(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(notifications.NavigateToProfileMsg{DID: "did:plc:author1"})
	app = updated.(ui.App)

	updated, _ = app.Update(profile.BackMsg{})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenFeed {
		t.Errorf("expected ScreenFeed after back, got %d", app.Screen())
	}
}

func TestProfileViewThread(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(notifications.NavigateToProfileMsg{DID: "did:plc:author1"})
	app = updated.(ui.App)

	updated, cmd := app.Update(profile.ViewThreadMsg{URI: "at://test/post/fromprofile"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenThread {
		t.Fatalf("expected ScreenThread, got %d", app.Screen())
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
}

func TestUnreadCountUpdatesStatusBar(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(notifications.UnreadCountMsg{Count: 7})
	app = updated.(ui.App)

	v := app.View()
	if !strings.Contains(v.Content, "7") {
		t.Errorf("expected status bar to show unread count 7")
	}
}

func TestThreadComposeReply(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(feed.ViewThreadMsg{URI: "at://test/post/1"})
	app = updated.(ui.App)

	updated, _ = app.Update(thread.ComposeReplyMsg{URI: "at://test/post/1", CID: "cid1"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenCompose {
		t.Errorf("expected ScreenCompose, got %d", app.Screen())
	}
}

func TestThreadViewProfile(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(feed.ViewThreadMsg{URI: "at://test/post/1"})
	app = updated.(ui.App)

	updated, cmd := app.Update(thread.ViewProfileMsg{DID: "did:plc:author1"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenProfile {
		t.Fatalf("expected ScreenProfile, got %d", app.Screen())
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
}

func TestFeedComposeReply(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(feed.ComposeReplyMsg{URI: "at://test/post/1", CID: "cid1"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenCompose {
		t.Errorf("expected ScreenCompose, got %d", app.Screen())
	}
}

func TestHelpToggle(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	if app.ShowHelp() {
		t.Fatal("expected help hidden initially")
	}

	updated, _ := app.Update(keyPress('?'))
	app = updated.(ui.App)
	if !app.ShowHelp() {
		t.Error("expected help visible after ?")
	}

	updated, _ = app.Update(keyPress('?'))
	app = updated.(ui.App)
	if app.ShowHelp() {
		t.Error("expected help hidden after second ?")
	}
}

func TestCtrlCQuits(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	_, cmd := app.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for ctrl+c")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", msg)
	}
}

func TestFeedActionDelegation(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	msgs := []tea.Msg{
		feed.FeedLoadedMsg{Posts: client.timelinePosts, Cursor: ""},
		feed.FeedErrorMsg{Err: nil},
		feed.FeedRefreshMsg{},
		feed.LikeResultMsg{PostURI: "at://test/post/1", LikeURI: "like1"},
		feed.UnlikeResultMsg{PostURI: "at://test/post/1"},
		feed.RepostResultMsg{PostURI: "at://test/post/1", RepostURI: "rp1"},
		feed.UnRepostResultMsg{PostURI: "at://test/post/1"},
	}
	for _, msg := range msgs {
		updated, _ := app.Update(msg)
		app = updated.(ui.App)
	}
	if app.Screen() != ui.ScreenFeed {
		t.Errorf("expected ScreenFeed, got %d", app.Screen())
	}
}

func TestSearchResults(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(keyPress('4'))
	app = updated.(ui.App)

	updated, _ = app.Update(search.SearchResultsMsg{Posts: []*bsky.FeedDefs_PostView{client.timelinePosts[0].Post}, Cursor: "", Mode: 0})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenSearch {
		t.Errorf("expected ScreenSearch, got %d", app.Screen())
	}
}

func TestWindowResize(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = updated.(ui.App)
	if app.Width() != 120 || app.Height() != 40 {
		t.Errorf("expected 120x40, got %dx%d", app.Width(), app.Height())
	}
}

func TestComposeError(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(feed.ComposeMsg{})
	app = updated.(ui.App)

	updated, _ = app.Update(compose.ComposeErrorMsg{Err: nil})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenCompose {
		t.Errorf("expected ScreenCompose, got %d", app.Screen())
	}
}

func TestSelfProfileTab(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, cmd := app.Update(keyPress('3'))
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenProfile {
		t.Fatalf("expected ScreenProfile, got %d", app.Screen())
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd for self-profile init")
	}
}

func TestVersionFlag(t *testing.T) {
	t.Parallel()
	if Version != "0.1.0-dev" {
		t.Errorf("expected version 0.1.0-dev, got %s", Version)
	}
}

func TestEscFromThread(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(feed.ViewThreadMsg{URI: "at://test/post/1"})
	app = updated.(ui.App)

	updated, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenFeed {
		t.Errorf("expected ScreenFeed after esc, got %d", app.Screen())
	}
}

func TestPostCreatedRefreshesFeed(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(feed.ComposeMsg{})
	app = updated.(ui.App)

	updated, cmd := app.Update(compose.PostCreatedMsg{URI: "at://test/post/new", CID: "newcid"})
	app = updated.(ui.App)
	if app.Screen() != ui.ScreenFeed {
		t.Fatalf("expected ScreenFeed, got %d", app.Screen())
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd for feed refresh")
	}
}

func TestProfileLoadAndError(t *testing.T) {
	t.Parallel()
	client := newMockClient()
	app := newTestApp(client)

	updated, _ := app.Update(notifications.NavigateToProfileMsg{DID: "did:plc:author1"})
	app = updated.(ui.App)

	updated, _ = app.Update(profile.ProfileLoadedMsg{Profile: client.profileData})
	app = updated.(ui.App)

	updated, _ = app.Update(profile.AuthorFeedLoadedMsg{Posts: client.timelinePosts, Cursor: ""})
	app = updated.(ui.App)

	if app.Screen() != ui.ScreenProfile {
		t.Errorf("expected ScreenProfile, got %d", app.Screen())
	}
}
