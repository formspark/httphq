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

## Configuration

httphq is configured entirely through environment variables.

| Variable          | Default       | Description                                                                                                 |
| ----------------- | ------------- | ----------------------------------------------------------------------------------------------------------- |
| `APPLICATION_ENV` | `development` | Set to `production` to bind all interfaces, raise the rate limit, and default logging to `info`.            |
| `LOG_LEVEL`       | env-dependent | Overrides the log level: `debug`, `info`, `warn`, or `error`.                                               |
| `PLATFORM`        | `direct`      | The hosting platform in front of httphq — selects which header the real client IP is read from (see below). |

### `PLATFORM`

httphq derives the client IP (used for rate limiting and shown on captured
requests) from the header your hosting platform sets. Declare your platform
and httphq picks the right strategy:

| `PLATFORM`       | Client IP source                                                        |
| ---------------- | ----------------------------------------------------------------------- |
| unset / `direct` | The TCP connection peer (no proxy).                                     |
| `cloudflare`     | `CF-Connecting-IP` header.                                              |
| `fly`            | `Fly-Client-IP` header.                                                 |
| `heroku`         | `X-Forwarded-For` (leftmost).                                           |
| `render`         | `X-Forwarded-For` (leftmost).                                           |
| `proxy`          | `X-Forwarded-For` (leftmost) — generic nginx / Traefik / load balancer. |

An unrecognised value falls back to `direct`.

Declaring the platform also strips its vendor-specific headers (e.g.
Cloudflare's `Cf-*`) from captured requests, so users inspect their own traffic
without the noise of whatever provider sits in front of httphq.

> **Security note.** Setting `PLATFORM` trusts that platform's client-IP
> header unconditionally — httphq cannot tell a real platform header from one
> a client forged. You must ensure inbound traffic cannot reach httphq
> bypassing the platform (e.g. lock your origin to the platform's IP ranges),
> or a client can spoof its IP and evade rate limiting. Conversely, if you run
> behind a proxy but leave `PLATFORM` unset, every request appears to come
> from the proxy and rate limiting becomes global.

## Logging

httphq logs to stdout as structured JSON via the standard library's `log/slog` — one JSON object per line, with no log files or shipping built in, so any collector can pick the logs up. Field names follow OpenTelemetry conventions (`service.name`, `http.request.method`, `url.path`, `http.response.status_code`, ...). Every request gets a correlation `request_id` (a valid inbound `X-Request-Id` is reused, otherwise one is minted) that is echoed back on the response header and stamped onto every log line emitted while handling that request. Each request produces one access-log line; headers and bodies are never logged, paths are logged without their query string, and a denylist masks sensitive keys as a backstop. The level defaults to `info` in production (`debug` elsewhere) and is overridable with `LOG_LEVEL`; Kubernetes probe traffic to `/api/health` logs at `debug` so it stays out of production logs.

## License

[MIT](https://opensource.org/licenses/MIT)
