package usecase

import (
	"context"
	"testing"

	"pass_gen/internal/security/password"
)

func TestRegisterPasswordSuccess(t *testing.T) {
	processor := NewPasswordProcessor(nil)
	key, err := password.NewTransportKey()
	if err != nil {
		t.Fatalf("NewTransportKey returned error: %v", err)
	}

	result, err := processor.RegisterPassword(context.Background(), "MyS3cure!Pass", key)
	if err != nil {
		t.Fatalf("RegisterPassword returned error: %v", err)
	}
	if result.PasswordHash == "" {
		t.Fatal("expected non-empty password hash")
	}
	if result.TransportCiphertext == "" {
		t.Fatal("expected non-empty transport ciphertext")
	}

	verified, err := processor.VerifyPassword(context.Background(), "MyS3cure!Pass", result.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if !verified {
		t.Fatal("expected verification to succeed")
	}
}

func TestRegisterPasswordValidation(t *testing.T) {
	processor := NewPasswordProcessor(nil)
	key, err := password.NewTransportKey()
	if err != nil {
		t.Fatalf("NewTransportKey returned error: %v", err)
	}

	if _, err := processor.RegisterPassword(context.Background(), "", key); err == nil {
		t.Fatal("expected error for empty password")
	}

	if _, err := processor.RegisterPassword(context.Background(), "GoodPass!9", []byte("short")); err == nil {
		t.Fatal("expected error for invalid transport key")
	}
}

func TestVerifyPasswordValidationAndMismatch(t *testing.T) {
	processor := NewPasswordProcessor(nil)

	if _, err := processor.VerifyPassword(context.Background(), "", "hash"); err == nil {
		t.Fatal("expected validation error for empty password")
	}

	if _, err := processor.VerifyPassword(context.Background(), "pass", ""); err == nil {
		t.Fatal("expected validation error for empty hash")
	}

	key, err := password.NewTransportKey()
	if err != nil {
		t.Fatalf("NewTransportKey returned error: %v", err)
	}

	result, err := processor.RegisterPassword(context.Background(), "Real!Pass123", key)
	if err != nil {
		t.Fatalf("RegisterPassword returned error: %v", err)
	}

	verified, err := processor.VerifyPassword(context.Background(), "Wrong!Pass123", result.PasswordHash)
	if err != nil {
		t.Fatalf("VerifyPassword mismatch case returned error: %v", err)
	}
	if verified {
		t.Fatal("expected verification to fail for mismatched password")
	}
}

func TestValidateAndStrength(t *testing.T) {
	processor := NewPasswordProcessor(nil)

	validation, err := processor.ValidatePassword(context.Background(), "Abc!1234")
	if err != nil {
		t.Fatalf("ValidatePassword returned error: %v", err)
	}
	if !validation.Valid {
		t.Fatal("expected validation to pass")
	}

	strength, err := processor.PasswordStrength(context.Background(), "Abc!1234")
	if err != nil {
		t.Fatalf("PasswordStrength returned error: %v", err)
	}
	if strength.Score <= 0 {
		t.Fatalf("expected positive strength score, got %d", strength.Score)
	}
}
