package osint

// Additional BuiltIn connectors for the intelligence OS (U6).
// Each produces Observations only — never verified facts.
// Fetch uses existing OSINT cache / safe HTTPS collectors. No WWV/WM code.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-aethel/intelligence"
	"go-aethel/intelligence/connectors"
)

// domainRSSConnector filters OSINT engine events by domain into Observations.
type domainRSSConnector struct {
	name   string
	domain string // empty = all curated news (alias of builtin-rss family)
	mu     sync.Mutex
	last   time.Time
}

func (c *domainRSSConnector) Descriptor() connectors.Descriptor {
	return connectors.Descriptor{
		Name:            c.name,
		Version:         "1.0.0",
		SourceTypes:     []string{"rss", "atom"},
		Permissions:     []string{"network.fetch.public"},
		PollingInterval: 15 * time.Minute,
		RateLimitPerMin: 40,
		LicenseInfo:     "in-tree AETHEL domain slice over OSINT collectors",
		TrustTier:      connectors.TrustBuiltIn,
		Activated:       true,
	}
}

func (c *domainRSSConnector) HealthCheck() error {
	if state == nil || state.osint == nil {
		return errors.New("osint engine not started")
	}
	return nil
}

func (c *domainRSSConnector) Fetch() ([]intelligence.Observation, error) {
	if err := c.HealthCheck(); err != nil {
		return nil, err
	}
	d := c.Descriptor()
	c.mu.Lock()
	doRefresh := c.last.IsZero() || time.Since(c.last) >= d.PollingInterval
	if doRefresh {
		c.last = time.Now().UTC()
	}
	c.mu.Unlock()
	if doRefresh {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		_ = state.osint.RefreshNow(ctx)
		cancel()
	}
	domain := c.domain
	if domain == "" {
		domain = "all"
	}
	events := state.osint.GetEvents(domain)
	limit := d.RateLimitPerMin
	out := make([]intelligence.Observation, 0, minInt(limit, len(events)))
	for i, ev := range events {
		if i >= limit {
			break
		}
		src := strings.TrimSpace(ev.Source)
		if src == "" {
			src = c.name
		}
		raw := strings.TrimSpace(ev.Title + " " + ev.Summary)
		if raw == "" {
			continue
		}
		id := strings.TrimSpace(ev.ID)
		if id == "" {
			id = fmt.Sprintf("%s-%d", c.name, i)
		}
		out = append(out, intelligence.Observation{
			ID: "conn-" + c.name + "-" + id, SourceID: "rss-" + strings.ReplaceAll(src, " ", "_"),
			RawText: raw, ObservedAt: ev.Timestamp, Latitude: ev.Lat, Longitude: ev.Lon, Domain: string(ev.Domain),
		})
	}
	return out, nil
}

// usgsConnector pulls the public USGS all-day GeoJSON feed into Observations.
type usgsConnector struct {
	mu   sync.Mutex
	last time.Time
}

func (c *usgsConnector) Descriptor() connectors.Descriptor {
	return connectors.Descriptor{
		Name:            "builtin-usgs",
		Version:         "1.0.0",
		SourceTypes:     []string{"geojson", "api"},
		Permissions:     []string{"network.fetch.public"},
		PollingInterval: 20 * time.Minute,
		RateLimitPerMin: 50,
		Regions:         []string{"global"},
		LicenseInfo:     "USGS public earthquake feed; AETHEL local ingest only",
		TrustTier:      connectors.TrustBuiltIn,
		Activated:       true,
	}
}

func (c *usgsConnector) HealthCheck() error {
	// No hard dependency on osint; network check is lazy at Fetch.
	return nil
}

func (c *usgsConnector) Fetch() ([]intelligence.Observation, error) {
	c.mu.Lock()
	if !c.last.IsZero() && time.Since(c.last) < 2*time.Minute {
		c.mu.Unlock()
		// Cooldown: avoid hammering USGS; return empty rather than inventing data
		return []intelligence.Observation{}, nil
	}
	c.last = time.Now().UTC()
	c.mu.Unlock()

	const url = "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_day.geojson"
	if err := validatePublicCollectorURL(url); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "VGT-AETHEL-OSINT/1.0 (USGS connector)")
	req.Header.Set("Accept", "application/json")
	client := newSafeCollectorHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("usgs http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Features []struct {
			ID         string `json:"id"`
			Properties struct {
				Title string  `json:"title"`
				URL   string  `json:"url"`
				Mag   float64 `json:"mag"`
				Time  int64   `json:"time"`
			} `json:"properties"`
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	out := make([]intelligence.Observation, 0, 40)
	for i, f := range payload.Features {
		if i >= 40 {
			break
		}
		if len(f.Geometry.Coordinates) < 2 || strings.TrimSpace(f.Properties.Title) == "" {
			continue
		}
		lon := f.Geometry.Coordinates[0]
		lat := f.Geometry.Coordinates[1]
		// Tagged for frontend hazard layer (magnitude bands on globe)
		raw := fmt.Sprintf("[earthquake] M %.1f | %s | magnitude %.1f | %s",
			f.Properties.Mag, f.Properties.Title, f.Properties.Mag, f.Properties.URL)
		out = append(out, intelligence.Observation{
			ID: "usgs-" + f.ID, SourceID: "usgs-earthquakes", RawText: raw,
			ObservedAt: time.UnixMilli(f.Properties.Time).UTC(),
			Latitude: lat, Longitude: lon, Domain: "geo",
		})
	}
	return out, nil
}

// sharedStoreConnector re-emits recent intelligence.SharedIntelStore events as Observations (local loopback / reprocess).
type sharedStoreConnector struct{}

func (c *sharedStoreConnector) Descriptor() connectors.Descriptor {
	return connectors.Descriptor{
		Name:            "builtin-shared-replay",
		Version:         "1.0.0",
		SourceTypes:     []string{"local"},
		Permissions:     []string{"local.read"},
		PollingInterval: 60 * time.Minute,
		RateLimitPerMin: 20,
		LicenseInfo:     "local intelligence.SharedIntelStore replay; no network",
		TrustTier:      connectors.TrustBuiltIn,
		Activated:       true,
	}
}

func (c *sharedStoreConnector) HealthCheck() error {
	if intelligence.SharedIntelStore == nil {
		return errors.New("intelligence.SharedIntelStore unavailable")
	}
	return nil
}

func (c *sharedStoreConnector) Fetch() ([]intelligence.Observation, error) {
	if err := c.HealthCheck(); err != nil {
		return nil, err
	}
	// Replay is a no-op for ingest (would duplicate); expose empty to keep honest.
	// Operators use LiveNexus / events API for existing data.
	return []intelligence.Observation{}, nil
}

// eonetConnector pulls open natural events from NASA EONET (geo disasters / wildfires / storms).
// Observations only — never verified facts. Public API, no API key.
type eonetConnector struct {
	mu   sync.Mutex
	last time.Time
}

func (c *eonetConnector) Descriptor() connectors.Descriptor {
	return connectors.Descriptor{
		Name:            "builtin-eonet",
		Version:         "1.0.0",
		SourceTypes:     []string{"json", "api"},
		Permissions:     []string{"network.fetch.public"},
		PollingInterval: 30 * time.Minute,
		RateLimitPerMin: 40,
		Regions:         []string{"global"},
		LicenseInfo:     "NASA EONET public API; AETHEL local ingest only",
		TrustTier:      connectors.TrustBuiltIn,
		Activated:       true,
	}
}

func (c *eonetConnector) HealthCheck() error { return nil }

func (c *eonetConnector) Fetch() ([]intelligence.Observation, error) {
	c.mu.Lock()
	if !c.last.IsZero() && time.Since(c.last) < 3*time.Minute {
		c.mu.Unlock()
		return []intelligence.Observation{}, nil
	}
	c.last = time.Now().UTC()
	c.mu.Unlock()

	const url = "https://eonet.gsfc.nasa.gov/api/v3/events?status=open&limit=40"
	if err := validatePublicCollectorURL(url); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "VGT-AETHEL-OSINT/1.0 (EONET connector)")
	req.Header.Set("Accept", "application/json")
	client := newSafeCollectorHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eonet http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Events []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Link   string `json:"link"`
			Closed string `json:"closed"`
			Categories []struct {
				Title string `json:"title"`
			} `json:"categories"`
			Geometry []struct {
				Date        string    `json:"date"`
				Type        string    `json:"type"`
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"events"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	out := make([]intelligence.Observation, 0, 40)
	for i, ev := range payload.Events {
		if i >= 40 || strings.TrimSpace(ev.Title) == "" {
			continue
		}
		// Prefer latest geometry point
		var lat, lon float64
		var observed time.Time
		for gi := len(ev.Geometry) - 1; gi >= 0; gi-- {
			g := ev.Geometry[gi]
			if len(g.Coordinates) >= 2 {
				// EONET GeoJSON: [lon, lat]
				lon = g.Coordinates[0]
				lat = g.Coordinates[1]
				if t, err := time.Parse(time.RFC3339, g.Date); err == nil {
					observed = t.UTC()
				}
				break
			}
		}
		if observed.IsZero() {
			observed = time.Now().UTC()
		}
		cat := "natural"
		if len(ev.Categories) > 0 && ev.Categories[0].Title != "" {
			cat = ev.Categories[0].Title
		}
		isVolcano := strings.Contains(strings.ToLower(cat), "volcano") ||
			strings.Contains(strings.ToLower(ev.Title), "volcano") ||
			strings.Contains(strings.ToLower(ev.Title), "eruption")
		// Open EONET volcano events = active/erupting for globe (always red when erupting)
		raw := fmt.Sprintf("%s | category %s | %s", ev.Title, cat, strings.TrimSpace(ev.Link))
		if isVolcano {
			raw = fmt.Sprintf("[volcano erupting] %s | category %s | %s", ev.Title, cat, strings.TrimSpace(ev.Link))
		}
		id := strings.TrimSpace(ev.ID)
		if id == "" {
			id = fmt.Sprintf("eonet-%d", i)
		}
		srcID := "nasa-eonet"
		if isVolcano {
			srcID = "nasa-eonet-volcano"
		}
		out = append(out, intelligence.Observation{
			ID: "eonet-" + id, SourceID: srcID, RawText: raw,
			ObservedAt: observed, Latitude: lat, Longitude: lon, Domain: "geo",
		})
	}
	return out, nil
}

// volcanoConnector is a dedicated eruption feed: NASA EONET open events filtered to volcanoes.
// Erupting / open volcanic events are tagged for always-red globe markers.
type volcanoConnector struct {
	mu   sync.Mutex
	last time.Time
}

func (c *volcanoConnector) Descriptor() connectors.Descriptor {
	return connectors.Descriptor{
		Name:            "builtin-volcano",
		Version:         "1.0.0",
		SourceTypes:     []string{"json", "api"},
		Permissions:     []string{"network.fetch.public"},
		PollingInterval: 30 * time.Minute,
		RateLimitPerMin: 40,
		Regions:         []string{"global"},
		LicenseInfo:     "NASA EONET volcano category; AETHEL local ingest only",
		TrustTier:      connectors.TrustBuiltIn,
		Activated:       true,
	}
}

func (c *volcanoConnector) HealthCheck() error { return nil }

func (c *volcanoConnector) Fetch() ([]intelligence.Observation, error) {
	c.mu.Lock()
	if !c.last.IsZero() && time.Since(c.last) < 3*time.Minute {
		c.mu.Unlock()
		return []intelligence.Observation{}, nil
	}
	c.last = time.Now().UTC()
	c.mu.Unlock()

	// EONET category 12 = Volcanoes (API v3 category filter)
	const url = "https://eonet.gsfc.nasa.gov/api/v3/events?status=open&category=volcanoes&limit=40"
	if err := validatePublicCollectorURL(url); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "VGT-AETHEL-OSINT/1.0 (volcano connector)")
	req.Header.Set("Accept", "application/json")
	client := newSafeCollectorHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("volcano eonet http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var payload struct {
		Events []struct {
			ID         string `json:"id"`
			Title      string `json:"title"`
			Link       string `json:"link"`
			Categories []struct {
				Title string `json:"title"`
			} `json:"categories"`
			Geometry []struct {
				Date        string    `json:"date"`
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"events"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	out := make([]intelligence.Observation, 0, 40)
	for i, ev := range payload.Events {
		if i >= 40 || strings.TrimSpace(ev.Title) == "" {
			continue
		}
		var lat, lon float64
		var observed time.Time
		for gi := len(ev.Geometry) - 1; gi >= 0; gi-- {
			g := ev.Geometry[gi]
			if len(g.Coordinates) >= 2 {
				lon = g.Coordinates[0]
				lat = g.Coordinates[1]
				if t, err := time.Parse(time.RFC3339, g.Date); err == nil {
					observed = t.UTC()
				}
				break
			}
		}
		if observed.IsZero() {
			observed = time.Now().UTC()
		}
		raw := fmt.Sprintf("[volcano erupting] %s | category Volcanoes | %s", ev.Title, strings.TrimSpace(ev.Link))
		id := strings.TrimSpace(ev.ID)
		if id == "" {
			id = fmt.Sprintf("volcano-%d", i)
		}
		out = append(out, intelligence.Observation{
			ID: "volcano-" + id, SourceID: "nasa-eonet-volcano", RawText: raw,
			ObservedAt: observed, Latitude: lat, Longitude: lon, Domain: "geo",
		})
	}
	return out, nil
}

// RegisterAllConnectors registers BuiltIn RSS/domain/USGS/EONET/volcano connectors.
func RegisterAllConnectors() {
	_ = connectors.Register(&builtinRSSConnector{})
	_ = connectors.Register(&domainRSSConnector{name: "builtin-cyber", domain: "cyber"})
	_ = connectors.Register(&domainRSSConnector{name: "builtin-geo", domain: "geo"})
	_ = connectors.Register(&domainRSSConnector{name: "builtin-economic", domain: "economic"})
	_ = connectors.Register(&domainRSSConnector{name: "builtin-humanitarian", domain: "humanitarian"})
	_ = connectors.Register(&usgsConnector{})
	_ = connectors.Register(&eonetConnector{})
	_ = connectors.Register(&volcanoConnector{})
	_ = connectors.Register(&sharedStoreConnector{})
}
