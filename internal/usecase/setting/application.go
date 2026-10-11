package setting

import (
	"context"
	"strings"
	"time"
	_ "time/tzdata" // Validate IANA names even in containers without zoneinfo files.

	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/helper"
	"github.com/edygunawan/golang-pkgs/errors"
)

func validateApplication(req *ApplicationSettingsRequest) error {
	req.ApplicationName = strings.TrimSpace(req.ApplicationName)
	req.Description = strings.TrimSpace(req.Description)
	req.Timezone = strings.TrimSpace(req.Timezone)
	req.DateFormat = strings.TrimSpace(req.DateFormat)
	if req.ApplicationName == "" || !helper.MaxLength(req.ApplicationName, 150) {
		return errors.New(errors.Invalid, "application_name is required and must contain at most 150 characters")
	}
	if !helper.MaxLength(req.Description, 500) {
		return errors.New(errors.Invalid, "description must contain at most 500 characters")
	}
	if req.Timezone == "" || req.Timezone == "Local" || !helper.MaxLength(req.Timezone, 100) {
		return errors.New(errors.Invalid, "timezone must be a valid IANA timezone")
	}
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		return errors.New(errors.Invalid, "timezone must be a valid IANA timezone")
	}
	switch req.DateFormat {
	case "DD/MM/YYYY", "MM/DD/YYYY", "YYYY-MM-DD":
	default:
		return errors.New(errors.Invalid, "unsupported date_format")
	}
	return nil
}
func (u *usecase) GetApplication(ctx context.Context) (*ApplicationSettingsResponse, error) {
	item, err := u.repo.GetApplication(ctx)
	if err != nil {
		return nil, err
	}
	result := ApplicationSettingsResponse(*item)
	if err := validateApplication(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
func (u *usecase) SaveApplication(ctx context.Context, req ApplicationSettingsRequest) (*ApplicationSettingsResponse, error) {
	if err := validateApplication(&req); err != nil {
		return nil, err
	}
	if err := u.repo.SaveApplication(ctx, domain.ApplicationSettings(req)); err != nil {
		return nil, err
	}
	return &req, nil
}
