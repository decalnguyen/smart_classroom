#!/usr/bin/env bash
# Convert the InsightFace buffalo_l ONNX models to TensorRT FP16 engines on the
# Jetson Nano. Run this ON the Nano (engines are hardware/TensorRT-version
# specific and are NOT portable between machines).
#
# buffalo_l ships these ONNX files (downloaded by insightface on first run to
# ~/.insightface/models/buffalo_l/):
#   det_10g.onnx        -> SCRFD face detector
#   w600k_r50.onnx      -> ArcFace recognizer (512-d embedding)   <-- the model
#
# We only NEED to convert these for speed; onnxruntime's TensorRT EP can also
# build engines at runtime (slower first start). Pre-building caches them.
set -euo pipefail

MODELS_DIR="${MODELS_DIR:-$HOME/.insightface/models/buffalo_l}"
OUT_DIR="${OUT_DIR:-$HOME/trt_engines}"
mkdir -p "$OUT_DIR"

build() {
  local name="$1"; shift
  local onnx="$MODELS_DIR/$name.onnx"
  local engine="$OUT_DIR/$name.fp16.engine"
  if [[ ! -f "$onnx" ]]; then
    echo "!! missing $onnx — run a recognise pass once so insightface downloads buffalo_l"
    return 1
  fi
  echo ">> building $engine"
  /usr/src/tensorrt/bin/trtexec \
    --onnx="$onnx" \
    --saveEngine="$engine" \
    --fp16 \
    --workspace=2048 \
    "$@"
  echo "   done: $engine"
}

# Detector: dynamic-ish input; keep default shapes from the ONNX.
build det_10g
# Recognizer: fixed 112x112x3 ArcFace input.
build w600k_r50 --minShapes=input.1:1x3x112x112 \
                --optShapes=input.1:1x3x112x112 \
                --maxShapes=input.1:4x3x112x112

echo
echo "Engines in $OUT_DIR. To make onnxruntime reuse them, point the TensorRT EP"
echo "cache at this dir: ORT_TENSORRT_ENGINE_CACHE_ENABLE=1 ORT_TENSORRT_CACHE_PATH=$OUT_DIR"
