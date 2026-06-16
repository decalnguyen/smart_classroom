# Đăng ký & Nhận diện khuôn mặt — quy trình hoàn chỉnh

Tài liệu này mô tả **toàn bộ vòng đời** dữ liệu khuôn mặt sinh viên: từ lúc *đăng
ký* (enroll) đến lúc *nhận diện* (recognize) để điểm danh.

## Nguyên tắc thiết kế: tách Đăng ký ↔ Nhận diện

| | Nhận diện (recognize) | Đăng ký (enroll) |
|---|---|---|
| Tần suất | Liên tục, mọi khung hình | 1 lần / sinh viên |
| Cần GPU? | Có → **chạy ở Jetson + camera** | Không → CPU đủ |
| Đầu ra | Điểm danh realtime | Embedding 512-d tham chiếu (pgvector) |

Cả hai dùng **chung một model** `InsightFace buffalo_l` (SCRFD + ArcFace 512-d) để
embedding so khớp được với nhau. Khớp bằng **kNN k=5, vote theo cosine, ngưỡng
0.60** — đúng như notebook `NhanDangMSSV/` đã train & kiểm thử.

```
                         ┌──────────────── ĐĂNG KÝ (enroll) ────────────────┐
   Ảnh SV ──┬─ Web upload/webcam ─▶ face-enroll (CPU) ─┐
            ├─ CLI enroll_student.py ───────────────────┤─▶ embedding(s) 512-d
            └─ Bulk embeddings.pkl/id_map.json ──────────┘        │
                                                                  ▼
                                              POST /enrollment/face  →  pgvector
                                                       (nhiều vector / SV)
                                                                  │  GET /enrollment/gallery
                         ┌──────────── NHẬN DIỆN (recognize) ─────┼────────────┐
   Camera ─▶ Jetson (SCRFD+ArcFace, GPU) ─▶ kNN vote 0.60 ────────┘
            └─▶ POST /attendance/scan {embedding}  ─▶ server kNN vote ─▶ điểm danh
                                                     ├ 0.45–0.60 → hàng đợi duyệt
                                                     └ <0.45     → unknown
```

## Lưu trữ: nhiều embedding / sinh viên

Bảng `face_embeddings (id, student_id, mssv, student_name, source, embedding vector(512))`
giữ **nhiều** vector cho mỗi SV (ảnh gốc + augmented) — tái tạo đúng gallery FAISS.
Khi nhận diện, server lấy **k=5 vector gần nhất** trong phạm vi lớp, cộng cosine
theo từng SV, lấy SV điểm cao nhất; `confidence = tổng_cosine_SV_thắng / k`.

## 3 cách đăng ký (đều dùng cùng model)

### Cách 1 — Import hàng loạt từ gallery đã train (nhanh nhất)
Bạn đã có `embeddings.pkl` + `id_map.json` (MSSV) sau khi train. Import thẳng:

```bash
cd edge/jetson
BACKEND_URL=http://localhost:8091 ADMIN_JWT=<jwt_admin> \
python3 enroll_from_gallery.py \
  --embeddings ../../NhanDangMSSV/models/embeddings.pkl \
  --id-map     ../../NhanDangMSSV/models/id_map.json
```
> MSSV trong `id_map.json` phải khớp cột `students.mssv` trong DB.

### Cách 2 — Web (trang “Đăng ký khuôn mặt”, admin)
Bật dịch vụ trích xuất (chỉ khi cần — không chạy mặc định):

```bash
docker compose --profile enroll up -d face-enroll   # lần đầu tải model ~300MB
```
Vào **Đăng ký khuôn mặt** → tìm SV → **Đăng ký** → chụp webcam hoặc tải ảnh → Lưu.
Backend gửi ảnh sang `face-enroll /embed`, nhận embedding (gốc + augmented), lưu pgvector.

### Cách 3 — CLI cho 1 sinh viên (máy có model)
```bash
cd edge/jetson
BACKEND_URL=http://localhost:8091 ADMIN_JWT=<jwt> \
python3 enroll_student.py --mssv 22520001 --images ./photos/22520001
```

## Lấy JWT admin
```bash
curl -s -X POST http://localhost:8091/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])"
```

## Nhận diện trên Jetson
Jetson đồng bộ gallery của lớp rồi khớp tại chỗ, hoặc gửi embedding cho server:

```bash
# edge match (FAISS local, k=5, ngưỡng 0.60)
EDGE_MATCH=true CLASSROOM_ID=1 python3 recognize_service.py
# server match (gửi embedding lên /attendance/scan, server tự khớp)
EDGE_MATCH=false CLASSROOM_ID=1 python3 recognize_service.py
```
Chi tiết triển khai model lên Jetson: [JETSON_DEPLOYMENT.md](JETSON_DEPLOYMENT.md).

## Ngưỡng (đồng bộ giữa server & Jetson)
| Env | Mặc định | Ý nghĩa |
|-----|----------|--------|
| `FACE_T_HIGH` | 0.60 | ≥ → điểm danh ngay |
| `FACE_T_LOW`  | 0.45 | trong [0.45, 0.60) → hàng đợi duyệt; < 0.45 → unknown |
| `FACE_KNN`    | 5    | số láng giềng cho vote |

## Quản lý
- **Trang “Duyệt nhận diện”** (admin/GV): xác nhận/từ chối các lượt độ tin cậy thấp.
- **Trang “Đăng ký khuôn mặt”** (admin): xem ai đã đăng ký, số mẫu, đăng ký lại, xoá.
- API: `GET /enrollment/status`, `DELETE /enrollment/face/:student_id`.
