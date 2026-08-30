package database_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"httphq/src/database"
)

// freshDB gives a test its own empty in-memory database, so no test depends on
// what another one left behind.
func freshDB(t *testing.T) {
	t.Helper()
	database.Connect(":memory:")
}

// captureLogs points the process logger at a buffer for one test. Every
// operation here answers with a zero value, so a failure it ran into is
// reported through the logs or nowhere, and this is the only way to read it.
// The logger is process-wide, so it is restored before the next test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var out bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&out, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &out
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

// Every operation answers with a zero value rather than an error, because the
// caller is a request handler that still has to respond. That makes a broken
// store indistinguishable from an empty one at every call site, so the log line
// is the only report there is and it has to be there.
func TestBrokenStore(t *testing.T) {
	t.Run("a failed query answers empty and reports itself to the logs", func(t *testing.T) {
		freshDB(t)
		logs := captureLogs(t)
		require.NoError(t, database.DB.Migrator().DropTable(&database.Request{}))

		assert.Equal(t, int64(0), database.CountRequests(t.Context()))
		assert.Contains(t, logs.String(), "count requests failed")
	})

	t.Run("a failed listing answers with no requests at all", func(t *testing.T) {
		freshDB(t)
		logs := captureLogs(t)
		require.NoError(t, database.DB.Migrator().DropTable(&database.Request{}))

		listed := database.GetRequestsForEndpointID(t.Context(), "any-endpoint", "", time.Time{}, 10)

		assert.Empty(t, listed, "a caller must not be handed a half-read page")
		assert.Contains(t, logs.String(), "get requests failed")
		assert.Contains(t, logs.String(), "any-endpoint",
			"the log names the endpoint, or an operator cannot tell which one broke")
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
			database.GetRequestsForEndpointID(t.Context(), "test-id", "", time.Time{}, 32))
	})

	t.Run("round-trips every stored field", func(t *testing.T) {
		freshDB(t)

		stored := storeRequest(t.Context(), "uuid-1",
			withBody("test-body-1"),
			withHeaders(`{ "Test": "Test-Header-1" }`),
			withQueryString("a=1"))

		items := database.GetRequestsForEndpointID(t.Context(), stored.EndpointID, "", time.Time{}, 1)

		assert.Len(t, items, 1)
		assert.Equal(t, stored.UUID, items[0].UUID)
		assert.Equal(t, stored.EndpointID, items[0].EndpointID)
		assert.Equal(t, stored.IP, items[0].IP)
		assert.Equal(t, stored.Method, items[0].Method)
		assert.Equal(t, stored.Path, items[0].Path)
		assert.Equal(t, stored.QueryString, items[0].QueryString)
		assert.Equal(t, stored.Body, items[0].Body)
		assert.Equal(t, stored.Headers, items[0].Headers)
		assert.Equal(t, time.Now().UTC().Format(time.ANSIC), items[0].CreatedAt.UTC().Format(time.ANSIC))
	})

	t.Run("returns only the requested endpoint's requests", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withEndpointID("wanted"))
		storeRequest(t.Context(), "uuid-2", withEndpointID("wanted"))
		storeRequest(t.Context(), "uuid-3", withEndpointID("other"))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "wanted", "", time.Time{}, 32), 2)
	})

	t.Run("orders newest first", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withBody("older"))
		storeRequest(t.Context(), "uuid-2", withBody("newer"))

		items := database.GetRequestsForEndpointID(t.Context(), "test-id", "", time.Time{}, 32)

		assert.Equal(t, "newer", items[0].Body)
		assert.Equal(t, "older", items[1].Body)
	})

	t.Run("never returns more than the limit", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1")
		storeRequest(t.Context(), "uuid-2")

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "", time.Time{}, 1), 1)
	})

	t.Run("an empty search applies no filtering", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withBody("alpha"))
		storeRequest(t.Context(), "uuid-2", withBody("beta"))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "", time.Time{}, 32), 2)
	})

	t.Run("searches the body", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withBody("test-body-1"))
		storeRequest(t.Context(), "uuid-2", withBody("test-body-2"))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "test-body", time.Time{}, 32), 2)
		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "test-body-1", time.Time{}, 32), 1)
	})

	t.Run("searches the headers", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withHeaders(`{ "Test": "Test-Header-1" }`))
		storeRequest(t.Context(), "uuid-2", withHeaders(`{ "Test": "Test-Header-2" }`))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "Test-Header", time.Time{}, 32), 2)
		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "Test-Header-1", time.Time{}, 32), 1)
	})

	t.Run("searches the query string", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withQueryString("event=charge.succeeded"))
		storeRequest(t.Context(), "uuid-2", withQueryString("event=charge.failed"))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "event=charge", time.Time{}, 32), 2)
		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "charge.failed", time.Time{}, 32), 1)
	})

	t.Run("a search that matches nothing returns an empty slice", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "uuid-1", withBody("alpha"))

		assert.Empty(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "no-such-term", time.Time{}, 32))
	})
}

// uuidsOf names the captures in a page, in the order they came back. Ordering
// is half of what the cursor guarantees, so the assertions have to see it.
func uuidsOf(requests []database.Request) []string {
	uuids := make([]string, 0, len(requests))
	for _, request := range requests {
		uuids = append(uuids, request.UUID)
	}
	return uuids
}

// drainWithCursor reads an endpoint's whole stream the way a poller does: a page
// at a time, advancing the cursor to the last capture it was handed, until a
// page comes back empty. Returns the UUIDs it was given, in order, and how many
// calls it took.
func drainWithCursor(t *testing.T, endpointID string, cursor time.Time, limit int) ([]string, int) {
	t.Helper()

	var seen []string
	calls := 0
	for {
		page := database.GetRequestsForEndpointID(t.Context(), endpointID, "", cursor, limit)
		if len(page) == 0 {
			return seen, calls
		}
		calls++
		require.Less(t, calls, 10, "the drain is not converging")
		seen = append(seen, uuidsOf(page)...)
		cursor = page[len(page)-1].CreatedAt
	}
}

func TestGetRequestsForEndpointIDSince(t *testing.T) {
	// A fixed base keeps the ordering assertions readable and keeps the rows
	// clear of time.Now(), so nothing depends on how fast the test runs.
	base := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	at := func(offset time.Duration) time.Time { return base.Add(offset) }

	t.Run("returns only captures strictly newer than the cursor", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "before", withCreatedAt(at(-time.Minute)))
		storeRequest(t.Context(), "on-the-cursor", withCreatedAt(base))
		storeRequest(t.Context(), "after", withCreatedAt(at(time.Minute)))

		items := database.GetRequestsForEndpointID(t.Context(), "test-id", "", base, 32)

		assert.Equal(t, []string{"after"}, uuidsOf(items))
	})

	t.Run("returns them oldest first", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "third", withCreatedAt(at(3*time.Minute)))
		storeRequest(t.Context(), "first", withCreatedAt(at(time.Minute)))
		storeRequest(t.Context(), "second", withCreatedAt(at(2*time.Minute)))

		items := database.GetRequestsForEndpointID(t.Context(), "test-id", "", base, 32)

		assert.Equal(t, []string{"first", "second", "third"}, uuidsOf(items))
	})

	t.Run("a zero cursor still returns newest first", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "older", withCreatedAt(at(time.Minute)))
		storeRequest(t.Context(), "newer", withCreatedAt(at(2*time.Minute)))

		items := database.GetRequestsForEndpointID(t.Context(), "test-id", "", time.Time{}, 32)

		assert.Equal(t, []string{"newer", "older"}, uuidsOf(items))
	})

	t.Run("a zero cursor is still limited", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "older", withCreatedAt(at(time.Minute)))
		storeRequest(t.Context(), "newer", withCreatedAt(at(2*time.Minute)))

		assert.Len(t, database.GetRequestsForEndpointID(t.Context(), "test-id", "", time.Time{}, 1), 1)
	})

	t.Run("the search and the cursor compose", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "old-match", withBody("alpha"), withCreatedAt(at(-time.Minute)))
		storeRequest(t.Context(), "new-miss", withBody("beta"), withCreatedAt(at(time.Minute)))
		storeRequest(t.Context(), "new-match", withBody("alpha"), withCreatedAt(at(2*time.Minute)))

		items := database.GetRequestsForEndpointID(t.Context(), "test-id", "alpha", base, 32)

		assert.Equal(t, []string{"new-match"}, uuidsOf(items))
	})

	t.Run("the cursor is scoped to its endpoint", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "wanted", withCreatedAt(at(time.Minute)))
		storeRequest(t.Context(), "other", withEndpointID("other-id"), withCreatedAt(at(time.Minute)))

		items := database.GetRequestsForEndpointID(t.Context(), "test-id", "", base, 32)

		assert.Equal(t, []string{"wanted"}, uuidsOf(items))
	})

	// The one that matters. With DESC and a full page, a burst larger than the
	// limit hands back its newest captures, the caller's cursor advances past
	// everything older, and the backlog is never served. Draining has to reach
	// every capture exactly once.
	t.Run("a burst larger than the limit drains with nothing lost or repeated", func(t *testing.T) {
		freshDB(t)

		const (
			burst = 25
			limit = 10
		)
		// The expectation is built from the same names that were stored, in the
		// order they were stored, so the assertion cannot drift from the fixture.
		want := make([]string, 0, burst)
		for i := range burst {
			uuid := fmt.Sprintf("capture-%02d", i)
			want = append(want, uuid)
			storeRequest(t.Context(), uuid, withCreatedAt(at(time.Duration(i)*time.Second)))
		}

		seen, calls := drainWithCursor(t, "test-id", base.Add(-time.Second), limit)

		assert.Equal(t, want, seen, "every capture exactly once, in order")
		assert.Equal(t, 3, calls, "25 captures at 10 a page is three pages")
	})
}

func TestCreateRequest(t *testing.T) {
	t.Run("stores a retrievable request stamped with the current time", func(t *testing.T) {
		freshDB(t)

		storeRequest(t.Context(), "test-uuid")

		items := database.GetRequestsForEndpointID(t.Context(), "test-id", "", time.Time{}, 1)

		assert.Equal(t, "test-uuid", items[0].UUID)
		assert.Equal(t, time.Now().UTC().Format(time.ANSIC), items[0].CreatedAt.UTC().Format(time.ANSIC))
	})

	// created_at is compared and ordered as text, so a row carrying a zone
	// offset would be ranked against the others by its digits rather than its
	// instant. Every write normalises, whatever zone the caller was holding.
	t.Run("stamps in UTC whatever zone the caller supplies", func(t *testing.T) {
		freshDB(t)

		brussels, err := time.LoadLocation("Europe/Brussels")
		require.NoError(t, err)
		stamp := time.Date(2026, 8, 8, 19, 36, 13, 0, brussels)

		storeRequest(t.Context(), "zoned-uuid", withCreatedAt(stamp))

		items := database.GetRequestsForEndpointID(t.Context(), "test-id", "", time.Time{}, 1)

		assert.Equal(t, time.UTC, items[0].CreatedAt.Location())
		assert.True(t, stamp.Equal(items[0].CreatedAt), "the instant must survive the conversion")
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
			database.GetRequestsForEndpointID(t.Context(), "keep-id", "", time.Time{}, 1)[0].EndpointID)
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
			database.GetRequestsForEndpointID(t.Context(), "test-id", "", time.Time{}, 1)[0].UUID)
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
