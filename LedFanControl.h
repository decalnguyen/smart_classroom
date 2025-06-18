#ifndef LED_FAN_CONTROL_H
#define LED_FAN_CONTROL_H

void setupLedFan();
void handleLedButton();
void handleFanButton();
int getLedLevel();
int getFanLevel();
void setLedLevel(int level);
void setFanLevel(int level);
#endif
