package handler

type Handlers struct {
	Setting      *SettingHandler
	SettingGroup *SettingGroupHandler
	User         *UserHandler
	Auth         *AuthHandler
	Role         *RoleHandler
	Permission   *PermissionHandler
}
