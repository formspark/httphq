package database_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"httphq/src/database"
)

// The database standards, asked of the schema AutoMigrate actually produces
// rather than of the struct tags that ask for it. The sibling repositories run
// the equivalent as SQL against Postgres after applying their migrations; there
// are no migration files here and no information_schema to query, so the same
// questions are asked of sqlite_schema and the pragmas instead.
//
// Two of their four checks do not apply. There is no email column anywhere, and
// nothing carries updated_at: a capture is written once and never edited, which
// is the whole shape of this product, so a column nothing writes would be
// overhead on the only table there is.

var snakeCase = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// columnsOf reads the schema the driver built, not the model that asked for it.
// A `gorm:"column:..."` tag naming something else would satisfy the struct and
// fail here, which is the point.
func columnsOf(table string) []string {
	var columns []string
	database.DB.Raw(`SELECT name FROM pragma_table_info(?)`, table).Scan(&columns)
	return columns
}

// indexedColumns returns every column carrying an index, including the ones
// GORM creates from an `index` tag and the implicit primary key.
func indexedColumns(table string) map[string]bool {
	var indexes []string
	database.DB.Raw(`SELECT name FROM pragma_index_list(?)`, table).Scan(&indexes)

	indexed := map[string]bool{}
	for _, index := range indexes {
		var columns []string
		database.DB.Raw(`SELECT name FROM pragma_index_info(?)`, index).Scan(&columns)
		for _, column := range columns {
			indexed[column] = true
		}
	}
	return indexed
}

func TestDatabaseStandards(t *testing.T) {
	t.Run("names every table in snake_case", func(t *testing.T) {
		freshDB(t)

		var tables []string
		database.DB.Raw(`SELECT name FROM sqlite_schema WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&tables)

		for _, table := range tables {
			assert.Regexp(t, snakeCase, table)
		}
	})

	t.Run("names every column in snake_case", func(t *testing.T) {
		freshDB(t)

		for _, column := range columnsOf("requests") {
			assert.Regexp(t, snakeCase, column)
		}
	})

	t.Run("records when a row was written", func(t *testing.T) {
		freshDB(t)

		assert.Contains(t, columnsOf("requests"), "created_at")
	})

	// The three the application filters or orders by. The retention sweep reads
	// created_at, the endpoint screen reads endpoint_id, and the rate limiter
	// reads ip. An unindexed one of these is a full scan on the only table
	// there is, and nothing else would report it.
	t.Run("indexes every column a query filters or orders by", func(t *testing.T) {
		freshDB(t)

		indexed := indexedColumns("requests")
		for _, column := range []string{"created_at", "endpoint_id", "ip"} {
			assert.True(t, indexed[column], "expected an index on %s", column)
		}
	})
}
