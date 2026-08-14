# Protocol sources

The browser-to-appliance JetKVM protocol is observed behavior rather than a promised stable vendor API. The implementation and tests are grounded in these sources.

## JetKVM

- Repository: <https://github.com/jetkvm/kvm>
- Inspected commit: `b3c29a44d9e2862b8ff7530830781803ce27b060`
- Relevant surfaces:
  - `web.go` and `ui/src/routes/devices.$id.tsx`: local authentication and WebSocket signaling
  - `jsonrpc.go`: RPC method registration and parameter names
  - `jsonrpc.go`: `ping` returns the constant `pong`, and `getActiveExtension` reads the current configured extension without mutation
  - `ota.go`: `getLocalVersion` reads local application/system version metadata without mutation
  - `usb.go`: keyboard, mouse, wheel, and USB wake RPC methods
  - `usb_mass_storage.go` and `ui/src/routes/devices.$id.mount.tsx`: virtual-media state, mount, unmount, resumable upload, and authenticated upload endpoint
  - `video.go`: video state behavior

JetKVM upstream is GPLv2. This project uses upstream source as protocol evidence and does not copy upstream implementation.

The machine-checked reviewed-surface manifest, compact evidence ledger, and
offline drift procedure are maintained in
[`docs/compatibility/`](compatibility/README.md). A focused comparison on
2026-08-14 found changes on every declared surface between the inspected commit
and upstream `fe77acd5f00300a4ab9acd5da57d7bb0916351d9`; that result requires
source review and does not move this inspected pin or establish incompatibility.

The local debug CLI permits exactly `ping`, `getLocalVersion`, and
`getActiveExtension` without an unsafe acknowledgement because those handlers
were source-reviewed at the pinned commit as read-only. Method names alone are
not treated as evidence: every other raw RPC remains unreviewed or mutating and
requires the per-invocation consequence acknowledgement documented by the
product contract and ADR 0005.

The inspected firmware can resume `filename.incomplete` using only its current byte count. Because that does not prove the remote prefix belongs to the currently opened local file, this client deliberately discards stale partials and requires a fresh offset-zero upload.

## Model Context Protocol

- Specification generation: <https://modelcontextprotocol.io/specification/2026-07-28>
- Official Go SDK: <https://github.com/modelcontextprotocol/go-sdk>
- SDK tag: `v1.7.0`
- Inspected SDK commit: `bc72835f62eb94d0fb484439f886b6885b075f36`
- Official conformance runner: <https://github.com/modelcontextprotocol/conformance>, npm `0.2.0-alpha.11`, source commit `c321dd32035556e6769d3724a8ee97d87c3faaac`
- Official Inspector: <https://github.com/modelcontextprotocol/inspector>, npm `2.2.0`, release commit `672f9f41c548487a468b9e7007d2f9de14da5a69`

The server uses the official SDK for tool schemas, stdio, and stateless Streamable HTTP. It does not implement deprecated HTTP+SSE.

The executable scenario classification, source pins, fixture-safe Inspector
calls, and artifact policy are documented in
[`protocol-gates.md`](protocol-gates.md).

## Selective legacy reuse

The H.264 RTP parser in `internal/jetkvm/video.go` was selectively transferred from Ben Manning’s MIT-licensed archived implementation after confirming that the file had only standard-library and Pion dependencies and no policy/domain imports. Its tests were curated into the clean architecture. Authentication, signaling, RPC, power, HID, virtual media, receiver lifecycle, FFmpeg decoding, MCP registration, configuration, and packaging were re-authored in the clean repository.
