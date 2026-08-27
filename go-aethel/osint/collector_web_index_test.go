package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"net/url"
	"strings"
	"testing"

	"go-aethel/intelligence"
	"golang.org/x/net/html"
)

func TestWebIndexCollectorDiscoversFeedsAndBoundedSameOriginHeadlines(t *testing.T) {
	document, err := html.Parse(strings.NewReader(`<!doctype html><html><head>
<link rel="alternate" type="application/rss+xml" href="/press/feed.xml">
</head><body><main>
<a href="/news/operational-update-2026">Operational situation update reports material changes</a>
<a href="https://attacker.example/news/fake">Foreign injected headline must not be collected</a>
<a href="/privacy">Privacy statement with deliberately long anchor text</a>
</main></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	base, _ := url.Parse("https://official.example/press")
	collector := NewWebIndexCollector(OSINTCollectorConfig{Name: "Official", URL: base.String(), Domain: intelligence.OSINTDomain("official")})
	feeds, events := collector.parseDocument(document, base)
	if len(feeds) != 1 || feeds[0] != "https://official.example/press/feed.xml" {
		t.Fatalf("unexpected feed discovery: %v", feeds)
	}
	if len(events) != 1 || events[0].URL != "https://official.example/news/operational-update-2026" {
		t.Fatalf("unexpected headline discovery: %+v", events)
	}
}
