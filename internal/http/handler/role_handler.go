package handler

import (
	"net/http"

	"github.com/edygunawan/go-rest-api/internal/http/request"
	"github.com/edygunawan/go-rest-api/internal/http/response"
	"github.com/edygunawan/go-rest-api/internal/usecase/role"
	"github.com/edygunawan/golang-pkgs/logger"
	"github.com/go-chi/chi/v5"
)

type RoleHandler struct {
	roleUC role.Usecase
}

func NewRoleHandler(roleUC role.Usecase) *RoleHandler {
	return &RoleHandler{
		roleUC: roleUC,
	}
}

func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req role.CreateRoleRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.roleUC.Create(r.Context(), req)
	if err != nil {
		log.Error(err, "create role failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req role.UpdateRoleRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.roleUC.Update(r.Context(), id, req)
	if err != nil {
		log.Error(err, "update role failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.roleUC.Delete(r.Context(), id); err != nil {
		log.Error(err, "delete role failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "role deleted",
	})
}

func (h *RoleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	result, err := h.roleUC.GetByID(r.Context(), id)
	if err != nil {
		log.Error(err, "get role failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *RoleHandler) GetAllRoles(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	options, err := request.ParseList(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.roleUC.List(r.Context(), options)
	if err != nil {
		log.Error(err, "get roles failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
