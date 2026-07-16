package handlers

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"go-aethel/provider"
	"go-aethel/security"
)

const maxSettingsStateBytes = 2 << 20

var (
	settingsStateMu sync.Mutex
	personaID       = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

type customPersona struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SystemPrompt string `json:"system_prompt"`
}

type apiCostLedger struct {
	Days   map[string]float64 `json:"days"`
	Months map[string]float64 `json:"months"`
}

func settingsStatePath(name string) string {
	return filepath.Join(security.WorkspaceDir, name)
}

func readSettingsJSON(path string, target interface{}) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, maxSettingsStateBytes))
	return decoder.Decode(target)
}

func writeSettingsJSON(path string, value interface{}) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if len(encoded) > maxSettingsStateBytes {
		return errors.New("settings state exceeds size boundary")
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".aethel-settings-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0600); err != nil {
		return err
	}
	if _, err := temporary.Write(encoded); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	_ = os.Remove(path)
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}

func handleBrowserScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := settingsStatePath("browser_screenshot.png")
	if _, err := os.Stat(path); err != nil {
		http.Error(w, "browser screenshot unavailable", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, path)
}

func LogAPICall(modelID string, inputTokens, outputTokens, cachedTokens int) {
	if inputTokens < 0 || outputTokens < 0 || cachedTokens < 0 {
		return
	}
	cost := provider.CalculateInferenceCost(modelID, inputTokens, outputTokens, cachedTokens)
	if cost <= 0 {
		return
	}
	settingsStateMu.Lock()
	defer settingsStateMu.Unlock()
	ledger := apiCostLedger{Days: map[string]float64{}, Months: map[string]float64{}}
	_ = readSettingsJSON(settingsStatePath("api_costs.json"), &ledger)
	if ledger.Days == nil {
		ledger.Days = map[string]float64{}
	}
	if ledger.Months == nil {
		ledger.Months = map[string]float64{}
	}
	now := time.Now()
	ledger.Days[now.Format("2006-01-02")] += cost
	ledger.Months[now.Format("2006-01")] += cost
	_ = writeSettingsJSON(settingsStatePath("api_costs.json"), ledger)
}

func handleCosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	settingsStateMu.Lock()
	defer settingsStateMu.Unlock()
	ledger := apiCostLedger{Days: map[string]float64{}, Months: map[string]float64{}}
	_ = readSettingsJSON(settingsStatePath("api_costs.json"), &ledger)
	now := time.Now()
	_ = json.NewEncoder(w).Encode(map[string]float64{
		"today": ledger.Days[now.Format("2006-01-02")],
		"month": ledger.Months[now.Format("2006-01")],
	})
}

func handlePersonas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	settingsStateMu.Lock()
	defer settingsStateMu.Unlock()
	path := settingsStatePath("custom_personas.json")
	personas := make([]customPersona, 0)
	if err := readSettingsJSON(path, &personas); err != nil {
		http.Error(w, "persona registry unavailable", http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		_ = json.NewEncoder(w).Encode(personas)
	case http.MethodPost:
		var incoming customPersona
		decoder := json.NewDecoder(io.LimitReader(r.Body, 16<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&incoming); err != nil || strings.TrimSpace(incoming.Name) == "" || strings.TrimSpace(incoming.SystemPrompt) == "" || utf8.RuneCountInString(incoming.Name) > 120 || utf8.RuneCountInString(incoming.SystemPrompt) > 12000 {
			http.Error(w, "invalid persona payload", http.StatusBadRequest)
			return
		}
		incoming.Name = strings.TrimSpace(incoming.Name)
		incoming.SystemPrompt = strings.TrimSpace(incoming.SystemPrompt)
		if incoming.ID == "" {
			identifier := make([]byte, 16)
			if _, err := rand.Read(identifier); err != nil {
				http.Error(w, "persona identifier unavailable", http.StatusInternalServerError)
				return
			}
			incoming.ID = hex.EncodeToString(identifier)
		} else if !personaID.MatchString(incoming.ID) {
			http.Error(w, "invalid persona identifier", http.StatusBadRequest)
			return
		}
		updated := false
		for index := range personas {
			if personas[index].ID == incoming.ID {
				personas[index] = incoming
				updated = true
				break
			}
		}
		if !updated {
			if len(personas) >= 128 {
				http.Error(w, "persona registry capacity reached", http.StatusConflict)
				return
			}
			personas = append(personas, incoming)
		}
		if err := writeSettingsJSON(path, personas); err != nil {
			http.Error(w, "persona registry write failed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "persona": incoming})
	case http.MethodDelete:
		id := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("id")))
		if !personaID.MatchString(id) {
			http.Error(w, "invalid persona identifier", http.StatusBadRequest)
			return
		}
		filtered := personas[:0]
		found := false
		for _, persona := range personas {
			if persona.ID == id {
				found = true
				continue
			}
			filtered = append(filtered, persona)
		}
		if !found {
			http.Error(w, "persona not found", http.StatusNotFound)
			return
		}
		if err := writeSettingsJSON(path, filtered); err != nil {
			http.Error(w, "persona registry write failed", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleProviderHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"providers": []provider.ProviderHealth{}})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"providers": provider.ProbeProviderHealth(ctx, state)})
}
