package intelligence

import (
	"strings"
	"testing"
)

func TestUSGSEarthquakeAdapterPreservesGeospatialCoordinates(t *testing.T) {
	// Drives shipped parseUSGSEarthquakes (package-private helper used by usgsCollector).
	feed := `{"features":[{"id":"abc","properties":{"title":"M 5.1 - Test","url":"https://example.invalid/event","mag":5.1,"time":1700000000000},"geometry":{"coordinates":[7.2,50.7,10]}}]}`
	events, err := parseUSGSEarthquakes(strings.NewReader(feed), &IntelligenceSource{Name: "USGS"})
	if err != nil || len(events) != 1 {
		t.Fatalf("earthquake feed parse failed: %#v / %v", events, err)
	}
	if events[0].Longitude != 7.2 || events[0].Latitude != 50.7 || events[0].Severity != "medium" {
		t.Fatalf("geospatial data missing: %#v", events[0])
	}
}
