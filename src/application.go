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
	"gorm.io/datatypes"
	"httphq/src/database"
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
	"Trace",
	"Traceparent",
	"Tracestate",
	"Via",
}

func main() {
	/* Database */

	database.Connect("local.db")

	/* Haiku maker */

	haikuMaker := haikunator.New()

	/* Cron */

	c := cron.New()

	everyFiveMinutes := "*/5 * * * *"
	if _, err := c.AddFunc(everyFiveMinutes, func() {
		threshold := time.Now().Add(-1 * 4 * time.Hour)
		database.DeleteOldRequests(threshold)
		database.DeleteOldSocketClients(threshold)
	}); err != nil {
		log.Fatalln(err)
	}

	c.Start()

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

	maxRequestsPerMinute := 9999
	if isProduction {
		maxRequestsPerMinute = 125
	}
	application.Use(limiter.New(
		limiter.Config{
			Max:        maxRequestsPerMinute,
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

	application.Get("/api/health", func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	application.Get("/api/debug", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"host":         string(c.Request().Host()),
			"isProduction": isProduction,
			"requests":     database.CountRequests(),
			"sockets":      database.CountSocketClients(),
		})
	})

	application.Get("/api/endpoints/:endpoint/requests", func(c *fiber.Ctx) error {
		endpointID := c.Params("endpoint")
		search := c.Query("search")
		requests := database.GetRequestsForEndpointID(endpointID, search, 128)
		return c.JSON(fiber.Map{
			"requests": requests,
		})
	})

	application.Delete("/api/endpoints/:endpoint/requests", func(c *fiber.Ctx) error {
		endpointID := c.Params("endpoint")
		database.DeleteRequestsForEndpointID(endpointID)
		return c.SendStatus(http.StatusOK)
	})

	application.Delete("/api/endpoints/:endpoint/requests/:request", func(c *fiber.Ctx) error {
		requestUUID := c.Params("request")
		database.DeleteRequestForUUID(requestUUID)
		return c.SendStatus(http.StatusOK)
	})

	application.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "httphq",
		})
	})

	application.Get("/contact", func(c *fiber.Ctx) error {
		return c.Render("contact", fiber.Map{
			"Title": "Contact | httphq",
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
			"Title":                endpointID + " | httphq",
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

		IP := c.IP()
		forwardedIPs := c.IPs()
		if len(forwardedIPs) > 0 {
			IP = forwardedIPs[0]
		}

		method := c.Method()

		path := c.Path()

		body := c.Body()

		headers := c.GetReqHeaders()

		if spoofCurl, ok := headers["Httphq-Spoof-Curl"]; ok && spoofCurl == "true" {
			delete(headers, "Accept-Encoding")
			delete(headers, "Accept-Language")
			delete(headers, "Connection")
			delete(headers, "Httphq-Spoof-Curl")
			delete(headers, "Origin")
			delete(headers, "Referer")
			delete(headers, "Sec-Fetch-Dest")
			delete(headers, "Sec-Fetch-Mode")
			delete(headers, "Sec-Ch-Ua")
			delete(headers, "Sec-Ch-Ua-Mobile")
			delete(headers, "Sec-Ch-Ua-Platform")
			delete(headers, "Sec-Fetch-Mode")
			delete(headers, "Sec-Fetch-Site")

			headers["Content-Type"] = "application/x-www-form-urlencoded"
			headers["User-Agent"] = "curl/7.79.1"
		}
		for _, omittedHeader := range omittedHeaders {
			delete(headers, omittedHeader)
		}
		jsonHeaders, err := json.Marshal(headers)
		if err != nil {
			log.Println(err)
		}

		request := database.Request{
			UUID:       UUID,
			EndpointID: endpointID,
			IP:         IP,
			Method:     method,
			Path:       path,
			Body:       string(body),
			Headers:    datatypes.JSON(jsonHeaders),
		}
		database.CreateRequest(&request)

		socketClients := database.GetSocketClientsForEndpointID(endpointID, 32)
		for _, socketClient := range socketClients {
			marshalled, marshalErr := json.Marshal(request)
			if marshalErr != nil {
				log.Println(marshalErr)
			} else {
				emitErr := ikisocket.EmitTo(socketClient.UUID, marshalled)
				if emitErr != nil {
					log.Println(emitErr)
					database.DeleteSocketClientForUUID(socketClient.UUID)
				}
			}
		}

		c.Set("Httphq-Request-Uuid", UUID)
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
