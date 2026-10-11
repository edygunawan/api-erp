package user

import (
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/golang-pkgs/errors"
)

func validateFields(name, username string, email, phone *string, roles []string) error {
	if !helper.MaxLength(name, 150) || !helper.ValidUsername(username) || !helper.ValidEmail(email) || !helper.OptionalMaxLength(phone, 30) || !helper.ValidUUIDs(roles) {
		return errors.New(errors.Invalid, "invalid user fields or role IDs")
	}
	return nil
}
