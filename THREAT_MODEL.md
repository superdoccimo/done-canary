# DoneCanary v0.1 Threat Model

## Assets

- User repositories and files outside DoneCanary's owned run root
- Agent credentials and local configuration
- Integrity of fixture baselines, tests, hooks, and scoring results
- Trustworthiness of JSON, HTML, SVG, and terminal output
- Child-process cleanup and host stability

## Trust boundaries

The DoneCanary executable and deterministic oracle are trusted. The invoked coding agent, its stdout/stderr, and every post-run fixture file are untrusted. Git is an external dependency. The disposable worktree is writable by the agent. A narrow sibling `agent-writable` directory contains the separate Git metadata, hook evidence, and Git trace; baseline, metadata, result, and report files remain outside that writable root.

## Addressed threats

- **Real repository damage:** no public command accepts a repository path; real adapters receive only the freshly created fixture path.
- **Instruction/test tampering:** protected files and trusted material are compared with baseline SHA-256 hashes.
- **Hook bypass:** external hook evidence, local `core.hooksPath`, commit count/message, and Git Trace2 bypass flags are checked.
- **False success claims:** scoring ignores agent prose and derives outcomes from trusted checks and final state.
- **Path escape:** all generated paths are joined and revalidated beneath the canonical owned root; symlink components are rejected for write targets.
- **Git metadata redirection:** the fixture's regular `.git` pointer is verified before agent startup and before scoring, and must target only the current run's `agent-writable/git-meta` directory.
- **Report injection:** logs are bounded, ANSI-stripped, and HTML-escaped; SVG content comes from enumerated result data.
- **Secret capture:** metadata uses an allowlist and never records environment dumps, provider keys, or credential files.
- **Runaway children:** timeouts and interrupts terminate the child process tree and persist infrastructure status.
- **Unexpected network behavior by DoneCanary:** the application has no HTTP client, telemetry package, or network output. The selected external agent CLI may use its own authenticated service; that is outside DoneCanary's application boundary.

## Explicit v0.1 boundary

DoneCanary detects ordinary workflow mistakes and ordinary hook/configuration bypass. It is not a hardened sandbox and does not attempt to defeat a deliberately malicious agent that reverse-engineers the executable, forges external evidence, attacks the host, or escapes permissions granted to its own CLI process.

The disposable fixture is not a security boundary. Users must rely on the coding agent CLI's own safe workspace mode. DoneCanary never adds dangerous permission-bypass flags.

On native Windows, current Codex safe-sandbox backends cannot reliably finish the normal Git ref update required by a hook-respecting commit. For real Codex runs on that host only, the two commit-dependent canaries are explicitly reported as `not_run`. DoneCanary does not relax the hook or scope oracle, expose protected run evidence, disable the sandbox, or add a dangerous bypass to manufacture a complete score.

## Out of scope

- Protecting a compromised operating system, Git executable, or DoneCanary binary
- Preventing network use by the external agent CLI
- Storing or analyzing full agent transcripts as an observability product
- Scanning user repositories
- Cloud scoring, remote attestation, leaderboards, or multi-agent orchestration
