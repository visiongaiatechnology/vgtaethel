package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func handleSecurityLeases(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		leases := state.leases.GetLeases()
		json.NewEncoder(w).Encode(leases)
		return
	} else if r.Method == http.MethodPost {
		var req struct {
			Capability      string `json:"capability"`
			DurationMinutes int    `json:"duration_minutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		allowedCapabilities := map[Capability]bool{
			CapSysExec: true, CapFsRead: true, CapFsWrite: true, CapFsMount: true,
			CapGuiMoveMouse: true, CapGuiClick: true, CapGuiType: true, CapGuiPressKey: true,
			CapMediaControl: true, CapBrowserOpen: true, CapBrowserCtl: true, CapScreenRead: true,
			CapMemoryRead: true, CapMemoryWrite: true, CapCodexHandoff: true,
		}
		capability := Capability(req.Capability)
		if !allowedCapabilities[capability] || req.DurationMinutes < 1 || req.DurationMinutes > 60 {
			http.Error(w, "Invalid lease capability or duration", http.StatusBadRequest)
			return
		}

		leaseID := fmt.Sprintf("lease_%d", time.Now().UnixNano())
		lease := PermissionLease{
			LeaseID:             leaseID,
			Capability:          capability,
			CreatedAt:           time.Now(),
			ExpiresAt:           time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute),
			RequiresVisibleMode: true,
			Revocable:           true,
			ApprovedBy:          "user",
			ApprovalMethod:      "ui",
		}

		if lease.Capability == CapSysExec {
			lease.Scope.ForbiddenTargets = []string{"rm", "dd", "mkfs", "format", "shred", "wipe"}
		} else if lease.Capability == CapGuiPressKey {
			lease.Scope.ForbiddenKeys = []string{"alt+f4", "win+r"}
		}

		err := state.leases.AddLease(lease)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"lease_id": leaseID,
			"expires":  lease.ExpiresAt.Format(time.RFC3339),
		})
		return
	} else if r.Method == http.MethodDelete || (r.Method == http.MethodGet && r.URL.Query().Get("action") == "revoke") {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		revoked := state.leases.RevokeLease(id)
		if revoked {
			json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Lease revoked successfully"})
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Lease not found"})
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleSecurityAudit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logs := state.audit.GetLogs()
	json.NewEncoder(w).Encode(logs)
}

func handleSecurityStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	leases := state.leases.GetLeases()
	status := "ACTIVE"
	auditTampered, lastAuditHash := state.audit.Status()
	if auditTampered {
		status = "TAMPERED"
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"firewall_status":     "ACTIVE",
		"active_leases_count": len(leases),
		"audit_tampered":      auditTampered,
		"security_status":     status,
		"last_activity_hash":  lastAuditHash,
	})
}
