package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"crypto/sha256"
	"encoding/json"
	"strings"
	"time"
)

var instructionSignalPatterns = []struct {
	label string
	text  string
}{
	{label: "instruction_override", text: "ignore previous instructions"},
	{label: "instruction_override", text: "ignore all previous"},
	{label: "system_impersonation", text: "system message:"},
	{label: "tool_coercion", text: "call the tool"},
	{label: "secret_exfiltration", text: "reveal your system prompt"},
	{label: "secret_exfiltration", text: "show your api key"},
	{label: "role_impersonation", text: "you are now"},
}

// DetectInstructionSignals marks potentially executable language in untrusted
// source material. It never interprets the content or changes operator policy.
func DetectInstructionSignals(raw string) []string {
	normalized := strings.ToLower(raw)
	seen := make(map[string]bool)
	result := make([]string, 0, 4)
	for _, pattern := range instructionSignalPatterns {
		if strings.Contains(normalized, pattern.text) && !seen[pattern.label] {
			seen[pattern.label] = true
			result = append(result, pattern.label)
		}
	}
	return result
}

func (s *Store) appendCustodyLocked(evidenceID, action, actor, detail string) CustodyEvent {
	now := time.Now().UTC()
	previousHash := ""
	if count := len(s.state.CustodyEvents); count > 0 {
		previousHash = s.state.CustodyEvents[count-1].EventHash
	}
	event := CustodyEvent{
		EvidenceID:   evidenceID,
		Action:       action,
		Actor:        actor,
		Detail:       detail,
		At:           now,
		PreviousHash: previousHash,
	}
	event.ID = "custody-" + contentSHA256(evidenceID + "\x00" + action + "\x00" + now.Format(time.RFC3339Nano))[:24]
	canonical, _ := json.Marshal(struct {
		ID           string    `json:"id"`
		EvidenceID   string    `json:"evidence_id"`
		Action       string    `json:"action"`
		Actor        string    `json:"actor"`
		Detail       string    `json:"detail"`
		At           time.Time `json:"at"`
		PreviousHash string    `json:"previous_hash"`
	}{event.ID, event.EvidenceID, event.Action, event.Actor, event.Detail, event.At, event.PreviousHash})
	digest := sha256.Sum256(canonical)
	event.EventHash = contentSHA256(string(digest[:]))
	s.state.CustodyEvents = append(s.state.CustodyEvents, event)
	return event
}

func VerifyCustodyChain(events []CustodyEvent) bool {
	previousHash := ""
	for _, event := range events {
		if event.PreviousHash != previousHash || event.EventHash == "" {
			return false
		}
		canonical, err := json.Marshal(struct {
			ID           string    `json:"id"`
			EvidenceID   string    `json:"evidence_id"`
			Action       string    `json:"action"`
			Actor        string    `json:"actor"`
			Detail       string    `json:"detail"`
			At           time.Time `json:"at"`
			PreviousHash string    `json:"previous_hash"`
		}{event.ID, event.EvidenceID, event.Action, event.Actor, event.Detail, event.At, event.PreviousHash})
		if err != nil {
			return false
		}
		digest := sha256.Sum256(canonical)
		if contentSHA256(string(digest[:])) != event.EventHash {
			return false
		}
		previousHash = event.EventHash
	}
	return true
}
