package handlers

import "strings"

// Canonical sensor device_type vocabulary. EVERY write path normalizes to these
// SHORT codes so telemetry (sen_sor_data), the device registry (sensors), and
// every reader (classrooms overview, alarm, threshold calibration) agree on ONE
// taxonomy — regardless of transport (MQTT vs HTTP) or source (real ESP32 vs
// simulator). This removes the historical split where MQTT stored "temp"/"humi"
// while the registry + REST overview expected "temperature"/"humidity" (which
// made the dashboard temperature/humidity tiles + temp danger flag silently 0).
// See docs/DATA_MODEL.md.
//
//	sensors   : temp, humi, light, smoke
//	actuators : led, fan, buzzer   (state echoes; passed through unchanged)
//
// canonicalType maps known aliases and lower-cases; unknown tokens (led/fan/
// buzzer/ip/...) pass through trimmed + lower-cased so actuators are untouched.
func canonicalType(t string) string {
	s := strings.ToLower(strings.TrimSpace(t))
	switch {
	case strings.Contains(s, "smoke"), strings.Contains(s, "mq2"), strings.Contains(s, "gas"):
		return "smoke"
	case strings.Contains(s, "temp"):
		return "temp"
	case strings.HasPrefix(s, "hum"): // hum, humi, humidity
		return "humi"
	case strings.Contains(s, "light"), strings.Contains(s, "lux"):
		return "light"
	}
	return s
}
