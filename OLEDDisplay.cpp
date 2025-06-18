#include <Wire.h>
#include <Adafruit_GFX.h>
#include <Adafruit_SSD1306.h>
#include "OLEDDisplay.h"
#include <Preferences.h>
#include <WebServer.h>
#include <ArduinoJson.h>
#include "BuzzerControl.h" // Thêm dòng này ở đầu file để dùng getBuzzerState()

#define SCREEN_WIDTH 128
#define SCREEN_HEIGHT 32

Adafruit_SSD1306 display(SCREEN_WIDTH, SCREEN_HEIGHT, &Wire, -1);

// Biến lưu trạng thái hiện tại
String currentRoom, currentIp;
int currentLight, currentLed, currentFan, currentSmoke; // Thêm currentSmoke
float currentTemp, currentHumi;

// Khai báo server (nếu đã có ở file khác thì dùng extern)
extern WebServer server;

void setupOLED() {
  if (!display.begin(SSD1306_SWITCHCAPVCC, 0x3C)) {
    Serial.println(F("SSD1306 allocation failed"));
    while (true);
  }
  display.clearDisplay();
  display.display();
}
// -----------------------------
// Hàm hiển thị lên màn OLED
// -----------------------------
void displayMessage(const String &msg) {
  display.clearDisplay();
  display.setTextSize(1);
  display.setTextColor(SSD1306_WHITE);
  display.setCursor(0, 10);
  display.println(msg);
  display.display();
}

void displaySensorData(const String &room, const String &ip, int light, int smoke, float temp, float humi, int led, int fan, int buzzer) {
  // Lưu trạng thái hiện tại
  currentRoom = room;
  currentIp = ip;
  currentLight = light;
  currentSmoke = smoke;
  currentTemp = temp;
  currentHumi = humi;
  currentLed = led;
  currentFan = fan;

  display.clearDisplay();
  display.setTextSize(1);
  display.setTextColor(SSD1306_WHITE);
  display.setCursor(0, 0);

  display.print(room);
  display.print(" ");
  display.println(ip);

  display.print("L:");
  display.print(light);
  display.print(" S:");
  display.print(smoke);
  display.print(" B:");
  display.println(buzzer);

  display.print("T:");
  display.print(temp);
  display.print((char)247);
  display.print("C H:");
  display.println(humi);

  display.print("LED:");
  display.print(led);
  display.print("/3 FAN:");
  display.print(fan);
  display.println("/3");

  display.display();
}

// Các hàm GET cho từng endpoint
void handleGetRoom()  { server.send(200, "application/json", "{\"room\":\"" + currentRoom + "\"}"); }
void handleGetIp()    { server.send(200, "application/json", "{\"ip\":\"" + currentIp + "\"}"); }
void handleGetLight() { server.send(200, "application/json", "{\"light\":" + String(currentLight) + "}"); }
void handleGetSmoke() { server.send(200, "application/json", "{\"smoke\":" + String(currentSmoke) + "}"); }
void handleGetTemp()  { server.send(200, "application/json", "{\"temp\":" + String(currentTemp) + "}"); }
void handleGetHumi()  { server.send(200, "application/json", "{\"humi\":" + String(currentHumi) + "}"); }
void handleGetLed()   { server.send(200, "application/json", "{\"led\":" + String(currentLed) + "}"); }
void handleGetFan()   { server.send(200, "application/json", "{\"fan\":" + String(currentFan) + "}"); }
void handleGetBuzzer() {
  int buzzer = getBuzzerState();
  server.send(200, "application/json", "{\"buzzer\":" + String(buzzer) + "}");
}
void handleGetDisplay() {
  StaticJsonDocument<256> doc;
  doc["id"] = currentRoom;
  doc["ip"] = currentIp;
  doc["light"] = currentLight;
  doc["smoke"] = currentSmoke;
  doc["buzzer"] = getBuzzerState() ? 1 : 0;
  doc["temp"] = currentTemp;
  doc["humi"] = currentHumi;
  doc["led"] = currentLed;
  doc["fan"] = currentFan;
  
  
  String response;
  serializeJson(doc, response);
  server.send(200, "application/json", response);
}


// Đăng ký endpoint (gọi trong setup)
void setupDisplayEndpoints() {
  server.on("/display", HTTP_GET, handleGetDisplay);
  server.on("/display/room", HTTP_GET, handleGetRoom);
  server.on("/display/ip", HTTP_GET, handleGetIp);
  server.on("/display/light", HTTP_GET, handleGetLight);
  server.on("/display/temp", HTTP_GET, handleGetTemp);
  server.on("/display/humi", HTTP_GET, handleGetHumi);
  server.on("/display/led", HTTP_GET, handleGetLed);
  server.on("/display/fan", HTTP_GET, handleGetFan);
  server.on("/display/buzzer", HTTP_GET, handleGetBuzzer);
  server.on("/display/smoke", HTTP_GET, handleGetSmoke);
  server.on("/warning", HTTP_POST, handlePostWarning); // Thêm dòng này
}

void handlePostWarning() {
  buzzerOn();
  setBuzzerState(true);
  server.send(200, "application/json", "{\"buzzer_status\":\"on\"}");
}