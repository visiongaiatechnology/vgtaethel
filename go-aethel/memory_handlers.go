package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func handleMemory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		memories := state.memory.GetAll()
		json.NewEncoder(w).Encode(memories)
		return
	} else if r.Method == http.MethodPost {
		var req struct {
			Content    string     `json:"content"`
			Category   string     `json:"category"`
			Source     string     `json:"source"`
			Consent    bool       `json:"operator_approved"`
			ExpiresAt  *time.Time `json:"expires_at,omitempty"`
			Supersedes string     `json:"supersedes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Content == "" {
			http.Error(w, "Content is required", http.StatusBadRequest)
			return
		}
		if len([]rune(req.Content)) > 12000 || len([]rune(req.Category)) > 80 {
			http.Error(w, "Memory payload exceeds size limit", http.StatusRequestEntityTooLarge)
			return
		}

		if !req.Consent {
			http.Error(w, "Explicit operator approval is required for persistent memory", http.StatusForbidden)
			return
		}
		if req.Source == "" {
			req.Source = "operator"
		}
		if req.Supersedes != "" {
			if err := validateResourceID(req.Supersedes); err != nil {
				http.Error(w, "Invalid supersedes id", http.StatusBadRequest)
				return
			}
		}
		id, err := state.memory.AddWithConsent(req.Content, req.Category, req.Source, req.Consent, req.ExpiresAt, req.Supersedes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "id": id})
		return
	} else if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}
		if err := validateResourceID(id); err != nil {
			http.Error(w, "Invalid id parameter", http.StatusBadRequest)
			return
		}

		deleted := state.memory.Delete(id)
		if deleted {
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Memory not found"})
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleMemoryExplain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if err := validateResourceID(id); err != nil {
		http.Error(w, "Invalid memory id", http.StatusBadRequest)
		return
	}
	entry, explanation, ok := state.memory.Explain(id)
	if !ok {
		http.Error(w, "Memory not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"memory": entry, "why": explanation})
}

func handleMemoryExport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"exported_at": time.Now().UTC(), "memories": state.memory.GetAll()})
}

func handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len([]rune(req.Query)) > 1000 {
		http.Error(w, "Search query exceeds size limit", http.StatusRequestEntityTooLarge)
		return
	}

	results := state.memory.Search(req.Query)
	if results == nil {
		results = []MemoryEntry{}
	}
	json.NewEncoder(w).Encode(results)
}
