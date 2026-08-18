package main

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

// The headers are set by middleware rather than per route, so the assertion
// that matters is that no surface can be reached without them. The paths below
// are one of each response shape the app produces: a rendered page, a file from
// the static directory, an API route, the fallthrough 404, and a request a
// handler refuses by returning an error rather than writing a status. The
// static and refused responses are the ones that matter most, because both are
// built by something that discards a response set on the way in.
func TestSecurityHeaders(t *testing.T) {
	t.Run("every response carries them", func(t *testing.T) {
		responseShapes := []string{
			"/",
			"/contact",
			"/robots.txt",
			"/api/health",
			"/no/such/page",
			"/ws/purple-frog-0691",
		}

		for _, path := range responseShapes {
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
