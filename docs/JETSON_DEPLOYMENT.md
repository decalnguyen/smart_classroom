# Quy trình nhúng model AI nhận diện lên Jetson Nano

Hướng dẫn đưa **model nhận diện khuôn mặt đã huấn luyện** (`NhanDangMSSV/`) lên
**Jetson Nano** và tích hợp với backend Smart Classroom để điểm danh thời gian thực.

Model hiện tại = **InsightFace `buffalo_l`** (SCRFD phát hiện khuôn mặt + ArcFace
`w600k_r50` sinh embedding **512 chiều**) + **FAISS `IndexFlatIP`** (cosine) với
gallery đã train: `embeddings.pkl`, `faiss.index`, `id_map.json` (index → MSSV).

```
┌──────────── Jetson Nano (edge) ───────────┐      ┌──────── Backend (server) ────────┐
│ Camera → SCRFD → align → ArcFace(512-d)   │      │  POST /attendance/scan           │
│        → L2 normalize                     │      │   ├─ match pgvector (cosine)     │
│        → [A] gửi embedding lên server  ───┼─────▶│   ├─ sim≥0.60  → điểm danh        │
│          [B] match FAISS tại chỗ → sid    │      │   ├─ 0.45..0.60 → hàng đợi duyệt  │
│  GET /enrollment/gallery (đồng bộ gallery)│◀─────┤   └─ <0.45     → unknown          │
└───────────────────────────────────────────┘      │  GET /enrollment/gallery         │
                                                    │  POST /enrollment/face (enroll)  │
                                                    └──────────────────────────────────┘
```

Có **2 chế độ tích hợp** (chọn bằng `EDGE_MATCH`):

- **[A] Server match (mặc định, khuyến nghị cho đồ án)** — Jetson chỉ gửi
  embedding 512-d lên `POST /attendance/scan`, server so khớp bằng `pgvector` và
  áp ngưỡng tin cậy. Gallery là "nguồn sự thật" duy nhất ở server.
- **[B] Edge match** — Jetson tự so khớp với FAISS gallery (đồng bộ từ
  `GET /enrollment/gallery`), chỉ gửi `student_id` + độ tin cậy. Độ trễ thấp,
  chạy được khi mất mạng tạm thời, nhưng phải giữ gallery ở edge.

---

## 0. Chuẩn bị artifact model

Trên máy đã train (`NhanDangMSSV/`) bạn đã có:

```
NhanDangMSSV/models/
├── embeddings.pkl     # mảng embedding 512-d (hàng i ↔ id_map["i"])
├── faiss.index        # FAISS IndexFlatIP đã build
└── id_map.json        # { "<row>": "<MSSV>" }
```

Model `buffalo_l` (file `.onnx`) **không cần copy thủ công** — `insightface` sẽ
tự tải về `~/.insightface/models/buffalo_l/` ở lần chạy đầu tiên trên Nano
(`det_10g.onnx` = SCRFD, `w600k_r50.onnx` = ArcFace).

---

## 1. Chuẩn bị Jetson Nano

```bash
# JetPack 4.6.x (Nano) đã có CUDA/cuDNN/TensorRT + OpenCV (CUDA) + numpy.
sudo nvpmodel -m 0          # MAXN: mở khóa toàn bộ công suất
sudo jetson_clocks          # khóa xung nhịp ở mức cao

# Nano 4GB RAM → cần thêm swap để build engine TensorRT / load model.
sudo fallocate -l 4G /swapfile && sudo chmod 600 /swapfile
sudo mkswap /swapfile && sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### onnxruntime-gpu cho Jetson (KHÔNG dùng wheel PyPI)

Cài wheel NVIDIA build sẵn khớp JetPack (có sẵn `TensorrtExecutionProvider` +
`CUDAExecutionProvider`). Ví dụ JetPack 4.6 / Python 3.6:

```bash
# Tra cứu wheel đúng tại: https://elinux.org/Jetson_Zoo#ONNX_Runtime
wget <onnxruntime_gpu-*-cp36-*aarch64.whl>
pip3 install onnxruntime_gpu-*aarch64.whl
python3 -c "import onnxruntime as ort; print(ort.get_available_providers())"
# kỳ vọng có: ['TensorrtExecutionProvider', 'CUDAExecutionProvider', 'CPUExecutionProvider']
```

### Phần còn lại

```bash
sudo mkdir -p /opt/smart-classroom-edge
sudo cp edge/jetson/* /opt/smart-classroom-edge/
cd /opt/smart-classroom-edge
pip3 install -r requirements-jetson.txt   # insightface, requests, python-dotenv
cp config.example.env .env                # sửa BACKEND_URL, DEVICE_API_KEY, CLASSROOM_ID...
```

> `insightface` trên Nano có thể cần build (Cython/onnx). Nếu chậm/khó, có thể bỏ
> qua `insightface` và tự nạp 2 file `.onnx` bằng `onnxruntime` trực tiếp — nhưng
> dùng `FaceAnalysis` của insightface là đường ngắn nhất vì đã gồm detect+align.

---

## 2. Chuyển model ONNX → TensorRT FP16 (tăng tốc trên Nano)

Chạy **1 lần trên Nano** để tải `buffalo_l` về:

```bash
python3 - <<'PY'
from insightface.app import FaceAnalysis
app = FaceAnalysis(name='buffalo_l')   # tải ONNX về ~/.insightface/models/buffalo_l
app.prepare(ctx_id=0, det_size=(640,640))
print("downloaded buffalo_l")
PY
```

Sau đó build engine FP16 (nhanh hơn FP32 ~2x, vẫn đủ chính xác cho ArcFace):

```bash
cd /opt/smart-classroom-edge
bash convert_to_trt.sh     # tạo ~/trt_engines/{det_10g,w600k_r50}.fp16.engine
```

`convert_to_trt.sh` dùng `trtexec --fp16`. Lưu ý: **engine TensorRT gắn chặt với
phần cứng + phiên bản TensorRT** → phải build trên chính Nano sẽ chạy, không copy
từ máy khác.

Để `onnxruntime` tái sử dụng engine đã build (tránh build lại mỗi lần khởi động):

```bash
# trong .env
ORT_TENSORRT_ENGINE_CACHE_ENABLE=1
ORT_TENSORRT_CACHE_PATH=/home/jetson/trt_engines
```

> **Cách khác (đơn giản hơn):** bỏ qua `trtexec`, để `TensorrtExecutionProvider`
> tự build engine ở lần chạy đầu (lần đầu chậm 1–3 phút, sau đó cache lại). Đặt
> `ORT_PROVIDERS=TensorrtExecutionProvider,CUDAExecutionProvider,CPUExecutionProvider`
> là đủ.

---

## 3. Đăng ký (enroll) khuôn mặt vào hệ thống

Embedding tham chiếu phải nằm trong DB (`pgvector`) để server so khớp. Có 2 đường:

### 3a. Đẩy gallery đã train lên server (khuyến nghị)

```bash
# Lấy ADMIN_JWT bằng cách đăng nhập tài khoản admin (POST /login) — /enrollment/face là admin-only.
BACKEND_URL=http://192.168.1.10:8091 \
DEVICE_API_KEY=<device-key> ADMIN_JWT=<jwt> \
python3 enroll_from_gallery.py \
  --embeddings /path/NhanDangMSSV/models/embeddings.pkl \
  --id-map     /path/NhanDangMSSV/models/id_map.json
```

Script gom các embedding theo **MSSV**, lấy **centroid** (trung bình rồi
L2-normalize — ổn định hơn 1 ảnh), rồi `POST /enrollment/face {mssv, embedding}`.
Server lưu vào bảng `face_embeddings (student_id, mssv, embedding vector(512))`.

> Điều kiện: MSSV trong `id_map.json` phải khớp cột `students.mssv` trong DB.

### 3b. Enroll trực tiếp từ ảnh (khi có ảnh mới)

Chụp/đưa ảnh → `app.get(img)[0].normed_embedding` (512-d) → `POST /enrollment/face`
với `student_id` hoặc `mssv`. Đây cũng là cách bổ sung học sinh mới sau này.

---

## 4. Chạy dịch vụ nhận diện edge

```bash
cd /opt/smart-classroom-edge
set -a && source .env && set +a
python3 recognize_service.py
```

Pipeline mỗi `FRAME_STRIDE` khung hình: `app.get(frame)` → với mỗi mặt đủ lớn
(`MIN_FACE`) lấy `normed_embedding` →

- **[A] `EDGE_MATCH=false`:** `POST /attendance/scan {classroom_id, device_id, embedding, event_id}`.
  Server trả: điểm danh (`sim≥0.60`), "chờ duyệt" (`0.45≤sim<0.60` → tạo `FaceReview`),
  hoặc "khuôn mặt không xác định" (`sim<0.45`).
- **[B] `EDGE_MATCH=true`:** match FAISS local (gallery đồng bộ từ
  `GET /enrollment/gallery?classroom_id=...`), chỉ gửi `student_id` khi `sim≥FACE_T_HIGH`.

Có `COOLDOWN_SEC` để không spam cùng một người, và heartbeat
`POST /device/heartbeat` mỗi 30 s để server biết camera còn sống.

### Chạy nền bằng systemd (cách thủ công — tham khảo)

```bash
sudo cp smart-classroom-edge.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now smart-classroom-edge
journalctl -u smart-classroom-edge -f
```

> Cách trên chỉ để hiểu cơ chế. **Cách deploy khuyến nghị** (có version, rollback) là
> `package.sh` + `install.sh` ở §4b.

---

## 4b. Đóng gói & deploy module AI (có version) — KHUYẾN NGHỊ

**AI của hệ thống đóng gói ở 2 nơi:**
- **Phía server (đã container hoá):** nhận diện pgvector kNN nằm trong image `backend`
  (Go), trích xuất embedding khi ghi danh nằm trong image `face-enroll` (InsightFace) —
  cả hai chạy bằng `docker compose`. Không cần làm gì thêm.
- **Phía edge (Jetson Nano):** code Python (`recognize_service.py`…) được đóng gói thành
  **artifact có version** rồi cài bằng systemd. Không container hoá trên Nano (xem "Vì sao
  không Docker" cuối mục) — Nano 4GB + CUDA/TensorRT/OpenCV của JetPack hợp với cài bare‑metal hơn.

### Cơ chế: tarball có version + venv `--system-site-packages` + symlink `current`
```
/opt/smart-classroom-edge/
├── releases/
│   ├── edge-v0.1.0/   ← code + venv riêng của bản này
│   └── edge-v0.1.1/
├── current  ─► releases/edge-v0.1.1   (đổi symlink = update/rollback tức thì)
└── .env                                (config + khoá per‑device, chmod 600, GIỮ khi nâng cấp)
```
Mô hình `buffalo_l` (~280MB) trong `~/.insightface` và **engine cache TensorRT** trong
`~/trt_engines` là **node‑local, không portable** → cố ý **để NGOÀI** thư mục release, nên
update/rollback **không** phải tải lại model hay build lại engine.

### Quy trình
```bash
# 1) Trên máy DEV (không cần Jetson): tạo artifact
cd edge/jetson
VERSION=edge-v0.1.0 ./package.sh
#   -> dist/smart-classroom-edge-edge-v0.1.0.tar.gz (+ .sha256, + manifest.json sha256 từng file)

# 2) Copy 2 file (.tar.gz + .sha256) sang Nano (scp), rồi cài TRÊN Nano:
sudo ./install.sh smart-classroom-edge-edge-v0.1.0.tar.gz
#   install.sh: kiểm aarch64 + preflight JetPack, verify sha256, giải nén vào
#   releases/<ver>, tạo venv --system-site-packages (DÙNG LẠI onnxruntime-gpu /
#   OpenCV-CUDA / numpy của JetPack — KHÔNG pip từ PyPI), pip cài insightface/requests,
#   verify get_available_providers() có TensorRT/CUDA, tạo .env (lần đầu), đổi symlink
#   `current`, cài + restart systemd unit.

# 3) Nâng cấp = chạy lại install.sh với tarball mới. Rollback:
sudo ln -sfn /opt/smart-classroom-edge/releases/edge-v0.1.0 /opt/smart-classroom-edge/current
sudo systemctl restart smart-classroom-edge
```

> **Điều kiện một lần (đã nêu §1–2, install.sh GIẢ ĐỊNH đã làm):** `nvpmodel -m 0` +
> `jetson_clocks`, swapfile 4G, `timedatectl set-ntp true` (chống‑replay từ chối `ts`
> lệch giờ), và **wheel onnxruntime‑gpu của NVIDIA Jetson‑Zoo** đã cài vào **system python3**
> (không phải PyPI). Tag git: repo hiện chưa có tag → `git describe` ra SHA ngắn; cắt
> `edge-v0.1.0` để version sạch.

### Khi có NHIỀU camera (fleet) — đường thứ cấp
`git pull` theo **tag** + `systemd timer` (repo đã ở `/opt/smart-classroom-edge`), hoặc
Ansible cho push‑provision. Cho phép cập nhật từ xa; vẫn dùng chung systemd unit ở trên.

### Vì sao KHÔNG Docker trên Nano (cho đồ án)
Container hoá 1 camera trên Nano 4GB là **chi phí thừa + nhiều bẫy**: phải pin tag L4T
`r32.7.1` (không phải r35/r36 = JetPack 5/6), `l4t-base` **không có cv2** (phải `l4t-ml`),
wheel onnxruntime‑gpu PyPI **âm thầm rớt về CPU**, compose v1 trên JetPack 4.6 không tôn
trọng `gpus:`/`runtime: nvidia` per‑service (phải đặt default‑runtime nvidia), camera **CSI
không chạy** trong Docker (cần Argus socket + privileged + pipeline GStreamer mà code chưa
có — dùng USB/RTSP thì được), và Docker + ORT + build engine TRT dễ **OOM** 4GB. Giữ Docker
như phương án "nếu sau này gộp edge vào compose".

---

## 5. Tích hợp với backend (tóm tắt API)

| Endpoint | Ai gọi | Mục đích |
|----------|--------|----------|
| `POST /attendance/scan` | Jetson | gửi embedding (hoặc student_id) → điểm danh / hàng đợi duyệt |
| `GET /enrollment/gallery?classroom_id=` | Jetson (`X-Device-Key`) | đồng bộ gallery cho edge match |
| `POST /enrollment/face` | Admin | lưu embedding tham chiếu (`student_id` hoặc `mssv`) |
| `GET /review-queue` / `POST /review-queue/:id` | Giáo viên/Admin | duyệt các lượt độ tin cậy thấp (FE: **Duyệt nhận diện**) |
| `POST /device/heartbeat` | Jetson | báo sống |

Ngưỡng tin cậy cấu hình ở server qua env `FACE_T_HIGH` (mặc định `0.60`) và
`FACE_T_LOW` (mặc định `0.45`) — giữ đồng bộ với `.env` của Jetson.

---

## 6. Kiểm thử nhanh (không cần camera)

```bash
# Tạo 1 vector ngẫu nhiên 512-d, enroll cho 1 MSSV rồi scan lại đúng vector đó → phải MATCH.
python3 - <<'PY'
import json, requests, numpy as np, uuid
from datetime import datetime, timezone
B="http://localhost:8091"
KEY="<DEVICE_API_KEY>"   # = giá trị trong .env (hoặc token per-device trong device_credentials)
v=np.random.randn(512); v=(v/np.linalg.norm(v)).round(6).tolist()
# enroll (cần admin JWT thật; ở demo có thể seed trực tiếp DB)
print(requests.post(f"{B}/enrollment/face", json={"mssv":"SV0001","embedding":v},
      headers={"X-Device-Key":KEY,"Authorization":"Bearer <ADMIN_JWT>"}).text)
# scan — BẮT BUỘC có event_id + ts (chống phát lại); ts phải trong cửa sổ ±5 phút.
print(requests.post(f"{B}/attendance/scan",
      json={"classroom_id":1,"embedding":v,
            "event_id":str(uuid.uuid4()),
            "ts":datetime.now(timezone.utc).astimezone().isoformat()},
      headers={"X-Device-Key":KEY}).text)
PY
```

Kỳ vọng: lần scan trả `confidence ≈ 1.0` và điểm danh thành công (vì cosine với
chính nó = 1). Hạ `FACE_T_LOW`/`FACE_T_HIGH` để thử nhánh "chờ duyệt".

---

## 7. Tinh chỉnh hiệu năng & xử lý sự cố trên Nano

| Triệu chứng | Cách xử lý |
|-------------|-----------|
| FPS thấp / giật | tăng `FRAME_STRIDE`, giảm `DET_SIZE` (640→480/320), đảm bảo `nvpmodel -m 0` + `jetson_clocks` |
| Lần khởi động đầu rất lâu | TensorRT EP đang build engine — bật engine cache (`ORT_TENSORRT_CACHE_PATH`) |
| OOM khi load model | bật swap 4G (mục 1), `OOMScoreAdjust` trong unit |
| Không nhận diện được ai | kiểm tra gallery đã enroll chưa (`GET /enrollment/gallery`), MSSV khớp DB, ánh sáng/độ phân giải |
| Nhiều "chờ duyệt" | ánh sáng kém hoặc ngưỡng quá cao → chỉnh `FACE_T_HIGH`, bổ sung ảnh enroll đa điều kiện |
| Sai người (false match) | nâng `FACE_T_HIGH`, dùng centroid nhiều ảnh/MSSV (mục 3a) |

---

## 7b. Bảo mật kênh Jetson ↔ server (Jetson cần làm gì)

Dữ liệu nhận diện (embedding + điểm danh) đi từ Jetson lên server. Cơ chế bảo mật
**đang áp dụng** và **việc Jetson phải làm** để tuân thủ:

### Đang dùng (server enforce)
1. **Xác thực thiết bị — `X-Device-Key`** trên mọi request (`RequireDevice`):
   - **Khuyến nghị:** token **per-device** trong bảng `device_credentials` (mỗi
     Jetson 1 token, có cờ `active` để **thu hồi riêng**, server ghi `last_seen`).
   - Hoặc **master key** dùng chung `DEVICE_API_KEY` (chỉ tiện cho fleet sau gateway).
     **Không** còn default `dev-device-key`; ở production để trống master ⇒ chỉ token
     per-device được chấp nhận; dùng default ⇒ server **từ chối khởi động**.
2. **Chống phát lại (anti-replay)** trên `/attendance/scan` và `/device/heartbeat`:
   - `ts` (RFC3339/epoch) phải nằm trong **±`DEVICE_EVENT_MAX_SKEW_SECONDS`** (mặc định 300s).
   - `event_id` (duy nhất/lần phát hiện) được nhớ trong cửa sổ → phát lại = **idempotent**
     (không cộng trùng), gói cũ = **401**. Tắt bằng `DEVICE_REPLAY_PROTECT=off`.
3. **Quyền riêng tư sinh trắc:** Jetson gửi **embedding 512-d**, KHÔNG gửi ảnh mặt thô.
4. **Gating nghiệp vụ:** server chỉ ghi điểm danh khi **đang có tiết** trong phòng
   (`findOngoingClass`) và theo ngưỡng tin cậy (`≥0.60` ghi, `0.45–0.60` chờ duyệt).

### ✅ Checklist Jetson phải làm
- [ ] **Khoá riêng:** đặt `DEVICE_API_KEY` = token per-device của chính nó (xin từ admin),
      không hardcode trong ảnh đĩa; ưu tiên đọc từ biến môi trường / file 600.
- [ ] **Gửi `event_id` + `ts`** mỗi lần `POST /attendance/scan` (và `ts` cho `/device/heartbeat`).
      `recognize_service.py` đã làm sẵn (`iso_now()`); nếu tự viết client thì phải thêm.
- [ ] **Đồng bộ NTP** (`sudo timedatectl set-ntp true`) — Nano không có RTC tin cậy; lệch
      giờ > cửa sổ ⇒ bị từ chối (sai-âm). Kiểm tra `timedatectl status`.
- [ ] **Retry idempotent:** khi mất mạng, gửi lại **cùng `event_id`** (server trả idempotent,
      không đếm trùng). Có backoff.
- [ ] **TLS:** trỏ `BACKEND_URL=https://...` khi server đã bật TLS (xem dưới); không gửi
      khoá qua HTTP trần trên mạng không tin cậy.
- [ ] **Đồng bộ gallery** định kỳ `GET /enrollment/gallery?classroom_id=` (đã có `maybe_sync`).

### Nên nâng cấp (production)
- **mTLS per-device:** cấp client-cert cho mỗi Jetson (Nano đủ sức) thay/bổ sung
  `X-Device-Key` → chống giả mạo & nghe lén ở tầng kết nối. Hoặc ký payload bằng
  HMAC khoá-riêng + `ts`/`event_id` (đã có nonce).
- **HTTPS/WSS + MQTT/TLS `:8883`** toàn data-plane; cô lập camera vào **VLAN riêng**,
  allowlist IP server; xoay vòng khoá định kỳ; rate-limit per-device.
- **Mã hoá at-rest** cột `face_embeddings` (pgvector), chính sách lưu trữ/đồng ý
  (dữ liệu sinh trắc), audit truy cập.

---

## 8. Lộ trình production (tham chiếu)

Xem [docs/ARCHITECTURE.md](ARCHITECTURE.md) cho kiến trúc tổng thể, bảo mật thiết
bị (`X-Device-Key`/mTLS), độ tin cậy hàng đợi (manual-ack + DLQ), và các hạng mục
production còn lại. Phần observability/backup/HA (mục #5 trong roadmap) được để lại
chủ đích theo phạm vi đồ án.
