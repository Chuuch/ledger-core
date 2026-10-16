COMPOSE_DIR := deployments
COMPOSE := docker compose -f $(COMPOSE_DIR)/docker-compose.yml
SERVICES := account-service ledger-service transfer-service
ACCOUNT_DATABASE_URL ?= postgres://ledger_user:ledger_password@127.0.0.1:5432/account_db?sslmode=disable
LEDGER_DATABASE_URL ?= postgres://ledger_user:ledger_password@127.0.0.1:5432/ledger_db?sslmode=disable
VAULT_ADDR ?= http://127.0.0.1:8200
VAULT_TOKEN ?= root-token
OIDC_ISSUER ?= http://127.0.0.1:8080/realms/ledger
KEYCLOAK_TOKEN_URL ?= http://127.0.0.1:8080/realms/ledger/protocol/openid-connect/token

.PHONY: test-account test-account-integration run-account build-account token-dev help up down restart ps logs pull vault-init tidy test build ci vet test migrate-ledger-up migrate-ledger-down sqlc-ledger run-ledger build-ledger test-ledger test-ledger-integration

help:
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf " %-16s %s\n", $$1, $$2}'

up: ## Start local infra stack
	$(COMPOSE) up -d

down: ## Stop stack (keep volumes)
	$(COMPOSE) down

restart: ## Restart stack
	$(COMPOSE) restart

ps: ## Show container status
	$(COMPOSE) ps

logs: ## Tail all compose logs
	$(COMPOSE) logs -f --tail=100

pull: ## Pull compose images (no start)
	$(COMPOSE) pull

vet: ## go vet
	go vet ./...

test: ## All Go tests (unit; integration needs ACCOUNT_DATABASE_URL)
	go test ./... -count=1

ci: vet test build ## Local approximation of CI (no Postgres service)
	@echo "ci ok (run integration separately if Postgres is up)"

vault-init: ## Seed Vault KV secrets (stack must be up)
	chmod +x deployments/init-vault.sh
	./deployments/init-vault.sh

tidy: ## go mod tidy
	go mod tidy

build: ## Build all service binaries
	@for s in $(SERVICES); do \
		echo "building $$s"; \
		go build -o bin/$$s ./cmd/$$s; \
	done


run-account: ## Run account-service (host; Vault on localhost)
	DB_CONN='$(ACCOUNT_DATABASE_URL)' \
	VAULT_ADDR='$(VAULT_ADDR)' \
	VAULT_TOKEN='$(VAULT_TOKEN)' \
	OIDC_ISSUER='$(OIDC_ISSUER)' \
	go run ./cmd/account-service

build-account: ## Build account-service binary
	go build -o bin/account-service ./cmd/account-service

test-account: ## Unit/handler tests for account-service
	go test ./internal/account/...

test-account-integration: ## Integration tests (Postgres  required)
	ACCOUNT_DATABASE_URL='$(ACCOUNT_DATABASE_URL)' go test ./internal/account/service -run Integration -count=1

token-dev: ## Fetch Keycloak access token for devuser / ledger-cli
	curl -sS -X POST '$(KEYCLOAK_TOKEN_URL)' \
	  -H 'Content-Type: application/x-www-form-urlencoded' \
	  --data-urlencode 'grant_type=password' \
	  --data-urlencode 'client_id=ledger-cli' \
	  --data-urlencode 'client_secret=ledger-cli-secret' \
	  --data-urlencode 'username=devuser' \
	  --data-urlencode 'password=devuser' \
	  > /tmp/ledger-token.json
	@python3 -c 'import json; json.load(open("/tmp/ledger-token.json"))["access_token"]; print("token ok -> /tmp/ledger-token.json")'
	@echo 'Run: export ACCESS_TOKEN=$$(python3 -c '"'"'import json;print(json.load(open("/tmp/ledger-token.json"))["access_token"])'"'"')'

run-ledger: ## Run ledger-service on host
	DB_CONN='$(LEDGER_DATABASE_URL)' \
	VAULT_ADDR='$(VAULT_ADDR)' \
	VAULT_TOKEN='$(VAULT_TOKEN)' \
	OIDC_ISSUER='$(OIDC_ISSUER)' \
	go run ./cmd/ledger-service

build-ledger: ## Build ledger-service binary
	go build -o bin/ledger-service ./cmd/ledger-service

test-ledger: ## Unit/handler tests for ledger-service
	go test ./internal/ledger/...

test-ledger-integration: ## Integration tests (Postgres ledger_db)
	LEDGER_DATABASE_URL='$(LEDGER_DATABASE_URL)' go test ./internal/ledger/service -run Integration -count=1

run-transfer: ## Run transfer-service
	go run ./cmd/transfer-service

migrate-account-up: ## Aplly account_db migrations
	migrate -path db/account/migrations -database "$(ACCOUNT_DATABASE_URL)" up

migrate-account-down: ## Roll back on account_db migration
	migrate -path db/account/migrations -database "$(ACCOUNT_DATABASE_URL)" down 1

migrate-ledger-up: ## Apply ledger_db migrations
	migrate -path db/ledger/migrations -database "$(LEDGER_DATABASE_URL)" up

migrate-ledger-down: ## Roll back one ledger_db migration
	migrate -path db/ledger/migrations -database "$(LEDGER_DATABASE_URL)" down 1

sqlc-account: ## Generate account store from sqlc
	sqlc generate -f db/account/sqlc.yaml

sqlc-ledger: ## Generate ledger store from sqlc
	sqlc generate -f db/ledger/sqlc.yaml

db-account: migrate-account-up sqlc-account ## Migrate + generate account store


