# API & WebSocket Reference — Smart Classroom

- **HTTP API base:** `http://localhost:8091` (host) → container `:8081`
- **WebSocket base:** `ws://localhost:8082`
- **Auth:** JWT (HS256). Send `Authorization: Bearer <token>` header (cookie `auth_token` also accepted).
- **Roles:** `admin`, `teacher`, `student`. Groups below: *Public*, *Auth* (any logged-in), *Staff* (admin+teacher), *Admin*.
- **Errors:** JSON `{"error": "..."}` with appropriate HTTP status (400/401/403/404/409/500).

---

## 1. Authentication

### POST /signup — *Public*
Request:
```json
{ "username": "alice", "password": "secret", "role": "student" }
```
`role` optional (default `student`; one of admin/teacher/student).
Response `200`:
```json
{ "message": "User created successfully", "account_id": "uuid", "role": "student" }
```

### POST /login — *Public*
Request: `{ "username": "admin", "password": "admin123" }`
Response `200`:
```json
{ "token": "<jwt>", "role": "admin", "account_id": "uuid", "username": "admin" }
```

### POST /logout — *Public*  → `{ "message": "Logged out successfully" }`
### GET /user — *Auth*  → `{ "account_id", "username", "role" }`

**Seeded demo accounts:** `admin/admin123`, `teacher/teacher123`, `student/student123`.

---

## 2. Sensors & devices

### POST /sensor — *Public (device ingestion: ESP32 / simulator)*
Request:
```json
{ "device_id": "A101-smoke", "device_type": "smoke", "value": 320.5, "status": "active" }
```
`device_type`: `light` | `temperature` | `humidity` | `smoke` | …
Side effects: persists reading, publishes `sensor.data` (→ `/ws/sensor`), and runs danger-threshold evaluation (smoke/temperature) → may raise an alert (→ `/ws/notifications`) + buzzer command.
Response `200`: `{ "message": "Data received" }`

### GET /sensor/:device_id?start=&end= — *Auth*
Returns historical readings (array of SensorData). `start`/`end` optional time filters.

### PUT /sensor/:device_id — *Staff* — update latest reading.

### GET /sensorinf — *Auth* — list registered devices (array of Sensor).
### POST /sensorinf — *Admin* — body `Sensor`; `409` if device_id exists.
### PUT /sensorinf/:device_id — *Admin*.
### DELETE /sensorinf/:device_id — *Admin*.

### POST /device/:device_type/:device_id/mode — *Staff* — control a device (light/fan).
Request: `{ "mode": 1 }` (int; 0=off, 1=on). `device_type`/`device_id` must match `^[A-Za-z0-9_.-]{1,128}$`.
Response `200`: `{ "message": "Command sent" | "Command queued (device offline)", "device", "mode" }`
Also publishes `command.device`.

### Electricity — `GET /electricity?id=&type=` (*Auth*), `POST/PUT/DELETE /electricity[/:id]` (*Staff*).

---

## 3. School entities (CRUD)

| Entity | List (Auth) | Create/Update/Delete |
|--------|-------------|----------------------|
| Buildings | `GET /buildings` | *Admin* `POST /buildings`, `PUT/DELETE /buildings/:id` |
| Classrooms | `GET /classrooms` | *Admin* `POST /classrooms`, `PUT/DELETE /classrooms/:id` |
| Classes | `GET /classes/:id` (ongoing class of a classroom) | *Staff* `POST /classes`, `PUT/DELETE /classes/:id` |
| Students | `GET /students?search=&limit=&offset=` | *Admin* `POST /students`, `PUT/DELETE /students/:id` |
| Teachers | `GET /teachers` | *Admin* `POST /teachers`, `PUT/DELETE /teachers/:id` |

`GET /students` supports `search` (name/MSSV), `limit` (≤200), `offset`; returns the `X-Total-Count` header.

Model shapes:
```jsonc
Building  { "building_id", "building_name", "location" }
Classroom { "classroom_id", "classroom_name", "subject", "building_id" }
Class     { "class_id", "subject", "classroom_id", "day_of_week", "start_time", "end_time" }
Student   { "student_id", "mssv", "student_name", "age", "phone", "email", "account_id" }
Teacher   { "teacher_id", "teacher_name", "subject", "account_id" }
Sensor    { "device_id", "device_name", "device_type", "location", "status", "timestamp" }
```

---

## 4. Attendance

### GET /attendance?classroom_id= — *Auth*
Returns the present/absent roster of the classroom's currently ongoing class.
Response `200`:
```json
[ { "student_id": 22520001, "mssv": "22520001", "student_name": "Nguyễn Văn An",
    "status": "present", "phone": "09...", "email": "...@student.uit.edu.vn" } ]
```
`404` if no class is ongoing in that classroom.

### GET /my/classrooms — *Auth* — classrooms in the caller's scope
admin/student → all classrooms; **teacher → only assigned classrooms** (via `ClassroomTeacher`). Returns an array of Classroom. Used by the attendance/report classroom selectors.

### GET /reports/attendance — *Staff* — attendance analytics (role-scoped) ⭐
Teacher sees only their assigned classrooms; admin sees all.
Query: `?date=YYYY-MM-DD` (per-classroom breakdown, default today), `?from=&to=` (daily trend, default last 7 days).
Response `200`:
```jsonc
{
  "scope": "admin", "is_all": true, "date": "2026-06-08", "from": "...", "to": "...",
  "totals": { "present": 262, "enrolled": 700, "absent": 438, "rate": 0.374 },
  "by_classroom": [ { "classroom_id": 1, "classroom_name": "A101", "subject": "Lập trình",
                      "present": 8, "enrolled": 70, "absent": 62, "rate": 0.114 } ],
  "by_date": [ { "date": "2026-06-08", "present": 262 } ]
}
```

### POST /attendance — *Staff* — manual marking.
Body `attendance_status` accepts `present` | `late` | `absent`. `GET /attendance?classroom_id=` is role-scoped (teacher → only assigned classrooms, else `403`) and **omits phone/email when the caller is a student** (privacy). Roster `status` is one of present/late/absent.

### GET /my/attendance — *Auth (student)* — the caller's own attendance history
```jsonc
{ "linked": true,
  "student": { "student_id": 22520001, "mssv": "22520001", "student_name": "Nguyễn Văn An" },
  "summary": { "total": 7, "present": 6, "late": 1 },
  "records": [ { "date": "2026-06-08", "detection_time": "07:35:00", "subject": "Lập trình",
                 "status": "present", "classroom_name": "A101" } ] }
```
`linked:false` if the account isn't tied to a Student row.

### GET /reports/attendance/export — *Staff* — CSV download of the per-classroom report (`?date=`), role-scoped. Returns `text/csv` (cols: Ngay, Phong, Mon, Si so, Co mat, Di muon, Vang, Ti le %). The report JSON now also includes `late` per classroom and in totals.

### Teacher ↔ classroom assignment — *Admin*
- `GET /classroom-teachers` → `[{classroom_id, classroom_name, teacher_id, teacher_name}]`
- `POST /classroom-teachers` `{classroom_id, teacher_id}` (409 if exists)
- `DELETE /classroom-teachers?classroom_id=&teacher_id=`

Note: `GET /schedules` for a **linked student** is derived from real class enrollment (subject + classroom per weekday) merged with personal entries.
Request: `{ "student_id": 22520001, "classroom_id": 1, "attendance_status": "present", "device_id": "A101-cam" }`

### PUT /attendance/:id — *Staff*. ### DELETE /attendance/:id — *Staff*.

### POST /attendance/scan — *Public (AI camera / simulator)*  ⭐ face-scan success
Simulates the edge camera reporting a recognized face.
Request:
```json
{ "classroom_id": 1, "student_id": 22520001, "device_id": "cam-1" }
```
`student_id` optional — if omitted the server picks a random enrolled student of the ongoing class not yet present. Persists `present` attendance and broadcasts an `attendance.event` (→ `/ws/attendance`).
Response `200`:
```json
{ "message": "Face recognized", "event": { ...AttendanceEvent } }
```

---

## 5. Schedule (per-account)

### GET /schedules — *Auth* — caller's weekly timetable, grouped by day:
```json
{ "Monday": [ { "time": "07:30", "title": "Lập trình", "desc": "...", "room": "A101" } ],
  "Tuesday": [], "...": [] }
```
### POST /schedules — *Auth* — `{ "title", "desc", "room", "day": "Monday", "time": "08:00" }`
### PUT /schedules/:id — *Auth* (owner). ### DELETE /schedules/:id — *Auth* (owner).

---

## 6. Notifications

### GET /notifications — *Auth* — caller's notifications + system broadcasts (`account_id="ALL"`), newest first.
```json
[ { "id": "uuid", "account_id": "ALL", "title": "alert",
    "message": "🔥 Phát hiện khói...", "is_read": false, "created_at": "..." } ]
```
`title="alert"` ⇒ danger/safety alert.
### POST /notifications?account_id= — *Staff* — create (optionally target a user).
### PUT /notifications/:id — *Auth* — `{ "is_read": true }`. ### DELETE /notifications/:id — *Auth*.

---

## 7. WebSocket channels (`ws://localhost:8082`)

Connect and receive JSON frames (server → client push). No auth handshake required for the demo; origin is restricted to the configured frontend origins.

### GET /ws/sensor — live sensor readings
```json
{ "id": 123, "device_id": "A101-temp", "device_type": "temperature",
  "value": 28.6, "status": "active", "timestamp": "2026-06-08T07:30:00+07:00" }
```

### GET /ws/notifications — live notifications / safety alerts
```json
{ "id": "uuid", "account_id": "ALL", "title": "alert",
  "message": "🌡️ Nhiệt độ vượt ngưỡng...", "is_read": false, "created_at": "..." }
```

### GET /ws/attendance — live face-scan recognitions
```json
{ "student_id": 22520061, "mssv": "22520061", "student_name": "Hồ Quang Lan",
  "classroom_id": 1, "class_id": 7, "subject": "Lập trình",
  "attendance_status": "present", "detection_time": "23:56:12",
  "date": "2026-06-07", "device_id": "cam-1" }
```

---

## 8. Message broker (RabbitMQ)

Durable **topic** exchange `main_exchange`. Producers (HTTP API) publish; the WebSocket server binds queues and fans out to the channels above.

| Routing key | Bound queue (binding) | Fanned out to |
|-------------|-----------------------|---------------|
| `sensor.data` | `sensor_data` (`sensor.*`) | `/ws/sensor` |
| `notify.data` | `notification_data` (`notify.*`) | `/ws/notifications` |
| `attendance.event` | `attendance_data` (`attendance.*`) | `/ws/attendance` |
| `command.buzzer`, `command.device` | (device command channel) | — |
