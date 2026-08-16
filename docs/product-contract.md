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

The separately maintained `jetkvm-mcp-mutation-checklist` command is source-run,
offline, dry-run-only plan validation with no MCP client, device transport, or
mutation path. The command and checked plan are not included in binary archives
or the container and have no packaged-artifact guarantee. The [mutation
validation checklist](mutation-validation.md) owns the operator procedure; the
compatibility surface is declared below.

#### Mutation-checklist plan and report

The command accepts only the required `--plan` flag and no positional
arguments. Its value names one regular file of at most 64 KiB. Empty input,
final-component symlinks, non-regular files, and unknown fields are rejected.
Duplicate JSON object members are rejected, as are trailing JSON values and
more than one top-level value. The file is exactly one JSON object with these
required root members. Member names are case-sensitive.

| Member | Type and required value |
|---|---|
| `schema` | String exactly `jetkvm.mutation-validation.v1`. |
| `mode` | String exactly `dry_run`. No live or execute mode exists. |
| `target` | Object with the required booleans declared below. |
| `controls` | Object with the required booleans declared below. |
| `steps` | Array containing exactly 13 step objects: every operation in the closed inventory exactly once. |

The `target` object requires `marked_expendable`, `identity_confirmed`, and
`non_production`, all boolean `true`. The `controls` object requires
`observer_ready`, `recovery_ready`, `emergency_stop_ready`, and
`per_plan_acknowledgement` as boolean `true`, and `execution_approved` as boolean
`false`. No target identifier or other free-form target data is accepted.

The closed operation inventory is:

- `jetkvm_keyboard` and `jetkvm_mouse`;
- `jetkvm_press_host_power_button`, `jetkvm_press_host_reset_button`,
  `jetkvm_force_host_power_off`, `jetkvm_turn_host_dc_power_on`, and
  `jetkvm_turn_host_dc_power_off`;
- `jetkvm_wake_host_lan` and `jetkvm_wake_host_usb`; and
- `jetkvm_upload_virtual_media_file`, `jetkvm_mount_virtual_media_url`,
  `jetkvm_mount_virtual_media_file`, and `jetkvm_unmount_virtual_media`.

Every step requires string `operation`; boolean `consequence_acknowledged`,
`preconditions_confirmed`, `postcondition_observable`, `recovery_ready`, and
`never_retry_unknown_outcome`, all `true`; and integer `timeout_seconds`. The
timeout must be from 1 through 300. A `jetkvm_keyboard` step also requires string
`hid_operation` set to `type_text` or `press_key`. A `jetkvm_mouse` step requires
`hid_operation` set to `move_absolute`, `move_relative`, `click`, or `scroll`.
Every non-HID step must omit `hid_operation`; an empty string or JSON null is not
omission.

Media controls are operation-specific. `jetkvm_upload_virtual_media_file` and
`jetkvm_mount_virtual_media_file` require boolean `integrity_check_planned` and
`cleanup_planned`, both `true`. `jetkvm_mount_virtual_media_url` and
`jetkvm_unmount_virtual_media` require `cleanup_planned` as `true` and must omit
`integrity_check_planned`. All other steps must omit both media-control members.
False and JSON null do not satisfy a required media control.

The command writes one sanitized JSON object with these members and no others:

| Member | Type and value |
|---|---|
| `schema` | String exactly `jetkvm.mutation-validation.report.v1`. |
| `result` | String `pass` or `fail`. |
| `checked_steps` | Integer 13 on `pass`, otherwise 0. |
| `execution_authorized` | Boolean always `false`. |

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

Each source-run validator writes one sanitized JSON report to stdout; stderr is
reserved for validator diagnostics, and qualification child-server stderr is
deliberately discarded. `jetkvm-mcp-validate` exits 0 for a passing
qualification, 1 for a validation failure, and 2 for an argument or
required-input failure detected before validation.
`jetkvm-mcp-mutation-checklist` exits 0 only after a passing report is written,
1 for invalid plan input, failed plan validation, a null report sink, or a
report-write failure, and 2 for flag, missing-`--plan`, or extra-argument
errors. A report-write failure may leave partial or no stdout, but never exits
0.

## Versions and release support

Product releases use ordinary Semantic Versioning. During v0.x, the canonical
release record identifies the current non-prerelease product version and thus
the current `v0.<minor>.x` line. Only the latest non-prerelease release in that
minor line is supported. From v1 onward, only the latest stable release is
supported. This document does not duplicate a current release number. There is
no SLA, blanket backport promise, or EOL service.

### Accepted public v1 release destination

The first trustworthy public-release target is `v1.0.0`. The canonical GitHub
repository is to become public, and that release is to provide native
`linux/amd64` and `linux/arm64` archives through GitHub Releases plus a
multi-platform Linux container through GitHub Container Registry. Darwin and
Windows binaries are not part of the v1 support or artifact matrix. The native
archives and container must be built from the same protected release tag and
must report the same product version.

The existing `v0.1.0` tag and assets remain unchanged as unsupported,
private-era historical evidence. This accepted target does not claim that the
repository, release, or container is already public, and it does not authorize
the separate owner-controlled acts of changing repository visibility or
publishing a release.

One GitHub-hosted workflow builds every supported release deliverable from the
same protected tag. It produces GoReleaser Linux amd64 and arm64 archives, a
GHCR Linux amd64 and arm64 image, SHA-256 checksums, artifact-specific SPDX JSON
SBOMs including container runtime packages, the project license and required
third-party notices, a keyless Cosign bundle for the checksum manifest, a
keyless signature for the image digest, and GitHub-hosted provenance or
attestations for every archive, SBOM, and image digest. The workflow verifies
subjects and publishes consumer commands constrained to this repository,
workflow, and tag identity. Release tools and Actions are immutably pinned and
use OIDC without a long-lived signing key. The project does not claim
reproducible builds or a SLSA level from this evidence.

The supported container uses a digest-pinned Debian stable-slim runtime with
distro-provided FFmpeg and CA certificates. Each release records the base
digest, installed package versions, FFmpeg identity, and SPDX SBOM; carries OCI
source, revision, version, license, and creation labels; runs as fixed non-root
UID/GID 10001 without embedded configuration or credentials; and supports a
read-only root filesystem when writable temporary storage is supplied. Release
verification performs actual H.264-to-PNG decoding in each platform image. A
relevant base, FFmpeg, or CA-package security update produces a product patch
release. Exact apt package pins, custom FFmpeg builds, scratch or distroless
conversion, and a separate container release cadence are not part of v1.

An owner push of a protected annotated `vX.Y.Z` tag at the exact green `main`
commit is the single release-publication authorization. The workflow stages,
rebuilds, and verifies all deliverables before publishing an immutable GitHub
Release and `ghcr.io/bendmanning/jetkvm-mcp:vX.Y.Z`. Only `latest` moves after a
stable release succeeds; no moving major or minor image tags are published.
The version tag and image digest are canonical. Published tags and artifacts
are never updated, reused, or repaired in place. A failed tagged release
consumes that version and correction requires a new version.

Ordinary defects are corrected only by a new patch release, movement of
`latest`, clear identification of affected versions, and a Go module retraction
when useful. For suspected compromise, the owner may stop workflows and
publication, revoke affected authority, disable mutable distribution pointers,
and remove actively dangerous assets only when necessary to prevent continuing
harm. The response must preserve hashes and available evidence, publish an
advisory naming affected versions and digests, and issue a clean new version.
Release revocation, deletion, and replacement publication remain owner-only
actions; compromised history is never silently rewritten.

Each immutable GitHub Release is the sole canonical changelog record; the
repository does not duplicate it in a checked-in `CHANGELOG.md`. Curated release
notes state compatibility and migration changes; exact supported MCP, Go,
native, container, and physically qualified surfaces; security-relevant fixes
and known limitations; artifact digests and verification identities; and any
superseded or retracted versions. A generated commit list may seed the notes but
is never published without review.

The v1 MCP compatibility surface supports exactly revision `2026-07-28` over
both stdio and Streamable HTTP. Legacy and unknown revisions must be rejected
rather than advertised or accepted through incidental Go SDK compatibility.
The official Go SDK remains an implementation dependency and does not expand
the product contract. Sessions, legacy HTTP+SSE, tasks, prompts, resources,
sampling, and other unadvertised SDK capabilities remain unsupported.

The v1 tool surface contains the 17 one-purpose tools and removes the deprecated
multiplexed `jetkvm_virtual_media` compatibility tool. Release documentation
must map each removed operation to its dedicated status, URL mount, file mount,
file upload, or unmount replacement. No evidence of external reliance justifies
carrying the duplicate private-era surface into the first public release.

V1 consequence annotations are deliberately conservative. Every mutation has
`idempotentHint=false`; an annotation must never suggest blind retry after an
`unknown` outcome. Keyboard and mouse operations have `destructiveHint=true`
because they can execute commands, alter data, or activate destructive UI.
Read-only tools remain non-destructive and idempotent, and URL mounting remains
the only open-world tool. These hints communicate consequences but do not grant
authority or prove user intent.

V1 applies the same identifier boundary in configuration and MCP input:
configured device aliases and Wake-on-LAN target aliases contain 1 through 128
Unicode code points after trimming. A keyboard `key` contains 1 through 32
Unicode code points. A modifier array contains at most the four unique members
from its existing closed enum. Caller-visible validation failures use stable,
value-free `invalid_input` messages and never echo submitted text, aliases,
paths, URLs, keys, or other caller values.

Every successful v1 tool returns typed `structuredContent`, and output schemas
constrain product-owned fixed vocabularies to the values the server can emit.
Screen capture preserves PNG `ImageContent` as its first content block and adds
a second JSON `TextContent` compatibility fallback containing only the device
alias, capture time, MIME type, dimensions, and byte count. It never duplicates
PNG data into text. Operational failures remain sanitized `IsError` tool
results rather than JSON-RPC errors.

V1 publication requires one owner-authorized physical qualification run against
a disposable attached host. Retained sanitized evidence must identify the exact
release candidate, JetKVM model and firmware, runtime and FFmpeg, and must cover
every public operation class: reads and capture, HID, power and DC control,
wake, upload, mount, unmount, cleanup, and observable outcomes. Claims remain
limited to the recorded combination and checks. Repeat qualification only when
device protocol behavior, firmware, relevant runtime, or covered operations
change; do not infer a model family, firmware range, or indefinite guarantee.

The source-run `jetkvm-mcp-mutation-checklist`, its synthetic JSON plan, and its
detailed compatibility surface are not part of the public v1 design. An offline
file of self-asserted booleans neither authorizes execution nor proves a hardware
safeguard. Physical qualification instead uses a concise fixture-specific
runbook, separate owner authorization, observable postconditions, retained
sanitized evidence, and the established rule that an `unknown` outcome ends the
current mutation window.

Source builds support only the current and immediately previous supported Go
release families, each at its latest security patch. Compatibility checks use
`GOTOOLCHAIN=local`. Primary CI, native release archives, and the container
binary use the same exact current-family Go toolchain. At this decision point
the minimum is Go `1.25.13` and the release toolchain is Go `1.26.6`; after Go
1.27 makes the 1.25 family unsupported, the minimum moves to the selected
current 1.26 patch and release builds move to a selected patched 1.27 release.

Dependency maintenance uses monthly grouped Dependabot updates with at most one
open PR for each of Go modules and tracked Go tools, GitHub Actions, and Docker
base images. Security advisories are handled promptly outside that cadence.
Staticcheck and `govulncheck` are native tracked Go tool dependencies. MCP
conformance and Inspector npm pins change only through deliberate protocol
review rather than generic version-update automation. Dependency, toolchain,
Action, and base-image updates are never auto-merged; review covers upstream
changes, relevant advisories, licenses, and affected tests. Unreachable
module-level advisories receive a recorded risk disposition rather than an
automatic failure or a blind transitive upgrade.

The following are post-adoption decisions, not public-v1 blockers: broader
hardware, native-runtime, soak, or performance matrices; OAuth, multiple
principals, or finer authority; a second maintainer or formal governance;
contribution-volume automation; additional scanners or long fuzzing;
reproducibility, VEX, notarization, or duplicate SBOM formats; and Kubernetes,
benchmarks, pooled sessions, sidecars, policy engines, AI firewalls, or
observability backends. Reconsider one only for a concrete user, failure,
advisory, contribution volume, or consumer requirement.

SIGINT and SIGTERM cancel one process-root context shared by stdio and every
active HTTP request. HTTP shutdown permits at most five seconds for handler
cleanup and graceful draining, then force-closes remaining connections and
joins the serving goroutine before process exit. A mutation interrupted after
dispatch may have begun remains `unknown` and non-retryable; shutdown never
turns process termination into evidence that a physical effect did or did not
occur.

Streamable HTTP retains one full-authority administrator bearer and does not add
OAuth, users, roles, or per-tool scopes. Binding is loopback by default. Every
non-loopback `/mcp` request requires exactly one `Authorization` header with a
case-insensitive `Bearer` scheme, one nonempty token, and constant-time token
comparison. Host and Origin admission remain exact; browser use is same-origin,
CORS is absent, and forwarded headers are not trusted. Requests have a 1 MiB
body limit, a five-second header timeout, a 15-second body-read timeout, and a
60-second idle timeout. Tool-specific deadlines replace a global write timeout.
`/healthz` is unauthenticated and content-minimal. It reports only that startup
configuration and FFmpeg validation succeeded and the HTTP process is accepting
requests. Device reachability remains an MCP read concern; v1 adds no `/readyz`,
background device probe, or aggregate device-health state. TLS, connection-rate
limits, and public-edge controls belong to the reverse proxy. Multi-principal
or narrower authority requires an external gateway or a future product
decision.

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

Sensitive reports use the private route in the repository
[`SECURITY.md`](../SECURITY.md). Ordinary best-effort support uses GitHub Issues
as defined in [`SUPPORT.md`](../SUPPORT.md). Secrets, credentials, device data,
and unsanitized qualification output must not be put in public issue or release
records.

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
| Mutation-checklist CLI | Declared source-run interface: required `--plan`; strict `jetkvm.mutation-validation.v1` plan with concrete HID subtypes; sanitized `jetkvm.mutation-validation.report.v1` JSON and exit status | Unit tests exercise strict decoding, bounded regular-file input, plan inventory and controls, report shape, output failures, and the checked synthetic plan. It has no execution path and grants no mutation authority. No checklist binary is distributed. |
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

Each `jetkvm_capture_screen` operation has a server-owned 30-second maximum,
including fresh video-session setup, fresh-frame wait, and local PNG decode.
This bound applies even when a caller provides no deadline; an earlier caller
cancellation or deadline takes precedence. Expiry is returned through the
normal read-error timeout result.

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
lists tools and exercises configured-device discovery, status, dedicated
virtual-media status, and capture; it does not qualify keyboard, mouse,
virtual-media mutation, power, wake, raw RPC, transport deployment, or every
returned firmware field. Evidence must be refreshed when the product's device
protocol behavior changes, the model or firmware changes, the relevant runtime
or FFmpeg changes, or the retained checks no longer cover the claim. It does not
become an indefinite vendor compatibility guarantee. No current retained
evidence satisfies this policy, so no model or firmware is presently qualified.

The sanitized evidence ledger and focused upstream-source drift triggers are in
[`compatibility/`](compatibility/README.md). An unattributed historical run,
`review_required` source drift, source inspection, or fake-device result remains
evidence of only its recorded checks and cannot become a positive claim.

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
