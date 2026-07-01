package crypto

import (
	"testing"
)

func TestCrypto(t *testing.T) {
	key := "12345678901234567890123456789012" // 32 bytes
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := "my-secret-bank-details"
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("expected %q, got %q", plaintext, decrypted)
	}

	// Test fallback/raw value (compatibility)
	raw := "Steward Bank"
	decryptedRaw, err := enc.Decrypt(raw)
	if err != nil {
		t.Fatalf("failed to decrypt raw: %v", err)
	}
	if decryptedRaw != raw {
		t.Errorf("expected %q, got %q", raw, decryptedRaw)
	}

	// Test PIN Hashing
	pin := "123456"
	hash, err := HashPIN(pin)
	if err != nil {
		t.Fatalf("failed to hash PIN: %v", err)
	}

	err = ComparePIN(hash, pin)
	if err != nil {
		t.Errorf("failed to compare correct PIN: %v", err)
	}

	err = ComparePIN(hash, "654321")
	if err == nil {
		t.Error("expected error for incorrect PIN, got nil")
	}

	// Test invalid PIN
	_, err = HashPIN("12345")
	if err == nil {
		t.Error("expected error for short PIN, got nil")
	}
	_, err = HashPIN("12345a")
	if err == nil {
		t.Error("expected error for alphanumeric PIN, got nil")
	}
}
