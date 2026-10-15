package main

import (
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

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
// proxy on the ranges from trustedProxyConfig, and reads the peer it reports
// from the same forwarded header the application does.
func proxyTrustApp() *fiber.App {
	return fiber.New(fiber.Config{
		TrustProxy:       true,
		TrustProxyConfig: trustedProxyConfig(),
		ProxyHeader:      fiber.HeaderXForwardedFor,
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
}

// The table itself, rather than the lookup into it. A platform entry that
// disagrees with the header it names is wrong however it was resolved.
func TestPlatforms(t *testing.T) {
	t.Run("X-Forwarded-For is read from the right and only from the right", func(t *testing.T) {
		for name, config := range platforms {
			assert.Equalf(t, config.ipHeader == fiber.HeaderXForwardedFor, config.ipRightmost,
				"%s: the client writes the left of X-Forwarded-For, so only that list is read from the right", name)
		}
	})

	t.Run("render reads the header its Cloudflare edge overwrites", func(t *testing.T) {
		assert.Equal(t, "Cf-Connecting-Ip", platforms["render"].ipHeader)
	})
}

// peerIP is the TCP peer the client-IP cases connect from, and forgedIP what a
// client writes into a header to pose as someone else.
const (
	peerIP   = "203.0.113.7"
	forgedIP = "192.0.2.66"
)

// clientIPFor resolves the client IP under platform for a request from peerIP
// carrying headers, each value of a header sent as a separate header line.
func clientIPFor(t *testing.T, app *fiber.App, platform string, headers map[string][]string) string {
	t.Helper()
	withPlatform(t, platform)

	c := contextWith(t, app, peerIP, nil)
	for name, values := range headers {
		for _, value := range values {
			c.Request().Header.Add(name, value)
		}
	}
	return resolveClientIP(c)
}

func TestResolveClientIP(t *testing.T) {
	app := fiber.New()

	t.Run("direct mode reads the connection peer", func(t *testing.T) {
		assert.Equal(t, peerIP, clientIPFor(t, app, "direct", map[string][]string{
			"Cf-Connecting-Ip": {"198.51.100.1"},
			"Fly-Client-Ip":    {"198.51.100.1"},
			"X-Forwarded-For":  {"198.51.100.2"},
		}))
	})

	t.Run("an IPv6 peer is kept and an IPv4-mapped one is reported as IPv4", func(t *testing.T) {
		withPlatform(t, "direct")

		assert.Equal(t, "2001:db8::7", resolveClientIP(contextWith(t, app, "2001:db8::7", nil)))
		assert.Equal(t, "198.51.100.7", resolveClientIP(contextWith(t, app, "::ffff:198.51.100.7", nil)))
	})

	// Behind a trusted proxy, Fiber's own c.IP() reads the leftmost
	// X-Forwarded-For entry, which the client wrote. The fallback must be the
	// socket peer instead.
	t.Run("the fallback behind a trusted proxy is the socket peer, not X-Forwarded-For", func(t *testing.T) {
		withPlatform(t, "cloudflare")

		c := contextWith(t, proxyTrustApp(), "10.0.0.1", map[string]string{
			fiber.HeaderXForwardedFor: forgedIP,
		})

		assert.Equal(t, "10.0.0.1", resolveClientIP(c))
	})

	// A connection with no readable peer reports 0.0.0.0 rather than nothing,
	// so that is what a capture stores and a reader sees. It looks like an
	// address the client owned, and nothing downstream corrects that.
	t.Run("an unreadable peer is reported as 0.0.0.0", func(t *testing.T) {
		withPlatform(t, "cloudflare")

		c := contextWith(t, app, "", map[string]string{"Cf-Connecting-Ip": "not-an-ip"})

		assert.Equal(t, "0.0.0.0", resolveClientIP(c))
	})
}

// One trusted hop appends the address that connected to it, and the client
// writes everything to its left.
func TestResolveClientIPFromForwardedFor(t *testing.T) {
	app := fiber.New()

	for _, platform := range []string{"heroku", "proxy"} {
		t.Run(platform, func(t *testing.T) {
			xff := func(values ...string) string {
				return clientIPFor(t, app, platform, map[string][]string{fiber.HeaderXForwardedFor: values})
			}

			t.Run("reads a lone entry", func(t *testing.T) {
				assert.Equal(t, "198.51.100.1", xff("198.51.100.1"))
			})

			t.Run("takes the rightmost entry of a multi-hop chain", func(t *testing.T) {
				assert.Equal(t, "198.51.100.1", xff("10.0.0.2, 10.0.0.1, 198.51.100.1"))
			})

			t.Run("ignores a leftmost entry the client forged", func(t *testing.T) {
				assert.Equal(t, "198.51.100.1", xff(forgedIP+", 198.51.100.1"))
				assert.Equal(t, "198.51.100.1", xff("192.0.2.67, 198.51.100.1"))
			})

			t.Run("reads the last entry across repeated headers", func(t *testing.T) {
				assert.Equal(t, "198.51.100.1", xff(forgedIP, "10.0.0.1, 198.51.100.1"))
			})

			t.Run("trims surrounding whitespace", func(t *testing.T) {
				assert.Equal(t, "198.51.100.1", xff("10.0.0.1 ,  198.51.100.1  "))
			})

			t.Run("reads IPv6 and unwraps an IPv4-mapped entry", func(t *testing.T) {
				assert.Equal(t, "2001:db8::1", xff(forgedIP+", 2001:db8::1"))
				assert.Equal(t, "198.51.100.1", xff(forgedIP+", ::ffff:198.51.100.1"))
			})

			t.Run("ignores the headers belonging to other platforms", func(t *testing.T) {
				assert.Equal(t, peerIP, clientIPFor(t, app, platform, map[string][]string{
					"Cf-Connecting-Ip": {"198.51.100.1"},
					"Fly-Client-Ip":    {"198.51.100.1"},
				}))
			})

			// Falling back rather than trusting the header keeps a garbage value
			// from becoming its own rate-limit bucket.
			for _, value := range []string{"", " , ", "198.51.100.1, ", "198.51.100.1, garbage", "198.51.100.1:443", "[::1]"} {
				t.Run("falls back to the peer for "+strconv.Quote(value), func(t *testing.T) {
					assert.Equal(t, peerIP, xff(value))
				})
			}
		})
	}
}

// The edge sets one address itself, whatever the client sent.
func TestResolveClientIPFromEdgeHeader(t *testing.T) {
	app := fiber.New()

	// foreign is the client-IP header of an edge this platform does not sit
	// behind, so a client can write it.
	for _, edge := range []struct{ platform, header, foreign string }{
		{"cloudflare", "Cf-Connecting-Ip", "Fly-Client-Ip"},
		{"fly", "Fly-Client-Ip", "Cf-Connecting-Ip"},
		{"render", "Cf-Connecting-Ip", "Fly-Client-Ip"},
	} {
		t.Run(edge.platform, func(t *testing.T) {
			single := func(values ...string) string {
				return clientIPFor(t, app, edge.platform, map[string][]string{edge.header: values})
			}

			t.Run("reads its header", func(t *testing.T) {
				assert.Equal(t, "198.51.100.1", single("198.51.100.1"))
				assert.Equal(t, "198.51.100.1", single(" 198.51.100.1 "))
			})

			t.Run("ignores X-Forwarded-For, which the client can write", func(t *testing.T) {
				assert.Equal(t, peerIP, clientIPFor(t, app, edge.platform, map[string][]string{
					fiber.HeaderXForwardedFor: {forgedIP},
				}))
				assert.Equal(t, "198.51.100.1", clientIPFor(t, app, edge.platform, map[string][]string{
					edge.header:               {"198.51.100.1"},
					fiber.HeaderXForwardedFor: {forgedIP},
				}))
			})

			t.Run("ignores the header belonging to another edge", func(t *testing.T) {
				assert.Equal(t, peerIP, clientIPFor(t, app, edge.platform, map[string][]string{
					edge.foreign: {forgedIP},
				}))
			})

			t.Run("reads IPv6 and unwraps an IPv4-mapped address", func(t *testing.T) {
				assert.Equal(t, "2001:db8::1", single("2001:db8::1"))
				assert.Equal(t, "198.51.100.1", single("::ffff:198.51.100.1"))
			})

			t.Run("falls back to the peer for a repeated header", func(t *testing.T) {
				assert.Equal(t, peerIP, single(forgedIP, "198.51.100.1"))
			})

			for _, value := range []string{"", "   ", "not-an-ip", "999.0.0.1", "198.51.100.1, " + forgedIP} {
				t.Run("falls back to the peer for "+strconv.Quote(value), func(t *testing.T) {
					assert.Equal(t, peerIP, single(value))
				})
			}
		})
	}
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

		withPlatform(t, "render")
		assert.True(t, omitHeader("CF-Ray"))
		assert.False(t, omitHeader("Fly-Region"))
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
