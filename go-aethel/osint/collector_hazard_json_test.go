package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"strings"
	"testing"

	"go-aethel/intelligence"
)

func TestHazardJSONCollectorParsesEarthquakesWithoutInventingGeo(t *testing.T) {
	collector := NewHazardJSONCollector(OSINTCollectorConfig{Name: "Operator Quakes", Type: CollectorTypeEarthquakeGeoJSON, Domain: intelligence.DomainGeo})
	events, err := collector.parseEarthquakeGeoJSON([]byte(`{
        "features":[
            {"id":"eq-1","properties":{"mag":4.7,"place":"Test Ridge","title":"M 4.7 Test Ridge","url":"https://example.org/eq-1","time":1710000000000},"geometry":{"coordinates":[13.4,52.5,10]}},
            {"id":"invalid","properties":{"mag":3.0,"place":"Outside"},"geometry":{"coordinates":[500,95]}},
            {"id":"bad-mag","properties":{"mag":99,"place":"Nope","title":"M 99"},"geometry":{"coordinates":[10,20]}}
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
	if !strings.Contains(events[0].Title, "[earthquake]") || !strings.Contains(events[0].Summary, "magnitude 4.7") {
		t.Fatalf("earthquake tags missing: title=%q summary=%q", events[0].Title, events[0].Summary)
	}
}

// Incomplete feature must not abort the rest of the USGS feed (break vs continue regression).
func TestHazardJSONCollectorContinuesAfterIncompleteGeometry(t *testing.T) {
	collector := NewHazardJSONCollector(OSINTCollectorConfig{Name: "USGS", Type: CollectorTypeEarthquakeGeoJSON, Domain: intelligence.DomainGeo})
	events, err := collector.parseEarthquakeGeoJSON([]byte(`{
        "features":[
            {"id":"incomplete","properties":{"mag":3.1,"place":"Nowhere","title":"M 3.1 Nowhere","time":1710000000000},"geometry":{"coordinates":[1]}},
            {"id":"eq-valid","properties":{"mag":5.2,"place":"Berlin","title":"M 5.2 Berlin","url":"https://example.org/eq","time":1710000001000},"geometry":{"coordinates":[13.4,52.5,8]}}
        ]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event after incomplete feature, got %d (%+v)", len(events), events)
	}
	if events[0].ID != "eq-valid" || events[0].Lat != 52.5 || events[0].Lon != 13.4 {
		t.Fatalf("valid feature after incomplete must be kept: %+v", events[0])
	}
}

func TestHazardJSONCollectorParsesLatestVolcanoGeometry(t *testing.T) {
	collector := NewHazardJSONCollector(OSINTCollectorConfig{Name: "Operator Volcanoes", Type: CollectorTypeVolcanoEONET, Domain: intelligence.DomainGeo})
	events, err := collector.parseVolcanoEONET([]byte(`{
        "events":[
          {"id":"vol-1","title":"Example Volcano","link":"https://example.org/vol-1","geometry":[
            {"date":"2026-01-01T00:00:00Z","coordinates":[10,20]},
            {"date":"2026-01-02T00:00:00Z","coordinates":[11,21]}
          ]},
          {"id":"vol-no-geo","title":"Ghost Volcano","link":"https://example.org/ghost","geometry":[]},
          {"id":"vol-bad-geo","title":"Bad Volcano","geometry":[{"date":"2026-01-02T00:00:00Z","coordinates":[999,999]}]}
        ]}`))
	if err != nil {
		t.Fatalf("parse volcano source: %v", err)
	}
	if len(events) != 1 || events[0].Lat != 21 || events[0].Lon != 11 || events[0].Status != "raw" {
		t.Fatalf("volcano event contract violated: %+v", events)
	}
	if !strings.Contains(events[0].Title, "[volcano erupting]") || !strings.Contains(events[0].Summary, "[volcano erupting]") {
		t.Fatalf("volcano tags missing: title=%q summary=%q", events[0].Title, events[0].Summary)
	}
}

func TestEnsureDefaultHazardCollectorsInjectsUSGSAndEONET(t *testing.T) {
	// Empty config gets all hazard feeds
	var empty []OSINTCollectorConfig
	if !EnsureDefaultHazardCollectors(&empty) {
		t.Fatal("empty config must be mutated")
	}
	if len(empty) != len(DefaultHazardCollectors()) {
		t.Fatalf("expected %d hazard feeds, got %d", len(DefaultHazardCollectors()), len(empty))
	}
	// Idempotent
	if EnsureDefaultHazardCollectors(&empty) {
		t.Fatal("second ensure must not mutate")
	}
	// Partial config still gets missing ones
	partial := []OSINTCollectorConfig{{Name: "News", Type: "rss", URL: "https://example.org/rss", Enabled: true}}
	if !EnsureDefaultHazardCollectors(&partial) {
		t.Fatal("partial config must gain hazard feeds")
	}
	types := map[string]int{}
	for _, c := range partial {
		types[c.Type]++
	}
	if types[CollectorTypeEarthquakeGeoJSON] < 1 || types[CollectorTypeVolcanoEONET] < 1 {
		t.Fatalf("missing hazard types after ensure: %+v", types)
	}
	// defaultFeeds must include hazard collectors for Live Globe path
	defs := defaultFeeds()
	var hasEQ, hasVol bool
	for _, c := range defs {
		if c.Type == CollectorTypeEarthquakeGeoJSON {
			hasEQ = true
		}
		if c.Type == CollectorTypeVolcanoEONET {
			hasVol = true
		}
	}
	if !hasEQ || !hasVol {
		t.Fatal("defaultFeeds must register earthquake-geojson and volcano-eonet collectors")
	}
}

func TestOSINTEngineAcceptsOnlyExplicitCollectorTypes(t *testing.T) {
	engine := NewOSINTEngine(t.TempDir() + "/sources.json")
	if err := engine.AddCollector(OSINTCollectorConfig{Name: "Bad Collector", Type: "arbitrary-code", URL: "https://example.org/source"}); err == nil {
		t.Fatal("unsupported collector type must be rejected")
	}
}


