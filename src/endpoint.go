package main

import (
	"net/http"
	"regexp"

	"github.com/gofiber/fiber/v3"
)

// endpointIDPattern bounds an endpoint ID to the shape haikunator emits
// (lowercase words and digits joined by hyphens). Rejecting anything else
// keeps attacker-controlled characters (quotes, angle brackets, parens) out of
// the rendered pages, the database, and the logs.
var endpointIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// reservedPaths are words a visitor or a crawler guesses as a site's own pages.
// Any slug-shaped path renders an endpoint page, so without this list
// /pricing would answer 200 with an empty capture stream and search engines
// would index it as a page of the site. They are refused as endpoint IDs
// everywhere, not only as pages, so no capture can be stored under an ID whose
// page cannot be opened. Minted IDs never collide: haikunator always appends a
// numeric token after a hyphen, and none of these carries one.
var reservedPaths = map[string]bool{
	"about": true, "admin": true, "api": true, "blog": true, "changelog": true,
	"contact": true, "dashboard": true, "docs": true, "faq": true,
	"features": true, "help": true, "llms": true, "login": true, "logout": true,
	"pricing": true, "privacy": true, "register": true, "security": true,
	"signin": true, "signup": true, "sitemap": true, "status": true,
	"support": true, "terms": true,
}

func validEndpointID(id string) bool {
	return len(id) <= 64 && endpointIDPattern.MatchString(id) && !reservedPaths[id]
}

// requireValidEndpoint guards every route carrying an :endpoint parameter
// except the endpoint page. It runs before the handler so no route can reach
// the database, a template or a log line with an unvalidated ID. The API, the
// capture route and the socket answer machines, so a refusal is a bare status.
func requireValidEndpoint(c fiber.Ctx) error {
	if !validEndpointID(c.Params("endpoint")) {
		return c.SendStatus(http.StatusNotFound)
	}
	return c.Next()
}

// requireValidEndpointPage is the same guard for the endpoint page, where the
// caller is usually a person: a refused ID is answered by notFound, so a
// guessed or mistyped address reads like any other missing page.
func requireValidEndpointPage(notFound fiber.Handler) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !validEndpointID(c.Params("endpoint")) {
			return notFound(c)
		}
		return c.Next()
	}
}

// endpointURLs builds the three public URLs for an endpoint: where captures are
// sent, where the live feed is read, and where the JSON listing is polled. They
// are built together so a caller cannot advertise one host for the capture URL
// and another for the API.
//
// Every one of them tracks the request's scheme and host rather than a
// configured name, so a self-hosted deployment advertises itself. The WebSocket
// scheme tracks it too: a ws:// socket on an HTTPS page is blocked by browsers
// as mixed content.
func endpointURLs(scheme, host, endpointID string) (endpointURL, websocketURL, apiURL string) {
	websocketScheme := "ws"
	if scheme == "https" {
		websocketScheme = "wss"
	}
	return scheme + "://" + host + "/to/" + endpointID,
		websocketScheme + "://" + host + "/ws/" + endpointID,
		scheme + "://" + host + "/api/endpoints/" + endpointID + "/requests"
}
