// Package http
package http

import (
	"context"
	"net/http"
	"time"

	"github.com/chuuch/ledger-core/internal/ledger/service"
	"github.com/chuuch/ledger-core/internal/platform/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(svc *service.Service, pool *pgxpool.Pool, authenticator *auth.Authenticator) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(middleware.Recoverer)

	h := NewHandler(svc)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if pool == nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	r.Route("/v1", func(r chi.Router) {
		if authenticator != nil {
			r.Use(authenticator.Middleware)
			r.With(auth.RequireRoles("ledger-write")).Post("/journals", h.PostJournal)
			r.With(auth.RequireRoles("ledger-read")).Get("/journals/{id}", h.GetJournal)
			r.With(auth.RequireRoles("ledger-read")).Get("/accounts/{accountId}/balance", h.GetBalance)
			return
		}

		r.Post("/journals", h.PostJournal)
		r.Get("/journals/{id}", h.GetJournal)
		r.Get("/accounts/{accountID}/balance", h.GetBalance)
	})

	return r
}
