package bluesky

import (
	"context"

	bsky "github.com/bluesky-social/indigo/api/bsky"
)

// SearchPosts searches for posts matching the query string.
// Returns matching posts, the next cursor, and any error.
func (c *Client) SearchPosts(ctx context.Context, query string, cursor string, limit int) ([]*bsky.FeedDefs_PostView, string, error) {
	var out bsky.FeedSearchPosts_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Get(ctx, "app.bsky.feed.searchPosts", map[string]any{
			"q":      query,
			"cursor": cursor,
			"limit":  int64(limit),
		}, &out)
	})
	if err != nil {
		return nil, "", wrapErr("SearchPosts", err)
	}

	nextCursor := ""
	if out.Cursor != nil {
		nextCursor = *out.Cursor
	}
	return out.Posts, nextCursor, nil
}

// SearchActors searches for user profiles matching the query string.
// Returns matching profiles, the next cursor, and any error.
func (c *Client) SearchActors(ctx context.Context, query string, cursor string, limit int) ([]*bsky.ActorDefs_ProfileView, string, error) {
	var out bsky.ActorSearchActors_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Get(ctx, "app.bsky.actor.searchActors", map[string]any{
			"q":      query,
			"cursor": cursor,
			"limit":  int64(limit),
		}, &out)
	})
	if err != nil {
		return nil, "", wrapErr("SearchActors", err)
	}

	nextCursor := ""
	if out.Cursor != nil {
		nextCursor = *out.Cursor
	}
	return out.Actors, nextCursor, nil
}
