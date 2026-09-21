package http

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/aidilbaihaqi/jejak-inderasakti-be/api/internal/store"
)

const (
	utf8BOM         = "\ufeff" // lets Excel open UTF-8 names correctly
	csvFormulaChars = "=+-@\t\r"
)

var csvHeader = []string{"rank", "nickname", "school", "jenjang", "score", "correct_count", "total_ms"}

// exportResults streams the room's final standings as CSV. Only the host who owns the room may download it.
func (a *API) exportResults(w http.ResponseWriter, r *http.Request) {
	room, err := a.Store.RoomByID(r.Context(), r.PathValue("id"))
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "ROOM_NOT_FOUND", "room not found")
		return
	case err != nil:
		writeInternal(w, "load room for export", err)
		return
	case room.HostID != hostIDFrom(r.Context()):
		writeError(w, http.StatusForbidden, "FORBIDDEN", "this room belongs to another host")
		return
	}
	rows, err := a.Store.RoomResults(r.Context(), room.ID)
	if err != nil {
		writeInternal(w, "load room results", err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="hasil-%s.csv"`, room.PIN))
	writeResultsCSV(w, rows)
}

func writeResultsCSV(w http.ResponseWriter, rows []store.ResultRow) {
	if _, err := w.Write([]byte(utf8BOM)); err != nil {
		return
	}
	out := csv.NewWriter(w)
	out.UseCRLF = true // RFC 4180, and what Excel expects
	records := [][]string{csvHeader}
	for _, row := range rows {
		records = append(records, []string{
			strconv.Itoa(row.Rank), safeCell(row.Nickname), safeCell(row.School), row.Jenjang,
			strconv.Itoa(row.Score), strconv.Itoa(row.CorrectCount), strconv.Itoa(row.TotalMs),
		})
	}
	if err := out.WriteAll(records); err != nil {
		slog.Debug("write results csv", "err", err) // the client went away; nothing left to tell it
	}
}

// safeCell stops spreadsheet apps from running a nickname or school name as a formula.
func safeCell(s string) string {
	if s != "" && strings.ContainsRune(csvFormulaChars, rune(s[0])) {
		return "'" + s
	}
	return s
}

func (a *API) schoolLeaderboard(w http.ResponseWriter, r *http.Request) {
	ranks, err := a.Store.SchoolLeaderboard(r.Context())
	if err != nil {
		writeInternal(w, "school leaderboard", err)
		return
	}
	body := make([]map[string]any, 0, len(ranks))
	for _, s := range ranks {
		body = append(body, map[string]any{"school": s.School, "avg_score": math.Round(s.AvgScore*100) / 100, "player_count": s.PlayerCount})
	}
	writeJSON(w, http.StatusOK, body)
}

// clientIP identifies the caller for rate limiting. Behind Caddy the peer is the proxy, so with
// trustProxy the last X-Forwarded-For entry (the one Caddy appends) is used instead.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		if ip := strings.TrimSpace(forwarded[len(forwarded)-1]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
