package thread

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

// mockClient implements bluesky.BlueskyClient
type mockClient struct {
	threadResult *bsky.FeedGetPostThread_Output
	errResult    error
}

func (m *mockClient) GetTimeline(ctx context.Context, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	return nil, "", nil
}
func (m *mockClient) GetPost(ctx context.Context, uri string) (*bsky.FeedDefs_PostView, error) {
	return nil, nil
}
func (m *mockClient) GetPostThread(ctx context.Context, uri string, depth int) (*bsky.FeedGetPostThread_Output, error) {
	return m.threadResult, m.errResult
}
func (m *mockClient) GetProfile(ctx context.Context, actor string) (*bsky.ActorDefs_ProfileViewDetailed, error) {
	return nil, nil
}
func (m *mockClient) GetAuthorFeed(ctx context.Context, actor string, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	return nil, "", nil
}
func (m *mockClient) FollowActor(ctx context.Context, did string) error   { return nil }
func (m *mockClient) UnfollowActor(ctx context.Context, did string) error { return nil }
func (m *mockClient) ListNotifications(ctx context.Context, cursor string, limit int) ([]*bsky.NotificationListNotifications_Notification, string, error) {
	return nil, "", nil
}
func (m *mockClient) GetUnreadCount(ctx context.Context) (int, error)                   { return 0, nil }
func (m *mockClient) MarkNotificationsRead(ctx context.Context, seenAt time.Time) error { return nil }
func (m *mockClient) CreatePost(ctx context.Context, text string, facets []*bsky.RichtextFacet, reply *bsky.FeedPost_ReplyRef, embed *bsky.FeedPost_Embed) (string, string, error) {
	return "", "", nil
}
func (m *mockClient) DeletePost(ctx context.Context, uri string) error            { return nil }
func (m *mockClient) Like(ctx context.Context, uri, cid string) (string, error)   { return "", nil }
func (m *mockClient) Unlike(ctx context.Context, likeURI string) error            { return nil }
func (m *mockClient) Repost(ctx context.Context, uri, cid string) (string, error) { return "", nil }
func (m *mockClient) UnRepost(ctx context.Context, repostURI string) error        { return nil }
func (m *mockClient) SearchPosts(ctx context.Context, query string, cursor string, limit int) ([]*bsky.FeedDefs_PostView, string, error) {
	return nil, "", nil
}
func (m *mockClient) SearchActors(ctx context.Context, query string, cursor string, limit int) ([]*bsky.ActorDefs_ProfileView, string, error) {
	return nil, "", nil
}

// Helper to create a dummy PostView
func makePost(uri, handle, text string) *bsky.FeedDefs_PostView {
	return &bsky.FeedDefs_PostView{
		Uri: uri,
		Cid: uri + "-cid",
		Author: &bsky.ActorDefs_ProfileViewBasic{
			Handle: handle,
			Did:    "did:plc:" + handle,
		},
		Record: &util.LexiconTypeDecoder{
			Val: &bsky.FeedPost{
				Text:      text,
				CreatedAt: time.Now().Format(time.RFC3339),
			},
		},
	}
}

func doUpdate(m ThreadModel, msg tea.Msg) ThreadModel {
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, cmd := range batch {
			if cmd != nil {
				subMsg := cmd()
				if subMsg != nil {
					m = doUpdate(m, subMsg)
				}
			}
		}
		return m
	}
	mModel, _ := m.Update(msg)
	return mModel.(ThreadModel)
}

// 1. TestThreadRenderSinglePost
func TestThreadRenderSinglePost(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://uri1", "user1", "Target Post"),
				},
			},
		},
	}
	m := NewThreadModel(client, "at://uri1", "", 80, 24, nil)

	cmd := m.Init()
	msg := cmd()
	m = doUpdate(m, msg)

	view := m.View().Content
	if !strings.Contains(view, "Target Post") {
		t.Errorf("Expected view to contain Target Post, got:\n%s", view)
	}
}

// 2. TestThreadRenderWithParent
func TestThreadRenderWithParent(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user2", "Target Post"),
					Parent: &bsky.FeedDefs_ThreadViewPost_Parent{
						FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
							Post: makePost("at://parent1", "user1", "Parent Post"),
						},
					},
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	view := m.View().Content
	if !strings.Contains(view, "Parent Post") || !strings.Contains(view, "Target Post") {
		t.Errorf("Missing content in view")
	}
	if !strings.Contains(view, "│") {
		t.Errorf("Expected parent to be prefixed with │")
	}
}

// 3. TestThreadRenderDeepParentChain
func TestThreadRenderDeepParentChain(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user", "Target"),
					Parent: &bsky.FeedDefs_ThreadViewPost_Parent{
						FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
							Post: makePost("at://parent1", "user", "Parent 1"),
							Parent: &bsky.FeedDefs_ThreadViewPost_Parent{
								FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
									Post: makePost("at://parent2", "user", "Parent 2"),
								},
							},
						},
					},
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	if len(m.threadPosts) != 3 {
		t.Fatalf("Expected 3 posts, got %d", len(m.threadPosts))
	}
	if m.threadPosts[0].Post.Uri != "at://parent2" {
		t.Errorf("Expected top parent to be parent2")
	}
}

// 4. TestThreadRenderReplies
func TestThreadRenderReplies(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user", "Target"),
					Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
						{
							FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
								Post: makePost("at://reply1", "user2", "Reply 1"),
							},
						},
					},
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	if len(m.threadPosts) != 2 {
		t.Fatalf("Expected 2 posts, got %d", len(m.threadPosts))
	}
	if m.threadPosts[1].Depth != 1 {
		t.Errorf("Expected reply to have depth 1, got %d", m.threadPosts[1].Depth)
	}
}

// 5. TestThreadRenderNestedReplies
func TestThreadRenderNestedReplies(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user", "Target"),
					Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
						{
							FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
								Post: makePost("at://reply1", "user", "Reply 1"),
								Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
									{
										FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
											Post: makePost("at://reply2", "user", "Nested Reply"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	if len(m.threadPosts) != 3 {
		t.Fatalf("Expected 3 posts, got %d", len(m.threadPosts))
	}
	if m.threadPosts[2].Depth != 2 {
		t.Errorf("Expected nested reply depth 2, got %d", m.threadPosts[2].Depth)
	}
}

// 6. TestThreadTargetPostHighlighted
func TestThreadTargetPostHighlighted(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user", "Target"),
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	if !m.threadPosts[0].IsTarget {
		t.Errorf("Expected first post to be marked as target")
	}
	if m.viewport.SelectedIndex() != 0 {
		t.Errorf("Expected selectedIndex to be set to target, got %d", m.viewport.SelectedIndex())
	}
}

// 7. TestThreadScrollNavigation
func TestThreadScrollNavigation(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user", "Target"),
					Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
						{FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{Post: makePost("at://reply1", "user", "R1")}},
						{FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{Post: makePost("at://reply2", "user", "R2")}},
					},
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	// Currently at index 0
	// Move down
	m = doUpdate(m, tea.KeyPressMsg{Text: "j"})
	if m.viewport.SelectedIndex() != 1 {
		t.Errorf("Expected selectedIndex 1, got %d", m.viewport.SelectedIndex())
	}

	// Move up
	m = doUpdate(m, tea.KeyPressMsg{Text: "k"})
	if m.viewport.SelectedIndex() != 0 {
		t.Errorf("Expected selectedIndex 0, got %d", m.viewport.SelectedIndex())
	}
}

// 8. TestThreadBackNavigation
func TestThreadBackNavigation(t *testing.T) {
	t.Parallel()
	m := NewThreadModel(&mockClient{}, "at://target", "", 80, 24, nil)

	_, cmd := m.Update(tea.KeyPressMsg{Text: "esc"})
	if cmd == nil {
		t.Fatalf("Expected cmd on esc")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("Expected BackMsg, got %T", msg)
	}
}

// 9. TestThreadMissingParent
func TestThreadMissingParent(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user", "Target"),
					Parent: &bsky.FeedDefs_ThreadViewPost_Parent{
						FeedDefs_NotFoundPost: &bsky.FeedDefs_NotFoundPost{Uri: "at://deleted"},
					},
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	if len(m.threadPosts) != 2 {
		t.Fatalf("Expected 2 posts, got %d", len(m.threadPosts))
	}
	if !m.threadPosts[0].NotFound {
		t.Errorf("Expected first post to be NotFound")
	}

	view := m.View().Content
	if !strings.Contains(view, "This post was deleted") {
		t.Errorf("Expected 'This post was deleted' in view")
	}
}

// 10. TestThreadActionKeybinds
func TestThreadActionKeybinds(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user", "Target"),
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	// Test Like (l)
	_, cmd := m.Update(tea.KeyPressMsg{Text: "l"})
	if _, ok := cmd().(feed.LikePostMsg); !ok {
		t.Errorf("Expected LikePostMsg")
	}

	// Test Repost (t)
	_, cmd = m.Update(tea.KeyPressMsg{Text: "t"})
	if _, ok := cmd().(feed.RepostMsg); !ok {
		t.Errorf("Expected RepostMsg")
	}

	// Test Reply (r)
	_, cmd = m.Update(tea.KeyPressMsg{Text: "r"})
	if _, ok := cmd().(ComposeReplyMsg); !ok {
		t.Errorf("Expected ComposeReplyMsg")
	}

	// Test Profile (p)
	_, cmd = m.Update(tea.KeyPressMsg{Text: "p"})
	if _, ok := cmd().(ViewProfileMsg); !ok {
		t.Errorf("Expected ViewProfileMsg")
	}
}

// 11. TestThreadLoadingState
func TestThreadLoadingState(t *testing.T) {
	t.Parallel()
	m := NewThreadModel(&mockClient{}, "at://target", "", 80, 24, nil)
	if !m.loading {
		t.Errorf("Expected loading state initially")
	}

	view := m.View().Content
	if !strings.Contains(view, "Loading thread...") {
		t.Errorf("Expected loading text, got: %s", view)
	}
}

// 12. TestThreadErrorState
func TestThreadErrorState(t *testing.T) {
	t.Parallel()
	m := NewThreadModel(&mockClient{}, "at://target", "", 80, 24, nil)
	m = doUpdate(m, ThreadErrorMsg{Err: errors.New("network error")})

	view := m.View().Content
	if !strings.Contains(view, "network error") {
		t.Errorf("Expected error text, got: %s", view)
	}
}

// 13. TestThreadBlockedParent
func TestThreadBlockedParent(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user", "Target"),
					Parent: &bsky.FeedDefs_ThreadViewPost_Parent{
						FeedDefs_BlockedPost: &bsky.FeedDefs_BlockedPost{Uri: "at://blocked"},
					},
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	if len(m.threadPosts) != 2 {
		t.Fatalf("Expected 2 posts, got %d", len(m.threadPosts))
	}
	if !m.threadPosts[0].Blocked {
		t.Errorf("Expected first post to be Blocked")
	}

	view := m.View().Content
	if !strings.Contains(view, "This post is from a blocked account") {
		t.Errorf("Expected 'This post is from a blocked account' in view")
	}
}

// 14. TestThreadEnterOnReply
func TestThreadEnterOnReply(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user", "Target"),
					Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
						{FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{Post: makePost("at://reply1", "user", "Reply 1")}},
					},
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	// Move to the reply (index 1)
	m = doUpdate(m, tea.KeyPressMsg{Text: "j"})
	if m.viewport.SelectedIndex() != 1 {
		t.Fatalf("Expected selectedIndex 1, got %d", m.viewport.SelectedIndex())
	}

	// Press enter - should emit ViewThreadMsg for the reply
	_, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Fatal("Expected cmd on enter")
	}
	msg := cmd()
	vtMsg, ok := msg.(feed.ViewThreadMsg)
	if !ok {
		t.Errorf("Expected ViewThreadMsg, got %T", msg)
	}
	if vtMsg.URI != "at://reply1" {
		t.Errorf("Expected URI 'at://reply1', got %q", vtMsg.URI)
	}
}

// 15. TestThreadWindowSizeMsg
func TestThreadWindowSizeMsg(t *testing.T) {
	t.Parallel()
	m := NewThreadModel(&mockClient{}, "at://target", "", 80, 24, nil)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = updated.(ThreadModel)

	if m.width != 100 {
		t.Errorf("Expected width to be 100, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("Expected height to be 50, got %d", m.height)
	}
}

// 16. TestThreadDeleteFirstDPress
func TestThreadDeleteFirstDPress(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "testuser", "Target Post"),
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "did:plc:testuser", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "d"})
	m = updated.(ThreadModel)

	if m.confirmDelete != 0 {
		t.Errorf("Expected confirmDelete=0, got %d", m.confirmDelete)
	}
	if cmd != nil {
		t.Error("Expected no command on first 'd' press")
	}

	v := m.View().Content
	if !strings.Contains(v, "Press d to confirm delete") {
		t.Error("Expected confirmation prompt in view")
	}
}

// 17. TestThreadDeleteSecondDPress
func TestThreadDeleteSecondDPress(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "testuser", "Target Post"),
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "did:plc:testuser", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	m = doUpdate(m, tea.KeyPressMsg{Text: "d"})

	_, cmd := m.Update(tea.KeyPressMsg{Text: "d"})
	if cmd == nil {
		t.Fatal("Expected command on second 'd' press")
	}
	msg := cmd()
	deleteMsg, ok := msg.(feed.DeletePostMsg)
	if !ok {
		t.Fatalf("Expected DeletePostMsg, got %T", msg)
	}
	if deleteMsg.URI != "at://target" {
		t.Errorf("Expected URI 'at://target', got %q", deleteMsg.URI)
	}
}

// 18. TestThreadDeleteCancelOnOtherKey
func TestThreadDeleteCancelOnOtherKey(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "testuser", "Target Post"),
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "did:plc:testuser", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	m = doUpdate(m, tea.KeyPressMsg{Text: "d"})
	if m.confirmDelete != 0 {
		t.Fatalf("Expected confirmDelete=0, got %d", m.confirmDelete)
	}

	m = doUpdate(m, tea.KeyPressMsg{Text: "j"})
	if m.confirmDelete != -1 {
		t.Errorf("Expected confirmDelete reset to -1, got %d", m.confirmDelete)
	}
}

// 19. TestThreadDeleteNotOwnPost
func TestThreadDeleteNotOwnPost(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "otheruser", "Other Post"),
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "did:plc:me", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "d"})
	m = updated.(ThreadModel)

	if m.confirmDelete != -1 {
		t.Errorf("Expected confirmDelete to remain -1 for non-own post, got %d", m.confirmDelete)
	}
	if cmd != nil {
		t.Error("Expected no command when pressing 'd' on non-own post")
	}
}

// 20. TestThreadDeletePostResultRemovesPost
func TestThreadDeletePostResultRemovesPost(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post: makePost("at://target", "user", "Target"),
					Replies: []*bsky.FeedDefs_ThreadViewPost_Replies_Elem{
						{FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{Post: makePost("at://reply1", "user", "Reply")}},
					},
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "did:plc:user", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	if len(m.threadPosts) != 2 {
		t.Fatalf("Expected 2 thread posts, got %d", len(m.threadPosts))
	}

	m = doUpdate(m, feed.DeletePostResultMsg{URI: "at://reply1", Err: nil})
	if len(m.threadPosts) != 1 {
		t.Errorf("Expected 1 thread post after delete, got %d", len(m.threadPosts))
	}
}

// 21. TestThreadMouseWheelDownScrolls
func TestThreadMouseWheelDownScrolls(t *testing.T) {
	t.Parallel()
	replies := make([]*bsky.FeedDefs_ThreadViewPost_Replies_Elem, 10)
	for i := range 10 {
		replies[i] = &bsky.FeedDefs_ThreadViewPost_Replies_Elem{
			FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
				Post: makePost("at://reply"+string(rune('0'+i)), "user", "Reply"),
			},
		}
	}
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post:    makePost("at://target", "user", "Target"),
					Replies: replies,
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	updated, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m = updated.(ThreadModel)

	if m.viewport.SelectedIndex() != 3 {
		t.Errorf("Expected selectedIndex=3 after mouse wheel down, got %d", m.viewport.SelectedIndex())
	}
}

// 22. TestThreadMouseWheelUpScrolls
func TestThreadMouseWheelUpScrolls(t *testing.T) {
	t.Parallel()
	replies := make([]*bsky.FeedDefs_ThreadViewPost_Replies_Elem, 10)
	for i := range 10 {
		replies[i] = &bsky.FeedDefs_ThreadViewPost_Replies_Elem{
			FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
				Post: makePost("at://reply"+string(rune('0'+i)), "user", "Reply"),
			},
		}
	}
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_ThreadViewPost: &bsky.FeedDefs_ThreadViewPost{
					Post:    makePost("at://target", "user", "Target"),
					Replies: replies,
				},
			},
		},
	}
	m := NewThreadModel(client, "at://target", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())
	m.viewport.SetSelectedIndex(6)

	updated, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	m = updated.(ThreadModel)

	if m.viewport.SelectedIndex() != 3 {
		t.Errorf("Expected selectedIndex=3 after mouse wheel up from 6, got %d", m.viewport.SelectedIndex())
	}
}

// 23. TestThreadSpinnerTickDuringLoad
func TestThreadSpinnerTickDuringLoad(t *testing.T) {
	t.Parallel()
	m := NewThreadModel(&mockClient{}, "at://target", "", 80, 24, nil)

	_, cmd := m.Update(m.spinner.Tick())
	if cmd == nil {
		t.Error("Expected spinner tick to return a command when loading")
	}
}

// 24. TestThreadSpinnerTickIgnoredWhenNotLoading
func TestThreadSpinnerTickIgnoredWhenNotLoading(t *testing.T) {
	t.Parallel()
	m := NewThreadModel(&mockClient{}, "at://target", "", 80, 24, nil)
	m.loading = false

	_, cmd := m.Update(m.spinner.Tick())
	if cmd != nil {
		t.Error("Expected spinner tick to return nil command when not loading")
	}
}

// 25. TestThreadNotFoundRoot
func TestThreadNotFoundRoot(t *testing.T) {
	t.Parallel()
	client := &mockClient{
		threadResult: &bsky.FeedGetPostThread_Output{
			Thread: &bsky.FeedGetPostThread_Output_Thread{
				FeedDefs_NotFoundPost: &bsky.FeedDefs_NotFoundPost{Uri: "at://notfound"},
			},
		},
	}
	m := NewThreadModel(client, "at://notfound", "", 80, 24, nil)
	m = doUpdate(m, m.Init()())

	if len(m.threadPosts) != 1 {
		t.Fatalf("Expected 1 post, got %d", len(m.threadPosts))
	}
	if !m.threadPosts[0].NotFound {
		t.Errorf("Expected post to be NotFound")
	}
}
