package security

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// TokenType is the kind of token a Claims represents. Refresh tokens are
// long-lived and only accepted on the refresh endpoint.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// Claims is the structured JWT body. UserID + SessionID are the canonical
// identifiers; Authorities is the union of role + permission names (each role
// prefixed with `ROLE_` to match Spring's convention).
type Claims struct {
	UserID         uint64    `json:"uid"`
	Username       string    `json:"sub"`
	SessionID      uint64    `json:"sid,omitempty"`
	ImpersonatorID uint64    `json:"imp,omitempty"`
	Authorities    []string  `json:"authorities"`
	Type           TokenType `json:"typ"`
	jwt.RegisteredClaims
}

// JwtService issues and verifies tokens. Construct with NewJwtService —
// the constructor decodes a base64-encoded secret so .env can carry it safely.
type JwtService struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewJwtService validates the configured secret and returns a JwtService.
// The secret is base64-decoded if it looks base64-encoded; otherwise its raw
// bytes are used (so non-base64 secrets still work, just with less entropy).
func NewJwtService(cfg config.JWTConfig) (*JwtService, error) {
	if cfg.Secret == "" {
		return nil, errors.New("jwt: empty secret")
	}
	raw, err := base64.StdEncoding.DecodeString(cfg.Secret)
	if err != nil || len(raw) < 32 {
		// Fall back to using the secret bytes directly; warn-worthy in prod.
		raw = []byte(cfg.Secret)
	}
	if len(raw) < 32 {
		return nil, errors.New("jwt: secret must be at least 32 bytes after decoding")
	}
	return &JwtService{
		secret:     raw,
		issuer:     cfg.Issuer,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
	}, nil
}

// IssueAccess builds and signs an access token.
func (s *JwtService) IssueAccess(userID uint64, username string, sessionID uint64, impersonatorID uint64, authorities []string) (string, time.Time, error) {
	return s.issue(userID, username, sessionID, impersonatorID, authorities, TokenTypeAccess, s.accessTTL)
}

// IssueRefresh builds and signs a refresh token. The refresh token carries the
// same authorities snapshot so token-rotate doesn't need an extra RBAC lookup
// in the happy path.
func (s *JwtService) IssueRefresh(userID uint64, username string, sessionID uint64, impersonatorID uint64, authorities []string) (string, time.Time, error) {
	return s.issue(userID, username, sessionID, impersonatorID, authorities, TokenTypeRefresh, s.refreshTTL)
}

func (s *JwtService) issue(userID uint64, username string, sessionID uint64, impersonatorID uint64, authorities []string, typ TokenType, ttl time.Duration) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(ttl)
	claims := Claims{
		UserID:         userID,
		Username:       username,
		SessionID:      sessionID,
		ImpersonatorID: impersonatorID,
		Authorities:    authorities,
		Type:           typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			Subject:   username,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// Parse verifies the token signature + standard claims and returns the body.
// Returns an error for expired, malformed, or wrong-issuer tokens.
func (s *JwtService) Parse(raw string) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	},
		jwt.WithIssuer(s.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, errors.New("token is invalid")
	}
	return claims, nil
}

// AccessTTL exposes the configured access TTL for callers that need to surface
// the expiry in HTTP responses.
func (s *JwtService) AccessTTL() time.Duration { return s.accessTTL }

// RefreshTTL exposes the configured refresh TTL.
func (s *JwtService) RefreshTTL() time.Duration { return s.refreshTTL }
