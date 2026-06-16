# Edge devices

Reference firmware/services for the two edge node types in the Smart Classroom.

```
edge/
├── jetson/                      # AI camera node (face recognition)
│   ├── recognize_service.py     # capture → SCRFD → ArcFace(512-d) → match/report
│   ├── enroll_from_gallery.py   # push trained embeddings.pkl/id_map.json to backend
│   ├── convert_to_trt.sh        # ONNX → TensorRT FP16 engines (run on the Nano)
│   ├── requirements-jetson.txt
│   ├── config.example.env
│   └── smart-classroom-edge.service   # systemd unit
└── esp32/                       # sensor + actuator node
    ├── sensor_node.ino          # DHT/LDR/MQ publish + buzzer/relay command sub
    └── README.md
```

## How they connect to the backend

| Node    | Protocol | Talks to                          | Auth                     |
|---------|----------|-----------------------------------|--------------------------|
| Jetson  | HTTPS/HTTP | `POST /attendance/scan`, `GET /enrollment/gallery`, `POST /device/heartbeat` | `X-Device-Key` header |
| ESP32   | MQTT (1883) | `classroom/<room>/sensor/*` (pub), `classroom/<room>/cmd/+` (sub) | MQTT user/pass |

MQTT topics map onto the backend topic exchange via the `rabbitmq_mqtt` plugin
(`mqtt.exchange=main_exchange`, `/`→`.`), so the Go `mqtt_bridge` consumer
(`classroom.*.sensor.*`) ingests readings and `PublishDeviceCommand()` drives the
`cmd` topics. See [docs/ARCHITECTURE.md](../docs/ARCHITECTURE.md).

## Deploying your AI model to the Jetson

The full step-by-step (model conversion → TensorRT → edge service → enrollment →
live attendance) is in **[docs/JETSON_DEPLOYMENT.md](../docs/JETSON_DEPLOYMENT.md)**.
