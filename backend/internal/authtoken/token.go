// Package authtoken issues and verifies the bearer tokens used across the
// v1 API (api-spec.md: "Authorization: Bearer <JWT>"). It implements the
// minimal HMAC-SHA256 JWT subset needed here with only the stdlib, so the
// backend doesn't need a third-party JWT dependency for V1.
package authtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type Claims struct {
	Subject   string `json:"sub"`
	Email     string `json:"email"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

// Issue returns a signed token and its lifetime in seconds (for the API's
// `expiresIn` response field).
func (i *Issuer) Issue(userID, email string) (token string, expiresIn int64) {
	now := time.Now()
	claims := Claims{
		Subject:   userID,
		Email:     email,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(i.ttl).Unix(),
	}
	header := base64URL(mustJSON(map[string]string{"alg": "HS256", "typ": "JWT"}))
	payload := base64URL(mustJSON(claims))
	signature := i.sign(header + "." + payload)
	return header + "." + payload + "." + signature, int64(i.ttl.Seconds())
}

func (i *Issuer) Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	expected := i.sign(parts[0] + "." + parts[1])
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) != 1 {
		return nil, ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrInvalidToken
	}
	return &claims, nil
}

func (i *Issuer) sign(signingInput string) string {
	mac := hmac.New(sha256.New, i.secret)
	mac.Write([]byte(signingInput))
	return base64URL(mac.Sum(nil))
}

func base64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
