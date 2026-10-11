package response

import (
	"errors"
	"net/http"

	apperrors "github.com/edygunawan/golang-pkgs/errors"
)

func FromError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) && appErr.Code >= 400 && appErr.Code < 500 {
		Error(w, int(appErr.Code), appErr.Message)
		return
	}
	Error(w, http.StatusInternalServerError, "internal server error")
}
