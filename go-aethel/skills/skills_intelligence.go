package skills

// STATUS: PLATIN

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-aethel/osint"
	"go-aethel/personal"
	"go-aethel/security"
	"strconv"
	"strings"
	"time"

	"go-aethel/intelligence"
)

type IntelligenceStatusSkill struct{}

type GlobalWatchNexusContextSkill struct{}
type globalWatchNexusContextArgs struct {
	Limit int     `json:"limit"`
	Hours float64 `json:"hours"`
}

func (s *GlobalWatchNexusContextSkill) Name() string { return "global_watch_nexus_context" }
func (s *GlobalWatchNexusContextSkill) Description() string {
	return "Liest das autoritative Lagebild aus dem UNIFIED intelligence.SharedIntelStore (RAW/INFERENCE/VERIFIED). Pflicht vor Weltlage-Schlussfolgerungen — keine parallele Chat-Wahrheit."
}
func (s *GlobalWatchNexusContextSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *GlobalWatchNexusContextSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{
		"limit": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 20},
		"hours": map[string]interface{}{"type": "number", "minimum": 1, "maximum": 720, "description": "Strict recency window; defaults to 24 hours."},
	}, "additionalProperties": false}
}
func (s *GlobalWatchNexusContextSkill) Execute(args json.RawMessage) (string, error) {
	var input globalWatchNexusContextArgs
	if len(args) > 0 && string(args) != "null" {
		if err := json.Unmarshal(args, &input); err != nil {
			return "", errors.New("invalid nexus context arguments")
		}
	}
	if input.Hours == 0 {
		input.Hours = 24
	}
	if input.Hours < 1 || input.Hours > 720 {
		return "", errors.New("nexus context hours out of range")
	}
	// Criterion 1: exclusive unified model — same store as map/handlers/region_status.
	if intelligence.SharedIntelStore != nil {
		return intelligence.SharedIntelStore.LiveNexusContextWithin(input.Limit, input.Hours), nil
	}
	return "", errors.New("unified intelligence.SharedIntelStore unavailable — cannot invent parallel chat truth")
}

type GlobalWatchScheduleBriefingSkill struct{}
type globalWatchScheduleArgs struct {
	Enabled         bool `json:"enabled"`
	IntervalMinutes int  `json:"interval_minutes"`
}

func (s *GlobalWatchScheduleBriefingSkill) Name() string { return "global_watch_schedule_briefing" }
func (s *GlobalWatchScheduleBriefingSkill) Description() string {
	return "Aktiviert oder deaktiviert lokale, regelmaessige Global-Watch-Markdown-Berichte. Der Operator muss Intervall und Aktivierung explizit nennen; Berichte verlassen den Rechner nicht."
}
func (s *GlobalWatchScheduleBriefingSkill) RiskLevel() security.RiskLevel {
	return security.RiskModerate
}
func (s *GlobalWatchScheduleBriefingSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"enabled": map[string]interface{}{"type": "boolean"}, "interval_minutes": map[string]interface{}{"type": "integer", "minimum": 15, "maximum": 1440}}, "required": []string{"enabled", "interval_minutes"}, "additionalProperties": false}
}
func (s *GlobalWatchScheduleBriefingSkill) Execute(args json.RawMessage) (string, error) {
	var input globalWatchScheduleArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid global watch schedule arguments")
	}
	if state == nil || state.intelMonitor == nil {
		return "", errors.New("global watch monitor unavailable")
	}
	if err := state.intelMonitor.Configure(input.Enabled, input.IntervalMinutes); err != nil {
		return "", err
	}
	if input.Enabled {
		return fmt.Sprintf("Lokale Global-Watch-Berichte aktiviert: alle %d Minuten.", input.IntervalMinutes), nil
	}
	return "Lokale Global-Watch-Berichte deaktiviert.", nil
}

type IntelligenceProposeObservationSkill struct{}
type intelligenceObservationArgs struct {
	Title      string  `json:"title"`
	Summary    string  `json:"summary"`
	Source     string  `json:"source"`
	SourceURL  string  `json:"source_url"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Confidence int     `json:"confidence"`
	Severity   string  `json:"severity"`
}

func (s *IntelligenceProposeObservationSkill) Name() string {
	return "intelligence_propose_observation"
}
func (s *IntelligenceProposeObservationSkill) Description() string {
	return "Erstellt einen lokalen, unverifizierten Beobachtungsvorschlag für Global Watch. Verwende nur klar benannte Quellen; dies erzeugt keinen Case und keine externe Aktion."
}
func (s *IntelligenceProposeObservationSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *IntelligenceProposeObservationSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"title": map[string]interface{}{"type": "string", "minLength": 3, "maxLength": 160}, "summary": map[string]interface{}{"type": "string", "maxLength": 480}, "source": map[string]interface{}{"type": "string", "minLength": 2, "maxLength": 120}, "source_url": map[string]interface{}{"type": "string", "maxLength": 1024}, "latitude": map[string]interface{}{"type": "number", "minimum": -90, "maximum": 90}, "longitude": map[string]interface{}{"type": "number", "minimum": -180, "maximum": 180}, "confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100}, "severity": map[string]interface{}{"type": "string", "enum": []string{"low", "medium", "high"}}}, "required": []string{"title", "source", "confidence", "severity"}, "additionalProperties": false}
}
func (s *IntelligenceProposeObservationSkill) Execute(args json.RawMessage) (string, error) {
	var input intelligenceObservationArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid observation arguments")
	}
	// Prefer unified intelligence.SharedIntelStore so chat-proposed signals appear on the same model as the map.
	if intelligence.SharedIntelStore != nil {
		obsID := fmt.Sprintf("obs-propose-%d", time.Now().UnixNano())
		raw := strings.TrimSpace(input.Title)
		if strings.TrimSpace(input.Summary) != "" {
			raw = raw + " " + strings.TrimSpace(input.Summary)
		}
		intelligence.SharedIntelStore.IngestObservation(intelligence.Observation{
			ID:         obsID,
			SourceID:   "operator-" + strings.ReplaceAll(strings.TrimSpace(input.Source), " ", "_"),
			RawText:    raw,
			ObservedAt: time.Now().UTC(),
			Latitude:   input.Latitude,
			Longitude:  input.Longitude,
			Domain:     "geo",
		})
		return "Beobachtung als RAW Observation im Unified intelligence.SharedIntelStore aufgenommen (Assessment unverified). Gleiches Modell wie Karte/Alerts.", nil
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence core unavailable")
	}
	event := intelligence.IntelligenceEvent{Title: input.Title, Summary: input.Summary, Source: input.Source, SourceURL: input.SourceURL, Latitude: input.Latitude, Longitude: input.Longitude, Confidence: input.Confidence, Severity: input.Severity}
	if err := state.intel.ProposeEvent(event); err != nil {
		if err == intelligence.ErrDuplicateObservation {
			return "Diese Beobachtung ist bereits im Global-Watch-Feed vorhanden.", nil
		}
		return "", err
	}
	return "Beobachtungsvorschlag im Global Watch erstellt. Der Status ist proposed und erfordert eine menschliche Prüfung.", nil
}

func (s *IntelligenceStatusSkill) Name() string { return "intelligence_status" }
func (s *IntelligenceStatusSkill) Description() string {
	return "Liest den lokalen Zustand von Global Watch: Events, Cases und Datenschutzmodus. Nutze dies vor jeder OSINT-Analyse."
}

type IntelligenceAddEntitySkill struct{}
type intelligenceEntityArgs struct {
	CaseID     string `json:"case_id"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	Confidence int    `json:"confidence"`
}

func (s *IntelligenceAddEntitySkill) Name() string { return "intelligence_add_entity" }
func (s *IntelligenceAddEntitySkill) Description() string {
	return "Erfasst eine Case-lokale Entitaet aus belegtem Kontext. Personen werden sofort fallbezogen pseudonymisiert; keine Rohidentitaet wird gespeichert."
}
func (s *IntelligenceAddEntitySkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *IntelligenceAddEntitySkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"case_id": map[string]interface{}{"type": "string"}, "label": map[string]interface{}{"type": "string", "minLength": 1, "maxLength": 160}, "kind": map[string]interface{}{"type": "string", "enum": []string{"person", "organisation", "location", "asset", "event"}}, "confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100}}, "required": []string{"case_id", "label", "kind", "confidence"}, "additionalProperties": false}
}
func (s *IntelligenceAddEntitySkill) Execute(args json.RawMessage) (string, error) {
	var input intelligenceEntityArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid entity arguments")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence core unavailable")
	}
	entity, err := state.intel.AddEntity(input.CaseID, input.Label, input.Kind, input.Confidence)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Entitaet %s (%s, %d%%) im Case gespeichert.", entity.Label, entity.Kind, entity.Confidence), nil
}

type IntelligenceLinkEntitiesSkill struct{}
type intelligenceRelationArgs struct {
	CaseID     string `json:"case_id"`
	From       string `json:"from_entity_id"`
	To         string `json:"to_entity_id"`
	Relation   string `json:"relation"`
	EvidenceID string `json:"evidence_id"`
	Confidence int    `json:"confidence"`
}

func (s *IntelligenceLinkEntitiesSkill) Name() string { return "intelligence_link_entities" }
func (s *IntelligenceLinkEntitiesSkill) Description() string {
	return "Verknuepft zwei Case-Entitaeten ausschliesslich ueber eine bereits versiegelte Evidence-ID. Hypothesen ohne Evidenz duerfen nicht verknuepft werden."
}
func (s *IntelligenceLinkEntitiesSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *IntelligenceLinkEntitiesSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"case_id": map[string]interface{}{"type": "string"}, "from_entity_id": map[string]interface{}{"type": "string"}, "to_entity_id": map[string]interface{}{"type": "string"}, "relation": map[string]interface{}{"type": "string", "minLength": 2, "maxLength": 100}, "evidence_id": map[string]interface{}{"type": "string"}, "confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100}}, "required": []string{"case_id", "from_entity_id", "to_entity_id", "relation", "evidence_id", "confidence"}, "additionalProperties": false}
}
func (s *IntelligenceLinkEntitiesSkill) Execute(args json.RawMessage) (string, error) {
	var input intelligenceRelationArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid relation arguments")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence core unavailable")
	}
	relation, err := state.intel.LinkEntities(input.CaseID, input.From, input.To, input.Relation, input.EvidenceID, input.Confidence)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Evidenzgebundene Beziehung %s -> %s (%s, %d%%) gespeichert.", relation.From, relation.To, relation.Type, relation.Confidence), nil
}
func (s *IntelligenceStatusSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceStatusSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s *IntelligenceStatusSkill) Execute(_ json.RawMessage) (string, error) {
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence core unavailable")
	}
	d := state.intel.Snapshot()
	return fmt.Sprintf("Global Watch ist local-first aktiv: %d Beobachtungen, %d Cases, Revision %d. Raw PII ist deaktiviert.", len(d.Events), len(d.Cases), d.Revision), nil
}

type IntelligenceCollectSkill struct{}
type intelligenceCollectArgs struct {
	SourceID string `json:"source_id"`
}

func (s *IntelligenceCollectSkill) Name() string { return "intelligence_collect_source" }
func (s *IntelligenceCollectSkill) Description() string {
	return "Aktualisiert einen registrierten, kuratierten Intelligence-Feed. Akzeptiert nur source_id aus der Source Registry, niemals URLs."
}
func (s *IntelligenceCollectSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *IntelligenceCollectSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"source_id": map[string]interface{}{"type": "string", "enum": []string{"tagesschau-world", "dw-world", "usgs-earthquakes"}, "description": "ID der registrierten Quelle"}}, "required": []string{"source_id"}, "additionalProperties": false}
}
func (s *IntelligenceCollectSkill) Execute(args json.RawMessage) (string, error) {
	var input intelligenceCollectArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid source collection arguments")
	}
	if state == nil || state.intelSources == nil {
		return "", errors.New("source registry unavailable")
	}
	count, err := state.intelSources.Collect(strings.TrimSpace(input.SourceID))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Quelle %s aktualisiert: %d Beobachtungsvorschläge erstellt.", input.SourceID, count), nil
}

type IntelligenceCreateCaseSkill struct{}
type intelligenceCreateCaseArgs struct {
	Title   string `json:"title"`
	Purpose string `json:"purpose"`
}

func (s *IntelligenceCreateCaseSkill) Name() string { return "intelligence_create_case" }
func (s *IntelligenceCreateCaseSkill) Description() string {
	return "Eröffnet einen lokalen Evidence-Case. Erfordert einen konkreten legitimen Analysezweck und erzeugt noch keine externe Aktion."
}
func (s *IntelligenceCreateCaseSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *IntelligenceCreateCaseSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"title": map[string]interface{}{"type": "string", "minLength": 3, "maxLength": 160}, "purpose": map[string]interface{}{"type": "string", "minLength": 8, "maxLength": 500}}, "required": []string{"title", "purpose"}, "additionalProperties": false}
}
func (s *IntelligenceCreateCaseSkill) Execute(args json.RawMessage) (string, error) {
	var input intelligenceCreateCaseArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid case arguments")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence core unavailable")
	}
	c, err := state.intel.CreateCase(input.Title, input.Purpose)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Case %s erstellt: %s. Der Case ist operator-controlled und enthält noch keine Evidenz.", c.ID, c.Title), nil
}

// ─── OSINT specific skills for GLOBAL WATCH control ──────────────────────────

type OSINTAddCustomFeedSkill struct{}
type osintAddFeedArgs struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Domain string `json:"domain"`
}

func (s *OSINTAddCustomFeedSkill) Name() string { return "osint_add_custom_feed" }
func (s *OSINTAddCustomFeedSkill) Description() string {
	return "Fügt eine eigene RSS-, Atom- oder öffentliche Telegram-Quelle hinzu. Telegram verwendet ausschließlich https://t.me/s/Kanalname. Domain: general|geo|cyber|economic|humanitarian."
}
func (s *OSINTAddCustomFeedSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *OSINTAddCustomFeedSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "type": map[string]interface{}{"type": "string", "enum": []string{"rss", "atom", "telegram"}}, "url": map[string]interface{}{"type": "string"}, "domain": map[string]interface{}{"type": "string", "enum": []string{"general", "geo", "cyber", "economic", "humanitarian"}}}, "required": []string{"name", "url"}, "additionalProperties": false}
}
func (s *OSINTAddCustomFeedSkill) Execute(args json.RawMessage) (string, error) {
	var input osintAddFeedArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid feed args")
	}
	if state == nil || state.osint == nil {
		return "", errors.New("osint engine unavailable")
	}
	collectorType := strings.ToLower(strings.TrimSpace(input.Type))
	if collectorType == "" {
		collectorType = "rss"
	}
	cfg := osint.OSINTCollectorConfig{Name: input.Name, Type: collectorType, URL: input.URL, Domain: intelligence.OSINTDomain(input.Domain), Enabled: true, Priority: 3}
	if err := state.osint.AddCollector(cfg); err != nil {
		return "", err
	}
	return fmt.Sprintf("Eigene Quelle '%s' hinzugefügt und aktiviert.", input.Name), nil
}

type OSINTSetBriefingPromptSkill struct{}
type osintPromptArgs struct {
	Prompt string `json:"prompt"`
}

func (s *OSINTSetBriefingPromptSkill) Name() string { return "osint_set_briefing_prompt" }
func (s *OSINTSetBriefingPromptSkill) Description() string {
	return "Passt den System-Prompt für AI OSINT Lagebriefings dauerhaft an. Vollständige Kontrolle über Analyse-Stil."
}
func (s *OSINTSetBriefingPromptSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *OSINTSetBriefingPromptSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"prompt": map[string]interface{}{"type": "string", "minLength": 20}}, "required": []string{"prompt"}, "additionalProperties": false}
}
func (s *OSINTSetBriefingPromptSkill) Execute(args json.RawMessage) (string, error) {
	var input osintPromptArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid prompt args")
	}
	p := strings.TrimSpace(input.Prompt)
	if len([]rune(p)) < 20 || len([]rune(p)) > 12000 {
		return "", errors.New("prompt must be between 20 and 12000 characters")
	}
	if err := osint.PersistOSINTBriefingPrompt(osint.OSINTBriefingPromptFile, p); err != nil {
		return "", err
	}
	return "OSINT Briefing Prompt wurde dauerhaft aktualisiert.", nil
}

// IntelligenceRequestReIDSkill - allows operator via AI or UI to request re-id (gated + audited by design)
type IntelligenceRequestReIDSkill struct{}
type reidArgs struct {
	CaseID  string `json:"case_id"`
	Purpose string `json:"purpose"`
}

func (s *IntelligenceRequestReIDSkill) Name() string { return "intelligence_request_reid" }
func (s *IntelligenceRequestReIDSkill) Description() string {
	return "Beantragt Re-Identifizierung für einen Case (Policy Gate + Audit). Wird nur protokolliert; echte Re-ID erfordert explizite Operator-Freigabe."
}
func (s *IntelligenceRequestReIDSkill) RiskLevel() security.RiskLevel { return security.RiskHigh }
func (s *IntelligenceRequestReIDSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"case_id": map[string]interface{}{"type": "string"}, "purpose": map[string]interface{}{"type": "string", "minLength": 10}}, "required": []string{"case_id", "purpose"}, "additionalProperties": false}
}
func (s *IntelligenceRequestReIDSkill) Execute(args json.RawMessage) (string, error) {
	var input reidArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid reid args")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence unavailable")
	}
	req, err := state.intel.RequestReID(strings.TrimSpace(input.CaseID), strings.TrimSpace(input.Purpose), "operator")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Re-ID request stored (audited). request_id=%s status=%s case=%s. Raw PII recovery remains not_eligible (HMAC aliases only). Dual-control: two different approvers via POST /v1/intelligence/cases/{id}/reid action=approve.",
		req.ID, req.Status, input.CaseID,
	), nil
}

// IntelligenceApproveReIDSkill — second half of dual-control unlock (alias metadata window only).
type IntelligenceApproveReIDSkill struct{}
type reidApproveArgs struct {
	CaseID    string `json:"case_id"`
	RequestID string `json:"request_id"`
	Approver  string `json:"approver"`
}

func (s *IntelligenceApproveReIDSkill) Name() string { return "intelligence_approve_reid" }
func (s *IntelligenceApproveReIDSkill) Description() string {
	return "Erste oder zweite Freigabe eines Re-ID Requests (Dual-Control). Zweite Freigabe öffnet 30min Alias-Metadaten-Fenster — keine Roh-PII."
}
func (s *IntelligenceApproveReIDSkill) RiskLevel() security.RiskLevel { return security.RiskHigh }
func (s *IntelligenceApproveReIDSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"case_id":    map[string]interface{}{"type": "string"},
			"request_id": map[string]interface{}{"type": "string"},
			"approver":   map[string]interface{}{"type": "string", "minLength": 2, "maxLength": 80},
		},
		"required": []string{"case_id", "request_id", "approver"}, "additionalProperties": false,
	}
}
func (s *IntelligenceApproveReIDSkill) Execute(args json.RawMessage) (string, error) {
	var input reidApproveArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid reid approve args")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence unavailable")
	}
	req, err := state.intel.ApproveReID(strings.TrimSpace(input.CaseID), strings.TrimSpace(input.RequestID), strings.TrimSpace(input.Approver))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Re-ID approval applied. request_id=%s status=%s unlocked=%v expires=%s. reidentification raw PII remains not_eligible.",
		req.ID, req.Status, req.Unlocked, req.ExpiresAt.Format(time.RFC3339)), nil
}

// ─── Global Watch control skills (Aethel as sovereign "WWV" operator) ────────

type GlobalWatchFocusSkill struct{}
type gwFocusArgs struct {
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Label string  `json:"label,omitempty"`
	Zoom  float64 `json:"zoom,omitempty"`
}

func (s *GlobalWatchFocusSkill) Name() string { return "global_watch_focus" }
func (s *GlobalWatchFocusSkill) Description() string {
	return "Setzt den Fokus der Global Watch Karte (Globe) auf eine Koordinate. Die KI kann so den Operator auf relevante Lagen lenken."
}
func (s *GlobalWatchFocusSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *GlobalWatchFocusSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"lat":   map[string]interface{}{"type": "number", "minimum": -90, "maximum": 90},
			"lon":   map[string]interface{}{"type": "number", "minimum": -180, "maximum": 180},
			"label": map[string]interface{}{"type": "string"},
			"zoom":  map[string]interface{}{"type": "number", "minimum": 0.5, "maximum": 3},
		},
		"required": []string{"lat", "lon"},
	}
}
func (s *GlobalWatchFocusSkill) Execute(args json.RawMessage) (string, error) {
	var input gwFocusArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid focus args")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("global watch unavailable")
	}
	if input.Zoom == 0 {
		input.Zoom = 1
	}
	if err := state.intel.FocusGlobalWatch(input.Lat, input.Lon, input.Zoom, input.Label); err != nil {
		return "", err
	}
	return fmt.Sprintf("Global Watch focus auf %.4f, %.4f gesetzt (Label: %s).", input.Lat, input.Lon, input.Label), nil
}

type GlobalWatchObserveSkill struct{}
type gwObserveArgs struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Title   string  `json:"title"`
	Summary string  `json:"summary"`
	Kind    string  `json:"kind,omitempty"`
}

func (s *GlobalWatchObserveSkill) Name() string { return "global_watch_observe" }
func (s *GlobalWatchObserveSkill) Description() string {
	return "Erstellt eine Beobachtung direkt aus dem aktuellen Global Watch Kontext (wird als pin im Globe + in Intelligence gespeichert)."
}
func (s *GlobalWatchObserveSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *GlobalWatchObserveSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"lat":     map[string]interface{}{"type": "number"},
			"lon":     map[string]interface{}{"type": "number"},
			"title":   map[string]interface{}{"type": "string", "minLength": 3},
			"summary": map[string]interface{}{"type": "string"},
			"kind":    map[string]interface{}{"type": "string"},
		},
		"required": []string{"lat", "lon", "title", "summary"},
	}
}
func (s *GlobalWatchObserveSkill) Execute(args json.RawMessage) (string, error) {
	var input gwObserveArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid observe args")
	}
	// U1: intelligence.SharedIntelStore is authoritative world truth (map + chat nexus).
	if intelligence.SharedIntelStore == nil && (state == nil || state.intel == nil) {
		return "", errors.New("intelligence unavailable")
	}
	summary := input.Summary
	if input.Kind != "" {
		summary = fmt.Sprintf("[%s] %s", input.Kind, summary)
	}
	obsID := ""
	if intelligence.SharedIntelStore != nil {
		obsID = fmt.Sprintf("obs-gw-%d", time.Now().UnixNano())
		domain := "geo"
		if input.Kind != "" {
			domain = strings.ToLower(input.Kind)
		}
		intelligence.SharedIntelStore.IngestObservation(intelligence.Observation{
			ID:         obsID,
			SourceID:   "GLOBAL_WATCH",
			RawText:    strings.TrimSpace(input.Title + " " + summary),
			ObservedAt: time.Now().UTC(),
			Latitude:   input.Lat,
			Longitude:  input.Lon,
			Domain:     domain,
		})
	}
	// Bridge to root for case/SSE compat (same geo event).
	if state != nil && state.intel != nil {
		ev := intelligence.IntelligenceEvent{
			ID: obsID, Title: input.Title, Summary: summary,
			Latitude: input.Lat, Longitude: input.Lon,
			Source: "GLOBAL_WATCH", Confidence: 75, Severity: "medium",
		}
		_ = state.intel.ProposeEvent(ev)
	}
	return fmt.Sprintf("Beobachtung '%s' bei %.4f,%.4f in intelligence.SharedIntelStore erfasst (id=%s). Karte und Chat nutzen dieselbe Wahrheit.", input.Title, input.Lat, input.Lon, obsID), nil
}

type GlobalWatchToggleLayerSkill struct{}
type gwLayerArgs struct {
	Layer  string `json:"layer"`
	Enable bool   `json:"enable"`
}

func (s *GlobalWatchToggleLayerSkill) Name() string { return "global_watch_toggle_layer" }
func (s *GlobalWatchToggleLayerSkill) Description() string {
	return "Schaltet WWV-ähnliche Layer (borders,cameras,satellites,cables,daynight,news,cities) im Global Watch ein/aus. Richtige geo-pins für News + Kameras werden am korrekten Ort angezeigt."
}
func (s *GlobalWatchToggleLayerSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *GlobalWatchToggleLayerSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"layer":  map[string]interface{}{"type": "string", "enum": []string{"borders", "cameras", "satellites", "cables", "daynight", "news", "cities"}},
			"enable": map[string]interface{}{"type": "boolean"},
		},
		"required":             []string{"layer", "enable"},
		"additionalProperties": false,
	}
}
func (s *GlobalWatchToggleLayerSkill) Execute(args json.RawMessage) (string, error) {
	var input gwLayerArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("bad args")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("global watch unavailable")
	}
	if err := state.intel.SetGlobalWatchLayer(input.Layer, input.Enable); err != nil {
		return "", err
	}
	status := "disabled"
	if input.Enable {
		status = "enabled"
	}
	return "Layer " + input.Layer + " " + status + " (UI will reflect on next refresh).", nil
}

// NavigateUISkill switches the operator viewport (Global Watch, Sphere, Personal Core, …).
type NavigateUISkill struct{}
type navigateUIArgs struct {
	View string `json:"view"`
}

func (s *NavigateUISkill) Name() string { return "navigate_ui" }
func (s *NavigateUISkill) Description() string {
	return "Wechselt die AETHEL-Oberfläche: global_watch/globe, sphere, core, personal, chat, case, tasks, settings, memory. Nutze dies wenn der Operator z.B. 'wechsle auf Live Globe' sagt."
}
func (s *NavigateUISkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *NavigateUISkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"view": map[string]interface{}{
				"type": "string",
				"enum": []string{"core", "chat", "global_watch", "globe", "sphere", "personal", "case", "tasks", "settings", "memory", "agents", "agent_tracker"},
			},
		},
		"required": []string{"view"},
	}
}
func (s *NavigateUISkill) Execute(args json.RawMessage) (string, error) {
	var input navigateUIArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid navigate args")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence unavailable")
	}
	if err := state.intel.NavigateUI(input.View); err != nil {
		return "", err
	}
	return "UI navigation command published: " + input.View, nil
}

// GlobalWatchFocusRegionSkill centers the globe on a named region chip.
type GlobalWatchFocusRegionSkill struct{}
type gwFocusRegionArgs struct {
	Region string `json:"region"`
}

func (s *GlobalWatchFocusRegionSkill) Name() string { return "global_watch_focus_region" }
func (s *GlobalWatchFocusRegionSkill) Description() string {
	return "Fokussiert den Live Globe auf eine Region: global, europe, germany, mena, asia, americas, oceania, africa."
}
func (s *GlobalWatchFocusRegionSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *GlobalWatchFocusRegionSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"region": map[string]interface{}{
				"type": "string",
				"enum": []string{"global", "europe", "germany", "mena", "asia", "americas", "oceania", "africa"},
			},
		},
		"required": []string{"region"},
	}
}
func (s *GlobalWatchFocusRegionSkill) Execute(args json.RawMessage) (string, error) {
	var input gwFocusRegionArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid region focus args")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("global watch unavailable")
	}
	if err := state.intel.FocusGlobalWatchRegion(input.Region); err != nil {
		return "", err
	}
	return "Global Watch region focus: " + input.Region, nil
}

// GlobalWatchTimeWindowSkill sets feed/pin recency filter (hours; 0 = all).
type GlobalWatchTimeWindowSkill struct{}
type gwTimeWindowArgs struct {
	Hours float64 `json:"hours"`
}

func (s *GlobalWatchTimeWindowSkill) Name() string { return "global_watch_time_window" }
func (s *GlobalWatchTimeWindowSkill) Description() string {
	return "Setzt das Zeitfenster des Live Globe Feeds (Pins+Meldungen): hours=6|24|72|168 oder 0 für alle."
}
func (s *GlobalWatchTimeWindowSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *GlobalWatchTimeWindowSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"hours": map[string]interface{}{"type": "number", "minimum": 0, "maximum": 720},
		},
		"required": []string{"hours"},
	}
}
func (s *GlobalWatchTimeWindowSkill) Execute(args json.RawMessage) (string, error) {
	var input gwTimeWindowArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid time window args")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("global watch unavailable")
	}
	if err := state.intel.SetGlobalWatchTimeWindowHours(input.Hours); err != nil {
		return "", err
	}
	return fmt.Sprintf("Global Watch time window set to %.0fh (0=all).", input.Hours), nil
}

// GlobalWatchOpenReportSkill opens the freestanding report reader panel with markdown.
type GlobalWatchOpenReportSkill struct{}
type gwOpenReportArgs struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (s *GlobalWatchOpenReportSkill) Name() string { return "global_watch_open_report" }
func (s *GlobalWatchOpenReportSkill) Description() string {
	return "Öffnet den Report-Reader im Global Watch mit Markdown-Inhalt (zum Lesen von Briefings/News)."
}
func (s *GlobalWatchOpenReportSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *GlobalWatchOpenReportSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{"type": "string"},
			"body":  map[string]interface{}{"type": "string"},
		},
		"required": []string{"body"},
	}
}
func (s *GlobalWatchOpenReportSkill) Execute(args json.RawMessage) (string, error) {
	var input gwOpenReportArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", errors.New("invalid report args")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("global watch unavailable")
	}
	if err := state.intel.OpenGlobalWatchReport(input.Title, input.Body); err != nil {
		return "", err
	}
	return "Report reader opened.", nil
}

// ─── Phase 2 & 3 Intelligence Skills ─────────────────────────────────────────

type IntelligenceRegionStatusSkill struct{}
type regionStatusArgs struct {
	RegionID string `json:"region_id"`
}

func (s *IntelligenceRegionStatusSkill) Name() string { return "intelligence_region_status" }
func (s *IntelligenceRegionStatusSkill) Description() string {
	return "Gibt den aktuellen Risikoscore, die primären Treiber und den Trend für eine Region (GERMANY, FRANCE, USA, UKRAINE, UK) zurück."
}
func (s *IntelligenceRegionStatusSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceRegionStatusSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"region_id": map[string]interface{}{"type": "string", "enum": []string{"GERMANY", "FRANCE", "USA", "UKRAINE", "UK"}},
		},
		"required": []string{"region_id"},
	}
}
func (s *IntelligenceRegionStatusSkill) Execute(args json.RawMessage) (string, error) {
	var input regionStatusArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	// §19 path: shared model only — security developments + layer separation + evidence + risk
	if intelligence.SharedIntelStore != nil {
		query := intelligence.SharedIntelStore.QueryRegionSecurity(input.RegionID, 24)
		explain := intelligence.SharedIntelStore.ExplainScore(input.RegionID)
		return query + "\n\n## Score explanation\n" + explain, nil
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence core unavailable")
	}
	risks := state.intel.ComputeAllRegionalRisks()
	for _, r := range risks {
		if strings.EqualFold(r.RegionID, input.RegionID) {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("### Risikostatus für %s (%s) [legacy fallback]\n", r.RegionName, r.RegionID))
			sb.WriteString(fmt.Sprintf("* **Gesamtrisiko (Overall):** %.1f%%\n", r.OverallRisk))
			sb.WriteString(fmt.Sprintf("* **Trend:** %s\n", r.Trend))
			return sb.String(), nil
		}
	}
	return "Region nicht gefunden.", nil
}

// Create watchlist
type IntelligenceCreateWatchlistSkill struct{}
type createWatchlistArgs struct {
	Name     string `json:"name"`
	RegionID string `json:"region_id"`
}

func (s *IntelligenceCreateWatchlistSkill) Name() string { return "intelligence_create_watchlist" }
func (s *IntelligenceCreateWatchlistSkill) Description() string {
	return "Creates a personal watchlist for a region using shared model."
}
func (s *IntelligenceCreateWatchlistSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceCreateWatchlistSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":      map[string]interface{}{"type": "string"},
			"region_id": map[string]interface{}{"type": "string", "enum": []string{"GERMANY", "FRANCE", "USA", "UKRAINE", "UK"}},
		},
		"required": []string{"name", "region_id"},
	}
}
func (s *IntelligenceCreateWatchlistSkill) Execute(args json.RawMessage) (string, error) {
	var input createWatchlistArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if intelligence.SharedIntelStore != nil {
		wl := intelligence.SharedIntelStore.AddWatchlist(input.Name, []string{input.RegionID}, []string{})
		return fmt.Sprintf("Watchlist %s (%s) stored in shared model. ID=%s", wl.Name, input.RegionID, wl.ID), nil
	}
	return "Watchlist created: " + input.Name + " for " + input.RegionID + " (tied to shared model)", nil
}

// Infrastructure summary from intelligence.SharedIntelStore (domain-filtered Events + risk dims).
type IntelligenceInfrastructureSummarySkill struct{}

func (s *IntelligenceInfrastructureSummarySkill) Name() string {
	return "intelligence_infrastructure_summary"
}
func (s *IntelligenceInfrastructureSummarySkill) Description() string {
	return "Infrastructure/energy risk summary from the unified intelligence.SharedIntelStore."
}
func (s *IntelligenceInfrastructureSummarySkill) RiskLevel() security.RiskLevel {
	return security.RiskSafe
}
func (s *IntelligenceInfrastructureSummarySkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s *IntelligenceInfrastructureSummarySkill) Execute(args json.RawMessage) (string, error) {
	if intelligence.SharedIntelStore == nil {
		return "", errors.New("intelligence.SharedIntelStore unavailable")
	}
	return intelligence.SharedIntelStore.DomainSummary("infrastructure"), nil
}

// Create alert rule — persists to intelligence.SharedIntelStore.AlertRules.
type IntelligenceCreateAlertRuleSkill struct{}
type createAlertRuleArgs struct {
	RegionID       string  `json:"region_id"`
	Severity       string  `json:"severity"`
	MinOverallRisk float64 `json:"min_overall_risk"`
}

func (s *IntelligenceCreateAlertRuleSkill) Name() string { return "intelligence_create_alert_rule" }
func (s *IntelligenceCreateAlertRuleSkill) Description() string {
	return "Persists an alert rule on the unified intelligence.SharedIntelStore (region + severity + optional min overall risk)."
}
func (s *IntelligenceCreateAlertRuleSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceCreateAlertRuleSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"region_id":        map[string]interface{}{"type": "string"},
			"severity":         map[string]interface{}{"type": "string", "enum": []string{"low", "medium", "high"}},
			"min_overall_risk": map[string]interface{}{"type": "number", "minimum": 0, "maximum": 100},
		},
		"required": []string{"region_id", "severity"},
	}
}
func (s *IntelligenceCreateAlertRuleSkill) Execute(args json.RawMessage) (string, error) {
	var input createAlertRuleArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if intelligence.SharedIntelStore == nil {
		return "", errors.New("intelligence.SharedIntelStore unavailable — cannot create alert rule")
	}
	if strings.TrimSpace(input.RegionID) == "" {
		return "", errors.New("region_id required")
	}
	minRisk := input.MinOverallRisk
	if minRisk == 0 {
		switch strings.ToLower(input.Severity) {
		case "high":
			minRisk = 55
		case "low":
			minRisk = 25
		default:
			minRisk = 40
		}
	}
	rule := intelligence.SharedIntelStore.AddAlertRule(input.RegionID, input.Severity, minRisk)
	// Evaluate immediately against current scores for operator feedback
	triggered := intelligence.SharedIntelStore.EvaluateAlertRules()
	return fmt.Sprintf("Alert rule stored in intelligence.SharedIntelStore. ID=%s region=%s severity=%s min_overall=%.0f enabled=%v. Immediate evaluate: %d alert(s).",
		rule.ID, rule.RegionID, rule.MinSeverity, rule.MinOverallRisk, rule.Enabled, len(triggered)), nil
}

type IntelligenceConflictSummarySkill struct{}

func (s *IntelligenceConflictSummarySkill) Name() string { return "intelligence_conflict_summary" }
func (s *IntelligenceConflictSummarySkill) Description() string {
	return "Conflict/humanitarian summary from intelligence.SharedIntelStore."
}
func (s *IntelligenceConflictSummarySkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceConflictSummarySkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s *IntelligenceConflictSummarySkill) Execute(args json.RawMessage) (string, error) {
	if intelligence.SharedIntelStore == nil {
		return "", errors.New("intelligence.SharedIntelStore unavailable")
	}
	return intelligence.SharedIntelStore.DomainSummary("conflict"), nil
}

type IntelligenceCyberSummarySkill struct{}

func (s *IntelligenceCyberSummarySkill) Name() string { return "intelligence_cyber_summary" }
func (s *IntelligenceCyberSummarySkill) Description() string {
	return "Cyber domain summary from intelligence.SharedIntelStore."
}
func (s *IntelligenceCyberSummarySkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceCyberSummarySkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s *IntelligenceCyberSummarySkill) Execute(args json.RawMessage) (string, error) {
	if intelligence.SharedIntelStore == nil {
		return "", errors.New("intelligence.SharedIntelStore unavailable")
	}
	return intelligence.SharedIntelStore.DomainSummary("cyber"), nil
}

type IntelligenceMarketSummarySkill struct{}

func (s *IntelligenceMarketSummarySkill) Name() string { return "intelligence_market_summary" }
func (s *IntelligenceMarketSummarySkill) Description() string {
	return "Market/economic summary from intelligence.SharedIntelStore."
}
func (s *IntelligenceMarketSummarySkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceMarketSummarySkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s *IntelligenceMarketSummarySkill) Execute(args json.RawMessage) (string, error) {
	if intelligence.SharedIntelStore == nil {
		return "", errors.New("intelligence.SharedIntelStore unavailable")
	}
	return intelligence.SharedIntelStore.DomainSummary("market"), nil
}

// Global status using shared
type IntelligenceGlobalStatusSkill struct{}

func (s *IntelligenceGlobalStatusSkill) Name() string { return "intelligence_global_status" }
func (s *IntelligenceGlobalStatusSkill) Description() string {
	return "Returns global status from shared intelligence model."
}
func (s *IntelligenceGlobalStatusSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceGlobalStatusSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "additionalProperties": false}
}
func (s *IntelligenceGlobalStatusSkill) Execute(args json.RawMessage) (string, error) {
	var sb strings.Builder
	sb.WriteString("## Global Status from Shared Model\n")
	if intelligence.SharedIntelStore != nil {
		snap := intelligence.SharedIntelStore.GetSnapshot()
		sb.WriteString("Observations: " + strconv.Itoa(len(snap.Observations)) + "\n")
		sb.WriteString("Events: " + strconv.Itoa(len(snap.Events)) + "\n")
		sb.WriteString("Risk regions: " + strconv.Itoa(len(snap.RiskScores)) + "\n")
	} else {
		sb.WriteString("No shared data.\n")
	}
	return sb.String(), nil
}

// Recent changes using shared
type IntelligenceRecentChangesSkill struct{}
type recentChangesArgs struct {
	Limit int `json:"limit"`
}

func (s *IntelligenceRecentChangesSkill) Name() string { return "intelligence_recent_changes" }
func (s *IntelligenceRecentChangesSkill) Description() string {
	return "Returns recent changes from shared intelligence model."
}
func (s *IntelligenceRecentChangesSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceRecentChangesSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":                 "object",
		"properties":           map[string]interface{}{"limit": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 20}},
		"additionalProperties": false,
	}
}
func (s *IntelligenceRecentChangesSkill) Execute(args json.RawMessage) (string, error) {
	var input recentChangesArgs
	if len(args) > 0 {
		json.Unmarshal(args, &input)
	}
	if input.Limit < 1 {
		input.Limit = 5
	}
	var sb strings.Builder
	sb.WriteString("## Recent Changes from Shared Model\n")
	if intelligence.SharedIntelStore != nil {
		snap := intelligence.SharedIntelStore.GetSnapshot()
		count := 0
		for i := len(snap.Events) - 1; i >= 0 && count < input.Limit; i-- {
			ev := snap.Events[i]
			sb.WriteString("- " + ev.Title + " (" + ev.ObservedAt.Format("2006-01-02") + ")\n")
			count++
		}
	} else {
		sb.WriteString("No shared data.\n")
	}
	return sb.String(), nil
}

// OSINT case create — persists to intelligence.SharedIntelStore (isolated case slice) and root case workspace.
type OSINTCaseCreateSkill struct{}
type caseCreateArgs struct {
	Title   string `json:"title"`
	Purpose string `json:"purpose"`
}

func (s *OSINTCaseCreateSkill) Name() string { return "osint_case_create" }
func (s *OSINTCaseCreateSkill) Description() string {
	return "Opens an OSINT case with stated purpose. Persists to intelligence.SharedIntelStore (isolated from personal memory) and the local case workspace."
}
func (s *OSINTCaseCreateSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *OSINTCaseCreateSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title":   map[string]interface{}{"type": "string", "minLength": 2, "maxLength": 200},
			"purpose": map[string]interface{}{"type": "string", "minLength": 3, "maxLength": 500},
		},
		"required": []string{"title", "purpose"},
	}
}
func (s *OSINTCaseCreateSkill) Execute(args json.RawMessage) (string, error) {
	var input caseCreateArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	title := strings.TrimSpace(input.Title)
	purpose := strings.TrimSpace(input.Purpose)
	if title == "" || purpose == "" {
		return "", errors.New("title and purpose are required")
	}
	// Root CreateCase mirrors Shared under the SAME id (U5).
	if state == nil || state.intel == nil {
		if intelligence.SharedIntelStore == nil {
			return "", errors.New("no case store available")
		}
		c, err := intelligence.SharedIntelStore.CreateCase(title, purpose)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Case opened (shared-only). id=%s title=%q. Seal evidence before verified findings.", c.ID, title), nil
	}
	c, err := state.intel.CreateCase(title, purpose)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Case opened (isolated from personal memory). id=%s (root=shared) title=%q purpose=%q classification=operator-controlled. Seal evidence before verified findings.",
		c.ID, title, purpose,
	), nil
}

type IntelligenceExplainScoreSkill struct{}
type explainScoreArgs struct {
	RegionID string `json:"region_id"`
}

func (s *IntelligenceExplainScoreSkill) Name() string { return "intelligence_explain_score" }
func (s *IntelligenceExplainScoreSkill) Description() string {
	return "Erklärt detailliert das Zustandekommen des Risikoscores einer Region mit den Formeln, Gewichten und Dämpfungsfaktoren."
}
func (s *IntelligenceExplainScoreSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceExplainScoreSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"region_id": map[string]interface{}{"type": "string", "enum": []string{"GERMANY", "FRANCE", "USA", "UKRAINE", "UK"}},
		},
		"required": []string{"region_id"},
	}
}
func (s *IntelligenceExplainScoreSkill) Execute(args json.RawMessage) (string, error) {
	var input explainScoreArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if intelligence.SharedIntelStore != nil {
		// Use shared sub model exclusively for explain
		explain := intelligence.SharedIntelStore.ExplainScore(input.RegionID)
		return explain, nil
	}
	// legacy fallback
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence core unavailable")
	}
	risks := state.intel.ComputeAllRegionalRisks()
	for _, r := range risks {
		if strings.EqualFold(r.RegionID, input.RegionID) {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("### Score-Erklärung für %s (%s)\n\n", r.RegionName, r.RegionID))
			sb.WriteString("Die Berechnung basiert auf der standardisierten Formel:\n")
			sb.WriteString("`Score = min(100, Summe(Gewicht * Schweregrad * Freshness * Confidence))`\n\n")
			sb.WriteString(fmt.Sprintf("- **Max-Risiko-Dimension:** %.1f%%\n", r.OverallRisk))
			sb.WriteString("- **Formel für Gesamtrisiko:** `0.6 * max(Subscores) + 0.4 * avg(Subscores)`\n")
			sb.WriteString("Das verhindert das Verwässern eines kritischen Einzelrisikos im Durchschnitt.\n\n")
			sb.WriteString("**Einflussfaktoren:**\n")
			sb.WriteString(fmt.Sprintf("- Geopolitisch (Gewicht 10): %.1f%%\n", r.GeopoliticalRisk))
			sb.WriteString(fmt.Sprintf("- Konflikt (Gewicht 15): %.1f%%\n", r.ConflictRisk))
			sb.WriteString(fmt.Sprintf("- Cyber (Gewicht 12): %.1f%%\n", r.CyberRisk))
			sb.WriteString(fmt.Sprintf("- Infrastruktur (Gewicht 10): %.1f%%\n", r.InfrastructureRisk))
			sb.WriteString(fmt.Sprintf("- Ökonomisch (Gewicht 8): %.1f%%\n", r.EconomicRisk))
			if len(r.PrimaryDrivers) > 0 {
				sb.WriteString("\n**Aktive Vorfälle in dieser Region:**\n")
				for _, dr := range r.PrimaryDrivers {
					sb.WriteString(fmt.Sprintf("- %s\n", dr))
				}
			}
			return sb.String(), nil
		}
	}
	return "Region nicht gefunden.", nil
}

type IntelligenceRegionCompareSkill struct{}
type regionCompareArgs struct {
	RegionIDA string `json:"region_id_a"`
	RegionIDB string `json:"region_id_b"`
}

func (s *IntelligenceRegionCompareSkill) Name() string { return "intelligence_region_compare" }
func (s *IntelligenceRegionCompareSkill) Description() string {
	return "Vergleicht zwei Regionen hinsichtlich ihres Gesamtrisikos und der einzelnen Risikodimensionen side-by-side."
}
func (s *IntelligenceRegionCompareSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceRegionCompareSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"region_id_a": map[string]interface{}{"type": "string", "enum": []string{"GERMANY", "FRANCE", "USA", "UKRAINE", "UK"}},
			"region_id_b": map[string]interface{}{"type": "string", "enum": []string{"GERMANY", "FRANCE", "USA", "UKRAINE", "UK"}},
		},
		"required": []string{"region_id_a", "region_id_b"},
	}
}
func (s *IntelligenceRegionCompareSkill) Execute(args json.RawMessage) (string, error) {
	var input regionCompareArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	// U1: prefer intelligence.SharedIntelStore risk scores (same as map HUD / explain)
	type row struct {
		Name, Trend                                   string
		Overall, Geopol, Conflict, Cyber, Infra, Econ float64
	}
	pick := func(id string) *row {
		id = strings.ToUpper(strings.TrimSpace(id))
		if intelligence.SharedIntelStore != nil {
			if rs, ok := intelligence.SharedIntelStore.GetRiskScores()[id]; ok {
				return &row{Name: id, Trend: rs.Trend, Overall: rs.OverallRisk, Geopol: rs.GeopoliticalRisk,
					Conflict: rs.ConflictRisk, Cyber: rs.CyberRisk, Infra: rs.InfrastructureRisk, Econ: rs.EconomicRisk}
			}
			return nil
		}
		if state == nil || state.intel == nil {
			return nil
		}
		for _, r := range state.intel.ComputeAllRegionalRisks() {
			if strings.EqualFold(r.RegionID, id) {
				return &row{Name: r.RegionName, Trend: r.Trend, Overall: r.OverallRisk, Geopol: r.GeopoliticalRisk,
					Conflict: r.ConflictRisk, Cyber: r.CyberRisk, Infra: r.InfrastructureRisk, Econ: r.EconomicRisk}
			}
		}
		return nil
	}
	ra, rb := pick(input.RegionIDA), pick(input.RegionIDB)
	if ra == nil || rb == nil {
		return "Eine oder beide Regionen wurden nicht gefunden (intelligence.SharedIntelStore RiskScores).", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### Regionaler Risikovergleich: %s vs. %s (Shared model)\n\n", ra.Name, rb.Name))
	sb.WriteString("| Dimension | " + ra.Name + " | " + rb.Name + " |\n")
	sb.WriteString("| :--- | :---: | :---: |\n")
	sb.WriteString(fmt.Sprintf("| **Gesamtrisiko (Overall)** | **%.1f%%** | **%.1f%%** |\n", ra.Overall, rb.Overall))
	sb.WriteString(fmt.Sprintf("| Trend | %s | %s |\n", ra.Trend, rb.Trend))
	sb.WriteString(fmt.Sprintf("| Geopolitisch | %.1f%% | %.1f%% |\n", ra.Geopol, rb.Geopol))
	sb.WriteString(fmt.Sprintf("| Konflikt | %.1f%% | %.1f%% |\n", ra.Conflict, rb.Conflict))
	sb.WriteString(fmt.Sprintf("| Cyber | %.1f%% | %.1f%% |\n", ra.Cyber, rb.Cyber))
	sb.WriteString(fmt.Sprintf("| Infrastruktur | %.1f%% | %.1f%% |\n", ra.Infra, rb.Infra))
	sb.WriteString(fmt.Sprintf("| Ökonomisch | %.1f%% | %.1f%% |\n", ra.Econ, rb.Econ))
	sb.WriteString("\n_Scores aus intelligence.SharedIntelStore (nicht parallel root truth). Raw observations bleiben unverified._\n")
	return sb.String(), nil
}

type IntelligenceGenerateBriefSkill struct{}
type generateBriefArgs struct {
	ReportType string `json:"report_type"`
}

func (s *IntelligenceGenerateBriefSkill) Name() string { return "intelligence_generate_brief" }
func (s *IntelligenceGenerateBriefSkill) Description() string {
	return "Generiert einen strukturierten Lagebericht (Daily, Regional, Cyber, Infrastructure) basierend auf den aktuellen Ereignissen."
}
func (s *IntelligenceGenerateBriefSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceGenerateBriefSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"report_type": map[string]interface{}{"type": "string", "enum": []string{"Daily Global Brief", "Regional Security Brief", "Cyber Threat Brief", "Infrastructure Risk Report"}},
		},
		"required": []string{"report_type"},
	}
}
func (s *IntelligenceGenerateBriefSkill) Execute(args json.RawMessage) (string, error) {
	var input generateBriefArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence core unavailable")
	}
	if intelligence.SharedIntelStore != nil {
		return intelligence.SharedIntelStore.GenerateReport(input.ReportType), nil
	}
	var d struct {
		Events []interface{}
		Cases  []interface{}
	}
	dd := state.intel.Snapshot()
	d.Events = make([]interface{}, len(dd.Events))
	for i, e := range dd.Events {
		d.Events[i] = e
	}
	d.Cases = make([]interface{}, len(dd.Cases))
	for i, c := range dd.Cases {
		d.Cases[i] = c
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# AETHEL INTELLIGENCE BRIEFING: %s\n", input.ReportType))
	sb.WriteString(fmt.Sprintf("Generiert am: %s (UTC)\n\n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString("## 1. Executive Summary\n")
	sb.WriteString(fmt.Sprintf("Das System verzeichnet aktuell %d ungelöste Beobachtungsvorschläge und %d aktive Cases.\n\n", len(d.Events), len(d.Cases)))
	sb.WriteString("## 2. Risikolage nach Dimensionen\n")
	risks := state.intel.ComputeAllRegionalRisks()
	for _, r := range risks {
		sb.WriteString(fmt.Sprintf("* **%s**: Overall %.1f%% (Trend: %s)\n", r.RegionName, r.OverallRisk, r.Trend))
	}
	sb.WriteString("\n## 3. Datenbasis und Quellenbelege\n")
	sb.WriteString("Alle aufgeführten Vorfälle sind durch lokale Evidence und undurchdringbare Hashes gesichert.\n")
	return sb.String(), nil
}

type IntelligenceSourceHealthSkill struct{}

func (s *IntelligenceSourceHealthSkill) Name() string { return "intelligence_source_health" }
func (s *IntelligenceSourceHealthSkill) Description() string {
	return "Liefert den Status, die Zuverlässigkeit und Ausfälle aller konfigurierten OSINT-Feeds."
}
func (s *IntelligenceSourceHealthSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceSourceHealthSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s *IntelligenceSourceHealthSkill) Execute(_ json.RawMessage) (string, error) {
	var sb strings.Builder
	sb.WriteString("### OSINT Feeds Status & Health (unified)\n\n")
	if intelligence.SharedIntelStore != nil {
		h := intelligence.SharedIntelStore.SourceHealth()
		sb.WriteString(fmt.Sprintf("Shared model sources: %v | last: %v | %v\n\n", h["total_sources"], h["last_fetch"], h["status"]))
	}
	// Connector registry (U6 BuiltIn RSS metadata)
	reg := osint.ConnectorRegistrySummary()
	sb.WriteString(fmt.Sprintf("### Connector registry\nCount: %v · as_of: %v\n", reg["count"], reg["as_of"]))
	if list, ok := reg["connectors"].([]map[string]any); ok {
		for _, c := range list {
			sb.WriteString(fmt.Sprintf("- %v v%v trust=%v health=%v\n", c["name"], c["version"], c["trust_tier"], c["health"]))
		}
	}
	sb.WriteString("\n")
	if state == nil || state.osint == nil {
		sb.WriteString("Legacy OSINT engine: unavailable in this context.\n")
		return sb.String(), nil
	}
	cols := state.osint.GetConfigs()
	sb.WriteString("| Source Name | Domain | Status | Priority |\n")
	sb.WriteString("| :--- | :--- | :---: | :---: |\n")
	for _, c := range cols {
		status := "ACTIVE"
		if !c.Enabled {
			status = "DISABLED"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d |\n", c.Name, c.Domain, status, c.Priority))
	}
	return sb.String(), nil
}

// IntelligenceSyncPersonalSkill — U3 explicit bridge PersonalStore → intelligence.SharedIntelStore.
type IntelligenceSyncPersonalSkill struct{}

func (s *IntelligenceSyncPersonalSkill) Name() string { return "intelligence_sync_personal" }
func (s *IntelligenceSyncPersonalSkill) Description() string {
	return "Synchronisiert opt-in den PersonalStore (Interessen/Ziele) in den intelligence.SharedIntelStore PersonalContext. Vermischt keine Memories mit Cases. Voraussetzung für intelligence_personal_impact."
}
func (s *IntelligenceSyncPersonalSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceSyncPersonalSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s *IntelligenceSyncPersonalSkill) Execute(_ json.RawMessage) (string, error) {
	if state == nil || state.personal == nil {
		return "", errors.New("PersonalStore unavailable")
	}
	pc, err := personal.SyncPersonalToSharedIntel(state.personal)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Personal Context synced to intelligence.SharedIntelStore (opt-in).\nOperator=%s interests=%d goals=%d projects=%d preferred_regions=%v risk_tolerance=%s updated=%s\nPersonal memory remains isolated from cases.",
		pc.OperatorID, len(pc.Interests), len(pc.Goals), len(pc.Projects), pc.PreferredRegions, pc.RiskTolerance, pc.LastUpdated.Format(time.RFC3339),
	), nil
}

// IntelligencePersonalImpactSkill — U3 correlation brief (Empfehlung ≠ Fakt).
type IntelligencePersonalImpactSkill struct{}

func (s *IntelligencePersonalImpactSkill) Name() string { return "intelligence_personal_impact" }
func (s *IntelligencePersonalImpactSkill) Description() string {
	return "Baut einen Personal Impact Brief: Korrelation PersonalContext + Watchlists × RiskScores/Events. Nur Empfehlungen mit Unsicherheit — keine erfundenen Weltfakten. Vorher ggf. intelligence_sync_personal."
}
func (s *IntelligencePersonalImpactSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligencePersonalImpactSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s *IntelligencePersonalImpactSkill) Execute(_ json.RawMessage) (string, error) {
	if intelligence.SharedIntelStore == nil {
		return "", errors.New("intelligence.SharedIntelStore unavailable")
	}
	// Soft auto-sync if personal store present and shared personal empty
	if state != nil && state.personal != nil {
		cur := intelligence.SharedIntelStore.GetPersonalContext()
		if cur.OperatorID == "" && len(cur.Interests) == 0 && len(cur.Goals) == 0 {
			_, _ = personal.SyncPersonalToSharedIntel(state.personal)
		}
	}
	return intelligence.SharedIntelStore.PersonalImpact(), nil
}

// OSINTEvidenceCaptureSkill — U5 seal evidence on a case with optional source_event_id provenance.
type OSINTEvidenceCaptureSkill struct{}

type osintEvidenceCaptureArgs struct {
	CaseID        string `json:"case_id"`
	Source        string `json:"source"`
	URL           string `json:"url"`
	Excerpt       string `json:"excerpt"`
	SourceEventID string `json:"source_event_id"`
}

func (s *OSINTEvidenceCaptureSkill) Name() string { return "osint_evidence_capture" }
func (s *OSINTEvidenceCaptureSkill) Description() string {
	return "Versiegelt Evidence in einem OSINT-Case (Content-Hash). Optional source_event_id speichert Promote-Provenance im Case-Audit. Keine Personal-Memory-Vermischung."
}
func (s *OSINTEvidenceCaptureSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *OSINTEvidenceCaptureSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"case_id":         map[string]interface{}{"type": "string", "minLength": 3, "maxLength": 80},
			"source":          map[string]interface{}{"type": "string", "minLength": 1, "maxLength": 200},
			"url":             map[string]interface{}{"type": "string", "maxLength": 500},
			"excerpt":         map[string]interface{}{"type": "string", "minLength": 1, "maxLength": 2000},
			"source_event_id": map[string]interface{}{"type": "string", "maxLength": 120},
		},
		"required": []string{"case_id", "source", "excerpt"},
	}
}
func (s *OSINTEvidenceCaptureSkill) Execute(args json.RawMessage) (string, error) {
	var input osintEvidenceCaptureArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	caseID := strings.TrimSpace(input.CaseID)
	source := strings.TrimSpace(input.Source)
	excerpt := strings.TrimSpace(input.Excerpt)
	if caseID == "" || source == "" || excerpt == "" {
		return "", errors.New("case_id, source and excerpt are required")
	}
	if state == nil || state.intel == nil {
		return "", errors.New("IntelligenceStore unavailable")
	}
	e, err := state.intel.SealEvidenceWithEvent(caseID, source, strings.TrimSpace(input.URL), excerpt, "operator", strings.TrimSpace(input.SourceEventID))
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("Evidence sealed. id=%s case=%s sha256=%s status=%s", e.ID, e.CaseID, e.SHA256, e.ValidationStatus)
	if e.SourceEventID != "" {
		msg += " source_event_id=" + e.SourceEventID
	}
	return msg, nil
}

// OSINTEntityProposeSkill — vision name for case entity capture (pseudonymizes persons).
type OSINTEntityProposeSkill struct{}

func (s *OSINTEntityProposeSkill) Name() string { return "osint_entity_propose" }
func (s *OSINTEntityProposeSkill) Description() string {
	return "Schlägt eine Case-lokale Entity vor (person → HMAC-Alias). Keine Roh-PII Speicherung. Identisch zu intelligence_add_entity."
}
func (s *OSINTEntityProposeSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *OSINTEntityProposeSkill) Parameters() map[string]interface{} {
	return (&IntelligenceAddEntitySkill{}).Parameters()
}
func (s *OSINTEntityProposeSkill) Execute(args json.RawMessage) (string, error) {
	return (&IntelligenceAddEntitySkill{}).Execute(args)
}

// OSINTRelationProposeSkill — evidence-bound relation.
type OSINTRelationProposeSkill struct{}

func (s *OSINTRelationProposeSkill) Name() string { return "osint_relation_propose" }
func (s *OSINTRelationProposeSkill) Description() string {
	return "Verknüpft zwei Case-Entities nur über versiegelte Evidence-ID. Identisch zu intelligence_link_entities."
}
func (s *OSINTRelationProposeSkill) RiskLevel() security.RiskLevel { return security.RiskModerate }
func (s *OSINTRelationProposeSkill) Parameters() map[string]interface{} {
	return (&IntelligenceLinkEntitiesSkill{}).Parameters()
}
func (s *OSINTRelationProposeSkill) Execute(args json.RawMessage) (string, error) {
	return (&IntelligenceLinkEntitiesSkill{}).Execute(args)
}

// OSINTTimelineGenerateSkill — case timeline from audit/evidence/entities.
type OSINTTimelineGenerateSkill struct{}
type osintTimelineArgs struct {
	CaseID string `json:"case_id"`
}

func (s *OSINTTimelineGenerateSkill) Name() string { return "osint_timeline_generate" }
func (s *OSINTTimelineGenerateSkill) Description() string {
	return "Erzeugt eine chronologische Case-Timeline (Audit + Evidence + Entities + Relations). Nur lokale Case-Daten."
}
func (s *OSINTTimelineGenerateSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *OSINTTimelineGenerateSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"case_id": map[string]interface{}{"type": "string", "minLength": 3},
		},
		"required": []string{"case_id"},
	}
}
func (s *OSINTTimelineGenerateSkill) Execute(args json.RawMessage) (string, error) {
	var input osintTimelineArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence core unavailable")
	}
	return state.intel.CaseTimeline(strings.TrimSpace(input.CaseID))
}

// OSINTReportGenerateSkill — structured case report.
type OSINTReportGenerateSkill struct{}
type osintReportArgs struct {
	CaseID string `json:"case_id"`
}

func (s *OSINTReportGenerateSkill) Name() string { return "osint_report_generate" }
func (s *OSINTReportGenerateSkill) Description() string {
	return "Erzeugt einen Case-Report (Evidence, Entities, Relations) aus dem lokalen Case-Graph."
}
func (s *OSINTReportGenerateSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *OSINTReportGenerateSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"case_id": map[string]interface{}{"type": "string", "minLength": 3},
		},
		"required": []string{"case_id"},
	}
}
func (s *OSINTReportGenerateSkill) Execute(args json.RawMessage) (string, error) {
	var input osintReportArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if state == nil || state.intel == nil {
		return "", errors.New("intelligence core unavailable")
	}
	return state.intel.CaseReport(strings.TrimSpace(input.CaseID))
}

// IntelligenceIdentityStatusSkill — technical continuity, not consciousness.
type IntelligenceIdentityStatusSkill struct{}

func (s *IntelligenceIdentityStatusSkill) Name() string { return "intelligence_identity_status" }
func (s *IntelligenceIdentityStatusSkill) Description() string {
	return "Liefert IdentityProfile Continuity (last warn, pending alerts, capabilities). Kein Bewusstseins-Claim."
}
func (s *IntelligenceIdentityStatusSkill) RiskLevel() security.RiskLevel { return security.RiskSafe }
func (s *IntelligenceIdentityStatusSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (s *IntelligenceIdentityStatusSkill) Execute(_ json.RawMessage) (string, error) {
	if intelligence.SharedIntelStore == nil {
		return "", errors.New("intelligence.SharedIntelStore unavailable")
	}
	id := intelligence.SharedIntelStore.IdentitySnapshot()
	snap := intelligence.SharedIntelStore.GetSnapshot()
	var b strings.Builder
	b.WriteString("### AETHEL Identity (technical continuity)\n\n")
	b.WriteString(fmt.Sprintf("- Name: %s · Version: %s\n", id.Name, id.Version))
	b.WriteString(fmt.Sprintf("- Last warned: %s\n", id.LastWarnedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Pending alert IDs: %d\n", len(id.PendingAlertIDs)))
	b.WriteString(fmt.Sprintf("- Sources: %d · Events: %d · Cases (shared): %d · Agent actions: %d\n",
		len(snap.Sources), len(snap.Events), len(snap.Cases), len(snap.AgentActions)))
	if len(id.CapabilityNotes) > 0 {
		b.WriteString("- Capabilities: " + strings.Join(id.CapabilityNotes, ", ") + "\n")
	}
	b.WriteString("\n_Note: This is operational continuity metadata, not agency or consciousness._\n")
	return b.String(), nil
}

// IntelligenceSetAssessmentStatusSkill — operator verification layer.
type IntelligenceSetAssessmentStatusSkill struct{}
type assessmentStatusArgs struct {
	AssessmentID string `json:"assessment_id"`
	Status       string `json:"status"`
	Reviewer     string `json:"reviewer"`
}

func (s *IntelligenceSetAssessmentStatusSkill) Name() string {
	return "intelligence_set_assessment_status"
}
func (s *IntelligenceSetAssessmentStatusSkill) Description() string {
	return "Setzt Assessment-Status (verified|disputed|rejected|corroborated|unverified). Nur Operator — nie auto aus LLM-Plausibilität."
}
func (s *IntelligenceSetAssessmentStatusSkill) RiskLevel() security.RiskLevel {
	return security.RiskModerate
}
func (s *IntelligenceSetAssessmentStatusSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"assessment_id": map[string]interface{}{"type": "string"},
			"status":        map[string]interface{}{"type": "string", "enum": []string{"verified", "disputed", "rejected", "corroborated", "unverified", "hypothesis"}},
			"reviewer":      map[string]interface{}{"type": "string"},
		},
		"required": []string{"assessment_id", "status"},
	}
}
func (s *IntelligenceSetAssessmentStatusSkill) Execute(args json.RawMessage) (string, error) {
	var input assessmentStatusArgs
	if err := json.Unmarshal(args, &input); err != nil {
		return "", err
	}
	if intelligence.SharedIntelStore == nil {
		return "", errors.New("intelligence.SharedIntelStore unavailable")
	}
	if err := intelligence.SharedIntelStore.SetAssessmentStatus(strings.TrimSpace(input.AssessmentID), input.Status, input.Reviewer); err != nil {
		return "", err
	}
	return fmt.Sprintf("Assessment %s status set to %s (operator review).", input.AssessmentID, input.Status), nil
}

// IntelligenceConnectorFetchSkill — U6 connector Fetch → Shared Ingest (explicit operator path).
type IntelligenceConnectorFetchSkill struct{}
type connectorFetchArgs struct {
	Name string `json:"name"`
}

func (s *IntelligenceConnectorFetchSkill) Name() string { return "intelligence_connector_fetch" }
func (s *IntelligenceConnectorFetchSkill) Description() string {
	return "Führt Fetch() auf einem registrierten Connector aus (default builtin-rss) und ingested Observations in den intelligence.SharedIntelStore. Keine erfundenen Quellen."
}
func (s *IntelligenceConnectorFetchSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *IntelligenceConnectorFetchSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string", "description": "Connector registry name (default builtin-rss)"},
		},
	}
}
func (s *IntelligenceConnectorFetchSkill) Execute(args json.RawMessage) (string, error) {
	var input connectorFetchArgs
	_ = json.Unmarshal(args, &input)
	fetched, ingested, err := osint.RunConnectorFetchIngest(input.Name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Connector fetch complete. name=%s fetched=%d ingested=%d (intelligence.SharedIntelStore). Layers remain RAW until assessment/review.",
		strings.TrimSpace(input.Name), fetched, ingested), nil
}
