# DoneCanary press kit

## One sentence

DoneCanary tests whether a coding agent actually finishes a development task by running it in a disposable Git repository and scoring the final state with a deterministic external oracle.

## Short description

DoneCanary gives Codex and Claude Code a tiny repository containing realistic workflow traps: protected instructions, trusted tests, a pre-commit hook, an exact commit requirement, and a clean-worktree requirement. It then produces a terminal score plus local JSON, HTML, and SVG artifacts. Agent claims never determine the score.

## Key facts

- Local-first, single-binary Go tool
- Never operates on the user's repository
- No LLM judge, cloud scoring, leaderboard, or telemetry
- Deterministic fake adapters validate the oracle without agent credentials
- Seven stable canaries produce a shareable score; capability-limited runs explicitly show NOT RUN and applicable coverage
- Current release: `v0.1.0-rc.3` prerelease candidate
- Release page: https://github.com/superdoccimo/done-canary/releases/tag/v0.1.0-rc.3

## Copy-ready credit

```text
DoneCanary by @superdoccimo
Canonical source: https://github.com/superdoccimo/done-canary
Developer: https://x.com/superdoccimo
```

Creator: 美濃加茂まむ
X: https://x.com/superdoccimo
GitHub: https://github.com/superdoccimo

Original announcement/article: `https://x.com/superdoccimo/status/2092930595206943129`

## Creator-first coverage policy

The person who introduces DoneCanary is not the source of the project. A blog
post, viral X post, YouTube video, newsletter, mirror, redirect, or third-party
landing page does not replace the canonical repository and original developer.

Coverage may earn traffic, followers, revenue, subscriptions, sponsorship
value, or reputation. That benefit is not prohibited. If the coverage receives
more attention than the developer, the creator link becomes more important,
not less. Monetized coverage must not erase the person whose work creates the
benefit.

Do not remove `DoneCanary · @superdoccimo` from official share assets.

### Short version — English

```text
Don't build your audience by making the developer disappear.

If you cover DoneCanary, credit the source:

DoneCanary by @superdoccimo
GitHub: https://github.com/superdoccimo/done-canary
X: https://x.com/superdoccimo
```

### 短縮版 — 日本語

```text
他人のOSSを紹介して、自分だけアクセスを取る。
そのために原作者を消す。

そんな紹介はやめてください。

DoneCanaryを紹介するなら、原作者への導線を残してください。

DoneCanary — @superdoccimo
GitHub: https://github.com/superdoccimo/done-canary
X: https://x.com/superdoccimo
```

The full bilingual policy and the distribution/attribution classification
template are in `ATTRIBUTION.md`.

## Assets

- `assets/scorecard-7-of-7.svg` — perfect-score example
- `assets/scorecard-5-of-7.svg` — partial-score example
- `examples/result-7-of-7.json` — machine-readable perfect result
- `examples/result-5-of-7.json` — machine-readable partial result

These assets may be used when accurately describing DoneCanary. Preserve the
visible creator attribution. Describe `v0.1.0-rc.3` as a prerelease candidate,
not as a stable release.

This coverage policy does not add software-use restrictions beyond the MIT License; software permissions are governed by `LICENSE`.
