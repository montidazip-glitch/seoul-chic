package handlers

import (
	"log"
	"net/http"
	"sync"
)

// SSEHub manages per-order SSE client channels and broadcasts status events.
type SSEHub struct {
	mu         sync.RWMutex
	clients    map[int]map[chan string]struct{} // orderID -> set of client channels
	register   chan subscription
	unregister chan subscription
	broadcast  chan sseEvent
}

type subscription struct {
	orderID int
	ch      chan string
}

type sseEvent struct {
	orderID int
	payload string // pre-formatted SSE data line
}

// NewSSEHub creates and returns an initialised SSEHub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients:    make(map[int]map[chan string]struct{}),
		register:   make(chan subscription, 16),
		unregister: make(chan subscription, 16),
		broadcast:  make(chan sseEvent, 64),
	}
}

// Run starts the hub's event loop. Call it in a goroutine.
func (h *SSEHub) Run() {
	for {
		select {
		case sub := <-h.register:
			h.mu.Lock()
			if h.clients[sub.orderID] == nil {
				h.clients[sub.orderID] = make(map[chan string]struct{})
			}
			h.clients[sub.orderID][sub.ch] = struct{}{}
			h.mu.Unlock()

		case sub := <-h.unregister:
			h.mu.Lock()
			if chans, ok := h.clients[sub.orderID]; ok {
				delete(chans, sub.ch)
				if len(chans) == 0 {
					delete(h.clients, sub.orderID)
				}
			}
			h.mu.Unlock()

		case evt := <-h.broadcast:
			h.mu.RLock()
			chans := h.clients[evt.orderID]
			h.mu.RUnlock()
			for ch := range chans {
				select {
				case ch <- evt.payload:
				default:
					log.Printf("[sse] Dropping event for order %d — client channel full", evt.orderID)
				}
			}
		}
	}
}

// BroadcastStatus sends an order status update to all connected SSE clients for that order.
func (h *SSEHub) BroadcastStatus(orderID int, status string) {
	payload := "data: {\"status\":\"" + status + "\"}\n\n"
	h.broadcast <- sseEvent{orderID: orderID, payload: payload}
}

// SSEHandler handles GET /events/{id} — streams order status via SSE.
func (h *SSEHub) SSEHandler(w http.ResponseWriter, r *http.Request) {
	orderID, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable Nginx buffering if proxied

	ch := make(chan string, 8)
	h.register <- subscription{orderID: orderID, ch: ch}
	defer func() { h.unregister <- subscription{orderID: orderID, ch: ch} }()

	// Send a heartbeat comment immediately so the browser confirms the connection
	w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, open := <-ch:
			if !open {
				return
			}
			w.Write([]byte(msg))
			flusher.Flush()
		}
	}
}
