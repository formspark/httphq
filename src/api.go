package main

import (
	"net/http"

	"github.com/gofiber/fiber/v3"

	"httphq/src/database"
)

// requestPageSize bounds one listing response. The page renders a window of the
// stream rather than its history, and an endpoint under load would otherwise
// return a payload no reader can use.
const requestPageSize = 128

func handleHealth(c fiber.Ctx) error {
	return c.SendStatus(http.StatusOK)
}

// handleDebug reports coarse process state. It carries no captured data: the
// counts say how much traffic the process is holding, never what was in it.
func handleDebug(registry *socketRegistry) fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"host":         string(c.Request().Host()),
			"isProduction": isProduction,
			"requests":     database.CountRequests(c.Context()),
			"sockets":      registry.count(),
		})
	}
}

// handleListRequests returns a window of an endpoint's captures, narrowed by
// the search. `total` is what the endpoint holds regardless of search or
// window, so the page can say what a control acting on the whole endpoint will
// affect rather than reporting the size of the current view.
func handleListRequests(c fiber.Ctx) error {
	endpointID := c.Params("endpoint")
	return c.JSON(fiber.Map{
		"requests": database.GetRequestsForEndpointID(
			c.Context(), endpointID, c.Query("search"), requestPageSize),
		"total": database.CountRequestsForEndpointID(c.Context(), endpointID),
	})
}

func handleDeleteRequests(c fiber.Ctx) error {
	database.DeleteRequestsForEndpointID(c.Context(), c.Params("endpoint"))
	return c.SendStatus(http.StatusOK)
}

// handleDeleteRequest deletes one capture by UUID. The endpoint ID is validated
// upstream but the UUID is not scoped to it, so this is a delete-by-key on a
// value the caller already had to read from that endpoint's stream.
func handleDeleteRequest(c fiber.Ctx) error {
	database.DeleteRequestForUUID(c.Context(), c.Params("request"))
	return c.SendStatus(http.StatusOK)
}
