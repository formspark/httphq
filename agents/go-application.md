# The Go application

Where each concern lives in `src`, how the process is wired and shut down, and
what the server logs.

## Code layout

`src` is a `main` package split by concern, one file per subject with its tests
beside it: `application.go` (wiring, the process lifecycle and the entry point),
`platform.go` (client IP and header stripping), `endpoint.go` (endpoint IDs and
URLs), `capture.go` (the capture handler), `api.go` (the JSON API and the health
check), `pages.go` (page rendering), `agent.go` (the prompt an endpoint page
hands to a coding agent), `llms.go` (the `/llms.txt` summary, built from the
same constants), `assets.go` (content-hashed asset URLs), `security.go`
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

## Configuration

The runtime variables are in [README.md](../README.md#configuration), because
the people self-hosting httphq read that file.

`APP_VERSION` is a build argument rather than a runtime variable: the build
script and the Dockerfile link it in, release.yml passes the image tag, and
`/api/health` reports it.

The listen port is not configurable: `port` in `src/application.go` is a
constant, and [docs/scripts.md](../docs/scripts.md#the-playwright-suite) says
how to run beside something already holding 8080.

## Shutdown

On SIGTERM the server stops accepting connections, waits up to
`shutdownTimeout` for requests in flight, then stops the retention sweep and
closes the store. A new background task that writes to the store has to stop
before the store closes; see `run` in `src/application.go`. `TestRun` holds the
drain of requests in flight; the order against the store is unenforced:
reviewer's call.

## Client IP is a trust decision

`PLATFORM` selects which header the real client IP is read from, and setting it
trusts that header unconditionally. [README.md](../README.md#platform) owns the
strategies and the security note; `platform_test.go` holds each one.

## Logging

Structured JSON to stdout via `log/slog`, with OpenTelemetry field names
(`service.name`, `http.request.method`, `http.route`, ...). Every request carries
a `request_id` that is reused from a valid inbound `X-Request-Id` or minted,
echoed on the response, and stamped onto every line emitted while handling it.

Only route patterns are logged. Endpoint IDs, raw paths, query strings, headers
and bodies are omitted. Sensitive attributes and arbitrary object values are
masked as a backstop. Error attributes omit their original text. Probe traffic to
`/api/health` logs at debug so it stays out of production logs.
