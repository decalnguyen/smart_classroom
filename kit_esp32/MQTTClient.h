#ifndef MQTT_CLIENT_H
#define MQTT_CLIENT_H

#include <PubSubClient.h>

// Khai báo hàm khởi tạo MQTT
void initMQTT();
void connectMQTT();
void setMQTTRoomID(const String &newRoomID);
void setMQTTBroker(const String &brokerIP);
String getMQTTBroker();
void publishSensorData(const String &roomID, int lightValue, int smokeValue, float temperature, float humidity, int ledLevel, int fanLevel, bool buzzerState, const String &ipAddress);
void handleMQTTMessages();
bool isMQTTConnected();

#endif
