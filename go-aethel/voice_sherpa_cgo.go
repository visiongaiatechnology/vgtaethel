//go:build cgo

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// VoiceInfo represents a Sherpa-ONNX voice configuration
type VoiceInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Language   string `json:"language"`
	Gender     string `json:"gender,omitempty"`
	Provider   string `json:"provider"`
	Offline    bool   `json:"offline"`
	Configured bool   `json:"configured"`
	ModelPath  string `json:"-"`
	TokensPath string `json:"-"`
	DataDir    string `json:"-"`
	SpeakerID  int    `json:"speaker_id"`
	SampleRate int    `json:"sample_rate"`
	IsPiper    bool   `json:"-"`
	ModelFile  string `json:"-"`
	// MissingFiles lists what's needed to make the voice usable
	MissingFiles []string `json:"-"`
}

// requiredVoiceFiles scans a voice data directory and returns:
//   - modelFile: the first .onnx file found (or "" if none)
//   - isPiper: true if it's a Piper model (requires voices.bin & espeak-ng-data)
//   - required: list of files/dirs that MUST exist for the voice to be usable
//   - optional: list of files that are nice-to-have
func requiredVoiceFiles(dataDir string) (modelFile string, isPiper bool, required []string, optional []string) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return "", false, nil, nil
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".onnx") {
			modelFile = e.Name()
			break
		}
	}
	if modelFile == "" {
		return "", false, nil, nil
	}

	baseName := strings.ToLower(filepath.Base(dataDir))
	hasVoices := fileExists(filepath.Join(dataDir, "voices.bin"))
	hasEspeak := dirExists(filepath.Join(dataDir, "espeak-ng-data"))

	isPiper = false

	// Strongest signal: voices.bin or espeak-ng-data exist -> definitely Piper
	if hasVoices || hasEspeak {
		isPiper = true
	}

	// Heuristic: folder name contains "piper" but NOT "vits"
	if strings.Contains(baseName, "piper") && !strings.Contains(baseName, "vits") {
		isPiper = true
	}

	// If folder name contains "vits", treat as VITS regardless of name heuristics
	if strings.Contains(baseName, "vits-piper") || strings.Contains(baseName, "thorsten") {
		return modelFile, true, []string{modelFile, "tokens.txt", "espeak-ng-data"}, nil
	}
	if isPiper {
		required = []string{modelFile, "tokens.txt", "voices.bin", "espeak-ng-data"}
		optional = nil
	} else {
		// VITS / Piper-VITS braucht tokens.txt für Sherpa zuverlässig
		required = []string{modelFile, "tokens.txt"}
		optional = nil
	}

	return
}

// SherpaVoiceEngine manages local offline TTS via Sherpa-ONNX
type SherpaVoiceEngine struct {
	modelRoot  string
	audioRoot  string
	voices     []VoiceInfo
	mu         sync.Mutex
	modelCache map[string]*sherpa.OfflineTts
}

// NewSherpaVoiceEngine creates a new Sherpa-ONNX voice engine
func NewSherpaVoiceEngine(modelRoot, audioRoot string) *SherpaVoiceEngine {
	if modelRoot == "" {
		modelRoot = "./vgt_workspace/models/sherpa"
	}
	if audioRoot == "" {
		audioRoot = "./vgt_workspace/audio"
	}

	return &SherpaVoiceEngine{
		modelRoot:  modelRoot,
		audioRoot:  audioRoot,
		voices:     make([]VoiceInfo, 0),
		modelCache: make(map[string]*sherpa.OfflineTts),
	}
}

// fileExists checks if a file exists and is readable
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// cleanPath prevents path traversal attacks
func cleanPath(base, sub string) (string, error) {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("invalid base path: %w", err)
	}
	joined := filepath.Join(absBase, sub)
	canon := filepath.Clean(joined)
	rel, err := filepath.Rel(absBase, canon)
	if err != nil || strings.HasPrefix(rel, "..") || strings.HasPrefix(rel, "\\..") {
		return "", fmt.Errorf("path traversal detected: %s", sub)
	}
	return canon, nil
}

// Init scans the model directory and discovers available voices
func (e *SherpaVoiceEngine) Init() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.cleanOldAudio()

	entries, err := os.ReadDir(e.modelRoot)
	if err != nil {
		_ = os.MkdirAll(e.audioRoot, 0755)
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		voiceID := entry.Name()
		dataDir := filepath.Join(e.modelRoot, voiceID)

		modelFile, isPiper, required, optional := requiredVoiceFiles(dataDir)
		if modelFile == "" {
			// Keine .onnx Datei – kein gültiges Modellverzeichnis
			continue
		}

		modelPath := filepath.Join(dataDir, modelFile)
		tokensPath := ""

		// Prüfe required files
		missing := []string{}
		for _, fn := range required {
			fpath := filepath.Join(dataDir, fn)
			if !fileExists(fpath) && !dirExists(fpath) {
				missing = append(missing, fn)
			}
		}

		// Prüfe optional files – tokens.txt ist für VITS optional
		optionalMissing := []string{}
		for _, fn := range optional {
			fpath := filepath.Join(dataDir, fn)
			if !fileExists(fpath) && !dirExists(fpath) {
				optionalMissing = append(optionalMissing, fn)
			}
		}

		configured := len(missing) == 0

		// Setze tokensPath nur, wenn tokens.txt existiert
		if fileExists(filepath.Join(dataDir, "tokens.txt")) {
			tokensPath = filepath.Join(dataDir, "tokens.txt")
		}

		vi := VoiceInfo{
			ID:         voiceID,
			Name:       voiceID,
			Language:   e.detectLanguage(voiceID),
			Provider:   "sherpa_local",
			Offline:    true,
			Configured: configured,
			ModelPath:  modelPath,
			TokensPath: tokensPath,
			DataDir:    dataDir,
			SpeakerID:  0,
			SampleRate: 22050,
			IsPiper:    isPiper,
			ModelFile:  modelFile,
		}

		if !configured {
			vi.MissingFiles = missing
			vi.Name = fmt.Sprintf("%s (fehlend: %s)", voiceID, strings.Join(missing, ", "))
		} else if len(optionalMissing) > 0 {
			vi.Name = fmt.Sprintf("%s (optional fehlend: %s)", voiceID, strings.Join(optionalMissing, ", "))
		}

		e.voices = append(e.voices, vi)
	}

	return nil
}

// detectLanguage tries to detect language from voice ID
func (e *SherpaVoiceEngine) detectLanguage(voiceID string) string {
	lower := strings.ToLower(voiceID)
	switch {
	case strings.Contains(lower, "de_") || strings.Contains(lower, "de-") || strings.HasPrefix(lower, "de"):
		return "de"
	case strings.Contains(lower, "fr_") || strings.Contains(lower, "fr-"):
		return "fr"
	case strings.Contains(lower, "es_") || strings.Contains(lower, "es-"):
		return "es"
	case strings.Contains(lower, "zh_") || strings.Contains(lower, "zh-") || strings.Contains(lower, "multi-lang"):
		return "multi"
	case strings.Contains(lower, "ja_") || strings.Contains(lower, "ja-"):
		return "ja"
	case strings.Contains(lower, "ko_") || strings.Contains(lower, "ko-"):
		return "ko"
	default:
		return "en"
	}
}

// loadModelUnsafe loads a Sherpa-ONNX model (caller must hold lock)
func (e *SherpaVoiceEngine) loadModelUnsafe(voiceID, modelPath, tokensPath, dataDir string, isPiper bool) (*sherpa.OfflineTts, error) {
	// Sherpa-ONNX erwartet den VOLLEN Pfad zur espeak-ng-data Directory,
	// nicht den übergeordneten Model-Ordner. Piper-Stimmen (z.B. Thorsten)
	// schlagen fehl, wenn DataDir auf den falschen Ordner zeigt.
	espeakDir := filepath.Join(dataDir, "espeak-ng-data")
	effectiveDataDir := ""
	if dirExists(espeakDir) {
		effectiveDataDir = espeakDir
	}

	config := sherpa.OfflineTtsConfig{
		Model: sherpa.OfflineTtsModelConfig{
			Vits: sherpa.OfflineTtsVitsModelConfig{
				Model:   modelPath,
				Tokens:  tokensPath,
				DataDir: effectiveDataDir,
			},
			NumThreads: 2,
			Debug:      0,
			Provider:   "cpu",
		},
		MaxNumSentences: 2,
	}

	tts := sherpa.NewOfflineTts(&config)
	if tts == nil {
		return nil, fmt.Errorf("sherpa.NewOfflineTts returned nil (DLLs/Missing?)")
	}

	return tts, nil
}

// getModel returns a cached TTS model or loads it
func (e *SherpaVoiceEngine) getModel(voiceID string) (*sherpa.OfflineTts, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if tts, ok := e.modelCache[voiceID]; ok {
		return tts, nil
	}

	var voice *VoiceInfo
	for i := range e.voices {
		if e.voices[i].ID == voiceID {
			voice = &e.voices[i]
			break
		}
	}
	if voice == nil {
		return nil, fmt.Errorf("SHERPA_VOICE_UNKNOWN: Stimme '%s' nicht gefunden", voiceID)
	}
	if !voice.Configured {
		msg := fmt.Sprintf("SHERPA_MODEL_MISSING: Fehlende Dateien: %s in %s/",
			strings.Join(voice.MissingFiles, ", "), voice.DataDir)
		return nil, fmt.Errorf(msg)
	}

	tts, err := e.loadModelUnsafe(voiceID, voice.ModelPath, voice.TokensPath, voice.DataDir, voice.IsPiper)
	if err != nil {
		return nil, fmt.Errorf("SHERPA_MODEL_LOAD_FAILED: %w", err)
	}

	e.modelCache[voiceID] = tts
	return tts, nil
}

// float32ToWav converts float32 audio samples to WAV byte buffer (16-bit PCM)
func float32ToWav(samples []float32, sampleRate int) ([]byte, error) {
	if len(samples) == 0 {
		return nil, fmt.Errorf("no audio samples to convert")
	}

	numSamples := len(samples)
	numChannels := 1
	bitsPerSample := 16
	bytesPerSample := bitsPerSample / 8
	byteRate := sampleRate * numChannels * bytesPerSample
	blockAlign := numChannels * bytesPerSample
	dataSize := numSamples * bytesPerSample

	var buf bytes.Buffer

	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")

	// fmt subchunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(numChannels))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&buf, binary.LittleEndian, uint32(byteRate))
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))
	binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample))

	// data subchunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))

	// Convert float32 -> int16 PCM with clamping
	for _, s := range samples {
		if s > 1.0 {
			s = 1.0
		} else if s < -1.0 {
			s = -1.0
		}
		pcm := int16(s * 32767)
		binary.Write(&buf, binary.LittleEndian, pcm)
	}

	return buf.Bytes(), nil
}

// cleanOldAudio removes old files, keeping max 50
func (e *SherpaVoiceEngine) cleanOldAudio() {
	files, err := os.ReadDir(e.audioRoot)
	if err != nil {
		return
	}
	now := time.Now()
	maxAge := 10 * time.Minute
	maxFiles := 50

	type fe struct {
		name    string
		modTime time.Time
	}
	var entries []fe
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		entries = append(entries, fe{name: f.Name(), modTime: info.ModTime()})
	}
	// bubble sort oldest first
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].modTime.After(entries[j].modTime) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	removed := 0
	for i, entry := range entries {
		shouldRemove := false
		if now.Sub(entry.modTime) > maxAge {
			shouldRemove = true
		}
		if len(entries)-removed > maxFiles && i < len(entries)-maxFiles {
			shouldRemove = true
		}
		if shouldRemove {
			os.Remove(filepath.Join(e.audioRoot, entry.name))
			removed++
		}
	}
}

// Synthesize runs TTS and returns WAV bytes
func (e *SherpaVoiceEngine) Synthesize(text string, voiceID string) ([]byte, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("text is empty")
	}
	if chars := []rune(text); len(chars) > 4000 {
		text = string(chars[:4000])
	}

	tts, err := e.getModel(voiceID)
	if err != nil {
		return nil, err
	}

	generated := tts.Generate(text, 0, 1.0)
	if generated == nil {
		return nil, fmt.Errorf("tts.Generate returned nil")
	}
	if len(generated.Samples) == 0 {
		return nil, fmt.Errorf("generated zero audio samples")
	}

	wavBytes, err := float32ToWav(generated.Samples, generated.SampleRate)
	if err != nil {
		return nil, fmt.Errorf("WAV conversion failed: %w", err)
	}

	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("aethel_tts_%d.wav", timestamp)
	if safePath, err := cleanPath(e.audioRoot, filename); err == nil {
		_ = os.WriteFile(safePath, wavBytes, 0600)
	}

	return wavBytes, nil
}

// ListVoices returns all discovered voice profiles
func (e *SherpaVoiceEngine) ListVoices() []VoiceInfo {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]VoiceInfo, len(e.voices))
	copy(result, e.voices)
	return result
}

// Health returns the health status of the Sherpa engine
func (e *SherpaVoiceEngine) Health() map[string]interface{} {
	e.mu.Lock()
	defer e.mu.Unlock()

	warnings := []string{}
	configuredCount := 0
	totalCount := len(e.voices)

	for _, v := range e.voices {
		if v.Configured {
			configuredCount++
		} else {
			if len(v.MissingFiles) > 0 {
				warnings = append(warnings, fmt.Sprintf(
					"Voice '%s': fehlend %s (in %s/)",
					v.ID, strings.Join(v.MissingFiles, ", "), v.DataDir))
			} else {
				warnings = append(warnings, fmt.Sprintf(
					"Voice '%s': unvollständig (keine .onnx Datei in %s)",
					v.ID, v.DataDir))
			}
		}
	}

	if _, err := os.Stat(e.modelRoot); os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("Modellverzeichnis fehlt: %s", e.modelRoot))
	}
	if _, err := os.Stat(e.audioRoot); os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("Audioverzeichnis fehlt: %s", e.audioRoot))
	}

	return map[string]interface{}{
		"provider":   "sherpa_local",
		"offline":    true,
		"configured": configuredCount > 0,
		"voices":     totalCount,
		"ready":      configuredCount,
		"model_root": e.modelRoot,
		"audio_root": e.audioRoot,
		"warnings":   warnings,
	}
}

// SynthesizeResponse wraps synthesis result for API JSON
type SynthesizeResponse struct {
	Provider    string `json:"provider"`
	Offline     bool   `json:"offline"`
	Format      string `json:"format"`
	AudioBase64 string `json:"audio_base64"`
	Voice       string `json:"voice"`
	Size        int    `json:"size"`
	Duration    string `json:"duration,omitempty"`
}

// SynthesizeWithResponse runs TTS and returns a JSON-ready response
func (e *SherpaVoiceEngine) SynthesizeWithResponse(text string, voiceID string) (*SynthesizeResponse, error) {
	wavBytes, err := e.Synthesize(text, voiceID)
	if err != nil {
		return nil, err
	}

	sampleRate := 22050
	for _, v := range e.ListVoices() {
		if v.ID == voiceID {
			sampleRate = v.SampleRate
			break
		}
	}
	numSamples := (len(wavBytes) - 44) / 2
	durationSec := float64(numSamples) / float64(sampleRate)

	return &SynthesizeResponse{
		Provider:    "sherpa_local",
		Offline:     true,
		Format:      "wav",
		AudioBase64: toBase64(wavBytes),
		Voice:       voiceID,
		Size:        len(wavBytes),
		Duration:    fmt.Sprintf("%.1fs", durationSec),
	}, nil
}

// toBase64 encodes bytes as standard base64 string
func toBase64(data []byte) string {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write(data)
	encoder.Close()
	return buf.String()
}
