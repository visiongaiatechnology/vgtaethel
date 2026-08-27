package osint

import (
	"strings"
	"testing"
)

// parseEONETObservations is the shipped body path used by eonetConnector.Fetch.
func TestParseEONETObservationsSkipsInvalidGeo(t *testing.T) {
	body := []byte(`{
		"events":[
			{"id":"e1","title":"Storm A","categories":[{"title":"Severe Storms"}],"geometry":[]},
			{"id":"e2","title":"Etna Volcano","categories":[{"title":"Volcanoes"}],"geometry":[
				{"date":"2024-06-15T12:00:00Z","coordinates":[15.0,37.75]}
			]},
			{"id":"e3","title":"Bad Volcano","categories":[{"title":"Volcanoes"}],"geometry":[
				{"date":"2024-06-15T12:00:00Z","coordinates":[999,95]}
			]}
		]
	}`)
	out, err := parseEONETObservations(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 observation, got %d (%+v)", len(out), out)
	}
	if out[0].SourceID != "nasa-eonet-volcano" {
		t.Fatalf("volcano source id: %q", out[0].SourceID)
	}
	if out[0].Latitude != 37.75 || out[0].Longitude != 15.0 {
		t.Fatalf("coords: lat=%v lon=%v", out[0].Latitude, out[0].Longitude)
	}
	if out[0].Latitude == 0 && out[0].Longitude == 0 {
		t.Fatal("must not invent 0,0")
	}
	if !strings.Contains(out[0].RawText, "[volcano erupting]") {
		t.Fatalf("volcano tag missing: %q", out[0].RawText)
	}
}

func TestParseEONETObservationsRejectsEmptyBodyEvents(t *testing.T) {
	out, err := parseEONETObservations([]byte(`{"events":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("empty events must yield empty obs, got %d", len(out))
	}
}
