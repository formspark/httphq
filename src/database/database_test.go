package database

import (
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
