package personal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"go-aethel/security"
)

const maxOperationInboxItems = 200

// OperationNotice is the single delivery contract for proactive Aethel output.
// Frontends render these values with textContent; Body never contains trusted HTML.
type OperationNotice struct {
	ID             string            `json:"id"`
	Kind           string            `json:"kind"`
	Priority       string            `json:"priority"`
	Title          string            `json:"title"`
	Body           string            `json:"body"`
	Source         string            `json:"source"`
	Speak          bool              `json:"speak"`
	RequireAck     bool              `json:"require_ack"`
	CreatedAt      time.Time         `json:"created_at"`
	DeliverAt      time.Time         `json:"deliver_at"`
	ExpiresAt      *time.Time        `json:"expires_at,omitempty"`
	AcknowledgedAt *time.Time        `json:"acknowledged_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type OperationsStore struct {
	mu       sync.RWMutex
	filePath string
	items    []OperationNotice
}

func NewOperationsStore(filePath string) *OperationsStore {
	return &OperationsStore{filePath: filePath, items: make([]OperationNotice, 0)}
}

func (s *OperationsStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, _, err := security.ReadSealedFile(s.filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var items []OperationNotice
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	s.items = items
	return nil
}

func operationID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "op_" + hex.EncodeToString(value[:]), nil
}

func (s *OperationsStore) Enqueue(notice OperationNotice) (OperationNotice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if notice.ID == "" {
		id, err := operationID()
		if err != nil {
			return notice, err
		}
		notice.ID = id
	}
	now := time.Now().UTC()
	if notice.CreatedAt.IsZero() {
		notice.CreatedAt = now
	}
	if notice.DeliverAt.IsZero() {
		notice.DeliverAt = now
	}
	notice.Kind = ClampPersonalText(strings.ToLower(notice.Kind), 40)
	notice.Priority = ClampPersonalText(strings.ToLower(notice.Priority), 20)
	notice.Title = ClampPersonalText(notice.Title, 160)
	notice.Body = ClampPersonalText(notice.Body, 12000)
	notice.Source = ClampPersonalText(notice.Source, 80)
	if notice.Kind == "" || notice.Title == "" || notice.Body == "" {
		return notice, errors.New("operation notice requires kind, title and body")
	}
	if notice.Priority == "" {
		notice.Priority = "normal"
	}
	s.items = append(s.items, notice)
	s.compactLocked(now)
	return notice, s.saveLocked()
}

func (s *OperationsStore) Pending(now time.Time) []OperationNotice {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]OperationNotice, 0)
	for _, item := range s.items {
		if item.AcknowledgedAt != nil || item.DeliverAt.After(now) {
			continue
		}
		if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
			continue
		}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

func (s *OperationsStore) Acknowledge(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for index := range s.items {
		if s.items[index].ID == id {
			s.items[index].AcknowledgedAt = &now
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}

func (s *OperationsStore) Snooze(id string, duration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if duration < time.Minute || duration > 24*time.Hour {
		return errors.New("unsupported snooze duration")
	}
	for index := range s.items {
		if s.items[index].ID == id {
			s.items[index].DeliverAt = time.Now().UTC().Add(duration)
			return s.saveLocked()
		}
	}
	return os.ErrNotExist
}

func (s *OperationsStore) saveLocked() error {
	if err := os.MkdirAll(filepathDir(s.filePath), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	return security.WriteSealedFile(s.filePath, data)
}

func (s *OperationsStore) compactLocked(now time.Time) {
	active := make([]OperationNotice, 0, len(s.items))
	for _, item := range s.items {
		if item.ExpiresAt != nil && item.ExpiresAt.Before(now.Add(-24*time.Hour)) {
			continue
		}
		active = append(active, item)
	}
	if len(active) > maxOperationInboxItems {
		active = active[len(active)-maxOperationInboxItems:]
	}
	s.items = active
}

func filepathDir(path string) string {
	index := strings.LastIndexAny(path, `/\\`)
	if index < 0 {
		return "."
	}
	return path[:index]
}

// OperationsService evaluates deterministic schedules locally. No model call is
// performed on the timer path, keeping routine token usage at zero.
type OperationsService struct {
	store           *OperationsStore
	personalStore   *PersonalStore
	stop            chan struct{}
	done            chan struct{}
	mu              sync.Mutex
	started         bool
	lastAlarmDay    string
	lastPlanDay     string
	lastWeatherDay  string
	weatherProvider func(city string) (title string, body string, err error)
}

func (s *OperationsService) SetWeatherProvider(provider func(city string) (title string, body string, err error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.weatherProvider = provider
}

func NewOperationsService(store *OperationsStore, personalStore *PersonalStore) *OperationsService {
	return &OperationsService{
		store: store, personalStore: personalStore,
		stop: make(chan struct{}), done: make(chan struct{}),
	}
}

func (s *OperationsService) Start() {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		s.evaluate(time.Now())
		for {
			select {
			case <-ticker.C:
				s.evaluate(time.Now())
			case <-s.stop:
				return
			}
		}
	}()
}

func (s *OperationsService) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.started = false
	close(s.stop)
	s.mu.Unlock()
	<-s.done
}

func (s *OperationsService) evaluate(now time.Time) {
	cfg, err := s.personalStore.LoadConfig()
	if err != nil || !cfg.Enabled {
		return
	}
	profile, _ := s.personalStore.LoadProfile()
	day := now.Format("2006-01-02")
	if cfg.AlarmEnabled && clockDue(now, cfg.AlarmTime, 3*time.Hour) && s.lastAlarmDay != day {
		s.lastAlarmDay = day
		name := strings.TrimSpace(profile.DisplayName)
		if name == "" {
			name = "Operator"
		}
		_, _ = s.store.Enqueue(OperationNotice{
			Kind: "alarm", Priority: "critical", Title: "Guten Morgen, " + name,
			Body:   "Deine konfigurierte Weckzeit ist erreicht. Aethel hält den Alarm aktiv, bis du ihn bestätigst.",
			Source: "personal_core", Speak: cfg.AlarmReadAloud, RequireAck: true,
		})
	}
	if cfg.DailyPlanEnabled && clockDue(now, planDeliveryTime(cfg.AlarmTime), 3*time.Hour) && s.lastPlanDay != day {
		s.lastPlanDay = day
		body := buildDeterministicDailyPlan(profile, now)
		_, _ = s.store.Enqueue(OperationNotice{
			Kind: "daily_plan", Priority: "normal", Title: "Tagesplan · " + now.Format("02.01.2006"),
			Body: body, Source: "personal_core", Speak: false, RequireAck: false,
		})
	}
	if cfg.WeatherUpdates && clockDue(now, planDeliveryTime(cfg.AlarmTime), 3*time.Hour) && s.lastWeatherDay != day {
		s.lastWeatherDay = day
		s.mu.Lock()
		provider := s.weatherProvider
		s.mu.Unlock()
		city := strings.TrimSpace(profile.LocationCity)
		if provider != nil && city != "" {
			if title, body, providerErr := provider(city); providerErr == nil {
				_, _ = s.store.Enqueue(OperationNotice{
					Kind: "weather", Priority: "normal", Title: title, Body: body,
					Source: "weather_lookup", Speak: false, RequireAck: false,
				})
			}
		}
	}
}

func clockDue(now time.Time, value string, grace time.Duration) bool {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return false
	}
	due := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
	return !now.Before(due) && now.Before(due.Add(grace))
}

func planDeliveryTime(alarmTime string) string {
	parsed, err := time.Parse("15:04", alarmTime)
	if err != nil {
		return "08:00"
	}
	return parsed.Add(5 * time.Minute).Format("15:04")
}

func buildDeterministicDailyPlan(profile PersonalProfile, now time.Time) string {
	lines := []string{
		fmt.Sprintf("%s · %s", now.Format("Monday"), now.Format("02.01.2006")),
		"1. Tagesziele prüfen und die wichtigste Aufgabe verbindlich priorisieren.",
	}
	for index, goal := range profile.Goals {
		if index >= 3 {
			break
		}
		goal = ClampPersonalText(goal, 240)
		if goal != "" {
			lines = append(lines, fmt.Sprintf("%d. Fokus: %s", len(lines), goal))
		}
	}
	lines = append(lines, fmt.Sprintf("%d. Offene Aethel-Runs prüfen und den Tagesabschluss dokumentieren.", len(lines)))
	return strings.Join(lines, "\n")
}
