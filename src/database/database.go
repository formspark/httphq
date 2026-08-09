package database

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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

func Connect(dsn string) *gorm.DB {
	slog.Info("connecting to database")

	var err error

	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})

	if err != nil {
		slog.Error("database connection failed", "err", err)
		os.Exit(1)
	}

	slog.Info("database connected")

	slog.Info("migrating database")

	if err := DB.AutoMigrate(&Request{}); err != nil {
		slog.Error("database migration failed", "err", err)
		os.Exit(1)
	}

	slog.Info("database migrated")
	return DB
}

func CountRequests(ctx context.Context) int64 {
	var count int64
	result := DB.Model(&Request{}).Count(&count)
	if result.Error != nil {
		slog.ErrorContext(ctx, "count requests failed", "err", result.Error)
	}
	return count
}

// CountRequestsForEndpointID counts everything stored for an endpoint,
// ignoring any search. The listing is both filtered and windowed, so it is the
// only way to say how much a control that acts on the whole endpoint will
// affect.
func CountRequestsForEndpointID(ctx context.Context, endpointID string) int64 {
	var count int64
	result := DB.Model(&Request{}).Where(&Request{EndpointID: endpointID}).Count(&count)
	if result.Error != nil {
		slog.ErrorContext(ctx, "count requests for endpoint failed", "err", result.Error, "endpoint_id", endpointID)
	}
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
	if result.Error != nil {
		slog.ErrorContext(ctx, "get requests failed", "err", result.Error, "endpoint_id", endpointID)
	}
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
func CreateRequest(ctx context.Context, request *Request) {
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now()
	}
	request.CreatedAt = request.CreatedAt.UTC()

	result := DB.Create(&request)
	if result.Error != nil {
		slog.ErrorContext(ctx, "create request failed", "err", result.Error)
	}
}

func DeleteRequestsForEndpointID(ctx context.Context, endpointID string) {
	result := DB.Where(&Request{EndpointID: endpointID}).Delete(&Request{})
	if result.Error != nil {
		slog.ErrorContext(ctx, "delete requests failed", "err", result.Error, "endpoint_id", endpointID)
	}
}

func DeleteRequestForUUID(ctx context.Context, UUID string) {
	result := DB.Where(&Request{UUID: UUID}).Delete(&Request{})
	if result.Error != nil {
		slog.ErrorContext(ctx, "delete request failed", "err", result.Error)
	}
}

func DeleteOldRequests(ctx context.Context, threshold time.Time) {
	// UTC for the same reason as the cursor in GetRequestsForEndpointID: the
	// comparison is textual, so both sides have to agree on a zone.
	result := DB.Where("created_at < ?", threshold.UTC()).Delete(&Request{})
	if result.Error != nil {
		slog.ErrorContext(ctx, "delete old requests failed", "err", result.Error)
	}
	slog.InfoContext(ctx, "deleted old requests", "count", result.RowsAffected)
}
