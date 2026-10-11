package permission

import "time"

type CreatePermissionRequest struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Module      string  `json:"module"`
}

type UpdatePermissionRequest struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Module      string  `json:"module"`
}

type PermissionResponse struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Module      string    `json:"module"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   *string   `json:"created_by"`
	UpdatedBy   *string   `json:"updated_by"`
}
