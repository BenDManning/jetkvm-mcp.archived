# Product, compatibility, and support contract

Status: maintained.

Ownership: this document owns JetKVM MCP product scope, versioning,
compatibility, and support policy. Changes require an accepted GitHub issue
and repository review. Review is required for every release and whenever a
public surface, prerequisite, support matrix, exclusion, or compatibility claim
changes. The [README](../README.md) owns setup and operation; the
[protocol sources](protocol-sources.md) own inspected upstream revisions and
provenance, and the [threat model](threat-model.md) owns current security and
privacy boundaries. The [architecture decision index](adr/README.md) owns the
rationale, rejected alternatives, consequences, and revisit triggers for
consequential current design choices. Code and tests remain authoritative for
executable behavior.

## Product boundary

`jetkvm-mcp` is a conventional, single-process Model Context Protocol (MCP)
server for operating configured JetKVM devices. It is an integration product,
not a policy or device-qualification system.

The shipped server entry point supports MCP over stdio by default and stateless
Streamable HTTP only when `--http` is supplied. It also supplies conventional
help, `--version`, offline `config validate`, and the local-only `debug rpc`
diagnostic; raw JetKVM RPC is not an MCP tool. Normal serving requires an
`ffmpeg` executable on `PATH`, including when only non-capture tools are
intended. Help, version, configuration validation, and `debug rpc` do not
initialize FFmpeg. Configuration validation performs the same strict load and
environment resolution as startup but opens no listener or device/network
session. Logs and diagnostics use stderr; stdio-server stdout is reserved for
MCP traffic. Operational details are in the [README](../README.md).

The separately maintained `jetkvm-mcp-validate` command is source-run,
read-only qualification tooling. Its documented flags, exit behavior, and
sanitized JSON report are compatibility surfaces, but it is not included in
the binary archives or container and has no packaged-artifact guarantee.

### CLI streams and exit status

Normal server diagnostics, serving notices, and errors use stderr. While
serving over stdio, stdout contains MCP traffic only. `--version` writes exactly
`jetkvm-mcp <version>` followed by a newline to stdout and exits 0. A successful
`debug rpc` writes one JSON object with a top-level `result` member to stdout and
exits 0. Help writes usage to stdout and exits 0. Successful `config validate`
writes exactly `configuration valid` followed by a newline to stdout and exits
0. Command, argument, validation, and runtime failures returned through the main
CLI write a `jetkvm-mcp:` diagnostic to stderr and exit 1. Flag errors preserve
the safe flag name without echoing its supplied value.

The validator writes one sanitized JSON report to stdout; stderr is reserved for
any validator diagnostics, and child-server stderr is deliberately discarded.
It exits 0 for a passing validation, 1 for a validation failure, and 2 for an
argument or required-input failure detected before validation.

## Versions and release support

Product releases use ordinary Semantic Versioning. During v0.x, the canonical
release record identifies the current non-prerelease product version and thus
the current `v0.<minor>.x` line. Only the latest non-prerelease release in that
minor line is supported. This document does not duplicate a current release
number. There is no SLA, blanket backport promise, or EOL service.

GoReleaser-built binaries receive GoReleaser's explicit version value.
Containers report the explicitly supplied `VERSION` build argument and
otherwise explicitly inject `dev`; any nonempty injected value remains
authoritative. Without an injected value, a versioned `go install` reports the
main module version from `debug.ReadBuildInfo`. Local source and `make build`
outputs report `devel+` plus the first 12 hexadecimal characters of
`vcs.revision`, with `.dirty` when `vcs.modified=true`; metadata-poor builds fall
back to `devel` or `dev`. The same resolved value is used by `--version` and as
the MCP implementation version. MCP revision dates are protocol-compatibility
metadata, not product versions. The Go SDK version is an implementation
dependency, not a second product version.

The release record is the only promised delivery record. Packaging definitions
state intended names and target matrices; they do not guarantee that an archive
or container was published, remains available, or belongs to a permanent
channel.

Deprecation notice must appear in documentation and release notes, identify a
replacement and migration when one exists, and normally leave the deprecated
surface available through at least one subsequent minor release. Removal is
still a major change. Security, safety, or factual-correctness fixes may shorten
that period, but must still be described in release notes and use the SemVer
classification dictated by their compatibility impact.

No designated public security-reporting route currently exists. Repository
issue and release records are the available support and change records; secrets,
credentials, device data, and unsanitized qualification output must not be put
in them.

## Compatibility matrix

“Declared” is a product promise; “exercised” is bounded test or build evidence;
“external” is a caller-supplied prerequisite. Unknown or unsupported surfaces
are not made supported by dependency behavior, cross-compilation, fake devices,
or point-in-time protocol observations.

| Surface | Current contract | Evidence and boundary |
|---|---|---|
| Product releases | During v0.x, declared support is the latest non-prerelease release in the current minor line | The canonical release record, not this document, identifies the current non-prerelease version and line. `dev` builds are unreleased and outside release support. |
| MCP revision and SDK | Declared: MCP revision `2026-07-28`; implemented with Go SDK `v1.7.0` | Exercised by stdio and Streamable HTTP tests. Older revisions the SDK may negotiate are unsupported unless this contract and repository tests add them. Exact SDK provenance is in [protocol sources](protocol-sources.md). |
| Go source and build | Declared source minimum: Go 1.25. CI/container build toolchain: Go 1.26.5 | `go.mod` declares 1.25; CI and the container use 1.26.5. Makefile, local, and arbitrary GoReleaser environments are not pinned by that fact. Go 1.25 is not exercised by current CI, and no promise covers every intervening toolchain or runtime combination. |
| Native OS/architecture | Declared binary targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` | CI cross-builds all four. Runtime qualification for every target is not recorded. Windows and all unlisted combinations are unsupported. |
| FFmpeg | External prerequisite: an executable named `ffmpeg` on `PATH` for normal serving and capture | Startup and decoder tests exercise executable discovery and bounded invocation. No implementation or version range is qualified; compatibility is unknown beyond successful checks in a specific environment. |
| JetKVM model/firmware | No model or firmware version is currently qualified or supported by a positive compatibility claim | Upstream source observations, fake-device tests, and validator runs are evidence, not guarantees. Claim requirements are defined below. |
| YAML configuration | Declared: the current strict, unversioned grammar; no schema-version field or migration engine | Loader tests cover the grammar. The exact example is [`config.example.yaml`](../config.example.yaml); unknown fields, empty or over-1 MiB input, multiple YAML documents, unsafe admission limits, and device-URL user information/query/fragment components are rejected. Device URL path prefixes remain supported. |
| Server CLI | Declared: help, `--config`, optional `--http`, `--version`, offline `config validate`, and `debug rpc` with its documented flags, streams, and JSON result | Parser and integration tests exercise these entry points. `debug rpc` permits only `ping`, `getLocalVersion`, and `getActiveExtension` by default; every other method requires per-invocation `--unsafe-acknowledge-risk`. Free-form diagnostic wording is not stable unless documented as structured output. |
| Validator CLI | Declared source-run interface: required `--binary`, `--config`, and `--device`; sanitized JSON and exit status | Unit tests exercise argument and report shape. Physical qualification is absent until retained evidence satisfies the policy below. No validator binary is distributed. |
| MCP tools, results, errors, and annotations | Declared: the 18 current tools, their schemas, structured results/content, tool-result error semantics, and annotations | [`server.go`](../internal/mcpserver/server.go), [`controls.go`](../internal/mcpserver/controls.go), and their tests are the executable source of truth. Execution failures use tool results with `IsError`, not protocol errors. `jetkvm_list_devices` returns sorted configured aliases and configuration-derived availability flags; it does not open a device session or qualify firmware capabilities. `jetkvm_virtual_media` is retained as a deprecated compatibility surface; clients should migrate each operation to the corresponding one-purpose `jetkvm_*_virtual_media*` tool. |
| Binary artifacts | Intended: one `jetkvm-mcp` binary in `jetkvm-mcp_<version>_<os>_<arch>.tar.gz` for each declared native target, plus `checksums.txt` | [GoReleaser configuration](../.goreleaser.yaml) and CI establish naming and cross-build evidence, not publication or runtime qualification. |
| Container artifacts | Intended: Linux amd64 and arm64, one `jetkvm-mcp` entry point, bundled FFmpeg, UID/GID 10001 | The [Dockerfile](../Dockerfile) and CI establish build intent. No image name, registry channel, publication, indefinite availability, or per-platform runtime qualification is promised. |

## Semantic Versioning rules

The compatibility rules apply literally before 1.0. There is no “anything may
change” exception for `v0.x`: given `v0.7.3`, a compatibility-breaking release
is major (`v1.0.0`), a backward-compatible addition is minor (`v0.8.0`), and a
backward-compatible fix is patch (`v0.7.4`). A change advertised as additive is
minor only when supported clients can ignore or omit it without breaking.

| Surface | Major: incompatible | Minor: backward-compatible addition | Patch: backward-compatible fix |
|---|---|---|---|
| MCP tools and input schemas | Rename `jetkvm_get_status`; require `max_width`; remove `mount_url`; narrow the capture bounds | Add a tool or an optional input that leaves calls such as `{"device":"lab"}` valid | Correct acceptance or rejection within the documented names, types, enums, bounds, and effects |
| Results, content, annotations, and errors | Rename `connected`; change `width` type; replace PNG content; turn a tool-result `IsError` into a protocol error; change `openWorldHint`, destructive, or idempotence metadata in a way that can alter client safety decisions | Add an optional status field or ignorable metadata without changing existing meaning | Correct result serialization or execution while preserving wire shape, meaning, error category, and annotations |
| YAML fields, semantics, and defaults | Rename `password_env`; make `http` required; incompatibly change behavior when `media_directory` is absent; reject previously accepted meaningful configuration | Add an optional field whose omission preserves current behavior | Correct compatible origin normalization or validation without invalidating meaningful accepted configuration |
| CLI and validator | Remove `--http`; require a formerly optional flag; move MCP data off stdout; incompatibly change exit semantics or the validator's sanitized JSON | Add an optional flag or command, or an optional sanitized JSON field | Fix execution or human-readable diagnostics while preserving documented invocation, streams, exit behavior, and structured output |
| Artifact targets, names, and layout | Remove `darwin/arm64`; rename `jetkvm-mcp_<version>_<os>_<arch>.tar.gz`, its binary, or `checksums.txt`; incompatibly change the container entry point or UID/GID | Add a target or artifact alongside the existing matrix and layout | Repair packaging or metadata without changing supported names, contents, layout, or runtime contract |

The same impact test governs timing or consequence changes to boot, power, wake,
and virtual-media operations: an incompatible consequence is major, an optional
new operation or control is minor, and a compatible correctness fix is patch.
Changing MCP revision or SDK does not by itself choose a product version:
dropping supported negotiation or behavior is major, adding compatible revision
support is minor, and an implementation-only compatible fix is patch. Dropping
a firmware combination after it has been positively qualified is major.

Free-form logs and human-readable error text may change in a patch when their
stream, category, exit semantics, and any documented structured representation
remain compatible. Status strings and JSON fields documented as structured
results are versioned surfaces, not free-form diagnostics.

Status and virtual-media success results expose only product-owned typed fields.
The `virtualMedia` status object and media-tool results report `mounted`, an
optional `sourceType` (`http` or `storage`), and an optional normalized `mode`;
they never return firmware JSON, URLs, paths, filenames, query strings,
fragments, or unknown firmware fields. This reviewed privacy correction is an
intentional breaking output-contract change relative to v0.1.0 and therefore
requires the next release carrying it to use the major version `v1.0.0` or
later under this contract's pre-1.0 SemVer rules.

Virtual-media URL mounting is also intentionally deny-by-default. Each device
must configure one or more `media_url_allowed_origins`; omitting the field makes
URL mounting unavailable while leaving status and local-file operations subject
to their existing preconditions. Because prior releases accepted unrestricted
HTTP(S) mount URLs without this field, the first release carrying this boundary
must likewise be `v1.0.0` or later.

### MCP tool-manifest review

[`internal/mcpserver/testdata/tool-manifest.json`](../internal/mcpserver/testdata/tool-manifest.json)
is the reviewed wire-contract fixture. It normalizes server discovery metadata,
the intentionally empty client capability set used by the contract test,
advertised server capabilities, every tool title, description, input and output
schema, and annotation, plus representative success, operational-error, and
protocol-error envelopes. The same fixture is exercised over in-memory, stdio
subprocess, and stateless HTTP transports. Structured success values are
validated against each advertised output schema.

Classify every deliberate fixture change before updating it:

- **breaking:** a removal or rename; tighter input acceptance; weaker output
  guarantees; incompatible schema, content, error, capability, or transport
  semantics;
- **additive:** a new optional tool, field, content form, or capability that
  existing supported clients can ignore; or
- **patch:** a correction that preserves accepted inputs and observable output
  semantics, such as wording or explicitly non-contractual metadata.

Run `make update-tool-manifest`, inspect the complete diff, record the
classification in review, and then run `make verify` and `make race`. CI runs
the manifest contract through `go test ./...`, so an unreviewed runtime delta
fails against the committed fixture. Regeneration produces review evidence; it
does not itself approve a compatibility change.

## Configuration evolution and migrations

The YAML grammar is currently unversioned and has no migration engine. Product
SemVer therefore governs fields, validation, environment-variable resolution,
semantics, and defaults. Additive optional fields are normally minor. Removal,
rename, newly required fields, or incompatible semantic/default changes are
major. Compatible validation fixes are patch unless they invalidate previously
accepted meaningful configurations, in which case they are major.

The optional `limits` mapping controls process-wide admission with defaults of
16 operations, 4 operations per device, 8 sessions, 2 captures, and 2 decoders.
Every limit is an integer from 1 through 1024. Per-device/session capacity may
not exceed global operations, and capture capacity may not exceed sessions.
Exhaustion returns the existing
non-retryable `busy`/`not_sent` tool error without device/provider dispatch and
does not queue work. Mutating HID, power, and virtual-media operations serialize
per device; waiting for that one bounded mutation slot observes cancellation.

A deprecation must identify the affected field or value and the replacement,
show a safe migration, retain the old form for the normal deprecation period,
and test old and new forms during that period. A major release that cannot
migrate automatically must provide explicit before-and-after configuration
examples. The absence of a schema-version field must not be treated as license
to reinterpret an existing file silently.

## Compatibility evidence

A positive JetKVM compatibility claim must name all of the following:

- the exact JetKVM MCP product version or commit;
- the exact device model and firmware version;
- the relevant host OS/architecture and FFmpeg identity when capture is in scope;
- the qualification date and exact checks performed; and
- retained sanitized validator JSON tied to that product commit and firmware.

Evidence qualifies only that combination and those checks. The current validator
lists tools and exercises status and capture; it does not qualify keyboard,
mouse, virtual media, power, wake, raw RPC, transport deployment, or every
returned firmware field. Evidence must be refreshed when the product's device
protocol behavior changes, the model or firmware changes, the relevant runtime
or FFmpeg changes, or the retained checks no longer cover the claim. It does not
become an indefinite vendor compatibility guarantee. No current retained
evidence satisfies this policy, so no model or firmware is presently qualified.

Repository tests, fake devices, cross-builds, the inspected upstream firmware
commit, and a validator run without retained sanitized evidence remain useful
engineering evidence but cannot establish physical compatibility or expand the
support matrix.

## Trust and external-effect boundaries

Configured device endpoints and credentials are trusted administration inputs.
Credentials are resolved through named environment variables; device and media
URLs reject inline user information. Device URL path prefixes are retained,
while query and fragment components are rejected because runtime endpoints
would otherwise discard them. Appliance HTTP bypasses environment proxy
settings, rejects redirects, and verifies TLS with a minimum of TLS 1.2 unless
`insecure_skip_verify` is explicitly enabled for a device. A configured media
directory is the local-file boundary. URL mounting additionally requires the
mount URL's normalized scheme, host, and effective port to match one exact
per-device `media_url_allowed_origins` entry before provider dispatch. Paths,
queries, and fragments do not select an origin. Exact deployment and media
behavior is documented in the [README](../README.md).

Raw RPC remains a local CLI escape hatch and is absent from MCP. The exact
source-reviewed read-only default set is `ping`, `getLocalVersion`, and
`getActiveExtension`. Unknown, unreviewed, and known mutating methods fail before
session dispatch unless that invocation includes `--unsafe-acknowledge-risk`.
The flag cannot be persisted through configuration or environment and does not
classify a method as safe: acknowledged calls may mutate hardware or
boot/storage state and may return sensitive raw firmware data to stdout.

Non-loopback Streamable HTTP requires a bearer token. Native clients may omit
`Origin`, subject to Host and authentication policy. Browser use is same-origin:
a separately hosted browser origin is unsupported. Every public reverse-proxy
endpoint requires exact `http.allowed_origins` configuration, including scheme,
authority, and any non-default port. This is Host/origin admission, not a CORS
grant; wildcard entries are invalid and the server emits no CORS response
headers. A present invalid, foreign, duplicate, empty, or opaque `null` Origin is
rejected with HTTP 403 before bearer authentication, method handling, or MCP
dispatch. An admitted same-origin OPTIONS request remains subject to any
configured bearer and receives HTTP 405 after authentication because the
supported mode needs no cross-origin preflight. TLS termination remains a
deployment/reverse-proxy responsibility. These controls do not add OAuth, user
identity, authorization grants, or a general trust/control plane.

Wildcard `http.allowed_origins` values were previously parsed as literal Host
authorities, but they never implemented wildcard matching and cannot identify a
supported exact public endpoint. Rejecting those values, and rejecting present
empty or duplicate Origin headers before authentication, are compatible security
validation corrections within the documented exact-Origin contract rather than
the removal of meaningful supported configuration or a new CORS surface.
Likewise, canonicalizing equivalent IP literals and treating explicit HTTP port
80 or HTTPS port 443 as the corresponding omitted effective port makes endpoint
admission match browser-origin semantics; scheme and non-default ports remain
distinct.

Rejecting device URL query and fragment components is a compatible validation
correction because those components were silently discarded rather than used as
meaningful endpoint semantics. In contrast, requiring
`--unsafe-acknowledge-risk` for previously accepted unreviewed raw RPC methods is
an intentional incompatible CLI safety boundary; the first release carrying it
must be `v1.0.0` or later under this contract's pre-1.0 SemVer rules. The same
release floor is already required by the typed-result and virtual-media URL
boundaries above.

`jetkvm_mount_virtual_media_url` reports `openWorldHint=true` because it asks the
appliance to retrieve a caller-selected HTTP(S) URL from a configured exact
origin. Every other current MCP tool reports `openWorldHint=false`. Origin
authorization does not resolve or pin DNS, inspect firmware redirects, classify
the resolved address, or enforce appliance routing. These annotations describe
the tools' intended interaction boundaries; they are not proof that firmware,
network, or hardware effects are contained. The allowed URL forms, fetching
actor, credential prohibition, result/error behavior, and annotations are
separately versioned public behavior.

Tool execution failures use a versioned caller-visible JSON object with
`version`, `code`, `message`, `outcome`, and `retryable`. Stable codes include
`invalid_input`, `busy`, `authentication_failed`, `device_unavailable`,
`video_unavailable`, `no_signal`, `protocol_error`, `canceled`, `timeout`, and
the conservative `operation_failed` fallback. Outcomes are `not_sent` when the
server knows dispatch did not begin, `failed` when failure is confirmed, and
`unknown` when a mutation may have been sent without a conclusive response.
Read-only transient failures may be marked retryable. Mutation failures are
never marked retryable, and an `unknown` outcome is never reported as completed
or as safe to retry. Messages omit raw firmware, URLs, paths, credentials, and
request payloads.

## Exclusions

Permanent product boundaries are: capability grants; qualification flags or
databases; firmware approval or override mechanisms; trust/control planes;
malicious-agent policy framing; decoder IPC or sidecars; multiple runtime
containers; and treating a fake appliance as product authority. Test fixtures
support implementation only.

The following are unsupported but revisitable only through a new accepted
objective and an update to this contract: Windows and unlisted OS/architecture
targets; MCP OAuth; deprecated HTTP+SSE; additional transports or integrations;
packaged validator artifacts; and new guaranteed publication channels. Anything
not declared in the compatibility matrix is unsupported or unknown, not
implicitly accepted. A permanent boundary cannot be treated as revisitable
without first amending this section through an accepted product-contract change.
