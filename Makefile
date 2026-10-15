COMPOSE_DIR := deployments
COMPOSE := docker compose -f $(COMPOSE_DIR)/docker-compose.yml
SERVICES := account-service ledger-service transfer-service

.PHONY: help up down restart ps logs pull vault-init tidy test build

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

vault-init: ## Seed Vault KV secrets (stack must be up)
	chmod +x deployments/init-vault.sh
	./deployments/init-vault.sh

tidy: ## go mod tidy
	go mod tidy

test: ## Run Go tests
	go test ./...

build: ## Build all service binaries
	@for s in $(SERVICES); do \
		echo "building $$s"; \
		go build -o bin/$$s ./cmd/$$s; \
	done

