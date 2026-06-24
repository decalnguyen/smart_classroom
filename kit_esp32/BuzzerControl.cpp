#include <Arduino.h>
#include "BuzzerControl.h"

#define BUZZER_PIN 26
#define BUZZER_BUTTON_PIN 13  // Đúng chân bạn dùng

volatile bool buzzerButtonPressed = false;
bool buzzerState = false;

// Timestamp tracking (which source changed most recently)
unsigned long lastBuzzerButtonChangeTime = 0;
unsigned long lastBuzzerMQTTChangeTime = 0;

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
      lastBuzzerButtonChangeTime = millis();  // Track button change time
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
  buzzerState = true;  // Update global variable
  lastBuzzerButtonChangeTime = millis();  // Track change time
}

void buzzerOff() {
  digitalWrite(BUZZER_PIN, LOW);
  buzzerState = false;  // Update global variable
  lastBuzzerButtonChangeTime = millis();  // Track change time
}

bool getBuzzerState() {
    return buzzerState;
}

void setBuzzerState(bool state) {
    buzzerState = state;
    digitalWrite(BUZZER_PIN, state ? HIGH : LOW);  // Update pin
    // Don't update timestamp - for button
}

void setBuzzerStateMQTT(bool state) {
    buzzerState = state;
    digitalWrite(BUZZER_PIN, state ? HIGH : LOW);  // Update pin
    lastBuzzerMQTTChangeTime = millis();  // Update MQTT timestamp
}

// Priority getter - chooses based on most recent change (within 300ms window)
bool getDisplayBuzzerState() {
    unsigned long now = millis();
    unsigned long timeSinceButton = now - lastBuzzerButtonChangeTime;
    unsigned long timeSinceMQTT = now - lastBuzzerMQTTChangeTime;
    
    // If MQTT changed within 300ms, it has priority
    if (timeSinceMQTT < 300 && timeSinceMQTT < timeSinceButton) {
        return buzzerState;  // MQTT is more recent
    }
    // Otherwise use current state (button or default)
    return buzzerState;
}
