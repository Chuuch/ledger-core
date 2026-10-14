#!/usr/bin/env bash
set -euo pipefail

# Seeds local Vault KV (dev). Prefer: make vault-init
# Requires: stack up, container name ledger-vault

VAULT_CONTAINER="${VAULT_CONTAINER:-ledger-vault}"
VAULT_ADDR_IN_CONTAINER="${VAULT_ADDR_IN_CONTAINER:-http://127.0.0.1:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-root-token}"

vault_exec() {
  docker exec \
    -e "VAULT_ADDR=${VAULT_ADDR_IN_CONTAINER}" \
    -e "VAULT_TOKEN=${VAULT_TOKEN}" \
    "${VAULT_CONTAINER}" \
    vault "$@"
}

echo "Waiting for Vault..."
for i in $(seq 1 30); do
  if vault_exec status >/dev/null 2>&1; then
    break
  fi
  sleep 1
  if [[ "$i" -eq 30 ]]; then
    echo "Vault did not become ready in time" >&2
    exit 1
  fi
done

# Dev mode usually mounts KV v2 at secret/. Enable if missing.
if ! vault_exec secrets list -format=json 2>/dev/null | grep -q '"secret/"'; then
  vault_exec secrets enable -path=secret kv-v2
fi

# DB hosts use Docker Compose service DNS names (for apps running in compose);
# Host-run apps: swap postgres/redis for 127.0.0.1.
vault_exec kv put secret/account-service \
  DB_CONN='postgres://ledger_user:ledger_password@postgres:5432/account_db?sslmode=disable' \
  REDIS_ADDR='redis:6379' \
  OTEL_ENDPOINT='otel-collector:4317'

vault_exec kv put secret/ledger-service \
  DB_CONN='postgres://ledger_user:ledger_password@postgres:5432/ledger_db?sslmode=disable' \
  REDIS_ADDR='redis:6379' \
  OTEL_ENDPOINT='otel-collector:4317'

vault_exec kv put secret/transfer-service \
  DB_CONN='postgres://ledger_user:ledger_password@postgres:5432/transfer_db?sslmode=disable' \
  REDIS_ADDR='redis:6379' \
  OTEL_ENDPOINT='otel-collector:4317' \
  ACCOUNT_SERVICE_URL='http://account-service:8081' \
  LEDGER_SERVICE_URL='http://ledger-service:8082'

echo "Vault secrets written:"
echo " secret/account-service"
echo " secret/ledger-service"
echo " secret/transfer-service"
