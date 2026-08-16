# Structured operation telemetry

JetKVM MCP emits bounded JSON Lines operation telemetry to the process stderr
stream. MCP protocol responses remain on stdout for stdio deployments. The
telemetry path is diagnostic only: a full queue, writer failure, or flush timeout
never changes an operation result or retry classification.

Every event includes a UTC event time, a random per-process instance identifier,
and the public server version. Bounded shutdown evidence reports the aggregate
dropped-event count and whether a writer failure was observed when the sink
remains writable. Missing telemetry never proves that an operation did not
execute.

Telemetry remains local to stderr, bounded, non-blocking, and payload-free. The
server does not add a network telemetry backend, durable audit database, or
stable device or user identity. Operators should use access-controlled stderr
rotation with 14-day routine retention and preserve only the relevant sanitized
window when investigating an incident. Deployment needs may require a different
retention period; the server does not enforce one. Sanitized CI evidence has a
separate 30-day retention contract.

## Schema

Every retained line uses `jetkvm.operation.v2`. Operation and stage events have
exactly these fields:

| Field | Values |
| --- | --- |
| `schema` | `jetkvm.operation.v2` |
| `time` | UTC RFC 3339 timestamp |
| `process_instance_id` | random process-local opaque `proc_` identifier |
| `server_version` | public version reported by `jetkvm-mcp --version` |
| `correlation_id` | process-generated opaque `op_` identifier |
| `transport` | `stdio`, `http` |
| `operation` | `inventory`, `status`, `power`, `hid`, `capture`, `media`, `debug_rpc`, `lifecycle` |
| `stage` | `tool`, `admission`, `connect`, `auth`, `signaling`, `rpc`, `capture`, `cleanup`, `shutdown` |
| `duration_ms` | non-negative integer capped at 60000 |
| `code` | fixed public result code such as `success`, `invalid_input`, `busy`, `canceled`, or `timeout` |
| `outcome` | `succeeded`, `failed`, `not_sent`, or `unknown` |

On close, the recorder reserves one `lifecycle` / `shutdown` /
`telemetry_summary` event. It has the fields above plus `dropped_events`, the
aggregate number of events rejected by full queues, and `writer_failed`, which
states whether any completed sink write returned an error. Its outcome is
`failed` when either value indicates loss and `succeeded` otherwise. A blocked
or permanently failed sink can also prevent the summary itself from being
retained; the one-second application close deadline still bounds shutdown. If
writing the summary itself reports a failure, the recorder makes one attempt to
append a distinct `telemetry_writer_failure` event with the same correlation ID.
This attempt is also bounded and may be absent when the sink remains
unavailable; it never rewrites or contradicts an already retained summary.

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
remains both bounded and nonblocking. Shutdown drains both queues before its
reserved summary, subject to the bounded close deadline. Writer errors are
recorded for the summary but never returned through an MCP or device operation.

## Verification scope

The repository tests cover pipe-backed stdio and loopback HTTP telemetry,
success and stable failure classes, cancellation, timeout, busy outcomes,
internal stage
correlation (including cancellation-detached cleanup), final output-schema and
capture-serialization outcomes, sensitive sentinel exclusion, concurrent line
integrity, stage-pressure terminal reservation, slow and failing writers, and
forced shutdown flush.

These fixture-only tests validate instrumentation and privacy behavior. They do
not measure real JetKVM latency, qualify hardware compatibility, or authorize
production load or soak testing.
