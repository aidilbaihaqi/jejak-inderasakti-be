package http

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/game"
	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

const (
	maxNicknameRunes = 20
	minAvatar        = 1
	maxAvatar        = 12
)

var (
	pinPattern   = regexp.MustCompile(`^[0-9]{6}$`)
	validJenjang = map[string]bool{"SD": true, "SMP": true, "SMA": true}
	validLang    = map[string]bool{"id": true, "en": true}

	// dummyHash keeps login timing similar whether or not the email exists.
	dummyHash = sync.OnceValue(func() []byte {
		hash, err := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)
		if err != nil {
			return nil
		}
		return hash
	})
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	host, err := a.Store.HostByEmail(r.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil && !isNotFound(err) {
		writeInternal(w, "load host", err)
		return
	}
	if !passwordMatches(host.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
		return
	}
	token, err := a.Tokens.IssueHost(host.ID)
	if err != nil {
		writeInternal(w, "issue host token", err)
		return
	}
	if err := a.Store.TouchHostLogin(r.Context(), host.ID); err != nil {
		writeInternal(w, "record host login", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func passwordMatches(hash, password string) bool {
	known := hash != ""
	if !known {
		hash = string(dummyHash())
	}
	matches := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	return known && matches
}

type createRoomRequest struct {
	Jenjang          string `json:"jenjang"`
	ShortSession     bool   `json:"short_session"`
	AccuracyMode     bool   `json:"accuracy_mode"`
	ConsentConfirmed bool   `json:"consent_confirmed"`
}

func (a *API) createRoom(w http.ResponseWriter, r *http.Request) {
	var req createRoomRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validJenjang[req.Jenjang] {
		writeInvalid(w, "jenjang must be SD, SMP or SMA")
		return
	}
	if !req.ConsentConfirmed {
		writeInvalid(w, "consent_confirmed must be true")
		return
	}
	pool, err := a.Store.QuestionPool(r.Context())
	if err != nil {
		writeInternal(w, "load question pool", err)
		return
	}
	ids, err := game.SelectQuestions(newRNG(), req.Jenjang, req.ShortSession, pool)
	if err != nil {
		writeInternal(w, "select questions", err)
		return
	}
	room, err := a.Store.CreateRoom(r.Context(), store.NewRoom{
		HostID: hostIDFrom(r.Context()), Jenjang: req.Jenjang,
		ShortSession: req.ShortSession, AccuracyMode: req.AccuracyMode, QuestionIDs: ids,
	})
	if errors.Is(err, store.ErrMaxRooms) {
		writeError(w, http.StatusConflict, "MAX_ROOMS", "5 rooms are already active")
		return
	}
	if err != nil {
		writeInternal(w, "create room", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"id": room.ID, "pin": room.PIN, "qr_url": fmt.Sprintf("%s/join?pin=%s", a.PublicBaseURL, room.PIN),
	})
}

func (a *API) getRoom(w http.ResponseWriter, r *http.Request) {
	room, ok := a.activeRoom(w, r)
	if !ok {
		return
	}
	count, err := a.Store.PlayerCount(r.Context(), room.ID)
	if err != nil {
		writeInternal(w, "count players", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": room.Status, "slots_left": max(0, store.MaxPlayersInRoom-count), "jenjang": room.Jenjang,
	})
}

type joinRequest struct {
	Nickname string `json:"nickname"`
	SchoolID *int   `json:"school_id"`
	Jenjang  string `json:"jenjang"`
	Avatar   int    `json:"avatar"`
	Lang     string `json:"lang"`
}

func (a *API) joinRoom(w http.ResponseWriter, r *http.Request) {
	if a.JoinLimiter != nil && !a.JoinLimiter.Allow(r.Context(), "join:"+clientIP(r, a.TrustProxy)) {
		writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many join attempts, try again in a minute")
		return
	}
	var req joinRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Nickname = strings.TrimSpace(req.Nickname)
	if msg := validateJoin(req); msg != "" {
		writeInvalid(w, msg)
		return
	}
	room, ok := a.activeRoom(w, r)
	if !ok {
		return
	}
	player, err := a.Store.JoinRoom(r.Context(), store.NewPlayer{
		RoomID: room.ID, Nickname: req.Nickname, SchoolID: req.SchoolID,
		Jenjang: req.Jenjang, Avatar: req.Avatar, Lang: req.Lang,
	})
	if !a.handleJoinError(w, err) {
		return
	}
	if err := a.Rooms.AddPlayer(r.Context(), room.ID, player); err != nil {
		writeError(w, http.StatusNotFound, "ROOM_NOT_FOUND", "room not found or already ended")
		return
	}
	token, err := a.Tokens.IssuePlayer(room.ID, player.ID)
	if err != nil {
		writeInternal(w, "issue player token", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"player_token": token})
}

// handleJoinError writes the error response and reports whether the join succeeded.
func (a *API) handleJoinError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, store.ErrRoomFull):
		writeError(w, http.StatusConflict, "ROOM_FULL", "room already has 15 players")
	case errors.Is(err, store.ErrNicknameTaken):
		writeError(w, http.StatusConflict, "NICKNAME_TAKEN", "nickname is already used in this room")
	case errors.Is(err, store.ErrInvalidSchool):
		writeInvalid(w, "unknown school_id")
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "ROOM_NOT_FOUND", "room not found or already ended")
	default:
		writeInternal(w, "join room", err)
	}
	return false
}

func (a *API) searchSchools(w http.ResponseWriter, r *http.Request) {
	schools, err := a.Store.SearchSchools(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeInternal(w, "search schools", err)
		return
	}
	body := make([]map[string]any, 0, len(schools))
	for _, s := range schools {
		body = append(body, map[string]any{"id": s.ID, "name": s.Name, "jenjang": s.Jenjang})
	}
	writeJSON(w, http.StatusOK, body)
}

func (a *API) activeRoom(w http.ResponseWriter, r *http.Request) (store.Room, bool) {
	pin := r.PathValue("pin")
	if !pinPattern.MatchString(pin) {
		writeError(w, http.StatusNotFound, "ROOM_NOT_FOUND", "room not found or already ended")
		return store.Room{}, false
	}
	room, err := a.Store.RoomByPIN(r.Context(), pin)
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "ROOM_NOT_FOUND", "room not found or already ended")
		return store.Room{}, false
	case err != nil:
		writeInternal(w, "load room", err)
		return store.Room{}, false
	}
	return room, true
}

func validateJoin(req joinRequest) string {
	switch {
	case req.Nickname == "" || utf8.RuneCountInString(req.Nickname) > maxNicknameRunes:
		return "nickname must be 1-20 characters"
	case strings.ContainsFunc(req.Nickname, unicode.IsControl):
		return "nickname contains invalid characters"
	case !validJenjang[req.Jenjang]:
		return "jenjang must be SD, SMP or SMA"
	case req.Avatar < minAvatar || req.Avatar > maxAvatar:
		return "avatar must be between 1 and 12"
	case !validLang[req.Lang]:
		return "lang must be id or en"
	}
	return ""
}

// newRNG returns a fresh generator per room so simultaneous rooms get different question sets.
func newRNG() *rand.Rand {
	return rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), rand.Uint64())) //nolint:gosec // game shuffling, not security
}
