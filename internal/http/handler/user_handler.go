package handler

import (
	"net/http"

	"github.com/edygunawan/go-rest-api/internal/http/request"
	"github.com/edygunawan/go-rest-api/internal/http/response"
	"github.com/edygunawan/go-rest-api/internal/usecase/user"
	"github.com/edygunawan/golang-pkgs/logger"
	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	userUC user.Usecase
}

func NewUserHandler(userUC user.Usecase) *UserHandler {
	return &UserHandler{
		userUC: userUC,
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req user.CreateUserRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.userUC.Create(r.Context(), req)
	if err != nil {
		log.Error(err, "create user failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req user.UpdateUserRequest
	if err := request.Decode(r, &req); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.userUC.Update(r.Context(), id, req)
	if err != nil {
		log.Error(err, "update user failed")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	result, err := h.userUC.GetByID(r.Context(), id)
	if err != nil {
		log.Error(err, "error")
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *UserHandler) GetAllUser(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	options, err := request.ParseList(r, "status")
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.userUC.List(r.Context(), options)

	if err != nil {
		log.Error(err, err.Error())
		response.FromError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
