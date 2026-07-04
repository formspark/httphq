package main

import (
	"net"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// schemeFor spins up a request from peerIP carrying the given
// X-Forwarded-Proto (empty = header absent) and returns the scheme Fiber
// resolves for it. The connection itself is plain HTTP, so anything other
// than "http" means the forwarded header was trusted and honoured.
func schemeFor(t *testing.T, app *fiber.App, peerIP, forwardedProto string) string {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	fctx.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(peerIP)})

	c := app.AcquireCtx(fctx)
	defer app.ReleaseCtx(c)

	if forwardedProto != "" {
		c.Request().Header.Set(fiber.HeaderXForwardedProto, forwardedProto)
	}
	return c.Scheme()
}

// proxyTrustApp mirrors the production Fiber config for a request that
// arrives with a PLATFORM configured (TrustProxy enabled): the app trusts
// the fronting proxy on the ranges from trustedProxyConfig.
func proxyTrustApp() *fiber.App {
	return fiber.New(fiber.Config{
		TrustProxy:       true,
		TrustProxyConfig: trustedProxyConfig(),
	})
}

func TestEndpointURLs(t *testing.T) {
	cases := []struct {
		name        string
		scheme      string
		wantEndURL  string
		wantSockURL string
	}{
		{
			name:        "https yields wss",
			scheme:      "https",
			wantEndURL:  "https://httphq.com/to/purple-frog-0691",
			wantSockURL: "wss://httphq.com/ws/purple-frog-0691",
		},
		{
			name:        "http yields ws",
			scheme:      "http",
			wantEndURL:  "http://httphq.com/to/purple-frog-0691",
			wantSockURL: "ws://httphq.com/ws/purple-frog-0691",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			endURL, sockURL := endpointURLs(tc.scheme, "httphq.com", "purple-frog-0691")
			assert.Equal(t, tc.wantEndURL, endURL)
			assert.Equal(t, tc.wantSockURL, sockURL)
		})
	}
}

// TestSchemeHonorsForwardedProtoFromTrustedProxy checks that with TrustProxy
// enabled and a trusted proxy peer (a private, loopback or link-local
// address), the app honours X-Forwarded-Proto: https and reports the https
// scheme, so a request served over HTTPS behind the proxy renders https://
// and wss:// URLs.
func TestSchemeHonorsForwardedProtoFromTrustedProxy(t *testing.T) {
	app := proxyTrustApp()

	// One address from each range trustedProxyConfig covers.
	peers := map[string]string{
		"private 10/8":       "10.0.0.1",
		"private 172.16/12":  "172.16.5.4",
		"private 192.168/16": "192.168.1.10",
		"loopback":           "127.0.0.1",
		"link-local":         "169.254.10.20",
	}

	for name, ip := range peers {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, "https", schemeFor(t, app, ip, "https"),
				"trusted proxy %s should have its X-Forwarded-Proto honoured", ip)
		})
	}
}

// TestSchemeIgnoresForwardedProtoFromUntrustedPeer checks that a peer outside
// the trusted ranges (a public address) has its X-Forwarded-Proto ignored: the
// app reports the real connection scheme, so the scheme and client IP can't be
// forged from there.
func TestSchemeIgnoresForwardedProtoFromUntrustedPeer(t *testing.T) {
	app := proxyTrustApp()

	assert.Equal(t, "http", schemeFor(t, app, "203.0.113.7", "https"),
		"a public peer must not be able to forge X-Forwarded-Proto")
}

// TestSchemeDirectModeIgnoresForwardedProto checks that with no PLATFORM
// configured TrustProxy is off, so the app reports the real connection scheme
// and ignores X-Forwarded-Proto even from a private-range peer.
func TestSchemeDirectModeIgnoresForwardedProto(t *testing.T) {
	direct := fiber.New(fiber.Config{
		TrustProxy:       false,
		TrustProxyConfig: trustedProxyConfig(),
	})

	assert.Equal(t, "http", schemeFor(t, direct, "10.0.0.1", "https"),
		"direct mode must ignore X-Forwarded-Proto entirely")
}

// TestResolvePlatformGatesProxyTrust checks the link between PLATFORM and
// proxy trust: TrustProxy is `ipHeader != ""`, so proxy-fronted platforms
// enable trust while an empty or unknown value resolves to direct with no
// trust.
func TestResolvePlatformGatesProxyTrust(t *testing.T) {
	trusts := []string{"proxy", "cloudflare", "fly", "heroku", "render"}
	for _, name := range trusts {
		assert.NotEmpty(t, resolvePlatform(name).ipHeader,
			"%s should enable proxy trust", name)
	}

	noTrust := []string{"", "direct", "not-a-platform"}
	for _, name := range noTrust {
		assert.Empty(t, resolvePlatform(name).ipHeader,
			"%q should fall back to direct (no proxy trust)", name)
	}
}
