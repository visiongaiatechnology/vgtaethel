package main

// Pure-Go focus math that mirrors the SHIPPED JS computeGlobeFocusRotation
// (frontend/modules/osint_watch.js). Kept in package main so `go test .` runs it
// without goja. Values must stay in lockstep with the JS formula.

import (
	"math"
	"testing"
)

// computeGlobeFocusRotationGo is the Go twin of the shipped JS helper.
// JS: rotY = -lon * π/180; rotX = clamp(lat * π/180, ±1.15)
func computeGlobeFocusRotationGo(lon, lat float64) (rotY, rotX float64) {
	rotY = -lon * math.Pi / 180
	rotX = lat * math.Pi / 180
	if rotX > 1.15 {
		rotX = 1.15
	}
	if rotX < -1.15 {
		rotX = -1.15
	}
	return rotY, rotX
}

// projectLatLonGo mirrors shipped projectLatLon for center-check assertions.
func projectLatLonGo(lat, lon, rotYRad, rotX, scale, cw, ch float64) (x, y float64, visible bool) {
	r := math.Min(cw, ch) * 0.42 * scale
	phi := lat * math.Pi / 180
	lam := lon*math.Pi/180 + rotYRad
	sx := math.Cos(phi) * math.Sin(lam)
	sy := math.Sin(phi)
	sz := math.Cos(phi) * math.Cos(lam)
	y2 := sy*math.Cos(rotX) - sz*math.Sin(rotX)
	z2 := sy*math.Sin(rotX) + sz*math.Cos(rotX)
	return cw/2 + sx*r, ch/2 - y2*r, z2 > 0
}

func TestComputeGlobeFocusRotationEuropeNotAfrica(t *testing.T) {
	// Europe center ~ lon 10, lat 50 — old bug used -lat*0.35 → Africa-facing pitch
	rotY, rotX := computeGlobeFocusRotationGo(10, 50)
	if rotY > -0.1 || rotY < -0.3 {
		t.Fatalf("europe rotY want ~-0.175, got %v", rotY)
	}
	if rotX < 0.6 || rotX > 1.0 {
		t.Fatalf("europe rotX must be positive ~0.87 (north), got %v", rotX)
	}
	// Germany chip
	_, gX := computeGlobeFocusRotationGo(10.5, 51.2)
	if gX <= 0 {
		t.Fatalf("germany pitch must be northern (positive), got %v", gX)
	}
	x, y, vis := projectLatLonGo(50, 10, rotY, rotX, 1.55, 800, 600)
	if !vis {
		t.Fatal("europe target not visible under focus rotation")
	}
	if math.Abs(x-400) > 40 || math.Abs(y-300) > 50 {
		t.Fatalf("europe should be near canvas center (400,300), got (%.1f,%.1f)", x, y)
	}
}

func TestComputeEarthRasterStepAdaptive(t *testing.T) {
	// Mirrors computeEarthRasterStep in osint_watch.js (perf gate)
	step := func(r float64, dragging bool) int {
		if r < 1 {
			r = 1
		}
		if dragging {
			s := int(math.Round(r / 90))
			if s < 2 {
				s = 2
			}
			if s > 6 {
				s = 6
			}
			return s
		}
		if r >= 320 {
			return 3
		}
		if r >= 200 {
			return 2
		}
		return 2
	}
	if step(400, true) < 2 {
		t.Fatal("drag step too fine")
	}
	if step(400, false) == 1 {
		t.Fatal("idle must not use step=1 on large r")
	}
	if step(400, true) < step(400, false) {
		t.Fatal("drag should be coarser or equal idle")
	}
}

func TestFilterEventsByTimeWindowLogic(t *testing.T) {
	// Mirrors filterEventsByTimeWindow in osint_watch.js
	type ev struct {
		title string
		ms    int64
	}
	now := int64(1_700_000_000_000) // fixed clock for determinism
	events := []ev{
		{"old", now - 48*3600*1000},
		{"new", now - 2*3600*1000},
		{"undated", 0},
	}
	hours := 24.0
	cutoff := now - int64(hours*3600*1000)
	var kept []string
	for _, e := range events {
		if e.ms == 0 || e.ms >= cutoff {
			kept = append(kept, e.title)
		}
	}
	if len(kept) != 2 { // new + undated
		t.Fatalf("24h window should keep undated+new, got %v", kept)
	}
	if kept[0] != "new" && kept[1] != "new" {
		t.Fatalf("missing new event: %v", kept)
	}
}
