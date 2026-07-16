# AETHEL INTELLIGENCE PLATFORM — TASK LIST

## Phase 0: Stocktaking & Fixes
- [x] Correct vertical globe coordinate projection math
- [x] Integrate Natural Earth high-fidelity coastline vectors
- [x] Implement glassmorphic Custom Dialogs & UX Toasts
- [x] Remove native blocking browser popups
- [x] PHASE0_ANALYSIS.md + foundation docs/ADRs

## Phase 1 & 2: Unified Intelligence Kernel & Regions (vertical milestone COMPLETE)
- [x] Create target docs/intelligence/ vision, threat, data, scoring & roadmap docs
- [x] Create 8 ADRs for core architectural decisions
- [x] Define standardized data model schemas in `data_model.go` (+ IdentityProfile, PersonalContext)
- [x] Implement async Event Bus in `bus.go`
- [x] Implement Ray-Casting PIP geofencing matching in `regions.go`
- [x] Implement multi-dimensional risk scoring with decay math in `scoring.go`
- [x] Vertical path: RSS → Observation → Ingest → Evidence/Event/Assessment/Risk/Alert/Audit/bus
- [x] ExplainScore + GenerateReport with Obs/Inference/Verified separation
- [x] QueryRegionSecurity (§19 Germany 24h style unit query)
- [x] SharedIntelStore wiring in app.go; handlers/skills/globe exclusive consumers
- [x] Unit tests store/scoring/regions + Berlin fixture + goja globe path
- [x] Connector registry metadata (BuiltInTrusted RSS descriptor)
- [x] SCRATCH verification artifacts (go shell unavailable this session)

## Deferred (Non-goals for current plan gating bar)
- [ ] Full Phase 3 multi-type golden briefs + advanced alert clustering UX
- [ ] Full Phase 4 Re-ID dual-approval production UX
- [ ] Full Phase 5 personal calendar/projects proactive mode
