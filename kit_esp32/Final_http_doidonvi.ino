#include "DHTSensor.h"
#include "LightSensor.h"
#include "SmokeSensor.h"
#include "LedFanControl.h"
#include "BuzzerControl.h"
#include "OLEDDisplay.h"
#include "WiFiConfig.h"
#include "GlobalPreferences.h"
#include "MQTTClient.h"
Preferences preferences;
String roomName = "";

// MQTT Publish Timer
unsigned long lastMQTTPublish = 0;
const unsigned long MQTT_PUBLISH_INTERVAL = 5000; // Publish mỗi 5 giây
unsigned long lastReconnectAttempt = 0;
const unsigned long MQTT_RECONNECT_INTERVAL = 30000; // Thử reconnect mỗi 30 giây

// -----------------------------
// Setup
// -----------------------------
void setup() {
  Serial.begin(115200);
  setupOLED();          //setup man hinh oled
  setupLedFan();
  initDHT();           //set up dht11
  initLightSensor();   //set up cam bien anh sang
  initSmokeSensor();   //set up mq2
  setupBuzzer();       //set up Buzzer

  String ip;
  connectToWiFi(ip);  // tự hiển thị lên màn hình

  preferences.begin("device", true);
  roomName = preferences.getString("room", "NoRoom");
  preferences.end();

  // Load MQTT broker IP from preferences (nếu đã lưu)
  preferences.begin("mqtt", true);
  String savedBroker = preferences.getString("broker", "");
  preferences.end();
  if (savedBroker.length() > 0) {
    setMQTTBroker(savedBroker);
  }

  // Initialize MQTT with roomID
  initMQTT();
  setMQTTRoomID(roomName);  // Set roomID before connecting
  connectMQTT();
}

void loop() {
  // Xử lý nhanh trước
  handleWiFiWebServer();
  handleLedButton();      // Xử lý nút bấm
  handleFanButton();
  handleBuzzerButton();
  
  // Xử lý MQTT mà không block (chỉ xử lý message incoming)
  handleMQTTMessages();
  
  // Thử reconnect MQTT mỗi 30 giây nếu mất kết nối
  if (!isMQTTConnected() && millis() - lastReconnectAttempt > MQTT_RECONNECT_INTERVAL) {
    connectMQTT();
    lastReconnectAttempt = millis();
  }

  // Đo cảm biến
  int lightValue = readLightAnalog();
  int smokeValue = readSmokeAnalog();
  float humidity = readHumidity();
  float temperature = readTemperature();
  int buzzerDisplay = getBuzzerState() ? 1 : 0;

  // Tự động bật còi nếu khí thoát
  int buzzerState = (smokeValue > 2000) ? 1 : 0;
  if (buzzerState) {
    buzzerOn();
  }

  // Hiển thị dữ liệu trên màn hình
  displaySensorData(roomName, WiFi.localIP().toString(), lightValue, smokeValue, temperature, humidity, getDisplayLedLevel(), getDisplayFanLevel(), getDisplayBuzzerState() ? 1 : 0);
  
  // Publish MQTT data mỗi 5 giây - CHỈ khi đã kết nối (không block button)
  if (isMQTTConnected() && millis() - lastMQTTPublish > MQTT_PUBLISH_INTERVAL) {
    publishSensorData(roomName, lightValue, smokeValue, temperature, humidity, getDisplayLedLevel(), getDisplayFanLevel(), getDisplayBuzzerState(), WiFi.localIP().toString());
    lastMQTTPublish = millis();
  }
  
  // Delay nhỏ (10ms) để cho hệ thống xử lý WiFi, giảm từ 100ms
  // Nếu vẫn bị delay, có thể xóa delay này hoàn toàn
  delay(100);
}
