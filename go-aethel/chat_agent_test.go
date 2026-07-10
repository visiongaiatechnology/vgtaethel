package main

import (
	"encoding/json"
	"testing"
)

func TestParseAgentSSEReassemblesToolCallsAndUsage(t *testing.T) {
	stream := "data: Analyse[VGT_NL]läuft\n\n" +
		"data: [[THINKING]]:Prüfe[VGT_NL]Plan\n\n" +
		"data: [[TOOL_DELTA]]:[{\"index\":0,\"id\":\"call_1\",\"function\":{\"name\":\"fs_read_file\",\"arguments\":\"{\\\"path\\\":\"}}]\n\n" +
		"data: [[TOOL_DELTA]]:[{\"index\":0,\"function\":{\"arguments\":\"\\\"notes.txt\\\"}\"}}]\n\n" +
		"data: [[USAGE]]:{\"prompt_tokens\":120,\"completion_tokens\":30,\"prompt_cache_hit_tokens\":20}\n\n" +
		"data: [[TOOL_COMMIT]]\n\n"
	result := parseAgentSSE(stream)
	if result.Err != nil {
		t.Fatalf("parse failed: %v", result.Err)
	}
	if result.Text != "Analyse\nläuft" || result.Thinking != "Prüfe\nPlan" {
		t.Fatalf("stream text mismatch: text=%q thinking=%q", result.Text, result.Thinking)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "fs_read_file" || string(result.ToolCalls[0].Arguments) != `{"path":"notes.txt"}` {
		t.Fatalf("tool reconstruction mismatch: %+v", result.ToolCalls)
	}
	if result.InputTokens != 120 || result.OutputTokens != 30 || result.CachedTokens != 20 {
		t.Fatalf("usage mismatch: %+v", result)
	}
}

func TestParseAgentSSERejectsMalformedToolArguments(t *testing.T) {
	result := parseAgentSSE("data: [[TOOL_DELTA]]:[{\"index\":0,\"id\":\"call_1\",\"function\":{\"name\":\"fs_read_file\",\"arguments\":\"{bad\"}}]\n\n")
	if result.Err == nil {
		t.Fatal("malformed streamed tool arguments were accepted")
	}
}

func TestParseAgentSSEPreservesTokenWhitespace(t *testing.T) {
	result := parseAgentSSE("data: Ich\n\ndata:  bin\n\ndata:  bereit.\n\n")
	if result.Err != nil || result.Text != "Ich bin bereit." {
		t.Fatalf("token whitespace was lost: %q err=%v", result.Text, result.Err)
	}
}

func TestShouldEnableAgentToolsOnlyForOperationalRequests(t *testing.T) {
	tests := []struct {
		objective string
		want      bool
	}{
		{"Was geht ab :D", false},
		{"Erkläre mir kurz Go Interfaces", false},
		{"Schau dir den Ordner C:\\Projects an", true},
		{"Erstelle einen Bericht über diese Datei", true},
	}
	for _, test := range tests {
		if got := shouldEnableAgentTools(test.objective); got != test.want {
			t.Errorf("shouldEnableAgentTools(%q) = %t, want %t", test.objective, got, test.want)
		}
	}
}

func TestPromoteCodeCartographyFallbackAcceptsOnlyExplicitCartographyJSON(t *testing.T) {
	result := agentInferenceResult{Text: "{\"path\": \"C:\\Users\\Masterboard\\Project\", \"max_files\": 100, \"output_path\": \"code_cartography.md\"}\n\n{\"path\": \"code_cartography.md\", \"content\": \"must not be promoted\"}\n\n{\"path\": \"C:\\Users\\Masterboard\\Project\", \"output_path\": \"code_cartography.md\", \"recursive\": true}"}
	promoteCodeCartographyFallback(&result, "Erstelle eine Code Kartografie für diesen Ordner")
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "code_cartography" || result.Text != "" {
		t.Fatalf("cartography fallback did not promote the dedicated call: %+v", result)
	}
	var args CodeCartographyArgs
	if err := json.Unmarshal(result.ToolCalls[0].Arguments, &args); err != nil || args.Path != `C:\Users\Masterboard\Project` || args.MaxFiles != 100 {
		t.Fatalf("cartography fallback arguments invalid: %+v err=%v", args, err)
	}

	nonCartography := agentInferenceResult{Text: `{"path":"C:\Users\Masterboard\Project"}`}
	promoteCodeCartographyFallback(&nonCartography, "Schau dir den Ordner an")
	if len(nonCartography.ToolCalls) != 0 {
		t.Fatal("fallback promoted unrelated assistant JSON")
	}
}
