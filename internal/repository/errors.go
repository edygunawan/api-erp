package repository

import (
	"database/sql"
	"errors"

	"github.com/edygunawan/go-rest-api/internal/helper/query"
	apperrors "github.com/edygunawan/golang-pkgs/errors"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

// Normalize keeps database details out of the public error response.
func Normalize(err error) error {
	if err == nil {
		return nil
	}
	var invalidQuery *query.ValidationError
	if errors.As(err, &invalidQuery) {
		return apperrors.Wrap(apperrors.Invalid, invalidQuery.Message, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return apperrors.New(apperrors.NotFound, "record not found")
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return apperrors.Wrap(apperrors.Conflict, "record already exists", err)
		case "23503":
			return apperrors.Wrap(apperrors.Invalid, "invalid record reference", err)
		case "23514", "22001", "22P02":
			return apperrors.Wrap(apperrors.Invalid, "invalid record", err)
		}
	}
	var my *mysql.MySQLError
	if errors.As(err, &my) {
		switch my.Number {
		case 1062:
			return apperrors.Wrap(apperrors.Conflict, "record already exists", err)
		case 1451, 1452:
			return apperrors.Wrap(apperrors.Invalid, "invalid record reference", err)
		case 1048, 1264, 1366, 1406, 3819:
			return apperrors.Wrap(apperrors.Invalid, "invalid record", err)
		}
	}
	return err
}

func CheckResult(result sql.Result, err error) error {
	if err != nil {
		return Normalize(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return apperrors.New(apperrors.NotFound, "record not found")
	}
	return nil
}
