package storage

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

var (
	// SessionKey is the derived key used for encryption/decryption during this session.
	// It is kept in memory and never stored on disk.
	SessionKey []byte
)

const (
	KeySize  = 32
	SaltSize = 16
)

// GenerateSalt creates a random 16-byte salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// DeriveKey derives a 32-byte key from the password and salt using Argon2id.
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, KeySize)
}

// Encrypt encrypts plaintext using XChaCha20-Poly1305.
// Returns the nonce appended with the ciphertext, Base64 encoded.
func Encrypt(plaintext string) (string, error) {
	if len(SessionKey) != KeySize {
		return "", errors.New("encryption key not initialized")
	}

	aead, err := chacha20poly1305.NewX(SessionKey)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a Base64 encoded ciphertext using XChaCha20-Poly1305.
func Decrypt(encodedCiphertext string) (string, error) {
	if len(SessionKey) != KeySize {
		return "", errors.New("encryption key not initialized")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return "", err
	}

	aead, err := chacha20poly1305.NewX(SessionKey)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aead.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce, encryptedMsg := ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():]

	plaintext, err := aead.Open(nil, nonce, encryptedMsg, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
