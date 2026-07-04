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
// the forwarding proxy on the ranges from trustedProxyConfig.
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

// TestSchemeHonorsForwardedProtoFromTrustedProxy is the regression guard for
// the http:// endpoint-URL bug. With TrustProxy enabled and an in-cluster
// proxy peer (Traefik on the pod network, or loopback for a sidecar), the
// app must honour X-Forwarded-Proto: https and report the https scheme —
// otherwise EndpointURL renders http:// and the live feed renders ws://.
func TestSchemeHonorsForwardedProtoFromTrustedProxy(t *testing.T) {
	app := proxyTrustApp()

	// One address from each range trustedProxyConfig covers.
	peers := map[string]string{
		"k3s pod network (private 10/8)": "10.42.0.212",
		"private 172.16/12":              "172.16.5.4",
		"private 192.168/16":             "192.168.1.10",
		"loopback (sidecar)":             "127.0.0.1",
		"link-local":                     "169.254.10.20",
	}

	for name, ip := range peers {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, "https", schemeFor(t, app, ip, "https"),
				"trusted proxy %s should have its X-Forwarded-Proto honoured", ip)
		})
	}
}

// TestSchemeIgnoresForwardedProtoFromUntrustedPeer proves the trust is
// bounded: a public-range peer that reaches the pod directly cannot spoof
// the scheme (nor, by the same check, the client IP).
func TestSchemeIgnoresForwardedProtoFromUntrustedPeer(t *testing.T) {
	app := proxyTrustApp()

	assert.Equal(t, "http", schemeFor(t, app, "203.0.113.7", "https"),
		"a public peer must not be able to forge X-Forwarded-Proto")
}

// TestSchemeDirectModeIgnoresForwardedProto guards the direct-deployment
// security property: with no PLATFORM configured TrustProxy is off, so the
// app reports the real connection scheme even for a private-range peer and
// never trusts a forwarded header.
func TestSchemeDirectModeIgnoresForwardedProto(t *testing.T) {
	direct := fiber.New(fiber.Config{
		TrustProxy:       false,
		TrustProxyConfig: trustedProxyConfig(),
	})

	assert.Equal(t, "http", schemeFor(t, direct, "10.42.0.212", "https"),
		"direct mode must ignore X-Forwarded-Proto entirely")
}

// TestResolvePlatformGatesProxyTrust documents the link between PLATFORM and
// proxy trust: TrustProxy is `ipHeader != ""`, so proxy-fronted platforms
// enable trust while an empty or unknown value fails safe to direct (no
// trust). A regression here silently re-breaks the scheme in either
// direction.
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
