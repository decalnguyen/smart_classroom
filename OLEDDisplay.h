#ifndef OLED_DISPLAY_H
#define OLED_DISPLAY_H

#include <Arduino.h>      // Thêm dòng này để nhận diện String, v.v.
#include <WebServer.h>    // Thêm dòng này để nhận diện WebServer

void setupOLED();
void displayMessage(const String &msg);
void displaySensorData(const String &room, const String &ip, int light, int smoke, float temp, float humi, int led, int fan, int buzzer);

// Khai báo các hàm endpoint để tránh lỗi khi include
void handleGetRoom();
void handleGetIp();
void handleGetLight();
void handleGetTemp();
void handleGetHumi();
void handleGetLed();
void handleGetFan();
void handleGetBuzzer(); // Thêm dòng này
void handlePostWarning();
void setupDisplayEndpoints();

#endif
