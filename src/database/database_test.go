package database

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConnect(t *testing.T) {
	var result []string
	Connect(":memory:")

	// It should create all tables
	DB.Raw(`SELECT name FROM sqlite_schema WHERE type = 'table' ORDER BY name`).Scan(&result)
	assert.Equal(t, []string{"requests", "socket_clients"}, result)
}
