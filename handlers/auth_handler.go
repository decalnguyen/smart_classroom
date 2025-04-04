package handlers

import (
	"log"
	"net/http"

	"smart_classroom/db"
	"smart_classroom/models"
	"smart_classroom/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func SignUp(c *gin.Context) {
	var userInput struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BindJSON(&userInput); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	//Hash the password using bcrypt
	password, _ := bcrypt.GenerateFromPassword([]byte(userInput.Password), bcrypt.DefaultCost)
	var existingUser models.User
	if err := db.DB.Where("username = ?", userInput.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	// Create a new user
	user := models.User{
		Username: userInput.Username,
		Password: password,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		log.Printf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User created successfully"})
}

func Login(c *gin.Context) {
	var userInput struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var dbUser models.User

	// Parse JSON input
	if err := c.BindJSON(&userInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Check if the user exists in the database
	if err := db.DB.Where("username = ?", userInput.Username).First(&dbUser).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username "})
		return
	}

	// Verify password (in a real app, hash passwords using bcrypt)
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(userInput.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateJWT(userInput.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.SetCookie(
		"auth_token", // Cookie name
		token,        // Cookie value (JWT token)
		3600,         // Max age in seconds (1 hour)
		"/",          // Path
		"",           // Domain (empty means default domain)
		false,        // Secure (set to true if using HTTPS)
		true,         // HttpOnly (prevents JavaScript access)
	)

	c.JSON(http.StatusOK, gin.H{"token": token})
}
