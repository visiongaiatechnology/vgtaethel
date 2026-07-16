package handlers

import (
	"go-aethel/security"
	"encoding/json"
	"net/http"
)

func handleSecrets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodGet {
		list := state.vault.List()
		if list == nil {
			list = []security.SecretItem{}
		}
		json.NewEncoder(w).Encode(list)
		return
	}

	if r.Method == http.MethodPost {
		var item security.SecretItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if item.ID == "" || item.Token == "" {
			http.Error(w, "ID and Token are required", http.StatusBadRequest)
			return
		}
		if err := security.ValidateResourceID(item.ID); err != nil {
			http.Error(w, "Invalid secret ID", http.StatusBadRequest)
			return
		}
		if len(item.Token) > 8192 {
			http.Error(w, "Secret token exceeds size limit", http.StatusRequestEntityTooLarge)
			return
		}
		err := state.vault.Add(item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "ID is required", http.StatusBadRequest)
			return
		}
		if err := security.ValidateResourceID(id); err != nil {
			http.Error(w, "Invalid secret ID", http.StatusBadRequest)
			return
		}
		err := state.vault.Delete(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
