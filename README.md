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
### Sovereign Intelligence Framework

[![License](https://img.shields.io/badge/License-AGPLv3-blue?style=for-the-badge)](https://www.gnu.org/licenses/agpl-3.0)
[![Version](https://img.shields.io/badge/Version-v1.0.0--beta.1-orange?style=for-the-badge)](#)
[![Status](https://img.shields.io/badge/Status-BETA-yellow?style=for-the-badge)](#)
[![Go](https://img.shields.io/badge/Go-1.21-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Framework](https://img.shields.io/badge/Framework-Wails_Desktop-00ADD8?style=for-the-badge)](#)
[![Platform](https://img.shields.io/badge/Platform-Windows_10%2F11_x64-0078D4?style=for-the-badge&logo=windows)](#)
[![Dependencies](https://img.shields.io/badge/Dependencies-Zero_(pure_stdlib)-brightgreen?style=for-the-badge)](#)
[![Vault](https://img.shields.io/badge/Vault-AES--256--GCM-gold?style=for-the-badge)](#)
[![KeyStore](https://img.shields.io/badge/KeyStore-Windows_DPAPI-purple?style=for-the-badge)](#)
[![TTS](https://img.shields.io/badge/TTS-Sherpa--ONNX_(Offline)-brightgreen?style=for-the-badge)](#-sherpa-onnx-integration-guide)
[![Audit](https://img.shields.io/badge/Audit_Log-Blockchain--chained-purple?style=for-the-badge)](#)
[![VGT](https://img.shields.io/badge/VGT-VisionGaiaTechnology-cyan?style=for-the-badge)](https://visiongaiatechnology.de)

**SOVEREIGN AI KERNEL · NATIVE DESKTOP APP · OPERATOR-GATED EXECUTION · OFFLINE TTS**

<img width="50%" alt="VGT AETHEL Neural Interface" src="https://github.com/user-attachments/assets/509b3a60-ea7f-44a8-8c11-bc61bbbcc188" />

</div>

---

## ⚠️ BETA SOFTWARE — EXPERIMENTAL R&D

VGT AETHEL is a **Proof of Concept (PoC)** and active research project at VisionGaia Technology. It is **not** a certified or production-ready product.

**Use at your own risk.** The software may contain security vulnerabilities, bugs or unexpected behavior. It may break your environment if misconfigured.

**Do not deploy in critical production environments** without thoroughly auditing the code and understanding the implications.

Found a vulnerability or have an improvement? **Open an issue or contact us.**

---

<img width="2560" height="1351" alt="image" src="https://github.com/user-attachments/assets/5bbcd1a1-930e-4570-aa59-776180f12ccb" />


---

## 🔍 What is VGT AETHEL?

AETHEL is not a chatbot. It is a **local sovereign AI kernel** — personal assistant, code agent, computer-control agent, voice assistant, task orchestrator and security kernel in one native desktop application.

```
Conventional AI Agents:
  Unsecured Python environment    → full system privileges
  No governance layer             → AI executes what it wants
  No audit trail                  → nothing is logged
  No operator gate                → changes applied silently
  Cloud-dependent TTS             → sends voice data externally

VGT AETHEL:
  Go Cortex (pure stdlib)         → zero external attack surface
  Wails native desktop app        → embedded frontend, no browser required
  Guard Kernel (policy engine)    → every tool call risk-scored before execution
  Operator gate                   → Moderate/High/Critical requires human confirmation
  Blockchain audit log            → every action chained and tamper-evident
  Windows DPAPI key store         → master key bound to OS user, not plaintext on disk
  AES-256-GCM sealed stores       → all local state encrypted at rest
  Sherpa-ONNX TTS                 → fully offline voice output, no cloud dependency
  Capability-based agent profiles → hard permission boundaries per role
  File snapshot & restore         → automatic rollback before destructive operations
```

AETHEL implements a **strict separation of intelligence, execution and communication** — the governance layer that most AI agent runtimes are missing.

<img width="2560" height="1351" alt="image" src="https://github.com/user-attachments/assets/9f36524d-4c74-4bfb-8eb3-ea3a375eba45" />


---

## 🏛️ Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                   OPERATOR (WAILS DESKTOP)                    │
│        ES6 Frontend — 16 Modules — Embedded via go:embed     │
│  Chat · Run Center · Agent Builder · Personal Mode · Voice   │
│  Security · Tasks · Personas · Memory · Diagnostics          │
├──────────────────────────────────────────────────────────────┤
│                    REST API v1 (HTTP/JSON)                    │
├──────────────────────────────────────────────────────────────┤
│              GO CORTEX — MODULAR (~77 source files)           │
│                                                              │
│  ┌──────────────┬──────────────┬─────────────┬────────────┐ │
│  │  Chat Engine │ Skills Layer │ Guard Kernel│  Voice     │ │
│  │  (Streaming) │ (Modular)    │ (Policy +   │  Engine    │ │
│  │              │ skills_gui   │  Audit Log) │  Sherpa    │ │
│  │              │ skills_fs    │             │  SAPI5     │ │
│  │              │ skills_browser│            │  Whisper   │ │
│  └──────────────┴──────────────┴─────────────┴────────────┘ │
│                                                              │
│  ┌──────────────┬──────────────┬─────────────┬────────────┐ │
│  │  Run Engine  │ Sealed Store │ Key Store   │ File       │ │
│  │  (Persistent │ (AES-256-GCM │ (DPAPI)     │ Snapshots  │ │
│  │   State      │  at rest)    │             │ & Restore  │ │
│  │   Machine)   │              │             │            │ │
│  └──────────────┴──────────────┴─────────────┴────────────┘ │
├──────────────────────────────────────────────────────────────┤
│                    NEXUS MEMORY STORE                         │
│         ./vgt_workspace  (Sealed JSON / Sled DB)             │
└──────────────────────────────────────────────────────────────┘
```

<img width="2560" height="1351" alt="image" src="https://github.com/user-attachments/assets/711672e6-59ef-4956-9a6a-7eeade41962f" />


---

## 📊 v0.6.0 → v1.0.0-beta.1 — What Changed

| Criterion | v0.6.0 Alpha | v1.0.0-beta.1 | Status |
|---|---|---|---|
| **Platform Format** | Go CLI + local webserver (localhost:3000) | Native Wails Desktop App | 🚀 Upgrade |
| **Key Store** | Plaintext `vault.key` on disk | Windows DPAPI (`config.key.dpapi`) — bound to OS user | 🔒 Hardened |
| **Local Databases** | Plaintext JSON in workspace | Encrypted sealed files (`AETHEL-SEAL-v1:`) | 🔒 Hardened |
| **Code Structure** | Monolithic (few large files) | Highly modular (~77 Go source files) | 🧹 Refactored |
| **Agent Runs** | Temporary web sessions | Persistent, pausable, budgeted runs with state machine | ✨ New |
| **TTS** | Cloud-dependent (Edge TTS) | Fully offline (Sherpa-ONNX / SAPI5 fallback) | 🎙️ Upgrade |
| **File Safety** | Path jail check only | Active snapshot & restore engine | 🛡️ Hardened |
| **Platform Focus** | Windows / macOS / Linux | Official focus: Windows 10/11 x64 | 🎯 Focused |
| **Frontend Modules** | 10 JS modules | 16 JS modules | ✨ Expanded |

---

<img width="2560" height="1351" alt="image" src="https://github.com/user-attachments/assets/c225fff9-bfa4-47a5-9d42-775956af4add" />


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

### Permission Leases

Operators can grant temporary, scoped permission unlocks:

- Time-bound via `ExpiresAt` timestamp
- Scope: app filter, action filter, forbidden targets
- Persisted in sealed store (`active_leases`)
- Revoked automatically on expiry

### Blockchain Audit Log

Every action is chained and tamper-evident:

```
Entry N:   action + SHA-256(Entry N-1)
Entry N+1: action + SHA-256(Entry N)
```

`ValidateChain()` performs complete cryptographic verification of the entire log on every server start.

---

<img width="2560" height="1351" alt="image" src="https://github.com/user-attachments/assets/5bbad35d-3035-4fc1-b8f7-f3bbe3e416ca" />



## 🔐 Security Architecture (Beta v1)

### Windows DPAPI Key Store

In v0.6.0, the AES vault master key was stored as plaintext `vault.key` on disk. Beta v1 replaces this entirely:

- Master key is encrypted using **Windows Data Protection API (DPAPI)** via `key_store_windows.go`
- The key (`config.key.dpapi`) is cryptographically bound to the logged-in Windows user
- An automated one-time migration removes old plaintext keys

### Sealed Local Stores

All local state — configuration, tasks, agent runs — is stored encrypted:

```
Storage prefix:  AETHEL-SEAL-v1:
Encryption:      AES-256-GCM
Implementation:  sealed_store.go
```

Nothing in `./vgt_workspace` is readable plaintext.

### Capability-Based Agent Profiles

Four hard, data-driven agent profiles with predefined permission boundaries — not just prompt-level restrictions:

| Profile | Permissions |
|---|---|
| **Researcher** | Read-only filesystem and browser access. No write rights. |
| **Developer** | Write rights, `sys_exec` — strictly scoped |
| **Browser Operator** | Media and visible web control only |
| **Personal Assistant** | Personalized mode — no system write rights |

### File Snapshot & Restore (`file_snapshots.go`)

Before any file modification, AETHEL automatically creates an encrypted in-memory snapshot (up to 5 MB). On error, the previous state can be restored with a single confirmation.

### Durable Agent Runs (`run_engine.go`)

Agent runs are now a persistent state machine:

```
queued → running → waiting_approval → completed
                ↓                   ↓
              paused             failed
```

- Each run has a unique ID and full trace log
- Budget limit: configurable max token cost (USD) per run
- Crash recovery: in-flight runs auto-pause on restart and can be explicitly resumed

---

<img width="2560" height="1351" alt="image" src="https://github.com/user-attachments/assets/8a217a5f-6792-499a-b37c-3c8727b52116" />


## 🔧 Skills & Tool Capabilities

| Tool | Capability | Risk Level |
|---|---|---|
| `gui_control / move` | Move mouse cursor | 🟢 Low (autonomous) |
| `gui_control / position` | Query mouse position | 🟢 Low (autonomous) |
| `gui_control / click` | Left click | 🟡 Moderate |
| `gui_control / right` | Right click | 🟡 Moderate |
| `gui_control / double` | Double click | 🟡 Moderate |
| `gui_control / type` | Type text input | 🔴 High |
| `gui_control / press` | Key combination | 🔴 High |
| `fs_read_file` | Read file | 🟢 Low (autonomous) |
| `fs_list_dir` | List directory | 🟢 Low (autonomous) |
| `fs_write_file` | Write file (with auto-snapshot) | 🟡 Moderate |
| `fs_mount_folder` | Mount folder | 🟡 Moderate |
| `sys_exec_cmd` | Execute system command | 🔴 High |
| `web_browser` | Open browser | 🟡 Moderate |
| `nexus_save` | Save to memory | ⚪ Safe (autonomous) |
| `nexus_recall` | Recall from memory | ⚪ Safe (autonomous) |
| `agent_handoff` | Hand off to ChatGPT / Gemini / Cursor | 🟡 Moderate |
| `viewport_screenshot` | Desktop screenshot (cached JPEG) | ⚪ Safe (autonomous) |

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

AETHEL uses Sherpa-ONNX for fully local, privacy-compliant text-to-speech output with no internet connection required. Follow these four steps to enable it.

### Step 1 — Configure CGO and Go Bindings

Sherpa-ONNX uses native C++ libraries. A GCC compiler is required — on Windows, [w64devkit](https://github.com/skeeto/w64devkit) is recommended.

```bash
# Enable CGO
set CGO_ENABLED=1

# Install Go bindings
go get github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx
go mod tidy
```

### Step 2 — Place Native DLLs (Windows)

The compiled C++ libraries must be placed next to `aethel.exe` at runtime.

Download all `.dll` files from the official repository:
**→ [Sherpa-ONNX Go Windows Native Libs (GNU)](https://github.com/k2-fsa/sherpa-onnx-go-windows/tree/master/lib/x86_64-pc-windows-gnu)**

Copy **all DLLs** (not just the API DLL) into your build root directory:

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

AETHEL does not download models at runtime for security reasons. Place models manually under `./vgt_workspace/models/sherpa/`.

**Recommended models:**

| Model | Language | Size | Download |
|---|---|---|---|
| **KittenTTS** | English | ~30 MB | [kitten-nano-en-v0_1-fp16.tar.bz2](https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2) |
| **Kokoro v1.0** | Multilingual | ~80 MB | [kokoro-multi-lang-v1_0.tar.bz2](https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-multi-lang-v1_0.tar.bz2) |

**Required directory structure:**

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
            │   ├── model.fp16.onnx      ← filename must be preserved exactly
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

Always build with CGO enabled:

```bash
set CGO_ENABLED=1
go build -ldflags="-s -w" -o aethel.exe .
```

> **Without CGO (`CGO_ENABLED=0`):** The stub module `voice_sherpa_stub.go` is compiled automatically. The application builds and runs, but local audio output will return an error at runtime. SAPI5 fallback remains active.

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

### Tools & Skills

| Method | Endpoint | Function |
|---|---|---|
| `POST` | `/v1/tools/execute` | Execute skill (with security gate) |
| `GET/POST/DELETE` | `/v1/kernel/tasks/` | Task engine CRUD |

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
| `GET` | `/v1/memory/search` | Semantic search |

### Vault & Setup

| Method | Endpoint | Function |
|---|---|---|
| `GET/POST/DELETE` | `/v1/secrets` | AES-256-GCM sealed vault |
| `GET/POST` | `/v1/setup` | First-run configuration |
| `GET` | `/v1/diagnostics` | Privacy-safe system report |

---

## 🖥️ Frontend Modules (16 Modules)

| Module | Size | Responsibility |
|---|---|---|
| `app.js` | — | Root bootstrap, import orchestrator |
| `state.js` | 4.0 KB | Global reactive state store |
| `api.js` | 2.6 KB | HTTP wrapper, fetch helpers |
| `chat.js` | 24.3 KB | Chat terminal, Markdown rendering, tool approval UI |
| `voice.js` | 21.9 KB | STT/TTS, wake-word, voice sphere, approval routing |
| `control.js` | 3.8 KB | Live operator panel, screenshot polling |
| `security.js` | 11.7 KB | Leases UI, audit log viewer, threat display |
| `tasks.js` | 8.5 KB | Task manager UI, CRUD, status monitoring |
| `memory.js` | 5.2 KB | Nexus memory UI, search |
| `secrets.js` | 3.8 KB | Vault UI, key management |
| `ui.js` | 6.7 KB | Navigation, panels, animations, themes |
| `agent_builder.js` | — | Agent team configuration |
| `personal_mode.js` | — | Persona, humor/initiative sliders, journal |
| `approval_dialog.js` | — | Tool call approval dialogs |
| `run_center.js` | — | Run state machine UI, trace logs, evidence screenshots |
| `diagnostics.js` | — | Privacy-safe system report export |

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
# Wails desktop window opens automatically
```

> Without CGO, build with `CGO_ENABLED=0` — Sherpa-ONNX is replaced by stub, SAPI5 fallback active.

---

## ⚙️ Technical Specifications

| Metric | Value |
|---|---|
| **Language** | Go 1.21 |
| **Framework** | Wails Desktop (WebView2 — embedded frontend) |
| **External Dependencies** | Zero stdlib / CGO only for Sherpa-ONNX |
| **Platform** | Windows 10/11 x64 (official) |
| **Backend Source Files** | ~77 Go source files |
| **Frontend Modules** | 16 JS modules (embedded via `go:embed`) |
| **Vault Encryption** | AES-256-GCM |
| **Key Storage** | Windows DPAPI (`config.key.dpapi`) |
| **Local State** | `AETHEL-SEAL-v1:` sealed encrypted stores |
| **Audit Log** | SHA-256 blockchain-chained + `ValidateChain()` |
| **TTS Primary** | Sherpa-ONNX (offline ONNX neural models) |
| **TTS Fallback** | Windows SAPI5 (local, no API key) |
| **STT Primary** | Groq Whisper `whisper-large-v3-turbo` |
| **Memory Persistence** | `./vgt_workspace` (sealed JSON / Sled DB) |
| **License** | AGPLv3 |

---

## 📋 Changelog

### v1.0.0-beta.1 — Native Desktop, DPAPI, Sealed Stores, Sherpa-ONNX *(Current)*

#### Architecture & Framework

- **Wails Desktop (WebView2):** migrated from Go CLI + localhost:3000 webserver to a true native Windows desktop application. The frontend is compiled directly into the binary via `go:embed`. No browser required.
- **Full Backend Modularization:** monolithic `skills.go` (~1,200 lines) and `main.go` (~1,450 lines) decomposed into ~77 dedicated source files. Skills split into functional modules: `skills_gui.go`, `skills_fs.go`, `skills_browser.go` and more.
- **Frontend Expansion:** 10 → 16 JS modules with dedicated files for `agent_builder.js`, `personal_mode.js`, `approval_dialog.js`, `run_center.js` and `diagnostics.js`.

#### Security Hardening

- **Windows DPAPI Key Store (`key_store_windows.go`):** master key for AES vault is now cryptographically bound to the logged-in Windows user via DPAPI. Plaintext `vault.key` is gone. Automated one-time migration removes old keys.
- **Sealed Local Stores (`sealed_store.go`):** all local state — config, tasks, agent runs — stored under `AETHEL-SEAL-v1:` prefix with AES-256-GCM encryption. Nothing readable as plaintext in `./vgt_workspace`.
- **Symlink-Resistant Path Jail:** switched from simple prefix check to `filepath.Rel` + `filepath.EvalSymlinks` — the only reliable method on Windows to block path traversal via symlinks.
- **Shell Metacharacter Blocker:** regex filter `[;&|><` + "`" + `$]|$$|\r|\n` blocks shell injection before commands reach PowerShell/sh context.
- **Blockchain Validation:** `ValidateChain()` performs complete SHA-256 cryptographic verification of the entire audit log chain on startup.
- **Frontend XSS Protection:** `DOMParser`-based local HTML sanitization + `jsArg()` parameter escaping blocks malicious code injection via LLM Markdown responses.
- **Secure File Permissions:** sensitive JSON databases created on Linux/macOS with `0600` (files) and `0700` (directories).
- **Header Hardening & CORS:** `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `X-Frame-Options: DENY` set globally. Wildcard CORS (`*`) removed.
- **Session-ID Whitelisting:** session IDs validated via `^session_[A-Za-z0-9_-]{1,80}$` — path traversal via manipulated IDs impossible.
- **Upload Limits:** request bodies capped at 16 MB to prevent memory exhaustion.

#### Agent Runs

- **Durable State Machine (`run_engine.go`):** persistent agent runs with unique IDs, lifecycle states (`queued → running → waiting_approval → completed / failed / paused`), configurable USD cost budget per run, and crash recovery (auto-pause + explicit resume).
- **Capability-Based Profiles:** four hard permission profiles (Researcher / Developer / Browser Operator / Personal Assistant) — data-driven, not just prompt-level.
- **File Snapshots (`file_snapshots.go`):** automatic encrypted snapshot (up to 5 MB) before any file modification. One-click restore on error.

#### Voice & Audio

- **Sherpa-ONNX TTS (`voice_sherpa_cgo.go`):** fully offline neural TTS via ONNX models — replaces cloud-dependent Edge TTS. CGO required; `voice_sherpa_stub.go` compiles without CGO (SAPI5 fallback active).
- **SAPI5 Fallback:** Windows SAPI5 as local fallback if Sherpa-ONNX cannot initialize.

#### New Features

- **Run Center:** real-time agent step monitoring with cost display, tool calls, trace logs and visual before/after evidence screenshots.
- **Personal Mode:** dedicated interface for humor, honesty and initiative sliders; interests, goals, journal and memory entries.
- **Ledger-Based API Cost Tracker:** all token transactions logged to `api_costs.json`. Daily (TODAY) and monthly (MONTH) costs shown live in header HUD.
- **Custom Personas (Aethel Gems):** user-defined system prompts configurable system-wide or per agent role (Orchestrator, Builder, Reviewer, etc.).
- **Provider Registry & Health Checks:** automatic availability validation of connected LLMs (OpenAI, Gemini, Anthropic, DeepSeek, Groq, Ollama) with dynamic error signaling in GUI header.
- **Privacy-Safe Diagnostics (`diagnostics.go`):** export system/performance report with strict filtering of prompts, paths, tokens, passwords and tool parameters.
- **Updated Model Registry:** GPT-5.5, GPT-5.4, Gemini 3.5/3.1, Claude 4.5/4.6, DeepSeek v4 with per-1M-token pricing including cache-hit discounts.
- **Unit Test Suite (`verify_security_test.go`):** native Go tests for jails, leases, ID conventions and tamper resistance.

---

### v0.6.0-alpha — Security Hardening & Feature Expansion *(archived)*

Path jail hardened via `filepath.Rel` + `filepath.EvalSymlinks`. Shell injection blocker added. Blockchain audit `ValidateChain()`. Frontend XSS protection. API cost tracker. Custom persona system. Model registry updated (GPT-5.5, Gemini 3.5, Claude 4.5, DeepSeek v4).

---

### v0.5.0-alpha — Foundation Release *(archived)*

Go Cortex (pure stdlib). Guard Kernel. AES-256-GCM vault. Blockchain audit log. Voice STT/TTS. Task engine. Screenshot engine. Nexus memory.

---

## 🚧 Known Limitations (v1.0.0-beta.1)

- No automatic update system
- Single-operator only — no multi-user support
- Groq API key required for Whisper STT (offline fallback: Windows SAPI)
- Sherpa-ONNX requires CGO + GCC compiler + manual DLL and model setup (see integration guide)
- Official platform focus: Windows 10/11 x64 (macOS/Linux builds possible without GUI control features)
- No HTTPS (localhost only — TLS optionally upgradeable)

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

---

## 🔗 VGT Ecosystem

| Tool | Type | Purpose |
|---|---|---|
| 🧠 **VGT AETHEL** | **Sovereign AI Kernel** | Local AI agent runtime with operator governance — you are here |
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

*VGT AETHEL v1.0.0-beta.1 — Sovereign AI Kernel // Wails Native Desktop // Go Pure Stdlib // DPAPI Key Store // AES-256-GCM Sealed Stores // Guard Kernel // Blockchain Audit Log // Sherpa-ONNX Offline TTS // Durable Agent Runs // Capability Profiles // File Snapshots // 16-Module Frontend // AGPLv3 // Windows 10/11 x64*

</div>
