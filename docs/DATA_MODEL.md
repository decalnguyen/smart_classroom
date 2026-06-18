# Mô hình dữ liệu & tính nhất quán giữa các tính năng

> Tài liệu này mô tả **mô hình dữ liệu chuẩn hoá** của hệ thống Lớp học thông minh và
> cách dữ liệu **persist & chia sẻ giữa các tính năng**. Nó được xây dựng từ một
> đợt audit toàn bộ (19 điểm không nhất quán được xác minh đối kháng trên mã nguồn +
> DB thật). Phần §5 liệt kê những gì đã sửa trong đợt này; §6 là nợ kỹ thuật còn lại
> (đã ghi nhận, có lộ trình).

## 1. Quy ước chuẩn (canonical conventions)

| Khía cạnh | Quy ước chuẩn | Ghi chú |
|---|---|---|
| **device_type cảm biến** | `temp`, `humi`, `light`, `smoke` (mã ngắn, lowercase) | `canonicalType()` chuẩn hoá MỌI đường ghi: `temperature→temp`, `humidity→humi`, `mq2/gas→smoke`, `lux→light`. Một từ vựng duy nhất cho mọi transport (MQTT/HTTP) & nguồn (ESP32/simulator/fallback). |
| **device_type actuator** | `led`, `fan`, `buzzer` | Trạng thái echo trên cùng họ topic; pass-through qua `canonicalType`. |
| **device_id** | `<ClassroomName>-<device_type>` vd `A101-temp` | Khóa ngoài *ngầm* tới `classrooms.classroom_name`. Registry và telemetry dùng **cùng** id → heartbeat khớp. |
| **room (định danh phòng)** | `classroom_name` string (`A101`..`B205`) là khóa ngoài chuẩn cho mọi cột text; `classroom_id` uint là khóa nội bộ | `roomOf(device_id)` tách prefix; `device_id prefix == classroom_name`. |
| **status** | lowercase `active` / `inactive` (cả `sensors` và `sen_sor_data`) | So sánh chéo bảng dùng `lower(status)`. |
| **thời gian** | `sen_sor_data.timestamp` = timestamptz (giờ VN). Tiết học = `Class.StartMin/EndMin` (phút) | Xem nợ §6 về cột text date của attendance. |
| **status đặc biệt** | `sen_sor_data.status='demo'` = dữ liệu mô phỏng fallback | Đường freshness loại trừ `demo` để nhường thiết bị thật. |

## 2. Từ điển dữ liệu (các bảng chính)

| Bảng | Vai trò | Khóa | Ghi chú nhất quán |
|---|---|---|---|
| `sen_sor_data` | **Bảng fact time-series** (đo lường) | PK `id` bigserial, index `device_id`, index `timestamp` | device_type canonical; giờ VN; retention `SENSOR_RETENTION_DAYS` (mặc định 7). |
| `sensors` | Registry/inventory + liveness | PK `device_id` | device_type canonical (khớp telemetry); status lowercase; heartbeat cập nhật `timestamp/status`. |
| `classrooms` | Phòng học (neo định danh phòng) | PK `classroom_id`; `building_id` (logical FK) | `classroom_name` = neo cho device_id/Location/Schedule.Room. |
| `buildings` | Tòa nhà | PK `building_id` | |
| `classes` | Lớp/tiết (thời khóa biểu thật) | PK `class_id`; `classroom_id/semester_id/teacher_id` | Giờ tiết = `StartMin/EndMin`. |
| `class_students` | Join lớp↔SV | `class_id,student_id` | |
| `students` / `teachers` | SV / GV | PK uint; `account_id`→`users` | |
| `attendances` | Điểm danh | PK `id` (uuid); unique `(student_id,class_id,date)` | `excused` suy ra từ `leave_requests` (không tạo row). |
| `leave_requests` | Đơn nghỉ phép | PK uint | Duyệt → trạng thái điểm danh `excused`. |
| `notifications` | Thông báo (per-user + broadcast `ALL`) | PK uuid | Cảnh báo cháy publish `notify.data`. |
| `audit_logs` | Nhật ký kiểm toán | PK uint | Ghi mọi CRUD danh mục + điểm danh. |
| `device_credentials` | Khóa thiết bị (X-Device-Key) | PK `device_id`; `classroom_id` | `Kind` = sensor\|camera. |
| `face_embeddings` (raw-SQL, pgvector) | **Kho khuôn mặt CHUẨN** (ArcFace 512-d, đa vector/SV) | PK `id` bigserial, index `student_id` | kNN cosine `<=>`. (Model GORM `Face` đã bỏ — xem §5.) |
| `face_reviews` | Hàng đợi duyệt nhận diện độ tin cậy thấp | PK uint | Ngưỡng 0.60/0.45. |
| `classroom_teachers` | Phân công GV↔phòng | (xem nợ §6: thiếu PK) | |
| `holidays`/`makeup_sessions`/`semesters` | Học vụ | PK uint | `findOngoingClass` tôn trọng ngày nghỉ/buổi bù. |

## 3. Ma trận persistence giữa các tính năng

| Tính năng | Đọc | Ghi | Nguồn sự thật |
|---|---|---|---|
| Ingest cảm biến (MQTT/HTTP) | — | `sen_sor_data` (+heartbeat `sensors`) | `saveReading()` — đường ghi **duy nhất** |
| Giám sát thời gian thực | WS `sensor.data` | — | `useSensorStream` (chuẩn hoá `metricOf`) |
| Dashboard tổng quan phòng | `sen_sor_data` (30') + `attendances` | — | `canonicalType` bucket; cờ `fresh`/`danger` |
| Cảnh báo an toàn | `sen_sor_data` (mỗi reading) | `notifications` + lệnh `buzzer` | `EvaluateAndAlert` (ngưỡng data-driven) |
| Ngưỡng báo cháy | `sen_sor_data` (μ,σ 14 ngày) | bộ nhớ (`calThresholds`) | `threshold_calibration.go` |
| Điều khiển thiết bị | desired-state | lệnh `/<room>/<device>/cmd` | `PublishDeviceCommand` |
| Tự động theo TKB | `classes`+`classrooms` | lệnh tắt đèn/quạt | `ScheduleAutoControl` |
| Điểm danh | `classes`/`class_students`/`leave_requests` | `attendances` | `findOngoingClass`, dedup unique index |
| Báo cáo điểm danh | `attendances`+`classes` | — | `computeByClassroom` |
| Nhận diện khuôn mặt | `face_embeddings` (kNN) | `attendances` / `face_reviews` | server-side pgvector |

## 4. Phủ phòng đồng đều & freshness (giải quyết "không đồng đều")

**Vấn đề:** A101–A103 thuộc **ESP32 thật** nên bị loại khỏi simulator (`SENSOR_EXCLUDE_ROOMS`);
khi chưa cắm phần cứng, 3 phòng này trống trong khi A104+ đầy dữ liệu → không đồng đều, và
"phòng trống" không phân biệt được với "đọc giá trị 0".

**Thiết kế:** `DemoTelemetryFallback` (server) — sinh dữ liệu cho các phòng thật **chỉ khi**
không có telemetry thật trong cửa sổ freshness:
- Đếm reading **thật** (`status<>'demo'`) trong `SENSOR_FRESH_SECONDS` (mặc định 30s).
- Nếu có → **nhường** (không sinh); nếu không → sinh `temp/humi/light/smoke` (tag `status='demo'`) mỗi `DEMO_FALLBACK_SECONDS` (5s).
- ESP32 thật vừa publish → fallback tự lùi ngay (đã kiểm chứng: stream thật → 0 row demo).
- Bật/tắt: `DEMO_FALLBACK` (mặc định `on` cho demo; đặt `off` khi có phần cứng).

**Freshness ở API/UI:** `/classrooms/overview` trả cờ `fresh` per phòng → UI phân biệt "offline/không
có dữ liệu" với "0" thật. Trang Cảm biến mặc định chọn phòng **hoạt động gần nhất** (theo heartbeat),
không hardcode A101.

## 5. Đã sửa trong đợt chuẩn hoá này

1. **Từ vựng device_type** — `canonicalType()` chuẩn hoá tại 2 điểm ingest (MQTT `ingestDeviceValue`, HTTP `HandlePostSensorData`); reader `HandleClassroomsOverview` bucket theo mã ngắn → **dashboard temp/humi hết 0** (10/10 phòng có giá trị) và cờ nguy hiểm nhiệt độ kích hoạt đúng.
2. **Registry khớp telemetry** — seed `sensors` dùng mã ngắn (`A101-temp`…) + tên hiển thị thân thiện → heartbeat khớp, hết false "Inactive". Migration idempotent đổi tên row cũ.
3. **status casing** — lowercase `active/inactive` toàn bộ (seed, ingest, downgrade, ip-heartbeat) + migration `lower(status)`.
4. **Transport nhất quán** — HTTP & MQTT cùng lưu mã ngắn (chuẩn hoá tại ingest).
5. **Firmware** — lệnh `light` trước đây điều khiển **chân quạt**; nay `led/light→LED`, `fan→FAN`, `buzzer→BUZZER` (nhất quán: tên lệnh server == device firmware == device_type registry).
6. **Phủ phòng đồng đều + freshness** — `DemoTelemetryFallback` + cờ `fresh` + phòng mặc định theo hoạt động (xem §4).
7. **Dọn model chết** — bỏ `models.Face` khỏi AutoMigrate; `face_embeddings` (pgvector) là kho chuẩn (ghi rõ trong db.go). `models.Device` không còn được tham chiếu.

## 6. Nợ kỹ thuật còn lại (đã ghi nhận, chưa sửa — rủi ro thấp/không chặn bảo vệ)

| # | Vấn đề | Hướng xử lý | Vì sao hoãn |
|---|---|---|---|
| D1 | Định danh phòng 3 dạng (id/name/prefix) chưa có FK DB trên cột text | Thêm FK / join theo `classroom_name` | Refactor schema rộng, hệ thống đang chạy ổn |
| D2 | Thời gian text trong attendance/holiday/semester/leave | Chuyển sang `date`/`timestamptz` | Cần migrate dữ liệu + sửa truy vấn |
| D3 | `electricity` PK=`device_id` (không lưu được time-series) | Gộp vào `sen_sor_data` hoặc thêm `id` autoincrement | Tính năng chưa dùng (0 row) |
| D4 | `attendances.id` là `*string`; `classroom_teachers` thiếu PK | NOT NULL UUID + PK ghép `(classroom_id,teacher_id)` | Đã có unique index chặn trùng điểm danh; CT trùng hiếm |
| D5 | actuator (led/fan/buzzer) lưu lẫn vào `sen_sor_data` | Thêm cột `kind` hoặc bảng riêng | Không sai số liệu, chỉ lẫn lịch sử |
| D6 | `Subject` lưu free-text ở 4 bảng | Thay bằng `subject_id` FK | Hiển thị vẫn đúng |
| D7 | `Student.MSSV` (string) trùng `StudentID` (uint) | Chọn 1 khóa chuẩn | Đang đồng bộ |
| D8 | `Schedule` song song `Class`, có thể lệch | Suy ra từ `Class` hoặc thêm `ClassID` FK | "Lịch cá nhân" tách biệt cố ý |
| D9 | `Class/Classroom.StartTime/EndTime` time.Time chết | Bỏ cột, chỉ giữ `StartMin/EndMin` | Không ảnh hưởng logic |
| D10 | id hub credential (`hub-<room>` vs firmware `<room>-hub`) | Thống nhất `<room>-hub` | Chỉ ảnh hưởng last_seen của hub |
