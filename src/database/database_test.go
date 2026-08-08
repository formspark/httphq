package database_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"

	"httphq/src/database"
)

// freshDB gives a test its own empty in-memory database, so no test depends on
// what another one left behind.
func freshDB(t *testing.T) {
	t.Helper()
	database.Connect(":memory:")
}

// requestOption customises a fixture request. Only the fields a test asserts on
// need naming at the call site; everything else stays at a valid default.
type requestOption func(*database.Request)

func withEndpointID(id string) requestOption {
	return func(r *database.Request) { r.EndpointID = id }
}

func withBody(body string) requestOption {
	return func(r *database.Request) { r.Body = body }
}

func withHeaders(json string) requestOption {
	return func(r *database.Request) { r.Headers = datatypes.JSON(json) }
}

func withQueryString(query string) requestOption {
	return func(r *database.Request) { r.QueryString = query }
}

func withCreatedAt(at time.Time) requestOption {
	return func(r *database.Request) { r.CreatedAt = at }
}

// storeRequest writes a valid request and returns it. The defaults are
// deliberately uninteresting: a test that cares about a field sets it.
func storeRequest(ctx context.Context, uuid string, options ...requestOption) database.Request {
	request := database.Request{
		UUID:       uuid,
		EndpointID: "test-id",
		IP:         "test-ip",
		Method:     "GET",
		Path:       "/test",
		Body:       "test-body",
		Headers:    datatypes.JSON(`{ "Test": "Test-Header" }`),
	}
	for _, option := range options {
		option(&request)
	}
	database.CreateRequest(ctx, &request)
	return request
}

func TestConnect(t *testing.T) {
	t.Run("migrates every table the application needs", func(t *testing.T) {
		freshDB(t)

		var tables []string
		database.DB.Raw(`SELECT name FROM sqlite_schema WHERE type = 'table' ORDER BY name`).Scan(&tables)
		assert.Equal(t, []string{"requests"}, tables)
	})
}

func TestCountRequests(t *testing.T) {
	t.Run("returns zero when nothing is stored", func(t *testing.T) {
		freshDB(t)

		assert.Equal(t, int64(0), database.CountRequests(t.Context()))
	})

	t.Run("counts every stored request", func(t *testing.T) {
		freshDB(t)

		const stored = 3
		for i := range stored {
			storeRequest(t.Context(), fmt.Sprint(i), withEndpointID(fmt.Sprint(i)))
		}

		assert.Equal(t, int64(stored), database.CountRequests(t.Context()))
	})
}

func TestCountRequestsForEndpointID(t *testing.T) {
	t.Run("counts only the named endpoint's requests", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withEndpointID("wanted"))
		storeRequest(t.Context(), "uuid-2", withEndpointID("wanted"))
		storeRequest(t.Context(), "uuid-3", withEndpointID("other"))

		assert.Equal(t, int64(2), database.CountRequestsForEndpointID(t.Context(), "wanted"))
	})

	t.Run("returns zero for an endpoint with no traffic", func(t *testing.T) {
		freshDB(t)

		assert.Equal(t, int64(0), database.CountRequestsForEndpointID(t.Context(), "never-used"))
	})
}

func TestGetRequestsForEndpointID(t *testing.T) {
	t.Run("returns an empty slice when nothing is stored", func(t *testing.T) {
		freshDB(t)

		assert.Equal(t, []database.Request{},
			database.GetRequestsForEndpointID(t.Context(), "test-id", "", 32))
	})

	t.Run("round-trips every stored field", func(t *testing.T) {
		freshDB(t)

		stored := storeRequest(t.Context(), "uuid-1",
			withBody("test-body-1"),
			withHeaders(`{ "Test": "Test-Header-1" }`),
			withQueryString("a=1"))

		items := database.GetRequestsForEndpointID(t.Context(), stored.EndpointID, "", 1)

		assert.Len(t, items, 1)
		assert.Equal(t, stored.UUID, items[0].UUID)
		assert.Equal(t, stored.EndpointID, items[0].EndpointID)
		assert.Equal(t, stored.IP, items[0].IP)
		assert.Equal(t, stored.Method, items[0].Method)
		assert.Equal(t, stored.Path, items[0].Path)
		assert.Equal(t, stored.QueryString, items[0].QueryString)
		assert.Equal(t, stored.Body, items[0].Body)
		assert.Equal(t, stored.Headers, items[0].Headers)
		assert.Equal(t, time.Now().Format(time.ANSIC), items[0].CreatedAt.Format(time.ANSIC))
	})

	t.Run("returns only the requested endpoint's requests", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withEndpointID("wanted"))
		storeRequest(t.Context(), "uuid-2", withEndpointID("wanted"))
		storeRequest(t.Context(), "uuid-3", withEndpointID("other"))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "wanted", "", 32), 2)
	})

	t.Run("orders newest first", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withBody("older"))
		storeRequest(t.Context(), "uuid-2", withBody("newer"))

		items := database.GetRequestsForEndpointID(t.Context(), "test-id", "", 32)

		assert.Equal(t, "newer", items[0].Body)
		assert.Equal(t, "older", items[1].Body)
	})

	t.Run("never returns more than the limit", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1")
		storeRequest(t.Context(), "uuid-2")

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "", 1), 1)
	})

	t.Run("an empty search applies no filtering", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withBody("alpha"))
		storeRequest(t.Context(), "uuid-2", withBody("beta"))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "", 32), 2)
	})

	t.Run("searches the body", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withBody("test-body-1"))
		storeRequest(t.Context(), "uuid-2", withBody("test-body-2"))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "test-body", 32), 2)
		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "test-body-1", 32), 1)
	})

	t.Run("searches the headers", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withHeaders(`{ "Test": "Test-Header-1" }`))
		storeRequest(t.Context(), "uuid-2", withHeaders(`{ "Test": "Test-Header-2" }`))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "Test-Header", 32), 2)
		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "Test-Header-1", 32), 1)
	})

	t.Run("searches the query string", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withQueryString("event=charge.succeeded"))
		storeRequest(t.Context(), "uuid-2", withQueryString("event=charge.failed"))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "event=charge", 32), 2)
		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "charge.failed", 32), 1)
	})

	t.Run("a search that matches nothing returns an empty slice", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withBody("alpha"))

		assert.Empty(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "no-such-term", 32))
	})
}

func TestCreateRequest(t *testing.T) {
	t.Run("stores a retrievable request stamped with the current time", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "test-uuid")

		items := database.GetRequestsForEndpointID(t.Context(), "test-id", "", 1)

		assert.Equal(t, "test-uuid", items[0].UUID)
		assert.Equal(t, time.Now().Format(time.ANSIC), items[0].CreatedAt.Format(time.ANSIC))
	})
}

func TestDeleteRequestsForEndpointID(t *testing.T) {
	t.Run("deletes only the named endpoint's requests", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "test-uuid-1", withEndpointID("delete-id"))
		storeRequest(t.Context(), "test-uuid-2", withEndpointID("keep-id"))

		database.DeleteRequestsForEndpointID(t.Context(), "delete-id")

		assert.Equal(t, int64(1), database.CountRequests(t.Context()))
		assert.Equal(t, "keep-id",
			database.GetRequestsForEndpointID(t.Context(), "keep-id", "", 1)[0].EndpointID)
	})
}

func TestDeleteRequestForUUID(t *testing.T) {
	t.Run("deletes only the named request", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "delete-uuid")
		storeRequest(t.Context(), "keep-uuid")

		database.DeleteRequestForUUID(t.Context(), "delete-uuid")

		assert.Equal(t, int64(1), database.CountRequests(t.Context()))
		assert.Equal(t, "keep-uuid",
			database.GetRequestsForEndpointID(t.Context(), "test-id", "", 1)[0].UUID)
	})
}

func TestDeleteOldRequests(t *testing.T) {
	threshold := time.Now().Add(-4 * time.Hour)

	t.Run("deletes requests created before the threshold", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-delete", withCreatedAt(threshold.Add(-time.Hour)))

		database.DeleteOldRequests(t.Context(), threshold)

		assert.Equal(t, int64(0), database.CountRequests(t.Context()))
	})

	t.Run("keeps requests created after the threshold", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-keep", withCreatedAt(threshold.Add(time.Hour)))

		database.DeleteOldRequests(t.Context(), threshold)

		assert.Equal(t, int64(1), database.CountRequests(t.Context()))
	})
}
