package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	issueTime := time.Now().UTC()
	jwtIssueTime := jwt.NewNumericDate(issueTime)
	jwtExpiresTime := jwt.NewNumericDate(issueTime.Add(expiresIn))
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Issuer: "chirpy", IssuedAt: jwtIssueTime, ExpiresAt: jwtExpiresTime, Subject: userID.String()})
	stringToken, err := newToken.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", fmt.Errorf("error signing the token with key: %w", err)
	}
	return stringToken, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	if !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user id in token")
	}

	return userID, nil
}

func GetBearerToken(header http.Header) (string, error) {
	authHeader := header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("empty authorization header")
	}
	tokenList := strings.Fields(authHeader)
	if len(tokenList) != 2 || tokenList[0] != "Bearer" {
		return "", fmt.Errorf("unknown authorization token")
	}
	return tokenList[1], nil
}
