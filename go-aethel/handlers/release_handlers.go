package handlers

import (
	"encoding/json"
	"net/http"

	"go-aethel/system"
)

func HandleReleaseInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"product": system.ProductName, "version": system.ProductVersion, "update_mode": "signed_manifest_manual_install"})
}

func HandleReleasePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		preferences, err := state.release.Preferences()
		if err != nil {
			http.Error(w, "Release preferences unavailable", http.StatusServiceUnavailable)
			return
		}
		json.NewEncoder(w).Encode(preferences)
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var preferences system.ReleasePreferences
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&preferences); err != nil || state.release.SavePreferences(preferences) != nil {
		http.Error(w, "Invalid release preferences", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func HandleReleaseCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result, err := state.release.CheckForUpdate()
	if err != nil {
		http.Error(w, "Update check unavailable", http.StatusServiceUnavailable)
		return
	}
	json.NewEncoder(w).Encode(result)
}
