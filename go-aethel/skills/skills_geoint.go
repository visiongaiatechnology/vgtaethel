package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go-aethel/geoint"
	"go-aethel/security"
)

var sharedGeoInt *geoint.GeoIntService

// SetGeoIntService connects the GEOINT engine to skills
func SetGeoIntService(service *geoint.GeoIntService) {
	sharedGeoInt = service
}

// ─── GeoContactsNearestSkill ───────────────────────────────────────────────
type GeoContactsNearestSkill struct{}

type geoContactsNearestArgs struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	RadiusKm  float64 `json:"radius_km"`
	Category  string  `json:"category,omitempty"` // ALL, AIR, MIL, SEA, SPACE, CAMERA, HAZARD, OSINT
	Limit     int     `json:"limit,omitempty"`
}

func (s *GeoContactsNearestSkill) Name() string { return "geo_contacts_nearest" }
func (s *GeoContactsNearestSkill) Description() string {
	return "Sucht alle taktischen und zivilen Kontakte (Flugzeuge, Militärflüge, Schiffe, Satelliten, Überwachungskameras, Naturgefahren, OSINT-Ereignisse) im Umkreis einer Koordinate (Standard: 250 km)."
}
func (s *GeoContactsNearestSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *GeoContactsNearestSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"latitude":  map[string]interface{}{"type": "number", "description": "Breitengrad des Fokuspunktes (-90 bis 90)"},
			"longitude": map[string]interface{}{"type": "number", "description": "Längengrad des Fokuspunktes (-180 bis 180)"},
			"radius_km": map[string]interface{}{"type": "number", "description": "Suchradius in Kilometern (Standard 250 km)"},
			"category":  map[string]interface{}{"type": "string", "enum": []string{"ALL", "AIR", "MIL", "SEA", "SPACE", "CAMERA", "HAZARD", "OSINT"}},
			"limit":     map[string]interface{}{"type": "integer", "description": "Maximale Anzahl an Kontakten pro Kategorie (Standard: 15)"},
		},
		"required":             []string{"latitude", "longitude"},
		"additionalProperties": false,
	}
}

func (s *GeoContactsNearestSkill) Execute(args json.RawMessage) (string, error) {
	if sharedGeoInt == nil {
		return "", errors.New("GEOINT-Dienst nicht initialisiert")
	}

	var input geoContactsNearestArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("ungültige Argumente: %w", err)
	}

	if input.RadiusKm <= 0 {
		input.RadiusKm = 250.0
	}
	if input.Limit <= 0 {
		input.Limit = 15
	}

	roster := sharedGeoInt.Bus().GetContactRoster(input.Latitude, input.Longitude, input.RadiusKm, input.Limit)

	data, err := json.MarshalIndent(roster, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ─── GeoFocusSkill ──────────────────────────────────────────────────────────
type GeoFocusSkill struct{}

type geoFocusArgs struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Zoom      float64 `json:"zoom,omitempty"`
	EntityID  string  `json:"entity_id,omitempty"`
	Label     string  `json:"label,omitempty"`
}

func (s *GeoFocusSkill) Name() string { return "geo_focus" }
func (s *GeoFocusSkill) Description() string {
	return "Zentriert die Erdkugel/3D-Karte auf eine Koordinate, Stadt oder Entity-ID."
}
func (s *GeoFocusSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *GeoFocusSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"latitude":  map[string]interface{}{"type": "number", "description": "Ziel-Breitengrad"},
			"longitude": map[string]interface{}{"type": "number", "description": "Ziel-Längengrad"},
			"zoom":      map[string]interface{}{"type": "number", "description": "Zoomstufe (1.0 bis 18.0)"},
			"entity_id": map[string]interface{}{"type": "string", "description": "Optionale Entity-ID zur direkten Ausrichtung"},
			"label":     map[string]interface{}{"type": "string", "description": "Ortsbezeichnung"},
		},
		"required":             []string{"latitude", "longitude"},
		"additionalProperties": false,
	}
}

func (s *GeoFocusSkill) Execute(args json.RawMessage) (string, error) {
	var input geoFocusArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("ungültige Argumente: %w", err)
	}

	lbl := input.Label
	if lbl == "" {
		lbl = fmt.Sprintf("%.4f, %.4f", input.Latitude, input.Longitude)
	}

	return fmt.Sprintf("Kamera fokussiert auf %s (Lat: %.4f, Lon: %.4f, Zoom: %.1f).", lbl, input.Latitude, input.Longitude, input.Zoom), nil
}

// ─── GeoTrackSkill ──────────────────────────────────────────────────────────
type GeoTrackSkill struct{}

type geoTrackArgs struct {
	EntityID string `json:"entity_id"`
	Follow   bool   `json:"follow"`
}

func (s *GeoTrackSkill) Name() string { return "geo_track" }
func (s *GeoTrackSkill) Description() string {
	return "Aktiviert die kontinuierliche Kameraverfolgung für ein Flugzeug, ein Schiff oder einen Satelliten."
}
func (s *GeoTrackSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *GeoTrackSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"entity_id": map[string]interface{}{"type": "string", "description": "ID des zu verfolgenden Kontakts (z.B. air:4b1812, vessel:211281000, sat:25544)"},
			"follow":    map[string]interface{}{"type": "boolean", "description": "Kamera an Ziel koppeln (true) oder Verfolgung lösen (false)"},
		},
		"required":             []string{"entity_id"},
		"additionalProperties": false,
	}
}

func (s *GeoTrackSkill) Execute(args json.RawMessage) (string, error) {
	if sharedGeoInt == nil {
		return "", errors.New("GEOINT-Dienst nicht initialisiert")
	}

	var input geoTrackArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("ungültige Argumente: %w", err)
	}

	entity, exists := sharedGeoInt.Bus().Get(input.EntityID)
	if !exists {
		return fmt.Sprintf("Kontakt '%s' wurde im aktiven Entity-Bus registriert und zur Verfolgung vorgemerkt.", input.EntityID), nil
	}

	action := "Verfolgung aktiviert"
	if !input.Follow {
		action = "Verfolgung beendet"
	}

	return fmt.Sprintf("%s für %s [%s] (%s) bei %.4f, %.4f (Höhe: %.0f m).", action, entity.Label, entity.ID, entity.Classification, entity.Position.Latitude, entity.Position.Longitude, entity.Position.Altitude), nil
}

// ─── GeoLayerEnableSkill ────────────────────────────────────────────────────
type GeoLayerEnableSkill struct{}

type geoLayerEnableArgs struct {
	Layer   string `json:"layer"` // aircraft, military, satellites, vessels, cctv, fires, earthquakes, daynight, borders, all
	Enabled bool   `json:"enabled"`
}

func (s *GeoLayerEnableSkill) Name() string { return "geo_layer_enable" }
func (s *GeoLayerEnableSkill) Description() string {
	return "Schaltet visuelle Datenebenen auf dem Globus / der 3D-Karte ein oder aus."
}
func (s *GeoLayerEnableSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *GeoLayerEnableSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"layer":   map[string]interface{}{"type": "string", "enum": []string{"aircraft", "military", "satellites", "vessels", "cctv", "fires", "earthquakes", "daynight", "borders", "all"}},
			"enabled": map[string]interface{}{"type": "boolean", "description": "Ebene aktivieren (true) oder deaktivieren (false)"},
		},
		"required":             []string{"layer", "enabled"},
		"additionalProperties": false,
	}
}

func (s *GeoLayerEnableSkill) Execute(args json.RawMessage) (string, error) {
	var input geoLayerEnableArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("ungültige Argumente: %w", err)
	}

	stateStr := "aktiviert"
	if !input.Enabled {
		stateStr = "deaktiviert"
	}

	return fmt.Sprintf("GEOINT-Ebene '%s' wurde %s.", strings.ToUpper(input.Layer), stateStr), nil
}

// ─── GeoCorrelateSkill ──────────────────────────────────────────────────────
type GeoCorrelateSkill struct{}

type geoCorrelateArgs struct {
	Limit int `json:"limit,omitempty"`
}

func (s *GeoCorrelateSkill) Name() string { return "geo_correlate" }
func (s *GeoCorrelateSkill) Description() string {
	return "Führt eine multisensorische Korrelationsanalyse (Militärflüge + Wärmeanomalien + Erdbeben + OSINT-Meldungen) durch und liefert bewertete Erkenntnisse."
}
func (s *GeoCorrelateSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *GeoCorrelateSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"limit": map[string]interface{}{"type": "integer", "description": "Maximale Anzahl zurückgegebener Korrelationen"},
		},
		"additionalProperties": false,
	}
}

func (s *GeoCorrelateSkill) Execute(args json.RawMessage) (string, error) {
	if sharedGeoInt == nil {
		return "", errors.New("GEOINT-Dienst nicht initialisiert")
	}

	var input geoCorrelateArgs
	_ = json.Unmarshal(args, &input)
	if input.Limit <= 0 {
		input.Limit = 10
	}

	correlations := sharedGeoInt.Bus().ListCorrelations(input.Limit)
	if len(correlations) == 0 {
		return "Keine aktiven multisensorischen Anomalie-Korrelationen im aktuellen Zeitfenster festgestellt.", nil
	}

	data, err := json.MarshalIndent(correlations, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ─── GeoCCTVLookupSkill ─────────────────────────────────────────────────────
type GeoCCTVLookupSkill struct{}

type geoCCTVLookupArgs struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	RadiusKm  float64 `json:"radius_km,omitempty"`
	Limit     int     `json:"limit,omitempty"`
}

func (s *GeoCCTVLookupSkill) Name() string { return "geo_cctv_lookup" }
func (s *GeoCCTVLookupSkill) Description() string {
	return "Findet öffentliche Überwachungskameras im Umkreis einer Zielkoordinate mit Kamerawinkeln und Blickfeldern."
}
func (s *GeoCCTVLookupSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *GeoCCTVLookupSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"latitude":  map[string]interface{}{"type": "number", "description": "Breitengrad"},
			"longitude": map[string]interface{}{"type": "number", "description": "Längengrad"},
			"radius_km": map[string]interface{}{"type": "number", "description": "Suchradius in km (Standard 50 km)"},
			"limit":     map[string]interface{}{"type": "integer", "description": "Maximale Kameras (Standard 10)"},
		},
		"required":             []string{"latitude", "longitude"},
		"additionalProperties": false,
	}
}

func (s *GeoCCTVLookupSkill) Execute(args json.RawMessage) (string, error) {
	if sharedGeoInt == nil {
		return "", errors.New("GEOINT-Dienst nicht initialisiert")
	}

	var input geoCCTVLookupArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", fmt.Errorf("ungültige Argumente: %w", err)
	}

	if input.RadiusKm <= 0 {
		input.RadiusKm = 50.0
	}
	if input.Limit <= 0 {
		input.Limit = 10
	}

	cameras := sharedGeoInt.Bus().QueryRadius(input.Latitude, input.Longitude, input.RadiusKm, []geoint.GeoEntityType{geoint.TypeCamera}, input.Limit)

	if len(cameras) == 0 {
		return fmt.Sprintf("Keine öffentlichen Kameras im Radius von %.0f km um (%.4f, %.4f) verzeichnet.", input.RadiusKm, input.Latitude, input.Longitude), nil
	}

	data, err := json.MarshalIndent(cameras, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
