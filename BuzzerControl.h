#ifndef BUZZER_CONTROL_H
#define BUZZER_CONTROL_H

void setupBuzzer();
void handleBuzzerButton();
void buzzerOn();
void buzzerOff();
bool getBuzzerState(); // Thêm dòng này
void setBuzzerState(bool state);
#endif
