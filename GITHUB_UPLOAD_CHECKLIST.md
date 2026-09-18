# Beta V4 GitHub upload checklist

- [ ] Review every staged file with `git status` and `git diff --cached`.
- [ ] Confirm no `vgt_workspace`, `_logs`, `_archive`, screenshot, diagnostic or local database.
- [ ] Confirm no `.env`, API key, private key, certificate or signing material.
- [ ] Run `go test ./...`, `go vet ./...` and `govulncheck ./...` from `go-aethel`.
- [ ] Run secret scanning across the complete Git history after the first commit.
- [ ] Push a release-candidate branch and build only through the reviewed CI workflow.
- [ ] Complete live-provider and clean-VM tests for the Beta V4 strategic command build.
- [ ] Sign and timestamp the EXE and installer; verify signatures and checksums.
- [ ] Tag the verified commit as `aethel-v1.0.0-beta.4`.
- [ ] Publish a signed update manifest only after the tagged CI release passes.
