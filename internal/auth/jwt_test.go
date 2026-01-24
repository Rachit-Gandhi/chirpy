package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	token, err := MakeJWT(userID, secret, time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	gotUserID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT returned error: %v", err)
	}

	if gotUserID != userID {
		t.Fatalf("expected userID %v, got %v", userID, gotUserID)
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	secret := "correct-secret"
	wrongSecret := "wrong-secret"
	userID := uuid.New()

	token, err := MakeJWT(userID, secret, time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Fatalf("expected error with wrong secret, got nil")
	}
}

func TestValidateJWT_Expired(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	token, err := MakeJWT(userID, secret, -time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Fatalf("expected error for expired token, got nil")
	}
}

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		headerValue string
		wantToken   string
		wantErr     bool
	}{
		{
			name:        "missing header",
			headerValue: "",
			wantErr:     true,
		},
		{
			name:        "wrong scheme",
			headerValue: "Basic abc123",
			wantErr:     true,
		},
		{
			name:        "bearer without token",
			headerValue: "Bearer",
			wantErr:     true,
		},
		{
			name:        "valid bearer token",
			headerValue: "Bearer token123",
			wantToken:   "token123",
			wantErr:     false,
		},
		{
			name:        "extra spaces",
			headerValue: "Bearer    token123",
			wantToken:   "token123",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := http.Header{}
			if tt.headerValue != "" {
				header.Set("Authorization", tt.headerValue)
			}

			token, err := GetBearerToken(header)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if token != tt.wantToken {
				t.Fatalf("expected token %q, got %q", tt.wantToken, token)
			}
		})
	}
}
