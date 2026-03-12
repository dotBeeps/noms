package voresky

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// MaxEnrichDIDs is the maximum number of DIDs accepted by the enrich endpoint.
const MaxEnrichDIDs = 200

// CropParams describes a crop rectangle as normalised 0.0–1.0 fractions.
type CropParams struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// BaseAvatar is the character's own avatar image with optional crop and HLS
// stream URL (for animated avatars).
type BaseAvatar struct {
	URL    string      `json:"url"`
	Crop   *CropParams `json:"crop,omitempty"`
	HlsURL string      `json:"hlsUrl,omitempty"`
}

// PredatorInfo identifies the predator in a caught interaction.
type PredatorInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	UserDID string `json:"userDid"`
}

// CaughtState is the enrichment state for a single DID. It contains the
// information needed to determine how to display a user's avatar — whether
// they're caught (showing a snapshot blob), have a Voresky character avatar,
// or should fall through to their Bluesky avatar.
type CaughtState struct {
	BlobHash         string        `json:"blobHash"`
	CurrentNodeID    string        `json:"currentNodeId"`
	Predator         *PredatorInfo `json:"predator,omitempty"`
	Phase            string        `json:"phase,omitempty"`
	BaseAvatar       *BaseAvatar   `json:"baseAvatar,omitempty"`
	BaseBannerURL    string        `json:"baseBannerUrl,omitempty"`
	BaseBannerHlsURL string        `json:"baseBannerHlsUrl,omitempty"`
	BaseBannerCrop   *CropParams   `json:"baseBannerCrop,omitempty"`
	IsPet            bool          `json:"isPet,omitempty"`
	HasVoreskyLabel  bool          `json:"hasVoreskyLabel,omitempty"`
	UpdatedAt        string        `json:"updatedAt,omitempty"`
}

// EnrichResponse is the shape returned by POST /api/game/enrich.
type EnrichResponse struct {
	CaughtOverrides    map[string]*CaughtState `json:"caughtOverrides"`
	BlurredPredatorIDs []string                `json:"blurredPredatorIds"`
}

// enrichRequest is the request body for POST /api/game/enrich.
type enrichRequest struct {
	DIDs        []string          `json:"dids"`
	KnownStates map[string]string `json:"knownStates,omitempty"`
}

// SnapshotNode is a single node inside a snapshot blob. We only decode the
// fields needed for avatar resolution.
type SnapshotNode struct {
	ID        string `json:"id"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	BannerURL string `json:"bannerUrl,omitempty"`
}

// SnapshotBlob is the response from GET /api/game/snapshots/{hash}.
// Blobs are immutable and can be cached indefinitely.
type SnapshotBlob struct {
	Nodes []SnapshotNode `json:"nodes"`
	// connections is ignored; we only need nodes for avatar resolution.
}

// Enrich calls POST /api/game/enrich with a batch of DIDs and returns their
// caught-state overrides. The caller may pass knownStates (DID → blobHash) so
// the server can skip unchanged entries.
func (c *VoreskyClient) Enrich(ctx context.Context, dids []string, knownStates map[string]string) (*EnrichResponse, error) {
	if len(dids) > MaxEnrichDIDs {
		return nil, fmt.Errorf("enrich: too many DIDs (%d > %d)", len(dids), MaxEnrichDIDs)
	}
	if len(dids) == 0 {
		return &EnrichResponse{CaughtOverrides: map[string]*CaughtState{}}, nil
	}

	body := enrichRequest{
		DIDs:        dids,
		KnownStates: knownStates,
	}

	resp, err := c.Post(ctx, "/api/game/enrich", body)
	if err != nil {
		return nil, fmt.Errorf("enrich: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, ParseError(resp)
	}

	var er EnrichResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return nil, fmt.Errorf("decode enrich response: %w", err)
	}
	if er.CaughtOverrides == nil {
		er.CaughtOverrides = map[string]*CaughtState{}
	}
	return &er, nil
}

// GetSnapshot fetches a snapshot blob by its content-addressed hash.
// Calls GET /api/game/snapshots/{hash}. The response is immutable and should
// be cached permanently by the caller.
func (c *VoreskyClient) GetSnapshot(ctx context.Context, hash string) (*SnapshotBlob, error) {
	resp, err := c.Get(ctx, "/api/game/snapshots/"+hash)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, ParseError(resp)
	}

	var sb SnapshotBlob
	if err := json.NewDecoder(resp.Body).Decode(&sb); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	return &sb, nil
}

// ResolveNodeAvatar finds the avatar URL for a specific node within a snapshot
// blob. Returns an empty string if the node is not found or has no avatar.
func ResolveNodeAvatar(blob *SnapshotBlob, nodeID string) string {
	if blob == nil || nodeID == "" {
		return ""
	}
	for _, node := range blob.Nodes {
		if node.ID == nodeID {
			return node.AvatarURL
		}
	}
	return ""
}
