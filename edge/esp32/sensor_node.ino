/*
 * Smart Classroom — ESP32 sensor + actuator node.
 *
 * Topic format (MQTT via RabbitMQ rabbitmq_mqtt plugin, mqtt.exchange=main_exchange):
 *
 *   PUBLISH  /<ROOM>/<device>/value     device telemetry / state
 *     /A101/temp/value   /A101/humi/value  /A101/light/value  /A101/smoke/value
 *     /A101/led/value    /A101/fan/value   /A101/buzzer/value /A101/ip/value
 *
 *   SUBSCRIBE /<ROOM>/+/cmd              commands from the server
 *     /A101/light/cmd  /A101/fan/cmd  /A101/buzzer/cmd  /A101/led/cmd
 *
 * Payload:
 *   value publish : { "value": 29.5 }   (numbers; smoke may be a string; ip is a string)
 *   cmd subscribe : { "value": 1 } or { "level": 3 }   (level for the fan)
 *
 * Libraries: WiFi (ESP32 core), PubSubClient, ArduinoJson, DHT + Adafruit Unified Sensor.
 */
#include <WiFi.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>
#include <DHT.h>

// ---- Config (edit per device) -------------------------------------------- //
const char* WIFI_SSID   = "classroom-wifi";
const char* WIFI_PASS   = "change-me";
const char* MQTT_HOST   = "192.168.1.10";   // RabbitMQ host (mqtt port 1883)
const uint16_t MQTT_PORT = 1883;
const char* MQTT_USER   = "admin";           // RabbitMQ user
const char* MQTT_PASS   = "admin";           // RabbitMQ pass
const char* ROOM        = "A101";            // this node's room id (use A102 / A103 for the others)
const char* DEVICE_ID   = "A101-hub";

// ---- Pins ---------------------------------------------------------------- //
#define DHT_PIN     4
#define DHT_TYPE    DHT22
#define LIGHT_PIN   34   // LDR analog
#define SMOKE_PIN   35   // MQ-2 analog
#define LED_PIN     2
#define FAN_PIN     26   // relay
#define BUZZER_PIN  25

DHT dht(DHT_PIN, DHT_TYPE);
WiFiClient net;
PubSubClient mqtt(net);

unsigned long lastPublish = 0;
const unsigned long PUBLISH_MS = 5000;
int ledState = 0, fanState = 0, buzzerState = 0;

// ---- Publish helpers ----------------------------------------------------- //
// /<ROOM>/<device>/value  with  {"value": <v>}
void publishValue(const char* device, float value) {
  char topic[64];
  snprintf(topic, sizeof(topic), "/%s/%s/value", ROOM, device);
  StaticJsonDocument<64> doc;
  doc["value"] = value;
  char buf[64];
  size_t n = serializeJson(doc, buf);
  mqtt.publish(topic, (const uint8_t*)buf, n, false);
}

void publishString(const char* device, const char* value) {
  char topic[64];
  snprintf(topic, sizeof(topic), "/%s/%s/value", ROOM, device);
  StaticJsonDocument<96> doc;
  doc["value"] = value;
  char buf[96];
  size_t n = serializeJson(doc, buf);
  mqtt.publish(topic, (const uint8_t*)buf, n, false);
}

void publishIP() {
  publishString("ip", WiFi.localIP().toString().c_str());
}

// Apply a command to an actuator and echo its new state back on /value.
void applyCommand(const char* device, int value) {
  if (strcmp(device, "buzzer") == 0) {
    buzzerState = value ? 1 : 0;
    digitalWrite(BUZZER_PIN, buzzerState ? HIGH : LOW);
    publishValue("buzzer", buzzerState);
  } else if (strcmp(device, "led") == 0) {
    ledState = value ? 1 : 0;
    digitalWrite(LED_PIN, ledState ? HIGH : LOW);
    publishValue("led", ledState);
  } else if (strcmp(device, "fan") == 0 || strcmp(device, "light") == 0) {
    // fan supports levels; light is on/off via the same relay pin here.
    fanState = value;
    digitalWrite(FAN_PIN, value > 0 ? HIGH : LOW);
    publishValue(device, value);
  }
}

// topic = /<ROOM>/<device>/cmd ; payload = {"value":N} or {"level":N}
void onCommand(char* topic, byte* payload, unsigned int len) {
  StaticJsonDocument<128> doc;
  if (deserializeJson(doc, payload, len)) return;
  // device = segment between room and "cmd"
  char* p2 = strrchr(topic, '/');           // -> "/cmd"
  if (!p2) return;
  *p2 = '\0';
  char* dev = strrchr(topic, '/');           // -> "/<device>"
  if (!dev) return;
  dev++;
  int value = doc.containsKey("level") ? (int)doc["level"]
            : doc.containsKey("value") ? (int)doc["value"] : 0;
  applyCommand(dev, value);
}

void connectWifi() {
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASS);
  while (WiFi.status() != WL_CONNECTED) { delay(500); }
}

void connectMqtt() {
  while (!mqtt.connected()) {
    if (mqtt.connect(DEVICE_ID, MQTT_USER, MQTT_PASS)) {
      char sub[32];
      snprintf(sub, sizeof(sub), "/%s/+/cmd", ROOM);   // all cmd topics for this room
      mqtt.subscribe(sub);
      publishIP();                                     // announce IP on (re)connect
    } else {
      delay(2000);
    }
  }
}

void setup() {
  pinMode(LED_PIN, OUTPUT);
  pinMode(FAN_PIN, OUTPUT);
  pinMode(BUZZER_PIN, OUTPUT);
  digitalWrite(LED_PIN, LOW);
  digitalWrite(FAN_PIN, LOW);
  digitalWrite(BUZZER_PIN, LOW);
  dht.begin();
  connectWifi();
  mqtt.setServer(MQTT_HOST, MQTT_PORT);
  mqtt.setCallback(onCommand);
  connectMqtt();
}

void loop() {
  if (WiFi.status() != WL_CONNECTED) connectWifi();
  if (!mqtt.connected()) connectMqtt();
  mqtt.loop();

  unsigned long now = millis();
  if (now - lastPublish >= PUBLISH_MS) {
    lastPublish = now;
    float t = dht.readTemperature();
    float h = dht.readHumidity();
    if (!isnan(t)) publishValue("temp", t);
    if (!isnan(h)) publishValue("humi", h);
    publishValue("light", analogRead(LIGHT_PIN) * (100.0 / 4095.0));
    publishValue("smoke", analogRead(SMOKE_PIN));
    // Periodically echo actuator state too.
    publishValue("led", ledState);
    publishValue("fan", fanState);
    publishValue("buzzer", buzzerState);
  }
}
