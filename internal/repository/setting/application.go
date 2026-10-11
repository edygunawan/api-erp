package setting

import (
	"context"
	"github.com/edygunawan/go-rest-api/internal/domain"
	"github.com/edygunawan/go-rest-api/internal/repository"
	"github.com/edygunawan/golang-pkgs/errors"
)

func (r *SettingRepository) GetApplication(ctx context.Context) (*domain.ApplicationSettings, error) {
	var rows []struct {
		GroupCode string  `db:"group_code"`
		Key       string  `db:"setting_key"`
		Value     *string `db:"setting_value"`
		DataType  string  `db:"data_type"`
	}
	err := r.dbx.SelectContext(ctx, &rows, `SELECT g.code AS group_code,s.setting_key,s.setting_value,s.data_type FROM settings s JOIN setting_groups g ON g.id=s.group_id WHERE (g.code='GENERAL' AND s.setting_key IN ('application_name','description')) OR (g.code='LOCALIZATION' AND s.setting_key IN ('timezone','date_format'))`)
	if err != nil {
		return nil, repository.Normalize(err)
	}
	if len(rows) != 4 {
		return nil, errors.New(errors.NotFound, "application settings are incomplete; run migrations or restore missing records")
	}
	result := &domain.ApplicationSettings{}
	for _, row := range rows {
		if row.DataType != "STRING" {
			return nil, errors.New(errors.Invalid, "application settings must use STRING values")
		}
		value := ""
		if row.Value != nil {
			value = *row.Value
		}
		switch row.GroupCode + "." + row.Key {
		case "GENERAL.application_name":
			result.ApplicationName = value
		case "GENERAL.description":
			result.Description = value
		case "LOCALIZATION.timezone":
			result.Timezone = value
		case "LOCALIZATION.date_format":
			result.DateFormat = value
		}
	}
	return result, nil
}
func (r *SettingRepository) SaveApplication(ctx context.Context, item domain.ApplicationSettings) error {
	tx, err := r.dbx.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Stable order locks shared rows consistently. Any missing record rolls back all edits.
	for _, field := range []struct{ group, key, value string }{
		{"GENERAL", "application_name", item.ApplicationName}, {"GENERAL", "description", item.Description},
		{"LOCALIZATION", "timezone", item.Timezone}, {"LOCALIZATION", "date_format", item.DateFormat},
	} {
		result, err := tx.ExecContext(ctx, tx.Rebind(`UPDATE settings SET setting_value=?,data_type='STRING',updated_at=`+r.updatedAtSQL+` WHERE setting_key=? AND group_id IN (SELECT id FROM setting_groups WHERE code=?)`), field.value, field.key, field.group)
		if err := repository.CheckResult(result, err); err != nil {
			return err
		}
	}
	return repository.Normalize(tx.Commit())
}
