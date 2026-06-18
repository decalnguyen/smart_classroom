#!/usr/bin/env bash
# Build a VERSIONED, reproducible release tarball of the Jetson edge AI module.
#
# Runs on a DEV machine (no Jetson needed) — it only stages text files. The heavy,
# NON-portable state (buffalo_l ~280MB .onnx model + TensorRT engine cache) is
# deliberately NOT packaged: it is node-local and rebuilt on the Nano.
#
# Output (in dist/):
#   smart-classroom-edge-<VERSION>.tar.gz         the release
#   smart-classroom-edge-<VERSION>.tar.gz.sha256  integrity (install.sh verifies)
# The tarball contains a manifest.json with VERSION + per-file sha256 (provenance).
#
# Usage:  ./package.sh            # VERSION from `git describe`
#         VERSION=edge-v0.1.0 ./package.sh
set -euo pipefail
cd "$(dirname "$0")" # edge/jetson
export COPYFILE_DISABLE=1 # don't let macOS tar add ._ AppleDouble files

# Strip leading "edge-" from VERSION to avoid double prefix in tarball name.
_RAW_VER="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
VERSION="${_RAW_VER#edge-}"
OUT_DIR="${OUT_DIR:-dist}"

# Exactly the edge payload (code + unit + example config). NOTE: enroll_*.py are
# included for convenience but enrollment is meant to run on the training PC
# (albumentations is intentionally not a Nano dependency).
FILES=(
  recognize_service.py
  sync_db.py
  enroll_student.py
  enroll_from_gallery.py
  convert_to_trt.sh
  requirements-jetson.txt
  config.example.env
  gallery.example.json
  schedule.example.json
  smart-classroom-edge.service
)

sha256() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$@"; else shasum -a 256 "$@"; fi; }

TMP="$(mktemp -d)"
STAGE="$TMP/smart-classroom-edge-$VERSION"
mkdir -p "$STAGE" "$OUT_DIR"

for f in "${FILES[@]}"; do
  [[ -f "$f" ]] || { echo "ERROR: missing $f" >&2; exit 1; }
  cp "$f" "$STAGE/"
done

# manifest.json — version + per-file checksums.
{
  echo "{"
  echo "  \"name\": \"smart-classroom-edge\","
  echo "  \"version\": \"$VERSION\","
  echo "  \"files\": {"
  n=${#FILES[@]}; i=0
  for f in "${FILES[@]}"; do
    i=$((i + 1)); sep=,; [[ $i -eq $n ]] && sep=
    sum=$(sha256 "$STAGE/$f" | awk '{print $1}')
    echo "    \"$f\": \"$sum\"$sep"
  done
  echo "  }"
  echo "}"
} >"$STAGE/manifest.json"

TARBALL="$OUT_DIR/smart-classroom-edge-$VERSION.tar.gz"
tar -czf "$TARBALL" -C "$TMP" "smart-classroom-edge-$VERSION"
(cd "$OUT_DIR" && sha256 "smart-classroom-edge-$VERSION.tar.gz" >"smart-classroom-edge-$VERSION.tar.gz.sha256")
rm -rf "$TMP"

echo "built  $TARBALL"
echo "sha256 $(cat "$OUT_DIR/smart-classroom-edge-$VERSION.tar.gz.sha256")"
echo "copy both files to the Nano, then: sudo ./install.sh $(basename "$TARBALL")"
