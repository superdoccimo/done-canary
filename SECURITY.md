# Security policy

## Supported version

Only the latest published DoneCanary release receives security fixes. `v0.1.0-rc.3` is a prerelease candidate, not a stable release, and does not create a long-term support commitment.

## Reporting a vulnerability

Use GitHub's private vulnerability reporting for `superdoccimo/done-canary` if it is enabled. If that channel is unavailable, contact the creator through the verified [superdoccimo GitHub profile](https://github.com/superdoccimo) without including credentials, tokens, private repository contents, or exploit data in a public issue.

## Security boundary

DoneCanary never accepts a user repository path and creates only disposable fixtures below its dedicated data root. It does not collect credentials, has no telemetry, and contains no application network client.

The fixture is not a hardened sandbox. DoneCanary relies on the invoked coding-agent CLI's safe workspace mode and intentionally refuses dangerous permission-bypass flags. See [THREAT_MODEL.md](THREAT_MODEL.md) for the full boundary.
