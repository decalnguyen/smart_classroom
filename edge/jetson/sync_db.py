#!/usr/bin/env python3
"""
Pull gallery + schedule from the backend and save them as local JSON files.

Run ONCE before the demo (or whenever roster/timetable changes):
  python3 sync_db.py

The recognize_service.py then reads gallery.json + schedule.json offline.
Config is read from the same .env / environment variables as recognize_service.py.
"""
import json
import logging
import os
import sys

import requests
from dotenv import load_dotenv

load_dotenv()
logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("sync_db")


def env(key, default=None):
    return os.environ.get(key, default)


BACKEND_URL  = env("BACKEND_URL",  "http://192.168.2.16:8091")
DEVICE_KEY   = env("DEVICE_API_KEY", "")
CLASSROOM_ID = int(env("CLASSROOM_ID", "1"))

GALLERY_FILE  = env("GALLERY_FILE",  "gallery.json")
SCHEDULE_FILE = env("SCHEDULE_FILE", "schedule.json")

session = requests.Session()
session.headers.update({"X-Device-Key": DEVICE_KEY, "Content-Type": "application/json"})


def sync_gallery():
    """GET /enrollment/gallery?classroom_id=X -> gallery.json"""
    log.info("Pulling gallery from %s ...", BACKEND_URL)
    r = session.get(
        f"{BACKEND_URL}/enrollment/gallery",
        params={"classroom_id": CLASSROOM_ID},
        timeout=30,
    )
    r.raise_for_status()
    faces = r.json().get("faces", [])

    # Group multiple embeddings per student.
    students: dict = {}
    for f in faces:
        sid = int(f["student_id"])
        emb = f["embedding"]
        arr = json.loads(emb) if isinstance(emb, str) else emb
        if sid not in students:
            students[sid] = {
                "student_id": sid,
                "mssv": f.get("mssv", ""),
                "name": f.get("name", ""),
                "embeddings": [],
            }
        students[sid]["embeddings"].append(arr)

    gallery = list(students.values())
    with open(GALLERY_FILE, "w", encoding="utf-8") as fp:
        json.dump(gallery, fp, ensure_ascii=False, indent=2)
    log.info("gallery.json saved: %d students", len(gallery))


def sync_schedule():
    """GET /schedule?classroom_id=X -> schedule.json"""
    log.info("Pulling schedule from %s ...", BACKEND_URL)
    r = session.get(
        f"{BACKEND_URL}/schedule",
        params={"classroom_id": CLASSROOM_ID},
        timeout=15,
    )
    r.raise_for_status()
    raw = r.json()

    # Normalise to the format recognize_service.py expects.
    # Backend returns entries with weekday (0=Mon) + time_start/time_end HH:MM.
    slots = []
    for entry in raw.get("schedules", raw if isinstance(raw, list) else []):
        slots.append({
            "class_id":   entry.get("class_id") or entry.get("id"),
            "subject":    entry.get("subject", ""),
            "weekday":    int(entry.get("weekday", 0)),
            "time_start": entry.get("time_start", "07:00"),
            "time_end":   entry.get("time_end",   "09:30"),
        })

    with open(SCHEDULE_FILE, "w", encoding="utf-8") as fp:
        json.dump(slots, fp, ensure_ascii=False, indent=2)
    log.info("schedule.json saved: %d slots", len(slots))


if __name__ == "__main__":
    errors = 0
    try:
        sync_gallery()
    except Exception as e:
        log.error("gallery sync failed: %s", e)
        errors += 1
    try:
        sync_schedule()
    except Exception as e:
        log.error("schedule sync failed: %s", e)
        errors += 1
    sys.exit(errors)
