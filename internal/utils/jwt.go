package utils

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	insecureDefaultSecret = "dev_insecure_secret_change_me" // legacy default we now refuse/warn on
	minSecretLen          = 32                              // HS256: secret should be >= 256 bits
	defaultIssuer         = "smart-classroom"
	defaultTTL            = 24 * time.Hour
)

// isProd reports whether we're running in a production-like environment.
func isProd() bool {
	return os.Getenv("GIN_MODE") == "release" || strings.EqualFold(os.Getenv("APP_ENV"), "production")
}

// jwtKey is resolved ONCE at startup from JWT_SECRET — never hardcoded in the
// binary. Best-practice rules:
//   - production (GIN_MODE=release / APP_ENV=production): a missing, too-short, or
//     known-insecure secret is FATAL (fail fast, don't ship a guessable key).
//   - development: warn loudly; if truly unset, generate a random EPHEMERAL secret
//     so local runs work (tokens reset on restart). For a stable dev secret put
//     JWT_SECRET in .env (git-ignored) — see .env.example.
//
// Only the http-api server signs/verifies JWTs (the WS server is origin-checked),
// so a single configured secret is sufficient.
var jwtKey = loadSecret()

func loadSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	prod := isProd()
	switch {
	case s == "":
		if prod {
			log.Fatal("FATAL: JWT_SECRET is required in production. Generate one: `openssl rand -base64 48`")
		}
		buf := make([]byte, 48)
		if _, err := rand.Read(buf); err != nil {
			log.Fatalf("cannot generate ephemeral JWT secret: %v", err)
		}
		log.Println("⚠️  JWT_SECRET unset — using an EPHEMERAL dev secret (sessions reset on restart). Set JWT_SECRET in .env for stable dev.")
		return []byte(hex.EncodeToString(buf))
	case s == insecureDefaultSecret:
		if prod {
			log.Fatal("FATAL: JWT_SECRET is the insecure default. Set a real secret in production.")
		}
		log.Println("⚠️  JWT_SECRET is the insecure DEV default — never use this in production.")
		return []byte(s)
	case len(s) < minSecretLen:
		if prod {
			log.Fatalf("FATAL: JWT_SECRET too short (%d bytes); need >= %d.", len(s), minSecretLen)
		}
		log.Printf("⚠️  JWT_SECRET is short (%d bytes); use >= %d bytes in production.", len(s), minSecretLen)
		return []byte(s)
	default:
		return []byte(s)
	}
}

func issuer() string {
	if v := os.Getenv("JWT_ISSUER"); v != "" {
		return v
	}
	return defaultIssuer
}

func tokenTTL() time.Duration {
	if v := os.Getenv("JWT_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Hour
		}
	}
	return defaultTTL
}

// CustomClaims carries the app-specific claims alongside the registered set.
type CustomClaims struct {
	AccountID string `json:"account_id"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT issues a signed HS256 token with the standard registered claims
// (iss/sub/iat/nbf/exp + a unique jti for future revocation/audit).
func GenerateJWT(id, role string) (string, error) {
	now := time.Now()
	jti := make([]byte, 16)
	_, _ = rand.Read(jti)
	claims := CustomClaims{
		AccountID: id,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer(),
			Subject:   id,
			ID:        hex.EncodeToString(jti),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL())),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtKey)
}

// ParseClaims verifies a token and returns its claims. Hardened against the
// common JWT pitfalls: it pins the algorithm to HS256 (blocks `alg:none` and
// RS/HS confusion), requires an expiry, validates the issuer, and allows a small
// clock-skew leeway. Signature + exp + nbf + iss are all checked by the parser.
func ParseClaims(tokenString string) (*CustomClaims, error) {
	claims := &CustomClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(issuer()),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// ValidateJWT validates a token and returns the account id (back-compat helper).
func ValidateJWT(tokenString string) (string, error) {
	claims, err := ParseClaims(tokenString)
	if err != nil {
		return "", err
	}
	return claims.AccountID, nil
}
