package handlers

// Exported aliases for router wiring (handlers remain implementable as unexported funcs).

import (
	"encoding/json"
	"net/http"
	"runtime"

	"go-aethel/provider"
)

var (
	HandleChat               = handleChat
	HandleChecklist          = handleChecklist
	HandleToolExecute        = handleToolExecute
	HandleBrowserScreenshot  = handleBrowserScreenshot
	HandleAudioSpeech        = handleAudioSpeech
	HandleAudioVoices        = handleAudioVoices
	HandleAudioTranscribe    = handleAudioTranscribe
	HandleChatHistory        = handleChatHistory
	HandleChatSessions       = handleChatSessions
	HandleChatSessionsLoad   = handleChatSessionsLoad
	HandleChatSessionsSave   = handleChatSessionsSave
	HandleChatSessionsDelete = handleChatSessionsDelete
	HandleKernelTasksPath    = handleKernelTasksPath
	HandleRuns               = handleRuns
	HandleRunsPath           = handleRunsPath
	HandleArtifacts          = handleArtifacts
	HandleSecurityLeases     = handleSecurityLeases
	HandleSecurityAudit      = handleSecurityAudit
	HandleSecurityStatus     = handleSecurityStatus
	HandleMemory             = handleMemory
	HandleMemoryExplain      = handleMemoryExplain
	HandleMemoryExport       = handleMemoryExport
	HandleMemorySearch       = handleMemorySearch
	HandleAudioHealth        = handleAudioHealth
	HandleAudioTest          = handleAudioTest
	HandleViewportScreenshot = handleViewportScreenshot
	HandleViewportStatus     = handleViewportStatus
	HandleWeather            = handleWeather
	HandleMarketQuotes       = handleMarketQuotes
	HandleSphereDocument     = handleSphereDocument
	HandleSphereWorkspace    = handleSphereWorkspace
	HandleSecrets            = handleSecrets
	HandleSettings           = handleSettings
	HandleSettingsReset      = handleSettingsReset
	HandleCosts              = handleCosts
	HandlePersonas           = handlePersonas
	HandleOSINTFeeds         = handleOSINTFeeds
	HandleOSINTBriefing      = handleOSINTBriefing
	HandleOSINTArticleReader = handleOSINTArticleReader
	HandleOSINTCollectors    = handleOSINTCollectors
	HandleMailConfig         = handleMailConfig
	HandleMailTest           = handleMailTest
	HandleSetup              = handleSetup
	HandleModels             = handleModels
	HandleProviderHealth     = handleProviderHealth
)

// HandleHealth reports readiness for the UI status chip.
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := "SETUP_REQUIRED"
	oReady, dsReady, ollamaReady := "false", "false", "false"
	if state != nil {
		groqKey := state.GetAPIKey()
		openaiKey := state.GetOpenAIKey()
		dsKey := state.GetDeepSeekKey()
		// Match main AppState.isConfigured: any valid provider (or local) unlocks the UI.
		if groqKey == "local" || provider.HasConfiguredProvider(state) {
			status = "READY"
		}
		if openaiKey != "" {
			oReady = "true"
		}
		if dsKey != "" {
			dsReady = "true"
		}
	}
	// Ollama probe is cached + hard-timeouted; never block boot on local daemon state.
	if models := provider.GetLocalOllamaModels(); len(models) > 0 {
		ollamaReady = "true"
	}
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":         status,
		"mode":           "STREAMING",
		"core":           "GO-CORTEX",
		"openai_ready":   oReady,
		"deepseek_ready": dsReady,
		"ollama_ready":   ollamaReady,
		"os":             runtime.GOOS,
	})
}
