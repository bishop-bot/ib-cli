package auth

import (
	"path/filepath"
	"testing"

	"github.com/bishop-bot/ib-cli/internal/models"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
)

func TestStore_NewStore(t *testing.T) {
	store, err := NewStore("test-master-password")
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	if store == nil {
		t.Fatal("NewStore() returned nil")
	}

	if store.dir == "" {
		t.Error("store.dir is empty")
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	// Create temp store
	tmpDir := t.TempDir()
	store := &Store{
		dir:    tmpDir,
		master: deriveTestKey("test-password"),
	}

	session := &models.SavedSession{
		SessionID: "test-session-123",
		Token:     "test-token-456",
		Username:  "testuser",
	}

	// Save
	if err := store.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.SessionID != session.SessionID {
		t.Errorf("SessionID = %v, want %v", loaded.SessionID, session.SessionID)
	}
	if loaded.Token != session.Token {
		t.Errorf("Token = %v, want %v", loaded.Token, session.Token)
	}
	if loaded.Username != session.Username {
		t.Errorf("Username = %v, want %v", loaded.Username, session.Username)
	}
}

func TestStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := &Store{
		dir:    tmpDir,
		master: deriveTestKey("test-password"),
	}

	// Create a session file
	session := &models.SavedSession{
		SessionID: "test-session",
		Username:  "testuser",
	}
	if err := store.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify it exists
	if !store.Exists() {
		t.Error("Exists() = false, want true after Save()")
	}

	// Delete
	if err := store.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	if store.Exists() {
		t.Error("Exists() = true, want false after Delete()")
	}
}

func TestStore_Exists(t *testing.T) {
	tmpDir := t.TempDir()

	// Fresh store
	store := &Store{
		dir:    tmpDir,
		master: deriveTestKey("test"),
	}

	if store.Exists() {
		t.Error("Exists() = true, want false for new store")
	}

	// After save
	session := &models.SavedSession{SessionID: "test", Username: "user"}
	if err := store.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if !store.Exists() {
		t.Error("Exists() = false, want true after Save()")
	}
}

func TestStore_DecryptFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Store with one key
	store1 := &Store{
		dir:    tmpDir,
		master: deriveTestKey("password1"),
	}

	session := &models.SavedSession{SessionID: "test", Username: "user"}
	if err := store1.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Store with different key
	store2 := &Store{
		dir:    tmpDir,
		master: deriveTestKey("password2"),
	}

	// Load should fail
	_, err := store2.Load()
	if err != ErrDecryptionFailed {
		t.Errorf("Load() error = %v, want %v", err, ErrDecryptionFailed)
	}
}

func TestStore_NoSession(t *testing.T) {
	tmpDir := t.TempDir()
	store := &Store{
		dir:    tmpDir,
		master: deriveTestKey("test"),
	}

	_, err := store.Load()
	if err != ErrNoSession {
		t.Errorf("Load() error = %v, want %v", err, ErrNoSession)
	}
}

func TestEncodeDecode(t *testing.T) {
	original := []byte("hello world")

	encoded := EncodeToBase64(original)
	decoded, err := DecodeFromBase64(encoded)
	if err != nil {
		t.Fatalf("DecodeFromBase64() error = %v", err)
	}

	if string(decoded) != string(original) {
		t.Errorf("decoded = %v, want %v", string(decoded), string(original))
	}
}

// deriveTestKey creates a test key for unit tests
func deriveTestKey(password string) []byte {
	return pbkdf2.Key([]byte(password), []byte("test-salt"), 1000, 32, sha3.New256)
}

func TestGetDefaultSessionDir(t *testing.T) {
	dir, err := getDefaultSessionDir()
	if err != nil {
		t.Fatalf("getDefaultSessionDir() error = %v", err)
	}

	if dir == "" {
		t.Error("getDefaultSessionDir() returned empty string")
	}

	// Should end with expected path
	expectedSuffix := filepath.Join(".ib-cli", "sessions")
	if filepath.Base(dir) != "sessions" {
		t.Errorf("getDefaultSessionDir() = %v, doesn't end with sessions", dir)
	}
	_ = expectedSuffix // Just checking the structure
}

func TestEncryptDecrypt(t *testing.T) {
	store := &Store{
		dir:    t.TempDir(),
		master: deriveTestKey("test-password"),
	}

	plaintext := []byte("sensitive session data with tokens and secrets")

	encrypted, err := store.encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt() error = %v", err)
	}

	// Encrypted should be different from plaintext
	if string(encrypted) == string(plaintext) {
		t.Error("encrypted data is identical to plaintext")
	}

	// Decrypt
	decrypted, err := store.decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt() error = %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted = %v, want %v", string(decrypted), string(plaintext))
	}
}

func TestEncryptDecrypt_DifferentKeys(t *testing.T) {
	store1 := &Store{master: deriveTestKey("key1")}
	store2 := &Store{master: deriveTestKey("key2")}

	plaintext := []byte("test data")

	encrypted, err := store1.encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt() error = %v", err)
	}

	// Decrypting with wrong key should fail
	_, err = store2.decrypt(encrypted)
	if err == nil {
		t.Error("decrypt() with wrong key should fail")
	}
}

func TestEncrypt_EmptyData(t *testing.T) {
	store := &Store{master: deriveTestKey("test")}
	_, err := store.encrypt([]byte{})
	if err != nil {
		t.Errorf("encrypt() with empty data should not fail, got %v", err)
	}
}