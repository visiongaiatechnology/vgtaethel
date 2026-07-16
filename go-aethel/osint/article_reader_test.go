package osint

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestExtractReadableHTMLExcludesExecutableAndNavigationContent(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<!doctype html><html><head><title>Verified title</title><script>steal()</script></head><body><nav>This navigation sentence must never enter the article output.</nav><main><h1>Primary article heading</h1><p>This is the first substantial paragraph with enough meaningful article content for extraction.</p><p>This is the second substantial paragraph and it remains separated for reader readability.</p></main><footer>Tracking and footer boilerplate must be excluded completely.</footer></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	title, text := extractReadableHTML(doc)
	if title != "Verified title" {
		t.Fatalf("unexpected title %q", title)
	}
	if !strings.Contains(text, "first substantial paragraph") || !strings.Contains(text, "second substantial paragraph") {
		t.Fatalf("article paragraphs missing: %q", text)
	}
	for _, forbidden := range []string{"steal", "navigation sentence", "Tracking and footer"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("blocked content leaked into reader text: %q", forbidden)
		}
	}
}

func TestArticleReaderRejectsLocalTargetsBeforeNetwork(t *testing.T) {
	for _, target := range []string{"http://example.com/article", "https://localhost/article", "https://127.0.0.1/article", "https://user:pass@example.com/article"} {
		if _, err := FetchReadableArticle(context.Background(), target); err == nil {
			t.Fatalf("expected target to be rejected: %s", target)
		}
	}
}
