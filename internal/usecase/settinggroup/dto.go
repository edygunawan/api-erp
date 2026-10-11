package settinggroup

import "time"

type CreateSettingGroupRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
type UpdateSettingGroupRequest = CreateSettingGroupRequest
type SettingGroupResponse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
