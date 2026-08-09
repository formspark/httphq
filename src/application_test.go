package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"httphq/src/database"
)

// storeCapture writes a capture for an endpoint at a chosen age. GORM only
// stamps CreatedAt when it is zero, so passing one keeps it.
func storeCapture(t *testing.T, endpointID, uuid string, createdAt time.Time) {
	t.Helper()
	database.CreateRequest(t.Context(), &database.Request{
		UUID:       uuid,
		EndpointID: endpointID,
		Method:     "POST",
		Path:       "/",
		CreatedAt:  createdAt,
	})
}

func uuidsFor(ctx context.Context, endpointID string) []string {
	var uuids []string
	for _, request := range database.GetRequestsForEndpointID(ctx, endpointID, "", time.Time{}, 10) {
		uuids = append(uuids, request.UUID)
	}
	return uuids
}

// newApplication builds the whole routing surface, so these cover what the
// wiring itself decides rather than what any one handler does.
func TestNewApplication(t *testing.T) {
	t.Run("serves static files from the public directory", func(t *testing.T) {
		response := get(t, "/robots.txt")

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, bodyOf(t, response), "User-agent")
	})

	// The fallthrough runs after every route, including the prefix-matched
	// capture surface, so a path that matches nothing must not be captured.
	t.Run("an unmatched path is a 404", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, get(t, "/no/such/page").StatusCode)
	})
}

func TestSweepRetention(t *testing.T) {
	t.Run("drops captures older than the retention window", func(t *testing.T) {
		endpointID := "sweep-old"
		storeCapture(t, endpointID, "sweep-old-expired",
			time.Now().Add(-retentionWindow).Add(-time.Minute))

		sweepRetention()

		assert.Empty(t, uuidsFor(t.Context(), endpointID))
	})

	t.Run("keeps captures inside the retention window", func(t *testing.T) {
		endpointID := "sweep-recent"
		storeCapture(t, endpointID, "sweep-recent-live",
			time.Now().Add(-retentionWindow).Add(time.Minute))

		sweepRetention()

		assert.Equal(t, []string{"sweep-recent-live"}, uuidsFor(t.Context(), endpointID))
	})
}
