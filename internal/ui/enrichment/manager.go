// Package enrichment manages Voresky caught-state enrichment for visible users.
// It caches enrichment data at the app level (views are created/destroyed) and
// resolves effective avatar URLs by combining caught-state, character avatar,
// and original Bluesky avatar data.
package enrichment

import (
	"github.com/dotBeeps/noms/internal/api/voresky"
)

// Manager caches enrichment state for visible users and resolves effective
// avatar overrides. It is NOT thread-safe — callers must only access it from
// the BubbleTea Update goroutine.
type Manager struct {
	// cache maps DID → full CaughtState from the enrich endpoint.
	cache map[string]*voresky.CaughtState

	// knownHashes maps DID → blobHash for incremental enrichment.
	// Sent as knownStates so the server can skip unchanged entries.
	knownHashes map[string]string

	// snapshots caches snapshot blobs by content hash. These are immutable
	// and never evicted.
	snapshots map[string]*voresky.SnapshotBlob

	// resolvedAvatars caches DID → resolved avatar URL for caught users
	// whose snapshots have been fetched and resolved. This avoids re-resolving
	// on every BuildAvatarOverrides call.
	resolvedAvatars map[string]string
}

// New creates an empty enrichment manager.
func New() *Manager {
	return &Manager{
		cache:           make(map[string]*voresky.CaughtState),
		knownHashes:     make(map[string]string),
		snapshots:       make(map[string]*voresky.SnapshotBlob),
		resolvedAvatars: make(map[string]string),
	}
}

// NeedEnrichment returns the subset of dids not already present in the cache.
func (m *Manager) NeedEnrichment(dids []string) []string {
	if len(dids) == 0 {
		return nil
	}
	var unknown []string
	for _, did := range dids {
		if _, ok := m.cache[did]; !ok {
			unknown = append(unknown, did)
		}
	}
	return unknown
}

// Store merges enrich response overrides into the cache. For each DID, it
// updates the cached CaughtState and tracks the blobHash for incremental
// enrichment. If a caught user's blobHash changes, any previously resolved
// avatar is cleared so it will be re-resolved from the new snapshot.
func (m *Manager) Store(overrides map[string]*voresky.CaughtState) {
	for did, state := range overrides {
		prev := m.cache[did]
		m.cache[did] = state

		newHash := ""
		if state != nil {
			newHash = state.BlobHash
		}

		// Track known hash for incremental enrichment.
		if newHash != "" {
			m.knownHashes[did] = newHash
		} else {
			delete(m.knownHashes, did)
		}

		// Clear resolved avatar if hash changed (needs re-resolve).
		prevHash := ""
		if prev != nil {
			prevHash = prev.BlobHash
		}
		if newHash != prevHash {
			delete(m.resolvedAvatars, did)
		}
	}
}

// StoreSnapshot caches an immutable snapshot blob by its content hash.
func (m *Manager) StoreSnapshot(hash string, blob *voresky.SnapshotBlob) {
	if hash != "" && blob != nil {
		m.snapshots[hash] = blob
	}
}

// GetCaughtState returns the cached CaughtState for a DID, or nil.
func (m *Manager) GetCaughtState(did string) *voresky.CaughtState {
	return m.cache[did]
}

// KnownStates returns a copy of the DID → blobHash map for passing to the
// enrich endpoint as knownStates. The copy is safe for use in a goroutine.
func (m *Manager) KnownStates() map[string]string {
	if len(m.knownHashes) == 0 {
		return nil
	}
	cp := make(map[string]string, len(m.knownHashes))
	for k, v := range m.knownHashes {
		cp[k] = v
	}
	return cp
}

// PendingSnapshots returns a list of (DID, blobHash, currentNodeID) tuples for
// caught users whose snapshot blobs have not yet been fetched and resolved.
type PendingSnapshot struct {
	DID           string
	BlobHash      string
	CurrentNodeID string
}

// PendingSnapshots returns caught users whose snapshot blobs have not yet been
// fetched. It is a pure query — it does not mutate any state. Call
// ResolveSnapshots first to resolve users whose blobs are already cached.
func (m *Manager) PendingSnapshots() []PendingSnapshot {
	var pending []PendingSnapshot
	for did, state := range m.cache {
		if state == nil || state.BlobHash == "" {
			continue
		}
		if _, ok := m.resolvedAvatars[did]; ok {
			continue
		}
		if _, ok := m.snapshots[state.BlobHash]; ok {
			continue // blob cached but not yet resolved; caller should call ResolveSnapshots
		}
		pending = append(pending, PendingSnapshot{
			DID:           did,
			BlobHash:      state.BlobHash,
			CurrentNodeID: state.CurrentNodeID,
		})
	}
	return pending
}

// ResolveSnapshots resolves avatar URLs for caught users using cached snapshot
// blobs. Call this after StoreSnapshot to resolve pending caught avatars.
func (m *Manager) ResolveSnapshots() {
	for did, state := range m.cache {
		if state == nil || state.BlobHash == "" {
			continue
		}
		if _, ok := m.resolvedAvatars[did]; ok {
			continue
		}
		if blob, ok := m.snapshots[state.BlobHash]; ok {
			url := voresky.ResolveNodeAvatar(blob, state.CurrentNodeID)
			if url != "" {
				m.resolvedAvatars[did] = url
			}
		}
	}
}

// BuildAvatarOverrides builds the complete DID → avatar URL map. For each
// cached DID:
//   - Caught with resolved snapshot avatar → use that
//   - Not caught but has baseAvatar.URL → use character avatar
//   - Otherwise → not in map (falls through to Bluesky avatar)
//
// The own-user override is always applied. If the own user is also in the
// enrichment cache, the enrichment data takes precedence (e.g. if caught).
func (m *Manager) BuildAvatarOverrides(ownDID, ownCharAvatar string) map[string]string {
	overrides := make(map[string]string)

	// Apply own-character override as baseline.
	if ownDID != "" && ownCharAvatar != "" {
		overrides[ownDID] = ownCharAvatar
	}

	// Layer enrichment data on top.
	for did, state := range m.cache {
		if state == nil {
			continue
		}

		// Caught with resolved avatar? Use it.
		if state.BlobHash != "" {
			if url, ok := m.resolvedAvatars[did]; ok && url != "" {
				overrides[did] = url
				continue
			}
			// Caught but snapshot not yet resolved — don't override (show
			// Bluesky avatar until resolved rather than stale data).
			continue
		}

		// Not caught — use character's base avatar if available.
		if state.BaseAvatar != nil && state.BaseAvatar.URL != "" {
			overrides[did] = state.BaseAvatar.URL
		}
	}

	return overrides
}
