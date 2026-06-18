/*
 * Smart Classroom — ESP32 sensor + actuator node (full KIT per báo cáo, Bảng 3.2).
 *
 * Peripherals: GL5516 light (ADC), DHT11 temp/humid, MQ-2 smoke (ADC), OLED SSD1306
 * (I2C status display), LED 5V + fan DC 5V (via MOSFET, GPIO/PWM), buzzer 5V (GPIO),
 * push button (manual control + fire alarm).
 *
 * Topic format (MQTT via RabbitMQ rabbitmq_mqtt, mqtt.exchange=main_exchange):
 *   PUBLISH  /<ROOM>/<device>/value     telemetry / state
 *     /A101/temp/value /A101/humi/value /A101/light/value /A101/smoke/value
 *     /A101/led/value  /A101/fan/value  /A101/buzzer/value /A101/ip/value
 *   SUBSCRIBE /<ROOM>/+/cmd             commands ({ "value": 1 } or { "level": 3 })
 *
 * Control model: LOCAL threshold logic runs on-device for instant, network-independent
 * safety (smoke/fire → buzzer) and energy use (light→LED, temp→fan); the SERVER can
 * override via desired-state cmd topics, and also auto-offs devices by timetable.
 *
 * Libraries: WiFi, PubSubClient, ArduinoJson, DHT + Adafruit Unified Sensor,
 *            Adafruit_GFX, Adafruit_SSD1306.
 */
#include <WiFi.h>
#include <Wire.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>
#include <DHT.h>
#include <Adafruit_GFX.h>
#include <Adafruit_SSD1306.h>

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
#define FAN_PIN     26   // relay / MOSFET
#define BUZZER_PIN  25
#define BUTTON_PIN  33   // manual control + fire alarm (INPUT_PULLUP)
#define OLED_W      128
#define OLED_H      32   // 0.91" SSD1306
#define OLED_ADDR   0x3C

// ---- Local thresholds (on-device safety/energy, độc lập mạng) ------------ //
// SMOKE_THR is data-driven: collected corpus has khói μ≈104, σ≈15.6, max=138;
// μ+5σ≈182 → 180 catches a smouldering fire (180–250) the old fixed 300 missed,
// while staying >30% above any normal reading (≈0 báo giả). The server applies
// the same μ+Kσ rule and auto-recalibrates (threshold_calibration.go).
const float SMOKE_THR     = 180;  // > → fire/gas danger → buzzer (μ+5σ from data)
const float LIGHT_ON_THR  = 30;   // light % < → bật đèn
const float TEMP_FAN_THR  = 30;   // °C > → bật quạt

DHT dht(DHT_PIN, DHT_TYPE);
WiFiClient net;
PubSubClient mqtt(net);
Adafruit_SSD1306 oled(OLED_W, OLED_H, &Wire, -1);

unsigned long lastPublish = 0;
const unsigned long PUBLISH_MS = 5000;
int ledState = 0, fanState = 0, buzzerState = 0;
float curTemp = 0, curHumi = 0, curLight = 0, curSmoke = 0;
bool manualOverride = false;  // server cmd or button sets explicit state

// ---- Publish helpers ----------------------------------------------------- //
void publishValue(const char* device, float value) {
  char topic[64];
  snprintf(topic, sizeof(topic), "/%s/%s/value", ROOM, device);
  StaticJsonDocument<64> doc; doc["value"] = value;
  char buf[64]; size_t n = serializeJson(doc, buf);
  mqtt.publish(topic, (const uint8_t*)buf, n, false);
}
void publishString(const char* device, const char* value) {
  char topic[64];
  snprintf(topic, sizeof(topic), "/%s/%s/value", ROOM, device);
  StaticJsonDocument<96> doc; doc["value"] = value;
  char buf[96]; size_t n = serializeJson(doc, buf);
  mqtt.publish(topic, (const uint8_t*)buf, n, false);
}
void publishIP() { publishString("ip", WiFi.localIP().toString().c_str()); }

// ---- OLED status display ------------------------------------------------- //
void updateOLED() {
  oled.clearDisplay();
  oled.setTextSize(1);
  oled.setTextColor(SSD1306_WHITE);
  oled.setCursor(0, 0);
  oled.printf("%s  %s\n", ROOM, buzzerState ? "!! ALARM" : "OK");
  oled.printf("T:%.0fC H:%.0f%% L:%.0f\n", curTemp, curHumi, curLight);
  oled.printf("Smoke:%.0f F:%d Led:%d\n", curSmoke, fanState, ledState);
  oled.display();
}

// Apply a command (server desired-state) and echo new state on /value.
// Mapping is consistent end-to-end (server cmd name == firmware device == registry type):
//   led / light -> LED_PIN (lighting)   |   fan -> FAN_PIN   |   buzzer -> BUZZER_PIN
void applyCommand(const char* device, int value) {
  manualOverride = true;
  if (strcmp(device, "buzzer") == 0) {
    buzzerState = value ? 1 : 0; digitalWrite(BUZZER_PIN, buzzerState ? HIGH : LOW);
    publishValue("buzzer", buzzerState);
  } else if (strcmp(device, "led") == 0 || strcmp(device, "light") == 0) {
    ledState = value ? 1 : 0; digitalWrite(LED_PIN, ledState ? HIGH : LOW);
    publishValue(device, ledState);  // lighting -> LED pin (was wrongly driving the fan)
  } else if (strcmp(device, "fan") == 0) {
    fanState = value; digitalWrite(FAN_PIN, value > 0 ? HIGH : LOW);
    publishValue("fan", value);
  }
}

// topic = /<ROOM>/<device>/cmd ; payload = {"value":N} or {"level":N}
void onCommand(char* topic, byte* payload, unsigned int len) {
  StaticJsonDocument<128> doc;
  if (deserializeJson(doc, payload, len)) return;
  char* p2 = strrchr(topic, '/'); if (!p2) return; *p2 = '\0';
  char* dev = strrchr(topic, '/'); if (!dev) return; dev++;
  int value = doc.containsKey("level") ? (int)doc["level"]
            : doc.containsKey("value") ? (int)doc["value"] : 0;
  applyCommand(dev, value);
}

void connectWifi() {
  WiFi.mode(WIFI_STA); WiFi.begin(WIFI_SSID, WIFI_PASS);
  while (WiFi.status() != WL_CONNECTED) { delay(500); }
}
void connectMqtt() {
  while (!mqtt.connected()) {
    if (mqtt.connect(DEVICE_ID, MQTT_USER, MQTT_PASS)) {
      char sub[32]; snprintf(sub, sizeof(sub), "/%s/+/cmd", ROOM);
      mqtt.subscribe(sub);
      publishIP();
    } else { delay(2000); }
  }
}

void setup() {
  pinMode(LED_PIN, OUTPUT); pinMode(FAN_PIN, OUTPUT); pinMode(BUZZER_PIN, OUTPUT);
  pinMode(BUTTON_PIN, INPUT_PULLUP);
  digitalWrite(LED_PIN, LOW); digitalWrite(FAN_PIN, LOW); digitalWrite(BUZZER_PIN, LOW);
  dht.begin();
  Wire.begin();
  oled.begin(SSD1306_SWITCHCAPVCC, OLED_ADDR);
  oled.clearDisplay(); oled.display();
  connectWifi();
  mqtt.setServer(MQTT_HOST, MQTT_PORT);
  mqtt.setCallback(onCommand);
  connectMqtt();
}

void loop() {
  if (WiFi.status() != WL_CONNECTED) connectWifi();
  if (!mqtt.connected()) connectMqtt();
  mqtt.loop();

  // Fire-alarm button: immediate local buzzer + publish a danger smoke reading.
  if (digitalRead(BUTTON_PIN) == LOW) {
    buzzerState = 1; digitalWrite(BUZZER_PIN, HIGH);
    publishValue("smoke", SMOKE_THR + 500);  // báo cháy thủ công
    publishValue("buzzer", 1);
    delay(200);
  }

  unsigned long now = millis();
  if (now - lastPublish >= PUBLISH_MS) {
    lastPublish = now;
    float t = dht.readTemperature();
    float h = dht.readHumidity();
    curLight = analogRead(LIGHT_PIN) * (100.0 / 4095.0);
    curSmoke = analogRead(SMOKE_PIN);
    if (!isnan(t)) curTemp = t;
    if (!isnan(h)) curHumi = h;

    // --- LOCAL threshold control (instant, network-independent) ---
    // Smoke/fire → buzzer (safety always wins, regardless of override).
    if (curSmoke > SMOKE_THR) { buzzerState = 1; digitalWrite(BUZZER_PIN, HIGH); }
    // Energy/comfort auto-control only when no manual/server override is active.
    if (!manualOverride) {
      ledState = (curLight < LIGHT_ON_THR) ? 1 : 0; digitalWrite(LED_PIN, ledState ? HIGH : LOW);
      fanState = (curTemp > TEMP_FAN_THR) ? 1 : 0;  digitalWrite(FAN_PIN, fanState ? HIGH : LOW);
    }

    // --- Publish telemetry + state ---
    if (!isnan(t)) publishValue("temp", curTemp);
    if (!isnan(h)) publishValue("humi", curHumi);
    publishValue("light", curLight);
    publishValue("smoke", curSmoke);
    publishValue("led", ledState);
    publishValue("fan", fanState);
    publishValue("buzzer", buzzerState);

    updateOLED();
  }
}
