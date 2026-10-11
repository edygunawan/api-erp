package handler

import (
	"net/http"

	"github.com/edygunawan/go-rest-api/internal/http/request"
	"github.com/edygunawan/go-rest-api/internal/http/response"
	"github.com/edygunawan/go-rest-api/internal/usecase/permission"
	"github.com/edygunawan/golang-pkgs/logger"
	"github.com/go-chi/chi/v5"
)

type PermissionHandler struct {
	permissionUC permission.Usecase
}

func NewPermissionHandler(permissionUC permission.Usecase) *PermissionHandler {
	return &PermissionHandler{
		permissionUC: permissionUC,
	}
}

func (h *PermissionHandler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req permission.CreatePermissionRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.permissionUC.Create(r.Context(), req)
	if err != nil {
		log.Error(err, "create permission failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

func (h *PermissionHandler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req permission.UpdatePermissionRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.permissionUC.Update(r.Context(), id, req)
	if err != nil {
		log.Error(err, "update permission failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *PermissionHandler) DeletePermission(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.permissionUC.Delete(r.Context(), id); err != nil {
		log.Error(err, "delete permission failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "permission deleted",
	})
}

func (h *PermissionHandler) GetPermission(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	result, err := h.permissionUC.GetByID(r.Context(), id)
	if err != nil {
		log.Error(err, "get permission failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *PermissionHandler) GetAllPermissions(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	options, err := request.ParseList(r, "module")
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.permissionUC.List(r.Context(), options)
	if err != nil {
		log.Error(err, "get permissions failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
