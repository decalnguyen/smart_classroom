#!/usr/bin/env bash
# Install / upgrade the Jetson edge AI module from a release tarball (package.sh).
# Run ON the Jetson Nano:  sudo ./install.sh smart-classroom-edge-<VERSION>.tar.gz
#
# Atomic & reversible: each release lands in /opt/smart-classroom-edge/releases/<ver>/
# with its own venv; a `current` symlink is flipped last. Rollback = repoint it.
#
# ASSUMES these one-time JetPack prerequisites are already done (see JETSON_DEPLOYMENT.md §1-2):
#   - sudo nvpmodel -m 0 && sudo jetson_clocks
#   - a 4G swapfile (Nano 4GB: mandatory for insightface build + TRT engine build)
#   - sudo timedatectl set-ntp true   (anti-replay rejects stale ts; Nano has no RTC)
#   - the NVIDIA Jetson-Zoo onnxruntime-gpu wheel installed into SYSTEM python3
#     (NOT PyPI). Verify: python3 -c 'import onnxruntime as o; print(o.get_available_providers())'
set -euo pipefail

TARBALL="${1:-}"
[[ -n "$TARBALL" && -f "$TARBALL" ]] || {
  echo "usage: sudo ./install.sh <smart-classroom-edge-VERSION.tar.gz>"; exit 1; }

# --- preflight: must be a real Jetson ---
[[ "$(uname -m)" == "aarch64" ]] || { echo "ERROR: not aarch64 — run this ON the Jetson Nano."; exit 1; }
[[ -x /usr/src/tensorrt/bin/trtexec ]] || echo "WARN: trtexec missing — is this JetPack? (pre-built engines unavailable; ORT will build at first run)"
command -v nvcc >/dev/null 2>&1 || echo "WARN: nvcc missing — JetPack CUDA toolkit may be absent."

BASE=/opt/smart-classroom-edge
RELEASES="$BASE/releases"
sha256() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$@"; else shasum -a 256 "$@"; fi; }

# --- integrity ---
if [[ -f "$TARBALL.sha256" ]]; then
  (cd "$(dirname "$TARBALL")" && sha256 -c "$(basename "$TARBALL").sha256") \
    || { echo "ERROR: checksum mismatch — refusing to install."; exit 1; }
else
  echo "WARN: no $TARBALL.sha256 next to the tarball — skipping integrity check."
fi

# --- unpack to releases/<version> ---
TMP="$(mktemp -d)"
tar -xzf "$TARBALL" -C "$TMP"
SRC="$(echo "$TMP"/smart-classroom-edge-*)"
[[ -d "$SRC" ]] || { echo "ERROR: unexpected tarball layout"; rm -rf "$TMP"; exit 1; }
VERSION="$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['version'])" "$SRC/manifest.json" 2>/dev/null \
  || basename "$SRC" | sed 's/^smart-classroom-edge-//')"
DEST="$RELEASES/$VERSION"
echo ">> installing version '$VERSION' -> $DEST"
sudo mkdir -p "$RELEASES"
sudo rm -rf "$DEST"; sudo mkdir -p "$DEST"
sudo cp -a "$SRC"/. "$DEST"/
rm -rf "$TMP"

# --- venv that REUSES JetPack system packages (cv2-CUDA, numpy, onnxruntime-gpu) ---
# --system-site-packages is mandatory: the GPU onnxruntime + OpenCV come from
# JetPack, NOT pip. We layer only insightface/requests on top.
[[ -d "$DEST/venv" ]] || sudo python3 -m venv --system-site-packages "$DEST/venv"
sudo "$DEST/venv/bin/pip" install --no-cache-dir -U pip
sudo "$DEST/venv/bin/pip" install --no-cache-dir -r "$DEST/requirements-jetson.txt"

# --- verify the GPU execution providers are actually visible (the #1 Jetson trap) ---
echo ">> onnxruntime execution providers in the venv:"
"$DEST/venv/bin/python" - <<'PY' || true
try:
    import onnxruntime as ort
    p = ort.get_available_providers(); print("   ", p)
    if not any(("Tensorrt" in x) or ("CUDA" in x) for x in p):
        print("    WARN: CPU-only! Recognition will be ~1-3 FPS. Install the NVIDIA")
        print("    Jetson-Zoo onnxruntime-gpu wheel into SYSTEM python3 (not PyPI).")
except Exception as e:
    print("    WARN: onnxruntime not importable:", e)
PY

# --- shared config lives OUTSIDE releases; create once, preserve on upgrade ---
if [[ ! -f "$BASE/.env" ]]; then
  sudo cp "$DEST/config.example.env" "$BASE/.env"
  sudo chmod 600 "$BASE/.env"
  echo ">> created $BASE/.env from example — EDIT it: BACKEND_URL, per-device DEVICE_API_KEY"
  echo "   (token from device_credentials), CLASSROOM_ID, DEVICE_ID, CAMERA_SRC."
fi

# --- atomic flip ---
sudo ln -sfn "$DEST" "$BASE/current"

# --- (re)install the systemd unit shipped in this release ---
sudo cp "$DEST/smart-classroom-edge.service" /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable smart-classroom-edge >/dev/null 2>&1 || true
sudo systemctl restart smart-classroom-edge

echo ">> DONE. version=$VERSION  current -> $(readlink -f "$BASE/current")"
echo ">> logs:     journalctl -u smart-classroom-edge -f"
echo ">> rollback: sudo ln -sfn $RELEASES/<old-version> $BASE/current && sudo systemctl restart smart-classroom-edge"
echo ">> releases: ls $RELEASES"
