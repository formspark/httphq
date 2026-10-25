package main

import (
	"html/template"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/atrox/haikunatorgo/v2"
	"github.com/gofiber/fiber/v3"
)

// socialImagePath is the preview image every page advertises. One image for the
// whole site: an unfurled link is a link to httphq, whichever page it points at.
const socialImagePath = "/social-card.png"

// repositoryURL is where the source lives. The footer links it on every page
// and the home page's structured data names it, so both read it from here.
const repositoryURL = "https://github.com/formspark/httphq"

// The home page's title and description are stated once and shared by its head
// tags and its structured data, so a search result and a rich result never
// describe the product two ways.
const (
	indexTitle       = "httphq: inspect HTTP requests in real time"
	indexDescription = "Generate a unique URL, point any client at it, and watch every request arrive: method, headers, body, query string, client IP. No account, free forever."
)

// pageBaseURL is the scheme+host a rendered page is being served from, used to
// build the absolute URLs that canonical and Open Graph tags require. It tracks
// the request rather than a configured hostname so a self-hosted deployment
// advertises itself, not httphq.com.
func pageBaseURL(c fiber.Ctx) string {
	return c.Scheme() + "://" + string(c.Request().Host())
}

// redirectWWW sends a page read on a www. host to the same path on the bare
// host, so a site reachable under both names has one canonical address
// instead of two that each canonicalise to themselves. It strips the prefix
// rather than naming a hostname, so a self-hosted deployment gets the same
// treatment. Only reads are redirected: a capture sent to a www. URL is
// stored where it landed, because most webhook senders do not follow a
// redirect and the capture would be lost.
func redirectWWW(c fiber.Ctx) error {
	host, found := strings.CutPrefix(string(c.Request().Host()), "www.")
	if !found || !slices.Contains(readMethods, c.Method()) {
		return c.Next()
	}
	return c.Redirect().Status(http.StatusMovedPermanently).To(c.Scheme() + "://" + host + c.OriginalURL())
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
		"RepositoryURL":   repositoryURL,
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

// webApplication is the schema.org description of httphq that the home page
// carries as JSON-LD. html/template marshals it to JSON inside the script
// element and escapes anything that could close the element early.
type webApplication struct {
	Context             string `json:"@context"`
	Type                string `json:"@type"`
	Name                string `json:"name"`
	URL                 string `json:"url"`
	Description         string `json:"description"`
	ApplicationCategory string `json:"applicationCategory"`
	OperatingSystem     string `json:"operatingSystem"`
	IsAccessibleForFree bool   `json:"isAccessibleForFree"`
	Offers              offer  `json:"offers"`
	SameAs              string `json:"sameAs"`
}

type offer struct {
	Type          string `json:"@type"`
	Price         string `json:"price"`
	PriceCurrency string `json:"priceCurrency"`
}

// renderIndex is the one page that carries structured data. Its URL is the
// page's own canonical, so a self-hosted deployment describes itself.
func renderIndex(assets *assetIndex) fiber.Handler {
	return func(c fiber.Ctx) error {
		page := pageMeta(c, assets, indexTitle, indexDescription, "/")
		page["StructuredData"] = webApplication{
			Context:             "https://schema.org",
			Type:                "WebApplication",
			Name:                "httphq",
			URL:                 pageBaseURL(c) + "/",
			Description:         indexDescription,
			ApplicationCategory: "DeveloperApplication",
			OperatingSystem:     "Any",
			IsAccessibleForFree: true,
			Offers:              offer{Type: "Offer", Price: "0", PriceCurrency: "USD"},
			SameAs:              repositoryURL,
		}
		return c.Render("index", page)
	}
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
			"AppScripts":  true,
			"EndpointID":  endpointID,
			"EndpointURL": endpointURL,
			// Typed rather than passed as a string because html/template trusts
			// only http, https and mailto in a URL attribute, and rewrites
			// anything else to #ZgotmplZ. The value is safe to exempt: its
			// scheme is derived here from c.Scheme(), and its endpoint ID has
			// already passed requireValidEndpoint. Attribute escaping still
			// applies, so the host cannot break out of the attribute.
			"EndpointWebSocketURL": template.URL(websocketURL),
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

// idMaker hands out endpoint IDs from one generator to concurrent requests. A
// Haikunator's random source is not safe for concurrent use: two requests
// minting at once can corrupt its state and panic, so every draw holds the lock.
type idMaker struct {
	mu         sync.Mutex
	haikunator *haikunator.Haikunator
}

func (m *idMaker) next() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.haikunator.Haikunate()
}

// createEndpoint mints an endpoint ID and sends the caller to its page. The ID
// is the only thing protecting a stream, so it comes from a generator with
// enough room that guessing one is not a search worth running.
func createEndpoint(haikuMaker *haikunator.Haikunator) fiber.Handler {
	ids := &idMaker{haikunator: haikuMaker}
	return func(c fiber.Ctx) error {
		endpointID := ids.next()
		slog.InfoContext(c.Context(), "endpoint created")
		return c.Redirect().To("/" + endpointID)
	}
}
