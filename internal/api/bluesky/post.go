package bluesky

import (
	"context"
	"fmt"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
)

// CreatePost creates a new post with optional facets, reply ref, and embed.
// Returns the post URI and CID.
func (c *Client) CreatePost(ctx context.Context, text string, facets []*bsky.RichtextFacet, reply *bsky.FeedPost_ReplyRef, embed *bsky.FeedPost_Embed) (string, string, error) {
	record := &bsky.FeedPost{
		LexiconTypeID: "app.bsky.feed.post",
		Text:          text,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	if len(facets) > 0 {
		record.Facets = facets
	}
	if reply != nil {
		record.Reply = reply
	}
	if embed != nil {
		record.Embed = embed
	}

	input := &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       c.did,
		Record:     &lexutil.LexiconTypeDecoder{Val: record},
	}

	var out comatproto.RepoCreateRecord_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Post(ctx, "com.atproto.repo.createRecord", input, &out)
	})
	if err != nil {
		return "", "", wrapErr("CreatePost", err)
	}
	return out.Uri, out.Cid, nil
}

// DeletePost deletes a post by its AT-URI.
func (c *Client) DeletePost(ctx context.Context, uri string) error {
	repo, collection, rkey, err := parseATURI(uri)
	if err != nil {
		return wrapErr("DeletePost", err)
	}
	if collection != "app.bsky.feed.post" {
		return wrapErr("DeletePost", fmt.Errorf("not a post URI: %s", uri))
	}

	input := &comatproto.RepoDeleteRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       repo,
		Rkey:       rkey,
	}

	err = withRetry(ctx, 2, func() error {
		var out comatproto.RepoDeleteRecord_Output
		return c.api.Post(ctx, "com.atproto.repo.deleteRecord", input, &out)
	})
	if err != nil {
		return wrapErr("DeletePost", err)
	}
	return nil
}

// Like creates a like record for the given post URI and CID.
// Returns the like record URI.
func (c *Client) Like(ctx context.Context, uri, cid string) (string, error) {
	record := &bsky.FeedLike{
		LexiconTypeID: "app.bsky.feed.like",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Subject: &comatproto.RepoStrongRef{
			Uri: uri,
			Cid: cid,
		},
	}

	input := &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.like",
		Repo:       c.did,
		Record:     &lexutil.LexiconTypeDecoder{Val: record},
	}

	var out comatproto.RepoCreateRecord_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Post(ctx, "com.atproto.repo.createRecord", input, &out)
	})
	if err != nil {
		return "", wrapErr("Like", err)
	}
	return out.Uri, nil
}

// Unlike removes a like by its AT-URI.
func (c *Client) Unlike(ctx context.Context, likeURI string) error {
	_, collection, rkey, err := parseATURI(likeURI)
	if err != nil {
		return wrapErr("Unlike", err)
	}
	if collection != "app.bsky.feed.like" {
		return wrapErr("Unlike", fmt.Errorf("not a like URI: %s", likeURI))
	}

	input := &comatproto.RepoDeleteRecord_Input{
		Collection: "app.bsky.feed.like",
		Repo:       c.did,
		Rkey:       rkey,
	}

	err = withRetry(ctx, 2, func() error {
		var out comatproto.RepoDeleteRecord_Output
		return c.api.Post(ctx, "com.atproto.repo.deleteRecord", input, &out)
	})
	if err != nil {
		return wrapErr("Unlike", err)
	}
	return nil
}

// Repost creates a repost record for the given post URI and CID.
// Returns the repost record URI.
func (c *Client) Repost(ctx context.Context, uri, cid string) (string, error) {
	record := &bsky.FeedRepost{
		LexiconTypeID: "app.bsky.feed.repost",
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Subject: &comatproto.RepoStrongRef{
			Uri: uri,
			Cid: cid,
		},
	}

	input := &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.repost",
		Repo:       c.did,
		Record:     &lexutil.LexiconTypeDecoder{Val: record},
	}

	var out comatproto.RepoCreateRecord_Output
	err := withRetry(ctx, 2, func() error {
		return c.api.Post(ctx, "com.atproto.repo.createRecord", input, &out)
	})
	if err != nil {
		return "", wrapErr("Repost", err)
	}
	return out.Uri, nil
}

// UnRepost removes a repost by its AT-URI.
func (c *Client) UnRepost(ctx context.Context, repostURI string) error {
	_, collection, rkey, err := parseATURI(repostURI)
	if err != nil {
		return wrapErr("UnRepost", err)
	}
	if collection != "app.bsky.feed.repost" {
		return wrapErr("UnRepost", fmt.Errorf("not a repost URI: %s", repostURI))
	}

	input := &comatproto.RepoDeleteRecord_Input{
		Collection: "app.bsky.feed.repost",
		Repo:       c.did,
		Rkey:       rkey,
	}

	err = withRetry(ctx, 2, func() error {
		var out comatproto.RepoDeleteRecord_Output
		return c.api.Post(ctx, "com.atproto.repo.deleteRecord", input, &out)
	})
	if err != nil {
		return wrapErr("UnRepost", err)
	}
	return nil
}
