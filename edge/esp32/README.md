# ESP32 sensor + actuator node

Reads DHT22 (temperature/humidity), an LDR (light), and an MQ-x gas sensor, and
publishes them over MQTT every 5 s. Subscribes to actuator commands (buzzer,
fan/relay, light) and ACKs each one.

## Wiring (default pins in `sensor_node.ino`)

| Sensor / actuator | ESP32 pin |
|-------------------|-----------|
| DHT22 data        | GPIO 4    |
| LDR (analog)      | GPIO 34   |
| MQ-x gas (analog) | GPIO 35   |
| Buzzer            | GPIO 25   |
| Relay (fan/light) | GPIO 26   |

## Build

Arduino IDE → Boards Manager → **esp32** by Espressif. Install libraries:
`PubSubClient`, `ArduinoJson`, `DHT sensor library` (+ `Adafruit Unified Sensor`).

Edit the config block at the top (`WIFI_*`, `MQTT_*`, `ROOM`, `DEVICE_ID`),
flash, open the serial monitor at 115200.

## Topics

```
PUBLISH   /<ROOM>/temp/value     {"value": 29.5}
PUBLISH   /<ROOM>/humi/value     {"value": 62}
PUBLISH   /<ROOM>/light/value    {"value": 78}
PUBLISH   /<ROOM>/smoke/value    {"value": 2000}     (may also be a string "2000")
PUBLISH   /<ROOM>/led/value      {"value": 1}        (current state)
PUBLISH   /<ROOM>/fan/value      {"value": 3}
PUBLISH   /<ROOM>/buzzer/value   {"value": 0}
PUBLISH   /<ROOM>/ip/value       {"value": "192.168.1.50"}

SUBSCRIBE /<ROOM>/+/cmd          {"value": 1}  or  {"level": 3}   (level for the fan)
  e.g. /A101/light/cmd  /A101/fan/cmd  /A101/buzzer/cmd  /A101/led/cmd
```

The leading `/` is kept: `rabbitmq_mqtt` maps `/A101/temp/value` → routing key
`.A101.temp.value` on `main_exchange`; the backend binds `#.value` to ingest every
room/device, and publishes commands to `.<room>.<device>.cmd` (→ `/<room>/<device>/cmd`).

## Broker credentials

MQTT user/pass = **admin / admin** (`RABBITMQ_DEFAULT_USER/PASS`). `loopback_users = none`
lets it connect remotely. Change for production.

## Quick test without hardware

```bash
# publish a temperature reading
mosquitto_pub -h <host> -p 1883 -u admin -P admin -t /A101/temp/value -m '{"value":29.5}'
# trigger the smoke alarm (>300) -> server publishes /A101/buzzer/cmd
mosquitto_pub -h <host> -p 1883 -u admin -P admin -t /A101/smoke/value -m '{"value":777}'
# watch the command come back
mosquitto_sub -h <host> -p 1883 -u admin -P admin -t '/A101/+/cmd' -v
```
