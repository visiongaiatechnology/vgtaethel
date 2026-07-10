package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProviderRegistryEnforcesCapabilitiesAndContext(t *testing.T) {
	registry := NewProviderRegistry()
	spec, tools, err := registry.ValidateChat("openai-native/gpt-5.5", "system", []json.RawMessage{json.RawMessage(`{"role":"user","content":"hello"}`)}, true, true)
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
	if _, _, err := registry.ValidateChat("llama-3.3-70b-versatile", "system", []json.RawMessage{oversized}, false, false); err == nil {
		t.Fatal("context budget overflow was accepted")
	}
}

func TestRegistryBackedCostCalculation(t *testing.T) {
	cost := CalculateInferenceCost("openai-native/gpt-5.5", 1000000, 1000000, 0)
	if cost != 35 {
		t.Fatalf("unexpected GPT-5.5 cost: %.4f", cost)
	}
}

func TestProviderFallbackPreservesRequiredCapabilities(t *testing.T) {
	registry := NewProviderRegistry()
	app := &AppState{apiKey: "gsk_test_configured"}
	selected, changed := registry.SelectAvailable("openai-native/gpt-5.5", app, true, true)
	if !changed || selected.Provider != "Groq" || !selected.SupportsTools || !selected.SupportsVision {
		t.Fatalf("capability-preserving fallback failed: selected=%+v changed=%t", selected, changed)
	}
	withoutProviders := &AppState{}
	requested, changed := registry.SelectAvailable("openai-native/gpt-5.5", withoutProviders, true, true)
	if changed || requested.ID != "openai-native/gpt-5.5" {
		t.Fatalf("fallback fabricated an unavailable provider: %+v changed=%t", requested, changed)
	}
}
