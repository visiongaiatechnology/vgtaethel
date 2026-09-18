package geoint

// STATUS: DIAMANT VGT SUPREME

import (
	"testing"
	"time"

	"go-aethel/intelligence"
)

type intelligenceSnapshotFixture struct{ state intelligence.StoreState }

func (fixture intelligenceSnapshotFixture) GetSnapshot() intelligence.StoreState { return fixture.state }

func TestIntelligenceGeoAdapterPreservesDomainAndUncertainty(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	fixture := intelligenceSnapshotFixture{state: intelligence.StoreState{
		Revision: 2,
		Events: []intelligence.Event{
			{ID: "cyber-1", SourceID: "source-1", Title: "Reported outage", Summary: "Source report", Domain: "cyber", Latitude: 52.5, Longitude: 13.4, Confidence: 63, ObservedAt: now.Add(-time.Hour)},
			{ID: "conflict-1", SourceID: "source-2", Title: "Border incident", Domain: "conflict", Latitude: 50, Longitude: 30, Confidence: 71, ObservedAt: now.Add(-2 * time.Hour)},
			{ID: "stale", Title: "Old event", Domain: "economic", Latitude: 1, Longitude: 1, ObservedAt: now.Add(-8 * 24 * time.Hour)},
			{ID: "unknown-location", Title: "No location", Domain: "energy", Latitude: 0, Longitude: 0, ObservedAt: now.Add(-time.Hour)},
		},
	}}
	bus := NewGeoEntityBus()
	adapter := NewIntelligenceGeoAdapter(bus, fixture)
	adapter.Sync(now)

	cyber := bus.ListByType(TypeCyber, 10)
	if len(cyber) != 1 || cyber[0].Assessment != AssessmentCorrelated || cyber[0].Location != PrecisionApproximate {
		t.Fatalf("cyber event uncertainty was flattened: %#v", cyber)
	}
	if cyber[0].SourceIDs[0] != "source-1" || cyber[0].Confidence != 63 {
		t.Fatalf("cyber provenance was lost: %#v", cyber[0])
	}
	conflict := bus.ListByType(TypeConflict, 10)
	if len(conflict) != 1 {
		t.Fatalf("expected conflict marker, got %#v", conflict)
	}
	if bus.ListAll(20); len(bus.ListAll(20)) != 2 {
		t.Fatalf("stale or locationless intelligence leaked into live GEOINT: %#v", bus.ListAll(20))
	}
}

func TestIntelligenceGeoAdapterReconcilesRemovedEvents(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	source := &mutableIntelligenceSnapshot{state: intelligence.StoreState{Revision: 1, Events: []intelligence.Event{{
		ID: "event-1", Title: "Event", Domain: "geo", Latitude: 10, Longitude: 10, ObservedAt: now,
	}}}}
	bus := NewGeoEntityBus()
	adapter := NewIntelligenceGeoAdapter(bus, source)
	adapter.Sync(now)
	source.state = intelligence.StoreState{Revision: 2}
	adapter.Sync(now)
	if _, exists := bus.Get("intel:event:event-1"); exists {
		t.Fatal("removed canonical intelligence event remained as a ghost marker")
	}
}

type mutableIntelligenceSnapshot struct{ state intelligence.StoreState }

func (source *mutableIntelligenceSnapshot) GetSnapshot() intelligence.StoreState { return source.state }

