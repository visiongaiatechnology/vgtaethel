VGT AETHEL :: DEPLOYMENT PROTOCOL

STATUS: PHASE V (DEPLOYMENT)
TARGET: DOCKERIZED ENVIRONMENT

1. VORAUSSETZUNGEN

Docker & Docker Compose installiert.

Ein gültiger GROQ_API_KEY in der .env Datei im Root-Verzeichnis.

2. GENESIS (Start System)

Erstelle eine .env Datei im Hauptverzeichnis:

GROQ_API_KEY=gsk_...dein_key_hier...


Starte die Sequenz:

docker-compose up --build -d


3. ZUGRIFFSPUNKTE

INTERFACE (UI): http://localhost:3001

CORTEX (API): http://localhost:3000

4. ARCHITEKTUR-NOTIZEN

Security: Der API-Container nutzt distroless. Es gibt keine Shell im Container. docker exec wird fehlschlagen. Das ist ein Feature, kein Bug.

Persistenz: Das Langzeitgedächtnis (Nexus) wird in ./vgt_workspace auf dem Host gemountet. Daten überleben Container-Neustarts.

Networking: Das UI kommuniziert über das interne Docker-Netzwerk vgt_neural_net.