package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/auth"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/game"
)

const (
	sendBuffer      = 32
	writeWait       = 10 * time.Second
	pingInterval    = 20 * time.Second
	pongWait        = 2*pingInterval + 5*time.Second
	maxMessageBytes = 4096
)

// Client is one WebSocket connection and implements game.Sink.
type Client struct {
	conn *websocket.Conn
	room *game.Room
	role auth.Role

	mu     sync.Mutex
	closed bool
	send   chan []byte
}

func newClient(conn *websocket.Conn, room *game.Room, role auth.Role) *Client {
	return &Client{conn: conn, room: room, role: role, send: make(chan []byte, sendBuffer)}
}

// Send queues a message without blocking; a client that cannot keep up is disconnected.
func (c *Client) Send(m game.Message) {
	data, err := json.Marshal(m)
	if err != nil {
		slog.Error("marshal ws message", "type", m.T, "err", err)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.send <- data:
	default:
		slog.Warn("ws client too slow, disconnecting", "type", m.T)
		c.closed = true
		close(c.send)
	}
}

// Close stops accepting messages; queued ones are still written before the socket closes.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	defer c.closeConn()
	for {
		select {
		case data, ok := <-c.send:
			if !ok {
				c.write(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if !c.write(websocket.TextMessage, data) {
				return
			}
		case <-ticker.C:
			if !c.write(websocket.PingMessage, nil) {
				return
			}
		}
	}
}

func (c *Client) write(messageType int, data []byte) bool {
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return false
	}
	return c.conn.WriteMessage(messageType, data) == nil
}

func (c *Client) readPump(clientID string) {
	defer c.room.Detach(c)
	defer c.Close()

	c.conn.SetReadLimit(maxMessageBytes)
	c.extendReadDeadline()
	c.conn.SetPongHandler(func(string) error { c.extendReadDeadline(); return nil })
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		c.extendReadDeadline()
		c.dispatch(clientID, data)
	}
}

func (c *Client) extendReadDeadline() {
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Debug("set read deadline", "err", err)
	}
}

func (c *Client) closeConn() {
	if err := c.conn.Close(); err != nil {
		slog.Debug("close ws conn", "err", err)
	}
}

type incoming struct {
	T string          `json:"t"`
	D json.RawMessage `json:"d"`
}

func (c *Client) dispatch(clientID string, data []byte) {
	var msg incoming
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Debug("ignoring malformed ws message", "client", clientID)
		return
	}
	switch {
	case msg.T == "ping":
		c.Send(game.Message{T: "pong"})
	case c.role == auth.RoleHost:
		c.dispatchHost(msg)
	}
}

func (c *Client) dispatchHost(msg incoming) {
	switch msg.T {
	case "host.start":
		c.room.StartSession()
	case "host.end":
		c.room.EndSession()
	case "host.kick":
		var d struct {
			PlayerID string `json:"player_id"`
		}
		if err := json.Unmarshal(msg.D, &d); err != nil || d.PlayerID == "" {
			return
		}
		c.room.Kick(d.PlayerID)
	}
}
