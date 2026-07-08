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

[![License](https://img.shields.io/badge/License-Proprietary-red?style=for-the-badge)](#)
[![Version](https://img.shields.io/badge/Version-v0.6.0-orange?style=for-the-badge)](#)
[![Status](https://img.shields.io/badge/Status-ALPHA-red?style=for-the-badge)](#)
[![Go](https://img.shields.io/badge/Go-1.21-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-Windows_10%2F11_x64-0078D4?style=for-the-badge&logo=windows)](#)
[![Dependencies](https://img.shields.io/badge/Dependencies-Zero_(pure_stdlib)-brightgreen?style=for-the-badge)](#)
[![Vault](https://img.shields.io/badge/Vault-AES--256--GCM-gold?style=for-the-badge)](#)
[![Audit](https://img.shields.io/badge/Audit_Log-Blockchain--chained-purple?style=for-the-badge)](#)
[![VGT](https://img.shields.io/badge/VGT-VisionGaiaTechnology-cyan?style=for-the-badge)](https://visiongaiatechnology.de)

**SOVEREIGN AI KERNEL · ZERO EXTERNAL DEPENDENCIES · OPERATOR-GATED EXECUTION**

<img width="50%" alt="VGT AETHEL Neural Interface" src="https://github.com/user-attachments/assets/509b3a60-ea7f-44a8-8c11-bc61bbbcc188" />

</div>

---

## ⚠️ ALPHA SOFTWARE — EXPERIMENTAL R&D

VGT AETHEL is a **Proof of Concept (PoC)** and active research project at VisionGaia Technology. It is **not** a certified or production-ready product.

**Use at your own risk.** The software may contain security vulnerabilities, bugs or unexpected behavior. It may break your environment if misconfigured.

**Do not deploy in critical production environments** without thoroughly auditing the code and understanding the implications.

Found a vulnerability or have an improvement? **Open an issue or contact us.**

---

<img width="1920" height="1009" alt="image" src="https://github.com/user-attachments/assets/7a026e26-f8e0-40bb-84c8-008c92b61513" />


## 🔍 What is VGT AETHEL?

AETHEL is not a chatbot. It is a **local sovereign AI kernel** — personal assistant, code agent, computer-control agent, voice assistant, task orchestrator and security kernel in one system.

```
Conventional AI Agents:
  Unsecured Python environment    → full system privileges
  No governance layer             → AI executes what it wants
  No audit trail                  → nothing is logged
  No operator gate                → changes applied silently

VGT AETHEL:
  Go Cortex (pure stdlib)         → zero external attack surface
  Guard Kernel (policy engine)    → every tool call risk-scored before execution
  Operator gate                   → Moderate/High/Critical requires human confirmation
  Blockchain audit log            → every action chained and tamper-evident
  AES-256-GCM vault               → secrets never in plaintext
  Voice approval                  → operator can confirm via voice
```

AETHEL implements a **strict separation of intelligence, execution and communication** — the governance layer that most AI agent runtimes are missing.

---

## 🏛️ Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        OPERATOR (UI)                          │
│           ES6 Frontend — Vanilla HTML / CSS / JS             │
│   Chat Terminal · Live Operator · Security · Tasks · Memory   │
├──────────────────────────────────────────────────────────────┤
│                    REST API v1 (HTTP/JSON)                    │
├──────────────────────────────────────────────────────────────┤
│                    GO CORTEX (Backend)                        │
│                                                              │
│  ┌─────────────┬───────────────┬──────────────┬───────────┐ │
│  │ Chat Engine │  Skills Layer │ Guard Kernel │  Voice    │ │
│  │ (Streaming) │  (Tool Exec)  │ (Policy +    │  Engine   │ │
│  │             │               │  Audit Log)  │  STT/TTS  │ │
│  └─────────────┴───────────────┴──────────────┴───────────┘ │
│                                                              │
│  main.go · skills.go · guard.go · voice.go                   │
│  secret.go · task_engine.go                                  │
├──────────────────────────────────────────────────────────────┤
│                    NEXUS MEMORY STORE                         │
│          ./vgt_workspace  (Sled DB / JSON persistence)       │
└──────────────────────────────────────────────────────────────┘
```

### Codebase Overview

| File | Lines | Size | Function |
|---|---|---|---|
| `main.go` | 1,452 | 39.9 KB | HTTP server, routing, state management |
| `skills.go` | 1,201 | 36.9 KB | Tool skills, GUI control, screenshot engine |
| `guard.go` | 581 | 15.9 KB | Policy engine, lease manager, audit logger |
| `voice.go` | 403 | 11.9 KB | STT (Whisper/Groq), TTS (Edge/SAPI) |
| `task_engine.go` | 386 | 9.4 KB | Background scheduler, autonomous tasks |
| `secret.go` | 226 | 4.7 KB | AES-256-GCM secrets vault |
| `verify_security_test.go` | — | 5.7 KB | Security unit tests |
| **Frontend (10 modules)** | — | 92.5 KB | Modular ES6 UI |

**Total backend:** ~4,249 lines of Go · **Total frontend:** 10 JS modules

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

### Permission Leases

Operators can grant temporary, scoped permission unlocks:

- Time-bound via `ExpiresAt` timestamp
- Scope: app filter, action filter, forbidden targets
- Persisted in `./vgt_workspace/active_leases.json`
- Revoked automatically on expiry

### Blockchain Audit Log

Every action is chained:

```
Entry N:  action + SHA-256(Entry N-1)
Entry N+1: action + SHA-256(Entry N)
```

Tamper detection runs at every server start. Stored in `./vgt_workspace/security_audit.json`.

---

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
| `fs_write_file` | Write file | 🟡 Moderate |
| `fs_mount_folder` | Mount folder | 🟡 Moderate |
| `sys_exec_cmd` | Execute system command | 🔴 High |
| `web_browser` | Open browser | 🟡 Moderate |
| `nexus_save` | Save to memory | ⚪ Safe (autonomous) |
| `nexus_recall` | Recall from memory | ⚪ Safe (autonomous) |
| `agent_handoff` | Hand off to ChatGPT / Gemini / Cursor | 🟡 Moderate |
| `viewport_screenshot` | Desktop screenshot (cached JPEG) | ⚪ Safe (autonomous) |

---

## 🎙️ Voice System

### Speech-to-Text (STT)

| Provider | Model | Mode |
|---|---|---|
| **Groq Whisper** (primary) | `whisper-large-v3-turbo` | Cloud (API key required) |
| **Windows SAPI** (fallback) | Native | Local / offline |

### Text-to-Speech (TTS)

| Provider | Quality | Mode |
|---|---|---|
| **Edge TTS** (primary) | High (neural) | Local (no API key) |
| **Windows SAPI** (fallback) | Standard | Local / offline |

### Voice Approval Commands

| Intent | Phrases |
|---|---|
| ✅ Approve | *"ja", "ok", "freigeben", "bestätigen", "erlauben", "mach", "go"* |
| ❌ Reject | *"nein", "nee", "stop", "block", "ablehnen"* |
| 🔐 High-Risk Approve | *"bestätige", "confirm"* (explicit) |

Fuzzy matching active — operator does not need exact phrasing.

---

## 🖥️ Live Operator (Viewport Control)

Real-time desktop visibility for the operator during AI execution.

| Feature | Detail |
|---|---|
| **Screenshot Engine** | GDI+ via PowerShell, JPEG 70% quality |
| **Resolution** | Auto-downscaled to max. 1280px width |
| **Multi-Monitor** | Cursor-follows: always the monitor with the mouse cursor |
| **Refresh Rate** | 800ms polling interval |
| **Caching** | In-memory cache, 800ms TTL (`sync.Mutex`) |
| **Conflict Protection** | Mutex prevents parallel PowerShell processes |
| **Error Fallback** | Last valid image used on transient capture failure |

---

## ⚙️ Task Engine — Autonomous Scheduling

| Feature | Detail |
|---|---|
| **Scheduler** | Background goroutine, tick every 5 seconds |
| **Task Types** | `once`, `recurring` (cron-style) |
| **Status Values** | `pending`, `running`, `completed`, `failed`, `cancelled` |
| **Persistence** | `./vgt_workspace/tasks.json` |
| **Security Integration** | All tasks pass through the full policy engine |
| **Max Execution Time** | Configurable per task |

---

## 🔑 Secrets Vault

| Feature | Detail |
|---|---|
| **Encryption** | AES-256-GCM (authenticated encryption) |
| **Key Derivation** | Randomly generated 256-bit key |
| **Storage** | `./vgt_workspace/secret_vault.enc` + `vault.key` |
| **API** | REST CRUD via `/v1/secrets` |
| **Git Protection** | `.gitignore` blocks commit of `.enc` / `.key` files |

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
| `POST` | `/v1/audio/speech` | Text-to-speech synthesis |
| `GET` | `/v1/audio/voices` | Available voices |
| `POST` | `/v1/audio/transcribe` | Speech-to-text (Whisper) |
| `GET` | `/v1/audio/health` | Audio stack health |
| `POST` | `/v1/audio/test` | TTS connection test |

### Security

| Method | Endpoint | Function |
|---|---|---|
| `GET/POST` | `/v1/security/leases` | Manage permission leases |
| `GET` | `/v1/security/audit` | Blockchain audit log |
| `GET` | `/v1/security/status` | Security status |

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
| `GET/POST/DELETE` | `/v1/secrets` | AES-256-GCM vault |
| `GET/POST` | `/v1/setup` | First-run configuration |

---

## 🖥️ Frontend Modules

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

---

## 🚀 Quick Start

```bash
# 1. Copy environment config
cp .env.example .env
# Set GROQ_API_KEY=gsk_your_key

# 2. Build and run
go build -o aethel.exe .
./aethel.exe
```

| Endpoint | URL |
|---|---|
| Interface | `http://localhost:3000` |
| API Gateway | `http://localhost:3000/v1` |
| Health Check | `http://localhost:3000/health` |

**Persistence:** All memory, tasks, leases and audit log stored in `./vgt_workspace/`.

---

## ⚙️ Technical Specifications

| Metric | Value |
|---|---|
| **Language** | Go 1.21 |
| **External Dependencies** | Zero — pure stdlib |
| **Platform** | Windows 10/11 x64 |
| **Runtime Port** | `localhost:3000` |
| **Backend Size** | ~4,249 lines Go |
| **Frontend Size** | ~92.5 KB (10 modules, vanilla JS) |
| **Vault Encryption** | AES-256-GCM |
| **Audit Log** | SHA-256 blockchain-chained |
| **STT Primary** | Groq Whisper `whisper-large-v3-turbo` |
| **TTS Primary** | Edge TTS (neural, local, no API key) |
| **Screenshot Engine** | GDI+ via PowerShell, JPEG 70% |
| **Memory Persistence** | `./vgt_workspace` (JSON + Sled DB) |
| **License** | Proprietary |

---

## 📋 Changelog

### v0.6.0 — Security Hardening & Feature Expansion

> Focused release: symlink-resistant path jail, shell injection blocker, blockchain audit validation, frontend XSS protection, API cost tracker, custom persona system.

#### 🔐 Security Notes

- **Path Jail (`skills.go`):** Switched from simple prefix checks to `filepath.Rel` combined with `filepath.EvalSymlinks` — the only reliable method under Windows to block path traversal via symbolic links.
- **Shell Blocker (`skills.go`):** Regex filter `[;&|><` + "`" + `$]|$$|\r|\n` reliably blocks shell injection attempts before commands reach a shell context (PowerShell/sh).
- **Blockchain Validation (`guard.go`):** The `ValidateChain()` function performs a complete cryptographic verification (SHA-256) of the entire audit log chain.
- **XSS Protection (`chat.js`):** `DOMParser`-based local HTML sanitization combined with parameter escaping in `jsArg()` reliably prevents malicious code injection via LLM Markdown responses into the UI.

---

#### 📊 v0.5.0-alpha vs. v0.6.0 — Feature Comparison

| Feature / Module | v0.5.0-alpha | v0.6.0 | Status |
|---|---|---|---|
| **Codebase Cleanup** | Contained unused Rust code remnants (`Cargo.toml`, `crates/vgt-*` subdirectories) | Pure Go/Wails project. All legacy artifacts removed from root directory and GitHub mirror | 🧹 Cleaned |
| **API Models & Pricing** | Outdated model registry with stale price calculations | Updated frontier model registry (GPT-5.5, GPT-5.4, Gemini 3.5/3.1, Claude 4.5/4.6, DeepSeek v4). Prices per 1M tokens incl. cache-hit discounts | 🧠 Updated |
| **API Cost Tracker** | No cost overview or tracking | Real-time API cost tracker. Logs all prompts to `api_costs.json`, displays daily (TODAY) and monthly (MONTH) costs live in header HUD | 💸 New |
| **Custom Personas (Gems)** | Behavior only controllable via fixed system prompt | Custom Personas System. Create custom Aethels with dedicated system prompts, configurable system-wide or per role in the agent team | 🤖 New |
| **UI Architecture & Tabs** | 9 tabs only. Personas were provisionally squeezed into a small card in the settings tab (with layout errors) | 10 tabs (incl. Persona Registry). Dedicated, clear tab in two-column layout (form & list) with flex-optimized, readable buttons | 🎨 Optimized |
| **Path Jail Security** | Unsafe directory prefix check (vulnerable to traversal via symlinks and sibling-jails) | Hard path jail via `filepath.Rel` and symlink resolution via `filepath.EvalSymlinks` | 🛡️ Hardened |
| **Shell Injection Protection** | No input validation for metacharacters before shell execution | Active Shell Metacharacter Blocker via regexp in `skills.go`. Interpreter blacklist tightened | 🛡️ Hardened |
| **Audit Log Validation** | No blockchain integrity verification in running code | Tamper-evident audit log with active integrity check via `ValidateChain()` | 🛡️ Hardened |
| **Frontend XSS Protection** | Unsanitized HTML injections possible in chat history and agent logs | Sanitized HTML Engine in frontend. Markdown, reasoning logs, tool arguments and IDs are filtered before display | 🛡️ Hardened |
| **Unit Tests** | Outdated, partially broken bash check scripts | Native Go test suite (`verify_security_test.go`) automatically verifies jails, leases, ID conventions and tamper resistance | 🧪 New |

---

#### 📝 Detailed Changelog

##### 1. Security & Hardening (Backend)

- **Symlink-Resistant Jail:** `skills.go` now resolves all symlinks before checking absolute file paths. Breakout attempts via symlinks pointing to directories outside the jail are blocked.
- **Secure File Permissions:** All sensitive JSON databases (`aethel_config.json`, `active_leases.json`, `security_audit.json`) are created on Linux/macOS with restrictive file permissions `0600` (owner: read/write) and directory permissions `0700`.
- **Header Hardening & CORS:** Security headers `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer` and `X-Frame-Options: DENY` are now globally set in `main.go`. Wildcard CORS (`Access-Control-Allow-Origin: *`) removed to prevent unauthorized browser-based access.
- **Session-ID Whitelisting:** Session IDs are strictly validated via regex `^session_[A-Za-z0-9_-]{1,80}$`. Path traversal via manipulated session IDs is impossible.
- **Upload Limits:** Request bodies capped at 16 MB in the API handler to prevent memory exhaustion (DoS attacks).

##### 2. UI/UX & Layout Optimizations (Frontend)

- **Tab: Persona Registry:** Persona management moved to a dedicated navigation view.
- **Symmetric Buttons:** The `SAVE` and `NEW / CLEAR` buttons in the persona form now share exactly 50/50 width, eliminating the display errors (squashed buttons) of the previous version.
- **Optgroup Model Structure:** The model selector groups available AIs clearly by provider (OpenAI, Gemini, Claude, DeepSeek, Groq, Ollama).

##### 3. New Features

- **Ledger-Based API Cost Tracker:** Automatic recording of all token transactions. Cumulative daily and monthly costs visible in the header HUD in real time.
- **Custom Personas (Aethel Gems):** User-defined prompts can be set system-wide in the sidebar dropdown or per role in the agent team (Orchestrator, Builder, Reviewer, etc.).

---

## 🚧 Known Limitations (v0.6.0)

> This is an alpha release for local operator use only. Not suitable for production environments.

- No automatic update system
- Single-operator only — no multi-user support
- Groq API key required for Whisper STT (offline fallback: Windows SAPI)
- GUI control is Windows-only (GDI+ / PowerShell)
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
| Cross-platform support (Linux, macOS) | 🔜 Planned |
| HTTPS / TLS out of the box | 🔜 Planned |

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

AETHEL is a free R&D project. Every contribution flows directly into development.

[![Donate](https://img.shields.io/badge/Donate-PayPal-00457C?style=for-the-badge&logo=paypal)](https://paypal.me/dergoldenelotus)

| Method | Address |
|---|---|
| **PayPal** | [paypal.me/dergoldenelotus](https://paypal.me/dergoldenelotus) |
| **Bitcoin** | `bc1q3ue5gq822tddmkdrek79adlkm36fatat3lz0dm` |
| **ETH / USDT (ERC-20)** | `0xD37DEfb09e07bD775EaaE9ccDaFE3a5b2348Fe85` |

---

## 📄 License

Proprietary · © 2026 VisionGaia Technology · Cologne, Germany

Commercial deployment without publishing connected server infrastructure code requires a commercial exception license. Enterprise deployments, TIER-0 audits (VGT SafetySys™) and commercial licenses: [visiongaiatechnology.de](https://visiongaiatechnology.de)

---

<div align="center">

**VISIONGAIATECHNOLOGY – WE ARCHITECT THE FUTURE OF SECURITY.**

[![VGT](https://img.shields.io/badge/VisionGaia-Technology-cyan?style=for-the-badge)](https://visiongaiatechnology.de)

*VGT AETHEL v0.6.0 — Sovereign AI Kernel // Go Pure Stdlib // Guard Kernel // Blockchain Audit Log // AES-256-GCM Vault // Permission Leases // Live Operator Viewport // Task Engine // Voice STT/TTS // Nexus Memory // Zero External Dependencies // Windows x64*

</div>
