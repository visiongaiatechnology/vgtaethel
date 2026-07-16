package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"go-aethel/intelligence"
)

const (
	CollectorTypeEarthquakeGeoJSON = "earthquake-geojson"
	CollectorTypeVolcanoEONET      = "volcano-eonet"
	maxHazardPayloadBytes          = 4 << 20
)

type HazardJSONCollector struct {
	cfg    OSINTCollectorConfig
	client *http.Client
}

func NewHazardJSONCollector(cfg OSINTCollectorConfig) *HazardJSONCollector {
	return &HazardJSONCollector{cfg: cfg, client: newSafeCollectorHTTPClient()}
}

func (c *HazardJSONCollector) Name() string                     { return c.cfg.Name }
func (c *HazardJSONCollector) Domain() intelligence.OSINTDomain { return intelligence.DomainGeo }

func (c *HazardJSONCollector) Collect(ctx context.Context) ([]intelligence.OSINTEvent, error) {
	if err := validatePublicCollectorURL(c.cfg.URL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "VGT-AETHEL-OSINT/1.0 (Hazard Intelligence Platform)")
	req.Header.Set("Accept", "application/geo+json, application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch hazard source: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("hazard source returned HTTP %d", resp.StatusCode)
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
	if mediaType != "application/json" && mediaType != "application/geo+json" && !strings.HasSuffix(mediaType, "+json") {
		return nil, errors.New("hazard source did not return JSON")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHazardPayloadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxHazardPayloadBytes {
		return nil, errors.New("hazard source payload exceeds 4 MiB")
	}
	switch c.cfg.Type {
	case CollectorTypeEarthquakeGeoJSON:
		return c.parseEarthquakeGeoJSON(body)
	case CollectorTypeVolcanoEONET:
		return c.parseVolcanoEONET(body)
	default:
		return nil, errors.New("unsupported hazard collector type")
	}
}

func validHazardCoordinates(lat, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lon) && !math.IsInf(lat, 0) && !math.IsInf(lon, 0) && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func boundedHazardText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}

func (c *HazardJSONCollector) parseEarthquakeGeoJSON(body []byte) ([]intelligence.OSINTEvent, error) {
	var feed struct {
		Features []struct {
			ID         string `json:"id"`
			Properties struct {
				Magnitude float64 `json:"mag"`
				Place     string  `json:"place"`
				Title     string  `json:"title"`
				URL       string  `json:"url"`
				Time      int64   `json:"time"`
			} `json:"properties"`
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	if err := decoder.Decode(&feed); err != nil {
		return nil, errors.New("invalid earthquake GeoJSON")
	}
	events := make([]intelligence.OSINTEvent, 0, min(len(feed.Features), 100))
	for _, feature := range feed.Features {
		if len(events) >= 100 || len(feature.Geometry.Coordinates) < 2 {
			break
		}
		lon, lat := feature.Geometry.Coordinates[0], feature.Geometry.Coordinates[1]
		if !validHazardCoordinates(lat, lon) || feature.Properties.Magnitude < -2 || feature.Properties.Magnitude > 12 {
			continue
		}
		timestamp := time.Now().UTC()
		if feature.Properties.Time > 0 {
			timestamp = time.UnixMilli(feature.Properties.Time).UTC()
		}
		place := boundedHazardText(feature.Properties.Place, 240)
		title := boundedHazardText(feature.Properties.Title, 320)
		if title == "" {
			title = fmt.Sprintf("[earthquake] M %.1f - %s", feature.Properties.Magnitude, place)
		} else {
			title = "[earthquake] " + title
		}
		id := boundedHazardText(feature.ID, 160)
		if id == "" {
			id = hashID(c.cfg.Name + title + timestamp.Format(time.RFC3339Nano))
		}
		events = append(events, intelligence.OSINTEvent{
			ID: id, Title: title, Summary: fmt.Sprintf("Magnitude %.1f · %s", feature.Properties.Magnitude, place),
			Source: c.cfg.Name, SourceURL: c.cfg.URL, URL: boundedHazardText(feature.Properties.URL, 1024),
			Domain: intelligence.DomainGeo, Timestamp: timestamp, Confidence: 0.9, Status: "raw", Lat: lat, Lon: lon, HasGeo: true,
		})
	}
	return events, nil
}

func (c *HazardJSONCollector) parseVolcanoEONET(body []byte) ([]intelligence.OSINTEvent, error) {
	var feed struct {
		Events []struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Link     string `json:"link"`
			Geometry []struct {
				Date        string    `json:"date"`
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"events"`
	}
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, errors.New("invalid volcano EONET JSON")
	}
	events := make([]intelligence.OSINTEvent, 0, min(len(feed.Events), 100))
	for _, sourceEvent := range feed.Events {
		if len(events) >= 100 || len(sourceEvent.Geometry) == 0 {
			continue
		}
		geometry := sourceEvent.Geometry[len(sourceEvent.Geometry)-1]
		if len(geometry.Coordinates) < 2 {
			continue
		}
		lon, lat := geometry.Coordinates[0], geometry.Coordinates[1]
		if !validHazardCoordinates(lat, lon) {
			continue
		}
		timestamp := time.Now().UTC()
		if parsed, err := time.Parse(time.RFC3339, geometry.Date); err == nil {
			timestamp = parsed.UTC()
		}
		title := boundedHazardText(sourceEvent.Title, 300)
		if title == "" {
			title = "Unbenanntes Vulkanereignis"
		}
		id := boundedHazardText(sourceEvent.ID, 160)
		if id == "" {
			id = hashID(c.cfg.Name + title + timestamp.Format(time.RFC3339Nano))
		}
		events = append(events, intelligence.OSINTEvent{
			ID: id, Title: "[volcano erupting] " + title, Summary: "Aktives Vulkanereignis aus konfigurierter EONET-kompatibler Quelle.",
			Source: c.cfg.Name, SourceURL: c.cfg.URL, URL: boundedHazardText(sourceEvent.Link, 1024),
			Domain: intelligence.DomainGeo, Timestamp: timestamp, Confidence: 0.85, Status: "raw", Lat: lat, Lon: lon, HasGeo: true,
		})
	}
	return events, nil
}
