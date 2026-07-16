# VGT AETHEL 1.0.0-beta.2 — BETA V2

**Sovereign Personal Intelligence OS · Production Candidate**

Beta V2 turns AETHEL from a broad alpha workspace into a governed, persistent intelligence
runtime. Conversation, orchestration, computer control, memory, writing and operational
awareness now share explicit effect contracts instead of relying on model claims.

## Highlights

- Separate Domain Model and Orchestrator selections with provider-aware routing and bounded
  context packages.
- Persistent runs with plan state, tool evidence, pause/resume, restart recovery, budgets,
  signed one-time approvals and verified completion reports.
- Deterministic intent routing and compact per-objective tool schemas to reduce token usage
  and prevent casual chat from invoking computer-control skills.
- Personal Core identity, location, consent, humor, honesty and proactivity controls.
- Encrypted local memory, profiles, authority stores and tamper-evident security decisions.
- Sphere Writer/Browser integration, live run flow, resizable panels and contextual widgets.
- Global Watch with a local textured globe, configurable feeds, strict time windows, alerts,
  cases, reader mode, briefings and AI-operable focus/layer contracts.
- Provider health, fallback visibility, reasoning configuration and configured-provider-only
  model lists.
- Hardened filesystem jails, browser egress, process execution, mail, OSINT ingestion, voice
  uploads, leases, approvals and audit persistence.

## Experience improvements

- Unified Beta V2 identity across splash, disclosure, shell, backend, installer and CI.
- Loader retains the manual **Starten** transition after initialization.
- Idle globe movement now uses a time-based 25 FPS cadence and a faster natural rotation.
- Failures and unavailable capabilities remain visible rather than being reported as active.
- Configured model and Orchestrator lists are served before live provider discovery; slow provider catalogs can no longer replace valid Groq models with an empty-provider state.
- A direct Wails model-registry binding now backs up the virtual HTTP route, so WebView fetch or asset-server failures cannot hide configured Groq models.

## Beta boundaries

- Public Windows binaries still require real-provider E2E evidence, Authenticode signing and
  clean-VM install/update testing against the tagged commit.
- Provider behavior depends on account access, model availability, network conditions and
  provider-specific limits.
- Host-control skills execute with the operator's Windows account after policy approval;
  application policy is not an operating-system sandbox.
- In-flight runs recover paused after restart and require explicit operator continuation.
- Offline voice availability depends on the installed local voice assets.

## Verification

```powershell
go test ./... -count=1
go vet -buildvcs=false ./...
govulncheck ./...
```

Release tag: `aethel-v1.0.0-beta.2`.
