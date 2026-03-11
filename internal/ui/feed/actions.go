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

// performLike fires an async API call to like a post and returns a LikeResultMsg.
func performLike(client bluesky.BlueskyClient, postURI, postCID string) tea.Cmd {
	return func() tea.Msg {
		likeURI, err := client.Like(context.Background(), postURI, postCID)
		return LikeResultMsg{
			PostURI: postURI,
			LikeURI: likeURI,
			Err:     err,
		}
	}
}

// performUnlike fires an async API call to unlike a post and returns an UnlikeResultMsg.
func performUnlike(client bluesky.BlueskyClient, postURI, likeURI string) tea.Cmd {
	return func() tea.Msg {
		err := client.Unlike(context.Background(), likeURI)
		return UnlikeResultMsg{
			PostURI: postURI,
			Err:     err,
		}
	}
}

// performRepost fires an async API call to repost a post and returns a RepostResultMsg.
func performRepost(client bluesky.BlueskyClient, postURI, postCID string) tea.Cmd {
	return func() tea.Msg {
		repostURI, err := client.Repost(context.Background(), postURI, postCID)
		return RepostResultMsg{
			PostURI:   postURI,
			RepostURI: repostURI,
			Err:       err,
		}
	}
}

// performUnRepost fires an async API call to unrepost a post and returns an UnRepostResultMsg.
func performUnRepost(client bluesky.BlueskyClient, postURI, repostURI string) tea.Cmd {
	return func() tea.Msg {
		err := client.UnRepost(context.Background(), repostURI)
		return UnRepostResultMsg{
			PostURI: postURI,
			Err:     err,
		}
	}
}

// optimisticLike applies an optimistic like to a post before the API call completes.
func optimisticLike(post *bsky.FeedDefs_FeedViewPost) {
	if post.Post.Viewer == nil {
		post.Post.Viewer = &bsky.FeedDefs_ViewerState{}
	}
	if post.Post.LikeCount == nil {
		zero := int64(0)
		post.Post.LikeCount = &zero
	}
	*post.Post.LikeCount++
	post.Post.Viewer.Like = new(string)
	*post.Post.Viewer.Like = "pending"
}

// rollbackLike reverts an optimistic like on failure.
func rollbackLike(post *bsky.FeedDefs_FeedViewPost) {
	if post.Post.LikeCount != nil && *post.Post.LikeCount > 0 {
		*post.Post.LikeCount--
	}
	if post.Post.Viewer != nil {
		post.Post.Viewer.Like = nil
	}
}

// optimisticUnlike applies an optimistic unlike to a post before the API call completes.
func optimisticUnlike(post *bsky.FeedDefs_FeedViewPost) {
	if post.Post.LikeCount != nil && *post.Post.LikeCount > 0 {
		*post.Post.LikeCount--
	}
	if post.Post.Viewer != nil {
		post.Post.Viewer.Like = nil
	}
}

// rollbackUnlike reverts an optimistic unlike on failure.
func rollbackUnlike(post *bsky.FeedDefs_FeedViewPost, likeURI string) {
	if post.Post.Viewer == nil {
		post.Post.Viewer = &bsky.FeedDefs_ViewerState{}
	}
	if post.Post.LikeCount == nil {
		zero := int64(0)
		post.Post.LikeCount = &zero
	}
	*post.Post.LikeCount++
	post.Post.Viewer.Like = &likeURI
}

// optimisticRepost applies an optimistic repost to a post before the API call completes.
func optimisticRepost(post *bsky.FeedDefs_FeedViewPost) {
	if post.Post.Viewer == nil {
		post.Post.Viewer = &bsky.FeedDefs_ViewerState{}
	}
	if post.Post.RepostCount == nil {
		zero := int64(0)
		post.Post.RepostCount = &zero
	}
	*post.Post.RepostCount++
	post.Post.Viewer.Repost = new(string)
	*post.Post.Viewer.Repost = "pending"
}

// rollbackRepost reverts an optimistic repost on failure.
func rollbackRepost(post *bsky.FeedDefs_FeedViewPost) {
	if post.Post.RepostCount != nil && *post.Post.RepostCount > 0 {
		*post.Post.RepostCount--
	}
	if post.Post.Viewer != nil {
		post.Post.Viewer.Repost = nil
	}
}

// optimisticUnRepost applies an optimistic unrepost to a post before the API call completes.
func optimisticUnRepost(post *bsky.FeedDefs_FeedViewPost) {
	if post.Post.RepostCount != nil && *post.Post.RepostCount > 0 {
		*post.Post.RepostCount--
	}
	if post.Post.Viewer != nil {
		post.Post.Viewer.Repost = nil
	}
}

// rollbackUnRepost reverts an optimistic unrepost on failure.
func rollbackUnRepost(post *bsky.FeedDefs_FeedViewPost, repostURI string) {
	if post.Post.Viewer == nil {
		post.Post.Viewer = &bsky.FeedDefs_ViewerState{}
	}
	if post.Post.RepostCount == nil {
		zero := int64(0)
		post.Post.RepostCount = &zero
	}
	*post.Post.RepostCount++
	post.Post.Viewer.Repost = &repostURI
}

// findPostByURI finds a post in the feed by its URI and returns a pointer to it.
// Returns nil if not found.
func findPostByURI(posts []*bsky.FeedDefs_FeedViewPost, uri string) *bsky.FeedDefs_FeedViewPost {
	for _, p := range posts {
		if p.Post != nil && p.Post.Uri == uri {
			return p
		}
	}
	return nil
}
