package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
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

// curlSpoofRequested reports whether the sender asked for its request to be
// captured as a command-line client's rather than the browser's.
func curlSpoofRequested(headers map[string][]string) bool {
	spoof := headers[spoofCurlHeader]
	return len(spoof) > 0 && spoof[0] == "true"
}

// spoofCurl rewrites a browser's headers so the capture reads as a command-line
// client's: the additions only a browser makes are dropped, and the values curl
// would have sent replace the ones it shares.
func spoofCurl(headers map[string][]string) {
	for _, name := range browserOnlyHeaders {
		delete(headers, name)
	}
	maps.Copy(headers, spoofedCurlHeaders)
}

// captureHeaders reduces the request's headers to what the user should see: the
// browser's own additions removed when curl is spoofed, then every
// infrastructure header stripped.
func captureHeaders(headers map[string][]string) map[string][]string {
	if curlSpoofRequested(headers) {
		spoofCurl(headers)
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

// broadcastCapture offers a stored capture to every socket watching its
// endpoint. It marshals once and never waits on a socket, so a slow client
// cannot stall the capture handler or pin captures in memory.
func broadcastCapture(ctx context.Context, registry *socketRegistry, request database.Request) {
	feeds := registry.subscribersFor(request.EndpointID)
	if len(feeds) == 0 {
		return
	}
	marshalled, err := json.Marshal(request)
	if err != nil {
		slog.ErrorContext(ctx, "websocket payload marshal failed", "err", err)
		return
	}
	for _, feed := range feeds {
		feed.offer(marshalled)
	}
}

// zstdDecoder decodes zstd capture bodies within the body limit. The
// framework's own decoder accepts whatever window a frame declares and
// allocates it up front, so a body of a few bytes could claim hundreds of
// megabytes. This one refuses a window, or an output, larger than the limit.
var zstdDecoder, _ = zstd.NewReader(nil,
	zstd.WithDecoderMaxWindow(bodyLimit),
	zstd.WithDecoderMaxMemory(bodyLimit),
	zstd.WithDecoderConcurrency(1))

// captureBody is the request body as the sender meant it, decoded from at most
// one Content-Encoding. Each decoding stage is bounded by the body limit but
// the number of stages is not, so a chain of them is refused outright: no
// client needs to compress a payload twice.
func captureBody(c fiber.Ctx) ([]byte, error) {
	encoding := strings.TrimSpace(c.Get(fiber.HeaderContentEncoding))
	if strings.Contains(encoding, ",") {
		return nil, fiber.ErrUnsupportedMediaType
	}
	if strings.EqualFold(encoding, "zstd") {
		return decodeZstd(c.Request().Body())
	}
	return c.Body(), nil
}

// decodeZstd decodes a zstd body. A frame that would need more than the body
// limit to decode is answered 413, as an oversized body is; one that is not
// valid zstd is answered 400.
func decodeZstd(body []byte) ([]byte, error) {
	decoded, err := zstdDecoder.DecodeAll(body, nil)
	switch {
	case err == nil:
		return decoded, nil
	case errors.Is(err, zstd.ErrWindowSizeExceeded), errors.Is(err, zstd.ErrDecoderSizeExceeded):
		return nil, fiber.ErrRequestEntityTooLarge
	default:
		return nil, fiber.ErrBadRequest
	}
}

// captureRecord builds the stored capture from an inbound request and its
// decoded body.
func captureRecord(c fiber.Ctx, body []byte) database.Request {
	headers := captureHeaders(c.GetReqHeaders())
	jsonHeaders, err := json.Marshal(flattenHeaders(headers))
	if err != nil {
		slog.ErrorContext(c.Context(), "request header marshal failed", "err", err)
	}
	return database.Request{
		UUID:        uuid.NewString(),
		EndpointID:  c.Params("endpoint"),
		IP:          resolveClientIP(c),
		Method:      c.Method(),
		Path:        c.Path(),
		QueryString: string(c.Request().URI().QueryString()),
		Body:        string(body),
		Headers:     datatypes.JSON(jsonHeaders),
	}
}

// captureRequest stores an inbound request against its endpoint and pushes it to
// every socket watching that endpoint. It answers 200 once the capture is
// stored, so a sender is never handed the UUID of a capture that does not
// exist; a store that refused the write is answered 503.
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

		body, err := captureBody(c)
		if err != nil {
			return err
		}
		request := captureRecord(c, body)
		if err := database.CreateRequest(c.Context(), &request); err != nil {
			return fiber.ErrServiceUnavailable
		}

		broadcastCapture(c.Context(), registry, request)

		c.Set(captureUUIDHeader, request.UUID)
		return c.SendStatus(http.StatusOK)
	}
}
