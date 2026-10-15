package config

import (
	"context"
	"fmt"
	"log/slog"

	vault "github.com/hashicorp/vault/api"
)

type Secrets struct {
	DBConn string
	RedisAddr string
	OTELEndpoint string
}

func LoadVaultSecrets(addr, token, secretPath string) (*Secrets, error) {
	cfg := vault.DefaultConfig()
	cfg.Address = addr

	client, err := vault.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("vault client: %w", err)
	}
	client.SetToken(token)

	secret, err := client.KVv2("secret").Get(context.Background(), secretPath)
	if err != nil {
		return nil, fmt.Errorf("vault get %q: %w", secretPath, err)
	}

	db, err := stringField(secret.Data, "DB_CONN")
	if err != nil {
		return nil, err
	}
	redis, err := stringField(secret.Data, "REDIS_ADDR")
	if err != nil {
		return nil, err
	}

	otel, _ := stringField(secret.Data, "OTEL_ENDPOINT")
	slog.Info("loaded secrets from vault", "path", "secret/"+secretPath)

	return &Secrets{
		DBConn: db,
		RedisAddr: redis,
		OTELEndpoint: otel,
	}, nil
}

func stringField(data map[string]any, key string) (string, error) {
	v, ok := data[key].(string)
	if !ok || v == "" {
		return "", fmt.Errorf("vault secret field %q missing or not a string", key)
	}
	return v, nil
}
