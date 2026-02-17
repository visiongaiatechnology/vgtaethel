<div align="center">

🌌 VGT AETHEL :: Sovereign Intelligence Interface

VGT Aethel ist ein modulares, hochperformantes KI-Agenten-Framework für maximale Souveränität und Sicherheit. Es agiert als autonome Entität mit direktem Systemzugriff, persistentem Gedächtnis und einer universellen Kommunikations-Bridge.

Features • Architektur • Installation • Sicherheit • Roadmap

</div>

🚧 Projektstatus: Aktiv (Phase II)

Das System befindet sich in der aktiven Expansion. Der Fokus liegt auf der Stabilisierung des Rust-Kerns und der Skalierung der Messenger-Adapter.
👉 Aktueller Fortschritt: ROADMAP.md (30% erreicht)

⚡ Features

🧠 VGT Core (The Brain)

Rust-Native Performance: Inferenz-Abstraktion via Axum/Tokio für Zero-Overhead.

Groq-Beschleunigung: Native LPU-Integration für Inferenz-Latenzen im Millisekundenbereich.

Nexus Neural Memory: Vektorbasiertes RAG-System (vgt-skills/rag.rs) für semantischen Recall.

🛡️ Security Guard (The Shield)

Heuristische Analyse: Real-time Scan auf Shell-Injections und Path-Traversals.

Human-in-the-Loop: Explizite Autorisierung kritischer Systembefehle über das Interface.

🌉 Polyglot Bridge (The Connector)

Unified Messaging: Standardisiertes NexusMessage-Protokoll für WhatsApp, Telegram, Discord und MS Teams.

Async Adapter: Hochkonkurrente Verarbeitung eingehender Events via TypeScript-Bridge.

🖥️ Supreme Interface (The Face)

Cyberpunk Terminal: Hochmodernes Next.js Dashboard mit Tailwind CSS und Framer Motion.

Real-Time Streaming: Token-by-Token Visualisierung via Server-Sent Events (SSE).

🏗️ Architektur

VGT Aethel nutzt eine strikte Monorepo-Struktur zur Gewährleistung der Systemintegrität:

VGT Aethel/
├── crates/
│   ├── vgt-api/       # Axum API Gateway & Auth (Rust)
│   ├── vgt-core/      # LLM-Orchestrierung & Logic (Rust)
│   ├── vgt-skills/    # RAG & Security Protocols (Rust)
│   ├── vgt-bridge/    # Messenger-Adapter (TypeScript)
│   └── vgt-ui/        # High-End Dashboard (Next.js)
├── infrastructures/   # Docker-Spezifikationen & K8s Manifeste
└── vgt_workspace/     # Persistente DB & Config-State


🚀 Installation & Deployment

Voraussetzungen

Docker & Docker Compose (v2.20+)

Groq API Key (LPU Inferenz)

Genesis Sequence

Repository klonen:

git clone [https://github.com/vgt-tech/vgt-aethel.git](https://github.com/vgt-tech/vgt-aethel.git)
cd vgt-aethel


Umgebung konfigurieren:
Erstelle eine .env Datei:

GROQ_API_KEY=gsk_...
TELEGRAM_TOKEN=...
DATABASE_URL=postgres://vgt:vgt@vgt-db:5432/aethel


System-Boot:

docker-compose up --build -d


Uplinks:

Interface: http://localhost:3001

API Core: http://localhost:3000/health

🔒 Sicherheitskonzept

Distroless Runtime: API-Container ohne Shell zur Minimierung der Angriffsfläche.

Strict Scoping: Dateizugriffe sind hardwareseitig auf den vgt_workspace limitiert.

Regex Guard: Proaktives Blocking destruktiver Befehle im Pre-Inference-Stadium.

📜 Lizenz

Lizenziert unter der GNU Affero General Public License v3 (AGPLv3).

<div align="center">
<sub>VGT AETHEL // SYSTEM STATUS: ACTIVE (30%) // SOVEREIGN INTELLIGENCE PROTOCOL</sub>
</div>
