package main

// STATUS: DIAMANT VGT SUPREME

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed frontend/modules/i18n.js
var i18nRuntime []byte

//go:embed frontend/modules/ui.js
var localizedUIRuntime []byte

func TestFrontendLanguageContractCoversRequiredLocales(t *testing.T) {
	runtime := string(i18nRuntime)
	for _, locale := range []string{"de", "en", "ru", "es"} {
		if !strings.Contains(runtime, locale+": Object.freeze({") {
			t.Fatalf("required UI locale %q missing", locale)
		}
		if !strings.Contains(string(agentBuilderHTML), `<option value="`+locale+`">`) {
			t.Fatalf("language selector option %q missing", locale)
		}
	}
	for _, token := range []string{"data-i18n", "aethel-language-select", "aethel:language-changed", "applyTranslations"} {
		if !strings.Contains(runtime+string(agentBuilderHTML), token) {
			t.Fatalf("language runtime linkage missing %q", token)
		}
	}
	if !strings.Contains(string(localizedUIRuntime), "OPERATOR LANGUAGE CONTRACT") || !strings.Contains(string(localizedUIRuntime), "currentLanguage()") {
		t.Fatal("selected UI language is not propagated into the AI response contract")
	}
}
