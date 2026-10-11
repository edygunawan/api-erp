package handler

import (
	"net/http"

	"github.com/edygunawan/go-rest-api/internal/http/request"
	"github.com/edygunawan/go-rest-api/internal/http/response"
	uc "github.com/edygunawan/go-rest-api/internal/usecase/settinggroup"
	"github.com/edygunawan/golang-pkgs/logger"
	"github.com/go-chi/chi/v5"
)

type SettingGroupHandler struct{ uc uc.Usecase }

func NewSettingGroupHandler(usecase uc.Usecase) *SettingGroupHandler {
	return &SettingGroupHandler{uc: usecase}
}
func (h *SettingGroupHandler) CreateSettingGroup(w http.ResponseWriter, r *http.Request) {
	var req uc.CreateSettingGroupRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.Create(r.Context(), req)
	if err != nil {
		logger.FromContext(r.Context()).Error(err, "create setting-groups failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, result)
}
func (h *SettingGroupHandler) UpdateSettingGroup(w http.ResponseWriter, r *http.Request) {
	var req uc.UpdateSettingGroupRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.Update(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		logger.FromContext(r.Context()).Error(err, "update setting-groups failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
func (h *SettingGroupHandler) DeleteSettingGroup(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		logger.FromContext(r.Context()).Error(err, "delete setting-groups failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "setting-groups deleted"})
}
func (h *SettingGroupHandler) GetSettingGroup(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		logger.FromContext(r.Context()).Error(err, "get setting-groups failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
func (h *SettingGroupHandler) ListSettingGroup(w http.ResponseWriter, r *http.Request) {
	options, err := request.ParseList(r, "code")
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.uc.List(r.Context(), options)
	if err != nil {
		logger.FromContext(r.Context()).Error(err, "list setting-groups failed")
		response.FromError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
