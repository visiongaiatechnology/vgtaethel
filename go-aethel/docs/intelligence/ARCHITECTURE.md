# ARCHITECTURE — VGT AETHEL UNIFIED PERSONAL INTELLIGENCE SYSTEM

## Component Architecture
Aethel's backend architecture is strictly structured around the local-first security policy. All data remains in the workspace folder of the operator.

```
go-aethel/
├── intelligence/    <- Core domain implementation
│   ├── cases/       <- OSINT Evidence & Cases Isolated storage (HMAC entity hashing)
│   ├── sources/     <- Source registry & health monitor
│   ├── ingest/      <- RSS/HTTP Ingestion connector run loops
│   ├── evidence/    <- Immutable sealed evidence store
│   ├── entities/    <- Case-local entity graph
│   ├── relations/   <- Semantic entity relationships
│   ├── regions/     <- Geo-polygon Region Engine & locator mapping
│   ├── scoring/     <- Multi-dimension deterministic risk scoring
│   ├── alerts/      <- Alerting Rules Engine, deduplicator & cooldowns
│   ├── briefings/   <- Report generator (Situation, Daily, Cyber)
│   ├── timeline/    <- Case-local chronological timeline builder
│   ├── connectors/  <- Polling Connector Sandbox
│   ├── audit/       <- Fully-audited operator actions (Audit logs)
│   └── policies/    <- Core safety & license boundaries
├── assistant/       <- Operator personality, memories & goals
└── agent/           <- Skills execution kernel
```

## Local Intelligence Event Bus
The event bus decoupling ensures that RSS ingestion, the 3D globe visualization, the AI chat interface, alerts, and risk scoring are not tightly coupled.
* **Communication Model**: Publishers submit structured events asynchronously to the event bus. Subscribers listen on channels and execute local actions (e.g. updating risk score or displaying alerts).
* **SSE (Server-Sent Events)**: Emits event bus triggers to the frontend module `osint_watch.js` for real-time reactive rendering.
