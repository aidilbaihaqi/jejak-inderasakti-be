package auth

import (
	"strings"
	"testing"
	"time"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func newTestTokens(t *testing.T) *Tokens {
	t.Helper()
	tokens, err := NewTokens(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	return tokens
}

func TestNewTokensRejectsShortSecret(t *testing.T) {
	if _, err := NewTokens("too-short"); err == nil {
		t.Error("short secret should be rejected")
	}
}

func TestHostTokenRoundTrip(t *testing.T) {
	tokens := newTestTokens(t)
	raw, err := tokens.IssueHost("host-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tokens.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != RoleHost || claims.Subject != "host-1" {
		t.Errorf("unexpected claims %+v", claims)
	}
	if ttl := claims.ExpiresAt.Sub(claims.IssuedAt.Time); ttl != 8*time.Hour {
		t.Errorf("host token ttl = %v, want 8h", ttl)
	}
}

func TestPlayerTokenCarriesRoomAndPlayer(t *testing.T) {
	tokens := newTestTokens(t)
	raw, err := tokens.IssuePlayer("room-1", "player-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tokens.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != RolePlayer || claims.RoomID != "room-1" || claims.Subject != "player-1" {
		t.Errorf("unexpected claims %+v", claims)
	}
}

func TestParseRejectsBadTokens(t *testing.T) {
	tokens := newTestTokens(t)
	valid, _ := tokens.IssueHost("host-1")
	other, _ := NewTokens(strings.Repeat("x", 40))
	foreign, _ := other.IssueHost("host-1")

	expired := newTestTokens(t)
	expired.now = func() time.Time { return time.Now().Add(-9 * time.Hour) }
	old, _ := expired.IssueHost("host-1")

	cases := map[string]string{
		"empty":           "",
		"garbage":         "not-a-jwt",
		"wrong signature": foreign,
		"expired":         old,
		"tampered":        valid[:len(valid)-2] + "xx",
	}
	for name, raw := range cases {
		if _, err := tokens.Parse(raw); err == nil {
			t.Errorf("%s: token should be rejected", name)
		}
	}
}
