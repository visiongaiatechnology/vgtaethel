package main

import (
	"encoding/json"
	"fmt"
)

// --- 3c. SKILL: SET CHECKLIST ---

type SetChecklistSkill struct{}

type SetChecklistArgs struct {
	Tasks []string `json:"tasks"`
}

func (s *SetChecklistSkill) Name() string { return "task_set_checklist" }
func (s *SetChecklistSkill) Description() string {
	return "Erstellt eine Schritt-für-Schritt-Planliste für die aktuelle Aufgabe im Chat Terminal."
}
func (s *SetChecklistSkill) RiskLevel() RiskLevel { return RiskSafe }

func (s *SetChecklistSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"tasks": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Die Liste der geplanten Einzelschritte",
			},
		},
		"required": []string{"tasks"},
	}
}

func (s *SetChecklistSkill) Execute(args json.RawMessage) (string, error) {
	var input SetChecklistArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	var list []map[string]interface{}
	for _, t := range input.Tasks {
		list = append(list, map[string]interface{}{
			"text":   t,
			"status": "pending",
		})
	}

	currentChecklistMu.Lock()
	currentChecklist = list
	currentChecklistMu.Unlock()

	return fmt.Sprintf("Aktionsplan erstellt mit %d Schritten.", len(input.Tasks)), nil
}

// --- 3d. SKILL: UPDATE CHECKLIST ---

type UpdateChecklistSkill struct{}

type UpdateChecklistArgs struct {
	Index  int    `json:"index"`
	Status string `json:"status"`
}

func (s *UpdateChecklistSkill) Name() string { return "task_update_checklist" }
func (s *UpdateChecklistSkill) Description() string {
	return "Aktualisiert den Status eines Schritts in der aktiven Planliste."
}
func (s *UpdateChecklistSkill) RiskLevel() RiskLevel { return RiskSafe }

func (s *UpdateChecklistSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"index":  map[string]interface{}{"type": "integer", "description": "Der 0-basierte Index des Schritts in der Planliste"},
			"status": map[string]interface{}{"type": "string", "enum": []string{"pending", "in_progress", "done"}, "description": "Der neue Status des Schritts"},
		},
		"required": []string{"index", "status"},
	}
}

func (s *UpdateChecklistSkill) Execute(args json.RawMessage) (string, error) {
	var input UpdateChecklistArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	currentChecklistMu.Lock()
	defer currentChecklistMu.Unlock()

	if input.Index < 0 || input.Index >= len(currentChecklist) {
		return "", fmt.Errorf("Index außerhalb des Bereichs: %d (Länge: %d)", input.Index, len(currentChecklist))
	}

	currentChecklist[input.Index]["status"] = input.Status
	return fmt.Sprintf("Schritt %d aktualisiert auf: %s", input.Index, input.Status), nil
}
