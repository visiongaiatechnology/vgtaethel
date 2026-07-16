package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-aethel/skills"
)

// handleWeather is the widget counterpart to weather_lookup. It shares the
// same constrained city-only backend and therefore cannot become a generic
// outbound request proxy.
func handleWeather(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	city := strings.TrimSpace(r.URL.Query().Get("city"))
	snapshot, err := skills.LookupWeather(city)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(snapshot)
}
