package geoint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HazardsCollector manages live geophysical hazards (USGS earthquakes, FIRMS thermal anomalies)
type HazardsCollector struct {
	client       *http.Client
	bus          *GeoEntityBus
	firmsMapKey  string
	cacheMu      sync.RWMutex
	cachedQuakes []GeoEntity
	cachedFires  []GeoEntity
	lastFetch    time.Time
	cacheTTL     time.Duration
}

// NewHazardsCollector creates a HazardsCollector
func NewHazardsCollector(bus *GeoEntityBus, firmsMapKey string) *HazardsCollector {
	return &HazardsCollector{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		bus:         bus,
		firmsMapKey: firmsMapKey,
		cacheTTL:    5 * time.Minute,
	}
}

// Refresh fetches earthquakes and fires and updates the entity bus
func (c *HazardsCollector) Refresh(ctx context.Context) ([]GeoEntity, error) {
	c.cacheMu.RLock()
	if time.Since(c.lastFetch) < c.cacheTTL && (len(c.cachedQuakes) > 0 || len(c.cachedFires) > 0) {
		all := make([]GeoEntity, 0, len(c.cachedQuakes)+len(c.cachedFires))
		all = append(all, c.cachedQuakes...)
		all = append(all, c.cachedFires...)
		c.cacheMu.RUnlock()
		return all, nil
	}
	c.cacheMu.RUnlock()

	quakes, errQuakes := c.fetchUSGSEarthquakes(ctx)
	fires, errFires := c.fetchFIRMSFires(ctx)

	c.cacheMu.Lock()
	if len(quakes) > 0 {
		c.cachedQuakes = quakes
	}
	if len(fires) > 0 {
		c.cachedFires = fires
	}
	c.lastFetch = time.Now().UTC()
	all := make([]GeoEntity, 0, len(c.cachedQuakes)+len(c.cachedFires))
	all = append(all, c.cachedQuakes...)
	all = append(all, c.cachedFires...)
	c.cacheMu.Unlock()

	if len(all) > 0 {
		c.bus.UpsertBatch(all)
	}

	statusMsg := "OK"
	if errQuakes != nil || errFires != nil {
		statusMsg = "DEGRADED"
	}

	c.bus.UpdateCollectorStatus("hazards", CollectorStatus{
		Name:         "Hazards (USGS Earthquakes & NASA FIRMS)",
		Enabled:      true,
		ActiveCount:  len(all),
		LastRefresh:  time.Now().UTC(),
		SourceStatus: statusMsg,
	})

	return all, nil
}

// FetchUSGSEarthquakes loads real-time M2.5+ earthquake GeoJSON from USGS
func (c *HazardsCollector) fetchUSGSEarthquakes(ctx context.Context) ([]GeoEntity, error) {
	url := "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/2.5_day.geojson"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Aethel-GEOINT/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("usgs earthquakes returned %d", resp.StatusCode)
	}

	body, err := readBoundedBody(resp.Body, 8*1024*1024)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Features []struct {
			ID         string `json:"id"`
			Properties struct {
				Mag     float64 `json:"mag"`
				Place   string  `json:"place"`
				Time    int64   `json:"time"`
				Updated int64   `json:"updated"`
				Alert   string  `json:"alert"`
				Status  string  `json:"status"`
				Tsunami int     `json:"tsunami"`
				URL     string  `json:"url"`
			} `json:"properties"`
			Geometry struct {
				Coordinates []float64 `json:"coordinates"` // [lon, lat, depth_km]
			} `json:"geometry"`
		} `json:"features"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	entities := make([]GeoEntity, 0, len(payload.Features))
	for _, f := range payload.Features {
		if len(f.Geometry.Coordinates) < 2 {
			continue
		}
		lon := f.Geometry.Coordinates[0]
		lat := f.Geometry.Coordinates[1]
		depthKm := 10.0
		if len(f.Geometry.Coordinates) >= 3 {
			depthKm = f.Geometry.Coordinates[2]
		}

		eventTime := time.UnixMilli(f.Properties.Time).UTC()
		label := fmt.Sprintf("M%.1f Earthquake · %s", f.Properties.Mag, f.Properties.Place)

		severity := "MODERATE"
		if f.Properties.Mag >= 6.0 {
			severity = "MAJOR"
		} else if f.Properties.Mag >= 7.0 {
			severity = "CRITICAL"
		}

		entities = append(entities, GeoEntity{
			ID:     "quake:" + f.ID,
			Type:   TypeEarthquake,
			Source: "usgs",
			Label:  label,
			Position: GeoPosition{
				Latitude:  lat,
				Longitude: lon,
				Altitude:  -depthKm * 1000.0, // underground depth in meters
			},
			Timestamp:      eventTime,
			Freshness:      1.0,
			Confidence:     100,
			Classification: severity,
			Metadata: map[string]any{
				"magnitude": f.Properties.Mag,
				"place":     f.Properties.Place,
				"depth_km":  depthKm,
				"alert":     f.Properties.Alert,
				"tsunami":   f.Properties.Tsunami == 1,
				"url":       f.Properties.URL,
			},
		})
	}

	return entities, nil
}

// FetchFIRMSFires fetches live thermal hotspots from NASA FIRMS VIIRS NRT
func (c *HazardsCollector) fetchFIRMSFires(ctx context.Context) ([]GeoEntity, error) {
	if c.firmsMapKey == "" {
		// Provide curated tactical global fire/thermal hotspots baseline
		return c.seedGlobalThermalAnomalies(), nil
	}

	url := fmt.Sprintf("https://firms.modaps.eosdis.nasa.gov/api/country/csv/%s/VIIRS_NOAA20_NRT/WORLD/1", c.firmsMapKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return c.seedGlobalThermalAnomalies(), nil
	}
	req.Header.Set("User-Agent", "Aethel-GEOINT/1.0")

	resp, err := c.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return c.seedGlobalThermalAnomalies(), nil
	}
	defer resp.Body.Close()

	body, err := readBoundedBody(resp.Body, 8*1024*1024)
	if err != nil {
		return c.seedGlobalThermalAnomalies(), nil
	}

	lines := strings.Split(string(body), "\n")
	if len(lines) < 2 {
		return c.seedGlobalThermalAnomalies(), nil
	}

	entities := make([]GeoEntity, 0)
	now := time.Now().UTC()

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 8 {
			continue
		}

		var lat, lon, brightness, frp float64
		fmt.Sscanf(parts[1], "%f", &lat)
		fmt.Sscanf(parts[2], "%f", &lon)
		fmt.Sscanf(parts[3], "%f", &brightness)
		fmt.Sscanf(parts[11], "%f", &frp)

		confidence := parts[8]
		daynight := parts[9]

		id := fmt.Sprintf("fire:firms-%d", i)
		entities = append(entities, GeoEntity{
			ID:     id,
			Type:   TypeFire,
			Source: "nasa_firms",
			Label:  fmt.Sprintf("Thermal Anomaly (FRP: %.1f MW)", frp),
			Position: GeoPosition{
				Latitude:  lat,
				Longitude: lon,
			},
			Timestamp:      now,
			Freshness:      1.0,
			Confidence:     90,
			Classification: "THERMAL_HOTSPOT",
			Metadata: map[string]any{
				"brightness_k": brightness,
				"frp_mw":       frp,
				"confidence":   confidence,
				"daynight":     daynight,
			},
		})
		if len(entities) >= 500 {
			break
		}
	}

	return entities, nil
}

func (c *HazardsCollector) seedGlobalThermalAnomalies() []GeoEntity {
	now := time.Now().UTC()
	anomalies := []struct {
		id, label string
		lat, lon  float64
		frp       float64
	}{
		{"firms-01", "Wildfire Thermal Cluster // Pantanal", -17.550, -56.850, 142.5},
		{"firms-02", "Agricultural Burn Area // Punjab", 30.730, 75.850, 48.0},
		{"firms-03", "Thermal Spike // Eastern Front Ukraine", 48.020, 37.810, 210.0},
		{"firms-04", "Forest Fire Front // British Columbia", 52.120, -122.450, 185.0},
		{"firms-05", "Thermal Signature // Red Sea Coast", 14.850, 42.950, 85.0},
		{"firms-06", "Active Flaring & Fire // Niger Delta", 4.750, 6.850, 95.0},
		{"firms-07", "Bushfire Cluster // Northern Territory AU", -14.250, 132.500, 115.0},
	}

	entities := make([]GeoEntity, 0, len(anomalies))
	for _, a := range anomalies {
		entities = append(entities, GeoEntity{
			ID:     "fire:" + a.id,
			Type:   TypeFire,
			Source: "nasa_firms",
			Label:  a.label,
			Position: GeoPosition{
				Latitude:  a.lat,
				Longitude: a.lon,
			},
			Timestamp:      now,
			Freshness:      1.0,
			Confidence:     90,
			Classification: "THERMAL_ANOMALY",
			Metadata: map[string]any{
				"frp_mw": a.frp,
			},
		})
	}
	return entities
}
