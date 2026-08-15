# Structured operation telemetry

JetKVM MCP emits bounded JSON Lines operation telemetry to the process stderr
stream. MCP protocol responses remain on stdout for stdio deployments. The
telemetry path is diagnostic only: a full queue, writer failure, or flush timeout
never changes an operation result or retry classification.

## Schema

Every retained line uses `jetkvm.operation.v1` and exactly these fields:

| Field | Values |
| --- | --- |
| `schema` | `jetkvm.operation.v1` |
| `correlation_id` | process-generated opaque `op_` identifier |
| `transport` | `stdio`, `http` |
| `operation` | `inventory`, `status`, `power`, `hid`, `capture`, `media`, `debug_rpc`, `lifecycle` |
| `stage` | `tool`, `admission`, `connect`, `auth`, `signaling`, `rpc`, `capture`, `cleanup`, `shutdown` |
| `duration_ms` | non-negative integer capped at 60000 |
| `code` | fixed public result code such as `success`, `invalid_input`, `busy`, `canceled`, or `timeout` |
| `outcome` | `succeeded`, `failed`, `not_sent`, or `unknown` |

All stages of one operation share its correlation ID. Stage timing measures the
existing boundary without moving or changing that boundary:

- `admission`: bounded session permit decision;
- `connect`: provider connection through data-channel readiness;
- `auth`: existing device authentication call;
- `signaling`: WebSocket dial through successful offer write and signaling pump launch;
- `rpc`: one firmware RPC call, excluding method and parameters;
- `capture`: the existing capture lifecycle, including validation and decoding;
- `cleanup`: connected-session teardown or incomplete media cleanup;
- `tool`: final public tool result;
- `shutdown`: server stop and telemetry flush.

## Privacy boundary

Telemetry never includes tool arguments or results, device aliases, target
addresses, URLs, local paths, typed keyboard text, image or video bytes,
firmware values, raw RPC methods/parameters/results, configuration values,
credentials/tokens, raw errors, HTTP bodies, subprocess commands, or child
process output. Call sites submit only closed enum values. Unknown values are
dropped before serialization.

The recorder uses separate bounded asynchronous queues for terminal tool events
and other stages, plus one writer goroutine. Stage pressure therefore cannot
evict a tool terminal event, and concurrent producers cannot interleave JSON
lines. A full queue still drops its new event rather than blocking the operation;
a persistently blocked or failed sink cannot be lossless while the recorder
remains both bounded and nonblocking. Shutdown drains both queues with a bounded
flush; writer errors are intentionally ignored.

## Verification scope

The repository tests cover stdio and loopback HTTP telemetry, success and stable
failure classes, cancellation, timeout, busy outcomes, internal stage
correlation (including cancellation-detached cleanup), final output-schema and
capture-serialization outcomes, sensitive sentinel exclusion, concurrent line
integrity, stage-pressure terminal reservation, slow and failing writers, and
forced shutdown flush.

These fixture-only tests validate instrumentation and privacy behavior. They do
not measure real JetKVM latency, qualify hardware compatibility, or authorize
production load or soak testing.
