package password

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

const (
	MinPasswordLength = 4
	MinPasswordsCount = 1
	MaxPasswordsCount = 50

	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024
	argonThreads uint8  = 2
	argonKeyLen  uint32 = 32
	saltSize            = 16
)

var (
	ErrPasswordLengthTooLow = errors.New("password length too low")
	ErrTooLowPasswordsCount = errors.New("too low passwords count")
	ErrTooBigPasswordsCount = errors.New("too big passwords count")
)

var (
	upperChars   = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	lowerChars   = []rune("abcdefghijklmnopqrstuvwxyz")
	digitChars   = []rune("0123456789")
	specialChars = []rune("!@#$%^&*")
)

type ValidationResult struct {
	Length         int  `json:"length"`
	HasUpper       bool `json:"has_upper"`
	HasLower       bool `json:"has_lower"`
	HasDigit       bool `json:"has_digit"`
	HasSpecial     bool `json:"has_special"`
	MinLengthMet   bool `json:"min_length_met"`
	ClassesMatched int  `json:"classes_matched"`
	Valid          bool `json:"valid"`
}

type StrengthResult struct {
	Score      int              `json:"score"`
	Label      string           `json:"label"`
	Validation ValidationResult `json:"validation"`
}

func Generate(length int, count int) ([]string, error) {
	if err := validateGenerateArgs(length, count); err != nil {
		return nil, err
	}

	passwords := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	marker := struct{}{}

	for len(passwords) < count {
		newPass, err := generateOne(length)
		if err != nil {
			return nil, err
		}

		if _, found := seen[newPass]; found {
			continue
		}

		passwords = append(passwords, newPass)
		seen[newPass] = marker
	}

	return passwords, nil
}

func Validate(plainPassword string) ValidationResult {
	result := ValidationResult{Length: len([]rune(plainPassword))}
	for _, r := range plainPassword {
		switch {
		case unicode.IsUpper(r):
			result.HasUpper = true
		case unicode.IsLower(r):
			result.HasLower = true
		case unicode.IsDigit(r):
			result.HasDigit = true
		default:
			result.HasSpecial = true
		}
	}

	if result.HasUpper {
		result.ClassesMatched++
	}
	if result.HasLower {
		result.ClassesMatched++
	}
	if result.HasDigit {
		result.ClassesMatched++
	}
	if result.HasSpecial {
		result.ClassesMatched++
	}

	result.MinLengthMet = result.Length >= MinPasswordLength
	result.Valid = result.MinLengthMet && result.ClassesMatched >= 3
	return result
}

func Strength(plainPassword string) StrengthResult {
	validation := Validate(plainPassword)
	score := 0

	if validation.MinLengthMet {
		score++
	}
	if validation.Length >= 8 {
		score++
	}
	if validation.Length >= 12 {
		score++
	}
	if validation.ClassesMatched >= 3 {
		score++
	}
	if validation.ClassesMatched == 4 {
		score++
	}

	if score > 4 {
		score = 4
	}

	return StrengthResult{
		Score:      score,
		Label:      strengthLabel(score),
		Validation: validation,
	}
}

func HashArgon2id(plainPassword string) (string, error) {
	salt, err := randomBytes(saltSize)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(plainPassword), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argonMemory, argonTime, argonThreads, saltB64, hashB64), nil
}

func VerifyArgon2id(plainPassword, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, err
	}
	if version != argon2.Version {
		return false, errors.New("incompatible argon2 version")
	}

	var memory uint32
	var timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	calculated := argon2.IDKey([]byte(plainPassword), salt, timeCost, memory, threads, uint32(len(hash)))
	return subtle.ConstantTimeCompare(hash, calculated) == 1, nil
}

func EncryptForTransportBase64(plainPassword string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce, err := randomBytes(aead.NonceSize())
	if err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nil, nonce, []byte(plainPassword), nil)
	packed := append(nonce, ciphertext...)
	return base64.RawStdEncoding.EncodeToString(packed), nil
}

func NewTransportKey() ([]byte, error) {
	return randomBytes(32)
}

func validateGenerateArgs(length int, count int) error {
	switch {
	case length < MinPasswordLength:
		return ErrPasswordLengthTooLow
	case count < MinPasswordsCount:
		return ErrTooLowPasswordsCount
	case count > MaxPasswordsCount:
		return ErrTooBigPasswordsCount
	}

	return nil
}

func generateOne(length int) (string, error) {
	runes := make([]rune, 0, length)

	requiredPools := [][]rune{upperChars, lowerChars, digitChars, specialChars}
	for _, pool := range requiredPools {
		ch, err := getRandRune(pool)
		if err != nil {
			return "", err
		}
		runes = append(runes, ch)
	}

	poolOfChars := slices.Concat(upperChars, lowerChars, digitChars, specialChars)
	for len(runes) < length {
		ch, err := getRandRune(poolOfChars)
		if err != nil {
			return "", err
		}
		runes = append(runes, ch)
	}

	if err := secureShuffle(runes); err != nil {
		return "", err
	}

	return string(runes), nil
}

func getRandRune(runes []rune) (rune, error) {
	if len(runes) == 0 {
		return 0, errors.New("rune pool is empty")
	}

	idx, err := randNumber(int64(len(runes)))
	if err != nil {
		return 0, err
	}

	return runes[idx], nil
}

func secureShuffle(runes []rune) error {
	for i := len(runes) - 1; i > 0; i-- {
		j, err := randNumber(int64(i + 1))
		if err != nil {
			return err
		}

		runes[i], runes[j] = runes[j], runes[i]
	}

	return nil
}

func randNumber(n int64) (int, error) {
	if n <= 0 {
		return 0, errors.New("random bound must be positive")
	}

	value, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		return 0, err
	}

	return int(value.Int64()), nil
}

func randomBytes(size int) ([]byte, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func strengthLabel(score int) string {
	switch score {
	case 0, 1:
		return "weak"
	case 2:
		return "fair"
	case 3:
		return "good"
	default:
		return "strong"
	}
}
