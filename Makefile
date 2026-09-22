GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.26.0
SQLC  := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
TYGO  := go run github.com/gzuidhof/tygo@v0.2.21
ACTIONLINT := go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
GOTESTSUM := go run gotest.tools/gotestsum@v1.13.0
SHADCN := bunx --bun shadcn@4.21.0
COMPONENT ?=
WEB_PORT ?= 3000
TEST ?= ./...
WEB_TEST ?=
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

AREAS := make/run.mk make/check.mk make/production.mk make/database.mk make/generate.mk
include $(AREAS)

.PHONY: help
help: ## List every target by area
	@for area in $(AREAS); do \
		printf '\n%s\n' "$$(head -1 "$$area" | sed 's/^# //')"; \
		grep -E '^[a-z][a-z-]*:.*##' "$$area" \
			| awk -F':.*## ' '{printf "  \033[1m%-22s\033[0m %s\n", $$1, $$2}'; \
	done
