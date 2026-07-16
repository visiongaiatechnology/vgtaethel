package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"go-aethel/agent"
	"go-aethel/handlers"
	"go-aethel/intelligence"
	"go-aethel/mailbox"
	"go-aethel/memory"
	"go-aethel/osint"
	"go-aethel/personal"
	"go-aethel/provider"
	"go-aethel/security"
	"go-aethel/skills"
	"go-aethel/system"
	"go-aethel/voice"
)

// App is the Wails application struct
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

// APIRouter holds all API handlers — set up in startup(), used by APIHandler
var APIRouter *http.ServeMux

// APIHandler is used as the assetserver.Handler fallback.
// Wails calls it whenever a request path doesn't match an embedded static file.
// So /index.html → served from embedded FS; /v1/chat → APIHandler → APIRouter.
// Everything runs on the same Wails virtual host = no CORS, no navigation, no PowerShell spawn.
var APIHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w = secureResponseWriter{ResponseWriter: w}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024*1024)
	if APIRouter != nil {
		APIRouter.ServeHTTP(w, r)
		return
	}
	http.Error(w, `{"error":"starting"}`, http.StatusServiceUnavailable)
})

// startup initialises all AETHEL state and registers API handlers.
// Runs synchronously before the window is shown.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("🛡️ VGT AETHEL :: STARTUP")

	memoryStore := memory.NewLocalMemoryStore()
	personalStore := personal.NewPersonalStore("./vgt_workspace/personal")
	releaseService := system.NewReleaseService("./vgt_workspace/release")
	registry := skills.NewSkillRegistry()
	providers := provider.NewProviderRegistry()
	registry.Register(&skills.ExecuteCommandSkill{})
	registry.Register(&skills.ReadFileSkill{})
	registry.Register(&skills.WriteFileSkill{})
	registry.Register(&skills.ReplaceFileContentSkill{})
	registry.Register(&system.RestoreSnapshotSkill{Store: system.DefaultFileSnapshots})
	registry.Register(&skills.SetChecklistSkill{})
	registry.Register(&skills.UpdateChecklistSkill{})
	registry.Register(&skills.MemorySaveSkill{Store: memoryStore})
	registry.Register(&skills.MemoryRecallSkill{Store: memoryStore})
	registry.Register(&skills.PersonalMemorySaveSkill{Store: personalStore})
	registry.Register(&skills.PersonalMemoryRecallSkill{Store: personalStore})
	registry.Register(&skills.WebBrowserSkill{})
	registry.Register(&skills.WeatherSkill{})
	registry.Register(&skills.MarketSkill{})
	registry.Register(&skills.SphereWriteDocumentSkill{})
	registry.Register(&skills.MediaControlSkill{})
	registry.Register(&skills.YouTubeControlSkill{})
	registry.Register(&skills.VisionContextSkill{})
	registry.Register(&skills.GUIControlSkill{})
	registry.Register(&skills.GUIWindowControlSkill{})
	registry.Register(&skills.ListDirSkill{})
	registry.Register(&skills.MountFolderSkill{})
	registry.Register(&skills.CodeCartographySkill{})
	registry.Register(&skills.ExternalAgentHandoffSkill{})
	registry.Register(&skills.IntelligenceStatusSkill{})
	registry.Register(&skills.GlobalWatchNexusContextSkill{})
	registry.Register(&skills.GlobalWatchScheduleBriefingSkill{})
	registry.Register(&skills.IntelligenceProposeObservationSkill{})
	registry.Register(&skills.IntelligenceAddEntitySkill{})
	registry.Register(&skills.IntelligenceLinkEntitiesSkill{})
	registry.Register(&skills.IntelligenceCollectSkill{})
	registry.Register(&skills.IntelligenceCreateCaseSkill{})
	registry.Register(&skills.OSINTAddCustomFeedSkill{})
	registry.Register(&skills.OSINTSetBriefingPromptSkill{})
	registry.Register(&skills.IntelligenceRequestReIDSkill{})
	registry.Register(&skills.GlobalWatchFocusSkill{})
	registry.Register(&skills.GlobalWatchObserveSkill{})
	registry.Register(&skills.GlobalWatchToggleLayerSkill{})
	registry.Register(&skills.NavigateUISkill{})
	registry.Register(&skills.GlobalWatchFocusRegionSkill{})
	registry.Register(&skills.GlobalWatchTimeWindowSkill{})
	registry.Register(&skills.GlobalWatchOpenReportSkill{})
	registry.Register(&skills.IntelligenceRegionStatusSkill{})
	registry.Register(&skills.IntelligenceExplainScoreSkill{})
	registry.Register(&skills.IntelligenceRegionCompareSkill{})
	registry.Register(&skills.IntelligenceGenerateBriefSkill{})
	registry.Register(&skills.IntelligenceSourceHealthSkill{})
	registry.Register(&skills.IntelligenceGlobalStatusSkill{})
	registry.Register(&skills.IntelligenceRecentChangesSkill{})
	registry.Register(&skills.OSINTCaseCreateSkill{})
	registry.Register(&skills.IntelligenceCreateWatchlistSkill{})
	registry.Register(&skills.IntelligenceMarketSummarySkill{})
	registry.Register(&skills.IntelligenceInfrastructureSummarySkill{})
	registry.Register(&skills.IntelligenceConflictSummarySkill{})
	registry.Register(&skills.IntelligenceCyberSummarySkill{})
	registry.Register(&skills.IntelligenceCreateAlertRuleSkill{})
	registry.Register(&skills.IntelligenceSyncPersonalSkill{})
	registry.Register(&skills.IntelligencePersonalImpactSkill{})
	registry.Register(&skills.OSINTEvidenceCaptureSkill{})
	registry.Register(&skills.OSINTEntityProposeSkill{})
	registry.Register(&skills.OSINTRelationProposeSkill{})
	registry.Register(&skills.OSINTTimelineGenerateSkill{})
	registry.Register(&skills.OSINTReportGenerateSkill{})
	registry.Register(&skills.IntelligenceApproveReIDSkill{})
	registry.Register(&skills.IntelligenceIdentityStatusSkill{})
	registry.Register(&skills.IntelligenceSetAssessmentStatusSkill{})
	registry.Register(&skills.IntelligenceConnectorFetchSkill{})
	registry.Register(&skills.MailListMessagesSkill{})
	registry.Register(&skills.MailSendMessageSkill{})

	gKey, oKey, dsKey, gemKey, claudeKey, mDirs, mounts := loadConfig()

	guard := security.NewSecurityGuard()
	leases := security.NewLeaseManager("./vgt_workspace/active_leases.json")
	audit := security.NewAuditLogger("./vgt_workspace/security_audit.json")
	policy := security.NewPolicyEngine(guard, leases, audit)
	approvals := security.NewApprovalManager("./vgt_workspace/approval_grants.json")

	// ─── Sherpa-ONNX Offline TTS (PRIMÄR) ───
	sherpaEngine := voice.NewSherpaVoiceEngine("./vgt_workspace/models/sherpa", "./vgt_workspace/audio")
	if err := sherpaEngine.Init(); err != nil {
		log.Printf("⚠️ Sherpa-ONNX Init-Fehler: %v", err)
	} else {
		log.Printf("✅ Sherpa-ONNX: %d Stimmen erkannt", len(sherpaEngine.ListVoices()))
	}

	voiceRegistry := voice.NewVoiceRegistry(sherpaEngine)

	vault, err := security.NewSecretVault("./vgt_workspace/secret_vault.enc", "./vgt_workspace/vault.key")
	if err != nil {
		log.Fatalf("Failed to initialize secret vault: %v", err)
	}
	mailbox.SharedService = mailbox.NewService("./vgt_workspace/mail_account.enc", vault)

	taskEngine := agent.NewTaskEngine("./vgt_workspace/tasks.json")
	_ = taskEngine.Load()
	runEngine := agent.NewRunEngine("./vgt_workspace/agent_runs.json")

	// Intelligence + OSINT before InitState (skills/handlers need them)
	osintEngine := osint.NewOSINTEngine("./vgt_workspace/osint_feeds.json")
	intelStore := intelligence.NewIntelligenceStore("./vgt_workspace/intelligence_core.json")
	intelStore.ChatEvaluator = func(systemPrompt, userPrompt string) (string, error) {
		msg := map[string]any{"role": "user", "content": userPrompt}
		rawMsg, _ := json.Marshal(msg)
		chatReq := struct {
			ModelID      string            `json:"model_id"`
			Messages     []json.RawMessage `json:"messages"`
			SystemPrompt string            `json:"system_prompt"`
			Temperature  float64           `json:"temperature"`
		}{
			ModelID:      "openai/gpt-oss-120b",
			Messages:     []json.RawMessage{rawMsg},
			SystemPrompt: systemPrompt,
			Temperature:  0.2,
		}
		payload, err := json.Marshal(chatReq)
		if err != nil {
			return "", err
		}
		req := httptest.NewRequest("POST", "/v1/chat", bytes.NewReader(payload))
		recorder := httptest.NewRecorder()
		handlers.HandleChat(recorder, req)
		result := agent.ParseAgentSSE(recorder.Body.String())
		if result.Err != nil {
			return "", result.Err
		}
		return result.Text, nil
	}
	intelStore.StartBackgroundEvaluationWorker()
	intelSources := intelligence.NewIntelligenceSourceRegistry(intelStore)
	intelMonitor := osint.NewGlobalWatchMonitor(intelStore, "./vgt_workspace/global_watch_schedule.json", "./vgt_workspace/intelligence_reports")
	intelMonitor.Start()

	sharedIntelBus := intelligence.NewEventBus()
	intelligence.SharedIntelStore = intelligence.NewStore("./vgt_workspace/intel_shared.json", sharedIntelBus)
	intelligence.SharedIntelStore.StartProactiveLoop(2 * time.Minute)
	if pc := personal.BuildSharedPersonalContext(personalStore); pc.OperatorID != "" || len(pc.Interests) > 0 || len(pc.Goals) > 0 {
		intelligence.SharedIntelStore.SetPersonalContext(pc)
	}
	osint.RegisterAllConnectors()

	osintEngine.SetRefreshHook(func(events []intelligence.OSINTEvent) {
		intelStore.SyncOSINTEvents(events)
		for _, ev := range events {
			obs := intelligence.Observation{
				ID: "obs-" + ev.ID, SourceID: "rss-" + strings.ReplaceAll(ev.Source, " ", "_"),
				RawText: ev.Title + " " + ev.Summary, ObservedAt: ev.Timestamp,
				Latitude: ev.Lat, Longitude: ev.Lon, Domain: string(ev.Domain),
			}
			intelligence.SharedIntelStore.IngestObservation(obs)
		}
		snap := intelligence.SharedIntelStore.GetSnapshot()
		for _, se := range snap.Events {
			re := intelligence.IntelligenceEvent{
				ID: "shared-" + se.ID, Title: se.Title,
				Summary: "[" + se.Domain + "] " + se.Summary, Source: "shared-intel",
				Latitude: se.Latitude, Longitude: se.Longitude,
				Confidence: se.Confidence, Severity: se.Severity, ObservedAt: se.ObservedAt,
			}
			_ = intelStore.ProposeEvent(re)
		}
	})
	osintEngine.Start()

	state = &AppState{
		apiKey: gKey, openaiAPIKey: oKey, deepseekAPIKey: dsKey, geminiAPIKey: gemKey, claudeAPIKey: claudeKey,
		mountedDirs: mDirs, mounts: mounts,
		guard: guard, leases: leases, audit: audit, policy: policy, approvals: approvals,
		skills: registry, providers: providers, memory: memoryStore, voice: voiceRegistry, vault: vault,
		tasks: taskEngine, runs: runEngine, personal: personalStore, release: releaseService,
		osint: osintEngine, intel: intelStore, intelSources: intelSources, intelMonitor: intelMonitor,
	}

	// Dependency injection into packages (after all services exist)
	security.InitState(state.MountAllows)
	skills.InitState(state.intelSources, state.personal, state.intelMonitor, state.intel, state.osint, state.GetMounts, state.AddMount, state.MountAllows, handlers.RecordFileChange)
	agent.InitState(state.runs, state.policy, state.skills, state.providers, state.audit, state.GetAPIKey)
	osint.InitState(state.osint)
	personal.InitState(state.personal, state.GetAPIKey, state.GetOpenAIKey, state.GetDeepSeekKey, state.GetGeminiKey, state.GetClaudeKey)
	system.InitState(state.release)
	voice.InitState(state.GetAPIKey, state.GetOpenAIKey)
	handlers.InitState(
		state.guard, state.leases, state.audit, state.policy, state.approvals,
		state.skills, state.providers, state.memory, state.voice, state.vault,
		state.tasks, state.runs, state.personal, state.release, state.osint,
		state.intel, state.intelSources, state.intelMonitor,
		state.GetAPIKey, state.GetOpenAIKey, state.GetDeepSeekKey, state.GetGeminiKey, state.GetClaudeKey,
		state.saveConfig, state.GetMountedDirs, state.GetMounts,
	)
	// Agent chat loop reuses HTTP chat handler without importing handlers (avoids cycle).
	agent.ChatHandler = handlers.HandleChat

	state.tasks.Start()

	// Install operator drop-in Earth basemap (1.jpg → frontend/assets/earth_day.jpg) if needed
	handlers.EnsureEarthTextureOnDisk()

	// Wire all API handlers — done before the window shows
	APIRouter = http.NewServeMux()
	APIRouter.HandleFunc("/health", handlers.HandleHealth)
	APIRouter.HandleFunc("/v1/assets/earth-texture", handlers.HandleEarthTexture)
	APIRouter.HandleFunc("/assets/earth_day.jpg", handlers.HandleEarthTexture)
	APIRouter.HandleFunc("/v1/intelligence/", handlers.HandleIntelligence)
	APIRouter.HandleFunc("/v1/setup", handlers.HandleSetup)
	APIRouter.HandleFunc("/v1/models", handlers.HandleModels)
	APIRouter.HandleFunc("/v1/providers/health", handlers.HandleProviderHealth)
	APIRouter.HandleFunc("/v1/diagnostics/export", handlers.HandleDiagnosticsExport)
	APIRouter.HandleFunc("/v1/release/info", handlers.HandleReleaseInfo)
	APIRouter.HandleFunc("/v1/release/preferences", handlers.HandleReleasePreferences)
	APIRouter.HandleFunc("/v1/release/check", handlers.HandleReleaseCheck)
	APIRouter.HandleFunc("/v1/chat", handlers.HandleChat)
	APIRouter.HandleFunc("/v1/chat/runs", handlers.HandleChatAgentRuns)
	APIRouter.HandleFunc("/v1/chat/checklist", handlers.HandleChecklist)
	APIRouter.HandleFunc("/v1/tools/execute", handlers.HandleToolExecute)
	APIRouter.HandleFunc("/browser/screenshot.png", handlers.HandleBrowserScreenshot)
	APIRouter.HandleFunc("/v1/audio/speech", handlers.HandleAudioSpeech)
	APIRouter.HandleFunc("/v1/audio/voices", handlers.HandleAudioVoices)
	APIRouter.HandleFunc("/v1/audio/transcribe", handlers.HandleAudioTranscribe)
	APIRouter.HandleFunc("/v1/kernel/logs", handlers.HandleKernelLogs)
	APIRouter.HandleFunc("/v1/chat/history", handlers.HandleChatHistory)
	APIRouter.HandleFunc("/v1/chat/sessions", handlers.HandleChatSessions)
	APIRouter.HandleFunc("/v1/chat/sessions/load", handlers.HandleChatSessionsLoad)
	APIRouter.HandleFunc("/v1/chat/sessions/save", handlers.HandleChatSessionsSave)
	APIRouter.HandleFunc("/v1/chat/sessions/delete", handlers.HandleChatSessionsDelete)
	APIRouter.HandleFunc("/v1/kernel/tasks/", handlers.HandleKernelTasksPath)
	APIRouter.HandleFunc("/v1/runs", handlers.HandleRuns)
	APIRouter.HandleFunc("/v1/runs/", handlers.HandleRunsPath)
	APIRouter.HandleFunc("/v1/artifacts", handlers.HandleArtifacts)
	APIRouter.HandleFunc("/v1/security/leases", handlers.HandleSecurityLeases)
	APIRouter.HandleFunc("/v1/security/audit", handlers.HandleSecurityAudit)
	APIRouter.HandleFunc("/v1/security/status", handlers.HandleSecurityStatus)
	APIRouter.HandleFunc("/v1/memory", handlers.HandleMemory)
	APIRouter.HandleFunc("/v1/memory/explain", handlers.HandleMemoryExplain)
	APIRouter.HandleFunc("/v1/memory/export", handlers.HandleMemoryExport)
	APIRouter.HandleFunc("/v1/memory/search", handlers.HandleMemorySearch)
	APIRouter.HandleFunc("/v1/audio/health", handlers.HandleAudioHealth)
	APIRouter.HandleFunc("/v1/audio/test", handlers.HandleAudioTest)
	APIRouter.HandleFunc("/v1/viewport/screenshot", handlers.HandleViewportScreenshot)
	APIRouter.HandleFunc("/v1/viewport/status", handlers.HandleViewportStatus)
	APIRouter.HandleFunc("/v1/weather", handlers.HandleWeather)
	APIRouter.HandleFunc("/v1/markets", handlers.HandleMarketQuotes)
	APIRouter.HandleFunc("/v1/sphere/document", handlers.HandleSphereDocument)
	APIRouter.HandleFunc("/v1/sphere/workspace", handlers.HandleSphereWorkspace)
	APIRouter.HandleFunc("/v1/secrets", handlers.HandleSecrets)
	APIRouter.HandleFunc("/v1/settings", handlers.HandleSettings)
	APIRouter.HandleFunc("/v1/settings/reset", handlers.HandleSettingsReset)
	APIRouter.HandleFunc("/v1/settings/costs", handlers.HandleCosts)
	APIRouter.HandleFunc("/v1/settings/personas", handlers.HandlePersonas)
	APIRouter.HandleFunc("/v1/personal/status", handlers.HandlePersonalStatus)
	APIRouter.HandleFunc("/v1/personal/config", handlers.HandlePersonalConfig)
	APIRouter.HandleFunc("/v1/personal/profile", handlers.HandlePersonalProfile)
	APIRouter.HandleFunc("/v1/personal/memories", handlers.HandlePersonalMemories)
	APIRouter.HandleFunc("/v1/personal/setup/questions", handlers.HandlePersonalSetupQuestions)
	APIRouter.HandleFunc("/v1/personal/setup", handlers.HandlePersonalSetup)
	APIRouter.HandleFunc("/v1/personal/learn", handlers.HandlePersonalLearn)
	APIRouter.HandleFunc("/v1/osint/feeds", handlers.HandleOSINTFeeds)
	APIRouter.HandleFunc("/v1/osint/briefing", handlers.HandleOSINTBriefing)
	APIRouter.HandleFunc("/v1/osint/article", handlers.HandleOSINTArticleReader)
	APIRouter.HandleFunc("/v1/osint/collectors", handlers.HandleOSINTCollectors)
	APIRouter.HandleFunc("/v1/mail/config", handlers.HandleMailConfig)
	APIRouter.HandleFunc("/v1/mail/test", handlers.HandleMailTest)

	log.Println("✅ VGT AETHEL :: API-ROUTER BEREIT")

	// DeepSeek cache warmup (asynchron, blockiert nicht den Startup)
	if dsKey != "" {
		agent.WarmupDeepSeekCache(dsKey)
	}
}

// domReady is called when the frontend DOM is loaded — no navigation needed
func (a *App) domReady(ctx context.Context) {
	log.Println("🌐 VGT AETHEL :: DOM BEREIT")
}

// beforeClose is called when the user closes the window. Return false to terminate the process.
func (a *App) beforeClose(ctx context.Context) bool {
	log.Println("🔴 VGT AETHEL :: USER CLOSED WINDOW - TERMINATING PROCESS")
	return false
}

// shutdown is called at app termination
func (a *App) shutdown(ctx context.Context) {
	log.Println("🔴 VGT AETHEL :: SHUTDOWN")
}

// HideToTray hides the window (callable from frontend via Wails binding)
func (a *App) HideToTray() {
	runtime.Hide(a.ctx)
}

// ShowWindow brings AETHEL back from tray
func (a *App) ShowWindow() {
	runtime.Show(a.ctx)
}

// SelectDirectory opens a native directory picker dialog and returns the selected path
func (a *App) SelectDirectory() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Projektverzeichnis für Aethel freigeben",
	})
	if err != nil {
		log.Printf("Failed to open directory dialog: %v", err)
		return ""
	}
	return dir
}

// GetVersion returns the current version string
func (a *App) GetVersion() string {
	return system.ProductVersion
}

// GetAvailableModels is the direct Wails fallback for the WebView model
// registry. It returns only verified models belonging to configured providers;
// live discovery can never block this path.
func (a *App) GetAvailableModels() []map[string]interface{} {
	if state == nil || state.providers == nil {
		return []map[string]interface{}{}
	}
	models := state.providers.AvailableModels(state)
	if localModels := provider.GetLocalOllamaModels(); len(localModels) > 0 {
		models = append(models, localModels...)
	}
	return models
}

// isAPIPath returns true for paths that should be routed to Go handlers
func isAPIPath(p string) bool {
	return p == "/health" ||
		strings.HasPrefix(p, "/v1/") ||
		strings.HasPrefix(p, "/browser/")
}
