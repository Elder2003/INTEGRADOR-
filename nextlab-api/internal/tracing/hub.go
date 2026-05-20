package tracing

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

// Hub maintains WebSocket connections grouped by session ID.
// Multiple browser tabs open for the same session each get their own entry.
type Hub struct {
	mu    sync.Mutex
	conns map[string][]*websocket.Conn
}

func NewHub() *Hub {
	return &Hub{conns: make(map[string][]*websocket.Conn)}
}

// Add registers a connection for the given session.
func (h *Hub) Add(sessionID string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[sessionID] = append(h.conns[sessionID], c)
}

// Remove unregisters a specific connection from the session.
func (h *Hub) Remove(sessionID string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	existing := h.conns[sessionID]
	filtered := existing[:0]
	for _, conn := range existing {
		if conn != c {
			filtered = append(filtered, conn)
		}
	}
	if len(filtered) == 0 {
		delete(h.conns, sessionID)
	} else {
		h.conns[sessionID] = filtered
	}
}

// GetConns returns all active connections for a session (safe copy).
func (h *Hub) GetConns(sessionID string) []*websocket.Conn {
	h.mu.Lock()
	defer h.mu.Unlock()
	src := h.conns[sessionID]
	if len(src) == 0 {
		return nil
	}
	out := make([]*websocket.Conn, len(src))
	copy(out, src)
	return out
}

// BroadcastClose sends a close message to all connections for a session and removes them.
func (h *Hub) BroadcastClose(sessionID string, msg []byte) {
	h.mu.Lock()
	conns := h.conns[sessionID]
	delete(h.conns, sessionID)
	h.mu.Unlock()

	for _, c := range conns {
		_ = c.WriteMessage(websocket.TextMessage, msg)
		c.Close()
	}
}
