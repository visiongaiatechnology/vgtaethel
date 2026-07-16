package osint

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"go-aethel/intelligence"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── Data Models ─────────────────────────────────────────────────────────────

// intelligence.OSINTDomain classifies the nature of an intelligence event

// intelligence.OSINTEvent is the universal on-globe data record

// gazetteerEntry for deterministic longest-match geo resolution (no map iteration randomness).
type gazetteerEntry struct {
	name string
	lat  float64
	lon  float64
}

// gazetteerEntries sorted longest-name-first, deduped. Pure deterministic.
var gazetteerEntries = []gazetteerEntry{
	// longest first
	{"united states", 37.0902, -95.7129},
	{"new york", 40.7128, -74.0060},
	{"hong kong", 22.3193, 114.1694},
	{"buenos aires", -34.6037, -58.3816},
	{"south africa", -30.5595, 22.9375}, // added for completeness
	{"washington", 38.9072, -77.0369},
	{"amsterdam", 52.3676, 4.9041},
	{"stockholm", 59.3293, 18.0686},
	{"istanbul", 41.0136, 28.9550},
	{"canberra", -35.2809, 149.1300},
	{"singapore", 1.3521, 103.8198},
	{"shanghai", 31.2304, 121.4737},
	{"beijing", 39.9042, 116.4074},
	{"brussels", 50.8503, 4.3517},
	{"taiwan", 23.6978, 120.9605},
	{"ukraine", 48.3794, 31.1656},
	{"delhi", 28.7041, 77.1025},
	{"seoul", 37.5665, 126.9780},
	{"taipei", 25.0330, 121.5654},
	{"tokyo", 35.6762, 139.6503},
	{"sydney", -33.8688, 151.2093},
	{"moscow", 55.7558, 37.6173},
	{"berlin", 52.52, 13.405},
	{"london", 51.5074, -0.1278},
	{"paris", 48.8566, 2.3522},
	{"rome", 41.9028, 12.4964},
	{"madrid", 40.4168, -3.7038},
	{"prague", 50.0755, 14.4378},
	{"vienna", 48.2082, 16.3738},
	{"warsaw", 52.2297, 21.0122},
	{"athens", 37.9839, 23.7275},
	{"lisbon", 38.7223, -9.1393},
	{"bangkok", 13.7563, 100.5018},
	{"jakarta", -6.2088, 106.8456},
	{"manila", 14.5995, 120.9842},
	{"pretoria", -25.7479, 28.2293},
	{"cairo", 30.0444, 31.2357},
	{"riyadh", 24.7136, 46.6753},
	{"tehran", 35.6892, 51.3890},
	{"ottawa", 45.4215, -75.6972},
	{"brasilia", -15.8267, -47.9218},
	// countries/regions shorter
	{"germany", 51.1657, 10.4515},
	{"france", 46.2276, 2.2137},
	{"uk", 55.3781, -3.4360},
	{"usa", 37.0902, -95.7129},
	{"russia", 61.5240, 105.3188},
	{"china", 35.8617, 104.1954},
	{"india", 20.5937, 78.9629},
	{"japan", 36.2048, 138.2529},
	{"australia", -25.2744, 133.7751},
	{"brazil", -14.2350, -51.9253},
	{"europe", 54.5260, 15.2551},
}

func init() {
	// Ensure deterministic longest-name-first order for gazetteer (no reliance on source order).
	sort.Slice(gazetteerEntries, func(i, j int) bool {
		return len(gazetteerEntries[i].name) > len(gazetteerEntries[j].name)
	})
}

// looksLikeCoord rejects obvious non-geo numbers (versions, dates, ids) that would produce bogus HasGeo.
func looksLikeCoord(a, b float64) bool {
	if a == 0 && b == 0 {
		return false
	}
	// rough sanity + reject large version-like or year-like
	if math.Abs(a) > 90 || math.Abs(b) > 180 {
		return false
	}
	if a > 100 || b > 1000 { // e.g. v2.1 or 2026
		return false
	}
	return true
}

// matchGazetteer finds the match whose name appears first (leftmost) in the text.
// This makes 'Paris and Berlin' pick Paris, 'Berlin and Paris' pick Berlin. Deterministic.
func matchGazetteer(lower string) (float64, float64, bool) {
	bestIdx := -1
	bestLat, bestLon := 0.0, 0.0
	for _, e := range gazetteerEntries {
		if idx := strings.Index(lower, e.name); idx >= 0 {
			if bestIdx == -1 || idx < bestIdx {
				bestIdx = idx
				bestLat = e.lat
				bestLon = e.lon
			}
		}
	}
	if bestIdx >= 0 {
		return bestLat, bestLon, true
	}
	return 0, 0, false
}

// extractGeoFromText: strict coords only if looksLikeCoord, then deterministic longest gazetteer match.
// No map range, no early return on first weak match. Pure + testable.
func ExtractGeoFromText(text string) (float64, float64, bool) {
	if text == "" {
		return 0, 0, false
	}
	lower := strings.ToLower(text)

	// 1) Explicit numeric patterns. Only accept if looksLikeCoord.
	reDeg := regexp.MustCompile(`(?i)(-?\d{1,3}(?:\.\d+)?)\s*°?\s*([NS])?\s*[,/ ]+\s*(-?\d{1,3}(?:\.\d+)?)\s*°?\s*([EW])?`)
	if m := reDeg.FindStringSubmatch(text); len(m) == 5 {
		lat, _ := strconv.ParseFloat(m[1], 64)
		lon, _ := strconv.ParseFloat(m[3], 64)
		if strings.EqualFold(m[2], "S") {
			lat = -lat
		}
		if strings.EqualFold(m[4], "W") {
			lon = -lon
		}
		if looksLikeCoord(lat, lon) {
			return lat, lon, true
		}
	}

	re := regexp.MustCompile(`(?i)(?:lat[:=\s]*|coordinates?[:=\s]*|position[:=\s]*|lat/lon[:=\s]*)?(-?\d{1,3}(?:\.\d+)?)\s*[,/ ]\s*(?:lon[:=\s]*)?(-?\d{1,3}(?:\.\d+)?)`)
	m := re.FindStringSubmatch(text)
	if len(m) == 3 {
		lat, err1 := strconv.ParseFloat(m[1], 64)
		lon, err2 := strconv.ParseFloat(m[2], 64)
		if err1 == nil && err2 == nil && looksLikeCoord(lat, lon) {
			return lat, lon, true
		}
	}

	// swapped detection, still guarded
	re2 := regexp.MustCompile(`(?i)(-?\d{1,3}(?:\.\d+)?)\s*[,/ ]\s*(-?\d{1,3}(?:\.\d+)?)`)
	m2 := re2.FindStringSubmatch(text)
	if len(m2) == 3 {
		a, _ := strconv.ParseFloat(m2[1], 64)
		b, _ := strconv.ParseFloat(m2[2], 64)
		if looksLikeCoord(a, b) {
			return a, b, true
		}
		if looksLikeCoord(b, a) {
			return b, a, true
		}
	}

	// 2) Deterministic gazetteer (longest first)
	if lat, lon, ok := matchGazetteer(lower); ok {
		return lat, lon, true
	}
	return 0, 0, false
}

// enrichEventGeo is THE single contract: called ONLY from parse time.
// Keeps JS thin (one isGeoEvent predicate).
func enrichEventGeo(ev *intelligence.OSINTEvent) {
	if ev == nil || ev.HasGeo {
		return
	}
	if lat, lon, ok := ExtractGeoFromText(ev.Title + " " + ev.Summary + " " + ev.URL); ok {
		ev.Lat = lat
		ev.Lon = lon
		ev.HasGeo = true
	}
}

// OSINTCollectorConfig is the persisted configuration for a feed
type OSINTCollectorConfig struct {
	Name     string                   `json:"name"`
	Type     string                   `json:"type"` // rss, earthquake-geojson, volcano-eonet
	URL      string                   `json:"url"`
	Domain   intelligence.OSINTDomain `json:"domain"`
	Enabled  bool                     `json:"enabled"`
	Priority int                      `json:"priority"` // 1=low, 5=high
}

// ─── FeedCollector Interface ──────────────────────────────────────────────────

// FeedCollector is the plugin interface for data sources
type FeedCollector interface {
	Name() string
	Domain() intelligence.OSINTDomain
	Collect(ctx context.Context) ([]intelligence.OSINTEvent, error)
}

// ─── RSS Collector ────────────────────────────────────────────────────────────

// RSSFeed is a minimal RSS/Atom parser struct
type RSSFeed struct {
	XMLName xml.Name    `xml:"rss"`
	Channel RSSChannel  `xml:"channel"`
	Entries []AtomEntry `xml:"entry"` // Atom support
}

type RSSChannel struct {
	Title string    `xml:"title"`
	Items []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Date        string `xml:"date"`
	GUID        string `xml:"guid"`
}

type AtomEntry struct {
	Title     string   `xml:"title"`
	Link      AtomLink `xml:"link"`
	Summary   string   `xml:"summary"`
	Updated   string   `xml:"updated"`
	Published string   `xml:"published"`
	ID        string   `xml:"id"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
}

// RSSCollector fetches events from an RSS or Atom feed
type RSSCollector struct {
	cfg    OSINTCollectorConfig
	client *http.Client
}

func NewRSSCollector(cfg OSINTCollectorConfig) *RSSCollector {
	return &RSSCollector{
		cfg:    cfg,
		client: newSafeCollectorHTTPClient(),
	}
}

// validatePublicCollectorURL rejects non-HTTPS, credentialed and local targets.
// DNS targets are checked again in DialContext to prevent DNS rebinding from
// turning a previously benign hostname into an internal request.
// ValidatePublicCollectorURL rejects non-HTTPS, private, credentialed, or non-443 collector URLs.
func ValidatePublicCollectorURL(raw string) error {
	return validatePublicCollectorURL(raw)
}

func validatePublicCollectorURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("collector URL must be a public HTTPS URL without credentials")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return errors.New("collector URL port is not permitted")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return errors.New("collector URL host is not permitted")
	}
	if ip := net.ParseIP(host); ip != nil && isBlockedCollectorIP(ip) {
		return errors.New("collector URL address is not permitted")
	}
	return nil
}
func isBlockedCollectorIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}
func newSafeCollectorHTTPClient() *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{Timeout: 8 * time.Second, KeepAlive: 20 * time.Second}
	base.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errors.New("collector network target invalid")
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil || len(ips) == 0 {
			return nil, errors.New("collector host resolution failed")
		}
		var permitted net.IP
		for _, resolved := range ips {
			if isBlockedCollectorIP(resolved.IP) {
				return nil, errors.New("collector resolved to a blocked address")
			}
			if permitted == nil {
				permitted = append(net.IP(nil), resolved.IP...)
			}
		}
		if permitted == nil {
			return nil, errors.New("collector has no permitted network address")
		}
		// Dial the exact address that passed validation. Re-resolving the host
		// here would reopen a DNS-rebinding window between check and connect.
		_, port, _ := net.SplitHostPort(address)
		return dialer.DialContext(ctx, network, net.JoinHostPort(permitted.String(), port))
	}
	return &http.Client{Timeout: 15 * time.Second, Transport: base, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return errors.New("collector redirect limit reached")
		}
		return validatePublicCollectorURL(req.URL.String())
	}}
}

func (c *RSSCollector) Name() string                     { return c.cfg.Name }
func (c *RSSCollector) Domain() intelligence.OSINTDomain { return c.cfg.Domain }

func (c *RSSCollector) Collect(ctx context.Context) ([]intelligence.OSINTEvent, error) {
	if err := validatePublicCollectorURL(c.cfg.URL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", c.cfg.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "VGT-AETHEL-OSINT/1.0 (Intelligence Platform)")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, application/atom+xml")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", c.cfg.URL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("collector returned status %d", resp.StatusCode)
	}

	const maxFeedBytes = 2 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxFeedBytes {
		return nil, errors.New("collector response exceeds size boundary")
	}

	var events []intelligence.OSINTEvent
	bodyStr := string(body)

	// Detect Atom vs RSS
	if strings.Contains(bodyStr, "<feed") || strings.Contains(bodyStr, "xmlns=\"http://www.w3.org/2005/Atom\"") {
		events = c.parseAtom(body)
	} else {
		events = c.ParseRSS(body)
	}

	return events, nil
}

func (c *RSSCollector) ParseRSS(body []byte) []intelligence.OSINTEvent {
	var feed RSSFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		// Try with workaround for namespace issues
		cleaned := strings.ReplaceAll(string(body), " xmlns:", " xmlnsX:")
		_ = xml.Unmarshal([]byte(cleaned), &feed)
	}

	var events []intelligence.OSINTEvent
	for i, item := range feed.Channel.Items {
		if i >= 20 { // limit per feed
			break
		}
		if item.Title == "" {
			continue
		}
		ts := parseRSSDate(firstNonEmptyRSSDate(item.PubDate, item.Date))
		if ts.IsZero() {
			continue
		}
		id := hashID(c.cfg.Name + item.Link + item.Title)
		ev := intelligence.OSINTEvent{
			ID:         id,
			Title:      item.Title,
			Summary:    stripHTML(item.Description),
			Source:     feed.Channel.Title,
			SourceURL:  c.cfg.URL,
			Domain:     c.cfg.Domain,
			Timestamp:  ts,
			Confidence: 0.75,
			Status:     "raw",
			URL:        item.Link,
		}
		enrichEventGeo(&ev)
		events = append(events, ev)
	}
	return events
}

func (c *RSSCollector) parseAtom(body []byte) []intelligence.OSINTEvent {
	type atomFeed struct {
		XMLName xml.Name    `xml:"feed"`
		Title   string      `xml:"title"`
		Entries []AtomEntry `xml:"entry"`
	}
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		cleaned := strings.ReplaceAll(string(body), " xmlns:", " xmlnsX:")
		_ = xml.Unmarshal([]byte(cleaned), &feed)
	}

	var events []intelligence.OSINTEvent
	for i, entry := range feed.Entries {
		if i >= 20 {
			break
		}
		if entry.Title == "" {
			continue
		}
		ts := parseRSSDate(firstNonEmptyRSSDate(entry.Updated, entry.Published))
		if ts.IsZero() {
			continue
		}
		url := entry.Link.Href
		if url == "" {
			url = entry.ID
		}
		id := hashID(c.cfg.Name + url + entry.Title)
		ev := intelligence.OSINTEvent{
			ID:         id,
			Title:      entry.Title,
			Summary:    stripHTML(entry.Summary),
			Source:     feed.Title,
			SourceURL:  c.cfg.URL,
			Domain:     c.cfg.Domain,
			Timestamp:  ts,
			Confidence: 0.75,
			Status:     "raw",
			URL:        url,
		}
		enrichEventGeo(&ev)
		events = append(events, ev)
	}
	return events
}

// ─── OSINT Engine ─────────────────────────────────────────────────────────────

// OSINTEngine manages all collectors and the event cache
type OSINTEngine struct {
	mu              sync.RWMutex
	collectors      []FeedCollector
	configs         []OSINTCollectorConfig
	events          []intelligence.OSINTEvent
	lastRefresh     time.Time
	configPath      string
	refreshInterval time.Duration
	cancel          context.CancelFunc
	onRefresh       func([]intelligence.OSINTEvent)
}

// SetRefreshHook attaches the product-level bridge without coupling the
// collectors to storage, memory, UI, or provider implementations.
func (e *OSINTEngine) SetRefreshHook(hook func([]intelligence.OSINTEvent)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onRefresh = hook
}

// NewOSINTEngine creates a new engine
func NewOSINTEngine(configPath string) *OSINTEngine {
	return &OSINTEngine{
		configPath:      configPath,
		refreshInterval: 5 * time.Minute,
	}
}

// Start loads configuration, initializes collectors, and begins background refresh
func (e *OSINTEngine) Start() {
	if err := e.loadConfigs(); err != nil {
		log.Printf("⚠️ OSINT: Keine Feeds konfiguriert (%v), lade Defaults", err)
		e.configs = defaultFeeds()
		_ = e.saveConfigs()
	}
	e.buildCollectors()

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	// Initial fetch in background
	go func() {
		if err := e.refresh(ctx); err != nil {
			log.Printf("⚠️ OSINT initial refresh: %v", err)
		}
	}()

	// Background refresh loop
	go func() {
		ticker := time.NewTicker(e.refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := e.refresh(ctx); err != nil {
					log.Printf("⚠️ OSINT refresh: %v", err)
				}
			}
		}
	}()

	log.Printf("✅ OSINT Engine gestartet: %d Collector(s)", len(e.collectors))
}

// Stop halts the engine
func (e *OSINTEngine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
}

// refresh runs all collectors concurrently and merges results
func (e *OSINTEngine) refresh(ctx context.Context) error {
	e.mu.RLock()
	collectors := make([]FeedCollector, len(e.collectors))
	copy(collectors, e.collectors)
	e.mu.RUnlock()

	if len(collectors) == 0 {
		return nil
	}

	type result struct {
		events []intelligence.OSINTEvent
		err    error
	}

	ch := make(chan result, len(collectors))
	for _, col := range collectors {
		col := col
		go func() {
			ctx2, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			evts, err := col.Collect(ctx2)
			if err != nil {
				log.Printf("⚠️ OSINT [%s]: %v", col.Name(), err)
			}
			ch <- result{events: evts, err: err}
		}()
	}

	var allEvents []intelligence.OSINTEvent
	for range collectors {
		r := <-ch
		if r.events != nil {
			allEvents = append(allEvents, r.events...)
		}
	}

	// Deduplicate by ID
	seen := make(map[string]bool)
	var deduped []intelligence.OSINTEvent
	for _, ev := range allEvents {
		if ev.Timestamp.IsZero() || ev.Timestamp.After(time.Now().UTC().Add(10*time.Minute)) {
			continue
		}
		if !seen[ev.ID] {
			seen[ev.ID] = true
			deduped = append(deduped, ev)
		}
	}

	// Sort: newest first
	for i := 0; i < len(deduped)-1; i++ {
		for j := i + 1; j < len(deduped); j++ {
			if deduped[j].Timestamp.After(deduped[i].Timestamp) {
				deduped[i], deduped[j] = deduped[j], deduped[i]
			}
		}
	}

	// Keep newest 200
	if len(deduped) > 200 {
		deduped = deduped[:200]
	}

	e.mu.Lock()
	e.events = deduped
	e.lastRefresh = time.Now()
	hook := e.onRefresh
	e.mu.Unlock()
	if hook != nil {
		copyEvents := append([]intelligence.OSINTEvent(nil), deduped...)
		hook(copyEvents)
	}

	log.Printf("✅ OSINT refresh: %d Events von %d Collector(s)", len(deduped), len(collectors))
	return nil
}

// GetEvents returns all current events, optionally filtered by domain
func (e *OSINTEngine) GetEvents(domain string) []intelligence.OSINTEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if domain == "" || domain == "all" {
		result := make([]intelligence.OSINTEvent, len(e.events))
		copy(result, e.events)
		return result
	}
	var filtered []intelligence.OSINTEvent
	for _, ev := range e.events {
		if string(ev.Domain) == domain {
			filtered = append(filtered, ev)
		}
	}
	return filtered
}

// GetEventsWithin returns only events whose source timestamp is inside the
// requested window. A zero window intentionally means all retained events.
func (e *OSINTEngine) GetEventsWithin(domain string, hours float64, now time.Time) []intelligence.OSINTEvent {
	events := e.GetEvents(domain)
	if hours <= 0 {
		return events
	}
	if hours > 720 {
		hours = 720
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	cutoff := now.Add(-time.Duration(hours * float64(time.Hour)))
	result := make([]intelligence.OSINTEvent, 0, len(events))
	for _, event := range events {
		if event.Timestamp.IsZero() || event.Timestamp.Before(cutoff) || event.Timestamp.After(now.Add(10*time.Minute)) {
			continue
		}
		result = append(result, event)
	}
	return result
}

// GetLastRefresh returns the time of the last refresh
func (e *OSINTEngine) GetLastRefresh() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastRefresh
}

// GetConfigs returns all collector configs
func (e *OSINTEngine) GetConfigs() []OSINTCollectorConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]OSINTCollectorConfig, len(e.configs))
	copy(result, e.configs)
	return result
}

// AddCollector adds a new RSS collector config and rebuilds
func (e *OSINTEngine) AddCollector(cfg OSINTCollectorConfig) error {
	cfg.Type = strings.ToLower(strings.TrimSpace(cfg.Type))
	if cfg.Type == "" {
		cfg.Type = "rss"
	}
	if cfg.Type != "rss" && cfg.Type != "atom" && cfg.Type != CollectorTypeTelegram && cfg.Type != CollectorTypeEarthquakeGeoJSON && cfg.Type != CollectorTypeVolcanoEONET {
		return errors.New("collector type is not supported")
	}
	if cfg.Type == CollectorTypeEarthquakeGeoJSON || cfg.Type == CollectorTypeVolcanoEONET {
		cfg.Domain = intelligence.DomainGeo
	}
	if cfg.Type == CollectorTypeTelegram {
		canonical, err := normalizeTelegramPublicURL(cfg.URL)
		if err != nil {
			return err
		}
		cfg.URL = canonical
	} else if err := validatePublicCollectorURL(cfg.URL); err != nil {
		return err
	}
	if len([]rune(strings.TrimSpace(cfg.Name))) < 3 || len([]rune(cfg.Name)) > 120 {
		return errors.New("collector name must be between 3 and 120 characters")
	}
	e.mu.Lock()
	// Check for duplicate name
	for _, existing := range e.configs {
		if existing.Name == cfg.Name {
			e.mu.Unlock()
			return fmt.Errorf("collector '%s' existiert bereits", cfg.Name)
		}
	}
	cfg.Enabled = true
	e.configs = append(e.configs, cfg)
	e.mu.Unlock()
	e.buildCollectors()
	return e.saveConfigs()
}

// RemoveCollector removes a collector by name
func (e *OSINTEngine) RemoveCollector(name string) error {
	e.mu.Lock()
	var newConfigs []OSINTCollectorConfig
	found := false
	for _, cfg := range e.configs {
		if cfg.Name == name {
			found = true
		} else {
			newConfigs = append(newConfigs, cfg)
		}
	}
	if !found {
		e.mu.Unlock()
		return fmt.Errorf("collector '%s' nicht gefunden", name)
	}
	e.configs = newConfigs
	e.mu.Unlock()
	e.buildCollectors()
	return e.saveConfigs()
}

// RefreshNow triggers an immediate refresh
func (e *OSINTEngine) RefreshNow(ctx context.Context) error {
	return e.refresh(ctx)
}

// buildCollectors rebuilds the collector list from configs
func (e *OSINTEngine) buildCollectors() {
	e.mu.Lock()
	defer e.mu.Unlock()
	var collectors []FeedCollector
	for _, cfg := range e.configs {
		if !cfg.Enabled {
			continue
		}
		if err := validatePublicCollectorURL(cfg.URL); err != nil {
			log.Printf("[OSINT] collector %q disabled: %v", cfg.Name, err)
			continue
		}
		cfg := cfg
		switch cfg.Type {
		case "rss", "atom", "":
			collectors = append(collectors, NewRSSCollector(cfg))
		case CollectorTypeTelegram:
			collectors = append(collectors, NewTelegramCollector(cfg))
		case CollectorTypeEarthquakeGeoJSON, CollectorTypeVolcanoEONET:
			collectors = append(collectors, NewHazardJSONCollector(cfg))
		}
	}
	e.collectors = collectors
}

// loadConfigs reads collector configurations from disk
func (e *OSINTEngine) loadConfigs() error {
	data, err := os.ReadFile(e.configPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &e.configs)
}

// saveConfigs persists collector configurations to disk
func (e *OSINTEngine) saveConfigs() error {
	e.mu.RLock()
	configs := make([]OSINTCollectorConfig, len(e.configs))
	copy(configs, e.configs)
	e.mu.RUnlock()

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(e.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".osint-collectors-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpName, e.configPath)
}

// ─── Default Feed List ────────────────────────────────────────────────────────

func defaultFeeds() []OSINTCollectorConfig {
	return []OSINTCollectorConfig{
		// ── General/Geopolitics ──
		{Name: "Tagesschau Nachrichten", Type: "rss", URL: "https://www.tagesschau.de/xml/rss2", Domain: intelligence.DomainGeneral, Enabled: true, Priority: 5},
		{Name: "Tagesschau Ausland", Type: "rss", URL: "https://www.tagesschau.de/ausland/index~rss2.xml", Domain: intelligence.DomainGeo, Enabled: true, Priority: 5},
		{Name: "Reuters Top News", Type: "rss", URL: "https://feeds.reuters.com/reuters/topNews", Domain: intelligence.DomainGeneral, Enabled: true, Priority: 5},
		{Name: "DER SPIEGEL International", Type: "rss", URL: "https://www.spiegel.de/international/index.rss", Domain: intelligence.DomainGeneral, Enabled: true, Priority: 4},
		{Name: "Euronews World", Type: "rss", URL: "https://feeds.feedburner.com/euronews/en/home/", Domain: intelligence.DomainGeneral, Enabled: true, Priority: 4},
		{Name: "DW World News", Type: "rss", URL: "https://rss.dw.com/rdf/rss-en-world", Domain: intelligence.DomainGeneral, Enabled: true, Priority: 4},
		{Name: "Al Jazeera", Type: "rss", URL: "https://www.aljazeera.com/xml/rss/all.xml", Domain: intelligence.DomainGeo, Enabled: true, Priority: 4},
		// ── Cybersecurity ──
		{Name: "CISA Alerts", Type: "rss", URL: "https://www.cisa.gov/uscert/ncas/alerts.xml", Domain: intelligence.DomainCyber, Enabled: true, Priority: 5},
		{Name: "Heise Security", Type: "rss", URL: "https://www.heise.de/security/news/news-atom.xml", Domain: intelligence.DomainCyber, Enabled: true, Priority: 4},
		{Name: "Krebs on Security", Type: "rss", URL: "https://krebsonsecurity.com/feed/", Domain: intelligence.DomainCyber, Enabled: true, Priority: 4},
		{Name: "The Hacker News", Type: "rss", URL: "https://feeds.feedburner.com/TheHackersNews", Domain: intelligence.DomainCyber, Enabled: true, Priority: 3},
		{Name: "Threatpost", Type: "rss", URL: "https://threatpost.com/feed", Domain: intelligence.DomainCyber, Enabled: true, Priority: 3},
		// ── Open Source Intelligence ──
		{Name: "Bellingcat", Type: "rss", URL: "https://www.bellingcat.com/feed", Domain: intelligence.DomainGeo, Enabled: true, Priority: 5},
		{Name: "Global Voices", Type: "rss", URL: "https://globalvoices.org/feed/", Domain: intelligence.DomainHumanitarian, Enabled: true, Priority: 3},
		// ── Economic ──
		{Name: "Financial Times", Type: "rss", URL: "https://www.ft.com/rss/home/europe", Domain: intelligence.DomainEconomic, Enabled: true, Priority: 4},
		{Name: "Handelsblatt", Type: "rss", URL: "https://www.handelsblatt.com/rss/top100.rss", Domain: intelligence.DomainEconomic, Enabled: true, Priority: 4},
		// ── Humanitarian ──
		{Name: "ReliefWeb", Type: "rss", URL: "https://reliefweb.int/headlines/rss.xml", Domain: intelligence.DomainHumanitarian, Enabled: true, Priority: 4},
		{Name: "UNHCR News", Type: "rss", URL: "https://www.unhcr.org/rss/en/news.xml", Domain: intelligence.DomainHumanitarian, Enabled: true, Priority: 3},
		// ── Tech/Science ──
		{Name: "Heise Online", Type: "rss", URL: "https://www.heise.de/news-atom.xml", Domain: intelligence.DomainGeneral, Enabled: true, Priority: 3},
		{Name: "MIT Technology Review", Type: "rss", URL: "https://www.technologyreview.com/topnews.rss", Domain: intelligence.DomainGeneral, Enabled: true, Priority: 3},
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func hashID(input string) string {
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h[:8])
}

func stripHTML(s string) string {
	// Simple HTML tag stripping
	inTag := false
	var out strings.Builder
	for _, ch := range s {
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' {
			inTag = false
			continue
		}
		if !inTag {
			out.WriteRune(ch)
		}
	}
	result := strings.TrimSpace(out.String())
	// Unescape common entities
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&#39;", "'")
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	// Truncate
	if len(result) > 280 {
		result = result[:277] + "..."
	}
	return result
}

var rssDateFormats = []string{
	time.RFC1123Z, time.RFC1123,
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon, 02 Jan 2006 15:04:05 MST",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05.000Z",
	"2006-01-02",
}

func parseRSSDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, format := range rssDateFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func firstNonEmptyRSSDate(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

var OSINTBriefingPromptFile = "./vgt_workspace/osint_briefing_prompt.txt"

func PersistOSINTBriefingPrompt(path, prompt string) error {
	prompt = strings.TrimSpace(prompt)
	if len([]rune(prompt)) < 20 || len([]rune(prompt)) > 12000 {
		return fmt.Errorf("briefing prompt must be between 20 and 12000 characters")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".briefing-prompt-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0600); err == nil {
		_, err = tmp.WriteString(prompt)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}
