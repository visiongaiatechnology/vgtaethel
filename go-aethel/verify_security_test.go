package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== STARTING VGT AETHEL SECURITY INTERNALS VERIFICATION ===")
	
	// Create temporary workspace for testing
	tmpDir, err := ioutil.TempDir("", "vgt_test_workspace")
	if err != nil {
		fmt.Printf("FAIL: Cannot create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	testLeasesFile := filepath.Join(tmpDir, "active_leases.json")
	testAuditFile := filepath.Join(tmpDir, "security_audit.json")

	// 1. Initialize Subsystems
	guard := NewSecurityGuard()
	leases := NewLeaseManager(testLeasesFile)
	audit := NewAuditLogger(testAuditFile)
	
	err = audit.ValidateChain()
	if err != nil {
		fmt.Printf("FAIL: Empty audit log chain validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[PASS] Empty audit chain validated successfully.")

	policy := NewPolicyEngine(guard, leases, audit)

	// 2. Test Forbidden / Destructive Commands Blockage
	fmt.Println("\n--- Test Case 1: Hard Block on Forbidden Shell commands ---")
	forbiddenArgs := `{"command":"rm -rf /usr/bin"}`
	allowed, decision, report := policy.Evaluate("sys_exec_cmd", forbiddenArgs, false)
	
	if allowed {
		fmt.Printf("FAIL: Forbidden command rm -rf was allowed.\n")
		os.Exit(1)
	}
	if decision != "blocked" {
		fmt.Printf("FAIL: Expected decision 'blocked', got '%s'\n", decision)
		os.Exit(1)
	}
	if report.RiskLevel != RiskForbidden {
		fmt.Printf("FAIL: Expected RiskLevel 'Forbidden', got '%s'\n", report.RiskLevel)
		os.Exit(1)
	}
	fmt.Println("[PASS] rm -rf was hard-blocked as Forbidden.")

	// Test Forbidden Keystroke
	forbiddenKeys := `{"action":"press","keys":"alt+f4"}`
	allowed, decision, report = policy.Evaluate("gui_control", forbiddenKeys, false)
	if allowed || decision != "blocked" || report.RiskLevel != RiskForbidden {
		fmt.Printf("FAIL: Forbidden keystroke alt+f4 was not blocked. Decision: %s, Risk: %s\n", decision, report.RiskLevel)
		os.Exit(1)
	}
	fmt.Println("[PASS] alt+f4 keystroke was hard-blocked as Forbidden.")

	// 3. Test Lease Lifecycle (Needs Approval -> Approve Lease -> Autonomously execute -> Expire -> Needs Approval)
	fmt.Println("\n--- Test Case 2: Lease Lifecycle Transitions ---")
	execArgs := `{"command":"go version"}`
	
	// Transition A: Needs Approval when no lease active
	allowed, decision, report = policy.Evaluate("sys_exec_cmd", execArgs, false)
	if allowed || decision != "needs_approval" {
		fmt.Printf("FAIL: Expected 'needs_approval' for shell cmd without active lease, got allowed=%t, decision=%s\n", allowed, decision)
		os.Exit(1)
	}
	fmt.Println("[PASS] Transition A: Tool blocked, needs_approval.")

	// Transition B: Add lease
	leaseID, err := leases.AddLease(CapSysExec, 15, Scope{})
	if err != nil {
		fmt.Printf("FAIL: Could not add lease: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Lease added successfully: %s\n", leaseID)

	// Transition C: Evaluate again (must be allowed automatically!)
	allowed, decision, report = policy.Evaluate("sys_exec_cmd", execArgs, false)
	if !allowed || decision != leaseID {
		fmt.Printf("FAIL: Expected tool to execute automatically with lease, got allowed=%t, decision=%s\n", allowed, decision)
		os.Exit(1)
	}
	fmt.Println("[PASS] Transition C: Tool allowed automatically under lease.")

	// Transition D: Revoke lease
	revoked := leases.RevokeLease(leaseID)
	if !revoked {
		fmt.Println("FAIL: Lease revocation failed.")
		os.Exit(1)
	}
	fmt.Println("Lease revoked successfully.")

	// Transition E: Evaluate again (must require approval again!)
	allowed, decision, report = policy.Evaluate("sys_exec_cmd", execArgs, false)
	if allowed || decision != "needs_approval" {
		fmt.Printf("FAIL: Expected 'needs_approval' after lease revocation, got allowed=%t, decision=%s\n", allowed, decision)
		os.Exit(1)
	}
	fmt.Println("[PASS] Transition E: Tool blocked again after lease revocation.")

	// 4. Test Audit Log Hash Chain Verification & Tampering Check
	fmt.Println("\n--- Test Case 3: Cryptographic Audit Hash Chain ---")
	
	// Read current log
	logData, err := ioutil.ReadFile(testAuditFile)
	if err != nil {
		fmt.Printf("FAIL: Cannot read audit log: %v\n", err)
		os.Exit(1)
	}
	
	var logs []AuditEntry
	if err := json.Unmarshal(logData, &logs); err != nil {
		fmt.Printf("FAIL: Cannot parse audit log JSON: %v\n", err)
		os.Exit(1)
	}
	
	if len(logs) < 5 {
		fmt.Printf("FAIL: Expected at least 5 logs in chain, got %d\n", len(logs))
		os.Exit(1)
	}
	fmt.Printf("Total logged activities in chain: %d\n", len(logs))

	// Verify chain integrity
	err = audit.ValidateChain()
	if err != nil {
		fmt.Printf("FAIL: Hash chain verification failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[PASS] Integrity check succeeded on untampered log chain.")

	// Introduce tampering: manually modify an intermediate entry's decision in the JSON file
	fmt.Println("Modifying log entry to simulate tampering...")
	logs[2].Decision = "allowed" // Original was probably blocked or requested_approval
	
	tamperedData, _ := json.MarshalIndent(logs, "", "  ")
	err = ioutil.WriteFile(testAuditFile, tamperedData, 0644)
	if err != nil {
		fmt.Printf("FAIL: Cannot write tampered audit file: %v\n", err)
		os.Exit(1)
	}

	// Verify chain validation fails now!
	auditTampered := NewAuditLogger(testAuditFile)
	err = auditTampered.ValidateChain()
	if err == nil {
		fmt.Println("FAIL: Audit chain validation passed on tampered data!")
		os.Exit(1)
	}
	fmt.Printf("[PASS] Tamper detection succeeded! Error caught: %v\n", err)

	fmt.Println("\n=======================================================")
	fmt.Println("  ALL SECURITY AND AUDIT LOG TESTS PASSED SUCCESSFULLY! ")
	fmt.Println("=======================================================")
}
