package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"errors"
	"os"
	"strings"
	"time"
)

const legacyIntelligenceMigrationID = "legacy-intelligence-core-v1"

func (s *IntelligenceStore) publishCompatibility(eventType, subjectID string) {
	if s.bus == nil || s.canonical == nil {
		return
	}
	revision := s.canonical.GetSnapshot().Revision
	s.bus.Publish(IntelligenceBusEvent{Type: eventType, SubjectID: subjectID, Revision: revision, At: time.Now().UTC()})
}

func canonicalCompatibilitySnapshot(snapshot StoreState, evaluation NeuralCoreEvaluation) intelligenceState {
	if !snapshot.Evaluation.LastUpdated.IsZero() {
		evaluation = snapshot.Evaluation
	}
	result := intelligenceState{
		Events:     make([]IntelligenceEvent, 0, len(snapshot.Events)),
		Cases:      make([]IntelligenceCase, 0, len(snapshot.Cases)),
		Revision:   snapshot.Revision,
		Evaluation: evaluation,
	}
	for _, event := range snapshot.Events {
		result.Events = append(result.Events, IntelligenceEvent{
			ID: event.ID, Title: event.Title, Summary: event.Summary, Source: event.SourceID,
			Latitude: event.Latitude, Longitude: event.Longitude, Severity: event.Severity,
			Confidence: event.Confidence, ObservedAt: event.ObservedAt, Status: "proposed",
		})
	}
	for _, caseRecord := range snapshot.Cases {
		result.Cases = append(result.Cases, compatibilityCase(caseRecord))
	}
	return result
}

func MigrateLegacyIntelligence(path string, canonical *Store) error {
	if canonical == nil {
		return errors.New("canonical intelligence store is required")
	}
	if canonical.MigrationApplied(legacyIntelligenceMigrationID) {
		return nil
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return canonical.MarkMigration(legacyIntelligenceMigrationID)
	} else if err != nil {
		return err
	}
	legacy := NewIntelligenceStore(path)
	snapshot := legacy.Snapshot()
	adapter := NewCanonicalIntelligenceAdapter(canonical)
	for _, event := range snapshot.Events {
		if err := adapter.ProposeEvent(event); err != nil && !errors.Is(err, ErrDuplicateObservation) {
			return err
		}
	}
	for _, legacyCase := range snapshot.Cases {
		if _, err := canonical.CreateCaseWithID(legacyCase.ID, legacyCase.Title, legacyCase.Purpose); err != nil {
			return err
		}
		for _, evidence := range legacyCase.Evidence {
			if _, err := canonical.SealCaseEvidence(legacyCase.ID, evidence.ID, evidence.Source, evidence.URL, evidence.Excerpt, evidence.SHA256, evidence.SourceEventID); err != nil {
				return err
			}
		}
		for _, entity := range legacyCase.Entities {
			if err := canonical.AddCaseEntity(legacyCase.ID, entity.ID, entity.Label, entity.Kind, entity.Confidence); err != nil {
				return err
			}
		}
		for _, relation := range legacyCase.Relations {
			if err := canonical.LinkCaseRelation(legacyCase.ID, relation.From, relation.To, relation.Type, relation.EvidenceID, relation.Confidence); err != nil {
				return err
			}
		}
		if len(legacyCase.ReIDRequests) > 0 {
			if err := canonical.ImportReIDRequests(legacyCase.ID, legacyCase.ReIDRequests); err != nil {
				return err
			}
		}
	}
	if !snapshot.Evaluation.LastUpdated.IsZero() {
		if err := canonical.SetEvaluation(snapshot.Evaluation); err != nil {
			return err
		}
	}
	return canonical.MarkMigration(legacyIntelligenceMigrationID)
}

func (s *Store) ImportReIDRequests(caseID string, requests []IntelligenceReIDRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.state.Cases {
		if s.state.Cases[index].ID != strings.TrimSpace(caseID) {
			continue
		}
		s.state.Cases[index].ReIDRequests = append([]IntelligenceReIDRequest(nil), requests...)
		return s.save()
	}
	return errors.New("case not found")
}

func (s *Store) ReIDStatus(caseID string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for index := range s.state.Cases {
		caseRecord := &s.state.Cases[index]
		if caseRecord.ID != strings.TrimSpace(caseID) {
			continue
		}
		changed := expireCanonicalReID(caseRecord, now)
		unlocked := false
		var unlockUntil time.Time
		for _, request := range caseRecord.ReIDRequests {
			if request.Unlocked && request.Status == "unlocked" && now.Before(request.ExpiresAt) {
				unlocked = true
				unlockUntil = request.ExpiresAt
			}
		}
		if changed {
			if err := s.save(); err != nil {
				return nil, err
			}
		}
		return map[string]any{
			"case_id":           caseRecord.ID,
			"reidentification":  "not_eligible",
			"reason":            "Aethel stores only case-scoped HMAC aliases for person entities; raw identities are not retained. Dual-approval unlock only expands alias metadata visibility for a time-bound window.",
			"entities":          len(caseRecord.Entities),
			"request_count":     len(caseRecord.ReIDRequests),
			"requests":          append([]IntelligenceReIDRequest(nil), caseRecord.ReIDRequests...),
			"alias_unlock":      unlocked,
			"unlock_expires_at": unlockUntil,
		}, nil
	}
	return nil, errors.New("case not found")
}

func (s *Store) RequestReID(caseID, purpose, actor string) (IntelligenceReIDRequest, error) {
	purpose = strings.TrimSpace(purpose)
	actor = strings.TrimSpace(actor)
	if len(purpose) < 10 {
		return IntelligenceReIDRequest{}, errors.New("purpose required (min 10 chars)")
	}
	if actor == "" {
		actor = "operator"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.state.Cases {
		caseRecord := &s.state.Cases[index]
		if caseRecord.ID != strings.TrimSpace(caseID) {
			continue
		}
		now := time.Now().UTC()
		request := IntelligenceReIDRequest{
			ID:          intelID("reid"),
			Purpose:     purpose,
			RequestedBy: actor,
			RequestedAt: now,
			ExpiresAt:   now.Add(24 * time.Hour),
			Status:      "requested",
		}
		caseRecord.ReIDRequests = append(caseRecord.ReIDRequests, request)
		entry := AuditEvent{At: now, Action: "reid.requested", Actor: actor, Detail: request.ID + " " + purpose}
		caseRecord.Audit = append(caseRecord.Audit, entry)
		s.state.Audits = append(s.state.Audits, entry)
		s.appendCustodyLocked(request.ID, entry.Action, actor, caseRecord.ID)
		if err := s.save(); err != nil {
			return IntelligenceReIDRequest{}, err
		}
		return request, nil
	}
	return IntelligenceReIDRequest{}, errors.New("case not found")
}

func (s *Store) ApproveReID(caseID, requestID, approver string) (IntelligenceReIDRequest, error) {
	approver = strings.TrimSpace(approver)
	if approver == "" {
		return IntelligenceReIDRequest{}, errors.New("approver required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for caseIndex := range s.state.Cases {
		caseRecord := &s.state.Cases[caseIndex]
		if caseRecord.ID != strings.TrimSpace(caseID) {
			continue
		}
		expireCanonicalReID(caseRecord, now)
		for requestIndex := range caseRecord.ReIDRequests {
			request := &caseRecord.ReIDRequests[requestIndex]
			if request.ID != strings.TrimSpace(requestID) {
				continue
			}
			var action string
			switch request.Status {
			case "requested":
				request.FirstApprover = approver
				request.Status = "approved_once"
				action = "reid.approved_once"
			case "approved_once":
				if strings.EqualFold(request.FirstApprover, approver) {
					return IntelligenceReIDRequest{}, errors.New("second approver must differ from first (dual-control)")
				}
				request.SecondApprover = approver
				request.Status = "unlocked"
				request.Unlocked = true
				request.ExpiresAt = now.Add(30 * time.Minute)
				action = "reid.unlocked"
			case "unlocked":
				return *request, nil
			case "expired", "denied":
				return IntelligenceReIDRequest{}, errors.New("request is closed")
			default:
				return IntelligenceReIDRequest{}, errors.New("request not approvable in current state")
			}
			entry := AuditEvent{At: now, Action: action, Actor: approver, Detail: request.ID}
			caseRecord.Audit = append(caseRecord.Audit, entry)
			s.state.Audits = append(s.state.Audits, entry)
			s.appendCustodyLocked(request.ID, action, approver, caseRecord.ID)
			if err := s.save(); err != nil {
				return IntelligenceReIDRequest{}, err
			}
			return *request, nil
		}
		return IntelligenceReIDRequest{}, errors.New("reid request not found")
	}
	return IntelligenceReIDRequest{}, errors.New("case not found")
}

func expireCanonicalReID(caseRecord *Case, now time.Time) bool {
	changed := false
	for index := range caseRecord.ReIDRequests {
		request := &caseRecord.ReIDRequests[index]
		if (request.Status == "unlocked" || request.Unlocked) && !now.Before(request.ExpiresAt) {
			request.Status = "expired"
			request.Unlocked = false
			caseRecord.Audit = append(caseRecord.Audit, AuditEvent{At: now, Action: "reid.expired", Actor: "system", Detail: request.ID})
			changed = true
		}
	}
	return changed
}

func compatibilityCase(caseRecord Case) IntelligenceCase {
	result := IntelligenceCase{
		ID: caseRecord.ID, Title: caseRecord.Title, Purpose: caseRecord.Purpose,
		Classification: caseRecord.Classification, Status: caseRecord.Status, CreatedAt: caseRecord.CreatedAt,
		Evidence:     make([]IntelligenceEvidence, 0, len(caseRecord.Evidence)),
		Entities:     make([]IntelligenceEntity, 0, len(caseRecord.Entities)),
		Relations:    make([]IntelligenceRelation, 0, len(caseRecord.Relations)),
		Audit:        make([]IntelligenceAuditEntry, 0, len(caseRecord.Audit)),
		ReIDRequests: append([]IntelligenceReIDRequest(nil), caseRecord.ReIDRequests...),
	}
	for _, evidence := range caseRecord.Evidence {
		result.Evidence = append(result.Evidence, compatibilityEvidence(evidence))
	}
	for _, entity := range caseRecord.Entities {
		result.Entities = append(result.Entities, IntelligenceEntity{ID: entity.ID, Label: entity.Label, Kind: entity.Kind, Confidence: entity.Confidence})
	}
	for _, relation := range caseRecord.Relations {
		evidenceID := ""
		if len(relation.EvidenceIDs) > 0 {
			evidenceID = relation.EvidenceIDs[0]
		}
		result.Relations = append(result.Relations, IntelligenceRelation{From: relation.FromEntity, To: relation.ToEntity, Type: relation.RelationType, EvidenceID: evidenceID, Confidence: relation.Confidence})
	}
	for _, audit := range caseRecord.Audit {
		result.Audit = append(result.Audit, IntelligenceAuditEntry{At: audit.At, Action: audit.Action, Actor: audit.Actor, Detail: audit.Detail})
	}
	return result
}

func compatibilityEvidence(evidence Evidence) IntelligenceEvidence {
	return IntelligenceEvidence{
		ID: evidence.ID, CaseID: evidence.CaseID, Source: evidence.SourceID, URL: evidence.URL,
		Excerpt: evidence.Excerpt, SHA256: evidence.SHA256, CollectedAt: evidence.CollectedAt,
		Sealed: evidence.Sealed, ValidationStatus: evidence.ValidationStatus,
	}
}
