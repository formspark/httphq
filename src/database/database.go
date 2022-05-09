package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"time"
)

type Request struct {
	UUID       string    `json:"name" gorm:"primaryKey"`
	EndpointID string    `json:"endpointId"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"createdAt"`
	Headers    string    `json:"headers"`
}

type SocketClient struct {
	UUID       string    `json:"uuid" gorm:"primaryKey"`
	EndpointID string    `json:"endpointId"`
	CreatedAt  time.Time `json:"createdAt"`
}

func Connect() *gorm.DB {
	log.Println("Connecting to database...")

	db, err := gorm.Open(sqlite.Open("local.db"), &gorm.Config{})

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

func GetRequests(db *gorm.DB) []Request {
	var items []Request
	result := db.Order("created_at DESC").Find(&items)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return items
}

func GetRequestsForEndpointID(db *gorm.DB, endpointID string) []Request {
	var items []Request
	result := db.Where(&Request{EndpointID: endpointID}).Order("created_at DESC").Find(&items)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return items
}

func CreateRequest(db *gorm.DB, request *Request) {
	result := db.Create(&request)
	if result.Error != nil {
		log.Println(result.Error)
	}
}

func GetSocketClients(db *gorm.DB) []SocketClient {
	var items []SocketClient
	result := db.Order("created_at DESC").Find(&items)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return items
}

func GetSocketClientsForEndpointID(db *gorm.DB, endpointID string) []SocketClient {
	var items []SocketClient
	result := db.Where(&SocketClient{EndpointID: endpointID}).Order("created_at DESC").Find(&items)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return items
}

func CreateSocketClient(db *gorm.DB, socketClient *SocketClient) {
	result := db.Create(&socketClient)
	if result.Error != nil {
		log.Println(result.Error)
	}
}

func DeleteSocketClientForUUID(db *gorm.DB, UUID string) {
	result := db.Where("uuid = ?", UUID).Delete(&SocketClient{})
	if result.Error != nil {
		log.Println(result.Error)
	}
}
