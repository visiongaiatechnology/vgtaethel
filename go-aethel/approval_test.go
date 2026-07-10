package main

import (
	"path/filepath"
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
