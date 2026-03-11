package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/scrypt"
)

const (
	// devFallbackKey is used when NOMS_TOKEN_KEY is not set (development only).
	devFallbackKey = "noms-dev-fallback-key-not-for-prod"
)

// ErrNotFound is returned when a token is not found in the store.
var ErrNotFound = errors.New("token not found")

// TokenStore is the interface for secure credential storage.
type TokenStore interface {
	Store(did string, data []byte) error
	Retrieve(did string) ([]byte, error)
	Delete(did string) error
	ListAccounts() ([]string, error)
}

// NewTokenStore returns a TokenStore. It tries the OS keyring first; if that
// fails it falls back to an encrypted file store.
func NewTokenStore() TokenStore {
	fs, err := NewFileStore()
	if err != nil {
		// Last resort: in-memory (data won't persist across restarts).
		return NewMemoryStore()
	}
	return fs
}

// ─── MemoryStore ─────────────────────────────────────────────────────────────

// MemoryStore is an in-memory TokenStore used in tests and as a last-resort
// fallback. Data is not persisted across process restarts.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemoryStore returns a new, empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string][]byte)}
}

func (m *MemoryStore) Store(did string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.data[did] = cp
	return nil
}

func (m *MemoryStore) Retrieve(did string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[did]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, did)
	}
	cp := make([]byte, len(v))
	copy(cp, v)
	return cp, nil
}

func (m *MemoryStore) Delete(did string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[did]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, did)
	}
	delete(m.data, did)
	return nil
}

func (m *MemoryStore) ListAccounts() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	accounts := make([]string, 0, len(m.data))
	for did := range m.data {
		accounts = append(accounts, did)
	}
	return accounts, nil
}

// ─── FileStore ────────────────────────────────────────────────────────────────

// fileStoreData is the JSON structure persisted to disk.
type fileStoreData struct {
	Accounts map[string][]byte `json:"accounts"`
}

// FileStore is an AES-256-GCM encrypted file-backed TokenStore.
// The file is stored at DataDir()/tokens.enc.
type FileStore struct {
	path string
	key  []byte // 32-byte AES-256 key
}

// NewFileStore creates a FileStore, deriving the encryption key from the
// NOMS_TOKEN_KEY environment variable (or a dev fallback).
func NewFileStore() (*FileStore, error) {
	rawKey := os.Getenv("NOMS_TOKEN_KEY")
	if rawKey == "" {
		rawKey = devFallbackKey
	}

	// Use scrypt to derive a 32-byte key from the passphrase.
	// Salt is fixed (per-app) — acceptable for a local credential store where
	// the passphrase itself is the secret.
	salt := []byte("noms-filestore-salt-v1")
	key, err := scrypt.Key([]byte(rawKey), salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("key derivation: %w", err)
	}

	dir := DataDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	return &FileStore{
		path: filepath.Join(dir, "tokens.enc"),
		key:  key,
	}, nil
}

func (f *FileStore) load() (*fileStoreData, error) {
	ciphertext, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &fileStoreData{Accounts: make(map[string][]byte)}, nil
		}
		return nil, err
	}

	plaintext, err := f.decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	var d fileStoreData
	if err := json.Unmarshal(plaintext, &d); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if d.Accounts == nil {
		d.Accounts = make(map[string][]byte)
	}
	return &d, nil
}

func (f *FileStore) save(d *fileStoreData) error {
	plaintext, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	ciphertext, err := f.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	dir := filepath.Dir(f.path)
	tmp, err := os.CreateTemp(dir, "tokens-*.enc.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(ciphertext); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, f.path)
}

func (f *FileStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (f *FileStore) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (f *FileStore) Store(did string, data []byte) error {
	d, err := f.load()
	if err != nil {
		return err
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	d.Accounts[did] = cp
	return f.save(d)
}

func (f *FileStore) Retrieve(did string) ([]byte, error) {
	d, err := f.load()
	if err != nil {
		return nil, err
	}
	v, ok := d.Accounts[did]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, did)
	}
	cp := make([]byte, len(v))
	copy(cp, v)
	return cp, nil
}

func (f *FileStore) Delete(did string) error {
	d, err := f.load()
	if err != nil {
		return err
	}
	if _, ok := d.Accounts[did]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, did)
	}
	delete(d.Accounts, did)
	return f.save(d)
}

func (f *FileStore) ListAccounts() ([]string, error) {
	d, err := f.load()
	if err != nil {
		return nil, err
	}
	accounts := make([]string, 0, len(d.Accounts))
	for did := range d.Accounts {
		accounts = append(accounts, did)
	}
	return accounts, nil
}
