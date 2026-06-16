#!/usr/bin/env python3
"""
Jetson Nano edge recognition service for the Smart Classroom.

Pipeline (matches the training notebook NhanDangMSSV/):
  camera frame --> SCRFD detector --> face align --> ArcFace (w600k_r50, 512-d)
              --> L2-normalize --> match.

Two integration modes (set EDGE_MATCH):
  - EDGE_MATCH=false (default): send the raw 512-d embedding to the backend
    POST /attendance/scan. The server matches against pgvector and applies the
    confidence gate (accept / review-queue / unknown). This keeps the gallery
    authoritative on the server — recommended for the thesis demo.
  - EDGE_MATCH=true: match locally against a FAISS gallery synced from
    GET /enrollment/gallery, and send only the resolved student_id. Lower
    latency / works offline, but the edge must hold the gallery.

The model itself (InsightFace buffalo_l) is loaded via onnxruntime with the
TensorRT / CUDA execution providers — see docs/JETSON_DEPLOYMENT.md for how the
.onnx files become TensorRT FP16 engines on the Nano.

Run:  python3 recognize_service.py   (config from environment / .env)
"""
import os
import time
import json
import logging

import cv2
import numpy as np
import requests

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("edge")


# --------------------------------------------------------------------------- #
# Config (12-factor: everything via env, with sane Nano defaults)
# --------------------------------------------------------------------------- #
def env(key, default=None):
    return os.environ.get(key, default)


BACKEND_URL = env("BACKEND_URL", "http://192.168.1.10:8091")
DEVICE_KEY = env("DEVICE_API_KEY", "dev-device-key")     # X-Device-Key header
CLASSROOM_ID = int(env("CLASSROOM_ID", "1"))
DEVICE_ID = env("DEVICE_ID", f"cam-{CLASSROOM_ID}")
CAMERA_SRC = env("CAMERA_SRC", "0")                       # index, file, or RTSP url
DET_SIZE = int(env("DET_SIZE", "640"))                    # SCRFD input
COOLDOWN_SEC = float(env("COOLDOWN_SEC", "30"))           # per-student re-send guard
MIN_FACE = int(env("MIN_FACE", "60"))                     # min face box px to consider
EDGE_MATCH = env("EDGE_MATCH", "false").lower() == "true"
GALLERY_SYNC_SEC = float(env("GALLERY_SYNC_SEC", "300"))  # refresh local gallery
PROVIDERS = env(
    "ORT_PROVIDERS",
    "TensorrtExecutionProvider,CUDAExecutionProvider,CPUExecutionProvider",
).split(",")

session = requests.Session()
session.headers.update({"X-Device-Key": DEVICE_KEY, "Content-Type": "application/json"})


# --------------------------------------------------------------------------- #
# Model — InsightFace buffalo_l (SCRFD + ArcFace), TensorRT/CUDA via onnxruntime
# --------------------------------------------------------------------------- #
def load_model():
    from insightface.app import FaceAnalysis

    app = FaceAnalysis(name="buffalo_l", providers=PROVIDERS)
    app.prepare(ctx_id=0, det_size=(DET_SIZE, DET_SIZE))
    log.info("InsightFace ready (providers=%s)", PROVIDERS)
    return app


# --------------------------------------------------------------------------- #
# Local gallery (EDGE_MATCH=true) — synced from the backend
# --------------------------------------------------------------------------- #
class Gallery:
    """Holds normalized reference embeddings + ids; cosine == dot for unit vecs."""

    def __init__(self):
        self.ids = np.empty((0,), dtype=np.int64)
        self.mat = np.empty((0, 512), dtype=np.float32)
        self._last = 0.0

    def sync(self):
        try:
            r = session.get(
                f"{BACKEND_URL}/enrollment/gallery",
                params={"classroom_id": CLASSROOM_ID}, timeout=10,
            )
            r.raise_for_status()
            faces = r.json().get("faces", [])
            ids, vecs = [], []
            for f in faces:
                emb = f["embedding"]
                # pgvector returns the vector as a "[a,b,...]" text literal.
                arr = np.array(json.loads(emb) if isinstance(emb, str) else emb, dtype=np.float32)
                n = np.linalg.norm(arr)
                if n > 0:
                    ids.append(int(f["student_id"]))
                    vecs.append(arr / n)
            if vecs:
                self.ids = np.array(ids, dtype=np.int64)
                self.mat = np.vstack(vecs).astype(np.float32)
            self._last = time.time()
            log.info("Gallery synced: %d faces", len(self.ids))
        except Exception as e:  # noqa: BLE001 — keep running on transient errors
            log.warning("Gallery sync failed: %s", e)

    def maybe_sync(self):
        if time.time() - self._last > GALLERY_SYNC_SEC:
            self.sync()

    def match(self, emb, k=5):
        """kNN weighted vote (same scheme as the training notebook): take the k
        nearest gallery vectors, sum cosine per student, confidence = sum/k."""
        n = self.mat.shape[0]
        if n == 0:
            return None, 0.0
        sims = self.mat @ emb  # cosine (both unit-normalized)
        k = min(k, n)
        top = np.argpartition(-sims, k - 1)[:k]
        votes = {}
        for idx in top:
            sid = int(self.ids[idx])
            votes[sid] = votes.get(sid, 0.0) + float(sims[idx])
        best = max(votes, key=votes.get)
        return best, votes[best] / k


# --------------------------------------------------------------------------- #
# Backend reporting
# --------------------------------------------------------------------------- #
def report_embedding(emb, event_id):
    """Server-side matching path: push the raw embedding."""
    payload = {
        "classroom_id": CLASSROOM_ID,
        "device_id": DEVICE_ID,
        "embedding": [round(float(x), 6) for x in emb.tolist()],
        "event_id": event_id,
    }
    return session.post(f"{BACKEND_URL}/attendance/scan", json=payload, timeout=10)


def report_student(student_id, confidence, event_id):
    """Edge-matching path: send the resolved student + confidence."""
    payload = {
        "classroom_id": CLASSROOM_ID,
        "device_id": DEVICE_ID,
        "student_id": student_id,
        "event_id": event_id,
    }
    return session.post(f"{BACKEND_URL}/attendance/scan", json=payload, timeout=10)


def heartbeat():
    try:
        session.post(f"{BACKEND_URL}/device/heartbeat", json={"device_id": DEVICE_ID}, timeout=5)
    except Exception:  # noqa: BLE001
        pass


# --------------------------------------------------------------------------- #
# Main loop
# --------------------------------------------------------------------------- #
def open_camera(src):
    try:
        return cv2.VideoCapture(int(src))
    except ValueError:
        return cv2.VideoCapture(src)  # file / RTSP


def main():
    app = load_model()
    gallery = Gallery()
    if EDGE_MATCH:
        gallery.sync()

    cap = open_camera(CAMERA_SRC)
    if not cap.isOpened():
        log.error("Cannot open camera %s", CAMERA_SRC)
        return

    last_seen = {}   # student_id/face-hash -> ts, for cooldown
    last_hb = 0.0
    frame_i = 0
    log.info("Edge service started (classroom=%s, edge_match=%s)", CLASSROOM_ID, EDGE_MATCH)

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
        if EDGE_MATCH:
            gallery.maybe_sync()

        # Detect + embed every Nth frame (Nano throughput).
        if frame_i % int(env("FRAME_STRIDE", "5")) != 0:
            continue

        for face in app.get(frame):
            x1, y1, x2, y2 = face.bbox.astype(int)
            if min(x2 - x1, y2 - y1) < MIN_FACE:
                continue
            emb = face.normed_embedding  # already L2-normalized 512-d
            event_id = f"{DEVICE_ID}-{int(now)}-{x1}_{y1}"

            try:
                if EDGE_MATCH:
                    sid, sim = gallery.match(emb, k=int(env("FACE_KNN", "5")))
                    if sid is None or sim < float(env("FACE_T_HIGH", "0.60")):
                        continue
                    if now - last_seen.get(sid, 0) < COOLDOWN_SEC:
                        continue
                    last_seen[sid] = now
                    resp = report_student(sid, sim, event_id)
                else:
                    # Cooldown keyed by a coarse face hash to avoid spamming.
                    key = hash(emb[:8].tobytes())
                    if now - last_seen.get(key, 0) < COOLDOWN_SEC:
                        continue
                    last_seen[key] = now
                    resp = report_embedding(emb, event_id)
                log.info("scan -> %s %s", resp.status_code, resp.text[:160])
            except Exception as e:  # noqa: BLE001
                log.warning("report failed: %s", e)

    cap.release()


if __name__ == "__main__":
    main()
