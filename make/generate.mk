# Generated code

.PHONY: generate site-types
generate: site-types ## Regenerate the database code and the site's API types
	cd api && $(SQLC) generate

site-types: ## Write the site's API types from the Go request and response structs
	cd api && $(TYGO) generate --config tygo.yaml

.PHONY: refractive-assets
refractive-assets: ## Generate the refractive art assets
	cd web && bun scripts/generate-refractive-assets.mjs

.PHONY: quiet-page-art
quiet-page-art: ## Generate the empty and barren page artwork, one piece per type
	cd web && bun scripts/generate-quiet-page-art.mjs

.PHONY: archive-cutouts
archive-cutouts: ## Neutralize the archive mascot glass cutouts
	cd web && bun scripts/neutralize-archive-cutouts.mjs
