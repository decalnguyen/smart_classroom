package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"
	"smart_classroom/internal/utils"

	"github.com/gin-gonic/gin"
)

const insecureDeviceKey = "dev-device-key" // legacy default we now refuse/warn on

// deviceMasterKey is the optional shared fleet key, resolved ONCE at startup (not
// hardcoded). Empty = master key disabled (only per-device tokens in
// device_credentials are accepted). In production the insecure default is fatal.
var deviceMasterKey = loadDeviceMasterKey()

func loadDeviceMasterKey() string {
	k := os.Getenv("DEVICE_API_KEY")
	prod := os.Getenv("GIN_MODE") == "release" || strings.EqualFold(os.Getenv("APP_ENV"), "production")
	switch {
	case k == "":
		log.Println("ℹ️  DEVICE_API_KEY unset — shared master key disabled; only per-device tokens (device_credentials) are accepted.")
		return ""
	case k == insecureDeviceKey:
		if prod {
			log.Fatal("FATAL: DEVICE_API_KEY is the insecure default in production. Set a real key or use per-device tokens.")
		}
		log.Println("⚠️  DEVICE_API_KEY is the insecure DEV default — do not use in production.")
		return k
	default:
		return k
	}
}

// RateLimit is a simple in-memory sliding-window limiter per client IP.
// Protects brute-forceable endpoints (e.g. /login) and device ingestion.
var (
	rlMu   sync.Mutex
	rlHits = map[string][]int64{}
)

func RateLimit(maxPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now().UnixMilli()
		cutoff := now - 60_000
		rlMu.Lock()
		kept := rlHits[ip][:0]
		for _, t := range rlHits[ip] {
			if t > cutoff {
				kept = append(kept, t)
			}
		}
		if len(kept) >= maxPerMinute {
			rlHits[ip] = kept
			rlMu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Quá nhiều yêu cầu, vui lòng thử lại sau"})
			return
		}
		rlHits[ip] = append(kept, now)
		rlMu.Unlock()
		c.Next()
	}
}

// RequireDevice authenticates an edge device (ESP32 / Jetson) via the
// X-Device-Key header. Accepts either the shared DEVICE_API_KEY (env) or a
// per-device token registered in device_credentials. Sets device_id in context.
func RequireDevice() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Device-Key")
		if key == "" {
			key = ExtractToken(c) // also allow Authorization: Bearer <token>
		}
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing device key"})
			return
		}
		// Shared master key (fast path for fleets behind a gateway), if configured.
		if deviceMasterKey != "" && key == deviceMasterKey {
			c.Next()
			return
		}
		// Per-device token.
		var cred models.DeviceCredential
		if err := db.DB.Where("token = ? AND active = ?", key, true).First(&cred).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid device key"})
			return
		}
		db.DB.Model(&models.DeviceCredential{}).Where("device_id = ?", cred.DeviceID).Update("last_seen", time.Now())
		c.Set("device_id", cred.DeviceID)
		c.Set("device_kind", cred.Kind)
		c.Next()
	}
}

// ExtractToken pulls the JWT from either the `Authorization: Bearer <token>`
// header or the `auth_token` cookie, so both the SPA (header) and any
// cookie-based client work uniformly.
func ExtractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth != "" {
		// Accept both "Bearer <token>" and a raw token.
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			return strings.TrimSpace(auth[7:])
		}
		return strings.TrimSpace(auth)
	}
	if cookie, err := c.Cookie("auth_token"); err == nil {
		return cookie
	}
	return ""
}

// RequireRole returns a middleware that authenticates the request and, if any
// roles are provided, ensures the caller's role is one of them. Calling it with
// no roles means "any authenticated user". On success it stores account_id and
// role in the gin context for downstream handlers.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ExtractToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication token"})
			return
		}

		claims, err := utils.ParseClaims(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		c.Set("account_id", claims.AccountID)
		c.Set("role", claims.Role)

		if len(roles) > 0 {
			allowed := false
			for _, r := range roles {
				if claims.Role == r {
					allowed = true
					break
				}
			}
			if !allowed {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions for this action"})
				return
			}
		}

		c.Next()
	}
}

// ClassroomNetworkMiddleware restricts access to a classroom LAN prefix.
// Kept available (opt-in) for deployments that want network-level gating.
func ClassroomNetworkMiddleware(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.ClientIP(), prefix) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access restricted to classroom network"})
			return
		}
		c.Next()
	}
}
