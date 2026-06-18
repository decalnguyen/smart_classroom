package handlers

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"smart_classroom/internal/db"
)

// Data-driven alarm thresholds (auto-calibration).
//
// Rather than a hand-picked constant, the danger threshold for each metric is
// DERIVED FROM THE DISTRIBUTION OF THE SENSOR DATA WE HAVE ACTUALLY COLLECTED,
// using the standard anomaly-detection rule  T = μ + K·σ  (mean + K standard
// deviations of the normal readings).
//
// On the collected corpus (n≈5·10^5 readings) the normal range is tight:
//   khói   : μ=104, σ=15.6, max quan sát=138  → K=5 ⇒ T≈182
//   nhiệt  : μ=27.6, σ=1.5, max quan sát=30.7 → K=8 ⇒ T≈40
// T≈182 sits comfortably above the all-time observed normal max (138), so the
// false-positive rate is ~0 (5σ ≈ 1 in 3.5M readings), yet it is far below the
// old fixed 300 — so a smouldering fire that only pushes smoke to 180–250 (a
// clear anomaly: >30% above anything ever seen normally) is no longer MISSED.
//
// The computed value is clamped to [floor, ceiling]:
//   - floor:   never alarm on trivial fluctuation in an unusually quiet room.
//   - ceiling: an absolute danger level that ALWAYS alarms, so baseline drift
//              (or a noisy sensor) can never push the trip point dangerously high.
// Readings ≥ ceiling are excluded from the baseline so a past fire/spike cannot
// "poison" μ,σ. Re-runs every THRESHOLD_CAL_SECONDS (default 3600s) over the last
// THRESHOLD_CAL_WINDOW_DAYS (default 14). Disable with THRESHOLD_AUTOCAL=off; an
// explicit SMOKE_THRESHOLD / TEMP_THRESHOLD env value is a manual override that
// always wins. (Matches the thesis Ch5.3 hướng phát triển: thay ngưỡng cố định.)
type calMetric struct {
	like     []string // LIKE patterns matched against lower(device_type)
	kEnv     string   // env override for K (sigma multiplier)
	kDef     float64
	floorEnv string
	floorDef float64
	ceilEnv  string
	ceilDef  float64
}

var calMetrics = map[string]calMetric{
	"smoke": {like: []string{"%smoke%", "%gas%", "%mq%"}, kEnv: "SMOKE_SIGMA_K", kDef: 5, floorEnv: "SMOKE_FLOOR", floorDef: 150, ceilEnv: "SMOKE_CEILING", ceilDef: 300},
	"temp":  {like: []string{"%temp%"}, kEnv: "TEMP_SIGMA_K", kDef: 8, floorEnv: "TEMP_FLOOR", floorDef: 35, ceilEnv: "TEMP_CEILING", ceilDef: 50},
}

var (
	calMu         sync.RWMutex
	calThresholds = map[string]float64{} // metric -> calibrated danger threshold
)

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// CalibrateThresholds runs one calibration synchronously (so the first readings
// already use the data-driven value), then re-calibrates periodically.
func CalibrateThresholds() {
	if os.Getenv("THRESHOLD_AUTOCAL") == "off" {
		log.Println("Threshold auto-calibration disabled (THRESHOLD_AUTOCAL=off); using fixed/env thresholds")
		return
	}
	runCalibration()
	every := time.Duration(envInt("THRESHOLD_CAL_SECONDS", 3600)) * time.Second
	go func() {
		for {
			time.Sleep(every)
			runCalibration()
		}
	}()
}

func runCalibration() {
	if db.DB == nil {
		return
	}
	windowDays := envInt("THRESHOLD_CAL_WINDOW_DAYS", 14)
	minSamples := int64(envInt("THRESHOLD_CAL_MIN_SAMPLES", 500))
	since := time.Now().AddDate(0, 0, -windowDays)

	for name, m := range calMetrics {
		ceil := envFloat(m.ceilEnv, m.ceilDef)
		floor := envFloat(m.floorEnv, m.floorDef)
		k := envFloat(m.kEnv, m.kDef)

		cond := ""
		args := []interface{}{}
		for i, p := range m.like {
			if i > 0 {
				cond += " OR "
			}
			cond += "lower(device_type) LIKE ?"
			args = append(args, p)
		}

		var row struct {
			N     int64
			Mean  float64
			Sigma float64
		}
		err := db.DB.Table("sen_sor_data").
			Select("count(*) as n, coalesce(avg(value),0) as mean, coalesce(stddev_pop(value),0) as sigma").
			Where("("+cond+")", args...).
			Where("value < ?", ceil). // exclude danger spikes from the baseline
			Where("timestamp > ?", since).
			Scan(&row).Error
		if err != nil {
			log.Printf("[threshold-cal] %s: query failed: %v", name, err)
			continue
		}
		if row.N < minSamples {
			log.Printf("[threshold-cal] %s: only %d samples (<%d) → keep fixed threshold", name, row.N, minSamples)
			continue
		}
		t := clampF(row.Mean+k*row.Sigma, floor, ceil)
		calMu.Lock()
		calThresholds[name] = t
		calMu.Unlock()
		log.Printf("[threshold-cal] %s: n=%d μ=%.1f σ=%.1f → μ+%.0fσ=%.1f clamp[%.0f,%.0f] ⇒ ngưỡng=%.1f",
			name, row.N, row.Mean, row.Sigma, k, row.Mean+k*row.Sigma, floor, ceil, t)
	}
}

// dangerThreshold returns the active danger threshold for a metric ("smoke"/"temp").
// Precedence: explicit env override (envKey) > data-calibrated value > fixed fallback.
func dangerThreshold(metric, envKey string, fallback float64) float64 {
	if v := os.Getenv(envKey); v != "" { // manual override always wins
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	calMu.RLock()
	t, ok := calThresholds[metric]
	calMu.RUnlock()
	if ok {
		return t
	}
	return fallback
}
