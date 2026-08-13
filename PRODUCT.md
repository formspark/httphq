# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Two primary users, designed for together rather than in sequence:

- **The mid-debug first-timer.** Arrives from a search result or a colleague's
  link while stuck on a webhook, form post, or third-party integration. Needs a
  working capture URL within seconds of landing, and leaves once they
  understand what was actually sent. Has no prior knowledge of httphq and will
  not read documentation to get started.
- **The returning power user.** Already knows the tool and comes back
  regularly. Values speed, method filtering, full-text search, HAR export, and
  the send-a-test-request panel over any explanation of what httphq is.

Both reach the same surfaces; nothing gates or personalises for either, because
there are no accounts. Self-hosters (Docker, env-var configuration) are a real
audience served by the project, but not the audience the interface is designed
around.

## Product Purpose

httphq generates a unique, disposable HTTP endpoint and shows every request
sent to it in real time: method, path, client IP, headers, query string, and
body: so a developer can see exactly what a client, provider, or device puts
on the wire.

Success is a developer answering "what did it actually send?" in the shortest
possible path from landing to certainty: one click to a URL, point a client at
it, watch the request appear without a refresh.

## Positioning

Three things a neighbouring tool could not truthfully copy in combination:

- **No signup, no friction.** One click produces a working URL. No account, no
  email, no verification, no quota wall.
- **Open source and self-hostable.** MIT licensed, a single Go binary and a
  Docker image, so the same tool can be run against traffic that must not reach
  a third-party service.
- **Honest, unpolluted captures.** Generic forwarding headers (`X-Forwarded-*`,
  `Via`, `Trace*`, `X-Real-Ip`) and the configured platform's vendor headers
  (`Cf-*`, `Fly-*`) are stripped before display, so users inspect their own
  payload rather than the noise of whatever host sits in front of httphq.

Formspark sponsors the project and is credited in the footer and README. That
sponsorship is a fact to preserve, not a positioning claim: httphq is not
positioned as a marketing surface for Formspark.

## Operating Context

- The user is mid-task in another tool: a provider dashboard, a terminal, an
  HTML form, a device, a scheduler: and httphq is the second window they keep
  open beside it.
- The capture URL is pasted into somewhere else entirely (Stripe/GitHub/Slack
  webhook settings, a form `action`, a curl command, firmware config) and then
  the user returns to httphq to watch.
- Capture URLs are routinely shared: anyone holding the URL can read the same
  live stream, which is how two people on opposite sides of an integration
  settle whose payload is malformed.
- Sessions are short and bursty. A page may sit empty and waiting, then receive
  a burst of requests, then be abandoned.

## Capabilities and Constraints

Confirmed functionality:

- Create an endpoint with one POST; the ID is a generated haiku-style slug
  (`lowercase-words-and-digits`, max 64 chars). Capture URL is `/to/<id>`;
  the inspection page is `/<id>`.
- Captured per request: UUID, method, path, query string, body, headers,
  client IP, timestamp.
- Live delivery over WebSocket (`/ws/<id>`), with the scheme tracking the page
  scheme so HTTPS pages use `wss://`.
- Filter by HTTP method; server-side substring search across headers, query
  string, and body.
- Delete a single request or every request for an endpoint.
- Copy a single request or all visible requests as HAR-shaped JSON; copy
  headers or body alone.
- Send a test request to your own endpoint from the page (method, headers,
  body).
- Content-type-aware body rendering: pretty-printed and highlighted JSON,
  multipart/form-data part list, XML highlighting, escaped raw text otherwise.
- Poll the JSON listing with a cursor: echo the response's `cursor` back as
  `?since=` and each capture is handed over exactly once. The endpoint page
  carries a ready-made prompt that hands a coding agent this endpoint's URLs
  and that loop.
- Pages: home (`/`), endpoint (`/<id>`), contact (`/contact`). `/api/health`
  and `/api/debug` exist for operations, not for users.

Technical constraints:

- Retention is 4 hours; a sweep runs at startup and every 5 minutes after. An
  open page drops captures from its own list as they age out.
- Storage is SQLite on the container's writable layer. Capture history is lost
  on restart, by design. No durable store, no migration path.
- Request body limit is 1 MiB.
- The request list returns at most 128 requests: newest first when asked
  without a cursor, oldest first when asked with one, so a poller drains a
  burst in order.
- Rate limit is 150 requests per minute per client IP in production, bucketed
  on the platform-resolved IP.
- Client IP resolution is a trust decision driven by the `PLATFORM` env var;
  setting it trusts that platform's header unconditionally.
- The listen port (8080) is a constant, not configurable.
- Endpoint IDs are generated words, not secrets. There is no authentication and
  no access control on an endpoint; possession of the URL is the only gate, and
  IDs are guessable in principle.
- `robots.txt` allows only `/` and `/contact`; endpoint pages are disallowed.

Terminology: _endpoint_ (the generated capture target), _request_ (one captured
call), _capture URL_ (`/to/<id>`), _HAR_ (the export shape).

Undecided / not established: whether the 4-hour window, the 128-request list
cap, or the 1 MiB body limit should ever be surfaced as configurable to users.

## Brand Commitments

- Name is lowercase `httphq`, always. Canonical host is `httphq.com`.
- Existing marks: `public/logo.svg` (an indigo diamond in the brand accent),
  `public/logo.png`, and a full favicon set (`favicon.ico`, 16/32 PNG,
  `apple-touch-icon.png`, Android Chrome 192/512).
- The footer credits Formspark as sponsor and links the GitHub repository
  (`formspark/httphq`). Both must remain reachable.
- Voice in existing copy is plain, concrete, developer-to-developer, with no
  marketing inflation. Sentence case; no exclamation marks.

## Evidence on Hand

Real: the working product itself (a live capture stream is the demonstration),
the MIT licence, the public GitHub repository and its CI badges, the Formspark
sponsorship, and the logo/favicon assets above.

Absent, and must not be fabricated: testimonials, named customers, usage or
traffic numbers, uptime or performance benchmarks, awards, press coverage,
pricing, team or company claims beyond the Formspark sponsorship, and any
security or compliance certification.

## Product Principles

1. **Nothing stands between landing and a live URL.** Any addition that delays
   or conditions the first capture is working against the product.
2. **Free and anonymous, permanently.** No accounts, no auth, no quotas, no
   paid tier. Design and feature decisions may not assume a logged-in user or a
   future one.
3. **Ephemeral is the product, not a limitation.** Short retention and loss on
   restart are deliberate. Do not design toward archives, history, or durable
   storage.
4. **Show their traffic, not ours.** Infrastructure and vendor noise stays
   hidden; what is displayed should be what the client actually sent.
5. **Serve the first-timer and the regular in one surface.** The fast path must
   stay obvious to someone who has never seen httphq, without slowing down
   someone who uses it weekly.

## Accessibility & Inclusion

No product-specific standard has been established. Nothing in the product's
audience or context relaxes ordinary accessibility expectations for a web tool.
