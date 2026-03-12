package voresky

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnrich_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/game/enrich" {
			t.Errorf("path: got %s, want /api/game/enrich", r.URL.Path)
		}

		var req enrichRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if len(req.DIDs) != 2 {
			t.Errorf("dids count: got %d, want 2", len(req.DIDs))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(EnrichResponse{
			CaughtOverrides: map[string]*CaughtState{
				"did:plc:user1": {
					BlobHash:      "abc123hash",
					CurrentNodeID: "node-1",
					Phase:         "fatal",
					BaseAvatar:    &BaseAvatar{URL: "https://example.com/avatar1.png"},
				},
				"did:plc:user2": {
					BaseAvatar: &BaseAvatar{
						URL:  "https://example.com/avatar2.png",
						Crop: &CropParams{X: 0.1, Y: 0.2, Width: 0.5, Height: 0.5},
					},
					IsPet:           true,
					HasVoreskyLabel: true,
				},
			},
			BlurredPredatorIDs: []string{"did:plc:predator1"},
		})
	}))
	defer srv.Close()

	auth := newAuthWithCookie(srv.URL, testCookie, testDID)
	client := NewVoreskyClient(srv.URL, auth)

	resp, err := client.Enrich(context.Background(), []string{"did:plc:user1", "did:plc:user2"}, nil)
	if err != nil {
		t.Fatalf("Enrich: unexpected error: %v", err)
	}

	if len(resp.CaughtOverrides) != 2 {
		t.Fatalf("overrides count: got %d, want 2", len(resp.CaughtOverrides))
	}

	u1 := resp.CaughtOverrides["did:plc:user1"]
	if u1 == nil {
		t.Fatal("user1 override missing")
	}
	if u1.BlobHash != "abc123hash" {
		t.Errorf("user1 blobHash: got %q, want %q", u1.BlobHash, "abc123hash")
	}
	if u1.Phase != "fatal" {
		t.Errorf("user1 phase: got %q, want %q", u1.Phase, "fatal")
	}

	u2 := resp.CaughtOverrides["did:plc:user2"]
	if u2 == nil {
		t.Fatal("user2 override missing")
	}
	if u2.BaseAvatar == nil || u2.BaseAvatar.URL != "https://example.com/avatar2.png" {
		t.Error("user2 baseAvatar URL mismatch")
	}
	if !u2.IsPet {
		t.Error("user2 should be a pet")
	}

	if len(resp.BlurredPredatorIDs) != 1 || resp.BlurredPredatorIDs[0] != "did:plc:predator1" {
		t.Errorf("blurred predators: got %v", resp.BlurredPredatorIDs)
	}
}

func TestEnrich_EmptyDIDs(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called for empty DIDs")
	}))
	defer srv.Close()

	auth := newAuthWithCookie(srv.URL, testCookie, testDID)
	client := NewVoreskyClient(srv.URL, auth)

	resp, err := client.Enrich(context.Background(), []string{}, nil)
	if err != nil {
		t.Fatalf("Enrich empty: unexpected error: %v", err)
	}
	if resp.CaughtOverrides == nil {
		t.Error("expected non-nil empty map")
	}
	if len(resp.CaughtOverrides) != 0 {
		t.Errorf("expected 0 overrides, got %d", len(resp.CaughtOverrides))
	}
}

func TestEnrich_ExceedsMaxDIDs(t *testing.T) {
	t.Parallel()

	auth := newAuthWithCookie("http://unused", testCookie, testDID)
	client := NewVoreskyClient("http://unused", auth)

	dids := make([]string, MaxEnrichDIDs+1)
	for i := range dids {
		dids[i] = "did:plc:filler"
	}

	_, err := client.Enrich(context.Background(), dids, nil)
	if err == nil {
		t.Fatal("expected error for too many DIDs")
	}
}

func TestEnrich_WithKnownStates(t *testing.T) {
	t.Parallel()

	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"caughtOverrides":{},"blurredPredatorIds":[]}`))
	}))
	defer srv.Close()

	auth := newAuthWithCookie(srv.URL, testCookie, testDID)
	client := NewVoreskyClient(srv.URL, auth)

	known := map[string]string{"did:plc:cached": "oldhash"}
	_, err := client.Enrich(context.Background(), []string{"did:plc:cached"}, known)
	if err != nil {
		t.Fatalf("Enrich with known states: %v", err)
	}

	var req enrichRequest
	if err := json.Unmarshal(receivedBody, &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if req.KnownStates == nil || req.KnownStates["did:plc:cached"] != "oldhash" {
		t.Errorf("knownStates not sent correctly: %v", req.KnownStates)
	}
}

func TestEnrich_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer srv.Close()

	auth := newAuthWithCookie(srv.URL, testCookie, testDID)
	client := NewVoreskyClient(srv.URL, auth)

	_, err := client.Enrich(context.Background(), []string{"did:plc:user1"}, nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGetSnapshot_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/game/snapshots/testhash123" {
			t.Errorf("path: got %s, want /api/game/snapshots/testhash123", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SnapshotBlob{
			Nodes: []SnapshotNode{
				{ID: "node-1", AvatarURL: "https://example.com/pred-avatar.png"},
				{ID: "node-2", AvatarURL: "https://example.com/deep-avatar.png"},
				{ID: "node-3"},
			},
		})
	}))
	defer srv.Close()

	auth := newAuthWithCookie(srv.URL, testCookie, testDID)
	client := NewVoreskyClient(srv.URL, auth)

	blob, err := client.GetSnapshot(context.Background(), "testhash123")
	if err != nil {
		t.Fatalf("GetSnapshot: unexpected error: %v", err)
	}

	if len(blob.Nodes) != 3 {
		t.Fatalf("nodes count: got %d, want 3", len(blob.Nodes))
	}
	if blob.Nodes[0].AvatarURL != "https://example.com/pred-avatar.png" {
		t.Errorf("node-1 avatar: got %q", blob.Nodes[0].AvatarURL)
	}
}

func TestGetSnapshot_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"snapshot not found"}`))
	}))
	defer srv.Close()

	auth := newAuthWithCookie(srv.URL, testCookie, testDID)
	client := NewVoreskyClient(srv.URL, auth)

	_, err := client.GetSnapshot(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestResolveNodeAvatar(t *testing.T) {
	t.Parallel()

	blob := &SnapshotBlob{
		Nodes: []SnapshotNode{
			{ID: "node-1", AvatarURL: "https://example.com/avatar1.png"},
			{ID: "node-2", AvatarURL: "https://example.com/avatar2.png"},
			{ID: "node-3"},
		},
	}

	tests := []struct {
		name   string
		blob   *SnapshotBlob
		nodeID string
		want   string
	}{
		{"found", blob, "node-1", "https://example.com/avatar1.png"},
		{"found second", blob, "node-2", "https://example.com/avatar2.png"},
		{"empty avatar", blob, "node-3", ""},
		{"not found", blob, "node-99", ""},
		{"nil blob", nil, "node-1", ""},
		{"empty nodeID", blob, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveNodeAvatar(tt.blob, tt.nodeID)
			if got != tt.want {
				t.Errorf("ResolveNodeAvatar(%q): got %q, want %q", tt.nodeID, got, tt.want)
			}
		})
	}
}
