package ws

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var sensorClients = make(map[*websocket.Conn]bool) // Connected sensor clients
var sensorUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins (or customize)
	},
}

// SensorWsHandler handles WebSocket connections for sensor data
func SensorWsHandler(c *gin.Context) {
	conn, err := sensorUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	sensorClients[conn] = true
	log.Println("New sensor client connected")

	// Keep connection alive by reading messages
	go func() {
		defer func() {
			conn.Close()
			delete(sensorClients, conn)
			log.Println("Sensor client disconnected")
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Unexpected WebSocket close: %v", err)
				}
				break
			}
		}
	}()
}

func HandleSensorNotificationsWS(message []byte) {
	for client := range sensorClients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("❌ Failed to send message to sensor client: %v", err)
			client.Close()
			delete(sensorClients, client)
		} else {
			log.Printf("✅ Sent message to sensor client: %s", message)
		}
	}
}
