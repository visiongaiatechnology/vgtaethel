package handlers

import (
	"context"
	"encoding/json"
	"go-aethel/provider"
	"go-aethel/security"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	req.APIKey = strings.TrimSpace(req.APIKey)
	req.OpenAIAPIKey = strings.TrimSpace(req.OpenAIAPIKey)
	req.DeepSeekAPIKey = strings.TrimSpace(req.DeepSeekAPIKey)
	req.GeminiAPIKey = strings.TrimSpace(req.GeminiAPIKey)
	req.ClaudeAPIKey = strings.TrimSpace(req.ClaudeAPIKey)
	hasGroq := provider.IsConfiguredProviderKey("Groq", req.APIKey)
	hasDS := provider.IsConfiguredProviderKey("DeepSeek", req.DeepSeekAPIKey)
	hasOpenAI := provider.IsConfiguredProviderKey("OpenAI", req.OpenAIAPIKey)
	hasGemini := provider.IsConfiguredProviderKey("Gemini", req.GeminiAPIKey)
	hasClaude := provider.IsConfiguredProviderKey("Claude", req.ClaudeAPIKey)
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
		"groq_configured":      provider.IsConfiguredProviderKey("Groq", groqKey),
		"openai_configured":    provider.IsConfiguredProviderKey("OpenAI", openaiKey),
		"deepseek_configured":  provider.IsConfiguredProviderKey("DeepSeek", dsKey),
		"gemini_configured":    provider.IsConfiguredProviderKey("Gemini", gemKey),
		"claude_configured":    provider.IsConfiguredProviderKey("Claude", claudeKey),
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

	_ = os.Remove(filepath.Join(security.WorkspaceDir, "aethel_config.json"))

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Config reset. Setup required."})
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Serve configured, already verified models immediately. Live catalog
	// discovery augments the next request in the background and can therefore
	// never empty both dropdowns through a provider/network timeout.
	if state != nil && state.providers != nil {
		models := state.providers.AvailableModels(state)
		if ollamaModels := provider.GetLocalOllamaModels(); len(ollamaModels) > 0 {
			models = append(models, ollamaModels...)
		}
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"models": models}); err != nil {
			return
		}
		go func(registry *provider.ProviderRegistry, keys provider.KeyProvider) {
			discoveryContext, cancelDiscovery := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancelDiscovery()
			registry.RefreshAvailableModels(discoveryContext, keys)
		}(state.providers, state)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{"models": []map[string]interface{}{}})
}
