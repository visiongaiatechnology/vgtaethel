# VGT AETHEL Compatibility Matrix

| Bereich | Unterstützt | Hinweise |
|---|---|---|
| Desktop | Windows 10/11 x64 | Primäres Release-Ziel, Wails/WebView2 erforderlich. |
| Lokaler Speicher | Windows DPAPI | Schlüsselmaterial ist an den angemeldeten Windows-Nutzer gebunden. |
| GUI-Steuerung | Windows 10/11 x64 | Jede riskante Aktion benötigt die Policy-Freigabe; GUI-Runs erfassen visuelle Vor-/Nachher-Evidenz. |
| Browser-/Rechercheprofile | Windows 10/11 x64 | Browserzugriff und sichtbare Steuerung folgen dem Capability-Profil. |
| OpenAI | Registrierte Modelle | Tool- und Vision-Fähigkeiten werden durch die lokale Provider-Registry erzwungen. |
| Anthropic, Gemini, DeepSeek, Groq | Registrierte Modelle | Adapter und Health Checks sind verfügbar; Provider-spezifische API-Keys sind erforderlich. |
| Ollama | Lokale Modelle | Chat ist verfügbar; Tool-Runs bleiben bis zur expliziten Verifikation der Toolfähigkeit gesperrt. |
| macOS/Linux | Entwicklungsstand | Kein offizielles Release-Artefakt; der Nicht-Windows-Schlüsselspeicher ist kein Release-Ziel. |

## Mindestanforderungen

- Windows 10 22H2 oder Windows 11, 64 Bit
- Microsoft Edge WebView2 Runtime
- 8 GB RAM (16 GB empfohlen für lokale Modelle)
- Netzwerkzugang nur für aktiv konfigurierte Cloud-Provider
