# Running

.PHONY: setup
setup: ## Get a fresh clone ready to run
	@test -f api/.env || { cp api/.env.example api/.env; \
		linking_key=$$(openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n'); \
		sed -i "s/^LINKING_HMAC_KEY=$$/LINKING_HMAC_KEY=$$linking_key/" api/.env; \
		integration_key=$$(openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n'); \
		sed -i "s/^INTEGRATION_SECRET_KEY=$$/INTEGRATION_SECRET_KEY=$$integration_key/" api/.env; \
		echo "Wrote api/.env from the example. Check the database URLs in it."; }
	$(MAKE) web-install
	$(MAKE) migrate

.PHONY: api
api: need-db ## Run the API
	cd api && go run ./cmd/server

.PHONY: web
web: ## Run the site
	cd web && PORT=$(WEB_PORT) bun run dev

.PHONY: proxy
proxy: ## Run the local nginx proxy on port 8000
	$(NGINX) -g 'daemon off;'

.PHONY: web-install
web-install: ## Install the site's locked dependencies
	cd web && bun install --frozen-lockfile

.PHONY: web-ui-view web-ui-add
web-ui-view: ## Show a registry component's source and dependencies; set COMPONENT
	@test -n "$$COMPONENT" || { echo "Set COMPONENT, for example @diceui/timeline."; exit 1; }
	cd web && $(SHADCN) view "$$COMPONENT"

web-ui-add: ## Add one component from a registry; set COMPONENT
	@test -n "$$COMPONENT" || { echo "Set COMPONENT, for example @diceui/timeline."; exit 1; }
	cd web && $(SHADCN) add "$$COMPONENT"
