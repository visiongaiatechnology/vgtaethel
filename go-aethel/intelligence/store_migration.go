package intelligence

// STATUS: DIAMANT VGT SUPREME

import "time"

func migrateStoreState(state *StoreState) bool {
	changed := state.SchemaVersion != CurrentStoreSchemaVersion
	state.SchemaVersion = CurrentStoreSchemaVersion
	if state.Sources == nil {
		state.Sources = []Source{}
		changed = true
	}
	if state.Documents == nil {
		state.Documents = []SourceDocument{}
		changed = true
	}
	if state.Passages == nil {
		state.Passages = []Passage{}
		changed = true
	}
	if state.WebsiteMonitors == nil {
		state.WebsiteMonitors = []WebsiteMonitor{}
		changed = true
	}
	if state.WebsiteChanges == nil {
		state.WebsiteChanges = []WebsiteChange{}
		changed = true
	}
	if state.EntityVersions == nil {
		state.EntityVersions = []EntityVersion{}
		changed = true
	}
	if state.Observations == nil {
		state.Observations = []Observation{}
		changed = true
	}
	if state.Events == nil {
		state.Events = []Event{}
		changed = true
	}
	if state.Claims == nil {
		state.Claims = []Claim{}
		changed = true
	}
	for index := range state.Claims {
		if state.Claims[index].SourceNature == "" {
			state.Claims[index].SourceNature = "unknown"
			changed = true
		}
	}
	if state.SourceLineage == nil {
		state.SourceLineage = []SourceLineage{}
		changed = true
	}
	if state.Hypotheses == nil {
		state.Hypotheses = []Hypothesis{}
		changed = true
	}
	if state.InformationGaps == nil {
		state.InformationGaps = []InformationGap{}
		changed = true
	}
	if state.CollectionPlans == nil {
		state.CollectionPlans = []CollectionPlan{}
		changed = true
	}
	if state.ResolvedEntities == nil {
		state.ResolvedEntities = []ResolvedEntity{}
		changed = true
	}
	if state.ResolutionCandidates == nil {
		state.ResolutionCandidates = []EntityResolutionCandidate{}
		changed = true
	}
	if state.ResolutionDecisions == nil {
		state.ResolutionDecisions = []EntityResolutionDecision{}
		changed = true
	}
	if state.HypothesisEvidenceAssessments == nil {
		state.HypothesisEvidenceAssessments = []HypothesisEvidenceAssessment{}
		changed = true
	}
	if state.SavedSearches == nil {
		state.SavedSearches = []SavedSearchMonitor{}
		changed = true
	}
	if state.SearchAlerts == nil {
		state.SearchAlerts = []SearchMonitorAlert{}
		changed = true
	}
	if state.ImageFingerprints == nil {
		state.ImageFingerprints = []ImageFingerprint{}
		changed = true
	}
	if state.ImportedDocuments == nil {
		state.ImportedDocuments = []ImportedDocument{}
		changed = true
	}
	if state.CustodyEvents == nil {
		state.CustodyEvents = []CustodyEvent{}
		changed = true
	}
	if state.Migrations == nil {
		state.Migrations = make(map[string]time.Time)
		changed = true
	}
	if state.Assessments == nil {
		state.Assessments = []Assessment{}
		changed = true
	}
	if state.Evidence == nil {
		state.Evidence = []Evidence{}
		changed = true
	}
	if state.Cases == nil {
		state.Cases = []Case{}
		changed = true
	}
	if state.RiskScores == nil {
		state.RiskScores = make(map[string]RiskScore)
		changed = true
	}
	if state.Alerts == nil {
		state.Alerts = []Alert{}
		changed = true
	}
	if state.AlertRules == nil {
		state.AlertRules = []AlertRule{}
		changed = true
	}
	if state.Watchlists == nil {
		state.Watchlists = []Watchlist{}
		changed = true
	}
	if state.Audits == nil {
		state.Audits = []AuditEvent{}
		changed = true
	}
	if state.AgentActions == nil {
		state.AgentActions = []AgentAction{}
		changed = true
	}
	if state.Briefings == nil {
		state.Briefings = []Briefing{}
		changed = true
	}

	observationHashes := make(map[string][2]string, len(state.Observations))
	for index := range state.Observations {
		observation := &state.Observations[index]
		rawHash := contentSHA256(observation.RawText)
		normalizedHash := contentSHA256(normalizeHashInput(observation.RawText))
		if observation.RawSHA256 != rawHash || observation.NormalizedSHA256 != normalizedHash || observation.ContentHash != rawHash {
			observation.RawSHA256 = rawHash
			observation.NormalizedSHA256 = normalizedHash
			observation.ContentHash = rawHash
			changed = true
		}
		if observation.ParserVersion == "" {
			observation.ParserVersion = "legacy-migrated-v2"
			changed = true
		}
		flags := DetectInstructionSignals(observation.RawText)
		if len(flags) > 0 && !observation.Quarantined {
			observation.InstructionFlags = flags
			observation.Quarantined = true
			changed = true
		}
		observationHashes[observation.ID] = [2]string{rawHash, normalizedHash}
	}
	if migratePassages(state) {
		changed = true
	}

	for index := range state.Evidence {
		if migrateEvidenceHashes(&state.Evidence[index], observationHashes) {
			changed = true
		}
	}
	for caseIndex := range state.Cases {
		for evidenceIndex := range state.Cases[caseIndex].Evidence {
			if migrateEvidenceHashes(&state.Cases[caseIndex].Evidence[evidenceIndex], observationHashes) {
				changed = true
			}
		}
	}
	return changed
}

func migratePassages(state *StoreState) bool {
	changed := false
	existing := make(map[string]bool, len(state.Passages))
	for _, passage := range state.Passages {
		existing[passage.ID] = true
	}
	for _, observation := range state.Observations {
		id := "passage-" + observation.ID
		if existing[id] {
			continue
		}
		state.Passages = append(state.Passages, Passage{
			ID: id, DocumentID: "doc-" + observation.ID, Text: observation.RawText,
			StartOffset: 0, EndOffset: len([]rune(observation.RawText)),
		})
		existing[id] = true
		changed = true
	}
	return changed
}

func migrateEvidenceHashes(evidence *Evidence, observations map[string][2]string) bool {
	changed := false
	observationID := evidence.ID
	if len(observationID) > len("evid-") && observationID[:len("evid-")] == "evid-" {
		observationID = observationID[len("evid-"):]
	}
	if hashes, found := observations[observationID]; found {
		if evidence.SHA256 != hashes[0] || evidence.RawSHA256 != hashes[0] || evidence.NormalizedSHA256 != hashes[1] || evidence.HashAlgorithm != "sha256" {
			evidence.SHA256 = hashes[0]
			evidence.RawSHA256 = hashes[0]
			evidence.NormalizedSHA256 = hashes[1]
			evidence.HashAlgorithm = "sha256"
			changed = true
		}
	} else if evidence.HashAlgorithm == "" {
		if len(evidence.SHA256) == 64 {
			evidence.HashAlgorithm = "sha256"
		} else {
			evidence.HashAlgorithm = "legacy-unverified"
		}
		changed = true
	}
	if evidence.CaptureScope == "" {
		evidence.CaptureScope = "excerpt"
		changed = true
	}
	return changed
}
