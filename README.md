# VGT AETHEL :: SOVEREIGN INTELLIGENCE FRAMEWORK

> **"AETHEL is the sovereign AI agent runtime that OpenClaw should have been. Built in Rust. Governed by design."**
> 
> *— VisionGaia Technology*

ALPHA 1.0
---

<div align="center">
<img width="50%" alt="VGT AETHEL Neural Interface" src="https://github.com/user-attachments/assets/509b3a60-ea7f-44a8-8c11-bc61bbbcc188" />
</div>

---
## ⚠️ DISCLAIMER: EXPERIMENTAL R&D PROJECT

This project is a **Proof of Concept (PoC)** and part of ongoing research and development at
VisionGaia Technology. It is **not** a certified or production-ready product.

**Use at your own risk.** The software may contain security vulnerabilities, bugs, or
unexpected behavior. It may break your environment if misconfigured or used improperly.

**Do not deploy in critical production environments** unless you have thoroughly audited
the code and understand the implications. For enterprise-grade, verified protection,
we recommend established and officially certified solutions.

Found a vulnerability or have an improvement? **Open an issue or contact us.**



## 💙 SUPPORT THE MISSION

AETHEL is free & open source. If you'd like to support the project:

[![Donate](https://img.shields.io/badge/DONATE-PayPal-blue?style=for-the-badge&logo=paypal)](https://paypal.me/dergoldenelotus)

*Every contribution flows directly into the development of AETHEL and the VGT ecosystem.*

VGT Aethel is not a conventional chat interface.

It is an asymmetric, military-grade AI architecture designed for absolute data sovereignty and deterministic execution in high-security environments.

While other AI agents run in unsecured Python environments with full system privileges, Aethel implements a **strict separation of intelligence, execution and communication** – the governance layer that everyone else is missing.

Developed by **[VisionGaia Technology](https://visiongaiatechnology.de)**.

---

## 🏛 ARCHITECTURE PARADIGM (THE VGT STANDARD)

```
┌─────────────────────────────────────────────────────┐
│                   OPERATOR (UI)                     │
│              Red Card Lockdown System               │
├─────────────────┬───────────────────────────────────┤
│   THE CORTEX    │         THE BRIDGE                │
│   (Rust Core)   │      (TypeScript Daemon)          │
│                 │                                   │
│  vgt-api        │  WhatsApp · Telegram · Discord    │
│  vgt-core       │  Signal · Matrix · iMessage       │
│  vgt-skills     │                                   │
├─────────────────┴───────────────────────────────────┤
│              SECURITY GUARD (guard.rs)              │
│    Path Traversal · Shell Injection · rm -rf        │
│         BLOCKED BEFORE KERNEL EXECUTION             │
└─────────────────────────────────────────────────────┘
```

---

## 1. THE CORTEX (Rust Core)

The core of Aethel (`vgt-core`, `vgt-api`, `vgt-skills`) is written entirely in **Rust (Edition 2024)**.

**Memory Safety & Performance**  
Compiled with maximum optimization (`lto = "fat"`, `panic = "abort"`). Zero-Panic Policy. No runtime overhead.

**Neural RAG (Nexus)**  
In-process vector database using `fastembed` (local embeddings, AllMiniLML6V2) for long-term memory that never leaves the host. Persisted in `./vgt_workspace` – fully air-gap capable.

**The Security Guard (`guard.rs`)**  
A native regex-based firewall in the skill execution layer. Blocks **proactively** before LLM output reaches the kernel:

```
✗  Path Traversal    →  ../  ../../
✗  Shell Injections  →  && || ; | `
✗  Destructive       →  rm -rf  dd  mkfs
✗  Privilege Esc.    →  sudo  chmod 777
```

**Model Tiers**
| Tier | Model | Use Case |
|------|-------|----------|
| Diamond | OSS 120B | Complex Reasoning |
| Platinum | 20B Safeguard | Standard Operations |
| Local | Custom | Air-Gap Deployment |

---

## 2. THE INTERFACE (Next.js 15)

A forensic, reactive command center in the **VGT Supreme Glass Design**.

**Zero-Trust Execution**  
Every attempt by the AI to invoke system tools (filesystem writes, shell commands) is intercepted and requires explicit authorization by the operator.

**Red Card Lockdown System**  
When `guard.rs` triggers → UI immediately activates lockdown status. No tool call passes without operator visibility.

**Genesis Setup**  
First launch: `SetupWizard` dynamically injects API credentials via `/v1/setup` – no hardcoded keys in the repository.

---

## 3. THE NEXUS BRIDGE (TypeScript)

A polyglot daemon acting as translator between external networks and the Rust Cortex.

**Current Providers:**

| Provider | Protocol | Status |
|----------|----------|--------|
| Telegram | grammY | ✅ Active |
| WhatsApp | Baileys / QR | ✅ Active |
| Discord | discord.js | ✅ Active |
| Signal | signal-cli JSON-RPC | ✅ Active |
| Matrix | matrix-js-sdk | ✅ Active |
| iMessage | BlueBubbles | 🔄 Mockup |

---

## ⚙️ DEPLOYMENT (DISTROLESS)

Aethel is designed for **Zero-Day resilience**. The Rust Cortex runs in a **Distroless Container** – no shell, no package manager, no post-exploitation tools.

```bash
# 1. Initialize configuration
cp .env.example .env
# GROQ_API_KEY=gsk_your_key

# 2. Launch VGT Neural Sequence
docker-compose up --build -d
```

| Endpoint | URL |
|----------|-----|
| Interface | `http://localhost:3001` |
| API Gateway | `http://localhost:3000` |
| Health Check | `http://localhost:3000/health` |

**Persistence:** Nexus memory is automatically mounted in `./vgt_workspace`.

---

## 🏗 SYSTEM STRUCTURE

```
vgt-aethel/
├── crates/
│   ├── vgt-api/        # Axum Gateway & SSE Streaming (Rust)
│   ├── vgt-core/       # Neural Inference Engine & Model Registry (Rust)
│   ├── vgt-skills/     # FS, Shell, RAG & Security Guard (Rust)
│   ├── vgt-bridge/     # Polyglot Messenger Adapters (TypeScript)
│   └── vgt-ui/         # Cyberpunk Dashboard Interface (Next.js)
├── infrastructures/    # Docker Hardening & Build Specs
└── vgt_workspace/      # Persistent Nexus Storage (Sled DB)
```

---

## 🔒 LICENSE & USAGE

This project is licensed under the **[GNU AGPL-3.0 License](LICENSE)**.

Deployment in cloud environments (SaaS) requires full disclosure of all connected server infrastructure code. Anyone using Aethel commercially without publishing their own code requires a commercial exception license.

**Enterprise & Commercial:**  
For enterprise deployments, TIER-0 audits (VGT SafetySys™) and commercial exception licenses:  
→ **[VisionGaia Technology](https://visiongaiatechnology.de)**

---

<div align="center">

**CODE IS LAW. WE DEFINE THE STANDARD.**

*VGT AETHEL // SOVEREIGN INTELLIGENCE PROTOCOL // NO MERCY IN CODE*

[![VisionGaia Technology](https://img.shields.io/badge/BUILT%20BY-VisionGaia%20Technology-cyan?style=for-the-badge)](https://visiongaiatechnology.de)

</div>
