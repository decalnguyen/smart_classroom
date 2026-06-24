#!/usr/bin/env bash
# ============================================================================
# rebuild_index.sh — redeploy an UPDATED face model to an already-set-up Jetson
# WITHOUT a full reinstall (no package install, no buffalo_l re-download).
#
# Why this exists: faiss.index in the git repo is built on the Mac (faiss 1.8);
# the Jetson runs faiss 1.7.4 and would fail with "Index type not recognized" if
# the Mac index were copied directly. So the index MUST be rebuilt here, on the
# Jetson, from the (version-independent) embeddings.pkl.
#
# DEPLOY a new model:
#   1) From the dev Mac (source of truth = git repo), push the two model files:
#        scp NhanDangMSSV/models/embeddings.pkl  admin1@<JETSON_IP>:~/edge/models/
#        scp NhanDangMSSV/models/id_map.json     admin1@<JETSON_IP>:~/edge/models/
#        scp edge/jetson/rebuild_index.sh        admin1@<JETSON_IP>:~/edge/
#   2) On the Jetson:
#        chmod +x ~/edge/rebuild_index.sh
#        ~/edge/rebuild_index.sh
#
# Idempotent — safe to re-run. Requires setup_persistent.sh to have run once
# (it creates the venv + installs the edge.service).
# ============================================================================
set -euo pipefail

EDGE_DIR="$HOME/edge"
MODELS="$EDGE_DIR/models"
PY="${PY:-$HOME/venv/bin/python}"

[ -x "$PY" ] || { echo "!! venv python not found at $PY — run setup_persistent.sh first."; exit 1; }
[ -f "$MODELS/embeddings.pkl" ] || { echo "!! $MODELS/embeddings.pkl missing — scp it from the Mac first."; exit 1; }
[ -f "$MODELS/id_map.json" ]   || { echo "!! $MODELS/id_map.json missing — scp it from the Mac first."; exit 1; }

echo "==> Rebuilding faiss.index from embeddings.pkl (faiss 1.7.4, on the Jetson)"
"$PY" - "$MODELS" <<'PYEOF'
import os, sys, json, pickle
import numpy as np
# A pkl saved with numpy 2.x references numpy._core; alias it to numpy.core so it
# loads under the Jetson's pinned numpy 1.23.5.
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
n_map = len(json.load(open(os.path.join(models, "id_map.json"))))
print(f"faiss.index built (faiss {faiss.__version__}): {X.shape[0]} vectors x {X.shape[1]}")
print(f"id_map entries: {n_map}")
# A row<->MSSV misalignment would make the camera report the WRONG student. Fail loud.
assert X.shape[0] == n_map, f"MISMATCH: {X.shape[0]} embeddings != {n_map} id_map rows"
print("alignment OK")
PYEOF

echo "==> Restarting edge service"
if systemctl list-unit-files 2>/dev/null | grep -q '^edge.service'; then
  sudo systemctl restart edge
  echo "    restarted. live logs:  sudo journalctl -u edge -f"
else
  echo "    edge.service not installed — run setup_persistent.sh, or start manually:"
  echo "    INDEX_PATH=$MODELS/faiss.index IDMAP_PATH=$MODELS/id_map.json $PY $EDGE_DIR/recognize_service.py"
fi
echo "==> Done."
