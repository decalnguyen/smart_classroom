# Kiến trúc tổng thể — Smart Classroom (IoT + AI)

Tài liệu này mô tả: (1) luồng hiện tại (mô phỏng), (2) kiến trúc mục tiêu khi nhúng **Jetson Nano** (nhận diện khuôn mặt) và **ESP32** (cảm biến/chấp hành), (3) flow nhận diện trên Jetson, (4) phương án giao tiếp, (5) những phần còn thiếu cho production.

---

## 1. Luồng hiện tại (as-is, đang mô phỏng)

```
                         (simulator đóng vai ESP32 + Camera)
 ESP32(sim) ──HTTP POST /sensor {device_id,device_type,value}──┐
                                                               ▼
 Camera(sim) ─HTTP POST /attendance/scan {classroom_id}──┐   HTTP API (Go :8081)
                                                         ▼   ├─ lưu PostgreSQL
                                                         │   ├─ EvaluateAndAlert (ngưỡng khói/nhiệt)
                                                         │   └─ publish RabbitMQ (topic exchange main_exchange)
                                                         │            keys: sensor.data · attendance.event · notify.data · command.*
                                                         ▼
                                              RabbitMQ topic exchange
                                                 sensor_data ← sensor.*
                                                 attendance_data ← attendance.*
                                                 notification_data ← notify.*
                                                         │ AMQP consume
                                                         ▼
                                              WebSocket server (Go :8082)
                                                 /ws/sensor · /ws/attendance · /ws/notifications
                                                         │ WS push
                                                         ▼
                                              Frontend (React :3000)  ── REST + WS
```

**Điểm mấu chốt phải thay khi lên thật:**
- `POST /attendance/scan` hiện **bịa** danh tính (chọn ngẫu nhiên HS chưa điểm danh) → Jetson phải gửi **student_id/MSSV + confidence** thật.
- `command.buzzer`/`command.device` được publish nhưng **không có ai tiêu thụ** → cần kênh đẩy lệnh xuống thiết bị.
- Endpoint thiết bị (`/sensor`, `/attendance/scan`) **công khai, không xác thực** → phải có định danh thiết bị.
- Chưa có **endpoint ghi danh khuôn mặt** (`Face.FaceEmbedding []byte` rỗng, chưa dùng pgvector).
- `POST /device/{type}/{id}/mode` gọi HTTP *vào* thiết bị → fail sau NAT/Wi-Fi lớp học.

---

## 2. Kiến trúc tổng thể mục tiêu (Jetson + ESP32 production)

```
        EDGE (mỗi phòng học)                 BROKER / SERVER                          CLIENT
┌──────────────────────────────┐
│ ESP32  (cảm biến + chấp hành) │  MQTT/TLS :8883
│  pub classroom/{room}/sensor/{type}   QoS0 môi trường · QoS1 khói (retained last-value)
│  sub classroom/{room}/cmd/#           QoS1 (đèn/quạt/còi, retained desired-state)
│  pub classroom/{room}/cmd/{act}/ack   QoS1
│  LWT classroom/{room}/status (retained) ─ phát hiện offline tức thời
│  SNTP đồng bộ giờ                              │
└──────────────────────────────┘               ▼
                                       ┌───────────────────────────────┐      ┌────────────────────────┐
┌──────────────────────────────┐      │ RabbitMQ                       │ AMQP │ HTTP API (Go :8081)     │
│ Jetson Nano (camera + AI)     │      │  + rabbitmq_mqtt plugin        │◀────▶│  - /attendance/scan     │
│  capture→detect→align→embed   │ HTTPS│   :1883 / :8883 (mặt MQTT)     │      │  - /enrollment/*        │
│  →match(local gallery)→vote   │─────▶│  topic exchange main_exchange  │      │  - ingest + cmd publish │
│  →POST /attendance/scan       │ REST │  (AMQP = xương sống nội bộ)    │      │  - EvaluateAndAlert     │
│  {mssv,confidence,event_id}   │      │  keys: classroom.*.sensor.*    │      │  - device auth (mTLS/API key)
│  video Ở LẠI EDGE             │      │        classroom.*.cmd.*       │      └───────────┬────────────┘
│  buffer offline (SQLite)      │      │        attendance.* notify.*   │                  │ REST
│  sub cmd/camera (start/stop)  │      └───────────────┬────────────────┘      ┌───────────▼────────────┐
└──────────────────────────────┘                      │ AMQP consume          │ React FE (:3000)        │
                                              ┌────────▼────────┐  WS :8082    │  REST + WebSocket       │
        PostgreSQL + pgvector ◀── backend     │ WS fan-out      │──────────────▶  (SSE = phương án dự phòng)
        (telemetry, attendance,               │ /ws/sensor      │              └────────────────────────┘
         face embeddings, audit)              │ /ws/attendance  │
        TimescaleDB (time-series)             │ /ws/notifications
                                              └─────────────────┘
```

**Ba lớp:** Edge (ESP32 + Jetson, suy luận tại biên) · Server (HTTP API + WS + RabbitMQ + DB) · Client (React). **RabbitMQ là xương sống AMQP nội bộ; MQTT là "mặt tiền" cho thiết bị** qua plugin `rabbitmq_mqtt` (không phải dựng broker thứ hai).

---

## 3. Flow nhận diện khuôn mặt trên Jetson Nano

### 3.1 Pipeline suy luận (chạy tại biên — edge inference)
```
Camera (CSI/USB/RTSP, GStreamer, 720p @ ~5–10 FPS sampled)
   │  nvarguscamerasrc ! nvvidconv ! appsink   (decode/scale trên ISP/NVDEC, không tốn CPU)
   ▼
Detect khuôn mặt   → MediaPipe/BlazeFace hoặc SCRFD-500MF (ONNX→TensorRT FP16)
   │  gate: face ≥ 80×80px, conf ≥ 0.7
   ▼
Align 5 điểm → warp về template ArcFace 112×112 (cv2.warpAffine)  ← đòn bẩy chính xác lớn nhất
   ▼
Embed → ArcFace 512-d (TensorRT FP16 engine build on-device bằng trtexec) → L2-normalize
   ▼
Liveness/anti-spoofing (TRT gate)  ← chặn ảnh in/màn hình
   ▼
Match cục bộ → cosine vs gallery phòng (NumPy G@q, top-1)
   │  sim ≥ T_high(0.60) → present/late
   │  T_low ≤ sim < T_high → hàng đợi duyệt tay (review queue)
   │  sim < T_low → unknown (bỏ)
   ▼
Debounce → vote N/5 frame liên tiếp + cooldown 5 phút/HS
   ▼
POST /attendance/scan {mssv, classroom_id, confidence, event_id, ts}  (idempotent)
   │  (offline → ghi SQLite/JSONL, gửi lại khi có mạng)
   ▼
Server: tìm tiết đang diễn ra → chính sách đi muộn → dedup (student,class,date) → ghi DB
        → publish attendance.event → WS /ws/attendance → realtime trên web
```
**Vì sao suy luận tại biên:** không truyền video (băng thông + quyền riêng tư), mở rộng tuyến tính theo số phòng, độ trễ thấp. **Server vẫn nắm nghiệp vụ** (tiết đang diễn ra, đi muộn, dedup) — giữ nguyên `attendance_scan.go`.

### 3.2 Pipeline ghi danh (enrollment) — phần còn thiếu
```
Admin/GV: chụp ≥3 ảnh chân dung/HS → (Jetson hoặc server) detect+align+embed ArcFace 512-d
   → POST /enrollment/face {student_id, embeddings[]}  → lưu cột pgvector vector(512)
   → Jetson đồng bộ gallery: GET /enrollment/gallery?classroom_id=  (chỉ HS học phòng đó)
```
- **Đổi model `Face`:** `FaceEmbedding []byte` → cột pgvector `vector(512)`; sửa `Face.StudentID` (string) cho khớp `Student.StudentID` (uint); thêm index ANN (ivfflat/hnsw) để match được cả ở server.
- Lưu **embedding, không lưu ảnh** (quyền riêng tư); thêm trường **consent + hạn lưu trữ**.

### 3.3 Thay đổi API contract cần bổ sung
| Endpoint | Mục đích |
|---|---|
| `POST /attendance/scan` (mở rộng) | thêm `mssv/student_id`, `confidence`, `event_id` (idempotency), `ts` |
| `POST /enrollment/face` (admin/GV) | nạp embedding mẫu của HS vào pgvector |
| `GET /enrollment/gallery?classroom_id=` | Jetson tải gallery phòng để match cục bộ |
| `POST /device/heartbeat` | Jetson/ESP32 báo sống + model_version + trạng thái |
| `GET /review-queue` + confirm/reject | duyệt tay nhận diện confidence thấp |
| Device-auth middleware | gắn cho mọi route thiết bị (token/mTLS) |

---

## 4. Phương án giao tiếp

| Liên kết | Lựa chọn | Khuyến nghị |
|---|---|---|
| **ESP32 → server** (telemetry) | HTTP(hiện tại) · **MQTT** · WS · CoAP | **MQTT**: session bền, QoS theo mức độ, retained, LWT phát hiện offline, header 2 byte |
| **server → ESP32** (lệnh đèn/quạt/còi) | HTTP-vào-thiết-bị(hiện tại, hỏng sau NAT) · **MQTT cmd** | **MQTT cmd topics**: thiết bị *subscribe* (kết nối ra ngoài → hết lỗi NAT), QoS1, ack, idempotent |
| **Jetson → server** (kết quả điểm danh) | **REST**(hiện tại) · MQTT · gRPC | **REST** + `event_id` idempotency; video ở lại edge (RTSP chỉ khi cần ghi hình tập trung) |
| **server ↔ FE** | **REST + WebSocket**(hiện tại) · SSE | **Giữ REST+WS** (SSE chỉ là phương án dự phòng) |

**Thiết kế topic MQTT (phân cấp, theo phòng, tách chiều):**
```
classroom/{room}/sensor/{type}       # uplink: light|temperature|humidity|smoke   (QoS0; smoke=QoS1)
classroom/{room}/cmd/{actuator}      # downlink: light|fan|buzzer  (QoS1, retained = desired-state)
classroom/{room}/cmd/{actuator}/ack  # device → server xác nhận (cmd_id)
classroom/{room}/status              # online/offline (retained, gắn LWT)
```
- **QoS theo mức độ:** môi trường QoS0 (mất 1 mẫu vô hại) · **khói QoS1** (an toàn) · lệnh QoS1 (thiết kế idempotent, tránh QoS2 trên ESP32).
- **Retained:** last-value cảm biến + desired-state actuator → dashboard/thiết bị mới kết nối có ngay trạng thái.
- **LWT:** broker tự phát `status=offline` khi mất kết nối → **thay cho job polling** `CheckSensorStatus`.
- **Lệnh = desired-state, không phải delta** ("quạt = ON" thay vì "đảo trạng thái") + `cmd_id` + `expires_at` → replay an toàn, còi không kêu trễ sau khi đã an toàn.
- **RabbitMQ vẫn là backbone:** bật plugin `rabbitmq_mqtt` (MQTT :1883/8883), `classroom/A101/sensor/smoke` ↔ AMQP key `classroom.A101.sensor.smoke`; toàn bộ consumer/WS giữ nguyên AMQP.
- **NTP/SNTP** trên thiết bị (ESP32 không có RTC) — gửi kèm `ts` của thiết bị, server vẫn ghi giờ nhận.
- **Định dạng:** giữ **JSON** end-to-end; chỉ cân nhắc **CBOR/MessagePack** ở chặng ESP32↔broker nếu fleet lớn/RAM hạn chế (giải mã ở ingest, nội bộ vẫn JSON).

---

## 4b. Ngưỡng báo cháy theo dữ liệu (data-driven, tự hiệu chỉnh)

Ngưỡng còi **không** là hằng số tùy chọn. Nó được suy ra từ **phân bố của chính dữ liệu cảm biến đã thu thập**, theo quy tắc phát hiện bất thường **T = μ + K·σ** (`internal/handlers/threshold_calibration.go`):

| Cảm biến | n thu thập | μ | σ | max quan sát | K | **T = μ+Kσ** (kẹp) | Ngưỡng cũ |
|---|---|---|---|---|---|---|---|
| Khói | ~5,1·10⁵ | 104 | 15,6 | 138 | 5 | **182** (∈[150,300]) | 300 |
| Nhiệt | ~5,1·10⁵ | 27,6 | 1,5 | 30,7 | 8 | **39,7** (∈[35,50]) | 50 |

- **Vì sao tốt hơn ngưỡng cố định:** khói bình thường *chưa từng* vượt 138; ngưỡng cũ 300 ⇒ một đám cháy âm ỉ ở 180–250 (rõ ràng bất thường) **bị bỏ sót**. T=182 cao hơn max quan sát >30% nên **báo giả ≈ 0** (5σ ≈ 1/3,5 triệu mẫu) mà vẫn bắt sớm. *(Đã kiểm chứng: khói=205 → còi kêu + thông báo "ngưỡng 182"; khói=175 → không báo.)*
- **Kẹp [floor, ceiling]:** *floor* tránh báo trên dao động vặt ở phòng quá yên; *ceiling* là mức nguy hiểm tuyệt đối **luôn** báo, nên baseline trôi hay sensor nhiễu không thể đẩy điểm cắt lên cao mất an toàn.
- **Chống "nhiễm" baseline:** loại các mẫu ≥ ceiling khi tính μ,σ → một sự cố cháy quá khứ không làm tăng ngưỡng.
- **Tự hiệu chỉnh:** chạy lại mỗi `THRESHOLD_CAL_SECONDS` (mặc định 1h) trên cửa sổ `THRESHOLD_CAL_WINDOW_DAYS` (14 ngày). Tham số: `SMOKE_SIGMA_K`/`TEMP_SIGMA_K`, `SMOKE_FLOOR`/`CEILING`, `TEMP_FLOOR`/`CEILING`; tắt bằng `THRESHOLD_AUTOCAL=off`. Đặt `SMOKE_THRESHOLD`/`TEMP_THRESHOLD` = **ghi đè thủ công** (ưu tiên cao nhất). ESP32 dùng cùng giá trị (`SMOKE_THR=180`) cho fail-safe cục bộ.
- **Khớp hướng phát triển (Ch5.3):** thay cơ chế ngưỡng cố định bằng ngưỡng học từ dữ liệu — bước đệm cho dự báo cháy bằng ML.

---

## 5. Những gì còn thiếu cho một hệ thống production

### P0 — Chặn production / an toàn / toàn vẹn dữ liệu
- **Xác thực thiết bị (ĐÃ LÀM phần lớn):** `/sensor`, `/attendance/scan`, `/device/heartbeat`, `/enrollment/gallery` nay đều sau `RequireDevice` (`X-Device-Key`: token **per-device** trong `device_credentials` có cờ `active`/revoke, hoặc master key cấu hình — không còn default mất an toàn). Thêm **chống phát lại**: scan/heartbeat phải mang `ts` tươi (cửa sổ lệch) + `event_id` idempotent (`internal/handlers/replay.go`). **Còn lại:** nâng lên **mTLS per-device** + ACL theo phòng (token hiện chưa ràng buộc `classroom_id` của thiết bị với `classroom_id` trong payload).
- **TLS + secrets:** `JWT_SECRET` & `DEVICE_API_KEY` nay đọc từ env/.env, **fail-fast ở production** nếu thiếu/yếu/dùng default (`utils/jwt.go`, `middleware`). **Còn lại:** bật TLS toàn data-plane (HTTPS/WSS + MQTT `:8883`), DB `sslmode`, AMQP creds mạnh, đưa secrets ra vault.
- **Messaging không mất mát:** consumer đang `auto-ack=true` (crash giữa chừng = mất tin) → **manual ack + dead-letter queue**; thêm **unique index `(student_id,class_id,date)`** + `event_id` để điểm danh exactly-once (hiện chỉ chống trùng ở tầng ứng dụng, có race).
- **Schema & time-series:** thay **GORM AutoMigrate** bằng migration có version (golang-migrate/goose); **giới hạn bảng `sen_sor_data`** (đang ghi vô hạn) bằng TimescaleDB/partition + retention/downsampling.
- **AI thật:** thay nhận diện ngẫu nhiên bằng service suy luận thật trả `student_id + confidence`; **anti-spoofing/liveness**; **consent + hạn lưu** dữ liệu sinh trắc (GDPR-like) + đường xóa.
- **Còi báo cháy fail-safe:** `triggerBuzzer` chỉ publish + log, **không ai tiêu thụ** → cần đường actuation đảm bảo + ack (đời người, không để best-effort).
- **Health/readiness:** chưa có `/healthz`/`/readyz` (Postgres ping + AMQP) → `restart: always` không phân biệt "process sống" với "DB chết".

### P1 — Vận hành đáng tin
- **WS đa replica:** client lưu trong map in-process → không scale được 2 bản WS; cần Redis pub/sub hoặc mỗi replica bind queue riêng + sticky session.
- **Background worker singleton:** `SensorChecker`/`AutoAbsentChecker` chạy trong API → 2 replica = chốt vắng trùng; cần leader-election/locked job.
- **Bật cảnh báo thiết bị offline** (đang opt-in & tắt mặc định) — cảm biến khói chết phải báo người trực.
- **Retry/backoff + buffer offline** ở edge; **publisher confirms + reconnect** AMQP (hiện publish lỗi là drop âm thầm).
- **Confidence threshold + hàng đợi duyệt tay**; **model/index versioning** (FAISS/embeddings đang commit nhị phân không version).
- **Quan sát:** Prometheus metrics (tỉ lệ ingest, số lần báo động, publish fail), log có cấu trúc + correlation id (đang log cả PII ra stdout), tracing.
- **Backup + PITR** PostgreSQL (hiện chỉ có Docker volume); **danh sách sơ tán/realtime occupancy** từ check-in; **audit** mở rộng cho login + lệnh thiết bị + báo động.
- **CI/CD** (chưa có pipeline, chưa có test Go), **rate limiting** (login đang mở brute-force) + **circuit breaker/timeout** cho `http.Post` tới thiết bị (đang block goroutine).

### P2 — Trưởng thành / quy mô
- **OTA firmware** ESP32 + cập nhật model/edge runtime Jetson (ký số, rollout theo giai đoạn, rollback).
- **HA:** RabbitMQ cluster/quorum queue, Postgres replica, LB hỗ trợ WS, bỏ pin cổng host.
- **Drift monitoring** nhận diện; **IaC** (Terraform/Helm) + tách cấu hình theo môi trường; image pin digest, chạy non-root, base tối giản (đang `golang:latest`/`ubuntu`, root).
- **Tách seed dev** khỏi `main()` (đang seed `admin/admin123` + data giả mỗi lần boot).

---

## 6. Lộ trình đề xuất (ưu tiên cho đồ án + production)
1. **Bật MQTT** (plugin `rabbitmq_mqtt`) + chuyển ESP32 telemetry/command sang MQTT (QoS, LWT, ack).
2. **Định danh thiết bị** (token/mTLS) cho route thiết bị + alarm fail-safe + consumer còi thật.
3. **pgvector + enrollment API** + service nhận diện thật trên Jetson (confidence + liveness + review queue).
4. **Manual ack + DLQ + unique index** điểm danh; **migration có version**; **TimescaleDB + retention**.
5. **Health/metrics/tracing + backup + cảnh báo offline**; tách WS fan-out & worker để chạy HA.
6. **CI/CD + rate limit + circuit breaker + image hardening + OTA**.

> Sơ đồ và quyết định ở trên bám sát code hiện tại; xem `docs/API.md` cho hợp đồng endpoint hiện hành.
