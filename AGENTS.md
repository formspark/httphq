# AGENTS.md

Guidance for agents and contributors working in this repository.

## Code layout

`src` is one `main` package split by concern, one file per subject with its
tests beside it: `application.go` (wiring and entry point), `platform.go`
(client IP and header stripping), `endpoint.go` (endpoint IDs and URLs),
`capture.go` (the capture handler), `api.go` (the JSON API), `pages.go` (page
rendering), `agent.go` (the prompt an endpoint page hands to a coding agent),
`assets.go` (content-hashed asset URLs), `security.go` (CSP and security
headers), `sockets.go` (the live feed), `requestlog.go` (correlation IDs and
the access log).

A test file covers one subject, is named after it, and names each suite after
what it exercises, so a handler's tests are found beside the handler. The one
exception is `harness_test.go`, which is not a subject: it holds `TestMain` and
the fixtures shared by every test that drives a real request.

`newApplication` builds the entire routing surface from arguments, so tests
drive real requests through it without a listening socket. Anything that pulls
configuration out of the environment belongs in `main`, not in a handler.

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

## Client IP is a trust decision

`PLATFORM` selects which header the real client IP is read from, and setting it
trusts that header unconditionally: httphq cannot tell a platform's header from
one a client forged. Inbound traffic must not be able to reach the process
bypassing that platform, or a client can spoof its IP and evade rate limiting.
Leaving it unset behind a proxy is the opposite failure: every request looks
like it came from the proxy and rate limiting becomes global.

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
