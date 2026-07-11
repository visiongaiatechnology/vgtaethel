package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// --- SKILL: EXTERNAL AGENT HANDOFF ---

type ExternalAgentHandoffSkill struct{}

type HandoffArgs struct {
	AgentName      string `json:"agent_name"` // "codex" | "chatgpt" | "gemini" | "cursor"
	Objective      string `json:"objective"`
	ProjectContext string `json:"project_context,omitempty"`
	FilesContext   string `json:"files_context,omitempty"`
}

func (s *ExternalAgentHandoffSkill) Name() string { return "agent_handoff" }
func (s *ExternalAgentHandoffSkill) Description() string {
	return "Übergibt eine strukturierte Aufgabe an einen externen Agenten (z.B. Codex, ChatGPT, Gemini, Cursor). Erstellt einen detaillierten Übergabe-Prompt und öffnet das Agenten-Interface."
}
func (s *ExternalAgentHandoffSkill) RiskLevel() RiskLevel { return RiskModerate }

func (s *ExternalAgentHandoffSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agent_name":      map[string]interface{}{"type": "string", "enum": []string{"codex", "chatgpt", "gemini", "cursor"}, "description": "Name des Ziel-Agenten"},
			"objective":       map[string]interface{}{"type": "string", "description": "Konkrete Aufgabe für den externen Agenten"},
			"project_context": map[string]interface{}{"type": "string", "description": "Übergeordneter Projektkontext"},
			"files_context":   map[string]interface{}{"type": "string", "description": "Relevante Code-Auszüge oder Dateiinhalte"},
		},
		"required": []string{"agent_name", "objective"},
	}
}

func (s *ExternalAgentHandoffSkill) Execute(args json.RawMessage) (string, error) {
	var input HandoffArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	// Generate structured markdown prompt
	handoffPrompt := fmt.Sprintf(`# VGT AETHEL AGENT HANDOFF PROTOCOL
## TARGET: %s
## OBJECTIVE:
%s

## PROJECT CONTEXT:
%s

## FILES & CODEBASE CONTEXT:
%s

---
Bitte bearbeiten Sie diese Aufgabe autonom und liefern Sie nach Abschluss eine Zusammenfassung an den Operator.
`, strings.ToUpper(input.AgentName), input.Objective, input.ProjectContext, input.FilesContext)

	// Save to local workspace so frontend can render it
	payloadPath := "./vgt_workspace/handoff_payload.md"
	_ = os.MkdirAll(filepath.Dir(payloadPath), 0700)
	err := os.WriteFile(payloadPath, []byte(handoffPrompt), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to save handoff payload: %v", err)
	}

	// Open external application target
	var openURL string
	switch input.AgentName {
	case "chatgpt":
		openURL = "https://chatgpt.com"
	case "gemini":
		openURL = "https://gemini.google.com"
	case "codex", "cursor":
		openURL = "cursor" // custom application keyword to start Cursor
	default:
		openURL = "https://chatgpt.com"
	}

	LogKernelActivity("HANDOFF_START", input.AgentName, "PENDING")

	if openURL == "cursor" {
		cmd := exec.Command(trustedExecutable("cmd.exe"), "/c", "start", "cursor")
		_ = cmd.Start()
	} else {
		cmd := exec.Command(trustedExecutable("cmd.exe"), "/c", "start", openURL)
		_ = cmd.Start()
	}

	LogKernelActivity("HANDOFF", input.AgentName, "SUCCESS")

	return fmt.Sprintf("HANDOFF: Übergabe-Protokoll generiert in './vgt_workspace/handoff_payload.md'. Interface für '%s' wurde geöffnet.", input.AgentName), nil
}
