# go-aethel layout (post-split packages)

## Package map

| Package | Path | Role |
|---------|------|------|
| `main` | `.` | Wails entry, AppState, startup wiring |
| `handlers` | `handlers/` | HTTP API (chat, intel, OSINT, settings, …) |
| `skills` | `skills/` | Tool/skill implementations + registry |
| `intelligence` | `intelligence/` | SharedIntelStore + case store + scoring |
| `osint` | `osint/` | RSS engine, Global Watch monitor, connectors |
| `agent` | `agent/` | Run engine, chat agent, prompts, tasks |
| `security` | `security/` | Guard, policy, vault, mounts, audit |
| `provider` | `provider/` | LLM provider registry |
| `personal` | `personal/` | Personal store / profile |
| `memory` | `memory/` | Nexus memory store |
| `voice` | `voice/` | TTS / Sherpa |
| `system` | `system/` | Snapshots, release, diagnostics |

## Wiring rules

1. **`app.go` creates services**, then calls each package’s `InitState(...)` with deps.
2. **Handlers** read shared `handlers.state` (injected via `handlers.InitState`).
3. **Skills** read `skills.state` (injected via `skills.InitState`).
4. **`intelligence.SharedIntelStore`** is the package-level shared world model.
5. Earth texture: `handlers.EnsureEarthTextureOnDisk()` + `GET /v1/assets/earth-texture`.

## Frontend

```
frontend/          # //go:embed frontend/*
  assets/
    earth_day.jpg
    world-atlas-110m.topojson
  modules/
```

## Hygiene folders

| Path | Role |
|------|------|
| `docs/` | ADRs, STRUCTURE, project notes |
| `scripts/` | build + organize scripts |
| `_archive/` | old backups |
| `_logs/` | log dumps |
| `vgt_workspace/` | runtime data |
| `build/` | Wails build output |

## Do not

- Move remaining `package main` files into random subdirs without updating imports.
- Create import cycles (`handlers` must not be imported by `skills`/`agent` in a loop).
