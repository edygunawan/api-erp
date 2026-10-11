package settinggroup

import (
	"strings"

	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/golang-pkgs/errors"
)

func normalizeRequest(req *CreateSettingGroupRequest) error {
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	req.Name = strings.TrimSpace(req.Name)
	if req.Code == "" || !helper.MaxLength(req.Code, 50) {
		return errors.New(errors.Invalid, "group code is required and must contain at most 50 characters")
	}
	if req.Name == "" || !helper.MaxLength(req.Name, 100) {
		return errors.New(errors.Invalid, "group name is required and must contain at most 100 characters")
	}
	return nil
}
func normalizeFilters(options *query.Options) error {
	for i, value := range options.Filters["code"] {
		value = strings.ToUpper(value)
		if !helper.MaxLength(value, 50) {
			return errors.New(errors.Invalid, "code filter must contain at most 50 characters")
		}
		options.Filters["code"][i] = value
	}
	return nil
}
