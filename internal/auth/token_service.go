package auth

import "time"

type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	TokenType        string    `json:"token_type"`
	ExpiresIn        int64     `json:"expires_in"`
	RefreshExpiresAt time.Time `json:"-"`
}

type TokenService interface {
	GenerateTokens(identity Identity) (*TokenPair, error)
	Parse(token string) (*Identity, error)
	ParseRefresh(token string) (*Identity, error)
}
