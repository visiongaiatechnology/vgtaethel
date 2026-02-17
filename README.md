🌌 VGT AETHEL :: Sovereign Intelligence Interface

<div align="center">

"Sovereign Intelligence Interface :: System Online"

VGT Aethel ist ein modulares, hochperformantes KI-Agenten-Framework, entwickelt für maximale Souveränität, Sicherheit und Erweiterbarkeit. Es bricht das Paradigma herkömmlicher Chatbots auf und agiert als autonome Entität mit direktem Systemzugriff, persistentem Gedächtnis und einer universellen Kommunikations-Bridge.

Features • Architektur • Installation • Sicherheit • Roadmap

</div>

🚧 Entwicklungsstatus

Dieses System befindet sich in der aktiven Phase II (Expansion) der Entwicklung.

Aktueller Fortschritt: 30% der Gesamt-Roadmap.

Status: Kern-Inferenz und Security-Layer sind stabil; Multi-Channel-Adapter sind im Alpha-Stadium.

👉 Detaillierter Projektfortschritt: ROADMAP.md

⚡ Features

🧠 VGT Core (The Brain)

Rust-Native Performance: Hochperformante Inferenz-Abstraktion basierend auf Axum und Tokio.

Groq-Beschleunigung: Unterstützung für LLaMA- und Mixtral-Modelle mit extrem niedriger Latenz.

Nexus Neural Memory: Lokales RAG-System (vgt-skills/rag.rs) für persistente Wissensspeicherung und semantischen Recall.

🛡️ Security Guard (The Shield)

Sicherheit ist integraler Bestandteil der Architektur. Jeder Tool-Aufruf wird in Echtzeit gescannt:

Heuristische Analyse: Erkennt Shell-Injections, Path-Traversals und riskante Systembefehle.

Human-in-the-Loop: Kritische Operationen (File-Write, Shell-Exec) erfordern eine explizite Nutzer-Autorisierung im Interface.

🌉 Polyglot Bridge (The Connector)

Ein universeller Kommunikations-Layer, der Aethel mit externen Netzwerken verbindet:

Messenger-Integration: WhatsApp, Telegram, Discord, Signal, Matrix und MS Teams.

Unified Messaging: Alle Kanäle werden auf ein standardisiertes NexusMessage-Protokoll gemappt.

🖥️ Supreme Interface (The Face)

Cyberpunk Terminal: Modernes Next.js Interface mit Tailwind CSS für maximale Übersicht.

Real-Time Streaming: Direkte Token-Anzeige via SSE (Server-Sent Events).

Visual Debugging: Interaktive Karten für Tool-Status und Sicherheits-Interventionen.

🏗️ Architektur

VGT Aethel nutzt eine strikte Monorepo-Struktur für maximale Wartbarkeit und klare Modulgrenzen:

VGT Aethel/

├── 📄 ROADMAP.md          # Strategische Planung & Fortschritt

├── 📁 crates/             # Rust & TypeScript Module

│   ├── 🦀 vgt-api/        # Gateway, WebSocket & State Management

│   ├── 🦀 vgt-core/       # Inferenz-Logic & Modell-Registry

│   ├── 🦀 vgt-skills/     # Tool-Registry (FS, Shell, RAG) & Guard

│   ├── 🌉 vgt-bridge/     # Messenger-Adapter (TypeScript)

│   └── ⚛️ vgt-ui/         # Dashboard & Control Face (Next.js)

├── 📁 infrastructures/    # Docker-Files & Deployment-Scripts

└── 📁 vgt_workspace/      # Persistent Mount Point (Memory & Config)


#🚀 Installation & Deployment

VGT Aethel ist für den Betrieb in isolierten Docker-Umgebungen optimiert.

Voraussetzungen

Docker & Docker Compose

Ein gültiger Groq API Key

Schnellstart (Genesis Sequence)

Repository klonen

git clone [https://github.com/username/vgt-aethel.git](https://github.com/username/vgt-aethel.git)
cd vgt-aethel


Konfiguration
Erstelle eine .env Datei im Root:

GROQ_API_KEY=gsk_your_key_here
//Optional: Messenger Tokens
TELEGRAM_TOKEN=...


Bootvorgang

docker-compose up --build -d


Uplink herstellen

Interface: http://localhost:3001

API Core: http://localhost:3000/health

🔒 Sicherheitskonzept

Das System verfolgt einen Security-First Ansatz:

Isolation: Container-Runtime nutzt standardmäßig Distroless-Images ohne Shell-Zugriff für das API-Gateway.

Sandboxing: Dateizugriffe sind hardwareseitig auf den vgt_workspace gemappt.

Guard-Protocol: Proaktive Blockierung destruktiver Befehle durch kompilierte Regex-Pattern.

🤝 Mitwirken

Wir begrüßen Beiträge, die den Standard für souveräne Intelligenz weiter vorantreiben. Bitte beachte die CONTRIBUTING.md (in Arbeit) bezüglich der Coding-Standards in Rust und TypeScript.

📜 Lizenz

Dieses Projekt ist unter der GNU Affero General Public License v3 (AGPLv3) lizenziert.

Wichtiger Hinweis: Wenn Sie dieses System modifizieren und über ein Netzwerk bereitstellen, sind Sie verpflichtet, den Quellcode Ihrer Version unter derselben Lizenz öffentlich zugänglich zu machen.

<div align="center">
<sub>VGT AETHEL // SYSTEM STATUS: ACTIVE (30%) // SOVEREIGN INTELLIGENCE PROTOCOL</sub>
</div>
