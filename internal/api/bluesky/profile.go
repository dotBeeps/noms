package bluesky

import (
	"context"
	"fmt"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
)

// GetProfile fetches a user profile by DID or handle.
func (c *Client) GetProfile(ctx context.Context, actor string) (*bsky.ActorDefs_ProfileViewDetailed, error) {
	var out bsky.ActorDefs_ProfileViewDetailed
	err := withRetry(ctx, 2, func() error {
		return c.api.Get(ctx, "app.bsky.actor.getProfile", map[string]any{
			"actor": actor,
		}, &out)
	})
	if err != nil {
		return nil, wrapErr("GetProfile", err)
	}
	return &out, nil
}

// GetAuthorFeed fetches posts authored by the given actor.
// Returns posts, the next pagination cursor, and any error.
func (c *Client) GetAuthorFeed(ctx context.Context, actor string, cursor string, limit int) ([]*bsky.FeedDefs_FeedViewPost, string, error) {
	var out bsky.FeedGetAuthorFeed_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Get(ctx, "app.bsky.feed.getAuthorFeed", map[string]any{
			"actor":  actor,
			"cursor": cursor,
			"limit":  int64(limit),
		}, &out)
	})
	if err != nil {
		return nil, "", wrapErr("GetAuthorFeed", err)
	}

	nextCursor := ""
	if out.Cursor != nil {
		nextCursor = *out.Cursor
	}
	return out.Feed, nextCursor, nil
}

// FollowActor creates a follow record for the given DID.
func (c *Client) FollowActor(ctx context.Context, did string) error {
	record := &bsky.GraphFollow{
		LexiconTypeID: "app.bsky.graph.follow",
		Subject:       did,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	input := &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.graph.follow",
		Repo:       c.did,
		Record:     &lexutil.LexiconTypeDecoder{Val: record},
	}

	err := withRetry(ctx, 2, func() error {
		var out comatproto.RepoCreateRecord_Output
		return c.api.Post(ctx, "com.atproto.repo.createRecord", input, &out)
	})
	if err != nil {
		return wrapErr(fmt.Sprintf("FollowActor(%s)", did), err)
	}
	return nil
}

// UnfollowActor deletes the follow record for the given DID.
// It first looks up the follow record URI from the profile viewer state,
// then deletes it using the rkey extracted from the AT-URI.
func (c *Client) UnfollowActor(ctx context.Context, did string) error {
	// First get the profile to find the follow record URI
	profile, err := c.GetProfile(ctx, did)
	if err != nil {
		return wrapErr(fmt.Sprintf("UnfollowActor(%s)", did), err)
	}

	if profile.Viewer == nil || profile.Viewer.Following == nil {
		return wrapErr(fmt.Sprintf("UnfollowActor(%s)", did), &atProtoError{message: "not following this actor"})
	}

	followURI := *profile.Viewer.Following
	_, _, rkey, err := parseATURI(followURI)
	if err != nil {
		return wrapErr(fmt.Sprintf("UnfollowActor(%s)", did), err)
	}

	input := &comatproto.RepoDeleteRecord_Input{
		Collection: "app.bsky.graph.follow",
		Repo:       c.did,
		Rkey:       rkey,
	}

	err = withRetry(ctx, 2, func() error {
		var out comatproto.RepoDeleteRecord_Output
		return c.api.Post(ctx, "com.atproto.repo.deleteRecord", input, &out)
	})
	if err != nil {
		return wrapErr(fmt.Sprintf("UnfollowActor(%s)", did), err)
	}
	return nil
}
