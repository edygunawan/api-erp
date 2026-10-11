package setting

import (
	"github.com/edygunawan/go-rest-api/internal/domain"
	groupuc "github.com/edygunawan/go-rest-api/internal/usecase/settinggroup"
	"time"
)

type CreateSettingRequest struct {
	GroupID      string  `json:"group_id"`
	SettingKey   string  `json:"setting_key"`
	SettingValue *string `json:"setting_value"`
	DataType     string  `json:"data_type"`
	Description  *string `json:"description"`
}
type UpdateSettingRequest = CreateSettingRequest
type SettingResponse struct {
	ID           string                       `json:"id"`
	GroupID      string                       `json:"group_id"`
	Group        groupuc.SettingGroupResponse `json:"group"`
	SettingKey   string                       `json:"setting_key"`
	SettingValue *string                      `json:"setting_value"`
	DataType     string                       `json:"data_type"`
	Description  *string                      `json:"description"`
	CreatedAt    time.Time                    `json:"created_at"`
	UpdatedAt    time.Time                    `json:"updated_at"`
}

// ApplicationSettingsRequest is a complete replacement of the application form.
type ApplicationSettingsRequest struct {
	ApplicationName string `json:"application_name"`
	Description     string `json:"description"`
	Timezone        string `json:"timezone"`
	DateFormat      string `json:"date_format"`
}
type ApplicationSettingsResponse = ApplicationSettingsRequest

func settingResponse(item domain.Setting) SettingResponse {
	return SettingResponse{
		ID: item.ID, GroupID: item.GroupID, Group: groupuc.SettingGroupResponse(item.Group),
		SettingKey: item.SettingKey, SettingValue: item.SettingValue, DataType: item.DataType,
		Description: item.Description, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}
