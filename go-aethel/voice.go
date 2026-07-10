package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// VoiceProfile represents a configured speech engine voice option
type VoiceProfile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "sherpa" | "local_sapi5"
	Gender    string `json:"gender,omitempty"`
	Available bool   `json:"available"`
	Offline   bool   `json:"offline"`
	Language  string `json:"language,omitempty"`
}

// VoiceOutputProvider translates text to audio stream
type VoiceOutputProvider interface {
	Synthesize(text string, voice string) ([]byte, string, error)
	HealthCheck() bool
}

// TranscriptionProvider transcribes audio binary
type TranscriptionProvider interface {
	Transcribe(audioBytes []byte, filename string) (string, error)
	HealthCheck() bool
}

// Local Windows SAPI5 Speech Synthesizer Provider (OPTIONAL FALLBACK)
type Sapi5TTSProvider struct {
	mu      sync.Mutex
	enabled bool
}

func (s *Sapi5TTSProvider) Synthesize(text string, voice string) ([]byte, string, error) {
	if !s.enabled {
		return nil, "", fmt.Errorf("SAPI5 is disabled")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tempWav := filepath.Join(os.TempDir(), "aethel_speech.wav")
	_ = os.Remove(tempWav)

	pitch := "default"
	rate := "default"

	switch voice {
	case "sapi5-male":
		pitch = "-15%"
		rate = "+10%"
	case "sapi5-female":
		pitch = "+15%"
		rate = "+20%"
	case "sapi5-neutral":
		pitch = "default"
		rate = "+15%"
	}

	escapedText := strings.ReplaceAll(text, "&", "&amp;")
	escapedText = strings.ReplaceAll(escapedText, "<", "&lt;")
	escapedText = strings.ReplaceAll(escapedText, ">", "&gt;")
	escapedText = strings.ReplaceAll(escapedText, "\"", "&quot;")
	escapedText = strings.ReplaceAll(escapedText, "'", "&apos;")

	ssml := fmt.Sprintf("<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='de-DE'><prosody pitch='%s' rate='%s'>%s</prosody></speak>", pitch, rate, escapedText)

	psScript := fmt.Sprintf(`
		try {
			Add-Type -AssemblyName System.Speech;
			$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer;
			
			$targetGender = $null;
			if ('%[1]s' -eq 'sapi5-male') {
				$targetGender = 'Male';
			} elseif ('%[1]s' -eq 'sapi5-female') {
				$targetGender = 'Female';
			}
			
			$selected = $null;
			if ($targetGender) {
				$selected = $synth.GetInstalledVoices() | Where-Object { 
					$_.VoiceInfo.Culture.TwoLetterISOLanguageName -eq 'de' -and $_.VoiceInfo.Gender.ToString() -eq $targetGender 
				} | Select-Object -First 1;
			}
			if (-not $selected) {
				$selected = $synth.GetInstalledVoices() | Where-Object { $_.VoiceInfo.Culture.TwoLetterISOLanguageName -eq 'de' } | Select-Object -First 1;
			}
			
			if ($selected) {
				$synth.SelectVoice($selected.VoiceInfo.Name);
			}
			
			$synth.SetOutputToWaveFile('%[2]s');
			$synth.SpeakSsml('%[3]s');
		} catch {
			Write-Error $_.Exception.Message;
			exit 1;
		} finally {
			if ($synth) { $synth.Dispose() }
		}
	`, strings.ReplaceAll(voice, "'", "''"), tempWav, strings.ReplaceAll(ssml, "'", "''"))

	cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	if err != nil {
		return nil, "", fmt.Errorf("PowerShell SAPI5 failed: %v", err)
	}

	wavData, err := os.ReadFile(tempWav)
	if err != nil {
		return nil, "", err
	}

	return wavData, "audio/wav", nil
}

func (s *Sapi5TTSProvider) HealthCheck() bool {
	return s.enabled
}

// Groq Whisper Speech-To-Text Provider (for Chat/Cloud LLM transcription)
type GroqWhisperProvider struct{}

func (g *GroqWhisperProvider) Transcribe(audioBytes []byte, filename string) (string, error) {
	apiKey := state.getAPIKey()
	if apiKey == "" {
		apiKey = state.getOpenAIKey()
	}
	if apiKey == "" {
		return "", fmt.Errorf("Groq API key not configured")
	}

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(fileWriter, bytes.NewReader(audioBytes))
	if err != nil {
		return "", err
	}

	_ = bodyWriter.WriteField("model", "whisper-large-v3-turbo")
	_ = bodyWriter.WriteField("language", "de")

	err = bodyWriter.Close()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.groq.com/openai/v1/audio/transcriptions", bodyBuf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", bodyWriter.FormDataContentType())

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Groq Whisper API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Text, nil
}

func (g *GroqWhisperProvider) HealthCheck() bool {
	apiKey := state.getAPIKey()
	if apiKey == "" {
		apiKey = state.getOpenAIKey()
	}
	return apiKey != ""
}

// VoiceRegistry coordinates speech interfaces and fallbacks
// PRIMARY: Sherpa-ONNX lokal
// FALLBACK: SAPI5 (deaktiviert by default)
// STT: Groq Whisper (cloud)
type VoiceRegistry struct {
	mu          sync.RWMutex
	sherpa      *SherpaVoiceEngine
	sapi5TTS    *Sapi5TTSProvider
	groqSTT     *GroqWhisperProvider
	sapi5Voices []VoiceProfile
}

// NewVoiceRegistry creates a voice registry with Sherpa as primary engine
func NewVoiceRegistry(sherpa *SherpaVoiceEngine) *VoiceRegistry {
	return &VoiceRegistry{
		sherpa:      sherpa,
		sapi5TTS:    &Sapi5TTSProvider{enabled: false}, // disabled by default
		groqSTT:     &GroqWhisperProvider{},
		sapi5Voices: make([]VoiceProfile, 0),
	}
}

// EnableSAPI5 activates the legacy SAPI5 fallback for TTS
func (vr *VoiceRegistry) EnableSAPI5() {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	vr.sapi5TTS.enabled = true
	vr.scanLocalSAPI5Voices()
}

// DisableSAPI5 deactivates the SAPI5 fallback
func (vr *VoiceRegistry) DisableSAPI5() {
	vr.mu.Lock()
	defer vr.mu.Unlock()
	vr.sapi5TTS.enabled = false
}

// scanLocalSAPI5Voices queries Windows for installed SAPI5 voices
func (vr *VoiceRegistry) scanLocalSAPI5Voices() {
	var profiles []VoiceProfile

	psCmd := "Add-Type -AssemblyName System.Speech; $synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; $synth.GetInstalledVoices() | Where-Object { $_.VoiceInfo.Culture.TwoLetterISOLanguageName -eq 'de' } | ForEach-Object { $_.VoiceInfo.Name + ';' + $_.VoiceInfo.Gender }"
	cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psCmd)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Split(line, ";")
			name := parts[0]
			gender := "unbekannt"
			if len(parts) > 1 {
				g := strings.ToLower(parts[1])
				if strings.Contains(g, "female") {
					gender = "weiblich"
				} else if strings.Contains(g, "male") {
					gender = "männlich"
				}
			}
			displayName := fmt.Sprintf("%s (SAPI5 %s)", strings.TrimSuffix(name, " Desktop"), strings.ToUpper(gender[:1]))
			profiles = append(profiles, VoiceProfile{
				ID:        name,
				Name:      displayName,
				Type:      "local_sapi5",
				Gender:    gender,
				Available: true,
				Offline:   true,
				Language:  "de",
			})
		}
	}
	vr.sapi5Voices = profiles
}

// GetAvailableVoices returns all available voice profiles (Sherpa first, then SAPI5)
func (vr *VoiceRegistry) GetAvailableVoices() []VoiceProfile {
	vr.mu.RLock()
	defer vr.mu.RUnlock()

	var list []VoiceProfile

	// Sherpa-ONNX primary voices
	for _, v := range vr.sherpa.ListVoices() {
		list = append(list, VoiceProfile{
			ID:        v.ID,
			Name:      fmt.Sprintf("%s (Sherpa %s)", v.Name, strings.ToUpper(v.Language)),
			Type:      "sherpa",
			Gender:    v.Gender,
			Available: v.Configured,
			Offline:   true,
			Language:  v.Language,
		})
	}

	// SAPI5 fallback voices
	for _, v := range vr.sapi5Voices {
		list = append(list, v)
	}

	return list
}

// SynthesizeWithFallback synthesizes speech using Sherpa primary, SAPI5 fallback
func (vr *VoiceRegistry) SynthesizeWithFallback(text string, voice string) ([]byte, string, error) {
	// Try Sherpa first (always)
	vr.mu.RLock()
	sherpaHealthy := vr.sherpa != nil
	vr.mu.RUnlock()

	if sherpaHealthy {
		wavBytes, err := vr.sherpa.Synthesize(text, voice)
		if err == nil {
			return wavBytes, "audio/wav", nil
		}
		fmt.Printf("[VOICE] Sherpa synthesis failed (%v), versuche SAPI5...\n", err)
	}

	// Fallback: SAPI5 (if enabled)
	if vr.sapi5TTS.HealthCheck() {
		data, mime, err := vr.sapi5TTS.Synthesize(text, voice)
		if err == nil {
			return data, mime, nil
		}
		fmt.Printf("[VOICE] SAPI5 synthesis failed: %v\n", err)
	}

	return nil, "", fmt.Errorf("all TTS backends failed (Sherpa primary, SAPI5 fallback)")
}

// Transcribe audio via Groq Whisper
func (vr *VoiceRegistry) Transcribe(audioBytes []byte, filename string) (string, error) {
	if !vr.groqSTT.HealthCheck() {
		return "", fmt.Errorf("transcription provider unavailable (missing Groq credentials)")
	}
	return vr.groqSTT.Transcribe(audioBytes, filename)
}

// GetHealthStatus returns comprehensive health status of all voice subsystems
func (vr *VoiceRegistry) GetHealthStatus() map[string]interface{} {
	vr.mu.RLock()
	defer vr.mu.RUnlock()

	sherpaHealth := vr.sherpa.Health()
	sapi5Health := vr.sapi5TTS.HealthCheck()
	groqHealth := vr.groqSTT.HealthCheck()

	status := "ONLINE"
	if !sherpaHealth["configured"].(bool) && !sapi5Health {
		status = "DEGRADED (Keine lokale TTS-Stimme konfiguriert)"
	}

	result := map[string]interface{}{
		"status":                  status,
		"sherpa_local":            sherpaHealth,
		"local_sapi5_available":   sapi5Health,
		"groq_whisper_available":  groqHealth,
	}

	return result
}
