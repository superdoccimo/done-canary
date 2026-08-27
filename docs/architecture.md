# Architecture

DoneCanary is a single Go executable with a deliberately small internal architecture.

```text
CLI
 ├─ run-path ownership and boundary checks
 ├─ disposable fixture generator
 ├─ compiled-in adapter (Codex, Claude, or deterministic fake)
 ├─ bounded process runner and cancellation
 ├─ deterministic external oracle
 └─ terminal, JSON, HTML, and SVG reporters
```

## Run lifecycle

1. Resolve a dedicated data root and reject Git-worktree or path-escape targets.
2. Create a unique owned run directory.
3. Generate and baseline-commit the fixture worktree with its real Git metadata in the run's narrow `agent-writable/git-meta` directory.
4. Copy the current executable into `.canary/bin` for trusted fixture commands.
5. Start the selected agent only in `fixture-repo`. Codex receives only `agent-writable` as an additional writable root; Git Trace2 and hook evidence also live there.
6. Bound stdout/stderr and terminate the process tree on timeout or interruption.
7. Inspect final Git/filesystem state using trusted in-process checks.
8. Write versioned JSON and self-contained HTML/SVG outputs.

The agent process cannot select another repository through the DoneCanary CLI. Real adapter command construction revalidates that the working directory is the current run's `fixture-repo` child.

`fixture-repo/.git` is a regular Git-generated pointer file. Before agent startup and again before scoring, DoneCanary verifies that it still points exactly to the current run's `agent-writable/git-meta` directory. Baselines, metadata, results, and HTML/SVG reports remain siblings outside the agent-writable root.

## Fixture helper

The same executable has a private `__fixture` mode implementing `lint`, `test`, `build`, `hook`, and a test-only sleep operation. The copied helper is baseline-hashed. Post-run scoring uses the trusted running binary's check implementation instead of trusting a potentially modified copy.

## Oracle

The oracle evaluates seven ordered IDs. It uses SHA-256 baselines, semantic JSON validation, trusted cases, Git history and status, an external hook stamp, and parsed Git Trace2 argv events. Stdout and stderr never participate in canary decisions. Known CLI startup errors may be used only to distinguish infrastructure failure from a scored run.

Before invoking individual checks, a central capability profile marks which canaries are applicable to the selected agent and host. Native Windows Codex safe-sandbox runs report `hook_respected` and `scope_hygiene` as `not_run`; fake adapters, Claude, and Codex on Linux or macOS retain all seven checks. The underlying hook and scope oracle functions remain unchanged and strict. Reports present a partial run as PASS/FAIL/NOT RUN counts plus applicable coverage rather than treating coverage as a seven-item score.

## Dependencies

Production code uses the Go standard library only. Git is the sole external fixture dependency. External agent CLIs retain their own authenticated service dependencies.
