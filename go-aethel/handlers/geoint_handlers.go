package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-aethel/geoint"
	"go-aethel/intelligence"
)

var sharedGeoIntService *geoint.GeoIntService

// InitGeoInt initializes the shared GEOINT service reference for handlers
func InitGeoInt(service *geoint.GeoIntService) {
	sharedGeoIntService = service
}

func requireGeoIntGET(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet {
		return true
	}
	w.Header().Set("Allow", http.MethodGet)
	intelligence.WriteIntelJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	return false
}

// HandleGeoIntEntities returns filtered or all geospatial entities
func HandleGeoIntEntities(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		intelligence.WriteIntelJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if sharedGeoIntService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GEOINT service unavailable"})
		return
	}
	queryHash := sha256.Sum256([]byte(r.URL.Query().Encode()))
	etag := `"entities-` + strconv.FormatUint(sharedGeoIntService.Bus().Revision(), 10) + `-` + fmt.Sprintf("%x", queryHash[:6]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=5")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	q := r.URL.Query()
	typeParam := q.Get("type")
	hours := 168
	if raw := strings.TrimSpace(q.Get("hours")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 168 {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "hours must be between 1 and 168"})
			return
		}
		hours = parsed
	}
	cutoff := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	// 1. Check if spatial radius query
	latStr := q.Get("lat")
	lonStr := q.Get("lon")
	radStr := q.Get("radius_km")
	hasRadiusQuery := latStr != "" || lonStr != "" || radStr != ""
	if hasRadiusQuery {
		if latStr == "" || lonStr == "" || radStr == "" {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "lat, lon and radius_km must be supplied together"})
			return
		}
		lat, err1 := strconv.ParseFloat(latStr, 64)
		lon, err2 := strconv.ParseFloat(lonStr, 64)
		rad, err3 := strconv.ParseFloat(radStr, 64)
		if err1 != nil || err2 != nil || err3 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 || rad <= 0 || rad > 20000 {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid spatial radius query"})
			return
		}
		var types []geoint.GeoEntityType
		if typeParam != "" {
			types = append(types, geoint.GeoEntityType(strings.ToUpper(typeParam)))
		}
		entities := filterGeoEntitiesSince(sharedGeoIntService.Bus().QueryRadius(lat, lon, rad, types, 0), cutoff, limit)
		_ = json.NewEncoder(w).Encode(map[string]any{"entities": entities, "count": len(entities)})
		return
	}

	// 2. Check if bounding box query
	minLatStr := q.Get("min_lat")
	minLonStr := q.Get("min_lon")
	maxLatStr := q.Get("max_lat")
	maxLonStr := q.Get("max_lon")
	hasBBoxQuery := minLatStr != "" || minLonStr != "" || maxLatStr != "" || maxLonStr != ""
	if hasBBoxQuery {
		if minLatStr == "" || minLonStr == "" || maxLatStr == "" || maxLonStr == "" {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "all bounding box coordinates are required"})
			return
		}
		minLat, err1 := strconv.ParseFloat(minLatStr, 64)
		minLon, err2 := strconv.ParseFloat(minLonStr, 64)
		maxLat, err3 := strconv.ParseFloat(maxLatStr, 64)
		maxLon, err4 := strconv.ParseFloat(maxLonStr, 64)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || minLat < -90 || maxLat > 90 || minLon < -180 || maxLon > 180 || minLat > maxLat || minLon > maxLon {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid bounding box query"})
			return
		}
		var types []geoint.GeoEntityType
		if typeParam != "" {
			types = append(types, geoint.GeoEntityType(strings.ToUpper(typeParam)))
		}
		entities := filterGeoEntitiesSince(sharedGeoIntService.Bus().QueryBBox(minLat, minLon, maxLat, maxLon, types, 0), cutoff, limit)
		_ = json.NewEncoder(w).Encode(map[string]any{"entities": entities, "count": len(entities)})
		return
	}

	// 3. Type query or all
	if typeParam != "" {
		entities := filterGeoEntitiesSince(sharedGeoIntService.Bus().ListByType(geoint.GeoEntityType(strings.ToUpper(typeParam)), 0), cutoff, limit)
		_ = json.NewEncoder(w).Encode(map[string]any{"entities": entities, "count": len(entities)})
		return
	}

	entities := filterGeoEntitiesSince(sharedGeoIntService.Bus().ListAll(0), cutoff, limit)
	_ = json.NewEncoder(w).Encode(map[string]any{"entities": entities, "count": len(entities)})
}

func filterGeoEntitiesSince(entities []geoint.GeoEntity, cutoff time.Time, limit int) []geoint.GeoEntity {
	filtered := make([]geoint.GeoEntity, 0, len(entities))
	for _, entity := range entities {
		if entity.Timestamp.IsZero() || !entity.Timestamp.Before(cutoff) {
			filtered = append(filtered, entity)
			if limit > 0 && len(filtered) >= limit {
				break
			}
		}
	}
	return filtered
}

// HandleGeoIntRelations returns only evidence-bound directed SHADOW relations.
func HandleGeoIntRelations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		intelligence.WriteIntelJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if sharedGeoIntService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GEOINT service unavailable"})
		return
	}

	hours := 72
	if raw := strings.TrimSpace(r.URL.Query().Get("hours")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 168 {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "hours must be between 1 and 168"})
			return
		}
		hours = parsed
	}
	minimumConfidence := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("min_confidence")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > 100 {
			intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "min_confidence must be between 0 and 100"})
			return
		}
		minimumConfidence = parsed
	}
	category := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("category")))
	cutoff := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	relations, synchronizedAt, version := sharedGeoIntService.Relations()
	filtered := make([]geoint.GeoRelation, 0, len(relations))
	for _, relation := range relations {
		if relation.Timestamp.Before(cutoff) || relation.Confidence < minimumConfidence {
			continue
		}
		if category != "" && string(relation.Category) != category {
			continue
		}
		filtered = append(filtered, relation)
	}
	w.Header().Set("Cache-Control", "private, max-age=5")
	if version != "" {
		queryHash := sha256.Sum256([]byte(r.URL.Query().Encode()))
		etag := `"relations-` + version + `-` + fmt.Sprintf("%x", queryHash[:6]) + `"`
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"relations": filtered, "count": len(filtered), "synchronized_at": synchronizedAt, "version": version,
	})
}

// HandleGeoIntContacts returns a categorized 250km Contact Roster for a focal point
func HandleGeoIntContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireGeoIntGET(w, r) {
		return
	}
	if sharedGeoIntService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GEOINT service unavailable"})
		return
	}

	q := r.URL.Query()
	lat, errLat := strconv.ParseFloat(q.Get("lat"), 64)
	lon, errLon := strconv.ParseFloat(q.Get("lon"), 64)
	if errLat != nil || errLon != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "valid lat and lon required"})
		return
	}

	radiusKm, _ := strconv.ParseFloat(q.Get("radius_km"), 64)
	if radiusKm <= 0 {
		radiusKm = 250.0
	}
	if radiusKm > 20000 {
		intelligence.WriteIntelJSON(w, http.StatusBadRequest, map[string]string{"error": "radius_km exceeds boundary"})
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}

	roster := sharedGeoIntService.Bus().GetContactRoster(lat, lon, radiusKm, limit)
	_ = json.NewEncoder(w).Encode(roster)
}

// HandleGeoIntAircraft returns live aircraft ADS-B tracks
func HandleGeoIntAircraft(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireGeoIntGET(w, r) {
		return
	}
	if sharedGeoIntService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GEOINT service unavailable"})
		return
	}

	milOnly := r.URL.Query().Get("mil") == "1" || r.URL.Query().Get("mil") == "true"
	if milOnly {
		entities := sharedGeoIntService.Bus().ListByType(geoint.TypeMilAircraft, 2000)
		_ = json.NewEncoder(w).Encode(map[string]any{"aircraft": entities, "count": len(entities), "military_only": true})
		return
	}

	civil := sharedGeoIntService.Bus().ListByType(geoint.TypeAircraft, 2000)
	mil := sharedGeoIntService.Bus().ListByType(geoint.TypeMilAircraft, 2000)
	all := append(civil, mil...)
	_ = json.NewEncoder(w).Encode(map[string]any{"aircraft": all, "count": len(all)})
}

// HandleGeoIntSatellites returns active satellite orbits and subpoints
func HandleGeoIntSatellites(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireGeoIntGET(w, r) {
		return
	}
	if sharedGeoIntService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GEOINT service unavailable"})
		return
	}

	satellites := sharedGeoIntService.Bus().ListByType(geoint.TypeSatellite, 3000)
	_ = json.NewEncoder(w).Encode(map[string]any{"satellites": satellites, "count": len(satellites)})
}

// HandleGeoIntVessels returns active AIS vessels
func HandleGeoIntVessels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireGeoIntGET(w, r) {
		return
	}
	if sharedGeoIntService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GEOINT service unavailable"})
		return
	}

	vessels := sharedGeoIntService.Bus().ListByType(geoint.TypeVessel, 2000)
	_ = json.NewEncoder(w).Encode(map[string]any{"vessels": vessels, "count": len(vessels)})
}

// HandleGeoIntCCTV returns public camera mesh
func HandleGeoIntCCTV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireGeoIntGET(w, r) {
		return
	}
	if sharedGeoIntService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GEOINT service unavailable"})
		return
	}

	cameras := sharedGeoIntService.Bus().ListByType(geoint.TypeCamera, 1000)
	_ = json.NewEncoder(w).Encode(map[string]any{"cameras": cameras, "count": len(cameras)})
}

// HandleGeoIntCCTVFrame safely proxies a camera snapshot frame
func HandleGeoIntCCTVFrame(w http.ResponseWriter, r *http.Request) {
	if !requireGeoIntGET(w, r) {
		return
	}
	if sharedGeoIntService == nil {
		http.Error(w, "GEOINT service unavailable", http.StatusServiceUnavailable)
		return
	}

	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url query parameter required", http.StatusBadRequest)
		return
	}

	data, contentType, err := sharedGeoIntService.CCTV().FetchFrameSafely(r.Context(), url)
	if err != nil {
		intelligence.WriteIntelJSON(w, http.StatusBadGateway, map[string]string{"error": "camera frame unavailable"})
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=5")
	_, _ = w.Write(data)
}

// HandleGeoIntHazards returns live USGS earthquakes and NASA FIRMS fires
func HandleGeoIntHazards(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireGeoIntGET(w, r) {
		return
	}
	if sharedGeoIntService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GEOINT service unavailable"})
		return
	}

	quakes := sharedGeoIntService.Bus().ListByType(geoint.TypeEarthquake, 500)
	fires := sharedGeoIntService.Bus().ListByType(geoint.TypeFire, 1000)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"earthquakes": quakes,
		"fires":       fires,
		"total":       len(quakes) + len(fires),
	})
}

// HandleGeoIntCorrelations returns multi-sensor correlated findings
func HandleGeoIntCorrelations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireGeoIntGET(w, r) {
		return
	}
	if sharedGeoIntService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GEOINT service unavailable"})
		return
	}

	correlations := sharedGeoIntService.Bus().ListCorrelations(50)
	_ = json.NewEncoder(w).Encode(map[string]any{"correlations": correlations, "count": len(correlations)})
}

// HandleGeoIntStatus returns system health metrics
func HandleGeoIntStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !requireGeoIntGET(w, r) {
		return
	}
	if sharedGeoIntService == nil {
		intelligence.WriteIntelJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "GEOINT service unavailable"})
		return
	}

	status := sharedGeoIntService.Status()
	_ = json.NewEncoder(w).Encode(status)
}
