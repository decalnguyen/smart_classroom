#include <WiFi.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>
#include "MQTTClient.h"
#include "LedFanControl.h"
#include "BuzzerControl.h"
#include "OLEDDisplay.h"

// MQTT Broker Configuration
String MQTT_BROKER = "10.151.135.98";  // Default broker IP, có thể đổi qua web config
const int MQTT_PORT = 1883;
const char* MQTT_USER = "admin";  // Để trống nếu không dùng auth
const char* MQTT_PASS = "admin";

WiFiClient espClient;
PubSubClient client(espClient);

String roomID = "";

// lastReconnectAttempt và MQTT_RECONNECT_INTERVAL được định nghĩa trong Final_http_doidonvi.ino

// Set roomID from preferences
void setMQTTRoomID(const String &newRoomID) {
  roomID = newRoomID;
  Serial.println("[MQTT] RoomID set to: " + roomID);
}

// Set MQTT Broker IP
void setMQTTBroker(const String &brokerIP) {
  if (brokerIP.length() > 0) {
    MQTT_BROKER = brokerIP;
    client.setServer(MQTT_BROKER.c_str(), MQTT_PORT);
    Serial.println("[MQTT] Broker set to: " + MQTT_BROKER);
  }
}

// Get current MQTT Broker IP
String getMQTTBroker() {
  return MQTT_BROKER;
}

// ============================================
// MQTT Callback - Xử lý tin nhắn nhận được
// ============================================
void mqttCallback(char* topic, byte* payload, unsigned int length) {
  // Chuyển payload thành String
  String message = "";
  for (unsigned int i = 0; i < length; i++) {
    message += (char)payload[i];
  }
  
  String topicStr = String(topic);
  Serial.println("[MQTT] Received on topic: " + topicStr);
  Serial.println("[MQTT] Payload: " + message);

  // Parse JSON payload
  StaticJsonDocument<256> doc;
  DeserializationError error = deserializeJson(doc, message);
  
  if (error) {
    Serial.println("[MQTT] JSON parse error: " + String(error.c_str()));
    return;
  }

  // Xử lý các command topic
  // Format: /<RoomID>/<device>/cmd
  
  // LED Command
  if (topicStr.endsWith("/led/cmd")) {
    if (doc.containsKey("level")) {
      int level = doc["level"].as<int>();
      if (level >= 0 && level <= 3) {
        setLedLevelMQTT(level);  // Use MQTT setter to track timestamp
        ledcWrite(0, 85 * level);  // Update PWM directly
        Serial.println("[MQTT] LED set to level: " + String(level));
        displayMessage("LED Lv: " + String(level));
        
        // Publish LED value immediately
        String ledTopic = "/" + roomID + "/led/value";
        String payload = "{\"value\":" + String(level) + "}";
        client.publish(ledTopic.c_str(), (uint8_t*)payload.c_str(), payload.length(), true);
        Serial.println("[MQTT] Published LED value: " + payload);
      } else {
        Serial.println("[MQTT] LED level invalid: " + String(level));
        displayMessage("LED Lv Error!");
      }
    }
  }
  // Fan Command
  else if (topicStr.endsWith("/fan/cmd")) {
    if (doc.containsKey("level")) {
      int level = doc["level"].as<int>();
      if (level >= 0 && level <= 3) {
        setFanLevelMQTT(level);  // Use MQTT setter to track timestamp
        ledcWrite(1, 85 * level);  // Update PWM directly
        Serial.println("[MQTT] Fan set to level: " + String(level));
        displayMessage("Fan Lv: " + String(level));
        
        // Publish FAN value immediately
        String fanTopic = "/" + roomID + "/fan/value";
        String payload = "{\"value\":" + String(level) + "}";
        client.publish(fanTopic.c_str(), (uint8_t*)payload.c_str(), payload.length(), true);
        Serial.println("[MQTT] Published FAN value: " + payload);
      } else {
        Serial.println("[MQTT] Fan level invalid: " + String(level));
        displayMessage("Fan Lv Error!");
      }
    }
  }
  // Buzzer Command
  else if (topicStr.endsWith("/buzzer/cmd")) {
    Serial.println("[MQTT] Processing buzzer command...");
    
    // Handle both "state" and "level" keys
    if (doc.containsKey("level")) {
      int level = doc["level"].as<int>();
      Serial.println("[MQTT] Buzzer level: " + String(level));
      
      if (level == 1) {
        setBuzzerStateMQTT(true);
        digitalWrite(26, HIGH);
        Serial.println("[MQTT] Buzzer ON");
        displayMessage("Buzzer: ON");
        
        String buzzerTopic = "/" + roomID + "/buzzer/value";
        String payload = "{\"value\":1}";
        client.publish(buzzerTopic.c_str(), (uint8_t*)payload.c_str(), payload.length(), true);
        Serial.println("[MQTT] Published Buzzer value: " + payload);
      } else if (level == 0) {
        setBuzzerStateMQTT(false);
        digitalWrite(26, LOW);
        Serial.println("[MQTT] Buzzer OFF");
        displayMessage("Buzzer: OFF");
        
        String buzzerTopic = "/" + roomID + "/buzzer/value";
        String payload = "{\"value\":0}";
        client.publish(buzzerTopic.c_str(), (uint8_t*)payload.c_str(), payload.length(), true);
        Serial.println("[MQTT] Published Buzzer value: " + payload);
      }
    } else if (doc.containsKey("state")) {
      int state = doc["state"].as<int>();
      Serial.println("[MQTT] Buzzer state: " + String(state));
      
      if (state == 1) {
        setBuzzerStateMQTT(true);
        digitalWrite(26, HIGH);
        Serial.println("[MQTT] Buzzer ON");
        displayMessage("Buzzer: ON");
        
        String buzzerTopic = "/" + roomID + "/buzzer/value";
        String payload = "{\"value\":1}";
        client.publish(buzzerTopic.c_str(), (uint8_t*)payload.c_str(), payload.length(), true);
        Serial.println("[MQTT] Published Buzzer value: " + payload);
      } else if (state == 0) {
        setBuzzerStateMQTT(false);
        digitalWrite(26, LOW);
        Serial.println("[MQTT] Buzzer OFF");
        displayMessage("Buzzer: OFF");
        
        String buzzerTopic = "/" + roomID + "/buzzer/value";
        String payload = "{\"value\":0}";
        client.publish(buzzerTopic.c_str(), (uint8_t*)payload.c_str(), payload.length(), true);
        Serial.println("[MQTT] Published Buzzer value: " + payload);
      }
    } else {
      Serial.println("[MQTT] Buzzer payload missing 'level' or 'state' key!");
      displayMessage("Buzzer Error!");
    }
  }
}

// ============================================
// Khởi tạo MQTT
// ============================================
void initMQTT() {
  client.setServer(MQTT_BROKER.c_str(), MQTT_PORT);
  client.setCallback(mqttCallback);
}

// ============================================
// Kết nối MQTT
// ============================================
void connectMQTT() {
  if (client.connected()) {
    return;
  }


  if (WiFi.status() != WL_CONNECTED) {
    Serial.println("[MQTT] WiFi not connected!");
    return;
  }

  Serial.println("[MQTT] Attempting to connect to broker: " + MQTT_BROKER);
  
  String clientID = "ESP32_" + String(random(0xffff), HEX);
  
  bool connected = false;
  if (MQTT_USER[0] == '\0') {
    // Kết nối không cần xác thực
    connected = client.connect(clientID.c_str());
  } else {
    // Kết nối có xác thực
    connected = client.connect(clientID.c_str(), MQTT_USER, MQTT_PASS);
  }

  if (connected) {
    Serial.println("[MQTT] Connected!");
    
    // Subscribe to all command topics
    // Format: /<RoomID>/<device>/cmd
    if (roomID != "") {
      String ledCmdTopic = "/" + roomID + "/led/cmd";
      String fanCmdTopic = "/" + roomID + "/fan/cmd";
      String buzzerCmdTopic = "/" + roomID + "/buzzer/cmd";
      
      client.subscribe(ledCmdTopic.c_str());
      client.subscribe(fanCmdTopic.c_str());
      client.subscribe(buzzerCmdTopic.c_str());
      
      Serial.println("[MQTT] Subscribed to command topics for room: " + roomID);
    }
  } else {
    Serial.println("[MQTT] Connection failed, rc=" + String(client.state()));
  }
}

// ============================================
// Xử lý MQTT (gọi trong loop)
// ============================================
void handleMQTTMessages() {
  if (!client.connected()) {
    connectMQTT();
  }
  
  client.loop();
}

// ============================================
// Publish Sensor Data
// ============================================
void publishSensorData(const String &roomIDParam, int lightValue, int smokeValue, float temperature, float humidity, int ledLevel, int fanLevel, bool buzzerState, const String &ipAddress) {
  if (!client.connected()) {
    return;
  }

  roomID = roomIDParam;
  
  // Tạo JSON documents cho các sensor
  StaticJsonDocument<128> tempDoc;
  tempDoc["value"] = temperature;
  String tempPayload;
  serializeJson(tempDoc, tempPayload);
  
  StaticJsonDocument<128> humiDoc;
  humiDoc["value"] = humidity;
  String humiPayload;
  serializeJson(humiDoc, humiPayload);
  
  StaticJsonDocument<128> lightDoc;
  lightDoc["value"] = lightValue;
  String lightPayload;
  serializeJson(lightDoc, lightPayload);
  
  StaticJsonDocument<128> smokeDoc;
  smokeDoc["value"] = smokeValue;
  String smokePayload;
  serializeJson(smokeDoc, smokePayload);
  
  StaticJsonDocument<128> ledDoc;
  ledDoc["value"] = ledLevel;
  String ledPayload;
  serializeJson(ledDoc, ledPayload);
  
  StaticJsonDocument<128> fanDoc;
  fanDoc["value"] = fanLevel;
  String fanPayload;
  serializeJson(fanDoc, fanPayload);
  
  StaticJsonDocument<128> buzzerDoc;
  buzzerDoc["value"] = buzzerState ? 1 : 0;
  String buzzerPayload;
  serializeJson(buzzerDoc, buzzerPayload);
  
  StaticJsonDocument<128> ipDoc;
  ipDoc["value"] = ipAddress;
  String ipPayload;
  serializeJson(ipDoc, ipPayload);
  
  // Publish all topics
  String tempTopic = "/" + roomID + "/temp/value";
  String humiTopic = "/" + roomID + "/humi/value";
  String lightTopic = "/" + roomID + "/light/value";
  String smokeTopic = "/" + roomID + "/smoke/value";
  String ledTopic = "/" + roomID + "/led/value";
  String fanTopic = "/" + roomID + "/fan/value";
  String buzzerTopic = "/" + roomID + "/buzzer/value";
  String ipTopic = "/" + roomID + "/ip/value";
  
  client.publish(tempTopic.c_str(), (uint8_t*)tempPayload.c_str(), tempPayload.length(), true);
  client.publish(humiTopic.c_str(), (uint8_t*)humiPayload.c_str(), humiPayload.length(), true);
  client.publish(lightTopic.c_str(), (uint8_t*)lightPayload.c_str(), lightPayload.length(), true);
  client.publish(smokeTopic.c_str(), (uint8_t*)smokePayload.c_str(), smokePayload.length(), true);
  client.publish(ledTopic.c_str(), (uint8_t*)ledPayload.c_str(), ledPayload.length(), true);
  client.publish(fanTopic.c_str(), (uint8_t*)fanPayload.c_str(), fanPayload.length(), true);
  client.publish(buzzerTopic.c_str(), (uint8_t*)buzzerPayload.c_str(), buzzerPayload.length(), true);
  client.publish(ipTopic.c_str(), (uint8_t*)ipPayload.c_str(), ipPayload.length(), true);
  
  Serial.println("[MQTT] Published sensor data for room: " + roomID);
}

// ============================================
// Kiểm tra trạng thái kết nối
// ============================================
bool isMQTTConnected() {
  return client.connected();
}
