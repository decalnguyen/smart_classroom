#include "LightSensor.h"
#include <Arduino.h>

#define LIGHT_SENSOR_PIN 32    // GL5516

void initLightSensor() {
  pinMode(LIGHT_SENSOR_PIN, INPUT);
}

int readLightAnalog() {
  return 4095 - analogRead(LIGHT_SENSOR_PIN);
}