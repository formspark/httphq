package database

import (
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"strconv"
	"time"
)

type Request struct {
	UUID       string         `json:"uuid" gorm:"primaryKey"`
	EndpointID string         `json:"endpointId" gorm:"index"`
	IP         string         `json:"ip" gorm:"index"`
	Method     string         `json:"method"`
	Path       string         `json:"path"`
	Body       string         `json:"body"`
	CreatedAt  time.Time      `json:"createdAt" gorm:"index"`
	Headers    datatypes.JSON `json:"headers"`
}

type SocketClient struct {
	UUID       string    `json:"uuid" gorm:"primaryKey"`
	EndpointID string    `json:"endpointId" gorm:"index"`
	CreatedAt  time.Time `json:"createdAt"`
}

var db *gorm.DB

func Connect() *gorm.DB {
	log.Println("Connecting to database...")

	var err error

	db, err = gorm.Open(sqlite.Open("local.db"), &gorm.Config{})

	if err != nil {
		panic("failed to connect database")
	}

	log.Println("Connected to database")

	log.Println("Migrating database")

	if err := db.AutoMigrate(&Request{}, &SocketClient{}); err != nil {
		panic("failed to auto-migrate database")
	}

	log.Println("Migrated database")

	return db
}

func CountRequests() int64 {
	var count int64
	result := db.Model(&Request{}).Count(&count)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return count
}

func GetRequests() []Request {
	var items []Request
	result := db.Order("created_at DESC").Find(&items)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return items
}

func GetRequestsForEndpointID(endpointID string) []Request {
	var items []Request
	result := db.Where(&Request{EndpointID: endpointID}).Order("created_at DESC").Find(&items)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return items
}

func CreateRequest(request *Request) {
	result := db.Create(&request)
	if result.Error != nil {
		log.Println(result.Error)
	}
}

func DeleteOldRequests() {
	result := db.Where("created_at < ?", time.Now().Add(-1*4*time.Hour)).Delete(&Request{})
	if result.Error != nil {
		log.Println(result.Error)
	}
	log.Println("Deleted " + strconv.Itoa(int(result.RowsAffected)) + " old requests")
}

func CountSocketClients() int64 {
	var count int64
	result := db.Model(&SocketClient{}).Count(&count)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return count
}

func GetSocketClients() []SocketClient {
	var items []SocketClient
	result := db.Order("created_at DESC").Find(&items)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return items
}

func GetSocketClientsForEndpointID(endpointID string) []SocketClient {
	var items []SocketClient
	result := db.Where(&SocketClient{EndpointID: endpointID}).Order("created_at DESC").Find(&items)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return items
}

func CreateSocketClient(socketClient *SocketClient) {
	result := db.Create(&socketClient)
	if result.Error != nil {
		log.Println(result.Error)
	}
}

func DeleteSocketClientForUUID(UUID string) {
	result := db.Where("uuid = ?", UUID).Delete(&SocketClient{})
	if result.Error != nil {
		log.Println(result.Error)
	}
}

func DeleteOldSocketClients() {
	result := db.Where("created_at < ?", time.Now().Add(-1*4*time.Hour)).Delete(&SocketClient{})
	if result.Error != nil {
		log.Println(result.Error)
	}
	log.Println("Deleted " + strconv.Itoa(int(result.RowsAffected)) + " old socket clients")
}
