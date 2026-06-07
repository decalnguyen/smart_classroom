package handlers

import (
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ========================
// === CLIENT REGISTRY ====
// ========================
//
// The registries are touched by both the HTTP/WS goroutines (register) and the
// RabbitMQ consumer goroutines (broadcast), so every access is guarded by a
// mutex to avoid concurrent map access panics.

var (
	clientsMu           sync.Mutex
	notificationClients = make(map[*websocket.Conn]bool)
	sensorClients       = make(map[*websocket.Conn]bool)
	attendanceClients   = make(map[*websocket.Conn]bool)
)

// allowedWSOrigins is the set of browser origins permitted to open a socket.
// Empty origin (non-browser clients like the simulator / CLI) is always allowed.
func allowedWSOrigins() map[string]bool {
	m := map[string]bool{
		"http://localhost:3000":  true,
		"http://localhost:5173":  true,
		"http://127.0.0.1:3000":  true,
		"http://127.0.0.1:5173":  true,
	}
	if o := os.Getenv("FRONTEND_ORIGIN"); o != "" {
		m[o] = true
	}
	return m
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			return true // non-browser client
		}
		return allowedWSOrigins()[origin]
	},
}

// broadcast writes a message to every client in the given registry, dropping
// any client whose write fails.
func broadcast(clients map[*websocket.Conn]bool, message []byte, label string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("❌ Failed to send to %s client: %v", label, err)
			client.Close()
			delete(clients, client)
		}
	}
}

func register(clients map[*websocket.Conn]bool, conn *websocket.Conn) {
	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()
}

func unregister(clients map[*websocket.Conn]bool, conn *websocket.Conn) {
	clientsMu.Lock()
	delete(clients, conn)
	clientsMu.Unlock()
	conn.Close()
}

// keepAlive blocks reading from the connection until it closes, so the registry
// is cleaned up promptly when a client disconnects.
func keepAlive(clients map[*websocket.Conn]bool, conn *websocket.Conn, label string) {
	defer func() {
		unregister(clients, conn)
		log.Printf("⚠️ %s client disconnected", label)
	}()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
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
	register(notificationClients, conn)
	log.Println("✅ New notification client connected")
	go keepAlive(notificationClients, conn, "notification")
}

// HandleNotificationsWS pushes a message to all notification clients.
func HandleNotificationsWS(message []byte) {
	broadcast(notificationClients, message, "notification")
}

// HandleSensorWS pushes a message to all sensor clients.
func HandleSensorWS(message []byte) {
	broadcast(sensorClients, message, "sensor")
}

// HandleAttendanceWS pushes a message to all attendance clients.
func HandleAttendanceWS(message []byte) {
	broadcast(attendanceClients, message, "attendance")
}

// AttendanceWsHandler upgrades a client onto the realtime attendance feed.
func AttendanceWsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ Failed to upgrade attendance WS: %v", err)
		return
	}
	register(attendanceClients, conn)
	log.Println("✅ New attendance client connected")
	go keepAlive(attendanceClients, conn, "attendance")
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
	register(sensorClients, conn)
	log.Println("✅ New sensor client connected")
	go keepAlive(sensorClients, conn, "sensor")
}
