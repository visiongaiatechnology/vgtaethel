package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"testing"

	"go-aethel/intelligence"
)

func TestHazardJSONCollectorParsesEarthquakesWithoutInventingGeo(t *testing.T) {
	collector := NewHazardJSONCollector(OSINTCollectorConfig{Name: "Operator Quakes", Type: CollectorTypeEarthquakeGeoJSON, Domain: intelligence.DomainGeo})
	events, err := collector.parseEarthquakeGeoJSON([]byte(`{
        "features":[
            {"id":"eq-1","properties":{"mag":4.7,"place":"Test Ridge","title":"M 4.7 Test Ridge","url":"https://example.org/eq-1","time":1710000000000},"geometry":{"coordinates":[13.4,52.5,10]}},
            {"id":"invalid","properties":{"mag":3.0,"place":"Outside"},"geometry":{"coordinates":[500,95]}}
        ]}`))
	if err != nil {
		t.Fatalf("parse earthquake source: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one valid event, got %d", len(events))
	}
	if !events[0].HasGeo || events[0].Lat != 52.5 || events[0].Lon != 13.4 || events[0].Status != "raw" {
		t.Fatalf("earthquake event contract violated: %+v", events[0])
	}
}

func TestHazardJSONCollectorParsesLatestVolcanoGeometry(t *testing.T) {
	collector := NewHazardJSONCollector(OSINTCollectorConfig{Name: "Operator Volcanoes", Type: CollectorTypeVolcanoEONET, Domain: intelligence.DomainGeo})
	events, err := collector.parseVolcanoEONET([]byte(`{
        "events":[{"id":"vol-1","title":"Example Volcano","link":"https://example.org/vol-1","geometry":[
            {"date":"2026-01-01T00:00:00Z","coordinates":[10,20]},
            {"date":"2026-01-02T00:00:00Z","coordinates":[11,21]}
        ]}]}`))
	if err != nil {
		t.Fatalf("parse volcano source: %v", err)
	}
	if len(events) != 1 || events[0].Lat != 21 || events[0].Lon != 11 || events[0].Status != "raw" {
		t.Fatalf("volcano event contract violated: %+v", events)
	}
}

func TestOSINTEngineAcceptsOnlyExplicitCollectorTypes(t *testing.T) {
	engine := NewOSINTEngine(t.TempDir() + "/sources.json")
	if err := engine.AddCollector(OSINTCollectorConfig{Name: "Bad Collector", Type: "arbitrary-code", URL: "https://example.org/source"}); err == nil {
		t.Fatal("unsupported collector type must be rejected")
	}
}
