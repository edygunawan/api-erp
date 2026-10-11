package handler

import (
	"net/http"

	"github.com/edygunawan/go-rest-api/internal/http/request"
	"github.com/edygunawan/go-rest-api/internal/http/response"
	uc "github.com/edygunawan/go-rest-api/internal/usecase/setting"
	"github.com/edygunawan/golang-pkgs/logger"
	"github.com/go-chi/chi/v5"
)

type SettingHandler struct{ uc uc.Usecase }

func NewSettingHandler(usecase uc.Usecase) *SettingHandler { return &SettingHandler{uc: usecase} }
func (h *SettingHandler) CreateSetting(w http.ResponseWriter, r *http.Request) {
	var req uc.CreateSettingRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.Create(r.Context(), req)
	if err != nil {
		logger.FromContext(r.Context()).Error(err, "create settings failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, result)
}
func (h *SettingHandler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	var req uc.UpdateSettingRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.Update(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		logger.FromContext(r.Context()).Error(err, "update settings failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
func (h *SettingHandler) DeleteSetting(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		logger.FromContext(r.Context()).Error(err, "delete settings failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "settings deleted"})
}
func (h *SettingHandler) GetSetting(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		logger.FromContext(r.Context()).Error(err, "get settings failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
func (h *SettingHandler) ListSetting(w http.ResponseWriter, r *http.Request) {
	options, err := request.ParseList(r, "group_id", "group_code", "setting_key", "data_type")
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.List(r.Context(), options)
	if err != nil {
		logger.FromContext(r.Context()).Error(err, "list settings failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *SettingHandler) GetApplication(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.GetApplication(r.Context())
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
func (h *SettingHandler) SaveApplication(w http.ResponseWriter, r *http.Request) {
	var req uc.ApplicationSettingsRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.SaveApplication(r.Context(), req)
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
