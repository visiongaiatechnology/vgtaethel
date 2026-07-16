package osint

import (
	"strings"
	"testing"
	"time"

	"go-aethel/intelligence"
	"golang.org/x/net/html"
)

func TestNormalizeTelegramPublicURLRejectsNonPublicShapes(t *testing.T) {
	valid, err := normalizeTelegramPublicURL("https://t.me/s/Example_Channel")
	if err != nil || valid != "https://t.me/s/Example_Channel" {
		t.Fatalf("valid Telegram preview rejected: %q %v", valid, err)
	}
	for _, candidate := range []string{
		"http://t.me/s/channel", "https://evil.example/s/channel", "https://t.me/channel",
		"https://t.me/s/a", "https://t.me/s/channel?before=10", "https://user@t.me/s/channel",
	} {
		if _, err := normalizeTelegramPublicURL(candidate); err == nil {
			t.Errorf("unsafe Telegram URL accepted: %s", candidate)
		}
	}
}

func TestTelegramCollectorParsesDatedPublicMessages(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<!doctype html><html><body>
		<div class="tgme_widget_message" data-post="Example_Channel/42">
			<div class="tgme_widget_message_text">Aktuelle Meldung aus Berlin mit belastbarem Zeitstempel.</div>
			<a class="tgme_widget_message_date" href="https://t.me/Example_Channel/42"><time datetime="2026-07-14T08:00:00+00:00"></time></a>
		</div></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	collector := NewTelegramCollector(OSINTCollectorConfig{Name: "Telegram Test", Type: CollectorTypeTelegram, URL: "https://t.me/s/Example_Channel", Domain: intelligence.DomainGeneral})
	events := collector.parseDocument(doc)
	if len(events) != 1 {
		t.Fatalf("expected one Telegram event, got %+v", events)
	}
	if events[0].URL != "https://t.me/Example_Channel/42" || events[0].Timestamp.IsZero() || events[0].Status != "raw" {
		t.Fatalf("Telegram event contract invalid: %+v", events[0])
	}
}

func TestGetEventsWithinUsesStrictSourceTimestamp(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	engine := &OSINTEngine{events: []intelligence.OSINTEvent{
		{ID: "fresh", Domain: intelligence.DomainGeneral, Timestamp: now.Add(-23*time.Hour - 59*time.Minute)},
		{ID: "boundary", Domain: intelligence.DomainGeneral, Timestamp: now.Add(-24 * time.Hour)},
		{ID: "stale", Domain: intelligence.DomainGeneral, Timestamp: now.Add(-24*time.Hour - time.Nanosecond)},
		{ID: "undated", Domain: intelligence.DomainGeneral},
	}}
	events := engine.GetEventsWithin("all", 24, now)
	if len(events) != 2 || events[0].ID != "fresh" || events[1].ID != "boundary" {
		t.Fatalf("strict 24h filter returned %+v", events)
	}
}
