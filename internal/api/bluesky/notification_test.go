package bluesky

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListNotifications(t *testing.T) {
	t.Parallel()
	nextCursor := "notifcursor"
	resp := map[string]any{
		"cursor": nextCursor,
		"notifications": []map[string]any{
			{
				"uri":       "at://did:plc:a/app.bsky.feed.like/1",
				"cid":       "cid1",
				"author":    map[string]any{"did": "did:plc:a", "handle": "alice.bsky.social"},
				"reason":    "like",
				"isRead":    false,
				"indexedAt": "2024-01-01T00:00:00Z",
				"record":    map[string]any{"$type": "app.bsky.feed.like", "createdAt": "2024-01-01T00:00:00Z", "subject": map[string]any{"uri": "at://did:plc:b/app.bsky.feed.post/1", "cid": "cid2"}},
			},
		},
	}

	srv, reqs := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	notifs, cursor, err := c.ListNotifications(context.Background(), "", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cursor != nextCursor {
		t.Errorf("cursor = %q, want %q", cursor, nextCursor)
	}
	if len(notifs) != 1 {
		t.Fatalf("len(notifs) = %d, want 1", len(notifs))
	}
	if notifs[0].Reason != "like" {
		t.Errorf("reason = %q, want %q", notifs[0].Reason, "like")
	}

	q := (*reqs)[0].URL.Query()
	if q.Get("limit") != "20" {
		t.Errorf("limit param = %q, want %q", q.Get("limit"), "20")
	}
}

func TestListNotificationsPagination(t *testing.T) {
	t.Parallel()
	resp := map[string]any{
		"notifications": []any{},
	}
	srv, reqs := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	_, _, err := c.ListNotifications(context.Background(), "page2", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := (*reqs)[0].URL.Query()
	if q.Get("cursor") != "page2" {
		t.Errorf("cursor param = %q, want %q", q.Get("cursor"), "page2")
	}
}

func TestGetUnreadCount(t *testing.T) {
	t.Parallel()
	resp := map[string]any{"count": 42}

	srv, _ := newTestServer(t, 200, resp)
	c := newTestClient(srv)

	count, err := c.GetUnreadCount(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Errorf("count = %d, want 42", count)
	}
}

func TestGetUnreadCountError(t *testing.T) {
	t.Parallel()
	srv, _ := newTestServer(t, 500, map[string]string{"error": "InternalServerError"})
	c := newTestClient(srv)

	_, err := c.GetUnreadCount(context.Background())
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestMarkNotificationsRead(t *testing.T) {
	t.Parallel()
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.Client(), srv.URL, "did:plc:test")

	seenAt := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	err := c.MarkNotificationsRead(context.Background(), seenAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedBody == nil {
		t.Fatal("expected request body")
	}
	if sa, ok := capturedBody["seenAt"].(string); !ok || sa != "2024-06-15T12:00:00Z" {
		t.Errorf("seenAt = %v, want %q", capturedBody["seenAt"], "2024-06-15T12:00:00Z")
	}
}
