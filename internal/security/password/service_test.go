package password

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestGenerateValidationErrors(t *testing.T) {
	if _, err := Generate(MinPasswordLength-1, 1); err != ErrPasswordLengthTooLow {
		t.Fatalf("expected ErrPasswordLengthTooLow, got %v", err)
	}

	if _, err := Generate(MinPasswordLength, MinPasswordsCount-1); err != ErrTooLowPasswordsCount {
		t.Fatalf("expected ErrTooLowPasswordsCount, got %v", err)
	}

	if _, err := Generate(MinPasswordLength, MaxPasswordsCount+1); err != ErrTooBigPasswordsCount {
		t.Fatalf("expected ErrTooBigPasswordsCount, got %v", err)
	}
}

func TestGenerateReturnsExpectedCountAndStrongComposition(t *testing.T) {
	const (
		length = 12
		count  = 10
	)

	passwords, err := Generate(length, count)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(passwords) != count {
		t.Fatalf("expected %d passwords, got %d", count, len(passwords))
	}

	seen := make(map[string]struct{}, count)
	for _, p := range passwords {
		if len(p) != length {
			t.Fatalf("expected password length %d, got %d", length, len(p))
		}
		if _, ok := seen[p]; ok {
			t.Fatalf("password is duplicated in output: %q", p)
		}
		seen[p] = struct{}{}

		if !containsAny(p, upperChars) {
			t.Fatalf("password has no uppercase character: %q", p)
		}
		if !containsAny(p, lowerChars) {
			t.Fatalf("password has no lowercase character: %q", p)
		}
		if !containsAny(p, digitChars) {
			t.Fatalf("password has no digit character: %q", p)
		}
		if !containsAny(p, specialChars) {
			t.Fatalf("password has no special character: %q", p)
		}
	}
}

func TestHashAndVerifyArgon2id(t *testing.T) {
	const plain = "S7rong!Passw0rd"

	encoded, err := HashArgon2id(plain)
	if err != nil {
		t.Fatalf("HashArgon2id returned error: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$") {
		t.Fatalf("unexpected hash prefix: %q", encoded)
	}

	ok, err := VerifyArgon2id(plain, encoded)
	if err != nil {
		t.Fatalf("VerifyArgon2id returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected hash verification to pass")
	}

	ok, err = VerifyArgon2id("wrong-password", encoded)
	if err != nil {
		t.Fatalf("VerifyArgon2id with wrong password returned error: %v", err)
	}
	if ok {
		t.Fatal("expected hash verification to fail for wrong password")
	}
}

func TestVerifyArgon2idInvalidHashFormat(t *testing.T) {
	if _, err := VerifyArgon2id("pass", "bad-format"); err == nil {
		t.Fatal("expected error for invalid hash format")
	}
}

func TestEncryptForTransportBase64RoundTrip(t *testing.T) {
	const plain = "Transport!Pass123"
	key, err := NewTransportKey()
	if err != nil {
		t.Fatalf("NewTransportKey returned error: %v", err)
	}

	cipherB64, err := EncryptForTransportBase64(plain, key)
	if err != nil {
		t.Fatalf("EncryptForTransportBase64 returned error: %v", err)
	}
	if cipherB64 == plain {
		t.Fatal("ciphertext must not match plaintext")
	}

	decrypted, err := decryptTransportBase64(cipherB64, key)
	if err != nil {
		t.Fatalf("decryptTransportBase64 returned error: %v", err)
	}
	if decrypted != plain {
		t.Fatalf("expected decrypted password %q, got %q", plain, decrypted)
	}
}

func TestEncryptForTransportBase64InvalidKey(t *testing.T) {
	if _, err := EncryptForTransportBase64("pass", []byte("short-key")); err == nil {
		t.Fatal("expected error for invalid AES key length")
	}
}

func TestNewTransportKeyLength(t *testing.T) {
	key, err := NewTransportKey()
	if err != nil {
		t.Fatalf("NewTransportKey returned error: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func containsAny(password string, charset []rune) bool {
	for _, c := range charset {
		if strings.ContainsRune(password, c) {
			return true
		}
	}
	return false
}

func decryptTransportBase64(cipherB64 string, key []byte) (string, error) {
	packed, err := base64.RawStdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aead.NonceSize()
	if len(packed) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce := packed[:nonceSize]
	ciphertext := packed[nonceSize:]

	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}
