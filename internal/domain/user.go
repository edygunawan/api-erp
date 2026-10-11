package domain

import "time"

type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
)

type User struct {
	ID string

	FullName string
	Username string
	Password string

	Email *string
	Phone *string

	Status      UserStatus
	Roles       []string
	RoleDetails []UserRole

	LastLoginAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	CreatedBy *string
	UpdatedBy *string
}

type UserRole struct {
	ID   string
	Code string
	Name string
}
