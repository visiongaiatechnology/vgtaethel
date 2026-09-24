// STATUS: DIAMANT VGT SUPREME
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-aethel/agent"
	"go-aethel/coder"
	"go-aethel/geoint"
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
	ctx        context.Context
	operations *personal.OperationsService
	geoint     *geoint.GeoIntService
	codeMu     sync.Mutex
	codePath   string
}

func NewApp() *App {
	return &App{}
}

// APIRouter holds all API handlers — set up in startup(), used by APIHandler
var APIRouter *http.ServeMux

// APIHandler is used as the assetserver.Handler fallback and HTTP server API handler.
var APIHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w = secureResponseWriter{ResponseWriter: w}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024*1024)
	if APIRouter == nil {
		http.Error(w, `{"error":"starting"}`, http.StatusServiceUnavailable)
		return
	}
	if ServerAuth != nil && ServerAuth.IsAuthRequired() && !isPublicAuthPath(r.URL.Path) {
		if _, ok := ServerAuth.ValidateRequest(r); !ok {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprintf(w, `{"error":"authentication required","auth_required":true,"configured":%t}`, ServerAuth.IsConfigured())
			return
		}
	}
	APIRouter.ServeHTTP(w, r)
})

// startup initialises all AETHEL state and registers API handlers.
// Runs synchronously before the window is shown.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("🛡️ VGT AETHEL :: STARTUP")

	memoryStore := memory.NewLocalMemoryStore()
	personalStore := personal.NewPersonalStore("./vgt_workspace/personal")
	operationsStore := personal.NewOperationsStore("./vgt_workspace/personal/operations.enc")
	if err := operationsStore.Load(); err != nil {
		log.Printf("Personal Operations inbox unavailable: %v", err)
	}
	operationsService := personal.NewOperationsService(operationsStore, personalStore)
	operationsService.SetWeatherProvider(func(city string) (string, string, error) {
		snapshot, err := skills.LookupWeather(city)
		if err != nil {
			return "", "", err
		}
		body := fmt.Sprintf("%s · %.1f °C · Wind %.1f km/h\nStand: %s", snapshot.Summary, snapshot.Temperature, snapshot.WindSpeed, snapshot.ObservedAt)
		return "Wetter · " + snapshot.City, body, nil
	})
	a.operations = operationsService
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
	registry.Register(&skills.PersonalOperationsSkill{Store: personalStore, Inbox: operationsStore})
	registry.Register(&skills.MarketSkill{})
	registry.Register(&skills.SphereWriteDocumentSkill{})
	registry.Register(&skills.SphereProposeDocumentChangeSkill{})
	registry.Register(&skills.SphereManageTripSkill{})
	registry.Register(&skills.SphereManagePlanSkill{})
	registry.Register(&skills.SphereResearchCaptureSkill{})
	registry.Register(&skills.SphereSendToSkill{})
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
	registry.Register(&skills.GlobalWatchNaturalHazardsContextSkill{})
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
	registry.Register(&skills.MailReadMessageSkill{})
	registry.Register(&skills.MailManageSkill{})
	registry.Register(&skills.GeoContactsNearestSkill{})
	registry.Register(&skills.GeoFocusSkill{})
	registry.Register(&skills.GeoTrackSkill{})
	registry.Register(&skills.GeoLayerEnableSkill{})
	registry.Register(&skills.GeoCorrelateSkill{})
	registry.Register(&skills.GeoCCTVLookupSkill{})

	gKey, oKey, dsKey, gemKey, claudeKey, mDirs, mounts, permMode := loadConfig()

	guard := security.NewSecurityGuard()
	leases := security.NewLeaseManager("./vgt_workspace/active_leases.json")
	audit := security.NewAuditLogger("./vgt_workspace/security_audit.json")
	policy := security.NewPolicyEngine(guard, leases, audit)
	if permMode != "" {
		policy.SetMode(security.PermissionMode(permMode))
	}
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
	taskEngine.SetNotificationSink(func(task agent.TaskItem) {
		priority := "normal"
		if task.Status == "failed" || task.Status == "blocked" {
			priority = "high"
		}
		_, _ = operationsStore.Enqueue(personal.OperationNotice{
			Kind: "task_update", Priority: priority, Title: task.Text,
			Body: task.LastReport, Source: "task_engine", RequireAck: task.Status != "done",
			Metadata: map[string]string{"task_id": task.ID, "status": task.Status},
		})
	})
	runEngine := agent.NewRunEngine("./vgt_workspace/agent_runs.json")
	coderManager := coder.NewManager("./vgt_workspace/coder_sessions.sealed")

	// Intelligence + OSINT + GEOINT before InitState (skills/handlers need them)
	osintEngine := osint.NewOSINTEngine("./vgt_workspace/osint_feeds.json")
	shadowService := osint.NewShadowService("./vgt_workspace/shadow_osint.enc")
	geointService := geoint.NewGeoIntService("./vgt_workspace/geoint", "")
	geointService.AttachShadow(shadowService)
	geointService.Start(ctx)
	a.geoint = geointService
	sharedIntelBus := intelligence.NewEventBus()
	intelligence.SharedIntelStore = intelligence.NewStore("./vgt_workspace/intel_shared.json", sharedIntelBus)
	geointService.AttachIntelligence(intelligence.SharedIntelStore)
	if err := intelligence.MigrateLegacyIntelligence("./vgt_workspace/intelligence_core.json", intelligence.SharedIntelStore); err != nil {
		log.Printf("[INTELLIGENCE] Legacy migration failed closed: %v", err)
	}
	intelStore := intelligence.NewCanonicalIntelligenceAdapter(intelligence.SharedIntelStore)
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

	intelligence.SharedIntelStore.StartProactiveLoop(2 * time.Minute)
	if pc := personal.BuildSharedPersonalContext(personalStore); pc.OperatorID != "" || len(pc.Interests) > 0 || len(pc.Goals) > 0 {
		intelligence.SharedIntelStore.SetPersonalContext(pc)
	}
	osint.RegisterAllConnectors()
	osint.StartWebsiteMonitorScheduler(ctx, intelligence.SharedIntelStore)

	osintEngine.SetRefreshHook(func(events []intelligence.OSINTEvent) {
		for _, ev := range events {
			// Preserve hazard source labels so Live Globe can classify earthquakes / volcanoes.
			srcID := "rss-" + strings.ReplaceAll(ev.Source, " ", "_")
			srcLower := strings.ToLower(ev.Source + " " + ev.Title + " " + ev.Summary)
			switch {
			case strings.Contains(srcLower, "usgs") || strings.Contains(srcLower, "[earthquake]"):
				srcID = "usgs-earthquakes"
			case strings.Contains(srcLower, "eonet") || strings.Contains(srcLower, "[volcano"):
				srcID = "nasa-eonet-volcano"
			}
			obs := intelligence.Observation{
				ID: "obs-" + ev.ID, SourceID: srcID,
				RawText: ev.Title + " " + ev.Summary, ObservedAt: ev.Timestamp,
				Latitude: ev.Lat, Longitude: ev.Lon, Domain: string(ev.Domain),
				OriginalURL: ev.SourceURL, FinalURL: ev.URL, PublishedAt: ev.Timestamp,
				FetchedAt: time.Now().UTC(), MIMEType: "application/feed+json", ParserVersion: "osint-event-v2",
			}
			intelligence.SharedIntelStore.IngestObservation(obs)
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
	security.InitState(state.MountAllows, func() string {
		root, _ := handlers.CurrentCodeWorkspaceRoot()
		return root
	})
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
	handlers.InitOperations(operationsStore)
	handlers.InitCoderManager(coderManager)
	handlers.InitShadowService(shadowService)
	handlers.InitGeoInt(geointService)
	skills.SetGeoIntService(geointService)
	// Agent chat loop reuses HTTP chat handler without importing handlers (avoids cycle).
	agent.ChatHandler = handlers.HandleChat

	state.tasks.Start()
	operationsService.Start()
	// Seed sentinel home from Personal Core so emergency popups only fire for the operator location.
	if profile, err := personalStore.LoadProfile(); err == nil {
		agent.SharedSentinel.SetLocation(profile.LocationCity, profile.LocationCountry)
	}
	agent.SharedSentinel.Start()

	// Install operator drop-in Earth basemap (1.jpg → frontend/assets/earth_day.jpg) if needed
	handlers.EnsureEarthTextureOnDisk()

	// Wire all API handlers — done before the window shows
	APIRouter = http.NewServeMux()
	registerAuthRoutes(APIRouter)
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
	APIRouter.HandleFunc("/v1/code/workspace/tree", handlers.HandleCodeWorkspaceTree)
	APIRouter.HandleFunc("/v1/code/workspace/file", handlers.HandleCodeWorkspaceFile)
	APIRouter.HandleFunc("/v1/coder/sessions", handlers.HandleCoderSessions)
	APIRouter.HandleFunc("/v1/coder/sessions/", handlers.HandleCoderSessionPath)
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
	APIRouter.HandleFunc("/v1/security/mode", handlers.HandleSecurityMode)
	APIRouter.HandleFunc("/v1/memory", handlers.HandleMemory)
	APIRouter.HandleFunc("/v1/memory/explain", handlers.HandleMemoryExplain)
	APIRouter.HandleFunc("/v1/memory/export", handlers.HandleMemoryExport)
	APIRouter.HandleFunc("/v1/memory/search", handlers.HandleMemorySearch)
	APIRouter.HandleFunc("/v1/audio/health", handlers.HandleAudioHealth)
	APIRouter.HandleFunc("/v1/audio/test", handlers.HandleAudioTest)
	APIRouter.HandleFunc("/v1/sentinel/alerts", agent.HandleSentinelAlerts)
	APIRouter.HandleFunc("/v1/space/weather", handlers.HandleSpaceWeather)
	APIRouter.HandleFunc("/v1/space/sdo_image", handlers.HandleSdoImageProxy)
	APIRouter.HandleFunc("/v1/space/analysis", handlers.HandleSpaceAnalysis)
	APIRouter.HandleFunc("/v1/viewport/screenshot", handlers.HandleViewportScreenshot)
	APIRouter.HandleFunc("/v1/viewport/status", handlers.HandleViewportStatus)
	APIRouter.HandleFunc("/v1/weather", handlers.HandleWeather)
	APIRouter.HandleFunc("/v1/markets", handlers.HandleMarketQuotes)
	APIRouter.HandleFunc("/v1/sphere/document", handlers.HandleSphereDocument)
	APIRouter.HandleFunc("/v1/sphere/workspace", handlers.HandleSphereWorkspace)
	APIRouter.HandleFunc("/v1/sphere/state", handlers.HandleSphereState)
	APIRouter.HandleFunc("/v1/sphere/objects", handlers.HandleSphereObjects)
	APIRouter.HandleFunc("/v1/sphere/context", handlers.HandleSphereContext)
	APIRouter.HandleFunc("/v1/sphere/send_to", handlers.HandleSphereSendTo)
	APIRouter.HandleFunc("/v1/sphere/search", handlers.HandleSphereSearch)
	APIRouter.HandleFunc("/v1/sphere/trips", handlers.HandleSphereTrips)
	APIRouter.HandleFunc("/v1/sphere/trips/impact", handlers.HandleSphereTripImpact)
	APIRouter.HandleFunc("/v1/sphere/documents", handlers.HandleSphereDocuments)
	APIRouter.HandleFunc("/v1/sphere/documents/propose_change", handlers.HandleSphereDocumentProposeChange)
	APIRouter.HandleFunc("/v1/sphere/documents/apply_change", handlers.HandleSphereDocumentApplyChange)
	APIRouter.HandleFunc("/v1/sphere/plans", handlers.HandleSpherePlans)
	APIRouter.HandleFunc("/v1/sphere/plans/task", handlers.HandleSpherePlanTask)
	APIRouter.HandleFunc("/v1/sphere/research", handlers.HandleSphereResearch)
	APIRouter.HandleFunc("/v1/sphere/research/clips", handlers.HandleSphereResearchClips)
	APIRouter.HandleFunc("/v1/sphere/research/briefing", handlers.HandleSphereResearchBriefing)
	APIRouter.HandleFunc("/v1/sphere/browser/tabs", handlers.HandleSphereBrowserTabs)
	APIRouter.HandleFunc("/v1/sphere/browser/action", handlers.HandleSphereBrowserTabAction)
	APIRouter.HandleFunc("/v1/sphere/browser/screenshot", handlers.HandleSphereBrowserScreenshot)
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
	APIRouter.HandleFunc("/v1/personal/operations", handlers.HandlePersonalOperations)
	APIRouter.HandleFunc("/v1/osint/feeds", handlers.HandleOSINTFeeds)
	APIRouter.HandleFunc("/v1/osint/briefing", handlers.HandleOSINTBriefing)
	APIRouter.HandleFunc("/v1/osint/article", handlers.HandleOSINTArticleReader)
	APIRouter.HandleFunc("/v1/osint/collectors", handlers.HandleOSINTCollectors)
	APIRouter.HandleFunc("/v1/shadow/", handlers.HandleShadow)
	APIRouter.HandleFunc("/v1/mail/config", handlers.HandleMailConfig)
	APIRouter.HandleFunc("/v1/mail/test", handlers.HandleMailTest)
	APIRouter.HandleFunc("/v1/mail/folders", handlers.HandleMailFolders)
	APIRouter.HandleFunc("/v1/mail/messages", handlers.HandleMailMessages)
	APIRouter.HandleFunc("/v1/mail/message", handlers.HandleMailMessage)
	APIRouter.HandleFunc("/v1/mail/action", handlers.HandleMailAction)
	APIRouter.HandleFunc("/v1/mail/policies", handlers.HandleMailPolicies)
	APIRouter.HandleFunc("/v1/mail/calendar", handlers.HandleMailCalendar)
	APIRouter.HandleFunc("/v1/geoint/entities", handlers.HandleGeoIntEntities)
	APIRouter.HandleFunc("/v1/geoint/relations", handlers.HandleGeoIntRelations)
	APIRouter.HandleFunc("/v1/geoint/contacts", handlers.HandleGeoIntContacts)
	APIRouter.HandleFunc("/v1/geoint/aircraft", handlers.HandleGeoIntAircraft)
	APIRouter.HandleFunc("/v1/geoint/satellites", handlers.HandleGeoIntSatellites)
	APIRouter.HandleFunc("/v1/geoint/vessels", handlers.HandleGeoIntVessels)
	APIRouter.HandleFunc("/v1/geoint/cctv", handlers.HandleGeoIntCCTV)
	APIRouter.HandleFunc("/v1/geoint/cctv/frame", handlers.HandleGeoIntCCTVFrame)
	APIRouter.HandleFunc("/v1/geoint/hazards", handlers.HandleGeoIntHazards)
	APIRouter.HandleFunc("/v1/geoint/correlations", handlers.HandleGeoIntCorrelations)
	APIRouter.HandleFunc("/v1/geoint/status", handlers.HandleGeoIntStatus)

	log.Println("✅ VGT AETHEL :: API-ROUTER BEREIT")

	// SHADOW collection is bounded and rotates through the editable source
	// catalog. AI analysis starts only when a complete 40–60 item batch exists.
	go func() {
		collect := func() {
			if enabled, _ := shadowService.Autonomy(); !enabled {
				return
			}
			if _, err := shadowService.Collect(ctx, 8); err != nil {
				log.Printf("[SHADOW] collection: %v", err)
			}
			if shadowService.Status().PendingItems >= osint.ShadowBatchMin {
				if err := handlers.RunShadowAutoAnalysis(); err != nil {
					log.Printf("[SHADOW] analysis: %v", err)
				}
			}
		}
		collect()
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collect()
			}
		}
	}()

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
	if a.operations != nil {
		a.operations.Stop()
	}
	if a.geoint != nil {
		a.geoint.Stop()
	}
	log.Println("🔴 VGT AETHEL :: SHUTDOWN")
}

// HideToTray hides the window (callable from frontend via Wails binding)
func (a *App) HideToTray() {
	platformHideWindow(a.ctx)
}

// ShowWindow brings AETHEL back from tray
func (a *App) ShowWindow() {
	platformShowWindow(a.ctx)
}

// SelectDirectory opens a native directory picker dialog and returns the selected path
func (a *App) SelectDirectory() string {
	dir, err := platformOpenDirectoryDialog(a.ctx, "Projektverzeichnis für Aethel freigeben")
	if err != nil {
		log.Printf("Failed to open directory dialog: %v", err)
		return ""
	}
	return dir
}

// SelectCodeProject grants the explicitly selected project read/write access
// for the current coding session. Aethel runtime data is never auto-selected.
func (a *App) SelectCodeProject() map[string]string {
	dir, err := platformOpenDirectoryDialog(a.ctx, "Projekt in VGT Code öffnen")
	if err != nil {
		log.Printf("VGT Code project dialog failed: %v", err)
		return map[string]string{"status": "error", "message": "Projekt-Auswahl fehlgeschlagen."}
	}
	if strings.TrimSpace(dir) == "" {
		return map[string]string{"status": "cancelled"}
	}
	if state == nil {
		return map[string]string{"status": "error", "message": "Aethel Core ist noch nicht bereit."}
	}
	if err := state.AddMount(dir, security.MountWrite, 24*time.Hour); err != nil {
		log.Printf("VGT Code project authorization failed: %v", err)
		return map[string]string{"status": "error", "message": "Projekt konnte nicht freigegeben werden."}
	}
	if err := handlers.SetCodeWorkspaceRoot(dir); err != nil {
		log.Printf("VGT Code project activation failed: %v", err)
		return map[string]string{"status": "error", "message": "Projekt konnte nicht aktiviert werden."}
	}
	a.codeMu.Lock()
	previous := a.codePath
	a.codePath = dir
	a.codeMu.Unlock()
	if previous != "" && !strings.EqualFold(previous, dir) {
		_ = state.RemoveMount(previous, security.MountWrite)
	}
	return map[string]string{"status": "success", "name": filepath.Base(dir), "path": dir}
}

func (a *App) CloseCodeProject() {
	a.codeMu.Lock()
	path := a.codePath
	a.codePath = ""
	a.codeMu.Unlock()
	handlers.ClearCodeWorkspaceRoot()
	if state != nil && path != "" {
		_ = state.RemoveMount(path, security.MountWrite)
	}
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

// WindowMinimise minimises the Wails application window.
func (a *App) WindowMinimise() {
	platformWindowMinimise(a.ctx)
}

// WindowToggleMaximise toggles between maximised and restored window state.
func (a *App) WindowToggleMaximise() {
	platformWindowToggleMaximise(a.ctx)
}

// WindowClose gracefully shuts down the Wails application.
func (a *App) WindowClose() {
	platformWindowClose(a.ctx)
}

// WindowIsMaximised returns true if the application window is currently maximised.
func (a *App) WindowIsMaximised() bool {
	return platformWindowIsMaximised(a.ctx)
}

