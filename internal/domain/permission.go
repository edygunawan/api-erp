package domain

import "time"

type Permission struct {
	ID string

	Code        string
	Name        string
	Description *string
	Module      string

	CreatedAt time.Time
	UpdatedAt time.Time

	CreatedBy *string
	UpdatedBy *string
}
