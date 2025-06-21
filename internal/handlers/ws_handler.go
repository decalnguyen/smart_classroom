package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ========================
// === CLIENT REGISTRY ====
// ========================

var notificationClients = make(map[*websocket.Conn]bool)
var sensorClients = make(map[*websocket.Conn]bool)

// =====================
// === UPGRADERS =======
// =====================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ================================
// === NOTIFICATION WS HANDLER ====
// ================================

func NotificationsWsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ Failed to upgrade notification WS: %v", err)
		return
	}
	notificationClients[conn] = true
	log.Println("✅ New notification client connected")
}

// Send message to all notification clients
func HandleNotificationsWS(message []byte) {
	for client := range notificationClients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("❌ Failed to send message to notification client: %v", err)
			client.Close()
			delete(notificationClients, client)
		} else {
			log.Printf("📢 Sent to notification client: %s", message)
		}
	}
}

// ==========================
// === SENSOR WS HANDLER ====
// ==========================

func SensorWsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ Failed to upgrade sensor WS: %v", err)
		return
	}
	sensorClients[conn] = true
	log.Println("✅ New sensor client connected")

	// Keep connection alive
	go func() {
		defer func() {
			conn.Close()
			delete(sensorClients, conn)
			log.Println("⚠️ Sensor client disconnected")
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("⚠️ Unexpected sensor WS close: %v", err)
				}
				break
			}
		}
	}()
}

// Send message to all sensor clients
func HandleSensorNotificationsWS(message []byte) {
	for client := range sensorClients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("❌ Failed to send message to sensor client: %v", err)
			client.Close()
			delete(sensorClients, client)
		} else {
			log.Printf("📡 Sent to sensor client: %s", message)
		}
	}
}
