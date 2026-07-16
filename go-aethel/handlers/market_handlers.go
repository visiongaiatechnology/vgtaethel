package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-aethel/skills"
)

func handleMarketQuotes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	requested := strings.Split(strings.TrimSpace(r.URL.Query().Get("symbols")), ",")
	if len(requested) == 1 && requested[0] == "" {
		requested = nil
	}
	if len(requested) > 4 {
		http.Error(w, "too many market symbols", http.StatusBadRequest)
		return
	}
	quotes, err := skills.LookupMarketQuotes(r.Context(), requested)
	if err != nil {
		http.Error(w, "market data unavailable", http.StatusBadGateway)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"quotes": quotes, "cache_seconds": 30})
}
