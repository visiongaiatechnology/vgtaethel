package geoint

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SatelliteCollector fetches and propagates satellite orbits from CelesTrak TLEs
type SatelliteCollector struct {
	client      *http.Client
	bus         *GeoEntityBus
	cacheDir    string
	cacheMu     sync.RWMutex
	tleCatalogs map[string][]TLEData
	lastFetch   time.Time
	cacheTTL    time.Duration
}

// TLEData holds raw and parsed Two-Line Element set
type TLEData struct {
	Name         string    `json:"name"`
	Line1        string    `json:"line1"`
	Line2        string    `json:"line2"`
	NoradID      string    `json:"norad_id"`
	Inclination  float64   `json:"inclination_deg"`
	RAAN         float64   `json:"raan_deg"`
	Eccentricity float64   `json:"eccentricity"`
	ArgPerigee   float64   `json:"arg_perigee_deg"`
	MeanAnomaly  float64   `json:"mean_anomaly_deg"`
	MeanMotion   float64   `json:"mean_motion_rev_day"` // Revs per day
	EpochYear    int       `json:"epoch_year"`
	EpochDay     float64   `json:"epoch_day"`
	Group        string    `json:"group"`
	Category     string    `json:"category"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// NewSatelliteCollector creates a SatelliteCollector
func NewSatelliteCollector(bus *GeoEntityBus, cacheDir string) *SatelliteCollector {
	if cacheDir == "" {
		cacheDir = "./vgt_workspace/geoint/satellites"
	}
	_ = os.MkdirAll(cacheDir, 0700)
	return &SatelliteCollector{
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
		bus:         bus,
		cacheDir:    cacheDir,
		tleCatalogs: make(map[string][]TLEData),
		cacheTTL:    6 * time.Hour,
	}
}

// SatelliteGroups loaded for tactical & strategic space situational awareness
var SatelliteGroups = []struct {
	Group    string
	Category string
}{
	{"stations", "SPACE_STATION"},
	{"visual", "BRIGHTEST"},
	{"gps-ops", "NAVIGATION_GPS"},
	{"glo-ops", "NAVIGATION_GLONASS"},
	{"galileo", "NAVIGATION_GALILEO"},
	{"geo", "GEOSTATIONARY_COMMS"},
	{"weather", "METEOROLOGICAL"},
	{"resource", "EARTH_OBSERVATION"},
	{"starlink", "MEGA_CONSTELLATION"},
}

// Refresh updates TLE data from CelesTrak and emits propagated positions to the bus
func (c *SatelliteCollector) Refresh(ctx context.Context) ([]GeoEntity, error) {
	c.cacheMu.Lock()
	now := time.Now().UTC()
	needsNetwork := time.Since(c.lastFetch) > c.cacheTTL || len(c.tleCatalogs) == 0

	if needsNetwork {
		for _, g := range SatelliteGroups {
			tles, err := c.fetchOrLoadGroup(ctx, g.Group, g.Category)
			if err == nil && len(tles) > 0 {
				c.tleCatalogs[g.Group] = tles
			}
		}
		c.lastFetch = now
	}
	c.cacheMu.Unlock()

	// Propagate all satellites to current epoch
	entities := c.PropagateAll(now)
	if len(entities) > 0 {
		c.bus.UpsertBatch(entities)
		c.bus.UpdateCollectorStatus("satellites", CollectorStatus{
			Name:         "Satellite Orbits (CelesTrak)",
			Enabled:      true,
			ActiveCount:  len(entities),
			LastRefresh:  now,
			SourceStatus: "OK",
		})
	}

	return entities, nil
}

// FetchOrLoadGroup tries to fetch group from CelesTrak or loads cached file
func (c *SatelliteCollector) fetchOrLoadGroup(ctx context.Context, group, category string) ([]TLEData, error) {
	cacheFile := filepath.Join(c.cacheDir, group+".tle")

	// Try remote fetch
	url := fmt.Sprintf("https://celestrak.org/NORAD/elements/gp.php?GROUP=%s&FORMAT=tle", group)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err == nil {
		req.Header.Set("User-Agent", "Aethel-GEOINT/1.0")
		resp, err := c.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			tles := parseTLEStream(resp.Body, group, category)
			if len(tles) > 0 {
				// Save to disk cache
				c.saveTLECache(cacheFile, tles)
				return tles, nil
			}
		}
	}

	// Fallback to disk cache
	if f, err := os.Open(cacheFile); err == nil {
		defer f.Close()
		return parseTLEStream(f, group, category), nil
	}

	return nil, fmt.Errorf("could not load TLE for group %s", group)
}

func (c *SatelliteCollector) saveTLECache(path string, tles []TLEData) {
	var sb strings.Builder
	for _, t := range tles {
		sb.WriteString(t.Name + "\n")
		sb.WriteString(t.Line1 + "\n")
		sb.WriteString(t.Line2 + "\n")
	}
	_ = os.WriteFile(path, []byte(sb.String()), 0600)
}

func parseTLEStream(r io.Reader, group, category string) []TLEData {
	return parseTLEReader(r, group, category)
}

// PropagateAll computes current subpoints for all loaded satellites
func (c *SatelliteCollector) PropagateAll(t time.Time) []GeoEntity {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	entities := make([]GeoEntity, 0)
	seenNorad := make(map[string]bool)

	for _, groupTles := range c.tleCatalogs {
		for _, tle := range groupTles {
			if seenNorad[tle.NoradID] {
				continue
			}
			seenNorad[tle.NoradID] = true

			lat, lon, altKm, velocityKmS := PropagateKeplerian(tle, t)
			if math.IsNaN(lat) || math.IsNaN(lon) {
				continue
			}

			entities = append(entities, GeoEntity{
				ID:     "sat:" + tle.NoradID,
				Type:   TypeSatellite,
				Source: "celestrak",
				Label:  tle.Name,
				Position: GeoPosition{
					Latitude:  lat,
					Longitude: lon,
					Altitude:  altKm * 1000.0, // meters
					Velocity:  velocityKmS * 1000.0, // m/s
					Heading:   tle.Inclination,
					Course:    tle.Inclination,
				},
				Timestamp:      t,
				Freshness:      1.0,
				Confidence:     99,
				Classification: tle.Category,
				Metadata: map[string]any{
					"norad_id":             tle.NoradID,
					"name":                 tle.Name,
					"group":                tle.Group,
					"category":             tle.Category,
					"inclination_deg":      tle.Inclination,
					"mean_motion_rev_day":  tle.MeanMotion,
					"period_minutes":       1440.0 / math.Max(0.001, tle.MeanMotion),
					"altitude_km":          altKm,
					"line1":                tle.Line1,
					"line2":                tle.Line2,
				},
			})
		}
	}
	return entities
}

// PropagateKeplerian calculates simplified SGP4/Keplerian subpoint (lat, lon, altKm, velocity)
func PropagateKeplerian(tle TLEData, t time.Time) (lat, lon, altKm, velocityKmS float64) {
	if tle.MeanMotion <= 0 {
		return 0, 0, 400, 7.6
	}

	// GM and Earth Radius in km
	const mu = 398600.4418
	const earthRadius = 6378.137

	// Mean motion in radians per second
	n := (tle.MeanMotion * 2.0 * math.Pi) / 86400.0
	// Semi-major axis a = (mu / n^2)^(1/3)
	a := math.Cbrt(mu / (n * n))
	altKm = a - earthRadius
	if altKm < 100 {
		altKm = 100
	}

	// Circular orbital speed v = sqrt(mu / a)
	velocityKmS = math.Sqrt(mu / a)

	// Time elapsed since J2000 in days
	epoch := time.Date(2000+tle.EpochYear%100, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(tle.EpochDay * 24 * float64(time.Hour)))
	dtMinutes := t.Sub(epoch).Minutes()

	// Mean anomaly at target time
	M := (tle.MeanAnomaly + (tle.MeanMotion*360.0/1440.0)*dtMinutes) * DegToRad
	M = math.Mod(M, 2.0*math.Pi)

	// Approximate True Anomaly for near-circular orbit
	E := M + tle.Eccentricity*math.Sin(M)
	nu := 2.0 * math.Atan2(math.Sqrt(1.0+tle.Eccentricity)*math.Sin(E/2.0), math.Sqrt(1.0-tle.Eccentricity)*math.Cos(E/2.0))

	// Argument of latitude u = argPerigee + trueAnomaly
	u := (tle.ArgPerigee * DegToRad) + nu

	// Subsatellite latitude: sin(lat) = sin(inc) * sin(u)
	incRad := tle.Inclination * DegToRad
	sinLat := math.Sin(incRad) * math.Sin(u)
	lat = math.Asin(math.Max(-1.0, math.Min(1.0, sinLat))) * RadToDeg

	// Greenwich Mean Sidereal Time (GMST) approximation
	utcHours := float64(t.Hour()) + float64(t.Minute())/60.0 + float64(t.Second())/3600.0
	dayOfYear := float64(t.YearDay())
	gmstDeg := math.Mod(100.46 + 0.985647*dayOfYear + 15.0*utcHours, 360.0)

	// RAAN progression
	raan := tle.RAAN

	// Longitude of ascending node relative to Greenwich
	lon = math.Atan2(math.Cos(incRad)*math.Sin(u), math.Cos(u))*RadToDeg + raan - gmstDeg
	lon = math.Mod(lon+540.0, 360.0) - 180.0

	return lat, lon, altKm, velocityKmS
}

func parseTLEReader(r io.Reader, group, category string) []TLEData {
	scanner := bufio.NewScanner(r)
	var tles []TLEData
	var currentName string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "1 ") && len(line) >= 68 {
			line1 := line
			if scanner.Scan() {
				line2 := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line2, "2 ") && len(line2) >= 68 {
					tle := parseTLELines(currentName, line1, line2, group, category)
					if tle.NoradID != "" {
						tles = append(tles, tle)
					}
					currentName = ""
				}
			}
		} else if !strings.HasPrefix(line, "2 ") {
			currentName = line
		}
	}
	return tles
}

func parseTLELines(name, l1, l2, group, category string) TLEData {
	norad := strings.TrimSpace(l1[2:7])
	if name == "" {
		name = "SAT-" + norad
	}

	epochYear, _ := strconv.Atoi(strings.TrimSpace(l1[18:20]))
	epochDay, _ := strconv.ParseFloat(strings.TrimSpace(l1[20:32]), 64)

	inc, _ := strconv.ParseFloat(strings.TrimSpace(l2[8:16]), 64)
	raan, _ := strconv.ParseFloat(strings.TrimSpace(l2[17:25]), 64)
	eccStr := "0." + strings.TrimSpace(l2[26:33])
	ecc, _ := strconv.ParseFloat(eccStr, 64)
	argPer, _ := strconv.ParseFloat(strings.TrimSpace(l2[34:42]), 64)
	meanAnom, _ := strconv.ParseFloat(strings.TrimSpace(l2[43:51]), 64)
	meanMotion, _ := strconv.ParseFloat(strings.TrimSpace(l2[52:63]), 64)

	cat := category
	if strings.Contains(strings.ToUpper(name), "ISS") || norad == "25544" {
		cat = "ISS_SPACE_STATION"
	} else if strings.Contains(strings.ToUpper(name), "STARLINK") {
		cat = "STARLINK_CONSTELLATION"
	}

	return TLEData{
		Name:         name,
		Line1:        l1,
		Line2:        l2,
		NoradID:      norad,
		Inclination:  inc,
		RAAN:         raan,
		Eccentricity: ecc,
		ArgPerigee:   argPer,
		MeanAnomaly:  meanAnom,
		MeanMotion:   meanMotion,
		EpochYear:    epochYear,
		EpochDay:     epochDay,
		Group:        group,
		Category:     cat,
		FetchedAt:    time.Now().UTC(),
	}
}
