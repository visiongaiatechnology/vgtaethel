package main

// STATUS: DIAMANT VGT SUPREME

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

//go:embed frontend/modules/geo_renderer/quality_profiles.js
var geoQualityProfilesJS []byte

//go:embed frontend/modules/global_watch_preferences.js
var globalWatchLODPreferencesJS []byte

func TestGeoQualityProfilesClampOperatorOverrides(t *testing.T) {
	vm := goja.New()
	source := strings.ReplaceAll(string(geoQualityProfilesJS), "export ", "")
	if _, err := vm.RunString(source); err != nil {
		t.Fatalf("load quality profiles: %v", err)
	}
	normalize, ok := goja.AssertFunction(vm.Get("normalizeGeoQualityProfile"))
	if !ok {
		t.Fatal("quality profile normalizer is not callable")
	}
	value, err := normalize(goja.Undefined(), vm.ToValue("power_saver"), vm.ToValue(map[string]any{
		"targetFPS": 500, "maxVisibleEntities": -10, "resolutionScale": 8,
	}))
	if err != nil {
		t.Fatalf("normalize profile: %v", err)
	}
	profile := value.ToObject(vm)
	if got := profile.Get("targetFPS").ToInteger(); got != 60 {
		t.Fatalf("target FPS boundary failed: %d", got)
	}
	if got := profile.Get("maxVisibleEntities").ToInteger(); got != 100 {
		t.Fatalf("entity budget boundary failed: %d", got)
	}
	if got := profile.Get("resolutionScale").ToFloat(); got != 1.5 {
		t.Fatalf("resolution scale boundary failed: %f", got)
	}
}

func TestGlobalWatchLODPreferencesPersistAndClamp(t *testing.T) {
	vm := goja.New()
	storage := map[string]string{}
	if err := vm.Set("localStorage", map[string]any{
		"getItem": func(key string) any {
			if value, ok := storage[key]; ok {
				return value
			}
			return nil
		},
		"setItem": func(key, value string) { storage[key] = value },
	}); err != nil {
		t.Fatal(err)
	}
	source := strings.ReplaceAll(string(globalWatchLODPreferencesJS), "export ", "")
	if _, err := vm.RunString(source); err != nil {
		t.Fatalf("load Global Watch preferences: %v", err)
	}
	save, ok := goja.AssertFunction(vm.Get("saveGlobalWatchPreferences"))
	if !ok {
		t.Fatal("preference saver is not callable")
	}
	value, err := save(goja.Undefined(), vm.ToValue(map[string]any{
		"riskOverlayMinAltitude": 999999999,
		"satelliteMinAltitude":   2200000,
		"cameraMaxAltitude":      -5,
		"relationMaxAltitude":    16000000,
	}))
	if err != nil {
		t.Fatal(err)
	}
	preferences := value.ToObject(vm)
	if got := preferences.Get("riskOverlayMinAltitude").ToInteger(); got != 20000000 {
		t.Fatalf("risk altitude clamp failed: %d", got)
	}
	if got := preferences.Get("cameraMaxAltitude").ToInteger(); got != 5000 {
		t.Fatalf("camera altitude clamp failed: %d", got)
	}
	if storage["aethel.globalWatch.preferences.v7"] == "" {
		t.Fatal("LOD preferences were not persisted")
	}
}

func TestEnhancedGeoRendererHasEvidenceAndLifecycleGuards(t *testing.T) {
	read := func(name string) string {
		content, err := os.ReadFile(filepath.Join("frontend", "modules", "geo_renderer", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}
	enhanced := read("enhanced_cesium_renderer.js")
	manager := read("geo_manager.js")
	entities := read("geo_entity_renderer.js")
	layers := read("geo_layer_manager.js")
	for _, required := range []string{"baseLayer: false", "requestRenderMode: true", "RenderGovernor", "performanceController", "screenHandler.destroy", "layerManager.destroy"} {
		if !strings.Contains(enhanced, required) {
			t.Fatalf("enhanced renderer missing lifecycle marker %q", required)
		}
	}
	if strings.Contains(enhanced, "renderTicker") || strings.Contains(manager, "SovereignGlobeRenderer") {
		t.Fatal("obsolete continuous rendering or sovereign renderer path remains active")
	}
	for _, required := range []string{"applyQualityProfile(profile", "RequestScheduler.maximumRequests", "scene.requestRender()"} {
		if !strings.Contains(layers, required) {
			t.Fatalf("layer manager missing Cesium runtime marker %q", required)
		}
	}
	for _, required := range []string{"AbortController", "If-None-Match", "/v1/geoint/relations?hours=", "replaceChildren"} {
		if !strings.Contains(manager, required) {
			t.Fatalf("GEO manager missing bounded polling marker %q", required)
		}
	}
	if strings.Contains(entities, "BILATERAL_TENSION_VECTORS") || strings.Contains(entities, "tension:") {
		t.Fatal("hardcoded geopolitical claims remain in the active entity renderer")
	}
	for _, required := range []string{"setOptions(next", "visibleCount()", "riskOverlayMinAltitude", "satelliteMinAltitude", "routeActive", "24 * 3600000", "relation.evidence_ids", "relation.origin", "PolylineArrowMaterialProperty", "assessment_status"} {
		if !strings.Contains(entities, required) {
			t.Fatalf("relation renderer missing epistemic marker %q", required)
		}
	}
}
