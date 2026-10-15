package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/chuuch/ledger-core/internal/account/app"
	"github.com/chuuch/ledger-core/internal/platform/config"
)

func main() {
	os.Exit(run())
}

func run() int {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := config.Load("transfer-service", ":8083", "transfer-service")
	if err != nil {
		log.Error("load config", "err", err)
		return 1
	}

	secrets, err := config.LoadVaultSecrets(cfg.VaultAddr, cfg.VaultToken, cfg.VaultSecret)
	if err != nil {
		log.Error("load vault secrets", "err", err)
		return 1
	}

	application, err := app.New(context.Background(), cfg, secrets, log)
	if err != nil {
		log.Error("build app", "err", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		log.Error("run", "err", err)
		return 1
	}
	return 0
}
