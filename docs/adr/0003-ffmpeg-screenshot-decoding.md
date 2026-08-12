# 0003-ffmpeg-screenshot-decoding: FFmpeg subprocess screenshot decoding

Status: accepted

Date: 2026-08-12

## Context

A screenshot call receives one bounded H.264 access unit from the JetKVM video
session and must return a fully decoded PNG with caller-selected bounds. Go's
standard library has no H.264 decoder, while FFmpeg is available across the
supported native and container targets. Video and image data are untrusted and
can be large or malformed.

## Decision

Resolve `ffmpeg` from `PATH` during normal server startup, convert it to an
absolute executable regular-file path, and launch one subprocess for each
capture decode. Send only bounded Annex-B H.264 through stdin and receive one
bounded PNG through stdout. Use fixed non-shell arguments, a scrubbed
environment, pipe-only protocol access, allocation/pixel/thread/frame limits, a
15-second deadline, and bounded discarded stderr. Fully decode and validate the
PNG dimensions before returning it.

## Rationale

FFmpeg supplies a mature H.264 decoder on every declared target without adding a
large codec implementation to this repository. A fresh fixed-argument process
contains decoder state to one image and lets context cancellation terminate the
work. Pipes avoid temporary image files, and explicit resource and format checks
reduce the attack and exhaustion surface while preserving the single-process
product deployment outside the short-lived decoder child.

## Rejected alternatives

- **Rejected:** a pure-Go H.264 decoder, because no candidate is currently
  qualified against the product's packetization, image, resource, and platform
  requirements.
- **Rejected:** one long-lived FFmpeg daemon, because it would retain decoder
  state across captures and require framing, reset, concurrency, and recovery
  semantics.
- **Rejected:** decoder IPC or a sidecar container, because the current product
  explicitly keeps one runtime process and has no accepted isolation contract.

## Consequences

- Normal serving requires an external executable named `ffmpeg` even when a
  deployment intends to call only non-capture tools; version compatibility is
  externally supplied and not positively qualified.
- Each screenshot pays process-startup and decode cost.
- The child shares the server account's filesystem and process boundary; fixed
  arguments and environment are hardening, not a sandbox.
- Captures are limited to one frame, 3840 by 2160 pixels, bounded H.264 input,
  and bounded PNG output.
- No application-owned temporary image file is created.

## Evidence

- Capture orchestration and result validation: [`capture.go`](../../internal/jetkvm/capture.go)
- Decoder process and limits: [`decoder_ffmpeg.go`](../../internal/jetkvm/decoder_ffmpeg.go)
- Decoder argument/process/PNG tests: [`decoder_test.go`](../../internal/jetkvm/decoder_test.go)
- Fresh-frame and manager integration tests: [`capture_test.go`](../../internal/jetkvm/capture_test.go)
- FFmpeg support contract: [`product-contract.md`](../product-contract.md)
- Decoder threat controls and residual risk: [`threat-model.md`](../threat-model.md)

## Revisit trigger

Revisit when any of the following is true:

- a maintained in-process decoder fully decodes 100% of the retained H.264
  qualification corpus, passes the existing capture/decoder bounds and malformed
  input tests, and supports every declared OS/architecture target;
- a repeatable 100-capture benchmark shows subprocess startup consumes more than
  half of p95 capture latency and causes an accepted capture objective to fail;
- a declared target can no longer supply a maintained compatible FFmpeg; or
- an accepted security requirement demands a stronger decoder isolation
  boundary and defines how image bytes, cancellation, limits, and failures cross
  that boundary.
