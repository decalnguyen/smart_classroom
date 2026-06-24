package handlers

import (
	"log"
	"math"
	"os"
	"strings"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"
)

// DemoTelemetryFallback gives DEMO-only rooms (no real hardware) non-empty coverage
// without fighting real devices. It fills the configured rooms with synthetic
// readings ONLY when no real telemetry arrived within a freshness window. The
// real-hardware rooms A101/A102/B101 are deliberately EXCLUDED here (and from the
// simulator) so their device status reflects REAL data only — offline if silent.
//   - a demo without hardware shows non-empty data for the demo rooms;
//   - the moment a real ESP32 publishes to a covered room, the fallback YIELDS it.
//
// Default OFF in production logic, but enabled via DEMO_FALLBACK=on (set in
// docker-compose for the demo). Rooms: DEMO_FALLBACK_ROOMS (default A103,A104,A105 —
// demo-only rooms, NOT the real A101/A102/B101); cadence: DEMO_FALLBACK_SECONDS
// (default 5); freshness: SENSOR_FRESH_SECONDS (default 30).
func DemoTelemetryFallback() {
	if strings.ToLower(os.Getenv("DEMO_FALLBACK")) != "on" {
		return
	}
	rooms := splitCSV(envStr("DEMO_FALLBACK_ROOMS", "A103,A104,A105"))
	if len(rooms) == 0 {
		return
	}
	every := time.Duration(envInt("DEMO_FALLBACK_SECONDS", 5)) * time.Second
	fresh := time.Duration(envInt("SENSOR_FRESH_SECONDS", 30)) * time.Second
	log.Printf("Demo telemetry fallback ON for %v (yields to real devices within %s)", rooms, fresh)
	go func() {
		for cycle := 0; ; cycle++ {
			for _, room := range rooms {
				fillRoomIfStale(room, fresh, cycle)
			}
			time.Sleep(every)
		}
	}()
}

// fillRoomIfStale publishes one synthetic sweep for a room iff it has had no
// reading within the freshness window (i.e. no real device is currently active).
func fillRoomIfStale(room string, fresh time.Duration, cycle int) {
	// Count only REAL readings (status<>'demo') in the window. Synthetic rows are
	// tagged 'demo', so the fallback keeps a full cadence when no hardware is present
	// yet yields the instant a real ESP32 publishes its first 'active' reading.
	var n int64
	db.DB.Model(&models.SenSorData{}).
		Where("device_id LIKE ? AND status <> ? AND timestamp > ?", room+"-%", "demo", nowVN().Add(-fresh)).
		Count(&n)
	if n > 0 {
		return // a real device is active for this room -> yield
	}
	off := float64(len(room) + int(room[len(room)-1])) // per-room phase, deterministic
	t := float64(cycle)
	r1 := func(v float64) float64 { return math.Round(v*10) / 10 }
	saveReading(room, "light", r1(60+20*math.Sin(t/12+off)), "demo")
	saveReading(room, "temp", r1(26.5+2*math.Sin(t/15+off)), "demo")
	saveReading(room, "humi", r1(60+8*math.Sin(t/20+off)), "demo")
	saveReading(room, "smoke", r1(100+15*math.Sin(t/9+off)), "demo") // well under the ~182 alarm
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
