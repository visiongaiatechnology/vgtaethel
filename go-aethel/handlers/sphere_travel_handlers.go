// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/handlers/sphere_travel_handlers.go
// Purpose: HTTP Handlers for Sphere Trip Planning and Global Watch Intelligence Bridge

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go-aethel/sphere"
)

// HandleSphereTrips handles listing and creating trips.
func HandleSphereTrips(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	getSphereService() // Ensure init
	eng := globalTravelEng

	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		if id != "" {
			trip, err := eng.GetTrip(id)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(trip)
			return
		}

		trips, err := eng.ListTrips()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"trips": trips, "count": len(trips)})

	case http.MethodPost:
		var incoming sphere.TripObject
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid trip payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		created, err := eng.CreateTrip(&incoming)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(created)

	case http.MethodPut:
		var incoming sphere.TripObject
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid trip payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		if err := eng.UpdateTrip(&incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"id query parameter required"}`, http.StatusBadRequest)
			return
		}
		deleted, err := eng.DeleteTrip(id)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "deleted": deleted})

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// HandleSphereTripImpact handles evaluating live Global Watch risk alerts against an active trip.
func HandleSphereTripImpact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	getSphereService()
	eng := globalTravelEng

	if r.Method == http.MethodPost {
		var req struct {
			TripID     string `json:"trip_id"`
			Region     string `json:"region"`
			Signal     string `json:"signal"`
			Severity   string `json:"severity"`
			Confidence string `json:"confidence"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		if req.Severity == "" {
			req.Severity = "WATCH"
		}
		if req.Confidence == "" {
			req.Confidence = "HIGH"
		}

		impact, err := eng.CorrelateGlobalWatchIntelligence(req.TripID, req.Region, req.Signal, req.Severity, req.Confidence)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"has_risk": impact != nil,
			"impact":   impact,
		})
		return
	}

	if r.Method == http.MethodGet {
		tripID := strings.TrimSpace(r.URL.Query().Get("trip_id"))
		if tripID == "" {
			http.Error(w, `{"error":"trip_id query parameter required"}`, http.StatusBadRequest)
			return
		}
		trip, err := eng.GetTrip(tripID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"trip_id": trip.ID,
			"title":   trip.Title,
			"risks":   trip.GlobalWatchRisks,
			"count":   len(trip.GlobalWatchRisks),
		})
		return
	}

	http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
}
