# Result schema v0.1

The machine-readable schema is [`schemas/result-v0.1.schema.json`](../schemas/result-v0.1.schema.json).

Every result includes:

- schema and fixture versions;
- stable run, time, agent, invocation, and host metadata;
- an infrastructure status separate from scoring;
- seven ordered canaries with stable IDs and `pass`, `fail`, or `not_run` status;
- a passed/total score;
- bounded-process exit, timeout, interrupt, and truncation state.

Readers must ignore unknown fields for forward compatibility. They must not reinterpret `not_run` as `fail`, and they must check `infrastructure_status` before comparing scores.

Agent-controlled stdout and stderr are intentionally absent from `result.json`; they remain bounded local files and are safely escaped when included in HTML.
