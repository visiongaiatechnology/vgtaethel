package main

import (
	"testing"

	"go-aethel/provider"
)

func TestDirectWailsModelRegistryReturnsConfiguredGroqModels(t *testing.T) {
	previous := state
	state = &AppState{
		apiKey:    "gsk_configured_test_value",
		providers: provider.NewProviderRegistry(),
	}
	t.Cleanup(func() { state = previous })

	models := (&App{}).GetAvailableModels()
	if len(models) == 0 {
		t.Fatal("direct Wails registry returned no configured Groq models")
	}
	for _, model := range models {
		if model["provider"] != "Groq" && model["provider"] != "Ollama" {
			t.Fatalf("direct registry exposed an unconfigured provider: %+v", model)
		}
	}
}
