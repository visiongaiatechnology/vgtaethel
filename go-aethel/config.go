package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go-aethel/agent"
	"go-aethel/intelligence"
	"go-aethel/memory"
	"go-aethel/osint"
	"go-aethel/personal"
	"go-aethel/provider"
	"go-aethel/security"
	"go-aethel/skills"
	"go-aethel/system"
	"go-aethel/voice"
)

const configFile = "./vgt_workspace/aethel_config.json"

type Config struct {
	APIKey         string       `json:"api_key"`
	OpenAIAPIKey   string       `json:"openai_api_key,omitempty"`
	DeepSeekAPIKey string       `json:"deepseek_api_key,omitempty"`
	GeminiAPIKey   string       `json:"gemini_api_key,omitempty"`
	ClaudeAPIKey   string       `json:"claude_api_key,omitempty"`
	MountedDirs    []string     `json:"mounted_dirs,omitempty"`
	Mounts         []security.MountGrant `json:"mounts,omitempty"`
}

type AppState struct {
	mu             sync.RWMutex
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
	voice          *voice.VoiceRegistry
	vault          *security.SecretVault
	tasks          *agent.TaskEngine
	runs           *agent.RunEngine
	personal       *personal.PersonalStore
	release        *system.ReleaseService
	osint          *osint.OSINTEngine
	intel          *intelligence.IntelligenceStore
	intelSources   *intelligence.IntelligenceSourceRegistry
	intelMonitor   *osint.GlobalWatchMonitor
}

var state *AppState

func isPlainKey(val, prefix string) bool {
	if val == "" || val == "local" {
		return true
	}
	return strings.HasPrefix(val, prefix)
}

func maybeDecrypt(val string) string {
	if val == "" || val == "local" {
		return val
	}
	plainKnownPrefixes := []string{"gsk_", "sk-", "AIza", "sk-ant-"}
	for _, p := range plainKnownPrefixes {
		if strings.HasPrefix(val, p) {
			return val
		}
	}
	if dec, err := security.DecryptGCM(val); err == nil && dec != "" {
		return dec
	}
	return val
}

func loadConfig() (string, string, string, string, string, []string, []security.MountGrant) {
	key := os.Getenv("GROQ_API_KEY")
	oKey := os.Getenv("OPENAI_API_KEY")
	dsKey := os.Getenv("DEEPSEEK_API_KEY")
	var gemKey, claudeKey string
	var mountedDirs []string
	var mounts []security.MountGrant

	data, err := os.ReadFile(configFile)
	if err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err == nil {
			if key == "" {
				key = maybeDecrypt(cfg.APIKey)
			}
			if oKey == "" {
				oKey = maybeDecrypt(cfg.OpenAIAPIKey)
			}
			if dsKey == "" {
				dsKey = maybeDecrypt(cfg.DeepSeekAPIKey)
			}
			gemKey = maybeDecrypt(cfg.GeminiAPIKey)
			claudeKey = maybeDecrypt(cfg.ClaudeAPIKey)
			mountedDirs = cfg.MountedDirs
			mounts = cfg.Mounts

			hasPlainKey := (cfg.APIKey != "" && cfg.APIKey != "local" && !strings.HasPrefix(cfg.APIKey, "AQ") && isPlainKey(cfg.APIKey, "gsk_")) ||
				(cfg.OpenAIAPIKey != "" && isPlainKey(cfg.OpenAIAPIKey, "sk-")) ||
				(cfg.DeepSeekAPIKey != "" && isPlainKey(cfg.DeepSeekAPIKey, "sk-")) ||
				(cfg.GeminiAPIKey != "" && isPlainKey(cfg.GeminiAPIKey, "AIza")) ||
				(cfg.ClaudeAPIKey != "" && isPlainKey(cfg.ClaudeAPIKey, "sk-ant-"))
			if hasPlainKey {
				log.Println("[CONFIG] Detected plaintext keys — encrypting and migrating to AES-256-GCM...")
				encKey, _ := security.EncryptGCM(key)
				encOKey, _ := security.EncryptGCM(oKey)
				encDsKey, _ := security.EncryptGCM(dsKey)
				encGemKey, _ := security.EncryptGCM(gemKey)
				encClaudeKey, _ := security.EncryptGCM(claudeKey)
				if key == "local" {
					encKey = "local"
				}
				migratedCfg := Config{
					APIKey: encKey, OpenAIAPIKey: encOKey,
					DeepSeekAPIKey: encDsKey, GeminiAPIKey: encGemKey,
					ClaudeAPIKey: encClaudeKey, MountedDirs: mountedDirs,
				}
				if migratedData, err := json.MarshalIndent(migratedCfg, "", "  "); err == nil {
					_ = os.WriteFile(configFile, migratedData, 0600)
					log.Println("[CONFIG] Migration complete. Keys are now encrypted at rest.")
				}
			}
		}
	}
	if mountedDirs == nil {
		mountedDirs = []string{}
	}
	if mounts == nil {
		mounts = []security.MountGrant{}
		for _, dir := range mountedDirs {
			if strings.TrimSpace(dir) != "" {
				mounts = append(mounts, security.MountGrant{Path: dir, Access: security.MountRead, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(8 * time.Hour)})
			}
		}
	}
	return key, oKey, dsKey, gemKey, claudeKey, mountedDirs, mounts
}

func (s *AppState) saveConfig(key, oKey, dsKey, gemKey, claudeKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.apiKey = key
	s.openaiAPIKey = oKey
	s.deepseekAPIKey = dsKey
	s.geminiAPIKey = gemKey
	s.claudeAPIKey = claudeKey

	return s.persistConfigLocked()
}

func (s *AppState) persistConfigLocked() error {
	encKey, err := security.EncryptGCM(s.apiKey)
	if err != nil {
		return err
	}
	if s.apiKey == "local" {
		encKey = "local"
	}
	encOKey, err := security.EncryptGCM(s.openaiAPIKey)
	if err != nil {
		return err
	}
	encDsKey, err := security.EncryptGCM(s.deepseekAPIKey)
	if err != nil {
		return err
	}
	encGemKey, err := security.EncryptGCM(s.geminiAPIKey)
	if err != nil {
		return err
	}
	encClaudeKey, err := security.EncryptGCM(s.claudeAPIKey)
	if err != nil {
		return err
	}
	mounts := s.activeMountsLocked()
	cfg := Config{APIKey: encKey, OpenAIAPIKey: encOKey, DeepSeekAPIKey: encDsKey, GeminiAPIKey: encGemKey, ClaudeAPIKey: encClaudeKey, MountedDirs: mountedDirsFromGrants(mounts), Mounts: mounts}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configFile), 0700); err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0600)
}

func mountedDirsFromGrants(mounts []security.MountGrant) []string {
	paths := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		paths = append(paths, mount.Path)
	}
	return paths
}

func (s *AppState) activeMountsLocked() []security.MountGrant {
	now := time.Now().UTC()
	active := make([]security.MountGrant, 0, len(s.mounts))
	for _, mount := range s.mounts {
		if mount.Path != "" && now.Before(mount.ExpiresAt) {
			active = append(active, mount)
		}
	}
	return active
}

func (s *AppState) AddMount(dir string, access security.MountAccess, duration time.Duration) error {
	if access != security.MountRead && access != security.MountWrite || duration < time.Minute || duration > 24*time.Hour {
		return errors.New("invalid mount access or duration")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	absDir, err := security.CanonicalDir(dir)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	updated := false
	for i := range s.mounts {
		if s.mounts[i].Path == absDir && s.mounts[i].Access == access {
			s.mounts[i].ExpiresAt = now.Add(duration)
			updated = true
			break
		}
	}
	if !updated {
		s.mounts = append(s.mounts, security.MountGrant{Path: absDir, Access: access, CreatedAt: now, ExpiresAt: now.Add(duration)})
	}
	s.mountedDirs = mountedDirsFromGrants(s.activeMountsLocked())
	return s.persistConfigLocked()
}

func (s *AppState) GetMountedDirs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return mountedDirsFromGrants(s.activeMountsLocked())
}

func (s *AppState) GetMounts() []security.MountGrant {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]security.MountGrant(nil), s.activeMountsLocked()...)
}

func (s *AppState) MountAllows(path string, access security.MountAccess) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	canonicalPath, err := security.CanonicalTarget(path)
	if err != nil {
		return false
	}
	for _, mount := range s.activeMountsLocked() {
		if access == security.MountWrite && mount.Access != security.MountWrite {
			continue
		}
		if base, err := security.CanonicalDir(mount.Path); err == nil && security.IsPathInside(base, canonicalPath) {
			return true
		}
	}
	return false
}



func (s *AppState) GetAPIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.apiKey
}

func (s *AppState) GetOpenAIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.openaiAPIKey
}

func (s *AppState) GetDeepSeekKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deepseekAPIKey
}

func (s *AppState) GetGeminiKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.geminiAPIKey
}

func (s *AppState) GetClaudeKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.claudeAPIKey
}

func (s *AppState) isConfigured() bool {
	groqKey := s.GetAPIKey()
	dsKey := s.GetDeepSeekKey()
	gemKey := s.GetGeminiKey()
	clKey := s.GetClaudeKey()
	oaiKey := s.GetOpenAIKey()
	return groqKey == "local" ||
		(groqKey != "" && strings.HasPrefix(groqKey, "gsk_")) ||
		(dsKey != "" && strings.HasPrefix(dsKey, "sk-")) ||
		(gemKey != "" && strings.HasPrefix(gemKey, "AIza")) ||
		(clKey != "" && strings.HasPrefix(clKey, "sk-ant-")) ||
		(oaiKey != "" && strings.HasPrefix(oaiKey, "sk-"))
}
