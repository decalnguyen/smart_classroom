#!/usr/bin/env bash
# ============================================================================
# run_ram.sh — start the edge service from the RAM-disk env (corrupted-microSD
# workaround). Run AFTER setup_ram.sh, ON the Jetson, EVERY boot (RAM is wiped).
#
#   sudo bash ~/edge/setup_ram.sh        # once per boot: python+libs into /mnt/ram
#   BACKEND_URL=http://192.168.2.13:8091 DEVICE_API_KEY=<key> \
#     bash ~/edge/run_ram.sh
#
# Needs in ~/edge/models (or already in /mnt/ram/models): embeddings.pkl + id_map.json.
# If the microSD is too corrupt to read ~/edge/models, scp them straight into
# /mnt/ram/models from the Mac, then run this.
# ============================================================================
set -euo pipefail

RP=/mnt/ram/py38binx/usr/bin/python3.8
export PYTHONHOME=/mnt/ram/py38x/usr
export PYTHONPATH=/mnt/ram/pyenv/lib/python3.8/site-packages
export INSIGHTFACE_HOME=/mnt/ram/insightface
export NO_ALBUMENTATIONS_UPDATE=1
export BACKEND_URL="${BACKEND_URL:?set BACKEND_URL, e.g. http://192.168.2.13:8091}"
export DEVICE_API_KEY="${DEVICE_API_KEY:?set DEVICE_API_KEY to match the server .env}"
export CLASSROOM_ID="${CLASSROOM_ID:-1}"
export CAMERA_SRC="${CAMERA_SRC:-0}"
export INDEX_PATH=/mnt/ram/models/faiss.index
export IDMAP_PATH=/mnt/ram/models/id_map.json

[ -x "$RP" ] || { echo "RAM python missing — run 'sudo bash ~/edge/setup_ram.sh' first."; exit 1; }

mkdir -p /mnt/ram/models /mnt/ram/insightface
# Pull the gallery into RAM (from card; harmless if already there).
cp -n ~/edge/models/embeddings.pkl /mnt/ram/models/ 2>/dev/null || true
cp -n ~/edge/models/id_map.json    /mnt/ram/models/ 2>/dev/null || true
[ -f /mnt/ram/models/id_map.json ] || { echo "id_map.json not in /mnt/ram/models — scp it from the Mac."; exit 1; }

# Build faiss.index IN RAM with the RAM faiss 1.7.4 → never a version mismatch.
if [ -f /mnt/ram/models/embeddings.pkl ] && [ ! -f /mnt/ram/models/faiss.index ]; then
  echo "Building faiss.index in RAM..."
  "$RP" - <<'PYEOF'
import sys, pickle
import numpy as np
try:
    import numpy.core as _c
    sys.modules.setdefault("numpy._core", _c)
    sys.modules.setdefault("numpy._core.multiarray", _c.multiarray)
except Exception:
    pass
import faiss
with open("/mnt/ram/models/embeddings.pkl", "rb") as f:
    obj = pickle.load(f)
if isinstance(obj, dict) and "embeddings" in obj:
    obj = obj["embeddings"]
X = np.asarray(obj, dtype="float32")
faiss.normalize_L2(X)
idx = faiss.IndexFlatIP(X.shape[1])
idx.add(X)
faiss.write_index(idx, "/mnt/ram/models/faiss.index")
print(f"faiss.index built (faiss {faiss.__version__}): {X.shape[0]} x {X.shape[1]}")
PYEOF
fi

echo "Starting recognize_service (buffalo_l downloads into RAM on first run)..."
exec "$RP" ~/edge/recognize_service.py
