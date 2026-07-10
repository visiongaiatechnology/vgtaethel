//go:build !cgo

// Sherpa-ONNX stub for non-CGO builds.
// Provides empty/noop implementations so the project compiles without GCC.
package main

import (
	"fmt"
	"sync"
	"time"
)

// VoiceInfo represents a Sherpa-ONNX voice configuration (stub)
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
	// Concat message with specific file expectations
	MissingFiles []string `json:"-"`
}

// requiredVoiceFiles stub – always returns empty
func requiredVoiceFiles(voiceID string) (modelFile string, required []string, optional []string) {
	return "model.onnx", nil, nil
}

// SherpaVoiceEngine manages local offline TTS via Sherpa-ONNX (stub)
type SherpaVoiceEngine struct {
	mu sync.Mutex
}

// NewSherpaVoiceEngine creates a new Sherpa-ONNX voice engine (stub – returns nil)
func NewSherpaVoiceEngine(modelRoot, audioRoot string) *SherpaVoiceEngine {
	return &SherpaVoiceEngine{}
}

// Init stub – always succeeds with empty voice list
func (e *SherpaVoiceEngine) Init() error {
	return nil
}

// Synthesize stub – always returns error (Sherpa not available)
func (e *SherpaVoiceEngine) Synthesize(text string, voiceID string) ([]byte, error) {
	return nil, fmt.Errorf("SHERPA_UNAVAILABLE: Sherpa-ONNX wurde ohne CGO kompiliert. Bitte installiere GCC und setze CGO_ENABLED=1")
}

// ListVoices stub – empty list
func (e *SherpaVoiceEngine) ListVoices() []VoiceInfo {
	return []VoiceInfo{}
}

// Health stub – reports not configured
func (e *SherpaVoiceEngine) Health() map[string]interface{} {
	return map[string]interface{}{
		"provider":   "sherpa_local",
		"offline":    true,
		"configured": false,
		"voices":     0,
		"ready":      0,
		"warnings":   []string{"Sherpa-ONNX ist nicht verfügbar (Build ohne CGO)"},
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

// SynthesizeWithResponse stub – always returns error (Sherpa not available)
func (e *SherpaVoiceEngine) SynthesizeWithResponse(text string, voiceID string) (*SynthesizeResponse, error) {
	return nil, fmt.Errorf("SHERPA_UNAVAILABLE: Sherpa-ONNX wurde ohne CGO kompiliert. Bitte installiere GCC und setze CGO_ENABLED=1")
}

// cleanOldAudio stub – noop
func (e *SherpaVoiceEngine) cleanOldAudio() {}

// fileExists stub
func fileExists(path string) bool {
	return false
}

// dirExists stub
func dirExists(path string) bool {
	return false
}

// cleanPath stub
func cleanPath(base, sub string) (string, error) {
	return "", fmt.Errorf("not available without cgo")
}

// float32ToWav stub – always returns error
func float32ToWav(samples []float32, sampleRate int) ([]byte, error) {
	return nil, fmt.Errorf("not available without cgo")
}

// toBase64 stub – always returns empty
func toBase64(data []byte) string {
	return ""
}

// required for compile – ensure time is used
var _ = time.Now
