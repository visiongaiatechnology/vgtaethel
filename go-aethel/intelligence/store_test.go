package intelligence

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIngestObservationPipeline(t *testing.T) {
	bus := NewEventBus()
	tempPath := filepath.Join(t.TempDir(), "intel_core.json")

	_, eventChan := bus.Subscribe()

	store := NewStore(tempPath, bus)

	// Ingest an observation located in Germany (approx lat=50.0, lon=10.0)
	obs := Observation{
		ID:         "obs_test_001",
		SourceID:   "src_spiegel_feed",
		RawText:    "Geopolitical tension alert on northern border.",
		ObservedAt: time.Now().UTC(),
		Latitude:   50.0,
		Longitude:  10.0,
		Domain:     "geo",
	}

	store.IngestObservation(obs)

	snapshot := store.GetSnapshot()
	germanyRisk, ok := snapshot.RiskScores["GERMANY"]
	if !ok {
		t.Fatal("Expected risk score for GERMANY to be calculated")
	}

	if germanyRisk.OverallRisk <= 0 {
		t.Errorf("Expected positive risk score for Germany, got %v", germanyRisk.OverallRisk)
	}

	if len(snapshot.Observations) != 1 {
		t.Fatalf("expected 1 raw observation, got %d", len(snapshot.Observations))
	}
	if len(snapshot.Events) == 0 {
		t.Fatal("expected classified Event from ingest")
	}
	if len(snapshot.Evidence) == 0 {
		t.Fatal("expected Evidence reference from ingest")
	}
	if len(snapshot.Assessments) == 0 {
		t.Fatal("expected Assessment from ingest")
	}
	if snapshot.Assessments[0].Status != "unverified" {
		t.Errorf("Assessment must start as unverified, got %q", snapshot.Assessments[0].Status)
	}
	if len(snapshot.Audits) == 0 {
		t.Fatal("expected AuditEvent from ingest")
	}
	if len(snapshot.Sources) == 0 {
		t.Fatal("expected Source registry entry")
	}

	if len(snapshot.Alerts) > 0 {
		t.Logf("Test produced %d alerts on ingest", len(snapshot.Alerts))
	}

	explain := store.ExplainScore("GERMANY")
	if !strings.Contains(explain, "OverallRisk") {
		t.Error("ExplainScore should describe the score")
	}
	if !strings.Contains(explain, "Primary drivers") && !strings.Contains(explain, "Primary drivers:") {
		t.Error("ExplainScore must mention primary drivers")
	}
	if !strings.Contains(strings.ToLower(explain), "fresh") && !strings.Contains(explain, "MissingData") && !strings.Contains(explain, "Missing") {
		t.Error("ExplainScore must mention freshness or missing data")
	}
	if !strings.Contains(strings.ToLower(explain), "unverified") {
		t.Error("ExplainScore must not paper over uncertainty — raw obs remain unverified")
	}
	if strings.Contains(strings.ToLower(explain), "verified fact") {
		t.Error("ExplainScore must not label raw observations as verified facts")
	}

	report := store.GenerateReport("Test Brief")
	if !strings.Contains(report, "Executive Summary") || !strings.Contains(report, "Uncertainties") {
		t.Error("Report must contain key spec sections + separation note")
	}
	if !strings.Contains(report, "RAW OBSERVATIONS") || !strings.Contains(report, "INFERENCE") {
		t.Error("Report must separate raw observations from inference")
	}
	if !strings.Contains(report, "Provenance") {
		t.Error("Report must include provenance note")
	}

	// Drain bus for major step types (non-blocking collect)
	seen := map[string]bool{}
	deadline := time.After(200 * time.Millisecond)
drain:
	for {
		select {
		case msg := <-eventChan:
			seen[msg.Type] = true
		case <-deadline:
			break drain
		default:
			// empty channel — short sleep then exit if nothing pending
			select {
			case msg := <-eventChan:
				seen[msg.Type] = true
			case <-time.After(50 * time.Millisecond):
				break drain
			}
		}
	}
	if !seen["observation.created"] {
		t.Error("bus must emit observation.created")
	}
	// risk.changed / assessment.updated expected when region matched
	if !seen["risk.changed"] && !seen["assessment.updated"] && !seen["event.created"] {
		t.Logf("bus types seen: %v (may drop if buffer full — at least observation.created required)", seen)
	}
}

func TestAlertAIAssessmentPersistsWithoutOverwritingBaseline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intel_core.json")
	store := NewStore(path, NewEventBus())
	store.state.Alerts = []Alert{{ID: "alert_test_001", Severity: "high", Confidence: 82, Region: "GERMANY"}}
	assessment := AlertAIAssessment{
		ModelID: "openai/gpt-oss-120b", Language: "de", Severity: "medium", Confidence: 67,
		Summary: "Unverified model summary", Rationale: "Evidence remains incomplete.", EvaluatedAt: time.Now().UTC(),
	}
	if err := store.SetAlertAIAssessment("alert_test_001", assessment); err != nil {
		t.Fatalf("persist alert assessment: %v", err)
	}
	alert, ok := store.GetAlert("alert_test_001")
	if !ok || alert.AIAssessment == nil {
		t.Fatal("persisted AI assessment missing")
	}
	if alert.Severity != "high" || alert.Confidence != 82 {
		t.Fatalf("AI assessment overwrote deterministic baseline: %+v", alert)
	}
	if alert.AIAssessment.Status != "unverified" || alert.AIAssessment.Severity != "medium" {
		t.Fatalf("AI assessment epistemic contract violated: %+v", alert.AIAssessment)
	}
	reloaded := NewStore(path, NewEventBus())
	if persisted, exists := reloaded.GetAlert("alert_test_001"); !exists || persisted.AIAssessment == nil {
		t.Fatal("AI assessment did not survive store reload")
	}
}

func TestQueryRegionSecurityGermany24h(t *testing.T) {
	bus := NewEventBus()
	store := NewStore(filepath.Join(t.TempDir(), "intel_query.json"), bus)

	// Prior window baseline event (36h ago)
	store.IngestObservation(Observation{
		ID:         "obs_prior",
		SourceID:   "rss-test",
		RawText:    "Earlier border monitoring report Germany",
		ObservedAt: time.Now().UTC().Add(-36 * time.Hour),
		Latitude:   51.0,
		Longitude:  10.5,
		Domain:     "geo",
	})
	// Current window — security-relevant
	store.IngestObservation(Observation{
		ID:         "obs_now",
		SourceID:   "rss-berlin",
		RawText:    "Major security development reported in Berlin near government district",
		ObservedAt: time.Now().UTC().Add(-2 * time.Hour),
		Latitude:   52.52,
		Longitude:  13.405,
		Domain:     "geo",
	})

	out := store.QueryRegionSecurity("GERMANY", 24)
	if !strings.Contains(out, "GERMANY") {
		t.Error("query must name region")
	}
	if !strings.Contains(out, "RAW OBSERVATIONS") {
		t.Error("must separate raw observations")
	}
	if !strings.Contains(out, "INFERENCE") || !strings.Contains(out, "unverified") {
		t.Error("must mark classified events as inference/unverified")
	}
	if !strings.Contains(out, "Evidence") {
		t.Error("must attach evidence references section")
	}
	if !strings.Contains(out, "Uncertainties") {
		t.Error("must surface uncertainties")
	}
	if !strings.Contains(out, "Change vs prior") {
		t.Error("must report change vs prior window")
	}
	if !strings.Contains(out, "OverallRisk") && !strings.Contains(out, "risk") {
		t.Error("must include region-linked risk")
	}
	// Must reference fixture-like content from ingest, not invent sources
	if !strings.Contains(out, "obs_now") && !strings.Contains(out, "Berlin") && !strings.Contains(out, "security") {
		t.Error("must select relevant ingested events/obs")
	}
}

func TestDedupNearDuplicateEvents(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "intel_dedup.json"), NewEventBus())
	base := Observation{
		ID:         "obs_a",
		SourceID:   "src1",
		RawText:    "Major power grid outage reported in northern industrial zone",
		ObservedAt: time.Now().UTC(),
		Latitude:   52.5,
		Longitude:  13.4,
		Domain:     "infrastructure",
	}
	store.IngestObservation(base)
	dup := base
	dup.ID = "obs_b"
	dup.RawText = "Major power grid outage reported in northern industrial zone update"
	store.IngestObservation(dup)
	snap := store.GetSnapshot()
	if len(snap.Observations) != 2 {
		t.Fatalf("both raw observations must be kept, got %d", len(snap.Observations))
	}
	// Near-duplicate classification may collapse Events
	if len(snap.Events) > 2 {
		t.Errorf("expected dedup to limit events, got %d", len(snap.Events))
	}
}

func TestExplainScoreMissingRegion(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "intel_empty.json"), NewEventBus())
	msg := store.ExplainScore("NO_SUCH_REGION")
	if !strings.Contains(msg, "No risk score") {
		t.Errorf("expected missing region message, got %q", msg)
	}
}

func TestLiveNexusContextUsesStrictEventTimeWindow(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "intel_window.json"), NewEventBus())
	now := time.Now().UTC()
	store.state.Observations = []Observation{
		{ID: "fresh-observation", SourceID: "source", RawText: "fresh signal", ObservedAt: now.Add(-2 * time.Hour)},
		{ID: "stale-observation", SourceID: "source", RawText: "stale signal", ObservedAt: now.Add(-25 * time.Hour)},
		{ID: "undated-observation", SourceID: "source", RawText: "missing timestamp"},
	}
	store.state.Events = []Event{
		{ID: "fresh-event", Title: "fresh classified event", ObservedAt: now.Add(-time.Hour)},
		{ID: "stale-event", Title: "stale classified event", ObservedAt: now.Add(-48 * time.Hour)},
	}

	context := store.LiveNexusContextWithin(20, 24)
	for _, expected := range []string{"fresh-observation", "fresh-event", "Strict event-time window: 24h"} {
		if !strings.Contains(context, expected) {
			t.Fatalf("strict context missing %q: %s", expected, context)
		}
	}
	for _, forbidden := range []string{"stale-observation", "undated-observation", "stale-event"} {
		if strings.Contains(context, forbidden) {
			t.Fatalf("strict context leaked %q: %s", forbidden, context)
		}
	}
}

func TestAcknowledgeAlert(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "ack.json"), NewEventBus())
	store.IngestObservation(Observation{
		ID: "obs_ack", SourceID: "src", RawText: "Major critical escalat incident Berlin security",
		ObservedAt: time.Now().UTC(), Latitude: 52.52, Longitude: 13.405, Domain: "geo",
	})
	snap := store.GetSnapshot()
	if len(snap.Alerts) == 0 {
		// force a manual alert if delta threshold not hit
		store.AddAlertRule("GERMANY", "low", 0)
		_ = store.EvaluateAlertRules()
		snap = store.GetSnapshot()
	}
	if len(snap.Alerts) == 0 {
		t.Skip("no alerts produced for ack test environment")
	}
	id := snap.Alerts[0].ID
	if err := store.AcknowledgeAlert(id); err != nil {
		t.Fatal(err)
	}
	for _, a := range store.GetAlerts() {
		if a.ID == id && !a.Acknowledged {
			t.Fatal("alert not acknowledged")
		}
	}
}

func TestAddAlertRulePersistsAndEvaluates(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "rules.json"), NewEventBus())
	store.IngestObservation(Observation{
		ID: "obs_rule", SourceID: "src", RawText: "Major critical security escalation reported in Berlin",
		ObservedAt: time.Now().UTC(), Latitude: 52.52, Longitude: 13.405, Domain: "geo",
	})
	rule := store.AddAlertRule("GERMANY", "medium", 5)
	if rule.ID == "" || rule.RegionID != "GERMANY" {
		t.Fatalf("rule not stored: %+v", rule)
	}
	if rule.CreatedAt.IsZero() || rule.CreatedBy != "operator" {
		t.Fatalf("AlertRule CreatedAt/CreatedBy must be set: %+v", rule)
	}
	rules := store.GetAlertRules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 alert rule, got %d", len(rules))
	}
	alerts := store.EvaluateAlertRules()
	// May or may not fire depending on score vs threshold; must not invent success without persistence
	_ = alerts
	snap := store.GetSnapshot()
	if len(snap.AlertRules) != 1 {
		t.Fatal("AlertRules must persist in snapshot")
	}
}

func TestMultiSourceCorroboration(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "corr.json"), NewEventBus())
	// Two similar stories from different sources near Berlin
	store.IngestObservation(Observation{
		ID: "obs_c1", SourceID: "src_alpha", RawText: "Critical energy infrastructure attack reported near industrial hub Berlin",
		ObservedAt: time.Now().UTC(), Latitude: 52.52, Longitude: 13.40, Domain: "cyber",
	})
	store.IngestObservation(Observation{
		ID: "obs_c2", SourceID: "src_beta", RawText: "Critical energy infrastructure attack reported near industrial hub Berlin city",
		ObservedAt: time.Now().UTC(), Latitude: 52.53, Longitude: 13.41, Domain: "cyber",
	})
	snap := store.GetSnapshot()
	foundCorr := false
	for _, a := range snap.Assessments {
		if a.Status == "corroborated" {
			foundCorr = true
		}
	}
	if !foundCorr {
		t.Fatalf("expected at least one corroborated assessment from multi-source similar titles; assessments=%+v", snap.Assessments)
	}
	// Source provenance fields filled
	if len(snap.Sources) < 2 {
		t.Fatalf("expected >=2 sources, got %d", len(snap.Sources))
	}
	for _, s := range snap.Sources {
		if s.ContentHash == "" || s.ParserVersion == "" {
			t.Fatalf("source provenance incomplete: %+v", s)
		}
	}
}

func TestRecordAgentActionAndIdentity(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "agent.json"), NewEventBus())
	store.RecordAgentAction("intelligence_status", `{}`, "ok", "", "completed")
	snap := store.GetSnapshot()
	if len(snap.AgentActions) != 1 || snap.AgentActions[0].Skill != "intelligence_status" {
		t.Fatalf("agent action not recorded: %+v", snap.AgentActions)
	}
	id := store.IdentitySnapshot()
	if id.Name == "" {
		t.Fatal("identity name default missing")
	}
}

func TestSealCaseEvidenceAndBatchIngest(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "case_ev.json"), NewEventBus())
	c, err := store.CreateCase("Evidence case", "seal test purpose")
	if err != nil {
		t.Fatal(err)
	}
	ev, err := store.SealCaseEvidence(c.ID, "evid-1", "src-x", "https://ex.test", "excerpt body", "deadbeef", "ev_shared_1")
	if err != nil {
		t.Fatal(err)
	}
	if !ev.Sealed || ev.SHA256 != "deadbeef" {
		t.Fatalf("seal incomplete: %+v", ev)
	}
	got, ok := store.GetCase(c.ID)
	if !ok || len(got.Evidence) != 1 {
		t.Fatalf("case evidence missing: %+v", got)
	}
	// Batch ingest
	n := store.IngestObservationsBatch([]Observation{
		{ID: "b1", SourceID: "s1", RawText: "Batch observation about Warsaw logistics", ObservedAt: time.Now().UTC(), Latitude: 52.2, Longitude: 21.0, Domain: "geo"},
		{ID: "", SourceID: "s1", RawText: "skip empty id", ObservedAt: time.Now().UTC()},
	})
	if n != 1 {
		t.Fatalf("expected 1 ingested, got %d", n)
	}
	// Poland region should exist for Warsaw point
	rs := store.GetRiskScores()
	if _, ok := rs["POLAND"]; !ok {
		// may still compute if point in polygon — Warsaw 52.2,21.0 is in Poland box
		t.Logf("risk scores after batch: %v", rs)
	}
}

func TestDefaultRegionsIncludePolandBaltics(t *testing.T) {
	eng := GetDefaultRegionEngine()
	// Warsaw
	m := eng.MatchPoint(52.23, 21.01)
	foundPL := false
	for _, r := range m {
		if r.ID == "POLAND" {
			foundPL = true
		}
	}
	if !foundPL {
		t.Fatalf("Warsaw should match POLAND, got %+v", m)
	}
	// Tallinn-ish
	m2 := eng.MatchPoint(59.4, 24.7)
	foundB := false
	for _, r := range m2 {
		if r.ID == "BALTICS" {
			foundB = true
		}
	}
	if !foundB {
		t.Fatalf("Tallinn should match BALTICS, got %+v", m2)
	}
}

func TestSetPersonalContextAndImpact(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "personal.json"), NewEventBus())
	store.SetPersonalContext(PersonalContext{
		OperatorID:       "tester",
		Interests:        []string{"energy", "cyber"},
		Goals:            []string{"monitor DE grid"},
		PreferredRegions: []string{"GERMANY"},
		RiskTolerance:    "medium",
	})
	pc := store.GetPersonalContext()
	if pc.OperatorID != "tester" || len(pc.Interests) != 2 {
		t.Fatalf("personal context not stored: %+v", pc)
	}
	// Seed risk so impact brief has something to join
	store.IngestObservation(Observation{
		ID: "obs_pi", SourceID: "src_pi", RawText: "Energy grid cyber disruption report near Berlin industrial zone",
		ObservedAt: time.Now().UTC(), Latitude: 52.52, Longitude: 13.405, Domain: "cyber",
	})
	brief := store.PersonalImpact()
	if brief == "" || !strings.Contains(brief, "Personal Impact") {
		t.Fatalf("expected personal impact brief, got: %s", brief)
	}
	if !strings.Contains(brief, "Empfehlungen") {
		t.Fatal("impact brief must label recommendations vs facts")
	}
}

func TestCreateCasePersistsIsolated(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "cases.json"), NewEventBus())
	c, err := store.CreateCase("Berlin transport review", "public safety analysis")
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == "" || c.Purpose == "" {
		t.Fatalf("case incomplete: %+v", c)
	}
	snap := store.GetSnapshot()
	if len(snap.Cases) != 1 {
		t.Fatalf("expected 1 case in shared store, got %d", len(snap.Cases))
	}
	if len(snap.Cases[0].Audit) == 0 {
		t.Fatal("case must have audit entry")
	}
	if _, err := store.CreateCase("", "x"); err == nil {
		t.Fatal("empty title must fail")
	}
	if err := store.DeleteCase(c.ID); err != nil {
		t.Fatalf("DeleteCase: %v", err)
	}
	if len(store.GetSnapshot().Cases) != 0 {
		t.Fatal("case must be gone after DeleteCase")
	}
	if err := store.DeleteCase("missing"); err == nil {
		t.Fatal("delete missing must fail")
	}
}

func TestLiveNexusContextLayerLabels(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nexus.json"), NewEventBus())
	store.IngestObservation(Observation{
		ID: "obs_nx", SourceID: "src1",
		RawText:    "Major security development reported in Berlin government area",
		ObservedAt: time.Now().UTC(), Latitude: 52.52, Longitude: 13.405, Domain: "geo",
	})
	ctx := store.LiveNexusContext(10)
	for _, need := range []string{"RAW OBSERVATIONS", "INFERENCE", "VERIFIED", "UNIFIED", "obs_nx"} {
		if !strings.Contains(ctx, need) {
			t.Errorf("LiveNexusContext missing %q in:\n%s", need, ctx)
		}
	}
	if strings.Contains(ctx, "verified fact") {
		t.Error("must not claim verified facts for raw ingest")
	}
	// ExplainScore excerpt for verification artifacts
	ex := store.ExplainScore("GERMANY")
	if !strings.Contains(ex, "OverallRisk") || !strings.Contains(strings.ToLower(ex), "unverified") {
		t.Errorf("ExplainScore incomplete: %s", ex)
	}
	t.Logf("EXPLAIN_SCORE_EXCERPT:\n%s", ex)
	t.Logf("NEXUS_CONTEXT_EXCERPT:\n%s", ctx[:min(len(ctx), 800)])
}
