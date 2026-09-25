# Generated code

.PHONY: generate site-types
generate: site-types ## Regenerate the database code and the site's API types
	cd api && $(SQLC) generate

site-types: ## Write the site's API types from the Go request and response structs
	cd api && $(TYGO) generate --config tygo.yaml

.PHONY: copy-inventory
copy-inventory: ## Print the site and API copy checklist as Markdown
	@cd web && bun scripts/copy-inventory.ts

.PHONY: email-preview
email-preview: ## Render sample account emails in .ai/issues/email-templates
	mkdir -p .ai/issues/email-templates
	cd api && EMAIL_PREVIEW_DIR="$(CURDIR)/.ai/issues/email-templates" go test ./internal/account -run '^TestEmailPreview$$' -count=1

.PHONY: email-preview-serve
email-preview-serve: email-preview ## View sample emails at http://127.0.0.1:8899
	cd .ai/issues/email-templates && python3 -m http.server 8899 --bind 127.0.0.1

.PHONY: refractive-assets
refractive-assets: ## Generate the refractive art assets
	cd web && bun scripts/generate-refractive-assets.mjs

.PHONY: quiet-page-art
quiet-page-art: ## Generate the empty and barren page artwork, one piece per type
	cd web && bun scripts/generate-quiet-page-art.mjs

.PHONY: archive-cutouts
archive-cutouts: ## Neutralize the archive mascot glass cutouts
	cd web && bun scripts/neutralize-archive-cutouts.mjs
