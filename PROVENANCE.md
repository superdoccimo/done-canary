# DoneCanary public-source provenance

Prepared: 2026-08-27 (Asia/Tokyo)

The public `main` branch is a clean-root source snapshot derived from the
private development commit
`673ed1ca9a083561fc5bd21c64bf44eee83e443c`.

Before creating the public candidate, the project was renamed from its working
name “Agent Canary” to `DoneCanary`. The repository path, Go module, CLI, data
variable, packages, reports, examples and share assets were migrated together.
Machine-specific implementation logs and the internal implementation directive
were deliberately omitted from the clean public tree.

The rename did not weaken the deterministic oracle, hook evidence, scope
checks, sandbox requirements, path validation, escaping, bounded logging or
no-telemetry boundary. The renamed source is tested again by the public-candidate
GitHub Actions matrix.

Canonical source: `https://github.com/superdoccimo/done-canary`

Canonical announcement:
`https://x.com/superdoccimo/status/2092930595206943129`

This provenance record documents engineering history. It is not a legal
identity or trademark opinion.
