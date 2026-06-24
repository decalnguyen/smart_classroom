#include "DeviceControl.h"
#include <ArduinoJson.h>
#include "LedFanControl.h"  // Chứa các biến ledLevel, fanLevel, dutyLevels, v.v.

// Xử lý POST /device (giữ lại cho tương thích cũ)
void handleDeviceControl() {
  if (!server.hasArg("plain")) {
    server.send(400, "application/json", "{\"error\":\"No data\"}");
    return;
  }

  String body = server.arg("plain");
  StaticJsonDocument<200> doc;
  DeserializationError error = deserializeJson(doc, body);

  if (error) {
    server.send(400, "application/json", "{\"error\":\"Invalid JSON\"}");
    return;
  }

  int led = doc["led"] | -1;
  int fan = doc["fan"] | -1;

  bool changed = false;

  if (led >= 0 && led <= 3) {
    setLedLevel(led);
    changed = true;
  }

  if (fan >= 0 && fan <= 3) {
    setFanLevel(fan);
    changed = true;
  }

  if (changed) {
    server.send(200, "application/json", "{\"status\":\"success\"}");
  } else {
    server.send(400, "application/json", "{\"error\":\"Invalid values\"}");
  }
}

// Xử lý GET /device/status (giữ lại cho tương thích cũ)
void handleDeviceStatus() {
  StaticJsonDocument<100> doc;
  doc["led"] = getLedLevel();
  doc["fan"] = getFanLevel();

  String response;
  serializeJson(doc, response);
  server.send(200, "application/json", response);
}

// Xử lý GET /device/status/led
void handleLedStatusGet() {
  StaticJsonDocument<50> doc;
  doc["led"] = getLedLevel();
  String response;
  serializeJson(doc, response);
  server.send(200, "application/json", response);
}

// Xử lý POST /device/status/led
void handleLedStatusPost() {
  if (!server.hasArg("plain")) {
    server.send(400, "application/json", "{\"error\":\"No data\"}");
    return;
  }
  String body = server.arg("plain");
  StaticJsonDocument<50> doc;
  DeserializationError error = deserializeJson(doc, body);
  if (error) {
    server.send(400, "application/json", "{\"error\":\"Invalid JSON\"}");
    return;
  }
  int led = doc["led"] | -1;
  if (led >= 0 && led <= 3) {
    setLedLevel(led);
    server.send(200, "application/json", "{\"status\":\"success\"}");
  } else {
    server.send(400, "application/json", "{\"error\":\"Invalid value\"}");
  }
}

// Xử lý GET /device/status/fan
void handleFanStatusGet() {
  StaticJsonDocument<50> doc;
  doc["fan"] = getFanLevel();
  String response;
  serializeJson(doc, response);
  server.send(200, "application/json", response);
}

// Xử lý POST /device/status/fan
void handleFanStatusPost() {
  if (!server.hasArg("plain")) {
    server.send(400, "application/json", "{\"error\":\"No data\"}");
    return;
  }
  String body = server.arg("plain");
  StaticJsonDocument<50> doc;
  DeserializationError error = deserializeJson(doc, body);
  if (error) {
    server.send(400, "application/json", "{\"error\":\"Invalid JSON\"}");
    return;
  }
  int fan = doc["fan"] | -1;
  if (fan >= 0 && fan <= 3) {
    setFanLevel(fan);
    server.send(200, "application/json", "{\"status\":\"success\"}");
  } else {
    server.send(400, "application/json", "{\"error\":\"Invalid value\"}");
  }
}

// Đăng ký các endpoint
void setupDeviceEndpoints() {
  server.on("/device/status", HTTP_POST, handleDeviceControl);
  server.on("/device/status", HTTP_GET, handleDeviceStatus);

  server.on("/device/status/led", HTTP_GET, handleLedStatusGet);
  server.on("/device/status/led", HTTP_POST, handleLedStatusPost);

  server.on("/device/status/fan", HTTP_GET, handleFanStatusGet);
  server.on("/device/status/fan", HTTP_POST, handleFanStatusPost);
}
