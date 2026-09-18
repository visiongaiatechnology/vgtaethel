// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/handlers/sphere_research_handlers.go
// Purpose: HTTP Handlers for Sphere Research Desk, Evidence Provenance, and Briefing Synthesis

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go-aethel/sphere"
)

// HandleSphereResearch handles listing and creating research projects.
func HandleSphereResearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	getSphereService()
	eng := globalResEng
	store := getSphereService().GetStore()

	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		if id != "" {
			obj, exists := store.Get(id)
			if !exists || obj.Type != sphere.TypeProject {
				http.Error(w, `{"error":"Research project not found"}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(obj)
			return
		}

		projects := store.List(sphere.TypeProject, "", "")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"projects": projects, "count": len(projects)})

	case http.MethodPost:
		var incoming struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid project payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		created, err := eng.CreateProject(incoming.Title, incoming.Description, incoming.Tags)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(created)

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// HandleSphereResearchClips handles adding and listing evidence clippings.
func HandleSphereResearchClips(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	getSphereService()
	eng := globalResEng

	switch r.Method {
	case http.MethodGet:
		projectID := r.URL.Query().Get("project_id")
		items, err := eng.ListItems(projectID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": items, "count": len(items)})

	case http.MethodPost:
		var incoming struct {
			ProjectID     string `json:"project_id"`
			Title         string `json:"title"`
			SourceURL     string `json:"source_url"`
			SourceTitle   string `json:"source_title"`
			ExtractedText string `json:"extracted_text"`
			KeyTakeaway   string `json:"key_takeaway"`
			Provenance    string `json:"provenance"`
		}
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid clip payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		item, err := eng.AddClip(incoming.ProjectID, incoming.Title, incoming.SourceURL, incoming.SourceTitle, incoming.ExtractedText, incoming.KeyTakeaway, incoming.Provenance)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(item)

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// HandleSphereResearchBriefing generates a synthesized Markdown briefing from project evidence.
func HandleSphereResearchBriefing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	getSphereService()
	eng := globalResEng
	projectID := r.URL.Query().Get("project_id")

	briefing, err := eng.GenerateBriefing(projectID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"project_id": projectID,
		"briefing":   briefing,
	})
}
