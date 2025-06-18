#include "LedFanControl.h"
#include <Arduino.h>

// LED
const int ledPin = 18;
const int ledButtonPin = 27;
const int ledPwmChannel = 0;

// FAN
const int fanPin = 19;
const int fanButtonPin = 14;
const int fanPwmChannel = 1;

const int ledPwmFreq = 5000;
const int fanPwmFreq = 5000;
const int ledPwmResolution = 8;
const int fanPwmResolution = 8;

const int dutyLevels[] = {0, 85, 170, 255};
const int numLevels = sizeof(dutyLevels) / sizeof(dutyLevels[0]);

int ledLevel = 0;
int fanLevel = 0;
volatile bool ledButtonPressed = false;
volatile bool fanButtonPressed = false;
//nut ngat cho den
void IRAM_ATTR handleLedButtonInterrupt() {
  ledButtonPressed = true;
}
//nut ngat cho quat
void IRAM_ATTR handleFanButtonInterrupt() {
  fanButtonPressed = true;
}

void setupLedFan() {
  pinMode(ledButtonPin, INPUT_PULLUP);
  pinMode(fanButtonPin, INPUT_PULLUP);

  ledcSetup(ledPwmChannel, ledPwmFreq, ledPwmResolution);
  ledcAttachPin(ledPin, ledPwmChannel);

  ledcSetup(fanPwmChannel, fanPwmFreq, fanPwmResolution);
  ledcAttachPin(fanPin, fanPwmChannel);

  attachInterrupt(digitalPinToInterrupt(ledButtonPin), handleLedButtonInterrupt, FALLING);
  attachInterrupt(digitalPinToInterrupt(fanButtonPin), handleFanButtonInterrupt, FALLING);
}

void handleLedButton() {
  if (ledButtonPressed) {
    delay(50);
    if (digitalRead(ledButtonPin) == LOW) {
      ledLevel = (ledLevel + 1) % numLevels;
      ledcWrite(ledPwmChannel, dutyLevels[ledLevel]);
    }
    ledButtonPressed = false;
  }
}

void handleFanButton() {
  if (fanButtonPressed) {
    delay(50);
    if (digitalRead(fanButtonPin) == LOW) {
      fanLevel = (fanLevel + 1) % numLevels;
      ledcWrite(fanPwmChannel, dutyLevels[fanLevel]);
    }
    fanButtonPressed = false;
  }
}

int getLedLevel() {
  return ledLevel;
}

int getFanLevel() {
  return fanLevel;
}

void setLedLevel(int level) {
  ledLevel = level;
  ledcWrite(ledPwmChannel, dutyLevels[ledLevel]);
}

void setFanLevel(int level) {
  fanLevel = level;
  ledcWrite(fanPwmChannel, dutyLevels[fanLevel]);
}