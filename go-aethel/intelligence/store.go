package intelligence

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var SharedIntelStore *Store

// StoreState represents the persistent state of the intelligence core
type StoreState struct {
	Sources      []Source             `json:"sources"`
	Observations []Observation        `json:"observations"`
	Events       []Event              `json:"events"`
	Assessments  []Assessment         `json:"assessments"`
	Evidence     []Evidence           `json:"evidence"`
	Cases        []Case               `json:"cases"`
	RiskScores   map[string]RiskScore `json:"risk_scores"`
	Alerts       []Alert              `json:"alerts"`
	AlertRules   []AlertRule          `json:"alert_rules"`
	Watchlists   []Watchlist          `json:"watchlists"`
	Audits       []AuditEvent         `json:"audits"`
	AgentActions []AgentAction        `json:"agent_actions"`
	Briefings    []Briefing           `json:"briefings"`
	Personal     PersonalContext      `json:"personal_context,omitempty"`
	Identity     IdentityProfile      `json:"identity,omitempty"`
}

// Store coordinates the data ingestion, geofencing, and risk scoring pipelines
type Store struct {
	mu      sync.RWMutex
	path    string
	state   StoreState
	regions *RegionEngine
	scoring *ScoringEngine
	bus     *EventBus
}

func NewStore(path string, bus *EventBus) *Store {
	s := &Store{
		path:    path,
		bus:     bus,
		regions: GetDefaultRegionEngine(),
		scoring: NewScoringEngine(0.02),
		state: StoreState{
			RiskScores:   make(map[string]RiskScore),
			Alerts:       []Alert{},
			AlertRules:   []AlertRule{},
			Watchlists:   []Watchlist{},
			Audits:       []AuditEvent{},
			Assessments:  []Assessment{},
			Evidence:     []Evidence{},
			AgentActions: []AgentAction{},
			Briefings:    []Briefing{},
			Personal:     PersonalContext{LastUpdated: time.Now().UTC()},
			Identity:     defaultIdentity(),
		},
	}
	s.load()
	return s
}

func (s *Store) load() {
	if raw, err := os.ReadFile(s.path); err == nil {
		var loaded StoreState
		if json.Unmarshal(raw, &loaded) == nil {
			s.state = loaded
			if s.state.RiskScores == nil {
				s.state.RiskScores = make(map[string]RiskScore)
			}
			if s.state.Alerts == nil {
				s.state.Alerts = []Alert{}
			}
			if s.state.AlertRules == nil {
				s.state.AlertRules = []AlertRule{}
			}
			if s.state.Watchlists == nil {
				s.state.Watchlists = []Watchlist{}
			}
			if s.state.Audits == nil {
				s.state.Audits = []AuditEvent{}
			}
			if s.state.Assessments == nil {
				s.state.Assessments = []Assessment{}
			}
			if s.state.Evidence == nil {
				s.state.Evidence = []Evidence{}
			}
			if s.state.AgentActions == nil {
				s.state.AgentActions = []AgentAction{}
			}
			if s.state.Briefings == nil {
				s.state.Briefings = []Briefing{}
			}
			if s.state.Identity.Name == "" {
				s.state.Identity = defaultIdentity()
			}
		}
	}
}

func (s *Store) save() error {
	_ = os.MkdirAll(filepath.Dir(s.path), 0700)
	raw, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0600)
}

// publish emits a typed bus message if a bus is configured (non-blocking).
func (s *Store) publish(eventType, subjectID string, payload any) {
	if s.bus == nil {
		return
	}
	s.bus.PublishEvent(eventType, subjectID, payload)
}

func defaultIdentity() IdentityProfile {
	return IdentityProfile{
		Name:            "AETHEL",
		Version:         "intelligence-kernel-v1",
		LastUpdated:     time.Now().UTC(),
		CapabilityNotes: []string{"rss-ingest", "region-match", "risk-scoring", "explain-score", "briefings", "alerts", "personal-impact", "proactive-loop", "evidence-promote", "case-graph", "reid-dual-control", "corroboration", "agent-action-audit", "connectors-registry", "connector-fetch", "run-engine-audit"},
		DiagnosticsNote: "local-first; no claimed consciousness",
	}
}

func appendPending(ids []string, id string, max int) []string {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	ids = append(ids, id)
	if max > 0 && len(ids) > max {
		ids = ids[len(ids)-max:]
	}
	return ids
}

// GetSnapshot returns a read-only copy of the state
func (s *Store) GetSnapshot() StoreState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// IngestObservation implements the first vertical milestone pipeline:
// Observation (raw, from source) -> Region Detection -> Event Classification (inference layer) ->
// Evidence ref -> Risk recompute (with delta) -> Alert (if significant change) -> bus events + audit.
// Strict separation: Observation is raw/unchanged; Event carries classification; Risk/Alert are derived Assessments.
func (s *Store) IngestObservation(obs Observation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Record raw observation (never mutate — Observation layer only)
	s.state.Observations = append(s.state.Observations, obs)
	s.publish("observation.created", obs.ID, obs)

	// Content hash for provenance / multi-source corroboration (U6)
	contentHash := obs.ContentHash
	if contentHash == "" {
		contentHash = shortContentHash(obs.RawText)
	}
	obs.ContentHash = contentHash
	// Update stored observation hash (already appended; fix last entry)
	if n := len(s.state.Observations); n > 0 && s.state.Observations[n-1].ID == obs.ID {
		s.state.Observations[n-1].ContentHash = contentHash
	}

	// Register / upsert Source with fuller provenance fields
	src := Source{
		ID:                 obs.SourceID,
		Name:               obs.SourceID,
		SourceType:         "rss",
		FetchedAt:          time.Now().UTC(),
		TrustTier:          2, // default community until configured
		AvailabilityStatus: "ok",
		ContentHash:        contentHash,
		ParserVersion:      "ingest-v1",
		Freshness:          100,
		PermissionStatus:   "local-approved",
	}
	found := false
	for i := range s.state.Sources {
		if s.state.Sources[i].ID == src.ID {
			s.state.Sources[i].FetchedAt = src.FetchedAt
			s.state.Sources[i].ContentHash = contentHash
			s.state.Sources[i].Freshness = 100
			s.state.Sources[i].AvailabilityStatus = "ok"
			found = true
			break
		}
	}
	if !found {
		s.state.Sources = append(s.state.Sources, src)
		s.publish("source.fetched", src.ID, src)
	}

	// Evidence reference (raw level — not sealed case evidence)
	evid := Evidence{
		ID:               "evid-" + obs.ID,
		SourceID:         obs.SourceID,
		Excerpt:          safeTitle(obs.RawText),
		SHA256:           contentHash,
		CollectedAt:      time.Now().UTC(),
		Sealed:           false,
		ValidationStatus: "pending",
		ChainOfCustodyID: "coc-" + obs.ID,
	}
	s.state.Evidence = append(s.state.Evidence, evid)
	s.publish("evidence.sealed", evid.ID, evid)

	// 1. Region Detection (deterministic polygon)
	matchedRegions := s.regions.MatchPoint(obs.Latitude, obs.Longitude)

	// 2. Event Classification + dedup (doppelte Meldungen zusammenführen per §19)
	// Deterministic similarity: title words overlap + geo proximity + time window
	domain := classifyDomain(obs.RawText, obs.Domain)
	severity := classifySeverity(obs.RawText)
	baseConf := 65
	if obs.Domain != "" && obs.Domain != "general" {
		baseConf = 75
	}

	// Factor source TrustTier for Quellenqualität (§19)
	trustMultiplier := 1.0
	for _, src := range s.state.Sources {
		if src.ID == obs.SourceID {
			switch src.TrustTier {
			case 1:
				trustMultiplier = 1.2
			case 3:
				trustMultiplier = 0.85
			}
			break
		}
	}
	conf := int(float64(baseConf) * trustMultiplier)
	if conf > 95 {
		conf = 95
	}

	candidate := Event{
		ID:         "ev_" + obs.ID,
		Title:      safeTitle(obs.RawText),
		Summary:    obs.RawText,
		Domain:     domain,
		Latitude:   obs.Latitude,
		Longitude:  obs.Longitude,
		Severity:   severity,
		Confidence: conf,
		ObservedAt: obs.ObservedAt,
	}

	// Dedup: skip if highly similar recent event exists (title overlap + geo < ~50km + < 6h)
	isDup := false
	dupEventID := ""
	for _, existing := range s.state.Events {
		dt := candidate.ObservedAt.Sub(existing.ObservedAt).Hours()
		if dt < 0 {
			dt = -dt
		}
		if dt > 6 {
			continue
		}
		if titlesAreSimilar(candidate.Title, existing.Title) {
			dist := haversineDistance(candidate.Latitude, candidate.Longitude, existing.Latitude, existing.Longitude)
			if dist < 50.0 {
				isDup = true
				dupEventID = existing.ID
				break
			}
		}
	}

	if !isDup {
		s.state.Events = append(s.state.Events, candidate)
		s.publish("event.created", candidate.ID, candidate)
	}

	// Multi-source corroboration (U6): distinct sources on similar story (≥2) → Assessment corroborated
	// (still NOT operator-verified). Works even when second obs is geo-deduped against first event.
	assessStatus := "unverified"
	srcSet := map[string]bool{obs.SourceID: true}
	targetTitle := candidate.Title
	if isDup && dupEventID != "" {
		for _, existing := range s.state.Events {
			if existing.ID == dupEventID {
				targetTitle = existing.Title
				break
			}
		}
	}
	for _, o := range s.state.Observations {
		if o.ID == obs.ID {
			continue
		}
		dt := obs.ObservedAt.Sub(o.ObservedAt).Hours()
		if dt < 0 {
			dt = -dt
		}
		if dt > 24 {
			continue
		}
		if titlesAreSimilar(targetTitle, safeTitle(o.RawText)) || titlesAreSimilar(obs.RawText, o.RawText) {
			srcSet[o.SourceID] = true
		}
	}
	if len(srcSet) >= 2 {
		assessStatus = "corroborated"
		conf += 5
		if conf > 95 {
			conf = 95
		}
		candidate.Confidence = conf
		if !isDup && len(s.state.Events) > 0 {
			last := len(s.state.Events) - 1
			if s.state.Events[last].ID == candidate.ID {
				s.state.Events[last].Confidence = conf
			}
		}
		// Upgrade existing assessment on the original event when this is a corroborating dup
		if isDup && dupEventID != "" {
			upgraded := false
			for i := range s.state.Assessments {
				if s.state.Assessments[i].ID == "assess-"+dupEventID || strings.Contains(s.state.Assessments[i].ID, dupEventID) {
					s.state.Assessments[i].Status = "corroborated"
					s.state.Assessments[i].Confidence = conf
					s.state.Assessments[i].EvidenceIDs = append(s.state.Assessments[i].EvidenceIDs, evid.ID)
					s.state.Assessments[i].Statement = fmt.Sprintf("Multi-source corroboration (sources=%d) for %s", len(srcSet), dupEventID)
					s.publish("assessment.updated", s.state.Assessments[i].ID, s.state.Assessments[i])
					upgraded = true
					break
				}
			}
			if !upgraded {
				// Create a corroboration assessment linked to the original event
				s.state.Assessments = append(s.state.Assessments, Assessment{
					ID: "assess-corr-" + obs.ID, Statement: fmt.Sprintf("Corroborating observation for %s (sources=%d)", dupEventID, len(srcSet)),
					Confidence: conf, EvidenceIDs: []string{evid.ID}, GeneratedBy: "rule-engine",
					CreatedAt: time.Now().UTC(), Status: "corroborated",
				})
			}
		}
	}

	// Assessment is inference, never treated as verified fact without operator/evidence review.
	// "corroborated" = multi-source agreement only — still not operator-verified.
	if !isDup {
		assess := Assessment{
			ID:          "assess-" + candidate.ID,
			Statement:   fmt.Sprintf("Classified as domain=%s severity=%s from raw observation (sources=%d)", domain, severity, len(srcSet)),
			Confidence:  conf,
			EvidenceIDs: []string{evid.ID},
			GeneratedBy: "rule-engine",
			CreatedAt:   time.Now().UTC(),
			Status:      assessStatus,
		}
		s.state.Assessments = append(s.state.Assessments, assess)
		s.publish("assessment.updated", assess.ID, assess)
	}

	// 3. Recalculate Risk for matched regions using full multi-dim scoring & publish deltas
	alertCreated := false
	for _, r := range matchedRegions {
		regionEvents := []Event{}
		for _, e := range s.state.Events {
			p := Point{Lon: e.Longitude, Lat: e.Latitude}
			for _, poly := range r.Polygons {
				if IsPointInPolygon(p, poly) {
					regionEvents = append(regionEvents, e)
					break
				}
			}
		}

		oldScore := s.state.RiskScores[r.ID]
		newScore := s.scoring.ComputeRiskScore(regionEvents, oldScore.LastUpdated)

		// Fill remaining spec dimensions with reasonable defaults + decay influence
		if newScore.DataFreshness == 0 {
			newScore.DataFreshness = 80.0
		}
		if newScore.InformationReliability == 0 {
			newScore.InformationReliability = float64(conf)
		}
		if newScore.EnergyRisk == 0 {
			newScore.EnergyRisk = newScore.InfrastructureRisk * 0.6
		}
		if newScore.SupplyChainRisk == 0 {
			newScore.SupplyChainRisk = newScore.EconomicRisk * 0.7
		}
		if newScore.ClimateRisk == 0 {
			newScore.ClimateRisk = 10.0 // low default unless signal
		}
		if newScore.PublicSafetyRisk == 0 {
			newScore.PublicSafetyRisk = newScore.ConflictRisk * 0.5
		}
		if newScore.FinancialRisk == 0 {
			newScore.FinancialRisk = newScore.EconomicRisk * 0.8
		}
		newScore.Confidence = conf
		if len(newScore.Trend) == 0 {
			if newScore.OverallRisk > oldScore.OverallRisk+5 {
				newScore.Trend = "up"
			} else if newScore.OverallRisk < oldScore.OverallRisk-5 {
				newScore.Trend = "down"
			} else {
				newScore.Trend = "stable"
			}
		}
		if len(newScore.MissingData) == 0 {
			newScore.MissingData = []string{"No primary sources for all dimensions"}
		}

		s.state.RiskScores[r.ID] = newScore

		delta := newScore.OverallRisk - oldScore.OverallRisk
		s.publish("risk.changed", r.ID, map[string]any{
			"region_id":   r.ID,
			"old_score":   oldScore.OverallRisk,
			"new_score":   newScore.OverallRisk,
			"delta":       delta,
			"drivers":     newScore.PrimaryDrivers,
			"last_update": newScore.LastUpdated,
		})

		// 4. Alert on significant risk delta — spam rules via AlertManager
		if (delta > 8.0 || newScore.OverallRisk > 55) && !alertCreated {
			now := time.Now().UTC()
			eventID := candidate.ID
			alertCand := Alert{
				ID:              "alert-" + eventID + "-" + r.ID,
				Severity:        severity,
				Confidence:      conf,
				Region:          r.ID,
				AffectedDomains: []string{domain},
				EvidenceIDs:     []string{evid.ID},
				Reason:          fmt.Sprintf("Risk delta %.1f for %s (now %.1f%%). Primary: %v", delta, r.Name, newScore.OverallRisk, newScore.PrimaryDrivers),
				EscalationState: "new",
			}
			srcSet := map[string]bool{obs.SourceID: true}
			mgr := NewAlertManager()
			if created := mgr.CreateAlert(alertCand, s.state.Alerts, len(srcSet)); created != nil {
				al := *created
				if al.Reason == "" {
					al.Reason = alertCand.Reason
				}
				s.state.Alerts = append(s.state.Alerts, al)
				s.publish("alert.created", al.ID, al)
				alertCreated = true
				s.state.Identity.LastWarnedAt = now
				s.state.Identity.PendingAlertIDs = appendPending(s.state.Identity.PendingAlertIDs, al.ID, 20)
				s.state.Audits = append(s.state.Audits, AuditEvent{
					At: now, Action: "alert.created", Actor: "system",
					Detail: "Auto-alert from IngestObservation delta for " + r.ID + " event=" + eventID,
				})
			}
		}
	}

	// Always append basic audit for ingestion
	s.state.Audits = append(s.state.Audits, AuditEvent{
		At: time.Now().UTC(), Action: "observation.ingested", Actor: "ingest",
		Detail: fmt.Sprintf("Obs %s -> Event %s , regions=%d", obs.ID, candidate.ID, len(matchedRegions)),
	})

	_ = s.save()
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// classifyDomain applies lightweight deterministic rules to tag raw obs (inference layer, not fact)
func classifyDomain(text, fallback string) string {
	t := strings.ToLower(text)
	switch {
	case strings.Contains(t, "cyber") || strings.Contains(t, "hack") || strings.Contains(t, "ransomware") || strings.Contains(t, "breach"):
		return "cyber"
	case strings.Contains(t, "conflict") || strings.Contains(t, "war") || strings.Contains(t, "strike") || strings.Contains(t, "troop"):
		return "humanitarian"
	case strings.Contains(t, "market") || strings.Contains(t, "stock") || strings.Contains(t, "oil") || strings.Contains(t, "gas") || strings.Contains(t, "price"):
		return "economic"
	case strings.Contains(t, "power") || strings.Contains(t, "grid") || strings.Contains(t, "supply") || strings.Contains(t, "port") || strings.Contains(t, "transport"):
		return "infrastructure"
	case strings.Contains(t, "earthquake") || strings.Contains(t, "[earthquake]") || strings.Contains(t, "volcano") || strings.Contains(t, "[volcano"):
		return "geo"
	case strings.Contains(t, "climate") || strings.Contains(t, "flood") || strings.Contains(t, "storm"):
		return "general"
	}
	if fallback != "" && fallback != "general" {
		return fallback
	}
	return "geo"
}

func classifySeverity(text string) string {
	t := strings.ToLower(text)
	if strings.Contains(t, "major") || strings.Contains(t, "critical") || strings.Contains(t, "severe") || strings.Contains(t, "high alert") || strings.Contains(t, "escalat") {
		return "high"
	}
	if strings.Contains(t, "minor") || strings.Contains(t, "small") {
		return "low"
	}
	return "medium"
}

// GenerateBrief is a basic report generator using the shared model data (for vertical + build out)
func (s *Store) GenerateBrief(regionID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var b strings.Builder
	b.WriteString("## Basic Brief for " + regionID + "\n\n")
	count := 0
	for _, ev := range s.state.Events {
		if ev.Latitude != 0 || ev.Longitude != 0 {
			b.WriteString("- " + ev.Title + ": " + ev.Summary + " (domain: " + ev.Domain + ")\n")
			count++
			if count > 5 {
				break
			}
		}
	}
	if count == 0 {
		b.WriteString("No recent geo events.\n")
	}
	return b.String()
}

// GenerateReport implements structured Lagebericht per spec (Executive Summary, Changes, Risks per dimension.
// Report types containing "personal" / "impact" route to PersonalImpact().
// Conflicts, Infra/Energy/Cyber, Watchlist impact, Uncertainties, Sources/Evidence, Recommendations as clearly labeled).
// Clearly separates:
// - Confirmed raw observations (raw text from source)
// - Derived inferences / classifications (Events, RiskScores with drivers)
// - Uncertainties explicitly listed.
func (s *Store) GenerateReport(reportType string) string {
	rt := strings.ToLower(strings.TrimSpace(reportType))
	if strings.Contains(rt, "personal") || strings.Contains(rt, "impact") {
		return s.PersonalImpact()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var b strings.Builder
	now := time.Now().UTC()
	b.WriteString(fmt.Sprintf("# AETHEL %s\n\n", reportType))
	b.WriteString(fmt.Sprintf("Generated: %s (UTC)\n", now.Format(time.RFC3339)))
	b.WriteString("**SOVEREIGN LOCAL REPORT — NO EXTERNAL TRANSMISSION**\n\n")

	b.WriteString("## 1. Executive Summary\n")
	b.WriteString(fmt.Sprintf("Observations (raw): %d | Events (classified): %d | Regions tracked: %d | Active Alerts: %d\n\n",
		len(s.state.Observations), len(s.state.Events), len(s.state.RiskScores), len(s.state.Alerts)))

	// Layer separation (always visible in operator output)
	b.WriteString("## 1b. Layer separation (Observation vs Inference vs Verified)\n")
	b.WriteString("- **RAW OBSERVATIONS** (unreviewed claims captured from sources):\n")
	obsShown := 0
	for i := len(s.state.Observations) - 1; i >= 0 && obsShown < 5; i-- {
		o := s.state.Observations[i]
		b.WriteString(fmt.Sprintf("  - [%s] src=%s @ %.2f,%.2f — %s\n", o.ID, o.SourceID, o.Latitude, o.Longitude, safeTitle(o.RawText)))
		obsShown++
	}
	if obsShown == 0 {
		b.WriteString("  - (none)\n")
	}
	b.WriteString("- **INFERENCES / ASSESSMENTS** (rule-engine classification — status never auto-verified):\n")
	for i, a := range s.state.Assessments {
		if i >= 5 {
			break
		}
		b.WriteString(fmt.Sprintf("  - [%s] status=%s conf=%d%% — %s (evidence: %v)\n", a.ID, a.Status, a.Confidence, a.Statement, a.EvidenceIDs))
	}
	if len(s.state.Assessments) == 0 {
		b.WriteString("  - (none)\n")
	}
	b.WriteString("- **VERIFIED FINDINGS**: none unless operator/evidence review sets Assessment.Status=verified.\n\n")

	// 2. Wichtigste Veränderungen
	b.WriteString("## 2. Wichtigste Veränderungen seit letztem Bericht (explicit 24h window + source quality)\n")
	recentCount := 0
	olderCount := 0
	for _, ev := range s.state.Events {
		dt := now.Sub(ev.ObservedAt).Hours()
		if dt < 24 && dt >= 0 {
			recentCount++
			b.WriteString(fmt.Sprintf("- [NEW %s] %s (domain=%s, conf=%d%%, INFERENCE — unverified)\n", ev.ObservedAt.Format(time.RFC3339), safeTitle(ev.Summary), ev.Domain, ev.Confidence))
		} else if dt >= 24 {
			olderCount++
		}
	}
	b.WriteString(fmt.Sprintf("Summary: %d new classified events in last 24h, %d older for baseline.\n", recentCount, olderCount))
	if recentCount == 0 {
		b.WriteString("No significant changes in the last 24h window.\n")
	}
	b.WriteString("\n")

	// 3 Risk sections
	b.WriteString("## 3. Globale / Regionale Risikolage (multi-dimensional, erklärbar)\n")
	for id, rs := range s.state.RiskScores {
		b.WriteString(fmt.Sprintf("**%s**: Overall=%.1f%% (trend=%s, conf=%d%%)\n", id, rs.OverallRisk, rs.Trend, rs.Confidence))
		b.WriteString(fmt.Sprintf("  Drivers: %v\n", rs.PrimaryDrivers))
		b.WriteString(fmt.Sprintf("  GeoPol=%.0f Conflict=%.0f Cyber=%.0f Infra=%.0f Econ=%.0f Energy=%.0f Supply=%.0f\n\n",
			rs.GeopoliticalRisk, rs.ConflictRisk, rs.CyberRisk, rs.InfrastructureRisk, rs.EconomicRisk, rs.EnergyRisk, rs.SupplyChainRisk))
	}
	if len(s.state.RiskScores) == 0 {
		b.WriteString("No regional risk scores computed yet.\n")
	}

	b.WriteString("## 4. Konflikte und Eskalationsindikatoren\n")
	conflictN := 0
	for _, ev := range s.state.Events {
		if ev.Domain == "humanitarian" || ev.Severity == "high" {
			b.WriteString(fmt.Sprintf("- %s (sev=%s) @ %.1f,%.1f [INFERENCE]\n", safeTitle(ev.Title), ev.Severity, ev.Latitude, ev.Longitude))
			conflictN++
		}
	}
	if conflictN == 0 {
		b.WriteString("No high-severity/conflict-domain events in store.\n")
	}

	b.WriteString("\n## 5–10. Domain clusters (finance, energy/infra, cyber, supply, climate)\n")
	b.WriteString("(Derived from domain-tagged Events and dimension scores — all INFERENCE unless verified.)\n")
	for _, domain := range []string{"economic", "infrastructure", "cyber", "geo"} {
		n := 0
		for _, ev := range s.state.Events {
			if ev.Domain == domain {
				n++
			}
		}
		b.WriteString(fmt.Sprintf("- domain=%s event_count=%d\n", domain, n))
	}
	b.WriteString("\n")

	b.WriteString("## 11. Auswirkungen auf Operator-Watchlists (Personal Context + World State correlation)\n")
	impactCount := 0
	if len(s.state.Watchlists) > 0 {
		for _, wl := range s.state.Watchlists {
			b.WriteString(fmt.Sprintf("- Watchlist '%s' (regions: %v, keywords: %v)\n", wl.Name, wl.RegionIDs, wl.Keywords))
			for _, rid := range wl.RegionIDs {
				if rs, ok := s.state.RiskScores[rid]; ok {
					impactCount++
					b.WriteString(fmt.Sprintf("  → Current %s OverallRisk=%.1f%% (trend: %s). Primary drivers: %v.\n", rid, rs.OverallRisk, rs.Trend, rs.PrimaryDrivers))
					if rs.Trend == "up" && rs.OverallRisk > 40 {
						b.WriteString("     **HANDLUNGSBEDARF?** Rising risk on watched region. Review Evidence (recommendation, not fact).\n")
					}
				}
			}
		}
	}
	if impactCount == 0 {
		b.WriteString("No active watchlist-region overlaps with current RiskScores, or no Watchlists defined.\nUse intelligence_create_watchlist to link personal interests to regions.\n")
	}
	if s.state.Personal.OperatorID != "" {
		b.WriteString(fmt.Sprintf("Personal context last updated: %s. Risk tolerance: %s\n", s.state.Personal.LastUpdated.Format(time.RFC3339), s.state.Personal.RiskTolerance))
	}

	b.WriteString("\n## 12. Uncertainties / Unsicherheiten und Datenlücken\n")
	b.WriteString("- Raw Observations are unverified by definition (Assessment.Status starts as unverified).\n")
	b.WriteString("- Scores use defaults for missing signals; see MissingData per RiskScore.\n")
	b.WriteString("- No multi-source corroboration or operator verification yet.\n")
	b.WriteString("- Classifications are rule-engine inference, not confirmed facts.\n\n")

	b.WriteString("## 13. Quellen und Evidence\n")
	for i, src := range s.state.Sources {
		if i > 4 {
			break
		}
		b.WriteString(fmt.Sprintf("- %s (type=%s, trust=%d, fetched=%s)\n", src.ID, src.SourceType, src.TrustTier, src.FetchedAt.Format(time.RFC3339)))
	}
	for i, e := range s.state.Evidence {
		if i > 4 {
			break
		}
		b.WriteString(fmt.Sprintf("- Evidence %s src=%s status=%s coc=%s — %s\n", e.ID, e.SourceID, e.ValidationStatus, e.ChainOfCustodyID, safeTitle(e.Excerpt)))
	}
	b.WriteString(fmt.Sprintf("Raw Observations: %d | Events (classified): %d | Evidence: %d\n\n", len(s.state.Observations), len(s.state.Events), len(s.state.Evidence)))

	b.WriteString("## 14. Empfohlene Beobachtungspunkte / Handlungsempfehlungen\n")
	b.WriteString("**Diese sind Empfehlungen, keine Fakten oder Anweisungen.**\n")
	b.WriteString("- Überwachen Regionen mit steigenden Risiko-Deltas oder Watchlist overlaps.\n")
	b.WriteString("- Prüfen Sie Primärquellen der Events mit hoher Confidence.\n")
	b.WriteString("- Erstellen Sie Alert Rules oder Watchlists für persönliche Projekte/Infrastruktur.\n\n")

	b.WriteString("---\n**Provenance note**: Alle Werte stammen aus IngestObservation (RSS→Observation raw → region+event) oder manuellem Operator-Input. Keine LLM-Erfindungen. Scores deterministisch via ScoringEngine (0.6*max + 0.4*avg, freshness decay). Observation (raw) strikt getrennt von Event/Assessment/Risk (Inference).\n")

	if s.bus != nil {
		s.bus.PublishEvent("briefing.generated", reportType, map[string]any{"sections": 14, "observations": len(s.state.Observations)})
	}
	return b.String()
}

// shortContentHash is a non-crypto-strength fingerprint for local provenance (hex prefix of FNV-like mix).
func shortContentHash(raw string) string {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(raw); i++ {
		h ^= uint64(raw[i])
		h *= 1099511628211
	}
	return fmt.Sprintf("%016x", h)
}

// normalizeRegionKey maps common aliases onto RiskScore keys used by the region pack.
func normalizeRegionKey(rid string) string {
	u := strings.ToUpper(strings.TrimSpace(rid))
	switch u {
	case "DE", "DEU", "BRD", "DEUTSCHLAND":
		return "GERMANY"
	case "FR", "FRA":
		return "FRANCE"
	case "UA", "UKR":
		return "UKRAINE"
	case "US", "USA", "UNITED STATES":
		return "USA"
	case "GB", "UK", "GBR", "UNITED KINGDOM":
		return "UK"
	case "PL", "POL", "POLAND":
		return "POLAND"
	default:
		return u
	}
}

func safeTitle(s string) string {
	if len(s) > 100 {
		return s[:100]
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// titlesAreSimilar and haversineDistance for deterministic dedup (no external deps)
func titlesAreSimilar(t1, t2 string) bool {
	w1 := strings.Fields(strings.ToLower(t1))
	w2 := strings.Fields(strings.ToLower(t2))
	m1 := make(map[string]bool)
	for _, w := range w1 {
		if len(w) > 3 {
			m1[w] = true
		}
	}
	overlap := 0
	for _, w := range w2 {
		if len(w) > 3 && m1[w] {
			overlap++
		}
	}
	return overlap >= 3
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

// --- Additional accessors & explainers for handler / skills / UI (exclusive use of shared model) ---

func (s *Store) GetRiskScores() map[string]RiskScore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]RiskScore, len(s.state.RiskScores))
	for k, v := range s.state.RiskScores {
		cp[k] = v
	}
	return cp
}

func (s *Store) GetAlerts() []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]Alert, len(s.state.Alerts))
	copy(cp, s.state.Alerts)
	return cp
}

func (s *Store) GetAlert(id string) (Alert, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id = strings.TrimSpace(id)
	for _, alert := range s.state.Alerts {
		if alert.ID == id {
			return alert, true
		}
	}
	return Alert{}, false
}

func (s *Store) SetAlertAIAssessment(id string, assessment AlertAIAssessment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	assessment.Status = "unverified"
	assessment.EvaluatedAt = assessment.EvaluatedAt.UTC()
	if assessment.EvaluatedAt.IsZero() {
		assessment.EvaluatedAt = time.Now().UTC()
	}
	for index := range s.state.Alerts {
		if s.state.Alerts[index].ID != id {
			continue
		}
		copyAssessment := assessment
		s.state.Alerts[index].AIAssessment = &copyAssessment
		s.state.Audits = append(s.state.Audits, AuditEvent{
			At: time.Now().UTC(), Action: "alert.ai_assessed", Actor: assessment.ModelID, Detail: id,
		})
		return s.save()
	}
	return fmt.Errorf("alert not found")
}

// AcknowledgeAlert marks an alert as seen by the operator (prevents alert spam re-surface).
func (s *Store) AcknowledgeAlert(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	for i := range s.state.Alerts {
		if s.state.Alerts[i].ID == id {
			s.state.Alerts[i].Acknowledged = true
			s.state.Audits = append(s.state.Audits, AuditEvent{
				At: time.Now().UTC(), Action: "alert.acknowledged", Actor: "operator", Detail: id,
			})
			_ = s.save()
			return nil
		}
	}
	return fmt.Errorf("alert not found")
}

func (s *Store) GetSources() []Source {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]Source, len(s.state.Sources))
	copy(cp, s.state.Sources)
	return cp
}

func (s *Store) GetAudits(limit int) []AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.state.Audits) {
		limit = len(s.state.Audits)
	}
	cp := make([]AuditEvent, limit)
	copy(cp, s.state.Audits[len(s.state.Audits)-limit:])
	return cp
}

// ExplainScore returns a human-readable why-this-score string using PrimaryDrivers + data.
// Never labels raw observations as verified facts.
func (s *Store) ExplainScore(regionID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rs, ok := s.state.RiskScores[regionID]
	if !ok {
		return "No risk score computed for region " + regionID + " yet. Ingest observations with geo in the region first."
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## Why is this score this high?\nRegion %s OverallRisk=%.1f%% (trend %s, conf %d%%)\n", regionID, rs.OverallRisk, rs.Trend, rs.Confidence))
	b.WriteString(fmt.Sprintf("Primary drivers: %v\n", rs.PrimaryDrivers))
	if len(rs.PrimaryDrivers) == 0 {
		b.WriteString("Primary drivers: (none — no high-severity fresh signals)\n")
	}
	b.WriteString("Contributing dimensions (selected): ")
	b.WriteString(fmt.Sprintf("Geopol=%.0f Conflict=%.0f Cyber=%.0f Infra=%.0f Econ=%.0f Energy=%.0f\n",
		rs.GeopoliticalRisk, rs.ConflictRisk, rs.CyberRisk, rs.InfrastructureRisk, rs.EconomicRisk, rs.EnergyRisk))
	b.WriteString(fmt.Sprintf("Data freshness ~%.0f%%. MissingData: %v\n", rs.DataFreshness, rs.MissingData))
	b.WriteString("Formula: OverallRisk = 0.6*max(dims) + 0.4*avg(dims); per-event weight * severity * e^(-λ·hours) * confidence.\n")
	b.WriteString("Layer note: score is DERIVED (inference). Raw observations feeding it remain UNVERIFIED.\n")
	b.WriteString("Evidence: see EvidenceIDs on related Alerts/Assessments; source trust tier factors confidence.\n")
	return b.String()
}

// QueryRegionSecurity answers the §19 operator query style (e.g. security developments in Germany last 24h)
// using ONLY the shared store: sources, observations, classified events, evidence, risk explain, uncertainties, change window.
func (s *Store) QueryRegionSecurity(regionID string, hours float64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if hours <= 0 {
		hours = 24
	}
	now := time.Now().UTC()
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Security developments — %s (last %.0fh)\n\n", regionID, hours))
	b.WriteString("**Unified model only** — no parallel chat truth.\n\n")

	// Select region events via polygon match on shared Events
	var regionEvents []Event
	var priorEvents []Event
	for _, ev := range s.state.Events {
		matched := s.regions.MatchPoint(ev.Latitude, ev.Longitude)
		inRegion := false
		for _, r := range matched {
			if strings.EqualFold(r.ID, regionID) || strings.Contains(strings.ToUpper(r.ID), strings.ToUpper(regionID)) {
				inRegion = true
				break
			}
		}
		if !inRegion && strings.EqualFold(regionID, "GERMANY") {
			// also accept BERLIN as Germany context
			for _, r := range matched {
				if r.ID == "BERLIN" || r.ID == "GERMANY" {
					inRegion = true
					break
				}
			}
		}
		if !inRegion {
			continue
		}
		dt := now.Sub(ev.ObservedAt).Hours()
		if dt < 0 {
			dt = 0
		}
		if dt <= hours {
			regionEvents = append(regionEvents, ev)
		} else if dt <= hours*2 {
			priorEvents = append(priorEvents, ev)
		}
	}

	b.WriteString("## Sources selected\n")
	for i, src := range s.state.Sources {
		if i > 6 {
			break
		}
		b.WriteString(fmt.Sprintf("- %s type=%s trust_tier=%d status=%s\n", src.ID, src.SourceType, src.TrustTier, src.AvailabilityStatus))
	}
	if len(s.state.Sources) == 0 {
		b.WriteString("- (no sources registered yet)\n")
	}

	b.WriteString("\n## RAW OBSERVATIONS (unreviewed source claims)\n")
	rawN := 0
	for _, o := range s.state.Observations {
		matched := s.regions.MatchPoint(o.Latitude, o.Longitude)
		ok := false
		for _, r := range matched {
			if strings.EqualFold(r.ID, regionID) || r.ID == "GERMANY" || r.ID == "BERLIN" {
				if strings.EqualFold(regionID, "GERMANY") || strings.EqualFold(r.ID, regionID) {
					ok = true
					break
				}
			}
		}
		if !ok {
			continue
		}
		dt := now.Sub(o.ObservedAt).Hours()
		if dt < 0 {
			dt = 0
		}
		if dt > hours {
			continue
		}
		b.WriteString(fmt.Sprintf("- [%s] src=%s — %s\n", o.ID, o.SourceID, safeTitle(o.RawText)))
		rawN++
	}
	if rawN == 0 {
		b.WriteString("- No raw observations in window for this region.\n")
	}

	b.WriteString("\n## CLASSIFIED EVENTS (INFERENCE — Assessment status=unverified unless reviewed)\n")
	if len(regionEvents) == 0 {
		b.WriteString("- No classified events in window.\n")
	}
	for _, ev := range regionEvents {
		b.WriteString(fmt.Sprintf("- %s | domain=%s sev=%s conf=%d%% | %s\n", ev.ID, ev.Domain, ev.Severity, ev.Confidence, safeTitle(ev.Title)))
	}

	b.WriteString("\n## Change vs prior window\n")
	b.WriteString(fmt.Sprintf("- Events in last %.0fh: %d\n", hours, len(regionEvents)))
	b.WriteString(fmt.Sprintf("- Events in prior %.0fh window: %d\n", hours, len(priorEvents)))
	if len(regionEvents) > len(priorEvents) {
		b.WriteString("- Delta: more activity than prior window (increase).\n")
	} else if len(regionEvents) < len(priorEvents) {
		b.WriteString("- Delta: less activity than prior window (decrease).\n")
	} else {
		b.WriteString("- Delta: similar volume to prior window.\n")
	}

	b.WriteString("\n## Evidence references\n")
	evN := 0
	for _, e := range s.state.Evidence {
		if evN >= 8 {
			break
		}
		b.WriteString(fmt.Sprintf("- %s status=%s src=%s — %s\n", e.ID, e.ValidationStatus, e.SourceID, safeTitle(e.Excerpt)))
		evN++
	}
	if evN == 0 {
		b.WriteString("- No evidence sealed yet.\n")
	}

	b.WriteString("\n## Regional risk (explainable)\n")
	if rs, ok := s.state.RiskScores[regionID]; ok {
		b.WriteString(fmt.Sprintf("OverallRisk=%.1f%% trend=%s conf=%d%% drivers=%v missing=%v freshness=%.0f%%\n",
			rs.OverallRisk, rs.Trend, rs.Confidence, rs.PrimaryDrivers, rs.MissingData, rs.DataFreshness))
	} else if rs, ok := s.state.RiskScores["GERMANY"]; ok && strings.EqualFold(regionID, "GERMANY") {
		b.WriteString(fmt.Sprintf("OverallRisk=%.1f%% trend=%s conf=%d%% drivers=%v missing=%v freshness=%.0f%%\n",
			rs.OverallRisk, rs.Trend, rs.Confidence, rs.PrimaryDrivers, rs.MissingData, rs.DataFreshness))
	} else {
		b.WriteString("No RiskScore for region yet.\n")
	}

	b.WriteString("\n## Uncertainties\n")
	b.WriteString("- Observations are raw; Assessments start as unverified.\n")
	b.WriteString("- Single-source items lack corroboration.\n")
	b.WriteString("- Missing multi-domain coverage may understate or overstate risk.\n")
	b.WriteString("- Map pins and chat answers use this same store snapshot.\n")
	return b.String()
}

// AddWatchlist stores an operator watchlist for personal↔world correlation.
func (s *Store) AddWatchlist(name string, regionIDs []string, keywords []string) Watchlist {
	s.mu.Lock()
	defer s.mu.Unlock()
	wl := Watchlist{ID: "wl-" + fmt.Sprintf("%d", time.Now().UnixNano()), Name: name, RegionIDs: regionIDs, Keywords: keywords}
	s.state.Watchlists = append(s.state.Watchlists, wl)
	s.bus.PublishEvent("watchlist.created", wl.ID, wl)
	_ = s.save()
	return wl
}

// AddAlertRule persists an operator alert rule on the shared model (no fake success).
func (s *Store) AddAlertRule(regionID, minSeverity string, minOverall float64) AlertRule {
	s.mu.Lock()
	defer s.mu.Unlock()
	sev := strings.ToLower(strings.TrimSpace(minSeverity))
	if sev != "low" && sev != "medium" && sev != "high" {
		sev = "medium"
	}
	if minOverall < 0 {
		minOverall = 0
	}
	if minOverall > 100 {
		minOverall = 100
	}
	now := time.Now().UTC()
	rule := AlertRule{
		ID:             "rule-" + fmt.Sprintf("%d", now.UnixNano()),
		RegionID:       strings.ToUpper(strings.TrimSpace(regionID)),
		MinSeverity:    sev,
		MinOverallRisk: minOverall,
		Enabled:        true,
		CreatedAt:      now,
		CreatedBy:      "operator",
	}
	s.state.AlertRules = append(s.state.AlertRules, rule)
	s.state.Audits = append(s.state.Audits, AuditEvent{
		At: now, Action: "alert_rule.created", Actor: "operator",
		Detail: fmt.Sprintf("%s sev=%s min=%.0f", rule.RegionID, rule.MinSeverity, rule.MinOverallRisk),
	})
	if s.bus != nil {
		s.bus.PublishEvent("alert.rule.created", rule.ID, rule)
	}
	_ = s.save()
	return rule
}

// CreateCase opens an isolated OSINT case in the shared store (not personal memory).
func (s *Store) CreateCase(title, purpose string) (Case, error) {
	return s.CreateCaseWithID("", title, purpose)
}

// CreateCaseWithID opens/mirrors a case under a fixed ID (U5 dual-store alignment with root).
func (s *Store) CreateCaseWithID(id, title, purpose string) (Case, error) {
	title = strings.TrimSpace(title)
	purpose = strings.TrimSpace(purpose)
	if title == "" || purpose == "" {
		return Case{}, fmt.Errorf("title and purpose are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if strings.TrimSpace(id) == "" {
		id = "case-" + fmt.Sprintf("%d", now.UnixNano())
	} else {
		id = strings.TrimSpace(id)
		for _, existing := range s.state.Cases {
			if existing.ID == id {
				// Idempotent mirror — already present with same id
				return existing, nil
			}
		}
	}
	c := Case{
		ID:             id,
		Title:          title,
		Purpose:        purpose,
		Classification: "operator-controlled",
		AllowedSources: []string{},
		Evidence:       []Evidence{},
		Entities:       []Entity{},
		Relations:      []Relation{},
		Audit: []AuditEvent{{
			At: now, Action: "case.created", Actor: "operator",
			Detail: "Case opened with stated purpose (isolated from personal memory).",
		}},
	}
	s.state.Cases = append(s.state.Cases, c)
	s.state.Audits = append(s.state.Audits, AuditEvent{
		At: now, Action: "case.created", Actor: "operator", Detail: c.ID + " " + purpose,
	})
	if s.bus != nil {
		s.bus.PublishEvent("case.updated", c.ID, c)
	}
	_ = s.save()
	return c, nil
}

// AddCaseEntity mirrors a root-case entity into the shared case shell (same case id).
func (s *Store) AddCaseEntity(caseID, entityID, label, kind string, confidence int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Cases {
		if s.state.Cases[i].ID != caseID {
			continue
		}
		for _, e := range s.state.Cases[i].Entities {
			if e.ID == entityID {
				return nil
			}
		}
		s.state.Cases[i].Entities = append(s.state.Cases[i].Entities, Entity{
			ID: entityID, Label: label, Kind: kind, Confidence: confidence,
		})
		s.state.Audits = append(s.state.Audits, AuditEvent{
			At: time.Now().UTC(), Action: "entity.added", Actor: "operator", Detail: entityID + " case=" + caseID,
		})
		_ = s.save()
		return nil
	}
	return fmt.Errorf("case not found")
}

// LinkCaseRelation mirrors a root relation into the shared case shell.
func (s *Store) LinkCaseRelation(caseID, from, to, relType, evidenceID string, confidence int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Cases {
		if s.state.Cases[i].ID != caseID {
			continue
		}
		s.state.Cases[i].Relations = append(s.state.Cases[i].Relations, Relation{
			FromEntity: from, ToEntity: to, RelationType: relType,
			EvidenceIDs: []string{evidenceID}, Confidence: confidence, ValidFrom: time.Now().UTC(),
		})
		_ = s.save()
		return nil
	}
	return fmt.Errorf("case not found")
}

// SealCaseEvidence adds sealed evidence into the shared case graph (same id as root).
func (s *Store) SealCaseEvidence(caseID, evidenceID, sourceID, url, excerpt, sha256hex, sourceEventID string) (Evidence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Cases {
		if s.state.Cases[i].ID != caseID {
			continue
		}
		for _, existing := range s.state.Cases[i].Evidence {
			if existing.ID == evidenceID {
				return existing, nil
			}
		}
		now := time.Now().UTC()
		if evidenceID == "" {
			evidenceID = "case-evid-" + fmt.Sprintf("%d", now.UnixNano())
		}
		e := Evidence{
			ID: evidenceID, CaseID: caseID, SourceID: sourceID, URL: url, Excerpt: excerpt,
			SHA256: sha256hex, CollectedAt: now, Sealed: true, ValidationStatus: "pending",
			ChainOfCustodyID: "coc-" + evidenceID,
		}
		s.state.Cases[i].Evidence = append(s.state.Cases[i].Evidence, e)
		// Also keep a world-level evidence ref for provenance lists
		s.state.Evidence = append(s.state.Evidence, e)
		detail := evidenceID
		if sourceEventID != "" {
			detail += " source_event_id=" + sourceEventID
			s.state.Cases[i].Audit = append(s.state.Cases[i].Audit, AuditEvent{
				At: now, Action: "event.promoted", Actor: "operator", Detail: "source_event_id=" + sourceEventID + " evidence_id=" + evidenceID,
			})
		}
		s.state.Cases[i].Audit = append(s.state.Cases[i].Audit, AuditEvent{
			At: now, Action: "evidence.sealed", Actor: "operator", Detail: detail,
		})
		s.state.Audits = append(s.state.Audits, AuditEvent{At: now, Action: "evidence.sealed", Actor: "operator", Detail: caseID + " " + evidenceID})
		if s.bus != nil {
			s.bus.PublishEvent("evidence.sealed", evidenceID, e)
		}
		_ = s.save()
		return e, nil
	}
	return Evidence{}, fmt.Errorf("case not found")
}

// GetCase returns a shared case by id (copy).
func (s *Store) GetCase(caseID string) (Case, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.state.Cases {
		if c.ID == caseID {
			return c, true
		}
	}
	return Case{}, false
}

// ListCases returns shared case shells (title/purpose/counts).
func (s *Store) ListCases() []Case {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Case, len(s.state.Cases))
	copy(out, s.state.Cases)
	return out
}

// IngestObservationsBatch ingests connector-produced observations (U6 Fetch path).
func (s *Store) IngestObservationsBatch(obs []Observation) int {
	n := 0
	for _, o := range obs {
		if strings.TrimSpace(o.ID) == "" || strings.TrimSpace(o.RawText) == "" {
			continue
		}
		s.IngestObservation(o)
		n++
	}
	return n
}

// RecordAgentAction appends a skill-execution audit row (U7 continuity).
func (s *Store) RecordAgentAction(skill, argsJSON, result, approvedBy, status string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if len(argsJSON) > 2000 {
		argsJSON = argsJSON[:2000] + "…"
	}
	if len(result) > 2000 {
		result = result[:2000] + "…"
	}
	a := AgentAction{
		ID: "act-" + fmt.Sprintf("%d", now.UnixNano()), Skill: skill, Args: argsJSON, Result: result,
		ApprovedBy: approvedBy, ExecutedAt: now, Status: status,
	}
	s.state.AgentActions = append(s.state.AgentActions, a)
	// Cap history
	if len(s.state.AgentActions) > 200 {
		s.state.AgentActions = s.state.AgentActions[len(s.state.AgentActions)-200:]
	}
	s.state.Audits = append(s.state.Audits, AuditEvent{
		At: now, Action: "agent.action", Actor: "system", Detail: skill + " status=" + status,
	})
	_ = s.save()
}

// IdentitySnapshot returns technical continuity fields (not consciousness).
func (s *Store) IdentitySnapshot() IdentityProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id := s.state.Identity
	if id.Name == "" {
		id.Name = "AETHEL"
	}
	if id.Version == "" {
		id.Version = "unified-1"
	}
	return id
}

// SetAssessmentStatus lets the operator mark an assessment verified/disputed/rejected/corroborated.
func (s *Store) SetAssessmentStatus(assessID, status, reviewer string) error {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "verified", "disputed", "rejected", "corroborated", "unverified", "hypothesis":
	default:
		return fmt.Errorf("invalid assessment status")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Assessments {
		if s.state.Assessments[i].ID != assessID {
			continue
		}
		s.state.Assessments[i].Status = status
		s.state.Assessments[i].ReviewedBy = strings.TrimSpace(reviewer)
		if s.state.Assessments[i].ReviewedBy == "" {
			s.state.Assessments[i].ReviewedBy = "operator"
		}
		s.state.Audits = append(s.state.Audits, AuditEvent{
			At: time.Now().UTC(), Action: "assessment." + status, Actor: s.state.Assessments[i].ReviewedBy, Detail: assessID,
		})
		if s.bus != nil {
			s.bus.PublishEvent("assessment.updated", assessID, s.state.Assessments[i])
		}
		_ = s.save()
		return nil
	}
	return fmt.Errorf("assessment not found")
}

// UpsertSource registers or updates a source with full provenance fields (U6).
func (s *Store) UpsertSource(src Source) Source {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if src.FetchedAt.IsZero() {
		src.FetchedAt = now
	}
	if src.AvailabilityStatus == "" {
		src.AvailabilityStatus = "ok"
	}
	if src.TrustTier < 1 || src.TrustTier > 3 {
		src.TrustTier = 2
	}
	for i := range s.state.Sources {
		if s.state.Sources[i].ID == src.ID {
			// Preserve higher trust if already set lower tier number
			if s.state.Sources[i].TrustTier > 0 && s.state.Sources[i].TrustTier < src.TrustTier {
				src.TrustTier = s.state.Sources[i].TrustTier
			}
			s.state.Sources[i] = src
			_ = s.save()
			return src
		}
	}
	s.state.Sources = append(s.state.Sources, src)
	_ = s.save()
	return src
}

// DeleteCase removes a shared-store case by ID (audited).
func (s *Store) DeleteCase(caseID string) error {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" {
		return fmt.Errorf("case id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, c := range s.state.Cases {
		if c.ID == caseID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("case not found")
	}
	s.state.Cases = append(s.state.Cases[:idx], s.state.Cases[idx+1:]...)
	now := time.Now().UTC()
	s.state.Audits = append(s.state.Audits, AuditEvent{
		At: now, Action: "case.deleted", Actor: "operator", Detail: caseID,
	})
	if s.bus != nil {
		s.bus.PublishEvent("case.updated", caseID, map[string]any{"deleted": true})
	}
	return s.save()
}

// GetAlertRules returns a copy of stored alert rules.
func (s *Store) GetAlertRules() []AlertRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]AlertRule, len(s.state.AlertRules))
	copy(cp, s.state.AlertRules)
	return cp
}

// DomainSummary builds a domain-filtered status from the shared Events + RiskScores.
func (s *Store) DomainSummary(domain string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dom := strings.ToLower(strings.TrimSpace(domain))
	matchDomain := func(evDomain string) bool {
		d := strings.ToLower(evDomain)
		switch dom {
		case "cyber":
			return d == "cyber"
		case "market", "economic":
			return d == "economic"
		case "conflict":
			return d == "humanitarian"
		case "infrastructure":
			return d == "infrastructure" || d == "general"
		default:
			return dom == "" || d == dom
		}
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## %s summary (SharedIntelStore)\n", strings.ToUpper(dom)))
	n := 0
	for i := len(s.state.Events) - 1; i >= 0 && n < 12; i-- {
		ev := s.state.Events[i]
		if !matchDomain(ev.Domain) {
			continue
		}
		b.WriteString(fmt.Sprintf("- [INFERENCE] %s sev=%s conf=%d%% — %s\n", ev.ID, ev.Severity, ev.Confidence, safeTitle(ev.Title)))
		n++
	}
	if n == 0 {
		b.WriteString("- No classified events for this domain in the shared store.\n")
	}
	b.WriteString("\nRisk dimensions (derived analytical scores):\n")
	for id, rs := range s.state.RiskScores {
		switch dom {
		case "cyber":
			b.WriteString(fmt.Sprintf("- %s CyberRisk=%.1f Overall=%.1f\n", id, rs.CyberRisk, rs.OverallRisk))
		case "market", "economic":
			b.WriteString(fmt.Sprintf("- %s Econ=%.1f Financial=%.1f Overall=%.1f\n", id, rs.EconomicRisk, rs.FinancialRisk, rs.OverallRisk))
		case "conflict":
			b.WriteString(fmt.Sprintf("- %s Conflict=%.1f Overall=%.1f\n", id, rs.ConflictRisk, rs.OverallRisk))
		case "infrastructure":
			b.WriteString(fmt.Sprintf("- %s Infra=%.1f Energy=%.1f Overall=%.1f\n", id, rs.InfrastructureRisk, rs.EnergyRisk, rs.OverallRisk))
		default:
			b.WriteString(fmt.Sprintf("- %s Overall=%.1f\n", id, rs.OverallRisk))
		}
	}
	b.WriteString("\nUncertainties: domain tags are rule-engine inference; raw observations remain unverified.\n")
	return b.String()
}

// GetTimeline returns events in reverse chrono order (timeline view per spec).
func (s *Store) GetTimeline(limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.state.Events)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]Event, 0, limit)
	for i := n - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, s.state.Events[i])
	}
	return out
}

// EvaluateAlertRules evaluates stored AlertRules and rising-risk watchlists against RiskScores.
// All creates go through AlertManager (cooldown / min-confidence / dedup).
func (s *Store) EvaluateAlertRules() []Alert {
	s.mu.Lock()
	defer s.mu.Unlock()
	var triggered []Alert
	now := time.Now().UTC()
	mgr := NewAlertManager()

	// Operator-defined rules (persisted via AddAlertRule)
	for _, rule := range s.state.AlertRules {
		if !rule.Enabled || rule.RegionID == "" {
			continue
		}
		rs, ok := s.state.RiskScores[rule.RegionID]
		if !ok {
			continue
		}
		if rs.OverallRisk < rule.MinOverallRisk {
			continue
		}
		// Severity gate: high requires elevated overall
		if rule.MinSeverity == "high" && rs.OverallRisk < 55 {
			continue
		}
		conf := rs.Confidence
		if conf < 1 {
			conf = int(rs.OverallRisk)
		}
		if conf > 100 {
			conf = 100
		}
		sev := rule.MinSeverity
		if sev == "" {
			sev = "medium"
		}
		candidate := Alert{
			ID:              "rule-alert-" + rule.ID + "-" + now.Format("150405"),
			Severity:        sev,
			Confidence:      conf,
			Region:          rule.RegionID,
			AffectedDomains: []string{"alert_rule"},
			Reason:          fmt.Sprintf("AlertRule %s: OverallRisk %.1f >= %.0f (sev=%s)", rule.ID, rs.OverallRisk, rule.MinOverallRisk, sev),
			EscalationState: "new",
		}
		if created := mgr.CreateAlert(candidate, s.state.Alerts, 1); created != nil {
			al := *created
			triggered = append(triggered, al)
			s.state.Alerts = append(s.state.Alerts, al)
			s.state.Identity.LastWarnedAt = now
			s.state.Identity.PendingAlertIDs = appendPending(s.state.Identity.PendingAlertIDs, al.ID, 20)
			if s.bus != nil {
				s.bus.PublishEvent("alert.created", al.ID, al)
			}
		}
	}

	// Watchlist rising-risk correlation
	for rid, rs := range s.state.RiskScores {
		if rs.Trend != "up" || rs.OverallRisk <= 45 {
			continue
		}
		for _, wl := range s.state.Watchlists {
			for _, wr := range wl.RegionIDs {
				if !strings.EqualFold(wr, rid) {
					continue
				}
				conf := rs.Confidence
				if conf < 1 {
					conf = int(rs.OverallRisk)
				}
				if conf > 100 {
					conf = 100
				}
				candidate := Alert{
					ID:              "wl-alert-" + rid + "-" + now.Format("150405"),
					Severity:        "medium",
					Confidence:      conf,
					Region:          rid,
					AffectedDomains: []string{"watchlist"},
					Reason:          "Rising risk on personal watchlist region " + rid,
					EscalationState: "new",
				}
				if created := mgr.CreateAlert(candidate, s.state.Alerts, 1); created != nil {
					al := *created
					triggered = append(triggered, al)
					s.state.Alerts = append(s.state.Alerts, al)
					s.state.Identity.LastWarnedAt = now
					s.state.Identity.PendingAlertIDs = appendPending(s.state.Identity.PendingAlertIDs, al.ID, 20)
					if s.bus != nil {
						s.bus.PublishEvent("alert.created", al.ID, al)
					}
				}
			}
		}
	}
	_ = s.save()
	return triggered
}

// SetPersonalContext wires the Personal Context Model (operator-declared) into the shared World State store for correlation.
func (s *Store) SetPersonalContext(pc PersonalContext) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pc.LastUpdated = time.Now().UTC()
	s.state.Personal = pc
	if s.bus != nil {
		s.bus.PublishEvent("personal.updated", pc.OperatorID, pc)
	}
	_ = s.save()
}

// GetPersonalContext returns a copy of the operator-declared personal context (opt-in trust class).
func (s *Store) GetPersonalContext() PersonalContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Personal
}

// PersonalImpact builds an opt-in correlation of PersonalContext + Watchlists + RiskScores + Events.
// Recommendations are labeled; no invented world facts.
func (s *Store) PersonalImpact() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var b strings.Builder
	b.WriteString("# Personal Impact Brief (SharedIntelStore)\n\n")
	b.WriteString("**Empfehlungen ≠ Fakten.** Korrelation nur aus Operator-deklariertem Personal Context + World State.\n\n")
	pc := s.state.Personal
	if pc.OperatorID == "" && len(pc.Interests) == 0 && len(pc.Goals) == 0 && len(pc.PreferredRegions) == 0 && len(s.state.Watchlists) == 0 {
		b.WriteString("Kein Personal Context / keine Watchlists. Sync via intelligence_sync_personal oder UI.\n")
		return b.String()
	}
	b.WriteString("## Personal Context\n")
	if pc.OperatorID != "" {
		b.WriteString("- Operator: " + pc.OperatorID + "\n")
	}
	if len(pc.Interests) > 0 {
		b.WriteString("- Interessen: " + strings.Join(pc.Interests, ", ") + "\n")
	}
	if len(pc.Goals) > 0 {
		b.WriteString("- Ziele: " + strings.Join(pc.Goals, ", ") + "\n")
	}
	if len(pc.Projects) > 0 {
		b.WriteString("- Projekte: " + strings.Join(pc.Projects, ", ") + "\n")
	}
	if len(pc.PreferredRegions) > 0 {
		b.WriteString("- Bevorzugte Regionen: " + strings.Join(pc.PreferredRegions, ", ") + "\n")
	}
	b.WriteString(fmt.Sprintf("- Risk tolerance: %s · updated: %s\n\n", pc.RiskTolerance, pc.LastUpdated.Format(time.RFC3339)))

	b.WriteString("## Watchlist × Regional Risk\n")
	hits := 0
	for _, wl := range s.state.Watchlists {
		b.WriteString(fmt.Sprintf("- Watchlist **%s** regions=%v keywords=%v\n", wl.Name, wl.RegionIDs, wl.Keywords))
		for _, rid := range wl.RegionIDs {
			key := normalizeRegionKey(rid)
			if rs, ok := s.state.RiskScores[key]; ok {
				hits++
				b.WriteString(fmt.Sprintf("  → %s Overall=%.1f%% trend=%s drivers=%v\n", key, rs.OverallRisk, rs.Trend, rs.PrimaryDrivers))
				if rs.Trend == "up" && rs.OverallRisk > 40 {
					b.WriteString("     **EMPFEHLUNG (nicht Fakt):** Lage prüfen, Evidence ansehen, ggf. Alert-Rule schärfen.\n")
				}
			}
		}
	}
	for _, rid := range pc.PreferredRegions {
		key := normalizeRegionKey(rid)
		if rs, ok := s.state.RiskScores[key]; ok {
			hits++
			b.WriteString(fmt.Sprintf("- Preferred region **%s**: Overall=%.1f%% trend=%s\n", key, rs.OverallRisk, rs.Trend))
		}
	}
	if hits == 0 {
		b.WriteString("Keine Überlappung Watchlist/PreferredRegions mit berechneten RiskScores.\n")
	}

	b.WriteString("\n## Keyword hits in recent Events (INFERENCE only)\n")
	keys := append([]string{}, pc.Interests...)
	keys = append(keys, pc.Goals...)
	keys = append(keys, pc.Projects...)
	for _, wl := range s.state.Watchlists {
		keys = append(keys, wl.Keywords...)
	}
	matchN := 0
	for i := len(s.state.Events) - 1; i >= 0 && matchN < 8; i-- {
		ev := s.state.Events[i]
		blob := strings.ToLower(ev.Title + " " + ev.Summary)
		for _, k := range keys {
			k = strings.ToLower(strings.TrimSpace(k))
			if len(k) < 3 {
				continue
			}
			if strings.Contains(blob, k) {
				b.WriteString(fmt.Sprintf("- [INFERENCE] keyword=%q event=%s conf=%d%% — %s\n", k, ev.ID, ev.Confidence, safeTitle(ev.Title)))
				matchN++
				break
			}
		}
	}
	if matchN == 0 {
		b.WriteString("- Keine Keyword-Treffer in aktuellen Events.\n")
	}

	b.WriteString("\n## Uncertainties\n")
	b.WriteString("- Keyword-Matches sind heuristisch, keine kausale Beweiskette.\n")
	b.WriteString("- RiskScores sind abgeleitet; Observations bleiben unverified.\n")
	b.WriteString("- Keine automatische Case-Vermischung mit persönlicher Erinnerung.\n")
	return b.String()
}

// StartProactiveLoop runs EvaluateAlertRules on an interval (local-only, no external I/O).
func (s *Store) StartProactiveLoop(interval time.Duration) {
	if interval < 30*time.Second {
		interval = 2 * time.Minute
	}
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for range t.C {
			_ = s.EvaluateAlertRules()
		}
	}()
}

// SourceHealth returns simple health for registered sources (used by skill)
func (s *Store) SourceHealth() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h := map[string]any{"total_sources": len(s.state.Sources), "last_fetch": ""}
	if len(s.state.Sources) > 0 {
		h["last_fetch"] = s.state.Sources[len(s.state.Sources)-1].FetchedAt.Format(time.RFC3339)
		h["trust_summary"] = "default tier 2 (local approved)"
	}
	h["status"] = "healthy-local"
	return h
}

// LiveNexusContext is the authoritative chat/world-event context from the UNIFIED store only.
// Criterion 1: no parallel chat truth. Criterion 2: RAW / INFERENCE / VERIFIED labels always visible.
func (s *Store) LiveNexusContext(limit int) string {
	return s.LiveNexusContextWithin(limit, 24)
}

// LiveNexusContextWithin applies a strict event-time window. Missing, future
// and stale timestamps never enter a "latest" intelligence answer.
func (s *Store) LiveNexusContextWithin(limit int, hours float64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit < 1 {
		limit = 1
	}
	if limit > 20 {
		limit = 20
	}
	if hours < 1 || hours > 720 {
		hours = 24
	}
	now := time.Now().UTC()
	cutoff := now.Add(-time.Duration(hours * float64(time.Hour)))
	inWindow := func(timestamp time.Time) bool {
		return !timestamp.IsZero() && !timestamp.Before(cutoff) && !timestamp.After(now.Add(10*time.Minute))
	}
	windowObservations := 0
	windowEvents := 0
	windowAssessments := 0
	windowAlerts := 0
	windowRisks := 0
	for _, observation := range s.state.Observations {
		if inWindow(observation.ObservedAt) {
			windowObservations++
		}
	}
	for _, event := range s.state.Events {
		if inWindow(event.ObservedAt) {
			windowEvents++
		}
	}
	for _, assessment := range s.state.Assessments {
		if inWindow(assessment.CreatedAt) {
			windowAssessments++
		}
	}
	for _, alert := range s.state.Alerts {
		if !alert.Acknowledged && inWindow(alert.CreatedAt) {
			windowAlerts++
		}
	}
	for _, risk := range s.state.RiskScores {
		if inWindow(risk.LastUpdated) {
			windowRisks++
		}
	}
	var b strings.Builder
	b.WriteString("UNIFIED INTELLIGENCE LIVE NEXUS CONTEXT (SharedIntelStore only)\n")
	b.WriteString("Layers: RAW OBSERVATION | INFERENCE (unverified classification) | VERIFIED FINDING (operator/evidence only)\n")
	b.WriteString(fmt.Sprintf("Strict event-time window: %.0fh | generated_at=%s\n", hours, now.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Registered sources: %d | Window raw observations: %d | Window classified events: %d | Window assessments: %d | Active window alerts: %d | Fresh risk regions: %d\n\n",
		len(s.state.Sources), windowObservations, windowEvents, windowAssessments, windowAlerts, windowRisks))

	b.WriteString("## RAW OBSERVATIONS (unreviewed claims captured from sources)\n")
	rawN := 0
	for i := len(s.state.Observations) - 1; i >= 0 && rawN < limit; i-- {
		o := s.state.Observations[i]
		if !inWindow(o.ObservedAt) {
			continue
		}
		b.WriteString(fmt.Sprintf("- [RAW][%s] src=%s @ %.2f,%.2f — %s\n", o.ID, o.SourceID, o.Latitude, o.Longitude, safeTitle(o.RawText)))
		rawN++
	}
	if rawN == 0 {
		b.WriteString("- (none)\n")
	}

	b.WriteString("\n## INFERENCE / CLASSIFIED EVENTS (Assessment status=unverified unless reviewed)\n")
	evN := 0
	for i := len(s.state.Events) - 1; i >= 0 && evN < limit; i-- {
		e := s.state.Events[i]
		if !inWindow(e.ObservedAt) {
			continue
		}
		b.WriteString(fmt.Sprintf("- [INFERENCE][%s] domain=%s sev=%s conf=%d%% — %s\n", e.ID, e.Domain, e.Severity, e.Confidence, safeTitle(e.Title)))
		evN++
	}
	if evN == 0 {
		b.WriteString("- (none)\n")
	}

	b.WriteString("\n## ASSESSMENTS (explicit status — never auto-verified from LLM plausibility)\n")
	asN := 0
	for i := len(s.state.Assessments) - 1; i >= 0 && asN < limit; i-- {
		a := s.state.Assessments[i]
		if !inWindow(a.CreatedAt) {
			continue
		}
		b.WriteString(fmt.Sprintf("- [ASSESSMENT][%s] status=%s conf=%d%% evidence=%v — %s\n", a.ID, a.Status, a.Confidence, a.EvidenceIDs, a.Statement))
		asN++
	}
	if asN == 0 {
		b.WriteString("- (none)\n")
	}

	b.WriteString("\n## VERIFIED FINDINGS\n")
	verifiedN := 0
	for _, a := range s.state.Assessments {
		if inWindow(a.CreatedAt) && (a.Status == "verified" || a.Status == "corroborated") {
			b.WriteString(fmt.Sprintf("- [VERIFIED][%s] status=%s — %s\n", a.ID, a.Status, a.Statement))
			verifiedN++
		}
	}
	if verifiedN == 0 {
		b.WriteString("- none (operator/evidence review required; source claims and classifications remain unreviewed)\n")
	}

	b.WriteString("\n## ACTIVE ALERTS\n")
	alN := 0
	for i := len(s.state.Alerts) - 1; i >= 0 && alN < limit; i-- {
		a := s.state.Alerts[i]
		if a.Acknowledged || !inWindow(a.CreatedAt) {
			continue
		}
		b.WriteString(fmt.Sprintf("- [ALERT][%s] region=%s sev=%s conf=%d%% — %s\n", a.ID, a.Region, a.Severity, a.Confidence, safeTitle(a.Reason)))
		alN++
	}
	if alN == 0 {
		b.WriteString("- (none active)\n")
	}

	b.WriteString("\n## RISK SNAPSHOT (derived scores — inference, explain via intelligence_explain_score)\n")
	rsN := 0
	for id, rs := range s.state.RiskScores {
		if rsN >= 6 {
			break
		}
		if !inWindow(rs.LastUpdated) {
			continue
		}
		b.WriteString(fmt.Sprintf("- [RISK][%s] overall=%.1f%% trend=%s drivers=%v missing=%v\n", id, rs.OverallRisk, rs.Trend, rs.PrimaryDrivers, rs.MissingData))
		rsN++
	}
	if rsN == 0 {
		b.WriteString("- (no regional scores yet)\n")
	}

	b.WriteString("\n## SOURCES (registry)\n")
	for i, src := range s.state.Sources {
		if i >= limit {
			break
		}
		b.WriteString(fmt.Sprintf("- [SOURCE][%s] type=%s trust_tier=%d status=%s\n", src.ID, src.SourceType, src.TrustTier, src.AvailabilityStatus))
	}
	if len(s.state.Sources) == 0 {
		b.WriteString("- (none)\n")
	}

	b.WriteString("\n**Provenance:** All rows come from SharedIntelStore (RSS→Observation→Ingest). Map pins and chat use this same model. Do not invent sources or treat INFERENCE as VERIFIED.\n")
	return b.String()
}
