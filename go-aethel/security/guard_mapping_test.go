package security

import "testing"

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
