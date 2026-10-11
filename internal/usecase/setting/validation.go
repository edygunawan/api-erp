package setting

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/go-rest-api/internal/helper/query"
	"github.com/edygunawan/golang-pkgs/errors"
)

func validDataType(value string) bool {
	switch value {
	case "STRING", "NUMBER", "BOOLEAN", "JSON":
		return true
	}
	return false
}
func normalizeRequest(req *CreateSettingRequest) error {
	req.GroupID = strings.TrimSpace(req.GroupID)
	req.SettingKey = strings.ToLower(strings.TrimSpace(req.SettingKey))
	req.DataType = strings.ToUpper(strings.TrimSpace(req.DataType))
	if req.DataType == "" {
		req.DataType = "STRING"
	}
	if !helper.ValidUUID(req.GroupID) {
		return errors.New(errors.Invalid, "group_id must be a UUID")
	}
	if req.SettingKey == "" || !helper.MaxLength(req.SettingKey, 100) {
		return errors.New(errors.Invalid, "setting_key is required and must contain at most 100 characters")
	}
	if !validDataType(req.DataType) {
		return errors.New(errors.Invalid, "data_type must be STRING, NUMBER, BOOLEAN or JSON")
	}
	if !helper.OptionalMaxLength(req.Description, 255) {
		return errors.New(errors.Invalid, "description must contain at most 255 characters")
	}
	if req.Description != nil {
		text := strings.TrimSpace(*req.Description)
		req.Description = nil
		if text != "" {
			req.Description = &text
		}
	}
	// TEXT is bounded in bytes on MySQL; both drivers enforce the same API limit.
	if req.SettingValue == nil {
		return nil
	}
	value := *req.SettingValue
	if !utf8.ValidString(value) || len(value) > 65535 {
		return errors.New(errors.Invalid, "setting_value must be valid UTF-8 and contain at most 65535 bytes")
	}
	if req.DataType == "STRING" {
		return nil
	} // Preserve meaningful whitespace in strings.
	value = strings.TrimSpace(value)
	switch req.DataType {
	case "BOOLEAN":
		if value != "true" && value != "false" {
			return errors.New(errors.Invalid, "BOOLEAN setting_value must be true or false")
		}
	case "NUMBER":
		number, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || !json.Valid([]byte(value)) {
			return errors.New(errors.Invalid, "NUMBER setting_value must be a finite JSON number")
		}
	case "JSON":
		if !json.Valid([]byte(value)) {
			return errors.New(errors.Invalid, "JSON setting_value must contain valid JSON")
		}
	}
	req.SettingValue = &value
	return nil
}
func normalizeFilters(options *query.Options) error {
	for key, values := range options.Filters {
		for i, value := range values {
			switch key {
			case "group_id":
				if !helper.ValidUUID(value) {
					return errors.New(errors.Invalid, "group_id filter must contain UUIDs")
				}
			case "group_code":
				value = strings.ToUpper(value)
				if !helper.MaxLength(value, 50) {
					return errors.New(errors.Invalid, "group_code filter must contain at most 50 characters")
				}
			case "setting_key":
				value = strings.ToLower(value)
				if !helper.MaxLength(value, 100) {
					return errors.New(errors.Invalid, "setting_key filter must contain at most 100 characters")
				}
			case "data_type":
				value = strings.ToUpper(value)
				if !validDataType(value) {
					return errors.New(errors.Invalid, "invalid data_type filter")
				}
			}
			options.Filters[key][i] = value
		}
	}
	return nil
}
