// Package tokens issues and validates JWTs. It is pure computation and never touches the database.
package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour

	issuer = "MagicStream"
)

// Claims is the JWT payload. It carries only what authorization needs; names and email come from API responses.
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Manager holds two different signing keys, so an access token can never pass as a refresh token or vice versa.
type Manager struct {
	accessKey  []byte
	refreshKey []byte
	now        func() time.Time
}

func NewManager(accessKey, refreshKey []byte) *Manager {
	return &Manager{accessKey: accessKey, refreshKey: refreshKey, now: time.Now}
}

// Generate issues an access/refresh token pair.
func (m *Manager) Generate(userID, role string) (accessToken, refreshToken string, err error) {
	accessToken, err = m.sign(userID, role, AccessTokenTTL, m.accessKey)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}
	refreshToken, err = m.sign(userID, role, RefreshTokenTTL, m.refreshKey)
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}
	return accessToken, refreshToken, nil
}

func (m *Manager) ValidateAccess(tokenString string) (*Claims, error) {
	return m.parse(tokenString, m.accessKey)
}

func (m *Manager) ValidateRefresh(tokenString string) (*Claims, error) {
	return m.parse(tokenString, m.refreshKey)
}

func (m *Manager) sign(userID, role string, ttl time.Duration, key []byte) (string, error) {
	now := m.now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			// A random ID per token keeps two tokens issued in the same second distinct (refresh token rotation relies on this).
			ID: randomID(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
}

func (m *Manager) parse(tokenString string, key []byte) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims,
		func(*jwt.Token) (any, error) { return key, nil },
		// Restrict the algorithm before the signature is checked, to prevent alg confusion attacks.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil {
		return nil, err
	}
	if claims.UserID == "" {
		return nil, errors.New("token has no user id")
	}
	return claims, nil
}

// Hash returns the SHA-256 hex digest of a token. The database stores only the refresh token's digest,
// so a leaked database does not yield usable tokens.
// Tokens are high-entropy random strings, so a fast hash is enough; bcrypt is not needed.
func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // Since Go 1.24 crypto/rand.Read never returns an error (it crashes instead)
	return hex.EncodeToString(b)
}
