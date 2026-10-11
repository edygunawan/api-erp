package user

import (
	"context"
	"testing"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"golang.org/x/crypto/bcrypt"
)

const roleID = "10000000-0000-4000-8000-000000000001"

type memoryUserRepository struct {
	user            *domain.User
	passwordChanged bool
}

func (r *memoryUserRepository) Create(_ context.Context, u *domain.User) error {
	r.user = u
	return nil
}
func (r *memoryUserRepository) Update(_ context.Context, u *domain.User, changed bool) error {
	r.user = u
	r.passwordChanged = changed
	return nil
}
func (r *memoryUserRepository) FindByID(context.Context, string) (*domain.User, error) {
	if r.user == nil {
		return nil, nil
	}
	copy := *r.user
	return &copy, nil
}
func (r *memoryUserRepository) List(context.Context, query.Options) ([]domain.User, int64, error) {
	return []domain.User{}, 0, nil
}
func TestUserPasswordsAssignmentsAndStatus(t *testing.T) {
	repo := &memoryUserRepository{}
	uc := New(repo)
	ctx := context.Background()
	response, err := uc.Create(ctx, CreateUserRequest{FullName: " User ", Username: " user ", Password: " secret ", Roles: []string{roleID, roleID, " "}})
	if err != nil {
		t.Fatal(err)
	}
	if response.FullName != "User" || len(response.Roles) != 1 || bcrypt.CompareHashAndPassword([]byte(repo.user.Password), []byte(" secret ")) != nil {
		t.Fatal("creation changed password or duplicated assignments")
	}
	if bcrypt.CompareHashAndPassword([]byte(repo.user.Password), []byte("secret")) == nil {
		t.Fatal("password whitespace was trimmed")
	}
	repo.user.Status = domain.UserStatusInactive
	oldHash := repo.user.Password
	_, err = uc.Update(ctx, response.ID, UpdateUserRequest{FullName: "User", Username: "user", Roles: []string{roleID}})
	if err != nil {
		t.Fatal(err)
	}
	if repo.user.Status != domain.UserStatusInactive || repo.passwordChanged || repo.user.Password != oldHash {
		t.Fatal("omitted status/password was overwritten")
	}
	password := " next secret "
	_, err = uc.Update(ctx, response.ID, UpdateUserRequest{FullName: "User", Username: "user", Password: &password, Roles: []string{roleID}})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.passwordChanged || bcrypt.CompareHashAndPassword([]byte(repo.user.Password), []byte(password)) != nil {
		t.Fatal("password update missing")
	}
}
func TestMissingAndInvalidUser(t *testing.T) {
	uc := New(&memoryUserRepository{})
	ctx := context.Background()
	if _, err := uc.GetByID(ctx, roleID); err == nil {
		t.Fatal("missing user accepted")
	}
	if _, err := uc.GetByID(ctx, "bad-id"); err == nil {
		t.Fatal("invalid ID accepted")
	}
	if _, err := uc.Create(ctx, CreateUserRequest{FullName: "User", Username: "user", Password: "secret", Roles: []string{"bad-id"}}); err == nil {
		t.Fatal("invalid role ID accepted")
	}
	if _, err := uc.Create(ctx, CreateUserRequest{FullName: "User", Username: "user@email", Password: "secret"}); err == nil {
		t.Fatal("ambiguous username accepted")
	}
}
