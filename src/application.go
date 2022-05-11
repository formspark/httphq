package main

import (
	"encoding/json"
	"github.com/antoniodipinto/ikisocket"
	"github.com/atrox/haikunatorgo/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/template/html"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go-project/src/database"
	"gorm.io/datatypes"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const port = 8080

var isProduction = os.Getenv("APPLICATION_ENV") == "production"

var omittedHeaders = [...]string{
	"Fly-Client-Ip",
	"Fly-Dispatch-Start",
	"Fly-Forwarded-Port",
	"Fly-Forwarded-Proto",
	"Fly-Forwarded-Ssl",
	"Fly-Region",
	"Fly-Request-Id",
	"X-Forwarded-For",
	"X-Forwarded-Port",
	"X-Forwarded-Proto",
	"X-Forwarded-Ssl",
	"X-Request-Start",
}

func main() {
	/* Database */

	database.Connect()

	/* Haiku maker */

	haikuMaker := haikunator.New()

	/* Cron */

	cron := cron.New()

	cron.AddFunc("*/5 * * * *", func() {
		database.DeleteOldRequests()
		database.DeleteOldSocketClients()
	}) // TODO: handle error

	cron.Start()

	/* Server */

	engine := html.New("./src/views", ".html")

	if !isProduction {
		engine.Reload(true)
		engine.Debug(true)
	}

	application := fiber.New(fiber.Config{
		Views:       engine,
		ViewsLayout: "layouts/main",
	})

	application.Use(limiter.New(
		limiter.Config{
			Max:        125,
			Expiration: 1 * time.Minute,
		}))

	application.Use(compress.New())

	// Static handling

	application.Static("/", "./public")

	// WS handling

	application.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	application.Get("/ws/:endpoint", ikisocket.New(func(kws *ikisocket.Websocket) {
		endpointID := kws.Params("endpoint")
		// TODO: create or update
		database.CreateSocketClient(&database.SocketClient{
			UUID:       kws.UUID,
			EndpointID: endpointID,
		})
		log.Printf("%s connected to WS\n", endpointID)
	}))

	ikisocket.On(ikisocket.EventDisconnect, func(ep *ikisocket.EventPayload) {
		database.DeleteSocketClientForUUID(ep.Kws.UUID)
	})

	ikisocket.On(ikisocket.EventClose, func(ep *ikisocket.EventPayload) {
		database.DeleteSocketClientForUUID(ep.Kws.UUID)
	})

	// HTTP handling

	application.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "Home",
		})
	})

	application.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	application.Get("/favicon", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusNotFound)
	})

	application.Get("/robots", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusNotFound)
	})

	// TODO: hide sensitive data in production
	application.Get("/api/debug", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"host":         string(c.Request().Host()),
			"isProduction": isProduction,
			"requests":     len(database.GetRequests()),
			"sockets":      len(database.GetSocketClients()),
		})
	})

	application.Get("/api/endpoints/:endpoint/requests", func(c *fiber.Ctx) error {
		endpointID := c.Params("endpoint")
		return c.JSON(fiber.Map{
			"requests": database.GetRequestsForEndpointID(endpointID),
		})
	})

	application.Get("/:endpoint", func(c *fiber.Ctx) error {
		endpointID := c.Params("endpoint")
		host := string(c.Request().Host())
		protocol := c.Protocol()
		websocketProtocol := "ws"
		if protocol == "https" {
			websocketProtocol = "wss"
		}
		return c.Render("endpoint", fiber.Map{
			"Title":                "Endpoint",
			"EndpointID":           endpointID,
			"EndpointURL":          protocol + "://" + host + "/to/" + endpointID,
			"EndpointWebSocketURL": websocketProtocol + "://" + host + "/ws/" + endpointID,
		})
	})

	application.Post("/endpoint", func(c *fiber.Ctx) error {
		endpointID := haikuMaker.Haikunate()
		log.Printf("Created endpoint %s\n", endpointID)
		return c.Redirect("/" + endpointID)
	})

	application.Use("/to/:endpoint", func(c *fiber.Ctx) error {
		UUID := uuid.NewString()
		endpointID := c.Params("endpoint")
		log.Println("IP:")
		log.Println(c.IP()) // TODO
		log.Println("IPs:")
		log.Println(c.IPs()) // TODO
		log.Println("X-Forwarded-For")
		log.Println(c.Get("X-Forwarded-For")) // TODO
		log.Println("X-Forwarded-For with default")
		log.Println(c.Get("X-Forwarded-For"), c.IP()) // TODO
		method := c.Method()
		path := c.Path()
		body := c.Body()
		headers := c.GetReqHeaders()
		for _, omittedHeader := range omittedHeaders {
			delete(headers, omittedHeader)
		}
		jsonHeaders, _ := json.Marshal(headers)
		request := database.Request{
			UUID:       UUID,
			EndpointID: endpointID,
			Method:     method,
			Path:       path,
			Body:       string(body),
			Headers:    datatypes.JSON(jsonHeaders),
		}
		database.CreateRequest(&request)
		socketClients := database.GetSocketClientsForEndpointID(endpointID)
		for _, socketClient := range socketClients {
			// TODO: error handling
			marshalled, _ := json.Marshal(request)
			ikisocket.EmitTo(socketClient.UUID, marshalled)
		}
		return c.SendStatus(http.StatusOK)
	})

	application.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusNotFound)
	})

	host := "localhost:"
	if isProduction {
		host = ":"
	}
	log.Fatalln(application.Listen(host + strconv.Itoa(port)))
}
