package permission

import (
	"context"
	"strings"

	"github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/golang-pkgs/errors"
)

type usecase struct {
	permissionRepo Repository
}

func New(permissionRepo Repository) Usecase {
	return &usecase{
		permissionRepo: permissionRepo,
	}
}

func (u *usecase) Create(
	ctx context.Context,
	req CreatePermissionRequest,
) (*PermissionResponse, error) {
	req.Code = normalizeCode(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	req.Module = normalizeCode(req.Module)

	if req.Code == "" {
		return nil, errors.New(errors.Invalid, "permission code required")
	}
	if req.Name == "" {
		return nil, errors.New(errors.Invalid, "permission name required")
	}
	if req.Module == "" {
		return nil, errors.New(errors.Invalid, "permission module required")
	}

	if err := validateFields(req.Code, req.Name, req.Module, req.Description); err != nil {
		return nil, err
	}
	permissionID, err := helper.NewUUID()
	if err != nil {
		return nil, err
	}

	currentUserID := optionalCurrentUserID(ctx)
	newPermission := &domain.Permission{
		ID:          permissionID,
		Code:        req.Code,
		Name:        req.Name,
		Description: cleanStringPtr(req.Description),
		Module:      req.Module,
		CreatedBy:   currentUserID,
		UpdatedBy:   currentUserID,
	}

	if err := u.permissionRepo.Create(ctx, newPermission); err != nil {
		return nil, err
	}

	createdPermission, err := u.permissionRepo.FindByID(ctx, newPermission.ID)
	if err != nil {
		return nil, err
	}
	if createdPermission == nil {
		return nil, errors.New(errors.NotFound, "created permission not found")
	}

	res := newPermissionResponse(*createdPermission)
	return &res, nil
}

func (u *usecase) Update(
	ctx context.Context,
	id string,
	req UpdatePermissionRequest,
) (*PermissionResponse, error) {
	id = strings.TrimSpace(id)
	req.Code = normalizeCode(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	req.Module = normalizeCode(req.Module)

	if !helper.ValidUUID(id) {
		return nil, errors.New(errors.Invalid, "permission id required")
	}
	if req.Code == "" {
		return nil, errors.New(errors.Invalid, "permission code required")
	}
	if req.Name == "" {
		return nil, errors.New(errors.Invalid, "permission name required")
	}
	if req.Module == "" {
		return nil, errors.New(errors.Invalid, "permission module required")
	}

	if err := validateFields(req.Code, req.Name, req.Module, req.Description); err != nil {
		return nil, err
	}
	existingPermission, err := u.permissionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existingPermission == nil {
		return nil, errors.New(errors.NotFound, "permission not found")
	}

	if auth.IsCorePermission(existingPermission.Code) && (existingPermission.Code != req.Code || existingPermission.Module != req.Module) {
		return nil, errors.New(errors.Forbidden, "core permission code and module cannot be changed")
	}
	existingPermission.Code = req.Code
	existingPermission.Name = req.Name
	existingPermission.Description = cleanStringPtr(req.Description)
	existingPermission.Module = req.Module
	existingPermission.UpdatedBy = optionalCurrentUserID(ctx)

	if err := u.permissionRepo.Update(ctx, existingPermission); err != nil {
		return nil, err
	}

	updatedPermission, err := u.permissionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if updatedPermission == nil {
		return nil, errors.New(errors.NotFound, "updated permission not found")
	}

	res := newPermissionResponse(*updatedPermission)
	return &res, nil
}

func (u *usecase) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if !helper.ValidUUID(id) {
		return errors.New(errors.Invalid, "permission id required")
	}

	existing, err := u.permissionRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New(errors.NotFound, "permission not found")
	}
	if auth.IsCorePermission(existing.Code) {
		return errors.New(errors.Forbidden, "core permission cannot be deleted")
	}
	return u.permissionRepo.Delete(ctx, id)
}

func (u *usecase) GetByID(ctx context.Context, id string) (*PermissionResponse, error) {
	id = strings.TrimSpace(id)
	if !helper.ValidUUID(id) {
		return nil, errors.New(errors.Invalid, "permission id required")
	}

	permission, err := u.permissionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, errors.New(errors.NotFound, "permission not found")
	}

	res := newPermissionResponse(*permission)
	return &res, nil
}

func (u *usecase) List(ctx context.Context, options query.Options) (query.Page[PermissionResponse], error) {
	options, err := options.Normalize()
	if err != nil {
		return query.Page[PermissionResponse]{}, errors.Wrap(errors.Invalid, err.Error(), err)
	}
	for i, module := range options.Filters["module"] {
		if len([]rune(module)) > 100 {
			return query.Page[PermissionResponse]{}, errors.New(errors.Invalid, "module filter must contain at most 100 characters")
		}
		options.Filters["module"][i] = strings.ToUpper(module)
	}
	items, total, err := u.permissionRepo.List(ctx, options)
	if err != nil {
		return query.Page[PermissionResponse]{}, err
	}
	res := make([]PermissionResponse, len(items))
	for i, item := range items {
		res[i] = newPermissionResponse(item)
	}
	return query.NewPage(res, options, total), nil
}

func newPermissionResponse(permission domain.Permission) PermissionResponse {
	return PermissionResponse{
		ID:          permission.ID,
		Code:        permission.Code,
		Name:        permission.Name,
		Description: permission.Description,
		Module:      permission.Module,
		CreatedAt:   permission.CreatedAt,
		UpdatedAt:   permission.UpdatedAt,
		CreatedBy:   permission.CreatedBy,
		UpdatedBy:   permission.UpdatedBy,
	}
}

func optionalCurrentUserID(ctx context.Context) *string {
	userID := auth.CurrentUserID(ctx)
	if userID == "" {
		return nil
	}

	return &userID
}

func normalizeCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
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
