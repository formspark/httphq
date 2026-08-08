package main

import (
	"net/http"
	"regexp"

	"github.com/gofiber/fiber/v3"
)

// endpointIDPattern bounds an endpoint ID to the shape haikunator emits
// (lowercase words and digits joined by hyphens). Rejecting anything else
// keeps attacker-controlled characters — quotes, angle brackets, parens —
// out of the rendered pages, the database, and the logs.
var endpointIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func validEndpointID(id string) bool {
	return len(id) <= 64 && endpointIDPattern.MatchString(id)
}

// requireValidEndpoint guards every route carrying an :endpoint parameter. It
// runs before the handler so no route can reach the database, a template or a
// log line with an unvalidated ID.
func requireValidEndpoint(c fiber.Ctx) error {
	if !validEndpointID(c.Params("endpoint")) {
		return c.SendStatus(http.StatusNotFound)
	}
	return c.Next()
}

// endpointURLs builds the public capture and live-feed URLs shown on an
// endpoint page. The WebSocket scheme tracks the request scheme so an HTTPS
// page always advertises wss:// — a ws:// socket on an HTTPS page is blocked
// by browsers as mixed content.
func endpointURLs(scheme, host, endpointID string) (endpointURL, websocketURL string) {
	websocketScheme := "ws"
	if scheme == "https" {
		websocketScheme = "wss"
	}
	return scheme + "://" + host + "/to/" + endpointID,
		websocketScheme + "://" + host + "/ws/" + endpointID
}
