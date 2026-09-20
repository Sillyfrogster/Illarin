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

.PHONY: make-admin
make-admin: need-db ## Make an existing account an admin; set HANDLE
	@test -n "$(HANDLE)" || { echo "Set HANDLE to the account to make an admin."; exit 1; }
	cd api && go run ./cmd/make-admin -handle "$(HANDLE)"

.PHONY: need-db
need-db:
	@test -n "$(DATABASE_URL)" || { echo "DATABASE_URL is not set. Run make setup."; exit 1; }
