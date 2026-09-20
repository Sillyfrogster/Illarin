# Production

.PHONY: prod-deploy
prod-deploy: ## Deploy one Git commit to production; set VERSION
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
prod-restart: ## Restart one production service; set SERVICE to api, web or gateway
	@test -n "$(SERVICE)" || { echo "Set SERVICE to api, web or gateway."; exit 1; }
	@case "$(SERVICE)" in api|web|gateway) ;; *) echo "SERVICE must be api, web or gateway."; exit 1 ;; esac
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/compose.sh restart "$(SERVICE)"

.PHONY: prod-smoke
prod-smoke: ## Check the production gateway, API and site
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/smoke.sh

.PHONY: prod-config-check
prod-config-check: ## Check the production Compose configuration
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/compose.sh config --quiet

.PHONY: prod-make-admin
prod-make-admin: ## Make an existing production account an admin; set HANDLE
	@test -n "$$HANDLE" || { echo "Set HANDLE to your existing account handle."; exit 1; }
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/compose.sh exec -T api /app/make-admin -handle "$$HANDLE"

.PHONY: prod-backup prod-backup-init prod-backup-check
prod-backup: ## Run an off-box backup when backups are enabled
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/backup.sh run

prod-backup-init: ## Create the off-box backup repository
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/backup.sh init

prod-backup-check: ## Verify the off-box backup repository
	@ILLARIN_ENV_FILE="$(PROD_ENV)" ./ops/backup.sh check

.PHONY: release-package
release-package: ## Package the production control files for a host; set VERSION
	@test "$(VERSION)" != "" || { echo "Set VERSION to a full Git commit SHA."; exit 1; }
	@git archive --format=tar.gz --output="$(OUTPUT)" "$(VERSION)" \
		Makefile make compose.prod.yaml compose.microsoft365.yaml compose.npmplus.yaml \
		nginx/production.conf ops

.PHONY: image-api image-web images
image-api: ## Build the production Go image locally
	docker build -f api/Dockerfile -t illarin-api:local .

image-web: ## Build the production Next image locally
	docker build --build-arg ILLARIN_VERSION="$(or $(VERSION),development)" \
		-f web/Dockerfile -t illarin-web:local .

images: image-api image-web ## Build both production application images
