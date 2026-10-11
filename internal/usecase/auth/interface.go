package auth

import (
	"context"
	"time"

	tokenauth "github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/go-rest-api/internal/domain"
)

type Usecase interface {
	Login(context.Context, LoginRequest) (*tokenauth.TokenPair, error)
	RefreshToken(context.Context, RefreshTokenRequest) (*tokenauth.TokenPair, error)
	Logout(context.Context) error
	Me(context.Context) (*MeResponse, error)
	Authenticate(context.Context, string) (*tokenauth.Identity, error)
	HasPermission(context.Context, string) (bool, error)
}

type Session struct {
	PasswordHash string // Expected login credential, checked under a user row lock.
	ID           string
	UserID       string
	TokenHash    string
	ExpiresAt    time.Time
}

type Repository interface {
	FindByLogin(context.Context, string) (*domain.User, error)
	FindByID(context.Context, string) (*domain.User, error)
	CreateSession(context.Context, Session) error
	RotateSession(context.Context, string, Session) (bool, error)
	SessionActive(context.Context, string, string) (bool, error)
	RevokeSession(context.Context, string, string) error
}

type PermissionRepository interface {
	GetPermissionsByUserID(context.Context, string) ([]string, error)
}
