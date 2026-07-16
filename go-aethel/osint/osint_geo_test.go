package osint

import (
	_ "embed"
	"testing"
	"time"
)

//go:embed testdata/rss_berlin.xml
var rssBerlinXML []byte

func TestExtractGeoFromText_Table(t *testing.T) {
	cases := []struct {
		text    string
		wantLat float64
		wantLon float64
		wantHas bool
	}{
		{"Major event reported in Berlin today near Brandenburg Gate", 52.52, 13.405, true},
		{"Paris and Berlin talks continue", 48.8566, 2.3522, true}, // first mentioned wins
		{"Berlin and Paris talks continue", 52.52, 13.405, true},
		{"v2.1, 2026-07 release notes", 0, 0, false},
		{"Coordinates: 40.7128, -74.0060", 40.7128, -74.0060, true},
		{"Flooding reported near Tokyo station", 35.6762, 139.6503, true},
	}
	for _, c := range cases {
		lat, lon, has := ExtractGeoFromText(c.text)
		if has != c.wantHas || (has && (lat != c.wantLat || lon != c.wantLon)) {
			t.Errorf("extract(%q) = %v,%v,%v want %v,%v,%v", c.text, lat, lon, has, c.wantLat, c.wantLon, c.wantHas)
		}
	}
}

func TestRSSCollectorRejectsUndatedItemsInsteadOfInventingFreshness(t *testing.T) {
	c := &RSSCollector{cfg: OSINTCollectorConfig{Name: "undated-test"}}
	raw := []byte(`<rss><channel><title>Archive</title><item><title>Old item without a date</title><link>https://example.test/old</link></item></channel></rss>`)
	if events := c.ParseRSS(raw); len(events) != 0 {
		t.Fatalf("undated RSS item was incorrectly treated as fresh: %+v", events)
	}
	if parsed := parseRSSDate("not-a-date"); !parsed.Equal(time.Time{}) {
		t.Fatalf("invalid RSS date received invented timestamp: %v", parsed)
	}
}

// Integration: exercise RSSCollector.parseRSS on the real fixture bytes end-to-end to intelligence.OSINTEvent.
func TestRSSCollectorParse_BerlinFixture(t *testing.T) {
	c := &RSSCollector{cfg: OSINTCollectorConfig{Name: "fixture-test"}}
	events := c.ParseRSS(rssBerlinXML) // note: parseRSS is unexported but same package _test.go can call
	if len(events) == 0 {
		t.Fatal("parseRSS on rss_berlin.xml produced no events")
	}
	ev := events[0]
	if ev.Title == "" {
		t.Fatal("parsed event has no title from fixture")
	}
	if !ev.HasGeo || ev.Lat < 52 || ev.Lat > 53 {
		t.Fatalf("fixture event did not get Berlin geo: %+v", ev)
	}
	// also verify it would produce a pin at sensible loc (using shipped math via goja would, here just sanity)
	if ev.Lon < 13 || ev.Lon > 14 {
		t.Errorf("unexpected lon for Berlin: %v", ev.Lon)
	}
}
