# Product contract

Status: accepted for implementation in Forgejo issue #1.

## Purpose

`jetkvm-mcp` is a conventional Model Context Protocol server for operating one or more JetKVM devices. It is an integration product, not a policy or qualification system.

## Runtime

A single Go executable provides:

- MCP over stdio;
- MCP Streamable HTTP;
- local CLI diagnostics, including `jetkvm-mcp debug rpc`.

The process connects directly to JetKVM using its local HTTP, WebSocket, WebRTC, JSON-RPC, HID, and virtual-media interfaces. FFmpeg may be invoked locally for H.264 screenshot decoding. The container distribution contains one application process; there is no decoder service or sidecar.

Logs and diagnostics go to stderr. Stdio stdout is reserved for MCP protocol traffic.

## MCP tools

The server exposes ordinary, narrowly typed tools for:

- device status;
- screenshot capture;
- keyboard input;
- mouse input;
- virtual-media operations;
- short host power-button press: `jetkvm_press_host_power_button`;
- forced host power-off/long press: `jetkvm_force_host_power_off`;
- host reset-button press: `jetkvm_press_host_reset_button`;
- host DC power on: `jetkvm_turn_host_dc_power_on`;
- host DC power off: `jetkvm_turn_host_dc_power_off`;
- USB wake: `jetkvm_wake_host_usb`;
- Wake-on-LAN: `jetkvm_wake_host_lan`.

Each power/wake operation has its own schema, description, annotations, errors, and structured result. There is no combined action-selector power tool.

Raw arbitrary JetKVM RPC is never registered in `tools/list` and is not reachable through either MCP transport. It exists only as a local CLI diagnostic command using normal configured device details and credentials.

## Virtual media

A configured media directory is optional. When present, local media paths must resolve inside it. The server works without a media directory, and URL-based media remains available. URL media sources must not contain inline user information. No per-file grant list or predeclared hash database is required.

Local files must be non-empty regular files. Firmware partial-upload offsets are not trusted because the appliance does not provide a prefix hash: virtual-media operations are serialized per device, stale `.incomplete` artifacts are discarded, and upload negotiation must return offset zero. The confined source, exact uploaded bytes, and reopened configured path are content-hashed for consistency before mount or successful return; interrupted, ambiguous, or locally changed uploads are cleaned up rather than resumed.

## HTTP deployment

Streamable HTTP defaults to a loopback listener with no authentication. Optional bearer-token authentication is supported and is required before non-loopback binding. Reverse-proxy deployment requires explicit allowed public origins; unconfigured public Hosts are rejected even when their Host and Origin match, while no-Origin native clients remain supported on loopback or configured public Hosts. OAuth extensions and deprecated HTTP+SSE are not initially implemented.

## Installation and releases

User-facing installation methods are exactly:

- **Option A: Using Go (Recommended)**
- **Option B: Download Binary**
- **Option C: Container image**

Binary targets:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`

Container targets:

- `linux/amd64`
- `linux/arm64`

Windows is unsupported and not planned. The canonical module and repository coordinate is Forgejo.

## Explicit non-goals

The clean implementation does not contain capability grants, qualification flags or databases, firmware approval/override mechanisms, trust/control planes, malicious-agent framing, decoder IPC, multiple runtime containers, or a fake appliance treated as product authority. Test fixtures support implementation; they do not replace live-device compatibility evidence.
