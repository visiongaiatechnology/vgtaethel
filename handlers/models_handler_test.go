package handlers

// STATUS: DIAMANT VGT SUPREME

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-aethel/provider"
)

func TestModelsHandlerHidesProvidersWithoutConfiguredKeys(t *testing.T) {
	previous := state
	state = &appState{providers: provider.NewProviderRegistry()}
	t.Cleanup(func() { state = previous })

	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	response := httptest.NewRecorder()
	handleModels(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("models endpoint status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Models []map[string]any `json:"models"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode models endpoint: %v", err)
	}
	for index, model := range payload.Models {
		if model["id"] == nil || model["name"] == nil || model["provider"] == nil {
			t.Fatalf("model %d cannot populate UI: %+v", index, model)
		}
		if model["provider"] != "Ollama" {
			t.Fatalf("unconfigured remote provider leaked into model picker: %+v", model)
		}
	}
}

func TestModelsHandlerImmediatelyReturnsConfiguredGroqModels(t *testing.T) {
	previous := state
	state = &appState{
		providers: provider.NewProviderRegistry(),
		getAPIKey: func() string { return "gsk_test_configured" },
	}
	t.Cleanup(func() { state = previous })

	started := time.Now()
	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	response := httptest.NewRecorder()
	handleModels(response, request)
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("configured model registry waited on live discovery: %s", elapsed)
	}

	var payload struct {
		Models []map[string]any `json:"models"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode configured models: %v", err)
	}
	if len(payload.Models) == 0 {
		t.Fatal("configured Groq key produced an empty model picker")
	}
	for _, model := range payload.Models {
		if model["provider"] != "Groq" && model["provider"] != "Ollama" {
			t.Fatalf("unconfigured provider leaked into picker: %+v", model)
		}
	}
}
