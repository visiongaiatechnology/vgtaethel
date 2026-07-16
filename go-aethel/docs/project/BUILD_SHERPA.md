# AETHEL Local Voice Mode: Sherpa-ONNX

## Übersicht

AETHEL verwendet **Sherpa-ONNX** als primären lokalen Offline-TTS-Provider.
Keine Cloud-API erforderlich. Keine Netzwerkabhängigkeit zur Laufzeit.

## Voraussetzungen

### 1. CGO aktivieren (Windows)

```cmd
set CGO_ENABLED=1
```

### 2. Sherpa-ONNX Go Bindings

```cmd
cd go-aethel
set CGO_ENABLED=1
go get github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx
go mod tidy
```

Import im Code:
```go
import sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
```

### 3. Sherpa-ONNX Windows-DLLs

Die DLLs müssen aus dem offiziellen sherpa-onnx-go-windows Paket kopiert werden:

**Quelle:** https://github.com/k2-fsa/sherpa-onnx-go-windows/tree/master/lib/x86_64-pc-windows-gnu

**Ziel:** Direkt neben `aethel.exe` ablegen.

**Wichtig: ALLE *.dll aus diesem Ordner kopieren**, nicht nur `sherpa-onnx-c-api.dll` und `onnxruntime.dll`.

Typische DLL-Liste:
- `sherpa-onnx-c-api.dll`
- `sherpa-onnx-core-c-api.dll`
- `onnxruntime.dll`
- `piper_phonemize_c_api.dll`
- `espeak-ng_c_api.dll`
- `kaldi-native-fbank-core.dll`
- ... und weitere

### 4. Modelle herunterladen

Modelle müssen in folgender Struktur abgelegt werden:

```
./vgt_workspace/models/sherpa/
├── kitten-nano-en-v0_1-fp16/
│   ├── model.fp16.onnx      ← NICHT model.onnx!
│   ├── voices.bin
│   ├── tokens.txt
│   └── espeak-ng-data/      ← Ordner mit Phonem-Daten
├── kokoro-multi-lang-v1_0/
│   ├── model.onnx
│   ├── voices.bin
│   ├── tokens.txt
│   └── espeak-ng-data/
│   ├── lexicon-us-en.txt    ← optional, wird erkannt
│   ├── lexicon-gb-en.txt
│   ├── lexicon-zh.txt
│   ├── number-zh.fst
│   ├── phone-zh.fst
│   └── date-zh.fst
└── README.md
```

#### KittenTTS (Englisch, ~30 MB)

| Datei | Bedeutung |
|-------|-----------|
| `model.fp16.onnx` | ONNX-Modell (fp16 quantisiert) |
| `voices.bin` | Sprecher-Definitionen |
| `tokens.txt` | Tokenizer |
| `espeak-ng-data/` | Phonem-Daten |

**Download:**
```
https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kitten-nano-en-v0_1-fp16.tar.bz2
```

#### Kokoro (Multilingual, ~80 MB)

| Datei | Bedeutung |
|-------|-----------|
| `model.onnx` | ONNX-Modell |
| `voices.bin` | Sprecher-Definitionen |
| `tokens.txt` | Tokenizer |
| `espeak-ng-data/` | Phonem-Daten |
| `lexicon-*.txt` | Sprach-Lexika (optional) |
| `*.fst` | Chinesische Finite-State-Transducer (optional) |

**Download:**
```
https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/kokoro-multi-lang-v1_0.tar.bz2
```

## Verzeichnisstruktur nach Setup

```
go-aethel/
├── aethel.exe
├── sherpa-onnx-c-api.dll
├── sherpa-onnx-core-c-api.dll
├── onnxruntime.dll
├── piper_phonemize_c_api.dll
├── espeak-ng_c_api.dll
├── ... (alle DLLs)
├── vgt_workspace/
│   ├── models/sherpa/
│   │   ├── kitten-nano-en-v0_1-fp16/
│   │   │   ├── model.fp16.onnx
│   │   │   ├── voices.bin
│   │   │   ├── tokens.txt
│   │   │   └── espeak-ng-data/
│   │   └── kokoro-multi-lang-v1_0/
│   │       ├── model.onnx
│   │       ├── voices.bin
│   │       ├── tokens.txt
│   │       └── espeak-ng-data/
│   └── audio/              ← TTS-Cache
```

## Build

```cmd
cd go-aethel
set CGO_ENABLED=1
go mod tidy
go build -o aethel.exe .
```

## Health-Check

Der Health-Check prüft pro Voice dynamisch die benötigten Dateien:
- **Kitten:** `model.fp16.onnx`, `voices.bin`, `tokens.txt`, `espeak-ng-data/`
- **Kokoro:** `model.onnx`, `voices.bin`, `tokens.txt`, `espeak-ng-data/`
- **Andere:** `model.onnx`, `voices.bin`, `tokens.txt`, `espeak-ng-data/`

Fehlermeldung bei unvollständigem Modell:
```
SHERPA_MODEL_MISSING: Fehlende Dateien: model.fp16.onnx, tokens.txt in
./vgt_workspace/models/sherpa/kitten-nano-en-v0_1-fp16/
```

## API-Endpunkte

### GET /v1/audio/health
```json
{
  "sherpa_local": {
    "provider": "sherpa_local",
    "offline": true,
    "configured": true,
    "voices": 1,
    "ready": 1,
    "warnings": []
  }
}
```

### GET /v1/audio/voices
```json
[{
  "id": "kitten-nano-en-v0_1-fp16",
  "name": "kitten-nano-en-v0_1-fp16 (Sherpa EN)",
  "type": "sherpa",
  "available": true,
  "offline": true,
  "language": "en"
}]
```

### POST /v1/audio/speech
```json
{"text": "Hello world.", "voice": "kitten-nano-en-v0_1-fp16"}
```
→ `audio/wav`

### POST /v1/audio/speech?format=json
→ Base64-JSON:
```json
{
  "provider": "sherpa_local",
  "offline": true,
  "format": "wav",
  "audio_base64": "UklGRiQAAABXQVZF...",
  "voice": "kitten-nano-en-v0_1-fp16",
  "size": 45238,
  "duration": "2.5s"
}
```

## Keine automatischen Downloads

AETHEL lädt keine Modelle zur Laufzeit herunter.
Modelle müssen manuell in `vgt_workspace/models/sherpa/` abgelegt werden.
