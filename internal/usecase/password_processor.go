package usecase

import (
	"context"
	"errors"

	"pass_gen/internal/security/password"
)

type PasswordStore interface {
	SavePasswordHash(ctx context.Context, hash string) error
	SaveGenerationAudit(ctx context.Context, length int, count int) error
}

type PasswordProcessor struct {
	store PasswordStore
}

type RegisterResult struct {
	PasswordHash        string `json:"password_hash,omitempty"`
	TransportCiphertext string `json:"transport_ciphertext"`
}

func NewPasswordProcessor(store PasswordStore) *PasswordProcessor {
	return &PasswordProcessor{store: store}
}

func (p *PasswordProcessor) RegisterPassword(ctx context.Context, plainPassword string, transportKey []byte) (RegisterResult, error) {
	if plainPassword == "" {
		return RegisterResult{}, errors.New("plain password is empty")
	}

	hash, err := password.HashArgon2id(plainPassword)
	if err != nil {
		return RegisterResult{}, err
	}
	if p.store != nil {
		if err := p.store.SavePasswordHash(ctx, hash); err != nil {
			return RegisterResult{}, err
		}
	}

	ciphertext, err := password.EncryptForTransportBase64(plainPassword, transportKey)
	if err != nil {
		return RegisterResult{}, err
	}

	return RegisterResult{
		PasswordHash:        hash,
		TransportCiphertext: ciphertext,
	}, nil
}

func (p *PasswordProcessor) GenerateAndRegister(ctx context.Context, length int, count int, transportKey []byte) ([]RegisterResult, error) {
	generated, err := password.Generate(length, count)
	if err != nil {
		return nil, err
	}

	if p.store != nil {
		if err := p.store.SaveGenerationAudit(ctx, length, count); err != nil {
			return nil, err
		}
	}

	results := make([]RegisterResult, 0, len(generated))
	for _, plain := range generated {
		item, err := p.RegisterPassword(ctx, plain, transportKey)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	return results, nil
}

func (p *PasswordProcessor) VerifyPassword(_ context.Context, plainPassword, encodedHash string) (bool, error) {
	if plainPassword == "" || encodedHash == "" {
		return false, errors.New("password and hash are required")
	}

	return password.VerifyArgon2id(plainPassword, encodedHash)
}

func (p *PasswordProcessor) ValidatePassword(_ context.Context, plainPassword string) (password.ValidationResult, error) {
	if plainPassword == "" {
		return password.ValidationResult{}, errors.New("password is required")
	}
	return password.Validate(plainPassword), nil
}

func (p *PasswordProcessor) PasswordStrength(_ context.Context, plainPassword string) (password.StrengthResult, error) {
	if plainPassword == "" {
		return password.StrengthResult{}, errors.New("password is required")
	}
	return password.Strength(plainPassword), nil
}
