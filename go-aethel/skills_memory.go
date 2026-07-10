package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// --- 4. RAG / MEMORY SYSTEM (TF-IDF SEARCH ENGINE) ---

var memoryFile = "./vgt_workspace/nexus_memory.json"

type MemoryEntry struct {
	ID         string     `json:"id"`
	Content    string     `json:"content"`
	Timestamp  time.Time  `json:"timestamp"`
	Category   string     `json:"category"`
	Tags       []string   `json:"tags,omitempty"`
	Importance int        `json:"importance,omitempty"`
	Source     string     `json:"source"`
	ConsentAt  time.Time  `json:"consent_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	Supersedes string     `json:"supersedes,omitempty"`
}

type LocalMemoryStore struct {
	mu      sync.RWMutex
	entries []MemoryEntry
}

func NewLocalMemoryStore() *LocalMemoryStore {
	store := &LocalMemoryStore{entries: make([]MemoryEntry, 0)}
	store.loadFromDisk()
	return store
}

func (l *LocalMemoryStore) loadFromDisk() {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, _, err := readSealedFile(memoryFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &l.entries)
}

func (l *LocalMemoryStore) saveToDisk() error {
	data, err := json.MarshalIndent(l.entries, "", "  ")
	if err != nil {
		return err
	}
	return writeSealedFile(memoryFile, data)
}

func normalizeMemoryCategory(category string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	switch category {
	case "project", "preference", "fact", "decision", "workflow", "reference", "general":
		return category
	default:
		return "general"
	}
}

func memoryTags(content, category string) []string {
	seen := map[string]bool{category: true}
	tags := []string{category}
	for _, token := range tokenize(content) {
		if len(token) < 4 || seen[token] {
			continue
		}
		seen[token] = true
		tags = append(tags, token)
		if len(tags) == 7 {
			break
		}
	}
	return tags
}

func (l *LocalMemoryStore) Add(content, category string) string {
	id, _ := l.AddWithConsent(content, category, "legacy", true, nil, "")
	return id
}

func (l *LocalMemoryStore) AddWithConsent(content, category, source string, consent bool, expiresAt *time.Time, supersedes string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	content = strings.TrimSpace(content)
	category = normalizeMemoryCategory(category)
	source = strings.TrimSpace(source)
	if !consent {
		return "", errors.New("memory write requires explicit operator consent")
	}
	if source == "" || len([]rune(source)) > 120 {
		return "", errors.New("memory source is required and limited to 120 characters")
	}
	if containsSensitiveMemoryData(content) {
		return "", errors.New("sensitive data rejected: store credentials only in the secret vault")
	}
	if expiresAt != nil && !expiresAt.After(time.Now().UTC()) {
		return "", errors.New("memory expiry must be in the future")
	}
	for _, existing := range l.entries {
		if strings.EqualFold(strings.TrimSpace(existing.Content), content) && existing.Category == category {
			return existing.ID, nil
		}
	}
	if supersedes != "" {
		found := false
		now := time.Now().UTC()
		for i := range l.entries {
			if l.entries[i].ID == supersedes {
				l.entries[i].ExpiresAt = &now
				found = true
				break
			}
		}
		if !found {
			return "", errors.New("superseded memory does not exist")
		}
	}
	importance := 1
	if category == "decision" || category == "workflow" || category == "preference" {
		importance = 2
	}
	id := fmt.Sprintf("mem-%d", time.Now().UnixNano())
	entry := MemoryEntry{
		ID:         id,
		Content:    content,
		Timestamp:  time.Now(),
		Category:   category,
		Tags:       memoryTags(content, category),
		Importance: importance,
		Source:     source,
		ConsentAt:  time.Now().UTC(),
		ExpiresAt:  expiresAt,
		Supersedes: strings.TrimSpace(supersedes),
	}
	l.entries = append(l.entries, entry)
	if err := l.saveToDisk(); err != nil {
		return "", err
	}
	return id, nil
}

func (l *LocalMemoryStore) GetAll() []MemoryEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	now := time.Now().UTC()
	res := make([]MemoryEntry, 0, len(l.entries))
	for _, entry := range l.entries {
		if entry.ExpiresAt == nil || entry.ExpiresAt.After(now) {
			res = append(res, entry)
		}
	}
	return res
}

func (l *LocalMemoryStore) Explain(id string) (MemoryEntry, string, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, entry := range l.entries {
		if entry.ID != id || (entry.ExpiresAt != nil && !entry.ExpiresAt.After(time.Now().UTC())) {
			continue
		}
		explanation := fmt.Sprintf("Stored with operator consent from %s on %s.", entry.Source, entry.ConsentAt.Local().Format(time.RFC1123))
		if entry.ExpiresAt != nil {
			explanation += " It expires on " + entry.ExpiresAt.Local().Format(time.RFC1123) + "."
		}
		if entry.Supersedes != "" {
			explanation += " It supersedes " + entry.Supersedes + "."
		}
		return entry, explanation, true
	}
	return MemoryEntry{}, "", false
}

func (l *LocalMemoryStore) Delete(id string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	found := false
	for i, entry := range l.entries {
		if entry.ExpiresAt != nil && !entry.ExpiresAt.After(time.Now().UTC()) {
			continue
		}
		if entry.ID == id {
			l.entries = append(l.entries[:i], l.entries[i+1:]...)
			found = true
			break
		}
	}
	if found {
		_ = l.saveToDisk()
	}
	return found
}

// TF-IDF Math & Helper logic in pure Go stdlib
var reWords = regexp.MustCompile(`[a-zA-Z0-9äöüÄÖÜß]+`)
var sensitiveMemoryPattern = regexp.MustCompile(`(?i)(?:\b(?:password|passwort|api[_ -]?key|access[_ -]?token|bearer)\b\s*[:=]|\b(?:sk|gsk|AIza|AKIA|xox[baprs])[-_][A-Za-z0-9_-]{8,}|-----BEGIN [A-Z ]+PRIVATE KEY-----)`)

func containsSensitiveMemoryData(content string) bool {
	return sensitiveMemoryPattern.MatchString(content)
}

func tokenize(text string) []string {
	words := reWords.FindAllString(strings.ToLower(text), -1)
	var filtered []string
	// Filter out extremely short terms
	for _, w := range words {
		if len(w) > 1 {
			filtered = append(filtered, w)
		}
	}
	return filtered
}

// Search implements TF-IDF semantic approximation search via Cosine Similarity
func (l *LocalMemoryStore) Search(query string) []MemoryEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.entries) == 0 {
		return nil
	}

	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	// 1. Calculate Document Frequencies (DF)
	df := make(map[string]int)
	allDocTokens := make([][]string, len(l.entries))

	for i, entry := range l.entries {
		tokens := tokenize(entry.Content + " " + entry.Category + " " + strings.Join(entry.Tags, " "))
		allDocTokens[i] = tokens
		uniqueTokens := make(map[string]bool)
		for _, t := range tokens {
			uniqueTokens[t] = true
		}
		for t := range uniqueTokens {
			df[t]++
		}
	}

	numDocs := float64(len(l.entries))

	// Helper to compute TF-IDF vector for a set of tokens
	getVector := func(tokens []string) map[string]float64 {
		tf := make(map[string]float64)
		for _, t := range tokens {
			tf[t]++
		}
		// Normalize tf
		total := float64(len(tokens))
		vector := make(map[string]float64)
		for t, count := range tf {
			tfVal := count / total
			// idf = log(1 + numDocs / (1 + df))
			docFreq := float64(df[t])
			idfVal := math.Log(1.0 + (numDocs / (1.0 + docFreq)))
			vector[t] = tfVal * idfVal
		}
		return vector
	}

	queryVec := getVector(queryTokens)

	// Calculate norm of query vector
	queryNorm := 0.0
	for _, val := range queryVec {
		queryNorm += val * val
	}
	queryNorm = math.Sqrt(queryNorm)

	type ScoredResult struct {
		Entry MemoryEntry
		Score float64
	}
	var scoredResults []ScoredResult

	for i, entry := range l.entries {
		docTokens := allDocTokens[i]
		if len(docTokens) == 0 {
			continue
		}
		docVec := getVector(docTokens)

		// Cosine Similarity: Dot Product / (NormA * NormB)
		dotProduct := 0.0
		docNorm := 0.0
		for t, val := range docVec {
			docNorm += val * val
			if qVal, ok := queryVec[t]; ok {
				dotProduct += val * qVal
			}
		}
		docNorm = math.Sqrt(docNorm)

		if queryNorm == 0 || docNorm == 0 {
			continue
		}

		similarity := dotProduct / (queryNorm * docNorm)

		// 2. Compute Jaccard Tag Similarity
		tagMatchCount := 0
		uniqueTags := make(map[string]bool)
		for _, tag := range entry.Tags {
			uniqueTags[strings.ToLower(tag)] = true
		}
		for _, qt := range queryTokens {
			if uniqueTags[qt] {
				tagMatchCount++
			}
		}
		jaccard := 0.0
		if len(uniqueTags) > 0 || len(queryTokens) > 0 {
			unionSize := len(uniqueTags) + len(queryTokens) - tagMatchCount
			if unionSize > 0 {
				jaccard = float64(tagMatchCount) / float64(unionSize)
			}
		}

		// 3. Compute Fuzzy Prefix boost
		prefixBoost := 0.0
		for _, qt := range queryTokens {
			for tag := range uniqueTags {
				if strings.HasPrefix(tag, qt) || strings.HasPrefix(qt, tag) {
					prefixBoost += 0.05
				}
			}
		}
		if prefixBoost > 0.20 {
			prefixBoost = 0.20
		}

		// Combined Hybrid Score (60% TF-IDF cosine similarity + 40% Jaccard Tag similarity + prefix boost)
		hybridScore := 0.60*similarity + 0.40*jaccard + prefixBoost

		// 4. Time Decay and Importance boost
		ageDays := time.Since(entry.Timestamp).Hours() / 24
		if ageDays > 0 {
			hybridScore += 0.12 * math.Exp(-ageDays/180)
		}
		hybridScore += 0.04 * float64(entry.Importance)

		if hybridScore > 0.05 {
			scoredResults = append(scoredResults, ScoredResult{Entry: entry, Score: hybridScore})
		}
	}

	// Sort results descending by score
	sort.Slice(scoredResults, func(i, j int) bool {
		return scoredResults[i].Score > scoredResults[j].Score
	})

	var results []MemoryEntry
	for _, r := range scoredResults {
		results = append(results, r.Entry)
	}

	// Limit to top 5 results
	if len(results) > 5 {
		results = results[:5]
	}
	return results
}

// --- SKILL: SAVE TO NEXUS ---

type MemorySaveSkill struct {
	Store *LocalMemoryStore
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
func (s *MemorySaveSkill) RiskLevel() RiskLevel { return RiskSafe }

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
	if containsSensitiveMemoryData(content) {
		return "", errors.New("sensitive data rejected: store credentials only in the secret vault")
	}
	category := normalizeMemoryCategory(input.Category)
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
	Store *LocalMemoryStore
}

type PersonalMemorySaveSkill struct {
	Store *PersonalStore
}

type PersonalMemoryRecallSkill struct {
	Store *PersonalStore
}

type RecallArgs struct {
	Query string `json:"query"`
}

func (s *MemoryRecallSkill) Name() string { return "nexus_recall" }
func (s *MemoryRecallSkill) Description() string {
	return "Durchsucht das lokale Langzeitgedächtnis nach relevanten Infos basierend auf Suchbegriffen."
}
func (s *MemoryRecallSkill) RiskLevel() RiskLevel { return RiskSafe }

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
func (s *PersonalMemorySaveSkill) RiskLevel() RiskLevel { return RiskSafe }
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
	content := clampPersonalText(input.Content, 2000)
	memType := clampPersonalText(input.Type, 60)
	if content == "" || memType == "" || looksSecretLike(content) {
		return "", errors.New("personal memory benötigt type und content")
	}
	mem, err := s.Store.AppendMemory(PersonalMemory{
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
func (s *PersonalMemoryRecallSkill) RiskLevel() RiskLevel { return RiskSafe }
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
