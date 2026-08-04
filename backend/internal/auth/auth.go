package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLength  = 16
	argonKeyLength   = 32
)

func HashPassword(password string) (string, error) {
	salt, err := randomBytes(argonSaltLength)
	if err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false
	}
	if memory != argonMemory || iterations != argonIterations || parallelism != argonParallelism {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func NewSecret(bytes int) (string, []byte, error) {
	raw, err := randomBytes(bytes)
	if err != nil {
		return "", nil, err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	return secret, Digest(secret), nil
}

func NewUserCode() (string, error) {
	raw, err := randomBytes(4)
	if err != nil {
		return "", err
	}
	value := hex.EncodeToString(raw)
	return strings.ToUpper(value[:4] + "-" + value[4:]), nil
}

func Digest(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

func ParseBearer(header string) (string, error) {
	scheme, token, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", errors.New("invalid bearer authorization")
	}
	return token, nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must contain at least 8 characters")
	}
	if len(password) > 1024 {
		return errors.New("password is too long")
	}
	return nil
}

func randomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, errors.New("invalid random byte length: " + strconv.Itoa(length))
	}
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		return nil, err
	}
	return value, nil
}
