// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/sphere/travel/travel_engine_test.go

package travel

import (
	"testing"

	"go-aethel/sphere"
)

func TestTravelEngineCreateAndCorrelate(t *testing.T) {
	dir := t.TempDir()
	store, err := sphere.NewObjectStore(dir)
	if err != nil {
		t.Fatalf("NewObjectStore failed: %v", err)
	}

	engine := NewEngine(store)

	trip := &sphere.TripObject{
		Title:          "Japan Entdeckungsreise 2027",
		Destination:    "Tokyo, Japan",
		Destinations:   []string{"Kyoto", "Osaka"},
		StartDate:      "2027-10-10",
		EndDate:        "2027-10-22",
		BudgetTotalEUR: 3500,
	}

	created, err := engine.CreateTrip(trip)
	if err != nil {
		t.Fatalf("CreateTrip failed: %v", err)
	}
	if created.DurationDays != 13 {
		t.Fatalf("expected 13 days, got %d", created.DurationDays)
	}

	// Correlate with Global Watch Risk in Tokyo
	impact, err := engine.CorrelateGlobalWatchIntelligence(created.ID, "Tokyo", "Schwere Unwetterwarnung / Taifunlage", "CRITICAL", "HIGH")
	if err != nil {
		t.Fatalf("Correlate failed: %v", err)
	}
	if impact == nil || impact.Region != "Tokyo" {
		t.Fatalf("expected impact in Tokyo, got %+v", impact)
	}

	// Verify trip was updated with risk
	updated, err := engine.GetTrip(created.ID)
	if err != nil {
		t.Fatalf("GetTrip failed: %v", err)
	}
	if len(updated.GlobalWatchRisks) != 1 {
		t.Fatalf("expected 1 risk, got %d", len(updated.GlobalWatchRisks))
	}
}
