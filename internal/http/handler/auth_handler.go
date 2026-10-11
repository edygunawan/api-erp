package handler

import (
	"net/http"

	"github.com/edygunawan/go-rest-api/internal/http/request"
	"github.com/edygunawan/go-rest-api/internal/http/response"
	"github.com/edygunawan/go-rest-api/internal/usecase/auth"
	"github.com/edygunawan/golang-pkgs/logger"
)

type AuthHandler struct{ authUC auth.Usecase }

func NewAuthHandler(uc auth.Usecase) *AuthHandler { return &AuthHandler{authUC: uc} }
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var req auth.LoginRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.authUC.Login(r.Context(), req)
	if err != nil {
		logger.FromContext(r.Context()).Error(err, "login failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var req auth.RefreshTokenRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.authUC.RefreshToken(r.Context(), req)
	if err != nil {
		logger.FromContext(r.Context()).Error(err, "refresh failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.authUC.Logout(r.Context()); err != nil {
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "logout success"})
}
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	result, err := h.authUC.Me(r.Context())
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
