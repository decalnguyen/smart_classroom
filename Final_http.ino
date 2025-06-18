#include <HTTPClient.h>

#include "DHTSensor.h"
#include "LightSensor.h"
#include "SmokeSensor.h"
#include "LedFanControl.h"
#include "BuzzerControl.h"
#include "OLEDDisplay.h"
#include "WiFiConfig.h"
#include "GlobalPreferences.h"
Preferences preferences;
String roomName = "";

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
}

// -----------------------------
// Gửi dữ liệu sensor
// -----------------------------
void sendSensorData() {
  if (WiFi.status() == WL_CONNECTED) {
    HTTPClient http;
    String serverUrl = "http://192.168.32.44:8081/sensor";

    http.begin(serverUrl);
    http.addHeader("Content-Type", "application/json");

    String json = "{";
    json += "\"device_id\":\"esp32_001\",";
    json += "\"temperature\":25.6,";
    json += "\"humidity\":60.3";
    json += "}";

    int httpResponseCode = http.POST(json);

    if (httpResponseCode > 0) {
      String response = http.getString();
      Serial.println("Response code: " + String(httpResponseCode));
      Serial.println("Response: " + response);
      displayMessage("Data sent OK");
    } else {
      Serial.println("POST failed, error: " + http.errorToString(httpResponseCode));
      displayMessage("Send Error!");
    }

    http.end();
  } else {
    Serial.println("WiFi not connected!");
    displayMessage("WiFi Lost!");
  }
}

//unsigned long lastSend = 0;


void loop() {
  handleWiFiWebServer();
  handleLedButton();
  handleFanButton();
  handleBuzzerButton();

  int lightValue = readLightAnalog();
  int smokeValue = readSmokeAnalog();
  float humidity = readHumidity();
  float temperature = readTemperature();
  int buzzerDisplay = getBuzzerState() ? 1 : 0;

  int buzzerState = (smokeValue > 2000) ? 1 : 0; // 1: còi bật, 0: còi tắt
  if (buzzerState) {
    buzzerOn();
  }

  displaySensorData(roomName, WiFi.localIP().toString(), lightValue, smokeValue, temperature, humidity, getLedLevel(), getFanLevel(), buzzerDisplay);
  delay(100);
  /*
  if (WiFi.status() == WL_CONNECTED && millis() - lastSend > 10000) {
    sendSensorData();
    lastSend = millis();
  }
  */
}
