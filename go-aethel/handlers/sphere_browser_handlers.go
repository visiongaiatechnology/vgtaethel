// STATUS: DIAMANT VGT SUPREME
// Module: go-aethel/handlers/sphere_browser_handlers.go
// Purpose: HTTP Handlers for Sphere Browser 2.0 Multi-Tab Management and Interaction

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go-aethel/security"
	"go-aethel/skills"
)

// HandleSphereBrowserTabs handles listing open tabs and creating new tabs.
func HandleSphereBrowserTabs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	getSphereService()
	sm := globalBrowserMgr

	switch r.Method {
	case http.MethodGet:
		tabs := sm.GetTabs()
		active, _ := sm.GetActiveTab()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"tabs":   tabs,
			"active": active,
			"count":  len(tabs),
		})

	case http.MethodPost:
		var incoming struct {
			URL   string `json:"url"`
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid payload: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		tab := sm.NewTab(incoming.URL, incoming.Title)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(tab)

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// HandleSphereBrowserTabAction handles switching, closing, navigating, and updating tabs.
func HandleSphereBrowserTabAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	getSphereService()
	sm := globalBrowserMgr

	var req struct {
		Action        string `json:"action"` // "switch", "close", "navigate", "control"
		TabID         string `json:"tab_id"`
		URL           string `json:"url,omitempty"`
		Title         string `json:"title,omitempty"`
		AIControlling bool   `json:"ai_controlling,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Invalid action payload: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	switch strings.ToLower(req.Action) {
	case "switch":
		if err := sm.SwitchTab(req.TabID); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "switched"})

	case "close":
		if err := sm.CloseTab(req.TabID); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "closed"})

	case "navigate":
		targetURL := strings.TrimSpace(req.URL)
		if targetURL == "" {
			http.Error(w, `{"error":"URL is required"}`, http.StatusBadRequest)
			return
		}
		if !strings.Contains(targetURL, "://") {
			targetURL = "https://" + targetURL
		}
		// Validate against public HTTPS destination boundary
		resolveCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		dest, err := security.ResolvePublicHTTPSDestination(resolveCtx, targetURL)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Browser target rejected by security policy: %s"}`, err.Error()), http.StatusForbidden)
			return
		}
		if err := sm.UpdateTabState(req.TabID, dest.URL, req.Title, req.AIControlling); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "navigated",
			"target_url": dest.URL,
		})

	case "control":
		if err := sm.UpdateTabState(req.TabID, req.URL, req.Title, req.AIControlling); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	default:
		http.Error(w, fmt.Sprintf(`{"error":"Unsupported browser action: %s"}`, req.Action), http.StatusBadRequest)
	}
}

// HandleSphereBrowserScreenshot proxies the latest browser screenshot.
func HandleSphereBrowserScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := skills.GetLatestScreenshot()
	if err != nil {
		http.Error(w, fmt.Sprintf("Screenshot capture failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
