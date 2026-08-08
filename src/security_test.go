package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
