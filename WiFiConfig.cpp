#include <WiFi.h>
#include <WebServer.h>
#include "GlobalPreferences.h"
#include "OLEDDisplay.h"  // Bạn cần có hàm displayMessage()
#include "DeviceControl.h"
WebServer server(80);

// ======= TRANG GIAO DIỆN =======

void handleRoot() {
  String html = "<html><body>"
                "<h2>Connect WiFi</h2>"
                "<form action='/wifi' method='POST'>"
                "WiFi SSID: <input type='text' name='ssid'><br>"
                "WiFi Password: <input type='password' name='pass'><br><br>"
                "<input type='submit' value='Connect WiFi'>"
                "</form></body></html>";
  server.send(200, "text/html", html);
}

void handleWiFi() {
  String ssid = server.arg("ssid");
  String pass = server.arg("pass");

  WiFi.begin(ssid.c_str(), pass.c_str());
  displayMessage("Connecting WiFi...");
  int timeout = 10;
  while (WiFi.status() != WL_CONNECTED && timeout-- > 0) {
    delay(1000);
  }

  if (WiFi.status() == WL_CONNECTED) {
    preferences.begin("wifi", false);
    preferences.putString("ssid", ssid);
    preferences.putString("pass", pass);
    preferences.end();

    server.send(200, "text/html", "<h1>WiFi Connected!</h1><a href='/login'>Next: Login</a>");
  } else {
    server.send(400, "text/html", "<h1>Failed to connect WiFi. Try again.</h1>");
  }
}

void handleLoginPage() {
  String html = "<html><body>"
                "<h2>Teacher Login</h2>"
                "<form action='/login' method='POST'>"
                "Teacher ID: <input type='text' name='id'><br>"
                "Password: <input type='password' name='pass'><br><br>"
                "<input type='submit' value='Login'>"
                "</form></body></html>";
  server.send(200, "text/html", html);
}

void handleLogin() {
  String id = server.arg("id");
  String pass = server.arg("pass");

  // Giả sử chỉ có 1 tài khoản hợp lệ, bạn có thể mở rộng
  if (id == "admin" && pass == "1234") {
    preferences.begin("device", false);
    preferences.putString("teacher_id", id);
    preferences.putString("teacher_pass", pass);
    preferences.end();

    server.send(200, "text/html", "<h1>Login Successful</h1><a href='/room'>Next: Set Room</a>");
  } else {
    server.send(401, "text/html", "<h1>Login Failed</h1><a href='/login'>Try again</a>");
  }
}

void handleRoomPage() {
  String html = "<html><body>"
                "<h2>Set Room Name</h2>"
                "<form action='/room' method='POST'>"
                "Room: <input type='text' name='room'><br><br>"
                "<input type='submit' value='Save Room'>"
                "</form></body></html>";
  server.send(200, "text/html", html);
}

void handleRoom() {
  String room = server.arg("room");
  if (room.length() > 0) {
    preferences.begin("device", false);
    preferences.putString("room", room);
    preferences.end();

    server.send(200, "text/html", "<h1>Setup Complete! Restarting...</h1>");
    delay(2000);
    ESP.restart();
  } else {
    server.send(400, "text/html", "<h1>Please enter room name!</h1>");
  }
}

// ======= KẾT NỐI WIFI SAU KHỞI ĐỘNG =======

bool connectToWiFi(String &ipAddress) {
  preferences.begin("wifi", false);
  String ssid = preferences.getString("ssid", "");
  String pass = preferences.getString("pass", "");
  preferences.end();

  preferences.begin("device", false);
  String room = preferences.getString("room", "");
  preferences.end();

  String apName = room != "" ? room + "_Setup" : "ESP32_Setup";

  // Bật AP không mật khẩu nếu chưa có room
  if (room == "") {
    WiFi.softAP(apName.c_str());
  } else {
    WiFi.softAP(apName.c_str(), "12345678");
  }

  Serial.println("AP Mode ON: " + apName);
  displayMessage("AP: " + apName);

  // Thiết lập các route web
  server.on("/", handleRoot);
  server.on("/wifi", HTTP_POST, handleWiFi);
  server.on("/login", HTTP_GET, handleLoginPage);
  server.on("/login", HTTP_POST, handleLogin);
  server.on("/room", HTTP_GET, handleRoomPage);
  server.on("/room", HTTP_POST, handleRoom);
  // /device /devicetype/deviceid

  setupDeviceEndpoints();//setup http control device
  setupDisplayEndpoints();// setup http hien thi

  server.begin();

  // Nếu có sẵn SSID đã lưu thì thử kết nối
  if (ssid != "") {
    WiFi.begin(ssid.c_str(), pass.c_str());
    displayMessage("Connecting WiFi...");
    int timeout = 10;
    while (WiFi.status() != WL_CONNECTED && timeout-- > 0) {
      delay(1000);
    }

    if (WiFi.status() == WL_CONNECTED) {
      ipAddress = WiFi.localIP().toString();
      displayMessage("WiFi OK: " + ipAddress);
      return true;
    }
  }

  displayMessage("AP only mode");
  return false;
}


void handleWiFiWebServer() {
  server.handleClient();
}
