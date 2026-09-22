# Documentation source

Each guide in this directory is a public page at `/docs/<file name>`. The catalog in `src/lib/docs.ts` controls navigation order and page descriptions. Use explicit, stable headings because their slugs are public links. Keep protocol changes and their examples in these guides; `api/protocol.md` is a pointer for older readers.

The synthetic HTTP examples are exercised at the API boundary by `api/internal/connect/connection_handlers_test.go`, `permissions_test.go`, and `send_handlers_test.go`. Run them with `make test TEST=./internal/connect/...`. The documentation link check is `make test-web WEB_TEST=src/components/docs/DocMarkdown.test.ts`.
