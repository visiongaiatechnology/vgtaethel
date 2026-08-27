package handlers

import (
	"net/http"
	"path/filepath"
	"strings"
)

func HandleEarthTexture(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path, err := resolveEarthTexturePath()
	if err != nil {
		http.Error(w, `{"error":"earth texture not found — place earth_day_8k.jpg / earth_day.jpg in frontend/assets or 1.jpg in project root"}`, http.StatusNotFound)
		return
	}
	// Basic content-type; path is always jpg/jpeg in our candidates
	ct := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(path), ".png") {
		ct = "image/png"
	}
	w.Header().Set("Content-Type", ct)
	// Local high-res maps can be multi-MB; cache aggressively in the desktop WebView.
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	w.Header().Set("X-Aethel-Earth-Path", filepath.Base(path))
	http.ServeFile(w, r, path)
}
