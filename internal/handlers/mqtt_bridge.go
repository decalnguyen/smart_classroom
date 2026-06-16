package handlers

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"
	"smart_classroom/internal/rabbitmq"
)

// StartMQTTBridge consumes the device telemetry topics that the rabbitmq_mqtt
// plugin maps onto main_exchange. Devices publish:
//
//	/<room>/<device>/value      e.g. /A01/temp/value  -> routing key .A01.temp.value
//
// (a leading '/' in MQTT becomes a leading '.' in AMQP). The binding `#.value`
// catches every room/device, so new rooms work with no extra config. The server
// sends commands on /<room>/<device>/cmd (see PublishDeviceCommand).
func StartMQTTBridge() {
	rabbitmq.ConsumeKeyed("mqtt_device_ingest", "#.value", ingestDeviceValue)
}

// ingestDeviceValue handles one published value from a device.
func ingestDeviceValue(routingKey string, body []byte) {
	// .A01.temp.value (or A01.temp.value) -> [A01, temp, value]
	var parts []string
	for _, s := range strings.Split(routingKey, ".") {
		if s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) < 3 || parts[len(parts)-1] != "value" {
		return
	}
	room, device := parts[0], parts[1]

	var p struct {
		Value  interface{} `json:"value"`
		Status string      `json:"status"`
	}
	_ = json.Unmarshal(body, &p)

	// The IP topic carries a string address, not a numeric reading.
	if device == "ip" {
		ip := strings.TrimSpace(fmtValue(p.Value))
		log.Printf("[device-ip] room=%s ip=%s", room, ip)
		// Treat as a heartbeat: mark the room's devices alive.
		db.DB.Model(&models.Sensor{}).Where("device_id LIKE ?", room+"-%").Update("status", "Active")
		return
	}

	val, ok := toFloat(p.Value)
	if !ok {
		return
	}
	status := p.Status
	if status == "" {
		status = "active"
	}
	data := models.SenSorData{
		DeviceID: room + "-" + device, DeviceType: device, Value: val, Status: status, Timestamp: nowVN(),
	}
	if err := db.DB.Create(&data).Error; err != nil {
		log.Printf("MQTT ingest save error: %v", err)
		return
	}
	db.DB.Model(&models.Sensor{}).Where("device_id = ?", data.DeviceID).
		Updates(map[string]interface{}{"timestamp": data.Timestamp, "status": "Active"})
	rabbitmq.Publish("sensor.data", data) // -> WS /ws/sensor
	EvaluateAndAlert(data)                // safety thresholds (smoke/temp) -> may issue buzzer cmd
}

// toFloat accepts a JSON number or a numeric string (devices may send "2000").
func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func fmtValue(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// PublishDeviceCommand sends a command to an actuator: routing key .<room>.<device>.cmd
// maps to the MQTT topic /<room>/<device>/cmd the device subscribes to. Payload
// carries both `value` and `level` (and `action`) so simple on/off and level
// controls both work: e.g. fan -> {"value":3,"level":3,"action":"on"}.
func PublishDeviceCommand(room, device, action string, value int, reason string) {
	payload := map[string]interface{}{
		"value":  value,
		"level":  value,
		"action": action,
	}
	rabbitmq.Publish("."+room+"."+device+".cmd", payload)
	if reason != "" {
		log.Printf("cmd -> /%s/%s/cmd value=%d (%s)", room, device, value, reason)
	}
}

// roomOf extracts the room from a device_id like "A01-smoke" -> "A01".
func roomOf(deviceID string) string {
	if i := strings.LastIndex(deviceID, "-"); i > 0 {
		return deviceID[:i]
	}
	return deviceID
}
