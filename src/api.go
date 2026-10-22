package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"

	"httphq/src/database"
)

// requestPageSize bounds one listing response. The page renders a window of the
// stream rather than its history, and an endpoint under load would otherwise
// return a payload no reader can use.
const requestPageSize = 128

// listingByteBudget bounds the encoded size of one listing response.
// requestPageSize counts rows, and one row can carry a body of up to bodyLimit
// that JSON escaping grows several times over, so a page of large captures
// would otherwise be built whole in memory on every poll. A page stops at the
// budget and reports more to come; it always holds at least one capture, so a
// cursored poller can never stall behind a capture larger than the budget.
const listingByteBudget = 4 << 20

// version is the build the binary came from, set at link time from APP_VERSION
// (see the build script and the Dockerfile). A binary built without it reports
// "dev".
var version = "dev"

// handleHealth is the platform's probe. It touches nothing, so it answers while
// the store is busy, and it names the build so a deploy can be confirmed from
// outside.
func handleHealth(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"ok": true, "version": version})
}

// handleDebug reports coarse process state. It carries no captured data: the
// counts say how much traffic the process is holding, never what was in it.
func handleDebug(registry *socketRegistry) fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"host":         string(c.Request().Host()),
			"isProduction": isProduction,
			"requests":     database.CountRequests(c.Context()),
			"sockets":      registry.count(),
		})
	}
}

// newestCreatedAt is the latest capture time in a page. It scans rather than
// reading an end of the slice: the listing orders newest-first or oldest-first
// depending on whether it was given a cursor, so neither end is reliably the
// newest.
func newestCreatedAt(requests []database.Request) time.Time {
	newest := requests[0].CreatedAt
	for _, request := range requests[1:] {
		if request.CreatedAt.After(newest) {
			newest = request.CreatedAt
		}
	}
	return newest
}

// nextCursor is what a caller echoes back on its next call.
//
// It is the newest capture in the page, or the moment the query ran when the
// page came back empty, so a poller that starts before any traffic still has
// something to advance from rather than re-asking for the same empty window
// forever. It never moves backwards: a cursor already ahead of both survives
// the round trip, or the caller is handed captures it has already processed.
func nextCursor(requests []database.Request, queryStart, since time.Time) time.Time {
	cursor := queryStart
	if len(requests) > 0 {
		cursor = newestCreatedAt(requests)
	}
	if cursor.Before(since) {
		return since
	}
	return cursor
}

// withinBudget is the longest prefix of a page whose encoded captures fit the
// listing budget, and never less than the first capture.
func withinBudget(requests []database.Request) []database.Request {
	used := 0
	for i, request := range requests {
		used += encodedSize(request)
		if i > 0 && used > listingByteBudget {
			return requests[:i]
		}
	}
	return requests
}

// encodedSize is how many bytes a capture adds to a listing. A capture that
// will not encode adds nothing here; the response encoder reports it.
func encodedSize(request database.Request) int {
	encoded, _ := json.Marshal(request)
	return len(encoded)
}

// handleListRequests returns a window of an endpoint's captures, narrowed by
// the search and by an optional `since` cursor. `total` is what the endpoint
// holds regardless of search, cursor or window, so the page can say what a
// control acting on the whole endpoint will affect rather than reporting the
// size of the current view.
//
// `cursor` is what makes the listing usable by a poller: echoing it back is a
// promise that no capture is handed over twice and none is skipped. It is
// opaque on purpose. It happens to be a timestamp, but a caller that parses or
// synthesises one instead of echoing it can be broken by a change of format.
func handleListRequests(c fiber.Ctx) error {
	endpointID := c.Params("endpoint")

	// Read before the query. A capture landing while the query runs then falls
	// inside the next cursor window rather than into the gap between the two.
	queryStart := time.Now().UTC()

	var since time.Time
	if raw := c.Query("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			// Rejected rather than ignored. Falling back to the full window
			// would make a poller reprocess its whole history while looking
			// like a successful call.
			return c.Status(http.StatusBadRequest).
				JSON(fiber.Map{"error": "since must be an RFC 3339 timestamp"})
		}
		since = parsed.UTC()
	}

	stored := database.GetRequestsForEndpointID(
		c.Context(), endpointID, c.Query("search"), since, requestPageSize)
	requests := withinBudget(stored)

	return c.JSON(fiber.Map{
		"requests": requests,
		"total":    database.CountRequestsForEndpointID(c.Context(), endpointID),
		"cursor":   nextCursor(requests, queryStart, since).Format(time.RFC3339Nano),
		// A full page, or one cut short by the byte budget, means a burst is
		// still draining, so the caller should come straight back rather than
		// sleeping through the backlog.
		"hasMore": len(stored) == requestPageSize || len(requests) < len(stored),
	})
}

// handleDeleteRequests deletes every capture an endpoint holds. A delete the
// store did not carry out is answered 503, so the page does not tell a viewer
// their captures are gone while anyone with the URL can still read them.
func handleDeleteRequests(c fiber.Ctx) error {
	if err := database.DeleteRequestsForEndpointID(c.Context(), c.Params("endpoint")); err != nil {
		return fiber.ErrServiceUnavailable
	}
	return c.SendStatus(http.StatusOK)
}

// handleDeleteRequest deletes one capture by UUID. The endpoint ID is validated
// upstream but the UUID is not scoped to it, so this is a delete-by-key on a
// value the caller already had to read from that endpoint's stream.
func handleDeleteRequest(c fiber.Ctx) error {
	if err := database.DeleteRequestForUUID(c.Context(), c.Params("request")); err != nil {
		return fiber.ErrServiceUnavailable
	}
	return c.SendStatus(http.StatusOK)
}
