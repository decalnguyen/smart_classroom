package middleware

import (
	"net/http"
	"strings"

	"smart_classroom/internal/utils"

	"github.com/gin-gonic/gin"
)

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
