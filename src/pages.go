package main

import (
	"log/slog"
	"maps"

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

// pageMeta builds the head fields layouts/main renders. Every page goes through
// it, so no surface can quietly ship without the tags the others carry.
//
// Absolute URLs are derived from the request, so every deployment advertises
// its own host. The social image carries its content hash for the same reason
// the stylesheet does, so a card cached against one deploy is not served beside
// the next one's markup.
//
// Every surface that states how long a capture lives renders RetentionPhrase
// rather than typing a figure, so moving retentionWindow moves the promise
// wherever the pages make it. Prose that quoted its own number would go on
// quoting the old one, and a reader has no way to tell.
//
// An empty canonicalPath leaves the canonical and og:url tags off, which is
// what a page that is not the shared address for its content wants.
func pageMeta(c fiber.Ctx, assets *assetIndex, title, description, canonicalPath string) fiber.Map {
	base := pageBaseURL(c)
	meta := fiber.Map{
		"Title":           title,
		"Description":     description,
		"SocialImage":     base + assets.url(socialImagePath),
		"RetentionPhrase": retentionPhrase(retentionWindow),
	}
	if canonicalPath != "" {
		meta["Canonical"] = base + canonicalPath
	}
	return meta
}

// renderPage answers with a page that reads the same for every visitor, so its
// head fields are the whole of it. A surface carrying state of its own builds
// on pageMeta directly instead.
func renderPage(assets *assetIndex, view, title, description, canonicalPath string) fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.Render(view, pageMeta(c, assets, title, description, canonicalPath))
	}
}

func renderIndex(assets *assetIndex) fiber.Handler {
	return renderPage(assets, "index",
		"httphq: inspect HTTP requests in real time",
		"Generate a unique URL, point any client at it, and watch every request arrive: method, headers, body, query string, client IP. No account, free forever.",
		"/")
}

func renderContact(assets *assetIndex) fiber.Handler {
	return renderPage(assets, "contact",
		"Contact | httphq",
		"Found a bug, have an idea, or want to say hi? Get in touch with the people who build httphq.",
		"/contact")
}

// renderEndpoint renders the live capture stream for one endpoint. It carries no
// canonical URL: endpoint pages are per-user surfaces excluded by robots.txt,
// and pointing them at a shared URL would be a lie. It still advertises the
// social image, because a capture URL gets pasted into chat clients that unfurl
// it and a card is what they render.
func renderEndpoint(assets *assetIndex) fiber.Handler {
	return func(c fiber.Ctx) error {
		endpointID := c.Params("endpoint")
		endpointURL, websocketURL, apiURL := endpointURLs(
			c.Scheme(), string(c.Request().Host()), endpointID)
		retention := retentionPhrase(retentionWindow)

		page := fiber.Map{
			"AppScripts":           true,
			"EndpointID":           endpointID,
			"EndpointURL":          endpointURL,
			"EndpointWebSocketURL": websocketURL,
			// The page drops captures from its own list once they age out, so it
			// needs the window as a number rather than as the prose it renders.
			"RetentionSeconds": int(retentionWindow.Seconds()),
			// Rendered rather than written by hand so the prompt quotes this
			// deployment's own URLs and the limits actually in force.
			"AgentPrompt": agentPrompt(
				endpointURL, apiURL, productionRequestsPerMinute, retentionWindow),
		}
		maps.Copy(page, pageMeta(c, assets,
			endpointID+" | httphq",
			"Live capture stream for "+endpointID+". Requests sent to this endpoint appear here in real time and are deleted after "+retention+".",
			""))

		return c.Render("endpoint", page)
	}
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
