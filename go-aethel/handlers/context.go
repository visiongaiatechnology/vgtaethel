package handlers

import (
	"go-aethel/agent"
	"go-aethel/intelligence"
	"go-aethel/memory"
	"go-aethel/osint"
	"go-aethel/personal"
	"go-aethel/provider"
	"go-aethel/security"
	"go-aethel/skills"
	"go-aethel/system"
)

type VoiceService interface {
	SynthesizeWithFallback(text, voiceID string) ([]byte, string, error)
	SynthesizeJSON(text, voiceID string) (any, error)
	Transcribe(audioBytes []byte, filename string) (string, error)
	AvailableVoicesPayload() any
	GetHealthStatus() map[string]interface{}
}

type appState struct {
	mu             interface{} // placeholder if needed
	apiKey         string
	openaiAPIKey   string
	deepseekAPIKey string
	geminiAPIKey   string
	claudeAPIKey   string
	mountedDirs    []string
	mounts         []security.MountGrant
	guard          *security.SecurityGuard
	leases         *security.LeaseManager
	audit          *security.AuditLogger
	policy         *security.PolicyEngine
	approvals      *security.ApprovalManager
	skills         *skills.SkillRegistry
	providers      *provider.ProviderRegistry
	memory         *memory.LocalMemoryStore
	voice          VoiceService
	vault          *security.SecretVault
	tasks          *agent.TaskEngine
	runs           *agent.RunEngine
	personal       *personal.PersonalStore
	release        *system.ReleaseService
	osint          *osint.OSINTEngine
	intel          *intelligence.IntelligenceStore
	intelSources   *intelligence.IntelligenceSourceRegistry
	intelMonitor   *osint.GlobalWatchMonitor

	// Methods / Helper Functions from main package
	getAPIKey      func() string
	getOpenAIKey   func() string
	getDeepSeekKey func() string
	getGeminiKey   func() string
	getClaudeKey   func() string
	saveConfig     func(apiKey, oKey, dsKey, gemKey, claudeKey string) error
	GetMountedDirs func() []string
	GetMounts      func() []security.MountGrant
}

var state *appState

func InitState(
	guard *security.SecurityGuard,
	leases *security.LeaseManager,
	audit *security.AuditLogger,
	policy *security.PolicyEngine,
	approvals *security.ApprovalManager,
	skillsRegistry *skills.SkillRegistry,
	providersRegistry *provider.ProviderRegistry,
	memoryStore *memory.LocalMemoryStore,
	voiceRegistry VoiceService,
	vault *security.SecretVault,
	tasksEngine *agent.TaskEngine,
	runsEngine *agent.RunEngine,
	personalStore *personal.PersonalStore,
	releaseService *system.ReleaseService,
	osintEngine *osint.OSINTEngine,
	intelStore *intelligence.IntelligenceStore,
	intelSourcesRegistry *intelligence.IntelligenceSourceRegistry,
	intelMonitorMonitor *osint.GlobalWatchMonitor,
	getAPIKeyFn func() string,
	getOpenAIKeyFn func() string,
	getDeepSeekKeyFn func() string,
	getGeminiKeyFn func() string,
	getClaudeKeyFn func() string,
	saveConfigFn func(apiKey, oKey, dsKey, gemKey, claudeKey string) error,
	getMountedDirsFn func() []string,
	getMountsFn func() []security.MountGrant,
) {
	state = &appState{
		guard:          guard,
		leases:         leases,
		audit:          audit,
		policy:         policy,
		approvals:      approvals,
		skills:         skillsRegistry,
		providers:      providersRegistry,
		memory:         memoryStore,
		voice:          voiceRegistry,
		vault:          vault,
		tasks:          tasksEngine,
		runs:           runsEngine,
		personal:       personalStore,
		release:        releaseService,
		osint:          osintEngine,
		intel:          intelStore,
		intelSources:   intelSourcesRegistry,
		intelMonitor:   intelMonitorMonitor,
		getAPIKey:      getAPIKeyFn,
		getOpenAIKey:   getOpenAIKeyFn,
		getDeepSeekKey: getDeepSeekKeyFn,
		getGeminiKey:   getGeminiKeyFn,
		getClaudeKey:   getClaudeKeyFn,
		saveConfig:     saveConfigFn,
		GetMountedDirs: getMountedDirsFn,
		GetMounts:      getMountsFn,
	}
}

func (s *appState) GetAPIKey() string {
	if s.getAPIKey != nil {
		return s.getAPIKey()
	}
	return ""
}

func (s *appState) GetOpenAIKey() string {
	if s.getOpenAIKey != nil {
		return s.getOpenAIKey()
	}
	return ""
}

func (s *appState) GetDeepSeekKey() string {
	if s.getDeepSeekKey != nil {
		return s.getDeepSeekKey()
	}
	return ""
}

func (s *appState) GetGeminiKey() string {
	if s.getGeminiKey != nil {
		return s.getGeminiKey()
	}
	return ""
}

func (s *appState) GetClaudeKey() string {
	if s.getClaudeKey != nil {
		return s.getClaudeKey()
	}
	return ""
}
