package auth

import (
	"errors"
	"time"

	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
	Secret     string
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type JWTTokenService struct{ cfg JWTConfig }

type tokenClaims struct {
	jwt.RegisteredClaims
	SessionID string `json:"sid"`
	Kind      string `json:"token_type"`
}

func NewJWTTokenService(cfg JWTConfig) (*JWTTokenService, error) {
	if len(cfg.Secret) < 32 {
		return nil, errors.New("JWT secret must contain at least 32 bytes")
	}
	if cfg.Issuer == "" || cfg.AccessTTL <= 0 || cfg.RefreshTTL <= cfg.AccessTTL {
		return nil, errors.New("invalid JWT issuer or lifetimes")
	}
	return &JWTTokenService{cfg: cfg}, nil
}

func (s *JWTTokenService) GenerateTokens(id Identity) (*TokenPair, error) {
	if !helper.ValidUUID(id.UserID) || !helper.ValidUUID(id.SessionID) {
		return nil, errors.New("invalid token identity")
	}
	now := time.Now().UTC().Truncate(time.Second)
	access, _, err := s.sign(id, "access", now, s.cfg.AccessTTL)
	if err != nil {
		return nil, err
	}
	refresh, expires, err := s.sign(id, "refresh", now, s.cfg.RefreshTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresIn: int64(s.cfg.AccessTTL / time.Second), RefreshExpiresAt: expires}, nil
}

func (s *JWTTokenService) sign(id Identity, kind string, now time.Time, ttl time.Duration) (string, time.Time, error) {
	jti, err := helper.NewUUID()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := now.Add(ttl)
	claims := tokenClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: id.UserID, Issuer: s.cfg.Issuer, Audience: jwt.ClaimStrings{s.cfg.Issuer + ":" + kind}, ID: jti, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires)}, SessionID: id.SessionID, Kind: kind}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.Secret))
	return value, expires, err
}

func (s *JWTTokenService) Parse(value string) (*Identity, error) { return s.parse(value, "access") }
func (s *JWTTokenService) ParseRefresh(value string) (*Identity, error) {
	return s.parse(value, "refresh")
}
func (s *JWTTokenService) parse(value, kind string) (*Identity, error) {
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(value, claims, func(_ *jwt.Token) (any, error) { return []byte(s.cfg.Secret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(s.cfg.Issuer), jwt.WithAudience(s.cfg.Issuer+":"+kind), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.IssuedAt == nil || claims.Kind != kind || !helper.ValidUUID(claims.Subject) || !helper.ValidUUID(claims.SessionID) || !helper.ValidUUID(claims.ID) {
		return nil, errors.New("invalid token claims")
	}
	return &Identity{UserID: claims.Subject, SessionID: claims.SessionID, TokenID: claims.ID}, nil
}
