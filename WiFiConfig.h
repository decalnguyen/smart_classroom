#ifndef WIFI_SETUP_H
#define WIFI_SETUP_H

#include <WiFi.h>
#include <WebServer.h>
#include <Preferences.h>

// Khai báo các hàm từ file .cpp sẽ dùng
void startWiFiSetup();
void handleWiFiWebServer();
bool connectToWiFi(String &ipAddress);

#endif
