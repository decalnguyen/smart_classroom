#!/usr/bin/env python3
"""
Jetson edge: nhận diện khuôn mặt -> POST điểm danh lên server.

Cần có sẵn trên Jetson:
  - models/faiss.index   (từ notebook face_recognition_insightface_faiss.ipynb)
  - models/id_map.json   (index -> mssv)
  - buffalo_l model tại ~/.insightface/models/buffalo_l/

Config qua .env hoặc environment variable.
"""
import base64
import json
import logging
import os
import time
from datetime import datetime, timezone

import cv2
import faiss
import numpy as np
import requests
from insightface.app import FaceAnalysis

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("edge")

# ── Config ──────────────────────────────────────────────────────────────────
BACKEND_URL  = os.getenv("BACKEND_URL",  "http://192.168.2.16:8091")
DEVICE_KEY   = os.getenv("DEVICE_API_KEY", "camtok-1")
CLASSROOM_ID = int(os.getenv("CLASSROOM_ID", "1"))
DEVICE_ID    = os.getenv("DEVICE_ID", f"cam-{CLASSROOM_ID}")
CAMERA_SRC   = os.getenv("CAMERA_SRC", "0")
THRESHOLD    = float(os.getenv("FACE_THRESHOLD", "0.6"))
COOLDOWN_SEC = float(os.getenv("COOLDOWN_SEC", "30"))
FRAME_STRIDE = int(os.getenv("FRAME_STRIDE", "5"))
MIN_FACE     = int(os.getenv("MIN_FACE", "60"))
INDEX_PATH   = os.getenv("INDEX_PATH", "models/faiss.index")
IDMAP_PATH   = os.getenv("IDMAP_PATH", "models/id_map.json")
SHOW_WINDOW  = os.getenv("SHOW_WINDOW", "0") == "1"   # kiosk: fullscreen cv2 window
WINDOW_NAME  = "Diem danh - Smart Classroom"

session = requests.Session()
session.headers.update({"X-Device-Key": DEVICE_KEY, "Content-Type": "application/json"})

# ── Load FAISS + id_map ──────────────────────────────────────────────────────
log.info("Loading FAISS index: %s", INDEX_PATH)
index = faiss.read_index(INDEX_PATH)
with open(IDMAP_PATH, encoding="utf-8") as f:
    id_map = {int(k): v for k, v in json.load(f).items()}
log.info("FAISS ready: %d vectors", index.ntotal)

# ── Load InsightFace ─────────────────────────────────────────────────────────
_insightface_root = os.getenv("INSIGHTFACE_HOME", os.path.expanduser("~/.insightface"))
app = FaceAnalysis(name="buffalo_l", root=_insightface_root, allowed_modules=["detection", "recognition"])
app.prepare(ctx_id=0, det_size=(640, 640))
log.info("InsightFace ready")

# ── Kiosk display (optional, SHOW_WINDOW=1) ──────────────────────────────────
# Fullscreen window: live camera + bounding box + name/MSSV + a status banner
# driven by the server response (success / already / not-enrolled). Vietnamese
# text is drawn with PIL (cv2.putText can't render diacritics).
_font = _font_big = None
kiosk = SHOW_WINDOW   # runtime flag; flips off if OpenCV GUI / display is unavailable
if SHOW_WINDOW:
    try:
        from PIL import ImageFont
        _fp = next((p for p in [
            "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
            "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
        ] if os.path.exists(p)), None)
        if _fp:
            _font, _font_big = ImageFont.truetype(_fp, 22), ImageFont.truetype(_fp, 30)
    except Exception as e:
        log.warning("PIL/font unavailable → ASCII labels only: %s", e)
    try:
        cv2.namedWindow(WINDOW_NAME, cv2.WINDOW_NORMAL)
        try:
            cv2.setWindowProperty(WINDOW_NAME, cv2.WND_PROP_FULLSCREEN, cv2.WINDOW_FULLSCREEN)
        except Exception:
            pass
    except Exception as e:
        # opencv-python-headless or no X display → keep running headless.
        log.warning("OpenCV GUI không khả dụng (bản headless / không có màn hình) → chạy headless: %s", e)
        kiosk = False

# Status code -> RGB (matches the web colors). cv2 needs BGR (reversed).
_STATUS_RGB = {
    "success": (22, 163, 74), "already_present": (37, 99, 235),
    "not_enrolled": (220, 38, 38), "student_not_found": (220, 38, 38),
    "low_confidence": (234, 88, 12), "default": (100, 116, 139),
}
def _rgb(code): return _STATUS_RGB.get(code, _STATUS_RGB["default"])
def _bgr(code): r, g, b = _rgb(code); return (b, g, r)

def draw_kiosk(frame, faces_draw, banner):
    """faces_draw: [((x1,y1,x2,y2), label, code)]; banner: (text, code) or None."""
    for (x1, y1, x2, y2), _, code in faces_draw:
        cv2.rectangle(frame, (x1, y1), (x2, y2), _bgr(code), 2)
    if _font is not None:
        from PIL import Image, ImageDraw
        img = Image.fromarray(cv2.cvtColor(frame, cv2.COLOR_BGR2RGB))
        d = ImageDraw.Draw(img)
        for (x1, y1, x2, y2), label, code in faces_draw:
            ty = max(0, y1 - 26)
            d.rectangle([x1, ty, x1 + 11 * len(label) + 10, ty + 24], fill=_rgb(code))
            d.text((x1 + 5, ty + 1), label, font=_font, fill=(255, 255, 255))
        if banner:
            text, code = banner
            h, w = frame.shape[:2]
            d.rectangle([0, h - 58, w, h], fill=_rgb(code))
            d.text((18, h - 50), text[:80], font=_font_big, fill=(255, 255, 255))
        return cv2.cvtColor(np.array(img), cv2.COLOR_RGB2BGR)
    # ASCII fallback (no diacritics)
    for (x1, y1, x2, y2), label, code in faces_draw:
        cv2.putText(frame, label, (x1, max(15, y1 - 8)), cv2.FONT_HERSHEY_SIMPLEX, 0.6, _bgr(code), 2)
    if banner:
        text, code = banner
        h, w = frame.shape[:2]
        cv2.rectangle(frame, (0, h - 50), (w, h), _bgr(code), -1)
        cv2.putText(frame, text[:60], (16, h - 16), cv2.FONT_HERSHEY_SIMPLEX, 0.8, (255, 255, 255), 2)
    return frame

# ── Helpers ──────────────────────────────────────────────────────────────────
def iso_now():
    return datetime.now(timezone.utc).astimezone().isoformat()

def recognize(embedding):
    emb = np.array(embedding, dtype="float32").reshape(1, -1)
    D, I = index.search(emb, 1)
    if I.size == 0 or I[0, 0] < 0:
        return None, 0.0
    mssv = id_map.get(int(I[0, 0]))
    conf = float(D[0, 0])
    return mssv, conf

def crop_face_b64(frame, bbox, pad=0.25, max_w=160):
    """Crop the detected face (with padding), JPEG-encode, return base64 (or None).
    Sent to the server only to be RELAYED live to the web feed — not stored."""
    h, w = frame.shape[:2]
    x1, y1, x2, y2 = bbox.astype(int)
    bw, bh = x2 - x1, y2 - y1
    x1 = max(0, int(x1 - bw * pad)); y1 = max(0, int(y1 - bh * pad))
    x2 = min(w, int(x2 + bw * pad)); y2 = min(h, int(y2 + bh * pad))
    crop = frame[y1:y2, x1:x2]
    if crop.size == 0:
        return None
    if crop.shape[1] > max_w:  # downscale to keep the payload small (~5-15KB)
        scale = max_w / crop.shape[1]
        crop = cv2.resize(crop, (max_w, int(crop.shape[0] * scale)))
    ok, buf = cv2.imencode(".jpg", crop, [int(cv2.IMWRITE_JPEG_QUALITY), 80])
    return base64.b64encode(buf).decode("ascii") if ok else None

def send_attendance(mssv, conf, event_id, face_b64=None):
    payload = {
        "mssv":         mssv,
        "classroom_id": CLASSROOM_ID,
        "device_id":    DEVICE_ID,
        "confidence":   round(conf, 4),
        "event_id":     event_id,
        "ts":           iso_now(),
    }
    if face_b64:
        payload["face_image"] = face_b64
    return session.post(f"{BACKEND_URL}/attendance/scan", json=payload, timeout=10)

def heartbeat():
    try:
        session.post(f"{BACKEND_URL}/device/heartbeat",
                     json={"device_id": DEVICE_ID, "ts": iso_now()}, timeout=5)
    except Exception:
        pass

# ── Main loop ────────────────────────────────────────────────────────────────
def open_camera(src):
    try:
        return cv2.VideoCapture(int(src))
    except ValueError:
        return cv2.VideoCapture(src)

cap = open_camera(CAMERA_SRC)
if not cap.isOpened():
    log.error("Không mở được camera: %s", CAMERA_SRC)
    raise SystemExit(1)

last_seen    = {}    # mssv -> timestamp, tránh gửi trùng trong cooldown
last_hb      = 0.0
frame_i      = 0
faces_draw   = []    # boxes persisted between detection passes (kiosk)
banner       = None  # (text, code) — last server response
banner_until = 0.0
log.info("Bắt đầu nhận diện (classroom=%d, threshold=%.2f, kiosk=%s)", CLASSROOM_ID, THRESHOLD, kiosk)

while True:
    ok, frame = cap.read()
    if not ok:
        time.sleep(0.5)
        continue
    frame_i += 1

    now = time.time()
    if now - last_hb > 30:
        heartbeat()
        last_hb = now

    # Detect only every FRAME_STRIDE; draw persists in between for a smooth display.
    if frame_i % FRAME_STRIDE == 0:
        dets = []
        for face in app.get(frame):
            x1, y1, x2, y2 = face.bbox.astype(int)
            x1, y1 = max(0, x1), max(0, y1)
            if min(x2 - x1, y2 - y1) < MIN_FACE:
                continue

            mssv, conf = recognize(face.normed_embedding)
            if not mssv or conf < THRESHOLD:
                dets.append(((x1, y1, x2, y2), "Không rõ", "not_enrolled"))
                continue

            label, code = f"{mssv} ({conf:.2f})", "success"
            if now - last_seen.get(mssv, 0) >= COOLDOWN_SEC:
                last_seen[mssv] = now
                event_id = f"{DEVICE_ID}-{int(now)}-{mssv}"
                face_b64 = crop_face_b64(frame, face.bbox)
                try:
                    resp = send_attendance(mssv, conf, event_id, face_b64)
                    try:
                        body = resp.json()
                    except Exception:
                        body = {}
                    code = body.get("code") or ("success" if resp.status_code == 200 else "default")
                    msg = body.get("message") or body.get("error") or ""
                    label = f"{body.get('student_name') or mssv} ({conf:.2f})"
                    banner, banner_until = ((msg or label), code), now + 5
                    if resp.status_code == 200:
                        log.info("✓ %s conf=%.3f — %s", mssv, conf, msg)
                    else:
                        log.warning("✗ %s conf=%.3f HTTP=%d — %s", mssv, conf, resp.status_code, msg)
                except Exception as e:
                    log.warning("Gửi thất bại: %s", e)
                    code = "default"
            dets.append(((x1, y1, x2, y2), label, code))
        faces_draw = dets

    if kiosk:
        try:
            disp = draw_kiosk(frame.copy(), faces_draw, banner if now < banner_until else None)
            cv2.imshow(WINDOW_NAME, disp)
            if (cv2.waitKey(1) & 0xFF) in (27, ord("q")):  # ESC / q để thoát
                break
        except Exception as e:
            log.warning("Lỗi hiển thị kiosk → tắt cửa sổ, chạy headless: %s", e)
            kiosk = False

cap.release()
if kiosk:
    cv2.destroyAllWindows()
