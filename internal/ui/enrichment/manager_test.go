package enrichment

import (
	"testing"

	"github.com/dotBeeps/noms/internal/api/voresky"
)

func TestNew(t *testing.T) {
	m := New()
	if m.cache == nil || m.knownHashes == nil || m.snapshots == nil || m.resolvedAvatars == nil {
		t.Fatal("New() should initialise all internal maps")
	}
}

func TestNeedEnrichment_AllNew(t *testing.T) {
	m := New()
	dids := []string{"did:plc:aaa", "did:plc:bbb"}
	got := m.NeedEnrichment(dids)
	if len(got) != 2 {
		t.Fatalf("expected 2 unknown, got %d", len(got))
	}
}

func TestNeedEnrichment_SomeKnown(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = &voresky.CaughtState{}
	got := m.NeedEnrichment([]string{"did:plc:aaa", "did:plc:bbb"})
	if len(got) != 1 || got[0] != "did:plc:bbb" {
		t.Fatalf("expected [did:plc:bbb], got %v", got)
	}
}

func TestNeedEnrichment_AllKnown(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = &voresky.CaughtState{}
	m.cache["did:plc:bbb"] = &voresky.CaughtState{}
	got := m.NeedEnrichment([]string{"did:plc:aaa", "did:plc:bbb"})
	if len(got) != 0 {
		t.Fatalf("expected 0 unknown, got %d", len(got))
	}
}

func TestNeedEnrichment_Empty(t *testing.T) {
	m := New()
	got := m.NeedEnrichment(nil)
	if got != nil {
		t.Fatalf("expected nil for empty input, got %v", got)
	}
}

func TestStore_AndRetrieve(t *testing.T) {
	m := New()
	state := &voresky.CaughtState{
		Phase: "none",
		BaseAvatar: &voresky.BaseAvatar{
			URL: "https://example.com/avatar.png",
		},
	}
	m.Store(map[string]*voresky.CaughtState{
		"did:plc:aaa": state,
	})

	got := m.GetCaughtState("did:plc:aaa")
	if got == nil || got.Phase != "none" {
		t.Fatal("Store/GetCaughtState round-trip failed")
	}
	if got.BaseAvatar == nil || got.BaseAvatar.URL != "https://example.com/avatar.png" {
		t.Fatal("BaseAvatar not preserved")
	}
}

func TestStore_TracksKnownHashes(t *testing.T) {
	m := New()
	m.Store(map[string]*voresky.CaughtState{
		"did:plc:aaa": {BlobHash: "abc123"},
		"did:plc:bbb": {Phase: "none"},
	})

	known := m.KnownStates()
	if len(known) != 1 || known["did:plc:aaa"] != "abc123" {
		t.Fatalf("expected {did:plc:aaa: abc123}, got %v", known)
	}
}

func TestKnownStates_ReturnsNilWhenEmpty(t *testing.T) {
	m := New()
	if m.KnownStates() != nil {
		t.Fatal("expected nil for empty knownHashes")
	}
}

func TestKnownStates_ReturnsCopy(t *testing.T) {
	m := New()
	m.Store(map[string]*voresky.CaughtState{
		"did:plc:aaa": {BlobHash: "abc123"},
	})
	cp := m.KnownStates()
	cp["did:plc:aaa"] = "modified"
	if m.knownHashes["did:plc:aaa"] != "abc123" {
		t.Fatal("KnownStates should return a copy, not the original")
	}
}

func TestStore_ClearsResolvedOnHashChange(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = &voresky.CaughtState{BlobHash: "old"}
	m.resolvedAvatars["did:plc:aaa"] = "https://old-snapshot.png"

	m.Store(map[string]*voresky.CaughtState{
		"did:plc:aaa": {BlobHash: "new"},
	})

	if _, ok := m.resolvedAvatars["did:plc:aaa"]; ok {
		t.Fatal("resolved avatar should be cleared on hash change")
	}
}

func TestStore_KeepsResolvedOnSameHash(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = &voresky.CaughtState{BlobHash: "same"}
	m.resolvedAvatars["did:plc:aaa"] = "https://snapshot.png"

	m.Store(map[string]*voresky.CaughtState{
		"did:plc:aaa": {BlobHash: "same"},
	})

	if m.resolvedAvatars["did:plc:aaa"] != "https://snapshot.png" {
		t.Fatal("resolved avatar should be kept when hash unchanged")
	}
}

func TestStoreSnapshot(t *testing.T) {
	m := New()
	blob := &voresky.SnapshotBlob{
		Nodes: []voresky.SnapshotNode{{ID: "n1", AvatarURL: "https://a.png"}},
	}
	m.StoreSnapshot("hash1", blob)

	if _, ok := m.snapshots["hash1"]; !ok {
		t.Fatal("snapshot not stored")
	}
}

func TestStoreSnapshot_IgnoresEmpty(t *testing.T) {
	m := New()
	m.StoreSnapshot("", nil)
	m.StoreSnapshot("hash", nil)
	m.StoreSnapshot("", &voresky.SnapshotBlob{})
	if len(m.snapshots) != 0 {
		t.Fatal("should not store empty/nil snapshots")
	}
}

func TestPendingSnapshots_ReturnsCaughtWithoutBlob(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = &voresky.CaughtState{
		BlobHash:      "hash1",
		CurrentNodeID: "node1",
	}
	m.cache["did:plc:bbb"] = &voresky.CaughtState{
		Phase:      "none",
		BaseAvatar: &voresky.BaseAvatar{URL: "https://a.png"},
	}

	pending := m.PendingSnapshots()
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
	if pending[0].DID != "did:plc:aaa" || pending[0].BlobHash != "hash1" {
		t.Fatalf("unexpected pending: %+v", pending[0])
	}
}

func TestPendingSnapshots_ResolvesFromCachedBlob(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = &voresky.CaughtState{
		BlobHash:      "hash1",
		CurrentNodeID: "node1",
	}
	m.StoreSnapshot("hash1", &voresky.SnapshotBlob{
		Nodes: []voresky.SnapshotNode{
			{ID: "node1", AvatarURL: "https://resolved.png"},
		},
	})

	pending := m.PendingSnapshots()
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending (should auto-resolve), got %d", len(pending))
	}
	if m.resolvedAvatars["did:plc:aaa"] != "https://resolved.png" {
		t.Fatal("should have resolved avatar from cached blob")
	}
}

func TestPendingSnapshots_SkipsAlreadyResolved(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = &voresky.CaughtState{BlobHash: "hash1", CurrentNodeID: "n1"}
	m.resolvedAvatars["did:plc:aaa"] = "https://already.png"

	pending := m.PendingSnapshots()
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending, got %d", len(pending))
	}
}

func TestResolveSnapshots(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = &voresky.CaughtState{
		BlobHash:      "hash1",
		CurrentNodeID: "node1",
	}
	m.StoreSnapshot("hash1", &voresky.SnapshotBlob{
		Nodes: []voresky.SnapshotNode{
			{ID: "node1", AvatarURL: "https://snap.png"},
		},
	})

	m.ResolveSnapshots()
	if m.resolvedAvatars["did:plc:aaa"] != "https://snap.png" {
		t.Fatal("ResolveSnapshots should populate resolvedAvatars")
	}
}

func TestBuildAvatarOverrides_BaseAvatar(t *testing.T) {
	m := New()
	m.Store(map[string]*voresky.CaughtState{
		"did:plc:aaa": {
			BaseAvatar: &voresky.BaseAvatar{URL: "https://char-avatar.png"},
		},
	})

	overrides := m.BuildAvatarOverrides("", "")
	if overrides["did:plc:aaa"] != "https://char-avatar.png" {
		t.Fatalf("expected character avatar, got %q", overrides["did:plc:aaa"])
	}
}

func TestBuildAvatarOverrides_CaughtWithResolvedAvatar(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = &voresky.CaughtState{BlobHash: "hash1"}
	m.resolvedAvatars["did:plc:aaa"] = "https://snapshot-avatar.png"

	overrides := m.BuildAvatarOverrides("", "")
	if overrides["did:plc:aaa"] != "https://snapshot-avatar.png" {
		t.Fatalf("expected snapshot avatar, got %q", overrides["did:plc:aaa"])
	}
}

func TestBuildAvatarOverrides_CaughtWithoutResolvedAvatar(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = &voresky.CaughtState{BlobHash: "hash1"}

	overrides := m.BuildAvatarOverrides("", "")
	if _, ok := overrides["did:plc:aaa"]; ok {
		t.Fatal("caught user without resolved snapshot should NOT be in overrides")
	}
}

func TestBuildAvatarOverrides_NoCaughtNoBase(t *testing.T) {
	m := New()
	m.Store(map[string]*voresky.CaughtState{
		"did:plc:aaa": {Phase: "none"},
	})

	overrides := m.BuildAvatarOverrides("", "")
	if _, ok := overrides["did:plc:aaa"]; ok {
		t.Fatal("DID with no caught state and no base avatar should not be in overrides")
	}
}

func TestBuildAvatarOverrides_OwnUserOverride(t *testing.T) {
	m := New()
	overrides := m.BuildAvatarOverrides("did:plc:me", "https://my-char.png")
	if overrides["did:plc:me"] != "https://my-char.png" {
		t.Fatal("own user override should be in map")
	}
}

func TestBuildAvatarOverrides_CaughtOverridesOwn(t *testing.T) {
	m := New()
	m.cache["did:plc:me"] = &voresky.CaughtState{BlobHash: "hash1"}
	m.resolvedAvatars["did:plc:me"] = "https://caught-me.png"

	overrides := m.BuildAvatarOverrides("did:plc:me", "https://my-char.png")
	if overrides["did:plc:me"] != "https://caught-me.png" {
		t.Fatalf("caught state should override own character avatar, got %q", overrides["did:plc:me"])
	}
}

func TestBuildAvatarOverrides_OwnEmptyIgnored(t *testing.T) {
	m := New()
	overrides := m.BuildAvatarOverrides("", "")
	if len(overrides) != 0 {
		t.Fatal("empty own DID should not produce override")
	}
}

func TestBuildAvatarOverrides_NilStateSkipped(t *testing.T) {
	m := New()
	m.cache["did:plc:aaa"] = nil

	overrides := m.BuildAvatarOverrides("", "")
	if _, ok := overrides["did:plc:aaa"]; ok {
		t.Fatal("nil CaughtState should be skipped")
	}
}
