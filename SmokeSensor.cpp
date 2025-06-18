#include "SmokeSensor.h"
#include <Arduino.h>
#define SMOKE_SENSOR_PIN 33

void initSmokeSensor() {
  pinMode(SMOKE_SENSOR_PIN, INPUT);
}

int readSmokeAnalog() {
  return analogRead(SMOKE_SENSOR_PIN);
}
