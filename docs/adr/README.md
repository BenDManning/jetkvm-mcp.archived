# Architecture decision records

Status: maintained.

Architecture decision records (ADRs) preserve one consequential choice, its
rationale, rejected alternatives, consequences, evidence, and objective reasons
to revisit it. Together these records are the repository decision log. They
describe current product architecture; they do not authorize deferred
alternatives or replace executable code and tests.

The format follows the general ADR practice described by
[adr.github.io](https://adr.github.io/): capture an architecturally significant
decision and the reasons and trade-offs behind it.

## Status vocabulary

- `proposed` — under review and not yet part of the product architecture;
- `accepted` — the current decision;
- `superseded` — replaced by a later ADR, which must be linked from both records;
- `deferred` — considered but intentionally postponed pending a named trigger.

Status changes require repository review. Preserve superseded records as history;
do not silently rewrite them into the replacement decision. Factual link or typo
corrections do not change status.

## Index

| ADR | Decision | Status |
|---|---|---|
| [0001](0001-mcp-transports-and-authentication.md) | MCP transports and authentication boundary | accepted |
| [0002](0002-fresh-in-process-webrtc-sessions.md) | Fresh in-process WebRTC sessions | accepted |
| [0003](0003-ffmpeg-screenshot-decoding.md) | FFmpeg subprocess screenshot decoding | accepted |
| [0004](0004-virtual-media-integrity-and-cleanup.md) | Virtual-media integrity and cleanup | accepted |
| [0005](0005-local-only-raw-rpc.md) | Keep raw RPC local and outside MCP | accepted |
| [0006](0006-virtual-media-url-origin-boundary.md) | Deny virtual-media URL mounts outside configured origins | accepted |

## Adding or replacing a decision

1. Copy [`template.md`](template.md) to the next zero-padded number.
2. Keep one decision per record and select one defined status.
3. Link the owning implementation, tests, and product or threat contract.
4. Name at least one rejected alternative and concrete consequences.
5. Give observable revisit conditions rather than a calendar reminder.
6. Add the record here. When superseding a record, cross-link both records.
