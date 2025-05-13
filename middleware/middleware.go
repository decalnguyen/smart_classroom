package middleware

import (
	"net/http"
	"strings"

	"smart_classroom/utils"

	"github.com/gin-gonic/gin"
)

// Middleware to restrict access to teachers only
func TeacherOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("auth_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Validate the token and check the role
		username, err := utils.ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access restricted to teachers only"})
			c.Abort()
			return
		}

		// Add the username to the context for further use
		c.Set("username", username)
		c.Next()
	}
}

// Middleware to restrict access to the classroom network
func ClassroomNetworkMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !strings.HasPrefix(clientIP, "192.168.1.") { // Replace with your network prefix
			c.JSON(http.StatusForbidden, gin.H{"error": "Access restricted to classroom network"})
			c.Abort()
			return
		}
		c.Next()
	}
}
