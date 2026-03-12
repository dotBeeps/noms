package bluesky

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreatePost(t *testing.T) {
	t.Parallel()
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"uri": "at://did:plc:testuser123/app.bsky.feed.post/abc123",
			"cid": "cidabc123",
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:testuser123")

	uri, cid, err := c.CreatePost(context.Background(), "Hello world!", nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uri != "at://did:plc:testuser123/app.bsky.feed.post/abc123" {
		t.Errorf("uri = %q", uri)
	}
	if cid != "cidabc123" {
		t.Errorf("cid = %q", cid)
	}

	// Verify the record in the body
	if capturedBody == nil {
		t.Fatal("no request body captured")
	}
	if col, _ := capturedBody["collection"].(string); col != "app.bsky.feed.post" {
		t.Errorf("collection = %q, want app.bsky.feed.post", col)
	}
	if repo, _ := capturedBody["repo"].(string); repo != "did:plc:testuser123" {
		t.Errorf("repo = %q", repo)
	}
}

func TestCreatePostError(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t, 500, map[string]string{"error": "InternalServerError"})
	c := newTestClient(srv)

	_, _, err := c.CreatePost(context.Background(), "test", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeletePost(t *testing.T) {
	t.Parallel()
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:testuser123")

	err := c.DeletePost(context.Background(), "at://did:plc:testuser123/app.bsky.feed.post/abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rkey, _ := capturedBody["rkey"].(string); rkey != "abc123" {
		t.Errorf("rkey = %q, want abc123", rkey)
	}
	if col, _ := capturedBody["collection"].(string); col != "app.bsky.feed.post" {
		t.Errorf("collection = %q", col)
	}
}

func TestDeletePostInvalidURI(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t, 200, nil)
	c := newTestClient(srv)

	err := c.DeletePost(context.Background(), "not-an-at-uri")
	if err == nil {
		t.Fatal("expected error for invalid URI")
	}
}

func TestDeletePostWrongCollection(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t, 200, nil)
	c := newTestClient(srv)

	err := c.DeletePost(context.Background(), "at://did:plc:a/app.bsky.feed.like/abc")
	if err == nil {
		t.Fatal("expected error for wrong collection")
	}
	if !strings.Contains(err.Error(), "not a post URI") {
		t.Errorf("error = %q, should mention 'not a post URI'", err.Error())
	}
}

func TestLike(t *testing.T) {
	t.Parallel()
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"uri": "at://did:plc:testuser123/app.bsky.feed.like/like1",
			"cid": "cidlike1",
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:testuser123")

	likeURI, err := c.Like(context.Background(), "at://did:plc:a/app.bsky.feed.post/1", "cidpost1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if likeURI != "at://did:plc:testuser123/app.bsky.feed.like/like1" {
		t.Errorf("likeURI = %q", likeURI)
	}

	if col, _ := capturedBody["collection"].(string); col != "app.bsky.feed.like" {
		t.Errorf("collection = %q, want app.bsky.feed.like", col)
	}
}

func TestUnlike(t *testing.T) {
	t.Parallel()
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:testuser123")

	err := c.Unlike(context.Background(), "at://did:plc:testuser123/app.bsky.feed.like/like1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rkey, _ := capturedBody["rkey"].(string); rkey != "like1" {
		t.Errorf("rkey = %q", rkey)
	}
}

func TestUnlikeWrongCollection(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t, 200, nil)
	c := newTestClient(srv)

	err := c.Unlike(context.Background(), "at://did:plc:a/app.bsky.feed.post/abc")
	if err == nil {
		t.Fatal("expected error for wrong collection")
	}
}

func TestRepost(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"uri": "at://did:plc:testuser123/app.bsky.feed.repost/rp1",
			"cid": "cidrepost1",
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:testuser123")

	repostURI, err := c.Repost(context.Background(), "at://did:plc:a/app.bsky.feed.post/1", "cidpost1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repostURI != "at://did:plc:testuser123/app.bsky.feed.repost/rp1" {
		t.Errorf("repostURI = %q", repostURI)
	}
}

func TestUnRepost(t *testing.T) {
	t.Parallel()
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:testuser123")

	err := c.UnRepost(context.Background(), "at://did:plc:testuser123/app.bsky.feed.repost/rp1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rkey, _ := capturedBody["rkey"].(string); rkey != "rp1" {
		t.Errorf("rkey = %q, want rp1", rkey)
	}
}

func TestUnRepostWrongCollection(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t, 200, nil)
	c := newTestClient(srv)

	err := c.UnRepost(context.Background(), "at://did:plc:a/app.bsky.feed.post/abc")
	if err == nil {
		t.Fatal("expected error for wrong collection")
	}
}
