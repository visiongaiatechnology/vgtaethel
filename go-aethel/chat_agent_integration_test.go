//go:build integration && e2e

package main

// Optional live E2E chat-agent test (requires GROQ_API_KEY and full app wiring).
// Excluded from default `go test .` so package main compiles without agent
// unexported symbols / live credentials.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"go-aethel/agent"
	"go-aethel/memory"
	"go-aethel/personal"
	"go-aethel/provider"
	"go-aethel/security"
	"go-aethel/skills"
)

func TestChatAgentE2EIntegration(t *testing.T) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" || !strings.HasPrefix(apiKey, "gsk_") {
		t.Skip("Skipping E2E Chat-Runner test: GROQ_API_KEY is not configured in environment")
	}

	memoryStore := memory.NewLocalMemoryStore()
	personalStore := personal.NewPersonalStore("./vgt_workspace/personal_test")
	registry := skills.NewSkillRegistry()
	providers := provider.NewProviderRegistry()
	registry.Register(&skills.ListDirSkill{})

	guard := security.NewSecurityGuard()
	leases := security.NewLeaseManager("./vgt_workspace/active_leases_test.json")
	audit := security.NewAuditLogger("./vgt_workspace/security_audit_test.json")
	policy := security.NewPolicyEngine(guard, leases, audit)
	approvals := security.NewApprovalManager("./vgt_workspace/approval_grants_test.json")
	runEngine := agent.NewRunEngine("./vgt_workspace/agent_runs_test.json")

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

	_ = os.Remove("./vgt_workspace/agent_runs_test.json")

	run, err := state.runs.Create(agent.CreateRunRequest{
		Objective:     "List files in the current workspace directory using fs_list_dir tool.",
		ProfileID:     "developer",
		ModelID:       "openai/gpt-oss-120b",
		Mode:          "chat_agent",
		CostBudgetUSD: 1.0,
		Steps: []agent.RunStep{
			{Kind: agent.RunStepPlan, Title: "Arbeitsplan erstellen"},
		},
	})
	if err != nil {
		t.Fatalf("failed to create run: %v", err)
	}

	if run.Mode != "chat_agent" {
		t.Errorf("expected run mode to be chat_agent, got %s", run.Mode)
	}

	run, err = state.runs.Start(run.ID)
	if err != nil {
		t.Fatalf("failed to start run: %v", err)
	}

	t.Logf("Running E2E simulation for run %s using model %s", run.ID, run.ModelID)
	// Live model invocation path is package-agent internal; exercise create/start only here.
	_ = json.RawMessage(nil)
	_ = run
}
