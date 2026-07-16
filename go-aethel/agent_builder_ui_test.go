package main

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed frontend/index.html
var agentBuilderHTML []byte

//go:embed frontend/modules/agent_builder.js
var agentBuilderJS []byte

//go:embed frontend/modules/control.js
var liveOperatorJS []byte

//go:embed frontend/modules/security.js
var securityUIJS []byte

//go:embed frontend/modules/memory.js
var memoryUIJS []byte

//go:embed frontend/modules/secrets.js
var secretsUIJS []byte

//go:embed frontend/modules/personal_mode.js
var personalModeUIJS []byte

//go:embed frontend/modules/tasks.js
var runCenterUIJS []byte

//go:embed frontend/modules/settings.js
var settingsUIJS []byte

//go:embed frontend/modules/ui.js
var coreUIJS []byte

//go:embed frontend/modules/case_workspace.js
var caseWorkspaceUIJS []byte

//go:embed frontend/modules/chat.js
var chatWorkspaceUIJS []byte

func TestAgentBuilderTeamConfigurationIsConnectedToPersistentRun(t *testing.T) {
	html := string(agentBuilderHTML)
	javascript := string(agentBuilderJS)
	if strings.Contains(viewFragment(html, "view-agent", "view-control"), `style="`) {
		t.Fatal("Agent Tracker must not regress to inline layout styles")
	}
	if !strings.Contains(html, `id="custom-agents-container"`) || !strings.Contains(javascript, `getElementById('custom-agents-container')`) {
		t.Fatal("custom-agent UI and runtime must share the same container")
	}
	for _, role := range []string{"orchestrator", "builder", "reviewer", "repairer", "documentator"} {
		if !strings.Contains(javascript, `roleIDs = ['orchestrator', 'builder', 'reviewer', 'repairer', 'documentator']`) {
			t.Fatalf("team role %s is not part of the prompt assembly", role)
		}
		if !strings.Contains(html, `id="agent-prompt-`+role+`"`) {
			t.Fatalf("team role %s prompt is missing from UI", role)
		}
	}
	for _, token := range []string{"buildTeamSystemPrompt", `mode: 'agent_team'`, "waiting_approval", "Agent Team recovery", "monitorTeamRun(resumable.id)"} {
		if !strings.Contains(javascript, token) {
			t.Fatalf("persistent team-run integration missing %q", token)
		}
	}
}

func TestLiveOperatorControlsPersistentAgentRuns(t *testing.T) {
	html := string(agentBuilderHTML)
	control := string(liveOperatorJS)
	if strings.Contains(viewFragment(html, "view-control", "view-security"), `style="`) {
		t.Fatal("Live Operator must not regress to inline layout styles")
	}
	for _, token := range []string{
		"state.activeRunId",
		"/v1/runs/${encodeURIComponent(state.activeRunId)}/${action}",
		"/v1/runs/${encodeURIComponent(state.activeRunId)}/cancel",
		"RESUME RUN",
		"state.pendingToolRequest = null",
	} {
		if !strings.Contains(control, token) {
			t.Fatalf("Live Operator persistent-run control missing %q", token)
		}
	}
}

func TestRebuiltTrustKnowledgeAndIdentityViewsUseSafeComponents(t *testing.T) {
	html := string(agentBuilderHTML)
	for _, view := range [][2]string{{"view-security", "view-memory"}, {"view-memory", "view-personal"}, {"view-personal", "view-sphere"}} {
		fragment := viewFragment(html, view[0], view[1])
		if fragment == "" {
			t.Fatalf("view fragment %s missing", view[0])
		}
		if strings.Contains(fragment, `style="`) {
			t.Fatalf("%s must not regress to inline layout styles", view[0])
		}
	}
	for name, source := range map[string]string{
		"security": string(securityUIJS),
		"memory":   string(memoryUIJS),
		"secrets":  string(secretsUIJS),
		"personal": string(personalModeUIJS),
	} {
		if strings.Contains(source, ".innerHTML =") || strings.Contains(source, ".innerHTML=") {
			t.Fatalf("%s renderer must use DOM construction for persisted/user data", name)
		}
		if strings.Contains(source, "window.alert(") || strings.Contains(source, "alert(") {
			t.Fatalf("%s must expose failures through integrated UI feedback", name)
		}
	}
}

func TestRunCenterCreatesExecutablePersistentAgentRuns(t *testing.T) {
	html := string(agentBuilderHTML)
	fragment := viewFragment(html, "view-tasks", "view-archive")
	if fragment == "" {
		t.Fatal("Run Center view fragment missing")
	}
	if strings.Contains(fragment, `style="`) {
		t.Fatal("Run Center must not regress to inline layout styles")
	}
	source := string(runCenterUIJS)
	for _, token := range []string{`mode: 'chat_agent'`, "agent_messages: [{ role: 'user', content: objective }]", "max_agent_turns: 24", "requestRunApproval"} {
		if !strings.Contains(source, token) {
			t.Fatalf("Run Center persistent execution chain missing %q", token)
		}
	}
}

func TestPersonaRegistryUsesSafeLinkedComponents(t *testing.T) {
	html := string(agentBuilderHTML)
	fragment := viewFragment(html, "view-personas", "view-archive")
	if fragment == "" {
		t.Fatal("Persona Registry view fragment missing")
	}
	if strings.Contains(fragment, `style="`) {
		t.Fatal("Persona Registry must not regress to inline layout styles")
	}
	source := string(settingsUIJS)
	start := strings.Index(source, "function renderCustomPersonasList")
	end := strings.Index(source, "export async function loadProviderHealth")
	if start < 0 || end <= start {
		t.Fatal("Persona Registry runtime segment missing")
	}
	personaRuntime := source[start:end]
	for _, forbidden := range []string{".innerHTML", "onclick=", "alert("} {
		if strings.Contains(personaRuntime, forbidden) {
			t.Fatalf("Persona Registry renderer contains forbidden pattern %q", forbidden)
		}
	}
	for _, token := range []string{"api.savePersona(payload)", "loadCustomPersonasSettings()", "refreshPersonasDropdowns", "settings-persona-feedback"} {
		if !strings.Contains(source, token) {
			t.Fatalf("Persona Registry system linkage missing %q", token)
		}
	}
}

func TestSidebarModelRegistryUsesWebViewSafeFlatOptions(t *testing.T) {
	source := string(coreUIJS)
	// applyModelsToDropdowns + loadModels together own the sidebar KI-Modell / Orchestrator paint path.
	start := strings.Index(source, "function applyModelsToDropdowns")
	if start < 0 {
		start = strings.Index(source, "export async function loadModels")
	}
	end := strings.Index(source, "export async function loadVoices")
	if start < 0 || end <= start {
		t.Fatal("model registry UI segment missing")
	}
	modelRuntime := source[start:end]
	for _, required := range []string{"api.getModels()", "elModelDropdown.replaceChildren()", "document.createElement(\"option\")", "state.currentModel = elModelDropdown.value", "orchestrator-model-dropdown", "model-provider-separator"} {
		if !strings.Contains(modelRuntime, required) {
			t.Fatalf("model dropdown linkage missing %q", required)
		}
	}
	if strings.Contains(modelRuntime, "optgroup") {
		t.Fatal("sidebar model dropdown must stay flat for reliable WebView2 rendering")
	}
}

func TestCaseWorkspaceUsesEvidenceSafeDOMComponents(t *testing.T) {
	html := string(agentBuilderHTML)
	fragment := viewFragment(html, "view-case", "view-chat")
	if fragment == "" {
		t.Fatal("Case Workspace view fragment missing")
	}
	if strings.Contains(fragment, `style="`) {
		t.Fatal("Case Workspace must not regress to inline layout styles")
	}
	source := string(caseWorkspaceUIJS)
	for _, forbidden := range []string{".innerHTML", "alert(", "prompt(", "style.cssText", ".style."} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("Case Workspace contains forbidden legacy UI pattern %q", forbidden)
		}
	}
	for _, required := range []string{"showCaseDialog", "safeHTTPSURL", "source_event_id", "evidence_id", "AETHEL_PROMOTE_TO_CASE", "requestReID", "approveReID"} {
		if !strings.Contains(source, required) {
			t.Fatalf("Case Workspace chain missing %q", required)
		}
	}
}

func TestChatArchiveUsesSafeLocalSessionCards(t *testing.T) {
	html := string(agentBuilderHTML)
	startView := strings.Index(html, `id="view-archive"`)
	endView := strings.Index(html, `<!-- Full Machine Activation Modal -->`)
	if startView < 0 || endView <= startView {
		t.Fatal("Chat Archive view fragment missing")
	}
	fragment := html[startView:endView]
	if strings.Contains(fragment, `style="`) {
		t.Fatal("Chat Archive must not regress to inline layout styles")
	}
	source := string(chatWorkspaceUIJS)
	start := strings.Index(source, "export async function loadSessionsList")
	end := strings.Index(source, "export async function loadSession(id)")
	if start < 0 || end <= start {
		t.Fatal("Chat Archive runtime segment missing")
	}
	archiveRuntime := source[start:end]
	for _, forbidden := range []string{".innerHTML", "onclick=", "style.cssText", ".style."} {
		if strings.Contains(archiveRuntime, forbidden) {
			t.Fatalf("Chat Archive contains forbidden renderer pattern %q", forbidden)
		}
	}
	for _, required := range []string{"container.replaceChildren()", "textContent = session.title", "deleteSession(session.id)", "loadSessionAndSwitch(session.id)"} {
		if !strings.Contains(archiveRuntime, required) {
			t.Fatalf("Chat Archive linkage missing %q", required)
		}
	}
}

func viewFragment(source, startID, endID string) string {
	start := strings.Index(source, `id="`+startID+`"`)
	end := strings.Index(source, `id="`+endID+`"`)
	if start < 0 || end <= start {
		return ""
	}
	return source[start:end]
}
