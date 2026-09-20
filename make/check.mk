# Checking

.PHONY: check check-go check-web
check: check-go check-web workflow-check ## Everything CI runs

check-go: format-check vet test ## Check the Go code and run its tests

check-web: test-web lint ## Check the site and run its tests

.PHONY: test test-postgres test-postgres-stop
test: test-postgres ## Run the Go tests; narrow them with TEST=./internal/work/...
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
lint: site-types ## Check the site with Biome and the TypeScript compiler
	cd web && bun run lint
	cd web && bunx next typegen
	cd web && bunx tsc --noEmit

.PHONY: format
format: ## Format Go and site sources in place
	cd api && gofmt -w .
	cd web && bun run format

.PHONY: format-check
format-check: ## Fail if any Go file needs formatting
	@unformatted=$$(cd api && gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "needs gofmt:"; echo "$$unformatted"; exit 1; fi

.PHONY: workflow-check
workflow-check: ## Check GitHub Actions workflows and their shell commands
	$(ACTIONLINT)

.PHONY: proxy-check
proxy-check: ## Check the local nginx configuration
	$(NGINX) -t

.PHONY: production-proxy-check
production-proxy-check: ## Check the production nginx configuration
	docker run --rm -v "$(CURDIR)/nginx/production.conf:/etc/nginx/nginx.conf:ro" \
		nginx:1.31.2-alpine3.23 nginx -t
