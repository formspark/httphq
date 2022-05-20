package database

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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

	// It should return 0 if no requests exist
	assert.Equal(t, int64(0), CountRequests())

	// It should return the amount of existing requests
	var n = 3
	for i := 0; i < n; i++ {
		ID := fmt.Sprint(i)
		request := Request{
			UUID:       ID,
			EndpointID: ID,
			IP:         ID,
			Method:     "GET",
			Path:       "/test",
			Body:       "test",
			CreatedAt:  time.Time{},
			Headers:    nil,
		}
		CreateRequest(&request)
	}
	assert.Equal(t, int64(n), CountRequests())
}
