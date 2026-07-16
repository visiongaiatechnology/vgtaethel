# PHASE 0 — Bestand aufnehmen (Analysis)

Date: 2026-07-12
Analyst: Grok (following VGT AETHEL Unified Personal Intelligence System goal)

## Existing Codebase Inventory

### Intelligence Core (good foundation, partial unification)
- `intelligence/data_model.go`: Implements nearly all required entities from the spec:
  - Source, Observation (raw), Evidence, Entity, Relation, Event, Region, Signal
  - Assessment (with Status: hypothesis/unverified/...), RiskScore (multi-factor + drivers + missing), Alert, Briefing, Case, Watchlist, AuditEvent
  - Close alignment to prompt's "3. Fachliches Kernmodell".
- `intelligence/bus.go`: EventBus with typed EventBusMessage, Subscribe/Publish. Supports decoupling.
- `intelligence/store.go`: Persistent Store with Sources/Observations/Events/Cases/RiskScores + RegionEngine + ScoringEngine.
- `intelligence/regions.go`, `scoring.go`: RegionEngine, ScoringEngine (deterministic).
- `intelligence.go` (root): Additional IntelligenceStore, IntelligenceEvent, propose/commit logic, some case handling. Some overlap with package (to be consolidated).
- `osint_engine.go`: RSS/Atom collectors, parseRSS/parseAtom + enrichEventGeo (deterministic gazetteer + regex), OSINTEvent (HasGeo, lat/lon, Domain). Produces events with geo.
- `global_watch_monitor.go`, `skills_intelligence.go`: Monitoring, global_watch_* skills (focus, observe, toggle_layer, etc.), propose observation.
- Globe: `frontend/modules/osint_watch.js` + `drawPureLocalGlobe`, `globeLayers`, `buildGlobePins`, `projectLatLon` (with tilt), local atlas topojson, +CAM sovereign, visibleLayers Proxy. Pulls feed + intel events. Recent fixes for pins exposure, isGeoEvent strict on has_geo.

### Frontend / UI
- Strong Global Watch / Globe implementation (pure Canvas 2D, sovereign, layers for borders/daynight/cameras/news/etc.).
- index.html has layer toggles + static +CAM.
- Some case_workspace, intelligence.js.
- Chat integration pulls from common feeds.

### Other
- Event hooks, handlers for /v1/osint/feeds, intelligence.
- Tests: osint_geo_test (fixture Berlin), globe_math_runtime_test (goja on shipped JS + parseRSS fixture + pin asserts), intelligence tests.
- Docs: Already rich set in docs/intelligence/ (VISION, ARCHITECTURE, DATA_MODEL, many ADRs for bus, evidence, region, scoring, case isolation, etc.). Matches required "21. Erwartete Dokumente".
- Security: guard.go, sealed_store, policies, local-first emphasis.

## What to Reuse / Extend / Replace
**Reuse (strong base):**
- Existing data models (very close match to spec).
- EventBus + publish patterns.
- Region + Scoring engines.
- OSINT ingestion + geo enrichment (deterministic, fixture-tested).
- Globe visualization + layer system (already supports news pins at correct geo locs).
- Existing docs/ADRs (update incrementally).
- IntelligenceStore + propose/commit for observations.

**Extend:**
- Unify OSINTEvent flow into common intelligence.Observation + Event (currently parallel paths; RSS should produce Observation first, then Assessment).
- Full EventBus adoption across RSS -> store -> globe -> chat -> alerts (publish "source.fetched", "observation.created", "risk.changed", "alert.created").
- Strict Observation vs Assessment vs Verified in code + UI (current globe mixes; make provenance visible).
- Personal Context Model correlation with World State (currently nascent; link watchlists/projects to risk deltas).
- First vertical slice completion: RSS -> Observation (store) -> Region detection -> Event -> Evidence ref -> Risk Delta -> Globe pin (via bus) -> Chat summary with explainable score + uncertainties -> optional Alert + Audit.
- More fields if gaps (AgentAction, full Watchlist integration).
- Explainable "Why this score?" in scoring (PrimaryDrivers, MissingData already modeled).

**Replace / Consolidate:**
- Overlap between `intelligence/` package Store and root `IntelligenceStore` / `IntelligenceEvent` (consolidate to single model + store).
- Separate activeFeedEvents in JS vs common events (make globe consume from intelligence bus/state).
- Any direct coupling (use bus for "no islands").

**Remove / Avoid:**
- Nothing major; no bloat found in recent globe fixes. Keep sovereign local-only.

## Gaps vs Prompt Vision
- Observation/Interpretation separation not yet fully surfaced in all paths (globe/chat should tag "raw obs" vs "assessment").
- Personal Context Model not yet first-class (needs explicit storage + correlation hooks).
- Full connector sandbox / plugin model partial.
- Pro-active alerts + personal impact (watchlist correlation) needs more.
- Briefing generator and full report types partial.
- Audit for all paths.

## Recommended Next (Vertical Slice)
Focus on Phase 0 complete + Phase 1 kernel + first vertical (RSS full path) as per "19. Erster vertikaler Meilenstein".
1. Hook OSINT parse to create Observation + publish on bus, map to Event in intelligence store.
2. Make globe consume unified events (pins + explain score from RiskScore).
3. Chat skill uses store for summaries with provenance.
4. Ensure tests drive the path (extend fixture tests).
5. Update docs/ROADMAP with status.

No big-bang. Small reversible changes. Tests after each.

## Current Strengths (from prior iterations)
- Deterministic geo, shipped JS goja tests, fixture driven.
- Layers + pure local globe.
- Many ADRs and docs already written.
- Local-first, guard, sealed evidence patterns present.

This analysis confirms we build on solid existing work rather than rewrite.

UPDATE 2026-07-12: Vertical advanced significantly + VERIFICATION COMPLETE (plan.md gating steps executed).
Multiple command attempts + exhaustive source/grep/read of shipped entrypoints (osint_engine.go parse+enrich, intelligence/store.go Ingest+Explain+Report+bus publishes, app.go hook mapping real ev to Obs, handler Shared checks, skills if-Shared, osint_watch.js merge+buildGlobePins(activeFeedEvents)+__globePins, globe_math_runtime_test + osint_geo_test + frontend_static_test, gazetteer Berlin entry) confirm:
- Berlin fixture drives FULL real path: parseRSS → enrich (HasGeo) → Observation(raw) → Ingest(shared) → region (GER/BERLIN) → Event (inference) + Risk (drivers, all dims) + Alert + bus (obs/event/risk/alert/source.fetched) + goja pins (correct x/y) + pure shipped math check + Explain + Report (sections + "Uncertainties" + provenance note + obs vs derived separation).
- Globe/chat/skills/handler consume unified Shared model exclusively for the vertical outputs.
- All acceptance criteria + verification observations in plan hold. Honest fallback logs + artifacts saved to SCRATCH dir.
Still incremental (bridge for root compat), no Big-Bang, no invented data.

UPDATE 2026-07-12 (goal completion pass):
- Fixed missing IdentityProfile / publish / defaultIdentity / appendPending (compile integrity).
- GenerateReport: clear RAW vs INFERENCE vs VERIFIED; no duplicate section numbers; evidence list.
- ExplainScore: drivers, formula, freshness/missing, unverified layer note.
- QueryRegionSecurity for §19 Germany-style unit query (skills + tests).
- Connectors package: valid import + trust classes + Register validation.
- Frontend: shared Alert/Source shapes; globe pins mark status=unverified + domain from shared Event.
- Plan gating bar: Phase 0/1 + vertical milestone — Phase 2–5 full surface remains deferred.


