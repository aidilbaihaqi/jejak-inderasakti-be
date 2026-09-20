package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Role string

const (
	RoleHost   Role = "host"
	RolePlayer Role = "player"

	MinSecretLength = 32
	hostTokenTTL    = 8 * time.Hour
	playerTokenTTL  = 6 * time.Hour
)

var ErrInvalidToken = errors.New("invalid or expired token")

// Claims carries the identity in Subject (host id or player id); players also carry their room.
type Claims struct {
	Role   Role   `json:"role"`
	RoomID string `json:"room_id,omitempty"`
	jwt.RegisteredClaims
}

type Tokens struct {
	secret []byte
	now    func() time.Time
}

func NewTokens(secret string) (*Tokens, error) {
	if len(secret) < MinSecretLength {
		return nil, fmt.Errorf("JWT_SECRET must be at least %d characters", MinSecretLength)
	}
	return &Tokens{secret: []byte(secret), now: time.Now}, nil
}

func (t *Tokens) IssueHost(hostID string) (string, error) {
	return t.sign(Claims{Role: RoleHost}, hostID, hostTokenTTL)
}

func (t *Tokens) IssuePlayer(roomID, playerID string) (string, error) {
	return t.sign(Claims{Role: RolePlayer, RoomID: roomID}, playerID, playerTokenTTL)
}

func (t *Tokens) Parse(raw string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(raw, claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil || claims.Subject == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (t *Tokens) sign(claims Claims, subject string, ttl time.Duration) (string, error) {
	now := t.now()
	claims.Subject = subject
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
