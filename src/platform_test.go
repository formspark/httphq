package main

import (
	"net"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// contextWith builds a request context from peerIP carrying the given headers,
// so a test can exercise the header-reading helpers without a live server.
func contextWith(t *testing.T, app *fiber.App, peerIP string, headers map[string]string) fiber.Ctx {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	if peerIP != "" {
		fctx.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(peerIP)})
	}

	c := app.AcquireCtx(fctx)
	t.Cleanup(func() { app.ReleaseCtx(c) })

	for name, value := range headers {
		c.Request().Header.Set(name, value)
	}
	return c
}

// schemeFor returns the scheme Fiber resolves for a request from peerIP
// carrying the given X-Forwarded-Proto (empty = header absent). The connection
// itself is plain HTTP, so anything other than "http" means the forwarded
// header was trusted and honoured.
func schemeFor(t *testing.T, app *fiber.App, peerIP, forwardedProto string) string {
	t.Helper()

	headers := map[string]string{}
	if forwardedProto != "" {
		headers[fiber.HeaderXForwardedProto] = forwardedProto
	}
	return contextWith(t, app, peerIP, headers).Scheme()
}

// proxyTrustApp mirrors the production Fiber config for a request that arrives
// with a PLATFORM configured (TrustProxy enabled): the app trusts the fronting
// proxy on the ranges from trustedProxyConfig.
func proxyTrustApp() *fiber.App {
	return fiber.New(fiber.Config{
		TrustProxy:       true,
		TrustProxyConfig: trustedProxyConfig(),
	})
}

func TestResolvePlatform(t *testing.T) {
	t.Run("proxy-fronted platforms name a client-IP header", func(t *testing.T) {
		for _, name := range []string{"proxy", "cloudflare", "fly", "heroku", "render"} {
			assert.NotEmptyf(t, resolvePlatform(name).ipHeader,
				"%s should read the client IP from a header", name)
		}
	})

	// TrustProxy is `ipHeader != ""`, so a platform with no header is also a
	// platform with no proxy trust.
	t.Run("unset, direct and unrecognised values fall back to direct", func(t *testing.T) {
		for _, name := range []string{"", "direct", "not-a-platform"} {
			assert.Emptyf(t, resolvePlatform(name).ipHeader,
				"%q should fall back to direct, which trusts no header", name)
		}
	})

	t.Run("the name is trimmed and matched case-insensitively", func(t *testing.T) {
		assert.Equal(t, platforms["cloudflare"], resolvePlatform("  CloudFlare  "))
	})

	t.Run("list-valued platforms are the ones reading X-Forwarded-For", func(t *testing.T) {
		for name, config := range platforms {
			if config.ipHeader == fiber.HeaderXForwardedFor {
				assert.Truef(t, config.ipList, "%s reads a list header and must take the leftmost entry", name)
			}
		}
	})
}

func TestResolveClientIP(t *testing.T) {
	app := fiber.New()

	t.Run("direct mode reads the connection peer", func(t *testing.T) {
		withPlatform(t, "direct")

		c := contextWith(t, app, "203.0.113.7", map[string]string{
			"Cf-Connecting-Ip": "198.51.100.1",
			"X-Forwarded-For":  "198.51.100.2",
		})

		assert.Equal(t, "203.0.113.7", resolveClientIP(c))
	})

	t.Run("a configured platform reads its own header", func(t *testing.T) {
		withPlatform(t, "cloudflare")

		c := contextWith(t, app, "203.0.113.7", map[string]string{
			"Cf-Connecting-Ip": "198.51.100.1",
		})

		assert.Equal(t, "198.51.100.1", resolveClientIP(c))
	})

	t.Run("a list header yields the leftmost entry", func(t *testing.T) {
		withPlatform(t, "proxy")

		c := contextWith(t, app, "203.0.113.7", map[string]string{
			"X-Forwarded-For": " 198.51.100.1 , 10.0.0.1, 172.16.0.1",
		})

		assert.Equal(t, "198.51.100.1", resolveClientIP(c))
	})

	// Falling back rather than trusting the header keeps a garbage value from
	// becoming its own rate-limit bucket.
	t.Run("a missing or malformed platform header falls back to the peer", func(t *testing.T) {
		withPlatform(t, "cloudflare")

		for name, header := range map[string]string{
			"missing":     "",
			"not an IP":   "not-an-ip",
			"empty entry": "   ",
		} {
			t.Run(name, func(t *testing.T) {
				c := contextWith(t, app, "203.0.113.7", map[string]string{"Cf-Connecting-Ip": header})
				assert.Equal(t, "203.0.113.7", resolveClientIP(c))
			})
		}
	})

	t.Run("IPv6 survives round-tripping", func(t *testing.T) {
		withPlatform(t, "fly")

		c := contextWith(t, app, "203.0.113.7", map[string]string{
			"Fly-Client-Ip": "2001:db8::1",
		})

		assert.Equal(t, "2001:db8::1", resolveClientIP(c))
	})
}

func TestOmitHeader(t *testing.T) {
	t.Run("generic forwarding headers are hidden whatever the platform", func(t *testing.T) {
		withPlatform(t, "direct")

		for _, name := range []string{"X-Forwarded-For", "x-forwarded-proto", "Via", "Traceparent", "X-Real-Ip"} {
			assert.Truef(t, omitHeader(name), "%s is infrastructure noise and should be hidden", name)
		}
	})

	t.Run("a caller's own headers are kept", func(t *testing.T) {
		withPlatform(t, "direct")

		for _, name := range []string{"Content-Type", "Authorization", "User-Agent", "X-Forwarded-Custom-Thing"} {
			assert.Falsef(t, omitHeader(name), "%s belongs to the caller and should be captured", name)
		}
	})

	t.Run("vendor headers are hidden only for the platform that adds them", func(t *testing.T) {
		withPlatform(t, "cloudflare")
		assert.True(t, omitHeader("CF-Ray"))
		assert.True(t, omitHeader("cf-ipcountry"))
		assert.False(t, omitHeader("Fly-Region"))

		withPlatform(t, "fly")
		assert.True(t, omitHeader("Fly-Region"))
		assert.False(t, omitHeader("CF-Ray"))
	})

	t.Run("a header shorter than a vendor prefix is not mistaken for one", func(t *testing.T) {
		withPlatform(t, "cloudflare")

		assert.False(t, omitHeader("C"))
	})
}

func TestTrustedProxyConfig(t *testing.T) {
	// One address from each range trustedProxyConfig covers.
	trustedPeers := map[string]string{
		"private 10/8":       "10.0.0.1",
		"private 172.16/12":  "172.16.5.4",
		"private 192.168/16": "192.168.1.10",
		"loopback":           "127.0.0.1",
		"link-local":         "169.254.10.20",
	}

	t.Run("a trusted peer's X-Forwarded-Proto is honoured", func(t *testing.T) {
		app := proxyTrustApp()

		for name, ip := range trustedPeers {
			t.Run(name, func(t *testing.T) {
				assert.Equal(t, "https", schemeFor(t, app, ip, "https"),
					"a request forwarded by %s was served over HTTPS and must render https:// and wss:// URLs", ip)
			})
		}
	})

	t.Run("a public peer's X-Forwarded-Proto is ignored", func(t *testing.T) {
		assert.Equal(t, "http", schemeFor(t, proxyTrustApp(), "203.0.113.7", "https"),
			"a peer outside the trusted ranges must not be able to forge the scheme")
	})

	t.Run("proxy trust off ignores X-Forwarded-Proto entirely", func(t *testing.T) {
		direct := fiber.New(fiber.Config{
			TrustProxy:       false,
			TrustProxyConfig: trustedProxyConfig(),
		})

		assert.Equal(t, "http", schemeFor(t, direct, "10.0.0.1", "https"),
			"with no platform configured the app is directly exposed and reports the real connection")
	})
}
