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
	"time"
)

// VoiceProfile represents a configured speech engine voice option
type VoiceProfile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "premium" | "local" | "browser"
	Gender    string `json:"gender"`
	Available bool   `json:"available"`
}

// VoiceOutputProvider translates text to audio stream
type VoiceOutputProvider interface {
	Synthesize(text string, voice string) ([]byte, string, error) // Returns (audioBytes, contentType, error)
	HealthCheck() bool
}

// TranscriptionProvider transcribes audio binary
type TranscriptionProvider interface {
	Transcribe(audioBytes []byte, filename string) (string, error)
	HealthCheck() bool
}

// OpenAI TTS Provider
type OpenAiTTSProvider struct {
	mu sync.Mutex
}

func (o *OpenAiTTSProvider) Synthesize(text string, voice string) ([]byte, string, error) {
	key := state.getOpenAIKey()
	if key == "" {
		return nil, "", fmt.Errorf("OpenAI key not configured")
	}

	payload := map[string]interface{}{
		"model":           "tts-1",
		"input":           text,
		"voice":           voice,
		"response_format": "mp3",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/audio/speech", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("OpenAI speech synthesis status %d: %s", resp.StatusCode, string(body))
	}

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return audioBytes, "audio/mpeg", nil
}

func (o *OpenAiTTSProvider) HealthCheck() bool {
	return state.getOpenAIKey() != ""
}

// Local Windows SAPI5 Speech Synthesizer Provider
type Sapi5TTSProvider struct {
	mu sync.Mutex
}

func (s *Sapi5TTSProvider) Synthesize(text string, voice string) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tempWav := filepath.Join(os.TempDir(), "aethel_speech.wav")
	_ = os.Remove(tempWav)

	pitch := "default"
	rate := "default"

	switch voice {
	case "onyx":
		pitch = "-15%"
		rate = "+10%"
	case "nova":
		pitch = "+15%"
		rate = "+20%"
	case "alloy":
		pitch = "default"
		rate = "+15%"
	case "echo":
		pitch = "-25%"
		rate = "-10%"
	case "fable":
		pitch = "+5%"
		rate = "+25%"
	case "shimmer":
		pitch = "+25%"
		rate = "+5%"
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
			$voiceName = '%s';
			
			$targetGender = $null;
			if ($voiceName -eq 'onyx' -or $voiceName -eq 'echo') {
				$targetGender = 'Male';
			} elseif ($voiceName -eq 'nova' -or $voiceName -eq 'shimmer') {
				$targetGender = 'Female';
			}
			
			$selected = $null;
			if ($voiceName) {
				$selected = $synth.GetInstalledVoices() | Where-Object { $_.VoiceInfo.Name -eq $voiceName } | Select-Object -First 1;
			}
			if (-not $selected -and $targetGender) {
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
			
			$synth.SetOutputToWaveFile('%s');
			$synth.SpeakSsml('%s');
		} catch {
			Write-Error $_.Exception.Message;
			exit 1;
		} finally {
			if ($synth) { $synth.Dispose() }
		}
	`, strings.ReplaceAll(voice, "'", "''"), tempWav, strings.ReplaceAll(ssml, "'", "''"))

	cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psScript)
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
	// Verify if powershell is available and has at least one German voice
	psCmd := "Add-Type -AssemblyName System.Speech; $synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; $synth.GetInstalledVoices() | Where-Object { $_.VoiceInfo.Culture.TwoLetterISOLanguageName -eq 'de' } | Measure-Object | Select-Object -ExpandProperty Count"
	cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psCmd)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	countStr := strings.TrimSpace(string(output))
	return countStr != "" && countStr != "0"
}

// Groq Whisper Speech-To-Text Provider
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
type VoiceRegistry struct {
	mu           sync.RWMutex
	openaiTTS    *OpenAiTTSProvider
	sapi5TTS     *Sapi5TTSProvider
	groqSTT      *GroqWhisperProvider
	localVoices  []VoiceProfile
}

func NewVoiceRegistry() *VoiceRegistry {
	return &VoiceRegistry{
		openaiTTS:   &OpenAiTTSProvider{},
		sapi5TTS:    &Sapi5TTSProvider{},
		groqSTT:     &GroqWhisperProvider{},
		localVoices: make([]VoiceProfile, 0),
	}
}

func (vr *VoiceRegistry) LoadLocalVoices() {
	vr.mu.Lock()
	defer vr.mu.Unlock()

	var profiles []VoiceProfile
	
	// Query local SAPI5 voices
	psCmd := "Add-Type -AssemblyName System.Speech; $synth = New-Object System.Speech.Synthesis.SpeechSynthesizer; $synth.GetInstalledVoices() | Where-Object { $_.VoiceInfo.Culture.TwoLetterISOLanguageName -eq 'de' } | ForEach-Object { $_.VoiceInfo.Name + ';' + $_.VoiceInfo.Gender }"
	cmd := exec.Command(getPowerShellPath(), "-NoProfile", "-NonInteractive", "-Command", psCmd)
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
			displayName := fmt.Sprintf("%s (Lokal %s)", strings.TrimSuffix(name, " Desktop"), strings.ToUpper(gender[:1]))
			profiles = append(profiles, VoiceProfile{
				ID:        name,
				Name:      displayName,
				Type:      "local",
				Gender:    gender,
				Available: true,
			})
		}
	}
	vr.localVoices = profiles
}

func (vr *VoiceRegistry) GetAvailableVoices() []VoiceProfile {
	vr.mu.RLock()
	defer vr.mu.RUnlock()

	var list []VoiceProfile
	
	// Premium voices
	list = append(list, VoiceProfile{ID: "onyx", Name: "Aethel Onyx (Premium M)", Type: "premium", Gender: "männlich", Available: vr.openaiTTS.HealthCheck()})
	list = append(list, VoiceProfile{ID: "nova", Name: "Aethel Nova (Premium W)", Type: "premium", Gender: "weiblich", Available: vr.openaiTTS.HealthCheck()})
	list = append(list, VoiceProfile{ID: "alloy", Name: "Aethel Alloy (Premium N)", Type: "premium", Gender: "neutral", Available: vr.openaiTTS.HealthCheck()})
	list = append(list, VoiceProfile{ID: "echo", Name: "Aethel Echo (Premium M)", Type: "premium", Gender: "männlich", Available: vr.openaiTTS.HealthCheck()})
	list = append(list, VoiceProfile{ID: "fable", Name: "Aethel Fable (Premium N)", Type: "premium", Gender: "neutral", Available: vr.openaiTTS.HealthCheck()})
	list = append(list, VoiceProfile{ID: "shimmer", Name: "Aethel Shimmer (Premium W)", Type: "premium", Gender: "weiblich", Available: vr.openaiTTS.HealthCheck()})

	// Add scanned local SAPI5 voices
	list = append(list, vr.localVoices...)
	return list
}

func (vr *VoiceRegistry) SynthesizeWithFallback(text string, voice string) ([]byte, string, error) {
	// 1. Identify voice class
	isPremiumVoice := false
	for _, v := range []string{"onyx", "nova", "alloy", "echo", "fable", "shimmer"} {
		if voice == v {
			isPremiumVoice = true
			break
		}
	}

	// 2. OpenAI Key Check for Premium Synthesis
	if isPremiumVoice && vr.openaiTTS.HealthCheck() {
		data, mime, err := vr.openaiTTS.Synthesize(text, voice)
		if err == nil {
			return data, mime, nil
		}
		fmt.Printf("[VOICE REGISTRY ALERT]: OpenAI TTS failed (%v). Falling back to SAPI5.\n", err)
	}

	// 3. Fallback to Local SAPI5
	data, mime, err := vr.sapi5TTS.Synthesize(text, voice)
	if err == nil {
		return data, mime, nil
	}

	return nil, "", fmt.Errorf("all backend TTS synthesizers failed: %v", err)
}

func (vr *VoiceRegistry) Transcribe(audioBytes []byte, filename string) (string, error) {
	if !vr.groqSTT.HealthCheck() {
		return "", fmt.Errorf("transcription provider unavailable (missing Groq credentials)")
	}
	return vr.groqSTT.Transcribe(audioBytes, filename)
}

func (vr *VoiceRegistry) GetHealthStatus() map[string]interface{} {
	openaiHealth := vr.openaiTTS.HealthCheck()
	sapi5Health := vr.sapi5TTS.HealthCheck()
	groqHealth := vr.groqSTT.HealthCheck()

	status := "ONLINE"
	if !openaiHealth && !sapi5Health {
		status = "DEGRADED (Backend Offline, Client synthesis fallback active)"
	}

	return map[string]interface{}{
		"status":                 status,
		"openai_tts_available":  openaiHealth,
		"local_sapi5_available": sapi5Health,
		"groq_whisper_available": groqHealth,
		"local_voices_count":    len(vr.localVoices),
	}
}
