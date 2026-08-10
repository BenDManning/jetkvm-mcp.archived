# Protocol sources

The browser-to-appliance JetKVM protocol is observed behavior rather than a promised stable vendor API. The implementation and tests are grounded in these sources.

## JetKVM

- Repository: <https://github.com/jetkvm/kvm>
- Inspected commit: `b3c29a44d9e2862b8ff7530830781803ce27b060`
- Relevant surfaces:
  - `web.go` and `ui/src/routes/devices.$id.tsx`: local authentication and WebSocket signaling
  - `jsonrpc.go`: RPC method registration and parameter names
  - `usb.go`: keyboard, mouse, wheel, and USB wake RPC methods
  - `usb_mass_storage.go` and `ui/src/routes/devices.$id.mount.tsx`: virtual-media state, mount, unmount, resumable upload, and authenticated upload endpoint
  - `video.go`: video state behavior

JetKVM upstream is GPLv2. This project uses upstream source as protocol evidence and does not copy upstream implementation.

The inspected firmware can resume `filename.incomplete` using only its current byte count. Because that does not prove the remote prefix belongs to the currently opened local file, this client deliberately discards stale partials and requires a fresh offset-zero upload.

## Model Context Protocol

- Specification generation: <https://modelcontextprotocol.io/specification/2026-07-28>
- Official Go SDK: <https://github.com/modelcontextprotocol/go-sdk>
- SDK tag: `v1.7.0`
- Inspected SDK commit: `bc72835f62eb94d0fb484439f886b6885b075f36`

The server uses the official SDK for tool schemas, stdio, and stateless Streamable HTTP. It does not implement deprecated HTTP+SSE.

## Selective legacy reuse

The H.264 RTP parser in `internal/jetkvm/video.go` was selectively transferred from Ben Manning’s MIT-licensed archived implementation after confirming that the file had only standard-library and Pion dependencies and no policy/domain imports. Its tests were curated into the clean architecture. Authentication, signaling, RPC, power, HID, virtual media, receiver lifecycle, FFmpeg decoding, MCP registration, configuration, and packaging were re-authored in the clean repository.
