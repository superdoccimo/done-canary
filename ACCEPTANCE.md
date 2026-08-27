# DoneCanary v0.1 Acceptance Criteria

Status: **Core IDs and requirements were frozen before implementation; creator-first public coverage IDs were added as an explicit documentation and asset policy.**

Current release-candidate evidence is recorded in `docs/release-evidence-v0.1.0-rc.3.md`. Valid statuses are PASS, FAIL, SKIPPED, and NOT RUN.

## Core and oracle

- **AT-001**: Tool never accepts or modifies a user repository path in v0.1.
- **AT-002**: `selftest` uses fake adapters only and requires no network or agent credentials.
- **AT-003**: `fake-pass` scores exactly 7/7.
- **AT-004**: `fake-skip-hook` fails `hook_respected` and preserves valid evidence for all unaffected canaries.
- **AT-005**: `fake-edit-tests` fails `test_integrity` and any logically dependent hygiene check.
- **AT-006**: `fake-out-of-scope` fails `scope_hygiene`.
- **AT-007**: `fake-fail-build` fails `build_lint_pass`.
- **AT-008**: `fake-edit-protected` fails `instructions_respected`.
- **AT-009**: `fake-no-required-change` fails `required_change`.
- **AT-010**: The same captured final state produces byte-equivalent semantic scoring on repeated runs.

## Process and security

- **AT-011**: Timeout terminates the child and descendants and returns the documented exit code.
- **AT-012**: SIGINT cleans up the child process and leaves a valid interrupted run record.
- **AT-013**: Paths containing spaces work.
- **AT-014**: Japanese/Unicode paths work.
- **AT-015**: Symlink/path traversal cannot make DoneCanary write outside its owned run directory.
- **AT-016**: HTML escapes agent-controlled stdout/stderr.
- **AT-017**: Logs are size-bounded and truncation is clearly marked.
- **AT-018**: No telemetry or application network client exists; tests verify this boundary as far as practical.
- **AT-019**: Adapter command construction contains no dangerous permission-bypass flag.

## Output and schema

- **AT-020**: `result.json` validates against the versioned schema.
- **AT-021**: Infrastructure error is distinguishable from canary failure.
- **AT-022**: HTML and SVG render from both 7/7 and partial-failure sample results.
- **AT-023**: Terminal output is readable without color.

## Real adapters

- **AT-024**: Codex executable/version is detected from the local machine.
- **AT-025**: At least one real Codex disposable-fixture run is attempted and fully evidenced.
- **AT-026**: Claude Code state is explicitly one of verified / absent / unsupported / blocked, with evidence.
- **AT-027**: Real adapter never points at the implementation repository or another user repository.

## Build and release candidate

- **AT-028**: `go test ./...` passes.
- **AT-029**: `go vet ./...` passes.
- **AT-030**: Race-enabled tests pass where the platform supports them.
- **AT-031**: Windows/macOS/Linux target builds compile successfully.
- **AT-032**: No pre-existing local files outside the project were modified.
- **AT-033**: README first screen shows the score before architecture/caveats.
- **AT-034**: Press kit and attribution files contain verified creator links and no invented URL.
- **AT-035**: Local release-candidate artifacts and checksums are produced.

## Public-release gates

- **AT-036**: GitHub Actions matrix passes on Windows/macOS/Linux after human-approved push.
- **AT-037**: Five independent external users reproduce a run.
- **AT-038**: At least three external users understand and can share the scorecard without explanation from the author.
- **AT-039**: Final name collision check is completed before public release. Evidence is recorded in `NAME_CHECK.md`.
- **AT-040**: Human approves the visibility change and final public announcement.

### Recorded gate state for v0.1.0-rc.3

- **AT-036: PASS** — GitHub Actions run `33074490708` passed the public audit, Windows/Ubuntu/macOS test-vet-build jobs, Ubuntu race job, all three native packaging jobs, and the final bundle verification.
- **AT-037: NOT RUN** — five independent external reproductions have not yet been collected.
- **AT-038: NOT RUN** — three independent scorecard-comprehension/share tests have not yet been collected.
- **AT-039: PASS** — the practical final-name collision check is recorded in `NAME_CHECK.md`.
- **AT-040: NOT RUN** — repository visibility and the final public announcement remain human-controlled external publication decisions.

The verified prerelease is `v0.1.0-rc.3` and is published at `https://github.com/superdoccimo/done-canary/releases/tag/v0.1.0-rc.3`. Detailed checksums and evidence are in `docs/release-evidence-v0.1.0-rc.3.md`.

## Creator-first public coverage

- **AT-STRONG-001**: A strong "Don't erase the person who built it" section is public.
- **AT-STRONG-002**: The policy explicitly rejects a structure where only the introducer gets traffic and the creator disappears.
- **AT-STRONG-003**: The policy does not prohibit an introducer from gaining traffic or revenue.
- **AT-STRONG-004**: Greater third-party attention is stated to make the creator link more important, not less.
- **AT-STRONG-005**: The introducer is explicitly distinguished from the source.
- **AT-STRONG-006**: Blogs, viral posts, videos, newsletters, and mirrors are explicitly not canonical sources.
- **AT-STRONG-007**: The same creator-credit policy applies to monetized coverage.
- **AT-STRONG-008**: Coverage records can distinguish `DISTRIBUTION_SUCCESS` from `CREATOR_ATTRIBUTION_FAILURE` or `CREATOR_ATTRIBUTION_SUCCESS`.
- **AT-STRONG-009**: Required credit names `@superdoccimo`.
- **AT-STRONG-010**: Required credit links `https://x.com/superdoccimo`.
- **AT-STRONG-011**: Required credit links the canonical repository at `https://github.com/superdoccimo/done-canary`.
- **AT-STRONG-012**: The policy is generalized to other open-source maintainers and offered for reuse.
- **AT-STRONG-013**: Official generated and checked-in share assets visibly retain `DoneCanary · @superdoccimo`.
