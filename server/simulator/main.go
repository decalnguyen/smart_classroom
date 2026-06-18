// Command simulator emulates an ESP32 classroom KIT by periodically emitting
// light / temperature / humidity / smoke readings. By default it publishes over
// MQTT (classroom/<room>/sensor/<type>) exactly like a real ESP32 node; set
// SENSOR_TRANSPORT=http to fall back to POSTing the HTTP API instead. Every Nth
// cycle it can inject a smoke spike above the danger threshold so the alarm +
// realtime notification pipeline can be demonstrated without physical hardware.
// Face scans always go over HTTP (that path belongs to the Jetson camera).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// parseRoomSet parses a comma-separated room list into a lookup set.
func parseRoomSet(csv string) map[string]bool {
	set := map[string]bool{}
	for _, s := range strings.Split(csv, ",") {
		if s = strings.TrimSpace(s); s != "" {
			set[s] = true
		}
	}
	return set
}

type reading struct {
	DeviceID   string  `json:"device_id"`
	DeviceType string  `json:"device_type"`
	Value      float64 `json:"value"`
	Status     string  `json:"status"`
}

// mqttPayload is the body an ESP32 publishes; the server reads value/status and
// derives device_id + type from the topic (classroom/<room>/sensor/<type>).
type mqttPayload struct {
	Value  float64 `json:"value"`
	Status string  `json:"status"`
}

var (
	transport  = env("SENSOR_TRANSPORT", "mqtt") // mqtt | http
	mqttClient mqtt.Client
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

var deviceKey = env("DEVICE_API_KEY", "dev-device-key")

// postJSON sends an authenticated device request (X-Device-Key header).
func postJSON(url string, body []byte) (int, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-Key", deviceKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func post(apiURL string, r reading) {
	body, _ := json.Marshal(r)
	code, err := postJSON(apiURL, body)
	if err != nil {
		log.Printf("post %s failed: %v", r.DeviceType, err)
		return
	}
	log.Printf("sent %s=%.1f (status %d)", r.DeviceType, r.Value, code)
}

// initMQTT connects to the broker so the simulator can publish like an ESP32.
func initMQTT() {
	broker := env("MQTT_BROKER", "tcp://rabbitmq:1883")
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetUsername(env("MQTT_USER", "admin")).
		SetPassword(env("MQTT_PASS", "admin")).
		SetClientID("smart-classroom-simulator").
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectTimeout(10 * time.Second)
	mqttClient = mqtt.NewClient(opts)
	if tok := mqttClient.Connect(); tok.Wait() && tok.Error() != nil {
		log.Fatalf("MQTT connect failed (%s): %v", broker, tok.Error())
	}
	log.Printf("simulator MQTT connected to %s", broker)
}

// publishMQTT publishes one reading to /<room>/<device>/value, the same topic
// shape a real ESP32 uses. The backend MQTT bridge ingests it.
func publishMQTT(room, suffix string, value float64) {
	topic := fmt.Sprintf("/%s/%s/value", room, suffix)
	body, _ := json.Marshal(mqttPayload{Value: value, Status: "active"})
	tok := mqttClient.Publish(topic, 0, false, body)
	tok.Wait()
	if tok.Error() != nil {
		log.Printf("mqtt publish %s failed: %v", topic, tok.Error())
		return
	}
	log.Printf("pub %s=%.1f", topic, value)
}

// emit sends one sensor reading via the configured transport. The resulting
// device_id (room-suffix) is identical on both paths, so the dashboard is
// unchanged regardless of transport. httpType keeps the legacy HTTP device_type.
func emit(apiURL, room, suffix, httpType string, value float64) {
	if transport == "mqtt" {
		publishMQTT(room, suffix, value)
		return
	}
	post(apiURL, reading{DeviceID: room + "-" + suffix, DeviceType: httpType, Value: value, Status: "active"})
}

// postScan simulates the AI camera reporting a recognized face. The server
// resolves a random enrolled student of the ongoing class and records attendance.
func postScan(scanURL string, classroomID int, status string) {
	now := time.Now()
	body, _ := json.Marshal(map[string]interface{}{
		"classroom_id": classroomID,
		"device_id":    fmt.Sprintf("cam-%d", classroomID),
		"status":       status,
		// Anti-replay fields the backend now requires on device events.
		"event_id": fmt.Sprintf("cam-%d-%d", classroomID, now.UnixNano()),
		"ts":       now.Format(time.RFC3339),
	})
	code, err := postJSON(scanURL, body)
	if err != nil {
		log.Printf("scan failed: %v", err)
		return
	}
	log.Printf("face-scan classroom %d (%s) -> %d", classroomID, status, code)
}

// scanLoop periodically simulates successful face scans across classrooms,
// with ~15% of recognitions flagged as "late".
func scanLoop(scanURL string, every time.Duration, maxClassroom int) {
	time.Sleep(12 * time.Second) // let mock data seed first
	rng := rand.New(rand.NewSource(7))
	for {
		status := "present"
		if rng.Float64() < 0.15 {
			status = "late"
		}
		postScan(scanURL, 1+rng.Intn(maxClassroom), status)
		time.Sleep(every)
	}
}

// genRooms generates classroom names matching the seed (A101-A105, B201-B205…).
func genRooms(n int) []string {
	rooms := make([]string, 0, n)
	for i := 0; i < n; i++ {
		if i < 5 {
			rooms = append(rooms, fmt.Sprintf("A10%d", i+1))
		} else {
			rooms = append(rooms, fmt.Sprintf("B20%d", i-4))
		}
	}
	return rooms
}

func main() {
	apiURL := env("API_URL", "http://backend:8081/sensor")
	scanURL := env("SCAN_URL", "http://backend:8081/attendance/scan")
	interval := time.Duration(envInt("INTERVAL_SECONDS", 5)) * time.Second
	// Every spikeEvery cycles, push a smoke reading above SMOKE_THRESHOLD (0 = off).
	spikeEvery := envInt("SPIKE_EVERY", 0)
	smokeThreshold := float64(envInt("SMOKE_THRESHOLD", 300))
	scanEvery := time.Duration(envInt("SCAN_EVERY_SECONDS", 20)) * time.Second
	roomCount := envInt("ROOM_COUNT", 10)
	allRooms := genRooms(roomCount)

	// Rooms with real ESP32 hardware: skip simulated sensor data so we don't
	// clash with the physical devices publishing to the same topics.
	excludeCSV := env("SENSOR_EXCLUDE_ROOMS", "A101,A102,B101")
	excluded := parseRoomSet(excludeCSV)
	rooms := make([]string, 0, len(allRooms))
	for _, r := range allRooms {
		if !excluded[r] {
			rooms = append(rooms, r)
		}
	}

	if transport == "mqtt" {
		initMQTT()
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	log.Printf("simulator sensors via %s for %d rooms (excluded: %s); face-scan via HTTP %s", transport, len(rooms), excludeCSV, scanURL)

	// Background: simulate successful face scans -> realtime attendance.
	go scanLoop(scanURL, scanEvery, roomCount)

	// Give the API time to come up.
	time.Sleep(8 * time.Second)

	cycle := 0
	for {
		cycle++
		// Emit a reading set for EVERY classroom so the dashboard can show an
		// overview of all rooms (each room has slightly different baselines).
		for idx, room := range rooms {
			off := float64(idx)
			light := 380 + off*12 + 150*math.Sin(float64(cycle)/10+off) + rng.Float64()*40
			emit(apiURL, room, "light", "light", round(light))

			temp := 26.5 + off*0.25 + 2*math.Sin(float64(cycle)/15+off) + rng.Float64()*0.7
			emit(apiURL, room, "temp", "temperature", round(temp))

			hum := 58 + off*0.8 + 5*math.Sin(float64(cycle)/20+off) + rng.Float64()*2
			emit(apiURL, room, "humi", "humidity", round(hum))

			smoke := 70 + off*3 + rng.Float64()*50
			if spikeEvery > 0 && (cycle+idx)%spikeEvery == 0 {
				smoke = smokeThreshold + 40 + rng.Float64()*100
				log.Printf("🔥 smoke spike in %s: %.0f", room, smoke)
			}
			emit(apiURL, room, "smoke", "smoke", round(smoke))
		}
		time.Sleep(interval)
	}
}

func round(v float64) float64 {
	return math.Round(v*10) / 10
}
