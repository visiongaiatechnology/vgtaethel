package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-aethel/intelligence"
	"golang.org/x/net/html"
)

const maxWebIndexBytes = 3 << 20

// WebIndexCollector supports official press pages and RSS directory pages.
// It prefers advertised RSS/Atom endpoints and otherwise emits bounded,
// same-origin headline links as raw evidence for the SHADOW batch pipeline.
type WebIndexCollector struct {
	cfg    OSINTCollectorConfig
	client *http.Client
}

func NewWebIndexCollector(cfg OSINTCollectorConfig) *WebIndexCollector {
	return &WebIndexCollector{cfg: cfg, client: newSafeCollectorHTTPClient()}
}

func (c *WebIndexCollector) Name() string                     { return c.cfg.Name }
func (c *WebIndexCollector) Domain() intelligence.OSINTDomain { return c.cfg.Domain }

func (c *WebIndexCollector) Collect(ctx context.Context) ([]intelligence.OSINTEvent, error) {
	if err := ValidatePublicCollectorURL(c.cfg.URL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.URL, nil)
	if err != nil {
		return nil, errors.New("web index request invalid")
	}
	req.Header.Set("User-Agent", "VGT-AETHEL-SHADOW/1.0 (public index collector)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.New("web index unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, errors.New("web index rejected collection")
	}
	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || (mediaType != "text/html" && mediaType != "application/xhtml+xml") {
		return nil, errors.New("web index did not return HTML")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxWebIndexBytes+1))
	if err != nil || len(body) > maxWebIndexBytes {
		return nil, errors.New("web index response exceeds boundary")
	}
	document, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.New("web index HTML invalid")
	}
	base := resp.Request.URL
	feedURLs, events := c.parseDocument(document, base)
	if len(feedURLs) == 0 {
		return events, nil
	}
	merged := make([]intelligence.OSINTEvent, 0, 20)
	seen := map[string]bool{}
	for _, feedURL := range feedURLs {
		feedCfg := c.cfg
		feedCfg.URL = feedURL
		feedEvents, collectErr := NewRSSCollector(feedCfg).Collect(ctx)
		if collectErr != nil {
			continue
		}
		for _, event := range feedEvents {
			if !seen[event.ID] {
				seen[event.ID] = true
				merged = append(merged, event)
			}
			if len(merged) == 20 {
				return merged, nil
			}
		}
	}
	if len(merged) > 0 {
		return merged, nil
	}
	return events, nil
}

func (c *WebIndexCollector) parseDocument(document *html.Node, base *url.URL) ([]string, []intelligence.OSINTEvent) {
	feeds := make([]string, 0, 3)
	events := make([]intelligence.OSINTEvent, 0, 20)
	seenFeeds := map[string]bool{}
	seenLinks := map[string]bool{}
	var walk func(*html.Node, bool)
	walk = func(node *html.Node, blocked bool) {
		if len(events) >= 20 && len(feeds) >= 3 {
			return
		}
		if node.Type == html.ElementNode {
			tag := strings.ToLower(node.Data)
			if tag == "script" || tag == "style" || tag == "noscript" || tag == "svg" || tag == "footer" || tag == "form" {
				blocked = true
			}
			if !blocked && tag == "link" && len(feeds) < 3 {
				rel := strings.ToLower(htmlAttribute(node, "rel"))
				typeName := strings.ToLower(htmlAttribute(node, "type"))
				if strings.Contains(rel, "alternate") && (strings.Contains(typeName, "rss") || strings.Contains(typeName, "atom")) {
					if candidate := resolvePublicWebURL(base, htmlAttribute(node, "href")); candidate != "" && !seenFeeds[candidate] {
						seenFeeds[candidate] = true
						feeds = append(feeds, candidate)
					}
				}
			}
			if !blocked && tag == "a" && len(events) < 20 {
				title := normalizeArticleText(nodeText(node))
				candidate := resolvePublicWebURL(base, htmlAttribute(node, "href"))
				if len([]rune(title)) >= 20 && len([]rune(title)) <= 220 && candidate != "" && !seenLinks[candidate] && isLikelyNewsLink(base, candidate) {
					seenLinks[candidate] = true
					event := intelligence.OSINTEvent{
						ID: hashID("web-index:" + candidate), Title: title, URL: candidate,
						Source: c.cfg.Name, SourceURL: c.cfg.URL, Domain: c.cfg.Domain,
						Timestamp: time.Now().UTC(), Confidence: 0.45, Status: "raw",
					}
					enrichEventGeo(&event)
					events = append(events, event)
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child, blocked)
		}
	}
	walk(document, false)
	return feeds, events
}

func resolvePublicWebURL(base *url.URL, raw string) string {
	if base == nil {
		return ""
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(parsed)
	resolved.Fragment = ""
	if err := ValidatePublicCollectorURL(resolved.String()); err != nil {
		return ""
	}
	return resolved.String()
}

func isLikelyNewsLink(base *url.URL, candidate string) bool {
	parsed, err := url.Parse(candidate)
	if err != nil || base == nil || !strings.EqualFold(parsed.Hostname(), base.Hostname()) {
		return false
	}
	path := strings.ToLower(parsed.EscapedPath())
	for _, rejected := range []string{"/login", "/account", "/privacy", "/contact", "/about", "/terms", "/tag/", "/category/", "/author/"} {
		if strings.Contains(path, rejected) {
			return false
		}
	}
	return path != "" && path != "/"
}
