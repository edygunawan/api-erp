package permission

import (
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/golang-pkgs/errors"
)

func validateFields(code, name, module string, description *string) error {
	if !helper.MaxLength(code, 150) || !helper.MaxLength(name, 150) || !helper.MaxLength(module, 100) || !helper.OptionalMaxLength(description, 500) {
		return errors.New(errors.Invalid, "invalid permission fields")
	}
	return nil
}
