package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed frontend/*
var frontendFS embed.FS

const (
	groqURL      = "https://api.groq.com/openai/v1/chat/completions"
	deepseekURL  = "https://api.deepseek.com/chat/completions"
	configFile   = "./vgt_workspace/aethel_config.json"
)

type Activity struct {
	Timestamp string `json:"timestamp"`
	Op        string `json:"op"`
	Target    string `json:"target"`
	Status    string `json:"status"`
}

var (
	kernelLogs   []Activity
	kernelLogsMu sync.RWMutex

	// Global variables for conjoined file changes and checklist tracking
	currentChecklist   []map[string]interface{}
	currentChecklistMu sync.RWMutex

	currentSessionChanges   []FileChange
	currentSessionChangesMu sync.Mutex
)

type FileChange struct {
	Path    string `json:"path"`
	File    string `json:"file"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

func recordFileChange(path string, added, removed int) {
	currentSessionChangesMu.Lock()
	defer currentSessionChangesMu.Unlock()

	for i, change := range currentSessionChanges {
		if change.Path == path {
			currentSessionChanges[i].Added += added
			currentSessionChanges[i].Removed += removed
			return
		}
	}

	currentSessionChanges = append(currentSessionChanges, FileChange{
		Path:    path,
		File:    filepath.Base(path),
		Added:   added,
		Removed: removed,
	})
}


func LogKernelActivity(op, target, status string) {
	kernelLogsMu.Lock()
	defer kernelLogsMu.Unlock()

	// Shorten target text if very long (e.g. huge command lines)
	if len(target) > 100 {
		target = target[:97] + "..."
	}

	logItem := Activity{
		Timestamp: time.Now().Format("15:04:05"),
		Op:        op,
		Target:    target,
		Status:    status,
	}

	// Prepend
	kernelLogs = append([]Activity{logItem}, kernelLogs...)
	if len(kernelLogs) > 50 {
		kernelLogs = kernelLogs[:50]
	}
}

type Config struct {
	APIKey        string   `json:"api_key"`
	OpenAIAPIKey  string   `json:"openai_api_key,omitempty"`
	DeepSeekAPIKey string  `json:"deepseek_api_key,omitempty"`
	MountedDirs   []string `json:"mounted_dirs,omitempty"`
}

type AppState struct {
	mu              sync.RWMutex
	apiKey          string
	openaiAPIKey    string
	deepseekAPIKey  string
	mountedDirs     []string
	guard           *SecurityGuard
	leases          *LeaseManager
	audit           *AuditLogger
	policy          *PolicyEngine
	skills          *SkillRegistry
	memory          *LocalMemoryStore
	voice           *VoiceRegistry
	vault           *SecretVault
	tasks           *TaskEngine
}

var state *AppState

func loadConfig() (string, string, string, []string) {
	// 1. Env Variables
	key := os.Getenv("GROQ_API_KEY")
	oKey := os.Getenv("OPENAI_API_KEY")
	var mountedDirs []string

	// 2. Local config file
	dsKey := os.Getenv("DEEPSEEK_API_KEY")

	data, err := os.ReadFile(configFile)
	if err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err == nil {
			if key == "" {
				key = cfg.APIKey
			}
			if oKey == "" {
				oKey = cfg.OpenAIAPIKey
			}
			if dsKey == "" {
				dsKey = cfg.DeepSeekAPIKey
			}
			mountedDirs = cfg.MountedDirs
		}
	}
	if mountedDirs == nil {
		mountedDirs = []string{}
	}
	return key, oKey, dsKey, mountedDirs
}

func (s *AppState) saveConfig(key, oKey, dsKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.apiKey = key
	s.openaiAPIKey = oKey
	s.deepseekAPIKey = dsKey
	cfg := Config{APIKey: key, OpenAIAPIKey: oKey, DeepSeekAPIKey: dsKey, MountedDirs: s.mountedDirs}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(configFile), 0755)
	return os.WriteFile(configFile, data, 0644)
}

func (s *AppState) AddMountedDir(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	// Check if already mounted
	for _, d := range s.mountedDirs {
		if d == absDir {
			return nil
		}
	}

	s.mountedDirs = append(s.mountedDirs, absDir)

	// Save to config file
	cfg := Config{APIKey: s.apiKey, OpenAIAPIKey: s.openaiAPIKey, DeepSeekAPIKey: s.deepseekAPIKey, MountedDirs: s.mountedDirs}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

func (s *AppState) GetMountedDirs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Return a copy
	res := make([]string, len(s.mountedDirs))
	copy(res, s.mountedDirs)
	return res
}

func (s *AppState) getAPIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.apiKey
}

func (s *AppState) getOpenAIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.openaiAPIKey
}

func (s *AppState) getDeepSeekKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deepseekAPIKey
}

func (s *AppState) isConfigured() bool {
	// Configured if Groq key OR DeepSeek key is present
	groqKey := s.getAPIKey()
	dsKey := s.getDeepSeekKey()
	return (groqKey != "" && strings.HasPrefix(groqKey, "gsk_")) || (dsKey != "" && strings.HasPrefix(dsKey, "sk-"))
}

func main() {
	log.Println("🛡️ VGT AETHEL :: INITIALISIERUNG (WAILS DESKTOP)...")

	app := NewApp()

	// Embed the frontend. Wails' AssetServer serves it on the Wails virtual host.
	// API routes (/v1/*, /health) are NOT files → Wails calls Handler → APIHandler → APIRouter.
	// Everything on the same host = same-origin = no CORS, no navigation, no PowerShell windows.
	sub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatalf("Failed to load embedded frontend: %v", err)
	}

	err = wails.Run(&options.App{
		Title:             "VGT AETHEL",
		Width:             1440,
		Height:            900,
		MinWidth:          1024,
		MinHeight:         700,
		DisableResize:     false,
		StartHidden:       false,
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: 8, G: 8, B: 18, A: 255},
		AssetServer: &assetserver.Options{
			Assets:  sub,
			Handler: APIHandler, // fallback for paths not found in embedded FS
		},
		OnStartup:     app.startup,
		OnDomReady:    app.domReady,
		OnBeforeClose: app.beforeClose,
		OnShutdown:    app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})
	if err != nil {
		log.Fatalf("Wails failed: %v", err)
	}
}

// Handler: /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	status := "SETUP_REQUIRED"
	if state.isConfigured() {
		status = "READY"
	}

	oReady := "false"
	if state.getOpenAIKey() != "" {
		oReady = "true"
	}

	dsReady := "false"
	if state.getDeepSeekKey() != "" {
		dsReady = "true"
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":         status,
		"mode":           "STREAMING",
		"core":           "GO-CORTEX",
		"openai_ready":   oReady,
		"deepseek_ready": dsReady,
	})
}

// Handler: /v1/setup
func handleSetup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		APIKey         string `json:"api_key"`
		OpenAIAPIKey   string `json:"openai_api_key"`
		DeepSeekAPIKey string `json:"deepseek_api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	// At least one AI key must be present
	hasGroq := strings.HasPrefix(req.APIKey, "gsk_")
	hasDeepSeek := strings.HasPrefix(req.DeepSeekAPIKey, "sk-")
	if !hasGroq && !hasDeepSeek {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Mindestens ein gültiger Key erforderlich: Groq (gsk_...) oder DeepSeek (sk-...)." })
		return
	}

	// Validate OpenAI Key format if provided
	if req.OpenAIAPIKey != "" && !strings.HasPrefix(req.OpenAIAPIKey, "sk-") {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Ungültiges OpenAI API-Key Format. Muss mit 'sk-' beginnen."})
		return
	}

	if err := state.saveConfig(req.APIKey, req.OpenAIAPIKey, req.DeepSeekAPIKey); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Core configured successfully."})
}

// Handler: GET /v1/settings — returns current config status (keys masked)
func handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groqKey := state.getAPIKey()
	openaiKey := state.getOpenAIKey()
	dsKey := state.getDeepSeekKey()

	groqConfigured := groqKey != "" && strings.HasPrefix(groqKey, "gsk_")
	openaiConfigured := openaiKey != ""
	dsConfigured := dsKey != "" && strings.HasPrefix(dsKey, "sk-")

	groqPreview := ""
	if groqConfigured && len(groqKey) > 8 {
		groqPreview = groqKey[:8] + "****"
	}
	openaiPreview := ""
	if openaiConfigured && len(openaiKey) > 5 {
		openaiPreview = openaiKey[:5] + "****"
	}
	dsPreview := ""
	if dsConfigured && len(dsKey) > 5 {
		dsPreview = dsKey[:5] + "****"
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"groq_configured":      groqConfigured,
		"openai_configured":    openaiConfigured,
		"deepseek_configured":  dsConfigured,
		"groq_key_preview":     groqPreview,
		"openai_key_preview":   openaiPreview,
		"deepseek_key_preview": dsPreview,
		"mounted_dirs":         state.GetMountedDirs(),
	})
}

// Handler: POST /v1/settings/reset — clears all API keys and forces SETUP_REQUIRED
func handleSettingsReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := state.saveConfig("", "", ""); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
		return
	}

	// Also delete the config file entirely so it's a clean slate
	_ = os.Remove(configFile)

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Config reset. Setup required."})
}

// getLocalOllamaModels queries the local Ollama API to retrieve installed models.
func getLocalOllamaModels() []map[string]interface{} {
	client := &http.Client{
		Timeout: 300 * time.Millisecond, // fast timeout to prevent hanging if Ollama is not running
	}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	var list []map[string]interface{}
	for _, m := range result.Models {
		list = append(list, map[string]interface{}{
			"id":             "ollama/" + m.Name,
			"name":           m.Name,
			"provider":       "Ollama",
			"tier":           "Local",
			"cost_input_1m":  0.0,
			"cost_output_1m": 0.0,
		})
	}
	return list
}

// Handler: /v1/models
func handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	models := []map[string]interface{}{
		// --- DeepSeek ---
		{
			"id":             "deepseek/deepseek-v4-flash",
			"name":           "DeepSeek V4 Flash (Thinking)",
			"provider":       "DeepSeek",
			"tier":           "Diamond",
			"cost_input_1m":  0.14,
			"cost_output_1m": 0.28,
		},
		{
			"id":             "deepseek/deepseek-v4-pro",
			"name":           "DeepSeek V4 Pro (Thinking)",
			"provider":       "DeepSeek",
			"tier":           "Diamond",
			"cost_input_1m":  0.435,
			"cost_output_1m": 0.87,
		},
		// --- Groq ---
		{
			"id":             "openai/gpt-oss-120b",
			"name":           "GPT OSS 120B (Logic Core)",
			"provider":       "Groq",
			"tier":           "Diamond",
			"cost_input_1m":  0.15,
			"cost_output_1m": 0.60,
		},
		{
			"id":             "openai/gpt-oss-20b",
			"name":           "GPT OSS 20B (Safety Core)",
			"provider":       "Groq",
			"tier":           "Platinum",
			"cost_input_1m":  0.075,
			"cost_output_1m": 0.30,
		},
		{
			"id":             "qwen/qwen3.6-27b",
			"name":           "Qwen 3.6 27B",
			"provider":       "Groq",
			"tier":           "Platinum",
			"cost_input_1m":  0.08,
			"cost_output_1m": 0.12,
		},
		{
			"id":             "meta-llama/llama-4-scout-17b-16e-instruct",
			"name":           "Llama 4 Scout 17B 16E",
			"provider":       "Groq",
			"tier":           "Gold",
			"cost_input_1m":  0.06,
			"cost_output_1m": 0.10,
		},
	}

	// Merge Ollama local models dynamically
	if ollamaModels := getLocalOllamaModels(); len(ollamaModels) > 0 {
		models = append(models, ollamaModels...)
	} else {
		models = append(models, map[string]interface{}{
			"id":             "ollama/local-fallback",
			"name":           "Kein lokales Ollama gefunden (Offline/Nicht gestartet)",
			"provider":       "Ollama",
			"tier":           "Local",
			"cost_input_1m":  0.0,
			"cost_output_1m": 0.0,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"models": models,
	})
}

func handleBrowserScreenshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, "./vgt_workspace/browser_screenshot.png")
}

// sanitizeMessagesForAPI ensures that the message sequence is valid for LLM APIs.
// Specifically, it strips tool_calls from assistant messages if the corresponding
// tool responses are missing (which can happen on network drops), and strips orphan tool responses.
func sanitizeMessagesForAPI(messages []map[string]interface{}) []map[string]interface{} {
	// 1. Gather all tool response IDs that exist in the history
	activeToolResponses := make(map[string]bool)
	for _, msg := range messages {
		if msg["role"] == "tool" {
			if id, ok := msg["tool_call_id"].(string); ok {
				activeToolResponses[id] = true
			}
		}
	}

	// 2. Filter tool_calls in assistant messages and skip orphan tool messages
	var cleanMessages []map[string]interface{}
	for _, msg := range messages {
		if msg["role"] == "assistant" {
			if tcs, ok := msg["tool_calls"]; ok {
				if tcSlice, ok := tcs.([]interface{}); ok {
					var validTCs []interface{}
					for _, tc := range tcSlice {
						if tcMap, ok := tc.(map[string]interface{}); ok {
							if id, ok := tcMap["id"].(string); ok {
								if activeToolResponses[id] {
									validTCs = append(validTCs, tc)
								}
							}
						}
					}
					if len(validTCs) > 0 {
						msg["tool_calls"] = validTCs
					} else {
						delete(msg, "tool_calls")
					}
				}
			}

			// If both content and tool_calls are missing/empty, set a fallback content
			content, _ := msg["content"].(string)
			_, hasToolCalls := msg["tool_calls"]
			if strings.TrimSpace(content) == "" && !hasToolCalls {
				msg["content"] = "Aktion ausgeführt."
			}
		}

		if msg["role"] == "tool" {
			if id, ok := msg["tool_call_id"].(string); ok {
				// Verify if a corresponding assistant tool_call exists in the list
				hasMatchingCall := false
				for _, otherMsg := range messages {
					if otherMsg["role"] == "assistant" {
						if otherTCs, ok := otherMsg["tool_calls"].([]interface{}); ok {
							for _, tc := range otherTCs {
								if tcMap, ok := tc.(map[string]interface{}); ok {
									if otherId, ok := tcMap["id"].(string); ok && otherId == id {
										hasMatchingCall = true
										break
									}
								}
							}
						}
					}
				}
				if !hasMatchingCall {
					continue // Skip orphan tool response
				}
			}
		}

		cleanMessages = append(cleanMessages, msg)
	}

	return cleanMessages
}

// stripToolsFromMessages removes all tool messages and tool calls from history
// to support retrying requests on models that do not support tools at all.
func stripToolsFromMessages(messages []map[string]interface{}) []map[string]interface{} {
	var clean []map[string]interface{}
	for _, msg := range messages {
		if msg["role"] == "tool" {
			continue // remove tool responses
		}
		// clone msg
		copyMsg := make(map[string]interface{})
		for k, v := range msg {
			if k != "tool_calls" {
				copyMsg[k] = v
			}
		}
		// ensure content is not empty
		content, _ := copyMsg["content"].(string)
		if strings.TrimSpace(content) == "" {
			copyMsg["content"] = "Aktion ausgeführt."
		}
		clean = append(clean, copyMsg)
	}
	return clean
}

// Handler: /v1/chat (SSE Stream to browser)
type ChatRequest struct {
	ModelID      string            `json:"model_id"`
	Messages     []json.RawMessage `json:"messages"`
	Temperature  float64           `json:"temperature"`
	UseTools     bool              `json:"use_tools"`
	SystemPrompt string            `json:"system_prompt"`
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := state.getAPIKey()
	dsKey := state.getDeepSeekKey()

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Invalid JSON body.\n\n")
		return
	}

	// Reset conjoined file changes for this turn
	currentSessionChangesMu.Lock()
	currentSessionChanges = []FileChange{}
	currentSessionChangesMu.Unlock()

	// --- Determine provider from model ID prefix ---
	isOllama := strings.HasPrefix(req.ModelID, "ollama/")
	isDeepSeek := strings.HasPrefix(req.ModelID, "deepseek/")

	if isDeepSeek && dsKey == "" {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: DeepSeek API Key nicht konfiguriert. Bitte in den Settings eingeben.\n\n")
		return
	}
	if !isDeepSeek && !isOllama && apiKey == "" {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Groq API Key nicht konfiguriert.\n\n")
		return
	}

	// Strip provider prefix for the actual API call
	actualModelID := req.ModelID
	if isDeepSeek {
		actualModelID = strings.TrimPrefix(req.ModelID, "deepseek/")
	} else if isOllama {
		actualModelID = strings.TrimPrefix(req.ModelID, "ollama/")
		if actualModelID == "local-fallback" {
			fmt.Fprintf(w, "data: [SYSTEM ERROR]: Kein lokales Ollama gefunden. Bitte vergewissere dich, dass Ollama gestartet ist und Modelle installiert sind.\n\n")
			return
		}
	} else {
		// Map mock model IDs to real Groq model IDs
		switch actualModelID {
		case "openai/gpt-oss-120b":
			actualModelID = "llama-3.3-70b-versatile"
		case "openai/gpt-oss-20b":
			actualModelID = "llama-3.1-8b-instant"
		case "qwen/qwen3.6-27b":
			actualModelID = "qwen/qwen3.6-27b"
		case "meta-llama/llama-4-scout-17b-16e-instruct":
			actualModelID = "llama-3.3-70b-versatile"
		}
	}

	// Build messages
	messages := []map[string]interface{}{}
	if req.SystemPrompt != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}
	for _, m := range req.Messages {
		var mapped map[string]interface{}
		if err := json.Unmarshal(m, &mapped); err == nil {
			// Remove custom properties not supported by non-DeepSeek LLM APIs in the messages array
			if !isDeepSeek {
				delete(mapped, "reasoning_content")
			}
			messages = append(messages, mapped)
		}
	}

	// Clean up message sequence to satisfy API schemas (no orphan tool calls/responses)
	messages = sanitizeMessagesForAPI(messages)

	// For Ollama, we also prepend the system prompt to the first user message.
	// We keep the system role message as well, just in case the model's template uses it.
	if isOllama && req.SystemPrompt != "" {
		firstUserIndex := -1
		for i, m := range messages {
			if m["role"] == "user" {
				firstUserIndex = i
				break
			}
		}
		if firstUserIndex != -1 {
			if origContent, ok := messages[firstUserIndex]["content"].(string); ok {
				if !strings.HasPrefix(origContent, "SYSTEM PROTOCOL:") {
					messages[firstUserIndex]["content"] = fmt.Sprintf("SYSTEM PROTOCOL:\n%s\n\nOPERATOR INPUT:\n%s", req.SystemPrompt, origContent)
				}
			}
		}
	}

	payload := map[string]interface{}{
		"model":          actualModelID,
		"messages":       messages,
		"stream":         true,
		"stream_options": map[string]interface{}{"include_usage": true},
	}

	// DeepSeek: enable thinking mode, no temperature (unsupported in thinking mode)
	if isDeepSeek {
		payload["thinking"] = map[string]interface{}{"type": "enabled"}
		payload["reasoning_effort"] = "max"
	} else {
		payload["temperature"] = req.Temperature
	}

	if req.UseTools {
		payload["tools"] = state.skills.ToToolDefinitions()
		payload["tool_choice"] = "auto"
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Marshalling failed.\n\n")
		return
	}

	// Select endpoint + key
	targetURL := groqURL
	targetKey := apiKey
	if isDeepSeek {
		targetURL = deepseekURL
		targetKey = dsKey
	} else if isOllama {
		targetURL = "http://localhost:11434/v1/chat/completions"
		targetKey = "ollama"
	}

	httpClient := &http.Client{
		Timeout: 960 * time.Second,
	}

	var resp *http.Response
	var lastErr error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Recreate body buffer as Do() consumes it
		apiReq, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewBuffer(jsonBytes))
		if err != nil {
			fmt.Fprintf(w, "data: [SYSTEM ERROR]: Request creation failed: %v\n\n", err)
			return
		}
		apiReq.Header.Set("Authorization", "Bearer "+targetKey)
		apiReq.Header.Set("Content-Type", "application/json")

		resp, err = httpClient.Do(apiReq)
		if err == nil {
			break
		}

		lastErr = err
		log.Printf("[RETRY] Connection failed (attempt %d/%d): %v", attempt, maxRetries, err)

		if attempt < maxRetries {
			// Notify UI that a retry is happening
			fmt.Fprintf(w, "data: [SYSTEM WARNING]: Connection dropped. Retrying (Attempt %d/%d)... \n\n", attempt+1, maxRetries)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(time.Duration(attempt) * 800 * time.Millisecond)
		}
	}

	if lastErr != nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Connection failed after %d attempts: %v\n\n", maxRetries, lastErr)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)

		// Automatic tool fallback for local models or endpoints that don't support tools
		bodyLower := strings.ToLower(bodyStr)
		isToolError := strings.Contains(bodyLower, "does not support tools") ||
			strings.Contains(bodyLower, "tool_choice") ||
			strings.Contains(bodyLower, "does not support function") ||
			strings.Contains(bodyLower, "tools are not supported") ||
			strings.Contains(bodyLower, "tool calling") ||
			strings.Contains(bodyLower, "unsupported parameter") ||
			strings.Contains(bodyLower, "unknown parameter") ||
			strings.Contains(bodyLower, "unsupported field") ||
			strings.Contains(bodyLower, "invalid parameter") ||
			strings.Contains(bodyLower, "not support") ||
			resp.StatusCode == http.StatusBadRequest

		if isToolError && req.UseTools {
			log.Printf("[TOOL FALLBACK] Model %s does not support tools. Retrying without tools.", actualModelID)
			fmt.Fprintf(w, "data: [SYSTEM WARNING]: Core-Modell '%s' unterstützt keine Tools. Führe Anfrage ohne Tools aus... \n\n", actualModelID)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			
			// Remove tools from payload and strip tool messages from history
			delete(payload, "tools")
			delete(payload, "tool_choice")
			payload["messages"] = stripToolsFromMessages(messages)
			
			// Re-marshal and retry request
			retryBytes, err := json.Marshal(payload)
			if err == nil {
				retryReq, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewBuffer(retryBytes))
				if err == nil {
					retryReq.Header.Set("Authorization", "Bearer "+targetKey)
					retryReq.Header.Set("Content-Type", "application/json")
					
					retryResp, err := httpClient.Do(retryReq)
					if err == nil && retryResp.StatusCode == http.StatusOK {
						resp = retryResp
						defer resp.Body.Close()
						goto processStream // jump to streaming loop
					}
					if err == nil {
						bodyBytes, _ = io.ReadAll(retryResp.Body)
						bodyStr = string(bodyBytes)
						resp = retryResp
						defer resp.Body.Close()
					}
				}
			}
		}

		provider := "GROQ"
		if isDeepSeek {
			provider = "DEEPSEEK"
		} else if isOllama {
			provider = "OLLAMA"
		}
		fmt.Fprintf(w, "data: [%s API ERROR %d]: %s\n\n", provider, resp.StatusCode, bodyStr)
		return
	}

processStream:

	// Read stream chunks
	buffer := make([]byte, 4096)
	var streamBuf strings.Builder

	flusher, ok := w.(http.Flusher)

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			streamBuf.WriteString(string(buffer[:n]))
			lines := strings.Split(streamBuf.String(), "\n")

			if len(lines) > 0 {
				streamBuf.Reset()
				streamBuf.WriteString(lines[len(lines)-1])
				lines = lines[:len(lines)-1]
			}

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if line == "data: [DONE]" {
					break
				}

				if strings.HasPrefix(line, "data: ") {
					jsonStr := strings.TrimPrefix(line, "data: ")

					type DeltaCall struct {
						Index    int    `json:"index"`
						ID       string `json:"id,omitempty"`
						Type     string `json:"type,omitempty"`
						Function struct {
							Name      string `json:"name,omitempty"`
							Arguments string `json:"arguments,omitempty"`
						} `json:"function"`
					}
					type StreamChunk struct {
						Choices []struct {
							Delta struct {
								Content          string      `json:"content,omitempty"`
								ReasoningContent string      `json:"reasoning_content,omitempty"`
								ToolCalls        []DeltaCall `json:"tool_calls,omitempty"`
							} `json:"delta"`
							FinishReason string `json:"finish_reason,omitempty"`
						} `json:"choices"`
						Usage *struct {
							PromptTokens         int `json:"prompt_tokens"`
							CompletionTokens     int `json:"completion_tokens"`
							TotalTokens          int `json:"total_tokens"`
							PromptCacheHitTokens int `json:"prompt_cache_hit_tokens"`
						} `json:"usage,omitempty"`
					}

					var chunk StreamChunk
					if err := json.Unmarshal([]byte(jsonStr), &chunk); err == nil {
						if chunk.Usage != nil {
							usageJSON, _ := json.Marshal(chunk.Usage)
							fmt.Fprintf(w, "data: [[USAGE]]:%s\n\n", string(usageJSON))
							if ok {
								flusher.Flush()
							}
						}

						if len(chunk.Choices) > 0 {
							choice := chunk.Choices[0]

							// DeepSeek thinking/reasoning stream
							if choice.Delta.ReasoningContent != "" {
								reasoning := choice.Delta.ReasoningContent
								reasoning = strings.ReplaceAll(reasoning, "\n", "[VGT_NL]")
								reasoning = strings.ReplaceAll(reasoning, "\r", "")
								fmt.Fprintf(w, "data: [[THINKING]]:%s\n\n", reasoning)
								if ok {
									flusher.Flush()
								}
							}

							if choice.Delta.Content != "" {
								content := choice.Delta.Content
								content = strings.ReplaceAll(content, "\n", "[VGT_NL]")
								content = strings.ReplaceAll(content, "\r", "")
								fmt.Fprintf(w, "data: %s\n\n", content)
								if ok {
									flusher.Flush()
								}
							}

							if len(choice.Delta.ToolCalls) > 0 {
								toolCallJSON, _ := json.Marshal(choice.Delta.ToolCalls)
								fmt.Fprintf(w, "data: [[TOOL_DELTA]]:%s\n\n", string(toolCallJSON))
								if ok {
									flusher.Flush()
								}
							}

							if choice.FinishReason == "tool_calls" {
								fmt.Fprintf(w, "data: [[TOOL_COMMIT]]\n\n")
								if ok {
									flusher.Flush()
								}
							}
						}
					}
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(w, "data: [SYSTEM ERROR]: Stream break: %v\n\n", err)
			break
		}
	}

	// Flush conjoined file changes at the end of the stream
	currentSessionChangesMu.Lock()
	if len(currentSessionChanges) > 0 {
		changesJSON, _ := json.Marshal(currentSessionChanges)
		fmt.Fprintf(w, "data: [[FILE_CHANGES]]:%s\n\n", string(changesJSON))
		if ok {
			flusher.Flush()
		}
	}
	currentSessionChangesMu.Unlock()
}
// Handler: /v1/tools/execute
type ToolExecRequest struct {
	Name             string          `json:"name"`
	Args             json.RawMessage `json:"args"`
	OverrideSecurity bool            `json:"override_security"`
}

func handleToolExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

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

	// Policy Engine Evaluation
	argsStr := string(payload.Args)
	allowed, decision, report := state.policy.Evaluate(payload.Name, argsStr, payload.OverrideSecurity)

	if !allowed {
		if decision == "blocked" {
			LogKernelActivity("SECURITY_BLOCK", payload.Name, "BLOCKED")
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
			LogKernelActivity("SECURITY_WAIT", payload.Name, "WAITING")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "security_intervention", // Compatible name
				"risk_score": report.RiskScore,
				"risk_level": report.RiskLevel,
				"threats":    report.Threats,
				"capability": report.Capability,
				"message":    "VGT Security: Ausführung erfordert Operator-Zustimmung.",
			})
			return
		}
	}

	if payload.OverrideSecurity {
		LogKernelActivity("SECURITY_OVERRIDE", payload.Name, "OVERRIDDEN")
	} else {
		LogKernelActivity("EXECUTE_TOOL", payload.Name, "SUCCESS")
	}

	// EXECUTE
	result, err := skill.Execute(payload.Args)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	currentChecklistMu.RLock()
	checklistCopy := make([]map[string]interface{}, len(currentChecklist))
	copy(checklistCopy, currentChecklist)
	currentChecklistMu.RUnlock()

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

// Handler: /v1/security/leases
func handleSecurityLeases(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		leases := state.leases.GetLeases()
		json.NewEncoder(w).Encode(leases)
		return
	} else if r.Method == http.MethodPost {
		var req struct {
			Capability      string `json:"capability"`
			DurationMinutes int    `json:"duration_minutes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		leaseID := fmt.Sprintf("lease_%d", time.Now().UnixNano())
		lease := PermissionLease{
			LeaseID:             leaseID,
			Capability:          Capability(req.Capability),
			CreatedAt:           time.Now(),
			ExpiresAt:           time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute),
			RequiresVisibleMode: true,
			Revocable:           true,
			ApprovedBy:          "user",
			ApprovalMethod:      "ui",
		}

		// Setup scopes
		if lease.Capability == CapSysExec {
			lease.Scope.ForbiddenTargets = []string{"rm", "dd", "mkfs", "format", "shred", "wipe"}
		} else if lease.Capability == CapGuiPressKey {
			lease.Scope.ForbiddenKeys = []string{"alt+f4", "win+r"}
		}

		err := state.leases.AddLease(lease)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"lease_id": leaseID,
			"expires":  lease.ExpiresAt.Format(time.RFC3339),
		})
		return
	} else if r.Method == http.MethodDelete || (r.Method == http.MethodGet && r.URL.Query().Get("action") == "revoke") {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		revoked := state.leases.RevokeLease(id)
		if revoked {
			json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Lease revoked successfully"})
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Lease not found"})
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Handler: /v1/security/audit
func handleSecurityAudit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logs := state.audit.GetLogs()
	json.NewEncoder(w).Encode(logs)
}

// Handler: /v1/security/status
func handleSecurityStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	leases := state.leases.GetLeases()
	status := "ACTIVE"
	if state.audit.isTampered {
		status = "TAMPERED"
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"firewall_status":     "ACTIVE",
		"active_leases_count": len(leases),
		"audit_tampered":      state.audit.isTampered,
		"security_status":     status,
		"last_activity_hash":  state.audit.lastHash,
	})
}

// Handler: /v1/memory
func handleMemory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		memories := state.memory.GetAll()
		json.NewEncoder(w).Encode(memories)
		return
	} else if r.Method == http.MethodPost {
		var req struct {
			Content  string `json:"content"`
			Category string `json:"category"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Content == "" {
			http.Error(w, "Content is required", http.StatusBadRequest)
			return
		}

		id := state.memory.Add(req.Content, req.Category)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "id": id})
		return
	} else if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		deleted := state.memory.Delete(id)
		if deleted {
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Memory not found"})
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Handler: /v1/audio/speech (Premium TTS Proxy & Local Offline SAPI5 Fallback)
func handleAudioSpeech(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var text string
	var selectedVoice string
	if r.Method == http.MethodGet {
		text = r.URL.Query().Get("text")
		selectedVoice = r.URL.Query().Get("voice")
	} else {
		var req struct {
			Text  string `json:"text"`
			Voice string `json:"voice"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			text = req.Text
			selectedVoice = req.Voice
		}
	}

	if text == "" {
		http.Error(w, "Missing text parameter", http.StatusBadRequest)
		return
	}

	audioBytes, mime, err := state.voice.SynthesizeWithFallback(text, selectedVoice)
	if err != nil {
		http.Error(w, fmt.Sprintf("Speech synthesis failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mime)
	w.WriteHeader(http.StatusOK)
	w.Write(audioBytes)
}

// Handler: /v1/kernel/logs
func handleKernelLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	kernelLogsMu.RLock()
	defer kernelLogsMu.RUnlock()

	if kernelLogs == nil {
		kernelLogs = []Activity{}
	}

	json.NewEncoder(w).Encode(kernelLogs)
}

// Handler: /v1/chat/history (Persistence)
const historyFile = "./vgt_workspace/chat_history.json"

func handleChatHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		data, err := os.ReadFile(historyFile)
		if err != nil {
			// Return empty array if not exists
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	} else if r.Method == http.MethodPost {
		// Read body and validate it is valid JSON
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var testArr []interface{}
		if err := json.Unmarshal(bodyBytes, &testArr); err != nil {
			http.Error(w, "Invalid JSON array", http.StatusBadRequest)
			return
		}

		_ = os.MkdirAll(filepath.Dir(historyFile), 0755)
		if err := os.WriteFile(historyFile, bodyBytes, 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// SessionInfo represents metadata about a saved session
type SessionInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Timestamp string `json:"timestamp"`
}

// Handler: /v1/chat/sessions
func handleChatSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionsDir := "./vgt_workspace/sessions"
	_ = os.MkdirAll(sessionsDir, 0755)

	files, err := os.ReadDir(sessionsDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var list []SessionInfo
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), "session_") && strings.HasSuffix(f.Name(), ".json") {
			id := strings.TrimSuffix(strings.TrimPrefix(f.Name(), "session_"), ".json")
			
			filePath := filepath.Join(sessionsDir, f.Name())
			data, err := os.ReadFile(filePath)
			if err == nil {
				var msgs []map[string]interface{}
				title := "Leerer Impuls"
				if err := json.Unmarshal(data, &msgs); err == nil && len(msgs) > 0 {
					for _, m := range msgs {
						if r, ok := m["role"].(string); ok && (r == "user" || r == "assistant") {
							if val, ok := m["content"].(string); ok && val != "" {
								if len(val) > 28 {
									title = val[:25] + "..."
								} else {
									title = val
								}
								break
							}
						}
					}
				}
				
				info, _ := f.Info()
				list = append(list, SessionInfo{
					ID:        id,
					Title:     title,
					Timestamp: info.ModTime().Format("2006-01-02 15:04"),
				})
			}
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Timestamp > list[j].Timestamp
	})

	if list == nil {
		list = []SessionInfo{}
	}

	json.NewEncoder(w).Encode(list)
}

// Handler: /v1/chat/sessions/load?id=XYZ
func handleChatSessionsLoad(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("./vgt_workspace/sessions", "session_"+id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// Handler: /v1/chat/sessions/save
func handleChatSessionsSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID       string            `json:"id"`
		Messages []json.RawMessage `json:"messages"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	sessionsDir := "./vgt_workspace/sessions"
	_ = os.MkdirAll(sessionsDir, 0755)

	filePath := filepath.Join(sessionsDir, "session_"+req.ID+".json")
	marshaled, err := json.MarshalIndent(req.Messages, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(filePath, marshaled, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Handler: /v1/chat/sessions/delete?id=XYZ
func handleChatSessionsDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodDelete && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("./vgt_workspace/sessions", "session_"+id+".json")
	_ = os.Remove(filePath)

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Handler: /v1/kernel/tasks (Legacy proxy calling task engine state manager)
func handleKernelTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		list := state.tasks.GetAll()
		if list == nil {
			list = []TaskItem{}
		}
		json.NewEncoder(w).Encode(list)
		return
	} else if r.Method == http.MethodPost {
		var req struct {
			Text                 string   `json:"text"`
			Objective            string   `json:"objective"`
			ScheduleType         string   `json:"schedule_type"`
			IntervalSeconds      int      `json:"interval_seconds"`
			RequiredCapabilities []string `json:"required_capabilities"`
			RiskLevel            string   `json:"risk_level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Text == "" {
			http.Error(w, "Task description is required", http.StatusBadRequest)
			return
		}

		id := fmt.Sprintf("task_%d", time.Now().UnixNano())
		objective := req.Objective
		if objective == "" {
			objective = req.Text
		}
		scheduleType := req.ScheduleType
		if scheduleType == "" {
			scheduleType = "once"
		}
		riskLevel := req.RiskLevel
		if riskLevel == "" {
			riskLevel = "Moderate"
		}

		task := TaskItem{
			ID:                   id,
			Text:                 req.Text,
			Objective:            objective,
			Done:                 false,
			Status:               "pending",
			ScheduleType:         scheduleType,
			IntervalSeconds:      req.IntervalSeconds,
			RequiredCapabilities: req.RequiredCapabilities,
			RiskLevel:            riskLevel,
			LimitSteps:           5,
			LimitToolCalls:       10,
		}

		err := state.tasks.Add(task)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "success", "id": id})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Handler: /v1/audio/transcribe (Whisper Audio transcription endpoint forwarding to Groq API)
func handleAudioTranscribe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 * 1024 * 1024)
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "Missing audio file in form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		http.Error(w, "Failed to read audio data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	transcript, err := state.voice.Transcribe(buf.Bytes(), header.Filename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Transcription failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"text": transcript})
}

// Handler: /v1/audio/voices
func handleAudioVoices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	voices := state.voice.GetAvailableVoices()
	json.NewEncoder(w).Encode(voices)
}


// Handler: /v1/audio/health
func handleAudioHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := state.voice.GetHealthStatus()
	json.NewEncoder(w).Encode(health)
}

// Handler: /v1/audio/test
func handleAudioTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Voice string `json:"voice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	testText := "Aethel Audio-Verbindungstest erfolgreich."
	audioBytes, mime, err := state.voice.SynthesizeWithFallback(testText, req.Voice)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	// Just confirm synthesis worked, return success status
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"message":   "Voice synthesis worked.",
		"mime_type": mime,
		"size":      len(audioBytes),
	})
}

// Handler: /v1/kernel/tasks/ (Unified task subpath routing)
func handleKernelTasksPath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	path = strings.TrimPrefix(path, "/v1/kernel/tasks/")

	// If it's a subpath action (e.g. "/v1/kernel/tasks/task_123/run")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		// Route to normal tasks list handler
		handleKernelTasks(w, r)
		return
	}

	id := parts[0]

	if len(parts) == 2 {
		action := parts[1]
		if action == "run" && r.Method == http.MethodPost {
			err := state.tasks.TriggerManual(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
			return
		}
		if action == "pause" && r.Method == http.MethodPost {
			err := state.tasks.Pause(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "success"})
			return
		}
	}

	if r.Method == http.MethodDelete {
		err := state.tasks.Delete(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// Handler: /v1/viewport/screenshot
func handleViewportScreenshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := GetLatestScreenshot()
	if err != nil {
		http.Error(w, fmt.Sprintf("Screenshot capture failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// Handler: /v1/viewport/status
func handleViewportStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"active": true,
		"width":  1920,
		"height": 1080,
	})
}

// Handler: /v1/secrets
func handleSecrets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodGet {
		list := state.vault.List()
		if list == nil {
			list = []SecretItem{}
		}
		json.NewEncoder(w).Encode(list)
		return
	}

	if r.Method == http.MethodPost {
		var item SecretItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if item.ID == "" || item.Token == "" {
			http.Error(w, "ID and Token are required", http.StatusBadRequest)
			return
		}
		err := state.vault.Add(item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "ID is required", http.StatusBadRequest)
			return
		}
		err := state.vault.Delete(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Handler: /v1/memory/search
func handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results := state.memory.Search(req.Query)
	if results == nil {
		results = []MemoryEntry{}
	}
	json.NewEncoder(w).Encode(results)
}

// getPowerShellPath resolves the absolute system path to powershell.exe on Windows to prevent path environment errors.
func getPowerShellPath() string {
	if path, err := exec.LookPath("powershell.exe"); err == nil {
		return path
	}
	if path, err := exec.LookPath("powershell"); err == nil {
		return path
	}
	return "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"
}

// Handler: /v1/chat/checklist
func handleChecklist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		currentChecklistMu.RLock()
		defer currentChecklistMu.RUnlock()
		if currentChecklist == nil {
			json.NewEncoder(w).Encode([]interface{}{})
		} else {
			json.NewEncoder(w).Encode(currentChecklist)
		}
		return
	}

	if r.Method == http.MethodPost {
		var req []map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		currentChecklistMu.Lock()
		currentChecklist = req
		currentChecklistMu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}
}
