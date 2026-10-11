package factory_test

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/edygunawan/go-rest-api/internal/database"
	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/go-rest-api/internal/repository/factory"
	"github.com/edygunawan/go-rest-api/internal/testutil"
	authuc "github.com/edygunawan/go-rest-api/internal/usecase/auth"
	apperrors "github.com/edygunawan/golang-pkgs/errors"
)

const adminRole = "00000000-0000-4000-8000-000000000001"
const coreRead = "00000000-0000-4000-8001-000000000001"
const missingID = "ffffffff-ffff-4fff-8fff-ffffffffffff"

func id(t *testing.T) string {
	t.Helper()
	value, err := helper.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	return value
}
func code(t *testing.T, err error, want apperrors.Code) {
	t.Helper()
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != want {
		t.Fatalf("error=%v want code=%d", err, want)
	}
}
func fixture(t *testing.T, cfg database.Config) (*sql.DB, database.Config, factory.Repositories) {
	t.Helper()
	db, cfg := testutil.Isolate(t, cfg)
	// The empty-list contract uses a clean user table after checking seeds separately.
	if _, err := db.Exec(`DELETE FROM users WHERE id='00000000-0000-4000-8002-000000000001'`); err != nil {
		t.Fatal(err)
	}
	repos, err := factory.New(cfg.Driver, db)
	if err != nil {
		t.Fatal(err)
	}
	return db, cfg, repos
}
func TestRepositoryContracts(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		db, cfg, repos := fixture(t, cfg)
		ctx := context.Background()
		_, err := repos.User.FindByID(ctx, missingID)
		code(t, err, apperrors.NotFound)
		_, err = repos.Role.FindByID(ctx, missingID)
		code(t, err, apperrors.NotFound)
		_, err = repos.Permission.FindByID(ctx, missingID)
		code(t, err, apperrors.NotFound)
		_, err = repos.Auth.FindByLogin(ctx, "missing")
		code(t, err, apperrors.NotFound)
		code(t, repos.Role.Delete(ctx, missingID), apperrors.NotFound)
		code(t, repos.Permission.Delete(ctx, missingID), apperrors.NotFound)
		users, _, err := repos.User.List(ctx, query.Options{})
		if err != nil || users == nil || len(users) != 0 {
			t.Fatal("empty user list must be []")
		}
		// CRUD, timestamps, nullable fields and case-insensitive uniqueness.
		permission := &domain.Permission{ID: id(t), Code: "CONTRACT.READ", Name: "Contract read", Module: "CONTRACT"}
		if err := repos.Permission.Create(ctx, permission); err != nil {
			t.Fatal(err)
		}
		role := &domain.Role{ID: id(t), Code: "CONTRACT_READER", Name: "Reader", Permissions: []string{coreRead, permission.ID}}
		if err := repos.Role.Create(ctx, role); err != nil {
			t.Fatal(err)
		}
		email := "contract@example.test"
		user := &domain.User{ID: id(t), FullName: "Contract User", Username: "Contract", Email: &email, Password: "verified-hash-1", Status: domain.UserStatusActive, Roles: []string{role.ID}}
		if err := repos.User.Create(ctx, user); err != nil {
			t.Fatal(err)
		}
		actual, err := repos.User.FindByID(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if actual.Email == nil || *actual.Email != email || actual.Phone != nil || len(actual.Roles) != 1 || actual.Roles[0] != role.ID || actual.CreatedAt.IsZero() || actual.UpdatedAt.IsZero() {
			t.Fatalf("user round trip: %+v", actual)
		}
		if age := time.Since(actual.CreatedAt); age < -5*time.Second || age > 5*time.Second {
			t.Fatal("timestamp timezone differs between drivers")
		}
		login, err := repos.Auth.FindByLogin(ctx, "CONTRACT@EXAMPLE.TEST")
		if err != nil || login.ID != user.ID || !slices.Equal(login.Roles, []string{role.Code}) {
			t.Fatal("auth must return current role codes")
		}
		duplicate := *user
		duplicate.ID = id(t)
		duplicate.Username = "contract"
		code(t, repos.User.Create(ctx, &duplicate), apperrors.Conflict)
		duplicate.ID = id(t)
		duplicate.Username = "another"
		upper := "CONTRACT@EXAMPLE.TEST"
		duplicate.Email = &upper
		code(t, repos.User.Create(ctx, &duplicate), apperrors.Conflict)
		duplicatePermission := *permission
		duplicatePermission.ID = id(t)
		code(t, repos.Permission.Create(ctx, &duplicatePermission), apperrors.Conflict)
		duplicateRole := *role
		duplicateRole.ID = id(t)
		code(t, repos.Role.Create(ctx, &duplicateRole), apperrors.Conflict)
		populatedUsers, _, err := repos.User.List(ctx, query.Options{})
		if err != nil || len(populatedUsers) != 1 || !slices.Equal(populatedUsers[0].Roles, actual.Roles) {
			t.Fatal("user list relationship contract failed")
		}
		listedRoles, _, err := repos.Role.List(ctx, query.Options{})
		if err != nil {
			t.Fatal(err)
		}
		foundRole := false
		for _, item := range listedRoles {
			if item.ID == role.ID {
				foundRole = true
				if len(item.Permissions) != 2 {
					t.Fatal("role list relationships missing")
				}
			}
		}
		if !foundRole {
			t.Fatal("role absent from list")
		}
		listedPermissions, _, err := repos.Permission.List(ctx, query.Options{})
		if err != nil || len(listedPermissions) != 18 {
			t.Fatal("permission list contract failed")
		}
		// Failed replacement must roll back both the entity and old relationships.
		broken := *actual
		broken.FullName = "Broken"
		broken.Password = "should-not-persist"
		broken.Roles = []string{missingID}
		code(t, repos.User.Update(ctx, &broken, true), apperrors.Invalid)
		after, err := repos.User.FindByID(ctx, user.ID)
		if err != nil || after.FullName != actual.FullName || after.Password != actual.Password || !slices.Equal(after.Roles, actual.Roles) {
			t.Fatal("user rollback contract failed")
		}
		originalRole, err := repos.Role.FindByID(ctx, role.ID)
		if err != nil {
			t.Fatal(err)
		}
		brokenRole := *originalRole
		brokenRole.Code = "BROKEN"
		brokenRole.Permissions = []string{missingID}
		code(t, repos.Role.Update(ctx, &brokenRole), apperrors.Invalid)
		afterRole, err := repos.Role.FindByID(ctx, role.ID)
		if err != nil || afterRole.Code != originalRole.Code || !slices.Equal(afterRole.Permissions, originalRole.Permissions) {
			t.Fatal("role rollback contract failed")
		}
		invalid := *user
		invalid.ID = id(t)
		invalid.Username = "rollback"
		invalid.Email = nil
		invalid.Roles = []string{missingID}
		code(t, repos.User.Create(ctx, &invalid), apperrors.Invalid)
		_, err = repos.User.FindByID(ctx, invalid.ID)
		code(t, err, apperrors.NotFound)
		invalidRole := *role
		invalidRole.ID = id(t)
		invalidRole.Code = "ROLLBACK"
		invalidRole.Permissions = []string{missingID}
		code(t, repos.Role.Create(ctx, &invalidRole), apperrors.Invalid)
		_, err = repos.Role.FindByID(ctx, invalidRole.ID)
		code(t, err, apperrors.NotFound)
		permissions, err := repos.Permission.GetPermissionsByUserID(ctx, user.ID)
		if err != nil || !slices.Contains(permissions, permission.Code) || !slices.Contains(permissions, "USER.READ") {
			t.Fatal("permission join failed")
		}
		// Unchanged writes and clearing relationships share one result contract.
		if err := repos.Permission.Update(ctx, permission); err != nil {
			t.Fatalf("unchanged update: %v", err)
		}
		if err := repos.Role.Update(ctx, originalRole); err != nil {
			t.Fatal(err)
		}
		if err := repos.User.Update(ctx, actual, false); err != nil {
			t.Fatal(err)
		}
		permission.Name = "Renamed"
		if err := repos.Permission.Update(ctx, permission); err != nil {
			t.Fatal(err)
		}
		savedPermission, err := repos.Permission.FindByID(ctx, permission.ID)
		if err != nil || savedPermission.Name != "Renamed" {
			t.Fatal("permission update failed")
		}
		actual.Roles = []string{}
		if err := repos.User.Update(ctx, actual, false); err != nil {
			t.Fatal(err)
		}
		permissions, err = repos.Permission.GetPermissionsByUserID(ctx, user.ID)
		if err != nil || permissions == nil || len(permissions) != 0 {
			t.Fatal("no-role permissions must be []")
		}
		actual.Roles = []string{role.ID}
		if err := repos.User.Update(ctx, actual, false); err != nil {
			t.Fatal(err)
		}
		if err := repos.Permission.Delete(ctx, permission.ID); err != nil {
			t.Fatal(err)
		}
		afterRole, err = repos.Role.FindByID(ctx, role.ID)
		if err != nil || !slices.Equal(afterRole.Permissions, []string{coreRead}) {
			t.Fatal("permission deletion did not cascade")
		}
		if err := repos.Role.Delete(ctx, role.ID); err != nil {
			t.Fatal(err)
		}
		after, err = repos.User.FindByID(ctx, user.ID)
		if err != nil || after.Roles == nil || len(after.Roles) != 0 {
			t.Fatal("role deletion did not cascade")
		}
		actual.Roles = []string{}
		// Audit references survive a subsequent read; foreign keys reject invalid IDs.
		actual.CreatedBy = &user.ID
		actual.UpdatedBy = &user.ID
		actual.FullName = "Updated"
		if err := repos.User.Update(ctx, actual, false); err != nil {
			t.Fatal(err)
		}
		after, err = repos.User.FindByID(ctx, user.ID)
		if err != nil || after.UpdatedBy == nil || *after.UpdatedBy != user.ID {
			t.Fatal("audit actor missing")
		}
		// Missing updates must never report success on either SQL dialect.
		missingUser := *actual
		missingUser.ID = missingID
		code(t, repos.User.Update(ctx, &missingUser, false), apperrors.NotFound)
		missingRole := *role
		missingRole.ID = missingID
		code(t, repos.Role.Update(ctx, &missingRole), apperrors.NotFound)
		permission.ID = missingID
		code(t, repos.Permission.Update(ctx, permission), apperrors.NotFound)
		var count int
		if err := db.QueryRow(testutil.Bind(cfg.Driver, `SELECT COUNT(*) FROM users WHERE username=?`), "rollback").Scan(&count); err != nil || count != 0 {
			t.Fatal("failed create persisted a user")
		}
	})
}
func TestRefreshSessionContracts(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		db, cfg, repos := fixture(t, cfg)
		ctx := context.Background()
		user := &domain.User{ID: id(t), FullName: "Session User", Username: "session", Password: "hash-v1", Status: domain.UserStatusActive, Roles: []string{adminRole}}
		if err := repos.User.Create(ctx, user); err != nil {
			t.Fatal(err)
		}
		session := authuc.Session{ID: id(t), UserID: user.ID, PasswordHash: user.Password, TokenHash: authuc.HashToken("first-token"), ExpiresAt: time.Now().UTC().Add(time.Hour)}
		if err := repos.Auth.CreateSession(ctx, session); err != nil {
			t.Fatal(err)
		}
		active, err := repos.Auth.SessionActive(ctx, session.ID, user.ID)
		if err != nil || !active {
			t.Fatal("new session inactive")
		}
		wrong := session
		wrong.ID = id(t)
		wrong.PasswordHash = "stale-hash"
		code(t, repos.Auth.CreateSession(ctx, wrong), apperrors.Unauthorized)
		var count int
		if err := db.QueryRow(testutil.Bind(cfg.Driver, `SELECT COUNT(*) FROM refresh_sessions WHERE id=?`), wrong.ID).Scan(&count); err != nil || count != 0 {
			t.Fatal("stale password created session")
		}
		rotated := session
		rotated.TokenHash = authuc.HashToken("second-token")
		ok, err := repos.Auth.RotateSession(ctx, "wrong-hash", rotated)
		if err != nil || ok {
			t.Fatal("invalid refresh accepted")
		}
		const attempts = 8
		outcomes := make(chan bool, attempts)
		failures := make(chan error, attempts)
		var wg sync.WaitGroup
		for i := 0; i < attempts; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ok, err := repos.Auth.RotateSession(ctx, session.TokenHash, rotated)
				outcomes <- ok
				failures <- err
			}()
		}
		wg.Wait()
		close(outcomes)
		close(failures)
		wins := 0
		for ok := range outcomes {
			if ok {
				wins++
			}
		}
		for err := range failures {
			if err != nil {
				t.Fatal(err)
			}
		}
		if wins != 1 {
			t.Fatalf("refresh winners=%d", wins)
		}
		ok, err = repos.Auth.RotateSession(ctx, session.TokenHash, rotated)
		if err != nil || ok {
			t.Fatal("refresh replay accepted")
		}
		if err := repos.Auth.RevokeSession(ctx, session.ID, user.ID); err != nil {
			t.Fatal(err)
		}
		if err := repos.Auth.RevokeSession(ctx, session.ID, user.ID); err != nil {
			t.Fatal("logout not idempotent")
		}
		active, err = repos.Auth.SessionActive(ctx, session.ID, user.ID)
		if err != nil || active {
			t.Fatal("revoked session active")
		}
		// Invalidation survives writes outside the repository through the DB trigger.
		session.ID = id(t)
		session.TokenHash = authuc.HashToken("third-token")
		if err := repos.Auth.CreateSession(ctx, session); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(testutil.Bind(cfg.Driver, `UPDATE users SET password=? WHERE id=?`), "hash-v2", user.ID); err != nil {
			t.Fatal(err)
		}
		active, err = repos.Auth.SessionActive(ctx, session.ID, user.ID)
		if err != nil || active {
			t.Fatal("password trigger did not invalidate session")
		}
		rotated = session
		rotated.TokenHash = authuc.HashToken("rotated-after-password-change")
		ok, err = repos.Auth.RotateSession(ctx, session.TokenHash, rotated)
		if err != nil || ok {
			t.Fatal("password-version mismatch accepted")
		}
		user.Password = "hash-v2"
		session.PasswordHash = user.Password
		session.ID = id(t)
		session.TokenHash = authuc.HashToken("fourth-token")
		if err := repos.Auth.CreateSession(ctx, session); err != nil {
			t.Fatal(err)
		}
		user.Status = domain.UserStatusInactive
		if err := repos.User.Update(ctx, user, false); err != nil {
			t.Fatal(err)
		}
		active, err = repos.Auth.SessionActive(ctx, session.ID, user.ID)
		if err != nil || active {
			t.Fatal("inactive session accepted")
		}
		user.Status = domain.UserStatusActive
		if err := repos.User.Update(ctx, user, false); err != nil {
			t.Fatal(err)
		}
		active, err = repos.Auth.SessionActive(ctx, session.ID, user.ID)
		if err != nil || active {
			t.Fatal("reactivation restored revoked session")
		}
		session.ID = id(t)
		session.TokenHash = authuc.HashToken("expired-token")
		session.ExpiresAt = time.Now().UTC().Add(-time.Second)
		if err := repos.Auth.CreateSession(ctx, session); err != nil {
			t.Fatal(err)
		}
		active, err = repos.Auth.SessionActive(ctx, session.ID, user.ID)
		if err != nil || active {
			t.Fatal("expired session active")
		}
	})
}
func TestConcurrentBootstrapContract(t *testing.T) {
	testutil.ForEachDatabase(t, func(t *testing.T, cfg database.Config) {
		_, _, repos := fixture(t, cfg)
		ctx := context.Background()
		const attempts = 6
		ids := make(chan string, attempts)
		errs := make(chan error, attempts)
		var wg sync.WaitGroup
		for i := 0; i < attempts; i++ {
			userID := id(t)
			wg.Add(1)
			go func() {
				defer wg.Done()
				user := &domain.User{ID: userID, FullName: "Administrator", Username: "bootstrap", Password: "original-hash", Status: domain.UserStatusActive}
				value, err := repos.User.BootstrapAdmin(ctx, user)
				ids <- value
				errs <- err
			}()
		}
		wg.Wait()
		close(ids)
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		var first string
		for value := range ids {
			if first == "" {
				first = value
			}
			if value != first {
				t.Fatal("bootstrap produced multiple users")
			}
		}
		user, err := repos.User.FindByID(ctx, first)
		if err != nil || !slices.Equal(user.Roles, []string{adminRole}) {
			t.Fatal("bootstrap role assignment failed")
		}
		_, err = repos.User.BootstrapAdmin(ctx, &domain.User{ID: id(t), FullName: "Replacement", Username: "bootstrap", Password: "replacement-hash", Status: domain.UserStatusActive})
		if err != nil {
			t.Fatal(err)
		}
		user, err = repos.User.FindByID(ctx, first)
		if err != nil || user.Password != "original-hash" || user.FullName != "Administrator" {
			t.Fatal("repeat bootstrap overwrote credentials/profile")
		}
	})
}
