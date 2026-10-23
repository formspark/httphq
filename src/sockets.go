package main

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	// outboxSize is how many captures a socket may have waiting to be written.
	// A client that stops reading fills it and is disconnected, so one socket
	// never holds more than this many payloads in memory.
	outboxSize = 16

	// socketReadLimit caps a message a client sends. The feed reads nothing
	// from the page, so anything beyond a control-sized frame is refused and
	// the socket closed.
	socketReadLimit = 4 << 10

	// writeWait bounds one write to the client, so a socket whose peer stopped
	// reading is cut rather than holding its writer forever.
	writeWait = 10 * time.Second

	// pongWait is how long a socket lives without hearing from its peer, and
	// pingInterval is how often the server asks. A browser answers pings on its
	// own, so a dead connection is reaped while a quiet page stays open.
	pongWait     = 60 * time.Second
	pingInterval = pongWait / 2
)

// subscriber is one open socket on an endpoint's feed. Captures are offered to
// its outbox without waiting, so fan-out never blocks on a slow client.
type subscriber struct {
	id     string
	outbox chan []byte
	closed chan struct{}
	once   sync.Once
}

func newSubscriber() *subscriber {
	return &subscriber{
		id:     uuid.NewString(),
		outbox: make(chan []byte, outboxSize),
		closed: make(chan struct{}),
	}
}

// offer queues a payload for the socket. A socket whose outbox is already full
// has stopped keeping up, so it is closed rather than queued for.
func (s *subscriber) offer(payload []byte) {
	select {
	case s.outbox <- payload:
	default:
		s.close()
	}
}

// close tells the socket's writer to stop. Safe to call more than once.
func (s *subscriber) close() {
	s.once.Do(func() { close(s.closed) })
}

// socketRegistry tracks the sockets subscribed to each endpoint, so the capture
// hot path can fan out without a DB round-trip.
type socketRegistry struct {
	mu    sync.RWMutex
	byEnd map[string]map[string]*subscriber
}

func newSocketRegistry() *socketRegistry {
	return &socketRegistry{byEnd: make(map[string]map[string]*subscriber)}
}

func (r *socketRegistry) add(endpointID string, feed *subscriber) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set, ok := r.byEnd[endpointID]
	if !ok {
		set = make(map[string]*subscriber)
		r.byEnd[endpointID] = set
	}
	set[feed.id] = feed
}

// remove drops a socket from the endpoint it watched. Connections are looked up
// by endpoint, so its own endpoint is named rather than searched for.
func (r *socketRegistry) remove(endpointID, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set := r.byEnd[endpointID]
	delete(set, id)
	if len(set) == 0 {
		delete(r.byEnd, endpointID)
	}
}

func (r *socketRegistry) subscribersFor(endpointID string) []*subscriber {
	r.mu.RLock()
	defer r.mu.RUnlock()
	set := r.byEnd[endpointID]
	if len(set) == 0 {
		return nil
	}
	out := make([]*subscriber, 0, len(set))
	for _, feed := range set {
		out = append(out, feed)
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
func registerWebSockets(application *fiber.App, registry *socketRegistry) {
	application.Use("/ws", func(c fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	application.Get("/ws/:endpoint", requireValidEndpoint, websocket.New(func(conn *websocket.Conn) {
		serveFeed(conn, registry, conn.Params("endpoint"))
	}))
}

// serveFeed runs one socket until its peer goes away or stops keeping up. The
// reader only watches for the peer leaving; everything sent goes through the
// writer, which is the one goroutine allowed to write to the connection.
//
// The reader has to be finished before this returns: the connection is
// released for reuse as soon as the handler does, and a reader still blocked
// on it would race whoever gets it next. Closing it is what unblocks one.
func serveFeed(conn *websocket.Conn, registry *socketRegistry, endpointID string) {
	feed := newSubscriber()
	registry.add(endpointID, feed)
	defer registry.remove(endpointID, feed.id)
	slog.Info("websocket connected")

	conn.SetReadLimit(socketReadLimit)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		readUntilGone(conn, feed)
	}()
	writeFeed(conn, feed)
	_ = conn.Close()
	<-readerDone
}

// readUntilGone discards whatever the client sends and closes the feed once
// the connection fails, the peer goes quiet past pongWait, or a message
// exceeds the read limit.
func readUntilGone(conn *websocket.Conn, feed *subscriber) {
	defer feed.close()
	extend := func(string) error { return conn.SetReadDeadline(time.Now().Add(pongWait)) }
	if extend("") != nil {
		return
	}
	conn.SetPongHandler(extend)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

// writeFeed writes queued captures and periodic pings until the feed closes or
// a write fails.
func writeFeed(conn *websocket.Conn, feed *subscriber) {
	ping := time.NewTicker(pingInterval)
	defer ping.Stop()
	for {
		if !writeNext(conn, feed, ping.C) {
			return
		}
	}
}

// writeNext performs one write and reports whether the feed should go on.
func writeNext(conn *websocket.Conn, feed *subscriber, ping <-chan time.Time) bool {
	select {
	case payload := <-feed.outbox:
		return writeFrame(conn, websocket.TextMessage, payload) == nil
	case <-ping:
		return writeFrame(conn, websocket.PingMessage, nil) == nil
	case <-feed.closed:
		return false
	}
}

func writeFrame(conn *websocket.Conn, messageType int, payload []byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return conn.WriteMessage(messageType, payload)
}
