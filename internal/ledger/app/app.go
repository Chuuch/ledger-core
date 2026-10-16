// Package app
package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chuuch/ledger-core/internal/ledger/service"
	ledgerhttp "github.com/chuuch/ledger-core/internal/ledger/transport/http"
	"github.com/chuuch/ledger-core/internal/platform/auth"
	"github.com/chuuch/ledger-core/internal/platform/config"
	"github.com/chuuch/ledger-core/internal/platform/httpx"
	"github.com/chuuch/ledger-core/internal/platform/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	cfg    *config.Config
	log    *slog.Logger
	pool   *pgxpool.Pool
	server *httpx.Server
}

func New(ctx context.Context, cfg *config.Config, secrets *config.Secrets, log *slog.Logger) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if secrets == nil {
		return nil, fmt.Errorf("secrets are required")
	}
	if log == nil {
		log = slog.Default()
	}

	pool, err := postgres.NewPool(ctx, secrets.DBConn)
	if err != nil {
		return nil, fmt.Errorf("postgres pool: %w", err)
	}

	qstore := service.NewQueriesStore(pool)
	tx := &service.PoolTxRunner{Pool: pool}
	svc := service.New(qstore, tx)

	var authenticator *auth.Authenticator
	if cfg.OIDCIssuer != "" {
		a, err := auth.NewAuthenticator(ctx, auth.Config{Issuer: cfg.OIDCIssuer})
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("auth: %w", err)
		}
		authenticator = a
	}

	router := ledgerhttp.NewRouter(svc, pool, authenticator)

	return &App{
		cfg:    cfg,
		log:    log,
		pool:   pool,
		server: httpx.NewServer(cfg.ServiceName, cfg.HTTPAddr, router, log),
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.pool.Close()
	a.log.Info("starting", "service", a.cfg.ServiceName)
	return a.server.Run(ctx)
}
