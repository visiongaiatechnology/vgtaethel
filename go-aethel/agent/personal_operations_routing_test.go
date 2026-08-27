package agent

import "testing"

func TestPersonalOperationsCapabilityRouting(t *testing.T) {
	tests := []string{
		"Weck mich morgen um 07:00 Uhr",
		"Erstelle mir einen Tagesplan",
		"Öffne das Wetter Popup für Köln",
	}
	for _, objective := range tests {
		run := AgentRun{Objective: objective, ProfileID: "personal_assistant"}
		if !containsString(requiredExecutionEffects(run), "personal_operations") {
			t.Fatalf("personal operations effect missing for %q: %v", objective, requiredExecutionEffects(run))
		}
		if !containsString(toolAllowlistForRun(run), "personal_operations") {
			t.Fatalf("personal operations tool missing for %q", objective)
		}
	}
}
