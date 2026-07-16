package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"go-aethel/intelligence"
	"go-aethel/security"
	"go-aethel/skills"
)

type ToolExecRequest struct {
	Name          string          `json:"name"`
	Skill         string          `json:"skill,omitempty"` // alias used by some UI callers
	Args          json.RawMessage `json:"args"`
	ApprovalToken string          `json:"approval_token,omitempty"`
}

func (p *ToolExecRequest) skillName() string {
	if strings.TrimSpace(p.Name) != "" {
		return strings.TrimSpace(p.Name)
	}
	return strings.TrimSpace(p.Skill)
}

func handleToolExecuteLegacy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload ToolExecRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "Invalid request payload"})
		return
	}

	skill, exists := state.skills.Get(payload.Name)
	if !exists {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "Tool not found"})
		return
	}

	if strings.TrimSpace(payload.Name) == "" || len(payload.Args) == 0 || !json.Valid(payload.Args) {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "Invalid tool request"})
		return
	}

	argsStr := string(payload.Args)
	allowed, decision, report := state.policy.Evaluate(payload.Name, argsStr, false)

	if !allowed {
		if decision == "blocked" {
			security.LogKernelActivity("SECURITY_BLOCK", payload.Name, "BLOCKED")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "security_blocked",
				"risk_score": report.RiskScore,
				"risk_level": report.RiskLevel,
				"threats":    report.Threats,
				"capability": report.Capability,
				"message":    "VGT Firewall: Diese Aktion ist permanent blockiert (FORBIDDEN).",
			})
			return
		} else if decision == "needs_approval" {
			security.LogKernelActivity("SECURITY_WAIT", payload.Name, "WAITING")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "security_intervention",
				"risk_score": report.RiskScore,
				"risk_level": report.RiskLevel,
				"threats":    report.Threats,
				"capability": report.Capability,
				"message":    "VGT Security: Ausführung erfordert Operator-Zustimmung.",
			})
			return
		}
	}

	security.LogKernelActivity("EXECUTE_TOOL", payload.Name, "SUCCESS")

	result, err := skill.Execute(payload.Args)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	skills.CurrentChecklistMu.RLock()
	checklistCopy := make([]map[string]interface{}, len(skills.CurrentChecklist))
	copy(checklistCopy, skills.CurrentChecklist)
	skills.CurrentChecklistMu.RUnlock()

	currentSessionChangesMu.Lock()
	changesCopy := make([]FileChange, len(currentSessionChanges))
	copy(changesCopy, currentSessionChanges)
	currentSessionChangesMu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "success",
		"result":       result,
		"checklist":    checklistCopy,
		"file_changes": changesCopy,
	})
}

func handleToolExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload ToolExecRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.skillName() == "" || len(payload.Args) == 0 || !json.Valid(payload.Args) {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "Invalid tool request"})
		return
	}
	toolName := payload.skillName()
	skill, exists := state.skills.Get(toolName)
	if !exists {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "Tool not found"})
		return
	}
	args := string(payload.Args)
	allowed, decision, report := state.policy.Evaluate(toolName, args, false)
	if !allowed {
		if decision == "blocked" {
			security.LogKernelActivity("SECURITY_BLOCK", toolName, "BLOCKED")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "security_blocked", "risk_score": report.RiskScore, "risk_level": report.RiskLevel,
				"threats": report.Threats, "capability": report.Capability,
				"message": "VGT Firewall blocked this action.",
			})
			return
		}
		if decision != "needs_approval" {
			http.Error(w, "Policy decision unavailable", http.StatusConflict)
			return
		}
		if payload.ApprovalToken == "" {
			if state.approvals == nil {
				http.Error(w, "Approval service unavailable", http.StatusServiceUnavailable)
				return
			}
			grant, token, err := state.approvals.Issue(toolName, args, report.Capability, "")
			if err != nil {
				http.Error(w, "Approval service unavailable", http.StatusServiceUnavailable)
				return
			}
			security.LogKernelActivity("SECURITY_WAIT", toolName, "WAITING")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "security_intervention", "risk_score": report.RiskScore, "risk_level": report.RiskLevel,
				"threats": report.Threats, "capability": report.Capability, "approval_id": grant.ID,
				"approval_token": token, "approval_expires_at": grant.ExpiresAt,
				"message": "VGT Security requires an explicit, one-time operator approval.",
			})
			return
		}
		if state.approvals == nil || state.approvals.Consume(payload.ApprovalToken, toolName, args, report.Capability, "") != nil {
			security.LogKernelActivity("SECURITY_BLOCK", toolName, "INVALID_APPROVAL")
			json.NewEncoder(w).Encode(map[string]string{"status": "security_blocked", "message": "Approval is invalid, expired, or already consumed."})
			return
		}
		_, _ = state.audit.Log("operator", toolName, string(report.Capability), report.RiskLevel, "", "override", "One-time argument-bound approval consumed.", args)
		security.LogKernelActivity("SECURITY_APPROVED_ONCE", toolName, "SUCCESS")
	} else {
		security.LogKernelActivity("EXECUTE_TOOL", toolName, "SUCCESS")
	}

	result, err := skill.Execute(payload.Args)
	// U7: AgentAction audit trail in intelligence.SharedIntelStore (skill name + status; args/result truncated)
	if intelligence.SharedIntelStore != nil {
		status := "completed"
		resText := result
		if err != nil {
			status = "failed"
			resText = err.Error()
		}
		approvedBy := ""
		if payload.ApprovalToken != "" {
			approvedBy = "operator"
		}
		intelligence.SharedIntelStore.RecordAgentAction(toolName, string(payload.Args), resText, approvedBy, status)
	}
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "error", "error": err.Error()})
		return
	}
	skills.CurrentChecklistMu.RLock()
	checklist := append([]map[string]interface{}(nil), skills.CurrentChecklist...)
	skills.CurrentChecklistMu.RUnlock()
	currentSessionChangesMu.Lock()
	changes := append([]FileChange(nil), currentSessionChanges...)
	currentSessionChangesMu.Unlock()
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "result": result, "checklist": checklist, "file_changes": changes})
}
