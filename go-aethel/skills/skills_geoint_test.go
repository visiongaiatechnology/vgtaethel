package skills

import (
	"encoding/json"
	"testing"

	"go-aethel/geoint"
)

func TestGeoSkillsExecution(t *testing.T) {
	service := geoint.NewGeoIntService("./vgt_workspace/geoint_skills_test", "")
	SetGeoIntService(service)

	// Seed test contacts
	service.Bus().Upsert(geoint.GeoEntity{
		ID:     "air:test-fighter",
		Type:   geoint.TypeMilAircraft,
		Source: "adsb.lol",
		Label:  "VIPER-01",
		Position: geoint.GeoPosition{
			Latitude:  35.6892,
			Longitude: 51.3890,
			Altitude:  8500,
		},
		Classification: "MIL_FIGHTER",
	})

	service.Bus().Upsert(geoint.GeoEntity{
		ID:     "cam:austin-01",
		Type:   geoint.TypeCamera,
		Source: "cctv_mesh",
		Label:  "Austin 6th St",
		Position: geoint.GeoPosition{
			Latitude:  30.2680,
			Longitude: -97.7428,
		},
		Classification: "SURVEILLANCE_CAMERA",
	})

	// 1. Test geo_contacts_nearest
	contactsSkill := &GeoContactsNearestSkill{}
	argsJSON, _ := json.Marshal(map[string]any{
		"latitude":  35.6892,
		"longitude": 51.3890,
		"radius_km": 200,
	})
	res, err := contactsSkill.Execute(argsJSON)
	if err != nil {
		t.Fatalf("geo_contacts_nearest failed: %v", err)
	}
	if len(res) == 0 {
		t.Errorf("expected non-empty roster response")
	}

	// 2. Test geo_focus
	focusSkill := &GeoFocusSkill{}
	focusJSON, _ := json.Marshal(map[string]any{
		"latitude":  35.6892,
		"longitude": 51.3890,
		"zoom":      12.0,
		"label":     "Teheran",
	})
	res, err = focusSkill.Execute(focusJSON)
	if err != nil {
		t.Fatalf("geo_focus failed: %v", err)
	}
	if len(res) == 0 {
		t.Errorf("expected focus confirmation string")
	}

	// 3. Test geo_track
	trackSkill := &GeoTrackSkill{}
	trackJSON, _ := json.Marshal(map[string]any{
		"entity_id": "air:test-fighter",
		"follow":    true,
	})
	res, err = trackSkill.Execute(trackJSON)
	if err != nil {
		t.Fatalf("geo_track failed: %v", err)
	}
	if len(res) == 0 {
		t.Errorf("expected track confirmation string")
	}

	// 4. Test geo_cctv_lookup
	cctvSkill := &GeoCCTVLookupSkill{}
	cctvJSON, _ := json.Marshal(map[string]any{
		"latitude":  30.2680,
		"longitude": -97.7428,
		"radius_km": 20.0,
	})
	res, err = cctvSkill.Execute(cctvJSON)
	if err != nil {
		t.Fatalf("geo_cctv_lookup failed: %v", err)
	}
	if len(res) == 0 {
		t.Errorf("expected camera list response")
	}
}
