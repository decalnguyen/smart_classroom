# Bộ tài liệu chuẩn bị PHẢN BIỆN — KLTN "Ứng dụng IoT, AI trong giám sát và điều khiển lớp học"

> Tài liệu này bám theo **Phiếu nhận xét của Cán bộ phản biện (CBPB)**: phần thông tin đầu phiếu + phần chấm điểm theo Chuẩn đầu ra (LO3 = 2đ, LO4 = 4đ, LO5 = 3đ, LO7 = 1đ → tổng 10đ). Mọi dẫn chứng kỹ thuật đã được **đối chiếu trực tiếp với mã nguồn** (file:dòng) và **kiểm thử trên hệ thống đang chạy**.

---

## 0. Thông tin điền đầu phiếu CBPB

| Mục | Giá trị |
|---|---|
| Số chương | **5** |
| Số hình vẽ | **37** (Hình 1.1; 2.1–2.2; 3.1–3.12; 4.1–4.22) |
| Số bảng biểu | **12** (Bảng 1.1–1.2; 2.1; 3.1–3.5; 4.1–4.4) |
| Số tài liệu tham khảo | **19** ([1]–[19]) |

---

## 1. Bảng tự đánh giá theo Chuẩn đầu ra (rubric /10đ)

| CĐR | Tiêu chí | Điểm tối đa | Mức sẵn sàng | Dẫn chứng chính |
|---|---|---|---|---|
| **LO3** | Khảo sát, phân tích tài liệu, ý tưởng rõ ràng, TLTK đầy đủ | (2đ) | **Khá** | Chương 1–2 (khảo sát giải pháp, so sánh mô hình – Bảng 2.1), 19 TLTK |
| | Tính mới, sáng tạo, tiềm năng ứng dụng | | **Mạnh** | Tích hợp 3 bài toán trên 1 nền tảng; con-người-trong-vòng-lặp; edge/server match kép |
| **LO4** | Thiết kế giải pháp rõ ràng | (4đ) | **Mạnh** | Chương 3 + ERD (Hình 3.6) + 2 sơ đồ tuần tự + [ARCHITECTURE.md](ARCHITECTURE.md) |
| | Hiện thực giải pháp, hoàn thiện demo | | **Mạnh** | 7 dịch vụ Docker chạy 7 ngày; ma trận kiểm thử chức năng (§3) PASS |
| | Kịch bản đánh giá, trình bày kết quả | | **Khá** | Chương 4: LOO + t‑SNE + ma trận nhầm lẫn + Bảng 4.4 (15 chức năng "Đạt") |
| **LO5** | Báo cáo + slides + trình bày | (3đ) | **Khá** | Báo cáo đầy đủ theo mẫu; **slides cần làm** (xem §4, §6) |
| **LO7** | Lập kế hoạch, tổ chức, quản lý KLTN | (1đ) | **Mạnh** | Kiến trúc phân lớp, roadmap P0/P1/P2, tài liệu hoá đầy đủ. *Lưu ý phiếu: KLTN gia hạn = 0đ mục này* |

---

## 2. Điểm cần nhấn khi trình bày (theo từng LO)

**LO3 — Ý tưởng & tính mới**
- Khoảng trống thực tế: các giải pháp hiện có (Philips Hue/Kasa; Azure/Rekognition) **rời rạc**, phụ thuộc đám mây, không quản lý theo lớp/thời khóa biểu/phân quyền.
- Đóng góp: **tích hợp 3 bài toán** (giám sát môi trường + an toàn cháy nổ + điểm danh AI) trên **một nền tảng tự chủ mã nguồn, chi phí thấp, không phụ thuộc đám mây**.
- Điểm sáng tạo cụ thể: lưu **nhiều embedding/SV** (gốc + augment) thay vì 1 centroid → bỏ phiếu kNN có trọng số (giống pipeline huấn luyện); **cổng tin cậy 3 mức** (auto / hàng đợi duyệt / bỏ) — con‑người‑trong‑vòng‑lặp; **mô hình desired‑state** cho lệnh thiết bị (gửi lại an toàn).

**LO4 — Thiết kế & hiện thực**
- Kiến trúc **3 lớp** (Edge ESP32/Jetson · Server Go · Web) khớp mô hình IoT 3 lớp (Hình 2.1, 3.1).
- **Tách 2 server** (HTTP API + WebSocket) qua **RabbitMQ topic exchange** → realtime không bị nghẽn bởi CRUD.
- **pgvector kNN k=5** (ngưỡng 0.60/0.45) cho nhận diện phía server; **FAISS** phía Jetson — cùng một mô hình `buffalo_l` (SCRFD+ArcFace 512‑d).
- Bảo mật nhiều lớp: JWT HS256 24h + cookie HttpOnly, **RBAC 5 cấp truy cập / 3 vai trò**, khóa thiết bị `X-Device-Key`, rate‑limit đăng nhập, **nhật ký kiểm toán đầy đủ** (gồm cả CRUD danh mục).
- Toàn bộ container hoá; 1 lệnh `docker compose up` dựng lại nhất quán.

**LO4.3 — Đánh giá**
- Module AI đánh giá bằng **Leave‑One‑Out** + trực quan hoá **t‑SNE** (Hình 4.5) + **phân bố cosine trong‑lớp/giữa‑lớp** (Hình 4.6, làm cơ sở chọn ngưỡng 0.60) + **ma trận nhầm lẫn** (Hình 4.7).
- Ma trận chức năng (Bảng 4.4): **15/15 chức năng "Đạt"** — đã kiểm chứng lại trên hệ thống đang chạy (§3 bên dưới).

**LO5/LO7 — Trình bày & quản lý**
- Báo cáo theo mẫu, mạch lạc; tài liệu kỹ thuật phụ trợ (`ARCHITECTURE/API/ENROLLMENT/JETSON_DEPLOYMENT.md`).
- Kiến trúc tách lớp + roadmap P0/P1/P2 cho thấy **nhận thức phạm vi & rủi ro** (đúng tinh thần "quản lý KLTN").

---

## 3. Kết quả kiểm thử lại kiến trúc (đối chiếu LO4.2 — demo hoàn thiện)

Đã chạy lại trên hệ thống (uptime 7 ngày, 6 dịch vụ + face‑enroll tuỳ chọn):

| Kiểm thử | Kết quả |
|---|---|
| 6 dịch vụ Docker `running` (postgres/rabbitmq `healthy`) | ✅ |
| FE `:3000`=200, BE `/login`=200, RabbitMQ mgmt `:15672`=200 | ✅ |
| RBAC: admin (toàn quyền) / teacher (`/audit`→**403**, báo cáo scoped) / student (`/reports`→**403**, `/my/attendance`→200) | ✅ |
| Không token → **401**; thiết bị không `X-Device-Key` → **401**, có key → **200** | ✅ |
| `/stats/overview` trả KPI (10 phòng · 700 SV · 12 GV · 50/50 cảm biến · tỉ lệ) | ✅ |
| `/reports/attendance` (role‑scoped + CSV cột *Co phep*), `/audit`, `/enrollment/status` | ✅ |

> Các luồng đã verify ở các vòng trước: nhận diện kNN (≥0.60 điểm danh / 0.45–0.60 chờ duyệt / <0.45 bỏ); cảnh báo khói→còi (`/<room>/buzzer/cmd`); ingest MQTT `#.value`; điều khiển actuator 0–3; enroll ảnh thật → pgvector.

---

## 4. Dàn ý slide trình bày (đáp ứng LO5 "slides theo mẫu")

> **Cần tạo file `.pptx`** — repo hiện chưa có slide (xem rủi ro §6). Dàn ý ~16 slide bám cấu trúc báo cáo:

1. **Bìa** — tên đề tài (VI/EN), 2 SV + MSSV, GVHD ThS. Phạm Minh Quân, 2026.
2. **Đặt vấn đề** — 3 hạn chế lớp học truyền thống (đèn/quạt thủ công · điểm danh giấy/điểm danh hộ · chưa tự động giám sát cháy nổ).
3. **Khoảng trống & giải pháp liên quan** — Hue/Kasa, Azure/Rekognition → thiếu nền tảng tích hợp tự chủ (LO3.1).
4. **Mục tiêu & phạm vi** — 3 nhóm chức năng + demo 1 phòng, hướng mở rộng.
5. **Kiến trúc tổng thể** — Hình 3.1 (3 lớp + luồng dữ liệu).
6. **Phần cứng IoT** — Hình 3.2 + Bảng 3.2 (ESP32 + GL5516/DHT11/MQ‑2/OLED + MOSFET/đèn/quạt/còi).
7. **Hệ thống máy chủ** — 2 server + RabbitMQ + PostgreSQL/pgvector (Hình 3.3).
8. **Luồng cảm biến & cảnh báo realtime** — Hình 3.5 + sơ đồ tuần tự cảnh báo (Hình 3.11).
9. **MQTT & điều khiển thiết bị** — topic `/room/dev/value` & `/room/dev/cmd`, mức 0–3, desired‑state.
10. **Module điểm danh AI** — quy trình SCRFD→ArcFace→pgvector kNN→cổng tin cậy (Hình 3.7) + sơ đồ tuần tự (Hình 3.10).
11. **Ghi danh & hàng đợi duyệt** — 3 đường ghi danh + review queue (con‑người‑trong‑vòng‑lặp).
12. **Bảo mật** — JWT/bcrypt/RBAC/X‑Device‑Key/rate‑limit/audit (Hình 3.9).
13. **Triển khai Docker** — Hình 3.12 + Bảng 3.5.
14. **Kết quả AI** — t‑SNE (4.5) + phân bố cosine/ngưỡng (4.6) + ma trận nhầm lẫn (4.7) + nói rõ **phạm vi đánh giá**.
15. **Kết quả giao diện & realtime** — chọn 4–5 ảnh: Tổng quan, Cảnh báo, Điểm danh realtime, Báo cáo, Duyệt nhận diện (dùng `docs/screenshots/`).
16. **Kết luận – Hạn chế – Hướng phát triển** — Bảng 4.4 (15 "Đạt") + anti‑spoofing/ML dự báo cháy/HA.

*Mẹo cho LO5:* mỗi slide ≤ 6 dòng; demo trực tiếp xen giữa slide 8–11; chuẩn bị ảnh chụp dự phòng phòng khi mất mạng.

---

## 5. Ngân hàng câu hỏi phản biện & trả lời (đã đối chiếu mã nguồn)

### LO3 — Ý tưởng, tính mới
**Q: Điểm mới của đề tài so với giải pháp có sẵn là gì?**
A: Tính **tích hợp** — gộp giám sát môi trường, an toàn cháy nổ và điểm danh AI trên **một kiến trúc thống nhất, tự chủ mã nguồn, không phụ thuộc đám mây thương mại**; quản lý theo lớp/thời khóa biểu/phân quyền (điều mà Hue/Kasa hay Azure/Rekognition không có). Về kỹ thuật: lưu nhiều embedding/SV + bỏ phiếu kNN, cổng tin cậy 3 mức có hàng đợi duyệt, và mô hình lệnh desired‑state.

**Q: Vì sao chọn ArcFace + FAISS/pgvector mà không tự huấn luyện mô hình?**
A: ArcFace (`buffalo_l`, hàm mất mát góc biên cộng tính) là SOTA, đã pretrain trên hàng triệu khuôn mặt → embedding 512‑d phân biệt cao mà không cần dữ liệu lớn; FAISS/pgvector cho **thêm người mới không cần huấn luyện lại** (chỉ thêm vector). Đây là lựa chọn kỹ thuật hợp lý cho phạm vi đồ án, tập trung vào **tích hợp hệ thống**.

### LO4 — Thiết kế & hiện thực
**Q: Ngưỡng nhận diện thực tế là bao nhiêu?**
A: `FACE_T_HIGH=0.60` (≥ → tự điểm danh), `FACE_T_LOW=0.45` (< → bỏ qua "khuôn mặt lạ"), khoảng 0.45–0.60 → **hàng đợi duyệt tay**. Cấu hình qua env, mặc định trong `internal/handlers/enrollment_handler.go:23–24`. (Ngưỡng 0.60 chọn theo phân bố cosine trong‑lớp ≈0.62 vs giữa‑lớp ≈0 — Hình 4.6.)

**Q: kNN hoạt động ra sao ở phía server?**
A: `recognizeByEmbedding()` truy vấn pgvector bằng toán tử cosine `<=>`, lấy **k=5** láng giềng gần nhất **trong phạm vi gallery của phòng**, cộng cosine theo từng SV, độ tin cậy = tổng/k; chọn SV điểm cao nhất. Giống đúng cơ chế bỏ phiếu trong notebook huấn luyện.

**Q: Chống điểm danh trùng thế nào?**
A: Ràng buộc **một bản ghi / (student_id, class_id, date)** + chỉ mục duy nhất; quét trùng sẽ cập nhật trạng thái thay vì chèn dòng mới (`attendance_scan.go`) → tỉ lệ không vượt 100%.

**Q: Thiết bị xác thực ra sao (khác người dùng)?**
A: Middleware `RequireDevice()` kiểm tra `X-Device-Key` so với `DEVICE_API_KEY` hoặc token theo từng thiết bị trong `device_credentials` (seed `camtok-*`/`hubtok-*`). Endpoint thiết bị (`/sensor`, `/attendance/scan`, `/device/heartbeat`, `/enrollment/gallery`) **không dùng JWT người dùng** → chống giả mạo dữ liệu.

**Q: MQTT tích hợp thế nào, có cần broker thứ 2 không?**
A: Không. Bật plugin `rabbitmq_mqtt` (cổng 1883) **trên chính RabbitMQ**, ánh xạ topic MQTT `/A101/temp/value` → routing key AMQP `.A101.temp.value` trên `main_exchange`. `mqtt_bridge` bind `#.value`, lưu DB + republish `sensor.data` + chạy `EvaluateAndAlert`. Lệnh đi chiều ngược: `.room.device.cmd` → `/room/device/cmd`.

**Q: Cảnh báo cháy nổ hoạt động end‑to‑end?**
A: `EvaluateAndAlert()` so ngưỡng (khói 300, nhiệt 50, cấu hình env); vượt → tạo thông báo broadcast (`account_id='ALL'`) lên `notify.data` + `triggerBuzzer()` publish `/<room>/buzzer/cmd`; WS `/ws/notifications` đẩy tức thời; **cooldown 30s/thiết bị** chống spam. (Đã verify: khói 777 → nhận lệnh còi.)

### LO4.3 — Đánh giá
**Q: Độ chính xác nhận diện đo thế nào, bao nhiêu?**
A: Báo cáo (mục 4.6) đánh giá **Leave‑One‑Out trên 2 SV nhóm tác giả** (≈10 ảnh/người, 164 vector), đạt **100% (20/20)**, độ tin cậy dự đoán đúng 0.80–0.96; kèm t‑SNE phân tách 2 cụm rõ và phân bố cosine trong/giữa lớp. **Hạn chế đã nêu (Ch 5.2): tập nhỏ, cần đánh giá lại khi mở rộng.** (⚠ Xem rủi ro #1 §6 — chuẩn bị trả lời về quy mô dữ liệu.)

**Q: CSV báo cáo gồm cột gì?**
A: `Ngay, Phong, Mon, Si so, Co mat, Di muon, Co phep, Vang, Ti le(%)` — `reports_handler.go` (HandleAttendanceReportExport). Tỉ lệ = (có mặt+đi muộn)/(sĩ số − vắng có phép).

### LO5/LO7
**Q: RBAC là thật hay chỉ trên tài liệu?**
A: Thật — `RequireRole()` đọc role từ JWT; `/my/classrooms` & `/reports/attendance` **giới hạn theo phạm vi** (teacher chỉ phòng được phân công qua `classroom_teachers`, admin toàn bộ); `/attendance` ẩn phone/email với student. Đã verify bằng 3 token (admin/teacher/student) trả đúng 200/403.

**Q: Đã có AI thật hay chỉ mô phỏng?**
A: Module AI (InsightFace+FAISS) là **thật** và đã đánh giá (Ch 4.6 + notebook `NhanDangMSSV/`). Phần tích hợp server (pgvector kNN, ghi danh, hàng đợi duyệt, gallery sync) **chạy thật**. Trên **demo**, sự kiện quét được **mô phỏng** bằng simulator; mã chạy thật trên Jetson đã có sẵn (`edge/jetson/recognize_service.py` + `JETSON_DEPLOYMENT.md`) nhưng chưa cắm phần cứng thật trong buổi demo — đúng phạm vi đã nêu.

---

## 6. Điểm yếu đã biết & cách trả lời (Risk register)

| # | Rủi ro CBPB có thể hỏi | Cách xử lý / trả lời |
|---|---|---|
| **1 — QUAN TRỌNG** | **Thư mục `NhanDangMSSV/data/` có 16 nhãn** nhưng báo cáo chỉ đánh giá **2 SV (100%)**. Hội đồng mở repo có thể hỏi. | **Trả lời thẳng:** báo cáo cố ý giới hạn eval ở 2 SV (nhóm tác giả) làm proof‑of‑concept theo phạm vi đồ án (Ch 1.5, 5.2); 14 nhãn còn lại là dữ liệu dùng khi phát triển/thử. Nhấn: ArcFace pretrain hàng triệu mặt nên khái quát tốt; t‑SNE + phân bố cosine chứng minh khả thi. **KHÔNG trình bày con số ngoài báo cáo.** *Khuyến nghị trước buổi:* hoặc dọn `data/` chỉ còn dữ liệu đúng phạm vi, hoặc chạy lại eval trên tập đầy đủ và cập nhật số liệu cho nhất quán. |
| 2 | "Còi báo cháy có chạy trên phần cứng thật?" | Logic server→lệnh MQTT đã verify; thiết bị thật subscribe `/room/buzzer/cmd`. Trong demo dùng `mosquitto_sub` minh hoạ nhận lệnh. Nêu khuyến nghị fail‑safe: ESP32 nên có **ngưỡng cắt cục bộ** độc lập mạng (đã ghi ARCHITECTURE.md). |
| 3 | "Thiếu slide theo mẫu" (LO5b) | **Việc cần làm trước buổi bảo vệ:** tạo `.pptx` theo dàn ý §4 (đã có sẵn ảnh `docs/screenshots/`). |
| 4 | "Tài liệu tham khảo/khảo sát?" | **Báo cáo có Chương 2 (cơ sở lý thuyết) + 19 TLTK** ([1]–[19], gồm ArcFace [6], FAISS [8], InsightFace [7]…). (Lưu ý: *mã nguồn* không chứa trích dẫn — điều này bình thường; phần học thuật nằm ở báo cáo.) |
| 5 | "MQTT/AI mới ở mức dev?" | Đúng phạm vi đồ án: simulator + scaffold Jetson; nêu roadmap P0 (thiết bị thật, TLS) trong ARCHITECTURE.md. |
| 6 | "Chịu tải/HA, rate‑limit, circuit breaker?" | Đã nêu là **hạn chế & hướng phát triển** (Ch 5.2/5.3). WS hiện in‑process 1 replica; P1 đề xuất Redis pub/sub. Trả lời trung thực, không phóng đại. |
| 7 | Git commit mờ ("..", "."); chưa có test/CI | Thừa nhận; nhấn rằng trọng tâm đồ án là tích hợp hệ thống chạy được (đã verify). *Tuỳ chọn:* viết lại message vài commit gần nhất cho gọn trước khi nộp. |

---

## 7. Checklist hoàn thiện trước buổi bảo vệ

- [ ] **Tạo slide `.pptx`** theo dàn ý §4 (LO5b) — ưu tiên cao nhất.
- [ ] **Thống nhất số liệu eval** (rủi ro #1): chọn 1 trong 2 — (a) trình bày đúng phạm vi 2 SV như báo cáo và dọn `NhanDangMSSV/data/` cho khớp, hoặc (b) chạy lại LOO trên tập đầy đủ và cập nhật Bảng 4.2/4.3 + Hình 4.7/4.8.
- [ ] Chuẩn bị **demo trực tiếp** theo kịch bản (đăng nhập 3 vai trò → cảm biến realtime → kích cảnh báo khói → còi → điểm danh realtime/duyệt nhận diện → báo cáo + xuất CSV → nhật ký kiểm toán).
- [ ] Bật `face-enroll` nếu muốn demo ghi danh bằng ảnh: `docker compose --profile enroll up -d face-enroll`.
- [ ] Chụp/lưu **ảnh dự phòng** mọi màn hình (đề phòng mạng).
- [ ] (Đã xử lý) ✅ Sửa ngưỡng `0.70/0.50` → `0.60/0.45` trong `ARCHITECTURE.md` & `JETSON_DEPLOYMENT.md` cho khớp mã nguồn.
- [ ] (Tuỳ chọn) Dọn message vài commit cuối.

---

*Tài liệu liên quan: [README.md](../README.md) · [ARCHITECTURE.md](ARCHITECTURE.md) · [API.md](API.md) · [ENROLLMENT.md](ENROLLMENT.md) · [JETSON_DEPLOYMENT.md](JETSON_DEPLOYMENT.md). Ảnh minh hoạ: `docs/screenshots/`.*
