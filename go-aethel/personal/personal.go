package personal

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go-aethel/provider"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-aethel/intelligence"

	"go-aethel/security"
)

type PersonalConfig struct {
	Enabled          bool      `json:"enabled"`
	LearningEnabled  bool      `json:"learning_enabled"`
	HumorLevel       int       `json:"humor_level"`
	HonestyLevel     int       `json:"honesty_level"`
	InitiativeLevel  int       `json:"initiative_level"`
	PrimaryModel     string    `json:"primary_model"`
	FallbackModel    string    `json:"fallback_model"`
	WakeWord         string    `json:"wake_word"`
	StartupBriefing  bool      `json:"startup_briefing"`
	StartupReadAloud bool      `json:"startup_read_aloud"`
	ForbiddenTopics  []string  `json:"forbidden_topics,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type PersonalProfile struct {
	DisplayName     string    `json:"display_name"`
	PreferredTone   string    `json:"preferred_tone"`
	AssistantStyle  string    `json:"assistant_style"`
	Interests       []string  `json:"interests"`
	Goals           []string  `json:"goals"`
	Notes           string    `json:"notes"`
	LocationCity    string    `json:"location_city"`
	LocationCountry string    `json:"location_country"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PersonalMemory struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Content    string    `json:"content"`
	Confidence float64   `json:"confidence"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"created_at"`
	ConsentAt  time.Time `json:"consent_at"`
}

type PersonalSetupQuestion struct {
	ID       string `json:"id"`
	Target   string `json:"target"`
	Question string `json:"question"`
}

type PersonalStore struct {
	mu          sync.RWMutex
	baseDir     string
	configPath  string
	profilePath string
	memoryPath  string
}

func NewPersonalStore(baseDir string) *PersonalStore {
	return &PersonalStore{
		baseDir:     baseDir,
		configPath:  filepath.Join(baseDir, "config.json"),
		profilePath: filepath.Join(baseDir, "profile.json"),
		memoryPath:  filepath.Join(baseDir, "memories.jsonl"),
	}
}

func (ps *PersonalStore) ensureDir() error {
	return os.MkdirAll(ps.baseDir, 0700)
}

func defaultPersonalConfig() PersonalConfig {
	return PersonalConfig{
		Enabled:         false,
		LearningEnabled: false,
		HumorLevel:      35,
		HonestyLevel:    90,
		InitiativeLevel: 60,
		WakeWord:        "aethel",
		UpdatedAt:       time.Now().UTC(),
	}
}

func (ps *PersonalStore) LoadConfig() (PersonalConfig, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	cfg := defaultPersonalConfig()
	data, _, err := security.ReadSealedFile(ps.configPath)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (ps *PersonalStore) SaveConfig(cfg PersonalConfig) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if err := ps.ensureDir(); err != nil {
		return err
	}
	cfg.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return security.WriteSealedFile(ps.configPath, data)
}

func (ps *PersonalStore) LoadProfile() (PersonalProfile, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	profile := PersonalProfile{}
	data, _, err := security.ReadSealedFile(ps.profilePath)
	if errors.Is(err, os.ErrNotExist) {
		return profile, nil
	}
	if err != nil {
		return profile, err
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return profile, err
	}
	return profile, nil
}

func (ps *PersonalStore) SaveProfile(profile PersonalProfile) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if err := ps.ensureDir(); err != nil {
		return err
	}
	now := time.Now().UTC()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return security.WriteSealedFile(ps.profilePath, data)
}

func newPersonalMemoryID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "pmem_" + hex.EncodeToString(b[:]), nil
}

func (ps *PersonalStore) AppendMemory(mem PersonalMemory) (PersonalMemory, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if err := ps.ensureDir(); err != nil {
		return mem, err
	}
	if mem.ID == "" {
		id, err := newPersonalMemoryID()
		if err != nil {
			return mem, err
		}
		mem.ID = id
	}
	if mem.CreatedAt.IsZero() {
		mem.CreatedAt = time.Now().UTC()
	}
	if mem.ConsentAt.IsZero() {
		mem.ConsentAt = time.Now().UTC()
	}
	if mem.Confidence <= 0 || mem.Confidence > 1 {
		mem.Confidence = 0.75
	}
	memories, err := ps.listMemoriesUnlocked()
	if err != nil {
		return mem, err
	}
	memories = append(memories, mem)
	if err := ps.saveMemoriesUnlocked(memories); err != nil {
		return mem, err
	}
	return mem, nil
}

func (ps *PersonalStore) ListMemories() ([]PersonalMemory, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.listMemoriesUnlocked()
}

func (ps *PersonalStore) DeleteMemory(id string) error {
	if err := security.ValidateResourceID(id); err != nil {
		return err
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	memories, err := ps.listMemoriesUnlocked()
	if err != nil {
		return err
	}
	updated := make([]PersonalMemory, 0, len(memories))
	for _, mem := range memories {
		if mem.ID != id {
			updated = append(updated, mem)
		}
	}
	return ps.saveMemoriesUnlocked(updated)
}

func (ps *PersonalStore) UpdateMemory(mem PersonalMemory) (PersonalMemory, error) {
	if err := security.ValidateResourceID(mem.ID); err != nil {
		return mem, err
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	memories, err := ps.listMemoriesUnlocked()
	if err != nil {
		return mem, err
	}
	found := false
	for i, existing := range memories {
		if existing.ID != mem.ID {
			continue
		}
		found = true
		mem.CreatedAt = existing.CreatedAt
		if mem.Source == "" {
			mem.Source = existing.Source
		}
		if mem.Confidence <= 0 || mem.Confidence > 1 {
			mem.Confidence = existing.Confidence
		}
		memories[i] = mem
		break
	}
	if !found {
		return mem, os.ErrNotExist
	}
	if err := ps.saveMemoriesUnlocked(memories); err != nil {
		return mem, err
	}
	return mem, nil
}

func (ps *PersonalStore) listMemoriesUnlocked() ([]PersonalMemory, error) {
	data, sealed, err := security.ReadSealedFile(ps.memoryPath)
	if errors.Is(err, os.ErrNotExist) {
		return []PersonalMemory{}, nil
	}
	if err != nil {
		return nil, err
	}
	var memories []PersonalMemory
	if sealed {
		if err := json.Unmarshal(data, &memories); err != nil {
			return nil, err
		}
		return memories, nil
	}
	// Legacy JSONL migration path: legacy data is only read here and becomes
	// sealed on the next append/update/delete operation.
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var mem PersonalMemory
		if err := json.Unmarshal([]byte(line), &mem); err == nil {
			memories = append(memories, mem)
		}
	}
	return memories, nil
}

func (ps *PersonalStore) saveMemoriesUnlocked(memories []PersonalMemory) error {
	if err := ps.ensureDir(); err != nil {
		return err
	}
	data, err := json.Marshal(memories)
	if err != nil {
		return err
	}
	return security.WriteSealedFile(ps.memoryPath, data)
}

func ClampPersonalText(value string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > maxRunes {
		runes = runes[:maxRunes]
	}
	return string(runes)
}

// ClampPersonalLevel bounds a personal preference level to 0..100.
func ClampPersonalLevel(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

// BuildPersonalSetupQuestions returns setup questions (model or heuristic fallback).
func BuildPersonalSetupQuestions(cfg PersonalConfig) ([]PersonalSetupQuestion, string) {
	if questions, err := generatePersonalSetupQuestionsWithModels(cfg); err == nil && len(questions) > 0 {
		return questions, "model"
	}
	return defaultPersonalSetupQuestions(), "heuristic_fallback"
}

func defaultPersonalSetupQuestions() []PersonalSetupQuestion {
	return []PersonalSetupQuestion{
		{ID: "display_name", Target: "personal-display-name", Question: "Wie soll Aethel dich nennen?"},
		{ID: "preferred_tone", Target: "personal-tone", Question: "Wie soll Aethel mit dir sprechen? Beispiel: locker, direkt, ruhig, motivierend."},
		{ID: "assistant_style", Target: "personal-style", Question: "Was macht einen guten persoenlichen Assistenten fuer dich aus?"},
		{ID: "interests", Target: "personal-interests", Question: "Welche Interessen, Themen oder Medien soll Aethel ueber dich kennen?"},
		{ID: "goals", Target: "personal-goals", Question: "Welche Ziele oder Projekte soll Aethel langfristig im Blick behalten?"},
		{ID: "notes", Target: "personal-notes", Question: "Gibt es Grenzen, No-Gos oder wichtige persoenliche Hinweise?"},
		{ID: "wake_word", Target: "personal-wake-word", Question: "Welches Wake-Word moechtest du nutzen?"},
	}
}

func generatePersonalSetupQuestionsWithModels(cfg PersonalConfig) ([]PersonalSetupQuestion, error) {
	models := []string{strings.TrimSpace(cfg.PrimaryModel), strings.TrimSpace(cfg.FallbackModel)}
	var lastErr error
	for _, modelID := range models {
		if modelID == "" {
			continue
		}
		questions, err := generatePersonalSetupQuestionsWithModel(modelID)
		if err == nil && len(questions) > 0 {
			return questions, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no personal setup question model configured")
	}
	return nil, lastErr
}

func generatePersonalSetupQuestionsWithModel(modelID string) ([]PersonalSetupQuestion, error) {
	prompt := `Du bist Aethels Personal Setup Interview Agent.
Erzeuge ein kurzes deutschsprachiges Setup-Gespraech mit genau 7 Fragen.
Die Fragen muessen stabil persoenliche Profilfelder abdecken, keine Secrets erfragen und nicht nach Passwoertern, Tokens, API Keys oder privaten Schluesseln fragen.
Antworte ausschliesslich als JSON-Array. Kein Markdown.
Erlaubte IDs und Targets:
display_name -> personal-display-name
preferred_tone -> personal-tone
assistant_style -> personal-style
interests -> personal-interests
goals -> personal-goals
notes -> personal-notes
wake_word -> personal-wake-word
Schema:
[{"id":"display_name","target":"personal-display-name","question":"kurze konkrete Frage"}]`

	raw, err := personalModelCompletion(modelID, prompt)
	if err != nil {
		return nil, err
	}
	var decoded []PersonalSetupQuestion
	if err := json.Unmarshal([]byte(extractJSONArray(raw)), &decoded); err != nil {
		return nil, err
	}
	return normalizePersonalSetupQuestions(decoded), nil
}

func normalizePersonalSetupQuestions(input []PersonalSetupQuestion) []PersonalSetupQuestion {
	allowed := map[string]string{
		"display_name":    "personal-display-name",
		"preferred_tone":  "personal-tone",
		"assistant_style": "personal-style",
		"interests":       "personal-interests",
		"goals":           "personal-goals",
		"notes":           "personal-notes",
		"wake_word":       "personal-wake-word",
	}
	seen := make(map[string]bool, len(input))
	out := make([]PersonalSetupQuestion, 0, 7)
	for _, q := range input {
		id := ClampPersonalText(q.ID, 40)
		target, ok := allowed[id]
		question := ClampPersonalText(q.Question, 220)
		if !ok || seen[id] || question == "" || LooksSecretLike(question) {
			continue
		}
		out = append(out, PersonalSetupQuestion{ID: id, Target: target, Question: question})
		seen[id] = true
	}
	for _, fallback := range defaultPersonalSetupQuestions() {
		if seen[fallback.ID] {
			continue
		}
		out = append(out, fallback)
		seen[fallback.ID] = true
		if len(out) >= 7 {
			break
		}
	}
	return out
}

// BuildPersonalProfileFromSetup builds a profile from onboarding answers.
func BuildPersonalProfileFromSetup(cfg PersonalConfig, answers map[string]string) (PersonalProfile, string) {
	if profile, err := generatePersonalProfileWithModels(cfg, answers); err == nil {
		return profile, "model"
	}
	return fallbackPersonalProfileFromSetup(answers), "heuristic_fallback"
}

func fallbackPersonalProfileFromSetup(answers map[string]string) PersonalProfile {
	now := time.Now().UTC()
	return PersonalProfile{
		DisplayName:     ClampPersonalText(answers["display_name"], 80),
		PreferredTone:   ClampPersonalText(answers["preferred_tone"], 120),
		AssistantStyle:  ClampPersonalText(answers["assistant_style"], 300),
		Interests:       splitPersonalList(answers["interests"], 12),
		Goals:           splitPersonalList(answers["goals"], 12),
		Notes:           ClampPersonalText(answers["notes"], 4000),
		LocationCity:    ClampPersonalText(answers["location_city"], 120),
		LocationCountry: ClampPersonalText(answers["location_country"], 120),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func generatePersonalProfileWithModels(cfg PersonalConfig, answers map[string]string) (PersonalProfile, error) {
	models := []string{strings.TrimSpace(cfg.PrimaryModel), strings.TrimSpace(cfg.FallbackModel)}
	var lastErr error
	for _, modelID := range models {
		if modelID == "" {
			continue
		}
		profile, err := generatePersonalProfileWithModel(modelID, answers)
		if err == nil {
			return profile, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no personal setup model configured")
	}
	return PersonalProfile{}, lastErr
}

func generatePersonalProfileWithModel(modelID string, answers map[string]string) (PersonalProfile, error) {
	setupPayload, err := json.Marshal(map[string]string{
		"display_name":    ClampPersonalText(answers["display_name"], 160),
		"preferred_tone":  ClampPersonalText(answers["preferred_tone"], 240),
		"assistant_style": ClampPersonalText(answers["assistant_style"], 800),
		"interests":       ClampPersonalText(answers["interests"], 1000),
		"goals":           ClampPersonalText(answers["goals"], 1000),
		"notes":           ClampPersonalText(answers["notes"], 2000),
	})
	if err != nil {
		return PersonalProfile{}, err
	}
	prompt := `Du bist Aethels Personal Setup Agent.
Erzeuge aus den Setup-Antworten ein stabiles Anfangsprofil.
Speichere KEINE Secrets, API Keys, Passwörter, Tokens oder privaten Schlüssel.
Antworte ausschließlich als JSON-Objekt. Kein Markdown.
Schema:
{"display_name":"Name oder leer","preferred_tone":"kurze Tonbeschreibung","assistant_style":"kurze Assistenzbeschreibung","interests":["max 12 stabile Interessen"],"goals":["max 12 stabile Ziele"],"notes":"wichtige Grenzen, No-Gos oder Beziehungshinweise"}

Setup-Antworten:
` + string(setupPayload)

	raw, err := personalModelCompletion(modelID, prompt)
	if err != nil {
		return PersonalProfile{}, err
	}
	var decoded struct {
		DisplayName    string   `json:"display_name"`
		PreferredTone  string   `json:"preferred_tone"`
		AssistantStyle string   `json:"assistant_style"`
		Interests      []string `json:"interests"`
		Goals          []string `json:"goals"`
		Notes          string   `json:"notes"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(raw)), &decoded); err != nil {
		return PersonalProfile{}, err
	}
	base := fallbackPersonalProfileFromSetup(answers)
	displayName := clampPersonalProfileField(decoded.DisplayName, base.DisplayName, 80)
	preferredTone := clampPersonalProfileField(decoded.PreferredTone, base.PreferredTone, 120)
	assistantStyle := clampPersonalProfileField(decoded.AssistantStyle, base.AssistantStyle, 300)
	notes := clampPersonalProfileField(decoded.Notes, base.Notes, 4000)
	interests := clampPersonalStringList(decoded.Interests, 12, 120)
	if len(interests) == 0 {
		interests = base.Interests
	}
	goals := clampPersonalStringList(decoded.Goals, 12, 120)
	if len(goals) == 0 {
		goals = base.Goals
	}
	now := time.Now().UTC()
	return PersonalProfile{
		DisplayName:     displayName,
		PreferredTone:   preferredTone,
		AssistantStyle:  assistantStyle,
		Interests:       interests,
		Goals:           goals,
		Notes:           notes,
		LocationCity:    base.LocationCity,
		LocationCountry: base.LocationCountry,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// ExtractPersonalLearningCandidatesWithModels proposes memories via models when available.
func ExtractPersonalLearningCandidatesWithModels(cfg PersonalConfig, text string) ([]PersonalMemory, error) {
	models := []string{strings.TrimSpace(cfg.PrimaryModel), strings.TrimSpace(cfg.FallbackModel)}
	var lastErr error
	for _, modelID := range models {
		if modelID == "" {
			continue
		}
		candidates, err := extractPersonalLearningWithModel(modelID, text)
		if err == nil {
			return candidates, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no personal learning model configured")
	}
	return nil, lastErr
}

func extractPersonalLearningWithModel(modelID string, text string) ([]PersonalMemory, error) {
	prompt := `Du bist Aethels Personal Learning Agent.
Extrahiere ausschließlich stabile persönliche Informationen über den Nutzer.
Speichere KEINE Secrets, API Keys, Passwörter, Tokens, privaten Schlüssel oder einmalige Stimmungen.
Speichere KEINE Projekt-/Code-Fakten; dafür gibt es Nexus.
Antworte ausschließlich als JSON-Array. Kein Markdown.
Schema:
[{"type":"identity|preference|routine|goal|relationship|boundary","content":"kurzer stabiler Fakt auf Deutsch","confidence":0.0}]

Gesprächstext:
` + text

	raw, err := personalModelCompletion(modelID, prompt)
	if err != nil {
		return nil, err
	}
	var decoded []struct {
		Type       string  `json:"type"`
		Content    string  `json:"content"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(extractJSONArray(raw)), &decoded); err != nil {
		return nil, err
	}
	out := make([]PersonalMemory, 0, len(decoded))
	for _, item := range decoded {
		memType := ClampPersonalText(item.Type, 60)
		content := ClampPersonalText(item.Content, 2000)
		if memType == "" || content == "" || LooksSecretLike(content) {
			continue
		}
		if item.Confidence <= 0 || item.Confidence > 1 {
			item.Confidence = 0.7
		}
		out = append(out, PersonalMemory{
			Type:       memType,
			Content:    content,
			Confidence: item.Confidence,
			Source:     "model:" + modelID,
		})
		if len(out) >= 8 {
			break
		}
	}
	return out, nil
}

func extractJSONArray(raw string) string {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start >= 0 && end >= start {
		return raw[start : end+1]
	}
	return raw
}

func extractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end >= start {
		return raw[start : end+1]
	}
	return raw
}

func LooksSecretLike(value string) bool {
	lower := strings.ToLower(value)
	secretTerms := []string{"password", "passwort", "token", "api key", "secret", "private key", "bearer ", "sk-", "gsk_", "AIza"}
	for _, term := range secretTerms {
		if strings.Contains(lower, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

func personalModelCompletion(modelID string, prompt string) (string, error) {
	actualModelID, targetURL, targetKey, useClaude, err := resolvePersonalModelTarget(modelID)
	if err != nil {
		return "", err
	}

	var payload map[string]interface{}
	if useClaude {
		payload = map[string]interface{}{
			"model":      actualModelID,
			"max_tokens": 1200,
			"messages": []map[string]interface{}{
				{"role": "user", "content": prompt},
			},
		}
	} else {
		payload = map[string]interface{}{
			"model":       actualModelID,
			"messages":    []map[string]interface{}{{"role": "user", "content": prompt}},
			"temperature": 0.1,
			"stream":      false,
		}
		if strings.Contains(strings.ToLower(actualModelID), "deepseek") {
			payload["thinking"] = map[string]interface{}{"type": "disabled"}
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 90 * time.Second}
	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if useClaude {
		req.Header.Set("x-api-key", targetKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else if targetKey != "ollama" {
		req.Header.Set("Authorization", "Bearer "+targetKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var decoded map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("personal model %s failed with HTTP %d", modelID, resp.StatusCode)
	}
	if useClaude {
		if blocks, ok := decoded["content"].([]interface{}); ok {
			var parts []string
			for _, block := range blocks {
				if m, ok := block.(map[string]interface{}); ok {
					if text, ok := m["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, "\n"), nil
			}
		}
		return "", errors.New("claude response missing text content")
	}
	choices, ok := decoded["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", errors.New("model response missing choices")
	}
	first, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", errors.New("invalid model choice")
	}
	msg, ok := first["message"].(map[string]interface{})
	if !ok {
		return "", errors.New("model choice missing message")
	}
	if content, ok := msg["content"].(string); ok {
		return content, nil
	}
	return "", errors.New("model message missing content")
}

func resolvePersonalModelTarget(modelID string) (actualModelID string, targetURL string, targetKey string, useClaude bool, err error) {
	actualModelID = strings.TrimSpace(modelID)
	targetURL = provider.GroqURL
	targetKey = state.getAPIKey()

	switch {
	case strings.HasPrefix(modelID, "deepseek/"):
		actualModelID = strings.TrimPrefix(modelID, "deepseek/")
		targetURL = provider.DeepSeekURL
		targetKey = state.getDeepSeekKey()
	case strings.HasPrefix(modelID, "ollama/"):
		actualModelID = strings.TrimPrefix(modelID, "ollama/")
		targetURL = "http://localhost:11434/v1/chat/completions"
		targetKey = "ollama"
	case strings.HasPrefix(modelID, "gemini/"):
		actualModelID = strings.TrimPrefix(modelID, "gemini/")
		targetURL = provider.GeminiURL
		targetKey = state.getGeminiKey()
	case strings.HasPrefix(modelID, "claude/"):
		actualModelID = strings.TrimPrefix(modelID, "claude/")
		targetURL = provider.ClaudeURL
		targetKey = state.getClaudeKey()
		useClaude = true
	case strings.HasPrefix(modelID, "openai-native/"):
		actualModelID = strings.TrimPrefix(modelID, "openai-native/")
		targetURL = provider.OpenAIURL
		targetKey = state.getOpenAIKey()
	}
	if actualModelID == "" {
		err = errors.New("empty personal model id")
		return
	}
	if targetKey == "" && targetKey != "ollama" {
		err = fmt.Errorf("missing API key for personal model %s", modelID)
		return
	}
	return
}

// ExtractPersonalLearningCandidates is the heuristic fallback memory extractor.
func ExtractPersonalLearningCandidates(text string) []PersonalMemory {
	lines := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '.' || r == '!' || r == '?'
	})
	out := []PersonalMemory{}
	for _, raw := range lines {
		line := ClampPersonalText(raw, 300)
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(lower, "ich heiße") || strings.Contains(lower, "nenn mich") || strings.Contains(lower, "mein name ist"):
			out = append(out, PersonalMemory{Type: "identity", Content: line, Confidence: 0.82, Source: "learning_agent"})
		case strings.Contains(lower, "ich mag") || strings.Contains(lower, "ich liebe") || strings.Contains(lower, "ich schaue gerne"):
			out = append(out, PersonalMemory{Type: "preference", Content: line, Confidence: 0.72, Source: "learning_agent"})
		case strings.Contains(lower, "ich möchte") || strings.Contains(lower, "mein ziel") || strings.Contains(lower, "ich will"):
			out = append(out, PersonalMemory{Type: "goal", Content: line, Confidence: 0.68, Source: "learning_agent"})
		case strings.Contains(lower, "sprich mit mir") || strings.Contains(lower, "sei bitte") || strings.Contains(lower, "du sollst"):
			out = append(out, PersonalMemory{Type: "relationship", Content: line, Confidence: 0.65, Source: "learning_agent"})
		}
		if len(out) >= 6 {
			break
		}
	}
	return out
}

func splitPersonalList(value string, limit int) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := ClampPersonalText(part, 120)
		if item != "" {
			out = append(out, item)
		}
		if len(out) >= limit {
			break
		}
	}
	return out
}

func clampPersonalStringList(values []string, limit int, maxRunes int) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		item := ClampPersonalText(value, maxRunes)
		if item != "" && !LooksSecretLike(item) {
			out = append(out, item)
		}
		if len(out) >= limit {
			break
		}
	}
	return out
}

func clampPersonalProfileField(value string, fallback string, maxRunes int) string {
	clean := ClampPersonalText(value, maxRunes)
	if clean == "" || LooksSecretLike(clean) {
		return ClampPersonalText(fallback, maxRunes)
	}
	return clean
}

// BuildSharedPersonalContext maps PersonalStore profile fields into the intelligence PersonalContext
// trust class for opt-in World State correlation (never auto-mixes into cases or nexus memory).
func BuildSharedPersonalContext(store *PersonalStore) intelligence.PersonalContext {
	pc := intelligence.PersonalContext{
		Interests:        []string{},
		Goals:            []string{},
		Projects:         []string{},
		PreferredRegions: []string{},
		WatchlistIDs:     []string{},
		RiskTolerance:    "medium",
		LastUpdated:      time.Now().UTC(),
	}
	if store == nil {
		return pc
	}
	cfg, err := store.LoadConfig()
	if err != nil || !cfg.Enabled {
		return pc
	}
	profile, _ := store.LoadProfile()
	pc.OperatorID = ClampPersonalText(profile.DisplayName, 80)
	if pc.OperatorID == "" {
		pc.OperatorID = "operator"
	}
	pc.Interests = clampPersonalStringList(profile.Interests, 24, 80)
	pc.Goals = clampPersonalStringList(profile.Goals, 16, 120)
	pc.CommunicationStyle = clampPersonalProfileField(profile.AssistantStyle, profile.PreferredTone, 80)
	pc.LocationCity = ClampPersonalText(profile.LocationCity, 120)
	pc.LocationCountry = ClampPersonalText(profile.LocationCountry, 120)
	if pc.LocationCountry != "" {
		pc.PreferredRegions = append(pc.PreferredRegions, pc.LocationCountry)
	}
	// Notes may mention regions/projects; treat as project tags when short tokens (no secret scrape)
	if notes := ClampPersonalText(profile.Notes, 400); notes != "" {
		for _, part := range strings.FieldsFunc(notes, func(r rune) bool {
			return r == ',' || r == ';' || r == '\n'
		}) {
			t := strings.TrimSpace(part)
			if len(t) < 2 || len(t) > 48 {
				continue
			}
			upper := strings.ToUpper(t)
			// Known region tokens → PreferredRegions; else soft project label
			switch upper {
			case "GERMANY", "DE", "FRANCE", "UKRAINE", "USA", "UK", "POLAND", "BALTICS", "ISRAEL", "TAIWAN", "CHINA", "RUSSIA":
				pc.PreferredRegions = append(pc.PreferredRegions, upper)
			default:
				if len(pc.Projects) < 12 {
					pc.Projects = append(pc.Projects, t)
				}
			}
		}
	}
	// Risk tolerance from honesty/initiative is not world-risk; keep medium unless notes say otherwise
	pc.LastUpdated = time.Now().UTC()
	return pc
}

// SyncPersonalToSharedIntel performs the explicit U3 bridge (callable from skill/API).
func SyncPersonalToSharedIntel(store *PersonalStore) (intelligence.PersonalContext, error) {
	if intelligence.SharedIntelStore == nil {
		return intelligence.PersonalContext{}, errors.New("intelligence.SharedIntelStore unavailable")
	}
	pc := BuildSharedPersonalContext(store)
	intelligence.SharedIntelStore.SetPersonalContext(pc)
	return intelligence.SharedIntelStore.GetPersonalContext(), nil
}

func BuildPersonalContext(store *PersonalStore) string {
	if store == nil {
		return ""
	}
	cfg, err := store.LoadConfig()
	if err != nil || !cfg.Enabled {
		return ""
	}
	profile, _ := store.LoadProfile()
	memories, _ := store.ListMemories()

	var humorDesc string
	if cfg.HumorLevel < 30 {
		humorDesc = "Antworte absolut sachlich, ernst und ohne Humor oder Witze."
	} else if cfg.HumorLevel > 70 {
		humorDesc = "Nutze aktiv sehr trockenen Humor, Ironie, Sarkasmus und humorvolle, zynische Bemerkungen in deinen Antworten, solange die Klarheit nicht gefährdet wird."
	} else {
		humorDesc = "Antworte im Allgemeinen sachlich, streue aber bei passenden Gelegenheiten dezenten, situationsangemessenen Humor ein."
	}

	var honestyDesc string
	if cfg.HonestyLevel < 30 {
		honestyDesc = "Antworte diplomatisch, höflich und schonend. Weise den Operator nur vorsichtig auf Fehler oder Risiken hin."
	} else if cfg.HonestyLevel > 70 {
		honestyDesc = "Sei absolut und schonungslos ehrlich. Benenne Fehler des Operators, Sicherheitsrisiken, Lücken und eigene Unsicherheiten direkt, ungeschminkt und schonungslos. Erfinde niemals Fortschritt oder Sicherheit."
	} else {
		honestyDesc = "Sei ehrlich und transparent. Benenne Unsicherheiten, Risiken und Grenzen direkt, behalte aber eine professionelle Höflichkeit bei."
	}

	var initiativeDesc string
	if cfg.InitiativeLevel < 30 {
		initiativeDesc = "Verhalte dich rein reaktiv. Beantworte exakt das, was gefragt wurde. Mache keine ungefragten Folge- oder Verbesserungsvorschläge."
	} else if cfg.InitiativeLevel > 70 {
		initiativeDesc = "Verhalte dich hochgradig proaktiv und initiativreich. Schlage selbstständig sinnvolle nächste Schritte vor, antizipiere Probleme und liefere unaufgefordert Lösungswege oder Code-Entwürfe."
	} else {
		initiativeDesc = "Liefere passende nächste Schritte und erkennbare Risiken proaktiv, aber führe keine Aktionen ohne Freigabe aus."
	}

	var sb strings.Builder
	sb.WriteString("AETHEL PERSONAL CORE:\n")
	sb.WriteString(fmt.Sprintf("- Humorgrad: %d/100. %s\n", cfg.HumorLevel, humorDesc))
	sb.WriteString(fmt.Sprintf("- Ehrlichkeitsgrad: %d/100. %s\n", cfg.HonestyLevel, honestyDesc))
	sb.WriteString(fmt.Sprintf("- Initiative: %d/100. %s\n", cfg.InitiativeLevel, initiativeDesc))
	sb.WriteString("- Beziehung: Sei ein loyaler, ruhiger Arbeitskollege. Priorisiere die Ziele des Operators, bleibe respektvoll und bleibe in kritischen Situationen sachlich statt dramatisch.\n")
	sb.WriteString("PERSÖNLICHER MODUS AKTIV.\n")
	sb.WriteString("Dieser Kontext ist getrennt vom Nexus-Gedächtnis und beschreibt Beziehung, Profil und stabile persönliche Präferenzen.\n")
	if profile.DisplayName != "" {
		sb.WriteString("Name/Ansprache: " + profile.DisplayName + "\n")
	}
	if profile.PreferredTone != "" {
		sb.WriteString("Bevorzugter Ton: " + profile.PreferredTone + "\n")
	}
	if profile.AssistantStyle != "" {
		sb.WriteString("Assistant-Stil: " + profile.AssistantStyle + "\n")
	}
	if profile.LocationCity != "" || profile.LocationCountry != "" {
		sb.WriteString("Eigener Standort: " + strings.Trim(strings.Join([]string{profile.LocationCity, profile.LocationCountry}, ", "), ", ") + "\n")
		sb.WriteString("Bei Lage-, Wetter- und Nachrichtenfragen priorisiere diesen Standort, ohne andere Regionen auszublenden.\n")
	}
	if len(profile.Interests) > 0 {
		sb.WriteString("Interessen: " + strings.Join(profile.Interests, ", ") + "\n")
	}
	if len(profile.Goals) > 0 {
		sb.WriteString("Ziele: " + strings.Join(profile.Goals, ", ") + "\n")
	}
	if profile.Notes != "" {
		sb.WriteString("Profilnotizen: " + profile.Notes + "\n")
	}
	if len(memories) > 0 {
		sb.WriteString("Persönliche Erinnerungen:\n")
		start := 0
		if len(memories) > 12 {
			start = len(memories) - 12
		}
		for _, mem := range memories[start:] {
			sb.WriteString("- [" + mem.Type + "] " + mem.Content + "\n")
		}
	}
	if cfg.LearningEnabled {
		sb.WriteString("Lernmodus: aktiv. Stabile persönliche Erkenntnisse dürfen über die Personal-API gespeichert werden, keine Secrets und keine flüchtigen Emotionen als dauerhafte Fakten.\n")
	}
	return sb.String()
}
