package domain

import "time"

type SettingGroup struct {
	ID        string
	Code      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
