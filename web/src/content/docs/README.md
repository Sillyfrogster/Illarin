# Documentation source

Each guide in this directory is a public page at `/docs/<file name>`. The catalog in `src/lib/docs.ts` controls navigation order and page descriptions. Use explicit, stable headings because their slugs are public links. Keep protocol changes and their examples in these guides; `api/protocol.md` is a pointer for older readers.

`api/internal/connect/documentation_test.go` reads the published request examples and runs representative connection, permission, send, and library calls against the API with synthetic data. Run it with `make test TEST=./internal/connect/...`. The documentation render and link check is `make test-web WEB_TEST=src/components/docs/DocMarkdown.test.ts`.
