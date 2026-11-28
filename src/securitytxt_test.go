package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderSecurityTxt(t *testing.T) {
	t.Run("is served as plain text", func(t *testing.T) {
		response := get(t, "/.well-known/security.txt")

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "text/plain; charset=utf-8", response.Header.Get("Content-Type"))
	})

	// The canonical URL comes from the request, so a self-hosted deployment
	// never claims another deployment's address.
	t.Run("states the contact and this deployment's canonical URL", func(t *testing.T) {
		body := bodyOf(t, get(t, "/.well-known/security.txt"))

		assert.Equal(t, "Contact: mailto:bjorn@bjornkrols.com\n"+
			"Expires: 2027-11-28T00:00:00.000Z\n"+
			"Preferred-Languages: en\n"+
			"Canonical: http://example.com/.well-known/security.txt\n", body)
	})
}
