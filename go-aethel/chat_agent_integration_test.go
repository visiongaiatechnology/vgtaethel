package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestChatAgentE2EIntegration(t *testing.T) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" || !strings.HasPrefix(apiKey, "gsk_") {
		t.Skip("Skipping E2E Chat-Runner test: GROQ_API_KEY is not configured in environment")
	}

	// 1. Setup AppState
	memoryStore := NewLocalMemoryStore()
	personalStore := NewPersonalStore("./vgt_workspace/personal_test")
	registry := NewSkillRegistry()
	providers := NewProviderRegistry()
	registry.Register(&ListDirSkill{})

	guard := NewSecurityGuard()
	leases := NewLeaseManager("./vgt_workspace/active_leases_test.json")
	audit := NewAuditLogger("./vgt_workspace/security_audit_test.json")
	policy := NewPolicyEngine(guard, leases, audit)
	approvals := NewApprovalManager("./vgt_workspace/approval_grants_test.json")
	runEngine := NewRunEngine("./vgt_workspace/agent_runs_test.json")

	state = &AppState{
		apiKey:    apiKey,
		guard:     guard,
		leases:    leases,
		audit:     audit,
		policy:    policy,
		approvals: approvals,
		skills:    registry,
		providers: providers,
		memory:    memoryStore,
		runs:      runEngine,
		personal:  personalStore,
	}

	// Clean up previous runs
	_ = os.Remove("./vgt_workspace/agent_runs_test.json")

	// 2. Create a chat agent run
	// Fictional model openai/gpt-oss-120b will map to llama-3.3-70b-versatile
	run, err := state.runs.Create(CreateRunRequest{
		Objective:     "List files in the current workspace directory using fs_list_dir tool.",
		ProfileID:     "developer",
		ModelID:       "openai/gpt-oss-120b",
		Mode:          "chat_agent",
		CostBudgetUSD: 1.0,
		Steps: []RunStep{
			{Kind: RunStepPlan, Title: "Arbeitsplan erstellen"},
		},
	})
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	if run.Mode != "chat_agent" {
		t.Errorf("expected run mode to be chat_agent, got %s", run.Mode)
	}

	// 3. Start run
	run, err = state.runs.Start(run.ID)
	if err != nil {
		t.Fatalf("failed to start run: %v", err)
	}

	// 4. Simulate a turn of driveChatAgentRun synchronously for testing
	t.Logf("Running E2E simulation for run %s using model %s", run.ID, run.ModelID)
	
	// We can invoke the model through handleChat internally
	result := invokeAgentModel(run, false)
	if result.Err != nil {
		t.Fatalf("agent model invocation failed: %v", result.Err)
	}

	t.Logf("Agent model returned thinking: %s", result.Thinking)
	t.Logf("Agent model returned text: %s", result.Text)

	// Since we expect the model to output tool call for fs_list_dir, verify it
	if len(result.ToolCalls) == 0 {
		t.Fatal("expected agent model to call a tool, but got 0 calls")
	}

	toolCall := result.ToolCalls[0]
	if toolCall.Name != "fs_list_dir" {
		t.Errorf("expected tool call to be fs_list_dir, got %s", toolCall.Name)
	}

	// 5. Simulate approval flow, pause, and resume transitions
	t.Log("Simulating run lifecycle transitions (Pause, Resume, Cancel)")
	run, err = state.runs.Pause(run.ID)
	if err != nil || run.Status != RunPaused {
		t.Fatalf("failed to pause run: %v", err)
	}

	run, err = state.runs.Start(run.ID)
	if err != nil || run.Status != RunRunning {
		t.Fatalf("failed to resume run: %v", err)
	}

	// 6. Complete the run with a final report
	messages := append([]json.RawMessage(nil), run.AgentMessages...)
	messages = append(messages, json.RawMessage(`{"role":"assistant","content":"Done"}`))
	run, err = state.runs.CompleteAgent(run.ID, "Completion report text", messages)
	if err != nil {
		t.Fatalf("failed to complete run: %v", err)
	}

	if run.Status != RunCompleted {
		t.Errorf("expected run to be completed, got %s", run.Status)
	}
	if run.FinalReport != "Completion report text" {
		t.Errorf("unexpected final report: %s", run.FinalReport)
	}

	// Clean up files
	_ = os.Remove("./vgt_workspace/agent_runs_test.json")
	_ = os.Remove("./vgt_workspace/active_leases_test.json")
	_ = os.Remove("./vgt_workspace/security_audit_test.json")
	_ = os.Remove("./vgt_workspace/approval_grants_test.json")
}
