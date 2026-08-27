package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go-aethel/personal"
	"go-aethel/skills"
)

func HandlePersonalOperations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if operations == nil {
		http.Error(w, "Personal operations unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		items := operations.Pending(time.Now().UTC())
		if cfg, err := state.personal.LoadConfig(); err == nil && inQuietHours(time.Now(), cfg.QuietHoursStart, cfg.QuietHoursEnd) {
			filtered := make([]personal.OperationNotice, 0, len(items))
			for _, item := range items {
				if item.Priority == "critical" || item.Priority == "high" || item.RequireAck {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	case http.MethodPost:
		var req struct {
			Action  string `json:"action"`
			ID      string `json:"id"`
			Minutes int    `json:"minutes"`
			Kind    string `json:"kind"`
			City    string `json:"city"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid operations request", http.StatusBadRequest)
			return
		}
		switch strings.ToLower(req.Action) {
		case "acknowledge":
			if err := operations.Acknowledge(personal.ClampPersonalText(req.ID, 80)); err != nil {
				writeOperationError(w, err)
				return
			}
		case "snooze":
			if err := operations.Snooze(personal.ClampPersonalText(req.ID, 80), time.Duration(req.Minutes)*time.Minute); err != nil {
				writeOperationError(w, err)
				return
			}
		case "generate":
			if _, err := generatePersonalOperation(req.Kind, req.City); err != nil {
				http.Error(w, "Operation generation failed", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "Unsupported operations action", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func inQuietHours(now time.Time, start, end string) bool {
	startAt, startErr := time.Parse("15:04", start)
	endAt, endErr := time.Parse("15:04", end)
	if startErr != nil || endErr != nil || start == end {
		return false
	}
	minute := now.Hour()*60 + now.Minute()
	startMinute := startAt.Hour()*60 + startAt.Minute()
	endMinute := endAt.Hour()*60 + endAt.Minute()
	if startMinute < endMinute {
		return minute >= startMinute && minute < endMinute
	}
	return minute >= startMinute || minute < endMinute
}

func generatePersonalOperation(kind, city string) (personal.OperationNotice, error) {
	kind = strings.ToLower(personal.ClampPersonalText(kind, 40))
	profile, _ := state.personal.LoadProfile()
	switch kind {
	case "weather":
		city = personal.ClampPersonalText(city, 120)
		if city == "" {
			city = profile.LocationCity
		}
		weather, err := skills.LookupWeather(city)
		if err != nil {
			return personal.OperationNotice{}, err
		}
		encoded, err := json.Marshal(weather)
		if err != nil {
			return personal.OperationNotice{}, err
		}
		return operations.Enqueue(personal.OperationNotice{
			Kind: "weather", Priority: "normal", Title: "Wetter · " + city,
			Body: string(encoded), Source: "weather_lookup", Speak: false,
			Metadata: map[string]string{"format": "weather_json", "city": city},
		})
	case "daily_plan":
		goals := make([]string, 0, 4)
		for index, goal := range profile.Goals {
			if index >= 4 {
				break
			}
			if clean := personal.ClampPersonalText(goal, 240); clean != "" {
				goals = append(goals, clean)
			}
		}
		body := "1. Wichtigstes Tagesziel festlegen."
		for _, goal := range goals {
			body += "\n" + strconv.Itoa(strings.Count(body, "\n")+2) + ". " + goal
		}
		body += fmt.Sprintf("\n%d. Offene Runs prüfen und Tagesabschluss dokumentieren.", strings.Count(body, "\n")+2)
		return operations.Enqueue(personal.OperationNotice{
			Kind: "daily_plan", Priority: "normal", Title: "Tagesplan · " + time.Now().Format("02.01.2006"),
			Body: body, Source: "personal_core",
		})
	default:
		return personal.OperationNotice{}, errors.New("unsupported operation kind")
	}
}

func writeOperationError(w http.ResponseWriter, err error) {
	if errors.Is(err, os.ErrNotExist) {
		http.Error(w, "Operation not found", http.StatusNotFound)
		return
	}
	http.Error(w, "Operation request rejected", http.StatusBadRequest)
}
