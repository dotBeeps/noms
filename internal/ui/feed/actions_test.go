package feed

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
)

// actionMockClient extends mockBlueskyClient with controllable action responses.
type actionMockClient struct {
	mockBlueskyClient
	likeURI     string
	likeErr     error
	unlikeErr   error
	repostURI   string
	repostErr   error
	unrepostErr error

	likeCalledWith     [2]string
	unlikeCalledWith   string
	repostCalledWith   [2]string
	unrepostCalledWith string
}

func (m *actionMockClient) Like(ctx context.Context, uri, cid string) (string, error) {
	m.likeCalledWith = [2]string{uri, cid}
	return m.likeURI, m.likeErr
}

func (m *actionMockClient) Unlike(ctx context.Context, likeURI string) error {
	m.unlikeCalledWith = likeURI
	return m.unlikeErr
}

func (m *actionMockClient) Repost(ctx context.Context, uri, cid string) (string, error) {
	m.repostCalledWith = [2]string{uri, cid}
	return m.repostURI, m.repostErr
}

func (m *actionMockClient) UnRepost(ctx context.Context, repostURI string) error {
	m.unrepostCalledWith = repostURI
	return m.unrepostErr
}

func createTestPostWithViewer(text, uri, cid string, likeURI, repostURI *string, likeCount, repostCount int64) *bsky.FeedDefs_FeedViewPost {
	post := createTestPost(text, "test.bsky.social", "Test", uri, cid)
	post.Post.LikeCount = &likeCount
	post.Post.RepostCount = &repostCount
	post.Post.Viewer = &bsky.FeedDefs_ViewerState{
		Like:   likeURI,
		Repost: repostURI,
	}
	return post
}

// Test 1: TestLikePost — 'l' on unliked post applies optimistic update and fires API call.
func TestLikePost(t *testing.T) {
	t.Parallel()
	client := &actionMockClient{likeURI: "at://like/uri/123"}
	post := createTestPost("Hello", "test.bsky.social", "Test", "at://post/uri/1", "cid1")
	post.Post.LikeCount = int64Ptr(5)

	m := NewFeedModel(client, "", 80, 24, nil)
	m.posts = []*bsky.FeedDefs_FeedViewPost{post}
	m.loading = false

	_, cmd := m.Update(tea.KeyPressMsg{Text: "l"})
	if cmd == nil {
		t.Fatal("Expected 'l' to return a command")
	}

	likeMsg := cmd()
	if _, ok := likeMsg.(LikePostMsg); !ok {
		t.Fatalf("Expected LikePostMsg from 'l' key, got %T", likeMsg)
	}

	updated, cmd2 := m.Update(likeMsg)
	m = updated.(FeedModel)

	if cmd2 == nil {
		t.Fatal("Expected LikePostMsg handler to return a command")
	}

	if m.posts[0].Post.Viewer == nil || m.posts[0].Post.Viewer.Like == nil {
		t.Fatal("Expected optimistic like to be applied")
	}
	if *m.posts[0].Post.Viewer.Like != "pending" {
		t.Errorf("Expected viewer.Like to be 'pending', got %q", *m.posts[0].Post.Viewer.Like)
	}
	if *m.posts[0].Post.LikeCount != 6 {
		t.Errorf("Expected like count to be 6 after optimistic like, got %d", *m.posts[0].Post.LikeCount)
	}

	resultMsg := cmd2()
	if likeResult, ok := resultMsg.(LikeResultMsg); !ok {
		t.Fatalf("Expected LikeResultMsg from API cmd, got %T", resultMsg)
	} else if likeResult.Err != nil {
		t.Errorf("Expected no error, got %v", likeResult.Err)
	} else if likeResult.LikeURI != "at://like/uri/123" {
		t.Errorf("Expected like URI 'at://like/uri/123', got %q", likeResult.LikeURI)
	}
}

// Test 2: TestUnlikePost — 'l' on liked post applies optimistic unlike and fires API call.
func TestUnlikePost(t *testing.T) {
	t.Parallel()
	likeURI := "at://like/existing/456"
	client := &actionMockClient{}
	post := createTestPostWithViewer("Hello", "at://post/uri/2", "cid2", &likeURI, nil, 10, 0)

	m := NewFeedModel(client, "", 80, 24, nil)
	m.posts = []*bsky.FeedDefs_FeedViewPost{post}
	m.loading = false

	_, cmd := m.Update(tea.KeyPressMsg{Text: "l"})
	if cmd == nil {
		t.Fatal("Expected 'l' to return a command")
	}

	likeMsg := cmd()
	if _, ok := likeMsg.(LikePostMsg); !ok {
		t.Fatalf("Expected LikePostMsg, got %T", likeMsg)
	}

	updated, cmd2 := m.Update(likeMsg)
	m = updated.(FeedModel)

	if cmd2 == nil {
		t.Fatal("Expected LikePostMsg handler to return a command for unlike")
	}

	if m.posts[0].Post.Viewer != nil && m.posts[0].Post.Viewer.Like != nil {
		t.Errorf("Expected viewer.Like to be nil after optimistic unlike, got %q", *m.posts[0].Post.Viewer.Like)
	}
	if *m.posts[0].Post.LikeCount != 9 {
		t.Errorf("Expected like count to be 9 after optimistic unlike, got %d", *m.posts[0].Post.LikeCount)
	}

	resultMsg := cmd2()
	if unlikeResult, ok := resultMsg.(UnlikeResultMsg); !ok {
		t.Fatalf("Expected UnlikeResultMsg from API cmd, got %T", resultMsg)
	} else if unlikeResult.Err != nil {
		t.Errorf("Expected no error, got %v", unlikeResult.Err)
	} else if client.unlikeCalledWith != likeURI {
		t.Errorf("Expected Unlike to be called with %q, got %q", likeURI, client.unlikeCalledWith)
	}
}

// Test 3: TestRepostFromFeed — 't' on unreposted post applies optimistic repost.
func TestRepostFromFeed(t *testing.T) {
	t.Parallel()
	client := &actionMockClient{repostURI: "at://repost/uri/789"}
	post := createTestPost("Hello", "test.bsky.social", "Test", "at://post/uri/3", "cid3")
	post.Post.RepostCount = int64Ptr(2)

	m := NewFeedModel(client, "", 80, 24, nil)
	m.posts = []*bsky.FeedDefs_FeedViewPost{post}
	m.loading = false

	_, cmd := m.Update(tea.KeyPressMsg{Text: "t"})
	if cmd == nil {
		t.Fatal("Expected 't' to return a command")
	}

	repostMsg := cmd()
	if _, ok := repostMsg.(RepostMsg); !ok {
		t.Fatalf("Expected RepostMsg, got %T", repostMsg)
	}

	updated, cmd2 := m.Update(repostMsg)
	m = updated.(FeedModel)

	if cmd2 == nil {
		t.Fatal("Expected RepostMsg handler to return a command")
	}

	if m.posts[0].Post.Viewer == nil || m.posts[0].Post.Viewer.Repost == nil {
		t.Fatal("Expected optimistic repost to be applied")
	}
	if *m.posts[0].Post.Viewer.Repost != "pending" {
		t.Errorf("Expected viewer.Repost to be 'pending', got %q", *m.posts[0].Post.Viewer.Repost)
	}
	if *m.posts[0].Post.RepostCount != 3 {
		t.Errorf("Expected repost count to be 3 after optimistic repost, got %d", *m.posts[0].Post.RepostCount)
	}
}

// Test 4: TestUnRepostFromFeed — 't' on reposted post applies optimistic unrepost.
func TestUnRepostFromFeed(t *testing.T) {
	t.Parallel()
	repostURI := "at://repost/existing/999"
	client := &actionMockClient{}
	post := createTestPostWithViewer("Hello", "at://post/uri/4", "cid4", nil, &repostURI, 0, 7)

	m := NewFeedModel(client, "", 80, 24, nil)
	m.posts = []*bsky.FeedDefs_FeedViewPost{post}
	m.loading = false

	_, cmd := m.Update(tea.KeyPressMsg{Text: "t"})
	if cmd == nil {
		t.Fatal("Expected 't' to return a command")
	}

	repostMsg := cmd()
	if _, ok := repostMsg.(RepostMsg); !ok {
		t.Fatalf("Expected RepostMsg, got %T", repostMsg)
	}

	updated, cmd2 := m.Update(repostMsg)
	m = updated.(FeedModel)

	if cmd2 == nil {
		t.Fatal("Expected RepostMsg handler to return a command for unrepost")
	}

	if m.posts[0].Post.Viewer != nil && m.posts[0].Post.Viewer.Repost != nil {
		t.Errorf("Expected viewer.Repost to be nil after optimistic unrepost, got %q", *m.posts[0].Post.Viewer.Repost)
	}
	if *m.posts[0].Post.RepostCount != 6 {
		t.Errorf("Expected repost count to be 6 after optimistic unrepost, got %d", *m.posts[0].Post.RepostCount)
	}

	cmd2()

	if client.unrepostCalledWith != repostURI {
		t.Errorf("Expected UnRepost to be called with %q, got %q", repostURI, client.unrepostCalledWith)
	}
}

// Test 5: TestReplyOpensCompose — 'r' on selected post emits ComposeReplyMsg with correct URI/CID.
func TestReplyOpensCompose(t *testing.T) {
	t.Parallel()
	client := &actionMockClient{}
	post := createTestPost("Hello", "test.bsky.social", "Test", "at://post/uri/5", "cid5")

	m := NewFeedModel(client, "", 80, 24, nil)
	m.posts = []*bsky.FeedDefs_FeedViewPost{post}
	m.loading = false
	m.selectedIndex = 0

	_, cmd := m.Update(tea.KeyPressMsg{Text: "r"})
	if cmd == nil {
		t.Fatal("Expected 'r' to return a command")
	}

	msg := cmd()
	replyMsg, ok := msg.(ComposeReplyMsg)
	if !ok {
		t.Fatalf("Expected ComposeReplyMsg, got %T", msg)
	}
	if replyMsg.URI != "at://post/uri/5" {
		t.Errorf("Expected URI 'at://post/uri/5', got %q", replyMsg.URI)
	}
	if replyMsg.CID != "cid5" {
		t.Errorf("Expected CID 'cid5', got %q", replyMsg.CID)
	}
}

// Test 6: TestOptimisticUpdate — like count incremented immediately before API response.
func TestOptimisticUpdate(t *testing.T) {
	t.Parallel()
	client := &actionMockClient{likeURI: "at://like/new"}
	post := createTestPost("Hello", "test.bsky.social", "Test", "at://post/uri/6", "cid6")
	initialCount := int64(42)
	post.Post.LikeCount = &initialCount

	m := NewFeedModel(client, "", 80, 24, nil)
	m.posts = []*bsky.FeedDefs_FeedViewPost{post}
	m.loading = false

	_, cmd := m.Update(tea.KeyPressMsg{Text: "l"})
	likeMsg := cmd()

	updated, _ := m.Update(likeMsg)
	m = updated.(FeedModel)

	if *m.posts[0].Post.LikeCount != 43 {
		t.Errorf("Expected like count 43 immediately after optimistic update, got %d", *m.posts[0].Post.LikeCount)
	}
	if m.posts[0].Post.Viewer == nil || m.posts[0].Post.Viewer.Like == nil {
		t.Fatal("Expected viewer.Like to be set immediately after optimistic update")
	}
}

// Test 7: TestOptimisticRevertOnError — LikeResultMsg with error triggers rollback.
func TestOptimisticRevertOnError(t *testing.T) {
	t.Parallel()
	apiErr := errors.New("network failure")
	client := &actionMockClient{likeErr: apiErr}
	post := createTestPost("Hello", "test.bsky.social", "Test", "at://post/uri/7", "cid7")
	initialCount := int64(10)
	post.Post.LikeCount = &initialCount

	m := NewFeedModel(client, "", 80, 24, nil)
	m.posts = []*bsky.FeedDefs_FeedViewPost{post}
	m.loading = false

	_, cmd := m.Update(tea.KeyPressMsg{Text: "l"})
	likeMsg := cmd()

	updated, cmd2 := m.Update(likeMsg)
	m = updated.(FeedModel)

	if cmd2 == nil {
		t.Fatal("Expected LikePostMsg to return API cmd")
	}

	resultMsg := cmd2()

	updated, _ = m.Update(resultMsg)
	m = updated.(FeedModel)

	if *m.posts[0].Post.LikeCount != 10 {
		t.Errorf("Expected like count rolled back to 10, got %d", *m.posts[0].Post.LikeCount)
	}
	if m.posts[0].Post.Viewer != nil && m.posts[0].Post.Viewer.Like != nil {
		t.Errorf("Expected viewer.Like to be nil after rollback, got %q", *m.posts[0].Post.Viewer.Like)
	}
}

// Test 8: TestViewThreadNavigation — enter key emits ViewThreadMsg with correct URI.
func TestViewThreadNavigation(t *testing.T) {
	t.Parallel()
	client := &actionMockClient{}
	posts := []*bsky.FeedDefs_FeedViewPost{
		createTestPost("First", "a.bsky.social", "A", "at://uri/first", "cid1"),
		createTestPost("Second", "b.bsky.social", "B", "at://uri/second", "cid2"),
	}

	m := NewFeedModel(client, "", 80, 24, nil)
	m.posts = posts
	m.loading = false
	m.selectedIndex = 1

	_, cmd := m.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Fatal("Expected 'enter' to return a command")
	}

	msg := cmd()
	vtMsg, ok := msg.(ViewThreadMsg)
	if !ok {
		t.Fatalf("Expected ViewThreadMsg, got %T", msg)
	}
	if vtMsg.URI != "at://uri/second" {
		t.Errorf("Expected URI 'at://uri/second', got %q", vtMsg.URI)
	}
}

// Test 9: TestLikeResultMsgUpdatesRealURI — LikeResultMsg success updates viewer.Like with real URI.
func TestLikeResultMsgUpdatesRealURI(t *testing.T) {
	t.Parallel()
	client := &actionMockClient{likeURI: "at://like/real/uri"}
	post := createTestPost("Hello", "test.bsky.social", "Test", "at://post/uri/9", "cid9")
	post.Post.LikeCount = int64Ptr(5)

	m := NewFeedModel(client, "", 80, 24, nil)
	m.posts = []*bsky.FeedDefs_FeedViewPost{post}
	m.loading = false

	_, cmd := m.Update(tea.KeyPressMsg{Text: "l"})
	likeMsg := cmd()
	updated, cmd2 := m.Update(likeMsg)
	m = updated.(FeedModel)

	resultMsg := cmd2()
	updated, _ = m.Update(resultMsg)
	m = updated.(FeedModel)

	if m.posts[0].Post.Viewer == nil || m.posts[0].Post.Viewer.Like == nil {
		t.Fatal("Expected viewer.Like to be set after successful like")
	}
	if *m.posts[0].Post.Viewer.Like != "at://like/real/uri" {
		t.Errorf("Expected viewer.Like to be 'at://like/real/uri', got %q", *m.posts[0].Post.Viewer.Like)
	}
}

// Test 10: TestRepostResultMsgUpdatesRealURI — RepostResultMsg success updates viewer.Repost with real URI.
func TestRepostResultMsgUpdatesRealURI(t *testing.T) {
	t.Parallel()
	client := &actionMockClient{repostURI: "at://repost/real/uri"}
	post := createTestPost("Hello", "test.bsky.social", "Test", "at://post/uri/10", "cid10")
	post.Post.RepostCount = int64Ptr(3)

	m := NewFeedModel(client, "", 80, 24, nil)
	m.posts = []*bsky.FeedDefs_FeedViewPost{post}
	m.loading = false

	_, cmd := m.Update(tea.KeyPressMsg{Text: "t"})
	repostMsg := cmd()
	updated, cmd2 := m.Update(repostMsg)
	m = updated.(FeedModel)

	resultMsg := cmd2()
	updated, _ = m.Update(resultMsg)
	m = updated.(FeedModel)

	if m.posts[0].Post.Viewer == nil || m.posts[0].Post.Viewer.Repost == nil {
		t.Fatal("Expected viewer.Repost to be set after successful repost")
	}
	if *m.posts[0].Post.Viewer.Repost != "at://repost/real/uri" {
		t.Errorf("Expected viewer.Repost to be 'at://repost/real/uri', got %q", *m.posts[0].Post.Viewer.Repost)
	}
}

// Ensure actionMockClient satisfies the interface at compile time.
var _ interface {
	Like(ctx context.Context, uri, cid string) (string, error)
	Unlike(ctx context.Context, likeURI string) error
	Repost(ctx context.Context, uri, cid string) (string, error)
	UnRepost(ctx context.Context, repostURI string) error
} = (*actionMockClient)(nil)

// Suppress unused import warning.
var _ = time.Now
