package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"errors"
	"sort"
	"strings"
	"time"
)

func (s *Store) UpdateHypothesisFramework(id string, indicators []HypothesisIndicator, alternativeIDs, gapIDs, changeConditions []string, actor string) (Hypothesis, error) {
	id, actor = strings.TrimSpace(id), strings.TrimSpace(actor)
	if id == "" || actor == "" {
		return Hypothesis{}, errors.New("hypothesis and actor are required")
	}
	for index := range indicators {
		indicators[index].Description = strings.TrimSpace(indicators[index].Description)
		if indicators[index].Description == "" || indicators[index].Diagnosticity < 0 || indicators[index].Diagnosticity > 100 {
			return Hypothesis{}, errors.New("indicator description and diagnosticity are invalid")
		}
		if indicators[index].ID == "" {
			indicatorID, err := newIntelID("indicator")
			if err != nil {
				return Hypothesis{}, err
			}
			indicators[index].ID = indicatorID
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.state.Hypotheses {
		hypothesis := &s.state.Hypotheses[index]
		if hypothesis.ID != id {
			continue
		}
		if !validHypothesisReferences(s.state, *hypothesis, alternativeIDs, gapIDs, indicators) {
			return Hypothesis{}, errors.New("framework references must exist in the same case")
		}
		hypothesis.Indicators = append([]HypothesisIndicator(nil), indicators...)
		hypothesis.AlternativeHypothesisIDs = uniqueNonEmpty(alternativeIDs)
		hypothesis.InformationGapIDs = uniqueNonEmpty(gapIDs)
		hypothesis.ChangeConditions = uniqueNonEmpty(changeConditions)
		hypothesis.UpdatedAt = time.Now().UTC()
		s.state.Audits = append(s.state.Audits, AuditEvent{At: hypothesis.UpdatedAt, Action: "hypothesis.framework-updated", Actor: actor, Detail: hypothesis.ID})
		if err := s.save(); err != nil {
			return Hypothesis{}, err
		}
		return *hypothesis, nil
	}
	return Hypothesis{}, errors.New("hypothesis not found")
}

func (s *Store) AssessHypothesisEvidence(hypothesisID, evidenceID string, compatibility, diagnosticity int, reason, actor string) (HypothesisEvidenceAssessment, error) {
	hypothesisID, evidenceID = strings.TrimSpace(hypothesisID), strings.TrimSpace(evidenceID)
	reason, actor = strings.TrimSpace(reason), strings.TrimSpace(actor)
	if hypothesisID == "" || evidenceID == "" || reason == "" || actor == "" {
		return HypothesisEvidenceAssessment{}, errors.New("hypothesis, evidence, reason and actor are required")
	}
	if compatibility < -2 || compatibility > 2 || diagnosticity < 0 || diagnosticity > 100 {
		return HypothesisEvidenceAssessment{}, errors.New("compatibility or diagnosticity is outside policy")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	hypothesisIndex, caseID := -1, ""
	for index := range s.state.Hypotheses {
		if s.state.Hypotheses[index].ID == hypothesisID {
			hypothesisIndex, caseID = index, s.state.Hypotheses[index].CaseID
			break
		}
	}
	if hypothesisIndex < 0 || !caseEvidenceExists(s.state.Cases, caseID, evidenceID) {
		return HypothesisEvidenceAssessment{}, errors.New("hypothesis and evidence must exist in the same case")
	}
	id, err := newIntelID("ach-assessment")
	if err != nil {
		return HypothesisEvidenceAssessment{}, err
	}
	assessment := HypothesisEvidenceAssessment{
		ID: id, CaseID: caseID, HypothesisID: hypothesisID, EvidenceID: evidenceID,
		Compatibility: compatibility, Diagnosticity: diagnosticity, Reason: reason, Actor: actor, At: time.Now().UTC(),
	}
	s.state.HypothesisEvidenceAssessments = append(s.state.HypothesisEvidenceAssessments, assessment)
	hypothesis := &s.state.Hypotheses[hypothesisIndex]
	if compatibility > 0 {
		hypothesis.SupportingEvidenceIDs = appendUnique(hypothesis.SupportingEvidenceIDs, evidenceID)
		hypothesis.ContradictingEvidenceIDs = removeString(hypothesis.ContradictingEvidenceIDs, evidenceID)
	} else if compatibility < 0 {
		hypothesis.ContradictingEvidenceIDs = appendUnique(hypothesis.ContradictingEvidenceIDs, evidenceID)
		hypothesis.SupportingEvidenceIDs = removeString(hypothesis.SupportingEvidenceIDs, evidenceID)
	}
	hypothesis.UpdatedAt = assessment.At
	s.state.Audits = append(s.state.Audits, AuditEvent{At: assessment.At, Action: "hypothesis.evidence-assessed", Actor: actor, Detail: assessment.ID})
	s.appendCustodyLocked(evidenceID, "hypothesis.evidence-assessed", actor, assessment.ID)
	if err := s.save(); err != nil {
		return HypothesisEvidenceAssessment{}, err
	}
	s.publish("hypothesis.evidence-assessed", hypothesisID, assessment)
	return assessment, nil
}

func (s *Store) BuildACHMatrix(caseID string) (ACHMatrix, error) {
	caseID = strings.TrimSpace(caseID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !caseExists(s.state.Cases, caseID) {
		return ACHMatrix{}, errors.New("case not found")
	}
	latest := make(map[string]HypothesisEvidenceAssessment)
	evidenceSet := make(map[string]bool)
	for _, assessment := range s.state.HypothesisEvidenceAssessments {
		if assessment.CaseID != caseID {
			continue
		}
		key := assessment.HypothesisID + "\x00" + assessment.EvidenceID
		if previous, exists := latest[key]; !exists || assessment.At.After(previous.At) {
			latest[key] = assessment
		}
		evidenceSet[assessment.EvidenceID] = true
	}
	evidenceIDs := make([]string, 0, len(evidenceSet))
	for evidenceID := range evidenceSet {
		evidenceIDs = append(evidenceIDs, evidenceID)
	}
	sort.Strings(evidenceIDs)
	rows := make([]ACHMatrixRow, 0)
	for _, hypothesis := range s.state.Hypotheses {
		if hypothesis.CaseID != caseID {
			continue
		}
		row := ACHMatrixRow{HypothesisID: hypothesis.ID, Statement: hypothesis.Statement, Assessments: []HypothesisEvidenceAssessment{}}
		for _, evidenceID := range evidenceIDs {
			assessment, exists := latest[hypothesis.ID+"\x00"+evidenceID]
			if !exists {
				row.MissingEvidence++
				continue
			}
			row.Assessments = append(row.Assessments, assessment)
			if assessment.Compatibility < 0 {
				row.InconsistencyScore += -assessment.Compatibility * assessment.Diagnosticity
			}
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].InconsistencyScore == rows[j].InconsistencyScore {
			if rows[i].MissingEvidence == rows[j].MissingEvidence {
				return rows[i].HypothesisID < rows[j].HypothesisID
			}
			return rows[i].MissingEvidence < rows[j].MissingEvidence
		}
		return rows[i].InconsistencyScore < rows[j].InconsistencyScore
	})
	for index := range rows {
		rows[index].Rank = index + 1
	}
	return ACHMatrix{CaseID: caseID, EvidenceIDs: evidenceIDs, Rows: rows, GeneratedAt: time.Now().UTC()}, nil
}

func validHypothesisReferences(state StoreState, hypothesis Hypothesis, alternatives, gaps []string, indicators []HypothesisIndicator) bool {
	for _, id := range alternatives {
		if id == hypothesis.ID {
			return false
		}
		found := false
		for _, candidate := range state.Hypotheses {
			if candidate.ID == id && candidate.CaseID == hypothesis.CaseID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, id := range gaps {
		if !informationGapExists(state.InformationGaps, hypothesis.CaseID, id) {
			return false
		}
	}
	for _, indicator := range indicators {
		for _, evidenceID := range indicator.EvidenceIDs {
			if !caseEvidenceExists(state.Cases, hypothesis.CaseID, evidenceID) {
				return false
			}
		}
	}
	return true
}

func caseEvidenceExists(cases []Case, caseID, evidenceID string) bool {
	for _, caseRecord := range cases {
		if caseRecord.ID == caseID {
			for _, evidence := range caseRecord.Evidence {
				if evidence.ID == evidenceID {
					return true
				}
			}
		}
	}
	return false
}
func uniqueNonEmpty(values []string) []string {
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = appendUnique(result, value)
		}
	}
	return result
}
func appendUnique(values []string, wanted string) []string {
	if containsString(values, wanted) {
		return values
	}
	return append(values, wanted)
}
func removeString(values []string, unwanted string) []string {
	result := values[:0]
	for _, value := range values {
		if value != unwanted {
			result = append(result, value)
		}
	}
	return result
}
