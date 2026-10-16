package http

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/ledger-core/internal/account/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

type createAccountRequest struct {
	OwnerID  string `json:"owner_id"`
	Currency string `json:"currency"`
	Status   string `json:"status,omitempty"`
}

type updateStatusRequest struct {
	Status  string `json:"status"`
	Version int64  `json:"version"`
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := decodeNSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	acc, err := h.svc.Create(r.Context(), service.CreateInput{
		OwnerID:  req.OwnerID,
		Currency: req.Currency,
		Status:   req.Status,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, acc)
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	acc, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	ownerID := r.URL.Query().Get("owner_id")
	accs, err := h.svc.ListByOwner(r.Context(), ownerID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accs)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	var req updateStatusRequest
	if err := decodeNSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	acc, err := h.svc.UpdateStatus(r.Context(), service.UpdateStatusInput{
		ID:      id,
		Status:  req.Status,
		Version: req.Version,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrInvalidStatus):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func decodeNSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.UnmarshalRead(r.Body, dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.MarshalWrite(w, v); err != nil {
		return
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
