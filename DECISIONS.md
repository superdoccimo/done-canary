# DoneCanary v0.1 Decisions

These decisions are frozen unless executable evidence shows that a change is necessary to satisfy the acceptance criteria.

## D-001: Standard-library Go implementation

Use Go and the standard library only. This keeps the binary self-contained, makes the no-network boundary easy to audit, and avoids runtime package managers.

## D-002: DoneCanary owns every run path

The CLI derives a data root from `DONECANARY_DATA_DIR` or the user cache directory and creates unique `runs/<UTC timestamp>-<random id>` children. No public command accepts a repository argument.

## D-003: One executable, private helper mode

The built executable is copied into the fixture as `.canary/bin/canary-fixture[.exe]`. A private `__fixture` command implements `lint`, `test`, `build`, and `hook`. The hook stamp is written to an absolute, externally supplied path below the owned run directory.

## D-004: State-based oracle

Scoring consumes final Git/filesystem state plus process exit codes, external hook evidence, and Git Trace2. Agent stdout and stderr are report material only and never affect a canary status.

## D-005: Baseline hash set

The baseline records hashes for all initial fixture files. Integrity specifically covers `tests/`, `.canary/manifest.json`, `.canary/hooks/pre-commit`, and the copied helper. Protected-instruction scoring also covers task and instruction files.

## D-006: Process containment

On Windows, child processes are launched in a new process group and terminated with `taskkill /T /F` on timeout or interruption. On Unix, children use a process group and receive group termination. Logs use bounded writers.

## D-007: Real adapters are capability-checked

The Codex adapter uses `codex --config approval_policy="never" exec --ephemeral --sandbox workspace-write --color never --cd <fixture> -`, with the task prompt on stdin. The explicit safe sandbox remains required; confirmation suppression does not widen it. `--full-auto`, `danger-full-access`, and dangerous permission-bypass options are forbidden. The Claude adapter uses `claude --print --permission-mode acceptEdits --no-session-persistence --tools ... --allowedTools ... <prompt>`, subject to a local help capability check. Neither command receives an implementation-repository path.

## D-008: Reports are self-contained and inert

JSON is versioned. HTML has no remote assets or scripts and uses contextual escaping. SVG is generated from trusted result fields with XML escaping. ANSI control sequences are removed before log rendering.

## D-009: Human-gated public release

Source preparation, CI, packaging and pull-request review may be automated. Repository visibility changes, GitHub Release publication and the final public announcement remain human-approved actions.


## D-010: Public name is DoneCanary

The working name “Agent Canary” was retired before the first public release
after the final collision check found prior AI-agent projects using
AgentCanary. The public brand is `DoneCanary`; the repository, CLI, Go module,
environment variable, packages, reports and share assets use the new name.
There is no backward-compatibility promise for the unpublished working name.

## D-011: Canonical source and announcement

The canonical repository is `https://github.com/superdoccimo/done-canary`.
The canonical announcement is
`https://x.com/superdoccimo/status/2092930595206943129`.
