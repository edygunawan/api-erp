package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edygunawan/go-rest-api/internal/auth"
	"github.com/edygunawan/golang-pkgs/errors"
)

type fakeAuthenticator struct{ err error }

func (f fakeAuthenticator) Authenticate(context.Context, string) (*auth.Identity, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &auth.Identity{UserID: "user"}, nil
}

type fakePermissions struct {
	allowed bool
	err     error
}

func (f fakePermissions) HasPermission(context.Context, string) (bool, error) {
	return f.allowed, f.err
}
func TestMiddlewareFailsClosed(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.GetIdentity(r.Context()); !ok {
			t.Fatal("identity missing")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	for _, tc := range []struct {
		name, header string
		authErr      error
		allowed      bool
		permErr      error
		want         int
	}{
		{name: "missing bearer", want: 401}, {name: "wrong scheme", header: "Basic value", want: 401},
		{name: "bad token", header: "Bearer value", authErr: errors.New(errors.Unauthorized, "invalid token"), want: 401},
		{name: "permission denied", header: "Bearer value", want: 403},
		{name: "database failure", header: "Bearer value", permErr: errors.New(errors.Internal, "private database detail"), want: 500},
		{name: "allowed", header: "bearer value", allowed: true, want: 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()
			AuthMiddleware(fakeAuthenticator{err: tc.authErr})(RequirePermission(fakePermissions{allowed: tc.allowed, err: tc.permErr}, "USER.READ")(next)).ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status=%d want %d: %s", w.Code, tc.want, w.Body)
			}
			if tc.want >= 400 && w.Header().Get("Content-Type") != "application/json" {
				t.Fatal("non-JSON failure")
			}
			if tc.want == 500 && w.Body.String() != `{"message":"internal server error"}`+"\n" {
				t.Fatal("internal details exposed")
			}
		})
	}
	w := httptest.NewRecorder()
	RequirePermission(fakePermissions{allowed: true}, "USER.READ")(next).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 401 {
		t.Fatal("permission middleware accepted missing identity")
	}
}
