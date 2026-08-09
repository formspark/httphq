package main

import (
	"context"
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
	for _, request := range database.GetRequestsForEndpointID(ctx, endpointID, "", 10) {
		uuids = append(uuids, request.UUID)
	}
	return uuids
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
