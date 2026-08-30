package main

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"

	"httphq/src/database"
)

// requestPageSize bounds one listing response. The page renders a window of the
// stream rather than its history, and an endpoint under load would otherwise
// return a payload no reader can use.
const requestPageSize = 128

func handleHealth(c fiber.Ctx) error {
	return c.SendStatus(http.StatusOK)
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

	requests := database.GetRequestsForEndpointID(
		c.Context(), endpointID, c.Query("search"), since, requestPageSize)

	return c.JSON(fiber.Map{
		"requests": requests,
		"total":    database.CountRequestsForEndpointID(c.Context(), endpointID),
		"cursor":   nextCursor(requests, queryStart, since).Format(time.RFC3339Nano),
		// A full page means a burst is still draining, so the caller should
		// come straight back rather than sleeping through the backlog.
		"hasMore": len(requests) == requestPageSize,
	})
}

func handleDeleteRequests(c fiber.Ctx) error {
	database.DeleteRequestsForEndpointID(c.Context(), c.Params("endpoint"))
	return c.SendStatus(http.StatusOK)
}

// handleDeleteRequest deletes one capture by UUID. The endpoint ID is validated
// upstream but the UUID is not scoped to it, so this is a delete-by-key on a
// value the caller already had to read from that endpoint's stream.
func handleDeleteRequest(c fiber.Ctx) error {
	database.DeleteRequestForUUID(c.Context(), c.Params("request"))
	return c.SendStatus(http.StatusOK)
}
