package agent

import (
	"fmt"
	"strings"
)

// RuntimeCapability is Aethel's compact, deterministic self-model. Only
// capabilities relevant to the current objective are injected into the
// orchestrator prompt, keeping the prompt stable and token-efficient.
type RuntimeCapability struct {
	ID          string
	Tool        string
	Description string
	Intents     []string
}

var runtimeCapabilities = []RuntimeCapability{
	{ID: "weather", Tool: "weather_lookup", Description: "Current weather for a named city or location.", Intents: []string{"wetter", "weather", "temperatur", "forecast", "vorhersage"}},
	{ID: "personal_operations", Tool: "personal_operations", Description: "Set persistent alarms and open Personal Operations weather or daily-plan popups.", Intents: []string{"weck mich", "wecker", "alarm", "tagesplan", "tagesübersicht", "morgenroutine", "wetter popup"}},
	{ID: "market", Tool: "market_lookup", Description: "Current BTC, ETH, SOL and GOLD/PAXG market values.", Intents: []string{"bitcoin", "btc", "ethereum", "eth", "solana", "goldpreis", "gold price", "kurs"}},
	{ID: "mail_read", Tool: "mail_list_messages", Description: "Read message summaries from an operator-selected IMAP folder.", Intents: []string{"postfach", "inbox", "e-mail", "email", "mail", "ungelesen"}},
	{ID: "mail_detail", Tool: "mail_read_message", Description: "Read a specific message, spam assessment, deadlines and matching reply policies.", Intents: []string{"mail lesen", "email lesen", "e-mail lesen", "mail antwort", "email antwort", "spam prÃ¼fen", "spam analyse", "zahlungsaufforderung"}},
	{ID: "mail_manage", Tool: "mail_manage", Description: "Create an IMAP folder or move a message after an explicit instruction.", Intents: []string{"mail sortieren", "email sortieren", "ordner erstellen", "spam ordner", "mail verschieben", "email verschieben"}},
	{ID: "mail_send", Tool: "mail_send_message", Description: "Send one operator-approved message through the configured secure SMTP account.", Intents: []string{"e-mail", "email", "mail", "send email", "send mail"}},
	{ID: "ui_navigation", Tool: "navigate_ui", Description: "Switch Aethel's own view without desktop GUI automation.", Intents: []string{"tab", "ansicht", "bereich", "global watch", "live-globus", "live globus", "sphere", "sphäre", "personal core", "e-mail command", "mail tab", "run center", "agent tracker", "settings", "einstellungen"}},
	{ID: "web_research", Tool: "web_browser", Description: "Search and read public web sources in Aethel's internal browser.", Intents: []string{"recherch", "websuche", "web search", "internet", "website", "webseite", "quelle finden"}},
	{ID: "writer", Tool: "sphere_write_document", Description: "Create or replace visible content in the Sphere Writer.", Intents: []string{"writer", "gedicht", "geschichte", "dokument", "artikel schreiben"}},
	{ID: "global_watch", Tool: "global_watch_nexus_context", Description: "Read Aethel's unified current intelligence and Global Watch state.", Intents: []string{"weltlage", "nachrichtenlage", "global watch", "osint", "lagebriefing", "geopolit", "aktuelle news"}},
	{ID: "natural_hazards", Tool: "global_watch_natural_hazards_context", Description: "Read the isolated earthquake and volcano context after an explicit operator request.", Intents: []string{"erdbeben", "earthquake", "quake", "seismisch", "seismic", "vulkan", "volcano", "eruption", "naturgefahr", "natural hazard"}},
	{ID: "agent_plan", Tool: "task_set_checklist", Description: "Create a visible execution checklist for a multi-step objective.", Intents: []string{"agentenplan", "agent plan", "agenten-team", "agent team", "aktionsplan", "checkliste"}},
	{ID: "filesystem", Tool: "fs_list_dir", Description: "Inspect folders and source workspaces through the path jail.", Intents: []string{"ordner", "verzeichnis", "workspace", "dateien", "code kartografie", "code cartography"}},
}

func capabilitiesForObjective(objective string, sphereActive bool) []RuntimeCapability {
	text := normalizeAgentObjective(objective)
	selected := make([]RuntimeCapability, 0, 6)
	seen := make(map[string]bool)
	for _, capability := range runtimeCapabilities {
		if containsAny(text, capability.Intents...) && !seen[capability.Tool] {
			if capability.ID == "writer" && !sphereActive {
				continue
			}
			if capability.ID == "ui_navigation" && !requestsUINavigation(text) {
				continue
			}
			if capability.ID == "global_watch" && !objectiveRequestsGlobalWatch(text) {
				continue
			}
			if capability.ID == "natural_hazards" && !objectiveRequestsNaturalHazards(text) {
				continue
			}
			if capability.ID == "mail_read" && !objectiveRequestsMailRead(text) {
				continue
			}
			if capability.ID == "mail_detail" && !objectiveRequestsMailRead(text) {
				continue
			}
			if capability.ID == "mail_send" && !objectiveRequestsMailSend(text) {
				continue
			}
			seen[capability.Tool] = true
			selected = append(selected, capability)
		}
	}
	return selected
}

func capabilityEffects(objective string, sphereActive bool) []string {
	capabilities := capabilitiesForObjective(objective, sphereActive)
	effects := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		switch capability.ID {
		case "weather", "personal_operations", "market", "mail_read", "mail_detail", "mail_manage", "mail_send", "ui_navigation", "web_research", "agent_plan", "writer", "global_watch", "natural_hazards":
			effects = append(effects, capability.Tool)
		}
	}
	return uniqueStrings(effects)
}

func runtimeCapabilityContract(objective string, sphereActive bool, profileID ...string) string {
	capabilities := capabilitiesForObjective(objective, sphereActive)
	if len(profileID) > 0 && profileID[0] == "global_watch_operator" {
		found := false
		for _, capability := range capabilities {
			found = found || capability.ID == "global_watch"
		}
		if !found {
			for _, capability := range runtimeCapabilities {
				if capability.ID == "global_watch" {
					capabilities = append(capabilities, capability)
					break
				}
			}
		}
	}
	if len(capabilities) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("\n\nAETHEL RUNTIME CAPABILITY ROUTES FOR THIS OBJECTIVE:\n")
	for _, capability := range capabilities {
		builder.WriteString(fmt.Sprintf("- %s => %s: %s\n", capability.ID, capability.Tool, capability.Description))
	}
	builder.WriteString("Use native tool_calls. A route is complete only after its matching tool result is verified.")
	return builder.String()
}

// toolAllowlistForRun minimizes schema exposure per inference. Policy remains
// the authoritative execution boundary; this list reduces provider confusion,
// prompt size and accidental calls to unrelated capabilities.
func toolAllowlistForRun(run AgentRun) []string {
	tools := append([]string(nil), requiredExecutionEffects(run)...)
	for _, capability := range capabilitiesForObjective(run.Objective, run.SphereActive) {
		tools = append(tools, capability.Tool)
	}
	switch run.ProfileID {
	case "global_watch_operator":
		tools = append(tools,
			"global_watch_nexus_context", "global_watch_focus", "global_watch_focus_region",
			"global_watch_time_window", "global_watch_open_report", "global_watch_toggle_layer",
			"intelligence_region_status", "intelligence_explain_score", "intelligence_generate_brief",
			"intelligence_source_health", "intelligence_recent_changes", "fs_read_file",
			"sphere_write_document", "market_lookup", "sphere_market_overview", "weather_lookup")
	case "collector":
		tools = append(tools,
			"intelligence_connector_fetch", "intelligence_source_health",
			"global_watch_nexus_context", "fs_read_file")
	case "analyst":
		tools = append(tools,
			"global_watch_nexus_context", "intelligence_region_status", "intelligence_explain_score",
			"intelligence_region_compare", "intelligence_generate_brief", "intelligence_source_health",
			"intelligence_global_status", "intelligence_recent_changes", "intelligence_market_summary",
			"intelligence_infrastructure_summary", "intelligence_conflict_summary", "intelligence_cyber_summary")
	case "case_worker":
		tools = append(tools,
			"osint_case_create", "osint_evidence_capture", "osint_entity_propose",
			"osint_relation_propose", "osint_timeline_generate", "osint_report_generate",
			"intelligence_set_assessment_status", "global_watch_nexus_context")
	case "operator":
		tools = append(tools,
			"global_watch_nexus_context", "intelligence_connector_fetch", "osint_case_create",
			"osint_evidence_capture", "osint_entity_propose", "osint_relation_propose",
			"mail_list_messages", "mail_read_message", "mail_manage", "mail_send_message",
			"web_browser", "vision_context", "gui_control", "gui_window_control",
			"fs_read_file", "fs_write_file", "fs_replace_file_content")
	case "developer":
		tools = append(tools,
			"fs_list_dir", "fs_read_file", "fs_write_file", "fs_replace_file_content",
			"fs_mount_folder", "code_cartography", "sys_exec_cmd", "task_set_checklist",
			"market_lookup", "sphere_market_overview")
	case "researcher":
		tools = append(tools,
			"web_browser", "fs_read_file", "market_lookup", "sphere_market_overview", "weather_lookup")
	case "browser_operator":
		tools = append(tools, "web_browser", "vision_context", "gui_control", "gui_window_control", "market_lookup", "sphere_market_overview")
	case "sphere_workspace":
		tools = append(tools, "sphere_write_document", "web_browser", "weather_lookup", "market_lookup", "sphere_market_overview", "media_control", "fs_read_file", "fs_write_file")
	case "personal_assistant":
		tools = append(tools, "fs_read_file", "fs_write_file", "fs_replace_file_content", "sphere_write_document", "market_lookup", "sphere_market_overview", "weather_lookup", "personal_operations", "mail_list_messages", "mail_read_message", "mail_manage", "mail_send_message", "media_control")
	}
	return uniqueStrings(tools)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func objectiveNeedsDeveloperProfile(objective string) bool {
	text := normalizeAgentObjective(objective)
	return containsAny(text, "datei", "ordner", "verzeichnis", "workspace", "pfad", "code", "build", "programm", "terminal", "install", "refactor")
}
