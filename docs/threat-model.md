# JetKVM MCP threat model and privacy data flows

Status: maintained.

Review basis: MCP revision `2026-07-28`; reviewed 2026-08-16.

Ownership: this document owns the security and privacy model for the current
product boundary. It records implemented controls, deployment requirements,
and residual risks; it does not add an authorization service, policy engine, or
new product behavior. The [product contract](product-contract.md) owns supported
behavior, the [README](../README.md) owns operation, and code and tests remain
authoritative for executable behavior. Review this model whenever a tool,
transport, configuration field, device protocol, decoder, packaging path, or
privacy-sensitive diagnostic changes.

## Scope and security objective

`jetkvm-mcp` converts MCP calls from one trusted administrative principal into
observations and privileged actions on configured JetKVM appliances and their
attached hosts. The primary security objectives are:

1. only the intended local process or HTTP bearer holder can invoke MCP tools;
2. callers and operators can see each operation's real physical, network, and
   data consequence before invoking it;
3. untrusted network, firmware, video, and file inputs are bounded and parsed
   defensively;
4. secrets and private payloads cross only the boundary needed for the selected
   operation and are not added to application logs or durable local state; and
5. ambiguous failures do not get described as proof that a physical mutation
   did or did not occur.

The model covers stdio, stateless Streamable HTTP, configuration and environment
resolution, JetKVM HTTP/WebSocket/WebRTC/RPC, HID and power effects, screenshot
capture and FFmpeg, virtual media, local `debug rpc`, diagnostics, resource
exhaustion, CI, release artifacts, and the container. It does not claim to model
the security of an MCP
client, LLM, reverse proxy, JetKVM firmware, attached host, media origin, DNS
resolver, operating system, or artifact registry beyond the boundary they
present to this process.

No real appliance, attack traffic, credential change, deployment change, or
public disclosure is required to verify this document.

## Architecture, actors, entry points, and trust boundaries

```mermaid
flowchart LR
    O[Operator / deployment owner]
    C[Trusted MCP client]
    P[Reverse proxy / protected network]
    S[jetkvm-mcp process]
    F[Config file + environment]
    M[Configured media directory]
    X[FFmpeg executable]
    J[Configured JetKVM]
    H[Attached host]
    R[Caller-selected media origin]
    B[GitHub / CI / release inputs]

    O -->|configuration, env, argv| F
    F -->|device URLs, credential values, policy inputs| S
    C -->|newline JSON-RPC over stdio| S
    C -->|HTTPS outside this process| P
    P -->|HTTP POST /mcp, bearer, Host, Origin| S
    S -->|MCP responses and health| P
    S -->|login HTTP, session cookie, WebSocket signaling| J
    J -->|WebRTC data RPC and H.264 video| S
    S -->|bounded H.264 over pipes| X
    X -->|bounded PNG over pipes| S
    S -->|confined file read + upload| M
    S -->|upload bytes / control RPC| J
    J -->|HTTP(S) fetch for mount_url| R
    J -->|HID, reset, power, wake, virtual media| H
    B -->|source, dependencies, actions, base images| B
    B -->|binary / archive / image| O
```

| Boundary | What crosses it | Trust decision and current enforcement |
|---|---|---|
| B1 — MCP client to local process | Tool names, input fields, images, structured results, and tool errors | Stdio relies on operating-system process launch and stream ownership. There is no application authentication on stdio. The launching client is the administrative principal. Only MCP revision `2026-07-28` is admitted. |
| B2 — MCP client/proxy to HTTP server | Bearer token, `Host`, `Origin`, and stateless JSON-RPC POSTs | Loopback is the recommended direct boundary. Non-loopback bind requires a bearer token; public Hosts/origins require exact configuration. Only MCP revision `2026-07-28` is admitted. The server has one bearer principal, no end-user identity, scopes, OAuth grants, or stateful MCP sessions. TLS is outside the process. |
| B3 — operator state to process | YAML, environment-variable names and values, CLI arguments, media root, device URLs, Wake-on-LAN targets | These are trusted administration inputs. YAML is strict and bounded. Credentials are resolved from the environment rather than accepted inline. File permissions, environment isolation, and shell-history handling are deployment duties. |
| B4 — server to configured JetKVM | Password login, cookie, HTTP, WebSocket signaling, WebRTC data/video, RPC calls, and upload bytes | A configured endpoint is trusted as the selected appliance, but responses are treated as malformed until bounded and decoded. Device HTTP is direct, rejects redirects, and requires TLS 1.2 or newer unless the operator explicitly disables certificate verification. DNS and network routing remain deployment trust. |
| B5 — firmware and video into process | HTTP bodies, signaling, RPC JSON, ICE/SDP, RTP/H.264, state, and errors | Size limits, strict decoders, timeouts, pending-request bounds, and typed result handling reduce memory, parser, and disclosure risk. A compromised appliance can still lie, withhold replies, provide hostile network candidates, or cause physical effects through accepted calls. |
| B6 — local filesystem and subprocess | Config, media files, FFmpeg executable, H.264 stdin, PNG stdout, bounded stderr | The media root and executable are trusted deployment choices. Media opens are root-confined; FFmpeg gets fixed arguments, a scrubbed environment, bounded pipes, and a deadline. FFmpeg is not sandboxed from the process account's readable environment/filesystem. |
| B7 — appliance to media origin | Caller-selected HTTP(S) URL and fetched media | The JetKVM appliance, not this Go process, performs `mount_url`. URL mounting is deny-by-default and the normalized scheme, host, and effective port must match a configured exact per-device origin before provider dispatch. The process does not resolve or pin DNS, inspect firmware redirects, classify the resolved address, or enforce appliance routing. |
| B8 — source to delivered artifact | Repository changes, Go modules, CI actions, build helpers, base images, archives, checksums, and containers | Tests, module checksums, vulnerability analysis, immutable CI Action and container-image pins, explicit Buildx selection, non-root container execution, and release checksums provide bounded evidence. They do not provide artifact signatures, provenance attestations, an SBOM, or reproducible-build comparison. |

### Assets and sensitivity

| Asset | Sensitivity and lifetime in this implementation |
|---|---|
| Device passwords, HTTP bearer token, JetKVM session cookies | Credentials. Resolved or created in process memory and transmitted only to their corresponding boundary. They are not application log fields or structured results. Memory is not guaranteed to be zeroized. |
| Typed text, key presses, mouse actions | Typed text may contain credentials, commands, or private content. It is accepted transiently and sent through HID. The server has no history database or request logger. |
| Screen image and host state | A PNG can contain any visible secret. Status can reveal versions, power/video/USB state, extension state, and virtual-media state. Values are returned to the trusted MCP caller and may remain in process/client memory. |
| Device names, URLs, origins, Wake-on-LAN targets, addresses | Private deployment metadata, though not credential values. They exist in configuration/runtime state and selected MCP inputs/results; device/target names can also appear in caller-visible tool or local debug diagnostics. Device URL path prefixes are retained, while user information, query, and fragment components are rejected without echoing their values. The validator deliberately omits private inputs from its report. |
| Local media path and bytes | The configured root and relative source identify local deployment data; bytes may contain software or secrets. Files are read transiently and uploaded to appliance storage, where appliance retention applies. |
| Media URL | May reveal infrastructure and may cause an appliance-side network fetch. User information is rejected; query strings are allowed, so operators and callers must not put secrets in them. |
| Raw firmware bodies and raw RPC parameters/results | Unreviewed, potentially private or mutation-capable data. Typed product paths discard raw error bodies and expose bounded fields. `debug rpc` deliberately transmits raw parameters and writes a raw result to its caller's stdout. |
| Source, dependencies, binaries, archives, images | Integrity-sensitive supply-chain assets. A compromise can inherit all runtime authority available to the server process. |

## Trusted-principal model and deliberate exclusions

The server enforces authentication boundaries, not human intent. Any process
that owns the stdio streams, and any client that has the configured HTTP bearer,
is treated as the same fully trusted JetKVM administrator. There is no per-tool
ACL, per-device scope, distinct user identity, consent screen, approval prompt,
rate quota, or audit identity. Tool annotations are client-facing consequence
hints; they are not authorization checks. A compromised or prompt-injected MCP
client therefore has the authority of that principal.

The following remain deliberate product exclusions and are **Not implemented**:
MCP OAuth, token passthrough, dynamic client registration, deprecated HTTP+SSE,
stateful Streamable HTTP sessions, tool-list change notifications, MCP resources,
prompts, roots, sampling, elicitation, a trust/control plane, and raw RPC as an
MCP tool. Adding one is a product-contract change, not a documentation-only
mitigation. The static tools-only capability contract is frozen by the
[tool manifest fixture](../internal/mcpserver/testdata/tool-manifest.json).

## Entry-point consequence map

Consequence classes are cumulative:

- **O — observation:** returns potentially private device or host data;
- **H — host input mutation:** changes keyboard/mouse state and can cause any
  action the attached host permits through its console;
- **P — physical/power mutation:** can boot, reset, power, or wake equipment;
- **M — media/network mutation:** can read local media, transfer/persist it on
  the appliance, change a mount, or make the appliance fetch a URL; and
- **U — uncertain raw operation:** firmware method semantics are outside the
  reviewed MCP contract.

| Entry point | Class | Direct consequence and ambiguity |
|---|---|---|
| `jetkvm_list_devices` | O | Returns configured aliases in deterministic order with configuration-derived availability flags. It does not open a device session or probe hardware, and it omits device URLs, credentials, allowed origins, media directories, and Wake-on-LAN targets. The flags describe configured preconditions, not qualified firmware capabilities. |
| `jetkvm_get_status` | O | Reads connectivity, firmware/application versions, power/video/USB/extension state, warnings, and typed redacted virtual-media state. Raw firmware fields, media URLs, paths, and filenames are not returned. It does not prove that a later mutation is safe. |
| `jetkvm_capture_screen` | O (high confidentiality) | Creates a fresh video session, decodes one frame locally, and returns a PNG to the MCP caller. Any visible host secret can be captured. |
| `jetkvm_keyboard` | H | `type_text` sends up to 4096 bytes; `press_key` sends one key plus modifiers. Either can enter credentials, execute commands, or alter host data. A failed response does not prove that no key reached the host. |
| `jetkvm_mouse` | H | Moves, clicks, or scrolls the host pointer. It can activate destructive UI actions; retrying a click can duplicate the effect. |
| `jetkvm_press_host_power_button` | P (destructive hint) | Briefly presses ATX power. Host behavior depends on firmware/OS state, and acknowledgement is not durable-effect proof. |
| `jetkvm_force_host_power_off` | P (destructive hint) | Holds ATX power and can cause immediate workload interruption or data loss. |
| `jetkvm_press_host_reset_button` | P (destructive hint) | Resets the attached host and can interrupt workloads or corrupt data. |
| `jetkvm_turn_host_dc_power_off` | P (destructive hint) | Disables configured DC output; interruption and data loss are possible. |
| `jetkvm_turn_host_dc_power_on` | P | Enables configured DC output and may boot equipment. It is state-changing even though the manifest's destructive hint is false. |
| `jetkvm_wake_host_usb` | P/H | Sends a USB wake action and may resume or boot a host. |
| `jetkvm_wake_host_lan` | P/network | Makes the appliance send a magic packet to a named configured target and optional broadcast address. The MCP caller cannot supply an arbitrary MAC/address. |
| `jetkvm_get_virtual_media_status` | O | Reads current firmware virtual-media state without requesting a mutation. It does not prove that a later mutation is safe. |
| `jetkvm_mount_virtual_media_url` | M/network (destructive/open-world hints) | Makes the appliance fetch an HTTP(S) URL from a configured exact origin and replace its current mount. Without an allowed origin the operation is unavailable. An acknowledgement does not independently prove the resulting mount state or final network destination. |
| `jetkvm_mount_virtual_media_file` | M (destructive hint) | Reads and uploads a confined local file, then replaces the current mount. Upload cleanup cannot prove what firmware persisted after a lost acknowledgement. |
| `jetkvm_unmount_virtual_media` | M (destructive hint) | Requests removal of the current mount. A lost acknowledgement leaves outcome uncertain, so its v1 annotation is non-idempotent. |
| `jetkvm_upload_virtual_media_file` | M (destructive hint) | Reads and stores a confined local file without mounting it. Appliance-side storage lifetime remains outside this process. |
| `config validate` | O (local configuration) | Strictly loads configuration and required environment bindings, then exits without FFmpeg, a listener, or device/network action. Success reveals only that configuration is valid; errors name a safe field, device alias, or rule without private paths, URL contents, or resolved values. |
| `debug rpc --method --params` | O or U | The source-reviewed methods `ping`, `getLocalVersion`, and `getActiveExtension` are read-only observations by default. Every other method requires per-invocation `--unsafe-acknowledge-risk`, may read private firmware data or mutate hardware or boot/storage state, and remains intentionally absent from MCP discovery. The acknowledgement is not a safety classification. |
| `jetkvm-mcp-validate` | O (bounded) | Lists tools, validates that safe configured-device discovery contains the selected alias, reads status and the dedicated redacted virtual-media status, and captures/fully decodes one PNG. It never calls keyboard, mouse, media mutation, power, wake, or raw RPC and emits only a sanitized pass/fail report. |

## Input, output, and configuration field walkthrough

This walkthrough is checked against the committed MCP manifest by
`TestThreatModelInputOutputFieldWalkThrough`.

### MCP inputs

| Fields | Classification and flow |
|---|---|
| `device`, `target` | Private operational identifiers. They select only configured devices/Wake-on-LAN targets and can appear in normal structured results and caller-visible tool or local debug diagnostics. They are not part of the no-log list below, but retained support/validation evidence should still omit them. |
| `operation`, `key`, `modifiers`, `button`, `x`, `y`, `dx`, `dy`, `wheel_x`, `wheel_y` | Host-control intent. Values are schema/handler bounded, then translated to HID calls. Treat traces containing them as private operational data. |
| `text` | Potential secret or command content. It is bounded and translated to HID reports; it must never be logged or persisted by the application. |
| `path`, `url` | `path` selects a relative local media path confined below `media_directory`; `url` selects an HTTP(S) URL whose origin must match `media_url_allowed_origins` before it is sent to firmware. Local paths and URL secrets must never be logged or persisted. Query-string and fragment secrets are not rejected, so they must not be supplied. |
| `mode` | Media mutation intent (`read_only` or `read_write`); omitted values default to read-only at the device layer. |
| `max_width`, `max_height` | Capture resource bounds, not private payloads. They are constrained to 3840 by 2160 and affect FFmpeg output size. |

### MCP structured outputs and content

| Fields/content | Classification and flow |
|---|---|
| `devices`, `capabilities` | Configured aliases plus boolean, configuration-derived availability flags for URL mounting, local-file mounting/upload, and Wake-on-LAN. Results are sorted and omit URLs, credentials, allowed origins, media directories, and Wake-on-LAN target details. |
| `device`, `action`, `target`, `operation`, `status` | Operational identifiers and outcome labels returned to the same trusted caller. `status: completed` means the RPC returned successfully, not that an external observer proved the physical state. |
| `connected`, `applicationVersion`, `systemVersion`, `activeExtension`, `atxPowerOn`, `dcPowerOn`, `dcVoltage`, `videoReady`, `videoWidth`, `videoHeight`, `videoFPS`, `usbState`, `usbWakeAttached`, `warnings` | Private observed appliance/host state. Returned transiently to the MCP caller; not persisted or application-logged. |
| `virtualMedia` | Optional typed object containing only `mounted`, `sourceType`, and `mode`. Unknown firmware fields and source values are discarded rather than forwarded. |
| `capturedAt`, `mimeType`, `width`, `height`, `sizeBytes` | Capture metadata. It accompanies an MCP `image` content block containing PNG bytes. Metadata is less sensitive than the image but may reveal host activity and display characteristics. |
| PNG image content | Highly private transient output. The server does not write it to disk. The validator fully decodes then clears its byte slice; ordinary Go/server/client memory has no zeroization guarantee. |
| `mounted`, `sourceType`, `mode` | Redacted virtual-media state and outcome. `sourceType` reports only `http` or `storage`; source URLs, paths, filenames, queries, fragments, and unknown firmware fields are excluded. Appliance-side media/storage lifetime is outside this process. |
| Tool error content | Caller-visible operational diagnostics. Typed execution paths return a versioned JSON taxonomy (`code`, `outcome`, `retryable`) and discard raw firmware messages/bodies. `outcome: unknown` means a mutation may have been sent and is never retryable; `not_sent` is used only when dispatch is known not to have begun. SDK input-schema failures can echo rejected client-supplied values to that same trusted caller, so all tool-error content must be treated as private MCP output rather than retained diagnostic evidence. The application does not separately log or persist it. |

### Configuration, environment, CLI, and validator fields

| Surface | Classification and control |
|---|---|
| `devices` map key | Becomes the runtime `device` identifier. Names are private deployment metadata, not credentials. |
| `limits`, `max_operations`, `max_operations_per_device`, `max_sessions`, `max_captures`, `max_decoders` | Trusted local capacity bounds, not caller identity or a policy grant. They are strict positive integers with safe hierarchy checks; exhaustion returns a non-retryable `busy`/`not_sent` result before provider dispatch and never creates a request queue. |
| `url` | Trusted administrator-selected HTTP(S) appliance endpoint. Inline user information, query, and fragment components are rejected without echoing their values; a path prefix is retained. Redirects and environment proxies are disabled. |
| `password_env`, `bearer_token_env` | Names of environment variables, not secret values. Missing/invalid names can appear in diagnostics; resolved values cannot. |
| Resolved device password and HTTP bearer | Credentials held in process memory. The password is sent only in JetKVM login JSON; the bearer is compared to incoming HTTP credentials. Neither is a log/result field. |
| `insecure_skip_verify` | Explicitly weakens appliance identity verification. It is accepted only as trusted deployment configuration and must not be described as secure transport. |
| `media_directory` | Absolute private local root. It is cleaned and used with root-confined opens; it must not appear in diagnostics or retained reports. Filesystem permissions remain external. |
| `media_url_allowed_origins` | Per-device authorization policy for appliance-fetched media. Entries are exact HTTP(S) origins made from scheme, host or IP literal, and effective port; wildcards, credentials, paths, queries, and fragments are rejected. An absent or empty list disables URL mounting. Hostnames and literal address classes are not resolved or implicitly trusted by validation. |
| `wake_on_lan`, target keys, `mac_address`, `broadcast_ip` | Trusted destinations selected by named target. They bound MCP input but can direct appliance-emitted network traffic. |
| `http`, `http.allowed_origins` / `allowed_origins` | The optional `http` map names the bearer environment and exact HTTP(S) public endpoint origins. Origins establish Host/origin admission for native reverse-proxy and same-origin browser requests. This is not a CORS grant: wildcard entries and separately hosted browser origins are unsupported, and the server emits no CORS response headers. |
| `--config`, `--http`, `--version`, `config validate` | Config path is private; bind address and derived `devel+`/`vcs.revision` provenance are deployment/build metadata; release/module version is public. Offline validation resolves configured environment names but emits no values and performs no listener or device/network action. The process writes serving notices/errors to stderr and reserves stdio stdout for MCP. |
| `debug rpc`, `--device`, `--method`, `--params`, `--unsafe-acknowledge-risk` | Local administrative escape hatch. The unsafe acknowledgement is accepted only on one invocation and cannot be persisted. Raw parameters can be visible in argv and shell history; raw results are intentionally written to stdout. Do not put secrets in parameters and do not capture either stream into durable logs. |
| Validator `--binary`, `--config`, `--device` | Private local inputs deliberately omitted from the JSON report. Child stderr and detailed failures are discarded; output contains only check names and capture dimensions/size. |

## Privacy and diagnostic policy

Production application logs, stderr diagnostics, generated reports, temporary
files, and other application-owned durable state must never contain:

- typed `text` or reconstructed key input;
- screenshot/image bytes or decoded pixels;
- bearer tokens, passwords, session cookies, authorization headers, or other
  credentials;
- raw firmware HTTP bodies, raw status payloads, or firmware error messages;
- configuration paths, media roots, or caller-supplied local media paths;
- URLs containing user information, query-string secrets, fragments used as
  secrets, or other URL credentials; or
- raw RPC parameters or results.

This rule does not prevent the transient transfer required by an explicit
operation: credentials cross their authentication boundary, HID text reaches
the appliance, media bytes are uploaded, a PNG is returned to the MCP caller,
SDK input-schema failures can echo rejected client-supplied values in the tool
error returned to that caller, except that source-bearing virtual-media tools
validate locally and return value-free schema errors so URLs and paths are not
echoed. `debug rpc` returns raw output to its invoking terminal. Those
destinations can persist data independently; the caller/operator is responsible
for client history, shell history, process inspection, proxy logs, terminal
capture, crash/core dumps, appliance storage, media-origin logs, and MCP-host
telemetry. There is no application request/payload/audit logger and no local
database. That reduces disclosure but also means this single-principal product
cannot provide user-attributed forensic audit records.

The secret-sentinel verification covers strict-config errors including short
malformed scalar values, resolved secret handling, missing private configuration
paths, top-level diagnostic wording, the validator's sanitized report, and
operation telemetry over stdio and loopback HTTP. It complements review; it
cannot prove that every future dependency or client avoids retention.

## Threat-to-control traceability register

Labels have strict meanings:

- **Implemented control** — enforced in current code and covered by repository
  evidence named in the row;
- **Deployment requirement** — necessary operator/proxy/network behavior that
  this binary cannot enforce; and
- **Not implemented** — residual or future control, not a current claim.

| ID | Threat and consequence | Implemented control and evidence | Deployment requirement / residual risk |
|---|---|---|---|
| T-01 | Unauthorized MCP calls or a confused trusted deputy can exercise full console and power authority. | **Implemented control:** stdio inherits local process ownership; non-loopback HTTP refuses startup without a bearer; `/mcp` accepts exactly one syntactically valid Bearer credential and compares its token in constant time; all HTTP calls are stateless and share no server session. See [transport.go](../internal/mcpserver/transport.go), [transport_test.go](../internal/mcpserver/transport_test.go), and [http_integration_test.go](../cmd/jetkvm-mcp/http_integration_test.go). | **Deployment requirement:** keep direct HTTP on loopback or put it behind TLS and protected routing; isolate environment/stdio ownership and rotate a disclosed bearer. **Not implemented:** user identity, scopes, OAuth, per-device/tool ACLs, consent, and token audience claims. One bearer grants all tools. |
| T-02 | DNS rebinding or forged `Host`/browser `Origin` could reach a loopback or reverse-proxied server. | **Implemented control:** malformed/unconfigured public Hosts are rejected; a present browser Origin must be singular, nonempty, parseable, and match the request Host and configured scheme/authority; loopback requires the same scheme. Invalid Origin is rejected before bearer authentication, method handling, or MCP dispatch. Same-origin OPTIONS remains subject to any configured bearer and then receives 405; foreign or invalid preflight receives 403, and no CORS grant headers are emitted. See [transport.go](../internal/mcpserver/transport.go), [origin.go](../internal/httporigin/origin.go), [origin_test.go](../internal/mcpserver/origin_test.go), and [ADR 0007](adr/0007-same-origin-browser-http.md). | **Deployment requirement:** preserve the public Host at the TLS proxy and configure exact origins. Native clients may omit Origin, so Host and bearer remain decisive. A separately hosted browser origin is unsupported. **Not implemented:** cross-origin browser CORS, trusted-forwarded-header interpretation, or proxy discovery. Proxy/DNS compromise remains outside process control. |
| T-03 | SSRF through virtual-media URLs can make the appliance reach internal services or attacker content. | **Implemented control:** URL mounting is unavailable without a per-device allowlist; configured entries and mount URLs accept only HTTP(S), reject user information, and compare exact normalized scheme, host, and effective port before any provider session. Wildcards are rejected; unconfigured DNS names and IPv4/IPv6 loopback, private, or link-local literals do not dispatch. Paths, queries, and fragments are not authorization selectors. The tool identifies the firmware fetch and advertises `openWorldHint=true`. See [virtual_media.go](../internal/jetkvm/virtual_media.go), [virtual_media_test.go](../internal/jetkvm/virtual_media_test.go), and [0006-virtual-media-url-origin-boundary.md](adr/0006-virtual-media-url-origin-boundary.md). | **Deployment requirement:** use a dedicated trusted origin, keep secrets out of URLs, segment the appliance network, and restrict its media-fetch egress. **Not implemented:** application DNS resolution or IP pinning, DNS rebinding prevention, implicit private/link-local deny rules, firmware redirect enforcement, final-destination verification, content scanning, or application-enforced egress filtering. An explicitly configured literal is allowed, and a configured hostname can later resolve or redirect to an unintended address. |
| T-04 | Credentials or session material can be disclosed, replayed, or intercepted. | **Implemented control:** YAML accepts environment-variable references, rejects inline fields/user information, bounds config and login bodies, stores cookies only in an in-memory jar, avoids raw response bodies in errors, and tests sentinel absence. See [config.go](../internal/config/config.go), [auth.go](../internal/jetkvm/auth.go), [privacy_test.go](../internal/config/privacy_test.go), and [auth_test.go](../internal/jetkvm/auth_test.go). | **Deployment requirement:** restrict config/environment/process access; use HTTPS with a trusted appliance certificate and TLS at the MCP proxy; protect shell/core/crash data. `insecure_skip_verify` and plain HTTP surrender appliance confidentiality/identity. **Not implemented:** external secret manager, memory locking/zeroization, automatic rotation, or mTLS. |
| T-05 | Privacy disclosure through logs, diagnostics, MCP output, validation evidence, or client retention can expose typed text, screens, paths, firmware, URLs, or raw RPC. | **Implemented control:** stdout/stderr separation, no request logger/database, bounded typed outputs, value-free YAML decode errors, value-free source-bearing media schema errors, discarded firmware error detail, sanitized validator JSON, discarded child stderr, image-buffer clearing in the validator, and private-value/path regression tests. See [main.go](../cmd/jetkvm-mcp/main.go), [main.go](../cmd/jetkvm-mcp-validate/main.go), [main_test.go](../cmd/jetkvm-mcp-validate/main_test.go), [typed_results_test.go](../internal/mcpserver/typed_results_test.go), and [privacy_test.go](../internal/config/privacy_test.go). | **Deployment requirement:** configure MCP hosts, proxies, terminals, shells, and crash handling not to retain listed payloads; treat caller-visible tool errors as private and sanitize issue/support evidence. **Not implemented:** centralized redaction middleware, encrypted audit storage, client-side retention control, or full memory zeroization. SDK-managed schemas outside the protected source-bearing media surface can echo rejected arguments, and explicit MCP/debug outputs remain visible to their trusted caller. |
| T-06 | CPU, memory, file, subprocess, connection, or appliance exhaustion can deny service. | **Implemented control:** strict bounded global/per-device operation, live-session, capture, and decoder admission; context-cancelable per-device mutation serialization; 1 MiB configuration, MCP request-body, and HTTP-response limits; 16 KiB RPC frames; 64 KiB signaling frames; 64 pending RPCs; bounded H.264/PNG/FFmpeg buffers; decoder and network deadlines; fixed image dimensions; independent five-second HTTP header, 15-second MCP body-read, and 60-second idle limits; and cancellation propagation. Capacity exhaustion is an immediate `busy`/`not_sent` result, not a queue. See [manager.go](../internal/jetkvm/manager.go), [transport.go](../internal/mcpserver/transport.go), [main.go](../cmd/jetkvm-mcp/main.go), [http_integration_test.go](../cmd/jetkvm-mcp/http_integration_test.go), [rpc_session.go](../internal/jetkvm/rpc_session.go), [rpc_codec.go](../internal/jetkvm/rpc_codec.go), [video.go](../internal/jetkvm/video.go), [protocol_version.go](../internal/mcpserver/protocol_version.go), and [decoder_ffmpeg.go](../internal/jetkvm/decoder_ffmpeg.go). | **Deployment requirement:** restrict client connection rates and concurrency and constrain process CPU, memory, PIDs, files, and network externally; size media roots deliberately. **Not implemented:** per-principal quotas, policy/grant engines, distributed queues, circuit breakers, or public-edge rate limiting. Repeated admitted captures and 64 GiB uploads remain expensive. |
| T-07 | Malformed or malicious firmware/network data can exploit parsers, leak private error text, or corrupt state. | **Implemented control:** strict/duplicate-aware RPC decoding, generic firmware RPC errors, bounded HTTP/signaling/RPC/video data, H.264 reassembly limits, PNG full decode/geometry validation, typed state validation, and a minimal redacted virtual-media projection. See [rpc_codec.go](../internal/jetkvm/rpc_codec.go), [session_protocol_test.go](../internal/jetkvm/session_protocol_test.go), [virtual_media.go](../internal/jetkvm/virtual_media.go), and [decoder_test.go](../internal/jetkvm/decoder_test.go). | **Deployment requirement:** treat the appliance/firmware as privileged and keep it updated/segmented. **Not implemented:** firmware attestation or a malicious-device sandbox. Explicit local raw-RPC results can still contain attacker-controlled data returned to the invoking terminal. |
| T-08 | An uncertain mutation and retry can duplicate HID/power/media effects or misreport a lost acknowledgement. | **Implemented control:** reads are annotated safe and idempotent; every mutation is annotated non-idempotent; keyboard and mouse are also destructive; MCP cancellation reaches handlers; RPC and upload paths classify definitely-not-sent, confirmed-failed, and possibly-sent outcomes; unknown mutation outcomes are non-retryable; RPC calls have deadlines; and media operations are serialized/cleaned conservatively. See [errors.go](../internal/mcpserver/errors.go), [rpc_session.go](../internal/jetkvm/rpc_session.go), [server.go](../internal/mcpserver/server.go), [error_result_test.go](../internal/mcpserver/error_result_test.go), and [session_test.go](../internal/jetkvm/session_test.go). | **Deployment requirement:** obtain human authorization before destructive/non-idempotent calls, avoid automatic retries, and reconcile with independent status where possible. **Not implemented:** approval workflow, idempotency keys, durable operation journal, transaction protocol, or guaranteed read-after-write proof. Cancellation/lost response cannot undo a physical effect. |
| T-09 | Path traversal, symlink escape, file replacement, stale partial resume, or oversized local media can disclose/alter files or mount the wrong image. | **Implemented control:** absolute configured root, `os.OpenRoot` confinement, regular-file/64 GiB checks, per-device locking, fresh offset-zero upload, pre/upload/post hashes, reopen/SameFile checks, and best-effort artifact cleanup. See [virtual_media.go](../internal/jetkvm/virtual_media.go), [virtual_media_test.go](../internal/jetkvm/virtual_media_test.go), and [upload_test.go](../internal/jetkvm/upload_test.go). | **Deployment requirement:** mount only a least-privilege read-only media directory and protect it from concurrent writers; account for appliance-side copies. **Not implemented:** content trust/signature scanning, encrypted media, lower configurable byte quotas, or proof that firmware deleted every ambiguous partial artifact. |
| T-10 | A malicious H.264 stream, compromised FFmpeg binary, or decoder vulnerability can execute code or exhaust the host. | **Implemented control:** FFmpeg path resolves to an absolute executable regular file; invocation uses fixed non-shell arguments, scrubbed environment, pipe-only protocol whitelist, one thread/frame, allocation/pixel/input/output/stderr caps, deadline, and PNG validation. The container runs UID/GID 10001. See [decoder_ffmpeg.go](../internal/jetkvm/decoder_ffmpeg.go), [decoder_test.go](../internal/jetkvm/decoder_test.go), and [Dockerfile](../Dockerfile). | **Deployment requirement:** install a trusted patched FFmpeg and constrain the process/container filesystem, capabilities, and resources. **Not implemented:** decoder IPC sandbox/sidecar, seccomp profile, executable digest verification, or protection against replacing the executable after startup validation. |
| T-11 | Device endpoint, DNS/routing, TLS downgrade, WebRTC ICE, or signaling compromise can intercept credentials/data or make unexpected network connections. | **Implemented control:** configured HTTP(S) endpoints reject user information; requests bypass environment proxies and redirects; TLS minimum is 1.2 by default; signaling and RPC are bounded and connection/request timeouts apply. See [provider.go](../internal/jetkvm/provider.go), [provider_test.go](../internal/jetkvm/provider_test.go), [signaling.go](../internal/jetkvm/signaling.go), and [auth_test.go](../internal/jetkvm/auth_test.go). | **Deployment requirement:** trusted DNS/routing/certificates and network segmentation must constrain the configured appliance and its WebRTC candidates. **Not implemented:** endpoint IP allowlists/pinning, mTLS, certificate pinning, ICE candidate policy, TURN allowlists, or appliance attestation. Explicit TLS verification bypass remains high risk. |
| T-12 | Build compromise through source, dependency, CI action, base image, or release control can ship a binary with all server authority. | **Implemented control:** `go.sum`/`go mod verify`, race/tests/vet/vulnerability analysis, four-target builds, full-SHA CI Actions, a checksum-verified Buildx binary, digest-pinned QEMU, BuildKit, frontend, builder, and runtime images, a minimal non-root runtime, GoReleaser checksums, and reviewed static/release tool contracts. See [ci.yml](../.github/workflows/ci.yml), [Dockerfile](../Dockerfile), [go.mod](../go.mod), [tools/go.mod](../tools/go.mod), [.goreleaser.yaml](../.goreleaser.yaml), [dependency-policy.md](dependency-policy.md), and [manifest_contract_test.go](../internal/mcpserver/manifest_contract_test.go). | **Deployment requirement:** consume artifacts from the canonical release record, verify checksums over a trusted channel, review dependency/action updates, and protect GitHub/release credentials. **Not implemented:** signed artifacts, signed provenance, SBOM publication, or reproducible-build comparison. Checksums and pinning alone do not defend against a compromised source or publisher. |
| T-13 | Raw RPC bypass can expose private firmware data or invoke undocumented/destructive methods. | **Implemented control:** it is a separate local CLI subcommand absent from MCP tools. The exact source-reviewed default set is `ping`, `getLocalVersion`, and `getActiveExtension`; all other methods fail before session dispatch without per-invocation `--unsafe-acknowledge-risk`. Method/object inputs and RPC frames remain bounded, and wire errors are generic. See [debug.go](../internal/jetkvm/debug.go), [debug_test.go](../internal/jetkvm/debug_test.go), and [main_test.go](../cmd/jetkvm-mcp/main_test.go). | **Deployment requirement:** restrict local command execution, keep parameters/results out of shell history and logs, and never expose raw RPC as a remote service. **Not implemented:** semantic result redaction, method consequence detection, or proof that an acknowledged method is safe. The unsafe path's mutation class remains intentionally uncertain. |

## Deployment baseline

A deployment consistent with this model:

1. uses stdio under one trusted MCP host, or binds HTTP to loopback behind a TLS
   reverse proxy; any non-loopback listener has a strong bearer and exact public
   Host/origin configuration;
2. runs the process as a dedicated unprivileged identity with protected
   configuration/environment, no core dumps where practical, a read-only
   least-privilege media mount, and explicit CPU/memory/process/network limits;
3. uses HTTPS to a known JetKVM certificate, leaving
   `insecure_skip_verify` false, and segments both the appliance control plane and
   appliance-side media-fetch egress;
4. supplies media URLs only from a trusted allowlisted origin without URL
   credentials/query secrets, and protects local media from concurrent writers;
5. uses a trusted patched FFmpeg and verifies canonical release checksums; and
6. disables automatic retries for non-idempotent/destructive operations and
   requires human review before keyboard, mouse, power, reset, writable media,
   or raw RPC actions; and
7. sends SIGTERM or SIGINT for shutdown and permits the HTTP process at least
   five seconds to drain before an external force-kill.

## Source review and authority

Primary protocol authority:

- [MCP Security Best Practices, revision 2026-07-28](https://modelcontextprotocol.io/docs/2026-07-28/tutorials/security/security_best_practices)
  informs confused-deputy, credential, token, SSRF, and least-privilege analysis.
  This server does not implement MCP OAuth and does not pass its static bearer
  through to JetKVM.
- [MCP Streamable HTTP transport, revision 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http)
  requires Origin validation to resist DNS rebinding and recommends local-only
  binding where appropriate. The Host/origin and loopback controls implement
  that boundary while retaining stateless JSON responses.
- [MCP stdio transport, revision 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio)
  defines stdout as protocol traffic and stderr as the optional logging stream.

Applicable application-security guidance:

- [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
  favors strict destination allowlists and careful DNS/IP/redirect handling.
  The application enforces a strict configured origin allowlist, but firmware-side
  DNS resolution, redirects, and routing do not satisfy the stronger address and
  redirect controls; those remain residual and deployment risks rather than
  claimed application mitigation.
- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)
  identifies passwords, access tokens, session identifiers, sensitive personal
  data, and unsafe input as data to exclude, mask, sanitize, or classify. The
  no-log/persist policy above is intentionally stricter for console imagery,
  typed text, paths, firmware payloads, media URLs, and raw RPC.
- [OWASP Denial of Service Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Denial_of_Service_Cheat_Sheet.html)
  supports layered resource limits; current per-parser/process limits do not
  replace deployment quotas or rate limiting.

Secondary industry context:

- [Model Context Protocol Security](https://modelcontextprotocol-security.io/)
  is a Cloud Security Alliance community project. Its material is useful
  secondary threat context for excessive permission, tool integrity, supply
  chain, and operational hardening, but it is not MCP protocol authority. Where
  secondary guidance and the official MCP specification differ, the official
  specification governs protocol claims.

## Verification checklist

- **Input/output/config field walk-through:** the consequence and field tables
  were compared with strict YAML tags, CLI flags, and every current input/output
  property in the committed tool manifest. The automated walk-through fails if
  a manifest tool lacks a consequence row or a schema field lacks classification.
- **Secret-sentinel redaction review:** private values exercised resolved
  credentials, rejected inline credentials, short malformed YAML scalars,
  missing private config paths, source-bearing media schema failures, and
  validator/report output. The tests assert that these values are absent from
  application diagnostics/reports and media tool errors while proving the guard
  detects a leak. A separate executable check records that SDK-managed input
  schemas outside this protected media surface can echo rejected values to the
  trusted MCP caller.
- **Threat-to-control traceability check:** every T-01 through T-13 row names
  implemented code/test evidence, deployment obligations, and missing controls;
  the document test rejects missing rows or classification markers.
- **No general policy redesign:** the accepted per-device exact-origin boundary
  adds no OAuth, stateful HTTP, dynamic grant service, per-file approval database,
  raw-RPC tool, or wildcard authorization. Such controls require a separately
  accepted behavior change.
