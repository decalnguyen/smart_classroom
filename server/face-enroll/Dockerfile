# Face-enroll service: InsightFace (buffalo_l) on CPU for enrollment.
FROM python:3.10-slim

# OpenCV / onnxruntime runtime libs.
RUN apt-get update && apt-get install -y --no-install-recommends \
        libgl1 libglib2.0-0 libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY server/face-enroll/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY server/face-enroll/app.py .

ENV PORT=9000 AUG_PER_IMG=8 ORT_PROVIDERS=CPUExecutionProvider
EXPOSE 9000

# Single worker, long timeout (model download on first request can be slow).
CMD ["gunicorn", "-b", "0.0.0.0:9000", "-w", "1", "--timeout", "300", "app:app"]
