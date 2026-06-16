#!/usr/bin/env python3
"""
Face-enroll service — turns an uploaded student photo into ArcFace embeddings,
using the SAME model as the training notebook (NhanDangMSSV/) so the vectors are
directly comparable to what the Jetson produces at recognition time.

This is the *enrollment* half of the system. It is deliberately separate from
recognition (which runs on the Jetson): enrollment is infrequent and runs fine
on CPU, so the heavy GPU model only needs to live on the edge.

Endpoints:
  GET  /healthz            -> {status, model_loaded}
  POST /embed  (multipart) -> {embeddings: [[512]...], faces: N}
       field 'image'       extracts the largest face's normed embedding plus
                           AUG_PER_IMG augmented variants (robust gallery).
  POST /recognize-embed    -> same as /embed but returns only the single
       (multipart 'image')   original embedding (no augmentation) — handy for
                            an ad-hoc web check-in demo.

The model (InsightFace buffalo_l: SCRFD detector + ArcFace w600k_r50) downloads
on first request to ~/.insightface; mount a volume to cache it across restarts.
"""
import io
import os

import cv2
import numpy as np
from flask import Flask, request, jsonify

# ---- Config (mirrors NhanDangMSSV training config) ----
DET_SIZE = (int(os.environ.get("DET_SIZE", "640")), int(os.environ.get("DET_SIZE", "640")))
AUG_PER_IMG = int(os.environ.get("AUG_PER_IMG", "8"))
PROVIDERS = os.environ.get("ORT_PROVIDERS", "CPUExecutionProvider").split(",")

app = Flask(__name__)
_face_app = None  # lazy-loaded


def get_model():
    """Lazy-load buffalo_l so the container starts instantly; model downloads on
    first /embed (cached in ~/.insightface)."""
    global _face_app
    if _face_app is None:
        from insightface.app import FaceAnalysis
        m = FaceAnalysis(name="buffalo_l", allowed_modules=["detection", "recognition"], providers=PROVIDERS)
        m.prepare(ctx_id=-1, det_size=DET_SIZE)  # ctx_id=-1 => CPU
        _face_app = m
    return _face_app


# ---- Augmentation pipeline (identical to the training notebook) ----
def _build_aug():
    import albumentations as A
    return A.Compose([
        A.HorizontalFlip(p=0.5),
        A.RandomBrightnessContrast(brightness_limit=0.35, contrast_limit=0.35, p=0.7),
        A.GaussNoise(var_limit=(10.0, 60.0), p=0.5),
        A.Rotate(limit=18, border_mode=cv2.BORDER_REFLECT, p=0.6),
        A.ShiftScaleRotate(shift_limit=0.08, scale_limit=0.12, rotate_limit=10,
                           border_mode=cv2.BORDER_REFLECT, p=0.5),
        A.CLAHE(clip_limit=3.0, p=0.3),
        A.RandomGamma(gamma_limit=(75, 130), p=0.35),
        A.MotionBlur(blur_limit=5, p=0.25),
        A.CoarseDropout(num_holes_range=(1, 4), hole_height_range=(10, 25),
                        hole_width_range=(10, 25), p=0.3),
        A.HueSaturationValue(hue_shift_limit=12, sat_shift_limit=25, val_shift_limit=20, p=0.4),
    ])


_aug = None


def augment_image(img_bgr, n):
    global _aug
    if _aug is None:
        _aug = _build_aug()
    rgb = cv2.cvtColor(img_bgr, cv2.COLOR_BGR2RGB)
    out = []
    for _ in range(n):
        a = _aug(image=rgb)["image"]
        out.append(cv2.cvtColor(a, cv2.COLOR_RGB2BGR))
    return out


def largest_embedding(img_bgr):
    """Return the normed 512-d embedding of the largest face, or None."""
    faces = get_model().get(img_bgr)
    if not faces:
        return None
    largest = max(faces, key=lambda f: (f.bbox[2] - f.bbox[0]) * (f.bbox[3] - f.bbox[1]))
    return largest.normed_embedding.astype("float32"), len(faces)


def read_image():
    f = request.files.get("image")
    if f is None:
        return None
    buf = np.frombuffer(f.read(), dtype=np.uint8)
    return cv2.imdecode(buf, cv2.IMREAD_COLOR)


@app.get("/healthz")
def healthz():
    return jsonify(status="ok", model_loaded=_face_app is not None)


@app.post("/embed")
def embed():
    img = read_image()
    if img is None:
        return jsonify(error="missing image"), 400
    res = largest_embedding(img)
    if res is None:
        return jsonify(embeddings=[], faces=0), 200
    emb, n_faces = res
    embeddings = [emb.tolist()]
    # Augmented variants → multiple gallery vectors (robust to lighting/pose).
    for aug in augment_image(img, AUG_PER_IMG):
        r = largest_embedding(aug)
        if r is not None:
            embeddings.append(r[0].tolist())
    return jsonify(embeddings=[[round(float(x), 6) for x in e] for e in embeddings], faces=n_faces)


@app.post("/recognize-embed")
def recognize_embed():
    img = read_image()
    if img is None:
        return jsonify(error="missing image"), 400
    res = largest_embedding(img)
    if res is None:
        return jsonify(embedding=None, faces=0), 200
    emb, n = res
    return jsonify(embedding=[round(float(x), 6) for x in emb.tolist()], faces=n)


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", "9000")))
