# DoneCanary v0.1.0-rc.3 release-candidate evidence

Prepared: 2026-08-27 (Asia/Tokyo)

## Purpose

This document records the verified public-release-candidate boundary after the
project working name was changed to DoneCanary. It replaces machine-specific
internal implementation notes in the public tree.

## Identity

- Brand: `DoneCanary`
- Repository: `https://github.com/superdoccimo/done-canary`
- CLI: `done-canary`
- Go module: `github.com/superdoccimo/done-canary`
- Data-root variable: `DONECANARY_DATA_DIR`
- Release candidate: `0.1.0-rc.3`
- Canonical announcement: `https://x.com/superdoccimo/status/2092930595206943129`

## Verified public-candidate run

The release-candidate source was validated by GitHub Actions run
`33074490708` at commit:

```text
db98cb7e9c3822813563d307d9e4975797e1f09b
```

Every required job in that run completed successfully:

- public-tree/history/release-metadata audit: PASS;
- `go test ./...`: PASS on Windows, Ubuntu and macOS;
- `go vet ./...`: PASS on Windows, Ubuntu and macOS;
- `go build ./cmd/done-canary`: PASS on Windows, Ubuntu and macOS;
- `go test -race ./...`: PASS on Ubuntu;
- package-version guards: PASS on all three packaging hosts;
- native-format Windows package: PASS;
- native-format Linux package: PASS;
- native-format macOS package: PASS;
- final bundle verification: PASS.

The verified Actions bundle artifact was `done-canary-v0.1.0-rc.3`, artifact ID
`9647307704`, with GitHub-recorded artifact digest:

```text
sha256:c6ae6dee3530652127d4a3f8248e431d5faa0f142fa3f3e7aeedaf3502f12cc8
```

## Release assets

The prerelease is published at:

`https://github.com/superdoccimo/done-canary/releases/tag/v0.1.0-rc.3`

It contains exactly the three platform archives plus the checksum file.

Archive SHA-256 values:

```text
1a239b1ba81e212aa789ce97aa5b0ef57d369e5bdb7879ad3509b3aeba88f652  done-canary-v0.1.0-rc.3-windows-amd64.zip
15719645f71569f360c441acee3d6efa3386400118abe120322ebecc93c9fe84  done-canary-v0.1.0-rc.3-linux-amd64.tar.gz
eb43af3c78b79cc0e74b7f58d8392b817641824f4da05734b3e5d014b2f95b1f  done-canary-v0.1.0-rc.3-darwin-amd64.tar.gz
```

Windows uses ZIP. Linux and macOS use `tar.gz` so the `done-canary` executable
mode is preserved. The bundle verifier confirmed exact four-file package
contents, executable mode on Linux/macOS, embedded DoneCanary module paths,
clean VCS metadata and matching SHA-256 checksums.

The RC binaries are not code-signed or notarized. That limitation is stated in
the README and release notes; users whose policy requires signed artifacts
should build from source or apply their own trusted signing process.

## Oracle and safety boundary

The rename and public-preparation work did not weaken the oracle, hook checks,
scope checks, sandbox requirements, path validation, bounded-log behavior,
HTML/SVG escaping or the no-telemetry boundary.

Previously verified behavior remains part of the evidence record:

- deterministic fake-pass reaches 7/7;
- skipped pre-commit hook is detected as a canary failure;
- agent prose is not used as scoring proof;
- user repositories are not accepted as test targets;
- native Windows Codex capability-limited runs report two commit-dependent
  canaries as `NOT RUN` instead of disabling the safe sandbox to manufacture a
  seven-item score.

## Release gates

- AT-036: PASS — GitHub Actions run `33074490708` is green across the required
  Windows, Ubuntu, macOS, race, packaging, audit and bundle jobs.
- AT-037: NOT RUN — requires five independent external reproductions.
- AT-038: NOT RUN — requires three external scorecard-comprehension tests.
- AT-039: PASS — practical final-name check is recorded in `NAME_CHECK.md`.
- AT-040: NOT RUN — repository visibility and the final public announcement
  remain human-controlled external publication decisions.
