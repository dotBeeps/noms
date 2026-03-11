package bluesky

import (
	"context"
	"testing"
)

func TestSearchPosts(t *testing.T) {
	t.Parallel()
	nextCursor := "searchcursor"
	resp := map[string]any{
		"cursor": nextCursor,
		"posts": []map[string]any{
			{
				"uri":       "at://did:plc:a/app.bsky.feed.post/1",
				"cid":       "cid1",
				"author":    map[string]any{"did": "did:plc:a", "handle": "alice.bsky.social"},
				"record":    map[string]any{"$type": "app.bsky.feed.post", "text": "searchable post", "createdAt": "2024-01-01T00:00:00Z"},
				"indexedAt": "2024-01-01T00:00:00Z",
			},
		},
	}

	srv, reqs := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	posts, cursor, err := c.SearchPosts(context.Background(), "searchable", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cursor != nextCursor {
		t.Errorf("cursor = %q, want %q", cursor, nextCursor)
	}
	if len(posts) != 1 {
		t.Fatalf("len(posts) = %d, want 1", len(posts))
	}

	q := (*reqs)[0].URL.Query()
	if q.Get("q") != "searchable" {
		t.Errorf("q param = %q, want searchable", q.Get("q"))
	}
}

func TestSearchPostsError(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t, 500, map[string]string{"error": "InternalServerError"})
	c := newTestClient(srv)

	_, _, err := c.SearchPosts(context.Background(), "query", "", 10)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestSearchActors(t *testing.T) {
	t.Parallel()
	nextCursor := "actorcursor"
	resp := map[string]any{
		"cursor": nextCursor,
		"actors": []map[string]any{
			{
				"did":    "did:plc:alice",
				"handle": "alice.bsky.social",
			},
		},
	}

	srv, reqs := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	actors, cursor, err := c.SearchActors(context.Background(), "alice", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cursor != nextCursor {
		t.Errorf("cursor = %q, want %q", cursor, nextCursor)
	}
	if len(actors) != 1 {
		t.Fatalf("len(actors) = %d, want 1", len(actors))
	}
	if actors[0].Handle != "alice.bsky.social" {
		t.Errorf("handle = %q", actors[0].Handle)
	}

	q := (*reqs)[0].URL.Query()
	if q.Get("q") != "alice" {
		t.Errorf("q param = %q", q.Get("q"))
	}
}

func TestSearchActorsError(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t, 500, map[string]string{"error": "InternalServerError"})
	c := newTestClient(srv)

	_, _, err := c.SearchActors(context.Background(), "query", "", 10)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestSearchPostsCursorPagination(t *testing.T) {
	t.Parallel()
	resp := map[string]any{
		"posts": []any{},
	}
	srv, reqs := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	_, _, err := c.SearchPosts(context.Background(), "query", "page2", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := (*reqs)[0].URL.Query()
	if q.Get("cursor") != "page2" {
		t.Errorf("cursor param = %q, want page2", q.Get("cursor"))
	}
	if q.Get("limit") != "5" {
		t.Errorf("limit param = %q, want 5", q.Get("limit"))
	}
}
