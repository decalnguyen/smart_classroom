#!/usr/bin/env bash
# ============================================================================
# setup_persistent.sh — one-shot persistent install of the Jetson edge AI
# service onto a FRESHLY FLASHED microSD (Jetson has its own stable power).
#
# Run ONCE per fresh flash, ON the Jetson:
#     chmod +x ~/edge/setup_persistent.sh
#     ~/edge/setup_persistent.sh
#
# PREREQS — copy these from the dev Mac first (source of truth = the git repo):
#     scp edge/jetson/recognize_service.py        admin1@<JETSON_IP>:~/edge/
#     scp edge/jetson/setup_persistent.sh         admin1@<JETSON_IP>:~/edge/
#     scp NhanDangMSSV/models/embeddings.pkl      admin1@<JETSON_IP>:~/edge/models/
#     scp NhanDangMSSV/models/id_map.json         admin1@<JETSON_IP>:~/edge/models/
#
# What it does (idempotent — safe to re-run):
#   1) 4GB swap + NTP (Nano has no RTC; server anti-replay rejects stale ts)
#   2) Python 3.8 venv + PINNED packages (numpy 1.23.5, faiss-cpu 1.7.4 — the
#      two versions that have repeatedly bitten us)
#   3) download buffalo_l (SCRFD + ArcFace) once
#   4) BUILD faiss.index ON the Jetson from embeddings.pkl  ->  no 1.7.4-vs-1.8
#      "Index type not recognized" ever again
#   5) install + start the systemd service (auto-start on boot)
# ============================================================================
set -euo pipefail

EDGE_DIR="$HOME/edge"
MODELS="$EDGE_DIR/models"
VENV="$HOME/venv"
PY="$VENV/bin/python"

# ---- tweak these to your environment (or pre-export before running) ----
BACKEND_URL="${BACKEND_URL:-http://192.168.2.13:8091}"   # the Go server
DEVICE_API_KEY="${DEVICE_API_KEY:-camtok-1}"             # MUST match server .env
CLASSROOM_ID="${CLASSROOM_ID:-1}"
CAMERA_SRC="${CAMERA_SRC:-0}"
SHOW_WINDOW="${SHOW_WINDOW:-1}"                          # 1 = kiosk window on attached monitor
BANNER_SECONDS="${BANNER_SECONDS:-8}"                    # how long the result banner stays

echo "==> [1/5] System prep: swap + NTP"
if ! sudo swapon --show | grep -q /swapfile; then
  sudo fallocate -l 4G /swapfile
  sudo chmod 600 /swapfile
  sudo mkswap /swapfile
  sudo swapon /swapfile
  grep -q '/swapfile' /etc/fstab || echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
fi
sudo timedatectl set-ntp true || true

echo "==> [2/5] Python 3.8 venv + pinned packages"
if ! command -v python3.8 >/dev/null 2>&1; then
  sudo add-apt-repository ppa:deadsnakes/ppa -y
  sudo apt-get update
  sudo apt-get install -y python3.8 python3.8-dev python3.8-venv
fi
[ -d "$VENV" ] || python3.8 -m venv "$VENV"
"$PY" -m pip install --upgrade pip
# Pinned. numpy 1.23.5 (NOT 2.x → numpy._core pickle break); faiss-cpu 1.7.4
# (must match the index built here). onnxruntime CPU build (swap to the NVIDIA
# Jetson onnxruntime-gpu wheel if you want GPU — see docs/JETSON_DEPLOYMENT.md).
# opencv-python (FULL, not headless) → has highgui/imshow for the kiosk window.
# Both publish aarch64/cp38 wheels; the full one bundles Qt so imshow works on the
# Jetson desktop without extra GTK libs. (Recognition API is identical.)
"$PY" -m pip install --timeout 120 --retries 5 \
    numpy==1.23.5 \
    opencv-python==4.8.0.76 \
    onnxruntime==1.16.3 \
    insightface==0.7.3 \
    faiss-cpu==1.7.4 \
    requests==2.31.0 \
    pillow

echo "==> [3/5] Download buffalo_l model (once)"
NO_ALBUMENTATIONS_UPDATE=1 "$PY" - <<'PYEOF'
from insightface.app import FaceAnalysis
app = FaceAnalysis(name="buffalo_l", allowed_modules=["detection", "recognition"])
app.prepare(ctx_id=-1)
print("buffalo_l ready")
PYEOF

echo "==> [4/5] Build faiss.index ON the Jetson from embeddings.pkl"
if [ -f "$MODELS/embeddings.pkl" ]; then
  "$PY" - "$MODELS" <<'PYEOF'
import os, sys, pickle
import numpy as np
# Defensive: a pkl saved with numpy 2.x references numpy._core; alias it to
# numpy.core so it loads under the pinned numpy 1.23.5.
try:
    import numpy.core as _c
    sys.modules.setdefault("numpy._core", _c)
    sys.modules.setdefault("numpy._core.multiarray", _c.multiarray)
except Exception:
    pass
import faiss
models = sys.argv[1]
with open(os.path.join(models, "embeddings.pkl"), "rb") as f:
    obj = pickle.load(f)
if isinstance(obj, dict) and "embeddings" in obj:
    obj = obj["embeddings"]
X = np.asarray(obj, dtype="float32")
faiss.normalize_L2(X)                 # cosine via inner product on unit vectors
index = faiss.IndexFlatIP(X.shape[1])
index.add(X)
faiss.write_index(index, os.path.join(models, "faiss.index"))
print(f"faiss.index built (faiss {faiss.__version__}): {X.shape[0]} vectors x {X.shape[1]}")
PYEOF
else
  echo "    !! $MODELS/embeddings.pkl not found — skipping index build."
  echo "       scp it from the Mac, then re-run, or copy a faiss.index built with faiss 1.7.4."
fi

echo "==> [5/5] systemd service (kiosk on boot, headless fallback)"
# Starts after the graphical session so DISPLAY=:0 exists (needs desktop auto-login).
# If no display is available, recognize_service.py falls back to headless (still records).
sudo tee /etc/systemd/system/edge.service >/dev/null <<EOF
[Unit]
Description=Smart Classroom Edge (face recognition kiosk)
After=graphical.target network-online.target
Wants=network-online.target

[Service]
User=$USER
WorkingDirectory=$EDGE_DIR
Environment=BACKEND_URL=$BACKEND_URL
Environment=DEVICE_API_KEY=$DEVICE_API_KEY
Environment=CLASSROOM_ID=$CLASSROOM_ID
Environment=CAMERA_SRC=$CAMERA_SRC
Environment=INDEX_PATH=$MODELS/faiss.index
Environment=IDMAP_PATH=$MODELS/id_map.json
Environment=NO_ALBUMENTATIONS_UPDATE=1
Environment=SHOW_WINDOW=$SHOW_WINDOW
Environment=BANNER_SECONDS=$BANNER_SECONDS
Environment=DISPLAY=:0
Environment=XAUTHORITY=$HOME/.Xauthority
ExecStartPre=/bin/sh -c 'xhost +SI:localuser:$USER >/dev/null 2>&1 || true'
ExecStart=$PY $EDGE_DIR/recognize_service.py
Restart=always
RestartSec=5

[Install]
WantedBy=graphical.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable edge
sudo systemctl restart edge

echo
echo "==> Done. Live logs:  sudo journalctl -u edge -f"
echo "    Edit server IP/key in /etc/systemd/system/edge.service then: sudo systemctl daemon-reload && sudo systemctl restart edge"
