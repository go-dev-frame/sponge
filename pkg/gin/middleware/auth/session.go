package auth

// 1. Universal session management is implemented using the library available at https://github.com/gin-contrib/sessions,
// which provides Gin middleware for session management with support:
//
// cookie-based
// Redis
// memcached
// MongoDB
// GORM
// memstore
// PostgreSQL
// Filesystem

// -------------------------------------------------------------------------------------------

// 2. Special session for rails

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	json "github.com/bytedance/sonic"
	"golang.org/x/crypto/pbkdf2"
)

// DecodeSignedCookie decrypts a Rails 7.1+ encrypted cookie using the provided
// secretKeyBase and validates that its purpose matches the given cookieName.
// It returns the decoded session payload (the JSON contained in _rails.message).
//
// The Rails encrypted cookie format is: base64(data)--base64(iv)--base64(authTag)
// Key derivation: PBKDF2-HMAC-SHA256(secret_key_base, "authenticated encrypted cookie", 1000, 32)
// Cipher: AES-256-GCM, AAD: empty
func DecodeSignedCookie(secretKeyBase string, decodedCookie string, cookieName string) (map[string]any, error) {
	if secretKeyBase == "" {
		return nil, errors.New("missing secretKeyBase")
	}
	if decodedCookie == "" {
		return nil, errors.New("missing cookie value")
	}

	parts := strings.Split(decodedCookie, "--")
	if len(parts) != 3 {
		return nil, errors.New("invalid cookie format")
	}

	data, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode data: %w", err)
	}
	iv, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode iv: %w", err)
	}
	authTag, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode auth tag: %w", err)
	}
	if len(authTag) != 16 { // GCM tag size (bytes)
		return nil, errors.New("invalid auth tag size")
	}

	// Derive key
	const (
		salt       = "authenticated encrypted cookie"
		iterations = 1000
		keyLength  = 32 // AES-256
	)
	key := pbkdf2.Key([]byte(secretKeyBase), []byte(salt), iterations, keyLength, sha256.New)

	// AES-GCM decrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %w", err)
	}
	if len(iv) != gcm.NonceSize() {
		return nil, errors.New("invalid iv size")
	}

	// In Go, GCM expects ciphertext || tag
	ciphertext := make([]byte, 0, len(data)+len(authTag))
	ciphertext = append(ciphertext, data...)
	ciphertext = append(ciphertext, authTag...)

	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed")
	}

	// Parse envelope
	var envelope struct {
		Rails struct {
			Pur     string `json:"pur"`
			Message string `json:"message"`
		} `json:"_rails"`
	}
	if unmarshalEnvelopeErr := json.Unmarshal(plaintext, &envelope); unmarshalEnvelopeErr != nil {
		return nil, fmt.Errorf("failed to unmarshal envelope: %w", unmarshalEnvelopeErr)
	}
	if envelope.Rails.Pur == "" || envelope.Rails.Message == "" {
		return nil, errors.New("invalid envelope data")
	}
	if envelope.Rails.Pur != fmt.Sprintf("cookie.%s", cookieName) {
		return nil, errors.New("invalid cookie purpose")
	}

	// Decode inner message (base64 JSON)
	msgBytes, err := base64.StdEncoding.DecodeString(envelope.Rails.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode message: %w", err)
	}
	var session map[string]any
	if unmarshalSessionErr := json.Unmarshal(msgBytes, &session); unmarshalSessionErr != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", unmarshalSessionErr)
	}
	return session, nil
}

// UserIDFromSession tries to extract the warden user id from a Rails session.
// It returns the id and true if found, otherwise (nil, false).
func UserIDFromSession(session map[string]any) (any, bool) {
	val, ok := session["warden.user.user.key"]
	if !ok {
		return nil, false
	}
	// Expecting [[id], ...]
	outer, ok := val.([]any)
	if !ok || len(outer) == 0 {
		return nil, false
	}
	inner, ok := outer[0].([]any)
	if !ok || len(inner) == 0 {
		return nil, false
	}
	return inner[0], true
}
