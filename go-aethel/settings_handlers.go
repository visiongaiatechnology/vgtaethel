package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

func handleSetup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		APIKey         string `json:"api_key"`
		OpenAIAPIKey   string `json:"openai_api_key"`
		DeepSeekAPIKey string `json:"deepseek_api_key"`
		GeminiAPIKey   string `json:"gemini_api_key"`
		ClaudeAPIKey   string `json:"claude_api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid JSON payload"})
		return
	}

	hasGroq := strings.HasPrefix(req.APIKey, "gsk_")
	hasDS := strings.HasPrefix(req.DeepSeekAPIKey, "sk-")
	hasOpenAI := strings.HasPrefix(req.OpenAIAPIKey, "sk-")
	hasGemini := strings.HasPrefix(req.GeminiAPIKey, "AIza")
	hasClaude := strings.HasPrefix(req.ClaudeAPIKey, "sk-ant-")
	isLocal := req.APIKey == "local"
	if !hasGroq && !hasDS && !isLocal && !hasOpenAI && !hasGemini && !hasClaude {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Mindestens ein gültiger Key erforderlich (Groq, DeepSeek, OpenAI, Gemini, Claude) oder lokale KI."})
		return
	}

	if err := state.saveConfig(req.APIKey, req.OpenAIAPIKey, req.DeepSeekAPIKey, req.GeminiAPIKey, req.ClaudeAPIKey); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Core configured successfully."})
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groqKey := state.getAPIKey()
	openaiKey := state.getOpenAIKey()
	dsKey := state.getDeepSeekKey()
	gemKey := state.getGeminiKey()
	claudeKey := state.getClaudeKey()

	maskKey := func(k string) string {
		if len(k) > 8 {
			return k[:8] + "****"
		}
		if k != "" {
			return "****"
		}
		return ""
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"groq_configured":      groqKey != "" && strings.HasPrefix(groqKey, "gsk_"),
		"openai_configured":    openaiKey != "" && strings.HasPrefix(openaiKey, "sk-"),
		"deepseek_configured":  dsKey != "" && strings.HasPrefix(dsKey, "sk-"),
		"gemini_configured":    gemKey != "" && strings.HasPrefix(gemKey, "AIza"),
		"claude_configured":    claudeKey != "" && strings.HasPrefix(claudeKey, "sk-ant-"),
		"groq_key_preview":     maskKey(groqKey),
		"openai_key_preview":   maskKey(openaiKey),
		"deepseek_key_preview": maskKey(dsKey),
		"gemini_key_preview":   maskKey(gemKey),
		"claude_key_preview":   maskKey(claudeKey),
		"mounted_dirs":         state.GetMountedDirs(),
		"mounts":               state.GetMounts(),
		"os":                   runtime.GOOS,
	})
}

func handleSettingsReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := state.saveConfig("", "", "", "", ""); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
		return
	}

	_ = os.Remove(configFile)

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Config reset. Setup required."})
}

func getLocalOllamaModels() []map[string]interface{} {
	client := &http.Client{
		Timeout: 300 * time.Millisecond,
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

func handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if state != nil && state.providers != nil {
		models := state.providers.PublicModels()
		if ollamaModels := getLocalOllamaModels(); len(ollamaModels) > 0 {
			models = append(models, ollamaModels...)
		} else {
			models = append(models, map[string]interface{}{"id": "ollama/local-fallback", "name": "Kein lokales Ollama gefunden (Offline/Nicht gestartet)", "provider": "Ollama", "tier": "Local", "cost_input_1m": 0.0, "cost_output_1m": 0.0, "supports_tools": false, "supports_vision": false})
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"models": models})
		return
	}

	models := []map[string]interface{}{
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
		{
			"id":             "llama-3.3-70b-versatile",
			"name":           "Llama 3.3 70B Versatile",
			"provider":       "Groq",
			"tier":           "Gold",
			"cost_input_1m":  0.59,
			"cost_output_1m": 0.79,
		},
		{
			"id":             "openai-native/gpt-5.5",
			"name":           "GPT-5.5 (Reasoning)",
			"provider":       "OpenAI",
			"tier":           "Diamond",
			"cost_input_1m":  5.0,
			"cost_output_1m": 30.0,
		},
		{
			"id":             "openai-native/gpt-5.4",
			"name":           "GPT-5.4 (Reasoning)",
			"provider":       "OpenAI",
			"tier":           "Diamond",
			"cost_input_1m":  2.50,
			"cost_output_1m": 15.0,
		},
		{
			"id":             "openai-native/gpt-5.4-mini",
			"name":           "GPT-5.4 Mini (Reasoning)",
			"provider":       "OpenAI",
			"tier":           "Platinum",
			"cost_input_1m":  0.75,
			"cost_output_1m": 4.50,
		},
		{
			"id":             "gemini/gemini-3.5-flash",
			"name":           "Gemini 3.5 Flash",
			"provider":       "Gemini",
			"tier":           "Gold",
			"cost_input_1m":  0.075,
			"cost_output_1m": 0.30,
		},
		{
			"id":             "gemini/gemini-3.1-flash-lite",
			"name":           "Gemini 3.1 Flash-Lite",
			"provider":       "Gemini",
			"tier":           "Gold",
			"cost_input_1m":  0.03,
			"cost_output_1m": 0.12,
		},
		{
			"id":             "gemini/gemini-3.1-pro-preview",
			"name":           "Gemini 3.1 Pro Preview",
			"provider":       "Gemini",
			"tier":           "Diamond",
			"cost_input_1m":  1.25,
			"cost_output_1m": 5.0,
		},
		{
			"id":             "claude/claude-fable-5",
			"name":           "Claude Fable 5 (Reasoning)",
			"provider":       "Claude",
			"tier":           "Diamond",
			"cost_input_1m":  15.0,
			"cost_output_1m": 75.0,
		},
		{
			"id":             "claude/claude-opus-4-8",
			"name":           "Claude Opus 4.8",
			"provider":       "Claude",
			"tier":           "Diamond",
			"cost_input_1m":  15.0,
			"cost_output_1m": 75.0,
		},
		{
			"id":             "claude/claude-sonnet-4-6",
			"name":           "Claude Sonnet 4.6",
			"provider":       "Claude",
			"tier":           "Platinum",
			"cost_input_1m":  3.0,
			"cost_output_1m": 15.0,
		},
		{
			"id":             "claude/claude-haiku-4-5",
			"name":           "Claude Haiku 4.5",
			"provider":       "Claude",
			"tier":           "Gold",
			"cost_input_1m":  0.25,
			"cost_output_1m": 1.25,
		},
	}

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

func handleProviderHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"providers": ProbeProviderHealth(context.Background(), state)})
}

func handleBrowserScreenshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, "./vgt_workspace/browser_screenshot.png")
}

type CostRecord struct {
	Timestamp        time.Time `json:"timestamp"`
	ModelID          string    `json:"model_id"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CachedHitTokens  int       `json:"cached_hit_tokens"`
	CostUSD          float64   `json:"cost_usd"`
}

var costLedgerMu sync.Mutex

func CalculateInferenceCost(modelID string, inputTokens, outputTokens, cachedTokens int) float64 {
	registry := (*ProviderRegistry)(nil)
	if state != nil {
		registry = state.providers
	}
	if registry == nil {
		registry = NewProviderRegistry()
	}
	if spec, ok := registry.Resolve(modelID); ok {
		missTokens := inputTokens - cachedTokens
		if missTokens < 0 {
			missTokens = 0
		}
		return (float64(missTokens)*spec.InputCostPerMTok + float64(cachedTokens)*spec.CachedCostPerMTok + float64(outputTokens)*spec.OutputCostPerMTok) / 1000000.0
	}
	inputPrice := 0.0
	outputPrice := 0.0
	cacheHitPrice := 0.0

	id := modelID
	if strings.HasPrefix(id, "deepseek/") {
		id = strings.TrimPrefix(id, "deepseek/")
	} else if strings.HasPrefix(id, "openai-native/") {
		id = strings.TrimPrefix(id, "openai-native/")
	} else if strings.HasPrefix(id, "gemini/") {
		id = strings.TrimPrefix(id, "gemini/")
	} else if strings.HasPrefix(id, "claude/") {
		id = strings.TrimPrefix(id, "claude/")
	}

	switch id {
	case "openai/gpt-oss-20b":
		inputPrice = 0.075
		outputPrice = 0.30
	case "openai/gpt-oss-120b":
		inputPrice = 0.15
		outputPrice = 0.60
	case "meta-llama/llama-4-scout-17b-16e-instruct":
		inputPrice = 0.11
		outputPrice = 0.34
	case "llama-3.3-70b-versatile":
		inputPrice = 0.59
		outputPrice = 0.79
	case "qwen/qwen3.6-27b":
		inputPrice = 0.60
		outputPrice = 3.00
	case "deepseek-v4-flash":
		inputPrice = 0.14
		cacheHitPrice = 0.0028
		outputPrice = 0.28
	case "deepseek-v4-pro":
		inputPrice = 0.435
		cacheHitPrice = 0.003625
		outputPrice = 0.87
	case "gpt-5.5":
		inputPrice = 5.00
		cacheHitPrice = 0.50
		outputPrice = 30.00
	case "gpt-5.4":
		inputPrice = 2.50
		cacheHitPrice = 0.25
		outputPrice = 15.00
	case "gpt-5.4-mini":
		inputPrice = 0.75
		cacheHitPrice = 0.075
		outputPrice = 4.50
	case "gemini-3.5-flash":
		inputPrice = 1.50
		cacheHitPrice = 0.15
		outputPrice = 9.00
	case "gemini-3.1-flash-lite":
		inputPrice = 0.25
		cacheHitPrice = 0.025
		outputPrice = 1.50
	case "gemini-3.1-pro-preview":
		inputPrice = 2.00
		cacheHitPrice = 0.20
		outputPrice = 12.00
	case "claude-fable-5":
		inputPrice = 10.00
		cacheHitPrice = 1.00
		outputPrice = 50.00
	case "claude-opus-4-8":
		inputPrice = 5.00
		cacheHitPrice = 0.50
		outputPrice = 25.00
	case "claude-sonnet-4-6":
		inputPrice = 3.00
		cacheHitPrice = 0.30
		outputPrice = 15.00
	case "claude-haiku-4-5":
		inputPrice = 1.00
		cacheHitPrice = 0.10
		outputPrice = 5.00
	}

	missTokens := inputTokens - cachedTokens
	if missTokens < 0 {
		missTokens = 0
	}

	cost := (float64(missTokens) * inputPrice / 1000000.0) +
		(float64(cachedTokens) * cacheHitPrice / 1000000.0) +
		(float64(outputTokens) * outputPrice / 1000000.0)

	return cost
}

func LogAPICall(modelID string, inputTokens, outputTokens, cachedTokens int) {
	costLedgerMu.Lock()
	defer costLedgerMu.Unlock()

	cost := CalculateInferenceCost(modelID, inputTokens, outputTokens, cachedTokens)

	record := CostRecord{
		Timestamp:        time.Now(),
		ModelID:          modelID,
		PromptTokens:     inputTokens,
		CompletionTokens: outputTokens,
		CachedHitTokens:  cachedTokens,
		CostUSD:          cost,
	}

	filePath := "./vgt_workspace/api_costs.json"
	var records []CostRecord
	data, err := os.ReadFile(filePath)
	if err == nil {
		_ = json.Unmarshal(data, &records)
	}

	records = append(records, record)

	if newData, err := json.MarshalIndent(records, "", "  "); err == nil {
		_ = os.WriteFile(filePath, newData, 0600)
	}
}

func handleCosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	costLedgerMu.Lock()
	defer costLedgerMu.Unlock()

	filePath := "./vgt_workspace/api_costs.json"
	var records []CostRecord
	data, err := os.ReadFile(filePath)
	if err == nil {
		_ = json.Unmarshal(data, &records)
	}

	now := time.Now()
	todayCost := 0.0
	monthCost := 0.0

	for _, rec := range records {
		if rec.Timestamp.Year() == now.Year() && rec.Timestamp.YearDay() == now.YearDay() {
			todayCost += rec.CostUSD
		}
		if rec.Timestamp.Year() == now.Year() && rec.Timestamp.Month() == now.Month() {
			monthCost += rec.CostUSD
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"today": todayCost,
		"month": monthCost,
	})
}

type CustomPersona struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SystemPrompt string `json:"system_prompt"`
}

var personasMu sync.Mutex

func readCustomPersonas() []CustomPersona {
	filePath := "./vgt_workspace/custom_aethels.json"
	var list []CustomPersona
	data, err := os.ReadFile(filePath)
	if err == nil {
		_ = json.Unmarshal(data, &list)
	}
	if list == nil {
		list = []CustomPersona{}
	}
	return list
}

func writeCustomPersonas(list []CustomPersona) error {
	filePath := "./vgt_workspace/custom_aethels.json"
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0600)
}

func handlePersonas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	personasMu.Lock()
	defer personasMu.Unlock()

	list := readCustomPersonas()

	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(list)
		return
	}

	if r.Method == http.MethodPost {
		var req CustomPersona
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.SystemPrompt == "" {
			http.Error(w, "Name und System-Prompt sind erforderlich.", http.StatusBadRequest)
			return
		}
		if len([]rune(req.Name)) > 80 || len([]rune(req.SystemPrompt)) > 128000 {
			http.Error(w, "Persona payload exceeds size limit", http.StatusRequestEntityTooLarge)
			return
		}

		if req.ID == "" {
			req.ID = fmt.Sprintf("persona_%d", time.Now().UnixNano())
			list = append(list, req)
		} else {
			if err := validateResourceID(req.ID); err != nil {
				http.Error(w, "Invalid persona ID", http.StatusBadRequest)
				return
			}
			found := false
			for i, p := range list {
				if p.ID == req.ID {
					list[i] = req
					found = true
					break
				}
			}
			if !found {
				list = append(list, req)
			}
		}

		if err := writeCustomPersonas(list); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "id": req.ID})
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "ID erforderlich", http.StatusBadRequest)
			return
		}
		if err := validateResourceID(id); err != nil {
			http.Error(w, "Invalid persona ID", http.StatusBadRequest)
			return
		}

		var newList []CustomPersona
		for _, p := range list {
			if p.ID != id {
				newList = append(newList, p)
			}
		}

		if err := writeCustomPersonas(newList); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
