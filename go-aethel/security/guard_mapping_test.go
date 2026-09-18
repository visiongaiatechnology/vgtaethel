// STATUS: DIAMANT VGT SUPREME
package security

import (
	"os"
	"testing"
)

func TestSecurityGuardMapsReplaceAndWeatherCapabilities(t *testing.T) {
	guard := NewSecurityGuard()
	if report := guard.Scan("fs_replace_file_content", `{"path":"sphere_document.html","start_line":1,"end_line":1,"target_content":"a","replacement_content":"b"}`); report.Capability != CapFsWrite {
		t.Fatalf("replace capability = %q", report.Capability)
	}
	if report := guard.Scan("weather_lookup", `{"city":"Köln"}`); report.Capability != CapWeatherRead || report.RiskLevel != RiskLow {
		t.Fatalf("weather report = %+v", report)
	}
}

func TestSecurityGuardMapsMarketCapability(t *testing.T) {
	guard := NewSecurityGuard()
	report := guard.Scan("market_lookup", `{"symbols":["BTC","GOLD"]}`)
	if report.Capability != CapMarketRead || report.RiskLevel != RiskLow {
		t.Fatalf("market report = %+v", report)
	}
}

func TestSecurityGuardMapsMailCapabilities(t *testing.T) {
	guard := NewSecurityGuard()
	if report := guard.Scan("mail_list_messages", `{"limit":5}`); report.Capability != CapSecretUse || report.RiskLevel != RiskModerate {
		t.Fatalf("mail read report = %+v", report)
	}
	if report := guard.Scan("mail_send_message", `{"to":["user@example.com"],"subject":"Test","body":"Body"}`); report.Capability != CapMessagingSend || report.RiskLevel != RiskCritical {
		t.Fatalf("mail send report = %+v", report)
	}
}

func TestSecurityGuardMapsInternalNavigationCapability(t *testing.T) {
	guard := NewSecurityGuard()
	report := guard.Scan("navigate_ui", `{"view":"global_watch"}`)
	if report.Capability != CapUINavigation || report.RiskLevel != RiskSafe {
		t.Fatalf("navigation report = %+v", report)
	}
}

func TestSecurityGuardMapsIntelligenceCapabilities(t *testing.T) {
	guard := NewSecurityGuard()
	if report := guard.Scan("global_watch_nexus_context", `{}`); report.Capability != CapIntelRead || report.RiskLevel != RiskLow {
		t.Fatalf("nexus context report = %+v", report)
	}
	if report := guard.Scan("intelligence_create_case", `{"title":"Case","purpose":"public safety analysis"}`); report.Capability != CapIntelWrite || report.RiskLevel != RiskModerate {
		t.Fatalf("case report = %+v", report)
	}
	if report := guard.Scan("osint_add_custom_feed", `{"name":"Source","url":"https://example.com/feed"}`); report.Capability != CapIntelSources || report.RiskLevel != RiskModerate {
		t.Fatalf("feed report = %+v", report)
	}
	if report := guard.Scan("global_watch_focus", `{"lat":52.52,"lon":13.405,"zoom":1.2}`); report.Capability != CapIntelWrite || report.RiskLevel != RiskModerate {
		t.Fatalf("focus report = %+v", report)
	}
	if report := guard.Scan("global_watch_toggle_layer", `{"layer":"cities","enable":false}`); report.Capability != CapIntelWrite || report.RiskLevel != RiskModerate {
		t.Fatalf("layer report = %+v", report)
	}
	if report := guard.Scan("global_watch_schedule_briefing", `{"enabled":true,"interval_minutes":60}`); report.Capability != CapIntelWrite || report.RiskLevel != RiskModerate {
		t.Fatalf("schedule report = %+v", report)
	}
}

func TestPolicyEnginePermissionModes(t *testing.T) {
	guard := NewSecurityGuard()
	tmpDir := t.TempDir()
	leases := NewLeaseManager(tmpDir + "/leases.json")
	audit := NewAuditLogger(tmpDir + "/audit.json")
	engine := NewPolicyEngine(guard, leases, audit)

	// Default mode is interactive
	if mode := engine.GetMode(); mode != PermissionModeInteractive {
		t.Fatalf("expected default mode interactive, got %v", mode)
	}

	// In interactive mode, moderate/high risk actions require approval
	allowed, decision, report := engine.Evaluate("fs_write_file", `{"path":"test.txt","content":"hello"}`, false)
	if allowed || decision != "needs_approval" {
		t.Fatalf("expected needs_approval in interactive mode, got allowed=%v, decision=%q, report=%+v", allowed, decision, report)
	}

	// Switch to full_access mode ("Vollzugriff")
	engine.SetMode(PermissionModeFullAccess)
	if mode := engine.GetMode(); mode != PermissionModeFullAccess {
		t.Fatalf("expected mode full_access, got %v", mode)
	}

	// In full_access mode, moderate/high risk actions are auto-allowed
	allowed, decision, report = engine.Evaluate("fs_write_file", `{"path":"test.txt","content":"hello"}`, false)
	if !allowed || decision != "" {
		t.Fatalf("expected allowed in full_access mode, got allowed=%v, decision=%q, report=%+v", allowed, decision, report)
	}

	// Critical/Forbidden action (e.g. rm -rf) must STILL be blocked even in full_access mode!
	allowed, decision, report = engine.Evaluate("sys_exec_cmd", `{"command":"rm","args":["-rf","/"]}`, false)
	if allowed || decision != "blocked" {
		t.Fatalf("expected forbidden action to be blocked in full_access mode, got allowed=%v, decision=%q, report=%+v", allowed, decision, report)
	}
}

func TestValidatePathForAccessActiveWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	canonicalTmp, err := CanonicalDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	subFile := canonicalTmp + "/example.txt"
	if err := os.WriteFile(subFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Configure active workspace
	InitState(nil, func() string { return canonicalTmp })
	defer InitState(nil)

	// Test resolving relative path inside active workspace
	resolved, err := ValidatePathForAccess("example.txt", MountRead)
	if err != nil {
		t.Fatalf("expected relative path to resolve inside active workspace, got error: %v", err)
	}
	if !IsPathInside(canonicalTmp, resolved) {
		t.Fatalf("expected resolved path %q to be inside active workspace %q", resolved, canonicalTmp)
	}

	// Test write access to a new relative file inside active workspace
	writeResolved, err := ValidatePathForAccess("new_output.txt", MountWrite)
	if err != nil {
		t.Fatalf("expected write path inside active workspace to be allowed, got: %v", err)
	}
	if !IsPathInside(canonicalTmp, writeResolved) {
		t.Fatalf("expected write path %q to be inside active workspace %q", writeResolved, canonicalTmp)
	}
}


