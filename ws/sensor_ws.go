package ws

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var sensorClients = make(map[*websocket.Conn]bool) // Connected sensor clients
var sensorUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
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
}

func HandleSensorNotificationsWS(message []byte) {
	for client := range sensorClients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("Failed to send message to sensor client: %v", err)
			client.Close()
			delete(sensorClients, client)
		} else {
			log.Printf("Sent message to sensor client: %s", message)
		}
	}
}
