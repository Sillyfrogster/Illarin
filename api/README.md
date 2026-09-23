# API layout

The Go code under `internal/` has one package for each thing the product has:
`account`, `profile`, `notify` and so on. The package name says what the code
is about, never what sort of code it is.

A feature's handlers live in its package, next to the request and response
structs they read and write. Each feature package registers its own routes on
gin with a `Register` function, and gives each route a deadline as it does.

Plumbing that every feature needs lives in `internal/api`: the session and the
signed-in account, cookies, the check that a change came from the site, reading
path and query values, the error body, route deadlines, the panic recovery,
the no-store header on replies that carry credentials, and the helpers that let
a renamed field keep answering to its old name. Each feature keeps its old
paths and field names in one `aliases.go`. A feature package
imports `api`; `api` imports no feature.

`cmd/server` builds the gin engine, puts the shared middleware on it and calls
each feature's `Register`. Nothing else wires the routes together.

When a package grows past about fifteen files, split it by feature into smaller
packages. Never split it by layer, so there is no `handlers` or `models`
package.

Tests that send real requests import `internal/apitest` for the shared helpers,
and `internal/apitest/full` for a router that serves every route. A test in
`cmd/server` fails if that router and the server stop serving the same routes.

## The packages

Features, one per thing the product has:

- `account` — sign-in, email, password, Discord, deletion, suspension.
- `profile` — the profile, its avatar, banner and links.
- `page` — a work's page and its listings: details, visibility, following,
  deletion, browse, preserved data.
- `block` — the page's blocks and elements; `block/edit` saves changes to them.
- `version` — versions, drafted changes, history, comparison.
- `upload` — reading a file in, found images, replacement preview.
- `download` — formats, the main file, download records, format comparison.
- `image` — the pictures a work, a profile or a post owns.
- `private` — private prompts.
- `connect` — connected apps: connecting, permissions, sends, the app's library.
- `integration` — outgoing announcements: `integration/blog` picks the blog's
  integrations, `integration/discord` and `integration/dispatch` send them.
- `notify` — the inbox and what a person follows.
- `blog` — posts, versions, schedule, writers, categories, feeds, link cards,
  import; `blog/body` is the post body format.
- `staff` — reports, taking down works, restricting profiles, cases, strikes, appeals.

Everything else is plumbing a feature reaches for:

- `api` — what every feature needs from the request, as above.
- `work` — the stored work itself: its row, original files, versions, media,
  preserved payloads, and the service every feature builds on.
- `format` — one reader and writer per file format, plus recognition;
  `format/modules` holds the registry and the rest is one package per format.
- `media` — reading an image in and rendering its sizes.
- `storage` — blobs, the image cache, cleanup, purge.
- `summary` — the summaries browse and the download page read.
- `credential` — hashing and checking a secret.
- `secrets` — the sealing key.
- `jscode` — the check that an extension's JavaScript is safe to list.
- `discord` — the Discord sign-in client.
- `db` — sqlc output.
- `postgres` — the connection pool.
- `config` — what the server reads from the environment.
- `apitest`, `apitest/full`, `testdb` — test helpers, as above.

`cmd/` holds three programs: `server` serves the API, `backup` writes and
restores a dump, and `make-admin` makes an existing account an admin.

## Generated code

Two things under the API are written by tools and never edited by hand.
`make generate` from the repository root rewrites both.

- `internal/db` is sqlc's output from `migrations/` and
  `internal/db/queries.sql`. Change a query or a migration, then regenerate.
- The site's API types under `web/src/lib/api/shapes/` are tygo's output from
  the `shapes.go` in each feature package, following `tygo.yaml`. Change a
  request or response struct, then regenerate. `make lint` regenerates them
  before it type-checks the site, so a renamed field fails there.
