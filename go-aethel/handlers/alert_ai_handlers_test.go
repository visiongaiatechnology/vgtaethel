package handlers

// STATUS: DIAMANT VGT SUPREME

import (
	"strings"
	"testing"
)

func TestParseInternalSSESeparatesModelAssessmentFromMetadata(t *testing.T) {
	input := strings.Join([]string{
		"data: [[THINKING]]:private",
		"",
		"data: {\"severity\":\"high\",\"confidence\":80,",
		"",
		"data: \"summary\":\"Signal[VGT_NL]summary\",\"rationale\":\"Reason\",\"uncertainties\":[],\"recommended_actions\":[]}",
		"",
		"data: [[USAGE]]:{\"tokens\":10}",
	}, "\n")
	result, err := parseInternalSSE(input)
	if err != nil {
		t.Fatalf("parse internal SSE: %v", err)
	}
	if strings.Contains(result, "THINKING") || strings.Contains(result, "USAGE") || !strings.Contains(result, "Signal\nsummary") {
		t.Fatalf("SSE assessment separation failed: %q", result)
	}
	if _, err := alertAssessmentJSON(result); err != nil {
		t.Fatalf("extract assessment JSON: %v", err)
	}
}

func TestParseInternalSSERejectsProviderErrors(t *testing.T) {
	if _, err := parseInternalSSE("data: [SYSTEM ERROR]: provider unavailable\n\n"); err == nil {
		t.Fatal("provider errors must not be persisted as assessments")
	}
}

func TestBriefingReasoningContractHidesGPTOSSReasoning(t *testing.T) {
	payload := map[string]interface{}{}
	applyReasoningOptions(payload, "openai/gpt-oss-120b", ChatRequest{
		ReasoningEffort:     "low",
		ReasoningVisibility: "hidden",
	})
	if payload["reasoning_format"] != "hidden" || payload["reasoning_effort"] != "low" {
		t.Fatalf("briefing reasoning contract was not applied: %+v", payload)
	}
	if _, exposed := payload["include_reasoning"]; exposed {
		t.Fatalf("hidden reasoning must not be combined with include_reasoning: %+v", payload)
	}
}

func TestInvalidReasoningOverridesFallBackToProviderSafeValues(t *testing.T) {
	payload := map[string]interface{}{}
	applyReasoningOptions(payload, "openai/gpt-oss-20b", ChatRequest{ReasoningEffort: "max", UseTools: true})
	if payload["reasoning_effort"] != "medium" {
		t.Fatalf("unsupported Groq reasoning effort escaped validation: %+v", payload)
	}
}

func TestProviderSpecificReasoningContracts(t *testing.T) {
	t.Run("Groq Qwen", func(t *testing.T) {
		payload := map[string]interface{}{}
		applyReasoningOptions(payload, "qwen/qwen3.6-27b", ChatRequest{ReasoningEffort: "none", UseTools: true})
		if payload["reasoning_effort"] != "none" || payload["reasoning_format"] != "parsed" {
			t.Fatalf("Qwen reasoning contract mismatch: %+v", payload)
		}
	})
	t.Run("Gemini 3", func(t *testing.T) {
		payload := map[string]interface{}{}
		applyReasoningOptions(payload, "gemini-3.1-pro-preview", ChatRequest{ReasoningEffort: "high"})
		if payload["reasoning_effort"] != "high" {
			t.Fatalf("Gemini reasoning contract mismatch: %+v", payload)
		}
	})
	t.Run("DeepSeek disabled", func(t *testing.T) {
		payload := map[string]interface{}{"reasoning_effort": "high"}
		applyDeepSeekReasoningOptions(payload, ChatRequest{ReasoningEffort: "none"})
		thinking, ok := payload["thinking"].(map[string]interface{})
		if !ok || thinking["type"] != "disabled" {
			t.Fatalf("DeepSeek thinking toggle mismatch: %+v", payload)
		}
		if _, exists := payload["reasoning_effort"]; exists {
			t.Fatalf("DeepSeek disabled mode retained effort: %+v", payload)
		}
	})
	t.Run("Claude Opus adaptive", func(t *testing.T) {
		payload := map[string]interface{}{}
		if !applyClaudeReasoningOptions(payload, "claude-opus-4-8", ChatRequest{ReasoningEffort: "xhigh"}) {
			t.Fatal("Claude Opus reasoning was not activated")
		}
		thinking := payload["thinking"].(map[string]interface{})
		output := payload["output_config"].(map[string]interface{})
		if thinking["type"] != "adaptive" || output["effort"] != "xhigh" {
			t.Fatalf("Claude adaptive effort mismatch: %+v", payload)
		}
	})
	t.Run("Claude Sonnet rejects xhigh", func(t *testing.T) {
		payload := map[string]interface{}{}
		applyClaudeReasoningOptions(payload, "claude-sonnet-4-6", ChatRequest{ReasoningEffort: "xhigh"})
		output := payload["output_config"].(map[string]interface{})
		if output["effort"] != "medium" {
			t.Fatalf("unsupported Sonnet xhigh was not normalized: %+v", payload)
		}
	})
}
