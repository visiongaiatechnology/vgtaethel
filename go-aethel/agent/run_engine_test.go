package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"go-aethel/security"
	"go-aethel/skills"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failingReadScopeSkill struct{}

func (failingReadScopeSkill) Name() string                       { return "fs_list_dir" }
func (failingReadScopeSkill) Description() string                { return "test" }
func (failingReadScopeSkill) Parameters() map[string]interface{} { return map[string]interface{}{} }
func (failingReadScopeSkill) RiskLevel() security.RiskLevel      { return security.RiskLow }
func (failingReadScopeSkill) Execute(json.RawMessage) (string, error) {
	return "", errors.New("SECURITY VIOLATION: path escaped the configured Windows workspace jail")
}

func TestRunEnginePersistsControlledLifecycle(t *testing.T) {
	engine := NewRunEngine(filepath.Join(t.TempDir(), "runs.json"))
	run, err := engine.Create(CreateRunRequest{
		Objective: "Projektstatus strukturiert erfassen",
		ProfileID: "researcher",
		Steps: []RunStep{
			{Kind: RunStepPlan, Title: "Arbeitsplan bestätigen"},
			{Kind: RunStepReport, Title: "Ergebnis berichten"},
		},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if run.Status != RunQueued || len(run.Trace) != 1 {
		t.Fatalf("unexpected initial run: %+v", run)
	}
	if _, err := engine.Start(run.ID); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if run, err = engine.Advance(run.ID, security.NewPolicyEngine(security.NewSecurityGuard(), security.NewLeaseManager(filepath.Join(t.TempDir(), "leases.json")), security.NewAuditLogger(filepath.Join(t.TempDir(), "audit.json"))), skills.NewSkillRegistry()); err != nil {
		t.Fatalf("first advance failed: %v", err)
	}
	if run.Steps[0].Status != StepVerified || run.Status != RunRunning {
		t.Fatalf("plan step was not durably verified: %+v", run)
	}
	if _, err := engine.Pause(run.ID); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
	if _, err := engine.Advance(run.ID, nil, nil); err == nil {
		t.Fatal("paused run advanced")
	}
	if _, err := engine.Start(run.ID); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	if _, err := engine.Advance(run.ID, security.NewPolicyEngine(security.NewSecurityGuard(), security.NewLeaseManager(filepath.Join(t.TempDir(), "leases2.json")), security.NewAuditLogger(filepath.Join(t.TempDir(), "audit2.json"))), skills.NewSkillRegistry()); err != nil {
		t.Fatalf("report advance failed: %v", err)
	}
	if run, err = engine.Advance(run.ID, security.NewPolicyEngine(security.NewSecurityGuard(), security.NewLeaseManager(filepath.Join(t.TempDir(), "leases3.json")), security.NewAuditLogger(filepath.Join(t.TempDir(), "audit3.json"))), skills.NewSkillRegistry()); err != nil {
		t.Fatalf("completion advance failed: %v", err)
	}
	if run.Status != RunCompleted || run.CompletedAt == nil {
		t.Fatalf("run did not complete: %+v", run)
	}
}

func TestReadScopeJailErrorReturnsRecoveryContextInsteadOfFailingRun(t *testing.T) {
	engine := NewRunEngine(filepath.Join(t.TempDir(), "runs.json"))
	run, err := engine.Create(CreateRunRequest{
		Objective: "Ordner pruefen",
		ProfileID: "researcher",
		Steps:     []RunStep{{Kind: RunStepTool, Title: "Ordner lesen", ToolName: "fs_list_dir", ToolArgs: json.RawMessage(`{"path":"C:\\outside"}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = engine.Start(run.ID); err != nil {
		t.Fatal(err)
	}
	registry := skills.NewSkillRegistry()
	registry.Register(failingReadScopeSkill{})
	policy := security.NewPolicyEngine(security.NewSecurityGuard(), security.NewLeaseManager(filepath.Join(t.TempDir(), "leases.json")), security.NewAuditLogger(filepath.Join(t.TempDir(), "audit.json")))
	run, err = engine.Advance(run.ID, policy, registry)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunRunning || run.Steps[0].Status != StepVerified {
		t.Fatalf("recoverable read scope error stopped the run: %+v", run)
	}
	if !strings.Contains(run.Steps[0].Result, "fs_mount_folder") {
		t.Fatalf("missing recovery instruction: %q", run.Steps[0].Result)
	}
}

func TestRunEngineWaitsForApprovalBeforeToolExecution(t *testing.T) {
	engine := NewRunEngine(filepath.Join(t.TempDir(), "runs.json"))
	args, err := json.Marshal(skills.BrowserArgs{Action: "search", SearchQuery: "Aethel Produktstatus"})
	if err != nil {
		t.Fatal(err)
	}
	run, err := engine.Create(CreateRunRequest{
		Objective: "Eine Webrecherche kontrolliert starten",
		ProfileID: "researcher",
		Steps:     []RunStep{{Kind: RunStepTool, Title: "Webrecherche starten", ToolName: "web_browser", ToolArgs: args}},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := engine.Start(run.ID); err != nil {
		t.Fatal(err)
	}
	registry := skills.NewSkillRegistry()
	registry.Register(&skills.WebBrowserSkill{})
	run, err = engine.Advance(run.ID, security.NewPolicyEngine(security.NewSecurityGuard(), security.NewLeaseManager(filepath.Join(t.TempDir(), "leases.json")), security.NewAuditLogger(filepath.Join(t.TempDir(), "audit.json"))), registry)
	if err != nil {
		t.Fatalf("Advance failed: %v", err)
	}
	if run.Status != RunWaitingApproval || run.Steps[0].Status != StepWaitingApproval {
		t.Fatalf("unguarded tool did not wait for approval: %+v", run)
	}
	if _, err := engine.Start(run.ID); err == nil {
		t.Fatal("waiting approval could be resumed without a signed approval")
	}
}

func TestRunEngineRecoversInFlightRunAsPaused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.json")
	engine := NewRunEngine(path)
	run, err := engine.Create(CreateRunRequest{Objective: "Recovery testen", ProfileID: "researcher"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Start(run.ID); err != nil {
		t.Fatal(err)
	}
	reloaded := NewRunEngine(path)
	recovered, ok := reloaded.Get(run.ID)
	if !ok || recovered.Status != RunPaused {
		t.Fatalf("in-flight run was not recovered as paused: %+v", recovered)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("Recovery testen")) {
		t.Fatal("agent run journal was persisted in plaintext")
	}
}

func TestRunEnginePausesAtCostBudget(t *testing.T) {
	engine := NewRunEngine(filepath.Join(t.TempDir(), "runs.json"))
	run, err := engine.Create(CreateRunRequest{Objective: "Budget testen", ProfileID: "researcher", CostBudgetUSD: 1})
	if err != nil {
		t.Fatal(err)
	}
	run, err = engine.RecordCost(run.ID, 1.01)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunPaused || run.SpentUSD != 1.01 {
		t.Fatalf("cost budget did not pause run: %+v", run)
	}
}

func TestRunApprovalIsBoundToPendingRunStep(t *testing.T) {
	temp := t.TempDir()
	engine := NewRunEngine(filepath.Join(temp, "runs.json"))
	args, _ := json.Marshal(skills.BrowserArgs{Action: "search", SearchQuery: "Aethel"})
	run, err := engine.Create(CreateRunRequest{Objective: "Freigabe testen", ProfileID: "researcher", Steps: []RunStep{{Kind: RunStepTool, Title: "Web suchen", ToolName: "web_browser", ToolArgs: args}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Start(run.ID); err != nil {
		t.Fatal(err)
	}
	policy := security.NewPolicyEngine(security.NewSecurityGuard(), security.NewLeaseManager(filepath.Join(temp, "leases.json")), security.NewAuditLogger(filepath.Join(temp, "audit.json")))
	registry := skills.NewSkillRegistry()
	registry.Register(&skills.WebBrowserSkill{})
	run, err = engine.Advance(run.ID, policy, registry)
	if err != nil || run.Status != RunWaitingApproval {
		t.Fatalf("run did not wait: %+v err=%v", run, err)
	}
	step, ok := engine.PendingApproval(run.ID)
	if !ok {
		t.Fatal("pending approval step missing")
	}
	_, decision, report := policy.Evaluate(step.ToolName, string(step.ToolArgs), false)
	if decision != "needs_approval" {
		t.Fatalf("unexpected policy decision: %s", decision)
	}
	approvals := security.NewApprovalManager(filepath.Join(temp, "approvals.json"))
	_, token, err := approvals.Issue(step.ToolName, string(step.ToolArgs), report.Capability, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	run, err = engine.Approve(run.ID, token, approvals, policy)
	if err != nil || run.Status != RunRunning || !run.Steps[0].ApprovalGranted {
		t.Fatalf("run approval failed: %+v err=%v", run, err)
	}
	if _, err := engine.Approve(run.ID, token, approvals, policy); err == nil {
		t.Fatal("approval token was replayed")
	}
}

func TestRunPersistsLiveOperatorContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.sealed")
	engine := NewRunEngine(path)
	run, err := engine.Create(CreateRunRequest{
		Objective: "Inspect the visible application", ProfileID: "browser_operator", Mode: "chat_agent", LiveOperator: true,
		Steps: []RunStep{{Kind: RunStepPlan, Title: "Validate visual task"}},
	})
	if err != nil || !run.LiveOperator {
		t.Fatalf("live operator context was not created: run=%+v err=%v", run, err)
	}
	reloaded := NewRunEngine(path)
	persisted, ok := reloaded.Get(run.ID)
	if !ok || !persisted.LiveOperator {
		t.Fatalf("live operator context was not persisted: %+v", persisted)
	}
}

func TestSphereWorkspaceProfileAllowsWriterButNotHostControl(t *testing.T) {
	profile, ok := defaultAgentProfiles()["sphere_workspace"]
	if !ok {
		t.Fatal("sphere workspace profile missing")
	}
	if !profileAllows(profile, security.CapFsWrite) || !profileAllows(profile, security.CapBrowserOpen) || !profileAllows(profile, security.CapWeatherRead) {
		t.Fatal("sphere workspace profile is missing required workspace capabilities")
	}
	for _, capability := range []security.Capability{security.CapSysExec, security.CapFsMount, security.CapGuiType, security.CapGuiClick, security.CapBrowserCtl} {
		if profileAllows(profile, capability) {
			t.Fatalf("sphere workspace profile unexpectedly allows %s", capability)
		}
	}
}

func TestInteractiveProfilesAllowSafeLiveDataLookups(t *testing.T) {
	profiles := defaultAgentProfiles()
	for _, id := range []string{"researcher", "developer", "browser_operator", "sphere_workspace", "personal_assistant"} {
		profile := profiles[id]
		if !profileAllows(profile, security.CapWeatherRead) || !profileAllows(profile, security.CapMarketRead) {
			t.Fatalf("profile %s cannot execute safe live-data lookups", id)
		}
	}
}
