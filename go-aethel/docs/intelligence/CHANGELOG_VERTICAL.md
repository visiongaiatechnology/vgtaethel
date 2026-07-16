# Vertical milestone + Global Watch fix log

## 2026-07-12 (earth drop-in + repo structure)

### Earth basemap
- Operator may place `1.jpg` at project root or `frontend/assets/earth_day.jpg`
- Startup `ensureEarthTextureOnDisk()` copies drop-in into assets when missing
- API: `GET /v1/assets/earth-texture` (+ `/assets/earth_day.jpg` alias)
- Frontend tries asset → API → atlas bake

### Repo hygiene
- `scripts/organize_repo.ps1` — moves docs/logs/backups, installs earth texture
- `docs/STRUCTURE.md` — layout rules (`package main` stays at root)

## 2026-07-12 (Earth texture globe)

### Globe looks like Earth
- Orthographic sphere samples equirectangular basemap (same rot as pins)
- Prefer local `frontend/assets/earth_day.jpg` (no CDN)
- Fallback: bake land/ocean texture from TopoJSON world atlas at runtime
- Limb darkening + atmosphere rim; borders become thin coastlines when textured
- Copy helper: `frontend/assets/_copy_earth_day.ps1`

## 2026-07-12 (UI command center + connector pack)

### Global Watch UI (VGT original, WWV UX-inspired only)
- 3-column layout: Data Sources | Globe | Intel Detail/Feed
- Top search + region chips (Europe/MENA/Asia/…)
- Left rail layer rows with counts; Connectors list with FETCH
- Right DATA CONFIGURATION panel (lat/lon/domain/source/layer/id)
- Selection card media strip; FOCUS action
- Bottom timeline scrubber; map LAT/LON/SCALE HUD
- Screen-space pin clustering with count badges

### Connectors
- `builtin-rss`, `builtin-cyber`, `builtin-geo`, `builtin-economic`, `builtin-humanitarian`
- `builtin-usgs` (USGS GeoJSON all-day, safe HTTPS client)
- `builtin-shared-replay` (local metadata / no invent)
- Header **Connectors** multi-fetch button

## 2026-07-12 (continuation: shared evidence, connector fetch, run audit)

### Shared case graph depth
- `SealCaseEvidence` + root seal mirrors evidence into Shared
- GET cases returns `shared_cases` alignment summary
- `IngestObservationsBatch` for connector path

### Connector Fetch
- `builtin-rss` Fetch maps OSINT cache (+ rate-limited RefreshNow) → Observations
- Skill `intelligence_connector_fetch` + POST `/v1/intelligence/connectors/fetch`
- GET `/v1/intelligence/connectors` registry summary

### RunEngine AgentAction
- Tool steps in RunEngine call `RecordAgentAction` (same as HTTP tools)

### Regions + briefings
- POLAND, BALTICS, TAIWAN, ISRAEL region boxes
- Briefing CORROBORATED ≠ VERIFIED (golden tests)

## 2026-07-12 (U5–U7 open items)

### Case kernel
- Root/Shared **same case id** via CreateCaseWithID; entity/relation mirror
- osint_entity/relation/timeline/report skills + case timeline/report HTTP
- Re-ID dual-control API/UI (request + approve); raw reidentification stays not_eligible
- Assessment operator status API/skill

### Data quality / connectors
- Source content_hash + multi-source corroboration on ingest
- BuiltIn RSS connector registry entry + health

### Continuity
- AgentAction on tools/execute; identity skill/endpoint
- tool name accepts `skill` JSON alias

## 2026-07-12 (U3/U4/U5 slice)

### Personal ↔ World (U3)
- `BuildSharedPersonalContext` / `SyncPersonalToSharedIntel` bridge PersonalStore → Shared PersonalContext
- Skills: `intelligence_sync_personal`, `intelligence_personal_impact`
- API: `/v1/intelligence/personal/{context,sync,impact}`
- GW strip `#gw-personal-impact` (SYNC + REFRESH); Empfehlungen ≠ Fakten
- `GenerateReport("Personal Impact …")` routes to `PersonalImpact()`

### Proactive loop (U4)
- `StartProactiveLoop(2m)` from `app.go`
- `AlertManager` in package intelligence (no circular import with `intelligence/alerts` façade)
- Ingest + EvaluateAlertRules both use Manager cooldown/dedup/min-confidence
- Identity LastWarnedAt / PendingAlertIDs updated on rule fires

### Case promote provenance (U5 partial)
- `SealEvidenceWithEvent` + `source_event_id` field + audit `event.promoted`
- Skill `osint_evidence_capture`
- GW promote passes event id; geo_point entity when real coords

## 2026-07-12

### Unified model
- Chat nexus: `global_watch_nexus_context` → `SharedIntelStore.LiveNexusContext` only
- `/v1/intelligence/risk` maps SharedIntelStore RiskScores for HUD
- AlertRule entity (`CreatedAt`/`CreatedBy`) + `AddAlertRule` / `GetAlertRules` / EvaluateAlertRules
- `Store.CreateCase` + `osint_case_create` persists shared + root cases (no stub)
- DomainSummary for cyber/conflict/market/infrastructure skills

### Global Watch UI
- globeLayers.risks top-level; draws `cachedRiskMarkers` from SharedIntelStore-backed risk fetch
- showAethelToast: pure DOM textContent (no message innerHTML)
- Collector delete confirm: contentNode + textContent for source name
- initPureLocalGlobe preserves risk HUD / selection panel
- RISKS layer toggle in index.html
- **Geo fix:** orthographic projection east=+X (Europe no longer mirrored); real atlas fill; grid via projectLatLon; smaller risk dots; Europe default view
- **Case delete:** DELETE `/v1/intelligence/cases/{id}` + UI buttons (list + detail)
- WWV-inspired glass feed panel + map-primary layout

### Tests
- store_test: TestAddAlertRulePersistsAndEvaluates, TestSetPersonalContextAndImpact
- intelligence_test: SealEvidenceWithEvent promote audit
- frontend_static_test: personal impact paths + promote event id
