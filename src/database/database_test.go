package database

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConnect(t *testing.T) {
	Connect(":memory:")

	// It should create all tables
	var tables []string
	DB.Raw(`SELECT name FROM sqlite_schema WHERE type = 'table' ORDER BY name`).Scan(&tables)
	assert.Equal(t, []string{"requests", "socket_clients"}, tables)
}

func TestCountRequests(t *testing.T) {
	Connect(":memory:")

	// It should return 0 if no items exist
	assert.Equal(t, int64(0), CountRequests())

	// It should return the amount of existing items
	var n = 3
	for i := 0; i < n; i++ {
		ID := fmt.Sprint(i)
		CreateRequest(&Request{
			UUID:       ID,
			EndpointID: ID,
			IP:         ID,
			Method:     "GET",
			Path:       "/test",
			Body:       "test",
		})
	}
	assert.Equal(t, int64(n), CountRequests())
}

func TestGetRequestsForEndpointID(t *testing.T) {
	Connect(":memory:")

	endpointID := "test-id"

	var items []Request

	// It should return an empty array if no items exist
	items = GetRequestsForEndpointID(endpointID, "", 32)
	assert.Equal(t, []Request{}, items)

	CreateRequest(&Request{
		UUID:       "uuid-1",
		EndpointID: endpointID,
		IP:         "test-ip",
		Method:     "GET",
		Path:       "/test",
		Body:       "test-body-1",
	})
	CreateRequest(&Request{
		UUID:       "uuid-2",
		EndpointID: endpointID,
		IP:         "test-ip",
		Method:     "GET",
		Path:       "/test",
		Body:       "test-body-2",
	})
	CreateRequest(&Request{
		UUID:       "uuid-3",
		EndpointID: "other-id",
		IP:         "test-ip",
		Method:     "GET",
		Path:       "/test",
		Body:       "test-body-3",
	})

	// It should return items with the correct shape
	// TODO

	// It should only return items with the specified endpoint id
	items = GetRequestsForEndpointID(endpointID, "", 32)
	assert.Equal(t, 2, len(items))

	// It should not return more items than the limit
	items = GetRequestsForEndpointID(endpointID, "", 1)
	assert.Equal(t, 1, len(items))

	// It should return return items ordered by creation date
	// TODO

	// It should not apply any additional filtering if the search string is empty
	items = GetRequestsForEndpointID(endpointID, "", 32)
	assert.Equal(t, 2, len(items))

	// It should search headers and body based on the search string
	items = GetRequestsForEndpointID(endpointID, "test-body", 32)
	assert.Equal(t, 2, len(items))

	items = GetRequestsForEndpointID(endpointID, "test-body-1", 32)
	assert.Equal(t, 1, len(items))
}
