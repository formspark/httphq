# AGENTS.md

Guidance for agents and contributors working in this repository.

## Code layout

`src` is a `main` package split by concern, one file per subject with its
tests beside it: `application.go` (wiring and entry point), `platform.go`
(client IP and header stripping), `endpoint.go` (endpoint IDs and URLs),
`capture.go` (the capture handler), `api.go` (the JSON API), `pages.go` (page
rendering), `agent.go` (the prompt an endpoint page hands to a coding agent),
`assets.go` (content-hashed asset URLs), `security.go` (CSP and security
headers), `sockets.go` (the live feed), `requestlog.go` (correlation IDs and
the access log).

Two subpackages sit beneath it: `database` (the SQLite connection) and
`logging` (the slog handler and its sensitive-key redaction). `styles` and
`views` hold CSS and HTML templates rather than Go.

A test file covers one subject, is named after it, and names each suite after
what it exercises, so a handler's tests are found beside the handler. The one
exception is `harness_test.go`, which is not a subject: it holds `TestMain` and
the fixtures shared by every test that drives a real request.

`newApplication` builds the entire routing surface from arguments, so tests
drive real requests through it without a listening socket. Anything that pulls
configuration out of the environment belongs in `main`, not in a handler.

The live feed is the exception. A WebSocket upgrade needs a real connection
underneath it, so `sockets_test.go` starts an application of its own on a
loopback port and dials it. Everything else stays on the in-memory transport.

## The Playwright suite covers screens and page scripts separately

`e2e/tests` holds one spec per subject. A `*-screen.spec.ts` drives a screen
through captured traffic and asserts what a reader ends up seeing.
`render-body.spec.ts`, `page-helpers.spec.ts` and `har-export.spec.ts` drive the
browser scripts directly through `loadPageScripts`, because traffic cannot
express everything those scripts handle: the server parses and re-serialises a
multipart body before storing it, so several part shapes a client can put on the
wire never reach the page intact, and neither a repeated header nor a body that
is not ASCII survives the round trip through the test client.
`capture-api.spec.ts` covers what only a client on the wire can observe, such as
the body limit the server enforces around the handler rather than inside it.

A spec's own fixtures sit at the top of that spec, as the locator and multipart
helpers do. Fixtures used by more than one spec live in
`tests/support/harness.ts`, which is also the one place a type is asserted
rather than proven, at the `JSON.parse` and `response.json()` boundaries.

Screens are reached through `getByTestId`. `testIdAttribute` in
`playwright.config.ts` points it at the `data-test` attribute the templates
carry, so a test names a hook and never spells an attribute selector. An element
a test reaches for gets a `data-test`; anything a reader can identify by its
role or its text is reached that way instead.

## Comments

Comments describe what the code does now and warn about non-obvious constraints
or regressions. They are not a changelog, bug tracker, or ticket index. Write
them so they still make sense in a year.

- No ticket or PR references.
- No bug war stories: don't narrate a specific bug and how it was fixed ("this
  fixes the flicker when..."). Once fixed, that history is noise.
- No hyper-specific framing: don't tie comments to one-time scenarios; describe
  the generic, reusable purpose instead.
- Explain intent and non-obvious behaviour: why this branch exists, what
  invariant it protects.
- Flag regression risks the next dev must respect (e.g. "excluded by default so
  new surfaces are safe").
- Keep comments generic and reusable, especially in shared helpers and test
  fixtures.

## User-facing copy names the format, not the consumer

Button labels, tooltips and empty-state copy name the format or the action
(`Copy request`, `Copy shown (N)`), never a consumer such as agents or LLMs. The
constraint is on product copy only; commit messages and docs may mention agents
freely.

## Commits are squashed, and the subject carries the pull request number

Squash-merge, with the pull request number as a `(#N)` suffix on the subject.
Subjects are sentences that say what the change does, not conventional-commit
prefixes: "Refuse an oversized body instead of storing a bodyless capture", not
`fix: refuse oversized body`. The comment rules above govern commit bodies and
pull request descriptions too, so a paragraph carries a fact rather than a
summary of the diff.

No assistant attribution of any kind: no co-author trailer, no session link, no
generated-with footer.

## Every check runs on both paths

`pipeline.yml` is the single definition of what verified means, and both
`test.yml` and `release.yml` call it. A check cannot exist on pull requests yet
be skipped on the push that publishes the image. Adding a check means adding it
to `pipeline.yml`, never to one of the two callers.

The suite that gates a release is therefore the full one: `gofmt`,
`golangci-lint`,
`gocyclo`, the Go tests with coverage, ESLint, Prettier, the Playwright suite,
the stylesheet freshness check, and the production build with the flags the
image uses.

Coverage is printed, not gated. A threshold set before the number is known is a
guess, so the figure goes to the job summary and a ceiling can be set later at
what the suite actually reaches.

## Lint-staged globs are checked against Prettier

`scripts/check-lint-staged.mjs` fails when the tree contains a file extension
Prettier can format that no lint-staged pattern matches. Prettier decides what
counts rather than a list kept anywhere, so the two cannot drift.

The gap it exists for is silent: CI lints and formats the file, the pre-commit
hook never touches it. It has already been found by hand twice, months apart, in
two different repositories.

## Reading CI

- Read the checks table, not a watcher's exit code.
  `gh pr checks <n> --watch --fail-fast` has exited 0 with a failed run sitting
  in the table.
- `gh pr merge --auto` does not wait here. Auto-merge is disabled on this
  repository, so the flag is accepted and the merge happens immediately.
- A cancelled job is the cap, not a regression. Check with
  `gh api repos/{owner}/{repo}/actions/jobs/<id> --jq '.steps[]'` before treating
  a timeout as a code problem.

## Client IP is a trust decision

`PLATFORM` selects which header the real client IP is read from, and setting it
trusts that header unconditionally: httphq cannot tell a platform's header from
one a client forged. Inbound traffic must not be able to reach the process
bypassing that platform, or a client can spoof its IP and evade rate limiting.
Leaving it unset behind a proxy is the opposite failure: every request looks
like it came from the proxy and rate limiting becomes global.

## Go linting

`golangci-lint` at its default set, which is `go vet` plus the checks vet does
not cover: unchecked error returns, dead assignments, unused code, and
staticcheck's correctness rules. It replaces the bare `go vet` step.

Pinned in the workflow for the same reason `gocyclo` is: a later release must
not change the verdict on a tree that did not change.

`errcheck` is relaxed inside `_test.go`. A test that ignores an error is usually
asserting the value beside it, and a `t.Fatal` on every teardown call reads worse
than it protects.

The complexity ceiling stays separate. `gocyclo -over 4` is a ratchet rather
than a correctness check, and it is documented alongside the ESLint one.

## Database standards

`TestDatabaseStandards` in `src/database` asks the schema AutoMigrate actually
produces, rather than the struct tags that ask for it: snake_case tables and
columns, a `created_at`, and an index on every column a query filters or orders
by. A `gorm:"column:..."` tag naming something else satisfies the struct and
fails the test, which is the point.

The sibling repositories run the same questions as SQL against Postgres after
applying their migrations. There are no migration files here and no
information_schema to query, so these go through `sqlite_schema` and the pragmas.

Two of their checks do not apply. Nothing carries `updated_at`, because a
capture is written once and never edited, and there is no email column anywhere.

## Captured data is ephemeral

SQLite writes to the container's writable layer. Capture history is lost on
restart, by design. Nothing here is a durable store, and no migration path
exists for it.

## Logging

Structured JSON to stdout via `log/slog`, with OpenTelemetry field names
(`service.name`, `http.request.method`, `url.path`, ...). Every request carries
a `request_id` that is reused from a valid inbound `X-Request-Id` or minted,
echoed on the response, and stamped onto every line emitted while handling it.

Headers and bodies are never logged and paths are logged without their query
string; a denylist masks sensitive keys as a backstop. Probe traffic to
`/api/health` logs at debug so it stays out of production logs.

## The listen port is a constant

`port` in `src/application.go` is not configurable. Anything that needs to run
two instances, or to run alongside something already holding 8080, has to
change the constant.
