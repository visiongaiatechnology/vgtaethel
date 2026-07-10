package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
)

func handleAudioSpeech(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var text string
	var selectedVoice string
	jsonFormat := r.URL.Query().Get("format") == "json"

	if r.Method == http.MethodGet {
		text = r.URL.Query().Get("text")
		selectedVoice = r.URL.Query().Get("voice")
	} else {
		var req struct {
			Text   string `json:"text"`
			Voice  string `json:"voice"`
			Format string `json:"format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			text = req.Text
			selectedVoice = req.Voice
			if req.Format == "json" {
				jsonFormat = true
			}
		}
	}

	if text == "" {
		http.Error(w, "Missing text parameter", http.StatusBadRequest)
		return
	}
	if len([]rune(text)) > 2000 {
		http.Error(w, "Text exceeds synthesis limit", http.StatusRequestEntityTooLarge)
		return
	}

	if jsonFormat {
		vr := state.voice
		vr.mu.RLock()
		sherpa := vr.sherpa
		vr.mu.RUnlock()

		if sherpa != nil {
			resp, err := sherpa.SynthesizeWithResponse(text, selectedVoice)
			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
				return
			}
		}
	}

	audioBytes, mime, err := state.voice.SynthesizeWithFallback(text, selectedVoice)
	if err != nil {
		http.Error(w, fmt.Sprintf("Speech synthesis failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mime)
	w.WriteHeader(http.StatusOK)
	w.Write(audioBytes)
}

func handleAudioTranscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	const maxAudioBytes int64 = 10 * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxAudioBytes)
	err := r.ParseMultipartForm(10 * 1024 * 1024)
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "Missing audio file in form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, io.LimitReader(file, maxAudioBytes+1))
	if err != nil {
		http.Error(w, "Failed to read audio data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if int64(buf.Len()) > maxAudioBytes {
		http.Error(w, "Audio upload exceeds size limit", http.StatusRequestEntityTooLarge)
		return
	}

	transcript, err := state.voice.Transcribe(buf.Bytes(), filepath.Base(header.Filename))
	if err != nil {
		http.Error(w, fmt.Sprintf("Transcription failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"text": transcript})
}

func handleAudioVoices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	voices := state.voice.GetAvailableVoices()
	json.NewEncoder(w).Encode(voices)
}

func handleAudioHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := state.voice.GetHealthStatus()
	json.NewEncoder(w).Encode(health)
}

func handleAudioTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Voice string `json:"voice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	testText := "Aethel Audio-Verbindungstest erfolgreich."
	audioBytes, mime, err := state.voice.SynthesizeWithFallback(testText, req.Voice)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"message":   "Voice synthesis worked.",
		"mime_type": mime,
		"size":      len(audioBytes),
	})
}
