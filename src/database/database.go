package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Request struct {
	UUID        string         `json:"uuid" gorm:"primaryKey"`
	EndpointID  string         `json:"endpointId" gorm:"index"`
	IP          string         `json:"ip" gorm:"index"`
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	QueryString string         `json:"queryString"`
	Body        string         `json:"body"`
	CreatedAt   time.Time      `json:"createdAt" gorm:"index"`
	Headers     datatypes.JSON `json:"headers"`
}

var DB *gorm.DB

// logQueryError records a failed query without any of its values. A read
// answers with a zero value rather than an error: the caller is a request
// handler that still has to respond, and a capture surface that comes back
// empty is more use than one that refuses to render. A write also returns its
// error, because the caller must not tell a sender or a viewer that something
// was stored or deleted when it was not.
func logQueryError(ctx context.Context, result *gorm.DB, message string, attrs ...any) {
	if result.Error == nil {
		return
	}
	slog.ErrorContext(ctx, message, append([]any{"err", result.Error}, attrs...)...)
}

// Connect opens the store and brings the schema up to date, publishing it as DB
// only once both have succeeded. There is nothing to serve without somewhere
// to write captures, so the caller treats an error here as fatal; the package
// returns it rather than ending the process, so the failure can be tested.
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		// GORM's default logger prints slow and failed statements to stdout with
		// every bound value inlined, which for a capture means its headers and
		// body. Failures are reported by logQueryError instead.
		Logger: logger.Discard,
		// Every write here is a single statement, so GORM's wrapping transaction
		// adds nothing. It also outlives a commit that fails, leaving the pooled
		// connection holding the write lock.
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open the database: %w", err)
	}
	if err := db.AutoMigrate(&Request{}); err != nil {
		return nil, fmt.Errorf("migrate the database: %w", err)
	}

	DB = db
	slog.Info("database connected and migrated")
	return DB, nil
}

// Close closes the connection pool behind DB. The last connection to close
// checkpoints SQLite's write-ahead log into the database file.
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("reach the connection pool: %w", err)
	}
	return sqlDB.Close()
}

func CountRequests(ctx context.Context) int64 {
	var count int64
	result := DB.Model(&Request{}).Count(&count)
	logQueryError(ctx, result, "count requests failed")
	return count
}

// CountRequestsForEndpointID counts everything stored for an endpoint,
// ignoring any search. The listing is both filtered and windowed, so it is the
// only way to say how much a control that acts on the whole endpoint will
// affect.
func CountRequestsForEndpointID(ctx context.Context, endpointID string) int64 {
	var count int64
	result := DB.Model(&Request{}).Where(&Request{EndpointID: endpointID}).Count(&count)
	logQueryError(ctx, result, "count requests for endpoint failed", "endpoint_id", endpointID)
	return count
}

// GetRequestsForEndpointID returns a window of an endpoint's captures. A zero
// `since` means no cursor.
//
// The ordering flips with the cursor, and that is a correctness rule rather
// than a preference. A cursored caller reads a stream it intends to consume
// whole, so it has to be handed the OLDEST unseen page: with DESC and a full
// page, a burst larger than the limit returns the newest captures, the caller's
// cursor advances past the rest, and nothing ever goes back for them. ASC hands
// out the backlog in order and the next call resumes where this page ended.
func GetRequestsForEndpointID(
	ctx context.Context, endpointID string, search string, since time.Time, limit int,
) []Request {
	var items []Request
	query := DB.
		Where(&Request{EndpointID: endpointID}).
		Where("(? = '' OR (headers LIKE ? OR query_string LIKE ? OR body LIKE ?))",
			search, "%"+search+"%", "%"+search+"%", "%"+search+"%").
		Limit(limit)

	if since.IsZero() {
		query = query.Order("created_at DESC")
	} else {
		// UTC to match how CreateRequest stores the column. A bound value keeps
		// whatever zone it arrives in, and the comparison is textual.
		query = query.Where("created_at > ?", since.UTC()).Order("created_at ASC")
	}

	result := query.Find(&items)
	logQueryError(ctx, result, "get requests failed", "endpoint_id", endpointID)
	return items
}

// CreateRequest stores a capture, stamping it if the caller did not.
//
// CreatedAt is forced to UTC because SQLite has no date type and this column is
// compared and ordered as text. A row written with a +02:00 offset is ranked
// against one written with +00:00 by their digits rather than their instants,
// so a single row in the wrong zone breaks both `created_at > ?` and
// `ORDER BY created_at` for every row around it. Normalising on the way in is
// what lets the rest of this file treat those clauses as chronological.
func CreateRequest(ctx context.Context, request *Request) error {
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now()
	}
	request.CreatedAt = request.CreatedAt.UTC()

	result := DB.Create(&request)
	logQueryError(ctx, result, "create request failed")
	return result.Error
}

func DeleteRequestsForEndpointID(ctx context.Context, endpointID string) error {
	result := DB.Where(&Request{EndpointID: endpointID}).Delete(&Request{})
	logQueryError(ctx, result, "delete requests failed", "endpoint_id", endpointID)
	return result.Error
}

func DeleteRequestForUUID(ctx context.Context, UUID string) error {
	result := DB.Where(&Request{UUID: UUID}).Delete(&Request{})
	logQueryError(ctx, result, "delete request failed")
	return result.Error
}

func DeleteOldRequests(ctx context.Context, threshold time.Time) {
	// UTC for the same reason as the cursor in GetRequestsForEndpointID: the
	// comparison is textual, so both sides have to agree on a zone.
	result := DB.Where("created_at < ?", threshold.UTC()).Delete(&Request{})
	logQueryError(ctx, result, "delete old requests failed")
	slog.InfoContext(ctx, "deleted old requests", "count", result.RowsAffected)
}
