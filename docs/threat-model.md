# Threat model summary

The normative threat model is [THREAT_MODEL.md](../THREAT_MODEL.md).

The most important boundary is that agent-controlled prose and files are untrusted, while the currently running DoneCanary oracle and evidence directory are trusted. The tool prevents ordinary path escape, protected-file editing, test tampering, hook skipping, dirty worktrees, unbounded logs, and unsafe adapter flags.

DoneCanary is not a sandbox. A deliberately malicious agent with host-level capabilities may attack the evidence directory or executable. External agent network traffic is outside the DoneCanary application boundary; DoneCanary itself imports no Go networking package.
