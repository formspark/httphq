<p align="center">
   <img width="64" src="public/logo.svg" alt="httphq logo">
</p>

<h1 align="center">httphq</h1>

<p align="center">
    https://httphq.com
</p>

<p align="center">
    Generate custom endpoints to capture and inspect HTTP requests.
</p>

<p align="center">
    Sponsored by <a href="https://formspark.io">Formspark</a>, the simple & powerful form solution for developers.
</p>

## CI

[![test](https://github.com/formspark/httphq/actions/workflows/test.yml/badge.svg)](https://github.com/formspark/httphq/actions/workflows/test.yml) [![release](https://github.com/formspark/httphq/actions/workflows/release.yml/badge.svg)](https://github.com/formspark/httphq/actions/workflows/release.yml)

## Docs

[Scripts](docs/scripts.md)

## Logging

httphq logs to stdout as structured JSON via the standard library's `log/slog` — one JSON object per line, with no log files or shipping built in, so any collector can pick the logs up. Field names follow OpenTelemetry conventions (`service.name`, `http.request.method`, `url.path`, `http.response.status_code`, ...). Every request gets a correlation `request_id` (a valid inbound `X-Request-Id` is reused, otherwise one is minted) that is echoed back on the response header and stamped onto every log line emitted while handling that request. Each request produces one access-log line; headers and bodies are never logged, paths are logged without their query string, and a denylist masks sensitive keys as a backstop. The level defaults to `info` in production (`debug` elsewhere) and is overridable with `LOG_LEVEL`; Kubernetes probe traffic to `/api/health` logs at `debug` so it stays out of production logs.

## License

[MIT](https://opensource.org/licenses/MIT)
