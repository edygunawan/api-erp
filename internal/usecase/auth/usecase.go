package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"slices"
	"strings"

	tokenauth "github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/golang-pkgs/errors"
	"golang.org/x/crypto/bcrypt"
)

type usecase struct {
	repo        Repository
	permissions PermissionRepository
	token       tokenauth.TokenService
}

func NewUsecase(repo Repository, permissions PermissionRepository, token tokenauth.TokenService) Usecase {
	return &usecase{repo: repo, permissions: permissions, token: token}
}
func unauthorized() error { return errors.New(errors.Unauthorized, "invalid credentials or session") }
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
func missing(err error) bool {
	var e *errors.AppError
	return stderrors.As(err, &e) && e.Code == errors.NotFound
}

func (u *usecase) Login(ctx context.Context, req LoginRequest) (*tokenauth.TokenPair, error) {
	req.UsernameOrEmail = strings.TrimSpace(req.UsernameOrEmail)
	if req.UsernameOrEmail == "" || req.Password == "" || len(req.Password) > 72 {
		return nil, unauthorized()
	}
	user, err := u.repo.FindByLogin(ctx, req.UsernameOrEmail)
	if err != nil {
		if missing(err) {
			return nil, unauthorized()
		}
		return nil, err
	}
	if user == nil || user.Status != domain.UserStatusActive {
		return nil, unauthorized()
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		return nil, unauthorized()
	}
	sessionID, err := helper.NewUUID()
	if err != nil {
		return nil, err
	}
	pair, err := u.token.GenerateTokens(tokenauth.Identity{UserID: user.ID, SessionID: sessionID})
	if err != nil {
		return nil, err
	}
	err = u.repo.CreateSession(ctx, Session{ID: sessionID, UserID: user.ID, PasswordHash: user.Password, TokenHash: HashToken(pair.RefreshToken), ExpiresAt: pair.RefreshExpiresAt})
	if err != nil {
		return nil, err
	}
	return pair, nil
}

func (u *usecase) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*tokenauth.TokenPair, error) {
	if req.RefreshToken == "" {
		return nil, unauthorized()
	}
	id, err := u.token.ParseRefresh(req.RefreshToken)
	if err != nil {
		return nil, unauthorized()
	}
	// Reload status and roles; never trust authorization data from an old token.
	user, err := u.activeUser(ctx, id.UserID)
	if err != nil {
		return nil, err
	}
	id.Roles = user.Roles
	pair, err := u.token.GenerateTokens(*id)
	if err != nil {
		return nil, err
	}
	rotated, err := u.repo.RotateSession(ctx, HashToken(req.RefreshToken), Session{ID: id.SessionID, UserID: id.UserID, TokenHash: HashToken(pair.RefreshToken), ExpiresAt: pair.RefreshExpiresAt})
	if err != nil {
		return nil, err
	}
	if !rotated {
		return nil, unauthorized()
	}
	return pair, nil
}

func (u *usecase) activeUser(ctx context.Context, id string) (*domain.User, error) {
	user, err := u.repo.FindByID(ctx, id)
	if err != nil {
		if missing(err) {
			return nil, unauthorized()
		}
		return nil, err
	}
	if user == nil || user.Status != domain.UserStatusActive {
		return nil, unauthorized()
	}
	return user, nil
}

func (u *usecase) Authenticate(ctx context.Context, value string) (*tokenauth.Identity, error) {
	id, err := u.token.Parse(value)
	if err != nil {
		return nil, unauthorized()
	}
	user, err := u.activeUser(ctx, id.UserID)
	if err != nil {
		return nil, err
	}
	active, err := u.repo.SessionActive(ctx, id.SessionID, id.UserID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, unauthorized()
	}
	id.Roles = user.Roles
	return id, nil
}

func (u *usecase) Logout(ctx context.Context) error {
	id, ok := tokenauth.GetIdentity(ctx)
	if !ok || id == nil {
		return unauthorized()
	}
	return u.repo.RevokeSession(ctx, id.SessionID, id.UserID)
}

func (u *usecase) Me(ctx context.Context) (*MeResponse, error) {
	id, ok := tokenauth.GetIdentity(ctx)
	if !ok || id == nil {
		return nil, unauthorized()
	}
	user, err := u.activeUser(ctx, id.UserID)
	if err != nil {
		return nil, err
	}
	permissions, err := u.permissions.GetPermissionsByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &MeResponse{ID: user.ID, FullName: user.FullName, Username: user.Username, Email: user.Email, Phone: user.Phone, Roles: user.Roles, Permissions: permissions}, nil
}

func (u *usecase) HasPermission(ctx context.Context, permission string) (bool, error) {
	id, ok := tokenauth.GetIdentity(ctx)
	if !ok || id == nil {
		return false, unauthorized()
	}
	permissions, err := u.permissions.GetPermissionsByUserID(ctx, id.UserID)
	if err != nil {
		return false, err
	}
	return slices.Contains(permissions, permission), nil
}
