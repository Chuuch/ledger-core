// Package Config
package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServiceName string
	HTTPAddr string
	VaultAddr string
	VaultToken string
	VaultSecret string
	OTELEndpoint string
	OIDCIssuer string
}

func Load(serviceName, defaultHTTPAddr, vaultSecretPath string) (*Config, error) {
	cfg := &Config{
		ServiceName: serviceName,
		HTTPAddr: getenv("HTTP_ADDR", defaultHTTPAddr),
		VaultAddr: getenv("VAULT_ADDR", "http://127.0.0.1:8200"),
		VaultToken: getenv("VAULT_TOKEN", "root-token"),
		VaultSecret: getenv("VAULT_SECRET_PATH", vaultSecretPath),
		OTELEndpoint: getenv("OTEL_ENDPOINT", "localhost:4317"),
		OIDCIssuer: getenv("OIDC_ISSUER", "http://127.0.0.1:8080/realms/ledger"),
	}

	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("service name is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
