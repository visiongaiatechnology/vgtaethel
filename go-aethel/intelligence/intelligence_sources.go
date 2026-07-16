package intelligence

// STATUS: PLATIN
// Curated source adapters: no caller-supplied URL can reach the network path.

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type IntelligenceSource struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Domain          string     `json:"domain"`
	Category        string     `json:"category"`
	Adapter         string     `json:"adapter"`
	URL             string     `json:"-"`
	Enabled         bool       `json:"enabled"`
	LastCollectedAt *time.Time `json:"last_collected_at,omitempty"`
	LastSuccessAt   *time.Time `json:"last_success_at,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	Freshness       string     `json:"freshness"`
}
type IntelligenceSourceRegistry struct {
	mu         sync.RWMutex
	sources    map[string]*IntelligenceSource
	collectors map[string]IntelligenceCollector
	store      *IntelligenceStore
}

// IntelligenceCollector is the only extension boundary for collectors. The
// registry owns network access and uses a fixed descriptor URL, so plugins
// cannot turn model input into an arbitrary outbound request.
type IntelligenceCollector interface {
	Parse(source *IntelligenceSource, body io.Reader) ([]IntelligenceEvent, error)
}
type rssCollector struct{}

func (rssCollector) Parse(source *IntelligenceSource, body io.Reader) ([]IntelligenceEvent, error) {
	items, err := parseCuratedFeed(body)
	if err != nil {
		return nil, err
	}
	out := make([]IntelligenceEvent, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Title) == "" {
			continue
		}
		out = append(out, IntelligenceEvent{Title: truncateIntel(item.Title, 160), Summary: truncateIntel(item.Description, 480), Source: source.Name, SourceURL: truncateIntel(item.Link, 1024), Severity: "low", Confidence: 60})
	}
	return out, nil
}

type usgsCollector struct{}

func (usgsCollector) Parse(source *IntelligenceSource, body io.Reader) ([]IntelligenceEvent, error) {
	return parseUSGSEarthquakes(body, source)
}

func NewIntelligenceSourceRegistry(store *IntelligenceStore) *IntelligenceSourceRegistry {
	sources := map[string]*IntelligenceSource{
		"tagesschau-world": {ID: "tagesschau-world", Name: "Tagesschau Welt", Domain: "tagesschau.de", Category: "news", Adapter: "rss", URL: "https://www.tagesschau.de/xml/rss2/", Enabled: true, Freshness: "never"},
		"dw-world":         {ID: "dw-world", Name: "Deutsche Welle World", Domain: "dw.com", Category: "news", Adapter: "rss", URL: "https://rss.dw.com/rdf/rss-en-all", Enabled: true, Freshness: "never"},
		"usgs-earthquakes": {ID: "usgs-earthquakes", Name: "USGS Earthquake Feed", Domain: "earthquake.usgs.gov", Category: "geo", Adapter: "usgs_geojson", URL: "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_day.geojson", Enabled: true, Freshness: "never"},
		"eonet-volcanoes":  {ID: "eonet-volcanoes", Name: "NASA EONET Volcanoes", Domain: "eonet.gsfc.nasa.gov", Category: "geo", Adapter: "eonet_volcano", URL: "https://eonet.gsfc.nasa.gov/api/v3/events?status=open&category=volcanoes&limit=40", Enabled: true, Freshness: "never"},
	}
	return &IntelligenceSourceRegistry{store: store, sources: sources, collectors: map[string]IntelligenceCollector{"rss": rssCollector{}, "usgs_geojson": usgsCollector{}, "eonet_volcano": eonetVolcanoCollector{}}}
}
func (r *IntelligenceSourceRegistry) List() []IntelligenceSource {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]IntelligenceSource, 0, len(r.sources))
	for _, source := range r.sources {
		out = append(out, *source)
	}
	return out
}
func (r *IntelligenceSourceRegistry) Collect(id string) (int, error) {
	r.mu.Lock()
	source, ok := r.sources[id]
	if !ok {
		r.mu.Unlock()
		return 0, errors.New("source is not registered")
	}
	collector := r.collectors[source.Adapter]
	if !source.Enabled {
		r.mu.Unlock()
		return 0, errors.New("source is disabled")
	}
	if collector == nil {
		r.mu.Unlock()
		return 0, errors.New("source adapter is not registered")
	}
	now := time.Now().UTC()
	source.LastCollectedAt = &now
	r.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return 0, errors.New("source request could not be created")
	}
	req.Header.Set("User-Agent", "Aethel-Intelligence/1.0 (+local-first)")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		r.markError(id, "unreachable")
		return 0, errors.New("source is currently unreachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		r.markError(id, "http status rejected")
		return 0, errors.New("source rejected the collection request")
	}
	body := io.LimitReader(resp.Body, 2<<20)
	events, err := collector.Parse(source, body)
	if err != nil {
		r.markError(id, "feed parsing failed")
		return 0, errors.New("source response is not a supported feed")
	}
	created := 0
	for _, e := range events {
		if err := r.store.ProposeEvent(e); err == nil {
			created++
		}
	}
	r.mu.Lock()
	success := time.Now().UTC()
	source.LastSuccessAt = &success
	source.LastError = ""
	source.Freshness = "fresh"
	r.mu.Unlock()
	return created, nil
}
func parseUSGSEarthquakes(body io.Reader, source *IntelligenceSource) ([]IntelligenceEvent, error) {
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
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]IntelligenceEvent, 0, len(payload.Features))
	for _, feature := range payload.Features {
		if len(feature.Geometry.Coordinates) < 2 || strings.TrimSpace(feature.Properties.Title) == "" {
			continue
		}
		severity := "low"
		if feature.Properties.Mag >= 6 {
			severity = "high"
		} else if feature.Properties.Mag >= 4.5 {
			severity = "medium"
		}
		magStr := strconv.FormatFloat(feature.Properties.Mag, 'f', 1, 64)
		out = append(out, IntelligenceEvent{
			ID: "usgs_" + feature.ID,
			Title: truncateIntel(feature.Properties.Title, 160),
			Summary: "[earthquake] M " + magStr + " | USGS magnitude " + magStr,
			Source: source.Name, SourceURL: feature.Properties.URL,
			Longitude: feature.Geometry.Coordinates[0], Latitude: feature.Geometry.Coordinates[1],
			Severity: severity, Confidence: 90,
			ObservedAt: time.UnixMilli(feature.Properties.Time).UTC(),
		})
	}
	return out, nil
}

// eonetVolcanoCollector parses NASA EONET open volcano events (erupting / active).
type eonetVolcanoCollector struct{}

func (eonetVolcanoCollector) Parse(source *IntelligenceSource, body io.Reader) ([]IntelligenceEvent, error) {
	var payload struct {
		Events []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Link  string `json:"link"`
			Geometry []struct {
				Date        string    `json:"date"`
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"events"`
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]IntelligenceEvent, 0, len(payload.Events))
	for _, ev := range payload.Events {
		if strings.TrimSpace(ev.Title) == "" {
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
		id := strings.TrimSpace(ev.ID)
		if id == "" {
			id = strconv.FormatInt(observed.Unix(), 10)
		}
		out = append(out, IntelligenceEvent{
			ID: "volcano_" + id,
			Title: truncateIntel(ev.Title, 160),
			Summary: "[volcano erupting] " + ev.Title + " | NASA EONET open volcanic event",
			Source: source.Name, SourceURL: strings.TrimSpace(ev.Link),
			Longitude: lon, Latitude: lat,
			Severity: "high", Confidence: 85,
			ObservedAt: observed,
		})
	}
	return out, nil
}
func (r *IntelligenceSourceRegistry) markError(id, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s := r.sources[id]; s != nil {
		s.LastError = message
		s.Freshness = "degraded"
	}
}

type curatedFeed struct {
	Channel struct {
		Items []curatedItem `xml:"item"`
	} `xml:"channel"`
	Entries []curatedItem `xml:"entry"`
}
type curatedItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Summary     string `xml:"summary"`
}

func parseCuratedFeed(body io.Reader) ([]curatedItem, error) {
	var feed curatedFeed
	if err := xml.NewDecoder(body).Decode(&feed); err != nil {
		return nil, err
	}
	items := append(feed.Channel.Items, feed.Entries...)
	if len(items) > 40 {
		items = items[:40]
	}
	for i := range items {
		if items[i].Description == "" {
			items[i].Description = items[i].Summary
		}
	}
	return items, nil
}
// TruncateIntel shortens intel text for UI/context payloads.
func TruncateIntel(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		return value[:max]
	}
	return value
}

// truncateIntel keeps internal call sites working after the export rename.
func truncateIntel(value string, max int) string {
	return TruncateIntel(value, max)
}
