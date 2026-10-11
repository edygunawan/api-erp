package bootstrap

import (
	"context"
	"strings"

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/golang-pkgs/errors"
	"golang.org/x/crypto/bcrypt"
)

type Request struct {
	FullName string
	Username string
	Password string
	Email    *string
}
type Repository interface {
	BootstrapAdmin(context.Context, *domain.User) (string, error)
}
type Usecase struct{ repo Repository }

func New(repo Repository) *Usecase { return &Usecase{repo: repo} }
func (u *Usecase) Run(ctx context.Context, req Request) (string, error) {
	req.FullName = strings.TrimSpace(req.FullName)
	req.Username = strings.TrimSpace(req.Username)
	if req.Email != nil {
		email := strings.TrimSpace(*req.Email)
		if email == "" {
			req.Email = nil
		} else {
			req.Email = &email
		}
	}
	if req.FullName == "" || !helper.MaxLength(req.FullName, 150) || !helper.ValidUsername(req.Username) || !helper.ValidEmail(req.Email) || len(req.Password) == 0 || len(req.Password) > 72 {
		return "", errors.New(errors.Invalid, "valid admin name, username, email and password (1 to 72 bytes) are required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	id, err := helper.NewUUID()
	if err != nil {
		return "", err
	}
	return u.repo.BootstrapAdmin(ctx, &domain.User{ID: id, FullName: req.FullName, Username: req.Username, Email: req.Email, Password: string(hash), Status: domain.UserStatusActive})
}
