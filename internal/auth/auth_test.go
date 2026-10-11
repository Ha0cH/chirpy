package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTValid(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	parsedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	if parsedID != userID {
		t.Errorf("Expected user ID %v, got %v", userID, parsedID)
	}
}

func TestJWTExpired(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	token, err := MakeJWT(userID, secret, -time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Error("Expected an error for expired JWT, got nil")
	}
}

func TestJWTWrongSecret(t *testing.T) {
	userID := uuid.New()

	token, err := MakeJWT(userID, "correct-secret", time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	_, err = ValidateJWT(token, "wrong-secret")
	if err == nil {
		t.Error("Expected an error for incorrect secret, got nil")
	}
}
