// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/handlers/sphere_planner_handlers.go
// Purpose: HTTP Handlers for Sphere Master Planner, Goals, Milestones, and Task Dependencies

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go-aethel/sphere"
)

// HandleSpherePlans handles listing, creating, and updating plans.
func HandleSpherePlans(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	getSphereService()
	eng := globalPlanEng

	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		if id != "" {
			plan, err := eng.GetPlan(id)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(plan)
			return
		}

		plans, err := eng.ListPlans()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"plans": plans, "count": len(plans)})

	case http.MethodPost:
		var incoming sphere.PlanObject
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid plan payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		created, err := eng.CreatePlan(&incoming)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(created)

	case http.MethodPut:
		var incoming sphere.PlanObject
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid plan payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		if err := eng.UpdatePlan(&incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// HandleSpherePlanTask handles adding tasks with dependency validation.
func HandleSpherePlanTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	getSphereService()
	eng := globalPlanEng

	var req struct {
		PlanID string          `json:"plan_id"`
		Task   sphere.PlanTask `json:"task"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Invalid task payload: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if err := eng.AddTask(req.PlanID, req.Task); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
