package main

// STATUS: PLATIN

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go-aethel/intelligence"
	"go-aethel/osint"
	"go-aethel/skills"
)

func TestIntelligenceCaseDelete(t *testing.T) {
	store := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "del_case.json"))
	c, err := store.CreateCase("Temp delete case", "cleanup test")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteCase(c.ID); err != nil {
		t.Fatalf("DeleteCase: %v", err)
	}
	snap := store.Snapshot()
	for _, remaining := range snap.Cases {
		if remaining.ID == c.ID {
			t.Fatal("deleted case still present")
		}
	}
	if err := store.DeleteCase("no-such"); err == nil {
		t.Fatal("missing case must error")
	}
}

func TestIntelligenceCaseEvidenceAndPseudonymIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intelligence", "state.json")
	store := intelligence.NewIntelligenceStore(path)
	first, err := store.CreateCase("Public incident review", "public safety analysis")
	if err != nil {
		t.Fatalf("create first case: %v", err)
	}
	second, err := store.CreateCase("Independent review", "infrastructure analysis")
	if err != nil {
		t.Fatalf("create second case: %v", err)
	}
	evidence, err := store.SealEvidenceWithEvent(first.ID, "Public source", "https://example.invalid/source", "A verifiable observation.", "operator", "ev_shared_abc")
	if err != nil {
		t.Fatalf("seal evidence: %v", err)
	}
	if !evidence.Sealed || len(evidence.SHA256) != 64 {
		t.Fatalf("evidence seal missing: %#v", evidence)
	}
	if evidence.SourceEventID != "ev_shared_abc" {
		t.Fatalf("source_event_id not stored on evidence: %#v", evidence)
	}
	if evidence.ValidationStatus != "pending" {
		t.Fatalf("new evidence must await human validation: %#v", evidence)
	}
	// Audit must record promote provenance
	snap := store.Snapshot()
	foundPromote := false
	for _, c := range snap.Cases {
		if c.ID != first.ID {
			continue
		}
		for _, a := range c.Audit {
			if a.Action == "event.promoted" && strings.Contains(a.Detail, "ev_shared_abc") {
				foundPromote = true
			}
		}
	}
	if !foundPromote {
		t.Fatal("case audit must contain event.promoted with source_event_id")
	}
	validated, err := store.ValidateEvidence(first.ID, evidence.ID, "operator", "verified")
	if err != nil || validated.ValidationStatus != "verified" {
		t.Fatalf("human validation failed: %#v / %v", validated, err)
	}
	// Case-scoped person entities must produce distinct pseudonyms across cases
	ent1, err := store.AddEntity(first.ID, "Sample Identifier", "person", 80)
	if err != nil {
		t.Fatalf("first entity: %v", err)
	}
	ent2, err := store.AddEntity(second.ID, "Sample Identifier", "person", 80)
	if err != nil {
		t.Fatalf("second entity: %v", err)
	}
	if ent1.Label == ent2.Label {
		t.Fatal("case-scoped aliases must differ")
	}
	if !strings.HasPrefix(ent1.Label, "GX-PER-") {
		t.Fatalf("person must be pseudonymized, got %s", ent1.Label)
	}
	loaded := intelligence.NewIntelligenceStore(path).Snapshot()
	if len(loaded.Cases) != 2 || len(loaded.Cases[0].Evidence) != 1 {
		t.Fatalf("persisted intelligence state incomplete: %#v", loaded)
	}
}

func TestCuratedCollectorRegistryDeclaresAdapters(t *testing.T) {
	registry := intelligence.NewIntelligenceSourceRegistry(intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "state.json")))
	sources := registry.List()
	if len(sources) < 3 {
		t.Fatalf("expected curated plugins, got %#v", sources)
	}
	known := map[string]bool{"rss": true, "usgs_geojson": true, "eonet_volcano": true}
	for _, source := range sources {
		if source.Adapter == "" {
			t.Fatalf("source %s has no declared adapter", source.ID)
		}
		if !known[source.Adapter] {
			t.Fatalf("source %s references unknown adapter %q", source.ID, source.Adapter)
		}
	}
}

func TestIntelligenceDeduplicatesObservationsAndCorrelatesHeadlines(t *testing.T) {
	store := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "state.json"))
	first := intelligence.IntelligenceEvent{Title: "Flood disruption affects Cologne rail network", Source: "Source A", SourceURL: "https://example.invalid/a", Confidence: 70}
	if err := store.ProposeEvent(first); err != nil {
		t.Fatalf("first observation: %v", err)
	}
	if err := store.ProposeEvent(first); err != intelligence.ErrDuplicateObservation {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
	if err := store.ProposeEvent(intelligence.IntelligenceEvent{Title: "Cologne rail network disruption after flood", Source: "Source B", SourceURL: "https://example.invalid/b", Confidence: 75}); err != nil {
		t.Fatalf("second observation: %v", err)
	}
	if correlations := store.Correlations(); len(correlations) == 0 {
		t.Fatal("expected headline correlation")
	}
}

func TestIntelligenceBusPublishesCommittedChanges(t *testing.T) {
	store := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "state.json"))
	_, events := store.Bus().Subscribe()
	if err := store.ProposeEvent(intelligence.IntelligenceEvent{Title: "Port closure affects regional logistics", Source: "Source", SourceURL: "https://example.invalid/port", Confidence: 60}); err != nil {
		t.Fatalf("propose observation: %v", err)
	}
	select {
	case event := <-events:
		if event.Type != "observation.proposed" || event.Revision == 0 {
			t.Fatalf("unexpected bus event: %#v", event)
		}
	default:
		t.Fatal("expected committed bus event")
	}
}

func TestIntelligenceBriefingAndReIDGate(t *testing.T) {
	store := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "state.json"))
	caseRecord, err := store.CreateCase("Pseudonym case", "legitimate investigation")
	if err != nil {
		t.Fatalf("create case: %v", err)
	}
	if err := store.ProposeEvent(intelligence.IntelligenceEvent{Title: "Verified high impact incident", Source: "Source", SourceURL: "https://example.invalid/high", Confidence: 90, Severity: "high"}); err != nil {
		t.Fatalf("create alert event: %v", err)
	}
	if len(store.Alerts()) != 1 || !strings.Contains(store.Briefing(), "Priority alerts") {
		t.Fatal("briefing must include threshold alert")
	}
	gate, err := store.ReIDStatus(caseRecord.ID)
	if err != nil || gate["reidentification"] != "not_eligible" {
		t.Fatalf("re-id gate failed: %#v / %v", gate, err)
	}
	// Dual-control workflow still records requests; raw reidentification stays not_eligible
	req, err := store.RequestReID(caseRecord.ID, "Need alias metadata for threat verification review", "operator-a")
	if err != nil {
		t.Fatalf("request reid: %v", err)
	}
	if req.Status != "requested" {
		t.Fatalf("expected requested, got %s", req.Status)
	}
	once, err := store.ApproveReID(caseRecord.ID, req.ID, "operator-a")
	if err != nil || once.Status != "approved_once" {
		t.Fatalf("first approve: %#v / %v", once, err)
	}
	if _, err := store.ApproveReID(caseRecord.ID, req.ID, "operator-a"); err == nil {
		t.Fatal("same approver must not complete dual-control")
	}
	unlocked, err := store.ApproveReID(caseRecord.ID, req.ID, "operator-b")
	if err != nil || !unlocked.Unlocked || unlocked.Status != "unlocked" {
		t.Fatalf("second approve unlock: %#v / %v", unlocked, err)
	}
	gate2, err := store.ReIDStatus(caseRecord.ID)
	if err != nil || gate2["reidentification"] != "not_eligible" {
		t.Fatalf("raw reidentification must remain not_eligible: %#v / %v", gate2, err)
	}
	if gate2["alias_unlock"] != true {
		t.Fatalf("alias unlock window should be active: %#v", gate2)
	}
}

func TestCaseIDUnificationRootAndShared(t *testing.T) {
	root := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "cases.json"))
	shared := intelligence.NewStore(filepath.Join(t.TempDir(), "shared.json"), intelligence.NewEventBus())
	prev := intelligence.SharedIntelStore
	intelligence.SharedIntelStore = shared
	t.Cleanup(func() { intelligence.SharedIntelStore = prev })

	c, err := root.CreateCase("Unified case", "alignment test purpose")
	if err != nil {
		t.Fatal(err)
	}
	snap := shared.GetSnapshot()
	found := false
	for _, sc := range snap.Cases {
		if sc.ID == c.ID {
			found = true
			if sc.Title != c.Title {
				t.Fatalf("title mismatch root=%q shared=%q", c.Title, sc.Title)
			}
		}
	}
	if !found {
		t.Fatalf("shared store missing case id %s", c.ID)
	}
	// Entity mirror
	ent, err := root.AddEntity(c.ID, "Acme Org", "organisation", 80)
	if err != nil {
		t.Fatal(err)
	}
	snap2 := shared.GetSnapshot()
	entFound := false
	for _, sc := range snap2.Cases {
		if sc.ID != c.ID {
			continue
		}
		for _, e := range sc.Entities {
			if e.ID == ent.ID {
				entFound = true
			}
		}
	}
	if !entFound {
		t.Fatal("entity not mirrored into shared case")
	}
	tl, err := root.CaseTimeline(c.ID)
	if err != nil || !strings.Contains(tl, "Case Timeline") {
		t.Fatalf("timeline: %v %s", err, tl)
	}
	// Evidence seal mirrors into shared
	ev, err := root.SealEvidenceWithEvent(c.ID, "Public source", "https://example.invalid/x", "promoted excerpt body", "operator", "ev_feed_9")
	if err != nil {
		t.Fatal(err)
	}
	sc, ok := shared.GetCase(c.ID)
	if !ok || len(sc.Evidence) == 0 {
		t.Fatal("shared case missing sealed evidence")
	}
	foundEv := false
	for _, e := range sc.Evidence {
		if e.ID == ev.ID {
			foundEv = true
		}
	}
	if !foundEv {
		t.Fatalf("shared evidence id mismatch; root=%s shared=%+v", ev.ID, sc.Evidence)
	}
}

func TestIntelligenceAlertIsPublishedOnTheLocalBus(t *testing.T) {
	store := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "state.json"))
	_, events := store.Bus().Subscribe()
	if err := store.ProposeEvent(intelligence.IntelligenceEvent{Title: "High confidence proposed observation", Source: "Local test", Confidence: 90, Severity: "high"}); err != nil {
		t.Fatalf("propose alert event: %v", err)
	}
	first := <-events
	second := <-events
	if first.Type != "observation.proposed" {
		t.Fatalf("expected observation commit first, got %#v", first)
	}
	if second.Type != "global_watch.alert" || second.Alert == nil || second.Alert.EventID == "" || second.Alert.Confidence != 90 {
		t.Fatalf("expected typed local alert, got %#v", second)
	}
}

func TestGlobalWatchScheduleIsBoundedAndOperatorControlled(t *testing.T) {
	// Drive the real GlobalWatchMonitor bounds (same rules HTTP handler enforces).
	tmp := t.TempDir()
	store := intelligence.NewIntelligenceStore(filepath.Join(tmp, "intelligence.json"))
	monitor := osint.NewGlobalWatchMonitor(store, filepath.Join(tmp, "schedule.json"), filepath.Join(tmp, "reports"))
	if err := monitor.Configure(true, 1); err == nil {
		t.Fatal("unsafe schedule interval accepted")
	}
	if err := monitor.Configure(true, 30); err != nil {
		t.Fatalf("valid schedule rejected: %v", err)
	}
	if !monitor.Snapshot().Enabled || monitor.Snapshot().IntervalMinutes != 30 {
		t.Fatalf("schedule not applied: %#v", monitor.Snapshot())
	}
	if err := monitor.Configure(false, 30); err != nil {
		t.Fatalf("disable: %v", err)
	}
}

func TestGlobalWatchBridgeSyncsToLiveNexusContext(t *testing.T) {
	// Isolate: clear shared store so we test legacy bridge path labels.
	prev := intelligence.SharedIntelStore
	intelligence.SharedIntelStore = nil
	t.Cleanup(func() { intelligence.SharedIntelStore = prev })

	store := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "state.json"))
	created := store.SyncOSINTEvents([]intelligence.OSINTEvent{{ID: "feed-1", Title: "Berlin transport disruption", Summary: "Public report", Source: "Curated feed", SourceURL: "https://example.invalid/berlin", Domain: intelligence.DomainGeo, Lat: 52.52, Lon: 13.405, HasGeo: true, Timestamp: time.Now().UTC(), Confidence: .8}})
	if created != 1 {
		t.Fatalf("expected one bridged event, got %d", created)
	}
	if store.SyncOSINTEvents([]intelligence.OSINTEvent{{ID: "feed-1", Title: "Berlin transport disruption", Source: "Curated feed", SourceURL: "https://example.invalid/berlin", Domain: intelligence.DomainGeo, Timestamp: time.Now().UTC(), Confidence: .8}}) != 0 {
		t.Fatal("bridge must deduplicate refreshed collector events")
	}
	context := store.LiveNexusContext(5)
	if !strings.Contains(context, "Berlin transport disruption") || !strings.Contains(context, "source=Curated feed") {
		t.Fatalf("live nexus context is incomplete: %s", context)
	}
	if !strings.Contains(context, "RAW") || !strings.Contains(context, "VERIFIED") {
		t.Fatalf("legacy fallback must still label layers: %s", context)
	}
}

func TestLiveNexusContextUsesSharedStoreWhenPresent(t *testing.T) {
	prev := intelligence.SharedIntelStore
	bus := intelligence.NewEventBus()
	shared := intelligence.NewStore(filepath.Join(t.TempDir(), "shared_nexus.json"), bus)
	intelligence.SharedIntelStore = shared
	t.Cleanup(func() { intelligence.SharedIntelStore = prev })

	shared.IngestObservation(intelligence.Observation{
		ID: "obs-nexus-1", SourceID: "rss-test", RawText: "Security development in Berlin reported",
		ObservedAt: time.Now().UTC(), Latitude: 52.52, Longitude: 13.405, Domain: "geo",
	})
	// Call via legacy type — must delegate to intelligence.SharedIntelStore
	legacy := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "legacy_empty.json"))
	ctx := legacy.LiveNexusContext(5)
	if !strings.Contains(ctx, "intelligence.SharedIntelStore") && !strings.Contains(ctx, "UNIFIED") {
		t.Fatalf("must use unified context when intelligence.SharedIntelStore set: %s", ctx)
	}
	if !strings.Contains(ctx, "RAW OBSERVATIONS") || !strings.Contains(ctx, "INFERENCE") || !strings.Contains(ctx, "VERIFIED") {
		t.Fatalf("must label layers: %s", ctx)
	}
	if !strings.Contains(ctx, "obs-nexus-1") && !strings.Contains(ctx, "Berlin") {
		t.Fatalf("must surface ingested shared observation: %s", ctx)
	}
	// Skill path
	skill := &skills.GlobalWatchNexusContextSkill{}
	out, err := skill.Execute([]byte(`{"limit":5}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "RAW OBSERVATIONS") {
		t.Fatalf("skill must return unified labelled context: %s", out)
	}
}

func TestCustomCollectorRejectsNonPublicTargets(t *testing.T) {
	for _, target := range []string{"http://example.com/feed", "https://127.0.0.1/feed", "https://localhost/feed", "https://example.com:8443/feed", "https://user:secret@example.com/feed"} {
		if err := osint.ValidatePublicCollectorURL(target); err == nil {
			t.Fatalf("expected public collector validation to reject %q", target)
		}
	}
	if err := osint.ValidatePublicCollectorURL("https://feeds.example.com/news.xml"); err != nil {
		t.Fatalf("expected public HTTPS URL allowed: %v", err)
	}
}

func TestGlobalWatchExtendedCommandsNavigateRegionTimeReport(t *testing.T) {
	// Drive the real IntelligenceStore bus methods the AI skills call.
	path := filepath.Join(t.TempDir(), "intelligence", "gw_cmd.json")
	store := intelligence.NewIntelligenceStore(path)
	id, ch := store.Bus().Subscribe()
	t.Cleanup(func() { store.Bus().Unsubscribe(id) })

	if err := store.NavigateUI("global_watch"); err != nil {
		t.Fatalf("NavigateUI: %v", err)
	}
	if err := store.FocusGlobalWatchRegion("germany"); err != nil {
		t.Fatalf("FocusGlobalWatchRegion: %v", err)
	}
	if err := store.SetGlobalWatchTimeWindowHours(24); err != nil {
		t.Fatalf("SetGlobalWatchTimeWindowHours: %v", err)
	}
	if err := store.OpenGlobalWatchReport("Test", "# hello"); err != nil {
		t.Fatalf("OpenGlobalWatchReport: %v", err)
	}

	seen := map[string]bool{}
	deadline := time.After(2 * time.Second)
	for len(seen) < 4 {
		select {
		case ev := <-ch:
			if ev.Type != "global_watch.command" || ev.Command == nil {
				continue
			}
			seen[ev.Command.Action] = true
			switch ev.Command.Action {
			case "navigate":
				if ev.Command.View != "global_watch" {
					t.Fatalf("navigate view: %q", ev.Command.View)
				}
			case "focus_region":
				if ev.Command.Region != "germany" {
					t.Fatalf("region: %q", ev.Command.Region)
				}
			case "time_window":
				if ev.Command.Hours != 24 {
					t.Fatalf("hours: %v", ev.Command.Hours)
				}
			case "open_report":
				if !strings.Contains(ev.Command.Body, "hello") {
					t.Fatalf("body: %q", ev.Command.Body)
				}
			}
		case <-deadline:
			t.Fatalf("timeout waiting commands, seen=%v", seen)
		}
	}
}

func TestGlobalWatchFocusPublishesTypedCommandWithoutObservation(t *testing.T) {
	store := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "state.json"))
	_, events := store.Bus().Subscribe()
	if err := store.FocusGlobalWatch(52.52, 13.405, 1.4, "Berlin"); err != nil {
		t.Fatalf("focus: %v", err)
	}
	select {
	case event := <-events:
		if event.Type != "global_watch.command" || event.Command == nil || event.Command.Action != "focus" || event.Command.Label != "Berlin" {
			t.Fatalf("unexpected command: %#v", event)
		}
		if len(store.Snapshot().Events) != 0 {
			t.Fatal("focus must not create a false observation")
		}
	default:
		t.Fatal("expected global watch command")
	}
}

func TestGlobalWatchLayerPublishesTypedCommand(t *testing.T) {
	store := intelligence.NewIntelligenceStore(filepath.Join(t.TempDir(), "intelligence", "state.json"))
	_, events := store.Bus().Subscribe()
	if err := store.SetGlobalWatchLayer("cities", false); err != nil {
		t.Fatalf("set layer: %v", err)
	}
	select {
	case event := <-events:
		if event.Type != "global_watch.command" || event.Command == nil || event.Command.Action != "layer" || event.Command.Layer != "cities" || event.Command.Enable == nil || *event.Command.Enable {
			t.Fatalf("unexpected layer command: %#v", event)
		}
	default:
		t.Fatal("expected layer command")
	}
	if err := store.SetGlobalWatchLayer("invalid", true); err == nil {
		t.Fatal("invalid layer accepted")
	}
}

func TestGlobalWatchMonitorWritesPrivateLocalReport(t *testing.T) {
	tmp := t.TempDir()
	store := intelligence.NewIntelligenceStore(filepath.Join(tmp, "intelligence.json"))
	if err := store.ProposeEvent(intelligence.IntelligenceEvent{Title: "Test alert", Source: "test", Confidence: 90, Severity: "high"}); err != nil {
		t.Fatalf("event: %v", err)
	}
	monitor := osint.NewGlobalWatchMonitor(store, filepath.Join(tmp, "schedule.json"), filepath.Join(tmp, "reports"))
	path, err := monitor.GenerateReport()
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat report: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("report permissions too broad: %o", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "Test alert") {
		t.Fatalf("report content invalid: %v %q", err, string(data))
	}
	if err := monitor.Configure(true, 5); err == nil {
		t.Fatal("unsafe interval accepted")
	}
	if err := monitor.Configure(true, 15); err != nil {
		t.Fatalf("configure: %v", err)
	}
	if !monitor.Snapshot().Enabled {
		t.Fatal("schedule not enabled")
	}
	if err := monitor.Configure(false, 15); err != nil {
		t.Fatalf("disable: %v", err)
	}
}

// TestOSINTPromptPersistAndCustomCollectorAndPseudonym drives shipped osint + intelligence paths.
func TestOSINTPromptPersistAndCustomCollectorAndPseudonym(t *testing.T) {
	tmp := t.TempDir()
	// Persist via real osint helper used by handlers
	promptPath := filepath.Join(tmp, "osint_briefing_prompt.txt")
	prompt := "Custom sovereign OSINT prompt for test."
	if err := osint.PersistOSINTBriefingPrompt(promptPath, prompt); err != nil {
		t.Fatalf("persist prompt: %v", err)
	}
	read, err := os.ReadFile(promptPath)
	if err != nil || string(read) != prompt {
		t.Fatalf("prompt not persisted correctly: %v %q", err, string(read))
	}

	// Custom collector via engine (real OSINTEngine + Add/Remove)
	eng := osint.NewOSINTEngine(filepath.Join(tmp, "osint_feeds.json"))
	cfg := osint.OSINTCollectorConfig{Name: "MyCustomRSS", Type: "rss", URL: "https://example.test/feed.xml", Domain: intelligence.DomainCyber, Enabled: true, Priority: 3}
	if err := eng.AddCollector(cfg); err != nil {
		t.Fatalf("add custom collector: %v", err)
	}
	cfgs := eng.GetConfigs()
	found := false
	for _, c := range cfgs {
		if c.Name == "MyCustomRSS" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("custom collector not in GetConfigs")
	}
	if err := eng.RemoveCollector("MyCustomRSS"); err != nil {
		t.Fatalf("remove custom: %v", err)
	}

	// Pseudonym + SealEvidence on real intelligence.IntelligenceStore
	store := intelligence.NewIntelligenceStore(filepath.Join(tmp, "intel.json"))
	c, _ := store.CreateCase("Test case for custom", "test")
	ev, err := store.SealEvidence(c.ID, "custom-source", "https://ex.test", "evidence body for hash and pseudonym test", "operator")
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if ev.SHA256 == "" || !ev.Sealed {
		t.Fatal("evidence not sealed with hash")
	}
	ent, err := store.AddEntity(c.ID, "John Doe", "person", 80)
	if err != nil {
		t.Fatalf("add person entity: %v", err)
	}
	if !strings.HasPrefix(ent.Label, "GX-PER-") {
		t.Fatalf("person not pseudonymized, got %s", ent.Label)
	}
}
