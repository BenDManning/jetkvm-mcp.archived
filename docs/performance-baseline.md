# Read-only performance and cleanup baseline

`jetkvm-mcp-benchmark` measures only discovery, status, and screen capture. It
contains no power, HID, wake, upload, virtual-media mount/unmount, or arbitrary
RPC operation. Reports contain aggregate integer timings and fixed counters;
they exclude aliases, URLs, credentials, configuration and binary paths,
hostnames, RPC methods, firmware/status values, dimensions, payload sizes,
image/video bytes, raw errors, warnings, and logs.

## Fixture baseline

The committed baseline is produced without networking, hardware, or FFmpeg:

```sh
go run ./cmd/jetkvm-mcp-benchmark --mode fixture --iterations 100 \
  > docs/performance-baseline.fixture.json
go test -run '^$' -bench 'BenchmarkReadOnly(Discovery|Status|Capture)' \
  -benchmem -count=5 ./internal/jetkvm
go test -race ./internal/jetkvm -run '^TestReadOnlyFixtureSoak$' -count=1
```

The deterministic fixtures exercise real manager discovery, status, capture,
admission, result validation, copying, cancellation, and cleanup code. The soak
injects cancellation during provider setup, RPC, video wait, and decode. It
requires zero active fake sessions/decoders and empty manager permit channels.
Fixture timing is useful for regression mechanics and allocation visibility; it
is not appliance latency or compatibility evidence.

## Separately approved hardware run

Hardware mode is disabled unless all of `--config`, `--device`, and the explicit
`--acknowledge-read-only-hardware` flag are supplied:

```sh
jetkvm-mcp-benchmark \
  --mode hardware \
  --config /path/to/config.yaml \
  --device designated-non-production-device \
  --iterations 100 \
  --acknowledge-read-only-hardware
```

Supplying the acknowledgement does not itself authorize a run. The operator
must separately approve the designated device, time window, credentials,
retention location, and observation plan. The command constructs a manager with
only the selected device and invokes `ListDevices`, `Status`, and
`CaptureScreen`; it cannot select a mutation operation.

A hardware report remains `inconclusive` because no absolute latency objective
is accepted. The current candidate did not contact hardware and does not claim
real firmware, network, video, decoder, model, or compatibility performance.

## Stage instrumentation

An opt-in recorder carried only in a Go context measures these existing
boundaries:

- `session_setup`: provider connect through data-channel readiness or failure;
- `auth`: the authentication call;
- `signaling_setup`: WebSocket dial through offer write and pump launch;
- `data_ready`: the post-signaling wait for data-channel readiness;
- `rpc`: admission, request encode/send, response wait, and decode;
- `video_wait`: waiter registration through a fresh H.264 result or cancellation;
- `ffmpeg`: the complete `command.Run` child lifecycle including `WaitDelay`;
- `session_cleanup`: the one-time connected-session cleanup.

A nil/absent recorder is the production default. The recorder stores at most
4,096 samples, drops excess samples, is concurrency-safe, records each span
once, and accepts only the fixed stage/outcome enums. It never accepts an error
string, identifier, method, payload, or arbitrary callback.

## Report and decisions

Reports use schema version 1 and include:

- mode and requested/completed iteration counts;
- aggregate discovery/status/capture attempts, outcomes, and min/p50/p95/max
  integer microseconds, with percentiles calculated by nearest rank
  (`ceil(percentile / 100 * sample_count) - 1` in the sorted zero-based sample set);
- the same aggregate fields for allowlisted internal stages;
- the number of stage samples dropped by bounded retention;
- terminal cleanup/resource deltas;
- `fixture_only`, `keep`, `change_candidate`, or `inconclusive` architecture
  decisions.

Architecture changes require compound evidence:

- **Session change candidate:** at least 100 separately approved real-device
  operations, session setup above 50% of total p95, and failure of an accepted
  total-latency objective.
- **Decoder change candidate:** at least 100 separately approved real captures,
  FFmpeg above 50% of capture p95, and failure of an accepted capture-latency
  objective.
- **Keep:** correctness and cleanup pass and the corresponding compound trigger
  is not met.
- **Correctness defect, not optimization:** any unexpected operation failure or
  retained session, request, waiter, permit, decoder, child process, or
  goroutine.
- **Inconclusive:** fewer than 100 hardware samples or no accepted latency
  objective.

Shared-runner wall-clock p95 is not a CI gate. CI gates behavior, races, bounded
retention, allocation visibility, cancellation, and cleanup invariants.

Cleanup fields are measured from manager admission occupancy (sessions and
decoders), currently active bounded stage spans (RPC requests and video
waiters), and an O(1) process-local counter around decoder helper commands.
Helper accounting is identical across platforms and does not enumerate host
process metadata. Canceled or deadline-truncated session cleanup is classified
from its final context state rather than reported as successful.
