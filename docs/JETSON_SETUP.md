# Jetson Nano B01 — Hướng dẫn triển khai Edge AI

## 1. Thông số phần cứng

| Thông số | Giá trị |
|---|---|
| Module | Jetson Nano Developer Kit B01 |
| RAM | 4GB LPDDR4 |
| GPU | 128-core Maxwell |
| Storage | microSD |
| OS | Ubuntu 18.04 (JetPack 4.6) |

---

## 2. Sau khi flash OS — làm ngay (chưa cài gì)

```bash
# Swap 4GB — bắt buộc để cài insightface
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# IP tĩnh — không bao giờ đổi dù đổi mạng
sudo nmcli con mod "Wired connection 1" \
  ipv4.addresses 192.168.2.100/24 \
  ipv4.gateway 192.168.2.1 \
  ipv4.dns "8.8.8.8" \
  ipv4.method manual
sudo nmcli con up "Wired connection 1"
```

---

## 3. Copy file từ máy dev lên Jetson

```bash
# Chạy trên máy dev
ssh-copy-id admin1@192.168.2.100

scp edge/jetson/recognize_service.py admin1@192.168.2.100:~/edge/
scp NhanDangMSSV/models/faiss.index  admin1@192.168.2.100:~/edge/models/
scp NhanDangMSSV/models/id_map.json  admin1@192.168.2.100:~/edge/models/
```

---

## 4. Cài packages (SSH vào Jetson)

```bash
ssh admin1@192.168.2.100

mkdir -p ~/edge/models

# Cài Python 3.8
sudo add-apt-repository ppa:deadsnakes/ppa -y
sudo apt-get update
sudo apt-get install -y python3.8 python3.8-dev python3.8-venv

# Tạo virtualenv
python3.8 -m venv ~/venv

# Cài packages AI
~/venv/bin/pip install --timeout 120 --retries 5 \
    numpy==1.23.5 \
    opencv-python-headless==4.8.0.76 \
    onnxruntime==1.16.3 \
    insightface==0.7.3 \
    faiss-cpu==1.7.4 \
    requests==2.31.0 \
    matplotlib

# Tải buffalo_l model (~280MB, 1 lần duy nhất)
~/venv/bin/python -c "
from insightface.app import FaceAnalysis
FaceAnalysis(name='buffalo_l').prepare(ctx_id=-1)
print('buffalo_l OK')
"
```

---

## 5. Cài systemd — tự chạy khi boot

```bash
sudo tee /etc/systemd/system/edge.service << 'EOF'
[Unit]
Description=Smart Classroom Edge
After=network-online.target

[Service]
User=admin1
WorkingDirectory=/home/admin1/edge
Environment=BACKEND_URL=http://192.168.2.16:8091
Environment=DEVICE_API_KEY=camtok-1
Environment=CLASSROOM_ID=1
Environment=CAMERA_SRC=0
ExecStart=/home/admin1/venv/bin/python /home/admin1/edge/recognize_service.py
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable edge
sudo systemctl start edge
sudo journalctl -u edge -f
```

---

## 6. Lệnh thường dùng

```bash
# Xem log nhận diện
sudo journalctl -u edge -f

# Khởi động lại service
sudo systemctl restart edge

# SSH vào Jetson
ssh admin1@192.168.2.100

# Tắt Jetson đúng cách (KHÔNG rút điện trực tiếp)
sudo poweroff
```

---

## 7. Lưu ý tránh corrupt microSD

| Việc cần làm | Lý do |
|---|---|
| Luôn dùng `sudo poweroff` | Rút điện đột ngột gây corrupt filesystem |
| Dùng nguồn 5V/4A barrel jack | Micro-USB chỉ 2A → thiếu điện → tự reset |
| Cắm qua powerbank | Mất điện không ảnh hưởng |
| Không dùng Docker | Ghi nhiều data → tăng nguy cơ corrupt |

---

## 8. Lý do chọn kiến trúc phần mềm

### 8.1 Tại sao chạy thẳng Python thay vì Docker

| Tiêu chí | Docker | Python thẳng |
|---|---|---|
| RAM overhead | +200MB | 0 |
| Ghi disk | Rất nhiều (layers) | Tối thiểu |
| Nguy cơ corrupt microSD | Cao | Thấp |
| Độ phức tạp | Cao | Thấp |
| Tương thích GPU JetPack | Phức tạp (nvidia-runtime) | Tự nhiên |
| Phù hợp Nano 4GB | Không tối ưu | Tối ưu |

**Kết luận:** Docker phù hợp server có SSD và RAM dư. Jetson Nano 4GB + microSD → Python + systemd là lựa chọn thực tế và ổn định hơn.

### 8.2 Tại sao dùng FAISS thay vì pgvector trên edge

- **pgvector** nằm trên server — cần network để query → độ trễ cao, không hoạt động offline
- **FAISS local** — query trong RAM, dưới 1ms, hoàn toàn offline
- Gallery được sync 1 lần từ server trước demo → không cần kết nối liên tục

### 8.3 Tại sao InsightFace buffalo_l

- SCRFD detector + ArcFace w600k_r50 — bộ đôi state-of-the-art cho face recognition
- buffalo_l được train trên dataset lớn (WebFace600K) → độ chính xác cao trong điều kiện thực tế
- Hỗ trợ TensorRT FP16 trên Maxwell GPU → có thể tăng tốc 3-5x nếu cần

### 8.4 Tại sao tách edge và server

| Tầng | Nhiệm vụ | Lý do |
|---|---|---|
| Jetson (edge) | Nhận diện khuôn mặt | Camera cần xử lý real-time, không thể stream video lên server |
| Server | Lưu trữ, báo cáo, dashboard | Tập trung dữ liệu, nhiều phòng cùng lúc |

Kiến trúc edge-server giảm bandwidth (chỉ gửi kết quả, không gửi video), tăng privacy (video không rời khỏi phòng học), và hoạt động được khi mạng chập chờn.

### 8.5 Tại sao dùng systemd thay vì cron/supervisor

- systemd có sẵn trên Ubuntu, không cài thêm
- `Restart=always` tự khởi động lại khi crash
- `After=network-online.target` đảm bảo có mạng trước khi service chạy
- Log tích hợp qua `journalctl`, không cần cấu hình thêm

---

## 9. Flow hoạt động

```
[Jetson Nano]
  Camera → InsightFace (detect + embed)
         → FAISS search (gallery.json local)
         → confidence >= 0.6
         → POST /attendance/scan
              { mssv, classroom_id, confidence, ts, event_id }

[Server 192.168.2.16:8091]
  → Lưu điểm danh vào PostgreSQL
  → Hiển thị trên Dashboard real-time
```
