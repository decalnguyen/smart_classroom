package ws

import (
	"log"
	"net/http"

	// Import your models package here

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool) // Connected clients
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NotificationsWsHandler handles WebSocket connections for notifications
func NotificationsWsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	clients[conn] = true
	log.Println("New client connected")
}

func HandleNotificationsWS(message []byte) {
	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Failed to send message to client: %v", err)
			client.Close()
			delete(clients, client)
		} else {
			log.Printf("Sent message to client: %s", message)
		}
	}
}
