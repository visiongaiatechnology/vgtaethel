package intelligence

import (
	"strings"
	"testing"
)

func TestEONETVolcanoAdapterTagsEruptingEvents(t *testing.T) {
	feed := `{"events":[{"id":"EONET_999","title":"Etna Volcano","link":"https://example.invalid/etna","geometry":[{"date":"2024-06-15T12:00:00Z","coordinates":[15.0,37.75]}]}]}`
	events, err := eonetVolcanoCollector{}.Parse(&IntelligenceSource{Name: "NASA EONET Volcanoes"}, strings.NewReader(feed))
	if err != nil || len(events) != 1 {
		t.Fatalf("volcano feed parse failed: %#v / %v", events, err)
	}
	if events[0].Longitude != 15.0 || events[0].Latitude != 37.75 {
		t.Fatalf("volcano coords missing: %#v", events[0])
	}
	if !strings.Contains(events[0].Summary, "[volcano erupting]") {
		t.Fatalf("erupting tag missing in summary: %q", events[0].Summary)
	}
	if events[0].Severity != "high" {
		t.Fatalf("open volcano should be high severity, got %q", events[0].Severity)
	}
}

func TestUSGSEarthquakeSummaryCarriesMagnitudeTag(t *testing.T) {
	feed := `{"features":[{"id":"abc","properties":{"title":"M 5.1 - Test","url":"https://example.invalid/event","mag":5.1,"time":1700000000000},"geometry":{"coordinates":[7.2,50.7,10]}}]}`
	events, err := parseUSGSEarthquakes(strings.NewReader(feed), &IntelligenceSource{Name: "USGS"})
	if err != nil || len(events) != 1 {
		t.Fatalf("earthquake feed parse failed: %#v / %v", events, err)
	}
	if !strings.Contains(events[0].Summary, "[earthquake]") || !strings.Contains(events[0].Summary, "5.1") {
		t.Fatalf("earthquake magnitude tag missing: %q", events[0].Summary)
	}
}
