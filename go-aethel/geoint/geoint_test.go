package geoint

import (
	"context"
	"testing"
	"time"
)

func TestGeoEntityBusOperations(t *testing.T) {
	bus := NewGeoEntityBus()

	// 1. Test Upsert and Get
	e1 := GeoEntity{
		ID:     "air:test01",
		Type:   TypeMilAircraft,
		Source: "adsb.lol",
		Label:  "VIPER-1",
		Position: GeoPosition{
			Latitude:  35.6892,
			Longitude: 51.3890,
			Altitude:  8500,
			Heading:   180,
			Velocity:  250,
		},
		Classification: "MIL_FIGHTER",
		Confidence:     95,
	}

	bus.Upsert(e1)

	got, exists := bus.Get("air:test01")
	if !exists {
		t.Fatalf("expected entity to exist")
	}
	if got.Label != "VIPER-1" {
		t.Errorf("expected label VIPER-1, got %s", got.Label)
	}

	// 2. Test Radius Query
	nearby := bus.QueryRadius(35.7000, 51.4000, 50.0, nil, 10)
	if len(nearby) != 1 {
		t.Errorf("expected 1 nearby entity, got %d", len(nearby))
	}

	far := bus.QueryRadius(40.0, 10.0, 50.0, nil, 10)
	if len(far) != 0 {
		t.Errorf("expected 0 far entities, got %d", len(far))
	}

	// 3. Test Contact Roster Generation
	roster := bus.GetContactRoster(35.6892, 51.3890, 250.0, 10)
	if len(roster.MilAirContacts) != 1 {
		t.Errorf("expected 1 MilAir contact in roster, got %d", len(roster.MilAirContacts))
	}
	if roster.TotalContacts != 1 {
		t.Errorf("expected TotalContacts=1, got %d", roster.TotalContacts)
	}
}

func TestHaversineAndBearingMath(t *testing.T) {
	// Berlin (52.5200, 13.4050) to Paris (48.8566, 2.3522) ≈ 878 km
	dist := HaversineDistance(52.5200, 13.4050, 48.8566, 2.3522)
	if dist < 850 || dist > 900 {
		t.Errorf("expected ~878 km between Berlin and Paris, got %.1f", dist)
	}

	bearing := InitialBearing(52.5200, 13.4050, 48.8566, 2.3522)
	if bearing < 220 || bearing > 260 {
		t.Errorf("expected south-west bearing (230-250), got %.1f", bearing)
	}
}

func TestSatellitePropagation(t *testing.T) {
	tle := TLEData{
		Name:        "ISS (ZARYA)",
		NoradID:     "25544",
		Line1:       "1 25544U 98067A   26180.50000000  .00016717  00000+0  10270-3 0  9021",
		Line2:       "2 25544  51.6400 208.5000 0005000 120.0000 240.0000 15.50000000 12345",
		Inclination: 51.64,
		MeanMotion:  15.5,
		EpochYear:   26,
		EpochDay:    180.5,
		Group:       "stations",
		Category:    "SPACE_STATION",
	}

	lat, lon, altKm, vel := PropagateKeplerian(tle, time.Now().UTC())
	if lat < -52.0 || lat > 52.0 {
		t.Errorf("ISS latitude must be bounded by inclination (-51.64 to +51.64), got %f", lat)
	}
	if lon < -180.0 || lon > 180.0 {
		t.Errorf("Longitude out of range: %f", lon)
	}
	if altKm < 300 || altKm > 550 {
		t.Errorf("Expected LEO altitude ~400km, got %f", altKm)
	}
	if vel < 7.0 || vel > 8.0 {
		t.Errorf("Expected orbital velocity ~7.6 km/s, got %f", vel)
	}
}

func TestCorrelationEngine(t *testing.T) {
	bus := NewGeoEntityBus()
	engine := NewCorrelationEngine(bus)

	now := time.Now().UTC()
	// Add loitering mil flight
	bus.Upsert(GeoEntity{
		ID:        "air:fighter-01",
		Type:      TypeMilAircraft,
		Source:    "adsb.lol",
		Label:     "STRIKE-1",
		Position:  GeoPosition{Latitude: 34.0, Longitude: 45.0, Altitude: 7000},
		Timestamp: now,
	})

	// Add thermal fire anomaly within 30km and 5 minutes
	bus.Upsert(GeoEntity{
		ID:        "fire:firms-99",
		Type:      TypeFire,
		Source:    "nasa_firms",
		Label:     "Thermal Hotspot (250 MW)",
		Position:  GeoPosition{Latitude: 34.1, Longitude: 45.1},
		Timestamp: now.Add(-5 * time.Minute),
	})

	correlations := engine.AnalyzeCrossModalPatterns(context.Background())
	if len(correlations) == 0 {
		t.Fatalf("expected at least 1 correlated event")
	}

	c := correlations[0]
	if c.PrimaryType != TypeConflict {
		t.Errorf("expected PrimaryType CONFLICT, got %s", c.PrimaryType)
	}
	if c.Confidence < 0.8 {
		t.Errorf("expected high confidence correlation, got %f", c.Confidence)
	}
}

func TestGeoEntityBusPrunesExpiredMovingTracks(t *testing.T) {
	bus := NewGeoEntityBus()
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	bus.Upsert(GeoEntity{ID: "stale-air", Type: TypeAircraft, Timestamp: now.Add(-3 * time.Minute)})
	bus.Upsert(GeoEntity{ID: "current-air", Type: TypeAircraft, Timestamp: now.Add(-time.Minute)})
	bus.Upsert(GeoEntity{ID: "expired-report", Type: TypeConflict, Timestamp: now.Add(-time.Hour), ExpiresAt: now.Add(-time.Second)})
	if removed := bus.PruneStale(now); removed != 2 {
		t.Fatalf("expected two stale entities to be removed, got %d", removed)
	}
	if _, exists := bus.Get("current-air"); !exists {
		t.Fatal("current track was removed")
	}
}
