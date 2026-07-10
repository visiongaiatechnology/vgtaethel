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
)

const configFile = "./vgt_workspace/aethel_config.json"

type Config struct {
	APIKey         string       `json:"api_key"`
	OpenAIAPIKey   string       `json:"openai_api_key,omitempty"`
	DeepSeekAPIKey string       `json:"deepseek_api_key,omitempty"`
	GeminiAPIKey   string       `json:"gemini_api_key,omitempty"`
	ClaudeAPIKey   string       `json:"claude_api_key,omitempty"`
	MountedDirs    []string     `json:"mounted_dirs,omitempty"`
	Mounts         []MountGrant `json:"mounts,omitempty"`
}

type MountAccess string

const (
	MountRead  MountAccess = "read"
	MountWrite MountAccess = "write"
)

type MountGrant struct {
	Path      string      `json:"path"`
	Access    MountAccess `json:"access"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt time.Time   `json:"expires_at"`
}

type AppState struct {
	mu             sync.RWMutex
	apiKey         string
	openaiAPIKey   string
	deepseekAPIKey string
	geminiAPIKey   string
	claudeAPIKey   string
	mountedDirs    []string
	mounts         []MountGrant
	guard          *SecurityGuard
	leases         *LeaseManager
	audit          *AuditLogger
	policy         *PolicyEngine
	approvals      *ApprovalManager
	skills         *SkillRegistry
	providers      *ProviderRegistry
	memory         *LocalMemoryStore
	voice          *VoiceRegistry
	vault          *SecretVault
	tasks          *TaskEngine
	runs           *RunEngine
	personal       *PersonalStore
	release        *ReleaseService
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
	if dec, err := decryptGCM(val); err == nil && dec != "" {
		return dec
	}
	return val
}

func loadConfig() (string, string, string, string, string, []string, []MountGrant) {
	key := os.Getenv("GROQ_API_KEY")
	oKey := os.Getenv("OPENAI_API_KEY")
	dsKey := os.Getenv("DEEPSEEK_API_KEY")
	var gemKey, claudeKey string
	var mountedDirs []string
	var mounts []MountGrant

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
				encKey, _ := encryptGCM(key)
				encOKey, _ := encryptGCM(oKey)
				encDsKey, _ := encryptGCM(dsKey)
				encGemKey, _ := encryptGCM(gemKey)
				encClaudeKey, _ := encryptGCM(claudeKey)
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
		mounts = []MountGrant{}
		for _, dir := range mountedDirs {
			if strings.TrimSpace(dir) != "" {
				mounts = append(mounts, MountGrant{Path: dir, Access: MountRead, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(8 * time.Hour)})
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
	encKey, err := encryptGCM(s.apiKey)
	if err != nil {
		return err
	}
	if s.apiKey == "local" {
		encKey = "local"
	}
	encOKey, err := encryptGCM(s.openaiAPIKey)
	if err != nil {
		return err
	}
	encDsKey, err := encryptGCM(s.deepseekAPIKey)
	if err != nil {
		return err
	}
	encGemKey, err := encryptGCM(s.geminiAPIKey)
	if err != nil {
		return err
	}
	encClaudeKey, err := encryptGCM(s.claudeAPIKey)
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

func mountedDirsFromGrants(mounts []MountGrant) []string {
	paths := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		paths = append(paths, mount.Path)
	}
	return paths
}

func (s *AppState) activeMountsLocked() []MountGrant {
	now := time.Now().UTC()
	active := make([]MountGrant, 0, len(s.mounts))
	for _, mount := range s.mounts {
		if mount.Path != "" && now.Before(mount.ExpiresAt) {
			active = append(active, mount)
		}
	}
	return active
}

func (s *AppState) AddMount(dir string, access MountAccess, duration time.Duration) error {
	if access != MountRead && access != MountWrite || duration < time.Minute || duration > 24*time.Hour {
		return errors.New("invalid mount access or duration")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	absDir, err := canonicalDir(dir)
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
		s.mounts = append(s.mounts, MountGrant{Path: absDir, Access: access, CreatedAt: now, ExpiresAt: now.Add(duration)})
	}
	s.mountedDirs = mountedDirsFromGrants(s.activeMountsLocked())
	return s.persistConfigLocked()
}

func (s *AppState) GetMountedDirs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return mountedDirsFromGrants(s.activeMountsLocked())
}

func (s *AppState) GetMounts() []MountGrant {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]MountGrant(nil), s.activeMountsLocked()...)
}

func (s *AppState) MountAllows(path string, access MountAccess) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	canonicalPath, err := canonicalTarget(path)
	if err != nil {
		return false
	}
	for _, mount := range s.activeMountsLocked() {
		if access == MountWrite && mount.Access != MountWrite {
			continue
		}
		if base, err := canonicalDir(mount.Path); err == nil && isPathInside(base, canonicalPath) {
			return true
		}
	}
	return false
}

func (s *AppState) getAPIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.apiKey
}

func (s *AppState) getOpenAIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.openaiAPIKey
}

func (s *AppState) getDeepSeekKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deepseekAPIKey
}

func (s *AppState) getGeminiKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.geminiAPIKey
}

func (s *AppState) getClaudeKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.claudeAPIKey
}

func (s *AppState) isConfigured() bool {
	groqKey := s.getAPIKey()
	dsKey := s.getDeepSeekKey()
	gemKey := s.getGeminiKey()
	clKey := s.getClaudeKey()
	oaiKey := s.getOpenAIKey()
	return groqKey == "local" ||
		(groqKey != "" && strings.HasPrefix(groqKey, "gsk_")) ||
		(dsKey != "" && strings.HasPrefix(dsKey, "sk-")) ||
		(gemKey != "" && strings.HasPrefix(gemKey, "AIza")) ||
		(clKey != "" && strings.HasPrefix(clKey, "sk-ant-")) ||
		(oaiKey != "" && strings.HasPrefix(oaiKey, "sk-"))
}
