package feed

import (
	"context"

	tea "charm.land/bubbletea/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
)

// Result messages returned by async API calls.

// LikeResultMsg is returned after a like API call completes.
type LikeResultMsg struct {
	PostURI string // The post that was liked
	LikeURI string // The URI of the like record (for unlike)
	Err     error  // nil on success, error on failure → triggers rollback
}

// UnlikeResultMsg is returned after an unlike API call completes.
type UnlikeResultMsg struct {
	PostURI string
	Err     error
}

// RepostResultMsg is returned after a repost API call completes.
type RepostResultMsg struct {
	PostURI   string
	RepostURI string
	Err       error
}

// UnRepostResultMsg is returned after an unrepost API call completes.
type UnRepostResultMsg struct {
	PostURI string
	Err     error
}

// ComposeReplyMsg is emitted when the user presses 'r' to reply to a post.
type ComposeReplyMsg struct {
	URI string
	CID string
}

// DeletePostMsg is emitted when the user confirms deletion of their own post.
type DeletePostMsg struct {
	URI string
}

// DeletePostResultMsg is returned after a delete API call completes.
type DeletePostResultMsg struct {
	URI string
	Err error
}

// PerformLike fires an async API call to like a post and returns a LikeResultMsg.
func PerformLike(client bluesky.BlueskyClient, postURI, postCID string) tea.Cmd {
	return func() tea.Msg {
		likeURI, err := client.Like(context.Background(), postURI, postCID)
		return LikeResultMsg{
			PostURI: postURI,
			LikeURI: likeURI,
			Err:     err,
		}
	}
}

// PerformUnlike fires an async API call to unlike a post and returns an UnlikeResultMsg.
func PerformUnlike(client bluesky.BlueskyClient, postURI, likeURI string) tea.Cmd {
	return func() tea.Msg {
		err := client.Unlike(context.Background(), likeURI)
		return UnlikeResultMsg{
			PostURI: postURI,
			Err:     err,
		}
	}
}

// PerformRepost fires an async API call to repost a post and returns a RepostResultMsg.
func PerformRepost(client bluesky.BlueskyClient, postURI, postCID string) tea.Cmd {
	return func() tea.Msg {
		repostURI, err := client.Repost(context.Background(), postURI, postCID)
		return RepostResultMsg{
			PostURI:   postURI,
			RepostURI: repostURI,
			Err:       err,
		}
	}
}

// PerformUnRepost fires an async API call to unrepost a post and returns an UnRepostResultMsg.
func PerformUnRepost(client bluesky.BlueskyClient, postURI, repostURI string) tea.Cmd {
	return func() tea.Msg {
		err := client.UnRepost(context.Background(), repostURI)
		return UnRepostResultMsg{
			PostURI: postURI,
			Err:     err,
		}
	}
}

// OptimisticLike applies an optimistic like to a post before the API call completes.
func OptimisticLike(pv *bsky.FeedDefs_PostView) {
	if pv.Viewer == nil {
		pv.Viewer = &bsky.FeedDefs_ViewerState{}
	}
	if pv.LikeCount == nil {
		zero := int64(0)
		pv.LikeCount = &zero
	}
	*pv.LikeCount++
	pv.Viewer.Like = new(string)
	*pv.Viewer.Like = "pending"
}

// RollbackLike reverts an optimistic like on failure.
func RollbackLike(pv *bsky.FeedDefs_PostView) {
	if pv.LikeCount != nil && *pv.LikeCount > 0 {
		*pv.LikeCount--
	}
	if pv.Viewer != nil {
		pv.Viewer.Like = nil
	}
}

// OptimisticUnlike applies an optimistic unlike to a post before the API call completes.
func OptimisticUnlike(pv *bsky.FeedDefs_PostView) {
	if pv.LikeCount != nil && *pv.LikeCount > 0 {
		*pv.LikeCount--
	}
	if pv.Viewer != nil {
		pv.Viewer.Like = nil
	}
}

// RollbackUnlike reverts an optimistic unlike on failure.
func RollbackUnlike(pv *bsky.FeedDefs_PostView, likeURI string) {
	if pv.Viewer == nil {
		pv.Viewer = &bsky.FeedDefs_ViewerState{}
	}
	if pv.LikeCount == nil {
		zero := int64(0)
		pv.LikeCount = &zero
	}
	*pv.LikeCount++
	pv.Viewer.Like = &likeURI
}

// OptimisticRepost applies an optimistic repost to a post before the API call completes.
func OptimisticRepost(pv *bsky.FeedDefs_PostView) {
	if pv.Viewer == nil {
		pv.Viewer = &bsky.FeedDefs_ViewerState{}
	}
	if pv.RepostCount == nil {
		zero := int64(0)
		pv.RepostCount = &zero
	}
	*pv.RepostCount++
	pv.Viewer.Repost = new(string)
	*pv.Viewer.Repost = "pending"
}

// RollbackRepost reverts an optimistic repost on failure.
func RollbackRepost(pv *bsky.FeedDefs_PostView) {
	if pv.RepostCount != nil && *pv.RepostCount > 0 {
		*pv.RepostCount--
	}
	if pv.Viewer != nil {
		pv.Viewer.Repost = nil
	}
}

// OptimisticUnRepost applies an optimistic unrepost to a post before the API call completes.
func OptimisticUnRepost(pv *bsky.FeedDefs_PostView) {
	if pv.RepostCount != nil && *pv.RepostCount > 0 {
		*pv.RepostCount--
	}
	if pv.Viewer != nil {
		pv.Viewer.Repost = nil
	}
}

// RollbackUnRepost reverts an optimistic unrepost on failure.
func RollbackUnRepost(pv *bsky.FeedDefs_PostView, repostURI string) {
	if pv.Viewer == nil {
		pv.Viewer = &bsky.FeedDefs_ViewerState{}
	}
	if pv.RepostCount == nil {
		zero := int64(0)
		pv.RepostCount = &zero
	}
	*pv.RepostCount++
	pv.Viewer.Repost = &repostURI
}

// FindPostByURI finds a post in a feed slice by its URI and returns a pointer to it.
// Returns nil if not found.
func FindPostByURI(posts []*bsky.FeedDefs_FeedViewPost, uri string) *bsky.FeedDefs_FeedViewPost {
	for _, p := range posts {
		if p.Post != nil && p.Post.Uri == uri {
			return p
		}
	}
	return nil
}
