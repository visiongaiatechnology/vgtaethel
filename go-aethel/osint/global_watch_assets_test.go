package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"encoding/json"
	"image/jpeg"
	"os"
	"strings"
	"testing"
)

func TestGlobalWatchEmbedsLocalWorldAtlasAndUsesDemandRendering(t *testing.T) {
	texture, err := os.Open("../frontend/assets/earth_day_8k.jpg")
	if err != nil {
		t.Fatalf("open equirectangular earth texture: %v", err)
	}
	textureConfig, err := jpeg.DecodeConfig(texture)
	_ = texture.Close()
	if err != nil {
		t.Fatalf("decode earth texture header: %v", err)
	}
	if textureConfig.Width < 4096 || textureConfig.Width != textureConfig.Height*2 {
		t.Fatalf("earth texture must be high-resolution 2:1 equirectangular, got %dx%d", textureConfig.Width, textureConfig.Height)
	}

	raw, err := os.ReadFile("../frontend/assets/world-atlas-110m.topojson")
	if err != nil {
		t.Fatalf("read local world atlas: %v", err)
	}
	var atlas struct {
		Type    string `json:"type"`
		Objects map[string]struct {
			Geometries []json.RawMessage `json:"geometries"`
		} `json:"objects"`
	}
	if err := json.Unmarshal(raw, &atlas); err != nil {
		t.Fatalf("decode local world atlas: %v", err)
	}
	if atlas.Type != "Topology" || len(atlas.Objects["countries"].Geometries) < 150 {
		t.Fatalf("local world atlas is incomplete: type=%q countries=%d", atlas.Type, len(atlas.Objects["countries"].Geometries))
	}
	// Post-split: atlas + render live under frontend/modules/osint/*
	parts := []string{
		"../frontend/modules/osint/texture_atlas.js",
		"../frontend/modules/osint/globe_render.js",
		"../frontend/modules/osint/state.js",
		"../frontend/modules/osint_watch.js",
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
	for _, required := range []string{"world-atlas-110m.topojson", "decodeWorldAtlas", "requestGlobeRender"} {
		if !strings.Contains(code, required) {
			t.Fatalf("global watch renderer is missing %q", required)
		}
	}
	if strings.Contains(code, "requestAnimationFrame(renderLoop)") {
		t.Fatal("global watch must not keep a permanent frame loop")
	}
	if !strings.Contains(code, "localGlobeRotX") || !strings.Contains(code, "DRAG 2-AXIS ROTATE") {
		t.Fatal("global watch must expose two-axis globe control")
	}
	navigation, err := os.ReadFile("../frontend/index.html")
	if err != nil {
		t.Fatalf("read application navigation: %v", err)
	}
	for _, required := range []string{"nav-btn-sphere", "nav-btn-memory", "nav-btn-personal"} {
		if !strings.Contains(string(navigation), required) {
			t.Fatalf("workspace navigation is missing %q", required)
		}
	}
}
