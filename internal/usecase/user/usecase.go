package user

import (
	"context"
	"strings"

	"github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/golang-pkgs/errors"
	"golang.org/x/crypto/bcrypt"
)

type usecase struct {
	userRepo UserRepository
}

func New(
	userRepo UserRepository,
) Usecase {
	return &usecase{
		userRepo: userRepo,
	}
}

func (u *usecase) Create(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	req.FullName = strings.TrimSpace(req.FullName)
	req.Username = strings.TrimSpace(req.Username)

	if req.FullName == "" {
		return nil, errors.Wrap(errors.Invalid, "full name required", nil)
	}
	if req.Username == "" {
		return nil, errors.Wrap(errors.Invalid, "username required", nil)
	}
	if req.Password == "" || len(req.Password) > 72 {
		return nil, errors.Wrap(errors.Invalid, "password must contain 1 to 72 bytes", nil)
	}
	if req.Status == "" {
		req.Status = domain.UserStatusActive
	}
	if req.Status != domain.UserStatusActive && req.Status != domain.UserStatusInactive {
		return nil, errors.Wrap(errors.Invalid, "invalid user status", nil)
	}
	roleIDs := cleanStringSlice(req.Roles)
	if err := validateFields(req.FullName, req.Username, req.Email, req.Phone, roleIDs); err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	userID, err := helper.NewUUID()
	if err != nil {
		return nil, err
	}

	newUser := &domain.User{
		ID:        userID,
		FullName:  req.FullName,
		Username:  req.Username,
		Password:  string(passwordHash),
		Email:     cleanStringPtr(req.Email),
		Phone:     cleanStringPtr(req.Phone),
		Status:    req.Status,
		Roles:     roleIDs,
		CreatedBy: optionalCurrentUserID(ctx),
		UpdatedBy: optionalCurrentUserID(ctx),
	}

	if err := u.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	createdUser, err := u.userRepo.FindByID(ctx, newUser.ID)
	if err != nil {
		return nil, err
	}
	if createdUser == nil {
		return nil, errors.Wrap(errors.NotFound, "created user not found", nil)
	}

	res := NewUserResponse(*createdUser)
	return &res, nil
}

func (u *usecase) Update(
	ctx context.Context,
	id string,
	req UpdateUserRequest,
) (*UserResponse, error) {
	id = strings.TrimSpace(id)
	req.FullName = strings.TrimSpace(req.FullName)
	req.Username = strings.TrimSpace(req.Username)

	if !helper.ValidUUID(id) {
		return nil, errors.Wrap(errors.Invalid, "valid user id required", nil)
	}
	if req.FullName == "" {
		return nil, errors.Wrap(errors.Invalid, "full name required", nil)
	}
	if req.Username == "" {
		return nil, errors.Wrap(errors.Invalid, "username required", nil)
	}
	if req.Status != "" && req.Status != domain.UserStatusActive && req.Status != domain.UserStatusInactive {
		return nil, errors.Wrap(errors.Invalid, "invalid user status", nil)
	}

	existingUser, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existingUser == nil {
		return nil, errors.Wrap(errors.NotFound, "user not found", nil)
	}

	if req.Status == "" {
		req.Status = existingUser.Status
	}
	if req.Password != nil {
		password := *req.Password
		if password == "" || len(password) > 72 {
			return nil, errors.Wrap(errors.Invalid, "password must contain 1 to 72 bytes", nil)
		}

		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			return nil, err
		}
		existingUser.Password = string(passwordHash)
	}

	if err := validateFields(req.FullName, req.Username, req.Email, req.Phone, cleanStringSlice(req.Roles)); err != nil {
		return nil, err
	}
	existingUser.FullName = req.FullName
	existingUser.Username = req.Username
	existingUser.Email = cleanStringPtr(req.Email)
	existingUser.Phone = cleanStringPtr(req.Phone)
	existingUser.Status = req.Status
	existingUser.Roles = cleanStringSlice(req.Roles)
	existingUser.UpdatedBy = optionalCurrentUserID(ctx)

	if err := u.userRepo.Update(ctx, existingUser, req.Password != nil); err != nil {
		return nil, err
	}

	updatedUser, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if updatedUser == nil {
		return nil, errors.Wrap(errors.NotFound, "updated user not found", nil)
	}

	res := NewUserResponse(*updatedUser)
	return &res, nil
}

func (u *usecase) GetByID(ctx context.Context, id string) (*UserResponse, error) {
	if !helper.ValidUUID(id) {
		return nil, errors.Wrap(errors.Invalid, "valid user id required", nil)
	}
	user, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New(errors.NotFound, "user not found")
	}
	userDto := NewUserResponse(*user)

	return &userDto, nil
}

func (u *usecase) List(ctx context.Context, options query.Options) (query.Page[UserResponse], error) {
	options, err := options.Normalize()
	if err != nil {
		return query.Page[UserResponse]{}, errors.Wrap(errors.Invalid, err.Error(), err)
	}
	for i, status := range options.Filters["status"] {
		status = strings.ToUpper(status)
		if status != string(domain.UserStatusActive) && status != string(domain.UserStatusInactive) {
			return query.Page[UserResponse]{}, errors.New(errors.Invalid, "invalid user status filter")
		}
		options.Filters["status"][i] = status
	}
	items, total, err := u.userRepo.List(ctx, options)
	if err != nil {
		return query.Page[UserResponse]{}, err
	}
	res := make([]UserResponse, len(items))
	for i, item := range items {
		res[i] = NewUserResponse(item)
	}
	return query.NewPage(res, options, total), nil
}

func cleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}

	cleaned := strings.TrimSpace(*value)
	if cleaned == "" {
		return nil
	}

	return &cleaned
}

func optionalCurrentUserID(ctx context.Context) *string {
	userID := auth.CurrentUserID(ctx)
	if userID == "" {
		return nil
	}

	return &userID
}

func cleanStringSlice(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		cleaned := strings.TrimSpace(value)
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}

		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}

	return result
}
