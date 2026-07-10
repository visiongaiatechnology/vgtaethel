package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestProbeProviderHealthRunsSuccessfully(t *testing.T) {
	app := &AppState{
		apiKey:       "gsk_dummy_key_for_test",
		openaiAPIKey: "sk-dummy_key_for_test",
	}

	results := ProbeProviderHealth(context.Background(), app)
	if len(results) == 0 {
		t.Fatal("health probe returned empty results")
	}

	// Verify that dummy configuration is checked
	foundGroq := false
	for _, res := range results {
		if res.Provider == "Groq" {
			foundGroq = true
			if !res.Configured {
				t.Error("Groq should be configured with gsk_ prefix")
			}
		}
	}
	if !foundGroq {
		t.Error("Groq provider health was not probed")
	}
}

func TestLiveGroqAdapterIntegration(t *testing.T) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" || !strings.HasPrefix(apiKey, "gsk_") {
		t.Skip("Skipping live Groq integration test: GROQ_API_KEY is not configured in the environment")
	}

	app := &AppState{
		apiKey: apiKey,
	}

	results := ProbeProviderHealth(context.Background(), app)
	var groqHealth *ProviderHealth
	for i := range results {
		if results[i].Provider == "Groq" {
			groqHealth = &results[i]
		}
	}

	if groqHealth == nil {
		t.Fatal("Groq provider not found in health results")
	}
	if !groqHealth.Configured {
		t.Error("Groq should be reported as configured")
	}
	if !groqHealth.Reachable {
		t.Errorf("Groq is reported as unreachable: %s", groqHealth.Detail)
	}
}
