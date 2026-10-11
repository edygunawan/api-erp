package role

import (
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/golang-pkgs/errors"
)

func validateFields(code, name string, description *string, permissions []string) error {
	if !helper.MaxLength(code, 100) || !helper.MaxLength(name, 150) || !helper.OptionalMaxLength(description, 500) || !helper.ValidUUIDs(permissions) {
		return errors.New(errors.Invalid, "invalid role fields or permission IDs")
	}
	return nil
}
