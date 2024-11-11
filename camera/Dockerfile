# Stage 1: Build stage
FROM python:3.10.5 AS build

# Set the working directory in the build stage
WORKDIR /app

# Install build dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends gcc libc-dev

# Upgrade pip to the latest version
RUN pip install --upgrade pip

# Copy the requirements file and install dependencies
COPY requirements.txt /app/
RUN pip install mediapipe==0.10.15
RUN pip install opencv-python
RUN pip install --no-cache-dir -r requirements.txt
RUN pip install kuksa-client
# Stage 2: Production stage
FROM python:3.10.5-slim

# Set the working directory in the production stage
WORKDIR /app

# Install necessary runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    libgl1 libglib2.0-0 libsm6 libxext6 libxrender1 \
    libxkbcommon0 libxkbcommon-x11-0 libx11-xcb1 \
    libxcb1 libfontconfig1 libfreetype6 libxau6 libxdmcp6 \
    libxinerama1 libxrandr2 libxcursor1 libxfixes3 libxi6 \
    && rm -rf /var/lib/apt/lists/*
RUN pip install kuksa-client
# Copy installed packages and binaries from the build stage
COPY --from=build /usr/local/lib/python3.10/site-packages /usr/local/lib/python3.10/site-packages
COPY --from=build /usr/local/bin /usr/local/bin

# Copy the rest of the application code
COPY . /app
# Set environment variables for Qt
ENV QT_QPA_PLATFORM_PLUGIN_PATH=/usr/local/lib/python3.10/site-packages/cv2/qt/plugins
ENV QT_QPA_FONTDIR=/usr/share/fonts
ENV QT_DEBUG_PLUGINS=1

# Run app.py when the container launches
CMD ["python3", "camera.py"]
