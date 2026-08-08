package main

import (
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
)

var isProduction = os.Getenv("APPLICATION_ENV") == "production"

// Generic forwarding headers stripped from every captured request so users
// see their original payload, not infrastructure-added headers. Vendor headers
// specific to a hosting platform are stripped separately, see platformConfig.
var omittedHeaders = [...]string{
	"Cdn-Loop",
	"Trace",
	"Traceparent",
	"Tracestate",
	"Via",
	"X-Forwarded-For",
	"X-Forwarded-Host",
	"X-Forwarded-Port",
	"X-Forwarded-Proto",
	"X-Forwarded-Server",
	"X-Forwarded-Ssl",
	"X-Real-Ip",
	"X-Request-Start",
}

// platformConfig describes how a hosting platform exposes request metadata:
// which header carries the real client IP, and which vendor headers it adds
// that should be hidden from captured requests — users inspect their own
// traffic and shouldn't have to care which provider sits in front of httphq.
type platformConfig struct {
	ipHeader    string   // header with the real client IP; "" = TCP peer
	ipList      bool     // ipHeader is a comma-separated list; take the leftmost
	stripPrefix []string // captured-request header prefixes to drop as vendor noise
}

// platforms maps the PLATFORM env var to its config. Each platform overwrites
// (or reliably sets) its own headers; the operator is responsible for ensuring
// traffic cannot reach the app bypassing the platform.
var platforms = map[string]platformConfig{
	"direct":     {},
	"cloudflare": {ipHeader: "Cf-Connecting-Ip", stripPrefix: []string{"Cf-"}},
	"fly":        {ipHeader: "Fly-Client-Ip", stripPrefix: []string{"Fly-"}},
	"heroku":     {ipHeader: fiber.HeaderXForwardedFor, ipList: true},
	"render":     {ipHeader: fiber.HeaderXForwardedFor, ipList: true},
	"proxy":      {ipHeader: fiber.HeaderXForwardedFor, ipList: true},
}

// currentPlatform is the config resolved once from PLATFORM at startup.
var currentPlatform platformConfig

// resolvePlatform maps a PLATFORM value to its config. An empty value means
// "direct"; an unrecognised value fails safe to "direct" so a typo never
// causes a spoofable header to be trusted.
func resolvePlatform(name string) platformConfig {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "direct"
	}
	if p, ok := platforms[name]; ok {
		return p
	}
	slog.Warn("unknown PLATFORM, falling back to direct", "platform", name)
	return platforms["direct"]
}

// trustedProxyConfig lists the peers whose X-Forwarded-* headers Fiber may
// honour once TrustProxy is enabled. httphq is only ever fronted by a reverse
// proxy that reaches it from a private, loopback or link-local address; no
// legitimate direct client connects from those ranges. Bounding trust to them
// means a public client that reaches the app directly still can't spoof the
// scheme or client IP.
func trustedProxyConfig() fiber.TrustProxyConfig {
	return fiber.TrustProxyConfig{
		Private:   true,
		Loopback:  true,
		LinkLocal: true,
	}
}

// resolveClientIP returns the real client IP per the configured PLATFORM
// strategy: it reads the platform's client-IP header (leftmost entry when the
// header is a list) and validates it parses. With no platform configured, or
// when the header is missing or malformed, it falls back to the TCP peer.
func resolveClientIP(c fiber.Ctx) string {
	if currentPlatform.ipHeader != "" {
		v := c.Get(currentPlatform.ipHeader)
		if currentPlatform.ipList {
			if i := strings.IndexByte(v, ','); i >= 0 {
				v = v[:i]
			}
		}
		if ip := net.ParseIP(strings.TrimSpace(v)); ip != nil {
			return ip.String()
		}
	}
	if ip := net.ParseIP(c.IP()); ip != nil {
		return ip.String()
	}
	return ""
}

// omitHeader reports whether a captured-request header is infrastructure noise
// — a generic forwarding header or a vendor header added by the configured
// PLATFORM — and so should be hidden from the user.
func omitHeader(name string) bool {
	for _, h := range omittedHeaders {
		if strings.EqualFold(name, h) {
			return true
		}
	}
	for _, prefix := range currentPlatform.stripPrefix {
		if len(name) >= len(prefix) && strings.EqualFold(name[:len(prefix)], prefix) {
			return true
		}
	}
	return false
}
