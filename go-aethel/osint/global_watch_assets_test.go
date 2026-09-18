package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"os"
	"strings"
	"testing"
)

func TestGlobalWatchUsesLocalCesiumAndDemandRendering(t *testing.T) {
	parts := []string{
		"../frontend/modules/osint/globe_render.js",
		"../frontend/modules/geo_renderer/enhanced_cesium_renderer.js",
		"../frontend/modules/geo_renderer/geo_manager.js",
		"../frontend/modules/geo_renderer/render_governor.js",
	}
	var b strings.Builder
	for _, p := range parts {
		chunk, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		b.Write(chunk)
		b.WriteByte('\n')
	}
	code := b.String()
	for _, required := range []string{"sharedGeoManager.flyTo", "requestRenderMode: true", "baseLayer: false", "RenderGovernor"} {
		if !strings.Contains(code, required) {
			t.Fatalf("global watch renderer is missing %q", required)
		}
	}
	if strings.Contains(code, "renderTicker") || strings.Contains(code, "SovereignGlobeRenderer") || strings.Contains(code, "drawPureLocalGlobe") {
		t.Fatal("global watch must use Cesium demand rendering without a legacy globe path")
	}
	navigation, err := os.ReadFile("../frontend/index.html")
	if err != nil {
		t.Fatalf("read application navigation: %v", err)
	}
	html := string(navigation)
	if !strings.Contains(html, `window.CESIUM_BASE_URL = 'vendor/cesium/'`) || !strings.Contains(html, `src="vendor/cesium/Cesium.js"`) {
		t.Fatal("Cesium runtime must be bundled locally")
	}
	for _, required := range []string{"nav-btn-sphere", "nav-btn-memory", "nav-btn-personal"} {
		if !strings.Contains(html, required) {
			t.Fatalf("workspace navigation is missing %q", required)
		}
	}
}
