#ifndef DEVICE_CONTROL_H
#define DEVICE_CONTROL_H

#include <Arduino.h>
#include <WebServer.h>
extern WebServer server;

// Khai báo các hàm điều khiển thiết bị
void setupDeviceEndpoints();
void handleDeviceControl();
void handleDeviceStatus();
void handleLedStatusGet();
void handleLedStatusPost();
void handleFanStatusGet();
void handleFanStatusPost();

#endif
