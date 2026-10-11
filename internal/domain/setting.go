package domain

import "time"

type Setting struct {
	ID           string
	GroupID      string
	Group        SettingGroup
	SettingKey   string
	SettingValue *string
	DataType     string
	Description  *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ApplicationSettings struct {
	ApplicationName string
	Description     string
	Timezone        string
	DateFormat      string
}
