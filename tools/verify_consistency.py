#!/usr/bin/env python3
"""
verify_consistency.py — automated FE↔BE data-consistency harness.

Logs in as admin/teacher/student, calls the real API + Postgres, and asserts a
battery of invariants (attendance math, role scoping, referential integrity,
leaves/notifications). Run it after any change instead of eyeballing each page.

Usage:
    python3 tools/verify_consistency.py
Env overrides: BASE_URL (default http://localhost:8091), PG_CONTAINER (default postgres),
PG_USER (nhattoan), PG_DB (sensordata).

Invariant catalog derived in: workflow per-session-attendance + page-logic audits.
"""
import json
import os
import subprocess
import sys
import urllib.request
import urllib.error
from datetime import datetime, timezone, timedelta

BASE = os.getenv("BASE_URL", "http://localhost:8091")
PG_CONTAINER = os.getenv("PG_CONTAINER", "postgres")
PG_USER = os.getenv("PG_USER", "nhattoan")
PG_DB = os.getenv("PG_DB", "sensordata")
ACCOUNTS = {"admin": "admin123", "teacher": "teacher123", "student": "student123"}
VN = timezone(timedelta(hours=7))

# ---------- tiny HTTP + psql helpers ----------
def api(path, token=None, method="GET", body=None):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            raw = r.read().decode()
            return r.status, (json.loads(raw) if raw else None)
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, raw
    except Exception as e:
        return 0, str(e)

def login(user):
    st, body = api("/login", method="POST", body={"username": user, "password": ACCOUNTS[user]})
    if st == 200 and isinstance(body, dict):
        return body.get("token"), body.get("account_id")
    return None, None

def psql(sql):
    out = subprocess.run(
        ["docker", "exec", PG_CONTAINER, "psql", "-U", PG_USER, "-d", PG_DB, "-t", "-A", "-F", "|", "-c", sql],
        capture_output=True, text=True, timeout=30,
    )
    if out.returncode != 0:
        raise RuntimeError(out.stderr.strip())
    return [ln for ln in out.stdout.strip().splitlines() if ln != ""]

def psql_scalar(sql):
    rows = psql(sql)
    return rows[0] if rows else ""

# ---------- check framework ----------
RESULTS = []  # (severity, name, ok, detail)
def check(name, severity, ok, detail=""):
    RESULTS.append((severity, name, bool(ok), detail))

def approx(a, b, eps=1e-6):
    return abs(float(a) - float(b)) <= eps

def near_pct(a, b):  # rate stored as float; tolerate rounding
    return abs(float(a) - float(b)) <= 0.01

# ============================================================
# 1) ATTENDANCE MATH (admin)
# ============================================================
def check_attendance_math(admin):
    st, rep = api("/reports/attendance", admin)
    if st != 200 or not isinstance(rep, dict):
        check("reports.attendance reachable", "critical", False, f"HTTP {st}")
        return
    totals = rep.get("totals", {})
    bc = rep.get("by_classroom", [])
    bs = rep.get("by_session", [])
    byd = rep.get("by_date", [])

    def partition_ok(r):
        return r.get("present", 0) + r.get("late", 0) + r.get("excused", 0) + r.get("absent", 0) == r.get("enrolled", 0)

    check("by_session: P+L+E+A == enrolled (each)", "critical",
          all(partition_ok(r) for r in bs), f"{sum(1 for r in bs if not partition_ok(r))} bad rows")
    check("by_classroom: P+L+E+A == enrolled (each)", "critical",
          all(partition_ok(r) for r in bc), f"{sum(1 for r in bc if not partition_ok(r))} bad rows")
    check("totals: P+L+E+A == enrolled", "critical", partition_ok(totals), str(totals))

    def rate_ok(r):
        d = r.get("enrolled", 0) - r.get("excused", 0)
        exp = (r.get("present", 0) + r.get("late", 0)) / d if d > 0 else 0.0
        return near_pct(r.get("rate", 0), exp)
    check("by_session: rate == (P+L)/(enrolled-excused)", "critical", all(rate_ok(r) for r in bs))
    check("by_classroom: rate == (P+L)/(enrolled-excused)", "critical", all(rate_ok(r) for r in bc))
    check("totals: rate formula", "critical", rate_ok(totals))

    allrows = bc + bs + [totals]
    check("rate bounded [0,1] everywhere", "critical",
          all(0 <= r.get("rate", 0) <= 1 for r in allrows))
    check("counts non-negative everywhere", "critical",
          all(all(r.get(k, 0) >= 0 for k in ("present", "late", "excused", "absent", "enrolled")) for r in allrows))

    for k in ("present", "late", "excused", "absent", "enrolled"):
        check(f"sum(by_classroom.{k}) == totals.{k}", "critical",
              sum(r.get(k, 0) for r in bc) == totals.get(k, 0))
        check(f"sum(by_session.{k}) == totals.{k} (incl all_day)", "critical",
              sum(r.get(k, 0) for r in bs) == totals.get(k, 0))

    check("all_day flag == (end_min-start_min>=1380)", "warning",
          all((r.get("all_day", False)) == ((r.get("end_min", 0) - r.get("start_min", 0)) >= 1380) for r in bs))

    if byd:
        last = byd[-1]
        check("by_date[last].present == totals.present+late (slot grain)", "warning",
              last.get("present", 0) == totals.get("present", 0) + totals.get("late", 0),
              f"trend={last.get('present')} vs attended={totals.get('present',0)+totals.get('late',0)}")

    # stats/overview agreement with reports (same VN day)
    st2, stats = api("/stats/overview", admin)
    if st2 == 200 and isinstance(stats, dict):
        at = stats.get("attendance", {})
        check("stats: attended_today == present_today+late_today", "critical",
              at.get("attended_today") == at.get("present_today", 0) + at.get("late_today", 0))
        check("stats: P+L+E+A == enrolled_today", "critical",
              at.get("present_today", 0) + at.get("late_today", 0) + at.get("excused_today", 0) + at.get("absent_today", 0) == at.get("enrolled_today", 0))
        d = at.get("enrolled_today", 0) - at.get("excused_today", 0)
        exp = (at.get("present_today", 0) + at.get("late_today", 0)) / d if d > 0 else 0.0
        check("stats: rate formula", "critical", near_pct(at.get("rate", 0), exp))
        today = datetime.now(VN).strftime("%Y-%m-%d")
        if rep.get("to") == today:
            check("stats == reports (present, same day)", "critical", at.get("present_today") == totals.get("present"))
            check("stats == reports (enrolled, same day)", "critical", at.get("enrolled_today") == totals.get("enrolled"))

    # classrooms/overview attendance == by_classroom row
    st3, ov = api("/classrooms/overview", admin)
    if st3 == 200 and isinstance(ov, list):
        bc_by_id = {r["classroom_id"]: r for r in bc}
        mismatches = 0
        for room in ov:
            row = bc_by_id.get(room.get("classroom_id"))
            a = room.get("attendance", {})
            if row and any(a.get(k, 0) != row.get(k, 0) for k in ("present", "late", "excused", "absent", "enrolled")):
                mismatches += 1
        check("overview.attendance == reports.by_classroom row", "critical", mismatches == 0, f"{mismatches} mismatches")

# ============================================================
# 2) ROLE SCOPING
# ============================================================
def check_roles(admin, teacher, t_acct, student):
    st, rep = api("/reports/attendance", admin)
    check("admin reports is_all=true & scope=admin", "critical",
          isinstance(rep, dict) and rep.get("is_all") is True and rep.get("scope") == "admin")
    if isinstance(rep, dict):
        n_db = int(psql_scalar("SELECT count(*) FROM classrooms"))
        check("admin by_classroom count == all classrooms", "warning",
              len(rep.get("by_classroom", [])) == n_db, f"{len(rep.get('by_classroom',[]))} vs {n_db}")

    st, trep = api("/reports/attendance", teacher)
    check("teacher reports is_all=false & scope=teacher", "critical",
          isinstance(trep, dict) and trep.get("is_all") is False and trep.get("scope") == "teacher")

    # teacher assigned rooms from DB
    tid = psql_scalar(f"SELECT teacher_id FROM teachers WHERE account_id='{t_acct}'")
    assigned = set(psql(f"SELECT classroom_id FROM classroom_teachers WHERE teacher_id={tid}")) if tid else set()

    # review-queue scoped
    st, rq = api("/review-queue", teacher)
    if isinstance(rq, list):
        check("teacher review-queue scoped to assigned rooms", "critical",
              all(str(r.get("classroom_id")) in assigned for r in rq), f"{len(rq)} rows")

    # teacher sensorinf scoped
    st, sens = api("/sensorinf", teacher)
    if isinstance(sens, list):
        taught_rooms = set(psql(f"SELECT DISTINCT cr.classroom_name FROM classes c JOIN classrooms cr ON cr.classroom_id=c.classroom_id WHERE c.teacher_id={tid}")) if tid else set()
        check("teacher /sensorinf scoped to taught rooms", "critical",
              all((s.get("location") in taught_rooms) for s in sens), f"{len(sens)} devices")

    # teacher device control on an UNASSIGNED room -> 403; admin -> 200
    all_rooms = psql("SELECT classroom_id||'|'||classroom_name FROM classrooms")
    unassigned = next((r.split("|")[1] for r in all_rooms if r.split("|")[0] not in assigned), None)
    if unassigned:
        st, _ = api(f"/device/fan/{unassigned}-fan/mode", teacher, method="POST", body={"mode": 0})
        check("teacher device control on unassigned room -> 403", "critical", st == 403, f"HTTP {st} ({unassigned})")
    any_room = all_rooms[0].split("|")[1] if all_rooms else "A101"
    st, _ = api(f"/device/fan/{any_room}-fan/mode", admin, method="POST", body={"mode": 0})
    check("admin device control -> 200", "critical", st == 200, f"HTTP {st}")
    st, _ = api(f"/device/fan/{any_room}-fan/mode", admin, method="POST", body={"mode": 9})
    check("device mode out of range -> 400", "warning", st == 400, f"HTTP {st}")

    # student
    st, my = api("/my/attendance", student)
    check("student /my/attendance linked", "critical", isinstance(my, dict) and my.get("linked") is True)

    # student cannot reach staff/admin endpoints
    forbidden = ["/reports/attendance", "/review-queue", "/audit", "/classes"]
    codes = {p: api(p, student)[0] for p in forbidden}
    check("student blocked from staff/admin endpoints (403)", "critical",
          all(c == 403 for c in codes.values()), str(codes))

    # unauthenticated -> 401
    st, _ = api("/stats/overview", None)
    check("unauthenticated -> 401", "critical", st == 401, f"HTTP {st}")

# ============================================================
# 3) REFERENTIAL INTEGRITY (psql)
# ============================================================
def check_integrity():
    cases = [
        ("no duplicate class_students", "SELECT count(*) FROM (SELECT class_id,student_id FROM class_students GROUP BY 1,2 HAVING count(*)>1) x"),
        ("no duplicate attendances(student,class,date)", "SELECT count(*) FROM (SELECT student_id,class_id,date FROM attendances WHERE class_id IS NOT NULL GROUP BY 1,2,3 HAVING count(*)>1) x"),
        ("no duplicate classroom_teachers", "SELECT count(*) FROM (SELECT classroom_id,teacher_id FROM classroom_teachers GROUP BY 1,2 HAVING count(*)>1) x"),
        ("attendance.class_id exists in classes", "SELECT count(*) FROM attendances a WHERE a.class_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM classes c WHERE c.class_id=a.class_id)"),
        ("attendance.classroom_id exists", "SELECT count(*) FROM attendances a WHERE a.classroom_id<>0 AND NOT EXISTS (SELECT 1 FROM classrooms r WHERE r.classroom_id=a.classroom_id)"),
        ("attendance.student_id exists", "SELECT count(*) FROM attendances a WHERE NOT EXISTS (SELECT 1 FROM students s WHERE s.student_id=a.student_id)"),
        ("attendance_status domain", "SELECT count(*) FROM attendances WHERE attendance_status NOT IN ('present','late','excused','absent')"),
        ("leave status domain", "SELECT count(*) FROM leave_requests WHERE status NOT IN ('pending','approved','rejected')"),
        ("class_students.class_id exists", "SELECT count(*) FROM class_students cs WHERE NOT EXISTS (SELECT 1 FROM classes c WHERE c.class_id=cs.class_id)"),
        ("class_students.student_id exists", "SELECT count(*) FROM class_students cs WHERE NOT EXISTS (SELECT 1 FROM students s WHERE s.student_id=cs.student_id)"),
        ("classes.classroom_id exists", "SELECT count(*) FROM classes c WHERE NOT EXISTS (SELECT 1 FROM classrooms r WHERE r.classroom_id=c.classroom_id)"),
        ("attendance class_id/classroom_id consistent", "SELECT count(*) FROM attendances a JOIN classes c ON c.class_id=a.class_id WHERE a.class_id IS NOT NULL AND a.classroom_id<>c.classroom_id"),
    ]
    for name, sql in cases:
        try:
            n = int(psql_scalar(sql))
            check(name, "critical", n == 0, f"{n} offending rows")
        except Exception as e:
            check(name, "critical", False, f"SQL error: {e}")

# ============================================================
# 4) LEAVES / NOTIFICATIONS
# ============================================================
def check_leaves(admin):
    st, leaves = api("/leaves", admin)
    if isinstance(leaves, list):
        from collections import Counter
        c = Counter(x.get("status") for x in leaves)
        check("leave status counts sum to total", "critical",
              sum(c.values()) == len(leaves) and set(c) <= {"pending", "approved", "rejected"}, dict(c))
        # approved leave today => excused (not absent) for that student's enrolled class
        today = datetime.now(VN).strftime("%Y-%m-%d")
        rows = psql(f"SELECT count(*) FROM attendances WHERE date='{today}' AND attendance_status='absent' AND student_id IN (SELECT student_id FROM leave_requests WHERE date='{today}' AND status='approved')")
        check("no student with approved leave today is marked absent", "critical",
              int(rows[0]) == 0 if rows else True, f"{rows[0] if rows else 0} contradictions")

# ============================================================
def main():
    print(f"▶ Consistency harness against {BASE}\n")
    admin, _ = login("admin")
    teacher, t_acct = login("teacher")
    student, _ = login("student")
    if not admin:
        print("✗ cannot login as admin — is the backend up?")
        sys.exit(2)

    check_attendance_math(admin)
    if teacher and student:
        check_roles(admin, teacher, t_acct, student)
    check_integrity()
    check_leaves(admin)

    crit_fail = [r for r in RESULTS if not r[2] and r[0] == "critical"]
    warn_fail = [r for r in RESULTS if not r[2] and r[0] == "warning"]
    passed = [r for r in RESULTS if r[2]]
    for sev, name, ok, detail in RESULTS:
        icon = "✓" if ok else ("✗" if sev == "critical" else "⚠")
        line = f"{icon} [{sev[:4]}] {name}"
        if not ok and detail:
            line += f"  — {detail}"
        print(line)
    print(f"\n{'='*60}")
    print(f"PASS {len(passed)}  ·  CRITICAL FAIL {len(crit_fail)}  ·  WARN {len(warn_fail)}  ·  total {len(RESULTS)}")
    sys.exit(1 if crit_fail else 0)

if __name__ == "__main__":
    main()
