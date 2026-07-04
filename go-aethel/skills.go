package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)


// VgtSkill is the interface every local skill must implement
type VgtSkill interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	RiskLevel() RiskLevel
	Execute(args json.RawMessage) (string, error)
}

// SkillRegistry holds all available system tools
type SkillRegistry struct {
	skills map[string]VgtSkill
}

func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills: make(map[string]VgtSkill),
	}
}

func (sr *SkillRegistry) Register(skill VgtSkill) {
	sr.skills[skill.Name()] = skill
}

func (sr *SkillRegistry) Get(name string) (VgtSkill, bool) {
	s, ok := sr.skills[name]
	return s, ok
}

// ToToolDefinitions converts skills to the OpenAI schema format
func (sr *SkillRegistry) ToToolDefinitions() []interface{} {
	var defs []interface{}
	for _, s := range sr.skills {
		defs = append(defs, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        s.Name(),
				"description": s.Description(),
				"parameters":  s.Parameters(),
			},
		})
	}
	return defs
}

// Sandbox validation helper
const workspaceDir = "./vgt_workspace"

func validatePath(pathStr string) (string, error) {
	// Canonicalize/Resolve root workspace
	err := os.MkdirAll(workspaceDir, 0755)
	if err != nil {
		return "", err
	}
	absWorkspace, err := filepath.Abs(workspaceDir)
	if err != nil {
		return "", err
	}

	// Resolve target path (absolute or relative)
	var absTarget string
	if filepath.IsAbs(pathStr) {
		absTarget, err = filepath.Abs(pathStr)
	} else {
		absTarget, err = filepath.Abs(filepath.Join(workspaceDir, pathStr))
	}
	if err != nil {
		return "", err
	}

	// 1. Verify if target path is inside workspace (sandbox)
	if strings.HasPrefix(absTarget, absWorkspace) {
		return absTarget, nil
	}

	// 2. Verify if target path is inside any mounted directory
	if state != nil {
		mounted := state.GetMountedDirs()
		for _, dir := range mounted {
			absDir, err := filepath.Abs(dir)
			if err == nil {
				if strings.HasPrefix(absTarget, absDir) {
					return absTarget, nil
				}
			}
		}
	}

	return "", errors.New("SECURITY VIOLATION: Path is not inside the workspace or any mounted directory")
}

func resolveCommandPath(command string) string {
	lowerCmd := strings.ToLower(strings.TrimSpace(command))

	if lowerCmd == "cmd.exe" || lowerCmd == "cmd" {
		return "C:\\Windows\\System32\\cmd.exe"
	}

	if lowerCmd == "powershell.exe" || lowerCmd == "powershell" {
		return "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
	}

	if path, err := exec.LookPath(command); err == nil {
		return path
	}

	return command
}

// --- 1. SKILL: EXECUTE SYSTEM COMMAND ---

type ExecuteCommandSkill struct{}

type ExecArgs struct {
	Command    string   `json:"command"`
	Args       []string `json:"args"`
	Background bool     `json:"background,omitempty"`
}

func (s *ExecuteCommandSkill) Name() string        { return "sys_exec_cmd" }
func (s *ExecuteCommandSkill) Description() string { return "Führt einen Systembefehl auf dem Host aus. Sicherheitskritisch." }
func (s *ExecuteCommandSkill) RiskLevel() RiskLevel { return RiskCritical }

func (s *ExecuteCommandSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{"type": "string", "description": "Der Befehl (z.B. 'dir', 'git', 'ping')"},
			"args":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Argumente"},
			"background": map[string]interface{}{"type": "boolean", "description": "Ob der Befehl im Hintergrund gestartet werden soll (wichtig für GUI-Apps wie Browser-Tabs, Spotify, Notepad, um Hänger zu vermeiden)"},
		},
		"required": []string{"command", "args"},
	}
}

func (s *ExecuteCommandSkill) Execute(args json.RawMessage) (string, error) {
	var input ExecArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	// Double-check command blacklist
	blacklist := []string{"rm", "dd", "format", "mkfs", "shred", "wipe", "shutdown", "reboot"}
	for _, cmd := range blacklist {
		if strings.TrimSpace(strings.ToLower(input.Command)) == cmd {
			return "", fmt.Errorf("VGT SECURITY INTERVENTION: Befehl '%s' ist blockiert", cmd)
		}
	}

	cmdPath := resolveCommandPath(input.Command)
	cmd := exec.Command(cmdPath, input.Args...)
	if input.Background {
		err := cmd.Start()
		if err != nil {
			LogKernelActivity("EXEC_FAILED", input.Command+" "+strings.Join(input.Args, " "), "ERROR")
			return "", fmt.Errorf("background command failed: %v", err)
		}
		LogKernelActivity("EXEC_BG", input.Command+" "+strings.Join(input.Args, " "), "SUCCESS")
		return fmt.Sprintf("Befehl im Hintergrund gestartet (PID: %d)", cmd.Process.Pid), nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		LogKernelActivity("EXEC_FAILED", input.Command+" "+strings.Join(input.Args, " "), "ERROR")
		return "", fmt.Errorf("command failed: %s, output: %s", err.Error(), string(output))
	}
	LogKernelActivity("EXEC", input.Command+" "+strings.Join(input.Args, " "), "SUCCESS")
	return string(output), nil
}

// --- 2. SKILL: READ FILE ---

type ReadFileSkill struct{}

type ReadFileArgs struct {
	Path string `json:"path"`
}

func (s *ReadFileSkill) Name() string        { return "fs_read_file" }
func (s *ReadFileSkill) Description() string { return "Liest den Inhalt einer Textdatei aus dem VGT Workspace." }
func (s *ReadFileSkill) RiskLevel() RiskLevel { return RiskSafe }

func (s *ReadFileSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string", "description": "Relativer Pfad zur Datei (z.B. 'notes.txt')"},
		},
		"required": []string{"path"},
	}
}

func (s *ReadFileSkill) Execute(args json.RawMessage) (string, error) {
	var input ReadFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	safePath, err := validatePath(input.Path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		LogKernelActivity("READ_FAILED", input.Path, "ERROR")
		return "", err
	}
	LogKernelActivity("READ", input.Path, "SUCCESS")
	return string(data), nil
}

// --- 3. SKILL: WRITE FILE ---

type WriteFileSkill struct{}

type WriteFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (s *WriteFileSkill) Name() string        { return "fs_write_file" }
func (s *WriteFileSkill) Description() string { return "Schreibt Text in eine Datei im VGT Workspace. Überschreibt existierende Dateien." }
func (s *WriteFileSkill) RiskLevel() RiskLevel { return RiskCritical }

func (s *WriteFileSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":    map[string]interface{}{"type": "string", "description": "Relativer Zielpfad"},
			"content": map[string]interface{}{"type": "string", "description": "Vollständiger Textinhalt"},
		},
		"required": []string{"path", "content"},
	}
}

func (s *WriteFileSkill) Execute(args json.RawMessage) (string, error) {
	var input WriteFileArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	safePath, err := validatePath(input.Path)
	if err != nil {
		return "", err
	}

	// Ensure parent directories exist
	if err := os.MkdirAll(filepath.Dir(safePath), 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(safePath, []byte(input.Content), 0644); err != nil {
		LogKernelActivity("WRITE_FAILED", input.Path, "ERROR")
		return "", err
	}
	LogKernelActivity("WRITE", input.Path, "SUCCESS")
	return fmt.Sprintf("Datei erfolgreich geschrieben: %s", input.Path), nil
}

// --- 4. RAG / MEMORY SYSTEM (TF-IDF SEARCH ENGINE) ---

const memoryFile = "./vgt_workspace/nexus_memory.json"

type MemoryEntry struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Category  string    `json:"category"`
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

	data, err := os.ReadFile(memoryFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &l.entries)
}

func (l *LocalMemoryStore) saveToDisk() error {
	_ = os.MkdirAll(filepath.Dir(memoryFile), 0755)
	data, err := json.MarshalIndent(l.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(memoryFile, data, 0644)
}

func (l *LocalMemoryStore) Add(content, category string) string {
	l.mu.Lock()
	defer l.mu.Unlock()

	id := fmt.Sprintf("mem-%d", time.Now().UnixNano())
	entry := MemoryEntry{
		ID:        id,
		Content:   content,
		Timestamp: time.Now(),
		Category:  category,
	}
	l.entries = append(l.entries, entry)
	_ = l.saveToDisk()
	return id
}

func (l *LocalMemoryStore) GetAll() []MemoryEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	res := make([]MemoryEntry, len(l.entries))
	copy(res, l.entries)
	return res
}

func (l *LocalMemoryStore) Delete(id string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	found := false
	for i, entry := range l.entries {
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
		tokens := tokenize(entry.Content)
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
		// Set threshold
		if similarity > 0.05 {
			scoredResults = append(scoredResults, ScoredResult{Entry: entry, Score: similarity})
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
	Content  string `json:"content"`
	Category string `json:"category,omitempty"`
}

func (s *MemorySaveSkill) Name() string        { return "nexus_save" }
func (s *MemorySaveSkill) Description() string { return "Speichert wichtige Fakten, Passwörter, User-Vorlieben oder Projektdaten im Langzeitgedächtnis." }
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
	category := input.Category
	if category == "" {
		category = "general"
	}
	id := s.Store.Add(input.Content, category)
	return fmt.Sprintf("NEXUS: Information gespeichert. ID: %s", id), nil
}

// --- SKILL: RECALL FROM NEXUS ---

type MemoryRecallSkill struct {
	Store *LocalMemoryStore
}

type RecallArgs struct {
	Query string `json:"query"`
}

func (s *MemoryRecallSkill) Name() string        { return "nexus_recall" }
func (s *MemoryRecallSkill) Description() string { return "Durchsucht das lokale Langzeitgedächtnis nach relevanten Infos basierend auf Suchbegriffen." }
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
		output = append(output, fmt.Sprintf("- [%s] %s (Datum: %s)", entry.Category, entry.Content, entry.Timestamp.Format("2006-01-02")))
	}
	return strings.Join(output, "\n"), nil
}

// --- 5. SKILL: WEB BROWSER ---

type WebBrowserSkill struct{}

type BrowserArgs struct {
	Action      string `json:"action"` // "navigate" or "search"
	URL         string `json:"url,omitempty"`
	SearchQuery string `json:"search_query,omitempty"`
}

func (s *WebBrowserSkill) Name() string        { return "web_browser" }
func (s *WebBrowserSkill) Description() string { return "Öffnet einen lokalen Browser um Webseiten zu laden, Screenshots zu machen und Inhalte zu analysieren." }
func (s *WebBrowserSkill) RiskLevel() RiskLevel { return RiskCritical }

func (s *WebBrowserSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action":       map[string]interface{}{"type": "string", "enum": []string{"navigate", "search"}, "description": "Aktion: navigieren oder suchen"},
			"url":          map[string]interface{}{"type": "string", "description": "Die URL bei 'navigate' (z.B. 'https://wikipedia.org')"},
			"search_query": map[string]interface{}{"type": "string", "description": "Suchbegriff bei 'search'"},
		},
		"required": []string{"action"},
	}
}

func findChromePath() string {
	paths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`chrome`,
		`msedge`,
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		if !strings.Contains(p, `\`) {
			if _, err := exec.LookPath(p); err == nil {
				return p
			}
		}
	}
	return ""
}

func cleanHTMLText(html string) (string, string) {
	// 1. Title extraction
	title := "Unbekannter Titel"
	reTitle := regexp.MustCompile(`(?i)<title>(.*?)</title>`)
	if matches := reTitle.FindStringSubmatch(html); len(matches) > 1 {
		title = strings.TrimSpace(matches[1])
	}

	// 2. Strip scripts and styles
	reScript := regexp.MustCompile(`(?s)<script.*?>.*?</script>`)
	html = reScript.ReplaceAllString(html, " ")
	reStyle := regexp.MustCompile(`(?s)<style.*?>.*?</style>`)
	html = reStyle.ReplaceAllString(html, " ")
	reNav := regexp.MustCompile(`(?s)<nav.*?>.*?</nav>`)
	html = reNav.ReplaceAllString(html, " ")
	reFooter := regexp.MustCompile(`(?s)<footer.*?>.*?</footer>`)
	html = reFooter.ReplaceAllString(html, " ")

	// 3. Strip tags
	reTags := regexp.MustCompile(`<[^>]*>`)
	text := reTags.ReplaceAllString(html, " ")

	// 4. Condense spaces
	reSpaces := regexp.MustCompile(`\s+`)
	text = reSpaces.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Limit to ~2000 characters
	if len(text) > 2000 {
		text = text[:2000] + "... [Text gekürzt]"
	}

	return title, text
}

func (s *WebBrowserSkill) Execute(args json.RawMessage) (string, error) {
	var input BrowserArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	targetURL := input.URL
	if input.Action == "search" {
		escapedQuery := strings.ReplaceAll(input.SearchQuery, " ", "+")
		targetURL = "https://www.google.com/search?q=" + escapedQuery
	}

	if targetURL == "" {
		return "", errors.New("keine Ziel-URL oder Suchanfrage übergeben")
	}

	// Verify standard URL schemes
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	chromePath := findChromePath()
	if chromePath == "" {
		LogKernelActivity("BROWSER_FAILED", targetURL, "ERROR")
		return "", errors.New("kein installierter Browser (Google Chrome oder Microsoft Edge) auf dem Windows-Host gefunden")
	}
	LogKernelActivity("BROWSER_START", targetURL, "PENDING")

	screenshotPath, err := filepath.Abs("./vgt_workspace/browser_screenshot.png")
	if err != nil {
		return "", err
	}
	_ = os.MkdirAll(filepath.Dir(screenshotPath), 0755)
	_ = os.Remove(screenshotPath) // Delete old screenshot

	tempProfile := filepath.Join(os.TempDir(), "aethel_chrome_profile")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Step 1: Capture Screenshot
	cmdScreenshot := exec.CommandContext(ctx, chromePath,
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--incognito",
		"--user-data-dir="+tempProfile,
		"--window-size=1280,720",
		"--screenshot="+screenshotPath,
		targetURL,
	)
	_ = cmdScreenshot.Run() // Capture screenshot (ignore error, we check file next)

	// Verify screenshot exists
	if _, err := os.Stat(screenshotPath); err != nil {
		return "", fmt.Errorf("Fehler beim Laden der Webseite (Screenshot konnte nicht generiert werden / Timeout): %v", err)
	}

	// Step 2: Dump HTML
	cmdHTML := exec.CommandContext(ctx, chromePath,
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--incognito",
		"--user-data-dir="+tempProfile,
		"--dump-html",
		targetURL,
	)
	var outHTML bytes.Buffer
	cmdHTML.Stdout = &outHTML
	_ = cmdHTML.Run()

	title, content := cleanHTMLText(outHTML.String())

	LogKernelActivity("BROWSER", targetURL, "SUCCESS")
	result := fmt.Sprintf("Aktion: %s\nURL: %s\nTitel: %s\nExtrahierter Text-Inhalt:\n%s", input.Action, targetURL, title, content)
	return result, nil
}

// --- 6. SKILL: GUI CONTROL ---

type GUIControlSkill struct{}

type GUIControlArgs struct {
	Action string `json:"action"` // "move", "click", "type", "press", "position"
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Button string `json:"button,omitempty"` // "left", "right", "double"
	Text   string `json:"text,omitempty"`
	Keys   string `json:"keys,omitempty"` // e.g. "{ENTER}", "{TAB}", "^t", "^w"
}

func (s *GUIControlSkill) Name() string        { return "gui_control" }
func (s *GUIControlSkill) Description() string { return "Steuert Maus, Tastatur und GUI-Elemente (Tabs, Tastenanschläge, Klicks). Sicherheitskritisch." }
func (s *GUIControlSkill) RiskLevel() RiskLevel { return RiskCritical }

func (s *GUIControlSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type": "string",
				"enum": []string{"move", "click", "type", "press", "position"},
				"description": "Die auszuführende GUI-Aktion: 'move' (Maus bewegen), 'click' (Klicken), 'type' (Text tippen), 'press' (Spezialtasten/Shortcuts drücken), 'position' (Maus-Koordinaten und Auflösung abfragen)",
			},
			"x":      map[string]interface{}{"type": "integer", "description": "Absolute X-Koordinate (für 'move')"},
			"y":      map[string]interface{}{"type": "integer", "description": "Absolute Y-Koordinate (für 'move')"},
			"button": map[string]interface{}{"type": "string", "enum": []string{"left", "right", "double"}, "description": "Maustaste (für 'click')"},
			"text":   map[string]interface{}{"type": "string", "description": "Text, der getippt werden soll (für 'type')"},
			"keys":   map[string]interface{}{"type": "string", "description": "Tastenbezeichnung oder Tastenkombination wie '{ENTER}', '{TAB}', '^t' (Strg+T für neuen Tab), '^w' (Strg+W), '%{TAB}' (Alt+Tab) (für 'press')"},
		},
		"required": []string{"action"},
	}
}

func (s *GUIControlSkill) Execute(args json.RawMessage) (string, error) {
	var input GUIControlArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	defer func() {
		_ = CapturePrimaryDisplay()
	}()

	switch input.Action {
	case "move":
		psScript := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms;
			[System.Windows.Forms.Cursor]::Position = New-Object System.Drawing.Point(%d, %d);
		`, input.X, input.Y)
		cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		err := cmd.Run()
		if err != nil {
			LogKernelActivity("GUI_MOVE_FAILED", fmt.Sprintf("x:%d, y:%d", input.X, input.Y), "ERROR")
			return "", err
		}
		LogKernelActivity("GUI_MOVE", fmt.Sprintf("x:%d, y:%d", input.X, input.Y), "SUCCESS")
		return fmt.Sprintf("Mauszeiger erfolgreich zu X:%d Y:%d bewegt.", input.X, input.Y), nil

	case "click":
		var mouseEventCode string
		switch input.Button {
		case "right":
			// Right down (0x0008) and Right up (0x0010)
			mouseEventCode = "[Mouse]::mouse_event(0x0008, 0, 0, 0, 0); [Mouse]::mouse_event(0x0010, 0, 0, 0, 0);"
		case "double":
			// Left down/up twice
			mouseEventCode = "[Mouse]::mouse_event(0x0002, 0, 0, 0, 0); [Mouse]::mouse_event(0x0004, 0, 0, 0, 0); [Mouse]::mouse_event(0x0002, 0, 0, 0, 0); [Mouse]::mouse_event(0x0004, 0, 0, 0, 0);"
		default: // "left"
			// Left down (0x0002) and Left up (0x0004)
			mouseEventCode = "[Mouse]::mouse_event(0x0002, 0, 0, 0, 0); [Mouse]::mouse_event(0x0004, 0, 0, 0, 0);"
		}

		psScript := fmt.Sprintf(`
			Add-Type -TypeDefinition @'
			using System;
			using System.Runtime.InteropServices;
			public class Mouse {
				[DllImport("user32.dll")]
				public static extern void mouse_event(int dwFlags, int dx, int dy, int cButtons, int dwExtraInfo);
			}
'@;
			%s
		`, mouseEventCode)
		cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		err := cmd.Run()
		if err != nil {
			LogKernelActivity("GUI_CLICK_FAILED", input.Button, "ERROR")
			return "", err
		}
		LogKernelActivity("GUI_CLICK", input.Button, "SUCCESS")
		return fmt.Sprintf("Klick mit Taste '%s' erfolgreich ausgeführt.", input.Button), nil

	case "type":
		if input.Text == "" {
			return "", errors.New("kein text zum tippen übergeben")
		}
		// Escape single quotes for PowerShell
		escapedText := strings.ReplaceAll(input.Text, "'", "''")
		psScript := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms;
			[System.Windows.Forms.SendKeys]::SendWait('%s');
		`, escapedText)
		cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		err := cmd.Run()
		if err != nil {
			LogKernelActivity("GUI_TYPE_FAILED", input.Text, "ERROR")
			return "", err
		}
		LogKernelActivity("GUI_TYPE", input.Text, "SUCCESS")
		return "Text erfolgreich getippt.", nil

	case "press":
		if input.Keys == "" {
			return "", errors.New("keine tastenbezeichnung zum drücken übergeben")
		}
		// Escape single quotes for PowerShell
		escapedKeys := strings.ReplaceAll(input.Keys, "'", "''")
		psScript := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms;
			[System.Windows.Forms.SendKeys]::SendWait('%s');
		`, escapedKeys)
		cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		err := cmd.Run()
		if err != nil {
			LogKernelActivity("GUI_PRESS_FAILED", input.Keys, "ERROR")
			return "", err
		}
		LogKernelActivity("GUI_PRESS", input.Keys, "SUCCESS")
		return fmt.Sprintf("Taste/Shortcut '%s' erfolgreich gedrückt.", input.Keys), nil

	case "position":
		psScript := `
			Add-Type -AssemblyName System.Windows.Forms;
			$pos = [System.Windows.Forms.Cursor]::Position;
			$scr = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds;
			Write-Output ($pos.X.ToString() + ';' + $pos.Y.ToString() + ';' + $scr.Width.ToString() + ';' + $scr.Height.ToString())
		`
		cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
		out, err := cmd.Output()
		if err != nil {
			LogKernelActivity("GUI_POSITION_FAILED", "", "ERROR")
			return "", err
		}
		parts := strings.Split(strings.TrimSpace(string(out)), ";")
		if len(parts) < 4 {
			return "", errors.New("ungültige rückgabe von mouse/screen info")
		}
		LogKernelActivity("GUI_POSITION", "", "SUCCESS")
		return fmt.Sprintf("Maus-Position: X:%s, Y:%s | Bildschirmauflösung: %sx%s", parts[0], parts[1], parts[2], parts[3]), nil

	default:
		return "", fmt.Errorf("unbekannte GUI-Aktion: %s", input.Action)
	}
}

// --- 7. SKILL: LIST DIRECTORY ---

type ListDirSkill struct{}

type ListDirArgs struct {
	Path string `json:"path"`
}

func (s *ListDirSkill) Name() string        { return "fs_list_dir" }
func (s *ListDirSkill) Description() string { return "Listet alle Dateien und Ordner in einem Verzeichnis im Workspace oder einem eingehängten Ordner auf." }
func (s *ListDirSkill) RiskLevel() RiskLevel { return RiskSafe }

func (s *ListDirSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string", "description": "Relativer Pfad im Workspace (z.B. '.') oder ein absoluter Pfad zu einem eingehängten Ordner"},
		},
		"required": []string{"path"},
	}
}

func (s *ListDirSkill) Execute(args json.RawMessage) (string, error) {
	var input ListDirArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	safePath, err := validatePath(input.Path)
	if err != nil {
		return "", err
	}

	files, err := os.ReadDir(safePath)
	if err != nil {
		LogKernelActivity("LIST_DIR_FAILED", input.Path, "ERROR")
		return "", err
	}

	var fileInfos []string
	for _, f := range files {
		info, err := f.Info()
		var sizeStr string
		if err == nil {
			sizeStr = fmt.Sprintf("%d Bytes", info.Size())
		} else {
			sizeStr = "Größe unbekannt"
		}
		
		typeStr := "Datei"
		if f.IsDir() {
			typeStr = "Ordner"
			sizeStr = "-"
		}
		
		fileInfos = append(fileInfos, fmt.Sprintf("- [%s] %s (Größe: %s)", typeStr, f.Name(), sizeStr))
	}

	LogKernelActivity("LIST_DIR", input.Path, "SUCCESS")
	if len(fileInfos) == 0 {
		return "Verzeichnis ist leer.", nil
	}
	return strings.Join(fileInfos, "\n"), nil
}

// --- 8. SKILL: MOUNT FOLDER ---

type MountFolderSkill struct{}

type MountFolderArgs struct {
	Path string `json:"path"`
}

func (s *MountFolderSkill) Name() string        { return "fs_mount_folder" }
func (s *MountFolderSkill) Description() string { return "Hängt einen Ordner vom Host-PC in das System ein, damit Aethel lesend/schreibend darauf zugreifen darf." }
func (s *MountFolderSkill) RiskLevel() RiskLevel { return RiskCritical }

func (s *MountFolderSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string", "description": "Absoluter Pfad des Ordners, der eingehängt werden soll (z.B. 'C:\\Users\\Masterboard\\Documents')"},
		},
		"required": []string{"path"},
	}
}

func (s *MountFolderSkill) Execute(args json.RawMessage) (string, error) {
	var input MountFolderArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}

	// Verify if directory actually exists on the host
	info, err := os.Stat(input.Path)
	if err != nil {
		return "", fmt.Errorf("Ordner existiert nicht oder Zugriff verweigert: %v", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("Pfad '%s' ist kein Ordner", input.Path)
	}

	err = state.AddMountedDir(input.Path)
	if err != nil {
		LogKernelActivity("MOUNT_FAILED", input.Path, "ERROR")
		return "", fmt.Errorf("Fehler beim Speichern der Einhängung: %v", err)
	}

	LogKernelActivity("MOUNT_FOLDER", input.Path, "SUCCESS")
	
	// List currently mounted dirs
	mounts := state.GetMountedDirs()
	var mountsList []string
	for _, m := range mounts {
		mountsList = append(mountsList, "- "+m)
	}

	return fmt.Sprintf("Ordner '%s' erfolgreich eingehängt.\n\nAktuell eingehängte Ordner:\n%s", input.Path, strings.Join(mountsList, "\n")), nil
}

// --- SKILL: EXTERNAL AGENT HANDOFF ---

type ExternalAgentHandoffSkill struct{}

type HandoffArgs struct {
	AgentName      string `json:"agent_name"` // "codex" | "chatgpt" | "gemini" | "cursor"
	Objective      string `json:"objective"`
	ProjectContext string `json:"project_context,omitempty"`
	FilesContext   string `json:"files_context,omitempty"`
}

func (s *ExternalAgentHandoffSkill) Name() string        { return "agent_handoff" }
func (s *ExternalAgentHandoffSkill) Description() string { return "Übergibt eine strukturierte Aufgabe an einen externen Agenten (z.B. Codex, ChatGPT, Gemini, Cursor). Erstellt einen detaillierten Übergabe-Prompt und öffnet das Agenten-Interface." }
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
	_ = os.MkdirAll(filepath.Dir(payloadPath), 0755)
	err := os.WriteFile(payloadPath, []byte(handoffPrompt), 0644)
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
		cmd := exec.Command("cmd.exe", "/c", "start", "cursor")
		_ = cmd.Start()
	} else {
		cmd := exec.Command("cmd.exe", "/c", "start", openURL)
		_ = cmd.Start()
	}

	LogKernelActivity("HANDOFF", input.AgentName, "SUCCESS")

	return fmt.Sprintf("HANDOFF: Übergabe-Protokoll generiert in './vgt_workspace/handoff_payload.md'. Interface für '%s' wurde geöffnet.", input.AgentName), nil
}

func CapturePrimaryDisplay() error {
	screenshotPath, err := filepath.Abs("./vgt_workspace/screenshot.jpg")
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(screenshotPath), 0755)

	psScript := fmt.Sprintf(`
		[Reflection.Assembly]::LoadWithPartialName("System.Drawing") | Out-Null;
		[Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms") | Out-Null;
		
		# Get monitor containing the mouse cursor
		$mousePos = [System.Windows.Forms.Cursor]::Position;
		$targetScreen = [System.Windows.Forms.Screen]::FromPoint($mousePos);
		$bounds = $targetScreen.Bounds;
		
		$bmp = New-Object System.Drawing.Bitmap($bounds.Width, $bounds.Height);
		$graphics = [System.Drawing.Graphics]::FromImage($bmp);
		$graphics.CopyFromScreen($bounds.X, $bounds.Y, 0, 0, $bmp.Size);
		
		# Downscale if wider than 1280px
		$maxWidth = 1280;
		if ($bounds.Width -gt $maxWidth) {
			$targetWidth = $maxWidth;
			$targetHeight = [int]($bounds.Height * ($targetWidth / $bounds.Width));
			$resizedBmp = New-Object System.Drawing.Bitmap($targetWidth, $targetHeight);
			$gResized = [System.Drawing.Graphics]::FromImage($resizedBmp);
			$gResized.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBilinear;
			$gResized.DrawImage($bmp, 0, 0, $targetWidth, $targetHeight);
			
			$saveBmp = $resizedBmp;
			$gResized.Dispose();
		} else {
			$saveBmp = $bmp;
		}
		
		# Compress as JPEG with 70% Quality to minimize byte size and load instantly
		$jpegCodec = [System.Drawing.Imaging.ImageCodecInfo]::GetImageEncoders() | Where-Object { $_.FormatID -eq [System.Drawing.Imaging.ImageFormat]::Jpeg.Guid };
		$encoderParams = New-Object System.Drawing.Imaging.EncoderParameters(1);
		$encoderParams.Param[0] = New-Object System.Drawing.Imaging.EncoderParameter([System.Drawing.Imaging.Encoder]::Quality, 70);
		
		$saveBmp.Save('%s', $jpegCodec, $encoderParams);
		
		if ($saveBmp -ne $bmp) {
			$saveBmp.Dispose();
		}
		$graphics.Dispose();
		$bmp.Dispose();
	`, strings.ReplaceAll(screenshotPath, "'", "''"))

	cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
	return cmd.Run()
}

var (
	screenshotMutex  sync.Mutex
	lastCaptureTime  time.Time
	cachedScreenshot []byte
)

func GetLatestScreenshot() ([]byte, error) {
	screenshotMutex.Lock()
	defer screenshotMutex.Unlock()

	// If cache is fresh (less than 800ms old), serve directly
	if len(cachedScreenshot) > 0 && time.Since(lastCaptureTime) < 800*time.Millisecond {
		return cachedScreenshot, nil
	}

	// Capture new screenshot
	err := CapturePrimaryDisplay()
	if err != nil {
		if len(cachedScreenshot) > 0 {
			// Fallback to cache on transient capturing failures
			return cachedScreenshot, nil
		}
		return nil, err
	}

	screenshotPath, err := filepath.Abs("./vgt_workspace/screenshot.jpg")
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(screenshotPath)
	if err != nil {
		if len(cachedScreenshot) > 0 {
			return cachedScreenshot, nil
		}
		return nil, err
	}

	cachedScreenshot = data
	lastCaptureTime = time.Now()
	return cachedScreenshot, nil
}



