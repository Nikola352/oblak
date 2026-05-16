package authkey

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// encrypt encrypts plaintext with AES-GCM using a key derived from kek via SHA-256.
// Output format is base64(nonce || ciphertext+tag).
func encrypt(kek, plaintext string) (string, error) {
	key := sha256.Sum256([]byte(kek))

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// decrypt decrypts an AES-GCM ciphertext that was encrypted with the given KEK.
// The stored format is base64(nonce || ciphertext+tag).
// The AES key is derived from the KEK using SHA-256.
func decrypt(kek, encoded string) (string, error) {
	key := sha256.Sum256([]byte(kek))

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}
