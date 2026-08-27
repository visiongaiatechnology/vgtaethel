package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-aethel/intelligence"
	"go-aethel/security"
)

const (
	ShadowBatchMin = 40
	ShadowBatchMax = 60
)

type ShadowPercent int

func (p *ShadowPercent) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		return errors.New("SHADOW percentage is missing")
	}
	percentSuffix := false
	if strings.HasPrefix(text, "\"") {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return errors.New("invalid SHADOW percentage string")
		}
		text = strings.TrimSpace(value)
		percentSuffix = strings.HasSuffix(text, "%")
		text = strings.TrimSpace(strings.TrimSuffix(text, "%"))
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return errors.New("invalid SHADOW percentage")
	}
	if !percentSuffix && value >= 0 && value <= 1 {
		value *= 100
	}
	if value < 0 || value > 100 {
		return errors.New("SHADOW percentage outside 0-100 boundary")
	}
	*p = ShadowPercent(math.Round(value))
	return nil
}

type ShadowIntelItem struct {
	ID          string    `json:"id"`
	SourceID    string    `json:"source_id"`
	SourceName  string    `json:"source_name"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	URL         string    `json:"url"`
	Domain      string    `json:"domain"`
	PublishedAt time.Time `json:"published_at"`
	CollectedAt time.Time `json:"collected_at"`
	Processed   bool      `json:"processed"`
}

type ShadowRegionAssessment struct {
	RegionID      string        `json:"region_id"`
	RegionName    string        `json:"region_name"`
	Latitude      float64       `json:"latitude"`
	Longitude     float64       `json:"longitude"`
	SecurityScore ShadowPercent `json:"security_score"`
	ConflictLevel string        `json:"conflict_level"`
	Confidence    ShadowPercent `json:"confidence"`
	Trend         string        `json:"trend"`
	EvidenceIDs   []string      `json:"evidence_ids"`
	Assessment    string        `json:"assessment"`
}

type ShadowForecast struct {
	Sector      string        `json:"sector"`
	Horizon     string        `json:"horizon"`
	Prediction  string        `json:"prediction"`
	Probability ShadowPercent `json:"probability"`
	Direction   string        `json:"direction,omitempty"`
	Instruments []string      `json:"instruments,omitempty"`
	EvidenceIDs []string      `json:"evidence_ids"`
}

type ShadowMarketPoint struct {
	Symbol     string    `json:"symbol"`
	Name       string    `json:"name"`
	Category   string    `json:"category"`
	Currency   string    `json:"currency"`
	Price      float64   `json:"price"`
	Change24H  float64   `json:"change_24h_percent"`
	ObservedAt time.Time `json:"observed_at"`
	Source     string    `json:"source"`
}

type ShadowConflictLink struct {
	AttackerName      string        `json:"attacker_name"`
	TargetName        string        `json:"target_name"`
	AttackerLatitude  float64       `json:"attacker_latitude"`
	AttackerLongitude float64       `json:"attacker_longitude"`
	TargetLatitude    float64       `json:"target_latitude"`
	TargetLongitude   float64       `json:"target_longitude"`
	Action            string        `json:"action"`
	Confidence        ShadowPercent `json:"confidence"`
	EvidenceIDs       []string      `json:"evidence_ids"`
	Assessment        string        `json:"assessment"`
}

type ShadowReport struct {
	ID               string                   `json:"id"`
	Kind             string                   `json:"kind"`
	ThreatLevel      string                   `json:"threat_level"`
	Summary          string                   `json:"summary"`
	Situation        string                   `json:"situation"`
	CuiBono          string                   `json:"cui_bono"`
	StrategicReality string                   `json:"strategic_reality"`
	Divergences      string                   `json:"divergences,omitempty"`
	ConfirmedVectors string                   `json:"confirmed_vectors,omitempty"`
	Regions          []ShadowRegionAssessment `json:"regions"`
	ConflictLinks    []ShadowConflictLink     `json:"conflict_links"`
	Forecasts        []ShadowForecast         `json:"forecast_matrix"`
	MarketSnapshot   []ShadowMarketPoint      `json:"market_snapshot,omitempty"`
	EvidenceIDs      []string                 `json:"evidence_ids"`
	ItemsAnalyzed    int                      `json:"items_analyzed"`
	CreatedAt        time.Time                `json:"created_at"`
	ContentSHA256    string                   `json:"content_sha256"`
}

type ShadowState struct {
	Sources         []ShadowSource    `json:"sources"`
	Buffer          []ShadowIntelItem `json:"buffer"`
	Reports         []ShadowReport    `json:"reports"`
	SourceCursor    int               `json:"source_cursor"`
	SystemPrompt    string            `json:"system_prompt"`
	AutonomyEnabled bool              `json:"autonomy_enabled"`
	AutonomyModelID string            `json:"autonomy_model_id,omitempty"`
	LastCollectAt   time.Time         `json:"last_collect_at,omitempty"`
}

type ShadowStatus struct {
	Sources         int       `json:"sources"`
	EnabledSources  int       `json:"enabled_sources"`
	PendingItems    int       `json:"pending_items"`
	ProcessedItems  int       `json:"processed_items"`
	Reports         int       `json:"reports"`
	BatchMin        int       `json:"batch_min"`
	BatchMax        int       `json:"batch_max"`
	AnalysisRunning bool      `json:"analysis_running"`
	AutonomyEnabled bool      `json:"autonomy_enabled"`
	AutonomyModelID string    `json:"autonomy_model_id,omitempty"`
	AnalysisModelID string    `json:"analysis_model_id,omitempty"`
	AnalysisStarted time.Time `json:"analysis_started_at,omitempty"`
	LastAnalysisAt  time.Time `json:"last_analysis_at,omitempty"`
	LastAnalysisErr string    `json:"last_analysis_error,omitempty"`
	LastCollectAt   time.Time `json:"last_collect_at,omitempty"`
}

type ShadowService struct {
	mu              sync.RWMutex
	path            string
	state           ShadowState
	analysisRunning bool
	analysisModelID string
	analysisStarted time.Time
	lastAnalysisAt  time.Time
	lastAnalysisErr string
}

func NewShadowService(path string) *ShadowService {
	service := &ShadowService{path: path}
	if err := service.load(); err != nil {
		service.state = ShadowState{Sources: defaultShadowSources(), SystemPrompt: DefaultShadowSystemPrompt()}
		_ = service.saveLocked()
	}
	return service
}

func (s *ShadowService) load() error {
	data, sealed, err := security.ReadSealedFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return err
		}
		return fmt.Errorf("open SHADOW store: %w", err)
	}
	var state ShadowState
	if err := json.Unmarshal(data, &state); err != nil {
		return errors.New("invalid SHADOW store")
	}
	if len(state.Sources) == 0 {
		state.Sources = defaultShadowSources()
	}
	if strings.TrimSpace(state.SystemPrompt) == "" {
		state.SystemPrompt = DefaultShadowSystemPrompt()
	}
	s.state = state
	if !sealed {
		return s.saveLocked()
	}
	return nil
}

func (s *ShadowService) saveLocked() error {
	data, err := json.Marshal(s.state)
	if err != nil {
		return err
	}
	return security.WriteSealedFile(s.path, data)
}

func (s *ShadowService) Snapshot() ShadowState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, _ := json.Marshal(s.state)
	var result ShadowState
	_ = json.Unmarshal(data, &result)
	return result
}

func (s *ShadowService) Status() ShadowStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status := ShadowStatus{
		Sources: len(s.state.Sources), Reports: len(s.state.Reports), BatchMin: ShadowBatchMin, BatchMax: ShadowBatchMax,
		AnalysisRunning: s.analysisRunning, AutonomyEnabled: s.state.AutonomyEnabled, AutonomyModelID: s.state.AutonomyModelID,
		AnalysisModelID: s.analysisModelID, AnalysisStarted: s.analysisStarted, LastAnalysisAt: s.lastAnalysisAt,
		LastAnalysisErr: s.lastAnalysisErr, LastCollectAt: s.state.LastCollectAt,
	}
	for _, source := range s.state.Sources {
		if source.Enabled {
			status.EnabledSources++
		}
	}
	for _, item := range s.state.Buffer {
		if item.Processed {
			status.ProcessedItems++
		} else {
			status.PendingItems++
		}
	}
	return status
}

func (s *ShadowService) SetAutonomy(enabled bool, modelID string) error {
	modelID = strings.TrimSpace(modelID)
	if enabled && !validShadowModelID(modelID) {
		return errors.New("invalid SHADOW autonomy model")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previousEnabled, previousModel := s.state.AutonomyEnabled, s.state.AutonomyModelID
	s.state.AutonomyEnabled = enabled
	if modelID != "" {
		s.state.AutonomyModelID = modelID
	}
	if err := s.saveLocked(); err != nil {
		s.state.AutonomyEnabled, s.state.AutonomyModelID = previousEnabled, previousModel
		return err
	}
	return nil
}

func (s *ShadowService) Autonomy() (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.AutonomyEnabled, s.state.AutonomyModelID
}

func validShadowModelID(modelID string) bool {
	if modelID == "" || len(modelID) > 256 {
		return false
	}
	for _, char := range modelID {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("/._:-", char) {
			continue
		}
		return false
	}
	return true
}

func (s *ShadowService) UpsertSource(source ShadowSource) error {
	source.Name = strings.TrimSpace(source.Name)
	source.URL = strings.TrimSpace(source.URL)
	source.Type = strings.ToLower(strings.TrimSpace(source.Type))
	source.Domain = strings.ToLower(strings.TrimSpace(source.Domain))
	if source.Name == "" || len([]rune(source.Name)) > 120 || len(source.URL) > 2048 {
		return errors.New("invalid SHADOW source")
	}
	if source.Type != "rss" && source.Type != "telegram" && source.Type != "web" {
		return errors.New("unsupported SHADOW source type")
	}
	if source.Type == "telegram" {
		if _, err := normalizeTelegramPublicURL(source.URL); err != nil {
			return err
		}
	} else if err := ValidatePublicCollectorURL(source.URL); err != nil {
		return err
	}
	if source.Priority < 1 || source.Priority > 5 {
		source.Priority = 3
	}
	if source.ID == "" {
		source.ID = shadowSourceID(source.Name, source.URL)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.state.Sources {
		if s.state.Sources[index].ID == source.ID {
			source.LastState, source.LastError, source.LastFetch = s.state.Sources[index].LastState, s.state.Sources[index].LastError, s.state.Sources[index].LastFetch
			s.state.Sources[index] = source
			return s.saveLocked()
		}
	}
	s.state.Sources = append(s.state.Sources, source)
	return s.saveLocked()
}

func (s *ShadowService) DeleteSource(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.state.Sources {
		if s.state.Sources[index].ID == id {
			s.state.Sources = append(s.state.Sources[:index], s.state.Sources[index+1:]...)
			return s.saveLocked()
		}
	}
	return errors.New("SHADOW source not found")
}

func (s *ShadowService) SetSystemPrompt(prompt string) error {
	prompt = strings.TrimSpace(prompt)
	if len([]rune(prompt)) < 200 || len([]rune(prompt)) > 20000 {
		return errors.New("SHADOW doctrine length outside boundary")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.SystemPrompt = prompt
	return s.saveLocked()
}

func (s *ShadowService) AnalysisPrompt() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.SystemPrompt + "\n\n" + MandatoryShadowV3Contract()
}

func (s *ShadowService) Collect(ctx context.Context, sourceLimit int) (int, error) {
	if sourceLimit < 1 || sourceLimit > 20 {
		sourceLimit = 8
	}
	s.mu.Lock()
	selected := make([]ShadowSource, 0, sourceLimit)
	if len(s.state.Sources) > 0 {
		for checked := 0; checked < len(s.state.Sources) && len(selected) < sourceLimit; checked++ {
			index := (s.state.SourceCursor + checked) % len(s.state.Sources)
			source := s.state.Sources[index]
			if source.Enabled {
				selected = append(selected, source)
			}
		}
		s.state.SourceCursor = (s.state.SourceCursor + max(1, len(selected))) % len(s.state.Sources)
	}
	s.mu.Unlock()
	if len(selected) == 0 {
		return 0, errors.New("no enabled collectable SHADOW sources")
	}
	type result struct {
		source ShadowSource
		events []intelligence.OSINTEvent
		err    error
	}
	results := make(chan result, len(selected))
	semaphore := make(chan struct{}, 5)
	var wait sync.WaitGroup
	for _, source := range selected {
		wait.Add(1)
		go func(src ShadowSource) {
			defer wait.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			var collector FeedCollector
			cfg := OSINTCollectorConfig{Name: src.Name, URL: src.URL, Domain: intelligence.OSINTDomain(src.Domain), Enabled: true, Priority: src.Priority}
			if src.Type == "telegram" {
				collector = NewTelegramCollector(cfg)
			} else if src.Type == "web" {
				collector = NewWebIndexCollector(cfg)
			} else {
				collector = NewRSSCollector(cfg)
			}
			child, cancel := context.WithTimeout(ctx, 18*time.Second)
			defer cancel()
			events, err := collector.Collect(child)
			results <- result{source: src, events: events, err: err}
		}(source)
	}
	wait.Wait()
	close(results)
	now := time.Now().UTC()
	added := 0
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := make(map[string]bool, len(s.state.Buffer))
	for _, item := range s.state.Buffer {
		seen[item.ID] = true
	}
	for result := range results {
		for index := range s.state.Sources {
			if s.state.Sources[index].ID == result.source.ID {
				s.state.Sources[index].LastFetch = now.Format(time.RFC3339)
				if result.err != nil {
					s.state.Sources[index].LastState = "error"
					s.state.Sources[index].LastError = result.err.Error()
				} else {
					s.state.Sources[index].LastState = "healthy"
					s.state.Sources[index].LastError = ""
				}
			}
		}
		for _, event := range result.events {
			if seen[event.ID] {
				continue
			}
			seen[event.ID] = true
			s.state.Buffer = append(s.state.Buffer, ShadowIntelItem{ID: event.ID, SourceID: result.source.ID, SourceName: result.source.Name, Title: event.Title, Summary: event.Summary, URL: event.URL, Domain: result.source.Domain, PublishedAt: event.Timestamp.UTC(), CollectedAt: now})
			added++
		}
	}
	if len(s.state.Buffer) > 5000 {
		s.state.Buffer = append([]ShadowIntelItem(nil), s.state.Buffer[len(s.state.Buffer)-5000:]...)
	}
	s.state.LastCollectAt = now
	return added, s.saveLocked()
}

func (s *ShadowService) PrepareBatch(modelID string) ([]ShadowIntelItem, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.analysisRunning {
		return nil, "", errors.New("SHADOW analysis already running")
	}
	pending := make([]ShadowIntelItem, 0, ShadowBatchMax)
	for _, item := range s.state.Buffer {
		if !item.Processed {
			pending = append(pending, item)
			if len(pending) == ShadowBatchMax {
				break
			}
		}
	}
	if len(pending) < ShadowBatchMin {
		return nil, "", fmt.Errorf("SHADOW requires at least %d pending items", ShadowBatchMin)
	}
	s.analysisRunning = true
	s.analysisModelID = strings.TrimSpace(modelID)
	s.analysisStarted = time.Now().UTC()
	s.lastAnalysisErr = ""
	return pending, s.state.SystemPrompt + "\n\n" + MandatoryShadowV3Contract(), nil
}

func (s *ShadowService) AbortBatch() {
	s.mu.Lock()
	s.analysisRunning = false
	s.analysisModelID = ""
	s.analysisStarted = time.Time{}
	s.mu.Unlock()
}

func (s *ShadowService) FailBatch(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.analysisRunning = false
	s.analysisModelID = ""
	s.analysisStarted = time.Time{}
	s.lastAnalysisAt = time.Now().UTC()
	s.lastAnalysisErr = "SHADOW analysis failed"
	if err != nil {
		message := strings.TrimSpace(err.Error())
		if len(message) > 500 {
			message = message[:500]
		}
		if message != "" {
			s.lastAnalysisErr = message
		}
	}
}

func (s *ShadowService) CompleteBatch(items []ShadowIntelItem, report ShadowReport) (ShadowReport, error) {
	if len(items) < ShadowBatchMin || len(items) > ShadowBatchMax {
		return ShadowReport{}, errors.New("invalid SHADOW batch boundary")
	}
	allowed := make(map[string]bool, len(items))
	for _, item := range items {
		allowed[item.ID] = true
	}
	if err := validateShadowReport(&report, allowed); err != nil {
		return ShadowReport{}, err
	}
	report.ID = fmt.Sprintf("shadow-report-%d", time.Now().UTC().UnixNano())
	report.Kind = "batch"
	report.ItemsAnalyzed = len(items)
	report.CreatedAt = time.Now().UTC()
	content, _ := json.Marshal(report)
	sum := sha256.Sum256(content)
	report.ContentSHA256 = hex.EncodeToString(sum[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		s.analysisRunning = false
		s.analysisModelID = ""
		s.analysisStarted = time.Time{}
	}()
	previousBuffer := append([]ShadowIntelItem(nil), s.state.Buffer...)
	previousReports := append([]ShadowReport(nil), s.state.Reports...)
	for index := range s.state.Buffer {
		if allowed[s.state.Buffer[index].ID] {
			s.state.Buffer[index].Processed = true
		}
	}
	s.state.Reports = append([]ShadowReport{report}, s.state.Reports...)
	if len(s.state.Reports) > 365 {
		s.state.Reports = s.state.Reports[:365]
	}
	if err := s.saveLocked(); err != nil {
		s.state.Buffer = previousBuffer
		s.state.Reports = previousReports
		return ShadowReport{}, err
	}
	s.lastAnalysisAt = time.Now().UTC()
	s.lastAnalysisErr = ""
	return report, nil
}

func validateShadowReport(report *ShadowReport, allowed map[string]bool) error {
	report.ThreatLevel = strings.ToUpper(strings.TrimSpace(report.ThreatLevel))
	if report.ThreatLevel != "LOW" && report.ThreatLevel != "MEDIUM" && report.ThreatLevel != "HIGH" && report.ThreatLevel != "CRITICAL" {
		return errors.New("invalid SHADOW threat level")
	}
	if len([]rune(report.Summary)) < 40 || len([]rune(report.Summary)) > 12000 {
		return errors.New("invalid SHADOW summary")
	}
	for _, section := range []string{report.Situation, report.CuiBono, report.StrategicReality, report.Divergences, report.ConfirmedVectors} {
		if len([]rune(section)) > 24000 {
			return errors.New("SHADOW report section boundary exceeded")
		}
	}
	if len(report.EvidenceIDs) == 0 {
		return errors.New("SHADOW report requires batch evidence")
	}
	if len(report.EvidenceIDs) > ShadowBatchMax || len(report.Regions) > 160 || len(report.Forecasts) > 80 {
		return errors.New("SHADOW report collection boundary exceeded")
	}
	for _, id := range report.EvidenceIDs {
		if !allowed[id] {
			return errors.New("SHADOW report referenced evidence outside batch")
		}
	}
	for index := range report.Regions {
		region := &report.Regions[index]
		region.RegionID = strings.ToUpper(strings.TrimSpace(region.RegionID))
		region.ConflictLevel = strings.ToUpper(strings.TrimSpace(region.ConflictLevel))
		region.Trend = strings.ToUpper(strings.TrimSpace(region.Trend))
		if region.RegionID == "" || len([]rune(region.RegionID)) > 80 || len([]rune(region.RegionName)) > 160 || len([]rune(region.Assessment)) > 4000 || region.Latitude < -90 || region.Latitude > 90 || region.Longitude < -180 || region.Longitude > 180 || region.SecurityScore < 0 || region.SecurityScore > 100 || region.Confidence < 1 || region.Confidence > 100 {
			return errors.New("invalid SHADOW region assessment")
		}
		if region.ConflictLevel != "STABLE" && region.ConflictLevel != "TENSION" && region.ConflictLevel != "ESCALATION" && region.ConflictLevel != "WAR" {
			return errors.New("invalid SHADOW conflict classification")
		}
		if len(region.EvidenceIDs) == 0 {
			return errors.New("SHADOW region requires evidence")
		}
		for _, id := range region.EvidenceIDs {
			if !allowed[id] {
				return errors.New("SHADOW region referenced evidence outside batch")
			}
		}
	}
	if len(report.ConflictLinks) > 80 {
		return errors.New("SHADOW conflict link boundary exceeded")
	}
	for index := range report.ConflictLinks {
		link := &report.ConflictLinks[index]
		link.AttackerName = strings.TrimSpace(link.AttackerName)
		link.TargetName = strings.TrimSpace(link.TargetName)
		link.Action = strings.ToUpper(strings.TrimSpace(link.Action))
		link.Assessment = strings.TrimSpace(link.Assessment)
		if link.AttackerName == "" || link.TargetName == "" || strings.EqualFold(link.AttackerName, link.TargetName) ||
			len([]rune(link.AttackerName)) > 120 || len([]rune(link.TargetName)) > 120 ||
			link.AttackerLatitude < -90 || link.AttackerLatitude > 90 || link.TargetLatitude < -90 || link.TargetLatitude > 90 ||
			link.AttackerLongitude < -180 || link.AttackerLongitude > 180 || link.TargetLongitude < -180 || link.TargetLongitude > 180 ||
			link.Confidence < 1 || link.Confidence > 100 || len([]rune(link.Assessment)) < 20 || len([]rune(link.Assessment)) > 2000 {
			return errors.New("invalid SHADOW conflict link")
		}
		switch link.Action {
		case "ATTACK", "INVASION", "STRIKE", "BLOCKADE", "OCCUPATION", "PROXY_ATTACK", "CYBER_ATTACK", "MILITARY_SUPPORT":
		default:
			return errors.New("invalid SHADOW conflict action")
		}
		if len(link.EvidenceIDs) == 0 {
			return errors.New("SHADOW conflict link requires evidence")
		}
		for _, id := range link.EvidenceIDs {
			if !allowed[id] {
				return errors.New("SHADOW conflict link referenced evidence outside batch")
			}
		}
	}
	for index := range report.Forecasts {
		forecast := &report.Forecasts[index]
		forecast.Sector = strings.ToUpper(strings.TrimSpace(forecast.Sector))
		forecast.Horizon = strings.TrimSpace(forecast.Horizon)
		forecast.Direction = strings.ToUpper(strings.TrimSpace(forecast.Direction))
		if forecast.Probability < 0 || forecast.Probability > 100 {
			return errors.New("invalid SHADOW forecast probability")
		}
		if len([]rune(forecast.Sector)) > 80 || len([]rune(forecast.Horizon)) > 40 || len([]rune(forecast.Prediction)) < 10 || len([]rune(forecast.Prediction)) > 4000 || len(forecast.Instruments) > 20 {
			return errors.New("invalid SHADOW forecast structure")
		}
		if forecast.Direction != "" && forecast.Direction != "UP" && forecast.Direction != "DOWN" && forecast.Direction != "SIDEWAYS" && forecast.Direction != "VOLATILE" && forecast.Direction != "ESCALATION" && forecast.Direction != "IMPROVEMENT" && forecast.Direction != "STABLE" {
			return errors.New("invalid SHADOW forecast direction")
		}
		if len(forecast.EvidenceIDs) == 0 {
			return errors.New("SHADOW forecast requires evidence")
		}
		for _, id := range forecast.EvidenceIDs {
			if !allowed[id] {
				return errors.New("SHADOW forecast referenced evidence outside batch")
			}
		}
	}
	if len(report.MarketSnapshot) > 32 {
		return errors.New("SHADOW market snapshot boundary exceeded")
	}
	for _, point := range report.MarketSnapshot {
		if !validShadowMarketSymbol(point.Symbol) || point.Price <= 0 || point.ObservedAt.IsZero() || len([]rune(point.Name)) > 120 || len([]rune(point.Source)) > 120 {
			return errors.New("invalid SHADOW market snapshot")
		}
	}
	return nil
}

func validShadowMarketSymbol(symbol string) bool {
	switch strings.ToUpper(strings.TrimSpace(symbol)) {
	case "BTC", "ETH", "GOLD", "BRENT", "WTI", "SP500", "NASDAQ", "DAX", "EURUSD", "GBPUSD", "USDJPY":
		return true
	default:
		return false
	}
}

func (s *ShadowService) LatestRegions() []ShadowRegionAssessment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.state.Reports) == 0 {
		return []ShadowRegionAssessment{}
	}
	result := append([]ShadowRegionAssessment(nil), s.state.Reports[0].Regions...)
	sort.Slice(result, func(i, j int) bool { return result[i].SecurityScore < result[j].SecurityScore })
	return result
}

func (s *ShadowService) LatestConflictLinks() []ShadowConflictLink {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.state.Reports) == 0 {
		return []ShadowConflictLink{}
	}
	return append([]ShadowConflictLink(nil), s.state.Reports[0].ConflictLinks...)
}

func (s *ShadowService) Reports() []ShadowReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := append([]ShadowReport(nil), s.state.Reports...)
	return result
}

func (s *ShadowService) ReportsForDay(day time.Time) []ShadowReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	year, month, date := day.Date()
	result := make([]ShadowReport, 0)
	for _, report := range s.state.Reports {
		y, m, d := report.CreatedAt.In(day.Location()).Date()
		if y == year && m == month && d == date && report.Kind == "batch" {
			result = append(result, report)
		}
	}
	return result
}

func (s *ShadowService) SaveDailyReport(report ShadowReport) (ShadowReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	allowed := make(map[string]bool, len(s.state.Buffer))
	for _, item := range s.state.Buffer {
		allowed[item.ID] = true
	}
	if err := validateShadowReport(&report, allowed); err != nil {
		return ShadowReport{}, err
	}
	report.ID = fmt.Sprintf("shadow-daily-%d", time.Now().UTC().UnixNano())
	report.Kind = "daily"
	report.CreatedAt = time.Now().UTC()
	content, _ := json.Marshal(report)
	sum := sha256.Sum256(content)
	report.ContentSHA256 = hex.EncodeToString(sum[:])
	previousReports := append([]ShadowReport(nil), s.state.Reports...)
	s.state.Reports = append([]ShadowReport{report}, s.state.Reports...)
	if len(s.state.Reports) > 365 {
		s.state.Reports = s.state.Reports[:365]
	}
	if err := s.saveLocked(); err != nil {
		s.state.Reports = previousReports
		return ShadowReport{}, err
	}
	return report, nil
}
