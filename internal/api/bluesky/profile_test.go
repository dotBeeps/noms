package bluesky

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetProfile(t *testing.T) {
	t.Parallel()
	resp := map[string]any{
		"did":            "did:plc:alice",
		"handle":         "alice.bsky.social",
		"displayName":    "Alice",
		"followersCount": 100,
		"followsCount":   50,
		"postsCount":     200,
	}

	srv, reqs := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	profile, err := c.GetProfile(context.Background(), "alice.bsky.social")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Did != "did:plc:alice" {
		t.Errorf("DID = %q, want %q", profile.Did, "did:plc:alice")
	}
	if profile.Handle != "alice.bsky.social" {
		t.Errorf("Handle = %q", profile.Handle)
	}

	q := (*reqs)[0].URL.Query()
	if q.Get("actor") != "alice.bsky.social" {
		t.Errorf("actor param = %q", q.Get("actor"))
	}
}

func TestGetProfileByDID(t *testing.T) {
	t.Parallel()
	resp := map[string]any{
		"did":    "did:plc:alice",
		"handle": "alice.bsky.social",
	}

	srv, reqs := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	_, err := c.GetProfile(context.Background(), "did:plc:alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := (*reqs)[0].URL.Query()
	if q.Get("actor") != "did:plc:alice" {
		t.Errorf("actor param = %q", q.Get("actor"))
	}
}

func TestGetProfileError(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t, 404, map[string]string{"error": "NotFound"})
	c := newTestClient(srv)

	_, err := c.GetProfile(context.Background(), "nobody.bsky.social")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGetAuthorFeed(t *testing.T) {
	t.Parallel()
	nextCursor := "authorcursor"
	resp := map[string]any{
		"cursor": nextCursor,
		"feed": []map[string]any{
			{"post": map[string]any{
				"uri":       "at://did:plc:a/app.bsky.feed.post/1",
				"cid":       "cid1",
				"author":    map[string]any{"did": "did:plc:a", "handle": "alice.bsky.social"},
				"record":    map[string]any{"$type": "app.bsky.feed.post", "text": "my post", "createdAt": "2024-01-01T00:00:00Z"},
				"indexedAt": "2024-01-01T00:00:00Z",
			}},
		},
	}

	srv, _ := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	posts, cursor, err := c.GetAuthorFeed(context.Background(), "did:plc:a", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cursor != nextCursor {
		t.Errorf("cursor = %q, want %q", cursor, nextCursor)
	}
	if len(posts) != 1 {
		t.Errorf("len(posts) = %d, want 1", len(posts))
	}
}

func TestFollowActor(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Verify it's a POST to createRecord
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if col, ok := body["collection"].(string); !ok || col != "app.bsky.graph.follow" {
			t.Errorf("collection = %v, want app.bsky.graph.follow", body["collection"])
		}

		if err := json.NewEncoder(w).Encode(map[string]any{
			"uri": "at://did:plc:testuser123/app.bsky.graph.follow/abc",
			"cid": "cidfollow",
		}); err != nil {
			t.Fatalf("encode response body: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:testuser123")
	err := c.FollowActor(context.Background(), "did:plc:target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnfollowActor(t *testing.T) {
	t.Parallel()
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++

		if callCount == 1 {
			// First call: getProfile returns viewer state with following URI
			if err := json.NewEncoder(w).Encode(map[string]any{
				"did":    "did:plc:target",
				"handle": "target.bsky.social",
				"viewer": map[string]any{
					"following": "at://did:plc:testuser123/app.bsky.graph.follow/rkey123",
				},
			}); err != nil {
				t.Fatalf("encode profile response: %v", err)
			}
			return
		}

		// Second call: deleteRecord
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode delete body: %v", err)
		}
		if rkey, ok := body["rkey"].(string); !ok || rkey != "rkey123" {
			t.Errorf("rkey = %v, want rkey123", body["rkey"])
		}
		if err := json.NewEncoder(w).Encode(map[string]any{}); err != nil {
			t.Fatalf("encode delete response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:testuser123")
	err := c.UnfollowActor(context.Background(), "did:plc:target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnfollowActorNotFollowing(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"did":    "did:plc:target",
			"handle": "target.bsky.social",
			"viewer": map[string]any{},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:testuser123")
	err := c.UnfollowActor(context.Background(), "did:plc:target")
	if err == nil {
		t.Fatal("expected error when not following")
	}
	if !strings.Contains(err.Error(), "not following") {
		t.Errorf("error = %q, should mention 'not following'", err.Error())
	}
}
