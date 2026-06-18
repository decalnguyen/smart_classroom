# ── Stage 1: build ──────────────────────────────────────────────────────────
# Cài packages cần compiler (build-essential, cmake). Stage này bị bỏ sau khi build.
FROM python:3.8-slim AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential cmake \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir --user --timeout 120 --retries 5 \
        numpy==1.23.5 \
        opencv-python-headless==4.8.0.76 \
        onnxruntime==1.16.3 \
        insightface==0.7.3 \
        faiss-cpu==1.7.4 \
        requests==2.31.0 \
        python-dotenv==1.0.1

# ── Stage 2: runtime ─────────────────────────────────────────────────────────
# Chỉ giữ runtime lib + packages đã build — không có compiler (~300MB nhẹ hơn)
FROM python:3.8-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        libgl1-mesa-glx \
        libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

# Copy packages từ builder (không copy build tools)
COPY --from=builder /root/.local /root/.local

WORKDIR /app
COPY recognize_service.py .

ENV PATH=/root/.local/bin:$PATH
ENV INSIGHTFACE_HOME=/models

CMD ["python3", "recognize_service.py"]
