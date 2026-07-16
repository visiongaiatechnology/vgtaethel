<div align="center">

```
 █████╗ ███████╗████████╗██╗  ██╗███████╗██╗
██╔══██╗██╔════╝╚══██╔══╝██║  ██║██╔════╝██║
███████║█████╗     ██║   ███████║█████╗  ██║
██╔══██║██╔══╝     ██║   ██╔══██║██╔══╝  ██║
██║  ██║███████╗   ██║   ██║  ██║███████╗███████╗
╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝
```

# VGT AETHEL
### Sovereign Intelligence OS

[![License](https://img.shields.io/badge/License-AGPLv3-blue?style=for-the-badge)](https://www.gnu.org/licenses/agpl-3.0)
[![Version](https://img.shields.io/badge/Version-v2.0.0--beta.2-orange?style=for-the-badge)](#)
[![Status](https://img.shields.io/badge/Status-BETA-yellow?style=for-the-badge)](#)
[![Go](https://img.shields.io/badge/Go-1.21-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Framework](https://img.shields.io/badge/Framework-Wails_Desktop-00ADD8?style=for-the-badge)](#)
[![Platform](https://img.shields.io/badge/Platform-Windows_10%2F11_x64-0078D4?style=for-the-badge&logo=windows)](#)
[![Dependencies](https://img.shields.io/badge/Dependencies-Zero_(pure_stdlib)-brightgreen?style=for-the-badge)](#)
[![Vault](https://img.shields.io/badge/Vault-AES--256--GCM-gold?style=for-the-badge)](#)
[![KeyStore](https://img.shields.io/badge/KeyStore-Windows_DPAPI-purple?style=for-the-badge)](#)
[![TTS](https://img.shields.io/badge/TTS-Sherpa--ONNX_(Offline)-brightgreen?style=for-the-badge)](#-sherpa-onnx-integration-guide)
[![Audit](https://img.shields.io/badge/Audit_Log-Blockchain--chained-purple?style=for-the-badge)](#)
[![Orchestrator](https://img.shields.io/badge/Orchestrator-AI--Kernel--v2-cyan?style=for-the-badge)](#-orchestration--intent-routing)
[![VGT](https://img.shields.io/badge/VGT-VisionGaiaTechnology-cyan?style=for-the-badge)](https://visiongaiatechnology.de)

**SOVEREIGN AI OS · ORCHESTRATOR · PERSONAL CORE · GLOBAL WATCH · SPHERE WORKSPACE · NATIVE DESKTOP · OFFLINE TTS**

<img width="50%" alt="VGT AETHEL Neural Interface" src="https://github.com/user-attachments/assets/509b3a60-ea7f-44a8-8c11-bc61bbbcc188" />

</div>

---

## ⚠️ BETA SOFTWARE — EXPERIMENTAL R&D

VGT AETHEL is a **Proof of Concept (PoC)** and active research project at VisionGaia Technology. It is **not** a certified or production-ready product.

**Use at your own risk.** The software may contain security vulnerabilities, bugs or unexpected behavior. It may break your environment if misconfigured.

**Do not deploy in critical production environments** without thoroughly auditing the code and understanding the implications.

Found a vulnerability or have an improvement? **Open an issue or contact us.**

---

<img width="1920" height="1009" alt="image" src="https://github.com/user-attachments/assets/a06c1a29-b807-44a4-a044-a6272958cf8c" />


---

## 🔍 What is VGT AETHEL?

AETHEL is not a chatbot. It is a **local sovereign AI operating system** — a personal intelligence layer that connects chat, planning, computer control, global situational awareness and personal assistance into one cohesive system.

```
Conventional AI Agents:
  Unsecured Python environment    → full system privileges
  No governance layer             → AI executes what it wants
  No audit trail                  → nothing is logged
  No operator gate                → changes applied silently
  Cloud-dependent TTS             → sends voice data externally
  No intent routing               → greetings trigger tool calls

VGT AETHEL v2:
  Go Cortex (pure stdlib)         → zero external attack surface
  Wails native desktop app        → embedded frontend, no browser required
  AI Orchestrator (v2)            → separate orchestrator coordinates models, tools, UI
  Intent Router                   → deterministic: chat / agent / UI / writer / watch
  Guard Kernel (policy engine)    → every tool call risk-scored before execution
  Operator gate                   → Moderate/High/Critical requires human confirmation
  Blockchain audit log            → every action chained and tamper-evident
  Windows DPAPI key store         → master key bound to OS user
  AES-256-GCM sealed stores       → all local state encrypted at rest
  Sherpa-ONNX TTS                 → fully offline voice output
  Personal Core                   → identity, memory, humor, location, proactivity
  Sphere Workspace                → writer, browser, run flow, weather, market data
  Global Watch                    → 3D globe with events, news, risks, alerts
  Token-optimized context         → compact, goal-specific model payloads
  Capability-based agent profiles → hard permission boundaries per role
  File snapshot & restore         → automatic rollback before destructive operations
```

AETHEL implements a **strict separation of intelligence, execution and communication** — the governance layer that most AI agent runtimes are missing.

> In Beta V1, many capabilities existed in isolation. In Beta V2, they are connected for the first time into a coherent system.

<img width="1920" height="1009" alt="image" src="https://github.com/user-attachments/assets/e0ca45ac-7157-4317-bb4c-f28b490bbc59" />


---

## 🏛️ Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                      OPERATOR (WAILS DESKTOP)                         │
│         ES6 Frontend — 16+ Modules — Embedded via go:embed           │
│  Chat · Sphere Workspace · Global Watch · Run Center · Agent Builder │
│  Personal Mode · Voice · Security · Tasks · Memory · Diagnostics     │
├──────────────────────────────────────────────────────────────────────┤
│                       REST API v1 (HTTP/JSON)                         │
├──────────────────────────────────────────────────────────────────────┤
│                  GO CORTEX — MODULAR (~77+ source files)              │
│                                                                      │
│  ┌─────────────────┬──────────────┬────────────────┬──────────────┐ │
│  │  AI Orchestrator│ Chat Engine  │  Guard Kernel  │  Voice       │ │
│  │  (v2)           │ (Streaming)  │  (Policy +     │  Engine      │ │
│  │  Intent Router  │              │   Audit Log)   │  Sherpa      │ │
│  │  Token Optimizer│              │                │  SAPI5       │ │
│  │  Provider Health│              │                │  Whisper     │ │
│  └─────────────────┴──────────────┴────────────────┴──────────────┘ │
│                                                                      │
│  ┌─────────────────┬──────────────┬────────────────┬──────────────┐ │
│  │  Skills Layer   │  Run Engine  │  Sealed Store  │  Personal    │ │
│  │  skills_gui     │  (Persistent │  (AES-256-GCM) │  Core        │ │
│  │  skills_fs      │   State      │                │  Memory      │ │
│  │  skills_browser │   Machine)   │  Key Store     │  Profile     │ │
│  │  writer_tool    │              │  (DPAPI)       │  Consent     │ │
│  └─────────────────┴──────────────┴────────────────┴──────────────┘ │
│                                                                      │
│  ┌─────────────────┬──────────────┬────────────────┬──────────────┐ │
│  │  Global Watch   │ Intelligence │  Cases &       │  Feed        │ │
│  │  3D Globe       │ Layer        │  Evidence      │  Reader      │ │
│  │  Events/News    │ Alerts/Risk  │  (Isolated     │  RSS/Danger  │ │
│  │  Earthquakes    │ Briefings    │   Contexts)    │  Sources     │ │
│  └─────────────────┴──────────────┴────────────────┴──────────────┘ │
├──────────────────────────────────────────────────────────────────────┤
│                        NEXUS MEMORY STORE                             │
│           ./vgt_workspace  (Sealed JSON / Sled DB)                   │
└──────────────────────────────────────────────────────────────────────┘
```

<img width="1920" height="1009" alt="image" src="https://github.com/user-attachments/assets/89b8377b-7336-4ceb-9cd7-47ac44ba979f" />


---

## 📊 Beta V1 → Beta V2 — What Changed

| Area | Beta V2 |
|---|---|
| **AI Orchestration** | Separation between normal AI model and orchestrator. The main model generates solutions; the orchestrator coordinates AETHEL, tools, UI and execution. |
| **Intent Routing** | Deterministic routing between chat, agent task, UI control, writer task and Global Watch. A greeting no longer accidentally triggers computer control. |
| **Token Optimization** | Instead of always explaining the full system, the model receives compact, goal-specific context and tool packages. |
| **Agent Runs** | Improved planning chains, persistent state, pause/resume, crash recovery, cost budgets, tool evidence and verifiable completion reports. |
| **Approvals** | Signed, argument-bound one-time approvals appear as global pop-ups — regardless of which UI area the user is in. |
| **Provider System** | Central registry for Groq, OpenAI, DeepSeek, Gemini, Claude and local Ollama models. Only actually configured providers and available local models are shown. |
| **Reasoning Control** | Provider- and model-specific reasoning levels, capability gates, context limits and output limits. |
| **Provider Health** | Health checks, visible error states, fallback decisions and live model detection. |
| **Groq Stability** | Payload normalization, correct tool-call sequences, protection against invalid assistant messages, robust registry fallback via Wails. |
| **Personal Core** | Own identity, name, location, interests, consent, humor, honesty and proactivity. AETHEL greets the user personally and considers their local situation. |
| **Personal Assistance** | Optional startup analysis with news, regional relevance, assessment and readable situation report. |
| **Memory** | Encrypted personal profiles and memories, traceable origin, improved hybrid retrieval from TF-IDF, word overlap, recency and importance. |
| **Sphere Workspace** | Desktop-like workspace with writer, internal browser, live run flow, media control, weather and market data widgets. |
| **Writer** | AETHEL can create and edit documents via an explicit, provider-independent tool contract. |
| **Code Cartography** | New agent mode that recursively analyzes code projects, describes files and documents architecture and dependencies as a Markdown map. |
| **Global Watch** | Local, textured 3D globe with borders, events, cities, earthquakes, volcanoes, news, regional risks and automatic rotation. |
| **Intelligence Layer** | News correlation, regional situation pictures, alerts, risk scoring, briefings, time windows, watchlists and AI-guided map focus. |
| **Feeds & Reader** | Configurable and removable RSS/danger sources, strict time filters and internal reading mode for articles. |
| **Cases & Evidence** | Cases, evidence capture, isolated case contexts, entities, relationships and controlled pseudonymization/re-identification. |
| **Multilanguage** | UI foundation for German, English, Russian and Spanish; briefings can be generated in the selected language. |
| **UI/UX** | Complete futuristic redesign, new loader with manual "Start", revised warning, Agent Tracker, Run Center and responsive Sphere window. |
| **Status Truth** | Unreachable APIs, voices or providers are no longer falsely shown as "Active". Errors remain visible. |
| **Performance** | GPU-intensive animations reduced. The globe uses time-based rotation and limited frame rate instead of continuous rendering. |
| **Security** | Hardened path jails, mount limits, process execution, browser egress, mail, voice uploads, authority stores, approvals and audit persistence. |
| **Release Engineering** | Beta-V2 versioning in loader, UI, backend, installer and CI; GitHub release workflow, signature hooks, diagnostic packages, checksums and pinned native dependencies. |
| **Startup Stability** | Deterministic EXE workspace, core-readiness handshake and registry retries prevent empty model lists and wrong configurations at startup. |

<img width="1920" height="1009" alt="image" src="https://github.com/user-attachments/assets/3b8d0cfe-7309-460c-a8f9-3c6dbace73c6" />


---

## 🧠 Orchestration & Intent Routing

Beta V2 introduces a dedicated **AI Orchestrator** that sits above the chat engine and coordinates all system components.

```
User Input
    │
    ▼
┌─────────────────────────────────────────┐
│            INTENT ROUTER                │
│  chat · agent · ui_control · writer     │
│  global_watch · personal_assistance     │
└───────────────┬─────────────────────────┘
                │
    ┌───────────┴───────────┐
    │                       │
    ▼                       ▼
MAIN MODEL              ORCHESTRATOR
(solution generation)   (tool/UI/execution control)
```

- **Chat** → normal conversational response, no tool invocation
- **Agent Task** → persistent run with planning chain, budget and evidence
- **UI Control** → Guard Kernel-gated computer control
- **Writer** → provider-independent document creation/editing tool contract
- **Global Watch** → intelligence layer, news correlation, globe focus
- **Personal Assistance** → startup briefing, regional news, situation report

The orchestrator receives compact, goal-specific context packets — not the entire system state on every call.

---

## 🛡️ Guard Kernel — Security Policy Engine

Every tool call is risk-scored by `guard.go` **before** it reaches execution. The operator is the final authority.

### Risk Tiers

| Tier | Score | Behavior |
|---|---|---|
| ⚪ **Safe** | 0–9 | Immediate autonomous execution |
| 🟢 **Low** | 10–29 | Immediate autonomous execution |
| 🟡 **Moderate** | 30–64 | Operator confirmation required |
| 🔴 **High** | 65–89 | Operator confirmation + warning |
| 🚨 **Critical** | 90–99 | Explicit confirmation required |
| ⛔ **Forbidden** | 100 | Permanently blocked — not overridable |

### Threat Classes Detected

| Class | Examples |
|---|---|
| `SHELL_INJECTION` | `;` `&&` `\|` `$(...)` in commands |
| `DESTRUCTIVE_COMMAND` | `rm -rf`, `format`, `dd`, `mkfs` |
| `PATH_TRAVERSAL` | `../`, `/etc/`, `C:\Windows` |
| `NETWORK_EXFIL` | `curl`, `wget`, `nc`, `ssh` |
| `FORBIDDEN_SHORTCUT` | `Alt+F4`, `Win+R`, `^ESC` |
| `EXECUTABLE_WRITE` | Writing `.exe`, `.bat`, `.sh` files |

### Permission Leases (v2: Signed & Global)

Operators can grant temporary, scoped permission unlocks:

- Time-bound via `ExpiresAt` timestamp
- Scope: app filter, action filter, forbidden targets
- **v2: Signed, argument-bound one-time approvals**
- **v2: Global pop-up — visible regardless of active UI area**
- Persisted in sealed store (`active_leases`)
- Revoked automatically on expiry

### Blockchain Audit Log

Every action is chained and tamper-evident:

```
Entry N:   action + SHA-256(Entry N-1)
Entry N+1: action + SHA-256(Entry N)
```

`ValidateChain()` performs complete cryptographic verification of the entire log on every server start. Audit persistence hardened in v2.

---

<img width="1920" height="1009" alt="image" src="https://github.com/user-attachments/assets/205df63b-2f8a-428f-bcb3-56fae9316e22" />


## 🔐 Security Architecture (Beta V2)

### Windows DPAPI Key Store

Master key is encrypted using **Windows Data Protection API (DPAPI)** via `key_store_windows.go` — cryptographically bound to the logged-in Windows user. Nothing in `./vgt_workspace` is readable plaintext.

### Sealed Local Stores

```
Storage prefix:  AETHEL-SEAL-v1:
Encryption:      AES-256-GCM
Implementation:  sealed_store.go
```

### v2 Security Hardening (new)

- **Path Jails:** hardened mount limits, process execution controls, browser egress restrictions
- **Voice Uploads:** sanitized and scope-limited
- **Authority Stores:** hardened write controls
- **Approval Persistence:** audit entries for signed one-time approvals stored and verifiable
- **Mail Controls:** egress limits on mail actions

### Capability-Based Agent Profiles

| Profile | Permissions |
|---|---|
| **Researcher** | Read-only filesystem and browser access. No write rights. |
| **Developer** | Write rights, `sys_exec` — strictly scoped |
| **Browser Operator** | Media and visible web control only |
| **Personal Assistant** | Personalized mode — no system write rights |

### File Snapshot & Restore (`file_snapshots.go`)

Before any file modification, AETHEL automatically creates an encrypted in-memory snapshot (up to 5 MB). On error, the previous state can be restored with a single confirmation.

### Durable Agent Runs (`run_engine.go`)

```
queued → running → waiting_approval → completed
                ↓                   ↓
              paused             failed
```

- Unique ID and full trace log per run
- Configurable USD cost budget per run
- Crash recovery: auto-pause on restart, explicit resume
- **v2: Verifiable completion reports with tool evidence**

---

<img width="2560" height="1351" alt="image" src="https://github.com/user-attachments/assets/8a217a5f-6792-499a-b37c-3c8727b52116" />

## 🌍 Global Watch

A **local, textured 3D globe** with real-time intelligence overlay — no cloud rendering required.

| Feature | Detail |
|---|---|
| **Rendering** | Local WebGL, time-based rotation, capped frame rate |
| **Data Layers** | Borders, cities, earthquakes, volcanoes, news events, regional risks |
| **Intelligence Layer** | News correlation, risk scoring, alerts, watchlists, time windows |
| **Briefings** | Multilingual AI-generated situation reports (DE/EN/RU/ES) |
| **Feeds** | Configurable/removable RSS and danger sources, strict time filters |
| **Reader** | Internal article reading mode — no external browser |
| **Cases & Evidence** | Isolated case contexts, entities, relationships, pseudonymization |
| **AI Map Focus** | Orchestrator-driven globe navigation based on active intelligence |

---

## 🔵 Sphere Workspace

A **desktop-like workspace** embedded in AETHEL — replacing the need to switch between external applications.

| Component | Detail |
|---|---|
| **Writer** | AI-assisted document creation/editing via provider-independent tool contract |
| **Internal Browser** | Scoped web access within the AETHEL security perimeter |
| **Live Run Flow** | Real-time agent step visualization |
| **Media Control** | Playback and control widgets |
| **Weather Widget** | Local weather based on Personal Core location |
| **Market Data Widget** | Configurable financial data feeds |

---

## 🧬 Personal Core

AETHEL now maintains a **persistent personal identity layer**:

| Attribute | Detail |
|---|---|
| **Identity** | Name, location, language preference |
| **Interests & Goals** | Stored in encrypted personal profile |
| **Consent** | Explicit opt-in for proactive behaviors |
| **Humor / Honesty / Proactivity** | Configurable sliders |
| **Memory** | Encrypted, traceable, hybrid retrieval (TF-IDF + recency + importance) |
| **Greeting** | Personalized startup greeting with local context |
| **Startup Briefing** | Optional news analysis, regional relevance, AI-generated situation report |

---

## 🗺️ Code Cartography

New agent mode for software project analysis:

```
Input: any local code project directory
Output: structured Markdown map

Covers:
  - Recursive file traversal and description
  - Architecture overview
  - Dependency graph
  - Module relationships
  - Entry points and key interfaces
```

---

## 🔧 Skills & Tool Capabilities

| Tool | Capability | Risk Level |
|---|---|---|
| `gui_control / move` | Move mouse cursor | 🟢 Low |
| `gui_control / position` | Query mouse position | 🟢 Low |
| `gui_control / click` | Left click | 🟡 Moderate |
| `gui_control / right` | Right click | 🟡 Moderate |
| `gui_control / double` | Double click | 🟡 Moderate |
| `gui_control / type` | Type text input | 🔴 High |
| `gui_control / press` | Key combination | 🔴 High |
| `fs_read_file` | Read file | 🟢 Low |
| `fs_list_dir` | List directory | 🟢 Low |
| `fs_write_file` | Write file (auto-snapshot) | 🟡 Moderate |
| `fs_mount_folder` | Mount folder | 🟡 Moderate |
| `sys_exec_cmd` | Execute system command | 🔴 High |
| `web_browser` | Open browser | 🟡 Moderate |
| `nexus_save` | Save to memory | ⚪ Safe |
| `nexus_recall` | Recall from memory | ⚪ Safe |
| `agent_handoff` | Hand off to ChatGPT / Gemini / Cursor | 🟡 Moderate |
| `viewport_screenshot` | Desktop screenshot (cached JPEG) | ⚪ Safe |
| `writer_tool` | Create/edit document (v2) | 🟡 Moderate |
| `code_cartography` | Analyze and map code project (v2) | 🟢 Low |

---

## 🎙️ Voice System

### Text-to-Speech (TTS)

| Provider | Quality | Mode | Requirement |
|---|---|---|---|
| **Sherpa-ONNX** (primary) | High (neural ONNX) | Fully offline | CGO + DLLs + model files |
| **Windows SAPI5** (fallback) | Standard | Local / offline | None |

### Speech-to-Text (STT)

| Provider | Model | Mode |
|---|---|---|
| **Groq Whisper** (primary) | `whisper-large-v3-turbo` | Cloud (API key required) |
| **Windows SAPI** (fallback) | Native | Local / offline |

### Voice Approval Commands

| Intent | Phrases |
|---|---|
| ✅ Approve | *"ja", "ok", "freigeben", "bestätigen", "erlauben", "mach", "go"* |
| ❌ Reject | *"nein", "nee", "stop", "block", "ablehnen"* |
| 🔐 High-Risk Approve | *"bestätige", "confirm"* (explicit) |

---

## 🔊 Sherpa-ONNX Integration Guide

### Step 1 — Configure CGO and Go Bindings

```bash
set CGO_ENABLED=1
go get github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx
go mod tidy
```

### Step 2 — Place Native DLLs (Windows)

Download from: **[Sherpa-ONNX Go Windows Native Libs (GNU)](https://github.com/k2-fsa/sherpa-onnx-go-windows/tree/master/lib/x86_64-pc-windows-gnu)**

```
aethel.exe
onnxruntime.dll
sherpa-onnx-c-api.dll
sherpa-onnx-core-c-api.dll
piper_phonemize_c_api.dll
espeak-ng_c_api.dll
kaldi-native-fbank-core.dll
```

### Step 3 — Download TTS Models

| Model | Language | Size | Download |
|---|---|---|---|
| **KittenTTS** | English | ~30 MB | [kitten-nano-en-v0_1-fp16.tar.bz2](https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2) |
| **Kokoro v1.0** | Multilingual | ~80 MB | [kokoro-multi-lang-v1_0.tar.bz2](https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-multi-lang-v1_0.tar.bz2) |

```
go-aethel/
├── aethel.exe
├── onnxruntime.dll
├── sherpa-onnx-c-api.dll
│   ... (all further DLLs)
└── vgt_workspace/
    └── models/
        └── sherpa/
            ├── kitten-nano-en-v0_1-fp16/
            │   ├── model.fp16.onnx
            │   ├── voices.bin
            │   ├── tokens.txt
            │   └── espeak-ng-data/
            └── kokoro-multi-lang-v1_0/
                ├── model.onnx
                ├── voices.bin
                ├── tokens.txt
                └── espeak-ng-data/
```

### Step 4 — Compile with CGO

```bash
set CGO_ENABLED=1
go build -ldflags="-s -w" -o aethel.exe .
```

> **Without CGO (`CGO_ENABLED=0`):** Stub module compiles automatically. SAPI5 fallback remains active.

---

<img width="2560" height="1351" alt="image" src="https://github.com/user-attachments/assets/454c0eb7-2878-4fab-ba0b-160b0671ca40" />

## 🖥️ Live Operator (Viewport Control)

| Feature | Detail |
|---|---|
| **Screenshot Engine** | GDI+ via PowerShell, JPEG 70% quality |
| **Resolution** | Auto-downscaled to max. 1280px width |
| **Multi-Monitor** | Cursor-follows: always the monitor with the mouse cursor |
| **Refresh Rate** | 800ms polling interval |
| **Caching** | In-memory cache, 800ms TTL (`sync.Mutex`) |
| **Evidence Screenshots** | Before/after visual proof of GUI actions in Run Center |
| **Error Fallback** | Last valid image used on transient capture failure |

---

## ⚙️ Task Engine — Autonomous Scheduling

| Feature | Detail |
|---|---|
| **Scheduler** | Background goroutine, tick every 5 seconds |
| **Task Types** | `once`, `recurring` (cron-style) |
| **Status Values** | `pending`, `running`, `completed`, `failed`, `cancelled` |
| **Persistence** | Sealed encrypted store |
| **Security Integration** | All tasks pass through the full policy engine |
| **Max Execution Time** | Configurable per task |

---

## 🔑 Secrets Vault

| Feature | Detail |
|---|---|
| **Encryption** | AES-256-GCM (authenticated encryption) |
| **Key Storage** | Windows DPAPI — bound to OS user (`config.key.dpapi`) |
| **Data Storage** | `AETHEL-SEAL-v1:` sealed store (encrypted at rest) |
| **API** | REST CRUD via `/v1/secrets` |
| **Git Protection** | `.gitignore` blocks commit of `.dpapi` / `.enc` / `.key` files |

---

## 🌐 API Reference

### Chat & Models

| Method | Endpoint | Function |
|---|---|---|
| `POST` | `/v1/chat` | Streaming chat with AI model |
| `GET` | `/v1/models` | Available LLM models |
| `GET/POST` | `/v1/chat/history` | Read / save chat history |
| `GET` | `/v1/chat/sessions` | List all sessions |
| `POST` | `/v1/chat/sessions/load` | Load session |
| `POST` | `/v1/chat/sessions/save` | Save session |
| `DELETE` | `/v1/chat/sessions/delete` | Delete session |

### Orchestrator & Intent (v2)

| Method | Endpoint | Function |
|---|---|---|
| `POST` | `/v1/orchestrate` | Orchestrator entry point (intent routing) |
| `GET` | `/v1/providers` | Provider registry status |
| `GET` | `/v1/providers/health` | Live provider health check |

### Tools & Skills

| Method | Endpoint | Function |
|---|---|---|
| `POST` | `/v1/tools/execute` | Execute skill (with security gate) |
| `GET/POST/DELETE` | `/v1/kernel/tasks/` | Task engine CRUD |
| `POST` | `/v1/tools/writer` | Writer tool contract (v2) |
| `POST` | `/v1/tools/cartography` | Code cartography analysis (v2) |

### Audio

| Method | Endpoint | Function |
|---|---|---|
| `POST` | `/v1/audio/speech` | TTS synthesis (Sherpa-ONNX / SAPI5) |
| `GET` | `/v1/audio/voices` | Available voices |
| `POST` | `/v1/audio/transcribe` | STT (Whisper / SAPI) |
| `GET` | `/v1/audio/health` | Audio stack health |
| `POST` | `/v1/audio/test` | TTS connection test |

### Security

| Method | Endpoint | Function |
|---|---|---|
| `GET/POST` | `/v1/security/leases` | Manage permission leases |
| `GET` | `/v1/security/audit` | Blockchain audit log |
| `GET` | `/v1/security/status` | Security status |

### Agent Runs

| Method | Endpoint | Function |
|---|---|---|
| `POST` | `/v1/runs` | Create new agent run |
| `GET` | `/v1/runs/:id` | Get run state + trace |
| `POST` | `/v1/runs/:id/pause` | Pause run |
| `POST` | `/v1/runs/:id/resume` | Resume paused run |
| `POST` | `/v1/runs/:id/approve` | Approve pending tool call |

### Viewport & Memory

| Method | Endpoint | Function |
|---|---|---|
| `GET` | `/v1/viewport/screenshot` | Live desktop screenshot (JPEG, cached) |
| `GET` | `/v1/viewport/status` | Viewport status |
| `GET/POST/DELETE` | `/v1/memory` | Nexus memory CRUD |
| `GET` | `/v1/memory/search` | Hybrid semantic search (v2) |

### Global Watch & Intelligence (v2)

| Method | Endpoint | Function |
|---|---|---|
| `GET` | `/v1/watch/events` | Current globe events |
| `GET` | `/v1/watch/alerts` | Active risk alerts |
| `POST` | `/v1/watch/briefing` | Generate intelligence briefing |
| `GET/POST/DELETE` | `/v1/watch/feeds` | Feed source management |
| `GET/POST` | `/v1/watch/cases` | Cases and evidence |

### Personal Core (v2)

| Method | Endpoint | Function |
|---|---|---|
| `GET/POST` | `/v1/personal/profile` | Personal Core profile |
| `GET` | `/v1/personal/briefing` | Startup situation briefing |

### Vault & Setup

| Method | Endpoint | Function |
|---|---|---|
| `GET/POST/DELETE` | `/v1/secrets` | AES-256-GCM sealed vault |
| `GET/POST` | `/v1/setup` | First-run configuration |
| `GET` | `/v1/diagnostics` | Privacy-safe system report |

---

## 🖥️ Frontend Modules

| Module | Responsibility |
|---|---|
| `app.js` | Root bootstrap, import orchestrator |
| `state.js` | Global reactive state store |
| `api.js` | HTTP wrapper, fetch helpers |
| `chat.js` | Chat terminal, Markdown rendering, tool approval UI |
| `voice.js` | STT/TTS, wake-word, voice sphere, approval routing |
| `control.js` | Live operator panel, screenshot polling |
| `security.js` | Leases UI, audit log viewer, threat display |
| `tasks.js` | Task manager UI, CRUD, status monitoring |
| `memory.js` | Nexus memory UI, hybrid search |
| `secrets.js` | Vault UI, key management |
| `ui.js` | Navigation, panels, animations, themes |
| `agent_builder.js` | Agent team configuration |
| `personal_mode.js` | Personal Core UI, sliders, journal |
| `approval_dialog.js` | Global tool call approval dialogs (v2: always-on-top) |
| `run_center.js` | Run state machine UI, trace logs, evidence screenshots |
| `diagnostics.js` | Privacy-safe system report export |
| `sphere.js` | Sphere Workspace (writer, browser, widgets) — v2 |
| `global_watch.js` | 3D globe, intelligence overlay, feeds, cases — v2 |

---

## 🚀 Quick Start

```bash
# 1. Copy environment config
cp .env.example .env
# Set GROQ_API_KEY=gsk_your_key

# 2. Build (CGO required for Sherpa-ONNX TTS)
set CGO_ENABLED=1
go build -ldflags="-s -w" -o aethel.exe .

# 3. Run
./aethel.exe
# New loader opens — click "Start" to initialize
```

> Without CGO, build with `CGO_ENABLED=0` — Sherpa-ONNX replaced by stub, SAPI5 fallback active.

---

## ⚙️ Technical Specifications

| Metric | Value |
|---|---|
| **Language** | Go 1.21 |
| **Framework** | Wails Desktop (WebView2 — embedded frontend) |
| **External Dependencies** | Zero stdlib / CGO only for Sherpa-ONNX |
| **Platform** | Windows 10/11 x64 (official) |
| **Backend Source Files** | ~77+ Go source files |
| **Frontend Modules** | 18 JS modules (embedded via `go:embed`) |
| **Vault Encryption** | AES-256-GCM |
| **Key Storage** | Windows DPAPI (`config.key.dpapi`) |
| **Local State** | `AETHEL-SEAL-v1:` sealed encrypted stores |
| **Audit Log** | SHA-256 blockchain-chained + `ValidateChain()` |
| **TTS Primary** | Sherpa-ONNX (offline ONNX neural models) |
| **TTS Fallback** | Windows SAPI5 (local, no API key) |
| **STT Primary** | Groq Whisper `whisper-large-v3-turbo` |
| **Memory Persistence** | `./vgt_workspace` (sealed JSON / Sled DB) |
| **Supported Languages** | DE, EN, RU, ES |
| **License** | AGPLv3 |

---

## 📋 Changelog

### v2.0.0-beta.2 — Sovereign Intelligence OS *(Current)*

#### Orchestration & Intelligence

- **AI Orchestrator (v2):** dedicated orchestrator layer separates solution generation (main model) from system coordination (orchestrator). Controls tools, UI state and execution.
- **Intent Router:** deterministic classification of user input into `chat`, `agent_task`, `ui_control`, `writer`, `global_watch` and `personal_assistance`. A greeting no longer accidentally triggers computer control.
- **Token Optimization:** compact, goal-specific context and tool packages per request instead of full system context on every call.
- **Provider Registry:** central registry for Groq, OpenAI, DeepSeek, Gemini, Claude and Ollama. Health checks, visible error states, fallback decisions, live model detection.
- **Reasoning Control:** per-provider and per-model reasoning levels, capability gates, context and output limits.
- **Groq Stability:** payload normalization, correct tool-call sequences, invalid assistant message protection, registry fallback via Wails.

#### Personal Core

- **Identity & Profile:** name, location, interests, goals stored in encrypted personal profile.
- **Consent & Sliders:** explicit opt-in for proactive behaviors; humor, honesty and initiative configurable.
- **Startup Briefing:** optional AI-generated situation report with news, regional relevance and assessment.
- **Memory v2:** encrypted personal memories with traceable origin; improved hybrid retrieval (TF-IDF + word overlap + recency + importance).
- **Personalized Greeting:** AETHEL greets the user by name and incorporates local context on startup.

#### Global Watch

- **3D Globe:** local WebGL globe with textured surface, borders, cities and automatic time-based rotation. GPU-intensive animations removed, frame rate capped.
- **Data Layers:** earthquakes, volcanoes, news events, regional risk overlays, watchlists.
- **Intelligence Layer:** news correlation, risk scoring, alerts, briefings with time windows, AI-guided map focus.
- **Feeds & Reader:** configurable/removable RSS and danger sources, strict time filters, internal article reading mode.
- **Cases & Evidence:** isolated case contexts, entity and relationship tracking, controlled pseudonymization/re-identification.
- **Multilingual Briefings:** intelligence reports generatable in DE, EN, RU, ES.

#### Sphere Workspace

- **Writer Tool:** document creation and editing via explicit, provider-independent tool contract.
- **Internal Browser:** scoped web access inside the AETHEL security perimeter.
- **Live Run Flow:** real-time agent step visualization in workspace.
- **Widgets:** weather (location-aware), market data feeds, media control.

#### Agent Runs

- **Improved Planning Chains:** multi-step planning with explicit reasoning traces.
- **Verifiable Completion Reports:** signed reports with full tool evidence on run completion.
- **Enhanced Crash Recovery:** deterministic EXE workspace, core-readiness handshake, registry retries.

#### Security

- **Hardened Approvals:** signed, argument-bound one-time approvals as global pop-ups — visible regardless of active UI area.
- **Path Jail v2:** hardened mount limits, process execution controls, browser egress restrictions.
- **Voice Upload Sanitization:** scoped and sanitized.
- **Audit Persistence:** hardened storage and verification of audit chain entries.
- **Status Truth:** unreachable APIs, voices and providers no longer falsely shown as "Active".

#### New Agent Mode

- **Code Cartography:** recursive project analysis, file description, architecture mapping and dependency documentation as structured Markdown output.

#### UI/UX

- **Full Redesign:** futuristic UI overhaul.
- **New Loader:** manual "Start" trigger with initialization handshake.
- **Agent Tracker:** live agent status indicator in header.
- **Responsive Sphere Window:** adaptive layout for workspace panels.
- **Revised Beta Warning:** updated for v2 scope.

#### Release Engineering

- Beta-V2 versioning consistent across loader, UI, backend, installer and CI.
- GitHub release workflow with signature hooks, diagnostic packages and checksums.
- Pinned native dependencies.

---

### v1.0.0-beta.1 — Native Desktop, DPAPI, Sealed Stores, Sherpa-ONNX *(archived)*

Wails Desktop migration. DPAPI key store. Sealed local stores (`AETHEL-SEAL-v1:`). Sherpa-ONNX offline TTS. Durable agent runs. Capability-based profiles. File snapshots. 16-module frontend. Blockchain audit log. Frontend XSS protection. API cost tracker. Guard Kernel. Session ID whitelisting.

---

### v0.6.0-alpha — Security Hardening & Feature Expansion *(archived)*

Path jail via `filepath.Rel` + `filepath.EvalSymlinks`. Shell injection blocker. Blockchain audit `ValidateChain()`. Frontend XSS protection. API cost tracker. Custom personas. Model registry updated.

---

### v0.5.0-alpha — Foundation Release *(archived)*

Go Cortex (pure stdlib). Guard Kernel. AES-256-GCM vault. Blockchain audit log. Voice STT/TTS. Task engine. Screenshot engine. Nexus memory.

---

## 🚧 Known Limitations (v2.0.0-beta.2)

- No automatic update system
- Single-operator only — no multi-user support
- Groq API key required for Whisper STT (offline fallback: Windows SAPI)
- Sherpa-ONNX requires CGO + GCC compiler + manual DLL and model setup
- Official platform focus: Windows 10/11 x64 (macOS/Linux builds possible without GUI control features)
- No HTTPS (localhost only — TLS optionally upgradeable)
- Global Watch data layers require external feed configuration

---

## 🗺️ Roadmap

| Feature | Status |
|---|---|
| Nexus Bridge (WhatsApp, Discord, Matrix, Signal, Telegram) | 🔜 Planned |
| Multi-model routing (local models via Ollama) | 🔜 Planned |
| Plugin system for skills | 🔜 Planned |
| Automatic self-update | 🔜 Planned |
| Web dashboard (externally accessible, auth) | 🔜 Planned |
| Cross-platform support (Windows / macOS / Linux) | ✅ Partial (Windows focus) |
| HTTPS / TLS out of the box | 🔜 Planned |
| Wails Desktop native app | ✅ Done (beta.1) |
| Windows DPAPI key store | ✅ Done (beta.1) |
| Sealed local stores | ✅ Done (beta.1) |
| Sherpa-ONNX offline TTS | ✅ Done (beta.1) |
| Durable agent runs | ✅ Done (beta.1) |
| Capability-based profiles | ✅ Done (beta.1) |
| AI Orchestrator + Intent Router | ✅ Done (beta.2) |
| Personal Core | ✅ Done (beta.2) |
| Global Watch (3D Globe + Intelligence Layer) | ✅ Done (beta.2) |
| Sphere Workspace | ✅ Done (beta.2) |
| Writer Tool | ✅ Done (beta.2) |
| Code Cartography | ✅ Done (beta.2) |
| Multilanguage UI (DE/EN/RU/ES) | ✅ Done (beta.2) |
| Provider Registry & Health | ✅ Done (beta.2) |

---

## 🔗 VGT Ecosystem

| Tool | Type | Purpose |
|---|---|---|
| 🧠 **VGT AETHEL** | **Sovereign AI OS** | Local AI intelligence OS with operator governance — you are here |
| 🖥️ **[VGT WP-Desk](https://github.com/visiongaiatechnology/vgtdesk)** | **OS-Layer / UX** | Hardened WordPress operator workspace |
| ⚔️ **[VGT Sentinel](https://github.com/visiongaiatechnology/sentinelcom)** | **WAF / IDS** | Zero-Trust WordPress WAF |
| ⚡ **[VGT Auto-Punisher](https://github.com/visiongaiatechnology/vgt-auto-punisher)** | **IDS** | L4+L7 Hybrid IDS |
| 🔐 **[VGT Omega Vault](https://github.com/visiongaiatechnology/vgt-omega-vault)** | **Encrypted Forms** | AES-256-GCM WordPress form vault |
| 🌐 **[GaiaCom](https://github.com/visiongaiatechnology/GaiaCom)** | **Communication** | Post-quantum federated E2EE platform |
| 📊 **[VGT Dattrack](https://github.com/visiongaiatechnology/dattrack)** | **Analytics** | Sovereign local analytics |

---

## 💙 Support the Mission

[![Donate](https://img.shields.io/badge/Donate-PayPal-00457C?style=for-the-badge&logo=paypal)](https://paypal.me/dergoldenelotus)

| Method | Address |
|---|---|
| **PayPal** | [paypal.me/dergoldenelotus](https://paypal.me/dergoldenelotus) |
| **Bitcoin** | `bc1q3ue5gq822tddmkdrek79adlkm36fatat3lz0dm` |
| **ETH / USDT (ERC-20)** | `0xD37DEfb09e07bD775EaaE9ccDaFE3a5b2348Fe85` |

---

## 📄 License

**AGPLv3 · © 2026 VisionGaia Technology · Cologne, Germany**

VGT AETHEL is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, version 3. Any derivative work or network-deployed modification must be published under the same license.

Enterprise deployments, TIER-0 audits (VGT SafetySys™) and commercial exception licenses: [visiongaiatechnology.de](https://visiongaiatechnology.de)

---

<div align="center">

**VISIONGAIATECHNOLOGY – WE ARCHITECT THE FUTURE OF SECURITY.**

[![VGT](https://img.shields.io/badge/VisionGaia-Technology-cyan?style=for-the-badge)](https://visiongaiatechnology.de)

*VGT AETHEL v2.0.0-beta.2 — Sovereign AI OS // AI Orchestrator + Intent Router // Personal Core // Global Watch // Sphere Workspace // Writer Tool // Code Cartography // Wails Native Desktop // Go Pure Stdlib // DPAPI Key Store // AES-256-GCM Sealed Stores // Guard Kernel // Blockchain Audit Log // Sherpa-ONNX Offline TTS // Durable Agent Runs // Capability Profiles // File Snapshots // 18-Module Frontend // DE/EN/RU/ES // AGPLv3 // Windows 10/11 x64*

</div>
