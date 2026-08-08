package main

import "github.com/gofiber/fiber/v3"

// The origin the local design tooling serves its injected picker script and
// socket from. Development only: production must never carry it, so it is
// gated on the environment rather than on a request-time condition a client
// could influence.
const designToolingOrigin = "http://localhost:8400"

// contentSecurityPolicy assembles the CSP sent on every response. Its only
// variable part is fixed for the process lifetime, so it is built once.
//
//   - script-src needs 'unsafe-eval' for Alpine, which compiles directive
//     expressions via the Function constructor, and the two CDN origins that
//     serve Alpine and the syntax highlighter to the endpoint page. All page
//     scripts are external so script-src does NOT need 'unsafe-inline'.
//   - style-src needs the CDN origin for the highlighter's theme, and
//     'unsafe-inline' because Alpine's x-show toggles visibility through an
//     inline display style. The application's own stylesheet is first-party
//     and covered by 'self'.
func contentSecurityPolicy(allowDesignTooling bool) string {
	designTooling := ""
	if allowDesignTooling {
		designTooling = " " + designToolingOrigin
	}
	return "default-src 'self'; " +
		"script-src 'self' 'unsafe-eval' https://unpkg.com https://cdn.jsdelivr.net" + designTooling + "; " +
		"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
		"img-src 'self' data:; " +
		"connect-src 'self' ws: wss:" + designTooling + "; " +
		"frame-ancestors 'none'"
}

// securityHeaders stamps the fixed security headers onto every response.
//
// They are set on the way out, after the rest of the chain has run. A handler
// is free to reset the response it is building — the static middleware does
// exactly that when a path resolves to a directory rather than a file — and a
// header set on the way in would go with it. These have to hold for every
// response the app emits, so they are written where nothing downstream can
// discard them.
func securityHeaders(policy string) fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()
		c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
		c.Set(fiber.HeaderReferrerPolicy, "no-referrer")
		c.Set(fiber.HeaderXFrameOptions, "DENY")
		c.Set(fiber.HeaderContentSecurityPolicy, policy)
		return err
	}
}
