package app

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	tokenauth "github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/go-rest-api/internal/http/router"
	"github.com/edygunawan/go-rest-api/internal/repository/factory"
	"github.com/edygunawan/go-rest-api/internal/testutil"
	authuc "github.com/edygunawan/go-rest-api/internal/usecase/auth"
	"github.com/edygunawan/go-rest-api/internal/usecase/bootstrap"
	roleuc "github.com/edygunawan/go-rest-api/internal/usecase/role"
	useruc "github.com/edygunawan/go-rest-api/internal/usecase/user"
	apperrors "github.com/edygunawan/golang-pkgs/errors"
	"github.com/edygunawan/golang-pkgs/logger"
	"golang.org/x/crypto/bcrypt"
)

const adminRole = "00000000-0000-4000-8000-000000000001"
const userReadPermission = "00000000-0000-4000-8001-000000000001"
const missingID = "ffffffff-ffff-4fff-8fff-ffffffffffff"

type foundationFixture struct {
	repos   factory.Repositories
	db      *sql.DB
	cfg     database.Config
	handler http.Handler
	adminID string
}

func setupFoundation(t *testing.T, cfg database.Config) *foundationFixture {
	t.Helper()
	ctx := context.Background()
	db, cfg := testutil.Isolate(t, cfg)
	repos, err := factory.New(cfg.Driver, db)
	if err != nil {
		t.Fatal(err)
	}
	token, err := tokenauth.NewJWTTokenService(tokenauth.JWTConfig{Secret: "integration-secret-at-least-thirty-two-bytes", Issuer: "foundation-test", AccessTTL: time.Minute, RefreshTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	// Keep API fixtures independent of the migration's documented default credentials.
	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id='00000000-0000-4000-8002-000000000001'`); err != nil {
		t.Fatal(err)
	}
	email := "admin@example.test"
	adminID, err := bootstrap.New(repos.User).Run(ctx, bootstrap.Request{FullName: "Administrator", Username: "admin", Email: &email, Password: " admin password "})
	if err != nil {
		t.Fatal(err)
	}
	log := logger.New("foundation-test", false)
	logger.SetDefault(log)
	handlers, authUC := buildHTTPHandlers(repos, token)
	return &foundationFixture{repos: repos, db: db, cfg: cfg, adminID: adminID, handler: router.NewRouter(router.Dependencies{Logger: log, Handlers: handlers, AuthUC: authUC, CORSAllowedOrigins: []string{"https://client.example.test"}})}
}
func (f *foundationFixture) request(method, path, token string, body any) *httptest.ResponseRecorder {
	var encoded []byte
	if body != nil {
		encoded, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(string(encoded)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	return w
}
func wantStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("status=%d want=%d: %s", w.Code, want, w.Body.String())
	}
}
func decodeResponse[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func (f *foundationFixture) login(t *testing.T, login, password string) tokenauth.TokenPair {
	t.Helper()
	w := f.request("POST", "/auth/login", "", map[string]string{"username_or_email": login, "password": password})
	wantStatus(t, w, 200)
	return decodeResponse[tokenauth.TokenPair](t, w)
}
func TestFoundationIntegration(t *testing.T) { testutil.ForEachDatabase(t, testFoundationIntegration) }
func testFoundationIntegration(t *testing.T, cfg database.Config) {
	f := setupFoundation(t, cfg)
	admin := f.login(t, "ADMIN", " admin password ")
	wantStatus(t, f.request("GET", "/health", "", nil), 200)
	wantStatus(t, f.request("GET", "/users", "", nil), 401)
	wantStatus(t, f.request("GET", "/users", admin.RefreshToken, nil), 401)
	wantStatus(t, f.request("POST", "/auth/refresh-token", "", map[string]string{"refresh_token": admin.AccessToken}), 401)
	wantStatus(t, f.request("POST", "/auth/login", "", map[string]string{"username_or_email": "admin", "password": "admin password"}), 401)
	wantStatus(t, f.request("POST", "/auth/login", "", map[string]string{"username_or_email": "missing", "password": "wrong"}), 401)
	me := f.request("GET", "/auth/me", admin.AccessToken, nil)
	wantStatus(t, me, 200)
	var profile struct {
		Username           string
		Roles, Permissions []string
	}
	profile = decodeResponse[struct {
		Username           string
		Roles, Permissions []string
	}](t, me)
	if profile.Username != "admin" || len(profile.Roles) != 1 || profile.Roles[0] != "ADMIN" || len(profile.Permissions) != 17 {
		t.Fatalf("unexpected profile: %+v", profile)
	}
	if strings.Contains(me.Body.String(), "password") {
		t.Fatal("password leaked")
	}
	// Explicit bootstrap is repeatable and does not reset an existing password.
	email := "admin@example.test"
	id, err := bootstrap.New(f.repos.User).Run(context.Background(), bootstrap.Request{FullName: "Changed", Username: "admin", Email: &email, Password: "different password"})
	if err != nil || id != f.adminID {
		t.Fatalf("bootstrap repeat: %s %v", id, err)
	}
	f.login(t, "admin@example.test", " admin password ")
	wantStatus(t, f.request("POST", "/auth/login", "", map[string]string{"username_or_email": "admin", "password": "different password"}), 401)
	wantStatus(t, f.request("PUT", "/roles/"+adminRole, admin.AccessToken, roleuc.UpdateRoleRequest{Code: "RENAMED", Name: "Administrator"}), 403)
	wantStatus(t, f.request("DELETE", "/roles/"+adminRole, admin.AccessToken, nil), 403)
	wantStatus(t, f.request("DELETE", "/permissions/"+userReadPermission, admin.AccessToken, nil), 403)
	wantStatus(t, f.request("POST", "/roles", admin.AccessToken, map[string]any{"code": "ADMIN2", "name": "Role", "is_system": true}), 400)
	// Create a generic permission, use it in a role, and exercise permission CRUD.
	permission := f.request("POST", "/permissions", admin.AccessToken, map[string]string{"code": "audit.read", "name": "Read audit", "module": "audit"})
	wantStatus(t, permission, 201)
	var p struct{ ID string }
	p = decodeResponse[struct{ ID string }](t, permission)
	wantStatus(t, f.request("PUT", "/permissions/"+p.ID, admin.AccessToken, map[string]string{"code": "AUDIT.READ", "name": "Read audit log", "module": "AUDIT"}), 200)
	role := f.request("POST", "/roles", admin.AccessToken, roleuc.CreateRoleRequest{Code: "reader", Name: "Reader", Permissions: []string{userReadPermission, userReadPermission, p.ID}})
	wantStatus(t, role, 201)
	createdRole := decodeResponse[roleuc.RoleResponse](t, role)
	if createdRole.Code != "READER" || createdRole.IsSystem || len(createdRole.Permissions) != 2 {
		t.Fatalf("role=%+v", createdRole)
	}
	create := useruc.CreateUserRequest{FullName: "Reader", Username: "reader", Password: " reader password ", Roles: []string{createdRole.ID, createdRole.ID}}
	w := f.request("POST", "/users", admin.AccessToken, create)
	wantStatus(t, w, 201)
	createdUser := decodeResponse[useruc.UserResponse](t, w)
	if len(createdUser.Roles) != 1 || strings.Contains(w.Body.String(), "password") {
		t.Fatal("unsafe/incorrect user response")
	}
	wantStatus(t, f.request("POST", "/users", admin.AccessToken, create), 409)
	reader := f.login(t, "reader", " reader password ")
	wantStatus(t, f.request("GET", "/users", reader.AccessToken, nil), 200)
	wantStatus(t, f.request("GET", "/roles", reader.AccessToken, nil), 403)
	// Role permission revocation and restoration take effect with the same access token.
	wantStatus(t, f.request("PUT", "/roles/"+createdRole.ID, admin.AccessToken, roleuc.UpdateRoleRequest{Code: "READER", Name: "Reader", Permissions: []string{p.ID}}), 200)
	wantStatus(t, f.request("GET", "/users", reader.AccessToken, nil), 403)
	wantStatus(t, f.request("PUT", "/roles/"+createdRole.ID, admin.AccessToken, roleuc.UpdateRoleRequest{Code: "READER", Name: "Reader", Permissions: []string{userReadPermission}}), 200)
	wantStatus(t, f.request("GET", "/users", reader.AccessToken, nil), 200)
	// Failed relationship changes must roll back both entity and assignments.
	wantStatus(t, f.request("PUT", "/roles/"+createdRole.ID, admin.AccessToken, roleuc.UpdateRoleRequest{Code: "BROKEN", Name: "Broken", Permissions: []string{missingID}}), 400)
	roleAfter := decodeResponse[roleuc.RoleResponse](t, f.request("GET", "/roles/"+createdRole.ID, admin.AccessToken, nil))
	if roleAfter.Code != "READER" || len(roleAfter.Permissions) != 1 || roleAfter.Permissions[0] != userReadPermission {
		t.Fatal("role transaction did not roll back")
	}
	update := useruc.UpdateUserRequest{FullName: "Broken", Username: "reader", Roles: []string{missingID}}
	wantStatus(t, f.request("PUT", "/users/"+createdUser.ID, admin.AccessToken, update), 400)
	userAfter := decodeResponse[useruc.UserResponse](t, f.request("GET", "/users/"+createdUser.ID, admin.AccessToken, nil))
	if userAfter.FullName != "Reader" || len(userAfter.Roles) != 1 {
		t.Fatal("user transaction did not roll back")
	}
	create.Username = "broken"
	create.Roles = []string{missingID}
	wantStatus(t, f.request("POST", "/users", admin.AccessToken, create), 400)
	var count int
	if err := f.queryRow(`SELECT count(*) FROM users WHERE username='broken'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("create rollback count=%d err=%v", count, err)
	}
	// Removing user roles immediately denies permissions and /me returns empty arrays.
	update.FullName = "Reader"
	update.Roles = []string{}
	wantStatus(t, f.request("PUT", "/users/"+createdUser.ID, admin.AccessToken, update), 200)
	wantStatus(t, f.request("GET", "/users", reader.AccessToken, nil), 403)
	me = f.request("GET", "/auth/me", reader.AccessToken, nil)
	wantStatus(t, me, 200)
	if !strings.Contains(me.Body.String(), `"roles":[]`) || !strings.Contains(me.Body.String(), `"permissions":[]`) {
		t.Fatalf("empty arrays: %s", me.Body)
	}
	update.Roles = []string{createdRole.ID}
	wantStatus(t, f.request("PUT", "/users/"+createdUser.ID, admin.AccessToken, update), 200)
	wantStatus(t, f.request("GET", "/users", reader.AccessToken, nil), 200)
	// Rotation rejects reuse; logout invalidates access and refresh for that session.
	rotated := f.request("POST", "/auth/refresh-token", "", map[string]string{"refresh_token": reader.RefreshToken})
	wantStatus(t, rotated, 200)
	reader2 := decodeResponse[tokenauth.TokenPair](t, rotated)
	wantStatus(t, f.request("POST", "/auth/refresh-token", "", map[string]string{"refresh_token": reader.RefreshToken}), 401)
	var stored string
	if err := f.queryRow(`SELECT token_hash FROM refresh_sessions WHERE user_id=? AND revoked_at IS NULL LIMIT 1`, createdUser.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == reader2.RefreshToken || len(stored) != 64 {
		t.Fatal("refresh token stored without hashing")
	}
	wantStatus(t, f.request("POST", "/auth/logout", reader2.AccessToken, nil), 200)
	wantStatus(t, f.request("GET", "/auth/me", reader.AccessToken, nil), 401)
	wantStatus(t, f.request("GET", "/auth/me", reader2.AccessToken, nil), 401)
	wantStatus(t, f.request("POST", "/auth/refresh-token", "", map[string]string{"refresh_token": reader2.RefreshToken}), 401)
	// Password changes revoke sessions, even before their JWT expires.
	reader3 := f.login(t, "reader", " reader password ")
	password := " updated password "
	update.Password = &password
	wantStatus(t, f.request("PUT", "/users/"+createdUser.ID, admin.AccessToken, update), 200)
	wantStatus(t, f.request("GET", "/auth/me", reader3.AccessToken, nil), 401)
	wantStatus(t, f.request("POST", "/auth/refresh-token", "", map[string]string{"refresh_token": reader3.RefreshToken}), 401)
	wantStatus(t, f.request("POST", "/auth/login", "", map[string]string{"username_or_email": "reader", "password": " reader password "}), 401)
	reader4 := f.login(t, "reader", password)
	update.Password = nil
	update.Status = "INACTIVE"
	wantStatus(t, f.request("PUT", "/users/"+createdUser.ID, admin.AccessToken, update), 200)
	wantStatus(t, f.request("GET", "/auth/me", reader4.AccessToken, nil), 401)
	wantStatus(t, f.request("POST", "/auth/refresh-token", "", map[string]string{"refresh_token": reader4.RefreshToken}), 401)
	wantStatus(t, f.request("POST", "/auth/login", "", map[string]string{"username_or_email": "reader", "password": password}), 401)
	update.Status = "ACTIVE"
	wantStatus(t, f.request("PUT", "/users/"+createdUser.ID, admin.AccessToken, update), 200)
	wantStatus(t, f.request("GET", "/auth/me", reader4.AccessToken, nil), 401)
	f.login(t, "reader", password)
	var hash string
	if err := f.queryRow(`SELECT password FROM users WHERE id=?`, createdUser.ID).Scan(&hash); err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		t.Fatal("stored password invalid")
	}
	wantStatus(t, f.request("GET", "/users/"+missingID, admin.AccessToken, nil), 404)
	wantStatus(t, f.request("GET", "/users/not-a-uuid", admin.AccessToken, nil), 400)
	wantStatus(t, f.request("DELETE", "/permissions/"+p.ID, admin.AccessToken, nil), 200)
	wantStatus(t, f.request("DELETE", "/roles/"+createdRole.ID, admin.AccessToken, nil), 200)
	wantStatus(t, f.request("DELETE", "/roles/"+createdRole.ID, admin.AccessToken, nil), 404)
	wantStatus(t, f.request("GET", "/products", admin.AccessToken, nil), 404)
}
func TestConcurrentRefreshIntegration(t *testing.T) {
	testutil.ForEachDatabase(t, testConcurrentRefreshIntegration)
}
func testConcurrentRefreshIntegration(t *testing.T, cfg database.Config) {
	f := setupFoundation(t, cfg)
	pair := f.login(t, "admin", " admin password ")
	const attempts = 8
	codes := make(chan int, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- f.request("POST", "/auth/refresh-token", "", map[string]string{"refresh_token": pair.RefreshToken}).Code
		}()
	}
	wg.Wait()
	close(codes)
	success := 0
	for code := range codes {
		if code == 200 {
			success++
		} else if code != 401 {
			t.Fatalf("unexpected refresh status %d", code)
		}
	}
	if success != 1 {
		t.Fatalf("refresh winners=%d, want 1", success)
	}
}
func TestMigrationRollbackIntegration(t *testing.T) {
	testutil.ForEachDatabase(t, testMigrationRollbackIntegration)
}
func testMigrationRollbackIntegration(t *testing.T, cfg database.Config) {
	f := setupFoundation(t, cfg)
	if err := database.Migrate(f.cfg, -9); err != nil {
		t.Fatal(err)
	}
	var usersTable int
	query := `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_name='users'`
	if f.cfg.Driver == database.MySQL {
		query = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='users'`
	}
	if err := f.queryRow(query).Scan(&usersTable); err != nil || usersTable != 0 {
		t.Fatalf("rollback: %v %v", usersTable, err)
	}
	if err := database.Migrate(f.cfg, 0); err != nil {
		t.Fatal(err)
	}
	var users, permissions int
	if err := f.queryRow(`SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := f.queryRow(`SELECT count(*) FROM permissions`).Scan(&permissions); err != nil {
		t.Fatal(err)
	}
	if users != 1 || permissions != 17 {
		t.Fatal(fmt.Sprintf("migration seeded users=%d permissions=%d", users, permissions))
	}
	pair := f.login(t, "admin", "Admin123!")
	wantStatus(t, f.request("GET", "/users", pair.AccessToken, nil), 200)
}

func TestSessionExpiryAndStaleCredentialIntegration(t *testing.T) {
	testutil.ForEachDatabase(t, testSessionExpiryAndStaleCredentialIntegration)
}
func testSessionExpiryAndStaleCredentialIntegration(t *testing.T, cfg database.Config) {
	f := setupFoundation(t, cfg)
	pair := f.login(t, "admin", " admin password ")
	if _, err := f.exec(`UPDATE refresh_sessions SET expires_at=? WHERE user_id=?`, time.Now().UTC().Add(-time.Second), f.adminID); err != nil {
		t.Fatal(err)
	}
	wantStatus(t, f.request("GET", "/auth/me", pair.AccessToken, nil), 401)
	wantStatus(t, f.request("POST", "/auth/refresh-token", "", map[string]string{"refresh_token": pair.RefreshToken}), 401)
	users := f.repos.User
	user, err := users.FindByID(context.Background(), f.adminID)
	if err != nil {
		t.Fatal(err)
	}
	oldHash := user.Password
	newHash, err := bcrypt.GenerateFromPassword([]byte("new password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user.Password = string(newHash)
	if err := users.Update(context.Background(), user, true); err != nil {
		t.Fatal(err)
	}
	id, err := helper.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	err = f.repos.Auth.CreateSession(context.Background(), authuc.Session{ID: id, UserID: f.adminID, PasswordHash: oldHash, TokenHash: authuc.HashToken("new token"), ExpiresAt: time.Now().Add(time.Hour)})
	var appErr *apperrors.AppError
	if !stderrors.As(err, &appErr) || appErr.Code != apperrors.Unauthorized {
		t.Fatalf("stale credential session accepted: %v", err)
	}
	var count int
	if err := f.queryRow(`SELECT count(*) FROM refresh_sessions WHERE id=?`, id).Scan(&count); err != nil || count != 0 {
		t.Fatal("stale login created a session")
	}
	// An unrelated profile update must not restore a concurrently changed password.
	user.Password = oldHash
	user.FullName = "Updated Profile"
	if err := users.Update(context.Background(), user, false); err != nil {
		t.Fatal(err)
	}
	f.login(t, "admin", "new password")
}
func TestCORSIntegration(t *testing.T) { testutil.ForEachDatabase(t, testCORSIntegration) }
func testCORSIntegration(t *testing.T, cfg database.Config) {
	f := setupFoundation(t, cfg)
	for _, tc := range []struct {
		origin string
		want   int
	}{{"https://client.example.test", 204}, {"https://unknown.example.test", 403}} {
		r := httptest.NewRequest("OPTIONS", "/auth/login", nil)
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Access-Control-Request-Method", "POST")
		r.Header.Set("Access-Control-Request-Headers", "content-type,authorization")
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		wantStatus(t, w, tc.want)
		if tc.want == 204 && w.Header().Get("Access-Control-Allow-Origin") != tc.origin {
			t.Fatal("allowed CORS origin missing")
		}
	}
}

func (f *foundationFixture) queryRow(query string, args ...any) *sql.Row {
	return f.db.QueryRow(testutil.Bind(f.cfg.Driver, query), args...)
}
func (f *foundationFixture) exec(query string, args ...any) (sql.Result, error) {
	return f.db.Exec(testutil.Bind(f.cfg.Driver, query), args...)
}
