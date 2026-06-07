# Hệ thống Lớp học Thông minh (IoT + giám sát thời gian thực)

Hệ thống tích hợp IoT cho lớp học: thu thập dữ liệu cảm biến (ánh sáng, nhiệt độ, độ ẩm,
khói), điều khiển thiết bị (đèn/quạt), **phát hiện ngưỡng nguy hiểm → còi báo động + thông
báo realtime tới mọi người dùng**, điểm danh, lịch học, và quản trị — với phân quyền
admin / giáo viên / học sinh.

> Module nhận diện khuôn mặt (AI) **không nằm trong phạm vi** bản này — nhưng *kết quả*
> quét khuôn mặt được mô phỏng và chảy realtime lên web (họ tên, MSSV, thời gian, trạng thái).

📑 **Tài liệu đầy đủ API + WebSocket:** [docs/API.md](docs/API.md)

**Dữ liệu mẫu** được seed sẵn khi khởi động lần đầu: 2 tòa nhà, 10 phòng học, 12 giáo viên,
**700 học sinh** (70/lớp, có MSSV/email), thời khóa biểu mỗi ngày, ghi danh, 50 cảm biến.

## Kiến trúc

```
ESP32 / Simulator ──POST /sensor──> HTTP API (Go, :8081)
                                       │  lưu PostgreSQL + đánh giá ngưỡng
                                       ▼
                                   RabbitMQ (topic exchange)
                                   sensor.*  │  notify.*
                                       ▼
                                  WebSocket Server (Go, :8082)
                                       │  /ws/sensor   /ws/notifications
                                       ▼
                                  Frontend (React + Vite + MUI, :3000)
```

- **HTTP API** (`server/http-api`): xác thực JWT + bcrypt, RBAC theo nhóm route, CRUD
  (lớp học, người dùng, thiết bị, lịch, điểm danh), ingest cảm biến, điều khiển thiết bị.
- **WebSocket** (`server/ws`): consume `sensor_data` (`sensor.*`) và `notification_data`
  (`notify.*`) từ RabbitMQ, fan-out tới client realtime.
- **Cảnh báo an toàn** (`internal/handlers/alarm.go`): mỗi reading được so với
  `SMOKE_THRESHOLD` / `TEMP_THRESHOLD`; vượt ngưỡng → lưu + phát thông báo broadcast
  (`account_id="ALL"`) + lệnh còi, có cooldown 30s/thiết bị.
- **Simulator** (`server/simulator`): giả lập KIT ESP32, định kỳ bơm dữ liệu và thỉnh
  thoảng vượt ngưỡng khói để demo còi báo động.
- **PostgreSQL** (GORM AutoMigrate), **RabbitMQ**, tất cả đóng gói Docker Compose.

## Chạy nhanh

Yêu cầu: **Docker** + Docker Compose (không cần cài Go/Node).

```bash
docker compose up -d --build
```

| Thành phần      | URL                                  |
|-----------------|--------------------------------------|
| Giao diện web   | http://localhost:3000                |
| HTTP API        | http://localhost:8091  (→ :8081)     |
| WebSocket       | ws://localhost:8082                  |
| RabbitMQ UI     | http://localhost:15672 (guest/guest) |
| PostgreSQL      | localhost:5432                       |

> Cổng host của API là **8091** (cổng 8081 trên máy này đang bị ứng dụng khác chiếm).
> Đổi qua biến `BACKEND_HOST_PORT` trong `.env` nếu muốn.

### Tài khoản demo (tạo sẵn khi khởi động lần đầu)

| Tài khoản | Mật khẩu     | Vai trò   |
|-----------|--------------|-----------|
| admin     | `admin123`   | Quản trị  |
| teacher   | `teacher123` | Giáo viên |
| student   | `student123` | Học sinh  |

### Phân quyền (RBAC)

- **admin**: toàn quyền — quản lý người dùng, tòa nhà, phòng học, học sinh, giáo viên, thiết bị.
- **teacher**: điểm danh, quản lý lớp, điều khiển thiết bị, xem dashboard.
- **student**: xem dashboard, lịch học cá nhân, điểm danh, thông báo của mình.

## Cấu hình (biến môi trường)

Sao chép `.env.example` → `.env` để chỉnh. Mặc định dev đã có sẵn trong `docker-compose.yml`.

| Biến | Mặc định | Ý nghĩa |
|------|----------|---------|
| `JWT_SECRET` | `dev_insecure_secret_change_me` | Khóa ký JWT (**đổi khi production**) |
| `POSTGRES_USER/PASSWORD/DB` | nhattoan / test123 / sensordata | Thông tin DB |
| `SMOKE_THRESHOLD` | 300 | Ngưỡng khói báo động |
| `TEMP_THRESHOLD` | 50 | Ngưỡng nhiệt độ báo động (°C) |
| `FRONTEND_ORIGIN` | http://localhost:3000 | Origin cho CORS |
| `SIM_INTERVAL` / `SIM_SPIKE_EVERY` | 5 / 12 | Chu kỳ & tần suất bơm khói vượt ngưỡng của simulator |

Frontend đọc `VITE_API_BASE_URL` / `VITE_WS_BASE_URL` (xem `auth-frontend/.env`).

## Phát triển frontend

```bash
cd auth-frontend
npm install
npm run dev      # http://localhost:3000 (Vite)
```

## Cấu trúc thư mục

```
server/http-api      HTTP API (Gin)            server/ws         WebSocket server
server/simulator     Giả lập cảm biến          internal/handlers Handlers + alarm + seed
internal/db          Kết nối + migrate         internal/rabbitmq Topic exchange + consumer
internal/middleware  RBAC                      internal/models   Models GORM
internal/utils       JWT                       Database/         Image Postgres
auth-frontend/       React + Vite + MUI        docker-compose.yml
```

## Demo còi báo động

Simulator tự bơm khói vượt ngưỡng định kỳ; hoặc kích hoạt thủ công:

```bash
curl -X POST http://localhost:8091/sensor \
  -H 'Content-Type: application/json' \
  -d '{"device_id":"A101-smoke","device_type":"smoke","value":520,"status":"active"}'
```

→ Banner đỏ + âm báo động xuất hiện realtime trên mọi phiên đang đăng nhập.

## Demo điểm danh bằng khuôn mặt (mô phỏng)

Simulator tự động gọi `/attendance/scan` định kỳ (mô phỏng camera AI nhận diện thành công).
Kích hoạt thủ công một lần quét:

```bash
curl -X POST http://localhost:8091/attendance/scan \
  -H 'Content-Type: application/json' \
  -d '{"classroom_id":1,"device_id":"cam-1"}'
```

→ Server chọn một học sinh đang ghi danh, ghi nhận "có mặt", và phát realtime tới
trang **Điểm danh** (mục "Nhận diện khuôn mặt (thời gian thực)") qua `ws://localhost:8082/ws/attendance`.
