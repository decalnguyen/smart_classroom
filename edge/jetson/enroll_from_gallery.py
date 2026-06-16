#!/usr/bin/env python3
"""
Bulk enrollment: push a trained InsightFace/FAISS gallery to the backend.

Reads the artifacts from NhanDangMSSV/face_recognition_insightface_faiss.ipynb:
  - embeddings.pkl : array [N, 512] of ArcFace embeddings (row i -> id_map[i])
  - id_map.json    : { "<row index>": "<MSSV>" }

For each MSSV it sends ALL of that student's embeddings (original + augmented)
to /enrollment/face with replace=true, reproducing the FAISS gallery inside
pgvector so the server/Jetson can do the same kNN weighted vote.

Usage:
  BACKEND_URL=http://192.168.1.10:8091 ADMIN_JWT=<jwt> \
  python3 enroll_from_gallery.py --embeddings ../../NhanDangMSSV/models/embeddings.pkl \
                                 --id-map     ../../NhanDangMSSV/models/id_map.json
"""
import os
import json
import pickle
import argparse
from collections import defaultdict

import numpy as np
import requests


def load_embeddings(path):
    with open(path, "rb") as f:
        obj = pickle.load(f)
    if isinstance(obj, dict) and "embeddings" in obj:
        obj = obj["embeddings"]
    return np.asarray(obj, dtype=np.float32)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--embeddings", required=True)
    ap.add_argument("--id-map", required=True)
    ap.add_argument("--backend", default=os.environ.get("BACKEND_URL", "http://localhost:8091"))
    ap.add_argument("--key", default=os.environ.get("DEVICE_API_KEY", "dev-device-key"))
    ap.add_argument("--token", default=os.environ.get("ADMIN_JWT", ""),
                    help="Admin JWT (POST /enrollment/face is admin-only)")
    ap.add_argument("--max-per-student", type=int, default=64,
                    help="cap vectors sent per student (avoid huge requests)")
    args = ap.parse_args()

    embs = load_embeddings(args.embeddings)
    with open(args.id_map) as f:
        id_map = json.load(f)

    # Group embeddings by MSSV.
    buckets = defaultdict(list)
    for idx_str, mssv in id_map.items():
        i = int(idx_str)
        if 0 <= i < len(embs):
            buckets[mssv].append(embs[i])

    headers = {"Content-Type": "application/json", "X-Device-Key": args.key}
    if args.token:
        headers["Authorization"] = f"Bearer {args.token}"

    ok = fail = total_vecs = 0
    for mssv, vecs in buckets.items():
        vecs = vecs[: args.max_per_student]
        # L2-normalize each (ArcFace normed embeddings already are, but be safe).
        payload_embs = []
        for v in vecs:
            v = np.asarray(v, dtype=np.float32)
            n = np.linalg.norm(v)
            if n == 0 or v.shape[0] != 512:
                continue
            payload_embs.append([round(float(x), 6) for x in (v / n).tolist()])
        if not payload_embs:
            continue
        payload = {"mssv": mssv, "embeddings": payload_embs, "replace": True, "source": "bulk"}
        r = requests.post(f"{args.backend}/enrollment/face", json=payload, headers=headers, timeout=30)
        if r.ok:
            ok += 1
            total_vecs += len(payload_embs)
        else:
            fail += 1
            print(f"enroll {mssv} -> {r.status_code} {r.text[:120]}")

    print(f"Done: {ok} students enrolled ({total_vecs} vectors), {fail} failed, {len(buckets)} total.")


if __name__ == "__main__":
    main()
