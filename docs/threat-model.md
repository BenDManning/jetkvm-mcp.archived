# JetKVM MCP threat model and privacy data flows

Status: maintained.

Review basis: MCP revision `2026-07-28`; reviewed 2026-08-12.

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
    B[Forgejo / CI / release inputs]

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
| B1 — MCP client to local process | Tool names, input fields, images, structured results, and tool errors | Stdio relies on operating-system process launch and stream ownership. There is no application authentication on stdio. The launching client is the administrative principal. |
| B2 — MCP client/proxy to HTTP server | Bearer token, `Host`, `Origin`, and stateless JSON-RPC POSTs | Loopback is the recommended direct boundary. Non-loopback bind requires a bearer token; public Hosts/origins require exact configuration. The server has one bearer principal, no end-user identity, scopes, OAuth grants, or stateful MCP sessions. TLS is outside the process. |
| B3 — operator state to process | YAML, environment-variable names and values, CLI arguments, media root, device URLs, Wake-on-LAN targets | These are trusted administration inputs. YAML is strict and bounded. Credentials are resolved from the environment rather than accepted inline. File permissions, environment isolation, and shell-history handling are deployment duties. |
| B4 — server to configured JetKVM | Password login, cookie, HTTP, WebSocket signaling, WebRTC data/video, RPC calls, and upload bytes | A configured endpoint is trusted as the selected appliance, but responses are treated as malformed until bounded and decoded. Device HTTP is direct, rejects redirects, and requires TLS 1.2 or newer unless the operator explicitly disables certificate verification. DNS and network routing remain deployment trust. |
| B5 — firmware and video into process | HTTP bodies, signaling, RPC JSON, ICE/SDP, RTP/H.264, state, and errors | Size limits, strict decoders, timeouts, pending-request bounds, and typed result handling reduce memory, parser, and disclosure risk. A compromised appliance can still lie, withhold replies, provide hostile network candidates, or cause physical effects through accepted calls. |
| B6 — local filesystem and subprocess | Config, media files, FFmpeg executable, H.264 stdin, PNG stdout, bounded stderr | The media root and executable are trusted deployment choices. Media opens are root-confined; FFmpeg gets fixed arguments, a scrubbed environment, bounded pipes, and a deadline. FFmpeg is not sandboxed from the process account's readable environment/filesystem. |
| B7 — appliance to media origin | Caller-selected HTTP(S) URL and fetched media | The JetKVM appliance, not this Go process, performs `mount_url`. Scheme and inline user information are rejected, but host/IP, DNS answers, redirects performed by firmware, and destination network class are not controlled by this process. |
| B8 — source to delivered artifact | Repository changes, Go modules, CI actions, base images, archives, checksums, and containers | Tests, module checksums, pinned base-image digests, non-root container execution, and release checksums provide bounded evidence. They do not provide artifact signatures, provenance attestations, an SBOM, or immutable pins for every CI action. |

### Assets and sensitivity

| Asset | Sensitivity and lifetime in this implementation |
|---|---|
| Device passwords, HTTP bearer token, JetKVM session cookies | Credentials. Resolved or created in process memory and transmitted only to their corresponding boundary. They are not application log fields or structured results. Memory is not guaranteed to be zeroized. |
| Typed text, key presses, mouse actions | Typed text may contain credentials, commands, or private content. It is accepted transiently and sent through HID. The server has no history database or request logger. |
| Screen image and host state | A PNG can contain any visible secret. Status can reveal versions, power/video/USB state, extension state, and virtual-media state. Values are returned to the trusted MCP caller and may remain in process/client memory. |
| Device names, URLs, origins, Wake-on-LAN targets, addresses | Private deployment metadata, though not credential values. They exist in configuration/runtime state and selected MCP inputs/results; device/target names can also appear in caller-visible tool or local debug diagnostics. The validator deliberately omits them from its report. |
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
| `jetkvm_mount_virtual_media_url` | M/network (destructive/open-world hints) | Makes the appliance fetch an HTTP(S) URL and replace its current mount. An acknowledgement does not independently prove the resulting mount state. |
| `jetkvm_mount_virtual_media_file` | M (destructive hint) | Reads and uploads a confined local file, then replaces the current mount. Upload cleanup cannot prove what firmware persisted after a lost acknowledgement. |
| `jetkvm_unmount_virtual_media` | M (destructive/idempotent hints) | Requests removal of the current mount. Repetition is intended to converge, but a lost acknowledgement leaves outcome uncertain. |
| `jetkvm_upload_virtual_media_file` | M (destructive hint) | Reads and stores a confined local file without mounting it. Appliance-side storage lifetime remains outside this process. |
| `jetkvm_virtual_media` | O or M (deprecated, destructive hint) | Compatibility surface for `status`, `mount_url`, `mount_file`, `unmount`, and `upload`. Its whole-tool annotations cannot express per-operation consequences; clients must migrate to the one-purpose tools above. |
| `debug rpc --method --params` | U | A local CLI caller can invoke any syntactically accepted JetKVM RPC method. It may read private firmware data or mutate hardware. It is intentionally absent from MCP discovery and must not be exposed to untrusted automation. |
| `jetkvm-mcp-validate` | O (bounded) | Lists tools, reads status, and captures/fully decodes one PNG. It never calls keyboard, mouse, media, power, wake, or raw RPC and emits only a sanitized pass/fail report. |

## Input, output, and configuration field walkthrough

This walkthrough is checked against the committed MCP manifest by
`TestThreatModelInputOutputFieldWalkThrough`.

### MCP inputs

| Fields | Classification and flow |
|---|---|
| `device`, `target` | Private operational identifiers. They select only configured devices/Wake-on-LAN targets and can appear in normal structured results and caller-visible tool or local debug diagnostics. They are not part of the no-log list below, but retained support/validation evidence should still omit them. |
| `operation`, `key`, `modifiers`, `button`, `x`, `y`, `dx`, `dy`, `wheel_x`, `wheel_y` | Host-control intent. Values are schema/handler bounded, then translated to HID or deprecated compatibility-media calls. Treat traces containing them as private operational data. |
| `text` | Potential secret or command content. It is bounded and translated to HID reports; it must never be logged or persisted by the application. |
| `source`, `path`, `url` | `source` is the deprecated compatibility input. `path` selects a relative local media path confined below `media_directory`; `url` selects an HTTP(S) URL sent to firmware. Local paths and URL secrets must never be logged or persisted. Query-string secrets are not rejected, so they must not be supplied. |
| `mode` | Media mutation intent (`read_only` or `read_write`); omitted values default to read-only at the device layer. |
| `max_width`, `max_height` | Capture resource bounds, not private payloads. They are constrained to 3840 by 2160 and affect FFmpeg output size. |

### MCP structured outputs and content

| Fields/content | Classification and flow |
|---|---|
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
| `url` | Trusted administrator-selected HTTP(S) appliance endpoint. Inline user information is rejected; redirects and environment proxies are disabled. A path prefix is retained for appliance endpoints. |
| `password_env`, `bearer_token_env` | Names of environment variables, not secret values. Missing/invalid names can appear in diagnostics; resolved values cannot. |
| Resolved device password and HTTP bearer | Credentials held in process memory. The password is sent only in JetKVM login JSON; the bearer is compared to incoming HTTP credentials. Neither is a log/result field. |
| `insecure_skip_verify` | Explicitly weakens appliance identity verification. It is accepted only as trusted deployment configuration and must not be described as secure transport. |
| `media_directory` | Absolute private local root. It is cleaned and used with root-confined opens; it must not appear in diagnostics or retained reports. Filesystem permissions remain external. |
| `wake_on_lan`, target keys, `mac_address`, `broadcast_ip` | Trusted destinations selected by named target. They bound MCP input but can direct appliance-emitted network traffic. |
| `http`, `http.allowed_origins` / `allowed_origins` | The optional `http` map names the bearer environment and exact HTTP(S) public origins. Origins also establish accepted public `Host` authorities. They are routing/authentication policy, not CORS response configuration. |
| `--config`, `--http`, `--version` | Config path is private; bind address is deployment metadata; version is public. The process writes serving notices/errors to stderr and reserves stdio stdout for MCP. |
| `debug rpc`, `--device`, `--method`, `--params` | Local administrative escape hatch. Raw parameters can be visible in argv and shell history; raw results are intentionally written to stdout. Do not put secrets in parameters and do not capture either stream into durable logs. |
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
paths, top-level diagnostic wording, and the validator's sanitized report. It
complements review; it cannot prove that every future dependency or client avoids
retention.

## Threat-to-control traceability register

Labels have strict meanings:

- **Implemented control** — enforced in current code and covered by repository
  evidence named in the row;
- **Deployment requirement** — necessary operator/proxy/network behavior that
  this binary cannot enforce; and
- **Not implemented** — residual or future control, not a current claim.

| ID | Threat and consequence | Implemented control and evidence | Deployment requirement / residual risk |
|---|---|---|---|
| T-01 | Unauthorized MCP calls or a confused trusted deputy can exercise full console and power authority. | **Implemented control:** stdio inherits local process ownership; non-loopback HTTP refuses startup without a bearer; bearer comparison is constant-time; all HTTP calls are stateless and share no server session. See [transport.go](../internal/mcpserver/transport.go), [transport_test.go](../internal/mcpserver/transport_test.go), and [main_test.go](../cmd/jetkvm-mcp/main_test.go). | **Deployment requirement:** keep direct HTTP on loopback or put it behind TLS and protected routing; isolate environment/stdio ownership and rotate a disclosed bearer. **Not implemented:** user identity, scopes, OAuth, per-device/tool ACLs, consent, and token audience claims. One bearer grants all tools. |
| T-02 | DNS rebinding or forged `Host`/browser `Origin` could reach a loopback or reverse-proxied server. | **Implemented control:** malformed/unconfigured public Hosts are rejected; browser Origin must match Host and configured scheme/authority; loopback requires same scheme. See [transport.go](../internal/mcpserver/transport.go), [origin.go](../internal/httporigin/origin.go), and [origin_test.go](../internal/mcpserver/origin_test.go). | **Deployment requirement:** preserve the public Host at the proxy and configure exact origins. Native clients may omit Origin, so Host and bearer remain decisive. **Not implemented:** trusted-forwarded-header interpretation or proxy discovery. Proxy/DNS compromise remains outside process control. |
| T-03 | SSRF through virtual-media URLs can make the appliance reach internal services or attacker content. | **Implemented control:** `mount_url` accepts only HTTP(S), rejects URL user information, clearly delegates the fetch to firmware, and advertises `openWorldHint=true`; configured device HTTP separately disables redirects/proxies. See [virtual_media.go](../internal/jetkvm/virtual_media.go) and [virtual_media_test.go](../internal/jetkvm/virtual_media_test.go). | **Deployment requirement:** use a dedicated, trusted media origin reachable from a segmented appliance network and never put secrets in URL queries. **Not implemented:** destination allowlists, DNS/IP pinning, private/link-local address rejection, firmware redirect enforcement, content scanning, or egress filtering. The annotation identifies rather than removes this external fetch. |
| T-04 | Credentials or session material can be disclosed, replayed, or intercepted. | **Implemented control:** YAML accepts environment-variable references, rejects inline fields/user information, bounds config and login bodies, stores cookies only in an in-memory jar, avoids raw response bodies in errors, and tests sentinel absence. See [config.go](../internal/config/config.go), [auth.go](../internal/jetkvm/auth.go), [privacy_test.go](../internal/config/privacy_test.go), and [auth_test.go](../internal/jetkvm/auth_test.go). | **Deployment requirement:** restrict config/environment/process access; use HTTPS with a trusted appliance certificate and TLS at the MCP proxy; protect shell/core/crash data. `insecure_skip_verify` and plain HTTP surrender appliance confidentiality/identity. **Not implemented:** external secret manager, memory locking/zeroization, automatic rotation, or mTLS. |
| T-05 | Privacy disclosure through logs, diagnostics, MCP output, validation evidence, or client retention can expose typed text, screens, paths, firmware, URLs, or raw RPC. | **Implemented control:** stdout/stderr separation, no request logger/database, bounded typed outputs, value-free YAML decode errors, value-free source-bearing media schema errors, discarded firmware error detail, sanitized validator JSON, discarded child stderr, image-buffer clearing in the validator, and private-value/path regression tests. See [main.go](../cmd/jetkvm-mcp/main.go), [main.go](../cmd/jetkvm-mcp-validate/main.go), [main_test.go](../cmd/jetkvm-mcp-validate/main_test.go), [typed_results_test.go](../internal/mcpserver/typed_results_test.go), and [privacy_test.go](../internal/config/privacy_test.go). | **Deployment requirement:** configure MCP hosts, proxies, terminals, shells, and crash handling not to retain listed payloads; treat caller-visible tool errors as private and sanitize issue/support evidence. **Not implemented:** centralized redaction middleware, encrypted audit storage, client-side retention control, or full memory zeroization. SDK-managed schemas outside the protected source-bearing media surface can echo rejected arguments, and explicit MCP/debug outputs remain visible to their trusted caller. |
| T-06 | CPU, memory, file, subprocess, connection, or appliance exhaustion can deny service. | **Implemented control:** 1 MiB config/HTTP response limits, 16 KiB RPC frames, 64 KiB signaling frames, 64 pending RPCs, bounded H.264/PNG/FFmpeg buffers, decoder and network deadlines, fixed image dimensions, serialized per-device media operations, HTTP header/read-header/idle limits, and cancellation propagation. See [rpc_session.go](../internal/jetkvm/rpc_session.go), [rpc_codec.go](../internal/jetkvm/rpc_codec.go), [video.go](../internal/jetkvm/video.go), [decoder_ffmpeg.go](../internal/jetkvm/decoder_ffmpeg.go), and [cancellation_test.go](../internal/mcpserver/cancellation_test.go). | **Deployment requirement:** restrict client concurrency and process resources externally; size media roots deliberately. **Not implemented:** MCP request-body cap at this wrapper, rate limiting, global concurrency/CPU/memory quotas, per-principal quotas, circuit breakers, or backpressure across all stateless HTTP requests. Repeated capture and 64 GiB uploads remain expensive. |
| T-07 | Malformed or malicious firmware/network data can exploit parsers, leak private error text, or corrupt state. | **Implemented control:** strict/duplicate-aware RPC decoding, generic firmware RPC errors, bounded HTTP/signaling/RPC/video data, H.264 reassembly limits, PNG full decode/geometry validation, typed state validation, and a minimal redacted virtual-media projection. See [rpc_codec.go](../internal/jetkvm/rpc_codec.go), [session_protocol_test.go](../internal/jetkvm/session_protocol_test.go), [virtual_media.go](../internal/jetkvm/virtual_media.go), and [decoder_test.go](../internal/jetkvm/decoder_test.go). | **Deployment requirement:** treat the appliance/firmware as privileged and keep it updated/segmented. **Not implemented:** firmware attestation or a malicious-device sandbox. Explicit local raw-RPC results can still contain attacker-controlled data returned to the invoking terminal. |
| T-08 | An uncertain mutation and retry can duplicate HID/power/media effects or misreport a lost acknowledgement. | **Implemented control:** annotations distinguish read-only/destructive/idempotent intent; MCP cancellation reaches handlers; RPC and upload paths classify definitely-not-sent, confirmed-failed, and possibly-sent outcomes; unknown mutation outcomes are non-retryable; RPC calls have deadlines; and media operations are serialized/cleaned conservatively. See [errors.go](../internal/mcpserver/errors.go), [rpc_session.go](../internal/jetkvm/rpc_session.go), [server.go](../internal/mcpserver/server.go), [error_result_test.go](../internal/mcpserver/error_result_test.go), and [session_test.go](../internal/jetkvm/session_test.go). | **Deployment requirement:** obtain human authorization before destructive/non-idempotent calls, avoid automatic retries, and reconcile with independent status where possible. **Not implemented:** approval workflow, idempotency keys, durable operation journal, transaction protocol, or guaranteed read-after-write proof. Cancellation/lost response cannot undo a physical effect. |
| T-09 | Path traversal, symlink escape, file replacement, stale partial resume, or oversized local media can disclose/alter files or mount the wrong image. | **Implemented control:** absolute configured root, `os.OpenRoot` confinement, regular-file/64 GiB checks, per-device locking, fresh offset-zero upload, pre/upload/post hashes, reopen/SameFile checks, and best-effort artifact cleanup. See [virtual_media.go](../internal/jetkvm/virtual_media.go), [virtual_media_test.go](../internal/jetkvm/virtual_media_test.go), and [upload_test.go](../internal/jetkvm/upload_test.go). | **Deployment requirement:** mount only a least-privilege read-only media directory and protect it from concurrent writers; account for appliance-side copies. **Not implemented:** content trust/signature scanning, encrypted media, lower configurable byte quotas, or proof that firmware deleted every ambiguous partial artifact. |
| T-10 | A malicious H.264 stream, compromised FFmpeg binary, or decoder vulnerability can execute code or exhaust the host. | **Implemented control:** FFmpeg path resolves to an absolute executable regular file; invocation uses fixed non-shell arguments, scrubbed environment, pipe-only protocol whitelist, one thread/frame, allocation/pixel/input/output/stderr caps, deadline, and PNG validation. The container runs UID/GID 10001. See [decoder_ffmpeg.go](../internal/jetkvm/decoder_ffmpeg.go), [decoder_test.go](../internal/jetkvm/decoder_test.go), and [Dockerfile](../Dockerfile). | **Deployment requirement:** install a trusted patched FFmpeg and constrain the process/container filesystem, capabilities, and resources. **Not implemented:** decoder IPC sandbox/sidecar, seccomp profile, executable digest verification, or protection against replacing the executable after startup validation. |
| T-11 | Device endpoint, DNS/routing, TLS downgrade, WebRTC ICE, or signaling compromise can intercept credentials/data or make unexpected network connections. | **Implemented control:** configured HTTP(S) endpoints reject user information; requests bypass environment proxies and redirects; TLS minimum is 1.2 by default; signaling and RPC are bounded and connection/request timeouts apply. See [provider.go](../internal/jetkvm/provider.go), [provider_test.go](../internal/jetkvm/provider_test.go), [signaling.go](../internal/jetkvm/signaling.go), and [auth_test.go](../internal/jetkvm/auth_test.go). | **Deployment requirement:** trusted DNS/routing/certificates and network segmentation must constrain the configured appliance and its WebRTC candidates. **Not implemented:** endpoint IP allowlists/pinning, mTLS, certificate pinning, ICE candidate policy, TURN allowlists, or appliance attestation. Explicit TLS verification bypass remains high risk. |
| T-12 | Build compromise through source, dependency, CI action, base image, or release control can ship a binary with all server authority. | **Implemented control:** `go.sum`/`go mod verify`, race/tests/vet, four-target builds, base-image digest pins, minimal non-root runtime, GoReleaser checksums, and the reviewed static tool contract. See [ci.yml](../.forgejo/workflows/ci.yml), [Dockerfile](../Dockerfile), [go.mod](../go.mod), [.goreleaser.yaml](../.goreleaser.yaml), and [manifest_contract_test.go](../internal/mcpserver/manifest_contract_test.go). | **Deployment requirement:** consume artifacts from the canonical release record, verify checksums over a trusted channel, review dependency/action updates, and protect Forgejo/release credentials. **Not implemented:** signed artifacts, signed provenance, SBOM, reproducible-build comparison, dependency vulnerability gate, or immutable commit pins for all CI actions. Checksums alone do not defend against a compromised publisher. |
| T-13 | Raw RPC bypass can expose private firmware data or invoke undocumented/destructive methods. | **Implemented control:** it is a separate local CLI subcommand, absent from MCP tools, requires configured device/method/object params, limits RPC frames, and uses generic wire errors. See [debug.go](../internal/jetkvm/debug.go), [debug_test.go](../internal/jetkvm/debug_test.go), and [main_test.go](../cmd/jetkvm-mcp/main_test.go). | **Deployment requirement:** restrict local command execution, use only reviewed methods, keep parameters/results out of shell history and logs, and never expose it as a remote service. **Not implemented:** method allowlist, read-only classification, confirmation gate, output redaction, or semantic consequence detection. Its mutation class is intentionally uncertain. |

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
   or raw RPC actions.

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
  Current firmware-side `mount_url` does not satisfy those stronger controls;
  this is recorded as residual risk rather than claimed mitigation.
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
- **No scope redesign:** the review did not add OAuth, stateful HTTP, URL
  allowlists, approval gates, raw-RPC tools, deprecated capabilities, or tool
  metadata changes. Such controls require a separately accepted behavior change.
