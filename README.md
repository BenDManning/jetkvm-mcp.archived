# JetKVM MCP

A conventional, single-process Go [Model Context Protocol](https://modelcontextprotocol.io/) server for operating JetKVM devices. It supports MCP over stdio and MCP Streamable HTTP using protocol generation `2026-07-28` and the official Go SDK.

## Features

- JetKVM status and attached-host state
- Fresh PNG screenshots decoded locally with FFmpeg
- Keyboard typing and named key presses
- Absolute/relative mouse movement, clicks, and scrolling
- Seven separately named host-power tools
- Wake-on-LAN with named configured targets
- Virtual-media status, URL mount, confined local upload/mount, and unmount
- Multiple named JetKVM devices
- Stdio and stateless Streamable HTTP transports
- Optional HTTP bearer token; no deprecated HTTP+SSE endpoint
- Local-only raw diagnostic RPC command

## Installation

### Option A: Using Go (Recommended)

Requires Go 1.25.13 or newer and `ffmpeg` on `PATH`:

```sh
go install github.com/BenDManning/jetkvm-mcp/cmd/jetkvm-mcp@latest
```

`jetkvm-mcp --version` reports the installed module version for a versioned
`go install`. Local source builds without an injected release version report
`devel+<12-character vcs.revision>` and append `.dirty` when Go embeds that VCS
metadata; metadata-poor builds report `devel`. GoReleaser and container builds
keep their explicitly injected version.

### Option B: Download Binary

Download the archive for your platform from [GitHub Releases](https://github.com/BenDManning/jetkvm-mcp/releases), verify it against `checksums.txt`, extract `jetkvm-mcp`, and place it on `PATH`. FFmpeg must also be installed on the host.

Release binaries are built for:

- Linux amd64
- Linux arm64
- macOS amd64
- macOS arm64

Windows is unsupported and not planned.

### Option C: Container image

The image contains the Go server and FFmpeg in one image and runs one Go server process. Until a GitHub Container Registry package is published under the release policy, build it from the canonical checkout:

```sh
docker build -t jetkvm-mcp:local .

docker run --rm -i \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=64m \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --pids-limit 128 \
  --memory 512m \
  --cpus 2 \
  --stop-timeout 7 \
  -v "$PWD/config.yaml:/config.yaml:ro" \
  -v "/srv/jetkvm-media:/media:ro" \
  -e JETKVM_LAB_PASSWORD \
  jetkvm-mcp:local \
  --config /config.yaml
```

Container targets are Linux amd64 and Linux arm64. The limits above are
conservative starting points, not universal sizing guarantees. Adjust them for
the configured concurrency and capture workload while retaining explicit CPU,
memory, and process limits. The image runs as UID/GID 10001, supports a
read-only root filesystem when writable temporary storage is supplied, and
contains no configuration or credentials. Inject secret environment values at
process start; do not bake them into an image, command line, or log.

## Configuration

Copy [`config.example.yaml`](config.example.yaml). Credentials are never stored inline; the YAML names environment variables that contain them. Device URLs may retain an HTTP(S) path prefix but must not contain user information, a query, or a fragment. Appliance connections are direct rather than routed through `HTTP_PROXY` or `HTTPS_PROXY`.

```yaml
devices:
  lab:
    url: https://jetkvm.example.invalid
    password_env: JETKVM_LAB_PASSWORD
    media_directory: /media
    media_url_allowed_origins:
      - https://media.example.invalid
    wake_on_lan:
      server:
        mac_address: "02:00:00:00:00:01"
        broadcast_ip: 192.0.2.255

http:
  bearer_token_env: JETKVM_MCP_HTTP_TOKEN
  allowed_origins:
    - https://mcp.example.invalid
```

`limits` is optional and bounds all in-flight device work equally for stdio and
stateless HTTP: `max_operations` (16), `max_operations_per_device` (4),
`max_sessions` (8), `max_captures` (2), and `max_decoders` (2). Values must be
positive integers no greater than 1024; per-device and session capacity cannot
exceed global operations, and captures cannot exceed sessions. Exhausted global,
per-device, session, capture, or decoder
capacity returns the normal non-retryable `busy`/`not_sent` tool result before
device dispatch; requests are not queued. Mutating HID, power, and virtual-media
work is serialized per device, while unrelated devices and read-only work can
proceed within their separate limits.

Validate the complete strict configuration without starting FFmpeg, a listener,
or a device/network session:

```sh
jetkvm-mcp config validate --config config.yaml
```

Success writes `configuration valid` to stdout. Required environment variables
must be present, but their values, the private configuration path, and URL
contents are not included in validation errors.

`media_directory` is optional. Without it, local upload/mount operations are rejected. Local sources must be relative paths that resolve inside the configured directory.

URL mounting is deny-by-default per device. `media_url_allowed_origins` must list each permitted exact HTTP(S) origin; when it is absent or empty, URL mounts are rejected before a device session is opened. An origin contains only scheme, host or IP literal, and effective port. Hostnames and schemes are case-insensitive, omitted HTTP port 80 and HTTPS port 443 equal their explicit forms, and other ports must be listed explicitly. Wildcards, URL credentials, paths, queries, and fragments are invalid in configured origins. A mount URL may contain a path, query, or fragment after its origin matches, but do not put credentials or secrets in any URL component.

The configured origin is an authorization boundary, not proof of the appliance's final network destination. The JetKVM firmware performs the fetch; this process does not resolve or pin DNS, inspect or prevent firmware redirects, classify the resolved address, or enforce appliance routing. DNS names and loopback, private, link-local, or other IP literals work only when their exact origin is deliberately configured. Keep the appliance network segmented and restrict media-fetch egress to the intended service.

JetKVM firmware exposes partial-upload resumption by byte count but provides no prefix hash. To prevent a replaced local file from being combined with stale appliance data, `jetkvm-mcp` serializes virtual-media operations per device, deletes a matching `.incomplete` artifact before upload, and accepts only a fresh offset-zero upload. It hashes the confined source before upload, hashes the exact bytes consumed by the upload, and reopens and hashes the configured path before mounting or reporting completion. Interrupted, ambiguous, or locally changed uploads are cleaned up rather than resumed.

`insecure_skip_verify: true` is available for explicitly configured local appliances but should be avoided when the device has a trusted certificate.

## Run over stdio

```sh
jetkvm-mcp --config config.yaml
```

Logs and bounded structured telemetry go to stderr; stdout is reserved for MCP
messages. Missing telemetry never proves that an operation did not execute.
Keep routine telemetry for no more than 14 days in access-controlled stderr
rotation, preserving only the relevant sanitized window for an incident. See
[`docs/telemetry.md`](docs/telemetry.md) for the schema and privacy boundary.

## Run over Streamable HTTP

The recommended direct default is loopback without authentication:

```sh
jetkvm-mcp --config config.yaml --http 127.0.0.1:8080
```

The MCP endpoint is `/mcp`; liveness is the unauthenticated `/healthz`. A
successful health response is exactly `ok` followed by a newline and means only
that configuration and FFmpeg validation succeeded and this HTTP process is
accepting requests. Device reachability remains available through MCP reads.
There is no `/readyz`, background device probe, or aggregate readiness state.
The server is stateless and exposes no legacy SSE session endpoint. Binding to
a non-loopback address requires `http.bearer_token_env` to resolve to a
non-empty bearer token.

For remote access, keep the server on loopback and place it behind a TLS reverse proxy, or configure a bearer token and explicitly bind a protected interface. The server does not implement MCP OAuth.

Streamable HTTP supports native MCP clients and same-origin browser deployments. Native MCP clients may omit `Origin`, but their public `Host` must still be configured. Browser requests must use the MCP endpoint's external origin; a separately hosted browser origin is unsupported. The proxy must preserve that public `Host` header.

Every admitted public endpoint origin must be listed exactly under `http.allowed_origins`, including its scheme and any non-default port. The setting is Host/origin admission for supported deployments, not a CORS grant, and wildcard entries are rejected. Public Hosts that are not configured are rejected even when `Host` and `Origin` match. A present browser `Origin` must exactly match the request's admitted external scheme and authority. Invalid, foreign, duplicate, empty, and opaque `null` origins are rejected before bearer authentication or MCP handling.

The server does not emit CORS response headers. An admitted same-origin `OPTIONS` request remains subject to any configured bearer and then receives the endpoint's normal `405 Method Not Allowed`; an invalid or foreign preflight receives `403 Forbidden` before bearer authentication. Loopback and `localhost` Hosts remain trusted without an allowlist, but a present Origin must still use the same loopback scheme and authority.

Each `/mcp` request body is limited to 1 MiB. HTTP permits five seconds to read
headers, then 15 seconds to read the body, and 60 seconds for an idle
connection. Tool-specific operation deadlines own response duration;
there is no global write timeout. SIGINT and SIGTERM cancel stdio and active
HTTP requests through the same process context. HTTP then drains for at most
five seconds before force-closing remaining connections. An interrupted
possibly dispatched mutation still has `outcome: unknown` and is not retryable.

## Deployment guidance

### Native service

Run the binary as a dedicated unprivileged identity. Protect the configuration
and environment from other users, supply passwords and the HTTP bearer through
the service manager's secret or environment facility, disable core dumps where
practical, and mount local media read-only when it is only an upload source.
Apply explicit CPU, memory, process, file, and network limits outside the
program. Keep direct HTTP on loopback unless a protected non-loopback bind is
deliberate. Configure the supervisor to send SIGTERM and allow at least five
seconds plus normal process-manager overhead before SIGKILL.

### Container

Keep configuration and media on read-only mounts, provide only the writable
temporary storage the runtime needs, drop capabilities, retain the fixed
non-root identity, and set CPU, memory, and PID limits. Pass secrets at container
start rather than storing them in the image or mounted YAML. Give the runtime a
stop grace period longer than the server's five-second drain. Container
orchestration does not add device readiness semantics: use the content-minimal
`/healthz` only for process liveness.

### TLS reverse proxy

The proxy owns public TLS, connection and request-rate limits, edge timeouts,
and access-log protection. Preserve the external `Host` header, forward the
single `Authorization` header without logging it, and configure that exact
public origin under `http.allowed_origins`. The server deliberately ignores
`Forwarded` and `X-Forwarded-*`; those headers cannot grant admission or replace
the public `Host`/`Origin` checks. Keep the backend on loopback or a protected
network, cap proxy request bodies at no more than 1 MiB, and do not add CORS or
device probes at the proxy. During shutdown, stop routing new requests to the
backend, allow existing proxy requests to drain, send SIGTERM to the server,
and keep established backend connections available for longer than its
five-second drain before either layer force-closes them.

## MCP tools

Tool annotations are client-facing consequence hints, not authorization. All
tool execution failures use the structured `code`, `outcome`, and `retryable`
taxonomy. `outcome: unknown` means a mutation may have reached the appliance or
host: **Do not blindly retry** it; inspect status or host state first.
Device and Wake-on-LAN target aliases contain 1–128 Unicode code points after
trimming. Keyboard keys contain 1–32 code points, and modifier lists contain at
most one each of `ctrl`, `alt`, `shift`, and `meta`. Rejected inputs return a
value-free `invalid_input` result. Every success includes schema-validated
structured content; screen capture returns the PNG first and safe JSON metadata
second without duplicating image bytes in text.

| Tool | Arguments and consequence | Retry |
| --- | --- | --- |
| `jetkvm_list_devices` | No arguments. Lists configured aliases and configuration-derived availability flags; it does not open a device session and omits URLs, credentials, origins, media roots, and WOL details. | Read-only; follow `retryable`. |
| `jetkvm_get_status` | `device` selects a configured device. Returns private appliance/host power, video, USB, version, warning, and redacted media state. | Read-only; follow `retryable`. |
| `jetkvm_capture_screen` | `device`, optional `max_width`/`max_height`. Returns a fresh private PNG that can contain visible host secrets; it is not written to disk. | Read-only; follow `retryable`. |
| `jetkvm_keyboard` | `device`, `operation`, and operation-specific key/text fields. Sends private host-control intent through USB HID; typed text can enter credentials or commands. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_mouse` | `device`, `operation`, and operation-specific pointer fields. Sends USB HID movement/clicks/scrolling that can activate host UI actions. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_get_virtual_media_status` | `device`. Returns redacted mount state (`http`/`storage` source class and mode), never URLs, paths, filenames, or raw firmware fields. | Read-only; follow `retryable`. |
| `jetkvm_mount_virtual_media_url` | `device`, `url`, optional `mode`. The appliance fetches a private HTTP(S) URL and replaces its mount; its exact origin must be configured. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_mount_virtual_media_file` | `device`, confined relative `path`, optional `mode`. Reads local media below the configured media root, uploads it, and replaces the mount. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_unmount_virtual_media` | `device`. Requests removal of the current mount; valid even when no media is mounted. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_upload_virtual_media_file` | `device`, confined relative `path`. Reads local media and uploads it to appliance storage without mounting it. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_press_host_power_button` | `device`. Briefly presses physical ATX power; host response depends on firmware/OS state. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_force_host_power_off` | `device`. Holds physical ATX power and can interrupt work or cause data loss. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_press_host_reset_button` | `device`. Presses physical reset and can interrupt work or corrupt data. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_turn_host_dc_power_on` | `device`. Enables configured DC output and may boot equipment. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_turn_host_dc_power_off` | `device`. Disables configured DC output and can interrupt work or cause data loss. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_wake_host_usb` | `device`. Sends a USB HID wake action that may resume or boot the host. | Mutation; do not retry `outcome: unknown`. |
| `jetkvm_wake_host_lan` | `device`, configured `target`. Makes the appliance send a WOL network packet to that named configured target; callers cannot provide arbitrary destinations. | Mutation; do not retry `outcome: unknown`. |

### Migrate from the removed virtual-media tool

V1 removes the combined `jetkvm_virtual_media` tool. Replace its `operation`
and generic `source` arguments with the corresponding one-purpose call:

| Removed operation | V1 replacement | Argument change |
| --- | --- | --- |
| `status` | `jetkvm_get_virtual_media_status` | Keep `device`; omit `operation`, `source`, and `mode`. |
| `mount_url` | `jetkvm_mount_virtual_media_url` | Keep `device` and optional `mode`; rename `source` to `url`. |
| `mount_file` | `jetkvm_mount_virtual_media_file` | Keep `device` and optional `mode`; rename `source` to `path`. |
| `unmount` | `jetkvm_unmount_virtual_media` | Keep `device`; omit `operation`, `source`, and `mode`. |
| `upload` | `jetkvm_upload_virtual_media_file` | Keep `device`; rename `source` to `path` and omit `operation` and `mode`. |

For `jetkvm_keyboard` `type_text`, `text` is limited to 4096 bytes of
US-ASCII. Treat typed text, key intent, local paths, media URLs, screenshots,
and returned status as private operational data.

Safe discovery and observation examples (shown as MCP tool-call parameters):

```json
{"name":"jetkvm_list_devices","arguments":{}}
```

```json
{"name":"jetkvm_get_status","arguments":{"device":"lab"}}
```

## Local diagnostic RPC

Raw RPC is deliberately outside MCP `tools/list` and only available as a local CLI subcommand:

```sh
jetkvm-mcp debug rpc \
  --config config.yaml \
  --device lab \
  --method getLocalVersion \
  --params '{}'
```

The source-reviewed read-only default set is exactly `ping`,
`getLocalVersion`, and `getActiveExtension`. Every other method fails before a
device session unless that invocation includes `--unsafe-acknowledge-risk`:

```sh
jetkvm-mcp debug rpc \
  --config config.yaml \
  --device lab \
  --method getVideoState \
  --params '{}' \
  --unsafe-acknowledge-risk
```

The acknowledgement is not a safety classification or confirmation prompt. An
unreviewed method may mutate hardware or boot/storage state and may return
sensitive raw firmware data. Parameters may appear in shell history and the raw
result is written only to the explicitly invoked command's stdout; do not put
secrets in either or retain the streams unsafely.

## Development

The repository is intentionally independent from its archived predecessor. It contains no legacy Git history or remote.

Protocol provenance and exact inspected upstream revisions are recorded in
[`docs/protocol-sources.md`](docs/protocol-sources.md). Pinned official MCP
conformance and Inspector checks are described in
[`docs/protocol-gates.md`](docs/protocol-gates.md). Current actors, trust
boundaries, physical consequences, privacy rules, implemented controls,
deployment requirements, and residual risks are recorded in
[`docs/threat-model.md`](docs/threat-model.md). The bounded stderr-only operation
telemetry schema, stage boundaries, privacy exclusions, and fixture-only
verification scope are documented in [`docs/telemetry.md`](docs/telemetry.md).
The canonical GitHub Actions/local quality matrix, minimum and release Go
lanes, analyzer pins, fuzz/protocol gates, and coverage evidence policy are in
[`docs/ci-quality.md`](docs/ci-quality.md).

```sh
go test -race ./...
go vet ./...
make fuzz-smoke
```

`make fuzz-smoke` runs every target in the checked synthetic fuzz inventory once
and enforces the committed corpus privacy policy. Use `make fuzz` for a longer
30-second local run of each bounded target. Neither command uses network,
device, or FFmpeg inputs.

### Read-only real-hardware validation

After building the server, the separate validation runner can exercise one
configured device through actual MCP stdio. It lists tools, verifies the
configured-device discovery, device-status, dedicated virtual-media-status, and
capture tools' read-only annotations, confirms that discovery contains the
selected alias, validates both typed status results, and fully
decodes one PNG without writing or displaying it. It never invokes keyboard,
mouse, virtual-media mutation, power, wake, or raw-RPC operations.

```sh
go run ./cmd/jetkvm-mcp-validate \
  --binary /path/to/jetkvm-mcp \
  --config /path/to/config.yaml \
  --device configured-device-name
```

Output is a single sanitized JSON pass/fail record. It excludes configuration
paths, device names, server logs, status values, error details, and image data.
The capture call has a dedicated 30-second deadline, and its PNG buffer is
cleared after full in-memory decoding.

### Mutation-validation dry run

Before any separately approved live mutation window, validate the checked
offline plan and follow the [mutation validation
checklist](docs/mutation-validation.md):

```sh
go run ./cmd/jetkvm-mcp-mutation-checklist \
  --plan testdata/mutation-validation-plan.json
```

This source-run command only validates a closed dry-run plan. It has no device
or MCP execution path, and even a passing report explicitly states that
execution is not authorized.

The compact compatibility ledger, focused offline upstream-drift command, and
evidence-refresh triggers are documented in
[`docs/compatibility/`](docs/compatibility/README.md). Source review, fake-device
tests, and unattributed historical runs do not qualify a firmware version.

The checked-in Dockerfile builds the Linux amd64/arm64 image. `.goreleaser.yaml` builds the four supported binary targets.

## Project policies

- [`CONTRIBUTING.md`](CONTRIBUTING.md) defines contribution scope, validation,
  provenance, privacy, and untrusted-input expectations.
- [`SUPPORT.md`](SUPPORT.md) defines the project's best-effort support boundary.
- [`SECURITY.md`](SECURITY.md) provides the private vulnerability-reporting
  route and supported-version policy.

## License

MIT — see [`LICENSE`](LICENSE).
