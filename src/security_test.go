package main

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

// The headers are set by middleware rather than per route, so the assertion
// that matters is that no surface can be reached without them: a page, an API
// route, a probe and the fallthrough 404 all have to carry them.
func TestSecurityHeaders(t *testing.T) {
	t.Run("every response carries them", func(t *testing.T) {
		for _, path := range []string{"/", "/contact", "/api/health", "/no/such/page"} {
			t.Run(path, func(t *testing.T) {
				response := get(t, path)

				assert.Equal(t, "nosniff", response.Header.Get("X-Content-Type-Options"))
				assert.Equal(t, "no-referrer", response.Header.Get("Referrer-Policy"))
				assert.Equal(t, "DENY", response.Header.Get("X-Frame-Options"))
				assert.Contains(t, response.Header.Get(fiber.HeaderContentSecurityPolicy), "frame-ancestors 'none'")
			})
		}
	})
}

func TestContentSecurityPolicy(t *testing.T) {
	// The design tooling opens a socket back to a local origin. Shipping that
	// origin to production would widen the policy for every visitor, so the
	// switch is the environment rather than anything a request can influence.
	t.Run("the design tooling origin is development only", func(t *testing.T) {
		assert.NotContains(t, contentSecurityPolicy(false), designToolingOrigin)
		assert.Contains(t, contentSecurityPolicy(true), designToolingOrigin)
	})

	t.Run("the tooling origin reaches only the directives that need it", func(t *testing.T) {
		for _, directive := range strings.Split(contentSecurityPolicy(true), "; ") {
			name, _, _ := strings.Cut(directive, " ")
			if strings.Contains(directive, designToolingOrigin) {
				assert.Containsf(t, []string{"script-src", "connect-src"}, name,
					"%s should not carry the design tooling origin", name)
			}
		}
	})

	t.Run("the baseline directives hold in both environments", func(t *testing.T) {
		for _, policy := range []string{contentSecurityPolicy(false), contentSecurityPolicy(true)} {
			assert.Contains(t, policy, "default-src 'self'")
			assert.Contains(t, policy, "frame-ancestors 'none'")
			assert.Contains(t, policy, "img-src 'self' data:")
			// Every page script is external, so inline script never has to run.
			assert.NotContains(t, policy, "script-src 'self' 'unsafe-inline'")
		}
	})
}
