// Command simulator emulates an ESP32 classroom KIT by periodically POSTing
// light / temperature / humidity / smoke readings to the HTTP API. Every Nth
// cycle it injects a smoke spike above the danger threshold so the alarm +
// realtime notification pipeline can be demonstrated without physical hardware.
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
	"time"
)

type reading struct {
	DeviceID   string  `json:"device_id"`
	DeviceType string  `json:"device_type"`
	Value      float64 `json:"value"`
	Status     string  `json:"status"`
}

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

func post(apiURL string, r reading) {
	body, _ := json.Marshal(r)
	resp, err := http.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("post %s failed: %v", r.DeviceType, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("sent %s=%.1f (status %d)", r.DeviceType, r.Value, resp.StatusCode)
}

// postScan simulates the AI camera reporting a recognized face. The server
// resolves a random enrolled student of the ongoing class and records attendance.
func postScan(scanURL string, classroomID int, status string) {
	body, _ := json.Marshal(map[string]interface{}{
		"classroom_id": classroomID,
		"device_id":    fmt.Sprintf("cam-%d", classroomID),
		"status":       status,
	})
	resp, err := http.Post(scanURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("scan failed: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("face-scan classroom %d (%s) -> %d", classroomID, status, resp.StatusCode)
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

func main() {
	apiURL := env("API_URL", "http://backend:8081/sensor")
	scanURL := env("SCAN_URL", "http://backend:8081/attendance/scan")
	room := env("ROOM_ID", "A101")
	interval := time.Duration(envInt("INTERVAL_SECONDS", 5)) * time.Second
	// Every spikeEvery cycles, push a smoke reading above SMOKE_THRESHOLD.
	spikeEvery := envInt("SPIKE_EVERY", 12)
	smokeThreshold := float64(envInt("SMOKE_THRESHOLD", 300))
	scanEvery := time.Duration(envInt("SCAN_EVERY_SECONDS", 6)) * time.Second
	maxClassroom := envInt("SCAN_MAX_CLASSROOM", 10)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	log.Printf("simulator -> %s every %s (room %s, smoke spike every %d cycles)", apiURL, interval, room, spikeEvery)

	// Background: simulate successful face scans -> realtime attendance.
	go scanLoop(scanURL, scanEvery, maxClassroom)

	// Give the API time to come up.
	time.Sleep(8 * time.Second)

	cycle := 0
	for {
		cycle++

		// Light (lux): day-ish ambient with noise.
		light := 400 + 150*math.Sin(float64(cycle)/10) + rng.Float64()*40
		post(apiURL, reading{DeviceID: room + "-light", DeviceType: "light", Value: round(light), Status: "active"})

		// Temperature (°C): comfortable room range.
		temp := 27 + 2*math.Sin(float64(cycle)/15) + rng.Float64()*0.8
		post(apiURL, reading{DeviceID: room + "-temp", DeviceType: "temperature", Value: round(temp), Status: "active"})

		// Humidity (%).
		hum := 60 + 5*math.Sin(float64(cycle)/20) + rng.Float64()*2
		post(apiURL, reading{DeviceID: room + "-humidity", DeviceType: "humidity", Value: round(hum), Status: "active"})

		// Smoke (analog MQ-2): normally low, spikes above threshold periodically.
		smoke := 80 + rng.Float64()*60
		if spikeEvery > 0 && cycle%spikeEvery == 0 {
			smoke = smokeThreshold + 50 + rng.Float64()*100
			log.Printf("🔥 injecting smoke spike: %.0f", smoke)
		}
		post(apiURL, reading{DeviceID: room + "-smoke", DeviceType: "smoke", Value: round(smoke), Status: "active"})

		time.Sleep(interval)
	}
}

func round(v float64) float64 {
	return math.Round(v*10) / 10
}
