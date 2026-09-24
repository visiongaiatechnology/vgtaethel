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
### Sovereign Strategic Intelligence OS

[![License](https://img.shields.io/badge/License-AGPLv3-blue?style=for-the-badge)](https://www.gnu.org/licenses/agpl-3.0)
[![Version](https://img.shields.io/badge/Version-1.0.0--beta.4.1-D4AF37?style=for-the-badge)](#-beta-v4-changelog)
[![Status](https://img.shields.io/badge/Status-ACTIVE_BETA-111111?style=for-the-badge)](#-beta-software--experimental-rd)
[![Go](https://img.shields.io/badge/Go-1.26.6-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
[![Framework](https://img.shields.io/badge/Wails-2.15.0-DF0000?style=for-the-badge)](https://wails.io)
[![Platform](https://img.shields.io/badge/Platform-Windows_10%2F11_x64-0078D4?style=for-the-badge&logo=windows)](#)
[![Architecture](https://img.shields.io/badge/Architecture-Local--First_Sovereign_OS-brightgreen?style=for-the-badge)](ARCHITECTURE.md)
[![VGT Coder](https://img.shields.io/badge/VGT_Coder-Agentic_IDE-cyan?style=for-the-badge)](#-vgt-coder-autonomous-agentic-ide)
[![Sphere 2.0](https://img.shields.io/badge/Sphere_2.0-Operations_Desktop-purple?style=for-the-badge)](#-sphere-20-personal-operations-desktop-os)
[![Vault](https://img.shields.io/badge/Vault-AES--256--GCM-gold?style=for-the-badge)](#)
[![KeyStore](https://img.shields.io/badge/KeyStore-Windows_DPAPI-purple?style=for-the-badge)](#)
[![TTS](https://img.shields.io/badge/TTS-Sherpa--ONNX_(Offline)-brightgreen?style=for-the-badge)](#-sherpa-onnx-integration-guide)
[![Audit](https://img.shields.io/badge/Audit_Log-Blockchain--chained-purple?style=for-the-badge)](#)
[![OSINT](https://img.shields.io/badge/OSINT-SHADOW_COMMAND-D4AF37?style=for-the-badge)](#-shadow-osint-command-mode)
[![Security Gate](https://img.shields.io/badge/Security_Mode-Vollzugriff%20%7C%20Interaktiv-00f0ff?style=for-the-badge)](#-dual-permission-modes-vollzugriff-vs-interaktiv)
[![VGT](https://img.shields.io/badge/VGT-VisionGaiaTechnology-cyan?style=for-the-badge)](https://visiongaiatechnology.de)

**SOVEREIGN AI OS · VGT CODER AGENTIC IDE · SPHERE 2.0 DESKTOP · SHADOW OSINT · COMMAND GLOBE · DUAL PERMISSION MODES · OFFLINE SHERPA-ONNX**

<img width="50%" alt="VGT AETHEL Neural Interface" src="https://github.com/user-attachments/assets/509b3a60-ea7f-44a8-8c11-bc61bbbcc188" />

</div>

---

## ✨ Beta V4 Changelog

> **Beta V4.1 (`1.0.0-beta.4.1`) Architecture & UI Update:** Beta V4.1 introduces a **Custom Frameless Window Titlebar** matching Aethel's cybernetic design, native Wails drag zones, integrated window controls (minimize, maximize/restore, close), alongside the **VGT Coder (Autonomous Coding Agent)**, the **Sphere 2.0 Personal Operations Desktop OS**, a dynamic **Dual-Mode Security Engine** (`VOLLZUGRIFF` vs. `INTERAKTIV`), and an exhaustive technical cartography mapped in [`ARCHITECTURE.md`](ARCHITECTURE.md).
 
| Area | What changed in Beta V4.1 |
|---|---|
| **Custom Frameless Titlebar** | Native OS title bar eliminated in favor of a sovereign cybernetic dark-glass title bar with `--wails-draggable: drag` regions, responsive minimize/maximize/close controls, and double-click window toggle. |
| **VGT Coder (Coding Agent)** | Experimental autonomous coding agent runtime (`go-aethel/coder/`) capable of multi-turn software development, AST inspection, file rewriting, project search, test execution, and Git diff synthesis. |
| **Multi-Model Orchestration** | Dual-model architecture separating the **Coding Worker Model** (code generation & debugging) from the **Tool Orchestrator Model** (tool calling, argument verification, and step planning). |
| **Centered Cybernetic Layout** | Chat composer (`.vgt-code-composer`) and message history are cleanly centered in a focused 840px column, providing responsive aesthetics across 1080p, 1440p, and 4K ultra-wide monitors. |
| **Zero-Latency Conversation Restore** | Historical sessions render instantly from local cache without network blocking. Selecting a previous session immediately keeps the chat composer active and ready for follow-up prompts. |
| **Fast-Path Sync Engine** | Bypassed redundant Git diff scans on completed terminal sessions, reducing conversation switching latency from multi-second blocking to <1 ms. |
| **Dual Permission Modes** | **`VOLLZUGRIFF` (Full Access):** Grants autonomous execution rights for safe, moderate, and high-risk workspace commands and file writes without interrupting confirmation pop-ups. Hard blocks for destructive sabotage (`RiskForbidden`) remain strictly enforced.<br>**`INTERAKTIV` (Normal Mode):** Dispatches interactive operator confirmation pop-ups for sensitive actions. Controlled via cybernetic UI toggle and persisted across restarts. |
| **Workspace Authority & Path Jail** | `ValidatePathForAccess` resolves paths dynamically against the active project workspace root (`CurrentCodeWorkspaceRoot()`), eliminating false jail violations during project development. |
| **Write-Enabled Folder Mounts** | `MountFolderSkill` grants `security.MountWrite` permissions and expands lease durations up to 7 days (168 hours / 10,080 minutes). |
| **Sphere 2.0 Desktop OS** | Virtual Desktop Environment supporting 6 dedicated desktops (`PERSONAL`, `WORK`, `RESEARCH`, `TRAVEL`, `PROJECT`, `INCIDENT`), 8-way window resize handles, state persistence, and ambient lighting. |
| **Universal Context Bus & Send To...** | Unified entity dispatch system allowing 1-click transfers of tasks, documents, clippings, trips, and plans between Sphere apps (`Writer`, `Browser`, `Planner`, `Research`, `Travel`, `Files`). |
| **AI Track Changes Diff Review** | VGT Writer features an integrated AI Track Changes engine with word-level insertions/deletions diffing, visual diff review dialogs, and operator accept/reject controls. |
| **Research Desk & Provenance** | Evidence clipping workbench that captures citations and sources, producing 1-click synthesized Markdown intelligence briefings. |
| **Trip Planner & Risk Correlation** | Intelligent trip lifecycle engine correlating itineraries with real-time Global Watch intelligence feeds, active conflict vectors, and travel alerts. |
| **Master Planner & Dependencies** | Hierarchical goal and milestone tracking with automated task dependency graph resolution (`Blocks` / `DependsOn`). |
| **Offline Sherpa-ONNX Pipeline** | Native CGo Sherpa-ONNX speech engine (`onnxruntime.dll`, `sherpa-onnx-c-api.dll`) delivering 100% local, offline neural voice synthesis and transcription with a quick-mute toggle. |
| **Non-Destructive Build Pipeline** | Specialized Wails build workflow (`scripts/build_preserve.ps1`) ensuring `./vgt_workspace`, configs, local databases, and models are preserved during compilation. |
| **Complete System Cartography** | Published [`ARCHITECTURE.md`](ARCHITECTURE.md), an authoritative 2,100+ line technical architecture document detailing all 18 subsystems, 19 operational viewports, and complete data flows. |

### Verification status

- `go test ./... -count=1` — passed (100% across all packages)
- `go vet -buildvcs=false ./...` — passed
- `govulncheck ./...` — no reachable vulnerabilities
- Wails `2.15.0` non-destructive build — passed (`AETHEL.exe` 63.3 MB generated)

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

VGT AETHEL Beta V3:
  Modular Go Cortex              → typed, testable security and intelligence domains
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
  SHADOW OSINT                    → military command mode with evidence-bound conflict graph
  Directed conflict vectors      → attacker → target, action, confidence, exact evidence IDs
  AI-only regional scores        → no deterministic or hybrid map-color fallback
  Token-optimized context         → compact, goal-specific model payloads
  Capability-based agent profiles → hard permission boundaries per role
  File snapshot & restore         → automatic rollback before destructive operations
```

AETHEL implements a **strict separation of intelligence, execution and communication** — the governance layer that most AI agent runtimes are missing.

> Beta V2 connected AETHEL's core capabilities. Beta V3 adds a hardened strategic OSINT command layer without weakening operator governance.

<img width="1920" height="1009" alt="image" src="https://github.com/user-attachments/assets/e0ca45ac-7157-4317-bb4c-f28b490bbc59" />


---

## 🏛️ System Architecture & Technical Cartography

> 📖 **Authoritative Specification Document:**  
> The full system architecture, dependency graphs, IPC boundaries, data flows, and security models are exhaustively documented in **[`ARCHITECTURE.md`](ARCHITECTURE.md)** (2,100+ lines of authoritative technical cartography).

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                               OPERATOR (WAILS DESKTOP)                                 │
│                   ES6 Frontend — 65+ Modules — Embedded via go:embed                   │
│   Chat · VGT Coder · Sphere 2.0 · Global Watch · SHADOW OSINT · Runs · Workbench       │
│   Personal Mode · Offline Voice · Security · Tasks · Memory · Diagnostics · Settings   │
├────────────────────────────────────────────────────────────────────────────────────────┤
│                                REST API v1 (HTTP/JSON)                                 │
├────────────────────────────────────────────────────────────────────────────────────────┤
│                       GO CORTEX — MODULAR (250+ Go Source Files)                       │
│                                                                                        │
│  ┌──────────────────┬─────────────────┬──────────────────┬──────────────────────────┐  │
│  │  AI Orchestrator │ Chat Engine     │ Security Kernel  │ Voice Pipeline           │  │
│  │  (Multi-Model)   │ (Streaming)     │ (PolicyEngine +  │ (Sherpa-ONNX Native CGo, │  │
│  │  Intent Router   │ Agent Run Loops │  Blockchain Log) │  Local Offline VAD/TTS)  │  │
│  └──────────────────┴─────────────────┴──────────────────┴──────────────────────────┘  │
│                                                                                        │
│  ┌──────────────────┬─────────────────┬──────────────────┬──────────────────────────┐  │
│  │  VGT Coder IDE   │ Sphere 2.0 OS   │ Skills Registry  │ Personal Core            │  │
│  │  (Worker+Orch,   │ (6 Desktops,    │ (Path Jail,      │ (Encrypted Memory,       │  │
│  │   AST Diff,      │  Context Bus,   │  Write Mounts,   │  Profile, Habits,        │  │
│  │   Vollzugriff)   │  Track Changes) │  GUI, Browser)   │  Operations Queue)       │  │
│  └──────────────────┴─────────────────┴──────────────────┴──────────────────────────┘  │
│                                                                                        │
│  ┌──────────────────┬─────────────────┬──────────────────┬──────────────────────────┐  │
│  │  Global Watch    │ SHADOW OSINT    │ Cases & Evidence │ Encrypted Mailbox        │  │
│  │  (3D Globe,      │ (Conflict DAG,  │ (Workbench,      │ (TLS IMAP/SMTP,          │  │
│  │   AI Risk Map)   │  40–60 Batch)   │  Chain Custody)  │  Threat Scoring)         │  │
│  └──────────────────┴─────────────────┴──────────────────┴──────────────────────────┘  │
├────────────────────────────────────────────────────────────────────────────────────────┤
│                               NEXUS PERSISTENT STORES                                  │
│       ./vgt_workspace (Sealed AES-256-GCM / DPAPI Key Store / Sled DB / Parquet)       │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

For complete technical specifications across all 18 subsystems and 19 operational viewports, refer to **[`ARCHITECTURE.md`](ARCHITECTURE.md)**.

<img width="1920" height="1009" alt="image" src="https://github.com/user-attachments/assets/89b8377b-7336-4ceb-9cd7-47ac44ba979f" />

---

## 📊 Beta V4 Capability Matrix

| Area | Beta V4 Updates |
|---|---|
| **VGT Coder (Coding Agent)** | Experimental software development environment with multi-turn reasoning, AST inspection, file rewriting, project search, test execution, and Git diff synthesis. |
| **Dual-Model Orchestration** | Dedicated separation between Coding Worker Model (code generation/debugging) and Tool Orchestrator Model (plan synthesis/tool invocation). |
| **Centered Cybernetic Layout** | Responsive 840px centered chat composer and message history with seamless scaling across ultra-wide monitors. |
| **Zero-Latency History Restore** | Instant cached session rendering without network blocking; persistent chat composer visibility; sub-millisecond fast-path synchronization. |
| **Dual Permission Modes** | **`VOLLZUGRIFF` (Full Access):** Autonomous execution of safe, moderate, and high-risk workspace actions without confirmation pop-ups.<br>**`INTERAKTIV` (Normal Mode):** Operator confirmation pop-ups for critical actions. Persisted across sessions and toggled via 1-click cybernetic UI badge. |
| **Workspace Authority & Mounts** | Dynamic path jail integration against the active project workspace root (`CurrentCodeWorkspaceRoot()`) with write mounts up to 7 days (168 hours). |
| **Sphere 2.0 Desktop OS** | 6 Virtual Desktops (`PERSONAL`, `WORK`, `RESEARCH`, `TRAVEL`, `PROJECT`, `INCIDENT`), 8-way resize handles, focus stacking, and spatial state persistence. |
| **Universal Context Bus & Send To...** | 1-click cross-application entity transfers between Writer, Browser, Planner, Research, and Travel tools. |
| **AI Track Changes Engine** | Word-level insertions and deletions diff review for documents with operator accept/reject controls in VGT Writer. |
| **Research Desk & Provenance** | Evidence clipping workbench that captures citations and sources, producing 1-click synthesized Markdown intelligence briefings. |
| **Trip Planner & Risk Correlation** | Intelligent trip lifecycle engine correlating itineraries with real-time Global Watch intelligence feeds, active conflict vectors, and travel alerts. |
| **Master Planner & Dependencies** | Hierarchical goal and milestone tracking with automated task dependency graph resolution (`Blocks` / `DependsOn`). |
| **Offline Sherpa-ONNX Voice** | 100% offline local speech synthesis and recognition powered by CGo ONNX Runtime bindings (`onnxruntime.dll`, `sherpa-onnx-c-api.dll`) with a quick-mute toggle. |
| **SHADOW OSINT** | Compartmented military intelligence mode with dedicated doctrine, encrypted state, and black/gold command interface. |
| **Conflict Command Globe** | Local WebGL earth with evidence-bound regions and directed attacker-to-target vectors. |
| **Batch Intelligence** | Rotating collection, strict 40–60 item analysis, and daily master-dossier synthesis. |
| **Regional Risk Authority** | AI-only, evidence-bound scores. Missing AI assessment results in no score rather than an algorithmic substitute. |
| **AI Orchestration** | Separation between normal AI model and orchestrator. The main model generates solutions; the orchestrator coordinates AETHEL, tools, UI, and execution. |
| **Intent Routing** | Deterministic routing between chat, agent task, UI control, writer task, and Global Watch. |
| **Token Optimization** | Compact, goal-specific context and tool packages per request instead of full system context on every call. |
| **Agent Runs** | Planning chains, persistent state, pause/resume, crash recovery, cost budgets, tool evidence, and verifiable completion reports. |
| **Approvals** | Signed, argument-bound one-time approvals appear as global pop-ups across all UI views. |
| **Provider System** | Central registry for Groq, OpenAI, DeepSeek, Gemini, Claude, and local Ollama models. Only configured providers and available local models are shown. |
| **Personal Core** | Own identity, name, location, interests, consent, humor, honesty, and proactivity with startup situation briefings. |
| **Nexus Memory** | Encrypted personal profiles and memories with hybrid retrieval (TF-IDF + word overlap + recency + importance). |
| **Global Watch** | Local, textured 3D globe with borders, events, cities, earthquakes, volcanoes, news, regional risks, and automatic rotation. |
| **Cases & Evidence** | Isolated case contexts, entity/relationship tracking, and controlled pseudonymization/re-identification. |
| **Multilanguage UI** | Multi-language support for German, English, Russian, and Spanish; briefings generatable in selected language. |
| **Security Hardening** | Hardened path jails, mount limits, process execution, browser egress, mail, voice uploads, authority stores, approvals, and audit persistence. |
| **Non-Destructive Build Pipeline** | Specialized Wails build workflow (`scripts/build_preserve.ps1`) preserving `./vgt_workspace`, configs, local databases, and models during compilation. |
| **Complete System Cartography** | Authoritative [`ARCHITECTURE.md`](ARCHITECTURE.md) documenting all 18 subsystems, 19 operational viewports, and complete data flows. |

<img width="1920" height="1009" alt="image" src="https://github.com/user-attachments/assets/3b8d0cfe-7309-460c-a8f9-3c6dbace73c6" />


---

## 🧠 Orchestration & Intent Routing

Beta V3 retains the dedicated **AI Orchestrator** introduced in Beta V2 and extends its governed intelligence surfaces with SHADOW OSINT.

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


## 🔐 Security Architecture (Beta V3)

### Windows DPAPI Key Store

Master key is encrypted using **Windows Data Protection API (DPAPI)** via `key_store_windows.go` — cryptographically bound to the logged-in Windows user. Nothing in `./vgt_workspace` is readable plaintext.

### Sealed Local Stores

```
Storage prefix:  AETHEL-SEAL-v1:
Encryption:      AES-256-GCM
Implementation:  sealed_store.go
```

### Beta V3 Security Hardening

- **Path Jails:** hardened mount limits, process execution controls, browser egress restrictions
- **Voice Uploads:** sanitized and scope-limited
- **Authority Stores:** hardened write controls
- **Approval Persistence:** audit entries for signed one-time approvals stored and verifiable
- **Mail Controls:** egress limits on mail actions
- **Evidence-bound assessments:** regional scores and conflict directions require exact batch evidence IDs
- **Fail-closed risk display:** missing or failed AI evaluation cannot fall back to deterministic map coloring
- **Strict SHADOW schema:** unknown fields, trailing JSON, oversized responses and out-of-batch evidence are rejected
- **Atomic dossier persistence:** storage failures restore the previous in-memory report and processing state
- **Bounded rendering:** coordinates, region counts and conflict-vector counts are validated before WebGL overlay rendering

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
| **Data Layers** | Borders, cities, news events and isolated earthquake/volcano hazard layers |
| **Intelligence Layer** | News correlation, evidence-bound AI risk scoring, alerts, watchlists and time windows |
| **Briefings** | Multilingual AI-generated situation reports (DE/EN/RU/ES) |
| **Feeds** | Configurable/removable RSS and danger sources, strict time filters |
| **Reader** | Internal article reading mode — no external browser |
| **Cases & Evidence** | Isolated case contexts, entities, relationships, pseudonymization |
| **AI Map Focus** | Orchestrator-driven globe navigation based on active intelligence |
| **Score Authority** | AI-only; no deterministic or hybrid regional fallback coloring |
| **Hazard Isolation** | Earthquakes and volcanic events are not news and do not enter automatic AI risk context |

---

## 🛰️ SHADOW OSINT Command Mode

SHADOW is AETHEL's compartmented military and strategic intelligence environment. It uses an independent doctrine, isolated encrypted state and an evidence-first analysis pipeline.

| Capability | Detail |
|---|---|
| **Command Surface** | High-class black/gold strategic interface that reskins the complete AETHEL shell while active |
| **WebGL Globe** | Local textured 3D earth with drag rotation, wheel zoom, theater markers and orbital telemetry |
| **Conflict Vectors** | Directed attacker → target arcs with action, confidence, animation and supporting evidence IDs |
| **Accepted Actions** | Attack, invasion, strike, blockade, occupation, proxy attack, cyberattack and military support |
| **Evidence Gate** | A vector is rejected when its direction is inferred only from tension, rhetoric or historic hostility |
| **Source Registry** | Editable RSS, Web and Telegram sources across military, official, geopolitical, economy, energy, cyber and space domains |
| **Telegram** | Public-preview collection for `militaernews`; no account token required |
| **Collection** | Bounded rotating source acquisition with concurrency and response limits |
| **Analysis** | Exactly 40–60 pending intelligence objects per batch |
| **Daily Dossier** | Evidence-preserving master synthesis from multiple reports generated on the same day |
| **Exports** | JSON and Markdown, including regional assessments and directed conflict vectors |
| **Persistence** | AES-256-GCM sealed SHADOW state with rollback on failed writes |

The mandatory Beta V3 conflict contract is appended at runtime. Editing the operator doctrine cannot remove its evidence and direction requirements.

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

### SHADOW OSINT (Beta V3)

| Method | Endpoint | Function |
|---|---|---|
| `GET` | `/v1/shadow/status` | Source, buffer, batch and report status |
| `GET` | `/v1/shadow/snapshot` | Bounded operational snapshot |
| `GET/POST/PUT/DELETE` | `/v1/shadow/sources` | Editable RSS, Web and Telegram registry |
| `POST` | `/v1/shadow/collect` | Run bounded rotating source collection |
| `POST` | `/v1/shadow/analyze` | Analyze the next evidence batch of 40–60 items |
| `POST` | `/v1/shadow/daily` | Generate an evidence-preserving daily master dossier |
| `GET` | `/v1/shadow/reports` | Retrieve batch and daily dossiers |
| `GET` | `/v1/shadow/regions` | AI regions and directed conflict links |
| `GET/PUT` | `/v1/shadow/prompt` | Read or update the sealed operator doctrine |
| `GET` | `/v1/shadow/export` | Export a dossier as JSON or Markdown |

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
| `osint_watch.js` + `osint/*` | Global Watch globe, intelligence overlay, feeds, hazards and cases |
| `shadow_osint.js` | SHADOW command UI, source registry, reports and operational controls — Beta V3 |
| `shadow_globe.js` | Local WebGL sphere, region markers and directed conflict-vector rendering — Beta V3 |

---

## 🚀 Quick Start

### Option A: Windows 10/11 Desktop (Custom Frameless Wails Window)

```powershell
# 1. Enter the authoritative desktop runtime
cd go-aethel

# 2. Verify the source tree
go test ./... -count=1
go vet -buildvcs=false ./...

# 3. Build with the pinned Wails 2.15 toolchain
wails build -o AETHEL.exe -trimpath -nocolour

# 4. Start the desktop application
.\build\bin\AETHEL.exe
```

### Option B: Linux Server (Headless Web Runtime + Login Protection)

On Linux servers, AETHEL automatically runs in **Headless Server Mode**, serving the embedded frontend and REST API over a configurable port protected by an **Argon2id + AES-256-GCM Login Gate**:

```bash
# 1. Clone the repository on your Linux server
git clone https://github.com/visiongaiatechnology/vgtaethel.git
cd vgtaethel

# 2. Run the interactive server installer (sets operator password, builds static binary, optional systemd unit)
chmod +x install-server.sh
./install-server.sh
```

You can also run `install-server.sh` non-interactively in automated server provisioning:

```bash
AETHEL_PORT=8080 AETHEL_ADMIN_PASSWORD="YourStrongPassword123!" AETHEL_INSTALL_SYSTEMD=1 ./install-server.sh
```

API keys and provider settings are entered through AETHEL's first-run setup and encrypted local configuration. Never commit `vgt_workspace`, `.env`, keys, sessions or generated dossiers.

---

## ⚙️ Technical Specifications

| Metric | Value |
|---|---|
| **Language** | Go 1.26.6 |
| **Framework** | Wails 2.15.0 Desktop (Windows) / Native Hardened HTTP(S) Server (Linux) |
| **Architecture** | Local-first modular Go runtime with pinned Go modules and native Sherpa-ONNX runtime ([`ARCHITECTURE.md`](ARCHITECTURE.md)) |
| **Platform** | Windows 10/11 x64 (Desktop) · Linux x86_64 / arm64 (Headless Server) |
| **Backend Source Files** | 250+ Go source and test files |
| **Frontend Modules** | 65+ JavaScript modules (embedded via `go:embed`) |
| **Vault Encryption** | AES-256-GCM |
| **Server Auth** | Argon2id (`t=3, m=64MiB`) + AES-256-GCM Sealed Store + HttpOnly/CSRF Session Gate |
| **Key Storage** | Windows DPAPI (`config.key.dpapi`) / Sealed Key Vault (Linux `0600`) |
| **Local State** | `AETHEL-SEAL-v1:` sealed encrypted stores |
| **Audit Log** | SHA-256 blockchain-chained + `ValidateChain()` |
| **TTS Primary** | Sherpa-ONNX (offline ONNX neural models via CGo runtime) |
| **TTS Fallback** | Windows SAPI5 (local, no API key) |
| **STT Primary** | Groq Whisper `whisper-large-v3-turbo` + Local Sherpa-ONNX |
| **Memory Persistence** | `./vgt_workspace` (sealed JSON / Sled DB / Parquet) |
| **Supported Languages** | DE, EN, ES, FR, RU |
| **License** | AGPLv3 |

---

## 📋 Changelog

### 1.0.0-beta.4.2 — Dual Runtime Detection, Linux Server Mode & Login Gate *(Current)*

- **Automatic Platform & Runtime Detection:** Automatically launches the custom frameless Wails desktop window on Windows and switches to Headless HTTP/HTTPS Server Mode on Linux servers (or when `--server` / `AETHEL_SERVER_MODE=1` is passed).
- **Argon2id + AES-256-GCM Server Login Protection:** Added `ServerAuthManager` protecting all `/v1/*` and `/browser/*` endpoints in server mode with OWASP-grade Argon2id password hashing, sealed disk persistence (`server_auth.seal`), `HttpOnly` `SameSite=Strict` session cookies, `X-Aethel-CSRF` token validation on mutating requests, and per-IP brute-force lockout protection.
- **Cybernetic Login Gate & Session Lock UI:** Integrated `#aethel-auth-gate` login/setup screen in Aethel's dark-glass design and a header `LOCK` (logout) button, while automatically hiding desktop window controls when accessed via browser.
- **Zero-GUI-Dependency Linux Builds:** Isolated Wails desktop bindings (`desktop_windows.go` / `desktop_other.go`), DuckDB CGo tags, and Sherpa-ONNX stubs so Linux server builds compile cleanly with `CGO_ENABLED=0 GOOS=linux`.
- **1-Command Server Installer (`install-server.sh`):** Interactive and automated Linux server installation script with Go toolchain verification, password provisioning (`--set-password-stdin`), binary compilation, and hardened `systemd` unit setup.

---

### 1.0.0-beta.4.1 — Custom Frameless Window & Cybernetic Titlebar

- **Frameless Window Architecture:** Eliminated the native OS window titlebar and borders via Wails `Frameless: true` and Windows dark theme configuration.
- **Cybernetic Glass Titlebar:** Integrated custom window controls (`—` minimize, `▢` maximize/restore, `✕` close) directly into `.top-system-bar` in Aethel's signature cybernetic dark-glass aesthetic.
- **Native Wails Dragging:** Entire upper system bar operates as a native window drag region via `--wails-draggable: drag` with proper `--wails-draggable: no-drag` protection for all interactive controls.
- **Double-Click Maximize:** Native desktop behavior allowing double-clicking the header bar to toggle maximize/restore state, accompanied by dynamic SVG icon synchronization.
- **Go Runtime Window Bindings:** Exposed `WindowMinimise()`, `WindowToggleMaximise()`, `WindowClose()`, and `WindowIsMaximised()` on the Go `App` struct.

### 1.0.0-beta.4 — Architecture & Desktop Overhaul *(archived)*

#### VGT Coder (Experimental Coding Agent)
- **Autonomous Coding Agent:** Experimental multi-turn software development agent (`go-aethel/coder/`) capable of inspecting projects, editing files, running tests, and managing Git worktrees.
- **Dual-Model Orchestration:** Specialized separation between Coding Worker Model (code generation/debugging) and Tool Orchestrator Model (tool call validation and plan coordination).
- **Centered Cybernetic Layout:** Centered 840px chat composer (`.vgt-code-composer`) and message history, providing responsive aesthetics across 1080p, 1440p, and 4K monitors.
- **Zero-Latency History Restore:** Instant session rendering directly from memory cache; persistent chat composer visibility; automatic project context restore; sub-millisecond fast-path sync.
- **Dual Permission Modes:**
  - **`VOLLZUGRIFF` (Full Access):** Autonomous execution of safe, moderate, and high-risk workspace actions without confirmation pop-ups. Hard blocks for destructive sabotage (`RiskForbidden`) remain enforced.
  - **`INTERAKTIV` (Normal Mode):** Operator confirmation pop-ups for critical actions. Persisted across sessions and toggled via 1-click cybernetic UI badge.
- **Workspace Authority & Dynamic Path Jail:** `ValidatePathForAccess` resolves paths dynamically against active project workspace root (`CurrentCodeWorkspaceRoot()`), eliminating false jail violations during project development.
- **Write-Enabled Folder Mounts:** `MountFolderSkill` grants `security.MountWrite` permissions and expands lease durations up to 7 days (168 hours / 10,080 minutes).
- **AST Diff Review Engine:** Real-time unified and split diff view for code reviews before and after agent runs.

#### Sphere 2.0 (Personal Operations Desktop OS)
- **Virtual Desktops:** 6 dedicated virtual desktops (`PERSONAL`, `WORK`, `RESEARCH`, `TRAVEL`, `PROJECT`, `INCIDENT`) with spatial state persistence and ambient lighting.
- **Universal Context Bus & Send To...:** 1-click cross-application entity transfers between Writer, Browser, Planner, Research, and Travel tools.
- **AI Track Changes Engine:** Word-level insertions and deletions diff review for documents with operator accept/reject controls in VGT Writer.
- **Research Desk & Provenance:** Evidence clipping workbench that captures citations and sources, producing 1-click synthesized Markdown intelligence briefings.
- **Trip Planner & Risk Correlation:** Intelligent trip lifecycle engine correlating itineraries with real-time Global Watch intelligence feeds, active conflict vectors, and travel alerts.
- **Master Planner & Dependencies:** Hierarchical goal and milestone tracking with automated task dependency graph resolution (`Blocks` / `DependsOn`).

#### Offline Voice & Security Architecture
- **Offline Sherpa-ONNX Voice Pipeline:** 100% offline local speech synthesis and recognition powered by CGo ONNX Runtime bindings (`onnxruntime.dll`, `sherpa-onnx-c-api.dll`) with a quick-mute toggle.
- **Hardened Security & Governance Kernel:** Multi-tier PolicyEngine evaluating capability leases, one-time overrides, and permission modes with blockchain audit logging.
- **Non-Destructive Build Pipeline:** Specialized Wails build workflow (`scripts/build_preserve.ps1`) preserving `./vgt_workspace`, configs, local databases, and models during compilation.
- **Exhaustive System Cartography:** Published [`ARCHITECTURE.md`](ARCHITECTURE.md), an authoritative 2,100+ line technical architecture document detailing all 18 subsystems, 19 operational viewports, and complete data flows.

---

### 1.0.0-beta.3 — Strategic Command Build *(archived)*

- Added the compartmented SHADOW OSINT mode and independent editable doctrine.
- Added a local WebGL 3D command globe with evidence-bound regions.
- Added directed attacker-to-target conflict vectors and military-support relationships.
- Added the editable military/official/geopolitical/economic/energy/cyber/space source registry.
- Added Telegram public-preview acquisition for `militaernews`.
- Added secure RSS discovery and bounded same-origin headline extraction for Web sources.
- Added strict 40–60 item intelligence batches and encrypted SHADOW state.
- Added daily master dossiers and JSON/Markdown exports.
- Removed deterministic and hybrid regional-risk fallback coloring from Global Watch.
- Separated natural hazards from the news feed and automatic AI risk context.
- Added strict SHADOW JSON decoding, evidence validation, size/count boundaries and atomic persistence rollback.
- Reskinned the full AETHEL shell in SHADOW mode and unified Beta V3 release identity.

### 1.0.0-beta.2 — Sovereign Intelligence OS *(archived)*

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

## 🚧 Known Limitations (1.0.0-beta.4)

- No automatic update system
- Single-operator only — no multi-user support
- Groq API key required for Whisper STT (offline fallback: local Sherpa-ONNX & Windows SAPI)
- Sherpa-ONNX requires CGO + GCC compiler + reviewed runtime DLLs (`onnxruntime.dll`, `sherpa-onnx-c-api.dll`)
- Official platform focus: Windows 10/11 x64 (macOS/Linux builds possible without GUI control features)
- No HTTPS (localhost only — TLS optionally upgradeable)
- OSINT quality depends on source availability, provider access and evidence present in completed batches
- Directed conflict vectors appear only in new analyses; archived Beta V2 dossiers are not retroactively rewritten

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
| SHADOW OSINT Command Mode | ✅ Done (beta.3) |
| WebGL Conflict Command Globe | ✅ Done (beta.3) |
| Evidence-Bound Directed Conflict Vectors | ✅ Done (beta.3) |
| 40–60 Item Intelligence Batches | ✅ Done (beta.3) |
| Daily Master Dossiers & Export | ✅ Done (beta.3) |
| AI-Only Regional Risk Authority | ✅ Done (beta.3) |
| VGT Coder (Autonomous Agentic IDE) | ✅ Done (beta.4) |
| Dual Permission Modes (VOLLZUGRIFF / INTERAKTIV) | ✅ Done (beta.4) |
| Sphere 2.0 Personal Operations Desktop OS | ✅ Done (beta.4) |
| Universal Context Bus & Send To... Dispatcher | ✅ Done (beta.4) |
| AI Track Changes & Diff Review Engine | ✅ Done (beta.4) |
| Zero-Latency Conversation Restore & Fast Sync | ✅ Done (beta.4) |
| Complete Technical Cartography (ARCHITECTURE.md) | ✅ Done (beta.4) |

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

Commercial inquiries and custom licensing: [visiongaiatechnology.de](https://visiongaiatechnology.de)

---

<div align="center">

**VISIONGAIATECHNOLOGY – WE ARCHITECT THE FUTURE OF SECURITY.**

[![VGT](https://img.shields.io/badge/VisionGaia-Technology-cyan?style=for-the-badge)](https://visiongaiatechnology.de)

*VGT AETHEL 1.0.0-beta.4 — Sovereign Strategic Intelligence OS & Development Environment // VGT Coder Coding Agent // Sphere 2.0 Desktop OS // Dual Permission Modes (VOLLZUGRIFF / INTERAKTIV) // SHADOW OSINT // WebGL Command Globe // Universal Context Bus // AI Track Changes // Wails 2.15 Desktop // Go 1.26.6 // DPAPI Key Store // AES-256-GCM Sealed Stores // Guard Kernel // Tamper-Evident Blockchain Audit // Sherpa-ONNX Offline TTS // 65+ Frontend Modules // DE/EN/ES/FR/RU // AGPLv3 // Windows 10/11 x64*

</div>
