package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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

	memoryStore := NewLocalMemoryStore()
	registry := NewSkillRegistry()
	registry.Register(&ExecuteCommandSkill{})
	registry.Register(&ReadFileSkill{})
	registry.Register(&WriteFileSkill{})
	registry.Register(&ReplaceFileContentSkill{})
	registry.Register(&SetChecklistSkill{})
	registry.Register(&UpdateChecklistSkill{})
	registry.Register(&MemorySaveSkill{Store: memoryStore})
	registry.Register(&MemoryRecallSkill{Store: memoryStore})
	registry.Register(&WebBrowserSkill{})
	registry.Register(&GUIControlSkill{})
	registry.Register(&ListDirSkill{})
	registry.Register(&MountFolderSkill{})
	registry.Register(&ExternalAgentHandoffSkill{})

	gKey, oKey, dsKey, mDirs := loadConfig()

	guard := NewSecurityGuard()
	leases := NewLeaseManager("./vgt_workspace/active_leases.json")
	audit := NewAuditLogger("./vgt_workspace/security_audit.json")
	policy := NewPolicyEngine(guard, leases, audit)

	voiceRegistry := NewVoiceRegistry()
	go voiceRegistry.LoadLocalVoices()

	vault, err := NewSecretVault("./vgt_workspace/secret_vault.enc", "./vgt_workspace/vault.key")
	if err != nil {
		log.Fatalf("Failed to initialize secret vault: %v", err)
	}

	taskEngine := NewTaskEngine("./vgt_workspace/tasks.json")
	_ = taskEngine.Load()

	state = &AppState{
		apiKey:          gKey,
		openaiAPIKey:    oKey,
		deepseekAPIKey:  dsKey,
		mountedDirs:     mDirs,
		guard:           guard,
		leases:          leases,
		audit:           audit,
		policy:          policy,
		skills:          registry,
		memory:          memoryStore,
		voice:           voiceRegistry,
		vault:           vault,
		tasks:           taskEngine,
	}

	state.tasks.Start()

	// Wire all API handlers — done before the window shows
	APIRouter = http.NewServeMux()
	APIRouter.HandleFunc("/health", handleHealth)
	APIRouter.HandleFunc("/v1/setup", handleSetup)
	APIRouter.HandleFunc("/v1/models", handleModels)
	APIRouter.HandleFunc("/v1/chat", handleChat)
	APIRouter.HandleFunc("/v1/chat/checklist", handleChecklist)
	APIRouter.HandleFunc("/v1/tools/execute", handleToolExecute)
	APIRouter.HandleFunc("/browser/screenshot.png", handleBrowserScreenshot)
	APIRouter.HandleFunc("/v1/audio/speech", handleAudioSpeech)
	APIRouter.HandleFunc("/v1/audio/voices", handleAudioVoices)
	APIRouter.HandleFunc("/v1/audio/transcribe", handleAudioTranscribe)
	APIRouter.HandleFunc("/v1/kernel/logs", handleKernelLogs)
	APIRouter.HandleFunc("/v1/chat/history", handleChatHistory)
	APIRouter.HandleFunc("/v1/chat/sessions", handleChatSessions)
	APIRouter.HandleFunc("/v1/chat/sessions/load", handleChatSessionsLoad)
	APIRouter.HandleFunc("/v1/chat/sessions/save", handleChatSessionsSave)
	APIRouter.HandleFunc("/v1/chat/sessions/delete", handleChatSessionsDelete)
	APIRouter.HandleFunc("/v1/kernel/tasks/", handleKernelTasksPath)
	APIRouter.HandleFunc("/v1/security/leases", handleSecurityLeases)
	APIRouter.HandleFunc("/v1/security/audit", handleSecurityAudit)
	APIRouter.HandleFunc("/v1/security/status", handleSecurityStatus)
	APIRouter.HandleFunc("/v1/memory", handleMemory)
	APIRouter.HandleFunc("/v1/memory/search", handleMemorySearch)
	APIRouter.HandleFunc("/v1/audio/health", handleAudioHealth)
	APIRouter.HandleFunc("/v1/audio/test", handleAudioTest)
	APIRouter.HandleFunc("/v1/viewport/screenshot", handleViewportScreenshot)
	APIRouter.HandleFunc("/v1/viewport/status", handleViewportStatus)
	APIRouter.HandleFunc("/v1/secrets", handleSecrets)
	APIRouter.HandleFunc("/v1/settings", handleSettings)
	APIRouter.HandleFunc("/v1/settings/reset", handleSettingsReset)

	log.Println("✅ VGT AETHEL :: API-ROUTER BEREIT")
}

// domReady is called when the frontend DOM is loaded — no navigation needed
func (a *App) domReady(ctx context.Context) {
	log.Println("🌐 VGT AETHEL :: DOM BEREIT")
}

// beforeClose hides to system tray instead of quitting
func (a *App) beforeClose(ctx context.Context) bool {
	runtime.Hide(ctx)
	return true
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
	return "v0.5.0-alpha"
}

// isAPIPath returns true for paths that should be routed to Go handlers
func isAPIPath(p string) bool {
	return p == "/health" ||
		strings.HasPrefix(p, "/v1/") ||
		strings.HasPrefix(p, "/browser/")
}