package auth

import (
	"context"
	stderrors "errors"
	"sync"
	"testing"
	"time"

	tokenauth "github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/golang-pkgs/errors"
	"golang.org/x/crypto/bcrypt"
)

const testUserID = "10000000-0000-4000-8000-000000000001"

type memoryAuthRepository struct {
	user     *domain.User
	err      error
	storeErr error
	mu       sync.Mutex
	session  Session
	revoked  bool
}

func (r *memoryAuthRepository) FindByLogin(context.Context, string) (*domain.User, error) {
	return r.user, r.err
}
func (r *memoryAuthRepository) FindByID(context.Context, string) (*domain.User, error) {
	return r.user, r.err
}
func (r *memoryAuthRepository) CreateSession(_ context.Context, s Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.storeErr != nil {
		return r.storeErr
	}
	r.session = s
	r.revoked = false
	return nil
}
func (r *memoryAuthRepository) RotateSession(_ context.Context, old string, s Session) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return false, r.err
	}
	if r.revoked || r.session.ID != s.ID || r.session.TokenHash != old {
		return false, nil
	}
	r.session = s
	return true, nil
}
func (r *memoryAuthRepository) SessionActive(_ context.Context, id, userID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.revoked && r.session.ID == id && r.session.UserID == userID, r.err
}
func (r *memoryAuthRepository) RevokeSession(context.Context, string, string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revoked = true
	return r.err
}

type memoryPermissions struct {
	values []string
	err    error
}

func (p *memoryPermissions) GetPermissionsByUserID(context.Context, string) ([]string, error) {
	return p.values, p.err
}
func newAuthFixture(t *testing.T) (Usecase, *memoryAuthRepository, *memoryPermissions) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(" password "), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	repo := &memoryAuthRepository{user: &domain.User{ID: testUserID, Username: "tester", FullName: "Tester", Password: string(hash), Status: domain.UserStatusActive, Roles: []string{"READER"}}}
	permissions := &memoryPermissions{values: []string{tokenauth.UserRead}}
	token, err := tokenauth.NewJWTTokenService(tokenauth.JWTConfig{Secret: "unit-test-secret-at-least-thirty-two-bytes", Issuer: "auth-test", AccessTTL: time.Minute, RefreshTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	return NewUsecase(repo, permissions, token), repo, permissions
}
func expectCode(t *testing.T, err error, code errors.Code) {
	t.Helper()
	var appErr *errors.AppError
	if !stderrors.As(err, &appErr) || appErr.Code != code {
		t.Fatalf("error=%v want code=%d", err, code)
	}
}
func TestLoginRefreshLogoutLifecycle(t *testing.T) {
	uc, repo, permissions := newAuthFixture(t)
	ctx := context.Background()
	pair, err := uc.Login(ctx, LoginRequest{UsernameOrEmail: " tester ", Password: " password "})
	if err != nil {
		t.Fatal(err)
	}
	if repo.session.TokenHash != HashToken(pair.RefreshToken) || repo.session.PasswordHash != repo.user.Password {
		t.Fatal("session was not linked to verified credentials")
	}
	id, err := uc.Authenticate(ctx, pair.AccessToken)
	if err != nil || id.UserID != testUserID {
		t.Fatal("valid session rejected")
	}
	ctx = context.WithValue(ctx, tokenauth.IdentityKey, id)
	allowed, err := uc.HasPermission(ctx, tokenauth.UserRead)
	if err != nil || !allowed {
		t.Fatal("permission rejected")
	}
	permissions.values = []string{}
	allowed, err = uc.HasPermission(ctx, tokenauth.UserRead)
	if err != nil || allowed {
		t.Fatal("stale permission accepted")
	}
	me, err := uc.Me(ctx)
	if err != nil || me.Username != "tester" {
		t.Fatal("current user missing username")
	}
	repo.user.Roles = []string{"NEW_ROLE"}
	rotated, err := uc.RefreshToken(ctx, RefreshTokenRequest{RefreshToken: pair.RefreshToken})
	if err != nil {
		t.Fatal(err)
	}
	current, err := uc.Authenticate(ctx, rotated.AccessToken)
	if err != nil || len(current.Roles) != 1 || current.Roles[0] != "NEW_ROLE" {
		t.Fatal("stale roles used")
	}
	_, err = uc.RefreshToken(ctx, RefreshTokenRequest{RefreshToken: pair.RefreshToken})
	expectCode(t, err, errors.Unauthorized)
	if err := uc.Logout(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = uc.Authenticate(ctx, rotated.AccessToken)
	expectCode(t, err, errors.Unauthorized)
	_, err = uc.RefreshToken(ctx, RefreshTokenRequest{RefreshToken: rotated.RefreshToken})
	expectCode(t, err, errors.Unauthorized)
}
func TestAuthFailures(t *testing.T) {
	uc, repo, permissions := newAuthFixture(t)
	ctx := context.Background()
	for _, password := range []string{"wrong", "password", "", string(make([]byte, 73))} {
		_, err := uc.Login(ctx, LoginRequest{UsernameOrEmail: "tester", Password: password})
		expectCode(t, err, errors.Unauthorized)
	}
	repo.user.Status = domain.UserStatusInactive
	_, err := uc.Login(ctx, LoginRequest{UsernameOrEmail: "tester", Password: " password "})
	expectCode(t, err, errors.Unauthorized)
	repo.user.Status = domain.UserStatusActive
	repo.err = errors.New(errors.NotFound, "missing")
	_, err = uc.Login(ctx, LoginRequest{UsernameOrEmail: "tester", Password: " password "})
	expectCode(t, err, errors.Unauthorized)
	databaseErr := stderrors.New("database unavailable")
	repo.err = databaseErr
	_, err = uc.Login(ctx, LoginRequest{UsernameOrEmail: "tester", Password: " password "})
	if !stderrors.Is(err, databaseErr) {
		t.Fatal("database failure hidden as bad credentials")
	}
	repo.err = nil
	repo.storeErr = databaseErr
	pair, err := uc.Login(ctx, LoginRequest{UsernameOrEmail: "tester", Password: " password "})
	if pair != nil || !stderrors.Is(err, databaseErr) {
		t.Fatal("issued tokens without persisting session")
	}
	repo.storeErr = nil
	pair, err = uc.Login(ctx, LoginRequest{UsernameOrEmail: "tester", Password: " password "})
	if err != nil {
		t.Fatal(err)
	}
	id, err := uc.Authenticate(ctx, pair.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	authenticated := context.WithValue(ctx, tokenauth.IdentityKey, id)
	permissions.err = databaseErr
	allowed, err := uc.HasPermission(authenticated, tokenauth.UserRead)
	if allowed || !stderrors.Is(err, databaseErr) {
		t.Fatal("permission database failure allowed access")
	}
	repo.user.Status = domain.UserStatusInactive
	_, err = uc.Authenticate(ctx, pair.AccessToken)
	expectCode(t, err, errors.Unauthorized)
	_, err = uc.RefreshToken(ctx, RefreshTokenRequest{RefreshToken: pair.RefreshToken})
	expectCode(t, err, errors.Unauthorized)
	_, err = uc.Me(ctx)
	expectCode(t, err, errors.Unauthorized)
	err = uc.Logout(context.WithValue(ctx, tokenauth.IdentityKey, (*tokenauth.Identity)(nil)))
	expectCode(t, err, errors.Unauthorized)
}
