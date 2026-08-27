package handlers

import (
	"encoding/json"
	"errors"
	"go-aethel/mailbox"
	"net/http"
	"os"
)

func handleMailConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	service := mailbox.SharedService
	if service == nil {
		http.Error(w, "mail service unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg, err := service.LoadConfig()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				_ = json.NewEncoder(w).Encode(map[string]any{"configured": false})
				return
			}
			http.Error(w, "mail configuration unavailable", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"configured": true, "config": cfg})
	case http.MethodPut:
		var request struct {
			Config   mailbox.Config `json:"config"`
			Password string         `json:"password,omitempty"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&request) != nil {
			http.Error(w, "invalid mail configuration", http.StatusBadRequest)
			return
		}
		if err := service.SaveConfig(request.Config, request.Password); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "saved", "configured": true})
	case http.MethodDelete:
		if err := service.DeleteConfig(); err != nil {
			http.Error(w, "mail configuration could not be removed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "removed"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleMailTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if mailbox.SharedService == nil {
		http.Error(w, "mail service unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := mailbox.SharedService.TestConnection(); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "healthy"})
}
