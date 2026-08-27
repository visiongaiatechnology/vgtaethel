package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type SearchRequest struct {
	Query     string    `json:"query"`
	CaseID    string    `json:"case_id,omitempty"`
	Domains   []string  `json:"domains,omitempty"`
	SourceIDs []string  `json:"source_ids,omitempty"`
	From      time.Time `json:"from,omitempty"`
	To        time.Time `json:"to,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

type SearchHit struct {
	RecordType string    `json:"record_type"`
	RecordID   string    `json:"record_id"`
	CaseID     string    `json:"case_id,omitempty"`
	SourceID   string    `json:"source_id,omitempty"`
	Domain     string    `json:"domain,omitempty"`
	Title      string    `json:"title"`
	Snippet    string    `json:"snippet"`
	Score      int       `json:"score"`
	Timestamp  time.Time `json:"timestamp,omitempty"`
}

type SavedSearchMonitor struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Request      SearchRequest `json:"request"`
	MinimumScore int           `json:"minimum_score"`
	Enabled      bool          `json:"enabled"`
	SeenHitIDs   []string      `json:"seen_hit_ids"`
	CreatedAt    time.Time     `json:"created_at"`
	LastRunAt    time.Time     `json:"last_run_at,omitempty"`
}

type SearchMonitorAlert struct {
	ID        string    `json:"id"`
	MonitorID string    `json:"monitor_id"`
	Hit       SearchHit `json:"hit"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

func (s *Store) Search(request SearchRequest) ([]SearchHit, error) {
	prepared, terms, err := prepareSearchRequest(request)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := s.state
	var candidates map[string]bool
	if s.index != nil {
		if err := s.index.sync(snapshot); err == nil {
			candidates, _ = s.index.query(terms, prepared.Limit*8)
		}
	}
	var semanticScores map[string]float64
	if s.semantic != nil {
		s.semantic.sync(snapshot)
		semanticScores = s.semantic.query(prepared.Query, prepared.Limit*4)
	}
	return searchStateHybrid(snapshot, prepared, terms, candidates, semanticScores), nil
}

func (s *Store) CreateSearchMonitor(name string, request SearchRequest, minimumScore int) (SavedSearchMonitor, error) {
	name = strings.TrimSpace(name)
	prepared, terms, err := prepareSearchRequest(request)
	if err != nil {
		return SavedSearchMonitor{}, err
	}
	if name == "" || len([]rune(name)) > 160 || minimumScore < 1 || minimumScore > 100 {
		return SavedSearchMonitor{}, errors.New("monitor name or score threshold is invalid")
	}
	id, err := newIntelID("search-monitor")
	if err != nil {
		return SavedSearchMonitor{}, err
	}
	now := time.Now().UTC()
	monitor := SavedSearchMonitor{ID: id, Name: name, Request: prepared, MinimumScore: minimumScore, Enabled: true, CreatedAt: now, LastRunAt: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, hit := range searchState(s.state, prepared, terms) {
		monitor.SeenHitIDs = appendUniqueBounded(monitor.SeenHitIDs, searchHitKey(hit), 1000)
	}
	s.state.SavedSearches = append(s.state.SavedSearches, monitor)
	s.state.Audits = append(s.state.Audits, AuditEvent{At: now, Action: "search-monitor.created", Actor: "operator", Detail: monitor.ID})
	if err := s.save(); err != nil {
		return SavedSearchMonitor{}, err
	}
	return monitor, nil
}

func (s *Store) RunSearchMonitor(id string) ([]SearchMonitorAlert, error) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.state.SavedSearches {
		if s.state.SavedSearches[index].ID != id {
			continue
		}
		alerts := s.evaluateOneSearchMonitorLocked(&s.state.SavedSearches[index], time.Now().UTC())
		if err := s.save(); err != nil {
			return nil, err
		}
		return alerts, nil
	}
	return nil, errors.New("search monitor not found")
}

func (s *Store) evaluateSearchMonitorsLocked(now time.Time) {
	for index := range s.state.SavedSearches {
		if s.state.SavedSearches[index].Enabled {
			s.evaluateOneSearchMonitorLocked(&s.state.SavedSearches[index], now)
		}
	}
}

func (s *Store) evaluateOneSearchMonitorLocked(monitor *SavedSearchMonitor, now time.Time) []SearchMonitorAlert {
	prepared, terms, err := prepareSearchRequest(monitor.Request)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool, len(monitor.SeenHitIDs))
	for _, id := range monitor.SeenHitIDs {
		seen[id] = true
	}
	alerts := make([]SearchMonitorAlert, 0)
	for _, hit := range searchState(s.state, prepared, terms) {
		key := searchHitKey(hit)
		if hit.Score < monitor.MinimumScore || seen[key] {
			continue
		}
		alertID, idErr := newIntelID("search-alert")
		if idErr != nil {
			continue
		}
		alert := SearchMonitorAlert{ID: alertID, MonitorID: monitor.ID, Hit: hit, CreatedAt: now, Status: "new"}
		s.state.SearchAlerts = append(s.state.SearchAlerts, alert)
		alerts = append(alerts, alert)
		monitor.SeenHitIDs = appendUniqueBounded(monitor.SeenHitIDs, key, 1000)
		s.publish("search-monitor.match", alert.ID, alert)
	}
	monitor.LastRunAt = now
	return alerts
}

func prepareSearchRequest(request SearchRequest) (SearchRequest, []string, error) {
	request.Query = strings.TrimSpace(request.Query)
	if len([]rune(request.Query)) < 2 || len([]rune(request.Query)) > 500 {
		return SearchRequest{}, nil, errors.New("search query must contain between 2 and 500 characters")
	}
	terms := searchTerms(request.Query)
	if len(terms) == 0 || len(terms) > 16 {
		return SearchRequest{}, nil, errors.New("search query has no usable terms or exceeds the term limit")
	}
	if !request.To.IsZero() && !request.From.IsZero() && request.To.Before(request.From) {
		return SearchRequest{}, nil, errors.New("search time interval is invalid")
	}
	if request.Limit == 0 {
		request.Limit = 50
	}
	if request.Limit < 1 || request.Limit > 200 {
		return SearchRequest{}, nil, errors.New("search result limit is invalid")
	}
	request.CaseID = strings.TrimSpace(request.CaseID)
	request.Domains = uniqueSearchFilters(request.Domains)
	request.SourceIDs = uniqueSearchFilters(request.SourceIDs)
	return request, terms, nil
}

func searchState(state StoreState, request SearchRequest, terms []string) []SearchHit {
	return searchStateFiltered(state, request, terms, nil)
}

func searchStateFiltered(state StoreState, request SearchRequest, terms []string, candidates map[string]bool) []SearchHit {
	return searchStateHybrid(state, request, terms, candidates, nil)
}

func searchStateHybrid(state StoreState, request SearchRequest, terms []string, candidates map[string]bool, semanticScores map[string]float64) []SearchHit {
	hits := make([]SearchHit, 0, request.Limit)
	add := func(hit SearchHit, content string) {
		key := searchHitKey(hit)
		semanticScore := semanticScores[key]
		if candidates != nil && !candidates[key] && semanticScore < semanticThreshold {
			return
		}
		if request.CaseID != "" && hit.CaseID != request.CaseID {
			return
		}
		if len(request.Domains) > 0 && !containsFolded(request.Domains, hit.Domain) {
			return
		}
		if len(request.SourceIDs) > 0 && !containsFolded(request.SourceIDs, hit.SourceID) {
			return
		}
		if !request.From.IsZero() && hit.Timestamp.Before(request.From) {
			return
		}
		if !request.To.IsZero() && hit.Timestamp.After(request.To) {
			return
		}
		normalized := strings.ToLower(content)
		score := 0
		for _, term := range terms {
			if strings.Contains(normalized, term) {
				score += 100 / len(terms)
			}
		}
		semanticPercent := int(math.Round(semanticScore * 100))
		if semanticPercent > score {
			score = semanticPercent
		}
		if score == 0 {
			return
		}
		if strings.Contains(strings.ToLower(hit.Title), strings.ToLower(request.Query)) {
			score += 15
		}
		if score > 100 {
			score = 100
		}
		hit.Score = score
		hit.Snippet = safeBoundedSnippet(content, terms, 360)
		hits = append(hits, hit)
	}
	for _, observation := range state.Observations {
		add(SearchHit{RecordType: "observation", RecordID: observation.ID, SourceID: observation.SourceID, Domain: observation.Domain, Title: safeTitle(observation.RawText), Timestamp: observation.ObservedAt}, observation.RawText)
	}
	observationsByDocumentID := make(map[string]Observation, len(state.Observations))
	for _, observation := range state.Observations {
		observationsByDocumentID["doc-"+observation.ID] = observation
	}
	for _, passage := range state.Passages {
		observation := observationsByDocumentID[passage.DocumentID]
		add(SearchHit{
			RecordType: "passage",
			RecordID:   passage.ID,
			SourceID:   observation.SourceID,
			Domain:     observation.Domain,
			Title:      safeTitle(passage.Text),
			Timestamp:  observation.ObservedAt,
		}, passage.Text)
	}
	for _, event := range state.Events {
		add(SearchHit{RecordType: "event", RecordID: event.ID, SourceID: event.SourceID, Domain: event.Domain, Title: event.Title, Timestamp: event.ObservedAt}, event.Title+" "+event.Summary)
	}
	for _, claim := range state.Claims {
		add(SearchHit{RecordType: "claim", RecordID: claim.ID, CaseID: claim.CaseID, SourceID: claim.AssertingSourceID, Title: claim.Statement, Timestamp: claim.CreatedAt}, claim.Subject+" "+claim.Predicate+" "+claim.Object+" "+claim.Statement)
	}
	for _, caseRecord := range state.Cases {
		add(SearchHit{RecordType: "case", RecordID: caseRecord.ID, CaseID: caseRecord.ID, Title: caseRecord.Title, Timestamp: caseRecord.CreatedAt}, caseRecord.Title+" "+caseRecord.Purpose)
		for _, evidence := range caseRecord.Evidence {
			add(SearchHit{RecordType: "evidence", RecordID: evidence.ID, CaseID: caseRecord.ID, SourceID: evidence.SourceID, Title: evidence.Excerpt, Timestamp: evidence.CollectedAt}, evidence.Excerpt)
		}
		for _, entity := range caseRecord.Entities {
			add(SearchHit{RecordType: "entity", RecordID: entity.ID, CaseID: caseRecord.ID, Title: entity.Label}, entity.Label+" "+entity.Kind)
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].Timestamp.After(hits[j].Timestamp)
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > request.Limit {
		hits = hits[:request.Limit]
	}
	return hits
}

func searchTerms(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	return uniqueSearchFilters(fields)
}
func uniqueSearchFilters(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
func containsFolded(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(value, wanted) {
			return true
		}
	}
	return false
}
func boundedSnippet(content string, terms []string, maximum int) string {
	content = strings.Join(strings.Fields(content), " ")
	lower := strings.ToLower(content)
	start := 0
	for _, term := range terms {
		if index := strings.Index(lower, term); index >= 0 {
			start = index - 80
			if start < 0 {
				start = 0
			}
			break
		}
	}
	runes := []rune(content[start:])
	if len(runes) > maximum {
		return string(runes[:maximum]) + "…"
	}
	return string(runes)
}
func searchHitKey(hit SearchHit) string { return hit.RecordType + ":" + hit.RecordID }
func safeBoundedSnippet(content string, terms []string, maximum int) string {
	content = strings.Join(strings.Fields(content), " ")
	lower := strings.ToLower(content)
	startRune := 0
	for _, term := range terms {
		if index := strings.Index(lower, term); index >= 0 {
			startRune = utf8.RuneCountInString(content[:index]) - 80
			if startRune < 0 {
				startRune = 0
			}
			break
		}
	}
	runes := []rune(content)
	if startRune > len(runes) {
		startRune = 0
	}
	end := startRune + maximum
	if end < len(runes) {
		return string(runes[startRune:end]) + "…"
	}
	return string(runes[startRune:])
}
func appendUniqueBounded(values []string, value string, maximum int) []string {
	if containsString(values, value) {
		return values
	}
	values = append(values, value)
	if len(values) > maximum {
		values = values[len(values)-maximum:]
	}
	return values
}
