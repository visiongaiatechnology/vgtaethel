package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ModelSpec is the authoritative model contract shared by UI, chat, cost
// tracking and agent runs. Unregistered models are never given tool access.
type ModelSpec struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Provider          string  `json:"provider"`
	Tier              string  `json:"tier"`
	InputCostPerMTok  float64 `json:"cost_input_1m"`
	OutputCostPerMTok float64 `json:"cost_output_1m"`
	CachedCostPerMTok float64 `json:"cost_cached_input_1m,omitempty"`
	ContextTokens     int     `json:"context_tokens"`
	MaxOutputTokens   int     `json:"max_output_tokens"`
	SupportsTools     bool    `json:"supports_tools"`
	SupportsVision    bool    `json:"supports_vision"`
	SupportsReasoning bool    `json:"supports_reasoning"`
	DefaultRunBudget  float64 `json:"default_run_budget_usd"`
}

type ProviderHealth struct {
	Provider   string `json:"provider"`
	Configured bool   `json:"configured"`
	Reachable  bool   `json:"reachable"`
	Detail     string `json:"detail"`
}

type ProviderRegistry struct {
	mu     sync.RWMutex
	models map[string]ModelSpec
}

func NewProviderRegistry() *ProviderRegistry {
	r := &ProviderRegistry{models: make(map[string]ModelSpec)}
	for _, spec := range defaultModelSpecs() {
		r.models[spec.ID] = spec
	}
	return r
}

func defaultModelSpecs() []ModelSpec {
	// Pricing/capabilities are data, deliberately isolated from transport code.
	// OpenAI GPT-5 values follow the current official model catalog.
	return []ModelSpec{
		{ID: "deepseek/deepseek-v4-flash", Name: "DeepSeek V4 Flash (Thinking)", Provider: "DeepSeek", Tier: "Diamond", InputCostPerMTok: .14, OutputCostPerMTok: .28, CachedCostPerMTok: .0028, ContextTokens: 1000000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 2},
		{ID: "deepseek/deepseek-v4-pro", Name: "DeepSeek V4 Pro (Thinking)", Provider: "DeepSeek", Tier: "Diamond", InputCostPerMTok: .435, OutputCostPerMTok: .87, CachedCostPerMTok: .003625, ContextTokens: 1000000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 4},
		{ID: "openai/gpt-oss-120b", Name: "GPT OSS 120B (Logic Core)", Provider: "Groq", Tier: "Diamond", InputCostPerMTok: .15, OutputCostPerMTok: .60, ContextTokens: 131072, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 2},
		{ID: "openai/gpt-oss-20b", Name: "GPT OSS 20B (Safety Core)", Provider: "Groq", Tier: "Platinum", InputCostPerMTok: .075, OutputCostPerMTok: .30, ContextTokens: 131072, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 1},
		{ID: "qwen/qwen3.6-27b", Name: "Qwen 3.6 27B", Provider: "Groq", Tier: "Platinum", InputCostPerMTok: .08, OutputCostPerMTok: .12, ContextTokens: 131072, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 1},
		{ID: "meta-llama/llama-4-scout-17b-16e-instruct", Name: "Llama 4 Scout 17B 16E", Provider: "Groq", Tier: "Gold", InputCostPerMTok: .06, OutputCostPerMTok: .10, ContextTokens: 131072, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true, DefaultRunBudget: 1},
		{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B Versatile", Provider: "Groq", Tier: "Gold", InputCostPerMTok: .59, OutputCostPerMTok: .79, ContextTokens: 131072, MaxOutputTokens: 16384, SupportsTools: true, DefaultRunBudget: 2},
		{ID: "openai-native/gpt-5.5", Name: "GPT-5.5 (Reasoning)", Provider: "OpenAI", Tier: "Diamond", InputCostPerMTok: 5, OutputCostPerMTok: 30, CachedCostPerMTok: .5, ContextTokens: 1050000, MaxOutputTokens: 128000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 10},
		{ID: "openai-native/gpt-5.4", Name: "GPT-5.4 (Reasoning)", Provider: "OpenAI", Tier: "Diamond", InputCostPerMTok: 2.5, OutputCostPerMTok: 15, CachedCostPerMTok: .25, ContextTokens: 1050000, MaxOutputTokens: 128000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 6},
		{ID: "openai-native/gpt-5.4-mini", Name: "GPT-5.4 Mini (Reasoning)", Provider: "OpenAI", Tier: "Platinum", InputCostPerMTok: .75, OutputCostPerMTok: 4.5, CachedCostPerMTok: .075, ContextTokens: 400000, MaxOutputTokens: 128000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 3},
		{ID: "gemini/gemini-3.5-flash", Name: "Gemini 3.5 Flash", Provider: "Gemini", Tier: "Gold", InputCostPerMTok: 1.5, OutputCostPerMTok: 9, CachedCostPerMTok: .15, ContextTokens: 1000000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 3},
		{ID: "gemini/gemini-3.1-flash-lite", Name: "Gemini 3.1 Flash-Lite", Provider: "Gemini", Tier: "Gold", InputCostPerMTok: .25, OutputCostPerMTok: 1.5, CachedCostPerMTok: .025, ContextTokens: 1000000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, DefaultRunBudget: 1},
		{ID: "gemini/gemini-3.1-pro-preview", Name: "Gemini 3.1 Pro Preview", Provider: "Gemini", Tier: "Diamond", InputCostPerMTok: 2, OutputCostPerMTok: 12, CachedCostPerMTok: .2, ContextTokens: 1000000, MaxOutputTokens: 64000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 6},
		{ID: "claude/claude-fable-5", Name: "Claude Fable 5 (Reasoning)", Provider: "Claude", Tier: "Diamond", InputCostPerMTok: 10, OutputCostPerMTok: 50, CachedCostPerMTok: 1, ContextTokens: 200000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 10},
		{ID: "claude/claude-opus-4-8", Name: "Claude Opus 4.8", Provider: "Claude", Tier: "Diamond", InputCostPerMTok: 5, OutputCostPerMTok: 25, CachedCostPerMTok: .5, ContextTokens: 200000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 8},
		{ID: "claude/claude-sonnet-4-6", Name: "Claude Sonnet 4.6", Provider: "Claude", Tier: "Platinum", InputCostPerMTok: 3, OutputCostPerMTok: 15, CachedCostPerMTok: .3, ContextTokens: 200000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, DefaultRunBudget: 5},
		{ID: "claude/claude-haiku-4-5", Name: "Claude Haiku 4.5", Provider: "Claude", Tier: "Gold", InputCostPerMTok: 1, OutputCostPerMTok: 5, CachedCostPerMTok: .1, ContextTokens: 200000, MaxOutputTokens: 16000, SupportsTools: true, SupportsVision: true, DefaultRunBudget: 2},
	}
}

func (r *ProviderRegistry) Resolve(modelID string) (ModelSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if spec, ok := r.models[modelID]; ok {
		return spec, true
	}
	if strings.HasPrefix(modelID, "ollama/") && modelID != "ollama/local-fallback" {
		return ModelSpec{ID: modelID, Name: strings.TrimPrefix(modelID, "ollama/"), Provider: "Ollama", Tier: "Local", ContextTokens: 32768, MaxOutputTokens: 8192, DefaultRunBudget: 0}, true
	}
	return ModelSpec{}, false
}

func (r *ProviderRegistry) PublicModels() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ModelSpec, 0, len(r.models))
	for _, spec := range r.models {
		items = append(items, spec)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	models := make([]map[string]interface{}, 0, len(items))
	for _, spec := range items {
		models = append(models, map[string]interface{}{"id": spec.ID, "name": spec.Name, "provider": spec.Provider, "tier": spec.Tier, "cost_input_1m": spec.InputCostPerMTok, "cost_output_1m": spec.OutputCostPerMTok, "context_tokens": spec.ContextTokens, "max_output_tokens": spec.MaxOutputTokens, "supports_tools": spec.SupportsTools, "supports_vision": spec.SupportsVision, "supports_reasoning": spec.SupportsReasoning, "default_run_budget_usd": spec.DefaultRunBudget})
	}
	return models
}

func estimateChatTokens(systemPrompt string, rawMessages []json.RawMessage) int {
	bytes := len(systemPrompt)
	for _, raw := range rawMessages {
		bytes += len(raw)
	}
	return bytes/3 + 128 // conservative multilingual heuristic
}

func (r *ProviderRegistry) ValidateChat(modelID, systemPrompt string, messages []json.RawMessage, useTools, requiresVision bool) (ModelSpec, bool, error) {
	spec, ok := r.Resolve(modelID)
	if !ok {
		return ModelSpec{}, false, fmt.Errorf("model is not registered")
	}
	if estimateChatTokens(systemPrompt, messages) > spec.ContextTokens-spec.MaxOutputTokens {
		return ModelSpec{}, false, fmt.Errorf("conversation exceeds the safe context budget for this model")
	}
	if requiresVision && !spec.SupportsVision {
		return ModelSpec{}, false, fmt.Errorf("selected model does not support visual context")
	}
	return spec, useTools && spec.SupportsTools, nil
}

func providerConfigured(spec ModelSpec, app *AppState) bool {
	switch spec.Provider {
	case "Groq":
		return strings.HasPrefix(app.getAPIKey(), "gsk_")
	case "OpenAI":
		return strings.HasPrefix(app.getOpenAIKey(), "sk-")
	case "DeepSeek":
		return strings.HasPrefix(app.getDeepSeekKey(), "sk-")
	case "Gemini":
		return strings.HasPrefix(app.getGeminiKey(), "AIza")
	case "Claude":
		return strings.HasPrefix(app.getClaudeKey(), "sk-ant-")
	case "Ollama":
		return len(getLocalOllamaModels()) > 0
	default:
		return false
	}
}

// SelectAvailable preserves the requested model when possible. If its provider
// is unavailable it selects the least-cost configured model with the same
// required modalities; tool/vision requirements are never silently weakened.
func (r *ProviderRegistry) SelectAvailable(modelID string, app *AppState, needsTools, needsVision bool) (ModelSpec, bool) {
	requested, ok := r.Resolve(modelID)
	if !ok {
		return ModelSpec{}, false
	}
	if providerConfigured(requested, app) {
		return requested, false
	}
	r.mu.RLock()
	candidates := make([]ModelSpec, 0, len(r.models))
	for _, spec := range r.models {
		if needsTools && !spec.SupportsTools || needsVision && !spec.SupportsVision || !providerConfigured(spec, app) {
			continue
		}
		candidates = append(candidates, spec)
	}
	r.mu.RUnlock()
	if len(candidates) == 0 {
		return requested, false
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i].InputCostPerMTok + candidates[i].OutputCostPerMTok
		right := candidates[j].InputCostPerMTok + candidates[j].OutputCostPerMTok
		if left == right {
			return candidates[i].ID < candidates[j].ID
		}
		return left < right
	})
	return candidates[0], candidates[0].ID != modelID
}

type providerProbe struct {
	name       string
	url        string
	key        string
	configured bool
	claude     bool
}

func ProbeProviderHealth(ctx context.Context, app *AppState) []ProviderHealth {
	probes := []providerProbe{
		{name: "Groq", url: "https://api.groq.com/openai/v1/models", key: app.getAPIKey(), configured: strings.HasPrefix(app.getAPIKey(), "gsk_")},
		{name: "OpenAI", url: "https://api.openai.com/v1/models", key: app.getOpenAIKey(), configured: strings.HasPrefix(app.getOpenAIKey(), "sk-")},
		{name: "DeepSeek", url: "https://api.deepseek.com/models", key: app.getDeepSeekKey(), configured: strings.HasPrefix(app.getDeepSeekKey(), "sk-")},
		{name: "Gemini", url: "https://generativelanguage.googleapis.com/v1beta/openai/models", key: app.getGeminiKey(), configured: strings.HasPrefix(app.getGeminiKey(), "AIza")},
		{name: "Claude", url: "https://api.anthropic.com/v1/models", key: app.getClaudeKey(), configured: strings.HasPrefix(app.getClaudeKey(), "sk-ant-"), claude: true},
		{name: "Ollama", url: "http://localhost:11434/api/tags", configured: true},
	}
	results := make([]ProviderHealth, len(probes))
	var wg sync.WaitGroup
	for index, probe := range probes {
		index, probe := index, probe
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !probe.configured {
				results[index] = ProviderHealth{Provider: probe.name, Detail: "not configured"}
				return
			}
			probeCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, probe.url, nil)
			if err != nil {
				results[index] = ProviderHealth{Provider: probe.name, Configured: true, Detail: "probe creation failed"}
				return
			}
			if probe.key != "" {
				req.Header.Set("Authorization", "Bearer "+probe.key)
			}
			if probe.claude {
				req.Header.Del("Authorization")
				req.Header.Set("x-api-key", probe.key)
				req.Header.Set("anthropic-version", "2023-06-01")
			}
			resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
			if err != nil {
				results[index] = ProviderHealth{Provider: probe.name, Configured: true, Detail: "unreachable"}
				return
			}
			resp.Body.Close()
			detail := fmt.Sprintf("HTTP %d", resp.StatusCode)
			results[index] = ProviderHealth{Provider: probe.name, Configured: true, Reachable: resp.StatusCode < 500, Detail: detail}
		}()
	}
	wg.Wait()
	return results
}
