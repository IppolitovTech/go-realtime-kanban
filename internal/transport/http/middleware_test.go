package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/IppolitovTech/go-realtime-kanban/internal/auth"
	transporthttp "github.com/IppolitovTech/go-realtime-kanban/internal/transport/http"
)

func TestJWTAuth(t *testing.T) {
	secret := []byte("test-secret")
	userID := uuid.New()
	validToken, err := auth.Sign(secret, userID, time.Hour)
	if err != nil {
		t.Fatalf("auth.Sign() error = %v", err)
	}
	expiredToken, err := auth.Sign(secret, userID, -time.Minute)
	if err != nil {
		t.Fatalf("auth.Sign() error = %v", err)
	}

	var gotUserID uuid.UUID
	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		gotUserID = transporthttp.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := transporthttp.JWTAuth(secret)(next)

	tests := []struct {
		name       string
		setupReq   func(r *http.Request)
		wantStatus int
		wantNext   bool
	}{
		{"valid bearer header", func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+validToken)
		}, http.StatusOK, true},
		{"valid token query param (WS handshake)", func(r *http.Request) {
			q := r.URL.Query()
			q.Set("token", validToken)
			r.URL.RawQuery = q.Encode()
		}, http.StatusOK, true},
		{"missing token", func(r *http.Request) {}, http.StatusUnauthorized, false},
		{"malformed header", func(r *http.Request) {
			r.Header.Set("Authorization", validToken)
		}, http.StatusUnauthorized, false},
		{"expired token", func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+expiredToken)
		}, http.StatusUnauthorized, false},
		{"tampered token", func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+validToken+"xx")
		}, http.StatusUnauthorized, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled = false
			gotUserID = uuid.Nil
			req := httptest.NewRequest("GET", "/boards", nil)
			tt.setupReq(req)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNext {
				t.Errorf("next called = %v, want %v", nextCalled, tt.wantNext)
			}
			if tt.wantNext && gotUserID != userID {
				t.Errorf("context userID = %v, want %v", gotUserID, userID)
			}
		})
	}
}
