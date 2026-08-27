package handlers

// STATUS: DIAMANT VGT SUPREME

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetupMergesNewProviderKeyWithExistingSealedKeys(t *testing.T) {
	previous := state
	var savedGroq, savedDeepSeek string
	state = &appState{
		getAPIKey:      func() string { return "" },
		getOpenAIKey:   func() string { return "" },
		getDeepSeekKey: func() string { return "sk-existing-deepseek" },
		getGeminiKey:   func() string { return "" },
		getClaudeKey:   func() string { return "" },
		saveConfig: func(groq, _ string, deepseek string, _, _ string) error {
			savedGroq, savedDeepSeek = groq, deepseek
			return nil
		},
	}
	t.Cleanup(func() { state = previous })

	req := httptest.NewRequest(http.MethodPost, "/v1/setup", bytes.NewBufferString(`{"api_key":"gsk_new-groq","deepseek_api_key":""}`))
	rec := httptest.NewRecorder()
	handleSetup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if savedGroq != "gsk_new-groq" || savedDeepSeek != "sk-existing-deepseek" {
		t.Fatalf("provider merge lost a key: groq=%q deepseek=%q", savedGroq, savedDeepSeek)
	}
}
