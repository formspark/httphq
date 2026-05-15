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

func GetRequestsForEndpointID(ctx context.Context, endpointID string, search string, limit int) []Request {
	var items []Request
	result := DB.
		Where(&Request{EndpointID: endpointID}).
		Where("(? = '' OR (headers LIKE ? OR query_string LIKE ? OR body LIKE ?))",
			search, "%"+search+"%", "%"+search+"%", "%"+search+"%").
		Limit(limit).
		Order("created_at DESC").
		Find(&items)
	if result.Error != nil {
		slog.ErrorContext(ctx, "get requests failed", "err", result.Error, "endpoint_id", endpointID)
	}
	return items
}

func CreateRequest(ctx context.Context, request *Request) {
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
	result := DB.Where("created_at < ?", threshold).Delete(&Request{})
	if result.Error != nil {
		slog.ErrorContext(ctx, "delete old requests failed", "err", result.Error)
	}
	slog.InfoContext(ctx, "deleted old requests", "count", result.RowsAffected)
}
