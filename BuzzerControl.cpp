#include <Arduino.h>
#include "BuzzerControl.h"

#define BUZZER_PIN 26
#define BUZZER_BUTTON_PIN 13  // Đúng chân bạn dùng

volatile bool buzzerButtonPressed = false;
bool buzzerState = false;

void IRAM_ATTR handleBuzzerButtonInterrupt() {
  buzzerButtonPressed = true;
}

void setupBuzzer() {
  pinMode(BUZZER_PIN, OUTPUT);
  digitalWrite(BUZZER_PIN, LOW);
  pinMode(BUZZER_BUTTON_PIN, INPUT_PULLUP);
  attachInterrupt(digitalPinToInterrupt(BUZZER_BUTTON_PIN), handleBuzzerButtonInterrupt, FALLING);
}

void handleBuzzerButton() {
  if (buzzerButtonPressed) {
    delay(50); // chống dội nút
    if (digitalRead(BUZZER_BUTTON_PIN) == LOW) {
      buzzerState = !buzzerState;
      if (buzzerState) {
        buzzerOn();
      } else {
        buzzerOff();
      }
    }
    buzzerButtonPressed = false;
  }
}

void buzzerOn() {
  digitalWrite(BUZZER_PIN, HIGH);
  bool buzzerState = true;
}

void buzzerOff() {
  digitalWrite(BUZZER_PIN, LOW);
  bool buzzerState = false;
}

bool getBuzzerState() {
    return buzzerState;
}

void setBuzzerState(bool state) {
    buzzerState = state;
}
