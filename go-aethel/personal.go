package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type PersonalConfig struct {
	Enabled         bool      `json:"enabled"`
	LearningEnabled bool      `json:"learning_enabled"`
	HumorLevel      int       `json:"humor_level"`
	HonestyLevel    int       `json:"honesty_level"`
	InitiativeLevel int       `json:"initiative_level"`
	PrimaryModel    string    `json:"primary_model"`
	FallbackModel   string    `json:"fallback_model"`
	WakeWord        string    `json:"wake_word"`
	ForbiddenTopics []string  `json:"forbidden_topics,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PersonalProfile struct {
	DisplayName    string    `json:"display_name"`
	PreferredTone  string    `json:"preferred_tone"`
	AssistantStyle string    `json:"assistant_style"`
	Interests      []string  `json:"interests"`
	Goals          []string  `json:"goals"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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
	data, _, err := readSealedFile(ps.configPath)
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
	return writeSealedFile(ps.configPath, data)
}

func (ps *PersonalStore) LoadProfile() (PersonalProfile, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	profile := PersonalProfile{}
	data, _, err := readSealedFile(ps.profilePath)
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
	return writeSealedFile(ps.profilePath, data)
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
	if err := validateResourceID(id); err != nil {
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
	if err := validateResourceID(mem.ID); err != nil {
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
	data, sealed, err := readSealedFile(ps.memoryPath)
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
	return writeSealedFile(ps.memoryPath, data)
}

func clampPersonalText(value string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > maxRunes {
		runes = runes[:maxRunes]
	}
	return string(runes)
}

func handlePersonalStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, _ := state.personal.LoadConfig()
	profile, _ := state.personal.LoadProfile()
	memories, _ := state.personal.ListMemories()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"config":        cfg,
		"profile":       profile,
		"memory_count":  len(memories),
		"setup_needed":  profile.DisplayName == "",
		"storage_scope": "personal",
	})
}

func handlePersonalConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		cfg, err := state.personal.LoadConfig()
		if err != nil {
			http.Error(w, "Config unavailable", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(cfg)
	case http.MethodPost:
		var cfg PersonalConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "Invalid config", http.StatusBadRequest)
			return
		}
		cfg.WakeWord = clampPersonalText(cfg.WakeWord, 40)
		cfg.PrimaryModel = clampPersonalText(cfg.PrimaryModel, 160)
		cfg.FallbackModel = clampPersonalText(cfg.FallbackModel, 160)
		cfg.HumorLevel = clampPersonalLevel(cfg.HumorLevel)
		cfg.HonestyLevel = clampPersonalLevel(cfg.HonestyLevel)
		cfg.InitiativeLevel = clampPersonalLevel(cfg.InitiativeLevel)
		if cfg.WakeWord == "" {
			cfg.WakeWord = "aethel"
		}
		if err := state.personal.SaveConfig(cfg); err != nil {
			http.Error(w, "Config save failed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func clampPersonalLevel(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func handlePersonalProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		profile, err := state.personal.LoadProfile()
		if err != nil {
			http.Error(w, "Profile unavailable", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(profile)
	case http.MethodPost:
		var profile PersonalProfile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "Invalid profile", http.StatusBadRequest)
			return
		}
		profile.DisplayName = clampPersonalText(profile.DisplayName, 80)
		profile.PreferredTone = clampPersonalText(profile.PreferredTone, 120)
		profile.AssistantStyle = clampPersonalText(profile.AssistantStyle, 300)
		profile.Notes = clampPersonalText(profile.Notes, 4000)
		if err := state.personal.SaveProfile(profile); err != nil {
			http.Error(w, "Profile save failed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePersonalMemories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		memories, err := state.personal.ListMemories()
		if err != nil {
			http.Error(w, "Memories unavailable", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(memories)
	case http.MethodPost:
		var req struct {
			PersonalMemory
			OperatorApproved bool `json:"operator_approved"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid memory", http.StatusBadRequest)
			return
		}
		if !req.OperatorApproved {
			http.Error(w, "Explicit operator approval is required for personal memory", http.StatusForbidden)
			return
		}
		mem := req.PersonalMemory
		mem.Type = clampPersonalText(mem.Type, 60)
		mem.Content = clampPersonalText(mem.Content, 2000)
		mem.Source = clampPersonalText(mem.Source, 120)
		if mem.Type == "" || mem.Content == "" || looksSecretLike(mem.Content) {
			http.Error(w, "Memory requires type and content", http.StatusBadRequest)
			return
		}
		saved, err := state.personal.AppendMemory(mem)
		if err != nil {
			http.Error(w, "Memory save failed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(saved)
	case http.MethodPut:
		var mem PersonalMemory
		if err := json.NewDecoder(r.Body).Decode(&mem); err != nil {
			http.Error(w, "Invalid memory", http.StatusBadRequest)
			return
		}
		mem.Type = clampPersonalText(mem.Type, 60)
		mem.Content = clampPersonalText(mem.Content, 2000)
		mem.Source = clampPersonalText(mem.Source, 120)
		if mem.Type == "" || mem.Content == "" {
			http.Error(w, "Memory requires type and content", http.StatusBadRequest)
			return
		}
		updated, err := state.personal.UpdateMemory(mem)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "Memory not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Memory update failed", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(updated)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if err := state.personal.DeleteMemory(id); err != nil {
			http.Error(w, "Invalid memory id", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePersonalSetupQuestions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Config PersonalConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid setup question payload", http.StatusBadRequest)
		return
	}
	questions, mode := buildPersonalSetupQuestions(req.Config)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"mode":      mode,
		"questions": questions,
	})
}

func handlePersonalSetup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Answers map[string]string `json:"answers"`
		Config  PersonalConfig    `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid setup payload", http.StatusBadRequest)
		return
	}
	cfg := req.Config
	cfg.Enabled = true
	cfg.LearningEnabled = true
	if cfg.WakeWord == "" {
		cfg.WakeWord = "aethel"
	}
	profile, setupMode := buildPersonalProfileFromSetup(cfg, req.Answers)
	if err := state.personal.SaveProfile(profile); err != nil {
		http.Error(w, "Profile setup failed", http.StatusInternalServerError)
		return
	}
	if err := state.personal.SaveConfig(cfg); err != nil {
		http.Error(w, "Config setup failed", http.StatusInternalServerError)
		return
	}
	if profile.DisplayName != "" {
		_, _ = state.personal.AppendMemory(PersonalMemory{
			Type:       "identity",
			Content:    "Der Nutzer möchte mit dem Namen " + profile.DisplayName + " angesprochen werden.",
			Confidence: 0.95,
			Source:     "setup",
		})
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "mode": setupMode, "profile": profile, "config": cfg})
}

func buildPersonalSetupQuestions(cfg PersonalConfig) ([]PersonalSetupQuestion, string) {
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
		id := clampPersonalText(q.ID, 40)
		target, ok := allowed[id]
		question := clampPersonalText(q.Question, 220)
		if !ok || seen[id] || question == "" || looksSecretLike(question) {
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

func buildPersonalProfileFromSetup(cfg PersonalConfig, answers map[string]string) (PersonalProfile, string) {
	if profile, err := generatePersonalProfileWithModels(cfg, answers); err == nil {
		return profile, "model"
	}
	return fallbackPersonalProfileFromSetup(answers), "heuristic_fallback"
}

func fallbackPersonalProfileFromSetup(answers map[string]string) PersonalProfile {
	now := time.Now().UTC()
	return PersonalProfile{
		DisplayName:    clampPersonalText(answers["display_name"], 80),
		PreferredTone:  clampPersonalText(answers["preferred_tone"], 120),
		AssistantStyle: clampPersonalText(answers["assistant_style"], 300),
		Interests:      splitPersonalList(answers["interests"], 12),
		Goals:          splitPersonalList(answers["goals"], 12),
		Notes:          clampPersonalText(answers["notes"], 4000),
		CreatedAt:      now,
		UpdatedAt:      now,
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
		"display_name":    clampPersonalText(answers["display_name"], 160),
		"preferred_tone":  clampPersonalText(answers["preferred_tone"], 240),
		"assistant_style": clampPersonalText(answers["assistant_style"], 800),
		"interests":       clampPersonalText(answers["interests"], 1000),
		"goals":           clampPersonalText(answers["goals"], 1000),
		"notes":           clampPersonalText(answers["notes"], 2000),
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
		DisplayName:    displayName,
		PreferredTone:  preferredTone,
		AssistantStyle: assistantStyle,
		Interests:      interests,
		Goals:          goals,
		Notes:          notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func handlePersonalLearn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := state.personal.LoadConfig()
	if err != nil || !cfg.Enabled || !cfg.LearningEnabled {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "skipped", "reason": "personal learning disabled"})
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid learn payload", http.StatusBadRequest)
		return
	}
	text := clampPersonalText(req.Text, 6000)
	lower := strings.ToLower(text)
	secretTerms := []string{"password", "passwort", "token", "api key", "secret", "private key"}
	for _, term := range secretTerms {
		if strings.Contains(lower, term) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "skipped", "reason": "secret-like content"})
			return
		}
	}
	candidates, extractErr := extractPersonalLearningCandidatesWithModels(cfg, text)
	if extractErr != nil {
		candidates = extractPersonalLearningCandidates(text)
	}
	mode := "model"
	if extractErr != nil {
		mode = "heuristic_fallback"
	}
	// Learning is a proposal, never an implicit write. The UI may present the
	// candidates and create only the memories explicitly approved by the user.
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "proposal", "mode": mode, "candidates": candidates})
}

func extractPersonalLearningCandidatesWithModels(cfg PersonalConfig, text string) ([]PersonalMemory, error) {
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
		memType := clampPersonalText(item.Type, 60)
		content := clampPersonalText(item.Content, 2000)
		if memType == "" || content == "" || looksSecretLike(content) {
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

func looksSecretLike(value string) bool {
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
	targetURL = groqURL
	targetKey = state.getAPIKey()

	switch {
	case strings.HasPrefix(modelID, "deepseek/"):
		actualModelID = strings.TrimPrefix(modelID, "deepseek/")
		targetURL = deepseekURL
		targetKey = state.getDeepSeekKey()
	case strings.HasPrefix(modelID, "ollama/"):
		actualModelID = strings.TrimPrefix(modelID, "ollama/")
		targetURL = "http://localhost:11434/v1/chat/completions"
		targetKey = "ollama"
	case strings.HasPrefix(modelID, "gemini/"):
		actualModelID = strings.TrimPrefix(modelID, "gemini/")
		targetURL = geminiURL
		targetKey = state.getGeminiKey()
	case strings.HasPrefix(modelID, "claude/"):
		actualModelID = strings.TrimPrefix(modelID, "claude/")
		targetURL = claudeURL
		targetKey = state.getClaudeKey()
		useClaude = true
	case strings.HasPrefix(modelID, "openai-native/"):
		actualModelID = strings.TrimPrefix(modelID, "openai-native/")
		targetURL = openaiURL
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

func extractPersonalLearningCandidates(text string) []PersonalMemory {
	lines := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '.' || r == '!' || r == '?'
	})
	out := []PersonalMemory{}
	for _, raw := range lines {
		line := clampPersonalText(raw, 300)
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
		item := clampPersonalText(part, 120)
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
		item := clampPersonalText(value, maxRunes)
		if item != "" && !looksSecretLike(item) {
			out = append(out, item)
		}
		if len(out) >= limit {
			break
		}
	}
	return out
}

func clampPersonalProfileField(value string, fallback string, maxRunes int) string {
	clean := clampPersonalText(value, maxRunes)
	if clean == "" || looksSecretLike(clean) {
		return clampPersonalText(fallback, maxRunes)
	}
	return clean
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
	var sb strings.Builder
	sb.WriteString("AETHEL PERSONAL CORE:\n")
	sb.WriteString(fmt.Sprintf("- Humorgrad: %d/100. Nutze trockenen, situationsangemessenen Humor nur, wenn er Klarheit nicht verdrÃ¤ngt.\n", cfg.HumorLevel))
	sb.WriteString(fmt.Sprintf("- Ehrlichkeitsgrad: %d/100. Benenne Unsicherheit, Risiken und Grenzen direkt; erfinde niemals Fortschritt oder Sicherheit.\n", cfg.HonestyLevel))
	sb.WriteString(fmt.Sprintf("- Initiative: %d/100. Liefere passende nÃ¤chste Schritte und erkennbare Risiken proaktiv, aber fÃ¼hre keine riskanten Aktionen ohne die erforderliche Freigabe aus.\n", cfg.InitiativeLevel))
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
