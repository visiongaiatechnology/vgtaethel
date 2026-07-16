package intelligence

// STATUS: PLATIN
// Local-first Intelligence Core. It deliberately stores no raw personal identifiers:
// entities are case-scoped HMAC aliases and evidence is immutable after sealing.

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go-aethel/security"
)

// pseudonymEntity creates a stable, case-scoped pseudonym (GaiasEye-style HMAC).
// Never reveals real identifiers across cases. Used for persons and sensitive entities.
func pseudonymEntity(caseSecret []byte, normalized string) string {
	if len(caseSecret) == 0 {
		return ""
	}
	normalized = strings.TrimSpace(strings.ToLower(normalized))
	mac := hmac.New(sha256.New, caseSecret)
	mac.Write([]byte(normalized))
	sum := mac.Sum(nil)
	return "GX-PER-" + strings.ToUpper(hex.EncodeToString(sum[:12]))
}

type IntelligenceEvent struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Summary    string    `json:"summary"`
	Source     string    `json:"source"`
	SourceURL  string    `json:"source_url,omitempty"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Severity   string    `json:"severity"`
	Confidence int       `json:"confidence"`
	ObservedAt time.Time `json:"observed_at"`
	Status     string    `json:"status"`
}
type IntelligenceEvidence struct {
	ID               string    `json:"id"`
	CaseID           string    `json:"case_id"`
	Source           string    `json:"source"`
	URL              string    `json:"url,omitempty"`
	Excerpt          string    `json:"excerpt"`
	SHA256           string    `json:"sha256"`
	CollectedAt      time.Time `json:"collected_at"`
	Sealed           bool      `json:"sealed"`
	ValidatedBy      string    `json:"validated_by,omitempty"`
	ValidationStatus string    `json:"validation_status"`
	SourceEventID    string    `json:"source_event_id,omitempty"` // promote provenance (Shared/Root event id)
}
type IntelligenceEntity struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	Confidence int    `json:"confidence"`
}
type IntelligenceRelation struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Type       string `json:"type"`
	EvidenceID string `json:"evidence_id"`
	Confidence int    `json:"confidence"`
}
type IntelligenceCase struct {
	ID             string                    `json:"id"`
	Title          string                    `json:"title"`
	Purpose        string                    `json:"purpose"`
	Classification string                    `json:"classification"`
	Status         string                    `json:"status"`
	CreatedAt      time.Time                 `json:"created_at"`
	Evidence       []IntelligenceEvidence    `json:"evidence"`
	Entities       []IntelligenceEntity      `json:"entities"`
	Relations      []IntelligenceRelation    `json:"relations"`
	Audit          []IntelligenceAuditEntry  `json:"audit"`
	ReIDRequests   []IntelligenceReIDRequest `json:"reid_requests,omitempty"`
}

// IntelligenceReIDRequest is a dual-approval, time-bounded unlock request.
// Raw PII is never stored — unlock only expands case-scoped alias metadata visibility.
type IntelligenceReIDRequest struct {
	ID             string    `json:"id"`
	Purpose        string    `json:"purpose"`
	RequestedBy    string    `json:"requested_by"`
	FirstApprover  string    `json:"first_approver,omitempty"`
	SecondApprover string    `json:"second_approver,omitempty"`
	RequestedAt    time.Time `json:"requested_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	Status         string    `json:"status"` // requested | approved_once | unlocked | expired | denied
	Unlocked       bool      `json:"unlocked"`
}
type IntelligenceAuditEntry struct {
	At     time.Time `json:"at"`
	Action string    `json:"action"`
	Actor  string    `json:"actor"`
	Detail string    `json:"detail"`
}
type IntelligenceCorrelation struct {
	EventIDs []string `json:"event_ids"`
	Label    string   `json:"label"`
	Strength int      `json:"strength"`
	Reason   string   `json:"reason"`
}
type IntelligenceAlert struct {
	EventID    string    `json:"event_id"`
	Title      string    `json:"title"`
	Severity   string    `json:"severity"`
	Confidence int       `json:"confidence"`
	Reason     string    `json:"reason"`
	ObservedAt time.Time `json:"observed_at"`
}

var ErrDuplicateObservation = errors.New("observation already exists")

type NeuralCoreEvaluation struct {
	WorldState  string    `json:"world_state"`
	LocalState  string    `json:"local_state"`
	LastUpdated time.Time `json:"last_updated"`
}

type intelligenceState struct {
	Events     []IntelligenceEvent  `json:"events"`
	Cases      []IntelligenceCase   `json:"cases"`
	Revision   uint64               `json:"revision"`
	Evaluation NeuralCoreEvaluation `json:"evaluation"`
}
type IntelligenceStore struct {
	mu   sync.RWMutex
	path string
	data intelligenceState
	bus  *IntelligenceBus

	ChatEvaluator func(systemPrompt, userPrompt string) (string, error)
}

var intelligenceSequence uint64

func NewIntelligenceStore(path string) *IntelligenceStore {
	s := &IntelligenceStore{path: path, bus: NewIntelligenceBus()}
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &s.data)
	}
	if s.data.Events == nil {
		s.data.Events = []IntelligenceEvent{}
	}
	if s.data.Cases == nil {
		s.data.Cases = []IntelligenceCase{}
	}
	return s
}
func (s *IntelligenceStore) commitLocked(kind, subject string) error {
	err := s.saveLocked()
	if err == nil && s.bus != nil {
		s.bus.Publish(IntelligenceBusEvent{Type: kind, SubjectID: subject, Revision: s.data.Revision, At: time.Now().UTC()})
	}
	return err
}
func (s *IntelligenceStore) saveLocked() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".intel-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(raw)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, s.path)
}
func intelID(prefix string) string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		// A non-secret identifier must remain collision-resistant even if the OS RNG is temporarily unavailable.
		sum := sha256.Sum256([]byte(time.Now().UTC().Format(time.RFC3339Nano) + ":" + strconv.FormatUint(atomic.AddUint64(&intelligenceSequence, 1), 10)))
		copy(buf, sum[:len(buf)])
	}
	return prefix + "_" + hex.EncodeToString(buf)
}
func (s *IntelligenceStore) Snapshot() intelligenceState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.data
	return out
}
func (s *IntelligenceStore) CreateCase(title, purpose string) (IntelligenceCase, error) {
	return s.CreateCaseWithID("", title, purpose)
}

// CreateCaseWithID opens a root case. When id is empty a new id is generated.
// Mirrors into SharedIntelStore under the SAME id (U5 case graph alignment).
func (s *IntelligenceStore) CreateCaseWithID(id, title, purpose string) (IntelligenceCase, error) {
	title = strings.TrimSpace(title)
	purpose = strings.TrimSpace(purpose)
	if title == "" || purpose == "" {
		return IntelligenceCase{}, errors.New("title and purpose are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if strings.TrimSpace(id) == "" {
		id = intelID("case")
	} else {
		id = strings.TrimSpace(id)
		for _, existing := range s.data.Cases {
			if existing.ID == id {
				return IntelligenceCase{}, errors.New("case id already exists")
			}
		}
	}
	c := IntelligenceCase{
		ID: id, Title: title, Purpose: purpose, Classification: "operator-controlled", Status: "open",
		CreatedAt: now, Evidence: []IntelligenceEvidence{}, Entities: []IntelligenceEntity{},
		Relations: []IntelligenceRelation{}, ReIDRequests: []IntelligenceReIDRequest{},
		Audit: []IntelligenceAuditEntry{{At: now, Action: "case.created", Actor: "operator", Detail: "Case opened with stated purpose."}},
	}
	s.data.Cases = append(s.data.Cases, c)
	s.data.Revision++
	if err := s.commitLocked("case.created", c.ID); err != nil {
		return IntelligenceCase{}, err
	}
	// U5: same case id in SharedIntelStore (shell for chat/map linkage; graph authority remains root UI)
	if SharedIntelStore != nil {
		_, _ = SharedIntelStore.CreateCaseWithID(c.ID, title, purpose)
	}
	return c, nil
}

// DeleteCase removes a case and all nested evidence/entities/relations (local-first, audited).
func (s *IntelligenceStore) DeleteCase(caseID string) error {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" {
		return errors.New("case id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i, c := range s.data.Cases {
		if c.ID == caseID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return errors.New("case not found")
	}
	s.data.Cases = append(s.data.Cases[:idx], s.data.Cases[idx+1:]...)
	s.data.Revision++
	return s.commitLocked("case.deleted", caseID)
}

func (s *IntelligenceStore) ProposeEvent(event IntelligenceEvent) error {
	event.Title = strings.TrimSpace(event.Title)
	event.Source = strings.TrimSpace(event.Source)
	if event.Title == "" || event.Source == "" {
		return errors.New("event title and source are required")
	}
	if event.Confidence < 0 || event.Confidence > 100 || event.Latitude < -90 || event.Latitude > 90 || event.Longitude < -180 || event.Longitude > 180 {
		return errors.New("event coordinates or confidence are invalid")
	}
	if event.ID == "" {
		event.ID = intelID("obs")
	}
	if event.ObservedAt.IsZero() {
		event.ObservedAt = time.Now().UTC()
	}
	event.Status = "proposed"

	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Strict duplicate check first (exact match of Source, SourceURL, and Title)
	for _, existing := range s.data.Events {
		if strings.EqualFold(existing.Source, event.Source) &&
			strings.EqualFold(strings.TrimSpace(existing.SourceURL), strings.TrimSpace(event.SourceURL)) &&
			strings.EqualFold(existing.Title, event.Title) {
			return ErrDuplicateObservation
		}
	}

	// 2. Semantic clustering check next (overlapping updates from same source/URL)
	for idx, existing := range s.data.Events {
		timeDiff := event.ObservedAt.Sub(existing.ObservedAt)
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		if timeDiff < 30*time.Minute {
			sameURL := event.SourceURL != "" && strings.EqualFold(strings.TrimSpace(event.SourceURL), strings.TrimSpace(existing.SourceURL))
			sameSource := strings.EqualFold(event.Source, existing.Source)

			if sameURL || (sameSource && (titlesAreSimilar(event.Title, existing.Title) || strings.EqualFold(event.Title, existing.Title))) {
				dist := haversineDistance(event.Latitude, event.Longitude, existing.Latitude, existing.Longitude)
				if dist < 50.0 {
					log.Printf("[Intelligence] Clustered incoming event '%s' into existing event '%s' (dist: %.2f km)", event.Title, existing.Title, dist)
					if event.Confidence > existing.Confidence {
						s.data.Events[idx].Confidence = event.Confidence
					}
					if !strings.Contains(existing.Source, event.Source) {
						s.data.Events[idx].Source = existing.Source + ", " + event.Source
					}
					s.data.Revision++
					_ = s.saveLocked()
					return nil // Clustered, do not add duplicate event
				}
			}
		}
	}

	s.data.Events = append(s.data.Events, event)
	sort.Slice(s.data.Events, func(i, j int) bool { return s.data.Events[i].ObservedAt.After(s.data.Events[j].ObservedAt) })
	s.data.Revision++
	if err := s.commitLocked("observation.proposed", event.ID); err != nil {
		return err
	}
	if s.bus != nil && intelligenceAlertEligible(event) {
		s.bus.PublishGlobalWatchAlert(IntelligenceAlert{
			EventID: event.ID, Title: event.Title, Severity: event.Severity, Confidence: event.Confidence,
			Reason: "confidence and severity threshold reached", ObservedAt: event.ObservedAt,
		}, s.data.Revision)
	}
	return nil
}

func intelligenceAlertEligible(event IntelligenceEvent) bool {
	return event.Confidence >= 75 && (event.Severity == "high" || event.Severity == "medium")
}
func (s *IntelligenceStore) SealEvidence(caseID, source, url, excerpt, operator string) (IntelligenceEvidence, error) {
	return s.SealEvidenceWithEvent(caseID, source, url, excerpt, operator, "")
}

// SealEvidenceWithEvent seals evidence and optionally records promote provenance (source_event_id) in audit.
func (s *IntelligenceStore) SealEvidenceWithEvent(caseID, source, url, excerpt, operator, sourceEventID string) (IntelligenceEvidence, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ci := range s.data.Cases {
		if s.data.Cases[ci].ID == caseID {
			b := []byte(strings.TrimSpace(source) + "\n" + strings.TrimSpace(url) + "\n" + strings.TrimSpace(excerpt))
			if len(b) == 0 {
				return IntelligenceEvidence{}, errors.New("evidence content is required")
			}
			sum := sha256.Sum256(b)
			sourceEventID = strings.TrimSpace(sourceEventID)
			e := IntelligenceEvidence{
				ID: intelID("ev"), CaseID: caseID, Source: source, URL: url, Excerpt: excerpt,
				SHA256: hex.EncodeToString(sum[:]), CollectedAt: time.Now().UTC(), Sealed: true,
				ValidationStatus: "pending", SourceEventID: sourceEventID,
			}
			s.data.Cases[ci].Evidence = append(s.data.Cases[ci].Evidence, e)
			detail := e.ID
			if sourceEventID != "" {
				detail = e.ID + " source_event_id=" + sourceEventID
				s.data.Cases[ci].Audit = append(s.data.Cases[ci].Audit, IntelligenceAuditEntry{
					At: time.Now().UTC(), Action: "event.promoted", Actor: operator, Detail: "source_event_id=" + sourceEventID + " evidence_id=" + e.ID,
				})
			}
			s.data.Cases[ci].Audit = append(s.data.Cases[ci].Audit, IntelligenceAuditEntry{At: time.Now().UTC(), Action: "evidence.sealed", Actor: operator, Detail: detail})
			s.data.Revision++
			if err := s.commitLocked("evidence.sealed", e.ID); err != nil {
				return IntelligenceEvidence{}, err
			}
			// U5/U6: shared case graph gets the same sealed evidence (authority alignment)
			if SharedIntelStore != nil {
				_, _ = SharedIntelStore.SealCaseEvidence(caseID, e.ID, source, url, excerpt, e.SHA256, sourceEventID)
			}
			return e, nil
		}
	}
	return IntelligenceEvidence{}, errors.New("case not found")
}
func (s *IntelligenceStore) ValidateEvidence(caseID, evidenceID, operator, decision string) (IntelligenceEvidence, error) {
	if decision != "verified" && decision != "disputed" && decision != "rejected" {
		return IntelligenceEvidence{}, errors.New("validation decision is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ci := range s.data.Cases {
		if s.data.Cases[ci].ID != caseID {
			continue
		}
		for ei := range s.data.Cases[ci].Evidence {
			e := &s.data.Cases[ci].Evidence[ei]
			if e.ID != evidenceID {
				continue
			}
			e.ValidationStatus = decision
			e.ValidatedBy = operator
			s.data.Cases[ci].Audit = append(s.data.Cases[ci].Audit, IntelligenceAuditEntry{At: time.Now().UTC(), Action: "evidence." + decision, Actor: operator, Detail: evidenceID})
			s.data.Revision++
			return *e, s.commitLocked("evidence."+decision, evidenceID)
		}
		return IntelligenceEvidence{}, errors.New("evidence not found")
	}
	return IntelligenceEvidence{}, errors.New("case not found")
}
func (s *IntelligenceStore) Correlations() []IntelligenceCorrelation {
	d := s.Snapshot()
	out := []IntelligenceCorrelation{}
	for i := 0; i < len(d.Events); i++ {
		for j := i + 1; j < len(d.Events); j++ {
			shared := sharedIntelTokens(d.Events[i].Title, d.Events[j].Title)
			if len(shared) < 2 {
				continue
			}
			out = append(out, IntelligenceCorrelation{EventIDs: []string{d.Events[i].ID, d.Events[j].ID}, Label: strings.Join(shared[:minIntel(3, len(shared))], " / "), Strength: minIntel(100, len(shared)*25+d.Events[i].Confidence/4+d.Events[j].Confidence/4), Reason: "shared normalized headline terms"})
		}
	}
	return out
}
func (s *IntelligenceStore) Alerts() []IntelligenceAlert {
	d := s.Snapshot()
	out := []IntelligenceAlert{}
	for _, event := range d.Events {
		if intelligenceAlertEligible(event) {
			out = append(out, IntelligenceAlert{EventID: event.ID, Title: event.Title, Severity: event.Severity, Confidence: event.Confidence, Reason: "confidence and severity threshold reached", ObservedAt: event.ObservedAt})
		}
	}
	return out
}
func (s *IntelligenceStore) Briefing() string {
	d := s.Snapshot()
	alerts := s.Alerts()
	var b strings.Builder
	b.WriteString("# AETHEL GLOBAL WATCH BRIEFING\n\n")
	b.WriteString("Generated: " + time.Now().UTC().Format(time.RFC3339) + "\n\n")
	b.WriteString("## Situation\n")
	b.WriteString(strconv.Itoa(len(d.Events)) + " observations, " + strconv.Itoa(len(alerts)) + " threshold alerts, " + strconv.Itoa(len(d.Cases)) + " active cases.\n\n")
	b.WriteString("## Priority alerts\n")
	if len(alerts) == 0 {
		b.WriteString("- No observation currently meets the configured confidence/severity threshold.\n")
	} else {
		for _, a := range alerts {
			b.WriteString("- [" + a.EventID + "] " + a.Title + " (" + a.Severity + ", " + strconv.Itoa(a.Confidence) + "%)\n")
		}
	}
	b.WriteString("\n## Cross-stream correlations\n")
	correlations := s.Correlations()
	if len(correlations) == 0 {
		b.WriteString("- No evidence-grade correlation inferred.\n")
	} else {
		for _, c := range correlations {
			b.WriteString("- " + c.Label + " (strength " + strconv.Itoa(c.Strength) + "%, observations: " + strings.Join(c.EventIDs, ",") + ")\n")
		}
	}
	b.WriteString("\nAll observations remain proposals until a human opens and validates a case.\n")
	return b.String()
}
func (s *IntelligenceStore) ReIDStatus(caseID string) (map[string]any, error) {
	d := s.Snapshot()
	for _, c := range d.Cases {
		if c.ID != caseID {
			continue
		}
		// Expire unlocks past ExpiresAt (lazy)
		s.expireReIDLocked(caseID)
		d = s.Snapshot()
		for _, c2 := range d.Cases {
			if c2.ID != caseID {
				continue
			}
			active := []IntelligenceReIDRequest{}
			for _, r := range c2.ReIDRequests {
				active = append(active, r)
			}
			unlocked := false
			var unlockUntil time.Time
			for _, r := range active {
				if r.Unlocked && r.Status == "unlocked" && time.Now().UTC().Before(r.ExpiresAt) {
					unlocked = true
					unlockUntil = r.ExpiresAt
				}
			}
			return map[string]any{
				"case_id": caseID,
				// Raw PII recovery is never possible — not stored.
				"reidentification":  "not_eligible",
				"reason":            "Aethel stores only case-scoped HMAC aliases for person entities; raw identities are not retained. Dual-approval unlock only expands alias metadata visibility for a time-bound window.",
				"entities":          len(c2.Entities),
				"request_count":     len(active),
				"requests":          active,
				"alias_unlock":      unlocked,
				"unlock_expires_at": unlockUntil,
			}, nil
		}
	}
	return nil, errors.New("case not found")
}

// expireReIDLocked marks expired unlocks (best-effort; may acquire lock).
func (s *IntelligenceStore) expireReIDLocked(caseID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for ci := range s.data.Cases {
		if s.data.Cases[ci].ID != caseID {
			continue
		}
		changed := false
		for ri := range s.data.Cases[ci].ReIDRequests {
			r := &s.data.Cases[ci].ReIDRequests[ri]
			if (r.Status == "unlocked" || r.Unlocked) && now.After(r.ExpiresAt) {
				r.Status = "expired"
				r.Unlocked = false
				changed = true
				s.data.Cases[ci].Audit = append(s.data.Cases[ci].Audit, IntelligenceAuditEntry{
					At: now, Action: "reid.expired", Actor: "system", Detail: r.ID,
				})
			}
		}
		if changed {
			s.data.Revision++
			_ = s.commitLocked("reid.expired", caseID)
		}
		return
	}
}

// RequestReID records a dual-approval Re-ID workflow entry (does not recover raw PII).
func (s *IntelligenceStore) RequestReID(caseID, purpose, actor string) (IntelligenceReIDRequest, error) {
	purpose = strings.TrimSpace(purpose)
	actor = strings.TrimSpace(actor)
	if purpose == "" || len(purpose) < 10 {
		return IntelligenceReIDRequest{}, errors.New("purpose required (min 10 chars)")
	}
	if actor == "" {
		actor = "operator"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ci := range s.data.Cases {
		if s.data.Cases[ci].ID != caseID {
			continue
		}
		now := time.Now().UTC()
		req := IntelligenceReIDRequest{
			ID: intelID("reid"), Purpose: purpose, RequestedBy: actor,
			RequestedAt: now, ExpiresAt: now.Add(24 * time.Hour), Status: "requested", Unlocked: false,
		}
		s.data.Cases[ci].ReIDRequests = append(s.data.Cases[ci].ReIDRequests, req)
		s.data.Cases[ci].Audit = append(s.data.Cases[ci].Audit, IntelligenceAuditEntry{
			At: now, Action: "reid.requested", Actor: actor, Detail: req.ID + " " + purpose,
		})
		s.data.Revision++
		return req, s.commitLocked("reid.requested", req.ID)
	}
	return IntelligenceReIDRequest{}, errors.New("case not found")
}

// ApproveReID applies first or second approval. Second approval unlocks alias metadata for 30 minutes.
// First and second approver must differ (dual-control).
func (s *IntelligenceStore) ApproveReID(caseID, requestID, approver string) (IntelligenceReIDRequest, error) {
	approver = strings.TrimSpace(approver)
	if approver == "" {
		return IntelligenceReIDRequest{}, errors.New("approver required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ci := range s.data.Cases {
		if s.data.Cases[ci].ID != caseID {
			continue
		}
		for ri := range s.data.Cases[ci].ReIDRequests {
			r := &s.data.Cases[ci].ReIDRequests[ri]
			if r.ID != requestID {
				continue
			}
			now := time.Now().UTC()
			if r.Status == "expired" || r.Status == "denied" {
				return IntelligenceReIDRequest{}, errors.New("request is closed")
			}
			if r.Status == "requested" {
				r.FirstApprover = approver
				r.Status = "approved_once"
				s.data.Cases[ci].Audit = append(s.data.Cases[ci].Audit, IntelligenceAuditEntry{
					At: now, Action: "reid.approved_once", Actor: approver, Detail: r.ID,
				})
				s.data.Revision++
				return *r, s.commitLocked("reid.approved_once", r.ID)
			}
			if r.Status == "approved_once" {
				if strings.EqualFold(r.FirstApprover, approver) {
					return IntelligenceReIDRequest{}, errors.New("second approver must differ from first (dual-control)")
				}
				r.SecondApprover = approver
				r.Status = "unlocked"
				r.Unlocked = true
				r.ExpiresAt = now.Add(30 * time.Minute)
				s.data.Cases[ci].Audit = append(s.data.Cases[ci].Audit, IntelligenceAuditEntry{
					At: now, Action: "reid.unlocked", Actor: approver, Detail: r.ID + " window=30m alias-metadata-only",
				})
				s.data.Revision++
				return *r, s.commitLocked("reid.unlocked", r.ID)
			}
			if r.Status == "unlocked" {
				return *r, nil
			}
			return IntelligenceReIDRequest{}, errors.New("request not approvable in current state")
		}
		return IntelligenceReIDRequest{}, errors.New("reid request not found")
	}
	return IntelligenceReIDRequest{}, errors.New("case not found")
}

// CaseTimeline returns chronological audit + evidence + entity events for a case.
func (s *IntelligenceStore) CaseTimeline(caseID string) (string, error) {
	d := s.Snapshot()
	for _, c := range d.Cases {
		if c.ID != caseID {
			continue
		}
		var b strings.Builder
		b.WriteString("# Case Timeline: " + c.Title + "\n\n")
		b.WriteString("CaseID: " + c.ID + " · purpose: " + c.Purpose + "\n\n")
		// Merge audit + evidence collection + entities into a sorted narrative
		type row struct {
			at   time.Time
			line string
		}
		rows := make([]row, 0, len(c.Audit)+len(c.Evidence)+len(c.Entities))
		for _, a := range c.Audit {
			rows = append(rows, row{a.At, fmt.Sprintf("[%s] AUDIT %s by %s — %s", a.At.Format(time.RFC3339), a.Action, a.Actor, a.Detail)})
		}
		for _, e := range c.Evidence {
			srcEv := e.SourceEventID
			if srcEv != "" {
				srcEv = " source_event_id=" + srcEv
			}
			rows = append(rows, row{e.CollectedAt, fmt.Sprintf("[%s] EVIDENCE %s status=%s sealed=%v sha=%s%s — %s",
				e.CollectedAt.Format(time.RFC3339), e.ID, e.ValidationStatus, e.Sealed, e.SHA256, srcEv, e.Excerpt)})
		}
		for _, ent := range c.Entities {
			rows = append(rows, row{c.CreatedAt, fmt.Sprintf("[%s] ENTITY %s kind=%s conf=%d%% — %s",
				c.CreatedAt.Format(time.RFC3339), ent.ID, ent.Kind, ent.Confidence, ent.Label)})
		}
		for i := 0; i < len(rows); i++ {
			for j := i + 1; j < len(rows); j++ {
				if rows[j].at.Before(rows[i].at) {
					rows[i], rows[j] = rows[j], rows[i]
				}
			}
		}
		if len(rows) == 0 {
			b.WriteString("(empty timeline)\n")
		}
		for _, r := range rows {
			b.WriteString("- " + r.line + "\n")
		}
		b.WriteString("\n## Relations\n")
		if len(c.Relations) == 0 {
			b.WriteString("- (none)\n")
		}
		for _, rel := range c.Relations {
			b.WriteString(fmt.Sprintf("- %s — %s → %s [evidence=%s conf=%d%%]\n", rel.From, rel.Type, rel.To, rel.EvidenceID, rel.Confidence))
		}
		return b.String(), nil
	}
	return "", errors.New("case not found")
}

// SyncOSINTEvents folds the independent Global Watch collector cache into the
// governed Intelligence Core. It never writes to personal memory: raw public
// observations stay transient/proposed until an operator creates a case.
func (s *IntelligenceStore) SyncOSINTEvents(events []OSINTEvent) int {
	created := 0
	for _, event := range events {
		confidence := int(event.Confidence * 100)
		if confidence < 1 {
			confidence = 50
		}
		severity := "low"
		if event.Status == "alert" && confidence >= 85 {
			severity = "high"
		} else if event.Status == "alert" || event.Domain == DomainCyber {
			severity = "medium"
		}
		intelEvent := IntelligenceEvent{ID: "watch_" + strings.TrimSpace(event.ID), Title: event.Title, Summary: "[" + string(event.Domain) + "] " + event.Summary, Source: event.Source, SourceURL: event.SourceURL, Latitude: event.Lat, Longitude: event.Lon, Confidence: confidence, Severity: severity, ObservedAt: event.Timestamp}
		if strings.TrimSpace(intelEvent.ID) == "watch_" {
			intelEvent.ID = ""
		}
		if err := s.ProposeEvent(intelEvent); err == nil {
			created++
		}
	}
	return created
}

// LiveNexusContext is a compatibility shim. World-event chat MUST use
// SharedIntelStore.LiveNexusContext (unified model). When SharedIntelStore is set,
// this delegates there so legacy callers cannot diverge from the globe/handlers.
// When unset, still labels legacy rows as RAW proposals (never verified facts).
func (s *IntelligenceStore) LiveNexusContext(limit int) string {
	if SharedIntelStore != nil {
		return SharedIntelStore.LiveNexusContext(limit)
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 20 {
		limit = 20
	}
	d := s.Snapshot()
	var b strings.Builder
	b.WriteString("LEGACY FALLBACK NEXUS (SharedIntelStore unavailable)\n")
	b.WriteString("Layers: RAW OBSERVATION | INFERENCE | VERIFIED — verified only with sealed case evidence\n")
	alerts := s.Alerts()
	b.WriteString("Alerts: " + strconv.Itoa(len(alerts)) + " | Legacy events: " + strconv.Itoa(len(d.Events)) + "\n")
	for i, event := range d.Events {
		if i >= limit {
			break
		}
		// Legacy IntelligenceEvent is a proposed observation/signal, not a verified finding.
		status := strings.TrimSpace(event.Status)
		if status == "" {
			status = "proposed"
		}
		layer := "RAW/INFERENCE"
		if status == "verified" || status == "committed" {
			layer = "INFERENCE (not auto-verified)"
		}
		b.WriteString(fmt.Sprintf("- [%s][%s] %s | source=%s | confidence=%d%% | status=%s\n",
			layer, event.ID, event.Title, event.Source, event.Confidence, status))
	}
	b.WriteString("VERIFIED FINDINGS: none unless case evidence is sealed and reviewed.\n")
	b.WriteString("All items above are source-labelled proposals unless case evidence says otherwise.")
	return b.String()
}
func (s *IntelligenceStore) FocusGlobalWatch(latitude, longitude, zoom float64, label string) error {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 || zoom < 0.5 || zoom > 3 {
		return errors.New("global watch focus coordinates or zoom are invalid")
	}
	if s.bus == nil {
		return errors.New("global watch bus unavailable")
	}
	s.bus.PublishGlobalWatchCommand(GlobalWatchCommand{Action: "focus", Latitude: latitude, Longitude: longitude, Zoom: zoom, Label: strings.TrimSpace(label)})
	return nil
}

func (s *IntelligenceStore) SetGlobalWatchLayer(layer string, enabled bool) error {
	layer = strings.ToLower(strings.TrimSpace(layer))
	allowed := map[string]bool{"borders": true, "cameras": true, "satellites": true, "cables": true, "daynight": true, "news": true, "cities": true, "risks": true}
	if !allowed[layer] {
		return errors.New("global watch layer is invalid")
	}
	if s.bus == nil {
		return errors.New("global watch bus unavailable")
	}
	s.bus.PublishGlobalWatchCommand(GlobalWatchCommand{Action: "layer", Layer: layer, Enable: &enabled})
	return nil
}

// NavigateUI publishes a command for the frontend to switch viewport (global_watch, sphere, core, personal, …).
func (s *IntelligenceStore) NavigateUI(view string) error {
	view = strings.ToLower(strings.TrimSpace(view))
	allowed := map[string]bool{
		"core": true, "chat": true, "global_watch": true, "globe": true, "sphere": true,
		"personal": true, "case": true, "tasks": true, "settings": true, "memory": true,
		"agents": true, "agent_tracker": true,
	}
	if !allowed[view] {
		return errors.New("navigate view is invalid")
	}
	if s.bus == nil {
		return errors.New("global watch bus unavailable")
	}
	s.bus.PublishGlobalWatchCommand(GlobalWatchCommand{Action: "navigate", View: view})
	return nil
}

// FocusGlobalWatchRegion focuses a named region chip (europe, germany, …).
func (s *IntelligenceStore) FocusGlobalWatchRegion(region string) error {
	region = strings.ToLower(strings.TrimSpace(region))
	if region == "" {
		return errors.New("region is required")
	}
	if s.bus == nil {
		return errors.New("global watch bus unavailable")
	}
	s.bus.PublishGlobalWatchCommand(GlobalWatchCommand{Action: "focus_region", Region: region, Label: region})
	return nil
}

// SetGlobalWatchTimeWindowHours sets the Live Globe feed/pin time filter (0 = all).
func (s *IntelligenceStore) SetGlobalWatchTimeWindowHours(hours float64) error {
	if hours < 0 || hours > 24*30 {
		return errors.New("time window hours out of range")
	}
	if s.bus == nil {
		return errors.New("global watch bus unavailable")
	}
	s.bus.PublishGlobalWatchCommand(GlobalWatchCommand{Action: "time_window", Hours: hours})
	return nil
}

// OpenGlobalWatchReport opens the freestanding report reader with markdown body.
func (s *IntelligenceStore) OpenGlobalWatchReport(title, body string) error {
	if s.bus == nil {
		return errors.New("global watch bus unavailable")
	}
	if len([]rune(body)) > 120000 {
		return errors.New("report body too large")
	}
	s.bus.PublishGlobalWatchCommand(GlobalWatchCommand{
		Action: "open_report",
		Label:  strings.TrimSpace(title),
		Body:   body,
	})
	return nil
}
func sharedIntelTokens(a, b string) []string {
	set := map[string]bool{}
	for _, v := range strings.Fields(strings.ToLower(a)) {
		v = strings.Trim(v, ".,:;!?()[]\"'")
		if len(v) > 4 {
			set[v] = true
		}
	}
	out := []string{}
	for _, v := range strings.Fields(strings.ToLower(b)) {
		v = strings.Trim(v, ".,:;!?()[]\"'")
		if set[v] && len(v) > 4 {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
func minIntel(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func (s *IntelligenceStore) CaseReport(caseID string) (string, error) {
	d := s.Snapshot()
	for _, c := range d.Cases {
		if c.ID != caseID {
			continue
		}
		var b strings.Builder
		b.WriteString("# " + c.Title + "\n\n")
		b.WriteString("**Purpose:** " + c.Purpose + "\n\n")
		b.WriteString("**Classification:** " + c.Classification + "\n\n## Evidence\n")
		for _, e := range c.Evidence {
			b.WriteString("- [" + e.ID + "] " + e.Source + " · " + e.ValidationStatus + " · SHA-256 `" + e.SHA256 + "`\n")
		}
		b.WriteString("\n## Entities\n")
		for _, e := range c.Entities {
			b.WriteString("- " + e.Label + " (" + e.Kind + ", " + strconv.Itoa(e.Confidence) + "%)\n")
		}
		b.WriteString("\n## Relationships\n")
		for _, r := range c.Relations {
			b.WriteString("- " + r.From + " — " + r.Type + " → " + r.To + " [evidence: " + r.EvidenceID + "]\n")
		}
		return b.String(), nil
	}
	return "", errors.New("case not found")
}
func (s *IntelligenceStore) AddEntity(caseID, label, kind string, confidence int) (IntelligenceEntity, error) {
	label, kind = strings.TrimSpace(label), strings.ToLower(strings.TrimSpace(kind))
	if label == "" || (kind != "person" && kind != "organisation" && kind != "location" && kind != "asset" && kind != "event") || confidence < 0 || confidence > 100 {
		return IntelligenceEntity{}, errors.New("entity fields are invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ci := range s.data.Cases {
		if s.data.Cases[ci].ID == caseID {
			display := label
			if kind == "person" {
				alias, err := casePseudonym(caseID, label)
				if err != nil {
					return IntelligenceEntity{}, err
				}
				display = alias
			}
			entity := IntelligenceEntity{ID: intelID("ent"), Label: display, Kind: kind, Confidence: confidence}
			s.data.Cases[ci].Entities = append(s.data.Cases[ci].Entities, entity)
			s.data.Cases[ci].Audit = append(s.data.Cases[ci].Audit, IntelligenceAuditEntry{
				At: time.Now().UTC(), Action: "entity.added", Actor: "operator", Detail: entity.ID + " kind=" + kind,
			})
			s.data.Revision++
			if err := s.commitLocked("entity.added", entity.ID); err != nil {
				return IntelligenceEntity{}, err
			}
			if SharedIntelStore != nil {
				_ = SharedIntelStore.AddCaseEntity(caseID, entity.ID, entity.Label, entity.Kind, entity.Confidence)
			}
			return entity, nil
		}
	}
	return IntelligenceEntity{}, errors.New("case not found")
}
func (s *IntelligenceStore) LinkEntities(caseID, from, to, relation, evidenceID string, confidence int) (IntelligenceRelation, error) {
	if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" || strings.TrimSpace(relation) == "" || strings.TrimSpace(evidenceID) == "" || confidence < 0 || confidence > 100 {
		return IntelligenceRelation{}, errors.New("relationship fields are invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ci := range s.data.Cases {
		if s.data.Cases[ci].ID != caseID {
			continue
		}
		foundFrom, foundTo, foundEvidence := false, false, false
		for _, e := range s.data.Cases[ci].Entities {
			foundFrom = foundFrom || e.ID == from
			foundTo = foundTo || e.ID == to
		}
		for _, e := range s.data.Cases[ci].Evidence {
			foundEvidence = foundEvidence || e.ID == evidenceID
		}
		if !foundFrom || !foundTo || !foundEvidence {
			return IntelligenceRelation{}, errors.New("relationship must reference case-local entities and sealed evidence")
		}
		rel := IntelligenceRelation{From: from, To: to, Type: strings.TrimSpace(relation), EvidenceID: evidenceID, Confidence: confidence}
		s.data.Cases[ci].Relations = append(s.data.Cases[ci].Relations, rel)
		s.data.Cases[ci].Audit = append(s.data.Cases[ci].Audit, IntelligenceAuditEntry{
			At: time.Now().UTC(), Action: "relationship.added", Actor: "operator", Detail: from + "→" + to,
		})
		s.data.Revision++
		if err := s.commitLocked("relationship.added", caseID); err != nil {
			return IntelligenceRelation{}, err
		}
		if SharedIntelStore != nil {
			_ = SharedIntelStore.LinkCaseRelation(caseID, from, to, rel.Type, evidenceID, confidence)
		}
		return rel, nil
	}
	return IntelligenceRelation{}, errors.New("case not found")
}
func casePseudonym(caseID, identifier string) (string, error) {
	secret, err := deriveCaseSecret(caseID)
	if err != nil {
		return "", err
	}
	return pseudonymEntity(secret, identifier), nil
}

// deriveCaseSecret derives a stable case-scoped secret from platform-protected
// key material. No repository constant participates in the pseudonym key.
func deriveCaseSecret(caseID string) ([]byte, error) {
	if strings.TrimSpace(caseID) == "" {
		return nil, errors.New("case id is required")
	}
	master, err := security.GetSecretKey()
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, master)
	mac.Write([]byte(caseID))
	return mac.Sum(nil)[:32], nil
}

func WriteIntelJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type RegionalRiskData struct {
	RegionID           string   `json:"region_id"`
	RegionName         string   `json:"region_name"`
	OverallRisk        float64  `json:"overall_risk"`
	GeopoliticalRisk   float64  `json:"geopolitical_risk"`
	ConflictRisk       float64  `json:"conflict_risk"`
	CyberRisk          float64  `json:"cyber_risk"`
	InfrastructureRisk float64  `json:"infrastructure_risk"`
	EconomicRisk       float64  `json:"economic_risk"`
	PrimaryDrivers     []string `json:"primary_drivers"`
	Trend              string   `json:"trend"`
}

func (s *IntelligenceStore) ComputeAllRegionalRisks() []RegionalRiskData {
	d := s.Snapshot()

	regions := []struct {
		ID             string
		Name           string
		MinLat, MaxLat float64
		MinLon, MaxLon float64
	}{
		{"GERMANY", "Germany", 47.2, 55.0, 5.8, 15.0},
		{"FRANCE", "France", 42.3, 51.1, -5.0, 9.5},
		{"USA", "United States", 24.5, 49.0, -125.0, -66.9},
		{"UKRAINE", "Ukraine", 44.3, 52.4, 22.0, 40.2},
		{"UK", "United Kingdom", 49.9, 60.8, -8.6, 1.7},
	}

	out := []RegionalRiskData{}
	now := time.Now().UTC()

	for _, r := range regions {
		var geopot, conflict, cyber, infra, eco float64
		drivers := []string{}

		for _, ev := range d.Events {
			if ev.Latitude >= r.MinLat && ev.Latitude <= r.MaxLat && ev.Longitude >= r.MinLon && ev.Longitude <= r.MaxLon {
				dt := now.Sub(ev.ObservedAt).Hours()
				if dt < 0 {
					dt = 0
				}
				freshness := 1.0 / (1.0 + 0.02*dt)

				sevMult := 1.0
				if ev.Severity == "medium" {
					sevMult = 1.5
				} else if ev.Severity == "high" {
					sevMult = 2.5
				}

				confMult := float64(ev.Confidence) / 100.0

				weight := 10.0
				domain := strings.ToLower(ev.Summary)

				if strings.Contains(domain, "cyber") || ev.ID == "watch_cyber" {
					weight = 12.0
					cyber += weight * sevMult * freshness * confMult
					if ev.Severity == "high" && freshness > 0.5 {
						drivers = append(drivers, "Cyber: "+ev.Title)
					}
				} else if strings.Contains(domain, "eco") || strings.Contains(domain, "econ") {
					weight = 8.0
					eco += weight * sevMult * freshness * confMult
					if ev.Severity == "high" && freshness > 0.5 {
						drivers = append(drivers, "Economic: "+ev.Title)
					}
				} else if strings.Contains(domain, "conflict") || strings.Contains(domain, "hum") {
					weight = 15.0
					conflict += weight * sevMult * freshness * confMult
					if ev.Severity == "high" && freshness > 0.5 {
						drivers = append(drivers, "Conflict: "+ev.Title)
					}
				} else if strings.Contains(domain, "geo") {
					weight = 10.0
					geopot += weight * sevMult * freshness * confMult
					if ev.Severity == "high" && freshness > 0.5 {
						drivers = append(drivers, "Geopolitical: "+ev.Title)
					}
				} else {
					weight = 10.0
					infra += weight * sevMult * freshness * confMult
					if ev.Severity == "high" && freshness > 0.5 {
						drivers = append(drivers, "Infrastructure: "+ev.Title)
					}
				}
			}
		}

		if geopot > 100 {
			geopot = 100
		}
		if conflict > 100 {
			conflict = 100
		}
		if cyber > 100 {
			cyber = 100
		}
		if infra > 100 {
			infra = 100
		}
		if eco > 100 {
			eco = 100
		}

		maxRisk := geopot
		if conflict > maxRisk {
			maxRisk = conflict
		}
		if cyber > maxRisk {
			maxRisk = cyber
		}
		if infra > maxRisk {
			maxRisk = infra
		}
		if eco > maxRisk {
			maxRisk = eco
		}

		avgRisk := (geopot + conflict + cyber + infra + eco) / 5.0
		overall := 0.6*maxRisk + 0.4*avgRisk
		if overall > 100 {
			overall = 100
		}

		trend := "stable"
		if overall > 50 {
			trend = "up"
		} else if overall < 15 {
			trend = "down"
		}

		out = append(out, RegionalRiskData{
			RegionID:           r.ID,
			RegionName:         r.Name,
			OverallRisk:        overall,
			GeopoliticalRisk:   geopot,
			ConflictRisk:       conflict,
			CyberRisk:          cyber,
			InfrastructureRisk: infra,
			EconomicRisk:       eco,
			PrimaryDrivers:     drivers,
			Trend:              trend,
		})
	}
	return out
}

func (s *IntelligenceStore) GenerateNeuralCoreEvaluation() (string, error) {
	d := s.Snapshot()
	risks := s.ComputeAllRegionalRisks()

	var risksText strings.Builder
	for _, r := range risks {
		risksText.WriteString(fmt.Sprintf("- %s: Overall Risk %.1f%% (Trend: %s, Cyber: %.1f%%, Conflict: %.1f%%, Geopolitical: %.1f%%)\n", r.RegionName, r.OverallRisk, r.Trend, r.CyberRisk, r.ConflictRisk, r.GeopoliticalRisk))
	}

	var eventsText strings.Builder
	limit := 10
	if len(d.Events) < limit {
		limit = len(d.Events)
	}
	for i := 0; i < limit; i++ {
		ev := d.Events[len(d.Events)-1-i]
		eventsText.WriteString(fmt.Sprintf("- [%s] %s: %s\n", ev.Severity, ev.Title, ev.Summary))
	}

	systemPrompt := "You are VGT AETHEL, a sovereign local Personal Intelligence Operating System. Evaluate the current world state and local state situation."
	userPrompt := fmt.Sprintf(`Aktuelle Risikomatrix:
%s

Neueste Beobachtungen:
%s

Bitte erstelle eine prägnante, professionelle Lagebewertung im JSON-Format mit genau zwei Feldern:
1. "world_state": Eine geopolitische und cyber-sicherheitsbezogene Einschätzung der globalen Lage (max. 150 Wörter).
2. "local_state": Eine Bewertung, wie diese globalen Lagen Deutschland und die direkte Umgebung des Operators (Wirtschaft, Störungen, Cyber) beeinflussen (max. 150 Wörter).

Antworte ausschließlich im folgenden validen JSON-Format ohne Markdown-Wrapper:
{
  "world_state": "Globale Lagebeschreibung...",
  "local_state": "Lokale Auswirkungen..."
}`, risksText.String(), eventsText.String())

	if s.ChatEvaluator == nil {
		return "", errors.New("ChatEvaluator is not registered")
	}
	return s.ChatEvaluator(systemPrompt, userPrompt)
}

func (s *IntelligenceStore) UpdateEvaluation() (NeuralCoreEvaluation, error) {
	raw, err := s.GenerateNeuralCoreEvaluation()
	if err != nil {
		return NeuralCoreEvaluation{}, err
	}

	var parsed struct {
		WorldState string `json:"world_state"`
		LocalState string `json:"local_state"`
	}

	cleaned := raw
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		parsed.WorldState = raw
		parsed.LocalState = "Lokale Implikationen konnten aufgrund eines Formatierungsfehlers nicht separat extrahiert werden."
	}

	s.mu.Lock()
	s.data.Evaluation = NeuralCoreEvaluation{
		WorldState:  parsed.WorldState,
		LocalState:  parsed.LocalState,
		LastUpdated: time.Now().UTC(),
	}
	_ = s.saveLocked()
	s.mu.Unlock()

	s.bus.Publish(IntelligenceBusEvent{
		Type:      "evaluation.updated",
		SubjectID: "core",
		Revision:  s.data.Revision,
		At:        time.Now().UTC(),
	})
	return s.data.Evaluation, nil
}

func (s *IntelligenceStore) StartBackgroundEvaluationWorker() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Println("[Intelligence] Automatic hourly situation evaluation triggered...")
				_, _ = s.UpdateEvaluation()
			}
		}
	}()
}

func (s *IntelligenceStore) Bus() *IntelligenceBus {
	return s.bus
}
