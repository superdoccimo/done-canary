# Contributing to DoneCanary

Thank you for testing or improving DoneCanary.

## Before opening a change

- Keep the v0.1 boundary: disposable fixture only, no user-repository input.
- Do not add an LLM judge, telemetry, cloud scoring or dangerous permission bypass.
- Preserve PASS, FAIL, NOT RUN and infrastructure-error distinctions.
- Add executable regression coverage for behavior changes.

## Local checks

```console
go test ./...
go vet ./...
go test -race ./...
go build ./cmd/done-canary
```

Use `done-canary selftest` after building the binary.

## Security and privacy

Do not submit credentials, provider tokens, private repositories, private logs or
personal filesystem paths. See [SECURITY.md](SECURITY.md).
