package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"httphq/src/database"
)

type requestListing struct {
	Requests []database.Request `json:"requests"`
	Total    int64              `json:"total"`
	Cursor   string             `json:"cursor"`
	HasMore  bool               `json:"hasMore"`
}

// listRequests reads back what an endpoint has captured, through the same API
// the page uses. An empty `since` is the browser's call, with no cursor.
func listRequests(t *testing.T, id, search, since string) requestListing {
	t.Helper()

	response := get(t, "/api/endpoints/"+id+"/requests?search="+search+
		"&since="+url.QueryEscape(since))
	require.Equal(t, http.StatusOK, response.StatusCode)

	var payload requestListing
	require.NoError(t, json.Unmarshal([]byte(bodyOf(t, response)), &payload))
	return payload
}

func capturedRequests(t *testing.T, id, search string) []database.Request {
	t.Helper()
	return listRequests(t, id, search, "").Requests
}

func listedTotal(t *testing.T, id, search string) int64 {
	t.Helper()
	return listRequests(t, id, search, "").Total
}

func TestHandleHealth(t *testing.T) {
	t.Run("answers 200 so a platform probe can reach it", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, get(t, "/api/health").StatusCode)
	})
}

func TestHandleDebug(t *testing.T) {
	t.Run("reports process state and no captured data", func(t *testing.T) {
		response := get(t, "/api/debug")
		require.Equal(t, http.StatusOK, response.StatusCode)

		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(bodyOf(t, response)), &payload))

		assert.Contains(t, payload, "host")
		assert.Contains(t, payload, "requests")
		assert.Equal(t, false, payload["isProduction"])
		assert.Equal(t, float64(0), payload["sockets"])
	})
}

// The listing orders newest-first or oldest-first depending on whether it was
// given a cursor, so neither end of the page is reliably the newest.
func TestNewestCreatedAt(t *testing.T) {
	base := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	at := func(offset time.Duration) database.Request {
		return database.Request{CreatedAt: base.Add(offset)}
	}

	t.Run("finds the newest wherever it sits in the page", func(t *testing.T) {
		newest := base.Add(3 * time.Minute)

		for name, page := range map[string][]database.Request{
			"first":  {at(3 * time.Minute), at(time.Minute), at(2 * time.Minute)},
			"last":   {at(time.Minute), at(2 * time.Minute), at(3 * time.Minute)},
			"middle": {at(time.Minute), at(3 * time.Minute), at(2 * time.Minute)},
		} {
			t.Run(name, func(t *testing.T) {
				assert.Equal(t, newest, newestCreatedAt(page))
			})
		}
	})

	t.Run("a single capture is its own newest", func(t *testing.T) {
		assert.Equal(t, base, newestCreatedAt([]database.Request{at(0)}))
	})
}

func TestHandleListRequests(t *testing.T) {
	t.Run("lists an endpoint's captures, newest first", func(t *testing.T) {
		id := endpointID(t)
		post(t, "/to/"+id, "older")
		post(t, "/to/"+id, "newer")

		captured := capturedRequests(t, id, "")

		require.Len(t, captured, 2)
		assert.Equal(t, "newer", captured[0].Body)
		assert.Equal(t, "older", captured[1].Body)
	})

	t.Run("the search parameter narrows the list", func(t *testing.T) {
		id := endpointID(t)
		post(t, "/to/"+id, "alpha")
		post(t, "/to/"+id, "beta")

		assert.Len(t, capturedRequests(t, id, "alpha"), 1)
		assert.Empty(t, capturedRequests(t, id, "no-such-term"))
	})

	// The listing is both filtered and windowed, so its length says nothing
	// about the endpoint. A control that clears the whole endpoint has to
	// report what it will actually delete.
	t.Run("the total is the endpoint's, not the filtered view's", func(t *testing.T) {
		id := endpointID(t)
		post(t, "/to/"+id, "alpha")
		post(t, "/to/"+id, "beta")

		assert.Equal(t, int64(2), listedTotal(t, id, ""))
		assert.Equal(t, int64(2), listedTotal(t, id, "alpha"))
		assert.Equal(t, int64(2), listedTotal(t, id, "no-such-term"))
	})

	t.Run("an endpoint with no traffic totals zero", func(t *testing.T) {
		assert.Equal(t, int64(0), listedTotal(t, endpointID(t), ""))
	})

	t.Run("an endpoint with no traffic lists nothing", func(t *testing.T) {
		assert.Empty(t, capturedRequests(t, endpointID(t), ""))
	})
}

// The cursor is what makes the listing pollable. Its promise is that echoing it
// back returns every capture exactly once, so these cover the round trip rather
// than the field's presence.
func TestHandleListRequestsCursor(t *testing.T) {
	t.Run("an endpoint with no traffic still carries a cursor", func(t *testing.T) {
		listing := listRequests(t, endpointID(t), "", "")

		assert.NotEmpty(t, listing.Cursor)
		assert.False(t, listing.HasMore)
	})

	// Without this the caller has nothing to advance from and would re-ask for
	// the same empty window forever.
	t.Run("the cursor from an empty endpoint is usable", func(t *testing.T) {
		id := endpointID(t)
		cursor := listRequests(t, id, "", "").Cursor

		post(t, "/to/"+id, "after")

		captured := listRequests(t, id, "", cursor).Requests
		require.Len(t, captured, 1)
		assert.Equal(t, "after", captured[0].Body)
	})

	t.Run("round-tripping the cursor returns only what is new", func(t *testing.T) {
		id := endpointID(t)
		post(t, "/to/"+id, "first")

		first := listRequests(t, id, "", "")
		require.Len(t, first.Requests, 1)

		post(t, "/to/"+id, "second")

		second := listRequests(t, id, "", first.Cursor)
		require.Len(t, second.Requests, 1)
		assert.Equal(t, "second", second.Requests[0].Body)
	})

	t.Run("a cursor with nothing behind it returns nothing but still advances", func(t *testing.T) {
		id := endpointID(t)
		post(t, "/to/"+id, "only")

		first := listRequests(t, id, "", "")
		second := listRequests(t, id, "", first.Cursor)

		assert.Empty(t, second.Requests)
		assert.NotEmpty(t, second.Cursor)
	})

	// A cursor that moved backwards would hand the caller captures it has
	// already processed, which is the one thing echoing it back promises not to
	// do. An empty page reports where the query ran, so a cursor ahead of that
	// has to survive the round trip unchanged.
	t.Run("a cursor is never handed back older than it was sent", func(t *testing.T) {
		ahead := time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano)

		listing := listRequests(t, endpointID(t), "", ahead)

		assert.Empty(t, listing.Requests)
		assert.Equal(t, ahead, listing.Cursor)
	})

	// total is what a delete-all will affect, so it stays endpoint-wide however
	// the listing is narrowed.
	t.Run("the total ignores the cursor", func(t *testing.T) {
		id := endpointID(t)
		post(t, "/to/"+id, "first")

		first := listRequests(t, id, "", "")

		assert.Equal(t, int64(1), listRequests(t, id, "", first.Cursor).Total)
	})

	// Ignoring an unparseable cursor would hand back the whole window, which a
	// caller cannot tell from a legitimate reply and would reprocess in full.
	t.Run("a malformed since is rejected rather than ignored", func(t *testing.T) {
		response := get(t, "/api/endpoints/"+endpointID(t)+"/requests?since=not-a-timestamp")

		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		assert.Contains(t, bodyOf(t, response), "RFC 3339")
	})

	t.Run("a plain RFC 3339 second-precision cursor is accepted", func(t *testing.T) {
		id := endpointID(t)
		post(t, "/to/"+id, "only")

		listing := listRequests(t, id, "", time.Now().UTC().Add(-time.Hour).Format(time.RFC3339))

		require.Len(t, listing.Requests, 1)
		assert.Equal(t, "only", listing.Requests[0].Body)
	})

	// hasMore is the signal a poller throttles on: it sleeps between calls
	// unless a page came back full, and a full page means the backlog is still
	// draining. Reporting it wrongly either stalls the drain or turns the poll
	// into a hot loop.
	t.Run("a full page reports more to come, and draining clears it", func(t *testing.T) {
		id := endpointID(t)
		for i := range requestPageSize + 1 {
			post(t, "/to/"+id, "burst-"+strconv.Itoa(i))
		}

		first := listRequests(t, id, "", "")
		require.Len(t, first.Requests, requestPageSize)
		assert.True(t, first.HasMore)

		second := listRequests(t, id, "", first.Cursor)
		assert.False(t, second.HasMore, "the tail of a burst is not a full page")
	})
}

func TestHandleDeleteRequest(t *testing.T) {
	t.Run("deleting one capture leaves the rest", func(t *testing.T) {
		id := endpointID(t)
		post(t, "/to/"+id, "keep")
		doomed := post(t, "/to/"+id, "delete")
		uuid := doomed.Header.Get(captureUUIDHeader)

		response := do(t, testRequest{
			method: http.MethodDelete,
			path:   "/api/endpoints/" + id + "/requests/" + uuid,
		})

		assert.Equal(t, http.StatusOK, response.StatusCode)
		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		assert.Equal(t, "keep", captured[0].Body)
	})
}

func TestHandleDeleteRequests(t *testing.T) {
	t.Run("deleting an endpoint's captures clears only that endpoint", func(t *testing.T) {
		id, other := endpointID(t), endpointID(t)+"-other"
		post(t, "/to/"+id, "x")
		post(t, "/to/"+other, "y")

		response := do(t, testRequest{
			method: http.MethodDelete,
			path:   "/api/endpoints/" + id + "/requests",
		})

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Empty(t, capturedRequests(t, id, ""))
		assert.Len(t, capturedRequests(t, other, ""), 1)
	})
}
