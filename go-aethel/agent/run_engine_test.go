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
	"time"
)

type failingReadScopeSkill struct{}

func (failingReadScopeSkill) Name() string                       { return "fs_list_dir" }
func (failingReadScopeSkill) Description() string                { return "test" }
func (failingReadScopeSkill) Parameters() map[string]interface{} { return map[string]interface{}{} }
func (failingReadScopeSkill) RiskLevel() security.RiskLevel      { return security.RiskLow }
func (failingReadScopeSkill) Execute(json.RawMessage) (string, error) {
	return "", errors.New("SECURITY VIOLATION: path escaped the configured Windows workspace jail")
}

type retryTestSkill struct {
	attempts  int
	failUntil int
}

func (*retryTestSkill) Name() string                       { return "fs_list_dir" }
func (*retryTestSkill) Description() string                { return "bounded retry test" }
func (*retryTestSkill) Parameters() map[string]interface{} { return map[string]interface{}{} }
func (*retryTestSkill) RiskLevel() security.RiskLevel      { return security.RiskLow }
func (skill *retryTestSkill) Execute(json.RawMessage) (string, error) {
	skill.attempts++
	if skill.attempts <= skill.failUntil {
		return "", errors.New("transient read failure")
	}
	return `{"status":"ok"}`, nil
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

func TestGlobalWatchProfileIsReadOnly(t *testing.T) {
	profile := defaultAgentProfiles()["global_watch_operator"]
	for _, capability := range []security.Capability{security.CapFsWrite, security.CapIntelWrite, security.CapMemoryWrite, security.CapSphereWrite, security.CapSysExec} {
		if profileAllows(profile, capability) {
			t.Fatalf("global watch profile leaked mutating capability %s", capability)
		}
	}
	if !profileAllows(profile, security.CapIntelRead) || !profileAllows(profile, security.CapFsRead) {
		t.Fatal("global watch profile lost required read capabilities")
	}
}

func TestOSINTProfilesEnforceCapabilitySeparation(t *testing.T) {
	profiles := defaultAgentProfiles()
	for _, id := range []string{"collector", "analyst", "case_worker", "operator", "developer"} {
		if _, exists := profiles[id]; !exists {
			t.Fatalf("required capability profile %s is missing", id)
		}
	}
	for _, id := range []string{"collector", "analyst", "researcher", "global_watch_operator"} {
		profile := profiles[id]
		for _, capability := range []security.Capability{security.CapFsWrite, security.CapSysExec, security.CapMessagingSend, security.CapGuiClick, security.CapGuiType} {
			if profileAllows(profile, capability) {
				t.Fatalf("read-oriented profile %s leaked capability %s", id, capability)
			}
		}
	}
	if !profileAllows(profiles["collector"], security.CapIntelSources) || profileAllows(profiles["collector"], security.CapIntelWrite) {
		t.Fatal("collector must fetch through the source broker without general intelligence mutation rights")
	}
	if !profileAllows(profiles["case_worker"], security.CapIntelWrite) || profileAllows(profiles["case_worker"], security.CapFsWrite) {
		t.Fatal("case worker must be case-authority scoped and host-write isolated")
	}
	if !profileAllows(profiles["developer"], security.CapFsWrite) || !profileAllows(profiles["developer"], security.CapSysExec) {
		t.Fatal("developer profile lost its explicitly separated development authority")
	}
}

func TestOSINTToolSchemasDoNotExposeCrossDomainEffects(t *testing.T) {
	for _, profileID := range []string{"collector", "analyst", "case_worker", "global_watch_operator", "researcher"} {
		tools := toolAllowlistForRun(AgentRun{ProfileID: profileID, Objective: "Analyse public intelligence sources"})
		for _, forbidden := range []string{"fs_write_file", "fs_replace_file_content", "sys_exec_cmd", "mail_send_message", "gui_control", "gui_window_control"} {
			if containsString(tools, forbidden) {
				t.Fatalf("profile %s exposed forbidden tool %s", profileID, forbidden)
			}
		}
	}
}

func TestRunRetriesSafeStepAndCreatesDeterministicCheckpoint(t *testing.T) {
	root := t.TempDir()
	engine := NewRunEngine(filepath.Join(root, "runs.json"))
	run, err := engine.Create(CreateRunRequest{Objective: "Retry bounded transient reads", ProfileID: "researcher", Steps: []RunStep{{
		Kind: RunStepTool, Title: "Read source", ToolName: "fs_list_dir", ToolArgs: json.RawMessage(`{"path":"."}`),
		RetrySafe: true, MaxAttempts: 3, RetryBackoff: time.Millisecond,
		OutputSchema: json.RawMessage(`{"type":"object","required":["status"]}`), Postcondition: "json_valid",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Start(run.ID); err != nil {
		t.Fatal(err)
	}
	registry := skills.NewSkillRegistry()
	skill := &retryTestSkill{failUntil: 2}
	registry.Register(skill)
	policy := security.NewPolicyEngine(security.NewSecurityGuard(), security.NewLeaseManager(filepath.Join(root, "leases.json")), security.NewAuditLogger(filepath.Join(root, "audit.json")))
	run, err = engine.Advance(run.ID, policy, registry)
	if err != nil || run.Steps[0].Status != StepVerified || run.Steps[0].Attempts != 3 || len(run.Steps[0].CheckpointHash) != 64 {
		t.Fatalf("retry/checkpoint contract failed: %+v %v", run, err)
	}
	checkpoint := run.Steps[0].CheckpointHash
	reloaded := NewRunEngine(filepath.Join(root, "runs.json"))
	persisted, _ := reloaded.Get(run.ID)
	if persisted.Steps[0].CheckpointHash != checkpoint {
		t.Fatal("deterministic checkpoint did not survive restart")
	}
}

func TestRunRoutesExhaustedRetryAndSchemaFailureToDeadLetter(t *testing.T) {
	root := t.TempDir()
	engine := NewRunEngine(filepath.Join(root, "runs.json"))
	run, err := engine.Create(CreateRunRequest{Objective: "Reject invalid tool output", ProfileID: "researcher", Steps: []RunStep{{
		Kind: RunStepTool, Title: "Read source", ToolName: "fs_list_dir", ToolArgs: json.RawMessage(`{"path":"."}`),
		RetrySafe: true, MaxAttempts: 2, RetryBackoff: time.Millisecond,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Start(run.ID); err != nil {
		t.Fatal(err)
	}
	registry := skills.NewSkillRegistry()
	registry.Register(&retryTestSkill{failUntil: 5})
	policy := security.NewPolicyEngine(security.NewSecurityGuard(), security.NewLeaseManager(filepath.Join(root, "leases.json")), security.NewAuditLogger(filepath.Join(root, "audit.json")))
	run, err = engine.Advance(run.ID, policy, registry)
	if err != nil || run.Status != RunFailed || len(run.DeadLetters) != 1 || run.DeadLetters[0].Attempts != 2 {
		t.Fatalf("exhausted retries were not dead-lettered: %+v %v", run, err)
	}

	second, err := engine.Create(CreateRunRequest{Objective: "Enforce output schema", ProfileID: "researcher", Steps: []RunStep{{Kind: RunStepPlan, Title: "Plain plan", OutputSchema: json.RawMessage(`{"type":"object"}`)}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Start(second.ID); err != nil {
		t.Fatal(err)
	}
	second, err = engine.Advance(second.ID, policy, registry)
	if err != nil || second.Status != RunFailed || len(second.DeadLetters) != 1 {
		t.Fatalf("schema failure was not dead-lettered: %+v %v", second, err)
	}
}

func TestRunIdempotentReplayAndCancellationCompensation(t *testing.T) {
	root := t.TempDir()
	engine := NewRunEngine(filepath.Join(root, "runs.json"))
	compensationArgs := json.RawMessage(`{"path":"."}`)
	run, err := engine.Create(CreateRunRequest{Objective: "Verify replay and compensation", ProfileID: "researcher", Steps: []RunStep{
		{Kind: RunStepPlan, Title: "Same deterministic plan", CompensationTool: "fs_list_dir", CompensationArgs: compensationArgs},
		{Kind: RunStepPlan, Title: "Same deterministic plan"},
		{Kind: RunStepReport, Title: "Keep run cancellable"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Start(run.ID); err != nil {
		t.Fatal(err)
	}
	policy := security.NewPolicyEngine(security.NewSecurityGuard(), security.NewLeaseManager(filepath.Join(root, "leases.json")), security.NewAuditLogger(filepath.Join(root, "audit.json")))
	registry := skills.NewSkillRegistry()
	registry.Register(&retryTestSkill{})
	if run, err = engine.Advance(run.ID, policy, registry); err != nil {
		t.Fatal(err)
	}
	if run, err = engine.Advance(run.ID, policy, registry); err != nil {
		t.Fatal(err)
	}
	if run.Steps[1].Status != StepVerified || run.Steps[1].CheckpointHash != run.Steps[0].CheckpointHash {
		t.Fatalf("idempotent replay did not reuse checkpoint: %+v", run.Steps)
	}
	run, err = engine.Cancel(run.ID)
	if err != nil || len(run.Compensations) != 1 || run.Compensations[0].Status != "pending" {
		t.Fatalf("cancellation did not queue compensation: %+v %v", run, err)
	}
	run, err = engine.Compensate(run.ID, policy, registry)
	if err != nil || run.Compensations[0].Status != "completed" {
		t.Fatalf("compensation did not complete: %+v %v", run.Compensations, err)
	}
}
