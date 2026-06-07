package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtKey is loaded from the JWT_SECRET environment variable. A default is kept
// only as a development fallback so the stack still boots locally; production
// deployments MUST set JWT_SECRET (see docker-compose / k8s secrets).
var jwtKey = func() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("dev_insecure_secret_change_me")
}()

type CustomClaims struct {
	AccountID string `json:"account_id"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT generates a JWT token for a user.
func GenerateJWT(id, role string) (string, error) {
	claims := CustomClaims{
		AccountID: id,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// ParseClaims validates a token string and returns the full claim set
// (account id + role). Use this when role-based decisions are needed.
func ParseClaims(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			return nil, errors.New("token expired")
		}
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// ValidateJWT validates a token and returns the account id (kept for
// backward compatibility with existing call sites).
func ValidateJWT(tokenString string) (string, error) {
	claims, err := ParseClaims(tokenString)
	if err != nil {
		return "", err
	}
	return claims.AccountID, nil
}
