package handler

import (
	"net/http"

	"github.com/edygunawan/go-rest-api/internal/http/response"
)

type healtRes struct {
	Message string
}

func HealtCheck(w http.ResponseWriter, _ *http.Request) {
	result := healtRes{
		Message: "ok",
	}
	response.JSON(w, http.StatusOK, result)
}
