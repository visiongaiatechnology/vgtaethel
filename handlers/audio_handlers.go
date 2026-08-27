package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func handleAudioSpeech(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text   string `json:"text"`
		Voice  string `json:"voice"`
		Format string `json:"format"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid speech request", http.StatusBadRequest)
		return
	}
	if req.Text == "" {
		http.Error(w, "Missing text parameter", http.StatusBadRequest)
		return
	}
	if len([]rune(req.Text)) > 2000 || len([]rune(req.Voice)) > 160 {
		http.Error(w, "Text exceeds synthesis limit", http.StatusRequestEntityTooLarge)
		return
	}

	if req.Format == "json" {
		if response, err := state.voice.SynthesizeJSON(req.Text, req.Voice); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			_ = json.NewEncoder(w).Encode(response)
			return
		}
	} else if req.Format != "" && req.Format != "wav" {
		http.Error(w, "Unsupported speech response format", http.StatusBadRequest)
		return
	}

	audioBytes, mime, err := state.voice.SynthesizeWithFallback(req.Text, req.Voice)
	if err != nil {
		log.Printf("[AUDIO] speech synthesis failed: %v", err)
		http.Error(w, "Speech synthesis failed", http.StatusInternalServerError)
		return
	}
	if mime != "audio/wav" || len(audioBytes) == 0 || len(audioBytes) > 64*1024*1024 {
		http.Error(w, "Speech synthesis returned an unsupported media type", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(audioBytes) // #nosec G705 -- fixed audio/wav content type, nosniff, and bounded binary output.
}

func handleAudioTranscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	const maxAudioBytes int64 = 10 * 1024 * 1024
	const maxRequestBytes int64 = maxAudioBytes + 1024*1024
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	err := r.ParseMultipartForm(maxAudioBytes) // #nosec G120 -- the entire request body is bounded by MaxBytesReader above.
	if err != nil {
		http.Error(w, "Invalid or oversized audio form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "Missing audio file in form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, io.LimitReader(file, maxAudioBytes+1))
	if err != nil {
		http.Error(w, "Failed to read audio data", http.StatusInternalServerError)
		return
	}
	if int64(buf.Len()) > maxAudioBytes {
		http.Error(w, "Audio upload exceeds size limit", http.StatusRequestEntityTooLarge)
		return
	}

	detected := http.DetectContentType(buf.Bytes()[:min(buf.Len(), 512)])
	filename := "speech.webm"
	switch detected {
	case "video/webm", "audio/webm":
	case "audio/wave", "audio/wav", "audio/x-wav":
		filename = "speech.wav"
	case "audio/ogg", "application/ogg":
		filename = "speech.ogg"
	default:
		http.Error(w, "Unsupported audio media type", http.StatusUnsupportedMediaType)
		return
	}
	transcript, err := state.voice.Transcribe(buf.Bytes(), filename)
	if err != nil {
		log.Printf("[AUDIO] transcription failed: %v", err)
		http.Error(w, "Transcription failed", http.StatusInternalServerError)
		return
	}
	if len([]rune(transcript)) > 16*1024 {
		http.Error(w, "Transcription response exceeded limit", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_ = json.NewEncoder(w).Encode(map[string]string{"text": transcript})
}

func handleAudioVoices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	voices := state.voice.AvailableVoicesPayload()
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
