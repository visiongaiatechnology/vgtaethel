package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCognitiveTierKeepsSmallConversationOnDirectPath(t *testing.T) {
	if tier := classifyCognitiveTier("Was geht ab :D", false); tier != CognitiveDirect {
		t.Fatalf("small conversation selected expensive tier %q", tier)
	}
	if shouldUseOrchestrator(CognitiveDirect, "Was geht ab :D", nil) {
		t.Fatal("direct conversation unnecessarily enabled orchestrator stage")
	}
	if tier := classifyCognitiveTier("Schreibe im Sphere Writer ein Gedicht", true); tier != CognitiveOperational {
		t.Fatalf("sphere mutation selected tier %q", tier)
	}
}

func TestContinuityTokenBudgetRetainsUsageAndSignalsExhaustion(t *testing.T) {
	state := newContinuityState("Kurze Antwort", false)
	if state.recordUsage(200, 50, 100) {
		t.Fatal("ordinary response exhausted direct budget")
	}
	if state.TokenBudget.InputTokens != 200 || state.TokenBudget.OutputTokens != 50 || state.TokenBudget.CachedTokens != 100 {
		t.Fatalf("usage was not recorded: %+v", state.TokenBudget)
	}
	if !state.recordUsage(state.TokenBudget.MaxInputTokens, 0, 0) || !state.TokenBudget.Exhausted {
		t.Fatal("token exhaustion was not signaled")
	}
}

func TestContinuityLoopGuardRejectsThirdIdenticalAction(t *testing.T) {
	state := newContinuityState("Lies eine Datei", false)
	args := json.RawMessage(`{"path":"notes.txt"}`)
	if err := state.registerAction("fs_read_file", args); err != nil {
		t.Fatal(err)
	}
	if err := state.registerAction("fs_read_file", args); err != nil {
		t.Fatal(err)
	}
	if err := state.registerAction("fs_read_file", args); err == nil || !strings.Contains(err.Error(), "loop guard") {
		t.Fatalf("third identical action was not rejected: %v", err)
	}
}

func TestContinuityEvidenceUsesDigestInsteadOfDuplicatingToolOutput(t *testing.T) {
	state := newContinuityState("Prüfe Ergebnis", false)
	step := RunStep{ID: "step_01", Kind: RunStepTool, ToolName: "fs_read_file", Status: StepVerified, Result: strings.Repeat("sensitive-result", 100)}
	state.addEvidence(step)
	if len(state.Evidence) != 1 || !state.Evidence[0].Verified || !strings.HasPrefix(state.Evidence[0].ResultDigest, "sha256:") {
		t.Fatalf("verified evidence missing: %+v", state.Evidence)
	}
	if strings.Contains(state.Evidence[0].ResultDigest, "sensitive-result") {
		t.Fatal("raw tool output leaked into continuity evidence")
	}
}

func TestCompactAgentContextUsesDeterministicCheckpoint(t *testing.T) {
	run := AgentRun{Objective: "Prüfe das Projekt", Continuity: newContinuityState("Prüfe das Projekt", false)}
	messages := make([]json.RawMessage, 0, 30)
	for index := 0; index < 30; index++ {
		role := "assistant"
		if index%2 == 0 {
			role = "user"
		}
		raw, err := json.Marshal(map[string]string{"role": role, "content": strings.Repeat("x", 5000)})
		if err != nil {
			t.Fatal(err)
		}
		messages = append(messages, raw)
	}
	compacted := compactAgentContext(run, messages)
	if len(compacted) >= len(messages) || !strings.Contains(string(compacted[0]), "AETHEL_CONTINUITY_CHECKPOINT") {
		t.Fatalf("context was not compacted with checkpoint: %d -> %d", len(messages), len(compacted))
	}
	if len(messages) != 30 {
		t.Fatal("source journal messages were mutated")
	}
}

func TestReasoningPolicyMinimizesDirectRequests(t *testing.T) {
	if effort, visibility := reasoningPolicy(CognitiveDirect, "openai/gpt-oss-20b"); effort != "low" || visibility != "hidden" {
		t.Fatalf("unexpected GPT-OSS direct policy: %s %s", effort, visibility)
	}
	if effort, _ := reasoningPolicy(CognitiveDirect, "qwen/qwen3.6-27b"); effort != "none" {
		t.Fatalf("Qwen direct policy should disable reasoning, got %s", effort)
	}
	if effort, _ := reasoningPolicy(CognitiveExtended, "openai/gpt-oss-120b"); effort != "high" {
		t.Fatalf("extended work should retain high reasoning, got %s", effort)
	}
}
