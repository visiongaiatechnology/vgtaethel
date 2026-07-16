package agent

import (
	"encoding/json"
	"math"
	"testing"

	"go-aethel/skills"
)

func TestParseAgentSSEReassemblesToolCallsAndUsage(t *testing.T) {
	stream := "data: Analyse[VGT_NL]läuft\n\n" +
		"data: [[THINKING]]:Prüfe[VGT_NL]Plan\n\n" +
		"data: [[TOOL_DELTA]]:[{\"index\":0,\"id\":\"call_1\",\"function\":{\"name\":\"fs_read_file\",\"arguments\":\"{\\\"path\\\":\"}}]\n\n" +
		"data: [[TOOL_DELTA]]:[{\"index\":0,\"function\":{\"arguments\":\"\\\"notes.txt\\\"}\"}}]\n\n" +
		"data: [[USAGE]]:{\"prompt_tokens\":120,\"completion_tokens\":30,\"prompt_cache_hit_tokens\":20}\n\n" +
		"data: [[TOOL_COMMIT]]\n\n"
	result := parseAgentSSE(stream)
	if result.Err != nil {
		t.Fatalf("parse failed: %v", result.Err)
	}
	if result.Text != "Analyse\nläuft" || result.Thinking != "Prüfe\nPlan" {
		t.Fatalf("stream text mismatch: text=%q thinking=%q", result.Text, result.Thinking)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "fs_read_file" || string(result.ToolCalls[0].Arguments) != `{"path":"notes.txt"}` {
		t.Fatalf("tool reconstruction mismatch: %+v", result.ToolCalls)
	}
	if result.InputTokens != 120 || result.OutputTokens != 30 || result.CachedTokens != 20 {
		t.Fatalf("usage mismatch: %+v", result)
	}
}

func TestParseAgentSSERejectsMalformedToolArguments(t *testing.T) {
	result := parseAgentSSE("data: [[TOOL_DELTA]]:[{\"index\":0,\"id\":\"call_1\",\"function\":{\"name\":\"fs_read_file\",\"arguments\":\"{bad\"}}]\n\n")
	if result.Err == nil {
		t.Fatal("malformed streamed tool arguments were accepted")
	}
}

func TestParseAgentSSEPreservesTokenWhitespace(t *testing.T) {
	result := parseAgentSSE("data: Ich\n\ndata:  bin\n\ndata:  bereit.\n\n")
	if result.Err != nil || result.Text != "Ich bin bereit." {
		t.Fatalf("token whitespace was lost: %q err=%v", result.Text, result.Err)
	}
}

func TestShouldEnableAgentToolsOnlyForOperationalRequests(t *testing.T) {
	tests := []struct {
		objective string
		want      bool
	}{
		{"Was geht ab :D", false},
		{"Erstelle ein aktuelles Global Watch Lagebild", true},
		{"Wie ist die aktuelle Nachrichtenlage weltweit?", true},
		{"Prüfe die aktuelle Weltlage", true},
		{"Öffne den Global Watch Globus", true},
		{"Gibt es einen OSINT Alert fÃ¼r Berlin?", true},
		{"Erkläre mir kurz Go Interfaces", false},
		{"Schau dir den Ordner C:\\Projects an", true},
		{"Erstelle einen Bericht über diese Datei", true},
	}
	for _, test := range tests {
		if got := shouldEnableAgentTools(test.objective); got != test.want {
			t.Errorf("shouldEnableAgentTools(%q) = %t, want %t", test.objective, got, test.want)
		}
	}
}

func TestResolveChatAgentProfileIsServerAuthoritative(t *testing.T) {
	if got := ResolveChatAgentProfile(ChatAgentStartRequest{SphereActive: true, LiveOperatorActive: true, ProfileID: "developer"}); got != "sphere_workspace" {
		t.Fatalf("sphere request resolved to %q", got)
	}
	if got := ResolveChatAgentProfile(ChatAgentStartRequest{LiveOperatorActive: true, ProfileID: "developer"}); got != "browser_operator" {
		t.Fatalf("live operator request resolved to %q", got)
	}
	if got := ResolveChatAgentProfile(ChatAgentStartRequest{ProfileID: "browser_operator", Objective: "Wie ist das Wetter in New York?"}); got != "aethel_runtime" {
		t.Fatalf("ordinary chat request resolved to %q", got)
	}
	if got := ResolveChatAgentProfile(ChatAgentStartRequest{Objective: "Prüfe den Code in diesem Ordner"}); got != "developer" {
		t.Fatalf("developer objective resolved to %q", got)
	}
}

func TestGlobalWatchIntentSeparatesNavigationFromIntelligence(t *testing.T) {
	navigation := AgentRun{Objective: "Öffne den Tab Global Watch", ProfileID: "aethel_runtime"}
	navigationEffects := requiredExecutionEffects(navigation)
	if !containsString(navigationEffects, "navigate_ui") {
		t.Fatalf("explicit view transition missing navigate_ui: %+v", navigationEffects)
	}
	if containsString(navigationEffects, "global_watch_nexus_context") {
		t.Fatalf("pure navigation unexpectedly forced intelligence retrieval: %+v", navigationEffects)
	}

	query := AgentRun{Objective: "Dann nutze bitte die neuesten Infos über Global Watch", ProfileID: "global_watch_operator"}
	queryEffects := requiredExecutionEffects(query)
	if !containsString(queryEffects, "global_watch_nexus_context") {
		t.Fatalf("Global Watch query missing unified context: %+v", queryEffects)
	}
	if containsString(queryEffects, "navigate_ui") {
		t.Fatalf("Global Watch query was misrouted as navigation: %+v", queryEffects)
	}
}

func TestGlobalWatchFollowUpKeepsConversationDomain(t *testing.T) {
	previous, err := json.Marshal(map[string]string{"role": "user", "content": "Nutze die neuesten Infos über Global Watch"})
	if err != nil {
		t.Fatal(err)
	}
	current, err := json.Marshal(map[string]string{"role": "user", "content": "Also würdest du sagen, derzeit ist alles im grünen Bereich?"})
	if err != nil {
		t.Fatal(err)
	}
	request := ChatAgentStartRequest{
		Objective: "Also würdest du sagen, derzeit ist alles im grünen Bereich?",
		Messages:  []json.RawMessage{previous, current},
	}
	if profile := ResolveChatAgentProfile(request); profile != "global_watch_operator" {
		t.Fatalf("contextual Global Watch follow-up resolved to %q", profile)
	}
	if !RequiresChatOrchestrator(request) {
		t.Fatal("contextual Global Watch follow-up must require the tool-capable orchestrator")
	}
}

func TestGlobalWatchToolSchemaIsNarrowedPerRun(t *testing.T) {
	run := AgentRun{Objective: "Nutze die neuesten Infos über Global Watch", ProfileID: "global_watch_operator"}
	tools := toolAllowlistForRun(run)
	if !containsString(tools, "global_watch_nexus_context") {
		t.Fatalf("nexus tool missing from narrowed schema: %+v", tools)
	}
	for _, forbidden := range []string{"navigate_ui", "gui_control", "sys_exec_cmd", "fs_write_file"} {
		if containsString(tools, forbidden) {
			t.Fatalf("unrelated tool %q leaked into Global Watch schema: %+v", forbidden, tools)
		}
	}
}

func TestMailIntentSelectsOnlyReadOrSendRoute(t *testing.T) {
	read := requiredExecutionEffects(AgentRun{Objective: "Prüfe bitte mein Postfach auf neue E-Mails", ProfileID: "aethel_runtime"})
	if !containsString(read, "mail_list_messages") || containsString(read, "mail_send_message") {
		t.Fatalf("mail read routing invalid: %+v", read)
	}
	send := requiredExecutionEffects(AgentRun{Objective: "Sende eine E-Mail an user@example.com", ProfileID: "aethel_runtime"})
	if !containsString(send, "mail_send_message") || containsString(send, "mail_list_messages") {
		t.Fatalf("mail send routing invalid: %+v", send)
	}
	draft := requiredExecutionEffects(AgentRun{Objective: "Formuliere mir eine freundliche E-Mail an den Vermieter", ProfileID: "aethel_runtime"})
	if containsString(draft, "mail_list_messages") || containsString(draft, "mail_send_message") {
		t.Fatalf("mail drafting must not access or send mail: %+v", draft)
	}
}

func TestRequiresChatOrchestratorKeepsSimpleLocalChatDirect(t *testing.T) {
	if RequiresChatOrchestrator(ChatAgentStartRequest{Objective: "Hallo, wie geht es dir?"}) {
		t.Fatal("simple chat must not require a tool-capable orchestrator")
	}
	if !RequiresChatOrchestrator(ChatAgentStartRequest{Objective: "Wie ist das Wetter in Köln?"}) {
		t.Fatal("runtime tool objective must require an orchestrator")
	}
}

// TestSphereToolSchemaRetainsGlobalWatchControls verifies GW skill names remain
// registered (filtering lives in handlers.toolDefinitionsForRequest; skill
// names must stay stable for sphere allowlist matching).
func TestSphereToolSchemaRetainsGlobalWatchControls(t *testing.T) {
	registry := skills.NewSkillRegistry()
	registry.Register(&skills.GlobalWatchNexusContextSkill{})
	registry.Register(&skills.GlobalWatchFocusSkill{})
	registry.Register(&skills.GlobalWatchObserveSkill{})
	registry.Register(&skills.GlobalWatchToggleLayerSkill{})
	registry.Register(&skills.GlobalWatchScheduleBriefingSkill{})
	allowed := map[string]bool{}
	for _, raw := range registry.ToToolDefinitions() {
		tool, _ := raw.(map[string]interface{})
		function, _ := tool["function"].(map[string]interface{})
		name, _ := function["name"].(string)
		allowed[name] = true
	}
	for _, name := range []string{"global_watch_nexus_context", "global_watch_focus", "global_watch_observe", "global_watch_toggle_layer", "global_watch_schedule_briefing"} {
		if !allowed[name] {
			t.Fatalf("sphere schema unexpectedly missing Global Watch tool %q", name)
		}
	}
}

func TestPromoteCodeCartographyFallbackAcceptsOnlyExplicitCartographyJSON(t *testing.T) {
	result := agentInferenceResult{Text: "{\"path\": \"C:\\Users\\Masterboard\\Project\", \"max_files\": 100, \"output_path\": \"code_cartography.md\"}\n\n{\"path\": \"code_cartography.md\", \"content\": \"must not be promoted\"}\n\n{\"path\": \"C:\\Users\\Masterboard\\Project\", \"output_path\": \"code_cartography.md\", \"recursive\": true}"}
	promoteCodeCartographyFallback(&result, "Erstelle eine Code Kartografie für diesen Ordner")
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "code_cartography" || result.Text != "" {
		t.Fatalf("cartography fallback did not promote the dedicated call: %+v", result)
	}
	var args skills.CodeCartographyArgs
	if err := json.Unmarshal(result.ToolCalls[0].Arguments, &args); err != nil || args.Path != `C:\Users\Masterboard\Project` || args.MaxFiles != 100 {
		t.Fatalf("cartography fallback arguments invalid: %+v err=%v", args, err)
	}

	nonCartography := agentInferenceResult{Text: `{"path":"C:\Users\Masterboard\Project"}`}
	promoteCodeCartographyFallback(&nonCartography, "Schau dir den Ordner an")
	if len(nonCartography.ToolCalls) != 0 {
		t.Fatal("fallback promoted unrelated assistant JSON")
	}
}

func TestPromoteWebBrowserFallbackRejectsBlankPage(t *testing.T) {
	result := agentInferenceResult{Text: `{"tool_name":"web_browser","args":{"action":"navigate","url":"https://www.youtube.com"}}`}
	promoteWebBrowserFallback(&result, "Öffne YouTube.com")
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "web_browser" {
		t.Fatalf("browser wrapper was not promoted: %+v", result)
	}
	blank := agentInferenceResult{Text: `{"tool_name":"web_browser","args":{"action":"navigate","url":"about:blank"}}`}
	promoteWebBrowserFallback(&blank, "Öffne YouTube.com")
	if len(blank.ToolCalls) != 0 {
		t.Fatal("blank browser navigation was promoted")
	}
	unsafe := agentInferenceResult{Text: `{"tool_name":"web_browser","args":{"action":"navigate","url":"javascript:alert(1)"}}`}
	promoteWebBrowserFallback(&unsafe, "Öffne YouTube.com")
	if len(unsafe.ToolCalls) != 0 {
		t.Fatal("unsafe browser navigation was promoted")
	}
	bareDomain := agentInferenceResult{Text: `{"tool_name":"web_browser","args":{"action":"navigate","url":"youtube.com"}}`}
	promoteWebBrowserFallback(&bareDomain, "Öffne YouTube.com")
	if len(bareDomain.ToolCalls) != 1 {
		t.Fatal("valid browser domain was not promoted")
	}
}

func TestExecutionContractRequiresVisibleWriterEffect(t *testing.T) {
	run := AgentRun{Objective: "Schreibe im Writer ein Gedicht aus der Zukunft", SphereActive: true}
	effects := requiredExecutionEffects(run)
	if len(effects) != 1 || effects[0] != "sphere_write_document" {
		t.Fatalf("writer effect contract missing: %+v", effects)
	}
	if !invalidCompletionText("Die geplante Sequenz wurde vollständig ausgeführt.") {
		t.Fatal("generic fake-success response was accepted")
	}
	run.Steps = []RunStep{{Kind: RunStepTool, ToolName: "sphere_write_document", Status: StepVerified, Result: "Writer updated"}}
	if missing := missingExecutionEffects(run); len(missing) != 0 {
		t.Fatalf("verified writer effect was not recognized: %+v", missing)
	}
}

func TestExecutionContractRequiresGlobalWatchContextAndControls(t *testing.T) {
	run := AgentRun{Objective: "[GLOBAL WATCH OPERATOR]\nZeitfenster: 24h\nAnweisung: News aus Deutschland der letzten 24h"}
	effects := requiredExecutionEffects(run)
	for _, required := range []string{"global_watch_nexus_context", "global_watch_focus_region", "global_watch_time_window"} {
		found := false
		for _, effect := range effects {
			if effect == required {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing required Global Watch effect %s in %+v", required, effects)
		}
	}
}

func TestRuntimeCapabilityCatalogRoutesWeatherNavigationAndResearch(t *testing.T) {
	tests := []struct {
		objective string
		effect    string
	}{
		{"Wie ist das Wetter in New York?", "weather_lookup"},
		{"Öffne den Tab Global Watch", "navigate_ui"},
		{"Recherchiere aktuelle Entwicklungen im Web", "web_browser"},
		{"Lege einen Agentenplan für die Migration an", "task_set_checklist"},
	}
	for _, test := range tests {
		run := AgentRun{Objective: test.objective}
		if effects := requiredExecutionEffects(run); !containsString(effects, test.effect) {
			t.Errorf("objective %q missing effect %q in %+v", test.objective, test.effect, effects)
		}
		if !shouldEnableAgentTools(test.objective) {
			t.Errorf("objective %q did not enable tools", test.objective)
		}
	}
}

func TestNeedsDomainDraftAvoidsSecondModelForToolOnlyObjectives(t *testing.T) {
	weather := AgentRun{Objective: "Wie ist das Wetter in Köln?", Continuity: newContinuityState("Wie ist das Wetter in Köln?", false)}
	if needsDomainDraft(weather) {
		t.Fatal("weather lookup must not spend a separate domain-model call")
	}
	writer := AgentRun{Objective: "Schreibe im Writer ein Gedicht", SphereActive: true, Continuity: newContinuityState("Schreibe im Writer ein Gedicht", true)}
	if !needsDomainDraft(writer) {
		t.Fatal("creative Writer content requires a domain draft")
	}
	code := AgentRun{Objective: "Refactore den Go Code", Continuity: newContinuityState("Refactore den Go Code", false)}
	if !needsDomainDraft(code) {
		t.Fatal("developer work requires a domain draft")
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestOrchestratorAccountingPreservesPerModelCost(t *testing.T) {
	orchestrator := agentInferenceResult{InputTokens: 120, OutputTokens: 30, CostUSD: 0.004, CostCalculated: true}
	domain := agentInferenceResult{InputTokens: 900, OutputTokens: 180, CachedTokens: 250, CostUSD: 0.021, CostCalculated: true}
	mergeInferenceAccounting(&orchestrator, domain)
	if orchestrator.InputTokens != 1020 || orchestrator.OutputTokens != 210 || orchestrator.CachedTokens != 250 {
		t.Fatalf("two-model token accounting is incomplete: %+v", orchestrator)
	}
	if math.Abs(orchestrator.CostUSD-0.025) > 1e-12 || !orchestrator.CostCalculated {
		t.Fatalf("per-model orchestration cost was not preserved: %+v", orchestrator)
	}
}
