# 0008: Managed per-device WebRTC ownership

Status: accepted

Date: 2026-08-17

Specification: [GitHub issue #107](https://github.com/BenDManning/jetkvm-mcp/issues/107)

Replaces [ADR 0002](0002-fresh-in-process-webrtc-sessions.md).

## Context

JetKVM permits one authoritative operator session. Establishing another
authenticated WebRTC session immediately makes the new session authoritative,
notifies the displaced session with `otherSessionConnected`, clears HID state,
and closes the old peer after a short handoff interval. A brief overlap between
peers during that handoff does not grant concurrent operator authority.

The current server instead creates and closes a fresh WebRTC connection for
every operation. Same-device status, HID, power, virtual-media, and capture
calls can therefore negotiate independent peers concurrently, compete for the
single authoritative session, repeat authentication and signaling, and trigger
operator switching. This contradicts the appliance ownership boundary.

A video-capable connection already contains the RPC data channel, authenticated
HTTP state, and receive-only video track needed by all current operations.
Keeping one such connection per configured device removes the need for separate
data and capture sessions. It also creates cross-call lifecycle, scheduling,
failure, handoff, and shutdown responsibilities that must be explicit.

The governing principle is:

> Connection acquisition is consequential. Work dispatch is separately
> consequential. Neither authority implies the other, and neither permits
> replay.

## Decision

### Module boundary

The Manager owns process-wide admission and a registry containing one
independently testable `deviceOwner` per configured JetKVM. It does not directly
implement each device state machine. A device owner owns:

- its serialized lifecycle state machine and immutable snapshot;
- the current managed-session generation;
- ordinary, mutation, RPC, and capture scheduling;
- HID safety state;
- observed device state and bounded change notification;
- the selected capability profile; and
- idle, release, takeover, cleanup, and stop transitions.

A narrow internal `SessionConnector` establishes one video-capable connected
session. Its production adapter owns authentication, signaling, RPC, video,
HTTP, and pump resources for that generation. Deterministic test adapters
provide connection, loss, event, RPC, video, and cleanup behavior. MCP handlers
call Manager operations and never receive transport objects.

The owner loop performs only bounded in-memory transitions, lease decisions,
generation validation, worker registration, cancellation signaling, and
snapshot publication. Connection setup, RPC, frame collection, decoding,
uploads, hashing, transport closure, pump joining, and reconciliation execute
in bounded workers outside the loop. Every worker completion carries owner,
generation, worker, and dispatch evidence. A stale completion releases only its
own resources and cannot alter current state or satisfy a current waiter.

### Ownership and snapshots

Each owner publishes an immutable private snapshot to the Manager. Its axes are
orthogonal:

- ownership: `idle`, `active`, `taken_over`, `uncertain`, `released`, or
  `stopped`;
- transition: none, `connecting`, `draining`, or `closing`;
- health: `healthy`, `degraded`, or `unavailable`;
- observation freshness and provenance;
- bounded typed observations; and
- capability-profile revision.

Snapshots contain no peer, channel, cookie, HTTP client, mutable waiter, or
other transport pointer. They are authoritative. Bounded outward change
notifications are lossy wake hints that cause consumers to reread the snapshot.

Critical conditions are not represented only by mailbox entries. Takeover,
connection loss, shutdown, idle expiry, cancellation, and worker completion are
latched in owner state or have a reserved terminal path. Nonblocking producers,
including Pion callbacks, may coalesce a bounded wakeup because the underlying
condition remains recorded. Admission requests have owned response slots and
each worker has exactly one reserved terminal-completion path.

Policy intent is a separate future axis. Durable maintenance windows,
restart-surviving release, mutation-reconciliation requirements, and
automation-owned windows are not implemented by this decision.

### Acquisition, reuse, and idle release

Every generation negotiates the video-capable profile. Readiness requires the
RPC data channel and a successful `ping`; it does not require an active HDMI
signal or an already received video frame. The receiver continuously drains
valid video packets while the generation is active.

An ordinary operation admitted while ownership is `idle` may lazily acquire a
generation. Concurrent callers share one Manager-owned connection attempt and
wait cancellably. Canceling one caller removes only that waiter. If all waiters
cancel, the owner cancels and cleans up the attempt. No caller receives a
generation after its context expires. Setup uses a Manager-owned bounded
context, performs no background retry, and has no manager-owned retry cooldown.

Successful connection reuse consumes no connection-attempt permit. A failed
setup cleans up under the lesser of its remaining setup budget and five seconds
before releasing that permit.

An active generation remains resident for a process-wide idle lease, defaulting
to 60 seconds. The idle clock starts only when no admitted or waiting work still
needs the device generation. Once a complete encoded capture frame has been
copied into an immutable bounded buffer, decoding no longer holds the generation
lease. FFmpeg remains globally admitted and independently cancellable, but
decoding cannot prevent idle closure, takeover, or connection shutdown.

Idle expiry atomically closes the generation and returns ownership to `idle`.
Later ordinary demand may reacquire it. Idle cleanup uses its own detached
five-second context. A cleanup timeout latches `uncertain` and emits telemetry;
it does not launch a background reconnect.

### Explicit release and takeover

The v1 MCP surface adds two one-purpose lifecycle tools:

- `jetkvm_release_session`, accepting only `device`, returning only
  `{device, status: "released"}`, and carrying `idempotentHint=true`; and
- `jetkvm_take_over_session`, accepting only `device`, returning only
  `{device, status: "authoritative"}`, and carrying
  `idempotentHint=false`.

Both are closed-world and do not mutate appliance or attached-host state, but
takeover is consequential: it may immediately displace an external
authenticated operator without that operator's consent. Its name and tool
description must state that effect. An invocation is the v1 authority to
acquire ownership; no epoch or confirmation token is required. It never grants
permission to replay an earlier operation.

Release checks and latches the per-device lifecycle state atomically before it
counts as lifecycle work, so the release call does not make the device busy by
itself. It still requires global operation capacity. V1 release fails
immediately with `busy` / `not_sent` when device-bound admitted or waiting work
exists. It does not wait for a drain. Release on an already released owner
succeeds without reconnecting. Release from `idle` latches `released` without
connecting. Release from `uncertain` or `taken_over` retries bounded local
cleanup and becomes released only after every known local resource closes and
joins. Cleanup timeout returns `ownership_uncertain`, `failed`, and nonretryable.
Ordinary work while released returns `session_released`, `not_sent`, and
nonretryable.

The owner abstraction and transition model preserve a future
`active -> draining -> released` maintenance workflow. Fail-fast release is a
v1 policy, not a structural invariant. Released state remains process-local;
ordinary operations reject it until explicit takeover, and restart-surviving
maintenance is deferred.

Takeover on a healthy active generation validates it with `ping`. Takeover from
`idle`, `uncertain`, `taken_over`, or `released` performs one bounded connection
attempt. Before connecting from `uncertain` or `taken_over`, every resource from
the prior local generation must already be closed and joined or must complete
bounded cleanup. Cleanup failure returns `ownership_uncertain` and does not open
a replacement connection. Concurrent incompatible lifecycle transitions return
`busy` / `not_sent`; takeover is not queued behind them.

If validation of an active generation loses the connection, that takeover call
latches `uncertain` and fails with `ownership_uncertain`. It does not open a
replacement generation inside the same call; a later explicit takeover is new
ownership-acquisition authority.

### Loss, external takeover, and replay

`uncertain` and `taken_over` are sticky ownership states:

```text
idle       + admitted ordinary work -> connecting -> active
active     + ambiguous loss         -> uncertain
active     + recognized takeover    -> taken_over
uncertain  + ordinary work          -> reject: takeover required
taken_over + ordinary work          -> reject: takeover required
released   + ordinary work          -> reject: explicitly released
uncertain  + take_over_session      -> connecting -> active
taken_over + take_over_session      -> connecting -> active
```

The RPC parser gains an allowlisted unsolicited-event path for
`otherSessionConnected`. Recognized takeover immediately fences dispatch,
latches `taken_over`, and starts bounded local closure. Undispatched work fails
with `session_taken_over`, `not_sent`, and nonretryable. An inconclusive
already-dispatched read fails with `session_taken_over`, `failed`, and
nonretryable; a dispatched mutation whose response is not conclusive remains
`unknown`. The displaced generation sends no cleanup RPC: firmware has already
cleared HID state, and another RPC could interfere with the newly authoritative
operator.

An unexplained terminal loss latches `uncertain`. An inconclusive dispatched
read returns `ownership_uncertain`, `failed`, and nonretryable; a possibly
dispatched mutation remains `unknown`. Later ordinary demand rejected by the
sticky state returns `ownership_uncertain`, `not_sent`, and nonretryable. ICE
`Disconnected`, `Failed`, or `Closed`, data-channel closure, or a terminal
signaling failure ends the generation immediately. V1 has no same-generation
recovery grace and no automatic connection acquisition from `uncertain` or
`taken_over`.

Creating a new JetKVM connection is ownership acquisition, not neutral
transport recovery. Future recovery policy may authorize acquisition for an
explicit automation-owned window, but it must remain separate from authority
to dispatch a particular operation or reconcile a result. A possibly sent
mutation is never redispatched under any condition.

Conclusive work may finish after generation loss. A complete encoded capture
already copied into its bounded immutable buffer may decode and succeed. A
multi-probe status loses the whole result if its generation is lost before all
required probes finish. Frame acquisition and active HTTP transfers are
canceled; a mutation or upload interrupted after dispatch is `unknown`.

### Scheduling and HID safety

The initial capability profile permits one outstanding firmware RPC per device.
Complete mutation sequences remain serialized against other mutations, while
reads may interleave at explicit RPC boundaries. A bounded HTTP upload may
overlap a read or capture. One RPC may overlap one video frame acquisition
because data and video use distinct channels in the combined profile.

Captures wait in a cancellable per-device queue. Each admitted call receives a
fresh post-admission frame; frames are never shared. Cancellation before capture
ownership is `not_sent`. The existing global and per-device operation limits
bound waiting calls. The capture gate covers only frame acquisition; decoding
uses its separate global permit.

A read cannot interleave between a HID press and its matching release or neutral
report. The owner reserves cleanup capacity before dispatching the press and
retains the RPC gate through acknowledged neutralization. Caller cancellation
does not remove the bounded cleanup path. Recognized external takeover is the
exception: the displaced generation sends no further RPC because firmware has
already cleared HID state.

These concurrency choices are conservative scheduling rules in the initial
capability profile, not universal firmware invariants. Named physical evidence
may select a profile that safely relaxes them. Mutation non-replay, generation
fencing, boundedness, and external-operator precedence are unconditional.

### Admission and configuration

Configuration accepts at most 64 devices. The process-wide `limits` mapping
adds:

- `max_connection_attempts`, default 8, range 1 through 64, and no greater than
  `max_operations`; and
- `session_idle_timeout`, default 60 seconds, range 10 seconds through one hour.

`max_connection_attempts` covers only active authentication and signaling
attempts and releases after successful connection or complete failed-setup
cleanup. Every configured device may remain resident. The former `max_sessions`
field is removed and rejected immediately as an unknown/obsolete field; it is
not ignored or aliased. This is a compatibility break requiring the accepted
issue, migration note, and appropriate Semantic Versioning treatment.

Admission meanings become:

- `max_operations`: all admitted ordinary and lifecycle device work globally;
- `max_operations_per_device`: ordinary work waiting or running for one device;
- `max_connection_attempts`: simultaneous connector attempts;
- `max_captures`: admitted captures globally; and
- `max_decoders`: active FFmpeg decodes.

Remove `max_captures <= max_sessions`; retain
`max_captures <= max_operations`. Values outside their bounds, more than 64
configured devices, and unsafe parent relationships fail offline validation and
startup before any device connection opens.

### Cleanup and shutdown

A generation is locally released only after its signaling, RPC, video, peer,
HTTP, and pump resources close and join within a detached cleanup deadline.
This proves local release, not firmware state. Explicit release and idle cleanup
each receive an independent five-second detached context.

Process shutdown:

1. atomically stops new device and lifecycle admission;
2. cancels attempts, timers, frame acquisition, decoders, uploads, and active
   operation contexts;
3. closes all managed generations in parallel and joins their workers under a
   detached shutdown context;
4. bounds operation completion, generation cleanup, and HTTP draining inside
   the existing shared five-second process-shutdown budget, then force-closes;
   and
5. flushes telemetry under its existing separate one-second bound.

Mutation interruption after dispatch retains `unknown` semantics.

### Telemetry and qualification

Operation telemetry advances to `jetkvm.operation.v3` and adds a separate
`jetkvm.session.v1` lifecycle schema. Both use a random process-local opaque
`device_ref`; session events also use a random per-generation `session_ref`.
Neither is a configured alias, address, durable device identity, nor reusable
generation identifier. Lifecycle events distinguish reuse, connect attempt,
active, idle release, explicit release, recognized takeover, uncertain loss,
cleanup timeout, and shutdown. Missing telemetry never proves lifecycle state.

A durable deployment-scoped telemetry identity is a separate future privacy and
operations decision. V1 adds no network backend, durable journal, alias, or
cross-restart identifier.

Acceptance requires one owner-authorized run on a named build, model, firmware,
runtime, and FFmpeg identity. At minimum it covers:

- 100 sequential mixed status, media-status, capture, HID, and controlled
  power/media operations on one generation with zero switch prompts;
- 20 concurrent status-plus-capture pairs with fresh distinct captures;
- reads and captures during a bounded virtual-media upload;
- three explicit release, external-browser, and takeover cycles;
- ordinary 60-second idle release followed by browser connection without a
  switch prompt;
- recognized browser takeover during an in-flight read, proving
  `session_taken_over`, no automatic reconnect, and no post-takeover RPC;
- short and long terminal transport interruptions without takeover notification,
  both proving `ownership_uncertain` and no automatic reconnect;
- SIGTERM during active read/capture and separately during one owner-approved
  disposable mutation; and
- browser connection after explicit release, idle expiry, and shutdown.

A named observer records browser prompt state. The disposable host displays a
changing non-secret marker; retained evidence records only capture metadata and
sanitized marker observations, never images. Telemetry corroborates reuse but
does not replace observation. Physical mutations run only to observable
completion; forced ambiguous mutation behavior is fixture-tested unless a
separate destructive authorization and recovery plan is granted.

If RPC/frame or upload overlap destabilizes a generation, use session-wide
serialization and rerun the complete qualification. If sequential reuse itself
causes switch prompts, reject this design rather than qualifying it with a
warning.

## Rationale

One resident combined-profile generation matches JetKVM's authoritative-session
constraint and removes repeated authentication, signaling, and self-induced
switching. A deep per-device owner contains the difficult lifecycle and
scheduling rules behind a narrow Manager-facing interface. Generation fencing,
latched state, and detached bounded cleanup make cancellation, takeover, stale
completion, and shutdown testable without exposing peers to MCP handlers.

Separating acquisition, dispatch, and reconciliation authority prevents a
read-only purpose from being mistaken for harmless connection setup and prevents
recovery from becoming mutation replay. Conservative capability profiles allow
later firmware-specific improvement without weakening unconditional safety.

## Rejected alternatives

- **Rejected:** retain fresh sessions per operation. It competes with the
  appliance's single authoritative operator session and repeats setup work.
- **Rejected:** pool multiple sessions per device. Multiple server peers cannot
  provide independent authority and make switching behavior worse.
- **Rejected:** keep one monolithic Manager state machine. It couples fleet
  admission to transport, device scheduling, observations, and firmware quirks.
- **Rejected:** reconnect automatically after any loss. A replacement connection
  can displace an external operator even when intended only for reconciliation.
- **Rejected:** recover `Disconnected` peers for five seconds. Inspected firmware
  treats that state as terminal, and no qualified recovery behavior exists.
- **Rejected:** pipeline firmware RPC initially. The client codec can correlate
  responses, but firmware concurrency is not physically qualified.
- **Rejected:** share one captured frame among concurrent callers. Each capture
  promises a fresh post-admission observation.
- **Rejected:** hold the device generation through FFmpeg decoding. Once the
  encoded frame is copied, decoding has no device or transport dependency.
- **Rejected:** make release wait for work in v1. An atomic drain remains a
  future maintenance policy rather than an implicit lifecycle queue.
- **Rejected:** add durable mutation journals, maintenance intent, fleet policy,
  status caching, or stable telemetry identity in this refactor. Each requires
  its own authority, persistence, privacy, and reconciliation contract.

## Consequences

- Normal operations reuse one warm authenticated connection and video receiver.
- Each configured device gains resident cookies, peer state, video parsing, and
  goroutines until release, expiry, takeover, failure, or shutdown.
- An idle resident session can delay JetKVM OTA activity until released.
- MCP gains two lifecycle tools and three stable ownership error codes.
- The `max_sessions` configuration field becomes an intentional compatibility
  break with `max_connection_attempts` as its replacement.
- Correctness depends on a more complex but isolated owner state machine and
  deterministic concurrency tests.
- External takeover and unexplained loss deliberately require an explicit v1
  takeover before ordinary work resumes.
- Process restart forgets released and takeover intent; durable maintenance and
  resumable workflows remain unsupported.
- Qualification is required before claiming that mixed RPC/video/upload reuse
  is safe on a model and firmware combination.

## Evidence

- Managed-session connector: [`provider.go`](../../internal/jetkvm/provider.go)
- Current Manager and admission: [`manager.go`](../../internal/jetkvm/manager.go)
- RPC dispatch phases: [`rpc_session.go`](../../internal/jetkvm/rpc_session.go)
- Video capture ownership: [`video_receiver.go`](../../internal/jetkvm/video_receiver.go)
- Protocol provenance: [`protocol-sources.md`](../protocol-sources.md)
- Product behavior and compatibility: [`product-contract.md`](../product-contract.md)
- Telemetry contract: [`telemetry.md`](../telemetry.md)
- Physical qualification: [`physical-qualification.md`](../physical-qualification.md)
- Domain vocabulary: [`CONTEXT.md`](../../CONTEXT.md)

The implementation and qualification evidence named above do not yet satisfy
this proposed decision.

## Revisit trigger

Revisit when any of the following is true:

- qualified firmware provides authenticated, non-disruptive observer or
  multi-operator sessions;
- named physical evidence proves safe RPC pipelining, same-generation transport
  recovery, or a narrower profile without weakening operator precedence;
- a durable policy layer defines automation-owned windows, restart-surviving
  maintenance, reconciliation authority, and storage/privacy requirements;
- retained qualification shows sequential resident reuse triggers a switch
  prompt or cannot maintain one authoritative server generation; or
- an accepted isolation requirement moves WebRTC into a separately bounded
  process and defines its IPC and lifecycle contract.
