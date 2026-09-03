package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSignVerify_RoundTrip(t *testing.T) {
	secret := []byte("test-secret")
	userID := uuid.New()

	token, err := Sign(secret, userID, time.Hour)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	got, err := Verify(secret, token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if got != userID {
		t.Fatalf("Verify() userID = %v, want %v", got, userID)
	}
}

func TestVerify_Expired(t *testing.T) {
	secret := []byte("test-secret")
	token, err := Sign(secret, uuid.New(), -time.Minute)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	if _, err := Verify(secret, token); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("Verify() error = %v, want ErrTokenExpired", err)
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	token, err := Sign([]byte("secret-a"), uuid.New(), time.Hour)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	if _, err := Verify([]byte("secret-b"), token); !errors.Is(err, ErrTokenInvalidSignature) {
		t.Fatalf("Verify() error = %v, want ErrTokenInvalidSignature", err)
	}
}

func TestVerify_Malformed(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"missing segments", "abc.def"},
		{"too many segments", "a.b.c.d"},
		{"invalid base64 signature", "aGVhZGVy.cGF5bG9hZA.not-base64!!!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Verify([]byte("secret"), tt.token); !errors.Is(err, ErrTokenMalformed) {
				t.Fatalf("Verify() error = %v, want ErrTokenMalformed", err)
			}
		})
	}
}

func TestVerify_TamperedPayload(t *testing.T) {
	secret := []byte("test-secret")
	token, err := Sign(secret, uuid.New(), time.Hour)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	tampered := token[:len(token)-4] + "abcd"
	if _, err := Verify(secret, tampered); err == nil {
		t.Fatal("Verify() error = nil, want a signature/malformed error for tampered token")
	}
}
