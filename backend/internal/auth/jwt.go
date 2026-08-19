package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"kn-system/internal/config"
	"kn-system/internal/model"
)

// Claims is the JWT payload. UserID and Role are the only fields the middleware
// needs; keeping it small avoids leaking unnecessary identity into tokens.
type Claims struct {
	UserID uuid.UUID
	Email  string
	Role   model.Role
	jwt.RegisteredClaims
}

// JWT issues and verifies tokens. It is its own type so handlers depend on an
// interface-shaped dependency, not a concrete signing routine.
type JWT struct {
	secret string
	issuer string
	ttl    time.Duration
}

func NewJWT(cfg config.JWTConfig) *JWT {
	return &JWT{
		secret: cfg.Secret,
		issuer: cfg.Issuer,
		ttl:    time.Duration(cfg.ExpireHours) * time.Hour,
	}
}

// Issue creates a signed token for the given user. Expiry is computed once here
// so verification never has to trust client-supplied times.
func (j *JWT) Issue(u model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: u.ID,
		Email:  u.Email,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(j.secret))
}

// Verify validates the signature and expiry, returning the parsed claims.
var ErrInvalidToken = errors.New("invalid token")

func (j *JWT) Verify(raw string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(j.secret), nil
	}, jwt.WithIssuer(j.issuer))
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
