package main

// Earth texture path tests live in handlers package.
// Integration smoke: ensure frontend still references the API path.
import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEarthTextureFrontendReferencesAPI(t *testing.T) {
	// Post-split: texture loader lives in osint/texture_atlas.js; entry re-exports from osint_watch.js
	paths := []string{
		filepath.Join("frontend", "modules", "osint", "texture_atlas.js"),
		filepath.Join("frontend", "modules", "osint_watch.js"),
	}
	var combined strings.Builder
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		combined.Write(raw)
		combined.WriteByte('\n')
	}
	s := combined.String()
	if !strings.Contains(s, "/v1/assets/earth-texture") {
		t.Fatal("osint package must load earth texture from API fallback")
	}
	if !strings.Contains(s, "earth_day.jpg") {
		t.Fatal("osint package must reference earth_day.jpg asset")
	}
}
