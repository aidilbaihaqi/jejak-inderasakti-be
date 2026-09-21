package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store/queries"
)

const (
	MaxActiveRooms   = 5
	MaxPlayersInRoom = 15

	pinSpace        = 1_000_000
	pinAttempts     = 8
	pgUniqueViolate = "23505"
	pgForeignKey    = "23503"
)

var specialSchools = []string{"Sekolah lain", "Umum / General visitor"}

var (
	ErrNotFound      = errors.New("not found")
	ErrMaxRooms      = errors.New("maximum active rooms reached")
	ErrRoomFull      = errors.New("room is full")
	ErrNicknameTaken = errors.New("nickname already taken in this room")
	ErrInvalidSchool = errors.New("unknown school")
)

type Store struct {
	pool *pgxpool.Pool
	q    *queries.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: queries.New(pool)}
}

type Host struct {
	ID, Email, PasswordHash, Name string
}

type Room struct {
	ID, PIN, HostID, Jenjang string
	ShortSession             bool
	AccuracyMode             bool
	Status                   string
	QuestionIDs              []string
	StartedAt                time.Time // zero until the room starts
}

type NewRoom struct {
	HostID, Jenjang string
	ShortSession    bool
	AccuracyMode    bool
	QuestionIDs     []string
}

type Player struct {
	ID, Nickname, School, Lang string
	Avatar                     int

	// Progress, restored when a room is reloaded from the database.
	Score, CorrectCount, TotalMs, CurrentIndex, Streak int
	Finished                                           bool
}

type NewPlayer struct {
	RoomID, Nickname, Jenjang, Lang string
	SchoolID                        *int
	Avatar                          int
}

type School struct {
	ID      int
	Name    string
	Jenjang *string
}

type QuestionRef struct {
	ID          string
	Site, Level int
}

func (s *Store) HostByEmail(ctx context.Context, email string) (Host, error) {
	row, err := s.q.GetHostByEmail(ctx, email)
	if err != nil {
		return Host{}, notFoundOr(err, "get host")
	}
	return Host{ID: row.ID.String(), Email: row.Email, PasswordHash: row.PasswordHash, Name: row.Name}, nil
}

func (s *Store) TouchHostLogin(ctx context.Context, hostID string) error {
	id, err := uuid.Parse(hostID)
	if err != nil {
		return fmt.Errorf("parse host id: %w", err)
	}
	return s.q.TouchHostLogin(ctx, id)
}

func (s *Store) CreateHost(ctx context.Context, email, name, passwordHash string) error {
	return s.q.UpsertHost(ctx, queries.UpsertHostParams{Email: email, Name: name, PasswordHash: passwordHash})
}

// CreateRoom enforces the active-room limit and assigns a unique 6-digit PIN.
func (s *Store) CreateRoom(ctx context.Context, in NewRoom) (Room, error) {
	hostID, err := uuid.Parse(in.HostID)
	if err != nil {
		return Room{}, fmt.Errorf("parse host id: %w", err)
	}
	for attempt := 0; attempt < pinAttempts; attempt++ {
		pin, err := randomPIN()
		if err != nil {
			return Room{}, err
		}
		room, err := s.insertRoom(ctx, hostID, pin, in)
		if isViolation(err, pgUniqueViolate) {
			continue
		}
		return room, err
	}
	return Room{}, errors.New("could not allocate a unique PIN")
}

func (s *Store) insertRoom(ctx context.Context, hostID uuid.UUID, pin string, in NewRoom) (Room, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Room{}, fmt.Errorf("begin create room: %w", err)
	}
	defer rollback(ctx, tx)

	q := s.q.WithTx(tx)
	if err := q.LockRoomCreation(ctx); err != nil {
		return Room{}, fmt.Errorf("lock room creation: %w", err)
	}
	active, err := q.CountActiveRooms(ctx)
	if err != nil {
		return Room{}, fmt.Errorf("count active rooms: %w", err)
	}
	if active >= MaxActiveRooms {
		return Room{}, ErrMaxRooms
	}
	id, err := q.InsertRoom(ctx, queries.InsertRoomParams{
		Pin: pin, HostID: hostID, Jenjang: in.Jenjang,
		ShortSession: in.ShortSession, AccuracyMode: in.AccuracyMode, QuestionIds: in.QuestionIDs,
	})
	if err != nil {
		return Room{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Room{}, fmt.Errorf("commit create room: %w", err)
	}
	return Room{ID: id.String(), PIN: pin, HostID: hostID.String(), Jenjang: in.Jenjang,
		ShortSession: in.ShortSession, AccuracyMode: in.AccuracyMode, Status: "lobby", QuestionIDs: in.QuestionIDs}, nil
}

func (s *Store) RoomByPIN(ctx context.Context, pin string) (Room, error) {
	row, err := s.q.GetActiveRoomByPIN(ctx, pin)
	if err != nil {
		return Room{}, notFoundOr(err, "get room by pin")
	}
	return roomFromRow(row.ID, row.Pin, row.HostID, row.Jenjang, row.Status, row.ShortSession, row.AccuracyMode, row.QuestionIds, row.StartedAt), nil
}

func (s *Store) RoomByID(ctx context.Context, roomID string) (Room, error) {
	id, err := uuid.Parse(roomID)
	if err != nil {
		return Room{}, ErrNotFound
	}
	row, err := s.q.GetRoomByID(ctx, id)
	if err != nil {
		return Room{}, notFoundOr(err, "get room")
	}
	return roomFromRow(row.ID, row.Pin, row.HostID, row.Jenjang, row.Status, row.ShortSession, row.AccuracyMode, row.QuestionIds, row.StartedAt), nil
}

func (s *Store) PlayerCount(ctx context.Context, roomID string) (int, error) {
	id, err := uuid.Parse(roomID)
	if err != nil {
		return 0, ErrNotFound
	}
	n, err := s.q.CountRoomPlayers(ctx, id)
	return int(n), err
}

func (s *Store) PlayersOfRoom(ctx context.Context, roomID string) ([]Player, error) {
	id, err := uuid.Parse(roomID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := s.q.ListRoomPlayers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list players: %w", err)
	}
	players := make([]Player, 0, len(rows))
	for _, r := range rows {
		players = append(players, Player{
			ID: r.ID.String(), Nickname: r.Nickname, School: r.SchoolName, Lang: strings.TrimSpace(r.Lang), Avatar: int(r.Avatar),
			Score: int(r.Score), CorrectCount: int(r.CorrectCount), TotalMs: int(r.TotalMs),
			CurrentIndex: int(r.CurrentIndex), Streak: int(r.Streak), Finished: r.Finished,
		})
	}
	return players, nil
}

// JoinRoom adds a player while holding a row lock so the 15-player cap cannot be exceeded.
func (s *Store) JoinRoom(ctx context.Context, in NewPlayer) (Player, error) {
	roomID, err := uuid.Parse(in.RoomID)
	if err != nil {
		return Player{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Player{}, fmt.Errorf("begin join: %w", err)
	}
	defer rollback(ctx, tx)

	q := s.q.WithTx(tx)
	if _, err := q.LockRoomForJoin(ctx, roomID); err != nil {
		return Player{}, notFoundOr(err, "lock room")
	}
	count, err := q.CountRoomPlayers(ctx, roomID)
	if err != nil {
		return Player{}, fmt.Errorf("count players: %w", err)
	}
	if count >= MaxPlayersInRoom {
		return Player{}, ErrRoomFull
	}
	return insertPlayer(ctx, tx, q, roomID, in)
}

func insertPlayer(ctx context.Context, tx pgx.Tx, q *queries.Queries, roomID uuid.UUID, in NewPlayer) (Player, error) {
	params := queries.InsertPlayerParams{RoomID: roomID, Nickname: in.Nickname, Jenjang: in.Jenjang, Avatar: toInt16(in.Avatar), Lang: in.Lang}
	schoolName := ""
	if in.SchoolID != nil {
		name, err := q.GetSchoolName(ctx, toInt32(*in.SchoolID))
		if err != nil {
			return Player{}, ErrInvalidSchool
		}
		params.SchoolID = pgtype.Int4{Int32: toInt32(*in.SchoolID), Valid: true}
		schoolName = name
	}
	id, err := q.InsertPlayer(ctx, params)
	switch {
	case isViolation(err, pgUniqueViolate):
		return Player{}, ErrNicknameTaken
	case err != nil:
		return Player{}, fmt.Errorf("insert player: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Player{}, fmt.Errorf("commit join: %w", err)
	}
	return Player{ID: id.String(), Nickname: in.Nickname, School: schoolName, Lang: in.Lang, Avatar: in.Avatar}, nil
}

func (s *Store) DeletePlayer(ctx context.Context, playerID string) error {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return ErrNotFound
	}
	return s.q.DeletePlayer(ctx, id)
}

// MarkRoomStarted moves a lobby room to running; it reports false if the room was not in the lobby.
func (s *Store) MarkRoomStarted(ctx context.Context, roomID string) (bool, error) {
	id, err := uuid.Parse(roomID)
	if err != nil {
		return false, ErrNotFound
	}
	rows, err := s.q.MarkRoomStarted(ctx, id)
	return rows > 0, err
}

func (s *Store) MarkRoomEnded(ctx context.Context, roomID string) error {
	id, err := uuid.Parse(roomID)
	if err != nil {
		return ErrNotFound
	}
	return s.q.MarkRoomEnded(ctx, id)
}

// SearchSchools returns up to 20 matches followed by the always-available special entries.
func (s *Store) SearchSchools(ctx context.Context, query string) ([]School, error) {
	matches, err := s.q.SearchSchools(ctx, queries.SearchSchoolsParams{Column1: escapeLike(query), Column2: specialSchools})
	if err != nil {
		return nil, fmt.Errorf("search schools: %w", err)
	}
	special, err := s.q.ListSchoolsByName(ctx, specialSchools)
	if err != nil {
		return nil, fmt.Errorf("list special schools: %w", err)
	}
	schools := make([]School, 0, len(matches)+len(special))
	for _, m := range matches {
		schools = append(schools, School{ID: int(m.ID), Name: m.Name, Jenjang: textPtr(m.Jenjang)})
	}
	for _, m := range special {
		schools = append(schools, School{ID: int(m.ID), Name: m.Name, Jenjang: textPtr(m.Jenjang)})
	}
	return schools, nil
}

func (s *Store) QuestionPool(ctx context.Context) ([]QuestionRef, error) {
	rows, err := s.q.ListSelectableQuestions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	pool := make([]QuestionRef, 0, len(rows))
	for _, r := range rows {
		pool = append(pool, QuestionRef{ID: r.ID, Site: int(r.Site), Level: int(r.Level)})
	}
	return pool, nil
}

func roomFromRow(id uuid.UUID, pin string, hostID uuid.UUID, jenjang, status string, short, accuracy bool, questionIDs []string, startedAt pgtype.Timestamptz) Room {
	room := Room{ID: id.String(), PIN: pin, HostID: hostID.String(), Jenjang: jenjang, Status: status,
		ShortSession: short, AccuracyMode: accuracy, QuestionIDs: questionIDs}
	if startedAt.Valid {
		room.StartedAt = startedAt.Time
	}
	return room
}

func randomPIN() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(pinSpace))
	if err != nil {
		return "", fmt.Errorf("generate pin: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.TrimSpace(s))
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func notFoundOr(err error, op string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}

func isViolation(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

func rollback(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		slog.Warn("rollback", "err", err)
	}
}
