package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-aethel/memory"
	"go-aethel/security"
	"go-aethel/skills"
	"go-aethel/system"
)

func TestBetaV2ReleaseIdentityIsConsistent(t *testing.T) {
	if system.ProductVersion != "1.0.0-beta.2" {
		t.Fatalf("unexpected product version %q", system.ProductVersion)
	}

	files := map[string][]string{
		"wails.json":                            {`"productVersion": "1.0.0-beta.2"`, "Beta V2"},
		filepath.Join("frontend", "index.html"): {"BETA V2", "PRODUCTION CANDIDATE"},
		filepath.Join("build", "windows", "installer", "aethel.iss"):       {`#define AppVersion "1.0.0-beta.2"`},
		filepath.Join("docs", "project", "RELEASE_NOTES.md"):               {"1.0.0-beta.2", "BETA V2"},
		filepath.Join("..", ".github", "workflows", "windows-release.yml"): {"1.0.0-beta.2-dev"},
	}
	for path, required := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read release identity file %s: %v", path, err)
		}
		for _, marker := range required {
			if !strings.Contains(string(content), marker) {
				t.Errorf("%s is missing release marker %q", path, marker)
			}
		}
	}
}

func TestPolicyEngineBlocksForbiddenActions(t *testing.T) {
	tmpDir := t.TempDir()
	policy := security.NewPolicyEngine(
		security.NewSecurityGuard(),
		security.NewLeaseManager(filepath.Join(tmpDir, "active_leases.json")),
		security.NewAuditLogger(filepath.Join(tmpDir, "security_audit.json")),
	)

	allowed, decision, report := policy.Evaluate("sys_exec_cmd", `{"command":"rm","args":["-rf","/usr/bin"]}`, false)
	if allowed || decision != "blocked" || report.RiskLevel != security.RiskForbidden {
		t.Fatalf("rm -rf policy mismatch: allowed=%t decision=%q risk=%q threats=%v", allowed, decision, report.RiskLevel, report.Threats)
	}

	allowed, decision, report = policy.Evaluate("gui_control", `{"action":"press","keys":"alt+f4"}`, false)
	if allowed || decision != "blocked" || report.RiskLevel != security.RiskForbidden {
		t.Fatalf("alt+f4 policy mismatch: allowed=%t decision=%q risk=%q threats=%v", allowed, decision, report.RiskLevel, report.Threats)
	}

	allowed, decision, report = policy.Evaluate("sys_exec_cmd", `{"command":"powershell","args":["-NoProfile","-Command","Get-ChildItem"]}`, false)
	if allowed || decision != "needs_approval" || report.RiskLevel != security.RiskHigh {
		t.Fatalf("powershell interpreter policy mismatch: allowed=%t decision=%q risk=%q threats=%v", allowed, decision, report.RiskLevel, report.Threats)
	}

	allowed, decision, report = policy.Evaluate("sys_exec_cmd", `{"command":"git","args":["status; whoami"]}`, true)
	if allowed || decision != "blocked" || report.RiskLevel != security.RiskCritical {
		t.Fatalf("critical command-injection signal was overrideable: allowed=%t decision=%q risk=%q", allowed, decision, report.RiskLevel)
	}
}

func TestBuildScriptCannotCleanRuntimeWorkspace(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("scripts", "build_aethel.bat"))
	if err != nil {
		t.Fatal(err)
	}
	script := strings.ToLower(string(content))
	if strings.Contains(script, "wails build -clean") || strings.Contains(script, "wails build --clean") {
		t.Fatal("build script enables Wails bin cleanup and can delete the runtime workspace")
	}
	if !strings.Contains(script, "wails build -platform windows/amd64") {
		t.Fatal("non-destructive Wails build invocation missing")
	}
	if !strings.Contains(script, "build\\bin\\vgt_workspace") {
		t.Fatal("runtime workspace preservation contract missing from build script")
	}
}

func TestLeaseLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	leases := security.NewLeaseManager(filepath.Join(tmpDir, "active_leases.json"))
	policy := security.NewPolicyEngine(security.NewSecurityGuard(), leases, security.NewAuditLogger(filepath.Join(tmpDir, "security_audit.json")))

	args := `{"command":"go","args":["version"]}`
	allowed, decision, _ := policy.Evaluate("sys_exec_cmd", args, false)
	if allowed || decision != "needs_approval" {
		t.Fatalf("expected approval gate before lease, got allowed=%t decision=%q", allowed, decision)
	}

	lease := security.PermissionLease{
		LeaseID:             "lease_test",
		Capability:          security.CapSysExec,
		CreatedAt:           time.Now(),
		ExpiresAt:           time.Now().Add(15 * time.Minute),
		RequiresVisibleMode: true,
		Revocable:           true,
		ApprovedBy:          "test",
		ApprovalMethod:      "unit-test",
	}
	if err := leases.AddLease(lease); err != nil {
		t.Fatalf("AddLease failed: %v", err)
	}

	allowed, decision, _ = policy.Evaluate("sys_exec_cmd", args, false)
	if !allowed || decision != lease.LeaseID {
		t.Fatalf("expected lease allow, got allowed=%t decision=%q", allowed, decision)
	}

	if !leases.RevokeLease(lease.LeaseID) {
		t.Fatalf("expected lease revocation to succeed")
	}

	allowed, decision, _ = policy.Evaluate("sys_exec_cmd", args, false)
	if allowed || decision != "needs_approval" {
		t.Fatalf("expected approval gate after revocation, got allowed=%t decision=%q", allowed, decision)
	}
}

func TestLeaseRejectsForbiddenKeys(t *testing.T) {
	tmpDir := t.TempDir()
	leases := security.NewLeaseManager(filepath.Join(tmpDir, "active_leases.json"))
	lease := security.PermissionLease{
		LeaseID:    "lease_keys",
		Capability: security.CapGuiPressKey,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(time.Minute),
		Scope:      security.Scope{ForbiddenKeys: []string{"alt+f4"}},
	}
	if err := leases.AddLease(lease); err != nil {
		t.Fatalf("AddLease failed: %v", err)
	}
	if allowed, _ := leases.CheckLease(security.CapGuiPressKey, `{"keys":"alt+f4"}`, "gui_control"); allowed {
		t.Fatal("lease allowed a forbidden key sequence")
	}
}

func TestSecretVaultRefusesMalformedExistingKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "vault.key")
	if err := os.WriteFile(keyPath, []byte("invalid"), 0600); err != nil {
		t.Fatalf("write malformed key: %v", err)
	}
	if _, err := security.NewSecretVault(filepath.Join(tmpDir, "vault.enc"), keyPath); err == nil {
		t.Fatal("vault accepted and would have overwritten a malformed existing key")
	}
}

func TestMemoryClassifiesAndRejectsSensitiveValues(t *testing.T) {
	if !memory.ContainsSensitiveMemoryData("api_key=sk-example_secret_value_123456") {
		t.Fatal("API key pattern was accepted for general memory")
	}
	if memory.ContainsSensitiveMemoryData("Die bevorzugte Sprache ist Deutsch.") {
		t.Fatal("ordinary preference was misclassified as sensitive")
	}
	if got := memory.NormalizeMemoryCategory("unknown category"); got != "general" {
		t.Fatalf("unexpected fallback category: %q", got)
	}
}

func TestMemoryRequiresConsentAndStoresProvenance(t *testing.T) {
	oldMemoryFile := memory.MemoryFile
	memory.MemoryFile = filepath.Join(t.TempDir(), "nexus_memory.json")
	defer func() { memory.MemoryFile = oldMemoryFile }()
	store := memory.NewLocalMemoryStore()
	if _, err := store.AddWithConsent("Der Nutzer bevorzugt Deutsch.", "preference", "assistant_tool", false, nil, ""); err == nil {
		t.Fatal("memory write succeeded without operator consent")
	}
	if _, err := store.AddWithConsent("api_key=sk-example_secret_value_123456", "general", "operator", true, nil, ""); err == nil {
		t.Fatal("sensitive content was accepted with consent")
	}
	id, err := store.AddWithConsent("Der Nutzer bevorzugt Deutsch.", "preference", "operator", true, nil, "")
	if err != nil || id == "" {
		t.Fatalf("consented memory write failed: id=%q err=%v", id, err)
	}
	memories := store.GetAll()
	if len(memories) != 1 || memories[0].Source != "operator" || memories[0].ConsentAt.IsZero() {
		t.Fatalf("memory provenance was not retained: %+v", memories)
	}
	data, err := os.ReadFile(memory.MemoryFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "Der Nutzer bevorzugt Deutsch.") {
		t.Fatal("Nexus memory was written in plaintext")
	}
}

func TestMemorySupersedingCreatesExplainableActiveRecord(t *testing.T) {
	oldMemoryFile := memory.MemoryFile
	memory.MemoryFile = filepath.Join(t.TempDir(), "nexus_memory.json")
	defer func() { memory.MemoryFile = oldMemoryFile }()
	store := memory.NewLocalMemoryStore()
	oldID, err := store.AddWithConsent("Aethel soll dich duzen.", "preference", "operator", true, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	newID, err := store.AddWithConsent("Aethel soll dich siezen.", "preference", "operator", true, nil, oldID)
	if err != nil {
		t.Fatal(err)
	}
	memories := store.GetAll()
	if len(memories) != 1 || memories[0].ID != newID {
		t.Fatalf("superseded memory remained active: %+v", memories)
	}
	_, why, ok := store.Explain(newID)
	if !ok || !strings.Contains(why, "supersedes "+oldID) {
		t.Fatalf("memory explanation lost conflict provenance: %q", why)
	}
}

func TestMemorySearchUsesMetadata(t *testing.T) {
	oldMemoryFile := memory.MemoryFile
	memory.MemoryFile = filepath.Join(t.TempDir(), "nexus_memory_search.json")
	defer func() { memory.MemoryFile = oldMemoryFile }()
	store := memory.NewLocalMemoryStore()
	if _, err := store.AddWithConsent("Projekt Aethel Architektur", "project", "operator", true, nil, ""); err != nil {
		t.Fatal(err)
	}
	decisionID, err := store.AddWithConsent("Entscheidung: Aethel verwendet einen Policy Gate vor Tools", "decision", "operator", true, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	results := store.Search("Policy für Tools")
	if len(results) == 0 || results[0].ID != decisionID {
		t.Fatalf("metadata-aware recall did not rank the current decision first: %+v", results)
	}
}

func TestAuditChainDetectsTampering(t *testing.T) {
	tmpDir := t.TempDir()
	auditPath := filepath.Join(tmpDir, "security_audit.json")
	audit := security.NewAuditLogger(auditPath)

	if err := audit.ValidateChain(); err != nil {
		t.Fatalf("empty chain validation failed: %v", err)
	}

	if _, err := audit.Log("aethel", "sys_exec_cmd", "system.exec", security.RiskHigh, "", "requested_approval", "test", `{"command":"go"}`); err != nil {
		t.Fatalf("audit log 1 failed: %v", err)
	}
	if _, err := audit.Log("aethel", "fs_read_file", "fs.read", security.RiskLow, "", "allowed", "test", `{"path":"x"}`); err != nil {
		t.Fatalf("audit log 2 failed: %v", err)
	}
	if err := audit.ValidateChain(); err != nil {
		t.Fatalf("valid chain rejected: %v", err)
	}

	data, sealed, err := security.ReadSealedFile(auditPath)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	if !sealed {
		t.Fatal("audit log was not stored in authenticated sealed form")
	}
	var logs []security.AuditEntry
	if err := json.Unmarshal(data, &logs); err != nil {
		t.Fatalf("parse audit log: %v", err)
	}
	logs[0].Decision = "allowed"
	tampered, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		t.Fatalf("marshal tampered log: %v", err)
	}
	if err := security.WriteSealedFile(auditPath, tampered); err != nil {
		t.Fatalf("write tampered log: %v", err)
	}

	if err := security.NewAuditLogger(auditPath).ValidateChain(); err == nil {
		t.Fatalf("tampered chain was accepted")
	}
}

func TestValidatePathRejectsPrefixSibling(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	if err := os.MkdirAll("vgt_workspace_evil", 0700); err != nil {
		t.Fatalf("mkdir sibling: %v", err)
	}
	if _, err := security.ValidatePathForAccess(filepath.Join(tmpDir, "vgt_workspace_evil", "payload.txt"), security.MountWrite); err == nil {
		t.Fatalf("prefix sibling escaped workspace jail")
	}
}

func TestResourceIDValidation(t *testing.T) {
	valid := []string{"secret_1", "persona-abc", "mem.123"}
	for _, id := range valid {
		if err := security.ValidateResourceID(id); err != nil {
			t.Fatalf("valid id %q rejected: %v", id, err)
		}
	}

	invalid := []string{"", "../secret", "x/y", "-leading", "space id", strings.Repeat("a", 81)}
	for _, id := range invalid {
		if err := security.ValidateResourceID(id); err == nil {
			t.Fatalf("invalid id %q accepted", id)
		}
	}
}

func TestSystemCommandUsesAllowlistAndNeverAcceptsShellSyntax(t *testing.T) {
	skill := &skills.ExecuteCommandSkill{}
	if _, err := skill.Execute([]byte(`{"command":"powershell","args":["-Command","Get-ChildItem"]}`)); err == nil {
		t.Fatal("shell interpreter bypassed command allowlist")
	}
	if _, err := skill.Execute([]byte(`{"command":"git","args":["status; whoami"]}`)); err == nil {
		t.Fatal("shell metacharacter bypassed command validation")
	}
}

func TestMountScopesExpireAndDoNotEscalateReadToWrite(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "notes.txt")
	readOnly := &AppState{mounts: []security.MountGrant{{Path: base, Access: security.MountRead, ExpiresAt: time.Now().Add(time.Hour)}}}
	if !readOnly.MountAllows(target, security.MountRead) {
		t.Fatal("read mount rejected a child path")
	}
	if readOnly.MountAllows(target, security.MountWrite) {
		t.Fatal("read mount escalated to write")
	}
	expired := &AppState{mounts: []security.MountGrant{{Path: base, Access: security.MountWrite, ExpiresAt: time.Now().Add(-time.Minute)}}}
	if expired.MountAllows(target, security.MountWrite) {
		t.Fatal("expired mount remained active")
	}
}
