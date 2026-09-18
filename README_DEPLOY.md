# AETHEL Beta V4 — Windows release protocol

The Wails application in `go-aethel/` is the authoritative product runtime. The Docker and
legacy crate trees are development components and do not replace the Windows release gate.

## Build prerequisites

- Windows 10/11 and WebView2
- Go 1.26.5+
- Wails CLI 2.15.0
- Native compiler/toolchain required by the voice bindings
- Inno Setup for the installer stage
- Authenticode certificate for public artifacts

## Verified source build

```powershell
cd go-aethel
go test ./... -count=1
go vet -buildvcs=false ./...
govulncheck ./...
.\scripts\build_aethel.bat
```

The build script intentionally preserves an existing `build/bin/vgt_workspace` directory.
Never publish that directory. Release packages must be assembled from a clean staging root.

## Public release gate

1. Run the live provider/orchestrator E2E matrix.
2. Build from the reviewed release-candidate commit in CI.
3. Sign and timestamp the executable and installer.
4. Verify signatures and SHA-256 checksums.
5. Test install, startup, offline behavior, update and uninstall on a clean Windows VM.
6. Publish the signed update manifest only after those checks pass.

Required tag: `aethel-v1.0.0-beta.4`.

No `.env`, key, certificate, workspace, chat, memory, audit, screenshot or diagnostic payload
may enter the source archive or release artifact.
