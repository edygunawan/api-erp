package auth

import "context"

type contextKey string

const IdentityKey contextKey = "identity"

type Identity struct {
	UserID    string
	SessionID string
	TokenID   string
	Roles     []string
}

func GetIdentity(ctx context.Context) (*Identity, bool) {
	v := ctx.Value(IdentityKey)
	identity, ok := v.(*Identity)
	return identity, ok && identity != nil
}

func CurrentUserID(ctx context.Context) string {
	identity, ok := GetIdentity(ctx)
	if !ok {
		return ""
	}
	return identity.UserID
}
