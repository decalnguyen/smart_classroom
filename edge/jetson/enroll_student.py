#!/usr/bin/env python3
"""
Enroll ONE student from a folder of photos, using the same model as training.
Mirrors add_face() from NhanDangMSSV/: for each photo, extract the largest
face's embedding plus AUG_PER_IMG augmented variants, then POST them all to
/enrollment/face (replace=true).

Run on any machine with the model (training PC or Jetson):
  BACKEND_URL=http://192.168.1.10:8091 ADMIN_JWT=<jwt> \
  python3 enroll_student.py --mssv 22520001 --images ./photos/22520001
"""
import os
import glob
import argparse

import cv2
import numpy as np
import requests
from insightface.app import FaceAnalysis

AUG_PER_IMG = int(os.environ.get("AUG_PER_IMG", "8"))
DET_SIZE = int(os.environ.get("DET_SIZE", "640"))


def build_model():
    providers = os.environ.get("ORT_PROVIDERS", "CUDAExecutionProvider,CPUExecutionProvider").split(",")
    app = FaceAnalysis(name="buffalo_l", allowed_modules=["detection", "recognition"], providers=providers)
    app.prepare(ctx_id=0, det_size=(DET_SIZE, DET_SIZE))
    return app


def aug_pipeline():
    import albumentations as A
    return A.Compose([
        A.HorizontalFlip(p=0.5),
        A.RandomBrightnessContrast(brightness_limit=0.35, contrast_limit=0.35, p=0.7),
        A.GaussNoise(var_limit=(10.0, 60.0), p=0.5),
        A.Rotate(limit=18, border_mode=cv2.BORDER_REFLECT, p=0.6),
        A.ShiftScaleRotate(shift_limit=0.08, scale_limit=0.12, rotate_limit=10, border_mode=cv2.BORDER_REFLECT, p=0.5),
        A.CLAHE(clip_limit=3.0, p=0.3),
        A.RandomGamma(gamma_limit=(75, 130), p=0.35),
        A.MotionBlur(blur_limit=5, p=0.25),
        A.CoarseDropout(num_holes_range=(1, 4), hole_height_range=(10, 25), hole_width_range=(10, 25), p=0.3),
        A.HueSaturationValue(hue_shift_limit=12, sat_shift_limit=25, val_shift_limit=20, p=0.4),
    ])


def largest_emb(app, img):
    faces = app.get(img)
    if not faces:
        return None
    f = max(faces, key=lambda x: (x.bbox[2] - x.bbox[0]) * (x.bbox[3] - x.bbox[1]))
    return f.normed_embedding.astype("float32")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--mssv", required=True)
    ap.add_argument("--images", required=True, help="folder of the student's photos")
    ap.add_argument("--backend", default=os.environ.get("BACKEND_URL", "http://localhost:8091"))
    ap.add_argument("--key", default=os.environ.get("DEVICE_API_KEY", "dev-device-key"))
    ap.add_argument("--token", default=os.environ.get("ADMIN_JWT", ""))
    ap.add_argument("--no-aug", action="store_true")
    args = ap.parse_args()

    app = build_model()
    aug = None if args.no_aug else aug_pipeline()

    paths = []
    for ext in ("*.jpg", "*.jpeg", "*.png", "*.bmp"):
        paths += glob.glob(os.path.join(args.images, ext))
    if not paths:
        print("No images found in", args.images)
        return

    embs = []
    for p in paths:
        img = cv2.imread(p)
        if img is None:
            continue
        e = largest_emb(app, img)
        if e is not None:
            embs.append(e.tolist())
        if aug is not None:
            rgb = cv2.cvtColor(img, cv2.COLOR_BGR2RGB)
            for _ in range(AUG_PER_IMG):
                a = aug(image=rgb)["image"]
                ae = largest_emb(app, cv2.cvtColor(a, cv2.COLOR_RGB2BGR))
                if ae is not None:
                    embs.append(ae.tolist())

    if not embs:
        print("No faces detected.")
        return

    headers = {"Content-Type": "application/json", "X-Device-Key": args.key}
    if args.token:
        headers["Authorization"] = f"Bearer {args.token}"
    embs = [[round(float(x), 6) for x in e] for e in embs]
    r = requests.post(f"{args.backend}/enrollment/face",
                      json={"mssv": args.mssv, "embeddings": embs, "replace": True, "source": "cli"},
                      headers=headers, timeout=60)
    print(f"enroll {args.mssv}: {r.status_code} {r.text[:160]} ({len(embs)} vectors)")


if __name__ == "__main__":
    main()
