package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// DefaultMasterKeyPath is the default location for the master encryption key.
const DefaultMasterKeyPath = "/etc/vc-stack/master.key"

// GenerateMasterKey creates a new 32-byte (AES-256) master key and returns it as a base64 string.
func GenerateMasterKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("generate master key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// GetMasterKey reads the master key from the environment variable VC_MASTER_KEY
// or the default file path.
func GetMasterKey(keyPath string) ([]byte, error) {
	if envKey := os.Getenv("VC_MASTER_KEY"); envKey != "" {
		return base64.StdEncoding.DecodeString(strings.TrimSpace(envKey))
	}

	p := keyPath
	if p == "" {
		p = DefaultMasterKeyPath
	}

	keyFile, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read master key file %s: %w", p, err)
	}

	return base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyFile)))
}

// Encrypt encrypts a plain string using the provided 32-byte key (AES-256-GCM).
// The returned string is formatted as ENC(base64(nonce+ciphertext)).
func Encrypt(plain string, key []byte) (string, error) {
	if len(key) != 32 && len(key) != 16 && len(key) != 24 {
		return "", errors.New("invalid key length: must be 16, 24, or 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	encStr := base64.StdEncoding.EncodeToString(ciphertext)

	return fmt.Sprintf("ENC(%s)", encStr), nil
}

// Decrypt strips the "ENC(...)" wrapper (if present) and decrypts the string using the provided key.
// If the string does not have the "ENC(" prefix, it returns the string as is (plain text fallback).
func Decrypt(encrypted string, key []byte) (string, error) {
	if !strings.HasPrefix(encrypted, "ENC(") || !strings.HasSuffix(encrypted, ")") {
		// Return plain text as fallback (useful for backwards compatibility or dev environments)
		return encrypted, nil
	}

	encStr := encrypted[4 : len(encrypted)-1]
	data, err := base64.StdEncoding.DecodeString(encStr)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	if len(key) != 32 && len(key) != 16 && len(key) != 24 {
		return "", errors.New("invalid key length: must be 16, 24, or 32 bytes")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(data) < gcm.NonceSize() {
		return "", errors.New("malformed ciphertext")
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plain), nil
}
