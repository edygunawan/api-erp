package user

import (
	"time"

	"github.com/edygunawan/go-rest-api/internal/domain"
)

type CreateUserRequest struct {
	FullName string            `json:"full_name"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Email    *string           `json:"email"`
	Phone    *string           `json:"phone"`
	Status   domain.UserStatus `json:"status"`
	Roles    []string          `json:"roles"`
}

type UpdateUserRequest struct {
	FullName string            `json:"full_name"`
	Username string            `json:"username"`
	Password *string           `json:"password"`
	Email    *string           `json:"email"`
	Phone    *string           `json:"phone"`
	Status   domain.UserStatus `json:"status"`
	Roles    []string          `json:"roles"`
}

type UserRoleResponse struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type UserResponse struct {
	RoleDetails []UserRoleResponse `json:"role_details"`
	ID          string             `json:"id"`
	FullName    string             `json:"full_name"`
	Username    string             `json:"username"`
	Email       *string            `json:"email"`
	Phone       *string            `json:"phone"`
	Status      domain.UserStatus  `json:"status"`
	Roles       []string           `json:"roles"`
	LastLoginAt *time.Time         `json:"last_login_at"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	CreatedBy   *string            `json:"created_by"`
	UpdatedBy   *string            `json:"updated_by"`
}

func NewUserResponse(u domain.User) UserResponse {
	details := make([]UserRoleResponse, len(u.RoleDetails))
	for i, role := range u.RoleDetails {
		details[i] = UserRoleResponse{ID: role.ID, Code: role.Code, Name: role.Name}
	}
	return UserResponse{
		RoleDetails: details,
		ID:          u.ID,
		FullName:    u.FullName,
		Username:    u.Username,
		Email:       u.Email,
		Phone:       u.Phone,
		Status:      u.Status,
		Roles:       u.Roles,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		CreatedBy:   u.CreatedBy,
		UpdatedBy:   u.UpdatedBy,
	}
}
