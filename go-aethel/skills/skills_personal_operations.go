package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-aethel/personal"
	"go-aethel/security"
)

type PersonalOperationsSkill struct {
	Store *personal.PersonalStore
	Inbox *personal.OperationsStore
}

type personalOperationsArgs struct {
	Action string `json:"action"`
	Time   string `json:"time,omitempty"`
	City   string `json:"city,omitempty"`
}

func (s *PersonalOperationsSkill) Name() string { return "personal_operations" }
func (s *PersonalOperationsSkill) Description() string {
	return "Steuert Aethels Personal Operations: set_alarm konfiguriert einen Wecker, show_weather öffnet ein Wetter-Popup, show_daily_plan erstellt einen Tagesplan-Popup. Nutze die Funktion statt nur über die Aktion zu sprechen."
}
func (s *PersonalOperationsSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *PersonalOperationsSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{"type": "string", "enum": []string{"set_alarm", "show_weather", "show_daily_plan"}},
			"time":   map[string]interface{}{"type": "string", "description": "Lokale Weckzeit im 24-Stunden-Format HH:MM; nur für set_alarm."},
			"city":   map[string]interface{}{"type": "string", "description": "Optionaler Stadtname; Personal-Core-Standort wird sonst verwendet."},
		},
		"required":             []string{"action"},
		"additionalProperties": false,
	}
}

func (s *PersonalOperationsSkill) Execute(raw json.RawMessage) (string, error) {
	if s.Store == nil || s.Inbox == nil {
		return "", errors.New("personal operations unavailable")
	}
	var input personalOperationsArgs
	if err := json.Unmarshal(raw, &input); err != nil {
		return "", errors.New("invalid personal operations arguments")
	}
	profile, _ := s.Store.LoadProfile()
	switch strings.ToLower(strings.TrimSpace(input.Action)) {
	case "set_alarm":
		alarmTime := personal.ClampPersonalText(input.Time, 5)
		if _, err := time.Parse("15:04", alarmTime); err != nil {
			return "", errors.New("alarm time must use HH:MM")
		}
		cfg, err := s.Store.LoadConfig()
		if err != nil {
			return "", errors.New("personal configuration unavailable")
		}
		cfg.Enabled = true
		cfg.AlarmEnabled = true
		cfg.AlarmTime = alarmTime
		if err := s.Store.SaveConfig(cfg); err != nil {
			return "", errors.New("alarm configuration could not be saved")
		}
		return "Wecker wurde lokal und persistent auf " + alarmTime + " Uhr gestellt.", nil
	case "show_weather":
		city := personal.ClampPersonalText(input.City, 120)
		if city == "" {
			city = profile.LocationCity
		}
		weather, err := LookupWeather(city)
		if err != nil {
			return "", err
		}
		body := fmt.Sprintf("%s · %.1f °C · Wind %.1f km/h\nStand: %s", weather.Summary, weather.Temperature, weather.WindSpeed, weather.ObservedAt)
		_, err = s.Inbox.Enqueue(personal.OperationNotice{
			Kind: "weather", Priority: "normal", Title: "Wetter · " + weather.City,
			Body: body, Source: "weather_lookup",
		})
		if err != nil {
			return "", errors.New("weather popup could not be queued")
		}
		return "Aktuelle Wetterdaten wurden im Personal-Operations-Popup geöffnet.", nil
	case "show_daily_plan":
		lines := []string{"1. Wichtigste Aufgabe des Tages verbindlich priorisieren."}
		for _, goal := range profile.Goals {
			if len(lines) >= 5 {
				break
			}
			if goal = personal.ClampPersonalText(goal, 240); goal != "" {
				lines = append(lines, fmt.Sprintf("%d. %s", len(lines)+1, goal))
			}
		}
		lines = append(lines, fmt.Sprintf("%d. Offene Runs prüfen und Ergebnisse dokumentieren.", len(lines)+1))
		_, err := s.Inbox.Enqueue(personal.OperationNotice{
			Kind: "daily_plan", Priority: "normal", Title: "Tagesplan · " + time.Now().Format("02.01.2006"),
			Body: strings.Join(lines, "\n"), Source: "personal_core",
		})
		if err != nil {
			return "", errors.New("daily plan popup could not be queued")
		}
		return "Tagesplan wurde als Personal-Operations-Popup erstellt.", nil
	default:
		return "", errors.New("unsupported personal operations action")
	}
}
