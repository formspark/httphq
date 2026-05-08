package database

import (
	"log"
	"strconv"
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
	log.Println("Connecting to database...")

	var err error

	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	log.Println("Connected to database")

	log.Println("Migrating database")

	if err := DB.AutoMigrate(&Request{}); err != nil {
		panic("failed to auto-migrate database")
	}

	log.Println("Migrated database")
	return DB
}

func CountRequests() int64 {
	var count int64
	result := DB.Model(&Request{}).Count(&count)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return count
}

func GetRequestsForEndpointID(endpointID string, search string, limit int) []Request {
	var items []Request
	result := DB.
		Where(&Request{EndpointID: endpointID}).
		Where("(? = '' OR (headers LIKE ? OR query_string LIKE ? OR body LIKE ?))",
			search, "%"+search+"%", "%"+search+"%", "%"+search+"%").
		Limit(limit).
		Order("created_at DESC").
		Find(&items)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return items
}

func CreateRequest(request *Request) {
	result := DB.Create(&request)
	if result.Error != nil {
		log.Println(result.Error)
	}
}

func DeleteRequestsForEndpointID(endpointID string) {
	result := DB.Where(&Request{EndpointID: endpointID}).Delete(&Request{})
	if result.Error != nil {
		log.Println(result.Error)
	}
}

func DeleteRequestForUUID(UUID string) {
	result := DB.Where(&Request{UUID: UUID}).Delete(&Request{})
	if result.Error != nil {
		log.Println(result.Error)
	}
}

func DeleteOldRequests(threshold time.Time) {
	result := DB.Where("created_at < ?", threshold).Delete(&Request{})
	if result.Error != nil {
		log.Println(result.Error)
	}
	log.Println("Deleted " + strconv.Itoa(int(result.RowsAffected)) + " old requests")
}
