# 0002-fresh-in-process-webrtc-sessions: Fresh in-process WebRTC sessions

Status: accepted

Date: 2026-08-12

Proposed replacement: [ADR 0008](0008-managed-per-device-webrtc-ownership.md).
This record remains accepted until that proposal is accepted and implemented.

## Context

JetKVM control RPC and video arrive through authenticated WebRTC connections.
Operations need either a data-only profile or a video receive profile, and a
failed or cancelled operation must not leave peer, signaling, RPC, video, or
HTTP state alive for a later caller. The server is otherwise stateless and has
no device-session pool.

## Decision

Use Pion WebRTC in the server process. For every device operation, create a fresh
HTTP client and cookie jar, authenticate to the configured appliance, create a
new peer connection and RPC data channel, negotiate signaling, run exactly one
manager operation, and close all associated state afterward. Add a receive-only
H.264 track only for the video profile. Bound connection and RPC setup with
timeouts and propagate cancellation through session closure.

## Rationale

A fresh session makes ownership and cleanup coincide with one operation. It
avoids stale cookies, peer state, pending RPCs, and video packets crossing calls,
and avoids synchronization and recovery rules for a shared connection pool.
Keeping Pion in-process provides direct context cancellation, typed interfaces,
and one deployable process without an additional protocol between the server
and a WebRTC helper.

## Rejected alternatives

- **Rejected:** a persistent or pooled WebRTC session per device, because it
  would require cross-call lifecycle, concurrency, stale-state, and reconnect
  semantics that current operations do not need.
- **Rejected:** a WebRTC sidecar or helper process, because it would add IPC,
  deployment, crash-recovery, and version-skew boundaries without a current
  isolation requirement.
- **Rejected:** one video-capable profile for every call, because control-only
  operations do not need a media receiver or H.264 negotiation.

## Consequences

- Authentication and WebRTC negotiation occur for every operation, adding
  latency and appliance load compared with a connection pool.
- Calls do not share cookies, pending requests, ICE state, or received video.
- Data and video profiles remain explicit, and capture obtains a fresh frame
  from a fresh video session.
- The Go process shares memory and failure domain with Pion; this is not process
  isolation from malformed WebRTC input.
- Closing an operation tears down idle HTTP connections, signaling, peer state,
  RPC waiters, video capture, and pump goroutines.

## Evidence

- Provider and lifecycle implementation: [`provider.go`](../../internal/jetkvm/provider.go)
- WebRTC profile construction: [`signaling.go`](../../internal/jetkvm/signaling.go)
- End-to-end authenticated session test: [`provider_test.go`](../../internal/jetkvm/provider_test.go)
- Capture profile test: [`capture_test.go`](../../internal/jetkvm/capture_test.go)
- Product process boundary: [`product-contract.md`](../product-contract.md)
- Network and parser risks: [`threat-model.md`](../threat-model.md)

## Revisit trigger

Revisit when any of the following is true:

- a repeatable 100-operation benchmark against a named qualified device shows
  session setup consumes more than half of p95 operation latency and causes an
  accepted latency objective to fail;
- an accepted feature requires state or a media stream to survive across two MCP
  calls, or requires concurrent operations to share one peer connection;
- a supported JetKVM interface provides authenticated typed RPC or video without
  WebRTC and passes the same manager-level test coverage; or
- an accepted security requirement mandates process isolation for WebRTC and
  defines the IPC, lifecycle, and deployment contract for a helper.
