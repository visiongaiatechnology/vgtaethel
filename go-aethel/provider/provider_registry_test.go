package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLiveDiscoveryIsChatOnlyAndFiltersNonChatModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-future","context_window":262144,"max_completion_tokens":32768},{"id":"text-embedding-future"}]}`))
	}))
	defer server.Close()

	models := discoverProviderModels(context.Background(), discoveryTarget{
		provider: "OpenAI", url: server.URL, prefix: "openai-native/", defaultContext: 128000, defaultOutput: 8192,
	})
	if len(models) != 1 {
		t.Fatalf("expected one chat model, got %+v", models)
	}
	model := models[0]
	if model.ID != "openai-native/gpt-future" || model.SupportsTools || !model.Discovered || model.ContextTokens != 262144 {
		t.Fatalf("unsafe or incorrect discovery contract: %+v", model)
	}
}

func TestProviderRegistryEnforcesCapabilitiesAndContext(t *testing.T) {
	registry := NewProviderRegistry()
	spec, tools, err := registry.ValidateChat("openai-native/gpt-5.6-sol", "system", []json.RawMessage{json.RawMessage(`{"role":"user","content":"hello"}`)}, true, true)
	if err != nil || !tools || spec.ContextTokens != 1050000 || spec.MaxOutputTokens != 128000 {
		t.Fatalf("official OpenAI model contract mismatch: spec=%+v tools=%t err=%v", spec, tools, err)
	}
	_, tools, err = registry.ValidateChat("ollama/local-model", "system", nil, true, false)
	if err != nil || tools {
		t.Fatalf("unverified Ollama model received tools: tools=%t err=%v", tools, err)
	}
	if _, _, err := registry.ValidateChat("unregistered/model", "system", nil, false, false); err == nil {
		t.Fatal("unregistered model was accepted")
	}
	oversized := json.RawMessage(`{"role":"user","content":"` + strings.Repeat("x", 700000) + `"}`)
	if _, _, err := registry.ValidateChat("openai/gpt-oss-120b", "system", []json.RawMessage{oversized}, false, false); err == nil {
		t.Fatal("context budget overflow was accepted")
	}
}

func TestRegistryBackedCostCalculation(t *testing.T) {
	cost := CalculateInferenceCost("openai-native/gpt-5.6-sol", 1000000, 1000000, 0)
	if cost != 35 {
		t.Fatalf("unexpected GPT-5.6 Sol cost: %.4f", cost)
	}
}

func TestProviderFallbackPreservesRequiredCapabilities(t *testing.T) {
	registry := NewProviderRegistry()
	app := &mockKeyProvider{}
	selected, changed := registry.SelectAvailable("openai-native/gpt-5.6-sol", app, true, true)
	if !changed || selected.Provider != "Groq" || !selected.SupportsTools || !selected.SupportsVision {
		t.Fatalf("capability-preserving fallback failed: selected=%+v changed=%t", selected, changed)
	}
	withoutProviders := &mockNoKeyProvider{}
	requested, changed := registry.SelectAvailable("openai-native/gpt-5.6-sol", withoutProviders, true, true)
	if changed || requested.ID != "openai-native/gpt-5.6-sol" {
		t.Fatalf("fallback fabricated an unavailable provider: %+v changed=%t", requested, changed)
	}
}

func TestAvailableModelsOnlyExposeConfiguredProviders(t *testing.T) {
	registry := NewProviderRegistry()
	models := registry.AvailableModels(&mockKeyProvider{})
	if len(models) == 0 {
		t.Fatal("configured Groq provider returned no models")
	}
	for _, model := range models {
		if model["provider"] != "Groq" {
			t.Fatalf("unconfigured provider leaked into available models: %+v", model)
		}
	}
	if models := registry.AvailableModels(&mockNoKeyProvider{}); len(models) != 0 {
		t.Fatalf("models exposed without any configured provider: %+v", models)
	}
}

func TestProviderConfigurationGateRejectsOpaqueAndMalformedValues(t *testing.T) {
	if IsConfiguredProviderKey("Groq", "AQIDBAUGBwgJCgsMDQ4P") {
		t.Fatal("opaque encrypted value was accepted as a configured Groq key")
	}
	if IsConfiguredProviderKey("Groq", "not-a-key") {
		t.Fatal("malformed value was accepted as a configured Groq key")
	}
	if !IsConfiguredProviderKey("Groq", "  gsk_valid_test_value  ") {
		t.Fatal("trimmed Groq key was not recognized consistently")
	}
}
