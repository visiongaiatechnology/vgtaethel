# ROADMAP — VGT AETHEL UNIFIED PERSONAL INTELLIGENCE SYSTEM

## Gating bar
U1–U7 practical depth shipped. Remaining optional: shared-only case retire of root graph, richer multi-connector marketplace, chat-run cost/telemetry polish.

### Phase 0–1 — Foundation + Kernel (COMPLETE)
* Docs/ADRs, SharedIntelStore vertical, chat nexus exclusive Shared

### Phase U1 — Single Write Path (SHIPPED)
* observe / POST events / region_compare / monitor → Shared

### Phase U2 — Operator Surfaces (SHIPPED)
* Explain drawer, Alerts center + ACK, promote UX

### Phase U3 — Personal ↔ World (SHIPPED)
* Personal sync/impact skills + API + GW strip

### Phase U4 — Proactive Loop (SHIPPED)
* StartProactiveLoop + AlertManager

### Phase U5 — Case Kernel (SHIPPED practical)
* Same case id root↔shared; entity/relation/evidence mirror
* osint_* skills; Re-ID dual-control; timeline/report
* SealCaseEvidence on Shared

### Phase U6 — Data quality & Connectors (SHIPPED practical)
* Source hashes + multi-source corroboration (≠ verified)
* BuiltIn connectors: RSS, domain slices, **USGS**, shared-replay
* Fetch → Ingest skill/API + GW connectors rail
* Regions: POLAND, BALTICS, TAIWAN, ISRAEL
* GW command-center UI (3-col, layers rail, intel panel, clusters)

### Phase U7 — Continuity (SHIPPED practical)
* Identity skill/API
* AgentAction on tools/execute **and** RunEngine tool steps
* Briefings: CORROBORATED section separate from VERIFIED

### Optional later
* Root case store as pure adapter over Shared
* Additional BuiltIn connectors (earthquake, custom HTTPS)
* Golden fuzz for RSS parsers

### Shipped post-U7 (desktop polish)
* High-res Earth basemap pipeline (up to 8k, bilinear sample, HiDPI globe, largest-on-disk serve)
* BuiltIn connectors: USGS + **NASA EONET** natural events
* Install script: `scripts/install_earth_texture.ps1 -Download`

## Operator tests
```
go test ./intelligence/ -count=1
go test ./intelligence/briefings/ -count=1
go test . -count=1 -run "TestCase|TestMulti|TestRecord|TestIntelligence|TestOSINTFrontend|TestRunEngine"
```
