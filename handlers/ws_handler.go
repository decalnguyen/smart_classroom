package handlers

import (
	"log"
	"net/http"
	"time"

	"smart_classroom/models" // Import your models package here

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func WSNotificationHandler(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing account_id query parameter"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade:", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Example: simulate notification - replace with real data
			notification := models.Notification{
				ID:      123,                                         // replace with actual ID
				Message: "New notification for account " + accountID, // adjust fields as per your struct
			}

			err := conn.WriteJSON(notification)
			if err != nil {
				log.Println("Write error:", err)
				return
			}
		}
	}
}
