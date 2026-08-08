package main

import (
	"log/slog"

	"github.com/atrox/haikunatorgo/v2"
	"github.com/gofiber/fiber/v3"
)

// socialImagePath is the preview image every page advertises. One image for the
// whole site: an unfurled link is a link to httphq, whichever page it points at.
const socialImagePath = "/social-card.png"

// pageBaseURL is the scheme+host a rendered page is being served from, used to
// build the absolute URLs that canonical and Open Graph tags require. It tracks
// the request rather than a configured hostname so a self-hosted deployment
// advertises itself, not httphq.com.
func pageBaseURL(c fiber.Ctx) string {
	return c.Scheme() + "://" + string(c.Request().Host())
}

// pageMeta builds the head fields layouts/main renders for a public page.
// Absolute URLs are derived from the request, so every deployment advertises
// its own host.
func pageMeta(c fiber.Ctx, title, description, path string) fiber.Map {
	base := pageBaseURL(c)
	return fiber.Map{
		"Title":       title,
		"Description": description,
		"Canonical":   base + path,
		"SocialImage": base + socialImagePath,
	}
}

func renderIndex(c fiber.Ctx) error {
	return c.Render("index", pageMeta(c,
		"httphq: inspect HTTP requests in real time",
		"Generate a unique URL, point any client at it, and watch every request arrive: method, headers, body, query string, client IP. No account, free forever.",
		"/"))
}

func renderContact(c fiber.Ctx) error {
	return c.Render("contact", pageMeta(c,
		"Contact | httphq",
		"Found a bug, have an idea, or want to say hi? Get in touch with the people who build httphq.",
		"/contact"))
}

// renderEndpoint renders the live capture stream for one endpoint. It carries no
// canonical URL: endpoint pages are per-user surfaces excluded by robots.txt,
// and pointing them at a shared URL would be a lie.
func renderEndpoint(c fiber.Ctx) error {
	endpointID := c.Params("endpoint")
	endpointURL, websocketURL := endpointURLs(c.Scheme(), string(c.Request().Host()), endpointID)
	return c.Render("endpoint", fiber.Map{
		"Title":                endpointID + " | httphq",
		"Description":          "Live capture stream for " + endpointID + ". Requests sent to this endpoint appear here in real time and are deleted after 4 hours.",
		"AppScripts":           true,
		"EndpointID":           endpointID,
		"EndpointURL":          endpointURL,
		"EndpointWebSocketURL": websocketURL,
	})
}

// createEndpoint mints an endpoint ID and sends the caller to its page. The ID
// is the only thing protecting a stream, so it comes from a generator with
// enough room that guessing one is not a search worth running.
func createEndpoint(haikuMaker *haikunator.Haikunator) fiber.Handler {
	return func(c fiber.Ctx) error {
		endpointID := haikuMaker.Haikunate()
		slog.InfoContext(c.Context(), "endpoint created", "endpoint_id", endpointID)
		return c.Redirect().To("/" + endpointID)
	}
}
