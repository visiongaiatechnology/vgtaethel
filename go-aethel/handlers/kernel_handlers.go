package handlers

import (
	"encoding/json"
	"net/http"

	"go-aethel/security"
)

func HandleKernelLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	json.NewEncoder(w).Encode(security.KernelLogs())
}
