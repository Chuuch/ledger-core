// Package http
package http

import (
	"encoding/json/v2"
	"errors"
	"net/http"

	"github.com/chuuch/ledger-core/internal/ledger/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

type postingRequest struct {
	AccountID string `json:"account_id"`
	Direction string `json:"direction"`
	Amount int64 `json:"amount"`
}

type postJournalRequest struct {
	Reference *string `json:"reference,omitempty"`
	Description string `json:"description"`
	Currency string `json:"currency"`
	Postings []postingRequest `json:"postings"`
}

func (h *Handler) PostJournal(w http.ResponseWriter, r *http.Request) {
	var req postJournalRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	posts := make([]service.PostingInput, 0, len(req.Postings))
	for _, p := range req.Postings {
		id, err := uuid.Parse(p.AccountID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid account_id in postings")
			return
		}
		posts = append(posts, service.PostingInput{
			AccountID: id,
			Direction: p.Direction,
			Amount: p.Amount,
		})
	}

	view, err := h.svc.PostJournal(r.Context(), service.PostJournalInput{
		Reference: req.Reference,
		Description: req.Description,
		Currency: req.Currency,
		Postings: posts,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (h *Handler) GetJournal(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid journal id")
		return
	}
	view, err := h.svc.GetJournal(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	accountID, err := uuid.Parse(chi.URLParam(r, "accountID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}
	currency := r.URL.Query().Get("currency")
	bal, err := h.svc.GetBalance(r.Context(), accountID, currency)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bal)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrUnbalanced):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.UnmarshalRead(r.Body, dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.MarshalWrite(w, v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
