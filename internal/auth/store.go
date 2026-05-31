package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"github.com/bishop-bot/ibcli/internal/models"
)

// ErrNoSession is returned when no saved session exists.
var ErrNoSession = errors.New("no saved session found")

// ErrSessionExpired is returned when the saved session has expired.
var ErrSessionExpired = errors.New("session has expired")

// ErrDecryptionFailed is returned when decryption fails.
var ErrDecryptionFailed = errors.New("failed to decrypt session")

// Store manages encrypted session persistence.
type Store struct {
	dir    string
	master []byte
}

// NewStore creates a new session store with the given master password.
// The master password is derived using PBKDF2 with SHA-256.
func NewStore(masterPassword string) (*Store, error) {
	dir, err := getDefaultSessionDir()
	if err != nil {
		return nil, fmt.Errorf("getting session dir: %w", err)
	}

	// Derive a 32-byte key from the master password using PBKDF2
	key := pbkdf2.Key([]byte(masterPassword), []byte(salt), 100000, 32, sha256.New)

	return &Store{
		dir:    dir,
		master: key,
	}, nil
}

// Save encrypts and saves the session to disk.
func (s *Store) Save(session *models.SavedSession) error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("creating session dir: %w", err)
	}

	// Marshal to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	// Encrypt
	encrypted, err := s.encrypt(data)
	if err != nil {
		return fmt.Errorf("encrypting session: %w", err)
	}

	// Write to file
	filename := filepath.Join(s.dir, "session.enc")
	if err := os.WriteFile(filename, encrypted, 0600); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}

	return nil
}

// Load decrypts and returns the saved session.
// Returns ErrNoSession if no session exists.
// Returns ErrSessionExpired if the session has expired.
func (s *Store) Load() (*models.SavedSession, error) {
	filename := filepath.Join(s.dir, "session.enc")

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSession
		}
		return nil, fmt.Errorf("reading session file: %w", err)
	}

	// Decrypt
	decrypted, err := s.decrypt(data)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	// Unmarshal
	var session models.SavedSession
	if err := json.Unmarshal(decrypted, &session); err != nil {
		return nil, fmt.Errorf("unmarshaling session: %w", err)
	}

	// Check expiration
	if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return &session, nil
}

// Delete removes the saved session.
func (s *Store) Delete() error {
	filename := filepath.Join(s.dir, "session.enc")
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

// Exists checks if a saved session exists.
func (s *Store) Exists() bool {
	filename := filepath.Join(s.dir, "session.enc")
	_, err := os.Stat(filename)
	return err == nil
}

// encrypt encrypts data using AES-256-GCM.
func (s *Store) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.master)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Seal appends the encrypted data to nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-256-GCM.
func (s *Store) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.master)
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

func getDefaultSessionDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ib-cli", "sessions"), nil
}

// EncodeToBase64 encodes bytes to base64.
func EncodeToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeFromBase64 decodes base64 to bytes.
func DecodeFromBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// salt is used for key derivation. In production, this could be stored
// in the encrypted file or derived from the master password.
var salt = []byte("ib-cli-session-v1-salt")