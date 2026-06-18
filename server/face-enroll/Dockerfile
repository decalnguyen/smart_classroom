# Face-enroll service: InsightFace (buffalo_l) on CPU for enrollment.
FROM python:3.10-slim

# OpenCV / onnxruntime runtime libs + a C/C++ toolchain (insightface builds
# native extensions from source; slim has no compiler).
RUN apt-get update && apt-get install -y --no-install-recommends \
        libgl1 libglib2.0-0 libgomp1 \
        build-essential cmake \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY server/face-enroll/requirements.txt .
# Pre-install numpy(<2)+Cython so insightface compiles against the right ABI,
# and make pip resilient to slow/flaky mirrors (big wheels: onnxruntime/opencv).
RUN pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir --timeout 240 --retries 10 "numpy>=1.26,<2" Cython \
    && pip install --no-cache-dir --timeout 240 --retries 10 -r requirements.txt

COPY server/face-enroll/app.py .

ENV PORT=9000 AUG_PER_IMG=8 ORT_PROVIDERS=CPUExecutionProvider
EXPOSE 9000

# Single worker, long timeout (model download on first request can be slow).
CMD ["gunicorn", "-b", "0.0.0.0:9000", "-w", "1", "--timeout", "300", "app:app"]
