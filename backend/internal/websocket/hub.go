package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Message types
const (
	MessageTypeChat      = "chat"
	MessageTypeAck       = "ack"
	MessageTypeSync      = "sync"
	MessageTypePresence  = "presence"
	MessageTypeTyping    = "typing"
	MessageTypeReaction  = "reaction"
	MessageTypeError     = "error"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// ChatMessage represents a chat message payload
type ChatMessage struct {
	ChatID      string `json:"chat_id"`
	Content     string `json:"content,omitempty"`
	MessageType string `json:"message_type"`
	MediaURL    string `json:"media_url,omitempty"`
	ReplyTo     string `json:"reply_to,omitempty"`
	Version     int64  `json:"version"`
}

// AckMessage represents an acknowledgment
type AckMessage struct {
	RequestID string `json:"request_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

// SyncRequest represents a sync request
type SyncRequest struct {
	LastSyncTime time.Time   `json:"last_sync_time"`
	ChatIDs      []string    `json:"chat_ids,omitempty"`
	MaxMessages  int         `json:"max_messages"`
}

// SyncResponse represents a sync response
type SyncResponse struct {
	Messages   []ChatMessage `json:"messages"`
	LastSync   time.Time     `json:"last_sync"`
	HasMore    bool          `json:"has_more"`
}

// PresenceUpdate represents presence status change
type PresenceUpdate struct {
	UserID    string    `json:"user_id"`
	IsOnline  bool      `json:"is_online"`
	LastSeen  time.Time `json:"last_seen,omitempty"`
}

// TypingIndicator represents typing status
type TypingIndicator struct {
	ChatID  string `json:"chat_id"`
	UserID  string `json:"user_id"`
	IsTyping bool   `json:"is_typing"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID           string
	UserID       string
	DeviceID     string
	Conn         *websocket.Conn
	Send         chan []byte
	mu           sync.RWMutex
	lastActive   time.Time
	subscribedChats map[string]bool
}

// Hub manages WebSocket connections
type Hub struct {
	clients    map[string]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewHub creates a new WebSocket hub
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Shutting down WebSocket hub")
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			h.logger.Debug("Client registered", zap.String("client_id", client.ID))
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()
			h.logger.Debug("Client unregistered", zap.String("client_id", client.ID))
		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// Client buffer full, disconnect
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SubscribeToChat subscribes a client to a chat
func (c *Client) SubscribeToChat(chatID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.subscribedChats == nil {
		c.subscribedChats = make(map[string]bool)
	}
	c.subscribedChats[chatID] = true
}

// UnsubscribeFromChat unsubscribes a client from a chat
func (c *Client) UnsubscribeFromChat(chatID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscribedChats, chatID)
}

// IsSubscribedToChat checks if client is subscribed to a chat
func (c *Client) IsSubscribedToChat(chatID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.subscribedChats[chatID]
}

// SendJSON sends a JSON message to the client
func (c *Client) SendJSON(msg WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case c.Send <- data:
		c.mu.Lock()
		c.lastActive = time.Now()
		c.mu.Unlock()
		return nil
	default:
		return ErrBufferFull
	}
}

// ErrBufferFull indicates the send buffer is full
var ErrBufferFull = errors.New("send buffer full")

// WritePump handles writing messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			if _, err := w.Write(message); err != nil {
				w.Close()
				return
			}

			// Queue remaining messages
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				return
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump handles reading messages from the WebSocket connection
func (c *Client) ReadPump(hub *Hub, handler func(*Client, WSMessage) error) {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(1024 * 1024) // 1MB max message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log unexpected close
			}
			break
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			// Send error response
			errorMsg := WSMessage{
				Type:      MessageTypeError,
				Payload:   json.RawMessage(`{"error": "invalid message format"}`),
				Timestamp: time.Now().UnixMilli(),
			}
			c.SendJSON(errorMsg)
			continue
		}

		wsMsg.Timestamp = time.Now().UnixMilli()

		if err := handler(c, wsMsg); err != nil {
			errorMsg := WSMessage{
				Type:      MessageTypeError,
				Payload:   json.RawMessage(`{"error": "` + err.Error() + `"}`),
				RequestID: wsMsg.RequestID,
				Timestamp: time.Now().UnixMilli(),
			}
			c.SendJSON(errorMsg)
		}
	}
}
