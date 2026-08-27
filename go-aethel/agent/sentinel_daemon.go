// STATUS: DIAMANT VGT SUPREME
package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"go-aethel/security"
)

type ThreatSeverity string

const (
	SeverityNormal      ThreatSeverity = "NORMAL"
	SeverityElevated    ThreatSeverity = "ELEVATED"
	SeverityHigh        ThreatSeverity = "HIGH"
	SeverityExistential ThreatSeverity = "EXISTENTIAL"
)

type EmergencyAlert struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	City           string         `json:"city"`
	Country        string         `json:"country"`
	Severity       ThreatSeverity `json:"severity"`
	RiskScore      int            `json:"risk_score"`
	Source         string         `json:"source"`
	Timestamp      string         `json:"timestamp"`
	IsExistential  bool           `json:"is_existential"`
	VoiceAlertText string         `json:"voice_alert_text"`
	LocalMatch     bool           `json:"local_match"`
}

type SentinelDaemon struct {
	mu           sync.RWMutex
	userCity     string
	userCountry  string
	alerts       []EmergencyAlert
	activeTicker *time.Ticker
	stopChan     chan struct{}
	running      bool
}

var SharedSentinel = NewSentinelDaemon()

// countryAliases maps common country labels to alternate place-name tokens
// used in USGS/EONET place strings (e.g. "Germany" ↔ "Deutschland").
var countryAliases = map[string][]string{
	"germany":     {"germany", "deutschland", "federal republic of germany"},
	"deutschland": {"germany", "deutschland", "federal republic of germany"},
	"austria":     {"austria", "österreich", "oesterreich"},
	"österreich":  {"austria", "österreich", "oesterreich"},
	"oesterreich": {"austria", "österreich", "oesterreich"},
	"switzerland": {"switzerland", "schweiz", "suisse", "svizzera"},
	"schweiz":     {"switzerland", "schweiz", "suisse", "svizzera"},
	"france":      {"france", "frankreich"},
	"frankreich":  {"france", "frankreich"},
	"italy":       {"italy", "italien", "italia"},
	"italien":     {"italy", "italien", "italia"},
	"spain":       {"spain", "spanien", "españa", "espana"},
	"spanien":     {"spain", "spanien", "españa", "espana"},
	"usa":         {"united states", "u.s.", "usa", "u.s.a."},
	"us":          {"united states", "u.s.", "usa"},
	"united states": {"united states", "u.s.", "usa", "u.s.a."},
	"uk":          {"united kingdom", "u.k.", "great britain", "england", "scotland", "wales"},
	"united kingdom": {"united kingdom", "u.k.", "great britain"},
	"japan":       {"japan", "japan region"},
	"japan region": {"japan", "japan region"},
}

func NewSentinelDaemon() *SentinelDaemon {
	return &SentinelDaemon{
		alerts:   make([]EmergencyAlert, 0),
		stopChan: make(chan struct{}),
	}
}

func (s *SentinelDaemon) SetLocation(city, country string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userCity = strings.TrimSpace(city)
	s.userCountry = strings.TrimSpace(country)
}

func (s *SentinelDaemon) GetLocation() (city, country string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userCity, s.userCountry
}

func (s *SentinelDaemon) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.activeTicker = time.NewTicker(3 * time.Minute)
	s.mu.Unlock()

	go func() {
		s.runSentinelCheck()
		for {
			select {
			case <-s.activeTicker.C:
				s.runSentinelCheck()
			case <-s.stopChan:
				return
			}
		}
	}()
}

func (s *SentinelDaemon) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	if s.activeTicker != nil {
		s.activeTicker.Stop()
	}
	close(s.stopChan)
}

func (s *SentinelDaemon) GetAlerts() []EmergencyAlert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]EmergencyAlert(nil), s.alerts...)
}

func (s *SentinelDaemon) AddAlert(alert EmergencyAlert) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.alerts {
		if existing.ID == alert.ID {
			return
		}
	}

	s.alerts = append([]EmergencyAlert{alert}, s.alerts...)
	if len(s.alerts) > 50 {
		s.alerts = s.alerts[:50]
	}

	security.LogKernelActivity("SENTINEL_EMERGENCY_ALERT", alert.ID, string(alert.Severity))
}

func (s *SentinelDaemon) runSentinelCheck() {
	s.mu.RLock()
	city := s.userCity
	country := s.userCountry
	s.mu.RUnlock()

	// Location-bound threats only fire against the operator's configured home area.
	// Global space-weather / NEO items may still be catalogued, but never as
	// existential local emergency popups.
	s.checkUSGSFeed(city, country)
	s.checkSolarStormsNOAA()
	s.checkAsteroidsNASA()
	s.checkVolcanoesNASA(city, country)
	s.checkConflictTerrorFeeds(city, country)
}

// normalizePlaceToken lowercases and collapses whitespace for matching.
func normalizePlaceToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

// containsPlaceToken reports whether haystack contains needle as a whole token/phrase
// (not a raw substring of an unrelated word). Minimum length avoids "us"/"de" false positives.
func containsPlaceToken(haystack, needle string) bool {
	haystack = normalizePlaceToken(haystack)
	needle = normalizePlaceToken(needle)
	if haystack == "" || needle == "" || len(needle) < 3 {
		return false
	}
	if haystack == needle {
		return true
	}
	// Word-boundary style: needle must appear as its own token sequence.
	escaped := regexp.QuoteMeta(needle)
	pattern := `(^|[^a-z0-9äöüß])` + escaped + `([^a-z0-9äöüß]|$)`
	matched, err := regexp.MatchString(pattern, haystack)
	return err == nil && matched
}

// matchUserLocation classifies how closely an event place string matches the operator home.
// cityMatch is required for existential local alarms. countryMatch alone is informational at most.
func matchUserLocation(place, city, country string) (cityMatch, countryMatch bool) {
	place = normalizePlaceToken(place)
	city = normalizePlaceToken(city)
	country = normalizePlaceToken(country)

	if city != "" && len(city) >= 3 {
		cityMatch = containsPlaceToken(place, city)
	}

	if country != "" && len(country) >= 3 {
		aliases := countryAliases[country]
		if len(aliases) == 0 {
			aliases = []string{country}
		}
		for _, alias := range aliases {
			if containsPlaceToken(place, alias) {
				countryMatch = true
				break
			}
		}
		if !countryMatch {
			countryMatch = containsPlaceToken(place, country)
		}
	}
	return cityMatch, countryMatch
}

func hasConfiguredHome(city, country string) bool {
	return strings.TrimSpace(city) != "" || strings.TrimSpace(country) != ""
}

func (s *SentinelDaemon) checkUSGSFeed(city, country string) {
	// Without a configured home area, never invent "local" earthquake emergencies.
	if !hasConfiguredHome(city, country) {
		return
	}

	resp, err := http.Get("https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/4.5_day.geojson")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var geoJSON struct {
		Features []struct {
			ID         string `json:"id"`
			Properties struct {
				Mag     float64 `json:"mag"`
				Place   string  `json:"place"`
				Time    int64   `json:"time"`
				Title   string  `json:"title"`
				Alert   string  `json:"alert"`
				Tsunami int     `json:"tsunami"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geoJSON); err != nil {
		return
	}

	for _, feat := range geoJSON.Features {
		cityMatch, countryMatch := matchUserLocation(feat.Properties.Place, city, country)
		// Strict local gate: only events that mention the operator city or country.
		// Remote critical quakes are NOT operator emergencies.
		if !cityMatch && !countryMatch {
			continue
		}

		hasTsunamiWarning := feat.Properties.Tsunami == 1
		severity := SeverityElevated
		isExistential := false
		riskScore := int(feat.Properties.Mag * 12)
		if riskScore > 99 {
			riskScore = 99
		}

		title := fmt.Sprintf("🚨 NOTFALL-WARNUNG: %s", feat.Properties.Title)
		desc := fmt.Sprintf("Mag %.1f Erdbeben registriert nahe %s. Zeit: %s", feat.Properties.Mag, feat.Properties.Place, time.Unix(feat.Properties.Time/1000, 0).Format("15:04:05"))
		voiceText := fmt.Sprintf("Achtung Operator. Lokale Gefahr erkannt: %s. Stärke %.1f Magnitude.", feat.Properties.Title, feat.Properties.Mag)

		// Existential popup ONLY for city-level matches (or city+country).
		// Country-only matches stay elevated/high so a quake on the other side of
		// the same country does not force a full emergency modal.
		if cityMatch {
			if hasTsunamiWarning {
				severity = SeverityExistential
				isExistential = true
				riskScore = 98
				title = fmt.Sprintf("🚨 EXISTENTIELLE TSUNAMI-WARNUNG: M %.1f - %s", feat.Properties.Mag, strings.ToUpper(feat.Properties.Place))
				desc = fmt.Sprintf("⚠️ TSUNAMI-WARNUNG FÜR DEINEN STANDORT! Mag %.1f Erdbeben nahe %s.", feat.Properties.Mag, feat.Properties.Place)
				voiceText = fmt.Sprintf("ACHTUNG OPERATOR! EXISTENTIELLE TSUNAMI-WARNUNG FÜR DEINEN STANDORT %s AUSGELÖST!", strings.ToUpper(city))
			} else if feat.Properties.Mag >= 5.5 {
				severity = SeverityExistential
				isExistential = true
				riskScore = 95
			} else if feat.Properties.Mag >= 4.5 {
				severity = SeverityHigh
			}
		} else if countryMatch {
			// Country-only: catalogue as high/elevated, never existential overlay.
			if feat.Properties.Mag >= 6.5 || hasTsunamiWarning {
				severity = SeverityHigh
				if riskScore > 88 {
					riskScore = 88
				}
			} else {
				severity = SeverityElevated
			}
			title = fmt.Sprintf("⚠️ LANDES-WARNUNG: %s", feat.Properties.Title)
			voiceText = fmt.Sprintf("Achtung Operator. Erdbeben im Land %s: Stärke %.1f. Kein Stadt-Treffer für deinen Standort.", country, feat.Properties.Mag)
		}

		alert := EmergencyAlert{
			ID:             fmt.Sprintf("usgs_%s", feat.ID),
			Title:          title,
			Description:    desc,
			City:           city,
			Country:        country,
			Severity:       severity,
			RiskScore:      riskScore,
			Source:         "USGS & CORRELATED GLOBAL TSUNAMI MONITOR",
			Timestamp:      time.Now().Format(time.RFC3339),
			IsExistential:  isExistential,
			VoiceAlertText: voiceText,
			LocalMatch:     cityMatch,
		}
		s.AddAlert(alert)
	}
}

func (s *SentinelDaemon) checkSolarStormsNOAA() {
	resp, err := http.Get("https://services.swpc.noaa.gov/json/planetary_k_index_1m.json")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var kIndexData []struct {
		TimeTag string  `json:"time_tag"`
		Kp      float64 `json:"kp_index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&kIndexData); err != nil || len(kIndexData) == 0 {
		return
	}

	latest := kIndexData[len(kIndexData)-1]
	if latest.Kp < 6.0 {
		return
	}

	// Global space weather is never an existential *local* emergency popup.
	severity := SeverityHigh
	if latest.Kp >= 7.5 {
		severity = SeverityHigh
	}

	alert := EmergencyAlert{
		ID:             fmt.Sprintf("noaa_swpc_%s", strings.ReplaceAll(latest.TimeTag, " ", "_")),
		Title:          fmt.Sprintf("☀️ GEOMAGNETISCHER SONNENSTURM (NOAA SWPC): Kp %.1f", latest.Kp),
		Description:    fmt.Sprintf("Extreme Sonnenaktivität / Geomagnetischer Sturm registriert (Kp Index %.1f). Risiko für Satellitennavigation, GPS, Stromnetze und Kurzwellenfunk.", latest.Kp),
		City:           "Global / Orbit",
		Country:        "Space Weather",
		Severity:       severity,
		RiskScore:      int(latest.Kp * 11.5),
		Source:         "NOAA SPACE WEATHER PREDICTION CENTER",
		Timestamp:      time.Now().Format(time.RFC3339),
		IsExistential:  false,
		VoiceAlertText: fmt.Sprintf("Achtung Operator. Geomagnetischer Sonnensturm mit Kp-Index %.1f durch NOAA registriert. Störungen in GPS und Stromnetzen möglich.", latest.Kp),
		LocalMatch:     false,
	}
	s.AddAlert(alert)
}

func (s *SentinelDaemon) checkAsteroidsNASA() {
	todayStr := time.Now().Format("2006-01-02")
	resp, err := http.Get(fmt.Sprintf("https://api.nasa.gov/neo/rest/v1/feed?start_date=%s&end_date=%s&api_key=DEMO_KEY", todayStr, todayStr))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var neoData struct {
		NearEarthObjects map[string][]struct {
			ID                     string `json:"id"`
			Name                   string `json:"name"`
			IsPotentiallyHazardous bool   `json:"is_potentially_hazardous_asteroid"`
		} `json:"near_earth_objects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&neoData); err != nil {
		return
	}

	for _, objects := range neoData.NearEarthObjects {
		for _, obj := range objects {
			if !obj.IsPotentiallyHazardous {
				continue
			}
			// Catalogue only — never force local emergency overlay.
			alert := EmergencyAlert{
				ID:             fmt.Sprintf("nasa_neo_%s", obj.ID),
				Title:          fmt.Sprintf("☄️ POTENZIELL GEFÄHRLICHER ASTEROID (NASA NEO): %s", obj.Name),
				Description:    fmt.Sprintf("NASA Near-Earth Object Web Service hat potenziell gefährlichen Nahvorbeiflug von Asteroid %s registriert.", obj.Name),
				City:           "Erdbahn / Space",
				Country:        "Global Monitor",
				Severity:       SeverityElevated,
				RiskScore:      70,
				Source:         "NASA NEAR-EARTH OBJECT PROGRAM",
				Timestamp:      time.Now().Format(time.RFC3339),
				IsExistential:  false,
				VoiceAlertText: fmt.Sprintf("Achtung Operator. NASA Asteroiden-Hinweis: Potenziell gefährlicher Near-Earth Object %s erfasst.", obj.Name),
				LocalMatch:     false,
			}
			s.AddAlert(alert)
		}
	}
}

func (s *SentinelDaemon) checkVolcanoesNASA(city, country string) {
	if !hasConfiguredHome(city, country) {
		return
	}

	resp, err := http.Get("https://eonet.gsfc.nasa.gov/api/v3/categories/volcanoes")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var eonetData struct {
		Events []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"events"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&eonetData); err != nil {
		return
	}

	for _, ev := range eonetData.Events {
		cityMatch, countryMatch := matchUserLocation(ev.Title, city, country)
		if !cityMatch && !countryMatch {
			continue
		}
		// Never existential for remote volcano catalogue hits; city match → high.
		severity := SeverityElevated
		if cityMatch {
			severity = SeverityHigh
		}
		alert := EmergencyAlert{
			ID:             fmt.Sprintf("nasa_eonet_volcano_%s", ev.ID),
			Title:          fmt.Sprintf("🌋 VULKANAUSBRUCH / ASCHEWOLKE (NASA EONET): %s", ev.Title),
			Description:    fmt.Sprintf("NASA Earth Observatory Network meldet aktiven Vulkanausbruch im Bezug zu deinem Standort: %s.", ev.Title),
			City:           city,
			Country:        country,
			Severity:       severity,
			RiskScore:      80,
			Source:         "NASA EONET VOLCANO MONITOR",
			Timestamp:      time.Now().Format(time.RFC3339),
			IsExistential:  false,
			VoiceAlertText: fmt.Sprintf("Achtung Operator. NASA EONET Vulkanausbruch-Warnung nahe deinem Standort: %s.", ev.Title),
			LocalMatch:     cityMatch,
		}
		s.AddAlert(alert)
	}
}

func (s *SentinelDaemon) checkConflictTerrorFeeds(city, country string) {
	// Geopolitical Conflict / Terror threat scanner placeholder for OSINT integration
}

func HandleSentinelAlerts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		alerts := SharedSentinel.GetAlerts()
		city, country := SharedSentinel.GetLocation()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"alerts":  alerts,
			"city":    city,
			"country": country,
		})
	} else if r.Method == http.MethodPost {
		var req struct {
			City    string `json:"city"`
			Country string `json:"country"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			SharedSentinel.SetLocation(req.City, req.Country)
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	} else {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
