package geoint

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// CorrelationEngine runs multi-sensor spatial and temporal pattern detection
type CorrelationEngine struct {
	bus            *GeoEntityBus
	mu             sync.Mutex
	lastRun        time.Time
	maxWindow      time.Duration
	proximityKm    float64
	onCorrelatedFn func(event CorrelatedEvent)
}

// NewCorrelationEngine creates a CorrelationEngine
func NewCorrelationEngine(bus *GeoEntityBus) *CorrelationEngine {
	return &CorrelationEngine{
		bus:         bus,
		maxWindow:   45 * time.Minute,
		proximityKm: 75.0, // km correlation radius
	}
}

// SetOnCorrelated registers a callback invoked when a high-confidence correlation is synthesized
func (e *CorrelationEngine) SetOnCorrelated(fn func(event CorrelatedEvent)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onCorrelatedFn = fn
}

// AnalyzeCrossModalPatterns scans current entity bus for multi-sensor coincidences
func (e *CorrelationEngine) AnalyzeCrossModalPatterns(ctx context.Context) []CorrelatedEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now().UTC()
	e.lastRun = now

	// 1. Gather relevant subsets
	milAir := e.bus.ListByType(TypeMilAircraft, 200)
	fires := e.bus.ListByType(TypeFire, 200)
	quakes := e.bus.ListByType(TypeEarthquake, 100)
	news := e.bus.ListByType(TypeNewsEvent, 200)
	conflicts := e.bus.ListByType(TypeConflict, 100)

	var synthesized []CorrelatedEvent

	// Pattern A: Military Aircraft + Thermal Anomaly / Fire Spike (Strike / Kinetic event pattern)
	for _, air := range milAir {
		for _, fire := range fires {
			dist := HaversineDistance(air.Position.Latitude, air.Position.Longitude, fire.Position.Latitude, fire.Position.Longitude)
			timeDiff := math.Abs(air.Timestamp.Sub(fire.Timestamp).Minutes())

			if dist <= e.proximityKm && timeDiff <= e.maxWindow.Minutes() {
				event := CorrelatedEvent{
					ID:        fmt.Sprintf("corr-strike-%s-%s", air.ID, fire.ID),
					Title:     fmt.Sprintf("Kinetic / Strike Correlation: %s near %s", air.Label, fire.Label),
					Summary:   fmt.Sprintf("Military flight %s detected within %.1f km of a thermal anomaly (%s) within a %.0f minute temporal window.", air.Label, dist, fire.Label, timeDiff),
					CenterLat: (air.Position.Latitude + fire.Position.Latitude) / 2.0,
					CenterLon: (air.Position.Longitude + fire.Position.Longitude) / 2.0,
					Confidence: 0.88,
					TemporalWindowMinutes: timeDiff,
					PrimaryType: TypeConflict,
					EntityIDs:   []string{air.ID, fire.ID},
					Evidence: []string{
						fmt.Sprintf("ADS-B fix: %s (%s)", air.Label, air.Classification),
						fmt.Sprintf("Thermal signature: %s", fire.Label),
					},
					Assessment: fmt.Sprintf("Possible tactical or strike operation. Flight %s loitering near thermal anomaly.", air.Label),
					CreatedAt:  now,
				}
				synthesized = append(synthesized, event)
				e.bus.RecordCorrelation(event)
				if e.onCorrelatedFn != nil {
					e.onCorrelatedFn(event)
				}
			}
		}
	}

	// Pattern B: Severe Earthquake + News / OSINT reports (Disaster & Humanitarian impact)
	for _, quake := range quakes {
		mag, _ := quake.Metadata["magnitude"].(float64)
		if mag < 5.0 {
			continue
		}

		for _, item := range append(news, conflicts...) {
			dist := HaversineDistance(quake.Position.Latitude, quake.Position.Longitude, item.Position.Latitude, item.Position.Longitude)
			timeDiff := math.Abs(quake.Timestamp.Sub(item.Timestamp).Minutes())

			if dist <= 150.0 && timeDiff <= 120.0 {
				event := CorrelatedEvent{
					ID:        fmt.Sprintf("corr-disaster-%s-%s", quake.ID, item.ID),
					Title:     fmt.Sprintf("Seismic & Humanitarian Correlation: %s", quake.Label),
					Summary:   fmt.Sprintf("Major earthquake %s corroborated by nearby incident/report %s (distance %.1f km).", quake.Label, item.Label, dist),
					CenterLat: quake.Position.Latitude,
					CenterLon: quake.Position.Longitude,
					Confidence: 0.94,
					TemporalWindowMinutes: timeDiff,
					PrimaryType: TypeEarthquake,
					EntityIDs:   []string{quake.ID, item.ID},
					Evidence: []string{
						fmt.Sprintf("USGS seismic detection: M%.1f", mag),
						fmt.Sprintf("OSINT Report: %s", item.Label),
					},
					Assessment: fmt.Sprintf("High-impact natural disaster confirmed with local operational disruption reports."),
					CreatedAt:  now,
				}
				synthesized = append(synthesized, event)
				e.bus.RecordCorrelation(event)
				if e.onCorrelatedFn != nil {
					e.onCorrelatedFn(event)
				}
			}
		}
	}

	return synthesized
}
