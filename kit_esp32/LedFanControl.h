#ifndef LED_FAN_CONTROL_H
#define LED_FAN_CONTROL_H

void setupLedFan();
void handleLedButton();
void handleFanButton();
int getLedLevel();
int getFanLevel();
void setLedLevel(int level);
void setFanLevel(int level);
void setLedLevelMQTT(int level);  // MQTT set (update timestamp)
void setFanLevelMQTT(int level);  // MQTT set (update timestamp)
int getDisplayLedLevel();  // Choose based on most recent change
int getDisplayFanLevel();  // Choose based on most recent change
#endif
