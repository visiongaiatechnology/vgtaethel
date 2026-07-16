package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApprovalGrantIsArgumentBoundAndSingleUse(t *testing.T) {
	manager := NewApprovalManager(filepath.Join(t.TempDir(), "approvals.json"))
	grant, token, err := manager.Issue("fs_write_file", `{"path":"notes.txt","content":"A"}`, CapFsWrite, "run_test")
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	if grant.TokenHash == token || token == "" {
		t.Fatal("plaintext approval token was persisted")
	}
	if err := manager.Consume(token, "fs_write_file", `{"path":"notes.txt","content":"B"}`, CapFsWrite, "run_test"); err == nil {
		t.Fatal("approval token accepted changed arguments")
	}
	if err := manager.Consume(token, "fs_write_file", `{"path":"notes.txt","content":"A"}`, CapFsWrite, "wrong_run"); err == nil {
		t.Fatal("approval token accepted another run")
	}
	if err := manager.Consume(token, "fs_write_file", `{"path":"notes.txt","content":"A"}`, CapFsWrite, "run_test"); err != nil {
		t.Fatalf("valid approval token rejected: %v", err)
	}
	if err := manager.Consume(token, "fs_write_file", `{"path":"notes.txt","content":"A"}`, CapFsWrite, "run_test"); err == nil {
		t.Fatal("approval token was replayed")
	}
}

func TestApprovalAuthorityStoreRejectsPlaintextForgery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "approvals.json")
	manager := NewApprovalManager(path)
	_, token, err := manager.Issue("fs_write_file", `{"path":"notes.txt","content":"A"}`, CapFsWrite, "run_test")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), sealedStorePrefix) || strings.Contains(string(data), "fs_write_file") {
		t.Fatal("approval authority was not stored as an opaque authenticated payload")
	}
	forged := `[{"id":"forged","token_hash":"` + digestApprovalArgs(token) + `","tool_name":"fs_write_file","args_hash":"x"}]`
	if err := os.WriteFile(path, []byte(forged), 0600); err != nil {
		t.Fatal(err)
	}
	reloaded := NewApprovalManager(path)
	if err := reloaded.Consume(token, "fs_write_file", `{"path":"notes.txt","content":"A"}`, CapFsWrite, "run_test"); err == nil {
		t.Fatal("plaintext approval forgery was accepted")
	}
}
