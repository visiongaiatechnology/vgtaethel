# Contributing to AETHEL

## Development contract

- Keep the Wails runtime modular; UI domains belong below `go-aethel/frontend/modules/`.
- Treat model output, feeds, paths, network responses and persisted state as untrusted input.
- Route host effects through typed skills and the policy engine. Do not bypass approval or
  evidence verification.
- Do not add placeholders, silent fallbacks or success messages without verified effects.
- Never commit secrets, certificates, runtime workspaces, personal data or generated binaries.

## Verification

```powershell
cd go-aethel
go test ./... -count=1
go vet -buildvcs=false ./...
govulncheck ./...
```

Run focused race, frontend syntax and UI smoke tests for the modules changed. Public release
changes must also update the compatibility/audit documentation and preserve version identity
tests.

## Pull requests

Describe the operator-visible outcome, security impact, failure behavior and verification
evidence. Keep unrelated changes separate. Screenshots must contain no personal data, keys,
paths, chats or audit content.

