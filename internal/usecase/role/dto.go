package role

import "time"

type CreateRoleRequest struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

type UpdateRoleRequest struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

type RoleResponse struct {
	IsSystem    bool      `json:"is_system"`
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   *string   `json:"created_by"`
	UpdatedBy   *string   `json:"updated_by"`
}
