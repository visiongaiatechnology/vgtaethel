package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLeaseAuthorityStoreIsSealedAndRejectsPlaintextForgery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "leases.json")
	manager := NewLeaseManager(path)
	lease := PermissionLease{
		LeaseID:    "lease_real",
		Capability: CapSysExec,
		CreatedAt:  time.Now().UTC(),
		ExpiresAt:  time.Now().UTC().Add(time.Minute),
	}
	if err := manager.AddLease(lease); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), sealedStorePrefix) || strings.Contains(string(data), lease.LeaseID) {
		t.Fatal("lease authority was not stored as an opaque authenticated payload")
	}
	if allowed, _ := NewLeaseManager(path).CheckLease(CapSysExec, "", "sys_exec_cmd"); !allowed {
		t.Fatal("valid sealed lease did not survive reload")
	}

	forged := `[{"lease_id":"lease_forged","capability":"system.exec","expires_at":"2099-01-01T00:00:00Z"}]`
	if err := os.WriteFile(path, []byte(forged), 0600); err != nil {
		t.Fatal(err)
	}
	if allowed, _ := NewLeaseManager(path).CheckLease(CapSysExec, "", "sys_exec_cmd"); allowed {
		t.Fatal("plaintext lease forgery was accepted")
	}
}

func TestAuditStoreSealsArgumentsAndReloadsVerifiedChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	logger := NewAuditLogger(path)
	secretArgs := `{"content":"private operator text"}`
	if _, err := logger.Log("aethel", "fs_write_file", string(CapFsWrite), RiskHigh, "", "requested_approval", "test", secretArgs); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), sealedStorePrefix) || strings.Contains(string(data), "private operator text") {
		t.Fatal("audit store leaked raw tool arguments")
	}
	reloaded := NewAuditLogger(path)
	if tampered, _ := reloaded.Status(); tampered {
		t.Fatal("valid sealed audit chain failed reload verification")
	}
	logs := reloaded.GetLogs()
	if len(logs) != 1 || !strings.HasPrefix(logs[0].InputSummary, "sha256:") {
		t.Fatalf("unexpected audit evidence: %+v", logs)
	}
}
