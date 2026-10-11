package auth

const (
	ApplicationSettingRead   = "APPLICATIONSETTING.READ"
	ApplicationSettingUpdate = "APPLICATIONSETTING.UPDATE"
	SettingRead              = "SETTING.READ"
	SettingCreate            = "SETTING.CREATE"
	SettingUpdate            = "SETTING.UPDATE"
	SettingDelete            = "SETTING.DELETE"
	UserRead                 = "USER.READ"
	UserCreate               = "USER.CREATE"
	UserUpdate               = "USER.UPDATE"
	RoleRead                 = "ROLE.READ"
	RoleCreate               = "ROLE.CREATE"
	RoleUpdate               = "ROLE.UPDATE"
	RoleDelete               = "ROLE.DELETE"
	PermissionRead           = "PERMISSION.READ"
	PermissionCreate         = "PERMISSION.CREATE"
	PermissionUpdate         = "PERMISSION.UPDATE"
	PermissionDelete         = "PERMISSION.DELETE"
)

func IsCorePermission(code string) bool {
	switch code {
	case ApplicationSettingRead, ApplicationSettingUpdate, SettingRead, SettingCreate, SettingUpdate, SettingDelete, UserRead, UserCreate, UserUpdate, RoleRead, RoleCreate, RoleUpdate, RoleDelete, PermissionRead, PermissionCreate, PermissionUpdate, PermissionDelete:
		return true
	default:
		return false
	}
}
