package main

import (
	"log/slog"
	"sync"

	"github.com/gofiber/contrib/v3/socketio"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

// socketRegistry tracks WS UUIDs subscribed to each endpoint, so the capture
// hot path can fan-out without a DB round-trip.
type socketRegistry struct {
	mu    sync.RWMutex
	byEnd map[string]map[string]struct{}
}

func newSocketRegistry() *socketRegistry {
	return &socketRegistry{byEnd: make(map[string]map[string]struct{})}
}

func (r *socketRegistry) add(endpointID, uuid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set, ok := r.byEnd[endpointID]
	if !ok {
		set = make(map[string]struct{})
		r.byEnd[endpointID] = set
	}
	set[uuid] = struct{}{}
}

func (r *socketRegistry) remove(uuid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for endpointID, set := range r.byEnd {
		if _, ok := set[uuid]; ok {
			delete(set, uuid)
			if len(set) == 0 {
				delete(r.byEnd, endpointID)
			}
			return
		}
	}
}

func (r *socketRegistry) uuidsFor(endpointID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	set := r.byEnd[endpointID]
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for u := range set {
		out = append(out, u)
	}
	return out
}

func (r *socketRegistry) count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, set := range r.byEnd {
		n += len(set)
	}
	return n
}

// registerWebSockets mounts the live-feed socket for an endpoint and keeps the
// registry in step with the connections that are actually open.
//
// socketio dispatches lifecycle events through package-level state, so the
// subscriptions below are process-wide rather than scoped to one application.
// That stays correct with more than one application in a process because a
// registry ignores a UUID it does not hold.
func registerWebSockets(application *fiber.App, registry *socketRegistry) {
	application.Use("/ws", func(c fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	application.Get("/ws/:endpoint", requireValidEndpoint, socketio.New(func(kws *socketio.Websocket) {
		endpointID := kws.Params("endpoint")
		kws.SetAttribute("endpointID", endpointID)
		registry.add(endpointID, kws.UUID)
		slog.Info("websocket connected", "endpoint_id", endpointID)
	}))

	socketio.On(socketio.EventDisconnect, func(ep *socketio.EventPayload) {
		registry.remove(ep.Kws.UUID)
	})
	socketio.On(socketio.EventClose, func(ep *socketio.EventPayload) {
		registry.remove(ep.Kws.UUID)
	})
}
