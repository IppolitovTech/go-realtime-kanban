// Package auth implements a minimal HS256 JWT sign/verify pair. There's no
// standard-library JWT support and this project deliberately avoids adding
// a JWT dependency for it (see docs/ru/adr/005-jwt-vs-sessions.md) — the
// format is simple enough that hand-rolling it, with proper constant-time
// signature comparison, is both correct and a better demonstration of the
// underlying mechanism than a library import would be.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTokenMalformed        = errors.New("token malformed")
	ErrTokenInvalidSignature = errors.New("token signature invalid")
	ErrTokenExpired          = errors.New("token expired")
)

// header is constant for every token this package issues, so it's encoded
// once at init time rather than on every Sign call.
var encodedHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

type claims struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

// Sign issues a JWT for userID, valid for ttl from now.
func Sign(secret []byte, userID uuid.UUID, ttl time.Duration) (string, error) {
	now := time.Now()
	payload, err := json.Marshal(claims{
		Sub: userID.String(),
		Iat: now.Unix(),
		Exp: now.Add(ttl).Unix(),
	})
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)

	signingInput := encodedHeader + "." + encodedPayload
	signature := sign(secret, signingInput)

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// Verify checks token's signature and expiry against secret and returns the
// userID it was issued for.
func Verify(secret []byte, token string) (uuid.UUID, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return uuid.Nil, ErrTokenMalformed
	}
	encodedHeaderPart, encodedPayload, encodedSignature := parts[0], parts[1], parts[2]

	wantSignature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return uuid.Nil, ErrTokenMalformed
	}
	gotSignature := sign(secret, encodedHeaderPart+"."+encodedPayload)
	if !hmac.Equal(wantSignature, gotSignature) {
		return uuid.Nil, ErrTokenInvalidSignature
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return uuid.Nil, ErrTokenMalformed
	}
	var c claims
	if err := json.Unmarshal(payloadJSON, &c); err != nil {
		return uuid.Nil, ErrTokenMalformed
	}

	if time.Now().Unix() >= c.Exp {
		return uuid.Nil, ErrTokenExpired
	}

	userID, err := uuid.Parse(c.Sub)
	if err != nil {
		return uuid.Nil, ErrTokenMalformed
	}
	return userID, nil
}

func sign(secret []byte, signingInput string) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	return mac.Sum(nil)
}
