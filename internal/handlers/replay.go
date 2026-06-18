package handlers

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Replay/idempotency protection for device events (Jetson scan, heartbeat).
// Each event carries a `ts` (RFC3339 or epoch seconds) and an `event_id`:
//   - ts must be within ±DEVICE_EVENT_MAX_SKEW_SECONDS (default 300) of now, so a
//     captured request can't be replayed later (anti-replay).
//   - event_id is remembered for the window; a repeat is treated as idempotent
//     (legit edge retry) rather than re-processed (no double-count).
// Disable with DEVICE_REPLAY_PROTECT=off. Requires device clocks to be NTP-synced.

type eventResult int

const (
	eventOK eventResult = iota
	eventBadTS
	eventStale
	eventDuplicate
)

var (
	seenMu     sync.Mutex
	seenEvents = map[string]time.Time{} // event_id -> expiry
	lastSweep  time.Time
)

func replayEnabled() bool {
	return strings.ToLower(os.Getenv("DEVICE_REPLAY_PROTECT")) != "off"
}

func replayWindow() time.Duration {
	return time.Duration(envInt("DEVICE_EVENT_MAX_SKEW_SECONDS", 300)) * time.Second
}

// parseEventTS accepts RFC3339 ("2026-06-16T13:00:00+07:00") or epoch seconds.
func parseEventTS(ts string) (time.Time, bool) {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t, true
	}
	if n, err := strconv.ParseInt(ts, 10, 64); err == nil {
		return time.Unix(n, 0), true
	}
	return time.Time{}, false
}

// verifyDeviceEvent checks freshness + (optional) idempotency. requireEventID is
// true for attendance scans (dedup matters), false for heartbeats.
func verifyDeviceEvent(eventID, ts string, requireEventID bool) eventResult {
	if !replayEnabled() {
		return eventOK
	}
	win := replayWindow()
	t, ok := parseEventTS(ts)
	if !ok {
		return eventBadTS
	}
	now := time.Now()
	if d := now.Sub(t); d > win || d < -win {
		return eventStale
	}
	if eventID == "" {
		if requireEventID {
			return eventBadTS
		}
		return eventOK
	}
	seenMu.Lock()
	defer seenMu.Unlock()
	if now.Sub(lastSweep) > win { // opportunistic cleanup of expired ids
		for k, exp := range seenEvents {
			if now.After(exp) {
				delete(seenEvents, k)
			}
		}
		lastSweep = now
	}
	if exp, dup := seenEvents[eventID]; dup && now.Before(exp) {
		return eventDuplicate
	}
	seenEvents[eventID] = now.Add(win)
	return eventOK
}
