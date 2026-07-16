package skills

import (
	"go-aethel/security"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// WeatherSkill is deliberately limited to a city lookup. It never accepts an
// arbitrary remote URL, which keeps the convenience widget free of SSRF paths.
type WeatherSkill struct{}

type WeatherArgs struct {
	City string `json:"city"`
}

type weatherSnapshot struct {
	City        string  `json:"city"`
	Country     string  `json:"country,omitempty"`
	Temperature float64 `json:"temperature_c"`
	WindSpeed   float64 `json:"wind_speed_kmh"`
	WeatherCode int     `json:"weather_code"`
	Summary     string  `json:"summary"`
	ObservedAt  string  `json:"observed_at"`
}

func (s *WeatherSkill) Name() string { return "weather_lookup" }
func (s *WeatherSkill) Description() string {
	return "Liefert aktuelle Wetterbedingungen fuer eine Stadt. Nutze dieses Tool fuer Fragen wie 'Wie ist das Wetter in Koeln?'."
}
func (s *WeatherSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *WeatherSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"city": map[string]interface{}{"type": "string", "description": "Stadtname, zum Beispiel Koeln oder Berlin"},
		},
		"required": []string{"city"},
	}
}

func (s *WeatherSkill) Execute(args json.RawMessage) (string, error) {
	var input WeatherArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("ungueltige Wetteranfrage")
	}
	snapshot, err := lookupWeather(strings.TrimSpace(input.City))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Wetter in %s%s: %s, %.1f Grad C, Wind %.1f km/h (Stand: %s).", snapshot.City, weatherCountrySuffix(snapshot.Country), snapshot.Summary, snapshot.Temperature, snapshot.WindSpeed, snapshot.ObservedAt), nil
}

func weatherCountrySuffix(country string) string {
	if strings.TrimSpace(country) == "" {
		return ""
	}
	return ", " + country
}

// LookupWeather is the city-only weather lookup used by skills and HTTP handlers.
func LookupWeather(city string) (weatherSnapshot, error) {
	return lookupWeather(city)
}

func lookupWeather(city string) (weatherSnapshot, error) {
	if !validWeatherCity(city) {
		return weatherSnapshot{}, errors.New("bitte gib eine gueltige Stadt mit 2 bis 80 Zeichen an")
	}
	client := &http.Client{Timeout: 8 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	defer cancel()

	geoURL := "https://geocoding-api.open-meteo.com/v1/search?name=" + url.QueryEscape(city) + "&count=1&language=de&format=json"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, geoURL, nil)
	if err != nil {
		return weatherSnapshot{}, errors.New("Wetterdienst konnte nicht vorbereitet werden")
	}
	response, err := client.Do(request)
	if err != nil || response.StatusCode != http.StatusOK {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		return weatherSnapshot{}, errors.New("Wetterdienst ist momentan nicht erreichbar")
	}
	defer response.Body.Close()
	var geo struct {
		Results []struct {
			Name      string  `json:"name"`
			Country   string  `json:"country"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&geo); err != nil || len(geo.Results) == 0 {
		return weatherSnapshot{}, errors.New("Stadt wurde nicht gefunden")
	}
	place := geo.Results[0]
	forecastURL := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.6f&longitude=%.6f&current=temperature_2m,wind_speed_10m,weather_code&timezone=auto", place.Latitude, place.Longitude)
	forecastRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, forecastURL, nil)
	if err != nil {
		return weatherSnapshot{}, errors.New("Wetterdienst konnte nicht vorbereitet werden")
	}
	forecastResponse, err := client.Do(forecastRequest)
	if err != nil || forecastResponse.StatusCode != http.StatusOK {
		if forecastResponse != nil && forecastResponse.Body != nil {
			forecastResponse.Body.Close()
		}
		return weatherSnapshot{}, errors.New("Wetterdienst ist momentan nicht erreichbar")
	}
	defer forecastResponse.Body.Close()
	var forecast struct {
		Current struct {
			Temperature float64 `json:"temperature_2m"`
			WindSpeed   float64 `json:"wind_speed_10m"`
			WeatherCode int     `json:"weather_code"`
			Time        string  `json:"time"`
		} `json:"current"`
	}
	if err := json.NewDecoder(io.LimitReader(forecastResponse.Body, 1<<20)).Decode(&forecast); err != nil {
		return weatherSnapshot{}, errors.New("Wetterdaten konnten nicht gelesen werden")
	}
	return weatherSnapshot{City: place.Name, Country: place.Country, Temperature: forecast.Current.Temperature, WindSpeed: forecast.Current.WindSpeed, WeatherCode: forecast.Current.WeatherCode, Summary: weatherDescription(forecast.Current.WeatherCode), ObservedAt: forecast.Current.Time}, nil
}

func validWeatherCity(city string) bool {
	if utf8.RuneCountInString(city) < 2 || utf8.RuneCountInString(city) > 80 {
		return false
	}
	for _, char := range city {
		if !(char == ' ' || char == '-' || char == '\'' || char == '.' || char == ',' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || char >= 0x00C0) {
			return false
		}
	}
	return true
}

func weatherDescription(code int) string {
	switch code {
	case 0:
		return "klar"
	case 1, 2:
		return "leicht bewoelkt"
	case 3:
		return "bedeckt"
	case 45, 48:
		return "neblig"
	case 51, 53, 55, 56, 57:
		return "Nieselregen"
	case 61, 63, 65, 66, 67, 80, 81, 82:
		return "Regen"
	case 71, 73, 75, 77, 85, 86:
		return "Schnee"
	case 95, 96, 99:
		return "Gewitter"
	default:
		return "wechselhaft"
	}
}
