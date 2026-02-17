VGT AETHEL :: Sovereign Intelligence Interface

"Sovereign Intelligence Interface :: Online"

VGT Aethel ist ein modulares, hochperformantes KI-Agenten-Framework, entwickelt für maximale Souveränität, Sicherheit und Erweiterbarkeit. Anders als herkömmliche Chatbots agiert Aethel als autonome Entität mit direktem Zugriff auf System-Tools, einem persistenten Vektor-Gedächtnis (Nexus) und einer omnidirektionalen Kommunikations-Bridge.

Der Kern ("Cortex") ist in Rust geschrieben, um Latenz zu minimieren und Typsicherheit zu garantieren. Das Interface ("Face") nutzt Next.js für eine reaktive Echtzeit-Steuerung.

🚧 Entwicklungsstatus

Dieses Projekt befindet sich in aktiver Entwicklung. Derzeit sind ca. 30% der Roadmap erreicht (Phase I abgeschlossen). Es ist funktionsfähig, aber einige Schnittstellen (insbesondere in der Bridge) befinden sich noch im Alpha-Stadium.

👉 Hier geht es zur detaillierten ROADMAP

⚡ Kernfunktionen

🧠 VGT Core & API (The Brain)

Rust-Native Performance: Basierend auf Axum und Tokio für asynchrone Hochgeschwindigkeits-Verarbeitung.

Modell-Agnostisch: Integrierter Support für Groq-beschleunigte Modelle (LLaMA, Mixtral) via vgt-core.

Nexus Memory: Lokaler RAG-Vektor-Speicher (vgt-skills/rag.rs), der es dem Agenten erlaubt, Fakten zu lernen und sich an Kontext über Sessions hinweg zu erinnern.

🛡️ Security Guard System

Sicherheit ist kein Nachgedanke. Der integrierte Security Guard (vgt-skills/guard.rs) analysiert jeden Tool-Aufruf in Echtzeit:

Regex-basierte Bedrohungsanalyse: Erkennt Shell-Injection, Path-Traversal und destruktive Befehle (rm -rf, mkfs).

Risk Scoring: Klassifiziert Aktionen in Safe, Moderate und Critical.

Human-in-the-Loop: Kritische Systemeingriffe erfordern explizite Bestätigung durch den Benutzer im UI.

🌉 Polyglot Bridge (The Connector)

Ein zentraler TypeScript-Service verbindet Aethel mit der Außenwelt. Der Agent ist nicht im Browser gefangen.

Unterstützte Kanäle: WhatsApp, Telegram, Discord, Signal, Matrix, MS Teams.

Unified Protocol: Alle Nachrichten werden in ein internes NexusMessage-Format normalisiert.

🖥️ Supreme UI (The Face)

Cyberpunk/Terminal Ästhetik: Entwickelt mit Tailwind CSS, optimiert für Dark Mode und visuelle Klarheit.

Echtzeit-Streaming: Server-Sent Events (SSE) für sofortige Antwort-Generierung.

Tool-Visualisierung: Interaktive Karten für Tool-Approval und Status-Updates.

🏗️ Architektur

Das Projekt folgt einer strikten Monorepo-Struktur:
VGT Aethel/
├── ROADMAP.md        # Projektfortschritt & Planung
├── crates/
│   ├── vgt-api/      # HTTP Gateway, WebSocket & State Management (Rust)
│   ├── vgt-core/     # Inferenz-Engine, LLM-Connector & Nexus Logic (Rust)
│   ├── vgt-skills/   # Tool-Registry (FS, Shell, RAG) & Security Guard (Rust)
│   ├── vgt-bridge/   # Multi-Channel Messenger Adapter (TypeScript)
│   └── vgt-ui/       # Dashboard & Control Interface (Next.js)
├── infrastructures/  # Docker & Deployment Konfigurationen
└── vgt_workspace/    # Persistenter Speicher für Memory & Configs
🚀 Installation & DeploymentVGT Aethel ist für den Betrieb in containerisierten Umgebungen optimiert.VoraussetzungenDocker & Docker ComposeEin gültiger Groq API Key (für die Inferenz)SchnellstartRepository klonengit clone [https://github.com/username/vgt-aethel.git](https://github.com/username/vgt-aethel.git)
cd vgt-aethel
Umgebung konfigurierenErstellen Sie eine .env Datei im Hauptverzeichnis:# .env
GROQ_API_KEY=gsk_DeinGroqKeyHier...

# Optional: Bridge Credentials
TELEGRAM_TOKEN=...
DISCORD_TOKEN=...
System starten (Genesis)docker-compose up --build -d
ZugriffInterface: http://localhost:3001API Status: http://localhost:3000/health🛠️ EntwicklungBackend (Rust)Das Backend nutzt cargo workspace.# Tests ausführen
cargo test

# API lokal starten
cargo run --bin vgt-api
Frontend (Next.js)cd crates/vgt-ui
npm install
npm run dev
Bridge (Node.js)cd crates/vgt-bridge
npm install
npm run dev
🔒 Sicherheitskonzept

VGT Aethel gewährt einer KI Zugriff auf Systemebene. Um Risiken zu minimieren, gelten folgende Prinzipien:

Sandbox: In der Standard-Konfiguration operiert der Agent innerhalb eines Docker-Containers ohne Root-Rechte (Distroless Image).

File System Jail: Dateizugriffe sind auf ./vgt_workspace beschränkt.

Guard Intervention: Der Security Guard blockiert erkannte Angriffsvektoren proaktiv, bevor der Code ausgeführt wird.

🤝 Mitwirken

Beiträge sind willkommen. Bitte beachten Sie, dass dieses Projekt einen hohen Standard an Code-Qualität (Typsicherheit, Error Handling) pflegt.

📜 Lizenz

Veröffentlicht unter der GNU Affero General Public License v3 (AGPLv3).

Dies bedeutet kurzgefasst: Wenn Sie diesen Code (oder modifizierte Versionen davon) über ein Netzwerk (z.B. als Web-Service) verfügbar machen, müssen Sie den Quellcode Ihrer Version unter derselben Lizenz offenlegen.

<div align="center">
<sub>VGT AETHEL SYSTEM // STATUS: DEVELOPMENT (30%)</sub>
</div>
