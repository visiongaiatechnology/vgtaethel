package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-aethel/intelligence"
	"go-aethel/intelligence/connectors"
	"golang.org/x/net/html"
)

const (
	ofacActionsURL = "https://ofac.treasury.gov/recent-actions/sanctions-list-updates"
	faaNASURL      = "https://www.fly.faa.gov/ois/oisedit/summary_pub"
	uscgBNMURL     = "https://www.navcen.uscg.gov/broadcast-notice-to-mariners-search-results?date-range=1+month+ago--today&district=9&items_per_page=100&order=field_bnm_message_date&sector=0+13+14+15+16&sort=desc"
)

type officialStatusConnector struct {
	name, sourceID, rawURL, domain, licenseID, termsURL string
	polling                                             time.Duration
}

func (c *officialStatusConnector) Descriptor() connectors.Descriptor {
	return connectors.Descriptor{
		Name: c.name, Version: "1.0.0", SourceTypes: []string{"html", "official-status"},
		Permissions: []string{"network.fetch.public"}, PollingInterval: c.polling, RateLimitPerMin: 2,
		Regions: []string{"global"}, LicenseInfo: c.licenseID, TrustTier: connectors.TrustBuiltIn, Activated: true,
		Policy: connectors.PublicOSINTPolicy(c.licenseID, c.termsURL, 2, time.Minute, 365*24*time.Hour),
	}
}

func (*officialStatusConnector) HealthCheck() error { return nil }
func (c *officialStatusConnector) Fetch() ([]intelligence.Observation, error) {
	body, fetchedAt, err := fetchBoundedPublicDocument(c.rawURL, "text/html,application/xhtml+xml")
	if err != nil {
		return nil, err
	}
	return parseOfficialStatusPage(body, fetchedAt, c.sourceID, c.rawURL, c.domain)
}

func newOFACActionsConnector() connectors.Connector {
	return &officialStatusConnector{name: "builtin-ofac-actions", sourceID: "ofac-sanctions-actions", rawURL: ofacActionsURL, domain: "economic", licenseID: "US-Treasury-OFAC-public-information", termsURL: "https://home.treasury.gov/footer/privacy-policy", polling: 2 * time.Hour}
}
func newFAANASConnector() connectors.Connector {
	return &officialStatusConnector{name: "builtin-faa-nas-status", sourceID: "faa-atcscc", rawURL: faaNASURL, domain: "infrastructure", licenseID: "FAA-public-status", termsURL: "https://www.faa.gov/privacy", polling: 5 * time.Minute}
}
func newUSCGBNMConnector() connectors.Connector {
	return &officialStatusConnector{name: "builtin-uscg-bnm-great-lakes", sourceID: "uscg-navcen-bnm", rawURL: uscgBNMURL, domain: "infrastructure", licenseID: "USCG-NAVCEN-public-notices", termsURL: "https://www.dhs.gov/privacy-policy", polling: 30 * time.Minute}
}

func fetchBoundedPublicDocument(rawURL, accept string) ([]byte, time.Time, error) {
	if err := validatePublicCollectorURL(rawURL); err != nil {
		return nil, time.Time{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, time.Time{}, err
	}
	request.Header.Set("Accept", accept)
	request.Header.Set("User-Agent", "VGT-AETHEL-OSINT/2.1")
	response, err := newSafeCollectorHTTPClient().Do(request)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, time.Time{}, fmt.Errorf("official status endpoint returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if err != nil {
		return nil, time.Time{}, err
	}
	if len(body) > 4<<20 {
		return nil, time.Time{}, errors.New("official status response exceeds size boundary")
	}
	return body, time.Now().UTC(), nil
}

func parseOfficialStatusPage(body []byte, fetchedAt time.Time, sourceID, rawURL, domain string) ([]intelligence.Observation, error) {
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("official status HTML is invalid")
	}
	var builder strings.Builder
	var walk func(*html.Node, bool)
	walk = func(node *html.Node, hidden bool) {
		if node.Type == html.ElementNode && (node.Data == "script" || node.Data == "style" || node.Data == "nav" || node.Data == "footer") {
			hidden = true
		}
		if node.Type == html.TextNode && !hidden && builder.Len() <= 128<<10 {
			builder.WriteString(node.Data)
			builder.WriteByte(' ')
		}
		for child := node.FirstChild; child != nil && builder.Len() <= 128<<10; child = child.NextSibling {
			walk(child, hidden)
		}
	}
	walk(document, false)
	text := strings.Join(strings.Fields(builder.String()), " ")
	if len([]rune(text)) < 20 {
		return nil, errors.New("official status page contains no usable content")
	}
	runes := []rune(text)
	if len(runes) > 32<<10 {
		text = string(runes[:32<<10])
	}
	digest := sha256.Sum256([]byte(text))
	id := sourceID + "-" + hex.EncodeToString(digest[:12])
	return []intelligence.Observation{{
		ID: id, SourceID: sourceID, RawText: text, ObservedAt: fetchedAt, FetchedAt: fetchedAt,
		OriginalURL: rawURL, FinalURL: rawURL, Domain: domain, MIMEType: "text/html", ParserVersion: "official-status-html-v1",
	}}, nil
}
