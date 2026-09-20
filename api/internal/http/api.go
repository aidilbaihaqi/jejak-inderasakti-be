package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/auth"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

const maxBodyBytes = 1 << 16

// Store is the persistence the REST handlers use; *store.Store satisfies it.
type Store interface {
	HostByEmail(ctx context.Context, email string) (store.Host, error)
	TouchHostLogin(ctx context.Context, hostID string) error
	CreateRoom(ctx context.Context, in store.NewRoom) (store.Room, error)
	RoomByPIN(ctx context.Context, pin string) (store.Room, error)
	PlayerCount(ctx context.Context, roomID string) (int, error)
	JoinRoom(ctx context.Context, in store.NewPlayer) (store.Player, error)
	SearchSchools(ctx context.Context, query string) ([]store.School, error)
	QuestionPool(ctx context.Context) ([]store.QuestionRef, error)
}

// RoomRegistry connects REST actions to the live room goroutines; *game.Registry satisfies it.
type RoomRegistry interface {
	AddPlayer(ctx context.Context, roomID string, p store.Player) error
}

type Deps struct {
	Store         Store
	Rooms         RoomRegistry
	Tokens        *auth.Tokens
	WebSocket     http.Handler
	PublicBaseURL string
}

type API struct {
	Deps
	clock func() time.Time
}

func NewRouter(d Deps) http.Handler {
	api := &API{Deps: d, clock: time.Now}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", api.healthz)
	mux.HandleFunc("POST /api/auth/login", api.login)
	mux.HandleFunc("POST /api/rooms", api.requireHost(api.createRoom))
	mux.HandleFunc("GET /api/rooms/{pin}", api.getRoom)
	mux.HandleFunc("POST /api/rooms/{pin}/join", api.joinRoom)
	mux.HandleFunc("GET /api/schools", api.searchSchools)
	if d.WebSocket != nil {
		mux.Handle("GET /ws", d.WebSocket)
	}
	return recoverPanics(logRequests(mux))
}

func (a *API) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Warn("write json response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message})
}

func writeInvalid(w http.ResponseWriter, message string) {
	writeError(w, http.StatusBadRequest, "INVALID_REQUEST", message)
}

func writeInternal(w http.ResponseWriter, op string, err error) {
	slog.Error(op, "err", err)
	writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeInvalid(w, "request body must be valid JSON")
		return false
	}
	return true
}

func isNotFound(err error) bool { return errors.Is(err, store.ErrNotFound) }
