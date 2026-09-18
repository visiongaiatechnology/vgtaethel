package geoint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AircraftCollector manages live ADS-B tracking via OpenSky and adsb.lol fallback
type AircraftCollector struct {
	client       *http.Client
	bus          *GeoEntityBus
	cacheMu      sync.RWMutex
	cachedStates []GeoEntity
	lastFetch    time.Time
	cooldownEnd  time.Time
	cacheTTL     time.Duration
	openSkyUser  string
	openSkyPass  string
}

// NewAircraftCollector creates an AircraftCollector
func NewAircraftCollector(bus *GeoEntityBus) *AircraftCollector {
	return &AircraftCollector{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		bus:      bus,
		cacheTTL: 30 * time.Second,
	}
}

// SetCredentials sets optional OpenSky Network credentials
func (c *AircraftCollector) SetCredentials(user, pass string) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.openSkyUser = user
	c.openSkyPass = pass
}

// Refresh performs a live collection cycle
func (c *AircraftCollector) Refresh(ctx context.Context) ([]GeoEntity, error) {
	c.cacheMu.RLock()
	if time.Now().Before(c.cooldownEnd) && len(c.cachedStates) > 0 {
		cached := make([]GeoEntity, len(c.cachedStates))
		copy(cached, c.cachedStates)
		c.cacheMu.RUnlock()
		return cached, nil
	}
	if time.Since(c.lastFetch) < c.cacheTTL && len(c.cachedStates) > 0 {
		cached := make([]GeoEntity, len(c.cachedStates))
		copy(cached, c.cachedStates)
		c.cacheMu.RUnlock()
		return cached, nil
	}
	c.cacheMu.RUnlock()

	// 1. Try adsb.lol Military feed first for high-value contacts
	milEntities, errMil := c.fetchAdsbLolMilitary(ctx)

	// 2. Try OpenSky for general airspace
	generalEntities, errOpenSky := c.fetchOpenSky(ctx)

	var entities []GeoEntity
	if errOpenSky == nil && len(generalEntities) > 0 {
		// Merge military entities into general entities (giving military classification priority)
		milMap := make(map[string]GeoEntity)
		for _, m := range milEntities {
			milMap[m.ID] = m
		}
		for _, g := range generalEntities {
			if m, isMil := milMap[g.ID]; isMil {
				entities = append(entities, m)
				delete(milMap, g.ID)
			} else {
				entities = append(entities, g)
			}
		}
		for _, m := range milMap {
			entities = append(entities, m)
		}
	} else if len(milEntities) > 0 {
		entities = milEntities
	}

	if len(entities) > 0 {
		c.cacheMu.Lock()
		c.cachedStates = entities
		c.lastFetch = time.Now().UTC()
		c.cacheMu.Unlock()

		c.bus.UpsertBatch(entities)
		c.bus.UpdateCollectorStatus("aircraft", CollectorStatus{
			Name:         "Aircraft ADS-B",
			Enabled:      true,
			ActiveCount:  len(entities),
			LastRefresh:  time.Now().UTC(),
			SourceStatus: "OK",
		})
		return entities, nil
	}

	statusMsg := "OFFLINE"
	if errMil != nil && errOpenSky != nil {
		statusMsg = fmt.Sprintf("OpenSky: %v; adsb.lol: %v", errOpenSky, errMil)
	}
	c.bus.UpdateCollectorStatus("aircraft", CollectorStatus{
		Name:         "Aircraft ADS-B",
		Enabled:      true,
		ActiveCount:  len(c.cachedStates),
		LastRefresh:  time.Now().UTC(),
		LastError:    statusMsg,
		SourceStatus: "DEGRADED",
	})

	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	return c.cachedStates, nil
}

// FetchOpenSky queries OpenSky Network state vectors
func (c *AircraftCollector) fetchOpenSky(ctx context.Context) ([]GeoEntity, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://opensky-network.org/api/states/all", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Aethel-GEOINT/1.0")

	c.cacheMu.RLock()
	if c.openSkyUser != "" && c.openSkyPass != "" {
		req.SetBasicAuth(c.openSkyUser, c.openSkyPass)
	}
	c.cacheMu.RUnlock()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		c.cacheMu.Lock()
		c.cooldownEnd = time.Now().Add(5 * time.Minute)
		c.cacheMu.Unlock()
		return nil, fmt.Errorf("opensky rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opensky returned status %d", resp.StatusCode)
	}

	body, err := readBoundedBody(resp.Body, 16*1024*1024)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Time   int64           `json:"time"`
		States [][]interface{} `json:"states"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	entities := make([]GeoEntity, 0, len(payload.States))
	now := time.Now().UTC()

	for _, s := range payload.States {
		if len(s) < 17 {
			continue
		}
		icao24, _ := s[0].(string)
		if icao24 == "" {
			continue
		}
		callsign, _ := s[1].(string)
		callsign = strings.TrimSpace(callsign)
		originCountry, _ := s[2].(string)

		lonVal, okLon := s[5].(float64)
		latVal, okLat := s[6].(float64)
		if !okLon || !okLat {
			continue
		}
		baroAlt, _ := s[7].(float64)
		onGround, _ := s[8].(bool)
		velocity, _ := s[9].(float64)
		trueTrack, _ := s[10].(float64)
		verticalRate, _ := s[11].(float64)
		geoAlt, _ := s[13].(float64)
		squawk, _ := s[14].(string)

		alt := baroAlt
		if geoAlt > 0 {
			alt = geoAlt
		}

		isMil, milClass := classifyMilitaryAircraft(icao24, callsign, squawk)
		entityType := TypeAircraft
		if isMil {
			entityType = TypeMilAircraft
		}

		label := callsign
		if label == "" {
			label = strings.ToUpper(icao24)
		}

		entities = append(entities, GeoEntity{
			ID:     "air:" + strings.ToLower(icao24),
			Type:   entityType,
			Source: "opensky",
			Label:  label,
			Position: GeoPosition{
				Latitude:  latVal,
				Longitude: lonVal,
				Altitude:  alt,
				Heading:   trueTrack,
				Velocity:  velocity,
				ClimbRate: verticalRate,
				Course:    trueTrack,
			},
			Timestamp:      now,
			Freshness:      1.0,
			Confidence:     95,
			Classification: milClass,
			Metadata: map[string]any{
				"icao24":         strings.ToLower(icao24),
				"callsign":       callsign,
				"origin_country": originCountry,
				"squawk":         squawk,
				"on_ground":      onGround,
				"military":       isMil,
			},
		})
	}

	return entities, nil
}

// FetchAdsbLolMilitary queries adsb.lol military aircraft endpoint
func (c *AircraftCollector) fetchAdsbLolMilitary(ctx context.Context) ([]GeoEntity, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.adsb.lol/v2/mil", nil)
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
		return nil, fmt.Errorf("adsb.lol returned status %d", resp.StatusCode)
	}

	body, err := readBoundedBody(resp.Body, 16*1024*1024)
	if err != nil {
		return nil, err
	}

	var payload struct {
		AC []struct {
			Hex      string  `json:"hex"`
			Flight   string  `json:"flight"`
			Lat      float64 `json:"lat"`
			Lon      float64 `json:"lon"`
			AltBaro  any     `json:"alt_baro"`
			AltGeom  any     `json:"alt_geom"`
			Track    float64 `json:"track"`
			GS       float64 `json:"gs"`
			BaroRate float64 `json:"baro_rate"`
			Squawk   string  `json:"squawk"`
			Type     string  `json:"t"`
			Desc     string  `json:"desc"`
			OwnOp    string  `json:"ownOp"`
		} `json:"ac"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	entities := make([]GeoEntity, 0, len(payload.AC))
	now := time.Now().UTC()

	for _, ac := range payload.AC {
		if ac.Hex == "" || (ac.Lat == 0 && ac.Lon == 0) {
			continue
		}

		var alt float64
		if f, ok := ac.AltBaro.(float64); ok {
			alt = f * 0.3048 // feet to meters
		} else if s, ok := ac.AltBaro.(string); ok {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				alt = f * 0.3048
			}
		}

		callsign := strings.TrimSpace(ac.Flight)
		label := callsign
		if label == "" {
			if ac.Type != "" {
				label = ac.Type + " (" + strings.ToUpper(ac.Hex) + ")"
			} else {
				label = "MIL-" + strings.ToUpper(ac.Hex)
			}
		}

		classification := classifyMilitaryByType(ac.Type, ac.Desc, callsign)

		entities = append(entities, GeoEntity{
			ID:     "air:" + strings.ToLower(ac.Hex),
			Type:   TypeMilAircraft,
			Source: "adsb.lol",
			Label:  label,
			Position: GeoPosition{
				Latitude:  ac.Lat,
				Longitude: ac.Lon,
				Altitude:  alt,
				Heading:   ac.Track,
				Velocity:  ac.GS * 0.514444, // knots to m/s
				ClimbRate: ac.BaroRate * 0.00508, // fpm to m/s
				Course:    ac.Track,
			},
			Timestamp:      now,
			Freshness:      1.0,
			Confidence:     98,
			Classification: classification,
			Metadata: map[string]any{
				"icao24":    strings.ToLower(ac.Hex),
				"callsign":  callsign,
				"type":      ac.Type,
				"desc":      ac.Desc,
				"operator":  ac.OwnOp,
				"squawk":    ac.Squawk,
				"military":  true,
			},
		})
	}

	return entities, nil
}

// Helper: classifies military aircraft based on callsign / squawk / icao prefix
func classifyMilitaryAircraft(icao24, callsign, squawk string) (bool, string) {
	c := strings.ToUpper(callsign)
	for _, prefix := range []string{"RCH", "REACH", "VIPER", "FORTE", "HOMER", "LAGR", "TITAN", "DUKE", "JAKE", "NATO", "MAGIC", "COBRA", "TALON", "GHOST", "SWORD", "RAVEN", "DARK", "SHADOW", "BULL", "HAWK", "EAGLE"} {
		if strings.HasPrefix(c, prefix) {
			return true, "MIL_" + prefix
		}
	}
	if squawk == "7700" || squawk == "7600" || squawk == "7500" {
		return true, "EMERGENCY_" + squawk
	}
	return false, "CIVIL_AVIATION"
}

func classifyMilitaryByType(actype, desc, callsign string) string {
	t := strings.ToUpper(actype)
	d := strings.ToUpper(desc)
	switch {
	case strings.Contains(t, "F16") || strings.Contains(t, "F35") || strings.Contains(t, "F22") || strings.Contains(t, "F15") || strings.Contains(t, "FA18") || strings.Contains(t, "EUFI") || strings.Contains(t, "SU") || strings.Contains(t, "MIG"):
		return "MIL_FIGHTER"
	case strings.Contains(t, "C17") || strings.Contains(t, "C130") || strings.Contains(t, "A400") || strings.Contains(t, "IL76") || strings.Contains(t, "AN124"):
		return "MIL_TRANSPORT"
	case strings.Contains(t, "KC135") || strings.Contains(t, "KC46") || strings.Contains(t, "A330MRTT") || strings.Contains(t, "IL78"):
		return "MIL_TANKER"
	case strings.Contains(t, "RQ4") || strings.Contains(t, "MQ9") || strings.Contains(t, "FORTE") || strings.Contains(t, "E3TF") || strings.Contains(t, "RC135") || strings.Contains(t, "P8"):
		return "MIL_RECON_ISR"
	case strings.Contains(t, "B52") || strings.Contains(t, "B1") || strings.Contains(t, "B2") || strings.Contains(t, "TU95") || strings.Contains(t, "TU160"):
		return "MIL_BOMBER"
	case strings.Contains(d, "HELICOPTER") || strings.Contains(t, "H60") || strings.Contains(t, "H47") || strings.Contains(t, "AH64"):
		return "MIL_ROTORCRAFT"
	default:
		if strings.HasPrefix(strings.ToUpper(callsign), "RCH") {
			return "MIL_AIRLIFT"
		}
		return "MIL_TACTICAL"
	}
}
