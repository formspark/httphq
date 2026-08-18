package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gofiber/contrib/v3/socketio"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"httphq/src/database"
)

// spoofCurlHeader lets a caller ask for its request to be captured as though a
// command-line client had sent it. A browser adds a dozen headers of its own to
// every fetch; a caller demonstrating a payload wants the payload on screen,
// not the browser's fingerprint around it.
const spoofCurlHeader = "Httphq-Spoof-Curl"

// captureUUIDHeader carries the stored capture's UUID back on the response. It
// is how a caller finds its own request in a stream it shares with everything
// else pointed at the endpoint.
const captureUUIDHeader = "Httphq-Request-Uuid"

// browserOnlyHeaders are the headers a browser adds that no command-line client
// sends. Dropped together with the opt-in header itself when curl is spoofed.
var browserOnlyHeaders = []string{
	"Accept-Encoding",
	"Accept-Language",
	"Connection",
	spoofCurlHeader,
	"Origin",
	"Referer",
	"Sec-Fetch-Dest",
	"Sec-Fetch-Mode",
	"Sec-Fetch-Site",
	"Sec-Ch-Ua",
	"Sec-Ch-Ua-Mobile",
	"Sec-Ch-Ua-Platform",
}

// spoofedCurlHeaders replace the browser's own values, so the capture reads as
// a plain form post from curl.
var spoofedCurlHeaders = map[string][]string{
	"Content-Type": {"application/x-www-form-urlencoded"},
	"User-Agent":   {"curl/7.79.1"},
}

// captureHeaders reduces the request's headers to what the user should see: the
// browser's own additions removed when curl is spoofed, then every
// infrastructure header stripped.
func captureHeaders(headers map[string][]string) map[string][]string {
	if spoof, ok := headers[spoofCurlHeader]; ok && len(spoof) > 0 && spoof[0] == "true" {
		for _, name := range browserOnlyHeaders {
			delete(headers, name)
		}
		for name, value := range spoofedCurlHeaders {
			headers[name] = value
		}
	}
	for name := range headers {
		if omitHeader(name) {
			delete(headers, name)
		}
	}
	return headers
}

// flattenHeaders turns []string header values into a {key: scalar-or-array}
// shape for storage: RFC 7230 allows a header to repeat, and the UI renders a
// single value as a string rather than a one-element list.
func flattenHeaders(headers map[string][]string) map[string]any {
	flat := make(map[string]any, len(headers))
	for name, values := range headers {
		if len(values) == 1 {
			flat[name] = values[0]
		} else {
			flat[name] = values
		}
	}
	return flat
}

// captureRequest stores an inbound request against its endpoint and pushes it to
// every socket watching that endpoint. It answers 200 for anything within the
// body limit: there is nothing else a caller can send that httphq will not
// record.
func captureRequest(registry *socketRegistry) fiber.Handler {
	return func(c fiber.Ctx) error {
		// The body limit is enforced around the handler rather than in front of
		// it. A request declaring more than the limit still arrives here, with
		// its body dropped, and is answered 413 on the way out. Storing it
		// would leave a bodyless capture in the stream, which reads as a sender
		// that sent nothing rather than as a payload that was refused.
		if c.Request().Header.ContentLength() > bodyLimit {
			return c.SendStatus(http.StatusRequestEntityTooLarge)
		}

		endpointID := c.Params("endpoint")

		headers := captureHeaders(c.GetReqHeaders())
		jsonHeaders, err := json.Marshal(flattenHeaders(headers))
		if err != nil {
			slog.ErrorContext(c.Context(), "request header marshal failed", "err", err)
		}

		request := database.Request{
			UUID:        uuid.NewString(),
			EndpointID:  endpointID,
			IP:          resolveClientIP(c),
			Method:      c.Method(),
			Path:        c.Path(),
			QueryString: string(c.Request().URI().QueryString()),
			Body:        string(c.Body()),
			Headers:     datatypes.JSON(jsonHeaders),
		}
		database.CreateRequest(c.Context(), &request)

		// Fan-out to subscribed WS clients. Marshal once, then dispatch
		// asynchronously so a slow client doesn't stall the capture handler.
		if uuids := registry.uuidsFor(endpointID); len(uuids) > 0 {
			marshalled, marshalErr := json.Marshal(request)
			if marshalErr != nil {
				slog.ErrorContext(c.Context(), "websocket payload marshal failed", "err", marshalErr)
			} else {
				go socketio.EmitToList(uuids, marshalled)
			}
		}

		c.Set(captureUUIDHeader, request.UUID)
		return c.SendStatus(http.StatusOK)
	}
}
