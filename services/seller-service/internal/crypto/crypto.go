package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const prefix = "aes256gcm:"

type Encryptor struct {
	key []byte
}

func NewEncryptor(keyStr string) (*Encryptor, error) {
	// Try decoding as hex first
	key, err := hex.DecodeString(keyStr)
	if err != nil || len(key) != 32 {
		// Try base64
		if b, errBase64 := base64.StdEncoding.DecodeString(keyStr); errBase64 == nil && len(b) == 32 {
			key = b
		} else {
			// Fallback to raw bytes
			key = []byte(keyStr)
		}
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes, got %d bytes", len(key))
	}

	return &Encryptor{key: key}, nil
}

// Encrypt encrypts plaintext and returns base64 string formatted as "aes256gcm:<iv>:<ciphertext>"
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	iv := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("failed to generate IV: %w", err)
	}

	ciphertext := aesgcm.Seal(nil, iv, []byte(plaintext), nil)

	encodedIV := base64.StdEncoding.EncodeToString(iv)
	encodedCipher := base64.StdEncoding.EncodeToString(ciphertext)

	return fmt.Sprintf("%s%s:%s", prefix, encodedIV, encodedCipher), nil
}

// Decrypt decrypts ciphertext. For backward compatibility, if the string does
// not start with "aes256gcm:", it returns the raw input value as plaintext.
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	if !strings.HasPrefix(ciphertext, prefix) {
		return ciphertext, nil
	}

	parts := strings.Split(ciphertext[len(prefix):], ":")
	if len(parts) != 2 {
		return "", errors.New("invalid ciphertext format")
	}

	iv, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("failed to decode IV: %w", err)
	}

	cipherVal, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := aesgcm.Open(nil, iv, cipherVal, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	return string(plaintext), nil
}

// HashPIN hashes a 6-digit PIN using bcrypt
func HashPIN(pin string) (string, error) {
	if len(pin) != 6 {
		return "", errors.New("PIN must be exactly 6 digits")
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			return "", errors.New("PIN must contain only digits")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash PIN: %w", err)
	}

	return string(hash), nil
}

// ComparePIN compares a bcrypt hash with a PIN input
func ComparePIN(hash, pin string) error {
	if len(pin) != 6 {
		return errors.New("PIN must be exactly 6 digits")
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin))
}
