package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testUser = "10000000-0000-4000-8000-000000000001"
const testSession = "10000000-0000-4000-8000-000000000002"

func testService(t *testing.T) *JWTTokenService {
	t.Helper()
	s, err := NewJWTTokenService(JWTConfig{Secret: "test-secret-at-least-thirty-two-bytes", Issuer: "test-api", AccessTTL: time.Minute, RefreshTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestTokenTypesAndUniqueRotation(t *testing.T) {
	s := testService(t)
	id := Identity{UserID: testUser, SessionID: testSession}
	first, err := s.GenerateTokens(id)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := s.Parse(first.AccessToken)
	if err != nil || parsed.UserID != testUser || parsed.SessionID != testSession {
		t.Fatalf("access identity: %v %v", parsed, err)
	}
	if _, err := s.ParseRefresh(first.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Parse(first.RefreshToken); err == nil {
		t.Fatal("refresh accepted as access")
	}
	if _, err := s.ParseRefresh(first.AccessToken); err == nil {
		t.Fatal("access accepted as refresh")
	}
	second, err := s.GenerateTokens(id)
	if err != nil {
		t.Fatal(err)
	}
	if first.RefreshToken == second.RefreshToken || first.AccessToken == second.AccessToken {
		t.Fatal("tokens reused within same second")
	}
	if first.ExpiresIn != 60 {
		t.Fatalf("expires_in=%d", first.ExpiresIn)
	}
}
func TestRejectInvalidJWTClaims(t *testing.T) {
	s := testService(t)
	base := func() tokenClaims {
		return tokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: testUser, Issuer: "test-api", Audience: jwt.ClaimStrings{"test-api:access"}, ID: testUser, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}, SessionID: testSession, Kind: "access"}
	}
	cases := []struct {
		name   string
		change func(*tokenClaims)
		method jwt.SigningMethod
		secret string
	}{
		{name: "wrong issuer", change: func(c *tokenClaims) { c.Issuer = "another-api" }},
		{name: "wrong audience", change: func(c *tokenClaims) { c.Audience = jwt.ClaimStrings{"another-api:access"} }},
		{name: "missing issued at", change: func(c *tokenClaims) { c.IssuedAt = nil }},
		{name: "missing expiry", change: func(c *tokenClaims) { c.ExpiresAt = nil }},
		{name: "expired", change: func(c *tokenClaims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute)) }},
		{name: "future issued at", change: func(c *tokenClaims) { c.IssuedAt = jwt.NewNumericDate(time.Now().Add(time.Hour)) }},
		{name: "missing subject", change: func(c *tokenClaims) { c.Subject = "" }},
		{name: "missing session", change: func(c *tokenClaims) { c.SessionID = "" }},
		{name: "missing token id", change: func(c *tokenClaims) { c.ID = "" }},
		{name: "wrong kind", change: func(c *tokenClaims) { c.Kind = "refresh" }},
		{name: "wrong algorithm", method: jwt.SigningMethodHS384},
		{name: "wrong signature", secret: "wrong-secret-at-least-thirty-two-bytes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims := base()
			if tc.change != nil {
				tc.change(&claims)
			}
			method := tc.method
			if method == nil {
				method = jwt.SigningMethodHS256
			}
			secret := tc.secret
			if secret == "" {
				secret = s.cfg.Secret
			}
			value, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Parse(value); err == nil {
				t.Fatal("invalid JWT accepted")
			}
		})
	}
	if _, err := s.Parse("malformed"); err == nil {
		t.Fatal("malformed JWT accepted")
	}
}
func TestInvalidJWTConfiguration(t *testing.T) {
	for _, cfg := range []JWTConfig{{Secret: "short", Issuer: "api", AccessTTL: time.Minute, RefreshTTL: time.Hour}, {Secret: "test-secret-at-least-thirty-two-bytes", AccessTTL: time.Minute, RefreshTTL: time.Hour}, {Secret: "test-secret-at-least-thirty-two-bytes", Issuer: "api", AccessTTL: time.Hour, RefreshTTL: time.Minute}} {
		if _, err := NewJWTTokenService(cfg); err == nil {
			t.Fatal("invalid config accepted")
		}
	}
}
