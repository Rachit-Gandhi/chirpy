package auth

import (
	"fmt"

	"github.com/alexedwards/argon2id"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	//hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("error getting hash for password: %w", err)
	}
	return hashedPassword, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	isok, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, fmt.Errorf("error verifying password and hash: %w", err)
	}
	return isok, nil
}
