package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

// Claims are the JWT payload fields.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Config holds JWT signing secrets.
type Config struct {
	AccessSecret  string
	RefreshSecret string
}

// Manager handles token generation and validation.
type Manager struct {
	cfg Config
}

// New creates a new token Manager.
func New(cfg Config) *Manager {
	return &Manager{cfg: cfg}
}

// GenerateAccessToken creates a signed 15-minute access token.
func (m *Manager) GenerateAccessToken(userID, role string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.cfg.AccessSecret))
}

// GenerateRefreshToken creates a signed 7-day refresh token.
func (m *Manager) GenerateRefreshToken(userID, role string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.cfg.RefreshSecret))
}

// ValidateAccessToken parses and validates an access token.
func (m *Manager) ValidateAccessToken(tokenStr string) (*Claims, error) {
	return m.parse(tokenStr, m.cfg.AccessSecret)
}

// ValidateRefreshToken parses and validates a refresh token.
func (m *Manager) ValidateRefreshToken(tokenStr string) (*Claims, error) {
	return m.parse(tokenStr, m.cfg.RefreshSecret)
}

// HashToken returns a SHA-256 hex hash of the token for safe DB storage.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (m *Manager) parse(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
