// STATUS: DIAMANT VGT SUPREME
package system

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"go-aethel/security")

type ReleasePreferences struct {
	CrashReportingOptIn bool      `json:"crash_reporting_opt_in"`
	UpdateManifestURL   string    `json:"update_manifest_url,omitempty"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type SignedUpdateManifest struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature"`
}

type ReleaseService struct {
	mu              sync.RWMutex
	path, crashPath string
}

func NewReleaseService(base string) *ReleaseService {
	return &ReleaseService{path: filepath.Join(base, "release_preferences.json"), crashPath: filepath.Join(base, "crash_reports.jsonl")}
}

func (s *ReleaseService) Preferences() (ReleasePreferences, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, _, err := security.ReadSealedFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return ReleasePreferences{}, nil
	}
	if err != nil {
		return ReleasePreferences{}, err
	}
	var preferences ReleasePreferences
	return preferences, json.Unmarshal(data, &preferences)
}

func (s *ReleaseService) SavePreferences(preferences ReleasePreferences) error {
	u, err := url.Parse(strings.TrimSpace(preferences.UpdateManifestURL))
	if preferences.UpdateManifestURL != "" && (err != nil || u.Scheme != "https" || u.Host == "") {
		return errors.New("update manifest URL must use HTTPS")
	}
	preferences.UpdateManifestURL = strings.TrimSpace(preferences.UpdateManifestURL)
	preferences.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(preferences)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return security.WriteSealedFile(s.path, data)
}

func (s *ReleaseService) CheckForUpdate() (map[string]interface{}, error) {
	prefs, err := s.Preferences()
	if err != nil {
		return nil, err
	}
	if prefs.UpdateManifestURL == "" {
		return map[string]interface{}{"configured": false, "current_version": ProductVersion}, nil
	}
	publicKeyText := os.Getenv("AETHEL_UPDATE_PUBLIC_KEY")
	if publicKeyText == "" {
		return nil, errors.New("release public key is not configured")
	}
	publicKey, err := base64.StdEncoding.DecodeString(publicKeyText)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("release public key is invalid")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(prefs.UpdateManifestURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("update manifest request failed")
	}
	var manifest SignedUpdateManifest
	if err := json.NewDecoder(http.MaxBytesReader(nil, response.Body, 32<<10)).Decode(&manifest); err != nil {
		return nil, errors.New("update manifest is invalid")
	}
	if manifest.Version == "" || manifest.URL == "" || manifest.SHA256 == "" || manifest.Signature == "" {
		return nil, errors.New("update manifest is incomplete")
	}
	downloadURL, err := url.Parse(manifest.URL)
	if err != nil || downloadURL.Scheme != "https" || downloadURL.Host == "" {
		return nil, errors.New("update artifact URL must use HTTPS")
	}
	signature, err := base64.StdEncoding.DecodeString(manifest.Signature)
	if err != nil {
		return nil, errors.New("update signature is invalid")
	}
	signed := manifest.Version + "\n" + manifest.URL + "\n" + strings.ToLower(manifest.SHA256)
	if !ed25519.Verify(ed25519.PublicKey(publicKey), []byte(signed), signature) {
		return nil, errors.New("update signature verification failed")
	}
	return map[string]interface{}{"configured": true, "current_version": ProductVersion, "available_version": manifest.Version, "update_available": manifest.Version != ProductVersion, "download_url": manifest.URL, "sha256": strings.ToLower(manifest.SHA256), "auto_install": false}, nil
}

func (s *ReleaseService) CapturePanic(value interface{}) {
	prefs, err := s.Preferences()
	if err != nil || !prefs.CrashReportingOptIn {
		return
	}
	record := map[string]interface{}{"occurred_at": time.Now().UTC(), "panic": "application panic", "stack": string(debug.Stack())}
	data, err := json.Marshal(record)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = appendSealedJSONLine(s.crashPath, data)
}

func appendSealedJSONLine(path string, item []byte) error {
	var lines [][]byte
	data, _, err := security.ReadSealedFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(data) > 0 {
		_ = json.Unmarshal(data, &lines)
	}
	lines = append(lines, item)
	if len(lines) > 10 {
		lines = lines[len(lines)-10:]
	}
	encoded, err := json.Marshal(lines)
	if err != nil {
		return err
	}
	return security.WriteSealedFile(path, encoded)
}






