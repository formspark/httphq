package main

import (
	"log/slog"
	"net"
	"os"
	"slices"
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
// that should be hidden from captured requests. Users inspect their own
// traffic and shouldn't have to care which provider sits in front of httphq.
type platformConfig struct {
	ipHeader string // header with the real client IP; "" = TCP peer
	// ipRightmost marks ipHeader as an X-Forwarded-For list. The client writes
	// any list it likes and the one trusted hop in front of the app appends the
	// address that connected to it, so only the last entry is out of the
	// client's reach. Without it, ipHeader holds one address the edge sets
	// itself, whatever the client sent.
	ipRightmost bool
	stripPrefix []string // captured-request header prefixes to drop as vendor noise
}

// platforms maps the PLATFORM env var to its config. The operator is
// responsible for ensuring traffic cannot reach the app bypassing the
// platform, or a client can write the header itself.
var platforms = map[string]platformConfig{
	"direct":     {},
	"cloudflare": {ipHeader: "Cf-Connecting-Ip", stripPrefix: []string{"Cf-"}},
	// Fly Proxy sets the address it accepted the connection from.
	"fly": {ipHeader: "Fly-Client-Ip", stripPrefix: []string{"Fly-"}},
	// The Heroku router appends the address that connected to it.
	"heroku": {ipHeader: fiber.HeaderXForwardedFor, ipRightmost: true},
	// Render serves through Cloudflare, which keeps the client's
	// X-Forwarded-For and appends to it, so only Cf-Connecting-Ip is out of
	// the client's reach.
	"render": {ipHeader: "Cf-Connecting-Ip", stripPrefix: []string{"Cf-"}},
	// Exactly one reverse proxy that appends the peer it saw: nginx with
	// $proxy_add_x_forwarded_for, Traefik, ingress-nginx. With a second hop in
	// front of it, every visitor would share that hop's address.
	"proxy": {ipHeader: fiber.HeaderXForwardedFor, ipRightmost: true},
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
// strategy: it reads the platform's client-IP header (the rightmost entry when
// the header is an X-Forwarded-For list) and keeps it only when it parses as
// an address. With no platform configured, or when the header is missing or
// malformed, it falls back to the TCP peer. The peer comes from the socket
// rather than c.IP(), which behind a trusted proxy reads the leftmost
// X-Forwarded-For entry, the one the client wrote. String() reports an
// IPv4-mapped ::ffff: address as plain IPv4.
func resolveClientIP(c fiber.Ctx) string {
	if ip := net.ParseIP(platformClientIP(c)); ip != nil {
		return ip.String()
	}
	if ip := c.RequestCtx().RemoteIP(); ip != nil {
		return ip.String()
	}
	return ""
}

// platformClientIP returns the raw client IP the configured PLATFORM's header
// carries, trimmed. Empty when no platform is configured, which leaves the
// caller on the TCP peer.
func platformClientIP(c fiber.Ctx) string {
	if currentPlatform.ipHeader == "" {
		return ""
	}
	values := c.Request().Header.PeekAll(currentPlatform.ipHeader)
	if currentPlatform.ipRightmost {
		return rightmostEntry(values)
	}
	return soleValue(values)
}

// rightmostEntry returns the last entry of an X-Forwarded-For list. A repeated
// header keeps every value in the order it was sent, so the last entry of the
// last value is still the one the trusted hop appended.
func rightmostEntry(values [][]byte) string {
	if len(values) == 0 {
		return ""
	}
	last := string(values[len(values)-1])
	return strings.TrimSpace(last[strings.LastIndexByte(last, ',')+1:])
}

// soleValue returns a single-address header's value. One that repeats is
// malformed, so it yields nothing rather than whichever copy came first.
func soleValue(values [][]byte) string {
	if len(values) != 1 {
		return ""
	}
	return strings.TrimSpace(string(values[0]))
}

// omitHeader reports whether a captured-request header is infrastructure
// noise, meaning a generic forwarding header or a vendor header added by the
// configured PLATFORM, and so should be hidden from the user.
func omitHeader(name string) bool {
	sameName := func(h string) bool { return strings.EqualFold(name, h) }
	hasPrefix := func(prefix string) bool {
		return len(name) >= len(prefix) && strings.EqualFold(name[:len(prefix)], prefix)
	}
	return slices.ContainsFunc(omittedHeaders[:], sameName) ||
		slices.ContainsFunc(currentPlatform.stripPrefix, hasPrefix)
}
