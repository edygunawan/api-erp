package database

import "context"

// Target identifies the database account to authenticate. Providers can use it
// to sign a short-lived token for the exact endpoint and user (e.g. AWS RDS IAM).
type Target struct {
	Driver string
	Host   string
	Port   string
	Name   string
	User   string
}

type Credentials struct {
	Password string
	// RDS MySQL tokens use mysql_clear_password; the connector requires verified
	// TLS before enabling that plugin. Static passwords never enable it.
	AllowCleartext bool
}

// CredentialsProvider is called for every new physical connection, not every
// query. Implementations must support concurrent calls and context cancellation.
// Providers own credential refresh/caching; tokens never belong in a startup DSN.
type CredentialsProvider interface {
	Credentials(context.Context, Target) (Credentials, error)
}

type CredentialsFunc func(context.Context, Target) (Credentials, error)

func (f CredentialsFunc) Credentials(ctx context.Context, target Target) (Credentials, error) {
	return f(ctx, target)
}

type passwordProvider struct{ password string }

func (p passwordProvider) Credentials(context.Context, Target) (Credentials, error) {
	return Credentials{Password: p.password}, nil
}

type options struct {
	provider   CredentialsProvider
	migrations bool
}
type Option func(*options)

func WithCredentialsProvider(provider CredentialsProvider) Option {
	return func(o *options) { o.provider = provider }
}
