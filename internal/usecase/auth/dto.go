package auth

type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email"`
	Password        string `json:"password"`
}
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}
type MeResponse struct {
	ID          string   `json:"id"`
	FullName    string   `json:"full_name"`
	Username    string   `json:"username"`
	Email       *string  `json:"email"`
	Phone       *string  `json:"phone"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}
