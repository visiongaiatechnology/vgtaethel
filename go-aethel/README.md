# VGT AETHEL (`go-aethel`)

Local-first Personal Intelligence OS (Wails + Go + zero-CDN Global Watch).

## Quick start

```powershell
cd C:\Users\Masterboard\Downloads\vgtaethel-main\vgtaethel-main\go-aethel

# Verify compile after package split
& "C:\Program Files\Go\bin\go.exe" build -o NUL .

# Build desktop app
.\scripts\build_aethel.bat
# or: wails build
```

## Layout

See **[docs/STRUCTURE.md](docs/STRUCTURE.md)**.

Packages: `handlers`, `skills`, `intelligence`, `osint`, `agent`, `security`, `provider`, `personal`, `memory`, `voice`, `system` — wired from `app.go` via `InitState`.

## Global Watch Earth texture

| Location | Use |
|----------|-----|
| `frontend/assets/earth_day.jpg` | Preferred (also embedded on build) |
| `1.jpg` (project root) | Drop-in; copied/served automatically |

API: `GET /v1/assets/earth-texture`
