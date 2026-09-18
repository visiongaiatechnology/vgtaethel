package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-aethel/geoint"
)

func TestGeoIntHandlers(t *testing.T) {
	service := geoint.NewGeoIntService("./vgt_workspace/geoint_test", "")
	InitGeoInt(service)

	// Seed some test entities
	service.Bus().Upsert(geoint.GeoEntity{
		ID:     "air:test-01",
		Type:   geoint.TypeMilAircraft,
		Source: "adsb.lol",
		Label:  "F-16 VIPER",
		Position: geoint.GeoPosition{
			Latitude:  35.6892,
			Longitude: 51.3890,
			Altitude:  9000,
		},
		Classification: "MIL_FIGHTER",
	})

	service.Bus().Upsert(geoint.GeoEntity{
		ID:     "vessel:test-02",
		Type:   geoint.TypeVessel,
		Source: "ais",
		Label:  "TANKER OCEANIC",
		Position: geoint.GeoPosition{
			Latitude:  35.6000,
			Longitude: 51.3000,
		},
		Classification: "TANKER",
	})

	// 1. Test /v1/geoint/entities
	req := httptest.NewRequest("GET", "/v1/geoint/entities", nil)
	rec := httptest.NewRecorder()
	HandleGeoIntEntities(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// 2. Test /v1/geoint/contacts
	req = httptest.NewRequest("GET", "/v1/geoint/contacts?lat=35.6892&lon=51.3890&radius_km=300", nil)
	rec = httptest.NewRecorder()
	HandleGeoIntContacts(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for contacts, got %d", rec.Code)
	}

	// 3. Test /v1/geoint/status
	req = httptest.NewRequest("GET", "/v1/geoint/status", nil)
	rec = httptest.NewRecorder()
	HandleGeoIntStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for status, got %d", rec.Code)
	}
}

func TestHandleGeoIntEntitiesSupportsETagAndRejectsPartialSpatialQueries(t *testing.T) {
	service := geoint.NewGeoIntService("./vgt_workspace/geoint_test", "")
	InitGeoInt(service)
	service.Bus().Upsert(geoint.GeoEntity{ID: "etag-entity", Type: geoint.TypeAircraft})

	first := httptest.NewRecorder()
	HandleGeoIntEntities(first, httptest.NewRequest(http.MethodGet, "/v1/geoint/entities?limit=10", nil))
	if first.Code != http.StatusOK || first.Header().Get("ETag") == "" {
		t.Fatalf("expected cacheable entity response, got %d %q", first.Code, first.Header().Get("ETag"))
	}
	secondRequest := httptest.NewRequest(http.MethodGet, "/v1/geoint/entities?limit=10", nil)
	secondRequest.Header.Set("If-None-Match", first.Header().Get("ETag"))
	second := httptest.NewRecorder()
	HandleGeoIntEntities(second, secondRequest)
	if second.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", second.Code)
	}

	invalid := httptest.NewRecorder()
	HandleGeoIntEntities(invalid, httptest.NewRequest(http.MethodGet, "/v1/geoint/entities?lat=20", nil))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("partial spatial query must be rejected, got %d", invalid.Code)
	}
}
