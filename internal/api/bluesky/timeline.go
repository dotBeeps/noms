package bluesky

import (
	"context"

	bsky "github.com/bluesky-social/indigo/api/bsky"
)

// GetTimeline fetches the authenticated user's home timeline.
// Returns posts, the next pagination cursor, and any error.
func (c *Client) GetTimeline(ctx context.Context, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	var out bsky.FeedGetTimeline_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Get(ctx, "app.bsky.feed.getTimeline", map[string]any{
			"cursor": cursor,
			"limit":  int64(limit),
		}, &out)
	})
	if err != nil {
		return nil, "", wrapErr("GetTimeline", err)
	}

	nextCursor := ""
	if out.Cursor != nil {
		nextCursor = *out.Cursor
	}
	return out.Feed, nextCursor, nil
}

// GetPost fetches a single post by its AT-URI.
// Uses app.bsky.feed.getPosts with a single URI.
func (c *Client) GetPost(ctx context.Context, uri string) (*bsky.FeedDefs_PostView, error) {
	var out bsky.FeedGetPosts_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Get(ctx, "app.bsky.feed.getPosts", map[string]any{
			"uris": []string{uri},
		}, &out)
	})
	if err != nil {
		return nil, wrapErr("GetPost", err)
	}

	if len(out.Posts) == 0 {
		return nil, wrapErr("GetPost", &atProtoError{message: "post not found: " + uri})
	}
	return out.Posts[0], nil
}

// GetPostThread fetches a post thread by URI with the specified reply depth.
func (c *Client) GetPostThread(ctx context.Context, uri string, depth int) (*bsky.FeedGetPostThread_Output, error) {
	var out bsky.FeedGetPostThread_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Get(ctx, "app.bsky.feed.getPostThread", map[string]any{
			"uri":   uri,
			"depth": int64(depth),
		}, &out)
	})
	if err != nil {
		return nil, wrapErr("GetPostThread", err)
	}
	return &out, nil
}

// atProtoError is a simple error type for atproto-level errors.
type atProtoError struct {
	message string
}

func (e *atProtoError) Error() string {
	return e.message
}
