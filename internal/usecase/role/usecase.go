package role

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
	roleRepo Repository
}

func New(roleRepo Repository) Usecase {
	return &usecase{
		roleRepo: roleRepo,
	}
}

func (u *usecase) Create(ctx context.Context, req CreateRoleRequest) (*RoleResponse, error) {
	req.Code = normalizeCode(req.Code)
	req.Name = strings.TrimSpace(req.Name)

	if req.Code == "" {
		return nil, errors.New(errors.Invalid, "role code required")
	}
	if req.Name == "" {
		return nil, errors.New(errors.Invalid, "role name required")
	}
	permissionIDs := cleanStringSlice(req.Permissions)
	if err := validateFields(req.Code, req.Name, req.Description, permissionIDs); err != nil {
		return nil, err
	}

	roleID, err := helper.NewUUID()
	if err != nil {
		return nil, err
	}

	currentUserID := optionalCurrentUserID(ctx)
	newRole := &domain.Role{
		ID:          roleID,
		Code:        req.Code,
		Name:        req.Name,
		Description: cleanStringPtr(req.Description),
		IsSystem:    false,
		Permissions: permissionIDs,
		CreatedBy:   currentUserID,
		UpdatedBy:   currentUserID,
	}

	if err := u.roleRepo.Create(ctx, newRole); err != nil {
		return nil, err
	}

	createdRole, err := u.roleRepo.FindByID(ctx, newRole.ID)
	if err != nil {
		return nil, err
	}
	if createdRole == nil {
		return nil, errors.New(errors.NotFound, "created role not found")
	}

	res := newRoleResponse(*createdRole)
	return &res, nil
}

func (u *usecase) Update(ctx context.Context, id string, req UpdateRoleRequest) (*RoleResponse, error) {
	id = strings.TrimSpace(id)
	req.Code = normalizeCode(req.Code)
	req.Name = strings.TrimSpace(req.Name)

	if !helper.ValidUUID(id) {
		return nil, errors.New(errors.Invalid, "role id required")
	}
	if req.Code == "" {
		return nil, errors.New(errors.Invalid, "role code required")
	}
	if req.Name == "" {
		return nil, errors.New(errors.Invalid, "role name required")
	}
	permissionIDs := cleanStringSlice(req.Permissions)
	if err := validateFields(req.Code, req.Name, req.Description, permissionIDs); err != nil {
		return nil, err
	}

	existingRole, err := u.roleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existingRole == nil {
		return nil, errors.New(errors.NotFound, "role not found")
	}

	if existingRole.IsSystem && existingRole.Code != req.Code {
		return nil, errors.New(errors.Forbidden, "system role code cannot be changed")
	}
	existingRole.Code = req.Code
	existingRole.Name = req.Name
	existingRole.Description = cleanStringPtr(req.Description)
	// System classification is controlled by migrations, never by clients.
	existingRole.Permissions = permissionIDs
	existingRole.UpdatedBy = optionalCurrentUserID(ctx)

	if err := u.roleRepo.Update(ctx, existingRole); err != nil {
		return nil, err
	}

	updatedRole, err := u.roleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if updatedRole == nil {
		return nil, errors.New(errors.NotFound, "updated role not found")
	}

	res := newRoleResponse(*updatedRole)
	return &res, nil
}

func (u *usecase) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if !helper.ValidUUID(id) {
		return errors.New(errors.Invalid, "role id required")
	}

	existing, err := u.roleRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New(errors.NotFound, "role not found")
	}
	if existing.IsSystem {
		return errors.New(errors.Forbidden, "system role cannot be deleted")
	}
	return u.roleRepo.Delete(ctx, id)
}

func (u *usecase) GetByID(ctx context.Context, id string) (*RoleResponse, error) {
	id = strings.TrimSpace(id)
	if !helper.ValidUUID(id) {
		return nil, errors.New(errors.Invalid, "role id required")
	}

	role, err := u.roleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New(errors.NotFound, "role not found")
	}

	res := newRoleResponse(*role)
	return &res, nil
}

func (u *usecase) List(ctx context.Context, options query.Options) (query.Page[RoleResponse], error) {
	options, err := options.Normalize()
	if err != nil {
		return query.Page[RoleResponse]{}, errors.Wrap(errors.Invalid, err.Error(), err)
	}

	items, total, err := u.roleRepo.List(ctx, options)
	if err != nil {
		return query.Page[RoleResponse]{}, err
	}
	res := make([]RoleResponse, len(items))
	for i, item := range items {
		res[i] = newRoleResponse(item)
	}
	return query.NewPage(res, options, total), nil
}

func newRoleResponse(role domain.Role) RoleResponse {
	return RoleResponse{
		ID:          role.ID,
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		Permissions: role.Permissions,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
		CreatedBy:   role.CreatedBy,
		UpdatedBy:   role.UpdatedBy,
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
