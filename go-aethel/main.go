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
)

//go:embed frontend/*
var frontendFS embed.FS

const (
	groqURL    = "https://api.groq.com/openai/v1/chat/completions"
	configFile = "./vgt_workspace/aethel_config.json"
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
)

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
	APIKey       string   `json:"api_key"`
	OpenAIAPIKey string   `json:"openai_api_key,omitempty"`
	MountedDirs  []string `json:"mounted_dirs,omitempty"`
}

type AppState struct {
	mu           sync.RWMutex
	apiKey       string
	openaiAPIKey string
	mountedDirs  []string
	guard        *SecurityGuard
	leases       *LeaseManager
	audit        *AuditLogger
	policy       *PolicyEngine
	skills       *SkillRegistry
	memory       *LocalMemoryStore
	voice        *VoiceRegistry
	vault        *SecretVault
	tasks        *TaskEngine
}

var state *AppState

func loadConfig() (string, string, []string) {
	// 1. Env Variables
	key := os.Getenv("GROQ_API_KEY")
	oKey := os.Getenv("OPENAI_API_KEY")
	var mountedDirs []string

	// 2. Local config file
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
			mountedDirs = cfg.MountedDirs
		}
	}
	if mountedDirs == nil {
		mountedDirs = []string{}
	}
	return key, oKey, mountedDirs
}

func (s *AppState) saveConfig(key, oKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.apiKey = key
	s.openaiAPIKey = oKey
	cfg := Config{APIKey: key, OpenAIAPIKey: oKey, MountedDirs: s.mountedDirs}
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
	cfg := Config{APIKey: s.apiKey, OpenAIAPIKey: s.openaiAPIKey, MountedDirs: s.mountedDirs}
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

func (s *AppState) isConfigured() bool {
	key := s.getAPIKey()
	return key != "" && strings.HasPrefix(key, "gsk_")
}

func main() {
	log.Println("🛡️ VGT AETHEL :: INITIALISIERUNG (GO CORTEX)...")

	// Init State
	memoryStore := NewLocalMemoryStore()
	registry := NewSkillRegistry()
	registry.Register(&ExecuteCommandSkill{})
	registry.Register(&ReadFileSkill{})
	registry.Register(&WriteFileSkill{})
	registry.Register(&MemorySaveSkill{Store: memoryStore})
	registry.Register(&MemoryRecallSkill{Store: memoryStore})
	registry.Register(&WebBrowserSkill{})
	registry.Register(&GUIControlSkill{})
	registry.Register(&ListDirSkill{})
	registry.Register(&MountFolderSkill{})
	registry.Register(&ExternalAgentHandoffSkill{})


	gKey, oKey, mDirs := loadConfig()

	guard := NewSecurityGuard()
	leases := NewLeaseManager("./vgt_workspace/active_leases.json")
	audit := NewAuditLogger("./vgt_workspace/security_audit.json")
	policy := NewPolicyEngine(guard, leases, audit)

	voiceRegistry := NewVoiceRegistry()
	voiceRegistry.LoadLocalVoices()

	vault, err := NewSecretVault("./vgt_workspace/secret_vault.enc", "./vgt_workspace/vault.key")
	if err != nil {
		log.Fatalf("Failed to initialize secret vault: %v", err)
	}

	taskEngine := NewTaskEngine("./vgt_workspace/tasks.json")
	_ = taskEngine.Load()

	state = &AppState{
		apiKey:       gKey,
		openaiAPIKey: oKey,
		mountedDirs:  mDirs,
		guard:        guard,
		leases:       leases,
		audit:        audit,
		policy:       policy,
		skills:       registry,
		memory:       memoryStore,
		voice:        voiceRegistry,
		vault:        vault,
		tasks:        taskEngine,
	}

	state.tasks.Start()


	// 1. Static Web UI (Embedded)
	sub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatalf("Failed to load embedded frontend: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(sub)))

	// 2. Core API Routing
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/v1/setup", handleSetup)
	http.HandleFunc("/v1/models", handleModels)
	http.HandleFunc("/v1/chat", handleChat)
	http.HandleFunc("/v1/tools/execute", handleToolExecute)
	http.HandleFunc("/browser/screenshot.png", handleBrowserScreenshot)
	http.HandleFunc("/v1/audio/speech", handleAudioSpeech)
	http.HandleFunc("/v1/audio/voices", handleAudioVoices)
	http.HandleFunc("/v1/audio/transcribe", handleAudioTranscribe)
	http.HandleFunc("/v1/kernel/logs", handleKernelLogs)
	http.HandleFunc("/v1/chat/history", handleChatHistory)
	http.HandleFunc("/v1/chat/sessions", handleChatSessions)
	http.HandleFunc("/v1/chat/sessions/load", handleChatSessionsLoad)
	http.HandleFunc("/v1/chat/sessions/save", handleChatSessionsSave)
	http.HandleFunc("/v1/chat/sessions/delete", handleChatSessionsDelete)
	http.HandleFunc("/v1/kernel/tasks/", handleKernelTasksPath)
	http.HandleFunc("/v1/security/leases", handleSecurityLeases)
	http.HandleFunc("/v1/security/audit", handleSecurityAudit)
	http.HandleFunc("/v1/security/status", handleSecurityStatus)
	http.HandleFunc("/v1/memory", handleMemory)
	http.HandleFunc("/v1/memory/search", handleMemorySearch)
	http.HandleFunc("/v1/audio/health", handleAudioHealth)
	http.HandleFunc("/v1/audio/test", handleAudioTest)
	http.HandleFunc("/v1/viewport/screenshot", handleViewportScreenshot)
	http.HandleFunc("/v1/viewport/status", handleViewportStatus)
	http.HandleFunc("/v1/secrets", handleSecrets)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🌐 VGT CORE ONLINE: http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
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

	json.NewEncoder(w).Encode(map[string]string{
		"status":       status,
		"mode":         "STREAMING",
		"core":         "GO-CORTEX",
		"openai_ready": oReady,
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
		APIKey       string `json:"api_key"`
		OpenAIAPIKey string `json:"openai_api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	if !strings.HasPrefix(req.APIKey, "gsk_") {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Ungültiges Groq API-Key Format. Muss mit 'gsk_' beginnen."})
		return
	}

	// Validate OpenAI Key format if provided
	if req.OpenAIAPIKey != "" && !strings.HasPrefix(req.OpenAIAPIKey, "sk-") {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Ungültiges OpenAI API-Key Format. Muss mit 'sk-' beginnen."})
		return
	}

	if err := state.saveConfig(req.APIKey, req.OpenAIAPIKey); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Core configured successfully."})
}

// Handler: /v1/models
func handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	models := []map[string]interface{}{
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

	json.NewEncoder(w).Encode(map[string]interface{}{
		"models": models,
	})
}

func handleBrowserScreenshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, "./vgt_workspace/browser_screenshot.png")
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
	if apiKey == "" {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: API Key not configured.\n\n")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Invalid JSON body.\n\n")
		return
	}

	// Prepare payload for Groq
	groqMessages := []map[string]interface{}{}
	if req.SystemPrompt != "" {
		groqMessages = append(groqMessages, map[string]interface{}{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}
	for _, m := range req.Messages {
		var mapped map[string]interface{}
		if err := json.Unmarshal(m, &mapped); err == nil {
			groqMessages = append(groqMessages, mapped)
		}
	}

	payload := map[string]interface{}{
		"model":          req.ModelID,
		"messages":       groqMessages,
		"temperature":    req.Temperature,
		"stream":         true,
		"stream_options": map[string]interface{}{"include_usage": true},
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

	// Send HTTP Request to Groq
	groqReq, err := http.NewRequest(http.MethodPost, groqURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Creation failed: %v\n\n", err)
		return
	}
	groqReq.Header.Set("Authorization", "Bearer "+apiKey)
	groqReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(groqReq)
	if err != nil {
		fmt.Fprintf(w, "data: [SYSTEM ERROR]: Connection failed: %v\n\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(w, "data: [GROQ API ERROR %d]: %s\n\n", resp.StatusCode, string(bodyBytes))
		return
	}

	// Read stream chunks
	buffer := make([]byte, 4096)
	var streamBuf strings.Builder

	flusher, ok := w.(http.Flusher)

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			streamBuf.WriteString(string(buffer[:n]))
			lines := strings.Split(streamBuf.String(), "\n")

			// Keep last line in buffer if incomplete
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

					// Structs to parse stream frame
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
								Content   string      `json:"content,omitempty"`
								ToolCalls []DeltaCall `json:"tool_calls,omitempty"`
							} `json:"delta"`
							FinishReason string `json:"finish_reason,omitempty"`
						} `json:"choices"`
						Usage *struct {
							PromptTokens     int `json:"prompt_tokens"`
							CompletionTokens int `json:"completion_tokens"`
							TotalTokens      int `json:"total_tokens"`
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

							if choice.Delta.Content != "" {
								fmt.Fprintf(w, "data: %s\n\n", choice.Delta.Content)
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

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"result": result,
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
