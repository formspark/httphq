# AGENTS.md

Guidance for agents and contributors working in this repository. README.md says
what httphq is and how to configure it, docs/api.md is the JSON API,
docs/scripts.md is every command, DESIGN.md is the visual system and PRODUCT.md
is the product. This file is the constraints that are not obvious from the
source.

## `pnpm verify` is the definition of done

It runs, in order: govulncheck, gofmt, golangci-lint and gocyclo over the Go
code, the Go tests with the race detector and coverage, ESLint, Prettier, the
lint-staged glob check, the prose and spelling checks, the advisory check, the
type check, the stylesheet freshness check, the production build and the
Playwright suite. A
change that has not passed it is not finished. CI runs the same package.json
scripts, split into jobs in `pipeline.yml`, and also builds the image.

`check:audit` is `pnpm audit --prod --audit-level high`, over the Node
periphery. The Go side has govulncheck, which gates on reachability rather
than presence, and is the stricter of the two: it is why `gofiber/fiber` and
`golang.org/x/crypto` carry advisories at module level while verify stays
green. The npm scan has no such filter, so it is scoped to what ships and to
what is worth stopping for. An advisory that has no fix, or one that cannot be
reached from this code, goes in `auditConfig.ignoreCves` with the reason
beside it.

The pipeline only answers whether a change introduces an advisory. `audit.yml`
asks the other question once a week, against whatever is on master, because a
repository nobody pushes to runs nothing. The nightly `test.yml` already
covers the Go side the same way.

The spelling check runs typos through `uvx`, so it needs uv, and `check:audit`
reaches the registry. The Playwright
suite needs Chromium once (docs/scripts.md) and loads Alpine and the syntax
highlighter from public CDNs, so a screen that never hydrates is a network
failure before it is a code one: run it again before debugging it.

## Comments say what the code does now

A comment explains why, and names the constraint the next person has to
respect: "excluded by default so new surfaces are safe". Write it so it still
makes sense in a year.

- No history: not what the code used to do, not the bug that prompted it. That
  belongs in the commit message, where it is attached to the change.
- No ticket or issue references.
- A regression risk is worth stating as a rule rather than a story: "a header
  set on the way in is discarded by a handler that resets the response".
- No hyper-specific framing: describe the generic, reusable purpose rather than
  the one-time scenario, especially in shared helpers and test fixtures.

## Commits say what the change does

- The subject is a sentence in the imperative that says what the change does:
  "Refuse an oversized body instead of storing a bodyless capture", not
  `fix: refuse oversized body`. No conventional-commit prefixes.
- Pull requests are squash-merged, with the pull request number as a `(#N)`
  suffix on the subject.
- The body carries facts: why, what was measured, what was checked and how,
  and what was deliberately left out. Not a summary of the diff, which the
  reader already has. The comment rules above govern commit bodies and pull
  request descriptions too.
- No assistant attribution of any kind: no co-author trailer, no session link,
  no generated-with footer.
- One concern per commit. The pre-commit hook formats and lints what is
  staged, Go files included; do not skip it with `--no-verify`.

## Writing

No em dashes in prose: docs, comments, UI copy and commit messages. Rewrite the
clause instead. A paired aside becomes commas or parentheses; a dash that
introduces an explanation becomes a colon or a new sentence. `check:prose`
fails on one in any tracked file.

User-facing copy names the format or the action, never a consumer such as
agents or LLMs: `Copy request`, `Copy shown (N)`. The constraint is on product
copy only; commit messages and docs may mention agents freely.

## Configuration

The runtime variables are in README.md under Configuration, in a
`Variable | Required | Default | Secret | Notes` table, because the people
self-hosting httphq read that file. None is required and none is a secret.

`APP_VERSION` is a build argument rather than a runtime variable: the build
script and the Dockerfile link it in, release.yml passes the image tag, and
`/api/health` reports it.

The listen port is not configurable. `port` in `src/application.go` is a
constant, so anything that needs two instances, or to run beside something
already holding 8080, changes the constant.

## Tests

- Go: a test file covers one subject, is named after it, and names each suite
  after what it exercises, so a handler's tests are found beside the handler.
  One `TestSubject` per function or type, with sentence-style `t.Run` names,
  `t.Helper` in helpers, and `-race`. `harness_test.go` is not a subject: it
  holds `TestMain` and the fixtures shared by every test that drives a real
  request.
- Playwright: see "The Playwright suite covers screens and page scripts
  separately" below.
- Fix every instance of a defect, not only the one that was reported.
- A regression test has to be seen failing without the fix before it is
  trusted. Take the fix out, watch the test go red, put it back.

## Lint is clean, types are strict, and the caps only go down

- golangci-lint runs its default set, which is `go vet` plus the checks vet
  does not cover: unchecked error returns, dead assignments, unused code, and
  staticcheck's correctness rules. `errcheck` is relaxed inside `_test.go`: a
  test that ignores an error is usually asserting the value beside it.
- ESLint runs with `--max-warnings 0`. The Playwright suite is linted with type
  information: no type assertions other than `as const`, no non-null
  assertions, no `any`, and no `@ts-` comments of any kind. A value that
  arrives untyped, a JSON body or the clipboard, is parsed with a zod schema in
  `tests/support/harness.ts`.
- No `eslint-disable` comments: inline config is switched off, so one has no
  effect and is reported. A rule that cannot hold for a file gets an entry in
  `eslint.config.mjs`, scoped to that file and that rule, with the reason
  beside it.
- Both tsconfigs add `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`,
  `noImplicitOverride` and `noFallthroughCasesInSwitch` to `strict`.
- `gocyclo -over 4` for Go, and `complexity`, `max-params` and `max-depth` in
  ESLint, are capped at the worst function that exists. They are ratchets: when
  a new function trips one, split the function rather than raising the cap, and
  lower a cap when its worst offender is split.
- `check:lint-staged` fails when the tree holds a file extension Prettier can
  format that no lint-staged pattern matches. Prettier decides what counts, so
  the two cannot drift, and a gap would otherwise be silent: CI formats the
  file, the hook never touches it.

## Reading CI

- `pipeline.yml` is the single definition of what verified means, and both
  `test.yml` (pull requests and a nightly run) and `release.yml` (pushes to
  master) call it. A check cannot exist on pull requests yet be skipped on the
  push that publishes the image. Adding a check means adding a script to
  package.json, to `verify`, and to `pipeline.yml`, never to one of the two
  callers.
- The nightly run exists because the page scripts pull Alpine and highlight.js
  from public CDNs at runtime, so the suite can fail on a day nobody pushed.
- Read the checks table, not a watcher's exit code:
  `gh pr checks <n> --watch --fail-fast` can exit 0 with a failed run in the
  table.
- `gh pr merge --auto` does not wait here. Auto-merge is disabled on this
  repository, so the flag is accepted and the merge happens immediately.
- A cancelled job is the cap, not a regression. Check with
  `gh api repos/{owner}/{repo}/actions/jobs/<id> --jq '.steps[]'` before treating
  a timeout as a code problem.
- Coverage is printed to the job summary, not gated. A threshold set before the
  number is known is a guess.

## Traps

- The pins move together: `nodejs` and `pnpm` in `.tool-versions`,
  `packageManager` and `engines` in package.json, the `go` line in go.mod (the
  Dockerfile's `GOTOOLCHAIN=auto` fetches it), and the Go tool versions in the
  package.json scripts. CI reads `.tool-versions` and go.mod.
- Every Go command is scoped to `./src`: `./...` also walks `node_modules`, which
  ships a stray Go package once the npm tooling is installed.
- `public/app.css` is generated and committed, so the binary and the container
  need no Node. A template or script that gains a class needs `pnpm run css`
  and the rebuilt sheet in the same commit.
- On SIGTERM the server stops accepting connections, waits up to
  `shutdownTimeout` for requests in flight, then stops the retention sweep and
  closes the store. A new background task that writes to the store has to stop
  before the store closes; see `run` in `src/application.go`.

## Shared files

These files are kept identical, or in step, with the same files in the
repositories that deploy to the lab (lab, 8ctave, geobear, postcraft,
metric-tone and bjornkrols.com). Change them in all of them together.

- `scripts/check-prose.mjs`, `scripts/check-lint-staged.mjs` and
  `.husky/install.mjs`: identical in every one of them.
- `src/logging`: in step with `apps/hooks/logging.go` in the lab repository,
  which carries the same handler, redaction list and level rules.

## Code layout

`src` is a `main` package split by concern, one file per subject with its tests
beside it: `application.go` (wiring, the process lifecycle and the entry point),
`platform.go` (client IP and header stripping), `endpoint.go` (endpoint IDs and
URLs), `capture.go` (the capture handler), `api.go` (the JSON API and the health
check), `pages.go` (page rendering), `agent.go` (the prompt an endpoint page
hands to a coding agent), `assets.go` (content-hashed asset URLs), `security.go`
(CSP and security headers), `sockets.go` (the live feed), `requestlog.go`
(correlation IDs and the access log).

Two subpackages sit beneath it: `database` (the SQLite connection) and
`logging` (the slog handler and its sensitive-key redaction). `styles` and
`views` hold CSS and HTML templates rather than Go.

`newApplication` builds the entire routing surface from arguments, so tests
drive real requests through it without a listening socket. Anything that pulls
configuration out of the environment belongs in `main`, not in a handler. `run`
is everything `main` does after reading the environment and binding the port,
so `TestRun` covers the lifecycle on a loopback port.

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
`tests/support/harness.ts`, along with the zod schemas that parse what the
suite reads at the `JSON.parse` and `response.json()` boundaries. The HAR
schemas are strict, because the export is what those tests are about; the
listing schema drops keys it does not know, so a field the API gains does not
fail an unrelated test. The helpers the page scripts publish on `window` are
declared once, in `types/page-scripts.d.ts`, and the suite reads that
declaration rather than keeping its own.

Screens are reached through `getByTestId`. `testIdAttribute` in
`playwright.config.ts` points it at the `data-test` attribute the templates
carry, so a test names a hook and never spells an attribute selector. An element
a test reaches for gets a `data-test`; anything a reader can identify by its
role or its text is reached that way instead.

## Client IP is a trust decision

`PLATFORM` selects which header the real client IP is read from, and setting it
trusts that header unconditionally: httphq cannot tell a platform's header from
one a client forged. Inbound traffic must not be able to reach the process
bypassing that platform, or a client can spoof its IP and evade rate limiting.
Leaving it unset behind a proxy is the opposite failure: every request looks
like it came from the proxy and rate limiting becomes global.

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
