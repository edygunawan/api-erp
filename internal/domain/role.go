package domain

import "time"

type Role struct {
	ID string

	Code        string
	Name        string
	Description *string
	IsSystem    bool
	Permissions []string

	CreatedAt time.Time
	UpdatedAt time.Time

	CreatedBy *string
	UpdatedBy *string
}
