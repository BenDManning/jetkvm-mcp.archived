# Exact-revision repository evidence for the 2026 public-release baseline

## Evidence identity and method

- **Research date:** 2026-08-15 (America/Detroit).
- **Expected baseline supplied by the maintainer:** `176ec421f9ee6c801517180e1ad0ec9c84570e8e`.
- **Actual local HEAD at the beginning of this research:** `6e52f0027b13f928b768de0feeab4847ef9ca53e` on `cleanup/remove-custom-ci-policy`.
- **Initial working-tree record (captured before creating this result):**

  ```text
  ## cleanup/remove-custom-ci-policy
  ?? .agents/
  ?? skills-lock.json
  ```

- **Revision divergence:** actual HEAD is one commit after the expected baseline. `git diff --stat 176ec42..6e52f00` reports deletion of `cmd/jetkvm-ci-check/{command_other.go,command_unix.go,main.go,main_test.go}` and `internal/cipolicy/policy_test.go`, plus four changed lines in `Makefile` and twelve in `docs/ci-quality.md` (485 net deleted lines). The studied revision therefore has no custom CI-policy executable/test package; evidence below does not attribute those deleted controls to the studied tree. `origin/main` and local `main` were still `176ec42`; only the checked-out cleanup branch contained `6e52f00`.
- **Tree scope:** all 131 tracked files at actual HEAD were inventoried, including hidden GitHub, GoReleaser, Docker, ignore, source, test, fixture, validator, compatibility, protocol-gate, and documentation surfaces. The tree contains 20,942 lines of tracked Go and 271 `Test*`/`Fuzz*` declarations. Code, tests, configuration, and executable fixtures were treated as behavioral authority; prose was checked against them. The untracked `.agents/` and `skills-lock.json` were recorded as working-tree state but are agent configuration, not silently treated as shipped product behavior. Research outline/results created after the initial snapshot are likewise not part of the product revision.
- **Verification boundary:** this was static repository inspection plus read-only GitHub API inspection. No hardware was contacted and no test, build, workflow, source, setting, issue, PR, branch, or release was changed. Consequently a checked-in test is evidence of intended automated verification, not proof that it passed at this revision. CI results, real-device behavior, soak properties, and binary reproducibility require separate evidence.
- **Citation convention:** `path:Lx-Ly` cites the studied local file; `path:symbol` cites the named Go declaration/test where line ranges would obscure the evidence. Live API observations include their endpoint and access date.

## Snapshot verdict

The repository is unusually strong for a small pre-1.0 device-control MCP server in its bounded configuration, explicit consequence surface, ambiguous-mutation semantics, fixture/conformance separation, privacy-safe diagnostics, protocol provenance, and targeted parser/lifecycle tests. The strongest claims are implemented and tested rather than merely documented. It is nevertheless not yet demonstrably public-release-ready: release automation and container publication are absent; the only release provides checksums but no SBOM, signature, or provenance; Actions are tag-pinned and the repository does not require SHA pins; public security/contribution/support contracts are absent; private-repository branch rules could not be enabled/inspected on the current plan; and positive JetKVM compatibility evidence is deliberately incomplete.

The most important evidence distinction is that `make verify`, cross-compiles, the container build, fake sessions, conformance fixtures, and a single unattributed read-only hardware ledger entry establish different things. None establish broad physical-device or firmware support. The repository itself generally states this correctly.

## Current-state matrix

| # | Workstream | Repository state | Documentation/implementation agreement | Principal repository gap or unknown |
|---:|---|---|---|---|
| 1 | Go toolchain, modules, dependencies, vulnerabilities | **Substantially satisfied** | High | No binary `govulncheck`, license/inventory decision, dependency-review policy, or immutable analyzer installation graph; Docker Go patch differs from CI. |
| 2 | Architecture and public API | **Substantially satisfied** | High | Presentation DTOs live in `mcpserver` and are imported by `jetkvm`; acceptable now but a coupling seam to watch, not a reason for a framework. |
| 3 | Correctness, concurrency, cancellation, lifecycle | **Substantially satisfied** | High | No retained soak/leak evidence; real WebRTC/firmware cancellation and cleanup remain unqualified. |
| 4 | Testing, verification, qualification, performance | **Substantially satisfied** | High | No attributed model/firmware matrix, mutation qualification, soak, or accepted performance objective/threshold. |
| 5 | MCP specification/SDK/conformance | **Substantially satisfied** | High | Conformance execution result was not rerun here; list caching/metadata behavior partly belongs to SDK and needs revision-specific external confirmation. |
| 6 | Tool surface/consequences/compatibility | **Substantially satisfied** | High | 18-tool pre-1.0 surface includes a deprecated multiplexed compatibility tool, increasing permanent compatibility burden until removal policy is chosen. |
| 7 | Transports/auth/deployment | **Substantially satisfied** | High | No in-process TLS/rate limiter/OAuth (explicit boundary choices); reverse-proxy behavior and production limits lack deployment qualification. |
| 8 | Agentic-AI/MCP threats | **Partial** | Medium-high | Server controls authority and consequences, but cannot prevent client prompt injection, cross-server confusion, tool-result misuse, or compromised-agent approval; public docs need a concise client/deployment responsibility statement. |
| 9 | JetKVM/WebRTC/FFmpeg/HID/media | **Partial** | High | Strong defensive implementation; upstream protocol drift is `review_required`, firmware redirects/DNS remain appliance-side residual risk, and physical compatibility is not established. |
| 10 | Privacy/logging/telemetry | **Substantially satisfied** | High | No retention guidance for operator stderr/CI artifacts or incident-evidence procedure; telemetry is intentionally lossy. |
| 11 | AI-assisted development/untrusted contributions | **Gap** | Low/unknown | No contributor/agent review policy, dependency provenance checklist, or public fork threat guidance; CI permissions are good but Actions allow-list is broad. |
| 12 | GitHub governance/public readiness | **Gap** | High where documented | Private repo; rules/protection unavailable on current plan; no SECURITY, CONTRIBUTING, SUPPORT, CODEOWNERS, issue forms, incident or disclosure route. |
| 13 | Actions and CI/CD | **Partial** | High | Read-only permissions and concrete lanes are strong; Actions use mutable major tags, all Actions are allowed, no concurrency cancellation/retention, and no release workflow. |
| 14 | Releases/supply chain | **Partial** | High | Checksums and pinned container bases exist; no SBOM, signing, attestation/provenance, registry publication, rollback/verification contract, or FFmpeg package pin/provenance. |
| 15 | Framework/pragmatic operating model | **Partial** | N/A (synthesis) | Many applicable technical controls exist, but no concise one-maintainer release/security cadence or explicit framework crosswalk; avoid converting this into process machinery. |

## 1. Go toolchain, modules, dependencies, and vulnerability management

**Scope and surfaces.** `go.mod`, `go.sum`, `Makefile`, Dockerfile, `.github/workflows/ci.yml`, `.github/dependabot.yml`, `docs/ci-quality.md`.

**Verified repository evidence.** The module declares Go `1.25.0`; all product dependencies are versioned modules, including official MCP Go SDK `v1.7.0`, Pion WebRTC `v4.2.18`, and bounded direct dependencies (`go.mod:L1-L15`). Primary CI uses Go `1.26.6`, while a distinct minimum lane uses exactly `1.25.0` with `GOTOOLCHAIN=local`, preventing an implicit compiler download (`.github/workflows/ci.yml:L13-L62`; `docs/ci-quality.md:L7-L18`). The Makefile checks formatting, tidy drift with `GOWORK=off`, module sums, tests, race, vet, Staticcheck `v0.7.0`, source `govulncheck v1.7.0`, fuzz smoke, coverage, and target builds (`Makefile:L1-L65`). Monthly Dependabot groups Go and Actions updates and limits each ecosystem to one open PR (`.github/dependabot.yml:L1-L21`). `go.sum` provides module content hashes, and CI runs `go mod verify` (`Makefile:L22-L23`).

**Agreement and state.** The minimum/newer-toolchain and source vulnerability claims in `README.md`, `docs/ci-quality.md`, workflow, and Makefile agree. State: **substantially_satisfied**.

**Gaps/risks/unknowns.** Staticcheck and govulncheck are invoked with `go run ...@version`; their top-level versions are fixed, but they are not represented as a reviewed tools module and their download/build graph is not visible in `go.mod`/`go.sum`. Source-mode govulncheck does not establish the vulnerability state of a released binary or the Debian/FFmpeg runtime. No checked-in dependency/license review policy, license inventory, binary scan, OS-package vulnerability check, or evidence for standard-library vulnerability response was found. The Docker build uses Go `1.26.5`, not CI's `1.26.6` (`Dockerfile:L3`; `.github/workflows/ci.yml:L20,L71`), so “release toolchain” is not singular. Dependabot grouping reduces noise but can obscure which update caused a regression and does not replace review. There is no Renovate/custom updater, which is not itself a gap for this scale.

**Repository-grounded acceptance evidence for future claims.** A dependency/build-input policy should name owners, review evidence, emergency security update handling, direct versus transitive review, toolchain/container synchronization, and how a release binary/runtime are scanned. Proving a fixed dependency state requires exact `go version -m`/digest-bearing artifacts and successful source plus binary/runtime checks, not documentation alone. Revisit on every Go support-window change, MCP/Pion/FFmpeg advisory, or Dependabot graph change.

## 2. Go architecture, package boundaries, and public API design

**Scope and surfaces.** `cmd/**`, `internal/config`, `internal/jetkvm`, `internal/mcpserver`, `internal/httporigin`, `internal/telemetry`, `internal/protocolgate`, compatibility and validator commands, `CONTEXT.md`, ADRs.

**Verified repository evidence.** Product code is entirely command/internal code: there is no exported public Go library API. `cmd/jetkvm-mcp` owns CLI/lifecycle and wires config → provider/manager → MCP server (`cmd/jetkvm-mcp/main.go:L81-L155`). `internal/config` performs file/environment translation; `jetkvm.Manager` owns configured devices, admission and device operations; `WebRTCProvider` owns fresh authenticated sessions; `mcpserver` owns MCP registration, schemas and result translation. The primary test seams are small and behavior-oriented: `mcpserver.Device` has seven methods (`internal/mcpserver/server.go:L88-L97`), `jetkvm.SessionProvider` one method, `Session` three operations, and `Decoder` one operation (`internal/jetkvm/manager.go:L111-L126`; `internal/jetkvm/capture.go:Decoder`). Configuration passes concrete `DeviceConfig` into the manager, with constructor validation (`internal/jetkvm/manager.go:L140-L200`). Separate commands implement offline validation, mutation checklist, protocol gates, and upstream drift rather than adding runtime modes.

**Agreement and state.** `CONTEXT.md` describes one product context and a small vocabulary; behavior remains a conventional single process. State: **substantially_satisfied**.

**Gaps/risks/unknowns.** `internal/jetkvm` imports result/request DTOs from `internal/mcpserver`, so the physical-device layer depends on a transport-named package (`manager.go:L14`; `virtual_media.go:L17`). This is modest coupling, not currently demonstrated harm: the interfaces are small, tests are direct, and adding domain/service/provider frameworks would increase navigation cost. The manager combines inventory, status, concurrency, power, HID and media coordination across files; it is still cohesive around one device-control authority. No plugin/provider registry, dynamic tool construction, policy engine, DI container, or multi-context abstraction exists; repository evidence does not justify adding them.

**Acceptance/revisit.** Retain the present shape unless a second non-MCP caller, second provider, or repeated DTO churn produces concrete duplication. Any seam change should reduce dependency direction or test setup measurably and update an ADR; theoretical extensibility is not acceptance evidence.

## 3. Correctness, concurrency, cancellation, errors, and resource lifecycles

**Scope and surfaces.** manager/provider/RPC/signaling/video/capture/media code and their tests; MCP cancellation/error middleware; command shutdown.

**Verified repository evidence.** Process shutdown derives from `signal.NotifyContext`; stdio/HTTP serving receives that context and telemetry gets a bounded final flush (`cmd/jetkvm-mcp/main.go:L72-L78,L141-L155`). Admission is bounded globally, per device, by sessions/captures/decoders, and mutations serialize per device (`internal/jetkvm/manager.go:L128-L138,L202-L220,L385-L468`). Capacity rejects immediately; mutation waiting is context-aware. Each provider operation creates a fresh authenticated WebRTC connection and defers connected-session cleanup (`internal/jetkvm/provider.go:WithSession`; `provider.go:L72-L185`). Signaling frames are bounded, redirects are disabled for device HTTP, and connection/channel failures cancel the session (`provider.go:L119-L155,L187-L205`). RPC calls have per-request deadlines, a 64-entry pending bound, serialized send gate, cleanup on every exit, and classify any failure after `SendText` as unknown (`internal/jetkvm/rpc_session.go:L14-L15,L51-L121,L145-L180`). MCP error translation forces mutation errors to `retryable:false` and unknown unless a valid lower layer proves a different stage (`internal/mcpserver/errors.go:L59-L97`).

Capture has an overall deadline and independent bounded FFmpeg cleanup; media cleanup detaches from caller cancellation but retains its own deadline (`internal/jetkvm/capture.go:captureScreen`; `internal/jetkvm/decoder_ffmpeg.go:Decode`; `internal/jetkvm/virtual_media.go:cleanupStorageFiles`). Tests explicitly cover status cancellation, MCP cancellation, RPC send/response races, pending bounds, session shutdown, signaling close, receiver lifecycle, capture cleanup, upload cancellation and independent cleanup (`internal/mcpserver/cancellation_test.go`; `internal/jetkvm/session_test.go`; `internal/jetkvm/controls_manager_test.go`; `internal/jetkvm/decoder_test.go`; `internal/jetkvm/virtual_media_test.go`).

**Agreement and state.** The product-contract definition of unknown outcome and “never retry possibly delivered mutation” agrees with RPC stage classification and public result translation (`CONTEXT.md:L13`; `internal/jetkvm/rpc_session.go:L89-L120`; `internal/mcpserver/errors.go:L75-L95`). State: **substantially_satisfied**.

**Gaps/risks/unknowns.** No goleak-style retained test, long soak, fault-injected real network run, or real firmware cancellation evidence exists. Detached `context.Background()` is intentionally used for connected-session ownership and cleanup; tests support the design, but its behavior under stalled Pion/OS primitives remains library/runtime dependent. Unknown-outcome semantics prevent unsafe automatic retry but cannot prove whether the physical action occurred; callers still need independent observation. No durable operation journal exists, appropriately for a stateless single-user tool, so a process crash cannot reconstruct physical intent.

**Acceptance/revisit.** Claim leak resistance only after a representative repeated connect/cancel/capture workload with stable goroutine/FD/memory bounds. Preserve the rule that mutation timeout/cancellation after send is unknown and non-retryable; acceptance tests must include lost acknowledgment, late response, close during send, and cleanup timeout.

## 4. Testing, verification, qualification, and performance evidence

**Scope and surfaces.** all `_test.go`, fuzz corpora/manifest/runner, Makefile, CI, protocol gates, compatibility ledger, validation commands and docs.

**Verified repository evidence.** The tree contains broad focused tests and six explicitly inventoried fuzz boundaries enforced by `internal/fuzzpolicy/manifest_test.go` and `testdata/fuzz-targets.json`; the runner uses bounded durations (`Makefile:L55-L61`; `scripts/run-fuzz-targets.py`). Tests cover strict config/privacy, origin parsing, RPC/signaling/video/media codecs, lifecycle/cancellation, malformed stdio/fresh-process recovery, static MCP manifest/capabilities, schema/error/typed result contracts, protocol pins, compatibility ledger, and CLI validators. CI runs race, vet, Staticcheck, govulncheck, fuzz smoke, atomic coverage evidence, minimum Go, official MCP gates, four release cross-builds, and amd64/arm64 container architecture/build checks (`.github/workflows/ci.yml:L12-L100`; `Makefile:L40-L78`). Coverage is review evidence without an arbitrary percentage (`docs/ci-quality.md:L38-L43`).

Evidence classes are explicitly separated. Protocol gates classify every pinned official server scenario and have no expected-failure baseline (`docs/protocol-gates.md:L12-L29`; `testdata/mcp-gates/pins.json`). Compatibility records source review, source drift, and one read-only physical run; that physical entry omitted exact model and firmware and explicitly denies a positive compatibility claim (`docs/compatibility/jetkvm-ledger.json:L59-L80`). The mutation plan is checked in as `dry_run` with `execution_approved:false` (`testdata/mutation-validation-plan.json:L1-L15`). The failed performance benchmark was removed in the studied commit, consistent with requiring a real decision before benchmark machinery.

**Agreement and state.** Strong agreement: docs repeatedly say fixtures/source/cross-builds do not qualify hardware. State: **substantially_satisfied**, with qualification gaps intentionally visible.

**Gaps/risks/unknowns.** No test execution was performed in this research, so green status is not claimed. Cross-compilation proves compilation, not runtime compatibility. Multi-platform Docker builds prove buildability/ELF architecture, not runtime FFmpeg behavior. No exact JetKVM model/firmware compatibility matrix, mutation run, soak, lossy-network suite, resource leak profile, macOS runtime run, container runtime integration, or release-install test exists. There is no accepted latency/throughput/resource objective, representative workload, regression threshold, or hardware performance evidence; therefore resurrecting generic benchmarks would be unsupported.

**Acceptance/revisit.** Preserve the evidence taxonomy. A performance change needs a named decision, representative end-to-end workload, baseline distribution, accepted threshold, controlled environment, and proof the bottleneck influences that decision. A support claim needs exact server build, device model/firmware, runtime/FFmpeg identity, operation list, result, limitations and—separately approved for mutations—observable physical postconditions.

## 5. MCP specification, Go SDK, compatibility, and conformance

**Scope and surfaces.** MCP SDK dependency, server/transport, manifests, gates, provenance docs, CI.

**Verified repository evidence.** The repo pins and documents MCP generation `2026-07-28`, Go SDK `v1.7.0` and inspected commit `bc72835...` (`go.mod:L8`; `docs/protocol-sources.md:L36-L45`). Server construction advertises only tools and intentionally omits list-change notification because the manifest is static (`internal/mcpserver/server.go:L125-L130`). It uses SDK IO transport for stdio and `StreamableHTTPHandler` with `Stateless:true` for HTTP (`cmd/jetkvm-mcp/main.go:L148-L153`; `internal/mcpserver/transport.go:L22-L30`). The checked-in gate pins official conformance `0.2.0-alpha.11` and Inspector `2.2.0`, source commits and SHA-512 npm integrity; the npm lock graph is exact and scripts are disabled (`testdata/mcp-gates/pins.json`; `docs/protocol-gates.md:L20-L29`). Applicable initialization, tools, transport, headers, DNS-rebinding and caching scenarios are gated; unsupported resources/prompts/tasks/input-required behavior is explicitly classified N/A rather than claimed (`testdata/mcp-gates/pins.json`). Inspector exercises initialize/tools-list and the fixture-safe inventory tool across stdio and HTTP.

**Agreement and state.** README/provenance/gates match implementation: official SDK, tools-only, stdio and stateless Streamable HTTP, no deprecated HTTP+SSE. State: **substantially_satisfied**.

**Gaps/risks/unknowns.** This inspection did not execute the external gates or independently verify that `2026-07-28` remained the latest official revision; external-source agents must establish that. Capability metadata/caching behavior partly comes from the SDK and gate expectations, so dependency availability alone must not be treated as server support. The server does not expose resources, prompts, tasks, sampling, roots, elicitation or dynamic tools; those are intentionally absent, not omissions unless the product contract changes.

**Acceptance/revisit.** Protocol support requires a passing exact pinned gate artifact plus focused repository tests and a reviewed spec/SDK diff. Revisit on MCP revision, SDK/conformance/Inspector release or advisory; do not infer a feature merely because the SDK implements it.

## 6. MCP tool-surface design, consequence communication, and compatibility burden

**Scope and surfaces.** `internal/mcpserver/server.go`, `controls.go`, error/manifest tests, tool fixture, README table, product contract/threat model/ADRs.

**Verified repository evidence.** The committed manifest is a static 18-tool array. It includes explicit inventory/status/capture/HID, one-purpose power/wake tools, four one-purpose media mutations plus media status, and a deprecated multiplexed `jetkvm_virtual_media` compatibility tool. Every tool has title/description, input/output schema, and annotations; operation-specific schemas bound keyboard to 4096 ASCII bytes, pointer ranges, capture dimensions, media source lengths and enumerated modes (`internal/mcpserver/server.go:L425-L662`). Power and media tools encode read-only/destructive/idempotent/open-world consequences, and tests lock them (`internal/mcpserver/server_test.go:TestServerPublishesExplicitPowerTools`; `internal/mcpserver/virtual_media_tools_test.go:TestServerPublishesConsequenceCorrectVirtualMediaTools`). Typed results exclude raw media sources; stable errors include version/code/message/outcome/retryable and use tool-result errors rather than transport/protocol failure (`internal/mcpserver/errors.go:L10-L31`; `typed_results_test.go`). The manifest contract detects accidental surface drift (`internal/mcpserver/manifest_contract_test.go`).

**Agreement and state.** README's consequence table agrees with schemas/descriptions/annotations and unknown-outcome implementation. State: **substantially_satisfied**.

**Gaps/risks/unknowns.** Every tool is a compatibility and review commitment. The deprecated multiplexed media tool deliberately remains, meaning duplicate ways to perform consequential operations and conditional annotations (status vs mutation); its removal timing before/after 1.0 is not encoded. Annotations are client hints, not authorization or confirmation. Tool descriptions can warn agents but cannot guarantee comprehension or prevent malicious clients. Schema-validation errors outside specially sanitized media paths may echo rejected client arguments back to the trusted MCP caller, explicitly documented in the threat model (`docs/threat-model.md:L326-L339`).

**Acceptance/revisit.** Before 1.0, review whether each tool has a distinct user decision and observable result, decide a removal/deprecation policy for the compatibility tool, and keep fixture updates explicit. Do not add generic raw RPC to MCP; ADR 0005 confines it to a local CLI with per-call unsafe acknowledgment (`docs/adr/0005-local-only-raw-rpc.md`).

## 7. MCP transports, authentication, authorization, and deployment boundaries

**Scope and surfaces.** command serving, config HTTP fields, handler middleware/tests, README, ADRs 0001/0007, threat model.

**Verified repository evidence.** Stdio reserves stdout for SDK protocol and stderr for lifecycle/telemetry, tested through malformed input, clean EOF, discovery and cancellation (`cmd/jetkvm-mcp/stdio_integration_test.go`). HTTP exposes only `/mcp` POST and `/healthz`; method filtering precedes the stateless SDK handler (`internal/mcpserver/transport.go:L22-L62`). Host/origin admission parses exact configured HTTP(S) origins, rejects missing/unconfigured public hosts and malformed/duplicate/foreign/opaque origins, and requires a present Origin to equal the effective admitted endpoint (`transport.go:L64-L145`). Optional bearer comparison uses constant-time equality and `Cache-Control: no-store`; non-loopback bind without a bearer is rejected (`transport.go:L147-L161`; `cmd/jetkvm-mcp/main.go:L269-L303,L384-L391`). Browser mode is intentionally same-origin and does not grant CORS. Device credentials are separate from the HTTP bearer; device HTTP disables proxy inheritance and redirects (`internal/jetkvm/provider.go:L187-L205`).

**Agreement and state.** README and ADRs accurately state stateless HTTP, loopback preference, reverse-proxy/TLS boundary, exact Host/Origin admission, native clients without Origin, same-origin browsers, bearer limits, and no MCP OAuth (`README.md:L129-L143`; `docs/adr/0007-same-origin-browser-http.md`). State: **substantially_satisfied** for the stated single-principal boundary.

**Gaps/risks/unknowns.** The program provides no TLS termination, OAuth, scopes, per-device principals, rate limiter, distributed quota, audit authorization log, or multi-tenant isolation. Those are explicit deployment/non-goals, not automatic defects, but public non-loopback deployment depends on correct reverse proxy, bearer custody and network controls. A static bearer grants the complete configured tool authority. There is no request-body limit visible outside SDK behavior and no end-to-end production proxy test. Stateless MCP does not mean underlying device operations are consequence-free or independently isolated.

**Acceptance/revisit.** Public docs should make the single trusted administrative principal and complete-authority bearer unmistakable, include exact proxy/Host/Origin/TLS requirements, and reject token passthrough. Add OAuth/policy/multitenancy only after a concrete multi-principal product requirement; otherwise keep them out of process.

## 8. Agentic-AI and MCP-specific threat landscape

**Scope and surfaces.** public tool metadata/schemas, threat model, transport trust, config authority, raw-RPC ADR, telemetry and mutation validation.

**Actual attack paths and controls.** A compromised or prompt-injected client with MCP access can call keyboard/mouse/power/media/wake tools; the server cannot distinguish malicious intent from authorized intent. It reduces blast radius through static reviewed tools, configured device aliases/WOL targets/media roots/media URL origins, bounded schemas/work, explicit destructive annotations/descriptions, no generic MCP raw RPC, and unknown/non-retryable mutation errors (`server.go`; `manager.go`; ADR 0005). Cross-server tool confusion/shadowing is a client composition risk: this server publishes stable `jetkvm_*` names and a manifest but cannot inspect other servers. Tool-description/schema poisoning (“rug pull”) would require a changed binary/dependency/repository because tools are static; manifest tests detect local drift, while unsigned artifacts/Actions leave supply-chain residual risk. Tool-result poisoning is possible through untrusted device state/screen content reaching an agent; typed status and media results reduce raw firmware data, but screenshots intentionally contain arbitrary host pixels and can carry indirect prompt injection (`README.md:L157-L159`; `docs/threat-model.md`). A malicious MCP client can supply sensitive schema values and receive its own validation reflection; transport authorization assumes that caller is trusted. Secret exfiltration through server logs is reduced by closed telemetry fields and sentinel tests, but an authorized client receives private status/screens by design.

**State and boundary mapping.** **Partial.** The repository can **prevent/reduce** dynamic tool mutation, arbitrary WOL/media destinations, raw RPC exposure, blind retry and diagnostic leakage; it can **detect** manifest drift and coarse operation outcomes; it can only **document** screenshots/tool results as untrusted content, human review before consequence, multi-server naming confusion and excessive agency. Prompt-injection detection, model policy, approval UI, tool selection, secret handling across servers and malicious-client containment belong principally to the client/deployment boundary.

**Gaps/unknowns.** Threat model T-01–T-13 is implementation-traceable, but it is primarily server/data-boundary oriented; public client guidance for indirect prompt injection, compromised agents, tool shadowing and multi-server composition is not concise. No signed release/provenance makes rug-pull resistance incomplete. No call-time confirmation exists, appropriately avoiding a fake server-side “human” control, but deployments must supply meaningful approval for destructive operations.

**Acceptance/revisit.** Require an actual architecture path for each AI threat. Reject generic LLM checklists, an in-server “AI firewall,” dynamic policy engine, or multi-agent approval bureaucracy absent evidence. Revisit when remote multi-user auth, dynamic tools, prompts/resources, third-party plugins, or automated mutation execution enters scope.

## 9. JetKVM, WebRTC, FFmpeg, HID, power, wake, upload, and virtual-media boundaries

**Scope and surfaces.** all `internal/jetkvm`, protocol provenance/compatibility, ADRs 0002–0006, config and mutation validation.

**Verified repository evidence.** JetKVM behavior is based on official upstream commit `b3c29a44...`, with exact reviewed auth/signaling/RPC/video/HID/media paths; later `fe77acd5...` changes every declared surface and is recorded `review_required`, not silently adopted (`docs/protocol-sources.md:L5-L34`; `docs/compatibility/jetkvm-ledger.json:L31-L56`). Each operation uses a fresh in-process Pion session. Signaling frame, SDP/candidate/RTP/access-unit/image sizes are bounded and fuzzed; Pion owns DTLS/ICE/SRTP primitives. Device HTTP uses TLS ≥1.2 (with explicit per-device insecure opt-in), no environment proxy and no redirects (`provider.go:L187-L205`).

FFmpeg is discovered with `exec.LookPath`, converted to an absolute path, invoked without a shell through a fixed argument vector, receives H.264 only by pipe, has bounded PNG/stdout and 16 KiB stderr, deadline/kill cleanup and PNG validation (`internal/jetkvm/decoder_ffmpeg.go:L18-L135`; tests in `decoder_test.go`). The child inherits an environment; there is no environment scrubbing or OS sandbox. Local media uses `os.OpenRoot`, rejects absolute/traversal/symlink escapes, requires a nonempty regular file ≤ the configured code maximum, holds the opened descriptor, hashes before and during upload, detects replacement/change, deletes stale partials, and bounds cleanup (`internal/jetkvm/virtual_media.go:L145-L329,L332-L357`; extensive abuse/race tests at `virtual_media_test.go`). Appliance URL fetching requires an exact configured HTTP(S) origin, but intentionally allows configured private/loopback destinations; firmware controls DNS resolution, redirects and routing (`virtual_media.go:L382-L428`; ADR 0006). HID inputs and destinations are enumerated/bounded; WOL uses configured names rather than caller-supplied MAC/IP. Power descriptions expose physical/data-loss consequences.

**Agreement and state.** Defensive code agrees with ADRs and threat model, including explicit residuals. State: **partial** because protocol/security defenses are not physical compatibility.

**Gaps/risks/unknowns.** FFmpeg is a large native decoder attack surface installed from mutable Debian repositories at image build time; no package version/digest/SBOM/sandbox or runtime seccomp profile is supplied. Environment inheritance may affect FFmpeg/library discovery although arguments are fixed. Pion/upstream advisory status requires external research. Exact-origin appliance URL policy cannot stop same-origin redirects, DNS rebinding/changes or a compromised allowed server; no protocol mechanism binds resolved addresses. The upload integrity design protects local consistency but cannot cryptographically attest the appliance's completed content on the reviewed firmware. Physical HID/power/wake/media mutation and broad firmware compatibility remain unqualified.

**Acceptance/revisit.** Do not claim device compatibility without exact runtime/device evidence. Keep URL-fetch residual risk explicit and prefer deployment isolation. Revisit FFmpeg/Pion/JetKVM on every release/advisory or upstream drift. Stronger sandboxing/container restrictions are optional deployment controls only if operationally supported; do not claim them from non-root UID alone.

## 10. Privacy, logging, diagnostics, telemetry, and incident evidence

**Scope and surfaces.** config/error paths, telemetry recorder/call sites/tests, stdio integration, threat model, telemetry docs, validators and artifact policy.

**Verified repository evidence.** Config errors intentionally omit config content, resolved secret values and private paths (`internal/config/config.go:L66-L91`; `internal/config/privacy_test.go`). MCP stdio stdout stays protocol-only; diagnostics and bounded JSONL telemetry use stderr. Telemetry schema contains only schema/correlation/transport/operation/stage/duration/closed code/outcome, excludes arguments/results/device aliases/URLs/paths/typed text/images/raw RPC/errors/credentials/bodies/commands/child output, drops invalid values, and caps duration (`internal/telemetry/recorder.go:L13-L220`; `docs/telemetry.md:L8-L51`). Separate bounded queues reserve terminal events under stage pressure, use one writer goroutine and never block/change operation outcome; shutdown flush is bounded. Tests cover sentinel exclusion, concurrency, slow/failing writers, queue pressure, schema-output failure and both transports (`internal/telemetry/recorder_test.go`; `internal/mcpserver/telemetry_test.go`; `stdio_integration_test.go`). Typed media/status responses redact URLs/paths/raw fields; screenshots remain private data returned to the authorized caller.

**Agreement and state.** Documentation accurately describes implementation and deliberately lossy telemetry. State: **substantially_satisfied**.

**Gaps/risks/unknowns.** The repository cannot control the MCP client's storage of screenshots/status/tool errors, operator redirection of stderr, reverse-proxy logs, GitHub artifact download/retention, or OS/container logs. CI artifact retention is unspecified and inherits repository defaults. Correlation IDs support local reconstruction but there is no identity, device or action parameter by design, so telemetry is not a security audit log and cannot conclusively reconstruct who changed what. Writer failure/drop counters are intentionally absent, so missing incident evidence is expected under pressure. There is no documented retention/deletion cadence or incident evidence handling.

**Acceptance/revisit.** Public docs should state operator responsibilities for stderr, proxy/client logs and CI artifacts, and distinguish diagnostic telemetry from audit. Avoid adding sensitive request logging, high-cardinality identifiers, traces containing arguments, or a second telemetry control plane.

## 11. AI-assisted development and untrusted contribution security

**Scope and surfaces.** AGENTS instructions, GitHub workflow/permissions, Dependabot, manifests/provenance docs, public contribution/security files.

**Verified repository evidence.** Repository agent instructions reserve hardware access for explicit authorization, protect MCP stdout, require product/protocol/ADR review, and demand scoped verification/handoffs (`AGENTS.md`). CI has repository-level `contents:read`, no device/release/deployment credentials, and uses `pull_request`, not `pull_request_target` (`.github/workflows/ci.yml:L3-L10`; `docs/ci-quality.md:L45-L50`). Dependency automation is deliberately low-noise. Static tool manifests, strict fixtures, protocol pins, exact compatibility surfaces, race/fuzz/privacy tests and code reviewable package boundaries make shallow/fabricated changes easier to detect.

**Agreement and state.** Current internal workflow guidance is strong, but it is not a public untrusted-contribution contract. State: **gap**.

**Gaps/risks/unknowns.** No `CONTRIBUTING.md`, public generated-code/AI disclosure guidance, dependency-addition review checklist, license/provenance check, CODEOWNERS, fork policy or malicious-instruction warning exists. Agent instructions themselves are untrusted repository content to any highly privileged coding agent; technical enforcement cannot guarantee agents ignore poisoned issues/files. GitHub Actions allow all actions and do not require SHA pins (live API below), so an automated patch changing workflow dependencies deserves special review. There is no evidence agents receive repository secrets today, but live maintainer/app/runner permissions beyond queried defaults are unknown.

**Acceptance/revisit.** For one maintainer, practical control is human review of exact diff/dependencies/workflow permissions plus the existing tests, not multiple AI approvals. Contributions that add dependencies, change tools/consequences, workflows, generated blobs, network destinations or logging need explicit provenance and threat-boundary review. Reject autonomous merge/release authority and bureaucratic multi-agent sign-off.

## 12. GitHub repository governance, maintainer security, and public-project readiness

**Scope and surfaces.** `.github/**`, LICENSE, README/AGENTS, issue-tracker docs, and live repository API.

**Verified repository/live evidence (accessed 2026-08-15).** `GET /repos/BenDManning/jetkvm-mcp` reported `private:true`, `visibility:private`, default branch `main`, Issues enabled, not archived, and `security_and_analysis:null`. Ruleset and branch-protection endpoints returned HTTP 403: “Upgrade to GitHub Pro or make this repository public to enable this feature”; therefore no protection/rules are claimed. Private vulnerability reporting endpoint returned 404. Automated security fixes returned enabled; vulnerability-alert query succeeded without content. Actions default workflow permission is read and workflows cannot approve PR reviews. The repository has an MIT `LICENSE`, README, internal agent/work-tracking docs, but no tracked SECURITY, CONTRIBUTING, SUPPORT, CODEOWNERS, issue forms, PR template or incident plan. Open roadmap issues explicitly cover dependency policy (#24), hardened deployment (#28), contributor/support/private disclosure contracts (#29), release policy (#30), and verifiable release automation (#31); roadmap #32 remains open.

**State and staging.** **Gap before public release.** Current private-plan protection is unavailable/unknown; some rules become available when public, while account 2FA, app access, deploy/release authority and maintainer recovery were not inspectable. Mandatory multiple reviewers/CODEOWNERS approval would be theater for one maintainer unless an actual second owner exists. CODEOWNERS can still route attention later but cannot create independent review.

**Gaps/risks/unknowns.** No public disclosure route/support boundary/contributor expectations; no verified protected tag/default branch/required checks/signed tag policy; no plan for spam/abuse or maintainer compromise; bus factor is one. Administrator bypass, installed app permissions, environment secrets and 2FA were not available through the queried repository API. Issue/PR history shows active intentional merging but does not prove review quality or branch gates.

**Acceptance/revisit.** Before publicity, publish minimal SECURITY/support/contribution/release ownership contracts and configure the strongest useful free public rules after checking availability: required exact CI checks, no force/delete on main/tags, least-privilege Actions. Do not impose impossible review-count requirements. Revisit issue forms/spam/stale tooling only after external volume exists.

## 13. GitHub Actions and CI/CD security, reliability, and reproducibility

**Scope and surfaces.** sole CI workflow, Makefile, Dependabot, protocol npm lock, live Actions settings.

**Verified repository/live evidence.** One workflow runs on PRs and pushes to main with top-level `contents:read`, job timeouts and fixture-only execution (`.github/workflows/ci.yml:L1-L15`). Jobs support concrete claims: release/minimum toolchain quality, MCP conformance/Inspector, and container architectures. There is no `pull_request_target`, dynamic matrix, untrusted expression interpolated into shell, credentialed checkout, curl installer or publishing step. Protocol npm dependencies have exact versions/URLs/SHA-512 and install with scripts disabled. Coverage/protocol artifacts are sanitized by design. Live API reports Actions enabled, `allowed_actions:"all"`, `sha_pinning_required:false`, default workflow permissions `read`, and `can_approve_pull_request_reviews:false` (`GET /actions/permissions` and `/actions/permissions/workflow`, 2026-08-15).

**Agreement and state.** CI documentation accurately calls the workflow canonical and fixture-only. State: **partial**.

**Gaps/risks/unknowns.** `actions/checkout@v7`, `setup-go@v7`, `upload-artifact@v4`, `setup-node@v4`, and Docker setup Actions `@v4` use mutable major tags, not immutable commits (`ci.yml:L17-L18,L42,L56-L57,L68-L74,L85,L94-L96`). GitHub does not require SHA pins and permits all Actions. `ubuntu-latest` is mutable. No workflow/job concurrency cancellation exists, and artifact retention days are unspecified. `go run` analyzers download code/network dependencies during CI. Caches are enabled for Go; there is no documented cache threat analysis, though jobs have no write/release credentials. Duplicate tests/vet across Make targets increase time but also lane independence; evidence is insufficient to call it harmful. No release/publish workflow, runner hardening, artifact attestations or post-build verification exists.

**Acceptance/revisit.** Pin third-party Actions to reviewed commits with readable version comments or enable repository SHA enforcement when available; restrict allowed Actions to GitHub/verified required publishers if maintainable. Set explicit artifact retention and concurrency. Preserve read-only fork execution and never adopt `pull_request_target` to run untrusted code. Every added job needs a named product/security decision.

## 14. Releases, containers, SBOMs, signing, provenance, and software supply chain

**Scope and surfaces.** GoReleaser, Dockerfile, README installation, releases/API, CI/Issues.

**Verified repository/live evidence.** GoReleaser builds trimpath CGO-disabled Linux/Darwin amd64/arm64 archives and `checksums.txt` (`.goreleaser.yaml:L1-L38`). Docker uses digest-pinned Go `1.26.5-bookworm` and Debian bookworm-slim bases, builds CGO-disabled, has a scratch binary target, installs CA certificates/FFmpeg, runs as fixed non-root UID/GID 10001, and uses exec-form entrypoint (`Dockerfile:L1-L31`). README tells users to verify release archives against checksums and separately install FFmpeg (`README.md:L32-L46`). Live releases API shows exactly one published non-prerelease `v0.1.0` (2026-08-10) with four archives and `checksums.txt`; the REST asset objects expose digest fields, but there are no separate SBOM/signature/provenance assets. Those GitHub-recorded asset digests are useful platform evidence but do not authenticate the release independently of the GitHub repository/account boundary. Open issues #30/#31 acknowledge release policy/automation gaps. No checked-in release workflow or registry publishing definition exists.

**Assurance separation/state.** State: **partial**.

- **Artifact integrity:** checksums detect accidental/corrupt changes only when the checksum channel is trusted; both assets share the GitHub release trust boundary.
- **Build provenance:** absent—no GitHub artifact attestation or SLSA statement binds source/workflow/builder to artifacts.
- **Dependency inventory:** absent—no SPDX/CycloneDX SBOM for binaries/images and no FFmpeg/OS package inventory asset.
- **Signing:** absent—no signed checksums, tag signature policy, Sigstore/Cosign signature or verification identity.
- **Reproducibility/verifiability:** `-trimpath` and exact module inputs help; no two-builder reproduction, buildinfo comparison, SOURCE_DATE_EPOCH policy, or documented verification result exists.
- **Runtime hardening:** non-root and CGO-disabled product binary are positives; Debian+FFmpeg expands the image, with mutable apt packages and no read-only filesystem/capability/seccomp declaration.
- **Publication availability:** binary GitHub release exists; no container registry/multi-arch manifest, rollback/revocation/compromise procedure or installation smoke evidence.

**Gaps/risks/unknowns.** Base digests are immutable but installed packages are resolved at build time; identical Dockerfile can produce a different FFmpeg/runtime. Docker builder Go patch is behind CI. GoReleaser checksum support is configured, but its version/execution environment is not. Artifact attestations/private-plan availability requires external GitHub feature research; do not assume a paid control. VEX has no value until a concrete SBOM/vulnerability disposition workflow exists.

**Acceptance/revisit.** A proportional ladder is: retain checksums → deterministic release workflow/permissions → SBOM per artifact/image → keyless signed checksum/artifact plus identity-bound verification instructions → build provenance attestation → optional reproducibility comparison. Each layer should have a user-verifiable command and compromise response. Do not claim signing from checksums or provenance from an SBOM.

## 15. Framework mapping and pragmatic one-maintainer operating model

**Applicable repository control families.** Without treating scores as goals, the tree already supplies many likely framework-relevant controls: source/module integrity and vulnerability analysis (OpenSSF/SSDF), least-privilege CI, branch-work tracking, code reviewable changes, fuzz/race/static analysis, secure defaults for remote HTTP, strict input/resource bounds, threat model/ADRs, disclosure/release gaps explicitly tracked, pinned container bases, compatibility/protocol provenance, and separation of tests from physical qualification. External research must map exact 2026 framework versions; this file does not turn repository similarities into compliance claims.

**What is already strong and should be retained.** One bounded product context; small internal interfaces; static tools-only MCP surface; exact consequences and typed errors; non-retryable unknown mutations; strict config/origin/path/admission limits; fresh session ownership; privacy-safe stderr telemetry; official pinned protocol gates; source-drift and qualification honesty; read-only CI; low-noise dependency automation; no enterprise policy/governance runtime.

**Genuinely missing before public release.** Minimal SECURITY/contribution/support/release contracts; available branch/tag/required-check rules; immutable Action pins/restriction; a reproducible least-privilege release workflow; SBOM plus signature/provenance appropriate to available GitHub features; exact release verification/rollback/compromise instructions; dependency/build-input policy; and clear client/deployment responsibility for agentic risks. Exact physical support must either be qualified or remain explicitly limited.

**Defer until adoption.** Multiple maintainers/reviewer requirements, issue forms/spam automation, richer compatibility matrix, formal SLA, cross-origin browser identity, per-user auth/scopes, container registry, VEX, continuous fuzzing, extensive soak/performance infrastructure—unless actual users/decisions create the need.

**Monitor rather than implement.** Go/MCP/Pion/FFmpeg/JetKVM advisories and upstream drift; public GitHub feature/plan changes; indirect prompt/tool attacks; dependency/license changes; telemetry loss/operator retention; new client compatibility requirements.

**Explicitly reject as disproportionate now.** An internal OAuth server, policy engine, AI gateway, multi-tenant control plane, plugin/provider framework, dynamic tool registry, generic benchmark farm, compliance-score chasing, multi-agent approval bureaucracy, sensitive request/audit logging, and a separate governance/task platform. None has a demonstrated repository requirement, and several enlarge the trusted computing base.

**Highest-return repository-grounded sequence.** (1) minimal public security/support/contribution/release contracts; (2) immutable/restricted Actions and required checks when public-plan rules become available; (3) deterministic release workflow with least privilege; (4) SBOM + keyless signature/provenance and verification instructions; (5) reviewed dependency/toolchain/FFmpeg policy; (6) resolve or explicitly sunset the deprecated media tool before 1.0; (7) close the JetKVM upstream drift review and retain honest support limits; (8) publish hardened deployment/client-agent boundary guidance. These changes address observed gaps without adding a platform.

**Suggested evidence cadence.** Per PR: scoped tests/race/static/protocol checks appropriate to changed surfaces and human diff/dependency/workflow review. Monthly: grouped dependency/Action review and advisory triage. Per MCP/Go/Pion/JetKVM/FFmpeg revision: source/advisory diff and targeted gates. Per release: clean tagged source, exact toolchains, full validation, artifact/SBOM/signature/provenance verification, install/container smoke test, and recorded rollback target. Periodically after external adoption: review support evidence, issue volume, retention and whether any deferred control now has a real trigger.

## Cross-cutting contradictions and resolutions visible in the repository

1. **“Stateless HTTP” versus stateful physical work:** the MCP transport holds no session, but each call creates temporary WebRTC/device state and may mutate hardware. Resolution: retain “stateless transport,” never imply stateless consequence (`README.md:L137-L152`; ADR 0002).
2. **Idempotent/convergent hints versus lost acknowledgments:** DC/wake operations may intend convergence, but a timeout cannot prove delivery. Resolution: all mutation unknowns remain non-retryable regardless of annotation (`internal/mcpserver/errors.go:L87-L95`; mutation validation).
3. **Exact-origin SSRF reduction versus strong address control:** application validates configured origin, while the appliance resolves/follows behavior outside the server. Resolution: document residual DNS/redirect/private-address risk and use deployment isolation; do not claim complete SSRF prevention (ADR 0006; threat model).
4. **Source/cross-build/fixture pass versus compatibility:** implementation intentionally records these as different evidence. Resolution: no positive model/firmware claim until exact physical qualification (`docs/compatibility/README.md:L1-L15`).
5. **Newer “release” Go versus actual Docker builder:** docs/CI name 1.26.6 while Docker uses 1.26.5. Resolution needed: align or explicitly define different binary/container toolchains before claiming a singular reproducible release (`docs/ci-quality.md:L11-L14`; `Dockerfile:L3`).
6. **Strong CI narrative versus mutable CI dependencies:** permissions and pins for npm/container bases are strong, but Actions use major tags and live SHA enforcement is false. Resolution: do not call CI inputs immutable until Action commits/analyzer graphs are controlled.
7. **Release-ready packaging versus current publication assurance:** GoReleaser/checksums exist, but no automation, SBOM, signature or provenance. Resolution: describe v0.1.0 as checksum-published, not supply-chain attested.

## Unknowns requiring separate evidence

- Latest and superseding 2026 external specifications/advisories/frameworks; handled by external-source research, not inferred here.
- Current CI check results for actual HEAD; this research did not run validation.
- GitHub account 2FA, installed apps, secret/environment access, administrator bypass, tag rules and maintainer recovery.
- Branch/ruleset configuration after the repository becomes public; private endpoints were plan-blocked.
- Exact JetKVM model/firmware and FFmpeg/runtime compatibility, mutation behavior, long-run resource stability and lossy-network outcomes.
- Whether appliance URL fetching follows redirects/re-resolves DNS for each supported firmware.
- Whether released binaries/images can be reproduced and whether future GitHub attestations are available on the maintainer's plan/private/public state.
- Operator/client/proxy log and artifact retention practices.

## Later identical-tree observation

During deep-research batch 2, local `HEAD` and `origin/main` moved to merged commit `71445bc6bf2325e6c683e362393605089c336b63`. Read-only verification showed that both the initially studied `6e52f0027b13f928b768de0feeab4847ef9ca53e` commit and the later merged commit resolve to the exact Git tree `34e8b4451d76821950c23d7c06958d021700f3a7`; `git diff --stat 6e52f00..71445bc` was empty. Findings therefore continue to describe one byte-identical implementation tree. The distinct commit identities are retained so local/remote provenance is not silently conflated.

## Local and live primary-source registry

The complete tracked tree at `6e52f0027b13f928b768de0feeab4847ef9ca53e` is the primary repository source. Especially material evidence is cited inline from `go.mod`, `go.sum`, `Makefile`, `Dockerfile`, `.goreleaser.yaml`, `.github/workflows/ci.yml`, `.github/dependabot.yml`, all `cmd/**`, `internal/**`, fixtures under `testdata/**`, `README.md`, `CONTEXT.md`, `docs/product-contract.md`, `docs/threat-model.md`, `docs/protocol-sources.md`, `docs/protocol-gates.md`, `docs/ci-quality.md`, `docs/telemetry.md`, `docs/mutation-validation.md`, `docs/compatibility/**`, and `docs/adr/**`.

Read-only GitHub first-party API evidence was accessed 2026-08-15 at:

- `https://api.github.com/repos/BenDManning/jetkvm-mcp`
- `https://api.github.com/repos/BenDManning/jetkvm-mcp/rulesets` (403 plan/public-state limitation)
- `https://api.github.com/repos/BenDManning/jetkvm-mcp/branches/main/protection` (403 plan/public-state limitation)
- `https://api.github.com/repos/BenDManning/jetkvm-mcp/private-vulnerability-reporting` (404)
- `https://api.github.com/repos/BenDManning/jetkvm-mcp/actions/permissions`
- `https://api.github.com/repos/BenDManning/jetkvm-mcp/actions/permissions/workflow`
- `https://api.github.com/repos/BenDManning/jetkvm-mcp/automated-security-fixes`
- `https://api.github.com/repos/BenDManning/jetkvm-mcp/vulnerability-alerts`
- `https://api.github.com/repos/BenDManning/jetkvm-mcp/releases`
- `https://api.github.com/repos/BenDManning/jetkvm-mcp/issues?state=open&per_page=100`
- `https://api.github.com/repos/BenDManning/jetkvm-mcp/pulls?state=all&per_page=30`

API absence/403/404 is recorded as unknown or unavailable, not interpreted as proof a control is disabled unless the endpoint explicitly supplied that state.
