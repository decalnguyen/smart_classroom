package handlers

import (
	"log"
	"net/http"

	"smart_classroom/internal/db"
	"smart_classroom/internal/middleware"
	"smart_classroom/internal/models"
	"smart_classroom/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// validRoles is the set of roles a user may hold.
var validRoles = map[string]bool{"admin": true, "teacher": true, "student": true}

func SignUp(c *gin.Context) {
	var userInput struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := c.BindJSON(&userInput); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if userInput.Username == "" || userInput.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	// Default to the least-privileged role and reject unknown roles.
	role := userInput.Role
	if role == "" {
		role = "student"
	}
	if !validRoles[role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	password, err := bcrypt.GenerateFromPassword([]byte(userInput.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var existingUser models.User
	if err := db.DB.Where("username = ?", userInput.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	user := models.User{
		AccountID: uuid.New().String(),
		Username:  userInput.Username,
		Password:  password,
		Role:      role,
	}
	if err := db.DB.Create(&user).Error; err != nil {
		log.Printf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "User created successfully",
		"account_id": user.AccountID,
		"role":       user.Role,
	})
}

func Login(c *gin.Context) {
	var userInput struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var dbUser models.User

	if err := c.BindJSON(&userInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := db.DB.Where("username = ?", userInput.Username).First(&dbUser).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword(dbUser.Password, []byte(userInput.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	token, err := utils.GenerateJWT(dbUser.AccountID, dbUser.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Also set an HttpOnly cookie for cookie-based clients.
	c.SetCookie("auth_token", token, 24*3600, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"role":       dbUser.Role,
		"account_id": dbUser.AccountID,
		"username":   dbUser.Username,
	})
}

// User returns the authenticated user's profile based on the JWT.
func User(c *gin.Context) {
	token := middleware.ExtractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	claims, err := utils.ParseClaims(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	var user models.User
	if err := db.DB.Where("account_id = ?", claims.AccountID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account_id": user.AccountID,
		"username":   user.Username,
		"role":       user.Role,
	})
}

func Logout(c *gin.Context) {
	c.SetCookie("auth_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
