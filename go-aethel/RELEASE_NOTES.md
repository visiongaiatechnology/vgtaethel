# VGT AETHEL 1.0.0-beta.1 (BETA V1)

## Release-Schwerpunkte

- Persistente Agent-Runs mit Run-ID, Trace, Pause, Resume, Abbruch, Kostenlimit und signierter Einmalfreigabe.
- Agentenprofile mit minimalen Capability-Grenzen: Researcher, Developer, Browser Operator und Personal Assistant.
- Verschlüsselte lokale Journale, Erinnerungen und persönliche Profile; Windows verwendet DPAPI für Schlüsselmaterial.
- Provider-Registry mit Modellfähigkeiten, Kontextgrenzen, Tool-Validierung, Health Checks und Fallback-Routing.
- Run Center mit Status, Kosten, Toolanzahl, Trace und visueller Evidenz für GUI-Aktionen.
- Eigenständiger Zero-Trust-Freigabedialog für riskante Schritte.
- Datensparsames Diagnosepaket ohne Prompts, Erinnerungen, Toolargumente, Pfade, Secrets oder Log-Inhalte.

## Bekannte Beta-Grenzen

- Updates werden erst installiert, wenn ein signiertes Release-Manifest und ein öffentlicher Release-Schlüssel hinterlegt sind.
- Die Anwendung wird lokal gebaut; eine kommerzielle Verteilung benötigt ein Code-Signing-Zertifikat und die hinterlegten CI-Secrets.
- Nach einem App-Neustart werden in-flight Runs absichtlich pausiert. Der Run Center zeigt den Zustand und erlaubt eine explizite Wiederaufnahme.
- Externe Provider bleiben von Netzwerk, API-Key, ModellverfÃ¼gbarkeit und deren individuellen Limits abhÃ¤ngig. Provider-Health und Fehlermeldungen werden sichtbar angezeigt.
- AuÃŸerhalb des geschÃ¼tzten Workspace werden Ordner nur zeitlich begrenzt und explizit lesend oder schreibend eingehÃ¤ngt; ein Jail-VerstoÃŸ wird nicht umgangen.

## Verifikation

`go test ./...`, `go test -race ./...` und der Wails-Windows-Build sind vor Veröffentlichung auszuführen. Die GitHub-Workflow-Datei erzeugt einen Installer, SHA-256-Prüfsummen und einen Draft Release.
