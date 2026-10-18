# Database

The SQLite store under `src/database` and the test that holds its schema.

## Database standards

`TestDatabaseStandards` in `src/database` asks the schema AutoMigrate actually
produces, rather than the struct tags that ask for it: snake_case tables and
columns, a `created_at`, and an index on every column a query filters or orders
by. A `gorm:"column:..."` tag naming something else satisfies the struct and
fails the test, which is the point.

There are no migration files here and no information_schema to query, so these
questions go through `sqlite_schema` and the pragmas.

Nothing carries `updated_at`, because a capture is written once and never
edited, so the test does not ask for one.

## Captured data is ephemeral

SQLite writes to the container's writable layer, so capture history is lost on
restart. That is a product decision, recorded in PRODUCT.md under Technical
constraints and Product Principles: there is no durable store and no migration
path.
