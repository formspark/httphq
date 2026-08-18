# API

httphq's JSON API needs no account, no key and no create step. An endpoint
exists as soon as something addresses it, so the only thing a caller needs is an
endpoint ID.

Every URL below is relative to the host serving the page. A self-hosted
deployment answers on its own host.

## Routes

| Method   | Path                                         | Purpose                       |
| -------- | -------------------------------------------- | ----------------------------- |
| `ANY`    | `/to/:endpoint`                              | Capture a request             |
| `GET`    | `/api/endpoints/:endpoint/requests`          | List captures                 |
| `DELETE` | `/api/endpoints/:endpoint/requests`          | Delete an endpoint's captures |
| `DELETE` | `/api/endpoints/:endpoint/requests/:request` | Delete one capture by UUID    |

An endpoint ID is lowercase words and digits joined by hyphens, up to 64
characters. Anything else is a 404.

## Capturing

Send anything at all to `/to/:endpoint`. Every method is accepted, anything
within the body limit below answers `200`, and the capture's UUID comes back on
the `Httphq-Request-Uuid` header.

```bash
curl -X POST -d '{"hello":"world"}' https://httphq.com/to/purple-frog-0691
```

## Listing

```
GET /api/endpoints/:endpoint/requests?search=&since=
```

| Parameter | Notes                                                             |
| --------- | ----------------------------------------------------------------- |
| `search`  | Optional. Substring match across headers, query string and body.  |
| `since`   | Optional RFC 3339 timestamp. Returns only captures newer than it. |

<!-- prettier-ignore -->
```jsonc
{
  "requests": [ /* newest 128, or the oldest 128 newer than `since` */ ],
  "total": 42,                                  // the endpoint's, ignoring search and since
  "cursor": "2026-08-09T07:20:05.559121660Z",
  "hasMore": false
}
```

`total` is what the endpoint holds regardless of `search`, `since` or the page
size, so a caller can say what a delete-all would affect.

A malformed `since` is a `400` rather than a silently ignored parameter: a
caller that mistyped one would otherwise be handed its whole history back and
have no way to tell.

## Polling with the cursor

Echo `cursor` back as `since` and you are handed each capture exactly once.

1. Call the listing with no `since`. Read `requests` and `cursor`.
2. Call again with `?since=<cursor>` from the previous response.
3. If `hasMore` is true, call again immediately: a burst is still draining.
   Otherwise wait 2 seconds.

```bash
curl -s "https://httphq.com/api/endpoints/purple-frog-0691/requests?since=2026-08-09T07:20:05Z"
```

Three things are worth knowing:

- **The cursor is opaque.** It is a timestamp today. Echo it back rather than
  parsing one or building your own, or a change of format will break you.
- **It is server time.** Substituting your own clock reintroduces the skew the
  cursor exists to remove.
- **With `since`, captures come back oldest first**, and without it, newest
  first. A cursored caller is draining a stream in order; an uncursored one is
  looking at the latest activity.

An endpoint with no traffic still returns a cursor, so a poller that starts
before the first request has something to advance from.

## Limits

- **128 captures** per listing response.
- **1 MiB** request body. Anything larger is answered `413` and stored
  nowhere, so an oversized payload leaves no capture behind.
- **150 requests per minute per client IP** in production, across everything:
  page loads, captures and API calls share one budget. The recommended
  2 second poll is 30 a minute, which leaves room for the traffic under test.
- **4 hour retention.** Captures are deleted after that, and on restart.

Anyone holding an endpoint's URL can read everything sent to it. Nothing secret
should go through one.
