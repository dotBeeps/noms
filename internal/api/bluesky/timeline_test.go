package bluesky

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTimeline(t *testing.T) {
	t.Parallel()
	nextCursor := "cursor123"
	resp := map[string]any{
		"cursor": nextCursor,
		"feed": []map[string]any{
			{"post": map[string]any{
				"uri": "at://did:plc:a/app.bsky.feed.post/1",
				"cid": "cid1",
				"author": map[string]any{
					"did":    "did:plc:a",
					"handle": "alice.bsky.social",
				},
				"record":    map[string]any{"$type": "app.bsky.feed.post", "text": "hello", "createdAt": "2024-01-01T00:00:00Z"},
				"indexedAt": "2024-01-01T00:00:00Z",
			}},
		},
	}

	srv, reqs := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	posts, cursor, err := c.GetTimeline(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cursor != nextCursor {
		t.Errorf("cursor = %q, want %q", cursor, nextCursor)
	}
	if len(posts) != 1 {
		t.Fatalf("len(posts) = %d, want 1", len(posts))
	}
	if posts[0].Post.Uri != "at://did:plc:a/app.bsky.feed.post/1" {
		t.Errorf("post URI = %q", posts[0].Post.Uri)
	}

	// Verify request params
	if len(*reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(*reqs))
	}
	q := (*reqs)[0].URL.Query()
	if q.Get("limit") != "10" {
		t.Errorf("limit param = %q, want %q", q.Get("limit"), "10")
	}
}

func TestGetTimelineCursorPagination(t *testing.T) {
	t.Parallel()
	resp := map[string]any{
		"feed": []map[string]any{},
	}

	srv, reqs := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	_, _, err := c.GetTimeline(context.Background(), "page2cursor", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := (*reqs)[0].URL.Query()
	if q.Get("cursor") != "page2cursor" {
		t.Errorf("cursor param = %q, want %q", q.Get("cursor"), "page2cursor")
	}
}

func TestGetTimelineError(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t, 500, map[string]string{"error": "InternalServerError", "message": "server broke"})
	c := newTestClient(srv)

	_, _, err := c.GetTimeline(context.Background(), "", 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetPost(t *testing.T) {
	t.Parallel()
	resp := map[string]any{
		"posts": []map[string]any{
			{
				"uri": "at://did:plc:a/app.bsky.feed.post/1",
				"cid": "cid1",
				"author": map[string]any{
					"did":    "did:plc:a",
					"handle": "alice.bsky.social",
				},
				"record":    map[string]any{"$type": "app.bsky.feed.post", "text": "hello", "createdAt": "2024-01-01T00:00:00Z"},
				"indexedAt": "2024-01-01T00:00:00Z",
			},
		},
	}

	srv, _ := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	post, err := c.GetPost(context.Background(), "at://did:plc:a/app.bsky.feed.post/1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.Uri != "at://did:plc:a/app.bsky.feed.post/1" {
		t.Errorf("post URI = %q", post.Uri)
	}
}

func TestGetPostNotFound(t *testing.T) {
	t.Parallel()
	resp := map[string]any{"posts": []any{}}
	srv, _ := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	_, err := c.GetPost(context.Background(), "at://did:plc:a/app.bsky.feed.post/missing")
	if err == nil {
		t.Fatal("expected error for missing post")
	}
}

func TestGetPostThread(t *testing.T) {
	t.Parallel()
	resp := map[string]any{
		"thread": map[string]any{
			"$type": "app.bsky.feed.defs#threadViewPost",
			"post": map[string]any{
				"uri":    "at://did:plc:a/app.bsky.feed.post/1",
				"cid":    "cid1",
				"author": map[string]any{"did": "did:plc:a", "handle": "alice.bsky.social"},
				"record": map[string]any{
					"$type":     "app.bsky.feed.post",
					"text":      "thread root",
					"createdAt": "2024-01-01T00:00:00Z",
				},
				"indexedAt": "2024-01-01T00:00:00Z",
			},
		},
	}

	srv, reqs := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	thread, err := c.GetPostThread(context.Background(), "at://did:plc:a/app.bsky.feed.post/1", 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thread == nil {
		t.Fatal("expected non-nil thread")
	}

	q := (*reqs)[0].URL.Query()
	if q.Get("depth") != "6" {
		t.Errorf("depth param = %q, want %q", q.Get("depth"), "6")
	}
}

func TestGetPostThreadError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{"error": "NotFound", "message": "post not found"})
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:test")
	_, err := c.GetPostThread(context.Background(), "at://did:plc:a/app.bsky.feed.post/gone", 6)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
