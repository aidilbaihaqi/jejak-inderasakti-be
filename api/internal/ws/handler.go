package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/auth"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/game"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

const closeUnauthorized = 4401

// Rooms resolves a live room; *game.Registry satisfies it.
type Rooms interface {
	Get(ctx context.Context, roomID string) (*game.Room, error)
}

// Handler serves /ws?token=<player token>, or /ws?token=<host jwt>&room=<room id> for the host.
type Handler struct {
	rooms    Rooms
	tokens   *auth.Tokens
	upgrader websocket.Upgrader
}

// NewHandler builds the handler. allowAnyOrigin disables the origin check entirely (development
// only). Otherwise the Origin header must be empty (non-browser clients) or match one of
// allowedOrigins — the same allow-list used for REST CORS — since the frontend and API can be on
// different (sub)domains and gorilla's default same-host check would otherwise reject that.
func NewHandler(rooms Rooms, tokens *auth.Tokens, allowAnyOrigin bool, allowedOrigins []string) *Handler {
	h := &Handler{rooms: rooms, tokens: tokens}
	h.upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}
	h.upgrader.CheckOrigin = func(r *http.Request) bool {
		if allowAnyOrigin {
			return true
		}
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		for _, allowed := range allowedOrigins {
			if strings.EqualFold(allowed, origin) {
				return true
			}
		}
		return false
	}
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, err := h.tokens.Parse(r.URL.Query().Get("token"))
	if err != nil {
		reject(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
		return
	}
	roomID := claims.RoomID
	if claims.Role == auth.RoleHost {
		roomID = r.URL.Query().Get("room")
	}
	room, err := h.rooms.Get(r.Context(), roomID)
	if err != nil {
		h.rejectRoomError(w, err)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Debug("ws upgrade failed", "err", err)
		return
	}
	client := newClient(conn, room, claims.Role)
	if err := attach(room, claims, client); err != nil {
		closeUnauthorizedConn(conn)
		return
	}
	go client.writePump()
	client.readPump(claims.Subject)
}

func attach(room *game.Room, claims *auth.Claims, client *Client) error {
	if claims.Role == auth.RoleHost {
		return room.AttachHost(claims.Subject, client)
	}
	return room.AttachPlayer(claims.Subject, client)
}

func (h *Handler) rejectRoomError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		reject(w, http.StatusNotFound, "ROOM_NOT_FOUND", "room not found or already ended")
		return
	}
	slog.Error("resolve room for ws", "err", err)
	reject(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

func closeUnauthorizedConn(conn *websocket.Conn) {
	msg := websocket.FormatCloseMessage(closeUnauthorized, "unauthorized")
	if err := conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(writeWait)); err != nil {
		slog.Debug("write close frame", "err", err)
	}
	if err := conn.Close(); err != nil {
		slog.Debug("close ws conn", "err", err)
	}
}

func reject(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message}); err != nil {
		slog.Debug("write ws rejection", "err", err)
	}
}
