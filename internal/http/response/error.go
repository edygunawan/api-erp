package response

import "net/http"

type ErrorResponse struct {
	Message string `json:"message"`
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, ErrorResponse{
		Message: msg,
	})
}
