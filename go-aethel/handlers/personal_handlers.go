package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"go-aethel/personal"
)

func HandlePersonalStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, _ := state.personal.LoadConfig()
	profile, _ := state.personal.LoadProfile()
	memories, _ := state.personal.ListMemories()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"config":        cfg,
		"profile":       profile,
		"memory_count":  len(memories),
		"setup_needed":  profile.DisplayName == "",
		"storage_scope": "personal",
	})
}

func HandlePersonalConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		cfg, err := state.personal.LoadConfig()
		if err != nil {
			http.Error(w, "Config unavailable", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(cfg)
	case http.MethodPost:
		var cfg personal.PersonalConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "Invalid config", http.StatusBadRequest)
			return
		}
		cfg.WakeWord = personal.ClampPersonalText(cfg.WakeWord, 40)
		cfg.PrimaryModel = personal.ClampPersonalText(cfg.PrimaryModel, 160)
		cfg.FallbackModel = personal.ClampPersonalText(cfg.FallbackModel, 160)
		cfg.HumorLevel = personal.ClampPersonalLevel(cfg.HumorLevel)
		cfg.HonestyLevel = personal.ClampPersonalLevel(cfg.HonestyLevel)
		cfg.InitiativeLevel = personal.ClampPersonalLevel(cfg.InitiativeLevel)
		if cfg.WakeWord == "" {
			cfg.WakeWord = "aethel"
		}
		if err := state.personal.SaveConfig(cfg); err != nil {
			http.Error(w, "Config save failed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandlePersonalProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		profile, err := state.personal.LoadProfile()
		if err != nil {
			http.Error(w, "Profile unavailable", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(profile)
	case http.MethodPost:
		var profile personal.PersonalProfile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "Invalid profile", http.StatusBadRequest)
			return
		}
		profile.DisplayName = personal.ClampPersonalText(profile.DisplayName, 80)
		profile.PreferredTone = personal.ClampPersonalText(profile.PreferredTone, 120)
		profile.AssistantStyle = personal.ClampPersonalText(profile.AssistantStyle, 300)
		profile.Notes = personal.ClampPersonalText(profile.Notes, 4000)
		profile.LocationCity = personal.ClampPersonalText(profile.LocationCity, 120)
		profile.LocationCountry = personal.ClampPersonalText(profile.LocationCountry, 120)
		if err := state.personal.SaveProfile(profile); err != nil {
			http.Error(w, "Profile save failed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandlePersonalMemories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		memories, err := state.personal.ListMemories()
		if err != nil {
			http.Error(w, "Memories unavailable", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(memories)
	case http.MethodPost:
		var req struct {
			personal.PersonalMemory
			OperatorApproved bool `json:"operator_approved"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid memory", http.StatusBadRequest)
			return
		}
		if !req.OperatorApproved {
			http.Error(w, "Explicit operator approval is required for personal memory", http.StatusForbidden)
			return
		}
		mem := req.PersonalMemory
		mem.Type = personal.ClampPersonalText(mem.Type, 60)
		mem.Content = personal.ClampPersonalText(mem.Content, 2000)
		mem.Source = personal.ClampPersonalText(mem.Source, 120)
		if mem.Type == "" || mem.Content == "" || personal.LooksSecretLike(mem.Content) {
			http.Error(w, "Memory requires type and content", http.StatusBadRequest)
			return
		}
		saved, err := state.personal.AppendMemory(mem)
		if err != nil {
			http.Error(w, "Memory save failed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(saved)
	case http.MethodPut:
		var mem personal.PersonalMemory
		if err := json.NewDecoder(r.Body).Decode(&mem); err != nil {
			http.Error(w, "Invalid memory", http.StatusBadRequest)
			return
		}
		mem.Type = personal.ClampPersonalText(mem.Type, 60)
		mem.Content = personal.ClampPersonalText(mem.Content, 2000)
		mem.Source = personal.ClampPersonalText(mem.Source, 120)
		if mem.Type == "" || mem.Content == "" {
			http.Error(w, "Memory requires type and content", http.StatusBadRequest)
			return
		}
		updated, err := state.personal.UpdateMemory(mem)
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "Memory not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Memory update failed", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(updated)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if err := state.personal.DeleteMemory(id); err != nil {
			http.Error(w, "Invalid memory id", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandlePersonalSetupQuestions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Config personal.PersonalConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid setup question payload", http.StatusBadRequest)
		return
	}
	questions, mode := personal.BuildPersonalSetupQuestions(req.Config)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"mode":      mode,
		"questions": questions,
	})
}

func HandlePersonalSetup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Answers map[string]string       `json:"answers"`
		Config  personal.PersonalConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid setup payload", http.StatusBadRequest)
		return
	}
	cfg := req.Config
	cfg.Enabled = true
	cfg.LearningEnabled = true
	if cfg.WakeWord == "" {
		cfg.WakeWord = "aethel"
	}
	profile, setupMode := personal.BuildPersonalProfileFromSetup(cfg, req.Answers)
	if err := state.personal.SaveProfile(profile); err != nil {
		http.Error(w, "Profile setup failed", http.StatusInternalServerError)
		return
	}
	if err := state.personal.SaveConfig(cfg); err != nil {
		http.Error(w, "Config setup failed", http.StatusInternalServerError)
		return
	}
	if profile.DisplayName != "" {
		_, _ = state.personal.AppendMemory(personal.PersonalMemory{
			Type:       "identity",
			Content:    "Der Nutzer möchte mit dem Namen " + profile.DisplayName + " angesprochen werden.",
			Confidence: 0.95,
			Source:     "setup",
		})
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "mode": setupMode, "profile": profile, "config": cfg})
}

func HandlePersonalLearn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := state.personal.LoadConfig()
	if err != nil || !cfg.Enabled || !cfg.LearningEnabled {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "skipped", "reason": "personal learning disabled"})
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid learn payload", http.StatusBadRequest)
		return
	}
	text := personal.ClampPersonalText(req.Text, 6000)
	lower := strings.ToLower(text)
	secretTerms := []string{"password", "passwort", "token", "api key", "secret", "private key"}
	for _, term := range secretTerms {
		if strings.Contains(lower, term) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "skipped", "reason": "secret-like content"})
			return
		}
	}
	candidates, extractErr := personal.ExtractPersonalLearningCandidatesWithModels(cfg, text)
	if extractErr != nil {
		candidates = personal.ExtractPersonalLearningCandidates(text)
	}
	mode := "model"
	if extractErr != nil {
		mode = "heuristic_fallback"
	}
	// Learning is a proposal, never an implicit write. The UI may present the
	// candidates and create only the memories explicitly approved by the user.
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "proposal", "mode": mode, "candidates": candidates})
}
