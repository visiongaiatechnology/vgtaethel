package intelligence

// STATUS: DIAMANT VGT SUPREME

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

func newIntelID(prefix string) (string, error) {
	var entropy [16]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(entropy[:]), nil
}

func clampConfidence(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func (s *Store) AddSourceLineage(lineage SourceLineage) (SourceLineage, error) {
	lineage.UpstreamSource = strings.TrimSpace(lineage.UpstreamSource)
	lineage.DownstreamSource = strings.TrimSpace(lineage.DownstreamSource)
	lineage.Relationship = strings.ToLower(strings.TrimSpace(lineage.Relationship))
	if lineage.UpstreamSource == "" || lineage.DownstreamSource == "" || lineage.UpstreamSource == lineage.DownstreamSource {
		return SourceLineage{}, errors.New("source lineage requires two distinct source identifiers")
	}
	switch lineage.Relationship {
	case "primary", "republication", "quotation", "syndication", "common_origin", "independent":
	default:
		return SourceLineage{}, errors.New("invalid source lineage relationship")
	}
	if lineage.ID == "" {
		id, err := newIntelID("lineage")
		if err != nil {
			return SourceLineage{}, err
		}
		lineage.ID = id
	}
	lineage.Confidence = clampConfidence(lineage.Confidence)
	if lineage.DetectedBy == "" {
		lineage.DetectedBy = "operator"
	}
	if lineage.CreatedAt.IsZero() {
		lineage.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.state.SourceLineage {
		if existing.UpstreamSource == lineage.UpstreamSource && existing.DownstreamSource == lineage.DownstreamSource && existing.Relationship == lineage.Relationship {
			return existing, nil
		}
	}
	s.state.SourceLineage = append(s.state.SourceLineage, lineage)
	s.state.Audits = append(s.state.Audits, AuditEvent{At: lineage.CreatedAt, Action: "source.lineage.created", Actor: lineage.DetectedBy, Detail: lineage.ID})
	if err := s.save(); err != nil {
		return SourceLineage{}, err
	}
	s.publish("source.lineage.created", lineage.ID, lineage)
	return lineage, nil
}

func (s *Store) AddClaim(claim Claim) (Claim, error) {
	claim.Subject = strings.TrimSpace(claim.Subject)
	claim.Predicate = strings.TrimSpace(claim.Predicate)
	claim.Object = strings.TrimSpace(claim.Object)
	claim.Statement = strings.TrimSpace(claim.Statement)
	claim.AssertingSourceID = strings.TrimSpace(claim.AssertingSourceID)
	claim.SourceNature = strings.ToLower(strings.TrimSpace(claim.SourceNature))
	if claim.Subject == "" || claim.Predicate == "" || claim.Object == "" || claim.Statement == "" || claim.AssertingSourceID == "" {
		return Claim{}, errors.New("claim subject, predicate, object, statement and asserting source are required")
	}
	if len([]rune(claim.Subject)) > 500 || len([]rune(claim.Predicate)) > 240 || len([]rune(claim.Object)) > 1000 || len([]rune(claim.Statement)) > 4000 {
		return Claim{}, errors.New("claim field boundary violation")
	}
	if claim.SourceNature == "" {
		claim.SourceNature = "unknown"
	}
	switch claim.SourceNature {
	case "primary", "secondary", "unknown":
	default:
		return Claim{}, errors.New("claim source nature is invalid")
	}
	if claim.ID == "" {
		id, err := newIntelID("claim")
		if err != nil {
			return Claim{}, err
		}
		claim.ID = id
	}
	if claim.Status != "" && claim.Status != "unverified" {
		return Claim{}, errors.New("new claims must enter the unverified review state")
	}
	claim.Status = "unverified"
	claim.Confidence = clampConfidence(claim.Confidence)
	if claim.CreatedAt.IsZero() {
		claim.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !sourceExists(s.state.Sources, claim.AssertingSourceID) {
		return Claim{}, fmt.Errorf("asserting source does not exist")
	}
	if err := validateClaimReferences(s.state, claim); err != nil {
		return Claim{}, err
	}
	claim.IndependentSourceCount = independentSourceCount(s.state.Claims, s.state.SourceLineage, claim)
	s.state.Claims = append(s.state.Claims, claim)
	s.state.Audits = append(s.state.Audits, AuditEvent{At: claim.CreatedAt, Action: "claim.created", Actor: "analyst", Detail: claim.ID})
	if err := s.save(); err != nil {
		return Claim{}, err
	}
	s.publish("claim.created", claim.ID, claim)
	return claim, nil
}

func validateClaimReferences(state StoreState, claim Claim) error {
	if len(claim.PassageIDs) == 0 && len(claim.SupportingEvidenceIDs) == 0 && len(claim.ContradictingEvidenceIDs) == 0 {
		return errors.New("claim requires at least one passage or evidence reference")
	}
	passages := make(map[string]Passage, len(state.Passages))
	for _, passage := range state.Passages {
		passages[passage.ID] = passage
	}
	documents := make(map[string]SourceDocument, len(state.Documents))
	for _, document := range state.Documents {
		documents[document.ID] = document
	}
	for _, id := range claim.PassageIDs {
		passage, exists := passages[id]
		if !exists {
			return errors.New("claim passage reference does not exist")
		}
		if document, exists := documents[passage.DocumentID]; !exists || document.SourceID != claim.AssertingSourceID {
			return errors.New("claim passage does not belong to the asserting source")
		}
	}
	evidence := make(map[string]bool, len(state.Evidence))
	for _, item := range state.Evidence {
		evidence[item.ID] = true
	}
	for _, id := range append(append([]string(nil), claim.SupportingEvidenceIDs...), claim.ContradictingEvidenceIDs...) {
		if !evidence[id] {
			return errors.New("claim evidence reference does not exist")
		}
	}
	if claim.Confidence > 0 && strings.TrimSpace(claim.CalibrationBasis) == "" {
		return errors.New("claim confidence requires a calibration basis")
	}
	return nil
}

func (s *Store) ReviewClaim(id, status, actor, reason string) (Claim, error) {
	id, status, actor, reason = strings.TrimSpace(id), strings.ToLower(strings.TrimSpace(status)), strings.TrimSpace(actor), strings.TrimSpace(reason)
	if id == "" || actor == "" || reason == "" || len([]rune(reason)) > 2000 {
		return Claim{}, errors.New("claim review requires identifier, actor and bounded reason")
	}
	switch status {
	case "corroborated", "verified", "disputed", "rejected":
	default:
		return Claim{}, errors.New("claim review status is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.state.Claims {
		claim := &s.state.Claims[index]
		if claim.ID != id {
			continue
		}
		if status == "verified" && claim.IndependentSourceCount < 1 {
			return Claim{}, errors.New("claim cannot be verified without an attributable source")
		}
		now := time.Now().UTC()
		claim.Status, claim.ReviewedAt, claim.ReviewedBy = status, now, actor
		s.state.Audits = append(s.state.Audits, AuditEvent{At: now, Action: "claim." + status, Actor: actor, Detail: claim.ID + ": " + reason})
		if err := s.save(); err != nil {
			return Claim{}, err
		}
		updated := *claim
		s.publish("claim.reviewed", claim.ID, map[string]any{"claim": updated, "reason": reason})
		return updated, nil
	}
	return Claim{}, errors.New("claim not found")
}

func sourceExists(sources []Source, id string) bool {
	for _, source := range sources {
		if source.ID == id {
			return true
		}
	}
	return false
}

func independentSourceCount(claims []Claim, lineage []SourceLineage, candidate Claim) int {
	sources := map[string]bool{candidate.AssertingSourceID: true}
	for _, claim := range claims {
		if normalizeHashInput(claim.Statement) == normalizeHashInput(candidate.Statement) {
			sources[claim.AssertingSourceID] = true
		}
	}
	parent := make(map[string]string, len(sources))
	for source := range sources {
		parent[source] = source
	}
	var root func(string) string
	root = func(source string) string {
		if parent[source] != source {
			parent[source] = root(parent[source])
		}
		return parent[source]
	}
	for _, edge := range lineage {
		if edge.Relationship == "independent" || !sources[edge.UpstreamSource] || !sources[edge.DownstreamSource] {
			continue
		}
		leftRoot := root(edge.UpstreamSource)
		rightRoot := root(edge.DownstreamSource)
		if leftRoot != rightRoot {
			parent[rightRoot] = leftRoot
		}
	}
	groups := make(map[string]bool, len(sources))
	for source := range sources {
		groups[root(source)] = true
	}
	return len(groups)
}

func (s *Store) CreateHypothesis(hypothesis Hypothesis) (Hypothesis, error) {
	hypothesis.CaseID = strings.TrimSpace(hypothesis.CaseID)
	hypothesis.Statement = strings.TrimSpace(hypothesis.Statement)
	if hypothesis.CaseID == "" || hypothesis.Statement == "" {
		return Hypothesis{}, errors.New("hypothesis requires case and statement")
	}
	if hypothesis.ID == "" {
		id, err := newIntelID("hypothesis")
		if err != nil {
			return Hypothesis{}, err
		}
		hypothesis.ID = id
	}
	now := time.Now().UTC()
	hypothesis.Confidence = clampConfidence(hypothesis.Confidence)
	hypothesis.Status = "active"
	hypothesis.CreatedAt = now
	hypothesis.UpdatedAt = now
	hypothesis.ConfidenceHistory = append(hypothesis.ConfidenceHistory, ConfidencePoint{At: now, Confidence: hypothesis.Confidence, Reason: "initial assessment", Actor: "analyst"})

	s.mu.Lock()
	defer s.mu.Unlock()
	if !caseExists(s.state.Cases, hypothesis.CaseID) {
		return Hypothesis{}, errors.New("case not found")
	}
	s.state.Hypotheses = append(s.state.Hypotheses, hypothesis)
	s.state.Audits = append(s.state.Audits, AuditEvent{At: now, Action: "hypothesis.created", Actor: "analyst", Detail: hypothesis.ID})
	if err := s.save(); err != nil {
		return Hypothesis{}, err
	}
	s.publish("hypothesis.created", hypothesis.ID, hypothesis)
	return hypothesis, nil
}

func caseExists(cases []Case, id string) bool {
	for _, candidate := range cases {
		if candidate.ID == id {
			return true
		}
	}
	return false
}

func (s *Store) UpdateHypothesisConfidence(id string, confidence int, reason, actor string) (Hypothesis, error) {
	id = strings.TrimSpace(id)
	reason = strings.TrimSpace(reason)
	actor = strings.TrimSpace(actor)
	if id == "" || reason == "" {
		return Hypothesis{}, errors.New("hypothesis id and change reason are required")
	}
	if actor == "" {
		actor = "operator"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.state.Hypotheses {
		if s.state.Hypotheses[index].ID != id {
			continue
		}
		now := time.Now().UTC()
		s.state.Hypotheses[index].Confidence = clampConfidence(confidence)
		s.state.Hypotheses[index].UpdatedAt = now
		s.state.Hypotheses[index].ConfidenceHistory = append(s.state.Hypotheses[index].ConfidenceHistory, ConfidencePoint{At: now, Confidence: s.state.Hypotheses[index].Confidence, Reason: reason, Actor: actor})
		if err := s.save(); err != nil {
			return Hypothesis{}, err
		}
		updated := s.state.Hypotheses[index]
		s.publish("hypothesis.updated", id, updated)
		return updated, nil
	}
	return Hypothesis{}, errors.New("hypothesis not found")
}

func (s *Store) CreateInformationGap(gap InformationGap) (InformationGap, error) {
	gap.CaseID = strings.TrimSpace(gap.CaseID)
	gap.Question = strings.TrimSpace(gap.Question)
	gap.Rationale = strings.TrimSpace(gap.Rationale)
	gap.Priority = strings.ToLower(strings.TrimSpace(gap.Priority))
	if gap.CaseID == "" || gap.Question == "" {
		return InformationGap{}, errors.New("information gap requires case and question")
	}
	switch gap.Priority {
	case "low", "medium", "high", "critical":
	default:
		return InformationGap{}, errors.New("invalid information gap priority")
	}
	id, err := newIntelID("gap")
	if err != nil {
		return InformationGap{}, err
	}
	gap.ID = id
	gap.Status = "open"
	gap.CreatedAt = time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !caseExists(s.state.Cases, gap.CaseID) {
		return InformationGap{}, errors.New("case not found")
	}
	s.state.InformationGaps = append(s.state.InformationGaps, gap)
	s.state.Audits = append(s.state.Audits, AuditEvent{At: gap.CreatedAt, Action: "information-gap.created", Actor: "analyst", Detail: gap.ID})
	if err := s.save(); err != nil {
		return InformationGap{}, err
	}
	s.publish("information-gap.created", gap.ID, gap)
	return gap, nil
}

func (s *Store) CreateCollectionPlan(plan CollectionPlan) (CollectionPlan, error) {
	plan.CaseID = strings.TrimSpace(plan.CaseID)
	plan.InformationGapID = strings.TrimSpace(plan.InformationGapID)
	plan.OwnerProfile = strings.ToLower(strings.TrimSpace(plan.OwnerProfile))
	if plan.CaseID == "" || plan.InformationGapID == "" || len(plan.SourceTypes) == 0 || len(plan.Queries) == 0 {
		return CollectionPlan{}, errors.New("collection plan requires case, gap, source types and queries")
	}
	switch plan.OwnerProfile {
	case "collector", "case_worker", "operator":
	default:
		return CollectionPlan{}, errors.New("invalid collection plan owner profile")
	}
	id, err := newIntelID("collection-plan")
	if err != nil {
		return CollectionPlan{}, err
	}
	now := time.Now().UTC()
	plan.ID = id
	plan.Status = "proposed"
	plan.CreatedAt = now
	plan.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	if !caseExists(s.state.Cases, plan.CaseID) || !informationGapExists(s.state.InformationGaps, plan.CaseID, plan.InformationGapID) {
		return CollectionPlan{}, errors.New("case or information gap not found")
	}
	s.state.CollectionPlans = append(s.state.CollectionPlans, plan)
	s.state.Audits = append(s.state.Audits, AuditEvent{At: now, Action: "collection-plan.created", Actor: "analyst", Detail: plan.ID})
	if err := s.save(); err != nil {
		return CollectionPlan{}, err
	}
	s.publish("collection-plan.created", plan.ID, plan)
	return plan, nil
}

func informationGapExists(gaps []InformationGap, caseID, gapID string) bool {
	for _, gap := range gaps {
		if gap.CaseID == caseID && gap.ID == gapID {
			return true
		}
	}
	return false
}
