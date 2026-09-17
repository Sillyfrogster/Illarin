# API layout

The Go code under `internal/` has one package for each thing the product has:
`account`, `profile`, `notify` and so on. The package name says what the code
is about, never what kind of code it is.

A feature's handlers live in its package, next to the request and response
structs they read and write. Each feature package registers its own routes on
gin with a `Register` function, and gives each route a deadline as it does.

Plumbing that every feature needs lives in `internal/api`: the session and the
signed-in account, cookies, the check that a change came from the site, reading
path and query values, the error body, and route deadlines. A feature package
imports `api`; `api` imports no feature.

When a package grows past about fifteen files, split it by feature into smaller
packages. Never split it by layer, so there is no `handlers` or `models`
package.

Tests that send real requests import `internal/apitest` for the shared helpers,
and `internal/apitest/full` for a router that serves every route.
