GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.26.0
SQLC  := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
OAPI  := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0
ACTIONLINT := go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
GOTESTSUM := go run gotest.tools/gotestsum@v1.13.0
SHADCN := bunx --bun shadcn@4.21.0
COMPONENT ?=
WEB_PORT ?= 3000
TEST ?= ./...
VERSION ?=
SERVICE ?=
OUTPUT ?= illarin-release.tar.gz
PROD_ENV ?= $(or $(ILLARIN_ENV_FILE),/etc/illarin/production.env)
NGINX_IMAGE ?= nginx:alpine
TEST_JSON ?=
TEST_TIMEOUT ?= 4m
# The test Postgres keeps its data in memory and skips durability, since every test database is thrown away.
TEST_POSTGRES := illarin-test-postgres
TEST_POSTGRES_IMAGE := postgres:18.6-alpine3.23
TEST_POSTGRES_PORT ?= 55432
TEST_POSTGRES_URL := postgres://postgres:postgres@127.0.0.1:$(TEST_POSTGRES_PORT)/illarin_test?sslmode=disable
NGINX := docker run --rm --network host -v "$(CURDIR):/work:ro" -w /work $(NGINX_IMAGE) \
	nginx -p /work/ -c nginx/local.conf

-include api/.env
export

.DEFAULT_GOAL := help

.PHONY: help
help: ## List every target
	@grep -hE '^[a-z][a-z-]*:.*##' $(MAKEFILE_LIST) \
		| sort \
		| awk -F':.*## ' '{printf "  \033[1m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup: ## Get a fresh clone ready to run
	@test -f api/.env || { cp api/.env.example api/.env; \
		linking_key=$$(openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n'); \
		sed -i "s/^LINKING_HMAC_KEY=$$/LINKING_HMAC_KEY=$$linking_key/" api/.env; \
		publication_key=$$(openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n'); \
		sed -i "s/^PUBLICATION_SECRET_KEY=$$/PUBLICATION_SECRET_KEY=$$publication_key/" api/.env; \
		echo "Wrote api/.env from the example. Check the database URLs in it."; }
	$(MAKE) web-install
	$(MAKE) migrate

.PHONY: production-setup
production-setup: ## Walk through the reference production integrations
	@ENV_FILE="$(CURDIR)/ops/.env" ./ops/setup-production.sh

.PHONY: web-install
web-install: ## Install the site's locked dependencies
	cd web && bun install --frozen-lockfile

.PHONY: web-ui-view web-ui-add
web-ui-view: ## Inspect a component's source and dependencies; set COMPONENT
	@test -n "$$COMPONENT" || { echo "Set COMPONENT, for example @diceui/timeline."; exit 1; }
	cd web && $(SHADCN) view "$$COMPONENT"

web-ui-add: ## Add one component from a registry; set COMPONENT
	@test -n "$$COMPONENT" || { echo "Set COMPONENT, for example @diceui/timeline."; exit 1; }
	cd web && $(SHADCN) add "$$COMPONENT"

# Running

.PHONY: api
api: need-db ## Run the API
	cd api && go run ./cmd/server

.PHONY: web
web: ## Run the site
	cd web && PORT=$(WEB_PORT) bun run dev

.PHONY: proxy
proxy: ## Run the local nginx proxy on port 8000
	$(NGINX) -g 'daemon off;'

# Production

.PHONY: prod-deploy
prod-deploy: ## Deploy one immutable Git commit to production
	@ILLARIN_ENV_FILE="$(PROD_ENV)" VERSION="$(VERSION)" ./ops/deploy.sh

.PHONY: prod-rollback
prod-rollback: ## Restore the previously deployed application images
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/rollback.sh

.PHONY: prod-status
prod-status: ## Show production container and health status
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/compose.sh ps

.PHONY: prod-logs
prod-logs: ## Follow recent production logs; optionally set SERVICE
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/compose.sh logs --tail=200 --follow $(SERVICE)

.PHONY: prod-restart
prod-restart: ## Restart one production service with SERVICE=api, web or gateway
	@test -n "$(SERVICE)" || { echo "Set SERVICE to api, web or gateway."; exit 1; }
	@case "$(SERVICE)" in api|web|gateway) ;; *) echo "SERVICE must be api, web or gateway."; exit 1 ;; esac
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/compose.sh restart "$(SERVICE)"

.PHONY: prod-smoke
prod-smoke: ## Check the production gateway, API and site
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/smoke.sh

.PHONY: prod-config-check
prod-config-check: ## Validate the production Compose configuration
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/compose.sh config --quiet

.PHONY: prod-publication-authority
prod-publication-authority: ## Give an existing production account publication authority; set HANDLE
	@test -n "$$HANDLE" || { echo "Set HANDLE to your existing account handle."; exit 1; }
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/compose.sh exec -T api /app/publication-authority -handle "$$HANDLE"

.PHONY: prod-backup prod-backup-init prod-backup-check
prod-backup: ## Run an off-box backup when backups are enabled
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/backup.sh run

prod-backup-init: ## Create the configured off-box backup repository
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/backup.sh init

prod-backup-check: ## Verify the configured off-box backup repository
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/backup.sh check

.PHONY: release-package
release-package: ## Package the production control files for a host
	@test "$(VERSION)" != "" || { echo "Set VERSION to a full Git commit SHA."; exit 1; }
	@git archive --format=tar.gz --output="$(OUTPUT)" "$(VERSION)" \
		Makefile compose.prod.yaml compose.microsoft365.yaml compose.npmplus.yaml \
		nginx/production.conf ops

# Checking

.PHONY: check check-go check-web
check: check-go check-web workflow-check ## Everything CI runs

check-go: fmt-check vet test ## Check the Go code and run its tests

check-web: test-web lint openapi-check ## Check the site and run its tests

.PHONY: test test-postgres test-postgres-stop
test: test-postgres ## Run the Go tests; narrow them with TEST=./internal/http/...
	cd api && TEST_DATABASE_URL="$(TEST_POSTGRES_URL)" $(GOTESTSUM) --format-hide-empty-pkg \
		$(if $(TEST_JSON),--jsonfile "$(TEST_JSON)") -- -short -timeout $(TEST_TIMEOUT) $(TEST)

test-postgres: ## Start the in-memory Postgres the Go tests run against
	@docker container inspect -f '{{.State.Running}}' $(TEST_POSTGRES) 2>/dev/null | grep -qx true || { \
		docker rm -f $(TEST_POSTGRES) >/dev/null 2>&1; \
		docker run --detach --name $(TEST_POSTGRES) --tmpfs /var/lib/postgresql \
			--publish 127.0.0.1:$(TEST_POSTGRES_PORT):5432 \
			--env POSTGRES_PASSWORD=postgres --env POSTGRES_DB=illarin_test \
			$(TEST_POSTGRES_IMAGE) -c fsync=off -c synchronous_commit=off \
			-c full_page_writes=off -c max_connections=200 >/dev/null; }
	@for attempt in $$(seq 60); do \
		docker exec $(TEST_POSTGRES) pg_isready --quiet --host 127.0.0.1 --username postgres && exit 0; \
		sleep 0.5; \
	done; \
	echo "The test Postgres did not become ready:"; docker logs --tail 20 $(TEST_POSTGRES); exit 1

test-postgres-stop: ## Remove the in-memory test Postgres and every database in it
	docker rm -f $(TEST_POSTGRES)

.PHONY: test-web
test-web: ## Run the site tests
	cd web && bun test

.PHONY: cover
cover: test-postgres ## Report Go test coverage per package
	cd api && TEST_DATABASE_URL="$(TEST_POSTGRES_URL)" $(GOTESTSUM) --format-hide-empty-pkg \
		-- -short -cover ./...

.PHONY: vet
vet: ## Report suspicious Go code
	cd api && go vet ./...

.PHONY: lint
lint: ## Check the site with Biome and the TypeScript compiler
	cd web && bun run lint
	cd web && bunx next typegen
	cd web && bunx tsc --noEmit

.PHONY: fmt
fmt: ## Format Go and site sources in place
	cd api && gofmt -w .
	cd web && bun run format

.PHONY: fmt-check
fmt-check: ## Fail if any Go file needs formatting
	@unformatted=$$(cd api && gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "needs gofmt:"; echo "$$unformatted"; exit 1; fi

.PHONY: proxy-check
proxy-check: ## Check the local nginx configuration
	$(NGINX) -t

.PHONY: workflow-check
workflow-check: ## Check GitHub Actions workflows and their shell commands
	$(ACTIONLINT)

.PHONY: production-proxy-check
production-proxy-check: ## Check the production nginx configuration
	docker run --rm -v "$(CURDIR)/nginx/production.conf:/etc/nginx/nginx.conf:ro" \
		nginx:1.31.2-alpine3.23 nginx -t

.PHONY: image-api image-web images
image-api: ## Build the production Go image locally
	docker build -f api/Dockerfile -t illarin-api:local .

image-web: ## Build the production Next image locally
	docker build --build-arg ILLARIN_VERSION="$(or $(VERSION),development)" \
		-f web/Dockerfile -t illarin-web:local .

images: image-api image-web ## Build both production application images

.PHONY: production-stack-test
production-stack-test: VERSION := $(shell git rev-parse HEAD)
production-stack-test: images ## Run an isolated smoke test against the production Compose stack
	@./ops/test-production-stack.sh

# Database

.PHONY: migrate
migrate: need-db ## Apply migrations to the dev database
	cd api && $(GOOSE) -dir migrations postgres "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: need-db ## Roll the dev database back one migration
	cd api && $(GOOSE) -dir migrations postgres "$(DATABASE_URL)" down

.PHONY: migrate-status
migrate-status: need-db ## Show which migrations have run
	cd api && $(GOOSE) -dir migrations postgres "$(DATABASE_URL)" status

.PHONY: publication-authority
publication-authority: need-db ## Record which account holds publication authority; set HANDLE
	@test -n "$(HANDLE)" || { echo "Set HANDLE to the account that holds publication authority."; exit 1; }
	cd api && go run ./cmd/publication-authority -handle "$(HANDLE)"

# Generated code, never hand edited

.PHONY: generate openapi-bundle
generate: openapi-bundle ## Regenerate database code, server stubs, and the site's API types
	cd api && $(SQLC) generate
	cd api/openapi && $(OAPI) -config cfg-server.yaml openapi.gen.yaml
	cd web && bun run gen:api

openapi-bundle: ## Bundle the OpenAPI modules for publication and generation
	cd web && bun run bundle:api

.PHONY: openapi-check
openapi-check: ## Check that the published OpenAPI bundle is current
	cd web && bun run check:api

.PHONY: refractive-assets
refractive-assets: ## Generate the deterministic refractive art assets
	cd web && bun scripts/generate-refractive-assets.mjs

.PHONY: quiet-page-art
quiet-page-art: ## Generate the empty and barren page artwork, one piece per kind
	cd web && bun scripts/generate-quiet-page-art.mjs

.PHONY: archive-cutouts
archive-cutouts: ## Neutralize the archive mascot glass cutouts
	cd web && bun scripts/neutralize-archive-cutouts.mjs

# Guards

.PHONY: need-db
need-db:
	@test -n "$(DATABASE_URL)" || { echo "DATABASE_URL is not set. Run make setup."; exit 1; }
