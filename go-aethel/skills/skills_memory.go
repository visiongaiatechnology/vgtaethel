package skills

import (
	"strings"
	"errors"
	"go-aethel/security"
	"encoding/json"
	"fmt"
	
	"go-aethel/memory"
	"go-aethel/personal"
)

type MemorySaveSkill struct {
	Store *memory.LocalMemoryStore
}

type SaveArgs struct {
	Content          string `json:"content"`
	Category         string `json:"category,omitempty"`
	OperatorApproved bool   `json:"operator_approved"`
	Source           string `json:"source,omitempty"`
}

func (s *MemorySaveSkill) Name() string { return "nexus_save" }
func (s *MemorySaveSkill) Description() string {
	return "Speichert nicht-sensitive Fakten, Entscheidungen, Vorlieben, Workflows oder Projektdaten im Langzeitgedächtnis. Secrets gehören ausschließlich in den Secret Vault."
}
func (s *MemorySaveSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }

func (s *MemorySaveSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content":  map[string]interface{}{"type": "string", "description": "Die Information, die gespeichert werden soll."},
			"category": map[string]interface{}{"type": "string", "description": "Optionaler Bereich/Kategorie (Standard: 'general')"},
		},
		"required": []string{"content"},
	}
}

func (s *MemorySaveSkill) Execute(args json.RawMessage) (string, error) {
	var input SaveArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	content := strings.TrimSpace(input.Content)
	if len([]rune(content)) < 3 || len([]rune(content)) > 4000 {
		return "", errors.New("memory content must contain between 3 and 4000 characters")
	}
	if memory.ContainsSensitiveMemoryData(content) {
		return "", errors.New("sensitive data rejected: store credentials only in the secret vault")
	}
	category := memory.NormalizeMemoryCategory(input.Category)
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "assistant_tool"
	}
	id, err := s.Store.AddWithConsent(content, category, source, input.OperatorApproved, nil, "")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("NEXUS: Information gespeichert. ID: %s", id), nil
}

// --- SKILL: RECALL FROM NEXUS ---

type MemoryRecallSkill struct {
	Store *memory.LocalMemoryStore
}

type PersonalMemorySaveSkill struct {
	Store *personal.PersonalStore
}

type PersonalMemoryRecallSkill struct {
	Store *personal.PersonalStore
}

type RecallArgs struct {
	Query string `json:"query"`
}

func (s *MemoryRecallSkill) Name() string { return "nexus_recall" }
func (s *MemoryRecallSkill) Description() string {
	return "Durchsucht das lokale Langzeitgedächtnis nach relevanten Infos basierend auf Suchbegriffen."
}
func (s *MemoryRecallSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }

func (s *MemoryRecallSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string", "description": "Die Suchanfrage (z.B. 'Wie lautet das Passwort von Projekt Alpha?')"},
		},
		"required": []string{"query"},
	}
}

func (s *MemoryRecallSkill) Execute(args json.RawMessage) (string, error) {
	var input RecallArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	results := s.Store.Search(input.Query)
	if len(results) == 0 {
		return "NEXUS: Keine passenden Erinnerungen gefunden.", nil
	}

	var output []string
	for _, entry := range results {
		output = append(output, fmt.Sprintf("- [NEXUS-DATUM | %s | %s] %s", entry.Category, entry.Timestamp.Format("2006-01-02"), entry.Content))
	}
	return strings.Join(output, "\n"), nil
}

type PersonalMemorySaveArgs struct {
	Type             string  `json:"type"`
	Content          string  `json:"content"`
	Confidence       float64 `json:"confidence,omitempty"`
	OperatorApproved bool    `json:"operator_approved"`
}

func (s *PersonalMemorySaveSkill) Name() string { return "personal_memory_save" }
func (s *PersonalMemorySaveSkill) Description() string {
	return "Speichert eine stabile persönliche Erkenntnis im persönlichen Modus. Nicht für Secrets, flüchtige Stimmungen oder Projektdaten verwenden."
}
func (s *PersonalMemorySaveSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *PersonalMemorySaveSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type":       map[string]interface{}{"type": "string", "description": "Kategorie: identity, preference, routine, goal, relationship, boundary"},
			"content":    map[string]interface{}{"type": "string", "description": "Stabile persönliche Erkenntnis."},
			"confidence": map[string]interface{}{"type": "number", "description": "Konfidenz 0.0 bis 1.0"},
		},
		"required": []string{"type", "content"},
	}
}
func (s *PersonalMemorySaveSkill) Execute(args json.RawMessage) (string, error) {
	var input PersonalMemorySaveArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	cfg, err := s.Store.LoadConfig()
	if err != nil {
		return "", err
	}
	if !cfg.Enabled || !cfg.LearningEnabled {
		return "PERSONAL: Lernmodus ist deaktiviert.", nil
	}
	if !input.OperatorApproved {
		return "", errors.New("personal memory requires explicit operator approval")
	}
	content := personal.ClampPersonalText(input.Content, 2000)
	memType := personal.ClampPersonalText(input.Type, 60)
	if content == "" || memType == "" || personal.LooksSecretLike(content) {
		return "", errors.New("personal memory benötigt type und content")
	}
	mem, err := s.Store.AppendMemory(personal.PersonalMemory{
		Type:       memType,
		Content:    content,
		Confidence: input.Confidence,
		Source:     "assistant",
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("PERSONAL: Erinnerung gespeichert (%s).", mem.ID), nil
}

type PersonalMemoryRecallArgs struct {
	Query string `json:"query"`
}

func (s *PersonalMemoryRecallSkill) Name() string { return "personal_memory_recall" }
func (s *PersonalMemoryRecallSkill) Description() string {
	return "Liest persönliche Erinnerungen aus dem getrennten persönlichen Gedächtnis."
}
func (s *PersonalMemoryRecallSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *PersonalMemoryRecallSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string", "description": "Suchbegriff für persönliche Erinnerungen."},
		},
		"required": []string{"query"},
	}
}
func (s *PersonalMemoryRecallSkill) Execute(args json.RawMessage) (string, error) {
	var input PersonalMemoryRecallArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	memories, err := s.Store.ListMemories()
	if err != nil {
		return "", err
	}
	query := strings.ToLower(strings.TrimSpace(input.Query))
	var out []string
	for _, mem := range memories {
		if query == "" || strings.Contains(strings.ToLower(mem.Content), query) || strings.Contains(strings.ToLower(mem.Type), query) {
			out = append(out, fmt.Sprintf("- [%s] %s (%.0f%%)", mem.Type, mem.Content, mem.Confidence*100))
		}
		if len(out) >= 12 {
			break
		}
	}
	if len(out) == 0 {
		return "PERSONAL: Keine passenden persönlichen Erinnerungen gefunden.", nil
	}
	return strings.Join(out, "\n"), nil
}
