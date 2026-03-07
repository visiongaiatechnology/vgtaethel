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

## 💙 SUPPORT THE MISSION

AETHEL ist free & open source. Wenn du das Projekt unterstützen möchtest:

[![Donate](https://img.shields.io/badge/DONATE-PayPal-blue?style=for-the-badge&logo=paypal)](https://paypal.me/dergoldenelotus)

*Jede Unterstützung fließt direkt in die Weiterentwicklung von AETHEL und dem VGT Ökosystem.*

VGT Aethel ist keine herkömmliche Chat-Schnittstelle.

Es ist eine asymmetrische, militärisch gehärtete KI-Architektur, konzipiert für absolute Datenhoheit (**Sovereignty**) und deterministische Ausführung in Hochsicherheitsumgebungen.

Während andere AI-Agents in ungesicherten Python-Umgebungen mit vollen Systemrechten laufen, implementiert Aethel eine **strikte Trennung von Intelligenz, Ausführung und Kommunikation** – die Governance-Schicht die allen anderen fehlt.

Entwickelt von **[VisionGaia Technology](https://visiongaiatechnology.de)**.

---

## 🏛 ARCHITEKTUR-PARADIGMA (THE VGT STANDARD)

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

Der Kern von Aethel (`vgt-core`, `vgt-api`, `vgt-skills`) ist vollständig in **Rust (Edition 2024)** geschrieben.

**Memory Safety & Performance**  
Kompiliert mit maximaler Optimierung (`lto = "fat"`, `panic = "abort"`). Zero-Panic Policy. Kein Runtime-Overhead.

**Neural RAG (Nexus)**  
In-Process Vektor-Datenbank mittels `fastembed` (lokale Embeddings, AllMiniLML6V2) für Langzeitgedächtnis das niemals den Host verlässt. Persistiert in `./vgt_workspace` – vollständig air-gap-fähig.

**The Security Guard (`guard.rs`)**  
Eine native Regex-basierte Firewall in der Skill-Execution-Schicht. Blockiert **proaktiv** bevor die LLM-Ausgabe den Kernel erreicht:

```
✗  Path Traversal    →  ../  ../../
✗  Shell Injections  →  && || ; | `
✗  Destruktiv        →  rm -rf  dd  mkfs
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

Eine forensische, reaktive Kommandozentrale im **VGT Supreme Glass-Design**.

**Zero-Trust Execution**  
Jeder Versuch der KI, System-Tools aufzurufen (Dateisystem-Schreibzugriffe, Shell-Kommandos), wird abgefangen und erfordert explizite Autorisierung durch den Operator.

**Red Card Lockdown System**  
Wenn `guard.rs` anschlägt → UI triggert sofortigen Lockdown-Status. Kein Tool-Call passiert ohne Operator-Sichtbarkeit.

**Genesis Setup**  
Erster Start: `SetupWizard` injiziert API-Credentials dynamisch über `/v1/setup` – keine Hardcoded Keys im Repository.

---

## 3. THE NEXUS BRIDGE (TypeScript)

Ein polyglotter Daemon als Übersetzer zwischen externen Netzwerken und dem Rust-Cortex.

**Aktuelle Provider:**

| Provider | Protokoll | Status |
|----------|-----------|--------|
| Telegram | grammY | ✅ Active |
| WhatsApp | Baileys / QR | ✅ Active |
| Discord | discord.js | ✅ Active |
| Signal | signal-cli JSON-RPC | ✅ Active |
| Matrix | matrix-js-sdk | ✅ Active |
| iMessage | BlueBubbles | 🔄 Mockup |

---

## ⚙️ DEPLOYMENT (DISTROLESS)

Aethel ist für **Zero-Day-Resilienz** konzipiert. Der Rust-Cortex läuft in einem **Distroless Container** – keine Shell, kein Paketmanager, keine Post-Exploitation-Tools.

```bash
# 1. Konfiguration initialisieren
cp .env.example .env
# GROQ_API_KEY=gsk_your_key

# 2. VGT Neural Sequenz starten
docker-compose up --build -d
```

| Endpoint | URL |
|----------|-----|
| Interface | `http://localhost:3001` |
| API Gateway | `http://localhost:3000` |
| Health Check | `http://localhost:3000/health` |

**Persistenz:** Nexus-Gedächtnis wird automatisch in `./vgt_workspace` gemountet.

---

## 🏗 SYSTEM STRUCTURE

```
vgt-aethel/
├── crates/
│   ├── vgt-api/        # Axum Gateway & SSE Streaming (Rust)
│   ├── vgt-core/       # Neural Inferenz-Engine & Model-Registry (Rust)
│   ├── vgt-skills/     # FS, Shell, RAG & Security Guard (Rust)
│   ├── vgt-bridge/     # Polyglot Messenger Adapters (TypeScript)
│   └── vgt-ui/         # Cyberpunk Dashboard Interface (Next.js)
├── infrastructures/    # Docker Hardening & Build Specs
└── vgt_workspace/      # Persistenter Nexus-Speicher (Sled DB)
```

---

## 🔒 LIZENZ & NUTZUNG

Dieses Projekt steht unter der **[GNU AGPL-3.0 Lizenz](LICENSE)**.

Der Einsatz in Cloud-Umgebungen (SaaS) erfordert die vollständige Offenlegung der verbundenen Server-Infrastruktur-Codes. Wer Aethel kommerziell nutzt ohne den eigenen Code zu veröffentlichen, braucht eine kommerzielle Ausnahmelizenz.

**Enterprise & Kommerziell:**  
Für Enterprise-Deployments, TIER-0 Auditierungen (VGT SafetySys™) und kommerzielle Ausnahmelizenzen:  
→ **[VisionGaia Technology](https://visiongaiatechnology.de)**

---

<div align="center">

**CODE IS LAW. WE DEFINE THE STANDARD.**

*VGT AETHEL // SOVEREIGN INTELLIGENCE PROTOCOL // NO MERCY IN CODE*

[![VisionGaia Technology](https://img.shields.io/badge/BUILT%20BY-VisionGaia%20Technology-cyan?style=for-the-badge)](https://visiongaiatechnology.de)

</div>
