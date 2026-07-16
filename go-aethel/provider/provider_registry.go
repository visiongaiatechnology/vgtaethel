package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ModelSpec is the authoritative model contract shared by UI, chat, cost
// tracking and agent runs. Unregistered models are never given tool access.

type KeyProvider interface {
	GetAPIKey() string
	GetOpenAIKey() string
	GetDeepSeekKey() string
	GetGeminiKey() string
	GetClaudeKey() string
}

type ModelSpec struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Provider          string   `json:"provider"`
	Tier              string   `json:"tier"`
	InputCostPerMTok  float64  `json:"cost_input_1m"`
	OutputCostPerMTok float64  `json:"cost_output_1m"`
	CachedCostPerMTok float64  `json:"cost_cached_input_1m,omitempty"`
	ContextTokens     int      `json:"context_tokens"`
	MaxOutputTokens   int      `json:"max_output_tokens"`
	SupportsTools     bool     `json:"supports_tools"`
	SupportsVision    bool     `json:"supports_vision"`
	SupportsReasoning bool     `json:"supports_reasoning"`
	ReasoningOptions  []string `json:"reasoning_options,omitempty"`
	DefaultReasoning  string   `json:"default_reasoning,omitempty"`
	Discovered        bool     `json:"discovered,omitempty"`
	DefaultRunBudget  float64  `json:"default_run_budget_usd"`
}

type ProviderHealth struct {
	Provider   string `json:"provider"`
	Configured bool   `json:"configured"`
	Reachable  bool   `json:"reachable"`
	Detail     string `json:"detail"`
}

type ProviderRegistry struct {
	mu               sync.RWMutex
	models           map[string]ModelSpec
	lastDiscovery    time.Time
	discoveryRunning bool
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
		{ID: "deepseek/deepseek-v4-flash", Name: "DeepSeek V4 Flash (Thinking)", Provider: "DeepSeek", Tier: "Diamond", InputCostPerMTok: .14, OutputCostPerMTok: .28, CachedCostPerMTok: .0028, ContextTokens: 1000000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"none", "high", "max"}, DefaultReasoning: "high", DefaultRunBudget: 2},
		{ID: "deepseek/deepseek-v4-pro", Name: "DeepSeek V4 Pro (Thinking)", Provider: "DeepSeek", Tier: "Diamond", InputCostPerMTok: .435, OutputCostPerMTok: .87, CachedCostPerMTok: .003625, ContextTokens: 1000000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"none", "high", "max"}, DefaultReasoning: "high", DefaultRunBudget: 4},
		{ID: "openai/gpt-oss-120b", Name: "GPT OSS 120B (Logic Core)", Provider: "Groq", Tier: "Diamond", InputCostPerMTok: .15, OutputCostPerMTok: .60, ContextTokens: 131072, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"low", "medium", "high"}, DefaultReasoning: "medium", DefaultRunBudget: 2},
		{ID: "openai/gpt-oss-20b", Name: "GPT OSS 20B (Safety Core)", Provider: "Groq", Tier: "Platinum", InputCostPerMTok: .075, OutputCostPerMTok: .30, ContextTokens: 131072, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"low", "medium", "high"}, DefaultReasoning: "low", DefaultRunBudget: 1},
		{ID: "qwen/qwen3.6-27b", Name: "Qwen 3.6 27B", Provider: "Groq", Tier: "Platinum", InputCostPerMTok: .08, OutputCostPerMTok: .12, ContextTokens: 131072, MaxOutputTokens: 16384, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"none", "default"}, DefaultReasoning: "none", DefaultRunBudget: 1},
		{ID: "openai-native/gpt-5.6-sol", Name: "GPT-5.6 Sol", Provider: "OpenAI", Tier: "Diamond", InputCostPerMTok: 5, OutputCostPerMTok: 30, CachedCostPerMTok: .5, ContextTokens: 1050000, MaxOutputTokens: 128000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"none", "low", "medium", "high", "xhigh", "max"}, DefaultReasoning: "medium", DefaultRunBudget: 10},
		{ID: "openai-native/gpt-5.6-terra", Name: "GPT-5.6 Terra", Provider: "OpenAI", Tier: "Diamond", InputCostPerMTok: 2.5, OutputCostPerMTok: 15, CachedCostPerMTok: .25, ContextTokens: 1050000, MaxOutputTokens: 128000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"none", "low", "medium", "high", "xhigh", "max"}, DefaultReasoning: "medium", DefaultRunBudget: 6},
		{ID: "openai-native/gpt-5.6-luna", Name: "GPT-5.6 Luna", Provider: "OpenAI", Tier: "Platinum", InputCostPerMTok: 1, OutputCostPerMTok: 6, CachedCostPerMTok: .1, ContextTokens: 1050000, MaxOutputTokens: 128000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"none", "low", "medium", "high", "xhigh", "max"}, DefaultReasoning: "low", DefaultRunBudget: 3},
		{ID: "gemini/gemini-3.5-flash", Name: "Gemini 3.5 Flash", Provider: "Gemini", Tier: "Gold", InputCostPerMTok: 1.5, OutputCostPerMTok: 9, CachedCostPerMTok: .15, ContextTokens: 1000000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"minimal", "low", "medium", "high"}, DefaultReasoning: "medium", DefaultRunBudget: 3},
		{ID: "gemini/gemini-3.1-flash-lite", Name: "Gemini 3.1 Flash-Lite", Provider: "Gemini", Tier: "Gold", InputCostPerMTok: .25, OutputCostPerMTok: 1.5, CachedCostPerMTok: .025, ContextTokens: 1000000, MaxOutputTokens: 32000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"minimal", "low", "medium", "high"}, DefaultReasoning: "low", DefaultRunBudget: 1},
		{ID: "gemini/gemini-3.1-pro-preview", Name: "Gemini 3.1 Pro Preview", Provider: "Gemini", Tier: "Diamond", InputCostPerMTok: 2, OutputCostPerMTok: 12, CachedCostPerMTok: .2, ContextTokens: 1000000, MaxOutputTokens: 64000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"minimal", "low", "medium", "high"}, DefaultReasoning: "high", DefaultRunBudget: 6},
		{ID: "claude/claude-fable-5", Name: "Claude Fable 5 (Reasoning)", Provider: "Claude", Tier: "Diamond", InputCostPerMTok: 10, OutputCostPerMTok: 50, CachedCostPerMTok: 1, ContextTokens: 200000, MaxOutputTokens: 64000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"low", "medium", "high", "xhigh", "max"}, DefaultReasoning: "high", DefaultRunBudget: 10},
		{ID: "claude/claude-opus-4-8", Name: "Claude Opus 4.8", Provider: "Claude", Tier: "Diamond", InputCostPerMTok: 5, OutputCostPerMTok: 25, CachedCostPerMTok: .5, ContextTokens: 200000, MaxOutputTokens: 64000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"low", "medium", "high", "xhigh", "max"}, DefaultReasoning: "high", DefaultRunBudget: 8},
		{ID: "claude/claude-sonnet-4-6", Name: "Claude Sonnet 4.6", Provider: "Claude", Tier: "Platinum", InputCostPerMTok: 3, OutputCostPerMTok: 15, CachedCostPerMTok: .3, ContextTokens: 200000, MaxOutputTokens: 64000, SupportsTools: true, SupportsVision: true, SupportsReasoning: true, ReasoningOptions: []string{"low", "medium", "high", "max"}, DefaultReasoning: "medium", DefaultRunBudget: 5},
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
	return publicModelPayload(r.sortedModelsLocked())
}

// AvailableModels returns only providers the operator explicitly configured.
// Local Ollama models are appended by the HTTP layer after a live local probe.
func (r *ProviderRegistry) AvailableModels(app KeyProvider) []map[string]interface{} {
	if r == nil || app == nil {
		return []map[string]interface{}{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.sortedModelsLocked()
	available := make([]ModelSpec, 0, len(items))
	for _, spec := range items {
		if spec.Provider != "Ollama" && providerConfigured(spec, app) {
			available = append(available, spec)
		}
	}
	return publicModelPayload(available)
}

func (r *ProviderRegistry) sortedModelsLocked() []ModelSpec {
	items := make([]ModelSpec, 0, len(r.models))
	for _, spec := range r.models {
		items = append(items, spec)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items
}

func publicModelPayload(items []ModelSpec) []map[string]interface{} {
	models := make([]map[string]interface{}, 0, len(items))
	for _, spec := range items {
		models = append(models, map[string]interface{}{"id": spec.ID, "name": spec.Name, "provider": spec.Provider, "tier": spec.Tier, "cost_input_1m": spec.InputCostPerMTok, "cost_output_1m": spec.OutputCostPerMTok, "context_tokens": spec.ContextTokens, "max_output_tokens": spec.MaxOutputTokens, "supports_tools": spec.SupportsTools, "supports_vision": spec.SupportsVision, "supports_reasoning": spec.SupportsReasoning, "reasoning_options": append([]string(nil), spec.ReasoningOptions...), "default_reasoning": spec.DefaultReasoning, "discovered": spec.Discovered, "default_run_budget_usd": spec.DefaultRunBudget})
	}
	return models
}

type discoveredModel struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	DisplayName         string   `json:"displayName"`
	ContextWindow       int      `json:"context_window"`
	MaxCompletionTokens int      `json:"max_completion_tokens"`
	InputTokenLimit     int      `json:"inputTokenLimit"`
	OutputTokenLimit    int      `json:"outputTokenLimit"`
	SupportedMethods    []string `json:"supportedGenerationMethods"`
}

type modelListEnvelope struct {
	Data   []discoveredModel `json:"data"`
	Models []discoveredModel `json:"models"`
}

type discoveryTarget struct {
	provider       string
	url            string
	key            string
	prefix         string
	defaultContext int
	defaultOutput  int
	gemini         bool
	claude         bool
}

// RefreshAvailableModels augments the verified registry with live provider
// catalogs. Unknown live models are deliberately chat-only until their tool
// contract and pricing are verified; discovery can therefore never expand
// execution privileges implicitly.
func (r *ProviderRegistry) RefreshAvailableModels(ctx context.Context, app KeyProvider) {
	if r == nil || app == nil {
		return
	}
	r.mu.Lock()
	fresh := !r.lastDiscovery.IsZero() && time.Since(r.lastDiscovery) < 10*time.Minute
	if fresh || r.discoveryRunning {
		r.mu.Unlock()
		return
	}
	r.discoveryRunning = true
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.discoveryRunning = false
		r.mu.Unlock()
	}()

	targets := []discoveryTarget{
		{provider: "Groq", url: "https://api.groq.com/openai/v1/models", key: app.GetAPIKey(), defaultContext: 131072, defaultOutput: 16384},
		{provider: "OpenAI", url: "https://api.openai.com/v1/models", key: app.GetOpenAIKey(), prefix: "openai-native/", defaultContext: 128000, defaultOutput: 16384},
		{provider: "DeepSeek", url: "https://api.deepseek.com/models", key: app.GetDeepSeekKey(), prefix: "deepseek/", defaultContext: 128000, defaultOutput: 8192},
		{provider: "Gemini", url: "https://generativelanguage.googleapis.com/v1beta/models", key: app.GetGeminiKey(), prefix: "gemini/", defaultContext: 1000000, defaultOutput: 32768, gemini: true},
		{provider: "Claude", url: "https://api.anthropic.com/v1/models?limit=100", key: app.GetClaudeKey(), prefix: "claude/", defaultContext: 200000, defaultOutput: 16384, claude: true},
	}

	results := make(chan []ModelSpec, len(targets))
	var wg sync.WaitGroup
	attempted := 0
	for _, target := range targets {
		if !validDiscoveryKey(target.provider, target.key) {
			continue
		}
		attempted++
		wg.Add(1)
		go func(target discoveryTarget) {
			defer wg.Done()
			results <- discoverProviderModels(ctx, target)
		}(target)
	}
	if attempted == 0 {
		return
	}
	wg.Wait()
	close(results)

	r.mu.Lock()
	defer r.mu.Unlock()
	for specs := range results {
		for _, spec := range specs {
			if _, verified := r.models[spec.ID]; !verified {
				r.models[spec.ID] = spec
			}
		}
	}
	r.lastDiscovery = time.Now()
}

func validDiscoveryKey(providerName, key string) bool {
	return IsConfiguredProviderKey(providerName, key)
}

// IsConfiguredProviderKey is the single provider credential-format gate used
// by setup, health, model visibility and discovery. Keeping this decision in
// one place prevents an encrypted or malformed value from looking "ready" in
// one UI surface while being rejected by another.
func IsConfiguredProviderKey(providerName, key string) bool {
	key = strings.TrimSpace(key)
	switch providerName {
	case "Groq":
		return strings.HasPrefix(key, "gsk_")
	case "OpenAI", "DeepSeek":
		return strings.HasPrefix(key, "sk-")
	case "Gemini":
		return strings.HasPrefix(key, "AIza")
	case "Claude":
		return strings.HasPrefix(key, "sk-ant-")
	default:
		return false
	}
}

func HasConfiguredProvider(app KeyProvider) bool {
	if app == nil {
		return false
	}
	return IsConfiguredProviderKey("Groq", app.GetAPIKey()) ||
		IsConfiguredProviderKey("OpenAI", app.GetOpenAIKey()) ||
		IsConfiguredProviderKey("DeepSeek", app.GetDeepSeekKey()) ||
		IsConfiguredProviderKey("Gemini", app.GetGeminiKey()) ||
		IsConfiguredProviderKey("Claude", app.GetClaudeKey())
}

func discoverProviderModels(ctx context.Context, target discoveryTarget) []ModelSpec {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.url, nil)
	if err != nil {
		return nil
	}
	if target.gemini {
		req.Header.Set("x-goog-api-key", target.key)
	} else if target.claude {
		req.Header.Set("x-api-key", target.key)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+target.key)
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil
	}
	var envelope modelListEnvelope
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4<<20))
	if err := decoder.Decode(&envelope); err != nil {
		return nil
	}
	items := envelope.Data
	if len(items) == 0 {
		items = envelope.Models
	}
	models := make([]ModelSpec, 0, len(items))
	for _, item := range items {
		rawID := strings.TrimSpace(strings.TrimPrefix(item.ID, "models/"))
		if rawID == "" {
			rawID = strings.TrimSpace(strings.TrimPrefix(item.Name, "models/"))
		}
		if !isChatModelID(target.provider, rawID, item.SupportedMethods) {
			continue
		}
		contextTokens := firstPositive(item.ContextWindow, item.InputTokenLimit, target.defaultContext)
		outputTokens := firstPositive(item.MaxCompletionTokens, item.OutputTokenLimit, target.defaultOutput)
		name := strings.TrimSpace(item.DisplayName)
		if name == "" {
			name = rawID
		}
		reasoningOptions, defaultReasoning := discoveredReasoningContract(target.provider, rawID)
		models = append(models, ModelSpec{
			ID: target.prefix + rawID, Name: name, Provider: target.provider,
			Tier: "Live", ContextTokens: contextTokens, MaxOutputTokens: outputTokens,
			SupportsTools: false, SupportsVision: false, SupportsReasoning: len(reasoningOptions) > 0,
			ReasoningOptions: reasoningOptions, DefaultReasoning: defaultReasoning, Discovered: true, DefaultRunBudget: 1,
		})
	}
	return models
}

func discoveredReasoningContract(providerName, modelID string) ([]string, string) {
	id := strings.ToLower(modelID)
	if providerName == "Groq" && id == "qwen/qwen3-32b" {
		return []string{"none", "default"}, "default"
	}
	return nil, ""
}

func isChatModelID(providerName, id string, methods []string) bool {
	lower := strings.ToLower(id)
	if lower == "" || containsAnyModelFragment(lower, "embed", "moderation", "whisper", "tts", "audio", "image", "dall-e", "realtime", "transcrib", "prompt-guard", "safeguard", "compound") {
		return false
	}
	if providerName == "Gemini" {
		for _, method := range methods {
			if method == "generateContent" {
				return strings.Contains(lower, "gemini")
			}
		}
		return false
	}
	switch providerName {
	case "OpenAI":
		return strings.HasPrefix(lower, "gpt-") || strings.HasPrefix(lower, "o1") || strings.HasPrefix(lower, "o3") || strings.HasPrefix(lower, "o4")
	case "Claude":
		return strings.HasPrefix(lower, "claude-")
	default:
		return true
	}
}

func containsAnyModelFragment(value string, fragments ...string) bool {
	for _, fragment := range fragments {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 1
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

func providerConfigured(spec ModelSpec, app KeyProvider) bool {
	if app == nil {
		return false
	}
	switch spec.Provider {
	case "Groq":
		return IsConfiguredProviderKey("Groq", app.GetAPIKey())
	case "OpenAI":
		return IsConfiguredProviderKey("OpenAI", app.GetOpenAIKey())
	case "DeepSeek":
		return IsConfiguredProviderKey("DeepSeek", app.GetDeepSeekKey())
	case "Gemini":
		return IsConfiguredProviderKey("Gemini", app.GetGeminiKey())
	case "Claude":
		return IsConfiguredProviderKey("Claude", app.GetClaudeKey())
	case "Ollama":
		return len(GetLocalOllamaModels()) > 0
	default:
		return false
	}
}

// SelectAvailable preserves the requested model when possible. If its provider
// is unavailable it selects the least-cost configured model with the same
// required modalities; tool/vision requirements are never silently weakened.
func (r *ProviderRegistry) SelectAvailable(modelID string, app KeyProvider, needsTools, needsVision bool) (ModelSpec, bool) {
	requested, ok := r.Resolve(modelID)
	if !ok {
		return ModelSpec{}, false
	}
	if providerConfigured(requested, app) && (!needsTools || requested.SupportsTools) && (!needsVision || requested.SupportsVision) {
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
	// Prioritize openai/gpt-oss-120b as the default fallback for DeepSeek models
	if strings.HasPrefix(modelID, "deepseek/") {
		for _, c := range candidates {
			if c.ID == "openai/gpt-oss-120b" {
				return c, true
			}
		}
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

func ProbeProviderHealth(ctx context.Context, app KeyProvider) []ProviderHealth {
	probes := []providerProbe{
		{name: "Groq", url: "https://api.groq.com/openai/v1/models", key: app.GetAPIKey(), configured: IsConfiguredProviderKey("Groq", app.GetAPIKey())},
		{name: "OpenAI", url: "https://api.openai.com/v1/models", key: app.GetOpenAIKey(), configured: IsConfiguredProviderKey("OpenAI", app.GetOpenAIKey())},
		{name: "DeepSeek", url: "https://api.deepseek.com/models", key: app.GetDeepSeekKey(), configured: IsConfiguredProviderKey("DeepSeek", app.GetDeepSeekKey())},
		{name: "Gemini", url: "https://generativelanguage.googleapis.com/v1beta/openai/models", key: app.GetGeminiKey(), configured: IsConfiguredProviderKey("Gemini", app.GetGeminiKey())},
		{name: "Claude", url: "https://api.anthropic.com/v1/models", key: app.GetClaudeKey(), configured: IsConfiguredProviderKey("Claude", app.GetClaudeKey()), claude: true},
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

func CalculateInferenceCost(modelID string, inputTokens, outputTokens, cachedTokens int) float64 {
	registry := NewProviderRegistry()
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
	case "gpt-5.6-sol":
		inputPrice = 5.00
		cacheHitPrice = 0.50
		outputPrice = 30.00
	case "gpt-5.6-terra":
		inputPrice = 2.50
		cacheHitPrice = 0.25
		outputPrice = 15.00
	case "gpt-5.6-luna":
		inputPrice = 1.00
		cacheHitPrice = 0.10
		outputPrice = 6.00
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
