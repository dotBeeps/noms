package config_test

import (
	"bytes"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/dotBeeps/noms/internal/config"
)

// --- MemoryStore tests (default NewTokenStore in test env) ---

func TestKeyringStore(t *testing.T) {
	t.Parallel()
	store := config.NewMemoryStore()
	err := store.Store("did:plc:abc123", []byte("token-data"))
	if err != nil {
		t.Fatalf("Store() error: %v", err)
	}
}

func TestKeyringRetrieve(t *testing.T) {
	t.Parallel()
	store := config.NewMemoryStore()
	want := []byte("my-secret-token")
	if err := store.Store("did:plc:retrieve", want); err != nil {
		t.Fatalf("Store() error: %v", err)
	}
	got, err := store.Retrieve("did:plc:retrieve")
	if err != nil {
		t.Fatalf("Retrieve() error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("Retrieve() = %q, want %q", got, want)
	}
}

func TestKeyringDelete(t *testing.T) {
	t.Parallel()
	store := config.NewMemoryStore()
	if err := store.Store("did:plc:delete", []byte("to-delete")); err != nil {
		t.Fatalf("Store() error: %v", err)
	}
	if err := store.Delete("did:plc:delete"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	_, err := store.Retrieve("did:plc:delete")
	if err == nil {
		t.Error("Retrieve() after Delete() should return error, got nil")
	}
}

func TestKeyringListAccounts(t *testing.T) {
	t.Parallel()
	store := config.NewMemoryStore()
	dids := []string{"did:plc:user1", "did:plc:user2"}
	for _, did := range dids {
		if err := store.Store(did, []byte("token")); err != nil {
			t.Fatalf("Store(%q) error: %v", did, err)
		}
	}
	accounts, err := store.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts() error: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("ListAccounts() returned %d accounts, want 2", len(accounts))
	}
	sort.Strings(accounts)
	sort.Strings(dids)
	for i, got := range accounts {
		if got != dids[i] {
			t.Errorf("accounts[%d] = %q, want %q", i, got, dids[i])
		}
	}
}

func TestNewFileStore_ErrorsWithoutKeyAndNoHomeDir(t *testing.T) {
	t.Setenv("NOMS_TOKEN_KEY", "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_DATA_HOME", t.TempDir()) // avoid data dir error masking the home dir error

	_, err := config.NewFileStore()
	if err == nil {
		t.Fatal("NewFileStore() expected error when HOME and NOMS_TOKEN_KEY are both unset, got nil")
	}
}

// --- FileStore tests ---

func TestFileFallbackStore(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("NOMS_TOKEN_KEY", "test-encryption-key-32-bytes-ok!")

	store, err := config.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}
	if err := store.Store("did:plc:filetest", []byte("file-token")); err != nil {
		t.Fatalf("Store() error: %v", err)
	}
}

func TestFileFallbackRetrieve(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("NOMS_TOKEN_KEY", "test-encryption-key-32-bytes-ok!")

	store, err := config.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}

	want := []byte("file-secret-token")
	if err := store.Store("did:plc:fileretrieve", want); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	got, err := store.Retrieve("did:plc:fileretrieve")
	if err != nil {
		t.Fatalf("Retrieve() error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("Retrieve() = %q, want %q", got, want)
	}
}

func TestFileStoreConcurrentWrites(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("NOMS_TOKEN_KEY", "test-encryption-key-32-bytes-ok!")

	store, err := config.NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}

	const writers = 8
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			did := "did:plc:concurrent-" + string(rune('a'+i))
			if err := store.Store(did, []byte("token")); err != nil {
				t.Errorf("Store(%q) error: %v", did, err)
			}
		}(i)
	}
	wg.Wait()

	accounts, err := store.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts() error: %v", err)
	}
	if len(accounts) != writers {
		t.Fatalf("ListAccounts() = %d, want %d", len(accounts), writers)
	}
}
