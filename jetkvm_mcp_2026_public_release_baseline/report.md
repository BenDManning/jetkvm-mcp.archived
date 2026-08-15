# Evidence-backed 2026 public-release baseline for `BenDManning/jetkvm-mcp`

## Executive verdict

`jetkvm-mcp` already has the right basic character: a bounded, conventional Go server with a static typed tool surface, unusually explicit physical-consequence and unknown-outcome semantics, strict configuration and input boundaries, strong focused tests, privacy-minimizing telemetry, and honest separation of fixtures, conformance, builds, and physical qualification. Preserve that shape. A framework rewrite, embedded policy engine, OAuth subsystem, AI firewall, governance platform, or benchmark program would make this repository worse without addressing a demonstrated failure. [Exact-tree repository evidence](results/repository-evidence.md)

It is not yet ready for a trustworthy public release. The largest blockers are narrower and practical: the wire advertises MCP revisions the product does not claim or test; current release/build patch floors and container evidence need alignment; GitHub Actions use mutable tags with permissive live action policy; the public source/security/support contracts and enforceable public rules do not exist yet; and the release chain provides unauthenticated checksums but no trusted automated builder, artifact SBOM, signature, or provenance. Active HTTP mutations also need lifecycle cancellation during process shutdown. [MCP evidence](results/MCP_specification_Go_SDK_compatibility_and_conformance.json), [Go evidence](results/Go_toolchain_modules_dependencies_and_vulnerability_management.json), [CI evidence](results/GitHub_Actions_and_CI_CD_security_reliability_and_reproducibility.json), [governance evidence](results/GitHub_repository_governance_maintainer_security_and_public_project_readiness.json), [release evidence](results/Releases_containers_SBOMs_signing_provenance_and_software_supply_chain.json), [lifecycle evidence](results/Correctness_concurrency_cancellation_errors_and_resource_lifecycles.json)

The appropriate outcome is a small, high-value release-hardening pass followed by a boring cadence—not a new architecture or administrative system.

## Repository baseline and research date

- **Research date:** 2026-08-15 (America/Detroit). [Research outline](outline.yaml)
- **Expected baseline:** `176ec421f9ee6c801517180e1ad0ec9c84570e8e`.
- **Exact initial HEAD studied:** `6e52f0027b13f928b768de0feeab4847ef9ca53e` on `cleanup/remove-custom-ci-policy`.
- **Exact Git tree studied:** `34e8b4451d76821950c23d7c06958d021700f3a7`.
- **Later observation:** during research, `HEAD`/`origin/main` moved to merged commit `71445bc6bf2325e6c683e362393605089c336b63`; it resolves to the same exact Git tree, so implementation evidence remained byte-identical while commit provenance was recorded separately. [Baseline record](results/repository-evidence.md#later-identical-tree-observation)
- **Initial worktree:** pre-existing untracked `.agents/` and `skills-lock.json`; research created only this research directory. No product code, documentation, workflow, GitHub setting, issue, PR, branch, or release was changed.
- **Evidence boundary:** static repository inspection, focused automated checks, current primary sources, and read-only GitHub API observations. No physical hardware was accessed. Cross-builds, fixtures, fake devices, and upstream source review are not physical qualification.

## Existing strengths worth preserving

- One bounded product context and a conventional `cmd/` + `internal/` Go layout with small behavior-oriented interfaces.
- Static tools-only MCP surface, strict schemas/bounds, typed results, stable errors, and manifest fixtures.
- Explicit destructive/open-world consequences and the central rule that a possibly delivered physical mutation is unknown and never automatically retryable.
- Fresh bounded JetKVM/WebRTC sessions, pending-RPC bounds, staged errors, cleanup deadlines, `os.Root` media confinement, upload-integrity checks, and fixed-vector FFmpeg execution.
- Stdio stdout discipline; stateless HTTP; exact Host/Origin admission; loopback-first binding; same-origin/no-CORS browser posture; one explicit full-authority bearer boundary.
- Risk-directed unit, race, fuzz, subprocess, protocol-gate, cross-build, container, cancellation, and malformed-input checks.
- Maintained product contract, threat model, protocol provenance, compatibility ledger, telemetry contract, mutation validation, CI evidence taxonomy, and ADRs.
- Payload-free bounded telemetry and sentinel tests that protect credentials, paths, typed input, URLs, RPC payloads, and screenshots from diagnostics.
- GitHub-hosted `pull_request` CI with read-only token defaults and no repository secrets exposed to fork code.

## Highest-value actions

| Rank | Action | Risk/value | Effort | Recurring burden |
|---:|---|---|---|---|
| 1 | Make the MCP compatibility claim true: restrict both transports and `server/discover` to 2026-07-28, reject legacy revisions on raw wire, and correct gate/Inspector wording. [Evidence](results/MCP_specification_Go_SDK_compatibility_and_conformance.json) | Removes a false normative promise on safety-relevant tools. | S–M | Low |
| 2 | Align supported Go patch floors and release evidence: current security patch builders/minimum lane, tracked analyzer tool graph, exact release-binary `govulncheck`, and artifact license/notices. [Evidence](results/Go_toolchain_modules_dependencies_and_vulnerability_management.json) | Closes known toolchain/runtime and opaque-build-input gaps. | M | Low–medium |
| 3 | SHA-pin every Action and enforce selected-action/full-SHA policy when public; disable checkout credential persistence, add safe concurrency, explicit retention, and remove duplicate CI work. [Evidence](results/GitHub_Actions_and_CI_CD_security_reliability_and_reproducibility.json) | High supply-chain return for small changes. | S | Low |
| 4 | Establish public source governance: maintainer account/recovery audit, concise SECURITY/SUPPORT/CONTRIBUTING, private reporting/secret scanning, enforced solo-compatible main/tag rules, and compromise response. [Evidence](results/GitHub_repository_governance_maintainer_security_and_public_project_readiness.json) | Protects the sole maintainer and gives users a real disclosure/support contract. | S–M | Low |
| 5 | Automate trusted native releases from protected tags with checksums, artifact SPDX SBOMs, keyless signatures, hosted provenance/attestations, immutable publication, and usable consumer verification. [Evidence](results/Releases_containers_SBOMs_signing_provenance_and_software_supply_chain.json) | Adds authenticated identity, inventory, and provenance to currently mutable unsigned assets. | M | Low |
| 6 | Propagate process cancellation into active HTTP request contexts; cancel, bounded-shutdown, force-close, and join serving before exit. [Evidence](results/Correctness_concurrency_cancellation_errors_and_resource_lifecycles.json) | Prevents process exit from hiding an in-flight mutation's unknown outcome and cleanup. | S | Low |
| 7 | Correct the small public tool/HTTP safety mismatches: annotation hints, typed-text error reflection, remaining input bounds, case-insensitive single bearer parsing, explicit body limit/read timeout, and non-loopback tests. [Tool evidence](results/MCP_tool_surface_design_consequence_communication_and_compatibility_burden.json), [transport evidence](results/MCP_transports_authentication_authorization_and_deployment_boundaries.json) | Fixes caller comprehension, privacy, standards, and DoS boundaries without new machinery. | S–M | Low |
| 8 | Produce honest runtime/device evidence: mandatory real-FFmpeg release smoke and version capture, final artifact/container smoke, JetKVM drift adjudication, and named model/firmware qualification before compatibility claims. [Evidence](results/JetKVM_WebRTC_FFmpeg_HID_power_wake_upload_and_virtual_media_boundaries.json) | Converts build/fixture evidence into narrowly defensible runtime claims. | M | Medium, event-triggered |
| 9 | Add minimal incident and agent-boundary guidance: UTC/version/process telemetry context and loss indication; declare inbound instructions untrusted; document screen-result injection and full-admin caller authority. [Privacy evidence](results/Privacy_logging_diagnostics_telemetry_and_incident_evidence.json), [agent evidence](results/AI_assisted_development_and_untrusted_contribution_security.json) | Improves incident truth and resists agent/contribution authority confusion. | S | Low |

## Before public release

Complete actions 1–9 above, but keep their scope narrow. In particular: publish only claims supported by the exact release candidate; enable public-plan branch/tag rules immediately when visibility changes; verify the release as a consumer would; and preserve the distinction between server controls, client/agent controls, deployment controls, JetKVM firmware behavior, and physical qualification.

## After external adoption

- Add issue/PR forms, moderation/spam automation, CODEOWNERS, or a real independent reviewer only when contribution volume or a second maintainer makes them useful.
- Consider a tested read-only or per-device allowlist profile only if users need unattended agent operation; do not imply it already exists.
- Consider container publication only after there is demand and a plan for Debian/FFmpeg rebuild and advisory triage.
- Add native runtime matrices, scheduled soak, resource-leak evidence, and representative performance thresholds only when a support/performance decision needs them.
- Consider OAuth or multiple principals only after a concrete multi-user deployment requirement; it is not a precondition for the declared single-administrator product.

## Monitor

- Go security patches/support window; MCP spec/SDK/conformance/Inspector revisions and advisories; Pion/FFmpeg/JetKVM advisories and upstream drift.
- Dependabot alerts and update PRs; GitHub action provenance/transitive changes; release workflow identity and attestation verification.
- Pion goroutine/FD behavior under repeated cancellation; appliance URL-fetch redirects/DNS/egress; client handling of annotations, structured content, and multi-server tool identity.
- Contributor/report volume, actual use of the deprecated multiplexed media tool, container demand, and any need for a second maintainer.

## Do not do

- Do not add a DI container, service/domain framework, plugin registry, generated mocks, public Go API, pooled WebRTC sessions, durable mutation journal, sidecar, gateway, or second control plane without a demonstrated requirement.
- Do not add an in-server LLM/prompt-injection scanner, dynamic policy engine, multi-agent approval bureaucracy, AI-contribution detector, blanket AI ban, auto-merge, privileged coding agent, or self-hosted runner for public PRs.
- Do not chase Scorecard/framework scores, duplicate scanners, compliance badges, SLSA levels, reproducibility, VEX completeness, or hardware compatibility claims without the acceptance evidence each claim requires.
- Do not require a fake independent approval from a solo maintainer, add CODEOWNERS ceremony before ownership exists, or create a new governance/task platform.
- Do not resurrect benchmark infrastructure without a named decision, representative workload, baseline distribution, accepted threshold, and controlled evidence.

## Current-state matrix

| # | Workstream | Current state | Priority | Disposition |
|---:|---|---|---|---|
| 1 | [Go toolchain, modules, dependencies, and vulnerability management](#go-toolchain-modules-dependencies-and-vulnerability-management) | partial | P0 | change |
| 2 | [Go architecture, package boundaries, and public API design](#go-architecture-package-boundaries-and-public-api-design) | substantially_satisfied | P3 | retain |
| 3 | [Correctness, concurrency, cancellation, errors, and resource lifecycles](#correctness-concurrency-cancellation-errors-and-resource-lifecycles) | substantially_satisfied | P1 | change |
| 4 | [Testing, verification, qualification, and performance evidence](#testing-verification-qualification-and-performance-evidence) | substantially_satisfied | P1 | change |
| 5 | [MCP specification, Go SDK, compatibility, and conformance](#mcp-specification-go-sdk-compatibility-and-conformance) | partial | P0 | change |
| 6 | [MCP tool-surface design, consequence communication, and compatibility burden](#mcp-tool-surface-design-consequence-communication-and-compatibility-burden) | partial | P1 | simplify |
| 7 | [MCP transports, authentication, authorization, and deployment boundaries](#mcp-transports-authentication-authorization-and-deployment-boundaries) | substantially_satisfied | P1 | change |
| 8 | [Agentic-AI and MCP-specific threat landscape](#agentic-ai-and-mcp-specific-threat-landscape) | substantially_satisfied | P1 | retain |
| 9 | [JetKVM, WebRTC, FFmpeg, HID, power, wake, upload, and virtual-media boundaries](#jetkvm-webrtc-ffmpeg-hid-power-wake-upload-and-virtual-media-boundaries) | substantially_satisfied | P1 | change |
| 10 | [Privacy, logging, diagnostics, telemetry, and incident evidence](#privacy-logging-diagnostics-telemetry-and-incident-evidence) | substantially_satisfied | P1 | change |
| 11 | [AI-assisted development and untrusted contribution security](#ai-assisted-development-and-untrusted-contribution-security) | partial | P1 | add |
| 12 | [GitHub repository governance, maintainer security, and public-project readiness](#github-repository-governance-maintainer-security-and-public-project-readiness) | partial | P0 | add |
| 13 | [GitHub Actions and CI/CD security, reliability, and reproducibility](#github-actions-and-ci-cd-security-reliability-and-reproducibility) | partial | P0 | change |
| 14 | [Releases, containers, SBOMs, signing, provenance, and software supply chain](#releases-containers-sboms-signing-provenance-and-software-supply-chain) | partial | P0 | add |
| 15 | [Framework mapping and pragmatic one-maintainer operating model](#framework-mapping-and-pragmatic-one-maintainer-operating-model) | partial | P1 | simplify |

## MCP and agent threat map

| Concrete path | Actual boundary/assets | Server role | Residual/client or deployment duty |
|---|---|---|---|
| Screen pixels contain indirect instructions; agent reads them and calls HID/power/media or sends data to another server. | JetKVM/host pixels → server screenshot result → MCP client/model → tool authority/other servers. | Bound capture; return typed data; publish static known tools; minimize unrelated data. Cannot determine semantic intent. | Treat results as untrusted content; isolate servers/secrets; require meaningful approval for consequential calls. |
| Prompt-injected or malicious valid client calls a mutation. | Trusted stdio process or full-authority HTTP bearer → physical device/host. | Enforce configured aliases/targets, input bounds, serialization, consequences, and non-retryable unknown outcomes. | Protect caller/bearer; do not equate authentication with user intent; use least-agent authority outside server. |
| Lost acknowledgment after HID/power/media dispatch leads a client to retry. | RPC/WebRTC delivery boundary → physical outcome unknown. | Classify post-send failures as unknown and force `retryable:false`. | Observe state independently; never blind-retry a possibly delivered mutation. |
| Tool shadowing/cross-server confusion selects the wrong `jetkvm_*` tool. | MCP host composes multiple servers and metadata. | Stable prefixed names and fixture-locked manifest; authenticated release reduces rug-pull risk. | Bind approval to server identity plus exact tool/consequence; surface metadata changes. |
| Malicious/compromised build or mutable Action changes tool descriptions/schema after review. | GitHub/Actions/dependencies/release → server binary → MCP metadata. | Manifest tests catch local drift only. | SHA-pin/enforce Actions; protected source/tag; signed/attested release; consumer verification. |
| Untrusted schema/metadata/result causes resource exhaustion or leaks reflected input. | MCP client/device → JSON/schema/errors/logs. | Bounded schemas/results, typed redaction, body/frame/media caps; fix remaining identifier bounds and typed-text reflection. | Apply proxy connection/rate/read limits; keep client renderers/parsers defensive. |
| Appliance fetches an allowed-origin media URL that redirects/re-resolves internally. | Server validates URL string → JetKVM firmware performs DNS/HTTP fetch. | Exact configured origin reduces caller-selected destinations; cannot bind firmware DNS/redirect behavior. | Isolate appliance egress/allowed server; treat same-origin service as trusted; qualify firmware behavior. |

## GitHub and CI/CD control matrix

| Control | Current private state (observed 2026-08-15) | Public/plan constraint | Decision |
|---|---|---|---|
| Main/tag enforcement | Ruleset and branch-protection APIs returned 403: Pro or public required. PR+CI is voluntary. [Live evidence](results/GitHub_repository_governance_maintainer_security_and_public_project_readiness.json) | Public GitHub Free can make applicable repository rules available; verify exact live UI/API after visibility change. [GitHub ruleset availability](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) | On publication, enforce PRs with zero approvals for solo maintenance, required checks, conversation resolution, no force/delete, admin enforcement; protect release tags. |
| Actions authority | Hosted `pull_request`; default token read; cannot approve PRs; zero repo secrets/environments. | Available now. | Retain. Do not use `pull_request_target` or privileged/self-hosted public-PR runners. |
| Action provenance | `allowed_actions=all`; `sha_pinning_required=false`; workflow uses mutable major tags. [Live evidence](results/GitHub_Actions_and_CI_CD_security_reliability_and_reproducibility.json) | Selected-action and SHA policy is a live setting; exact availability must be rechecked at publication. [GitHub Actions security](https://docs.github.com/en/actions/reference/security/secure-use) | Full-SHA pin first-party/trusted actions and enforce selected allowlist/SHA policy. |
| Caches/artifacts | 39 caches using about 4.8 GB; artifact default observed as 90 days. | Retention configurable in workflow/repo. | Safe concurrency/cancellation; explicit 30-day sanitized evidence retention; review cache use/poisoning boundary. |
| Security intake/scanning | Private vulnerability reporting 404; secret scanning disabled; code scanning absent. | Private/public and plan availability differ; public repository unlocks important free controls. | Add SECURITY first; when public enable private reporting and secret scanning/push protection where available. Do not add duplicate scanners solely for score. |
| Maintainer ownership | One admin; recovery/2FA/passkeys not observable; unsigned/mutable v0.1.0 tag/assets. | Account security is outside repository API; immutable releases and rules are visibility/feature dependent. | P0 human account/recovery audit; least-privilege OIDC release; protected tags; compromise drill. |

## Release and supply-chain assurance ladder

| Layer | What it proves | Current evidence | Recommended rung |
|---|---|---|---|
| Checksums | Accidental corruption or comparison against a separately trusted digest. | `v0.1.0` has `checksums.txt`; checksum file itself is unauthenticated. [Release evidence](results/Releases_containers_SBOMs_signing_provenance_and_software_supply_chain.json) | Retain as base layer. |
| Authenticated signature | An identity signed the checksum/artifact digest. | None demonstrated. | Keyless Cosign bundle bound to repository/workflow/tag; publish verification command and identity constraints. |
| Dependency inventory (SBOM) | What modules/packages were observed in each artifact; not that they are safe. | No release SBOM. | Artifact-specific SPDX JSON for each archive; container SBOM only when publishing an image. |
| Build provenance/attestation | Which hosted workflow/source/ref produced a subject digest. [SLSA v1.2](https://slsa.dev/spec/v1.2/) | No attestation/provenance. [Release evidence](results/Releases_containers_SBOMs_signing_provenance_and_software_supply_chain.json) | Public GitHub artifact attestation or equivalent hosted SLSA provenance, verified by consumers; generating without verification has no value. [GitHub attestation verification](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/verify-attestations) |
| Source integrity | Protected stable history/ref and authenticated change attribution. | Private Free rules unavailable; merged PR history exists but enforcement is voluntary. | Public main/tag rules, admin enforcement, immutable release, recorded workflow identity. |
| Reproducibility/verifiability | Independent rebuild matches or explains an artifact. | Not demonstrated; Go/GoReleaser/container inputs are not fully normalized. | Monitor/defer; do not claim reproducibility. Retain exact tool/digest metadata so later comparison is possible. |
| Runtime hardening/availability | How artifact runs and where it can be obtained; separate from provenance. | Non-root digest-pinned container definition, but no image publication; FFmpeg apt version not pinned/evidenced. | Native release first. Publish GHCR only with demand and recurring FFmpeg/base rebuild ownership. |

## Applicable framework crosswalk

| Framework/version | Applicable signal | Repository resolution |
|---|---|---|
| [OSPS Baseline v2026.02.19](https://baseline.openssf.org/) / [OpenSSF Scorecard](https://scorecard.dev/) | Vulnerability intake, maintained dependencies, branch/source protection, pinned dependencies, CI tests, token permissions, release integrity. | Use as a gap lens. Implement the narrow public contracts, pins/rules, and release assurance above; do not chase a numeric score. |
| [SLSA v1.2 Source and Build tracks](https://slsa.dev/spec/v1.2/) | Protected source identity and hosted build provenance are distinct. | No level claim. Add public source/tag rules and hosted artifact provenance with consumer verification; reproducibility is separate and deferred. |
| [NIST SSDF v1.1](https://csrc.nist.gov/pubs/sp/800/218/final) | Prepare organization/project, protect software, produce well-secured software, respond to vulnerabilities. | Existing tests/threat/provenance docs are strong; add concise release/security/incident cadence. SSDF 1.2 draft and 800-218A are not automatically normative here. |
| [CISA Secure by Design](https://www.cisa.gov/securebydesign) | Safe defaults, transparency, vulnerability response, eliminate default credential/exposure hazards. | Retain loopback/no-CORS/bounded defaults and honest qualifications; add disclosure/support and release verification. |
| [OWASP GenAI/Agentic 2025–2026](https://genai.owasp.org/resource/agentic-ai-threats-and-mitigations/) | Excessive agency, tool misuse, indirect injection, data leakage, trust-boundary failures. | Map only concrete screen/client/multi-server paths; most semantic controls belong to the MCP client/deployment. Reject an in-server AI firewall. |
| MCP Security community / official MCP security guidance | Origin/Host/token validation, confused deputy, tool metadata/result trust, least authority. | Existing transport/tool boundaries are strong; fix bearer/limits and document full-admin plus multi-server duties. Community material is guidance, not protocol normativity. |
| AAIF ecosystem/governance | Monitor upstream MCP/agent interoperability and security coordination. | Monitor only. Membership, governance participation, or another framework is not a product control. |

## Authoritative-source contradictions and resolution

- **Expected commit vs studied checkout:** the requested `176ec42` baseline was not checked out. Research recorded initial `6e52f00`; later merged `71445bc` has the same tree. Resolution: key executable evidence to tree `34e8b445…`, preserve both commit identities, and do not attribute deleted CI-policy code to the tree.
- **Repository claims modern-only MCP vs SDK behavior:** documentation says 2026-07-28 only, while SDK v1.7.0 discovery/negotiation advertises legacy revisions. Resolution: wire behavior is authoritative; classify current state partial/P0 and restrict it before repeating the claim.
- **Protocol SHOULDs vs product safety hints:** MCP annotations are non-authoritative hints, yet current keyboard/mouse and wake semantics can mislead callers. Resolution: use conservative consequence hints backed by physical evidence and never treat hints as authorization/intent proof.
- **OAuth guidance vs product boundary:** official HTTP security material discusses OAuth for protected remote deployments, while this product intentionally uses one full-authority pre-shared bearer and no token passthrough. Resolution: retain the explicit single-principal deployment boundary; add OAuth only for a real multi-principal requirement.
- **Source `govulncheck` vs release assurance:** Go guidance supports source analysis, but source analysis does not establish the exact binary/standard library/FFmpeg runtime. Resolution: retain source scan and add exact release-binary/runtime evidence without implying complete vulnerability absence.
- **Attestation availability vs security value:** public GitHub can generate attestations, but official guidance notes attestations need verification policy to matter. Resolution: ship verifier commands and trusted repository/workflow/ref identity, not merely an extra artifact.
- **Framework controls vs proportionality:** OSPS/Scorecard/SLSA surface useful gaps but do not make every control a product requirement. Resolution: adopt only controls tied to actual failure paths and acceptance evidence; reject compliance/score theater.

## Unknowns requiring evidence or decisions

- Exact JetKVM models, firmware, runtime topology, HID/power/wake/media mutation outcomes, cancellation/cleanup, and soak behavior; no physical access was authorized.
- Whether any external user relies on the deprecated multiplexed `jetkvm_virtual_media` tool or legacy MCP revisions.
- Actual client/host behavior for annotations, caching, structured-content fallback, screen-result injection, approval UX, and multi-server identity.
- Pion goroutine/FD stability under long repeated real-device cancellation and the compatibility effect of candidate filtering.
- Firmware-side redirects, DNS changes/rebinding, and upload/content persistence after appliance URL fetch or connection loss.
- Maintainer 2FA/passkey/recovery-token/device posture and private account recovery; repository APIs cannot prove it.
- GitHub feature behavior immediately after public conversion, including exact ruleset/security/attestation settings; re-inspect live state.
- Consumer demand for containers, notarization, multiple maintainers, OAuth/multiple principals, native runtime matrices, performance thresholds, or long-term support lines.

## Recommended steady-state cadence

- **Every change:** focused tests for changed surfaces; manifest/schema/negative/privacy/outcome review for tool changes; dependency identity/license/necessity review for new modules/actions.
- **Monthly:** Dependabot and advisory triage; Go/MCP/Pion/FFmpeg/JetKVM update review; close unverifiable bulk security/contribution noise using one evidence bar.
- **Every release:** clean candidate; full validation; real-FFmpeg and final artifact smoke; exact versions/digests; SBOM/signature/provenance generation and consumer verification; protected tag and immutable release; release-note limitations.
- **Quarterly:** maintainer/release authority and recovery audit; GitHub rules/permissions/cache/artifact audit; upstream protocol/firmware drift and relevant advisory review.
- **Annually or after an incident/material architecture change:** threat model, support/EOL statement, framework crosswalk, compromise drill, and necessity of deferred controls.

## Workstream table of contents
1. [Go toolchain, modules, dependencies, and vulnerability management](#go-toolchain-modules-dependencies-and-vulnerability-management) — State: partial | Priority: P0 | Decision: change
2. [Go architecture, package boundaries, and public API design](#go-architecture-package-boundaries-and-public-api-design) — State: substantially_satisfied | Priority: P3 | Decision: retain
3. [Correctness, concurrency, cancellation, errors, and resource lifecycles](#correctness-concurrency-cancellation-errors-and-resource-lifecycles) — State: substantially_satisfied | Priority: P1 | Decision: change
4. [Testing, verification, qualification, and performance evidence](#testing-verification-qualification-and-performance-evidence) — State: substantially_satisfied | Priority: P1 | Decision: change
5. [MCP specification, Go SDK, compatibility, and conformance](#mcp-specification-go-sdk-compatibility-and-conformance) — State: partial | Priority: P0 | Decision: change
6. [MCP tool-surface design, consequence communication, and compatibility burden](#mcp-tool-surface-design-consequence-communication-and-compatibility-burden) — State: partial | Priority: P1 | Decision: simplify
7. [MCP transports, authentication, authorization, and deployment boundaries](#mcp-transports-authentication-authorization-and-deployment-boundaries) — State: substantially_satisfied | Priority: P1 | Decision: change
8. [Agentic-AI and MCP-specific threat landscape](#agentic-ai-and-mcp-specific-threat-landscape) — State: substantially_satisfied | Priority: P1 | Decision: retain
9. [JetKVM, WebRTC, FFmpeg, HID, power, wake, upload, and virtual-media boundaries](#jetkvm-webrtc-ffmpeg-hid-power-wake-upload-and-virtual-media-boundaries) — State: substantially_satisfied | Priority: P1 | Decision: change
10. [Privacy, logging, diagnostics, telemetry, and incident evidence](#privacy-logging-diagnostics-telemetry-and-incident-evidence) — State: substantially_satisfied | Priority: P1 | Decision: change
11. [AI-assisted development and untrusted contribution security](#ai-assisted-development-and-untrusted-contribution-security) — State: partial | Priority: P1 | Decision: add
12. [GitHub repository governance, maintainer security, and public-project readiness](#github-repository-governance-maintainer-security-and-public-project-readiness) — State: partial | Priority: P0 | Decision: add
13. [GitHub Actions and CI/CD security, reliability, and reproducibility](#github-actions-and-ci-cd-security-reliability-and-reproducibility) — State: partial | Priority: P0 | Decision: change
14. [Releases, containers, SBOMs, signing, provenance, and software supply chain](#releases-containers-sboms-signing-provenance-and-software-supply-chain) — State: partial | Priority: P0 | Decision: add
15. [Framework mapping and pragmatic one-maintainer operating model](#framework-mapping-and-pragmatic-one-maintainer-operating-model) — State: partial | Priority: P1 | Decision: simplify

## Detailed workstream evidence

### 1. Go toolchain, modules, dependencies, and vulnerability management

#### Identity And Scope

**Item Name**

Go toolchain, modules, dependencies, and vulnerability management

**Research Question**

Does jetkvm-mcp at exact HEAD 6e52f0027b13f928b768de0feeab4847ef9ca53e use supported, patched, reproducible Go toolchains and a proportionate system for module integrity, dependency and analyzer updates, vulnerability detection, and license awareness before public release?

**Scope**

This item covers the Go language/toolchain floor and build toolchains; go.mod, go.sum, minimal version selection, checksum authentication, direct/transitive and executable-tool dependency chains; update automation; source and binary vulnerability analysis; standard-library patch exposure; and dependency-license visibility. It covers the source-to-CI/container boundary because Go compiler and analyzer code executes against pull-request content, and the build-to-runtime boundary because the selected standard library is embedded in released binaries. It does not evaluate GitHub Action pinning, container OS/FFmpeg packages, artifact SBOM/signing/provenance, or application package architecture except where those surfaces affect the Go dependency graph.

**Repository Surfaces**

- go.mod lines 1-40 and go.sum (83 lines)
- Makefile lines 1-65, especially STATICCHECK_VERSION, GOVULNCHECK_VERSION, module-verify, staticcheck, govulncheck, ci-minimum, and ci-quality
- .github/workflows/ci.yml jobs test, minimum-go, protocol-gates, and container
- .github/dependabot.yml gomod and github-actions update entries
- Dockerfile ARG GO_IMAGE and module-download/build stages
- .goreleaser.yaml build and archive definitions
- README.md Go prerequisite and build/version statements
- docs/ci-quality.md toolchain matrix and gate descriptions
- docs/product-contract.md lines 171-185 support matrix
- docs/threat-model.md T-12 source-to-delivered-artifact claim
- LICENSE
- Live GitHub repository metadata, vulnerability-alert, automated-security-fix, Dependabot-alert, pull, and dependency-graph/SBOM REST endpoints for BenDManning/jetkvm-mcp, read 2026-08-15

**Applicability Stage**

current_private, before_public_release, after_external_adoption

#### Sources And Authority

**Source Class**

- Repository commit, local files, command output, and live GitHub API observations: repository_evidence.
- Go release policy, toolchain and module references: official_documentation with normative go-command semantics.
- Go vulnerability database entries: official_advisory.
- govulncheck and Staticcheck maintained command documentation: official_documentation.
- GitHub feature and REST API documentation: official_documentation.
- OSPS Baseline v2026.02.19: security_framework.

**Normative Status**

- Go toolchain and module selection/checksum behavior is normative for the go command (MUST-like executable semantics).
- The Go release support window and security patch statements are official project policy/guidance; they do not force this repository's compatibility promise.
- govulncheck findings are observations from a dated scan, not proof that unknown vulnerabilities are absent.
- GitHub feature availability is official service behavior; recommendations to enable or retain features are guidance.
- OSPS controls use MUST within that framework, but framework adoption is not itself a repository or protocol requirement; this report uses them as guidance and signals.

**Source Disagreements**

- Go 1.26 encourages new modules to use a supported earlier language floor, while the security release history requires current patch versions for patched toolchains. Resolution: retain a 1.25 language-family floor only while supported, but set and test the current 1.25 patch, not 1.25.0; build release artifacts with current 1.26 patch.
- docs/ci-quality.md says release CI is Go 1.26.6, while Dockerfile builds the shipped container binary with Go 1.26.5. Resolution: executable configuration is authoritative and the Dockerfile is stale.
- docs/product-contract.md says CI/container use 1.26.5 and Go 1.25 is not exercised; .github/workflows/ci.yml actually uses 1.26.6 and has a minimum-Go 1.25.0 job. Resolution: the contract text is stale and must not be used as evidence of current behavior.
- docs/threat-model.md says there is no dependency vulnerability gate, while CI runs govulncheck. Resolution: there is a reachable-symbol source vulnerability gate, but no all-module/dependency-change gate; govulncheck returned success despite one module-level advisory because affected openpgp symbols were not imported/reachable.

**Source Limitations**

- Known-vulnerability services lag disclosure and cannot detect unknown, malicious-but-unreported, or maintainer-compromise behavior.
- The dated govulncheck scan used the local Go 1.26.5-X:nodwarf5 toolchain rather than the CI image; source scan configuration is material, and recent 1.26.6 advisories may not yet have been represented in the database.
- go list -m -u reports semantically newer modules but does not establish that an indirect upgrade is compatible or independently selectable; direct-parent upgrades and tests remain authoritative.
- The GitHub dependency-graph SPDX export describes the default-branch repository graph, not a specific released binary, and omitted license conclusions for six GitHub Actions.
- License classifiers are evidence aids, not legal advice; aggregated repository licenses do not alone decide which notices or source offers must accompany a particular binary.
- Live API state is point-in-time and notification delivery preferences were not inspected.

#### Repository Evidence

**Exact Baseline Commit**

Studied HEAD: 6e52f0027b13f928b768de0feeab4847ef9ca53e on branch cleanup/remove-custom-ci-policy. This differs from the mission's expected baseline 176ec421f9ee6c801517180e1ad0ec9c84570e8e by one commit, 6e52f00 'cleanup: remove custom CI policy'. The worktree had pre-existing untracked .agents/, skills-lock.json, and jetkvm_mcp_2026_public_release_baseline/ research output. No evidence from the two revisions was silently mixed; repository claims refer to the studied HEAD, with the one-commit diff inspected explicitly.

**Current Repository Evidence**

- go.mod declares exact minimum go 1.25.0, nine direct requirements and 22 explicitly recorded indirect requirements, with no replace, exclude, toolchain, or tool directive. go.sum has 83 checksum lines.
- The full MVS build list obtained with go list -m all contained 46 external modules, including dependency test/tool requirements; govulncheck's application scan considered 31 external modules plus the main module and standard library.
- All nine direct requirements showed no newer version in go list -m -u on 2026-08-15. Several selected transitives had newer versions, including Pion ICE/SRTP/transport/TURN and x/crypto/x/net, but these are controlled by direct dependency requirements and must not be bumped blindly.
- Makefile pins Staticcheck v0.7.0 and govulncheck v1.7.0 by go run package@version. These executable dependencies are authenticated by the public module checksum database when downloaded, but they and their transitive graphs are absent from this repository's go.mod/go.sum, GitHub dependency graph, and gomod Dependabot PRs.
- make tidy uses GOWORK=off go mod tidy -diff; make module-verify runs go mod verify; ci-quality includes race, vet, Staticcheck, govulncheck, fuzz smoke, coverage, and cross-builds.
- CI test and protocol jobs select Go 1.26.6. The minimum job selects Go 1.25.0 with GOTOOLCHAIN=local. Go release history identifies current patched versions as 1.26.6 and 1.25.13.
- Dockerfile GO_IMAGE is digest-pinned golang:1.26.5-bookworm, so the container's static Go binary embeds the immediately superseded 1.26.5 standard library even though primary CI is 1.26.6. The runtime Debian image is separately digest-pinned.
- .github/dependabot.yml runs monthly grouped gomod and GitHub Actions version updates, one open PR per ecosystem. Dependabot merged Go dependency update PRs #55, #56, and grouped #57 plus grouped Actions PR #58 on 2026-08-13.
- Live GitHub API on 2026-08-15 returned 204 for vulnerability alerts, automated-security-fixes enabled=true/paused=false, and zero open Dependabot alerts.
- The live dependency-graph export was SPDX-2.3 with 38 packages: the project, 31 Go modules, and six Actions. Go-module license conclusions were permissive combinations (Apache-2.0, BSD-3-Clause, ISC, MIT, plus Google patent/CC attribution expressions); the six Actions had no license conclusion in the export.
- The exact govulncheck v1.7.0 verbose source scan on 2026-08-15 returned no symbol or package vulnerability affecting the code. It reported one module-level finding, GO-2026-5932 for unmaintained x/crypto/openpgp, but go mod why showed the repository reaches x/crypto through Pion DTLS cryptobyte, not openpgp.
- LICENSE is present and GitHub identifies the project as MIT. GoReleaser archives contain only the binary by default and checksums.txt is separate; no repository-generated third-party license/notice inventory is present.
- Local go env showed GOTOOLCHAIN=local, GOPROXY=https://proxy.golang.org,direct and GOSUMDB=sum.golang.org; these local values are observational and not repository-enforced release settings.

**Implementation And Documentation Agreement**

README's Go 1.25-or-newer statement agrees with go.mod at a language-family level, and docs/ci-quality.md accurately describes the primary 1.26.6 and minimum 1.25.0 jobs plus analyzer pins. However, docs/product-contract.md is materially stale: it says CI/container use 1.26.5 and that Go 1.25 is not exercised. Dockerfile really does use 1.26.5, while CI does not. docs/threat-model.md correctly names go.sum, go mod verify, digest-pinned base images, and absence of SBOM/signatures, but its unqualified statement that there is no dependency vulnerability gate understates the existing reachable-symbol govulncheck gate. No documentation explains that analyzer executable dependencies are outside the tracked application graph or defines a patch-release/update policy.

**Current State**

partial. The repository has a small explicit application graph, checksum verification, current direct dependencies, pinned current analyzers, strong tests, monthly Dependabot version updates, enabled Dependabot alerts/security fixes, and no currently reachable known Go vulnerability. It is not public-release-ready on this workstream until the just-released security patch levels are aligned: minimum CI/go.mod still use 1.25.0, the container builds with 1.26.5, tool dependencies are outside the reviewed graph, documentation contradicts executable state, and release archives lack reviewed license/notice evidence.

**Evidence Missing**

- A govulncheck source scan run under the exact Go 1.26.6 CI toolchain and binary scans of the four exact release outputs.
- A recorded policy decision on whether the source minimum tracks the oldest still-supported Go family and how quickly it moves after a new major release.
- A reviewed inventory of licenses/notices actually required for shipped binaries and containers; the GitHub repository SBOM is useful but is not artifact evidence.
- Evidence that Makefile-pinned executable tool graphs receive Dependabot or equivalent update review.
- Dependabot notification delivery/maintainer subscription state and evidence that update PR review includes upstream release/advisory inspection.
- The public-release date relative to the future stable Go 1.27 release and consequent Go 1.25 support transition.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- A release container built with Go 1.26.5 embeds standard-library code superseded by 1.26.6 security fixes in crypto/tls, encoding/asn1, encoding/xml, html/template, net, net/http, and net/url. A reachable defect can remain in the shipped binary even though CI tests pass under 1.26.6.
- The minimum CI job compiles attacker-controlled pull-request source with Go 1.25.0, thirteen patch releases behind 1.25.13. Compiler/go-command/parser vulnerabilities fixed across those patches can expose the runner or make the compatibility gate unreliable; users following 'Go 1.25 or newer' may also choose the insecure initial patch.
- A compromised or malicious analyzer release at a pinned tag executes in CI through go run. The version is visible in Makefile, but its transitive module changes are not committed, surfaced by the repository dependency graph, or updated by current gomod Dependabot configuration, weakening review of executable build dependencies.
- A new dependency or transitive update is checksum-authentic but malicious, abandoned, unexpectedly licensed, or behaviorally incompatible. go.sum proves content consistency, not trustworthiness, maintenance, license acceptability, or absence of intentionally harmful behavior.
- A reachable dependency or standard-library vulnerability is absent from or delayed in vuln.go.dev/GitHub Advisory Database, or source static analysis misses reflective/unsafe/platform-specific reachability. Green govulncheck and zero Dependabot alerts can therefore be misread as proof of safety.
- A grouped monthly update combines unrelated changes, making regression attribution harder, while open-pull-requests-limit 1 can delay ordinary version updates. Conversely, unreviewed automatic merging could ship a compromised dependency; current evidence shows human merges, not auto-merge.
- A release archive distributes compiled third-party code without a reviewed notice/license bundle. Even when all observed Go dependency licenses are permissive, missing attribution or license text can create legal/release friction and reduce consumer trust.

**Affected Assets And Trust Boundaries**

- Release binaries and containers, which inherit the compiler and standard library used at build time.
- JetKVM credentials, screen/device data, host HID and power authority reachable if a dependency/build compromise changes server behavior.
- GitHub-hosted CI runners and repository read token exposed to compiler, tests, analyzers, and downloaded executable dependency code when processing pull requests.
- Maintainer review authority crossing from Dependabot/upstream release metadata into merged go.mod/go.sum/toolchain changes.
- Module proxy, checksum database, upstream module maintainers, Go vulnerability database, GitHub Advisory Database, and license-classification service boundaries.
- Downstream users relying on source prerequisites, checksums, license terms, and vulnerability claims.

**Plausible Impact**

Worst case is arbitrary behavior in a released server with console/power/media authority, CI runner compromise, credential/screen disclosure, unsafe device mutation, or denial of service. More likely failures are shipping a known-patch-vulnerable standard library, false confidence from incomplete scanner coverage, delayed updates, incompatibility from an over-broad grouped dependency bump, or release/license rework. Documentation drift makes review and support decisions less reliable.

**Likelihood Or Preconditions**

Exploitation of a Go patch vulnerability requires a relevant affected package/path and attacker-controlled network, input, archive, certificate, source, or build condition; applicability of the 1.26.6 fixes to jetkvm-mcp was not individually demonstrated. Supply-chain compromise requires an upstream/tool account or release compromise, malicious new dependency, or review failure. Database lag and documentation drift are routine preconditions and already observable. The very old minimum patch executes only in one CI lane and source builds, but untrusted PR source makes that boundary important. Direct dependencies were current and no reachable known vulnerability was found, reducing immediate dependency-CVE likelihood.

**Existing Controls**

- Explicit go.mod requirements and committed go.sum; no local replace directives or hidden workspace dependency is used by tidy.
- Public Go checksum database authentication by default and go mod verify in both CI lanes.
- Latest-patch Go 1.26.6 in primary and protocol CI, plus GOTOOLCHAIN=local preventing silent minimum-lane switching.
- Pinned release versions for Staticcheck and govulncheck rather than latest/master.
- govulncheck source analysis, vet, Staticcheck, race, tests, fuzz smoke, coverage evidence, and four release-target cross-builds.
- Monthly grouped Dependabot gomod updates; live Dependabot alerts and automated security fixes enabled; recent update PRs were merged.
- Read-only workflow contents permission and no release/device credentials in PR CI.
- Digest-pinned Go and Debian container images, CGO disabled, trimpath builds, and a non-root runtime container.
- MIT project LICENSE and live dependency graph SPDX export with permissive conclusions for all enumerated Go dependencies.

**Residual Risk**

Checksums authenticate known module versions but cannot establish maintainer trust or benign intent. Scanners cover known advisories and particular build configurations only. The immediate residual is avoidable patch skew between CI and the container/minimum lane. Tool graphs remain outside repository review. Monthly updates can leave a bounded exposure window, while emergency security updates depend on GitHub/advisory coverage and maintainer response. License classification is incomplete evidence for release obligations. A single maintainer remains the final review and update bottleneck, which is acceptable if the process stays small and explicit.

**Compatibility Or Semver Effect**

Raising go.mod from 1.25.0 to 1.25.13 narrows source-build compatibility within the same Go language family and must be called out because the go directive is an exact minimum toolchain requirement. Moving to Go 1.26 after Go 1.25 leaves official support is a larger prerequisite change but not an MCP protocol change; before 1.0 it should still receive release-note notice. Updating patch toolchains, analyzer pins, dependency versions, or license artifacts should not alter the public MCP surface, but dependency behavior changes require full compatibility tests.

**Privacy Effect**

No additional runtime data collection is required. govulncheck sends queries to vuln.go.dev but documents that requests contain only module paths already known vulnerable, not source code. Dependency graphs disclose package names/versions to GitHub, already enabled for this private repository and public by nature after release. CI logs and license/SBOM artifacts must avoid embedding private module paths or credentials if private dependencies are ever introduced; none exist now.

**Operational Effect**

Patch alignment is a small build/CI change and removes a misleading difference between tested and shipped standard libraries. Tracking tools in go.mod increases module download/cache size and Dependabot diff volume modestly. Binary vulnerability scans add release-gate time but can operate on already-built four artifacts. Monthly update PRs remain the normal cadence, with out-of-band response for Go/security advisories. A reviewed notice file adds a release artifact but no runtime service.

**Maintainer Effect**

The current grouped monthly updates and two pinned tools are appropriately small for one maintainer. The valuable recurring work is: review one dependency PR and upstream notes monthly, react promptly to Go/security releases, update tool pins when released, and regenerate/review license evidence before releases. Avoid daily bot churn, blanket transitive latest upgrades, or multiple overlapping scanners. Making executable tools visible to the module graph reduces the chance that a forgotten Makefile pin or invisible transitive tool change escapes review, at the cost of a somewhat larger go.mod/go.sum diff.

#### Decision

**Disposition**

change

**Recommendation**

Retain the Go 1.25 source-family floor only while it remains officially supported, the current direct application dependencies, go.sum/sumdb verification, govulncheck, Staticcheck, and monthly reviewed Dependabot updates. Before public release, align every executing/building toolchain to security-patched releases: require/test Go 1.25.13 rather than 1.25.0, build containers with digest-pinned Go 1.26.6, keep primary CI on 1.26.6, and correct the product contract. Treat Go's twice-yearly major release as a revisit trigger: when Go 1.27 makes 1.25 unsupported, move the minimum to the then-current 1.26 patch unless concrete user evidence justifies a short documented exception. Add the two executable tools to Go's native tool dependency graph so versions and sums are reviewable and updateable. Keep reachable source govulncheck as the ordinary PR signal, and scan exact release binaries as release evidence rather than replacing source analysis. Add a small reviewed third-party license/notice inventory to release archives; do not mistake the GitHub repository SBOM for artifact evidence.

**Minimal Practical Change**

- Change go.mod and the minimum CI/docs from 1.25.0 to 1.25.13; preserve GOTOOLCHAIN=local so the compatibility assertion remains honest.
- Change Dockerfile's digest-pinned Go builder from 1.26.5 to the official 1.26.6 image/digest and ensure docs/product-contract.md matches .github/workflows/ci.yml and Dockerfile.
- Declare honnef.co/go/tools/cmd/staticcheck and golang.org/x/vuln/cmd/govulncheck with Go 1.24+ tool directives/requirements and invoke them with go tool, accepting the small main-graph expansion so Dependabot and go.sum cover executable tool code.
- During the release workflow, run govulncheck binary mode against each exact GoReleaser output in addition to the existing source scan; retain human triage because binary mode is less reachability-precise.
- Generate and review a third-party license/notice report for compiled packages, include the project LICENSE and required notices in archives, and fail only on unknown/disallowed licenses rather than adding a broad policy suite.

**Optional Stronger Control**

If external adoption or dependency churn materially increases, add the official GitHub dependency-review action after the repository becomes public, pinned according to the CI security decision, and gate only newly introduced high/critical known vulnerabilities plus explicitly disallowed licenses. Public repositories receive dependency review availability without the private organization paid-feature constraint. A separate verified tool module or prebuilt analyzer checksums could be considered only if native tool directives cause real graph conflicts. For high-assurance releases, retain a vulnerability scan/SBOM for each exact artifact and connect it to signed provenance in the release workstream.

**Rejected Or Overengineered Alternatives**

- Do not vendor the entire module graph solely for security; it transfers patch review/storage burden to one maintainer and does not make code trustworthy.
- Do not adopt Renovate plus Dependabot plus multiple commercial scanners. Native Dependabot, Go vulnerability tooling, and one license check cover the present graph with less duplicate noise.
- Do not auto-merge dependency, analyzer, or Go toolchain updates. Tests cannot evaluate maintainer compromise, semantic drift, release-note warnings, or license changes.
- Do not fail CI on every module-level advisory or every available transitive update. The observed openpgp advisory is not imported/reachable, and transitive versions are selected through direct parents; require a demonstrated path or reviewed risk disposition.
- Do not promise bit-for-bit reproducible Go builds merely from go.sum, trimpath, or fixed toolchains; those controls provide dependency integrity and reduced path variance, not a completed reproducibility study.
- Do not support an already unsupported Go family for speculative users. This is an application, not a widely consumed library; current and previous supported Go families are a proportionate ceiling on compatibility burden.

**Rationale**

This is a consequential device-control server, so a build/dependency compromise inherits unusually high runtime authority. The application dependency graph itself is in good condition: nine current direct dependencies, explicit indirect requirements, checksums, strong tests, active Dependabot maintenance, enabled alerts/security updates, and no reachable known vulnerability in the dated scan. The actionable failures are narrower and concrete. Official Go release evidence shows 1.26.5 and 1.25.0 are superseded security patch levels; executable configuration shows they still build the container and minimum lane. Official Go module tooling now supports tracked tool dependencies, removing an avoidable review blind spot without a new framework. Exact-artifact binary scanning complements source reachability across release configurations. A reviewed license inventory addresses public distribution evidence. These changes deliver more risk reduction than adding more scanners, governance, or abstraction.

**Dependencies Or Prerequisites**

- Official Go image digests for 1.26.6-bookworm and availability of Go 1.25.13 in actions/setup-go.
- Maintainer decision that the minimum follows the oldest officially supported Go family and current patch, with an explicit transition after Go 1.27.
- A test that the selected Staticcheck and govulncheck tool directives remain compatible with Go 1.25.13 and do not create unacceptable module version conflicts.
- Release job or local release procedure that exposes the four exact GoReleaser binaries before publication.
- A chosen, pinned license-report generator and a short explicit allow/deny policy reviewed against the actual dependency set.
- Coordination with release/supply-chain workstream for artifact SBOM/provenance so license and vulnerability evidence is generated once, not duplicated.

**Migration Or Rollout Considerations**

- Land the Go patch and Docker digest updates first; run the full existing minimum, quality, protocol, and container gates before changing dependency versions.
- Update support documentation in the same change so users do not interpret Go 1.25.0 as supported after it is no longer patched.
- Introduce tool directives in a separate reviewable change; compare staticcheck/govulncheck output before and after and inspect the newly added tool transitive graph.
- Add binary scans initially as evidence/non-blocking if output differs from source scanning; make them blocking only after the expected findings and false-positive handling are documented.
- Generate license inventory from the exact release package closure and review once before enforcing; include notices in archives without changing binary behavior.
- When Go 1.27 releases, update the minimum in a normal pre-1.0 release with release-note notice and retain the previous tag for users unable to upgrade.

**Priority**

P0 before public release for the 1.26.6 container builder and 1.25.13 minimum-patch alignment; P1 for tracked tool dependencies, exact-binary vulnerability evidence, documentation correction, and license notices.

**Implementation Effort**

M overall: XS for patch/toolchain/docs alignment, S for tool directives and binary scans, and S-M for accurate license/notice generation and release integration.

**Ongoing Maintenance Burden**

low to medium: one grouped dependency review monthly, prompt Go security patch updates, analyzer review when releases appear, and license/evidence regeneration at release time.

**Confidence**

high for repository state, live GitHub settings, official Go support/patch conclusions, and the immediate changes; medium for exact license obligations and binary-scan incremental findings because those require artifact-specific review.

#### Verification

**Acceptance Evidence**

- go.mod declares go 1.25.13 (or a newer supported decided floor), minimum CI prints that exact local toolchain, primary/protocol CI uses 1.26.6, and Docker build metadata reports Go 1.26.6 from a digest-pinned builder.
- docs/ci-quality.md, README.md, and docs/product-contract.md agree with executable configuration and state the patch/support transition policy.
- go.mod/go.sum (or an equally reviewable native Go tool manifest) contains Staticcheck v0.7.0 and govulncheck v1.7.0 tool dependencies; Dependabot demonstrates it can propose their updates.
- go mod tidy -diff and go mod verify pass from a cold module cache using the intended proxy/sumdb policy.
- govulncheck v1.7.0 source scan under Go 1.26.6 reports no reachable known vulnerabilities, and binary-mode results are retained for each exact release target with any module-only finding explicitly triaged.
- Live Dependabot alerts and security updates remain enabled, zero open alerts are either true or every alert has a documented disposition, and dependency update PRs pass required checks and human review.
- Release archives contain the project LICENSE plus a reviewed third-party notice/license inventory whose package closure matches the exact compiled artifact.

**Proposed Tests Or Checks**

- Run GOTOOLCHAIN=local with setup-go 1.25.13: go version; make ci-minimum.
- Run setup-go 1.26.6: go version; go mod verify; make ci-quality; make protocol-gates.
- Build the container without cache and inspect `go version -m` for /usr/local/bin/jetkvm-mcp to prove the embedded Go version and module list.
- Run govulncheck ./... and govulncheck -mode binary against linux/amd64, linux/arm64, darwin/amd64, and darwin/arm64 release outputs; store database timestamp/version and tool version with results.
- Run go list -m all and compare the graph against go.mod/go.sum after tool directives; ensure no unexpected replacement, private module, pseudo-version, or unsupported Go requirement appears.
- Exercise a synthetic dependency/tool update PR to confirm Dependabot notices native tool requirements and that one grouped PR limit does not suppress security updates.
- Generate the license report from compiled package dependencies, manually inspect unknown/multiple-license results, and compare packaged notices to the report.
- Query live Dependabot alerts immediately before release and record open/dismissed results without treating a zero count as proof of no vulnerabilities.

**Negative Or Abuse Cases**

- Attempt minimum CI with Go 1.25.12 and confirm the exact go directive/tool policy rejects or does not claim that unpatched version.
- Tamper with a cached module file and confirm go mod verify fails; alter a go.sum checksum and confirm module download/build fails authentication.
- Introduce a local replace directive, private module, or untracked workspace and confirm release/tidy policy detects it.
- Add a known vulnerable reachable test fixture dependency and verify source govulncheck fails; add a module-only but unreachable vulnerable package and verify the process produces a visible triage finding rather than either silently ignoring it or blocking without analysis.
- Build a platform-specific vulnerable import to show why one Linux source configuration is insufficient, then confirm the corresponding binary scan detects it if symbol information permits.
- Change a tool directive to a malicious/unreviewed pseudo-version and confirm dependency review/checksum/license evidence makes the change visible to human review.
- Introduce an unknown, reciprocal, or conflicting dependency license and confirm release blocks pending explicit maintainer disposition rather than auto-classifying it as safe.
- Simulate vuln.go.dev or proxy outage: CI must fail clearly for the vulnerability gate while ordinary cached local development behavior is documented; it must not silently report clean.

**Evidence Needed Before Claiming Support**

- Do not claim support for 'Go 1.25' without naming the current minimum patch and support-window policy; passing on 1.25.13 does not qualify earlier vulnerable patches.
- Do not claim a vulnerability-free binary from source govulncheck alone; retain exact tool/database/build configuration and artifact scan results, and phrase the claim as no known detected vulnerabilities at a date.
- Do not claim dependency reproducibility from go.sum alone; identify toolchain, environment, module proxy/checksum policy, CGO/build flags, and exact source revision, with a separate reproducibility comparison if claimed.
- Do not claim complete dependency/license inventory from GitHub's repository SBOM; generate and validate an exact artifact closure and required notices.
- Do not claim an indirect module upgrade is supported because go list -m -u reports it; update direct parents and pass behavioral/protocol/release tests.
- Do not claim analyzer coverage for language/toolchain versions newer than the analyzer release documents or demonstrates.

**Revisit Trigger**

- Any Go security release; update current/minimum supported patch versions promptly.
- Stable Go 1.27 release; under official policy this ends Go 1.25 support and triggers the minimum-family decision.
- Any Go, Pion, MCP Go SDK, websocket, YAML, Staticcheck, govulncheck, or transitive security advisory/retraction/maintainer compromise.
- A new direct dependency, private module, CGO use, build tag, platform target, plugin, generated code tool, or executable CI helper.
- A Dependabot outage/paused state, unresolved alert, update PR backlog longer than one monthly cycle, or evidence grouped updates obscure review.
- First public binary/container release, license dispute, consumer SBOM request, or claim of reproducible/verifiable builds.
- Material external adoption or contributor volume that justifies public dependency-review gating.

### 2. Go architecture, package boundaries, and public API design

#### Identity And Scope

**Item Name**

Go architecture, package boundaries, and public API design

**Research Question**

At exact commit 6e52f0027b13f928b768de0feeab4847ef9ca53e, does jetkvm-mcp use package, interface, configuration, provider, command, and result boundaries that make a consequential but small single-process Go MCP server understandable and testable without creating an unsupported Go library API or speculative abstraction burden?

**Scope**

This workstream examines the compile-time and behavioral architecture of the Go module: command composition, internal package responsibilities and import direction, configuration-to-runtime conversion, the MCP-to-device interface, WebRTC session and FFmpeg decoder seams, typed MCP request/result ownership, error-boundary ownership, test doubles, and ADR-backed architectural decisions. It includes the process/configuration, MCP client/server, device/session, subprocess, and local-validation boundaries insofar as they affect code structure. It does not independently re-evaluate transport security, protocol conformance, concurrency correctness, release automation, or physical compatibility, which belong to other workstreams. The supported public surfaces are the executable, CLI/configuration/MCP wire contracts, validator reports, and artifacts; this repository exposes no supported importable Go library package because implementation packages are under internal/.

**Repository Surfaces**

- go.mod and the complete package/import inventory from go list ./...
- cmd/jetkvm-mcp/main.go: run, parseArgs, serveHTTP, and command composition
- cmd/jetkvm-mcp-validate/main.go: toolCaller, validationSession, and black-box read-only validation
- cmd/jetkvm-mcp-mutation-checklist/main.go, cmd/jetkvm-mcp-protocol-gates/main.go, and cmd/jetkvm-mcp-upstream-drift/main.go
- internal/config/config.go: Runtime, fileConfig, Load, strict decoding, environment resolution, and conversion to jetkvm.DeviceConfig/jetkvm.Limits
- internal/httporigin/origin.go: shared exact-origin value and parsers
- internal/mcpserver/server.go: MCP-owned Device consumer interface, Status, PowerResult, DeviceList, constructors, registration, and tool telemetry middleware
- internal/mcpserver/controls.go: typed HID/capture/media request and result contracts and deadline middleware
- internal/mcpserver/errors.go and internal/jetkvm/errors.go: transport-safe error projection versus device-operation classification
- internal/mcpserver/transport.go: isolated HTTP adapter
- internal/jetkvm/manager.go: DeviceConfig, Limits, Session, SessionProvider, Manager, options, admission, device inventory, and operations
- internal/jetkvm/provider.go: concrete WebRTCProvider and connectedSession
- internal/jetkvm/capture.go and decoder_ffmpeg.go: consumer-owned Decoder seam and concrete subprocess implementation
- internal/jetkvm/hid.go, virtual_media.go, video.go, rpc_session.go, signaling.go, auth.go, debug.go, and telemetry.go
- internal/telemetry/recorder.go: independent privacy-safe event recorder
- internal/protocolgate/*.go and internal/compatibility/ledger_test.go
- internal/mcpserver/manifest_contract_test.go, architecture_decisions_test.go, typed_results_test.go, error_result_test.go, cancellation_test.go, and transport tests
- internal/jetkvm manager/provider/capture/HID/media/session tests and fuzz targets
- CONTEXT.md, README.md, docs/product-contract.md, docs/threat-model.md, docs/protocol-sources.md, docs/ci-quality.md, docs/telemetry.md, docs/mutation-validation.md, docs/adr/README.md, and ADRs 0001 through 0007
- Diff from mission-expected 176ec421f9ee6c801517180e1ad0ec9c84570e8e to studied commit, which removes the custom cmd/jetkvm-ci-check and internal/cipolicy machinery

**Applicability Stage**

current_private, before_public_release

#### Sources And Authority

**Source Class**

- The exact commit, source, tests, fixtures, and maintained repository documents are repository_evidence.
- Organizing a Go module and Go Doc Comments are official_documentation.
- Go Code Review Comments is official_documentation maintained by the Go project, but it is idiomatic guidance rather than a language requirement.
- Package names is official_documentation in the form of a first-party design article.

**Normative Status**

- Repository code and tests are observation for current executable structure; the product contract is repository-owned policy, not an external standard.
- Go's internal-directory import restriction is enforced behavior of the Go toolchain; the recommendation to keep server code internal and commands under cmd/ is guidance.
- Consumer-owned, use-driven interfaces are guidance, not a compiler MUST.
- Package and exported-name comments are official SHOULD-style documentation guidance, not a release-blocking language rule.
- The conclusion to retain a bounded-context package design and reject a generic domain/api/interfaces layer is a repository-specific engineering judgment.

**Source Disagreements**

There is no material contradiction. Generic layered-architecture advice might object that internal/jetkvm imports request/result types from internal/mcpserver, but the current official Go interface guidance puts the Device interface in its consuming package, which is exactly mcpserver. For this one bounded executable, introducing a generic api, types, or domain package solely to reverse that import would conflict with first-party Go guidance against catch-all API/type packages and would add mappings without a demonstrated second consumer. The resolution is to retain the current consumer-owned seam and revisit only if another real consumer requires a transport-independent device service contract.

#### Repository Evidence

**Exact Baseline Commit**

Studied local HEAD: 6e52f0027b13f928b768de0feeab4847ef9ca53e. This differs from the mission's expected baseline 176ec421f9ee6c801517180e1ad0ec9c84570e8e; the exact diff removes cmd/jetkvm-ci-check and internal/cipolicy plus small Makefile/docs/ci-quality.md adjustments. At the initial check, tracked files had no modifications and the pre-existing untracked entries were .agents/, jetkvm_mcp_2026_public_release_baseline/, and skills-lock.json. This result does not treat those research/tooling files as product evidence.

**Current Repository Evidence**

- The module has five cmd/* command packages and seven production internal packages (config, httporigin, jetkvm, mcpserver, protocolgate, telemetry, plus test-only compatibility/fuzzpolicy packages); it has no root or other externally importable library package. This matches Go's official server-project layout guidance.
- cmd/jetkvm-mcp/main.go is the composition root: it parses commands, loads config, constructs the concrete WebRTCProvider and FFmpeg Decoder, constructs Manager, constructs the MCP server, and selects stdio or HTTP. Device and protocol code do not construct the application around themselves.
- internal/config.Load owns YAML/file/environment parsing and converts it once into Runtime containing []jetkvm.DeviceConfig, jetkvm.Limits, and HTTP settings. Manager.NewManager independently enforces runtime invariants. There is deliberate validation overlap at the untrusted configuration and runtime-constructor boundaries, not a reflection-based configuration framework.
- internal/mcpserver.Device is a consumer-owned seven-method interface used by MCP handlers; jetkvm.Manager is the concrete implementation. The interface exactly covers list, status, power, capture, keyboard, mouse, and virtual media. Tests supply local fakes such as recordingDevice and contractDevice; no mocking framework or generated interface layer exists.
- Typed public wire projections live next to their MCP use: mcpserver.Status, DeviceList, PowerResult, CaptureOutput, KeyboardResult, MouseResult, VirtualMediaState, and VirtualMediaResult. jetkvm.Manager returns those types and therefore depends on the MCP adapter, but this avoids duplicate transport/domain projections and mapping code in the only real consumer.
- Device-private transport abstractions remain in internal/jetkvm: SessionProvider and Session support the Manager/WebRTC seam, and Decoder supports the capture/FFmpeg seam. WebRTCProvider and NewFFmpegDecoder return concrete implementations; tests use focused fake providers/sessions/decoders. These interfaces are tied to actual production consumers and extensive cancellation/error/media tests.
- Error ownership is layered: internal/jetkvm/errors.go classifies device-operation failures and delivery outcomes through OperationError methods; internal/mcpserver/errors.go projects only a closed, sanitized tool-error schema and defaults unclassified mutation failures to unknown/non-retryable. The MCP layer does not expose raw firmware errors.
- The command packages are purpose-specific. jetkvm-mcp is the shipped server; jetkvm-mcp-validate is a black-box source-run read-only validator; jetkvm-mcp-mutation-checklist is an offline dry-run parser; protocol-gates and upstream-drift are repository verification tools. docs/product-contract.md explicitly distinguishes their support and packaging status.
- The large jetkvm and mcpserver packages are divided by cohesive files rather than by additional packages: device auth/signaling/session/video/HID/media/provider logic remains under one appliance boundary, while MCP registration/schema/transport/error logic remains under one adapter boundary. Splitting them would require exporting additional implementation details across packages.
- internal/mcpserver/testdata/tool-manifest.json plus manifest_contract_test.go exercises the typed MCP surface over in-memory, subprocess stdio, and stateless HTTP. typed_results_test.go and error_result_test.go protect result and failure projections. Architecture_decisions_test.go checks seven accepted ADRs for status, rationale, code/test/contract evidence, rejected alternatives, and revisit triggers.
- Only cmd/jetkvm-mcp-validate and cmd/jetkvm-mcp-mutation-checklist begin with useful command package comments; the principal server, protocol-gates, upstream-drift, and the production internal packages lack package comments. Many exported internal symbols also lack doc comments. This is a documentation/navigability gap under official Go guidance, not evidence of an unsupported external API.
- Compared with 176ec421f9ee6c801517180e1ad0ec9c84570e8e, the studied tree deletes a bespoke CI wrapper/policy package instead of accumulating another abstraction. That direction is consistent with the stated preference for conventional tooling and low process machinery.

**Implementation And Documentation Agreement**

Implementation substantially agrees with CONTEXT.md and docs/product-contract.md: one bounded product context, a single-process composition root, internal-only Go implementation packages, static typed MCP tools, local-only raw RPC, separate source-run validators, and code/tests as executable authority. ADRs 0001-0007 link to the actual code and tests for transport/auth, fresh sessions, FFmpeg, media integrity, local raw RPC, URL origins, and browser origins; architecture_decisions_test.go verifies their structure and discoverability. The product contract correctly treats CLI/configuration/MCP/result/artifact behavior as compatibility surfaces rather than declaring internal Go packages a library API. The main discrepancy is documentary: official Go guidance says every package should have a package comment, but most production packages and three command packages do not. No documentation falsely claims that package comments or an importable Go API exist.

**Current State**

substantially_satisfied: the current design is proportional, conventional, testable, and intentionally internal. Responsibilities and high-risk seams are visible and backed by typed contracts, focused interfaces, black-box fixtures, and ADRs. The main gap is low-cost package-level orientation/documentation, while the main residual design risk is localized coupling of jetkvm.Manager to MCP-owned types. That coupling is acceptable for the present single consumer and should not trigger a pre-release architecture rewrite.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- A refactor that treats exported identifiers under internal/ as a promised Go API could freeze implementation details, increase SemVer burden, and induce adapters/configuration abstractions that no shipped client uses. The current internal layout prevents external modules from importing them and limits this failure path.
- A generic api/types/interfaces package introduced to remove jetkvm's import of mcpserver could become a dependency magnet, duplicate privacy-sensitive projections, and allow firmware fields to leak into MCP outputs through automatic mapping. Keeping reviewed public result projections at the MCP boundary makes that leakage easier to audit.
- Because Manager methods directly use mcpserver request/result types, a future second consumer could either import an MCP-named package or force an invasive mapping refactor. This is a real but currently conditional change-cost risk, not a current correctness failure.
- The seven-method Device interface means a narrow MCP unit test fake must normally satisfy the whole interface, encouraging embedding of a broad recordingDevice test fake. If the surface grows substantially, test doubles could conceal newly added behavior. At 18 static tools grouped into seven operations, the interface remains reviewable and contractDevice exercises all methods.
- Missing package comments make repository navigation harder for a new human or coding agent. An agent may infer the wrong owner for errors, typed results, origin parsing, or protocol gates, increasing duplicate or misplaced code even when compilation and tests still pass.
- Splitting the large jetkvm package prematurely could expose session, signaling, RTP/H.264, admission, or media internals across package boundaries; those new exported seams would make cross-cutting cleanup and unknown-outcome changes harder to review without proving any reduction in defects.

**Affected Assets And Trust Boundaries**

The design protects the supported MCP/CLI/configuration compatibility contract, the privacy-safe result/error projection, JetKVM credentials and device/host authority, local media paths and content, WebRTC/FFmpeg resources, and maintainer review attention. Relevant boundaries are untrusted configuration to Runtime, MCP JSON to typed handlers, mcpserver.Device to Manager, Manager to SessionProvider/WebRTC session, capture to Decoder/FFmpeg, and source-run validation commands to the shipped server. Architecture mistakes at these seams can broaden physical authority or leak private firmware/configuration data even without a conventional memory-safety flaw.

**Plausible Impact**

The highest plausible architectural impact is an incorrectly mapped consequence or privacy field at the MCP-device seam, producing unsafe client behavior, private-data exposure, or a breaking wire change. More likely impacts are review friction, duplicated validation, test-fake drift, and unnecessary long-term compatibility work. The current typed and internal boundaries materially limit these outcomes; missing package comments mainly affect comprehension rather than runtime safety.

**Existing Controls**

- All implementation packages are internal and all programs are under cmd/.
- One explicit composition root wires concrete configuration, provider, decoder, manager, telemetry, MCP server, and transport.
- Interfaces are small, consumer/use driven, and have real production plus test implementations; no dependency-injection container or mocking framework exists.
- Configuration decoding is strict and Manager construction rechecks device/runtime invariants.
- MCP outputs and tool errors are product-owned typed/sanitized projections rather than raw firmware JSON.
- Static manifest fixtures, typed-result tests, error-result tests, transport tests, fake-provider tests, and fuzz tests protect seams.
- Seven accepted ADRs contain code/test/contract evidence, consequences, rejected alternatives, and measurable revisit triggers, with a test enforcing those properties.
- The product contract explicitly owns compatibility and distinguishes declared support from test/build evidence.
- The studied commit removes a custom CI policy wrapper, reducing locally maintained process abstraction.

**Residual Risk**

Manager remains coupled to MCP-owned operation types, so a future alternate consumer would face mapping or relocation work. The Device interface can become too broad if more tools are added. Large cohesive packages still require disciplined file ownership and navigation, and most packages lack introductory comments. These risks are acceptable before public release because the repository has one bounded product context, one actual device-operation consumer, internal-only implementation APIs, and strong wire-level contract tests. They should be monitored against concrete change pressure rather than pre-solved.

**Compatibility Or Semver Effect**

No pre-release wire or CLI change is needed. Refactoring internal packages without observable behavior would not itself alter the supported Go API because no external Go package is exposed, but any change to MCP schemas/results/errors, configuration, CLI, or artifact behavior remains governed by the repository's explicit SemVer contract. Adding package comments is compatibility-neutral. Creating a public Go package would establish a new compatibility surface and should be a deliberate minor/additive product decision, not an incidental extraction.

**Privacy Effect**

Retaining MCP-owned typed projections keeps redaction decisions adjacent to schemas and handler serialization, reducing the chance that raw firmware values, media URLs/paths, or credentials become generic domain objects reused as public results. Package comments can make that ownership clearer without collecting data. A future mapping layer must preserve the same explicit projections; automatic serialization of device/provider structs would increase privacy risk.

**Operational Effect**

The retained architecture keeps one executable process and direct startup wiring, so diagnosis follows config -> manager -> provider/decoder -> MCP transport without a runtime plugin registry or control plane. Brief package comments have no runtime or CI cost beyond review. A speculative layer split would increase build/test and debugging hops without demonstrated availability benefit.

**Maintainer Effect**

For one maintainer, the current architecture has favorable economics: few packages, no framework, concrete constructors, local fakes, and ADRs for only consequential choices. The cost is that jetkvm and mcpserver are large packages and ownership must be learned from files/documents rather than package comments. Adding concise package comments is a one-time XS task with negligible recurring work. Maintaining an abstract domain layer, generated mocks, a service container, plugin/provider registry, or separate modules would create recurring synchronization and compatibility work disproportionate to current use.

#### Decision

**Disposition**

retain

**Recommendation**

Retain the current cmd/ plus internal/ architecture, the single composition root, consumer-owned Device/Decoder seams, concrete WebRTCProvider and Manager constructors, typed MCP-owned results/errors, strict configuration conversion, and separate black-box validation commands. Treat the MCP/CLI/config/artifact surfaces—not internal Go identifiers—as the supported API. Before or shortly after public release, add concise package comments that state responsibility and sensitive-data boundary for each production internal package and command. Do not move types or split packages merely to make a textbook dependency diagram. Revisit the Manager-to-mcpserver type coupling only when a second real consumer, independently useful reusable library, or repeated change friction provides evidence for a transport-neutral service contract.

**Minimal Practical Change**

Add one short package comment per production package and currently uncommented command, preferably in a doc.go or the natural first file: config owns strict untrusted-file-to-runtime conversion; httporigin owns exact origin/authority parsing; jetkvm owns configured-device sessions and operations; mcpserver owns MCP schemas/results/errors/transport adapters; protocolgate owns pinned conformance evidence parsing; telemetry owns privacy-safe bounded events; each cmd comment states whether it is shipped or source-run. Add one sentence to the maintained product/architecture documentation only if needed to make explicit that no importable Go library API is supported. This is XS, wire-neutral, and creates navigation value without new layers.

**Optional Stronger Control**

If an accepted second consumer needs device operations without MCP concepts, prototype a narrow transport-neutral service API based on that consumer's concrete calls, then adapt it explicitly to mcpserver types and measure the mapping/test burden. Consider splitting only the part with an independently testable lifecycle and stable vocabulary; keep interfaces in each consumer. An import-graph policy check becomes justified only after a documented direction is repeatedly violated, not as a preemptive custom CI framework.

**Rejected Or Overengineered Alternatives**

- Reject a generic domain, api, types, interfaces, common, or util package created solely to invert imports; it would obscure ownership and conflicts with official Go package-design guidance.
- Reject Clean Architecture/hexagonal layer multiplication, a dependency-injection container, service locator, provider/plugin registry, generated mocks, and repository-wide factories; there is one product, one JetKVM provider, one decoder strategy, and direct constructors already provide test seams.
- Reject publishing internal/jetkvm or internal/mcpserver as a reusable public Go SDK before a real external use case exists; it would create a new SemVer and security-review surface.
- Reject splitting files by line-count thresholds or one package per WebRTC/RTP/HID/media concept; package boundaries should follow independent responsibility and lifecycle evidence, not size metrics.
- Reject moving all CLI parsing into a framework or combining source-run validators into the shipped server. Their separate process/package boundaries constrain authority and preserve black-box evidence.
- Reject a new custom architecture linter or import-rule service for the current seven-package graph; ordinary review, go list, tests, and ADR triggers are sufficient.

**Rationale**

The exact tree implements the repository's stated bounded context with conventional Go mechanics: commands under cmd, implementation under internal, direct construction, small actual-use interfaces, and typed boundary objects. Official Go guidance specifically supports internal server packages, cmd layout, consumer-owned interfaces, and avoiding generic API/type packages. The unusual jetkvm -> mcpserver dependency is therefore not by itself a defect: mcpserver is the only real consumer and owns the wire-safe projections, while moving them now would add mappings and broaden the chance of privacy or consequence drift. Contract fixtures and ADR tests make the current seams reviewable. Package comments are the only clear official-guidance miss with a favorable cost/benefit ratio. This yields retain plus a small documentation improvement, not an architectural rewrite.

**Dependencies Or Prerequisites**

The package-comment improvement needs only maintainer agreement on concise ownership wording and should be coordinated with the product contract so it does not accidentally promise a public Go API. Any future service extraction requires an accepted second-consumer requirement, representative call patterns, explicit privacy/error mapping, compatibility classification, and tests proving unchanged MCP manifests/results/errors. No GitHub plan feature, hardware access, or upstream dependency change is required for the retained design.

**Migration Or Rollout Considerations**

Land package comments as a behavior-neutral documentation change and run gofmt plus changed-surface tests/full repository validation under the normal delivery workflow. Do not rename packages or move types in the same change. If a future neutral service seam is justified, first freeze current MCP manifest fixtures, introduce explicit adapters while preserving existing constructors, verify errors/outcomes and privacy projections, and only then remove direct type use. Because internal APIs are not externally importable, source migration is locally controllable, but observable MCP/config/CLI behavior must remain SemVer-classified.

**Priority**

P3: retain as-is for public-release readiness; concise package ownership comments are worthwhile but not a release blocker. Escalate the type-coupling question only upon a real second consumer or repeated maintenance failures.

**Implementation Effort**

XS for package/command comments and an explicit no-public-Go-API clarification; M or larger for a future evidence-backed service/adapter extraction, which is not recommended now.

**Ongoing Maintenance Burden**

low: package comments change only when package responsibility changes, and existing constructors/interfaces/tests already evolve with code. The rejected architectural frameworks would impose medium-to-high recurring synchronization and review burden.

**Confidence**

high for the retained current architecture and identified documentation gap because the complete Go tree, imports, core symbols, tests, contracts, and ADRs were inspected and align with current official Go guidance; medium for predictions about future second-consumer pressure because that depends on maintainer intent and adoption.

#### Verification

**Acceptance Evidence**

- go list ./... continues to show all implementation packages beneath internal/ and all executable entry points beneath cmd/, with no unintended public library package.
- Each production internal package and each command has one concise package comment accurately naming its responsibility and, where relevant, whether it is shipped or source-run.
- README/product contract continue to identify the supported CLI/config/MCP/result/artifact surfaces without promising internal Go packages as an API.
- Existing manifest contract, typed result, error result, config, manager/provider/decoder, validator, and architecture-decision tests pass without fixture changes for the comment-only improvement.
- Any later boundary refactor supplies a before/after import graph, explicit adapter tests, an unchanged MCP manifest unless separately classified, and evidence of the second consumer or repeated maintenance failure that justified it.

**Proposed Tests Or Checks**

- Run gofmt on any added doc.go files and go test ./...; use make verify and make race under the repository's normal delivery policy for the complete change set.
- Use go list -deps or go list -json ./... during review to confirm no new externally importable package or dependency cycle and to inspect new cross-package dependencies.
- Keep internal/mcpserver/manifest_contract_test.go and typed_results_test.go unchanged for a package-comment-only patch; any fixture delta requires the documented compatibility classification.
- For a future service extraction, run focused negative mapping tests that attempt to propagate raw firmware fields, URLs, paths, typed text, and raw errors and assert they remain absent from MCP results.
- Review new interfaces against actual production consumers: each method must be called by the consumer, and the constructor should return a concrete implementation unless an external contract requires otherwise.

**Negative Or Abuse Cases**

- Attempt from a separate module to import github.com/BenDManning/jetkvm-mcp/internal/jetkvm or internal/mcpserver; the Go tool must reject the import under the internal rule.
- Add a new Manager method/tool and verify contractDevice/recording fakes plus the manifest fixture force an intentional review rather than silently defaulting behavior.
- Return unexpected firmware fields, URLs, paths, filenames, query strings, or raw error messages from a fake session and verify only MCP-owned projections leave the server.
- Pass invalid Runtime-equivalent device configuration directly to NewManager and verify constructor validation remains effective even when config.Load is bypassed in tests or future callers.
- Cancel or fail provider/decoder calls and verify errors cross the jetkvm-to-mcpserver seam with the correct failed/not_sent/unknown classification; detailed concurrency behavior is owned by workstream 3.
- Introduce a proposed generic package or interface and require the change to name its actual second consumer, stable vocabulary, and removed duplication; reject it if it only moves existing names.

**Evidence Needed Before Claiming Support**

A public Go library claim requires an explicitly selected import path, package documentation, stable API design, examples, external-package tests, Go compatibility/SemVer policy, vulnerability and privacy review, and at least one real consumer; none exists now. A transport-neutral Manager claim requires adapter and mapping evidence independent of MCP. A reusable provider/plugin claim requires at least two qualified implementations and lifecycle/error compatibility evidence. Architecture and fake-device tests alone do not establish JetKVM model/firmware, OS/runtime, FFmpeg, physical-device, performance, or soak support.

**Revisit Trigger**

Revisit when an accepted issue adds a second non-MCP consumer or reusable Go library; the Device interface grows materially beyond the present seven operations; repeated changes require duplicate mapping or cause privacy/consequence defects; package ownership becomes unclear in review; a new provider or decoder implementation has genuinely different lifecycle needs; the MCP SDK forces transport/domain coupling changes; external contribution volume makes ownership enforcement valuable; or an incident traces to a package/interface boundary. Otherwise review the decision with consequential architecture changes and releases through the existing ADR process, not on a calendar-driven rewrite.

### 3. Correctness, concurrency, cancellation, errors, and resource lifecycles

#### Identity And Scope

**Item Name**

Correctness, concurrency, cancellation, errors, and resource lifecycles

**Research Question**

At exact commit 6e52f0027b13f928b768de0feeab4847ef9ca53e, do jetkvm-mcp's contexts, deadlines, goroutine and channel ownership, fresh WebRTC sessions, pending RPC calls, subprocesses, files, cleanup, shutdown, error taxonomy, and retry rules safely bound work and preserve honest outcomes for consequential device mutations; and what minimal change is required before public release?

**Scope**

This workstream examines request-to-device cancellation propagation; global, per-device, session, capture, decoder, and mutation admission; RPC send serialization and pending-call lifecycle; WebRTC signaling/video pump ownership; FFmpeg process and pipe termination; local-media file and cleanup lifecycles; HTTP and stdio process shutdown; partial multi-step HID/media failures; typed error classification; and leak evidence. It includes MCP-client, process, device/WebRTC, appliance mutation, filesystem, subprocess, and stderr boundaries where lifecycle affects correctness. It does not independently reassess authentication, parser security, physical compatibility, or release supply-chain controls. No physical JetKVM was accessed.

**Repository Surfaces**

- cmd/jetkvm-mcp/main.go: main, run, finishTelemetry, and serveHTTP
- cmd/jetkvm-mcp/stdio_integration_test.go: malformed-input process recovery and shutdown telemetry tests
- internal/jetkvm/manager.go: Manager admission channels, withOperation, runOperation, withSession, tryAcquire, acquireContext, and release
- internal/jetkvm/rpc_session.go and session_test.go: rpcSession, sendGate, pending map, request timeout, Close, pending limit, and dispatch-phase outcome tests
- internal/jetkvm/provider.go, provider_test.go, signaling.go, and session_protocol_test.go: per-operation connectedSession ownership, signaling and RTP pumps, CloseContext, Pion PeerConnection, WebSocket closure, and setup/teardown tests
- internal/jetkvm/video_receiver.go, video.go, video_test.go, capture.go, and capture_test.go: single capture waiter, receiver closure, bounded H.264 assembly, 30-second capture deadline, decoder admission, cancellation precedence, and permit-release tests
- internal/jetkvm/decoder_ffmpeg.go, decoder_test.go, and decoder_real_test.go: CommandContext, WaitDelay, bounded stdin/stdout/stderr, cancellation, timeout, and PNG validation
- internal/jetkvm/hid.go, controls_manager_test.go, and internal/mcpserver/controls.go: serialized mutation sequences, pressed-key/button cleanup, partial-sequence classification, and consequence-aware handlers
- internal/jetkvm/virtual_media.go, upload_test.go, virtual_media_test.go, and docs/adr/0004-virtual-media-integrity-and-cleanup.md: file ownership, hashing, upload phases, detached bounded cleanup, and ambiguous partial artifacts
- internal/jetkvm/errors.go, internal/mcpserver/errors.go, error_result_test.go, typed_results_test.go, and telemetry.go: stable code/outcome/retryable taxonomy and redaction boundary
- internal/telemetry/recorder.go, recorder_test.go, and docs/telemetry.md: bounded queues, one writer goroutine, nonblocking producers, terminal-event fallback, flush deadline, and synthetic goroutine/memory test
- docs/product-contract.md, docs/threat-model.md, docs/mutation-validation.md, docs/adr/0002-fresh-in-process-webrtc-sessions.md, and docs/adr/0003-ffmpeg-screenshot-decoding.md
- Makefile and .github workflows insofar as make race and test execution cover these paths

**Applicability Stage**

current_private, before_public_release

#### Sources And Authority

**Source Class**

- The exact commit, its code, tests, fixtures, ADRs, product contract, threat model, and mutation checklist are repository_evidence.
- context, net/http, os/exec, the Go context article, and the race-detector article are official_documentation from the Go Project.
- RFC 9110 is a normative_specification for HTTP retry semantics; its principle is applied by analogy to JetKVM RPC mutations rather than misrepresented as an MCP requirement.
- The Amazon Builders' Library article is a maintainer_case_study from the operator of real distributed APIs, not a normative standard.
- Pion package documentation is official_documentation for the exact upstream lifecycle API used by this repository.

**Normative Status**

- Repository source and tests are observation of current behavior; docs/product-contract.md and docs/mutation-validation.md are repository-owned policy.
- Go context propagation, cancel-function use, net/http Shutdown behavior, os/exec WaitDelay behavior, and Pion Close/GracefulClose behavior are official API contracts or guidance, not external product certification.
- RFC 9110 section 9.2.2 says clients SHOULD NOT automatically retry non-idempotent requests without idempotency or proof of non-application, and proxies MUST NOT do so. JetKVM MCP calls are not HTTP methods, so the RFC supplies an authoritative safety principle rather than direct protocol conformance language.
- AWS client-token design is guidance and observation. It demonstrates what would be needed to make retries safe but does not imply this small server MUST implement durable idempotency.
- The repository-specific decisions to retain conservative unknown outcomes, cancel active HTTP request contexts on process shutdown, and defer durable idempotency are engineering judgments.

**Source Disagreements**

There is no substantive disagreement about ambiguous non-idempotent retries: RFC 9110 discourages them without proof or idempotency, AWS explains the same lost-response dilemma, and the repository conservatively returns unknown/non-retryable. There is a lifecycle tradeoff between Go net/http's graceful Shutdown, which intentionally leaves active handlers running, and this product's need to stop device work on process termination. Resolve it by giving request contexts a server-lifecycle parent and then calling Shutdown with a bounded grace period: handlers receive cancellation and perform their existing bounded cleanup while the server still drains responses. Pion documents GracefulClose as the goroutine-waiting primitive, but it has no Context and can block; the repository instead closes peer, signal, RPC, and video state and bounds its own pump wait. Retain that design until a repeated-session test demonstrates a Pion-owned leak, rather than replacing bounded shutdown with an unbounded call.

#### Repository Evidence

**Exact Baseline Commit**

Studied local HEAD: 6e52f0027b13f928b768de0feeab4847ef9ca53e, not the mission-expected 176ec421f9ee6c801517180e1ad0ec9c84570e8e. At this delegated item's initial check, the shared worktree showed go.sum modified plus pre-existing untracked .agents/, jetkvm_mcp_2026_public_release_baseline/, and skills-lock.json; the go.sum delta was concurrent research-tool checksum material and was restored by the coordinating agent before completion. No source file was modified, and product conclusions use the committed source at HEAD. Only this required research result was created.

**Current Repository Evidence**

- manager.go uses bounded buffered channels for global operations (default 16), per-device operations (4), sessions (8), captures (2), decoders (2), and a one-slot per-device mutation semaphore. General capacity fails immediately as busy/not_sent; only same-device mutation ordering waits, and acquireContext observes caller cancellation. Every acquired permit has an adjacent defer or explicit rollback.
- rpc_session.go gives each call a bounded timeout, serializes DataChannel SendText with sendGate, caps pending requests at 64, protects the pending map with a mutex, removes waiters on every pre-send/send/wait exit, and releases all waiters on idempotent Close. Buffered waiter channels prevent response delivery from blocking after the caller leaves.
- RPC outcome classification is phase-aware: validation/admission/cancellation before send is not_sent; SendText failure, timeout/cancellation after send, session loss, malformed success response, or unclassified response failure is unknown; an explicit firmware RPC error is failed. session_test.go exercises these phases and verifies pending removal and the admission cap.
- mcpserver/errors.go projects only stable version/code/message/outcome/retryable fields. Mutation errors are never retryable; unclassified mutation failures fall back to unknown. Read cancellation, timeout, device-unavailable, and no-signal failures can be failed/retryable. error_result_test.go verifies unknown mutations are neither completed nor blindly retryable and that raw causes do not leak.
- provider.go creates a fresh authenticated HTTP client/cookie jar, Pion PeerConnection, RPC session, signaling WebSocket, and optional videoReceiver per operation. connectedSession owns a cancellable lifetime, a WaitGroup for the repository-created signaling/RTP pumps, closeOnce, and explicit closure of video waiter, RPC waiters, WebSocket, peer, and idle HTTP connections. ADR 0002 correctly describes this ownership and rejects pooling.
- CloseContext performs cancellation and all close signals before waiting for pumps. Its wait is bounded by the supplied operation/setup context, so teardown cannot extend an already-expired tool deadline. provider_test.go verifies that property and authentication-failure idle-connection cleanup. The remaining WaitGroup-wait helper goroutine can outlive CloseContext if an upstream read never unblocks; current code closes both reads first, but no repeated hostile-session leak test proves eventual exit.
- capture.go imposes a server-owned 30-second deadline across session setup, fresh-frame capture, decode, PNG validation, and result construction, while preserving an earlier caller cancellation. videoReceiver permits one waiter, removes it on cancellation, and releases it on Close. capture tests cover setup cancellation/deadline precedence and admission-permit release.
- decoder_ffmpeg.go uses an absolute validated executable, exec.CommandContext, fixed non-shell arguments, a scrubbed environment, bounded input/output/stderr buffers, a 15-second deadline, and 250ms WaitDelay. This bounds the direct child and pipe wait; it does not establish a process group or sandbox, which is acceptable only because FFmpeg is a trusted deployment prerequisite.
- hid.go serializes keyboard/mouse mutations per device and, if a key or mouse button may remain pressed, uses a new two-second background cleanup context before the session is closed. Multi-call failures after any prior dispatch are promoted to unknown. controls_manager_test.go covers release after cancellation and partial-sequence unknown outcomes.
- virtual_media.go confines and owns opened files with defers, hashes before/during/after upload, classifies partial sequences conservatively, and uses context.WithoutCancel only underneath a new two-second cleanup timeout to abort/delete partial artifacts. Tests cover cancellation during hashing, post-upload cleanup, and partial deletion timeouts. Appliance-side cleanup remains best effort and cannot prove deletion after an ambiguous loss.
- telemetry.Recorder uses one writer goroutine and bounded queues; producer calls never block, and tool-terminal events have a separate fallback queue. Close has a caller deadline. A permanently blocked stderr writer can strand that one goroutine and lose queued evidence, but cannot create one goroutine per operation; synthetic load checks bounded goroutine/memory growth.
- stdio serving receives the signal-derived top-level context directly. HTTP serving does not: serveHTTP constructs http.Server without BaseContext/ConnContext, and on ctx cancellation it calls Shutdown with a fresh five-second context. Go's documented Shutdown semantics do not cancel active handlers. If a long HTTP mutation/upload is active, Shutdown can time out, run returns an error, main calls os.Exit(1), and the operation is terminated without a delivered unknown result or guaranteed detached cleanup.
- No HTTP shutdown test exists; the only shutdown integration hit located was TestRunCanceledContextFlushesShutdownTelemetry for stdio. This makes the HTTP lifecycle gap both implementation-visible and evidence-visible.
- Focused verification on 2026-08-15 passed: go test ./internal/jetkvm ./internal/mcpserver ./internal/telemetry ./cmd/jetkvm-mcp. This is useful regression evidence, not hardware, soak, or all-schedule proof.

**Implementation And Documentation Agreement**

Implementation and documentation agree on bounded admission, cancellation propagation inside an accepted tool call, fresh per-operation sessions, pending-RPC limits, FFmpeg deadlines, detached-but-bounded HID/media cleanup, and unknown/non-retryable mutation outcomes. The product contract and threat model accurately state that cancellation cannot undo a physical effect and that no durable idempotency journal exists. ADR 0002's claim that closing an operation tears down repository-owned pumps is supported structurally and by bounded-teardown tests but lacks repeated hostile-session leak evidence. The material disagreement is HTTP process shutdown: documentation broadly claims cancellation propagation and bounded shutdown, but serveHTTP does not parent request contexts to the signal context, and net/http Shutdown explicitly does not interrupt active handlers. Therefore a signal does not reliably reach active HTTP tools before forced exit.

**Current State**

substantially_satisfied: Core call-level concurrency, cancellation, resource ownership, partial-failure classification, and non-retry semantics are unusually explicit and well tested for a small server. One concrete before-public-release gap remains: process cancellation is not propagated into active Streamable HTTP request contexts, so the five-second shutdown path can forcibly terminate consequential work without its typed unknown outcome or full cleanup. Pion-owned goroutine termination and hardware-side reconciliation remain unknown rather than demonstrated gaps.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- An HTTP client begins a slow virtual-media upload or a power/HID call, then SIGTERM arrives. Shutdown leaves the active handler running; after five seconds it returns deadline exceeded, main exits, and the process can die after appliance dispatch but before acknowledgement, result delivery, or cleanup. The caller sees transport loss and may retry, duplicating a physical effect.
- A JetKVM applies power/reset/HID/media mutation but the DataChannel response is lost. The server cannot distinguish success from failure. Current unknown/non-retryable output correctly prevents the server from asserting completion or safety to retry; independent observation is required.
- A multi-report keyboard string, click press/release pair, or upload/mount sequence fails after an earlier step. Retrying the whole logical tool could duplicate typed characters, clicks, uploads, or mounting. Current mutationSequenceError and detached release/cleanup reduce harm but cannot reverse already-applied device/host effects.
- A signaling or RTP read fails to unblock after session close due to an upstream defect. CloseContext returns at the operation deadline while its WaitGroup-wait helper remains. Repetition could accumulate goroutines. Current close ordering makes this unlikely, but current tests intentionally inject a pump that exits only later and do not prove real hostile Pion/WebSocket eventual termination.
- A telemetry destination blocks forever. Operation paths keep moving and bounded queues drop events, but the sole writer goroutine and flush cannot complete. This trades diagnostic completeness for availability and is acceptable for stderr-only best-effort telemetry, provided it is not described as an audit log.
- A trusted FFmpeg executable misbehaves or a replaced executable forks descendants that inherit descriptors. CommandContext and WaitDelay kill/bound the direct child and pipe wait, but do not manage a descendant process group. A malicious replacement already crosses the trusted-executable boundary and should be addressed by deployment isolation rather than portable process-tree machinery in this workstream.

**Affected Assets And Trust Boundaries**

- Physical host power, reset, wake, keyboard, mouse, boot-media, and storage state across the MCP client -> server -> authenticated JetKVM boundary
- Truthfulness of caller-visible completion, failed, not_sent, unknown, and retryable fields
- JetKVM credentials/cookies and fresh WebRTC signaling, ICE, DataChannel, RTP, and HTTP session state
- Process availability and bounded goroutines, channels, pending RPC map entries, sockets, HTTP connections, file descriptors, files, subprocesses, and memory
- Local media content and appliance-side partial artifacts across the filesystem/upload boundary
- Diagnostic completeness at the process -> stderr consumer boundary

**Plausible Impact**

The highest impact is duplicated or ambiguously completed physical mutation: data loss from repeated reset/power-off, unintended UI activation or repeated typed text, inconsistent virtual-media/storage state, or a host left with pressed input. Shutdown truncation can also prevent callers from receiving the designed unknown warning and can skip best-effort cleanup. Resource leaks can degrade availability or eventually exhaust the single process. Read-only timeouts primarily reduce availability. Telemetry loss impairs incident reconstruction but does not change device state. No reviewed path establishes memory safety or physical compatibility merely from these controls.

**Existing Controls**

- Strict positive hierarchical admission limits and immediate busy/not_sent rejection without an unbounded operation queue
- Cancelable one-at-a-time per-device mutation sequencing
- Fresh WebRTC session ownership per operation with closeOnce, session cancellation, explicit RPC/video/signal/peer/HTTP cleanup, and owned pump WaitGroup
- Per-RPC deadline, serialized sends, 64 pending-call maximum, waiter removal on all exits, and Close release
- Server-owned 30-second capture bound and bounded H.264/PNG/decoder resources
- CommandContext, fixed FFmpeg arguments/environment, bounded buffers, and WaitDelay
- Deferred file closure, root-confined file opens, size/hash/reopen checks, and detached two-second media cleanup
- Detached two-second HID key/button release after cancellation
- Stable typed error outcomes with every mutation non-retryable and unknown used after possible dispatch
- Focused cancellation/timeout/partial-sequence/permit-release tests, race validation in the repository quality command, and a limited telemetry goroutine/memory synthetic test
- Operator mutation checklist requiring stop, independent observation, and no retry on unknown

**Residual Risk**

The active-HTTP-request shutdown gap is unacceptable before public release because it bypasses the product's strongest safety communication exactly during process termination. Even after fixing it, a response can still be lost after a device effect and before client receipt; without appliance-supported idempotency tokens or a durable device-correlated operation record, exactly-once physical mutation cannot be promised. Best-effort HID/media cleanup can itself time out or have an unknown outcome. Fresh sessions reduce cross-call state but add repeated lifecycle exposure. Pion internal goroutine termination and real hardware behavior remain unqualified. These residuals must remain documented and reconciled operationally rather than hidden behind retries.

**Compatibility Or Semver Effect**

Preserving the existing error schema and conservative unknown semantics is no-action compatibility. Parenting HTTP request contexts to server lifecycle and making shutdown deterministic is a compatible correctness fix if normal request behavior, five-second bound, response taxonomy, and CLI exit contract remain unchanged; if shutdown exit status or grace duration is documented as stable and deliberately changed, classify that separately under the product contract. Adding durable idempotency keys, operation-status resources, or a journal would alter tool schemas and semantics and should not be slipped in as a patch.

**Privacy Effect**

Cancellation and cleanup changes need not collect new data. Retaining bounded structured telemetry avoids sensitive request logs. A leak/soak test should use synthetic values and inspect counts or sanitized goroutine stacks, not credentials, typed text, paths, URLs, screenshots, RPC bodies, or device identifiers. Durable idempotency/audit storage would create a new retention and privacy boundary and is therefore rejected absent a real requirement.

**Operational Effect**

A lifecycle-parented HTTP server should stop accepting work, cancel active handlers, permit their bounded cleanup and typed response attempt, and then finish within the existing grace period; this makes container/systemd termination more predictable. Focused repeated-cancellation tests add modest CI time. Preserving fail-fast admission and no internal retry avoids retry storms and queues. Operators still need independent read-only reconciliation after unknown physical outcomes.

**Maintainer Effect**

The existing explicit channels, mutexes, contexts, closeOnce values, and small interfaces are locally comprehensible and should be retained. The HTTP fix and one integration test are low recurring burden. A deterministic repeated-session lifecycle test is more valuable than adopting a broad goroutine-leak framework immediately. Durable journals, transactions, generalized retry middleware, circuit breakers, process supervisors, or pooled session recovery would impose state migrations, privacy policy, incident handling, and substantial concurrency expertise on one maintainer without upstream device support.

#### Decision

**Disposition**

change

**Recommendation**

Retain the current bounded-admission, fresh-session, phase-aware error, non-retryable unknown-outcome, FFmpeg WaitDelay, and detached bounded-cleanup designs. Before public release, connect Streamable HTTP request contexts to a server-lifecycle context, cancel that lifecycle on SIGTERM/SIGINT, then perform bounded http.Server.Shutdown and a deterministic final Close fallback if grace expires; ensure the Serve goroutine is always joined. Add focused tests proving an in-flight HTTP handler receives cancellation, its cleanup is attempted within its own bound, Shutdown does not leave Serve running, and a possibly dispatched mutation is never reported completed or retryable. Add a repeated real-fixture session cancellation/close test that asserts repository-owned pumps and pending calls return; treat Pion-owned leak behavior as monitor-until-demonstrated rather than introducing a new framework. Continue to require independent reconciliation and never automatically retry unknown physical mutations.

**Minimal Practical Change**

In serveHTTP, establish an explicit request-lifecycle context through http.Server.BaseContext (or an equivalent top-level handler context), retain a cancel function, and on process-context completion cancel requests before or together with bounded Shutdown. If Shutdown reaches its deadline, call Server.Close to force connection release and still receive/join the buffered Serve result before returning. Add one in-flight-handler integration test for cancellation/exit and one outcome assertion for a mutation canceled after dispatch. Keep the existing five-second grace unless test evidence shows bounded HID/media cleanup cannot fit.

**Optional Stronger Control**

If real repeated-session tests or production pprof evidence shows Pion-owned goroutines survive PeerConnection.Close, evaluate the exact pinned Pion version's GracefulClose in a design that cannot block process shutdown, and upstream any defect. If a future JetKVM firmware/API supplies durable idempotency tokens and authoritative operation status, consider opt-in idempotent retry for only those precisely defined operations. If outside adoption produces sustained leak incidents, add a narrowly configured leak checker or scheduled soak job with explicit baselines; do not gate every unit test preemptively.

**Rejected Or Overengineered Alternatives**

- Automatic retry of power, reset, HID, upload, mount, or unmount after timeout/cancellation/transport loss: rejected because no end-to-end idempotency proof or never-applied proof exists.
- A durable operation journal, transaction coordinator, exactly-once claim, generalized saga engine, or internal message queue: rejected because the appliance does not participate atomically and the machinery creates new storage, privacy, recovery, and migration burdens without making physical effects transactional.
- Persistent or pooled WebRTC sessions: retain ADR 0002's rejection until an accepted latency objective and evidence justify cross-call lifecycle complexity.
- Replacing bounded PeerConnection.Close with unconditional unbounded Pion GracefulClose: rejected because public shutdown must remain bounded and current evidence does not demonstrate a Pion leak.
- A blanket third-party goroutine-leak framework in every package, continuous pprof server, or production control plane: rejected now; focused deterministic ownership tests and incident-triggered profiles are more proportional.
- Portable process-group/session-tree supervision for trusted FFmpeg on every platform: defer unless real descendant leakage appears; a compromised FFmpeg already requires deployment isolation and supply-chain remediation.
- Longer global timeouts or queued admission as a reliability fix: rejected because they increase resource retention and do not resolve ambiguity.

**Rationale**

The repository already implements the essential distributed-safety rule correctly: once a consequential request may have crossed the device boundary, failure is unknown and never retryable. RFC 9110's normative retry rule and AWS's lost-response/idempotency analysis support that decision; without an appliance-correlated durable token, stronger exactly-once language would be false. Go's context and os/exec APIs support the repository's call-level design, and the focused tests pass. The remaining defect is narrowly evidenced by serveHTTP and net/http's documented semantics: Shutdown does not cancel active handlers, and no BaseContext ties them to the signal. Fixing that seam restores the existing safety taxonomy during shutdown at low cost. Broader lifecycle frameworks would add more burden than verified risk reduction.

**Dependencies Or Prerequisites**

- Maintainer confirmation that the existing five-second HTTP grace period and current CLI exit semantics are intended compatibility surfaces
- A fake/in-memory MCP device operation that blocks until request cancellation and records cleanup without physical hardware
- Use of the exact pinned MCP Go SDK and Go toolchain in the HTTP integration test
- For any Pion lifecycle change, a reproducing test against pinned github.com/pion/webrtc/v4 v4.2.18 and review of upstream release/advisory state
- For any safe-retry expansion, documented JetKVM firmware support for durable idempotency and authoritative operation reconciliation; absent that, no retry

**Migration Or Rollout Considerations**

Land the HTTP lifecycle test first or with the fix, exercise clean idle shutdown, in-flight read cancellation, in-flight post-dispatch mutation cancellation, a handler that exceeds grace, and repeated startup/shutdown. Preserve the wire error version and mutation retryable=false. Document that termination can still produce client-side transport loss and therefore requires no retry plus reconciliation. Run focused tests, make race, and make verify. Roll out without new configuration if the existing five-second constant remains sufficient; only expose a configurable grace period if real deployment evidence requires it, because configuration creates a permanent support surface.

**Priority**

P1: propagate and bound active HTTP request shutdown before public release; the call-level unknown-outcome and lifecycle controls themselves should be retained. Repeated Pion leak evidence is P2 monitoring/testing unless a reproducer appears.

**Implementation Effort**

S: one HTTP lifecycle context/fallback adjustment plus focused integration tests; M only if an upstream Pion leak is reproduced and requires redesign.

**Ongoing Maintenance Burden**

low: maintain a small shutdown integration test and existing phase/outcome tests; investigate Pion/FFmpeg lifecycle only on dependency updates, advisories, or observed leak evidence.

**Confidence**

high for the repository's call-level behavior and the HTTP shutdown gap because both follow directly from source and official net/http semantics; medium for absence of Pion-owned leaks and physical cleanup because no soak or hardware evidence exists.

#### Verification

**Acceptance Evidence**

- A focused HTTP integration test starts an admitted handler, cancels the server lifecycle, observes the same request context become canceled, observes bounded cleanup invocation, and proves serveHTTP and the Serve goroutine return within the grace bound
- A grace-expiry test proves Server.Close fallback releases the listener/connections and no Serve goroutine remains
- A post-dispatch mutation shutdown test yields unknown/retryable=false when a response can be delivered, and never yields completed or retryable; client transport loss is documented as unknown
- Repeated real-fixture WebRTC operations under cancellation leave rpcSession.pendingCount()==0, repository pump completion, receiver.waiter==nil, and stable file-descriptor/goroutine evidence within a justified tolerance
- go test for changed packages, make race, and make verify pass on the exact changed commit; no physical-compatibility claim is derived

**Proposed Tests Or Checks**

- Add TestServeHTTPShutdownCancelsActiveHandlerAndJoinsServe using a blocking fake Device and a real Streamable HTTP request
- Add a shutdown-time mutation phase table: canceled before admission => not_sent; canceled after fake dispatch => unknown; confirmed device error => failed; every mutation retryable=false
- Add a grace-expiry fake handler that ignores cancellation until released, verify forced Close and deterministic return, then release test resources to avoid test-induced leaks
- Repeat WebRTC fixture connect/call/cancel/close enough times to detect monotonic repository-owned goroutine, pending-call, or connection growth; use eventual synchronization and goroutine profiles for diagnosis rather than brittle exact global counts
- Retain RPC tests for 64 pending calls, cancellation while waiting for sendGate, late responses, duplicate/unsolicited IDs, and Close racing with HandleMessage under make race
- Retain FFmpeg timeout/cancellation tests and, only if a real supported platform needs it, add a helper that holds inherited pipes to confirm WaitDelay returns
- During an explicitly authorized hardware qualification window, inject response loss around one expendable mutation, stop without retry, and reconcile through an independent read-only path

**Negative Or Abuse Cases**

- SIGTERM before provider dispatch, during DataChannel SendText, after appliance application but before acknowledgement, during detached cleanup, and while the HTTP response is being serialized
- Concurrent same-device mutations with the waiter canceled, plus unrelated-device operations proving no cross-device serialization
- Pending RPC map at capacity, sendGate holder cancellation, late reply after timeout, malformed/duplicate/unsolicited reply, and Close concurrent with a reply
- Signaling WebSocket stalls, duplicate answer, excessive queued ICE candidates, RTP read stall, track close, and connection failure during CloseContext
- Keyboard press acknowledged but release lost; mouse-down acknowledged but mouse-up lost; upload begun but body/response lost; mount requested after upload then acknowledgement lost
- FFmpeg direct child ignores cancellation, output pipe remains open, output exceeds limit, stderr floods, and executable disappears/replaces after startup validation
- stderr writer blocks forever and event queues saturate; verify operations remain nonblocking and telemetry is described as lossy
- HTTP handler ignores cancellation past grace; verify force-close, joined Serve, nonzero/selected exit semantics, and no false completed result

**Evidence Needed Before Claiming Support**

Before claiming reliable shutdown, retain the HTTP cancellation/join/fallback tests under race. Before claiming leak-free operation, provide repeated real Pion/FFmpeg lifecycle evidence with goroutine/file-descriptor profiles and an explicit workload; ordinary unit-test success is insufficient. Before claiming a mutation is safe to retry or exactly once, require appliance-supported durable idempotency correlated to authoritative status and fault-injection evidence across lost-response phases. Before claiming device compatibility or cleanup guarantees, retain explicitly authorized model/firmware-specific physical evidence; fake sessions and cross-builds do not qualify hardware.

**Revisit Trigger**

- Any incident involving a repeated physical effect, lost acknowledgement, stuck key/button, partial media artifact, shutdown timeout, or monotonic goroutine/file-descriptor growth
- A Pion, coder/websocket, MCP Go SDK, Go net/http, or FFmpeg release/advisory that changes Close, cancellation, transport, or subprocess behavior
- Introduction of persistent sessions, server-side MCP sessions, background operations, queues, retries, multiple principals, or more than one process
- JetKVM firmware adds durable operation identifiers/idempotency tokens or authoritative mutation-status reconciliation
- Uploads or other valid operations routinely exceed the shutdown grace period in measured deployments
- Outside adoption creates enough concurrency or long-lived deployment evidence to justify scheduled soak/leak testing

### 4. Testing, verification, qualification, and performance evidence

#### Identity And Scope

**Item Name**

Testing, verification, qualification, and performance evidence

**Research Question**

At exact commit 6e52f0027b13f928b768de0feeab4847ef9ca53e, does jetkvm-mcp have proportionate evidence for correctness, concurrency, malformed input, cancellation, subprocess and protocol behavior, supported builds, containers, real FFmpeg, and JetKVM compatibility; where must the repository continue to distinguish fixture correctness, protocol conformance, build evidence, runtime compatibility, physical-device qualification, soak testing, and performance evidence before public release?

**Scope**

This workstream examines the complete automated test suite, fuzz inventory and corpora, race execution, coverage production, vet/static analysis as verification context, subprocess stdio tests, loopback HTTP tests, MCP contract fixtures and official protocol gates, cross-compilation, container builds, the real-FFmpeg smoke test, compatibility ledger, read-only hardware validator, mutation dry-run checklist, telemetry synthetic-load check, and all documentation claims about evidence. It covers the process-to-MCP, MCP-to-device, network-to-device, WebRTC/media-to-FFmpeg, filesystem/media, and physical-host consequence boundaries only as testing and qualification questions. It does not treat source inspection, fake sessions, dependency capability, cross-compilation, container construction, or an unattributed historical device run as positive runtime or physical compatibility. Live mutation, physical hardware access, production load, soak, and new benchmark machinery are excluded from this research-only run.

**Repository Surfaces**

- Makefile targets test, race, vet, staticcheck, govulncheck, coverage, verify, protocol-gates, fuzz-smoke, fuzz, ci-minimum, ci-quality, and container-verify
- .github/workflows/ci.yml jobs test, minimum-go, protocol-gates, and container
- docs/ci-quality.md, docs/product-contract.md, docs/protocol-gates.md, docs/telemetry.md, docs/mutation-validation.md, docs/compatibility/README.md, README.md, and CONTEXT.md
- All 290 top-level Test functions and all eight Fuzz functions under cmd/** and internal/**
- cmd/jetkvm-mcp/stdio_integration_test.go and cmd/jetkvm-mcp/main_test.go
- internal/mcpserver/manifest_contract_test.go, transport_test.go, cancellation_test.go, origin_test.go, typed_results_test.go, error_result_test.go, architecture_decisions_test.go, and threat_model_test.go
- internal/jetkvm/admission_test.go, capture_test.go, controls_manager_test.go, decoder_test.go, decoder_real_test.go, manager_test.go, provider_test.go, provider_video_test.go, session_test.go, session_protocol_test.go, signaling/video/RPC/media fuzz tests, upload_test.go, virtual_media_test.go, and typed_results_test.go
- internal/config config and privacy tests plus FuzzLoadConfig; internal/httporigin tests plus FuzzParseOrigin and FuzzParseAuthority
- internal/telemetry/recorder_test.go including concurrent output and synthetic bounded-goroutine/memory checks
- cmd/jetkvm-mcp-protocol-gates and internal/protocolgate tests; testdata/mcp-gates/pins.json and npm/package-lock.json
- cmd/jetkvm-mcp-validate and its tests; docs/compatibility/jetkvm-ledger.json and internal/compatibility/ledger_test.go
- cmd/jetkvm-mcp-mutation-checklist, its POSIX/Linux boundary tests, testdata/mutation-validation-plan.json, and docs/mutation-validation.md
- testdata/fuzz-targets.json, scripts/run-fuzz-targets.py, and checked testdata/fuzz corpora
- Dockerfile, .goreleaser.yaml, config.example.yaml, and the four Makefile cross-build commands
- Diff from mission-expected commit 176ec421f9ee6c801517180e1ad0ec9c84570e8e to the studied commit, which removes the bespoke CI policy wrapper while retaining conventional gates

**Applicability Stage**

current_private, before_public_release, after_external_adoption

#### Sources And Authority

**Source Class**

- The exact commit, code, tests, workflow, fixtures, ledger, and maintained repository documents are repository_evidence.
- Go testing, fuzzing, race-detector, coverage, and PGO pages are official_documentation.
- The Model Context Protocol conformance repository is official_documentation and executable upstream test tooling, not the normative specification itself.
- Docker multi-platform guidance is official_documentation from the build-tool publisher.

**Normative Status**

- Repository tests and local command results are observation of the exact tree and the executing environment; repository documentation is project policy and stated support scope.
- The Go testing API defines how tests, fuzz targets, and benchmarks execute; using race, fuzz duration, integration coverage, benchstat, or representative workloads is official guidance rather than an external release MUST.
- MCP conformance scenario results are observations against a pinned executable suite. Normative MCP behavior remains owned by the applicable specification, and a scenario marked not applicable must have a repository-specific justification rather than being silently counted as a pass.
- Docker documentation describes build mechanisms and their limitations; it does not certify this repository's target runtime compatibility.
- The recommendation to require evidence labels and a non-skipping FFmpeg/package smoke before a public release is repository-specific SHOULD-level judgment based on the product contract, not a universal standard.

**Source Disagreements**

There is no substantive conflict between primary sources and the repository's evidence taxonomy. Go's race documentation recommends realistic exercised workloads because the detector observes only executed paths, while this repository runs race primarily over fixtures; the resolution is to retain the strong race gate but not call it production concurrency proof. Docker describes multi-platform images as runnable variants, while the repository correctly says its cross-build/container checks establish build intent rather than native runtime qualification; actual native execution is required before a target-specific runtime claim. The official MCP conformance framework offers broad composite scenarios, but many require diagnostic fixture tools or capabilities the product intentionally does not expose; the repository resolves this by pinning four applicable blocking scenarios, giving explicit not-applicable reasons and focused replacement tests, rather than adding conformance-only product tools. Coverage tooling can produce a percentage, but neither the Go sources nor repository history supplies a product risk threshold; the existing review-evidence policy is therefore retained instead of inventing a gate.

#### Repository Evidence

**Exact Baseline Commit**

The exact studied revision is 6e52f0027b13f928b768de0feeab4847ef9ca53e. It differs from the mission-expected 176ec421f9ee6c801517180e1ad0ec9c84570e8e by removing cmd/jetkvm-ci-check and internal/cipolicy plus simplifying Makefile/docs/ci-quality.md. At this subtask's check, local HEAD was 71445bc6bf2325e6c683e362393605089c336b63, whose tracked tree is byte-for-byte identical to 6e52f0027b13f928b768de0feeab4847ef9ca53e; all static repository reads were explicitly addressed to the stipulated commit. Pre-existing untracked entries were .agents/, jetkvm_mcp_2026_public_release_baseline/, and skills-lock.json and are not product evidence.

**Current Repository Evidence**

- The exact tree contains 290 top-level ordinary Test functions across five commands and internal packages, eight native Go fuzz targets, and no Benchmark function. `go test ./...` and `go test -race ./...` passed locally on 2026-08-15; these are current local results, not retained release or hardware qualification.
- Makefile ci-minimum runs format, tidy-diff, module verification, tests, vet, and one-second fuzz smoke using Go 1.25.0 with GOTOOLCHAIN=local in CI. ci-quality adds race, pinned Staticcheck, pinned govulncheck, atomic coverage evidence, and four release-target cross-builds under Go 1.26.6.
- .github/workflows/ci.yml defines read-only PR/push jobs with explicit 15-30 minute job timeouts: the release test lane, Go 1.25 minimum lane, MCP protocol gate lane, and Buildx container lane. The release lane redundantly executes ordinary tests and vet after race/vet because `make verify` itself depends on test and vet; this costs time but does not add a different evidence class.
- Unit and integration fixtures cover strict configuration and privacy failures; Host/Origin parsing; authentication and redirect behavior; WebRTC signaling and malformed SDP; RPC message framing and response errors; H.264 depacketization and bounded video assembly; admission limits and panic/error permit release; HID release after cancellation; virtual-media path, symlink/race, integrity, cleanup, URL, and cancellation cases; typed results and sanitized errors; and telemetry concurrency/backpressure/shutdown.
- cmd/jetkvm-mcp/stdio_integration_test.go builds/runs a real subprocess and checks malformed JSONL shutdown/recovery, clean EOF, stdout protocol purity, stderr diagnostics, telemetry, and cancellation shutdown. internal/mcpserver tests exercise in-memory and loopback stateless HTTP, including propagation of HTTP cancellation to the tool handler.
- internal/mcpserver/manifest_contract_test.go compares one reviewed tool/discovery/result fixture over in-memory, subprocess stdio, and stateless HTTP transports. It validates structured successes against advertised output schemas and captures representative success, operational-error, and protocol-error envelopes. This is product wire-contract regression evidence, not by itself official protocol conformance.
- testdata/mcp-gates/pins.json pins official conformance 0.2.0-alpha.11 by package version, commit, npm integrity, MCP revision, and scenario inventory digest; gates tools-list, dns-rebinding-protection, caching, and http-header-validation with no expected failures. It separately pins Inspector 2.2.0 for stdio/HTTP discovery and fixture-safe device listing. Sanitized artifacts omit raw child output and endpoints. Many upstream scenarios are explicitly not applicable because the product lacks the capability or the scenario requires special fixture tools.
- The fuzz manifest covers config loading, authority/origin parsing, RPC response decoding, SDP decoding, H.264 access-unit depacketization and stream assembly, and allowed media URL parsing. internal/fuzzpolicy verifies exact inventory and corpus privacy. scripts/run-fuzz-targets.py validates manifest values, gives every target a fresh temporary GOCACHE, disables ordinary tests for the mutation run, uses one worker, and enforces bounded duration. All eight one-second targets passed locally on 2026-08-15; the Makefile also offers 30 seconds per target locally.
- The suite includes extensive deterministic cancellation and terminal-race tests around RPC send state, capture setup/frame/decode deadlines, FFmpeg cleanup, admission waits, provider teardown, upload/hash cleanup, HTTP request cancellation, process-group termination, and HID release. It has one broad synthetic telemetry check using runtime.NumGoroutine and heap deltas, but no repository-wide goroutine-leak harness or long-running session soak.
- internal/jetkvm/decoder_real_test.go uses the discovered local `ffmpeg` to generate H.264 with libx264 and decode it through the production decoder. It skips when FFmpeg is absent, so normal CI success does not itself prove that this integration ran. A focused uncached run passed locally on 2026-08-15. The fake executable tests separately validate arguments, pipes, output bounds, malformed output, deadline cleanup, and environment handling.
- Makefile verify cross-compiles CGO-disabled binaries for linux/amd64, linux/arm64, darwin/amd64, and darwin/arm64. container-verify builds amd64/arm64 binary stages, checks ELF machine values, and constructs the multi-platform image using Buildx/QEMU. Neither target runs the final images, invokes packaged help/version/config validation, verifies bundled FFmpeg at runtime, or executes on native target hosts.
- docs/product-contract.md correctly labels declared support, exercised build evidence, and external prerequisites. It says no model/firmware is positively qualified, cross-builds do not establish per-target runtime qualification, FFmpeg version compatibility is unknown, and packaging definitions do not prove publication or availability.
- cmd/jetkvm-mcp-validate is a separate read-only two-minute black-box stdio runner that verifies tool annotations, configured device discovery, connected typed status, typed media status, and a fully decoded PNG capture. Its sanitized report excludes identity and raw details. The compatibility ledger retains one 2026-08-09 non-production physical pass, but the firmware is `not_retained` and source is `not_attributed`; the ledger explicitly denies a positive compatibility claim.
- Mutation validation is deliberately separate: the offline command validates a strict dry-run plan and always reports execution_authorized=false. docs/mutation-validation.md requires a separately approved expendable target, observer, recovery, independent postconditions, per-operation deadlines, stop-on-unknown, and no retry. No mutation qualification record exists, which is the correct state absent explicit hardware authority.
- docs/telemetry.md explicitly says telemetry tests are fixture-only and do not measure real JetKVM latency, qualify hardware, or authorize production load/soak. There is no soak record, native runtime matrix, representative workload, accepted latency/throughput/resource threshold, benchmark, benchstat history, or performance support claim.
- Coverage is uploaded as coverage.out and coverage.txt for review without an arbitrary pass percentage. This is proportional because the repository has no historical risk-derived threshold and high-consequence branches are protected by explicit behavioral tests rather than a line-count proxy.

**Implementation And Documentation Agreement**

Implementation and documentation agree unusually well on evidence boundaries. The Makefile and CI implement the documented minimum/release Go, race, analyzer, vulnerability, fuzz-smoke, coverage, protocol, cross-build, and container lanes. The product contract accurately distinguishes declared surfaces from exercised builds and external prerequisites. Protocol-gates.md and pins.json expose exact official scenario dispositions instead of claiming blanket MCP certification. Compatibility docs and the ledger refuse to turn source review, fake sessions, or the unattributed historical hardware pass into firmware support. Mutation docs refuse to turn a dry-run plan into authority or physical success. Telemetry docs refuse to call synthetic load a performance or soak result. One evidence-presentation weakness remains: decoder_real_test.go skips if FFmpeg is absent, but CI/docs do not make a non-skip result an explicit release artifact, so a green run can lack the claimed real-executable check. Container construction and architecture checks are described carefully, but no packaged runtime smoke exists. No documentation claims benchmark, performance, soak, or full target-runtime evidence that the repository does not have.

**Current State**

substantially_satisfied: the repository has a broad, risk-directed, conventional verification suite and maintains the most important distinctions among fixture, protocol, build, runtime, physical, soak, and performance evidence. Before public release it should make the real-FFmpeg and packaged/container smoke results unambiguous and capture one clean release-candidate evidence set. Positive JetKVM model/firmware and mutation qualification, native target runtime, soak, and performance remain intentionally absent rather than falsely claimed.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- A green CI run occurs on a runner without usable FFmpeg or libx264; TestRealFFmpegH264ToPNG skips, fake-executable tests pass, and a release later fails to capture screens with the packaged FFmpeg. The missing evidence is non-skip integration plus packaged runtime smoke.
- A darwin or arm64 binary cross-compiles and has the expected machine type but fails at startup, signal handling, subprocess cleanup, path discovery, certificate use, or native WebRTC behavior. Cross-build evidence cannot detect these runtime failures.
- A fake JetKVM session accepts the expected RPCs while a current firmware changes signaling, status fields, video behavior, HID semantics, or media cleanup. Unit and source evidence pass, yet a real device operation fails or has a dangerous ambiguous outcome.
- A cancellation/resource race occurs only after many reconnects, concurrent captures, or blocked FFmpeg/telemetry operations. Deterministic unit tests and race execution do not trigger it because there is no representative long-running workload or broad leak soak.
- A malformed SDP, RTP/H.264 stream, RPC response, origin, URL, or config reaches a new parser path not represented by seeds during a one-second fuzz smoke. The bounded smoke proves target health and some exploration, not exhaustive security testing.
- An official conformance scenario outside the four applicable gates regresses, or the alpha scenario inventory changes upstream. Focused fixtures can still pass; the pin/inventory drift gate must trigger review rather than imply blanket MCP conformance.
- A maintainer optimizes a microbenchmark that omits device latency, WebRTC setup, H.264/FFmpeg decode, admission, or HTTP/MCP overhead. Code becomes more complex while end-to-end user latency, reliability, or memory does not improve and may regress.
- A coverage percentage rises through low-risk branches while outcome classification, cleanup, or physical-consequence behavior remains under-tested. Treating coverage as a target rather than review evidence creates false assurance.
- A historical unattributed hardware pass is advertised as firmware support. Users rely on power/media/HID compatibility that was never tested, potentially causing host interruption, data loss, repeated unknown-outcome mutations, or failed recovery.

**Affected Assets And Trust Boundaries**

Affected assets are device credentials and sessions; JetKVM/host availability and physical power state; HID input and host data; local and appliance media; screenshots and video; FFmpeg child processes and pipes; MCP stdout integrity; HTTP availability; published binaries/containers; and maintainer time and credibility. Relevant boundaries are authored tests versus untrusted parser inputs, fake SessionProvider versus real JetKVM firmware/network behavior, source protocol observations versus appliance runtime, Go process versus FFmpeg executable, build host/emulator versus native target runtime, loopback MCP clients versus deployed clients/proxies, read-only validator versus physical mutations, and a test artifact versus a support or performance claim.

**Plausible Impact**

Failures can range from protocol incompatibility and unavailable screen capture to goroutine/process leaks, memory growth, hung shutdown, request amplification, sensitive diagnostic leakage, incorrect mutation retry advice, repeated HID/power/media effects, host interruption, and data loss. Overclaiming cross-platform, firmware, soak, or performance support damages release trust and creates support obligations unsupported by evidence. Excessive test or benchmark machinery would instead consume one-maintainer review capacity and can make meaningful failures harder to see.

**Existing Controls**

- Risk-directed unit/integration tests across configuration, privacy, protocol, transport, WebRTC/RPC/video, FFmpeg, HID, media, telemetry, admission, cancellation, and cleanup.
- Race detection on the complete package suite and many deterministic terminal-race/cancellation tests.
- Eight checked, privacy-reviewed native fuzz targets with bounded CI smoke and a longer local command.
- Reviewed tool-manifest fixture exercised over in-memory, subprocess stdio, and stateless HTTP, plus exact official conformance/Inspector pins and sanitized artifacts.
- Minimum-Go and release-Go lanes, module verification, vet/staticcheck/govulncheck, review-only coverage, four cross-builds, and two-platform container construction.
- A real-FFmpeg integration test alongside argument-safe fake subprocess tests, though the real test is skippable.
- A black-box read-only validator, sanitized compatibility ledger, exact upstream source-review/drift process, and explicit refusal to infer firmware qualification.
- An offline mutation-plan validator with no execution path and a separate approval-gated supervised physical checklist.
- Maintained product-contract and telemetry language that prevents fixture/build evidence from being promoted into runtime, physical, soak, or performance claims.

**Residual Risk**

The suite cannot establish behavior on native target platforms, deployed proxies, every FFmpeg build, or real JetKVM firmware. Race detection remains execution-dependent; fuzz smoke is shallow by design; the one general synthetic resource check is not a system soak; official conformance coverage is bounded; and container builds do not prove startup. These are acceptable pre-adoption gaps only if support claims remain narrow. The pre-release residual worth reducing is silent FFmpeg skipping and absence of packaged-runtime smoke. Physical mutation risk remains accepted and unqualified until separately authorized hardware evidence exists.

**Compatibility Or Semver Effect**

Adding or tightening test gates has no direct wire compatibility effect. A test-discovered correction must still be classified under the product contract: changes to tool schemas/results/errors, CLI/config, artifact targets, or physical consequences can require patch, minor, or major treatment even before 1.0. Adding a positive platform or firmware support claim creates a support obligation; later removal can be breaking. Performance thresholds should not become a compatibility promise unless intentionally documented. Retaining fixtures and exact protocol pins protects current pre-1.0 review discipline.

**Privacy Effect**

Current synthetic fixtures, corpus policy, sanitized protocol artifacts, validator report, compatibility ledger, and mutation evidence schema reduce retention of credentials, endpoints, device identities, screenshots, paths, media names, typed text, and raw child/device output. Hardware, soak, and performance experiments would cross private device/host boundaries and require the same minimization. A packaged smoke can remain offline and synthetic. More verbose failure capture must not weaken existing redaction or place raw device/FFmpeg streams in CI artifacts.

**Operational Effect**

An explicit non-skipping FFmpeg lane and offline packaged/container smoke slightly increase CI duration while catching release-specific failures earlier. Removing duplicate ordinary test/vet execution can offset that cost. Periodic longer fuzzing or soak should be scheduled only with clear ownership and artifact retention. Native/hardware qualification remains an operator activity outside ordinary PR CI. Avoiding arbitrary coverage/performance thresholds keeps gates actionable and release diagnosis concise.

**Maintainer Effect**

The current conventional Make targets and single CI workflow are understandable and mostly self-maintaining. Exact fuzz and conformance inventories create intentional review work when parser or upstream protocol surfaces change, which is valuable. The main low-cost improvement is making evidence labels and skips visible, not adding a test platform. Every additional native runner, hardware rig, soak service, fuzz service, coverage policy, or performance dashboard adds recurring triage, environment renewal, flake diagnosis, retention, and support-claim work. For one maintainer, those controls should appear only after observed usage or a concrete release decision justifies them.

#### Decision

**Disposition**

change

**Recommendation**

Retain the risk-directed suite, exact fuzz/conformance inventories, race and minimum-Go lanes, review-only coverage, cross-build/container construction, read-only validator, approval-gated mutation checklist, and explicit evidence taxonomy. Before public release, change only the evidence-producing edge: make one release-candidate CI path prove that real FFmpeg did not skip; smoke the actual built binary and final linux container offline for help/version/config-validation, non-root execution, and FFmpeg discovery; retain the exact clean results with commit/toolchain identities; and simplify duplicated test/vet execution if needed to keep CI boring. Continue to state that cross-build is not runtime compatibility and that no model/firmware, mutation, soak, or performance support exists. Add native, hardware, soak, or performance work only when a concrete proposed claim, user problem, representative workload, threshold, and authority exist.

**Minimal Practical Change**

Add a small explicit release-evidence check that runs TestRealFFmpegH264ToPNG uncached and fails rather than skips in the release/container environment, then runs offline smoke commands against the built archive binary and each Linux final image: `--version`, help, `config validate` with a synthetic valid configuration, effective non-root UID/GID, and successful FFmpeg discovery without a device connection. Record pass/fail, exact commit, Go version, coarse OS/architecture, FFmpeg version, and artifact digest in sanitized CI artifacts. Keep ordinary PR tests fixture-only. Remove the duplicate ordinary test/vet invocation in the release lane only if the same gates remain clearly represented once.

**Optional Stronger Control**

After external adoption or a concrete incident, add a scheduled longer fuzz campaign for the eight existing deterministic targets and a bounded no-device session/subprocess soak that repeatedly opens/closes loopback MCP, provider fakes, and FFmpeg decoding while checking terminal goroutine/process/file counts. Add native runtime smoke on linux/arm64 or macOS only when downloads or users exist there. For a positive JetKVM claim, run the existing read-only validator on an approved non-production target and retain exact model/firmware/server/artifact/runtime identities; mutation qualification remains separately observed and approval-gated. For a proven performance decision, define the user-visible operation and SLO, construct a representative fixed-workload test, collect profiles, and use repeated benchstat-style A/B evidence in a stable environment.

**Rejected Or Overengineered Alternatives**

- Reject an arbitrary global coverage percentage: it rewards line execution without proving consequence, cleanup, runtime, or hardware behavior and creates low-value churn.
- Reject restoring the removed benchmark or adding a benchmark dashboard without a user decision, representative end-to-end workload, accepted threshold, stable environment, and comparative evidence.
- Reject property-testing, mocking, mutation-testing, or test-orchestration frameworks merely to expand the tool list; existing table tests, invariants, native fuzzing, and focused fakes cover actual risks conventionally.
- Reject a permanent hardware-in-CI rig, automatic mutation tests, or shared production-device qualification. They require credentials and destructive authority, create flakiness and safety risk, and violate the explicit approval boundary.
- Reject treating every official MCP conformance fixture tool as a product feature. Applicable official scenarios plus product-specific wire fixtures are stronger evidence for the intentionally narrow capability surface.
- Reject a full OS/architecture runtime matrix before users exist on those targets; retain cross-build evidence and add native smoke only when a release claim or adoption justifies its recurring cost.
- Reject immediate continuous fuzzing/OSS-Fuzz onboarding and a generalized leak framework absent incident evidence; a periodic longer run and one bounded synthetic soak are the proportional escalation path.
- Reject claiming reproducibility, device qualification, soak reliability, or performance from checksums, source review, fake fixtures, cross-builds, or one historical unattributed physical pass.

**Rationale**

This repository already tests the failure modes most likely to matter for a small consequential server: strict boundary parsing, wire contracts, stdout separation, cancellation before/after dispatch, unknown mutation outcome, resource teardown, subprocess bounds, admission, privacy-safe projection, and protocol applicability. Official Go guidance confirms that race results cover only executed paths, one-second fuzzing is bounded exploration rather than exhaustive assurance, package coverage is execution evidence rather than a product qualification, and meaningful performance work depends on representative workloads and comparative measurement. Repository evidence itself correctly refuses to conflate build, fixture, protocol, hardware, soak, and performance claims. The highest-value remaining pre-release gap is not another framework; it is proving that the external FFmpeg and packaged runtime paths actually execute. That is small, directly connected to a mandatory runtime prerequisite, and can be offset by removing duplicate gate execution.

**Dependencies Or Prerequisites**

- A release-candidate workflow or documented local release command with access to the exact archive/container outputs and an environment containing the intended FFmpeg package.
- A synthetic configuration that exercises offline validation without credentials, network, device access, or retained private values.
- A decision about whether linux/arm64 final-image smoke may run under QEMU or must wait for native evidence; either result must be labeled accurately.
- For positive hardware support: explicit authority, designated non-production device/host, exact model/firmware/artifact/runtime identities, sanitized retention, and no mutation authority inferred from read-only approval.
- For performance work: a named user problem or release decision, representative workload and data, accepted objective, stable measurement environment, and owner for regression triage.

**Migration Or Rollout Considerations**

First preserve the existing fixture/race/fuzz/protocol gates. Add the non-skip real-FFmpeg and offline packaged smoke as visible release-candidate checks without changing product behavior. Run them on the exact artifact intended for publication and keep only sanitized version/digest/result metadata. If the added smoke is flaky, do not weaken it silently: classify whether the instability is environmental or product behavior and retain the previous narrow support claim. Simplify duplicate test/vet commands after comparing job logs so required branch checks remain stable. Add no firmware/platform/performance declaration until its evidence record is accepted; publish limitations alongside any future positive claim. Hardware mutation remains a separate manually approved rollout with stop-on-unknown behavior.

**Priority**

P1: make external-prerequisite and packaged-runtime evidence unambiguous before public release; the broader suite and its narrow claims are already strong.

**Implementation Effort**

S for non-skipping FFmpeg plus offline artifact/container smoke and CI de-duplication; hardware, native runtime, soak, and representative performance programs are deliberately outside this estimate.

**Ongoing Maintenance Burden**

low for the recommended change: update coarse tool identities and diagnose only actionable release smoke failures. Native matrices, hardware qualification, scheduled soak, and performance baselines would be medium to high and are deferred until justified.

**Confidence**

high for static suite, workflow, fixture, and documentation findings; medium for release readiness because no live GitHub run, published artifact, native target, current attributed hardware, or soak evidence was inspected.

#### Verification

**Acceptance Evidence**

- A clean release-candidate record tied to 6e52f0027b13f928b768de0feeab4847ef9ca53e or its intentionally reviewed successor showing minimum-Go, race/analyzers, all eight fuzz-smoke targets, coverage artifact creation, four applicable official MCP gates plus Inspector, four release cross-builds, and two-platform container construction passed.
- An uncached real-FFmpeg test record that contains PASS and cannot return a successful skip, with the coarse FFmpeg version recorded and no media/device data.
- Offline smoke results from the actual archive binary and each final Linux image showing expected version/help/config-validation streams and exits, UID/GID 10001 in the image, and FFmpeg discovery; artifact digests bind results to outputs.
- Documentation continues to label fixture correctness, official protocol conformance, build evidence, runtime compatibility, physical-device qualification, soak, and performance as separate evidence classes.
- No new performance threshold or positive device/platform claim appears without its required decision and evidence record.

**Proposed Tests Or Checks**

- Run `go test -count=1 -v ./internal/jetkvm -run '^TestRealFFmpegH264ToPNG$'` in the release environment and fail the job if output contains SKIP or the required encoder/decoder path is unavailable.
- Build release artifacts, extract each archive, and assert `--version`, help, and offline `config validate` output/exit behavior from the extracted binary rather than a source-tree build.
- Run each final Linux image under its available platform, assert numeric user 10001:10001, execute help/version/offline validation, and confirm the intended FFmpeg binary/version is discoverable. Label QEMU execution as emulated runtime evidence, not native qualification.
- Keep `go test -race ./...`, but periodically add `-count=10` to the highest-risk deterministic cancellation/terminal-race tests after related changes rather than globally multiplying all tests.
- Run the existing 30-second fuzz target set before a security-sensitive parser/media release or on a low-frequency schedule; preserve only minimized non-sensitive reproducers after corpus review.
- Review the official MCP scenario inventory digest on every pinned package update; require explicit disposition and focused replacement evidence for each non-applicable scenario.
- When a real device support claim is proposed, run the read-only validator on the exact candidate artifact and record the narrow checks; execute mutations only under the separate checklist and approval.

**Negative Or Abuse Cases**

- FFmpeg absent, wrong executable first on PATH, missing libx264 encoder, malformed/truncated H.264, oversized PNG output, hung child, deadline during pipe reads, child ignoring termination, and private stderr content.
- Malformed/trailing/oversized YAML and JSON; unknown fields; duplicate members; non-origin Host/Origin; credentials/query/fragment in device URLs; redirect credential forwarding; malformed SDP/RPC/RTP/H.264/media URLs.
- Cancellation before dispatch, after possible send, during session setup, at frame/decode deadline, while waiting for admission, during hash/upload/cleanup, at process-group shutdown, and exactly concurrent with terminal success.
- Provider panic/error, queue saturation, slow/failing telemetry sink, connection close with pending RPCs, repeated session open/close, stuck subprocess descendant, partial upload, symlink exchange/FIFO/local media race, and cleanup timeout.
- Unsupported MCP method/capability, protocol revision/header mismatch, cache metadata regression, DNS rebinding/Host mismatch, tools-list shape drift, output-schema violation, stdout diagnostic leakage, and protocol-gate raw artifact leakage.
- Cross-built binary that has the right machine type but fails on native startup; final image that builds but uses the wrong UID, lacks FFmpeg, reports the wrong version, or cannot validate a synthetic configuration.
- Hardware acknowledgement lost after mutation send, wrong physical target, changed firmware semantics, repeated unknown-outcome operation, loss of observer, integrity mismatch, cleanup failure, and status result incorrectly treated as independent physical proof.
- Performance comparison with changed Go/FFmpeg/firmware/hardware, variable network/device load, cache-warm mismatch, insufficient repetitions, microbenchmark-only improvement, or threshold chosen after observing results.

**Evidence Needed Before Claiming Support**

Fixture correctness requires exact deterministic tests and reviewed synthetic data; protocol conformance requires a pinned official suite/version/scenario inventory with pass artifacts and explicit applicability; build support requires successful exact-toolchain cross-build/package construction; runtime compatibility requires execution of the candidate artifact on the named OS/architecture/environment with external prerequisites; physical JetKVM qualification requires exact model, firmware, server artifact/source, runtime and FFmpeg identities plus approved read-only checks, and separate supervised mutation evidence for each consequential operation; soak reliability requires a defined duration, workload, concurrency, failure injection, resource/cleanup thresholds, environment, and retained result; performance support requires a user-visible objective, representative fixed-work workload, accepted threshold, stable environment, repeated statistically compared measurements and profiles. None of these classes may be substituted for another.

**Revisit Trigger**

Revisit when Go, the MCP specification/SDK/conformance package, Pion, FFmpeg, Docker base image, or JetKVM firmware changes; when a parser, transport, cancellation, media, or mutation path changes; when a fuzz/race/CI failure or leak/latency incident occurs; before the first public release; before adding a target or firmware claim; when native target users appear; when external contributor volume makes flaky/slow CI costly; or annually if none occurs. Add performance work only after a concrete latency/throughput/resource decision; add soak after an uptime/reconnect claim or resource-lifecycle incident.

### 5. MCP specification, Go SDK, compatibility, and conformance

#### Identity And Scope

**Item Name**

MCP specification, Go SDK, compatibility, and conformance

**Research Question**

Does jetkvm-mcp's implemented MCP wire contract at the exact studied tree conform to the latest dated 2026-07-28 MCP specification, use the official Go SDK safely, expose only intended capabilities and revisions, and have proportionate compatibility and conformance evidence before public release?

**Scope**

This item covers the normative 2026-07-28 MCP lifecycle and wire model; server discovery and per-request capability/version metadata; the tools primitive, schemas, structured content, error categories, annotations, deterministic listing and caching; stdio and stateless Streamable HTTP; the official Go SDK's implemented versus merely available features; official conformance tooling and Inspector; protocol security advisories; and compatibility claims. It excludes the substantive safety of individual JetKVM operations, HTTP authentication/deployment policy, and generic supply-chain controls except where the SDK, conformance runner, or Inspector affects protocol assurance.

**Repository Surfaces**

- go.mod official Go SDK requirement github.com/modelcontextprotocol/go-sdk v1.7.0 and go.sum
- internal/mcpserver/server.go NewServer, tools-only ServerCapabilities, 18 AddTool registrations, typed schemas/results, annotations, and tool-error handling
- internal/mcpserver/transport.go NewStreamableHTTPHandler with Stateless=true and JSONResponse=true
- internal/mcpserver/manifest_contract_test.go and testdata/tool-manifest.json, including discovery supportedVersions, cache hints, capability surface, transport parity, schemas, annotations, and representative error contracts
- internal/mcpserver/transport_test.go, origin_test.go, tool_error_test.go, output_validation_test.go, and cmd/jetkvm-mcp/stdio_integration_test.go
- internal/protocolgate and cmd/jetkvm-mcp-protocol-gates
- testdata/mcp-gates/pins.json and package-lock.json
- docs/protocol-sources.md, docs/protocol-gates.md, docs/product-contract.md, docs/ci-quality.md, README.md, Makefile, and .github/workflows/ci.yml
- The downloaded official Go SDK v1.7.0 source, especially mcp/shared.go, mcp/transport.go, mcp/streamable.go, and mcp/server.go

**Applicability Stage**

current_private, before_public_release, after_external_adoption

#### Sources And Authority

**Source Class**

- The exact repository tree, local SDK source, generated fixture, and test executions are repository_evidence.
- The dated MCP specification pages and authoritative schema are normative_specification.
- The Go SDK release and source, conformance repository, Inspector repository, and MCP release announcement are official_documentation or maintainer_case_study.
- The two Inspector GHSAs are official_advisory.

**Normative Status**

- MUST: modern server/discover, per-request revision/capability metadata, required resultType, declared output-schema conformance, cache hints, stdout purity, and required Streamable HTTP validation behavior.
- SHOULD: deterministic tool ordering, text fallback for structured content, localhost binding and authorization guidance, and advisory annotations used as hints rather than security controls.
- MAY: pagination, SSE responses, list-change notifications, annotations, and support for more than one revision.
- Deprecated: roots, sampling, logging, HTTP+SSE and DCR; their presence in an SDK does not make them part of this product.
- Optional extension: tasks; MRTR/input-required behavior is unnecessary for the current complete, bounded tool calls.
- Observation: Inspector and alpha conformance results are evidence about exercised paths, not a normative certification.

**Source Disagreements**

- docs/product-contract.md and README declare only MCP 2026-07-28 supported, but the executable discovery fixture advertises 2026-07-28 plus 2025-11-25, 2025-06-18, 2025-03-26, and 2024-11-05. The SDK source explains the conflict: v1.7.0 intentionally retains every endpoint's backward compatibility and the repository does not restrict supported revisions. Resolution: wire discovery is authoritative to clients; the current implementation is multi-revision despite the documentation.
- docs/protocol-gates.md describes Inspector as checking initialization, while initialization was removed in 2026-07-28 and the modern client path uses discovery. Resolution: update the evidence language after raw-wire verification; do not call an SDK connection helper's internal setup an MCP initialize exchange.
- The official conformance inventory includes many tool-call scenarios, but most demand conformance-only fixture tools. The repository classifies them not applicable and substitutes product tests. Resolution: this is proportionate, but a pass means selected applicable scenario coverage, not full suite conformance.
- The SDK exposes deprecated primitives, legacy revisions, tasks, MRTR and other capabilities. Resolution: SDK availability is not specification requirement or product support; only advertised and tested surfaces count.

#### Repository Evidence

**Exact Baseline Commit**

The assigned baseline is 6e52f0027b13f928b768de0feeab4847ef9ca53e, tree 34e8b4451d76821950c23d7c06958d021700f3a7. At this workstream's start local HEAD was 71445bc6bf2325e6c683e362393605089c336b63, but its tree was exactly the same 34e8b4451d76821950c23d7c06958d021700f3a7 and git diff 6e52f00..HEAD was empty. The worktree also contained pre-existing untracked .agents/, skills-lock.json, and jetkvm_mcp_2026_public_release_baseline/. No evidence from differing trees was mixed.

**Current Repository Evidence**

- go.mod selects the official github.com/modelcontextprotocol/go-sdk v1.7.0; docs/protocol-sources.md records the tag and commit provenance.
- internal/mcpserver/server.go declares only ServerCapabilities.Tools with an empty ToolCapabilities object, producing listChanged=false, and statically registers 18 tools. No prompts, resources, logging, roots, sampling, tasks, subscriptions, or extensions are advertised.
- All 18 tool definitions have inputSchema and outputSchema; annotations explicitly classify read-only, destructive, idempotent, and open-world behavior. Typed AddTool handlers validate arguments and outputs and return structuredContent plus a text fallback. Operational failures become CallToolResult isError=true; malformed/unknown protocol requests remain JSON-RPC errors.
- The manifest contract fixture records protocolVersion 2026-07-28, a complete resultType, empty client capabilities, tools-only server capabilities, deterministic sorted tools, ttlMs=0 and cacheScope=public.
- That same fixture records supportedVersions as five revisions: 2026-07-28, 2025-11-25, 2025-06-18, 2025-03-26, and 2024-11-05. This contradicts docs/product-contract.md's declared modern-only support.
- Downloaded v1.7.0 source confirms the five-version list. StdioTransport does not implement the SDK's ProtocolVersionSupporter restriction seam; stateless Streamable HTTP's SupportsProtocolVersion returns true for both the 2026 revision and legacy revisions. Therefore the legacy advertisement is active SDK behavior, not a passive dependency capability.
- internal/mcpserver/transport.go selects Stateless=true and JSONResponse=true. The wrapper exposes POST /mcp without a protocol session or GET/SSE channel; JSON response is permitted for this request/response-only product.
- Manifest tests compare in-memory, stdio subprocess, and stateless HTTP behavior. Stdio integration tests, HTTP transport/origin/header tests, output-validation tests and tool-error tests exercise important focused paths.
- testdata/mcp-gates/pins.json pins official conformance 0.2.0-alpha.11 by version, commit, integrity and scenario-inventory hash. Four scenarios gate: tools-list, dns-rebinding-protection, caching, and http-header-validation. All other scenarios are classified with a product-specific reason and replacement evidence; none is an allowed unexplained skip.
- The Inspector gate pins 2.2.0 and tests discovery/list/call of the fixture-safe jetkvm_list_devices tool over both transports in an isolated temporary setup. It does not exercise consequential tools.
- The focused command go test ./internal/mcpserver ./internal/protocolgate ./cmd/jetkvm-mcp-protocol-gates passed on 2026-08-15.

**Implementation And Documentation Agreement**

Implementation and documentation agree on the 2026 wire model, official SDK, static tools-only surface, stateless HTTP, no deprecated capabilities, structured typed results, consequence annotations, selected conformance gates and transport parity. They materially disagree on supported protocol revisions: documentation says only 2026-07-28, while server/discover normatively advertises and the SDK accepts five revisions. The phrase 'Inspector initialization' is stale for the modern wire model. ttlMs=0/public agrees with the checked fixture and is conforming, though it deliberately disables useful freshness caching for a static public tool list.

**Current State**

partial: the repository has unusually strong modern MCP implementation and evidence for a small server, but the public compatibility declaration is false on the wire. The server advertises four unclaimed legacy revisions whose actual product semantics are not separately tested; selected alpha conformance and Inspector checks cannot close that mismatch.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- A legacy client sees its revision in server/discover and calls a consequential tool. The SDK's compatibility translation accepts the request, but repository tests and safety documentation reason about the 2026 contract. Differences in lifecycle, cancellation, metadata, headers, errors, schemas, or result handling can produce a lost acknowledgement or ambiguous mutation outcome.
- A maintainer changes a tool schema or annotation under the assumption that only 2026 clients exist; a backward-compatible SDK path silently preserves legacy access, cementing an unreviewed compatibility burden before 1.0.
- A CI report says 'official conformance passes' although only four applicable scenarios are gates and most call scenarios are fixture-specific N/A. Reviewers infer stronger coverage than exists and miss raw resultType, revision rejection, or transport translation defects.
- Inspector is pointed at an untrusted MCP server or run with broader command authority. Historical proxy/XSS vulnerabilities show how an official diagnostic client can become a command-execution boundary; a malicious tool description/result can attack the operator environment.
- A future SDK update exposes or defaults a capability, extension, or compatibility behavior. If capability and raw-wire fixtures do not remain exact, the product accidentally acquires new public protocol surface.
- A public cache scope is retained after tools/list becomes authorization-dependent or gains sensitive deployment metadata, allowing a client cache to share it across callers. Current static generic definitions do not have that property.

**Affected Assets And Trust Boundaries**

The affected assets are the stable public tool contract, client interpretation of consequential HID/power/upload/virtual-media outcomes, device and host state, credentials conveyed at deployment boundaries, server availability, and maintainer review capacity. Boundaries are MCP client to stdio child process, HTTP client/reverse proxy to POST /mcp, SDK wire translation to application handlers, cached discovery/tool metadata across authorization contexts, npm-based protocol tools to the CI runner, and Inspector-rendered remote metadata to the maintainer workstation.

**Plausible Impact**

The primary impact is compatibility and reliability: a server may promise a legacy behavior it does not intentionally validate, and an old caller may misinterpret a mutation acknowledgement or error. Secondary impacts are security and maintainer load from unplanned protocol surface, false confidence from partial conformance evidence, and possible local command execution if unsafe Inspector versions or untrusted servers are used. The current ttlMs=0 causes only modest repeated-list operational cost.

**Likelihood Or Preconditions**

Legacy-version risk requires a client that sends or selects one of the four advertised old revisions; this is plausible precisely because discovery invites it, but current user adoption is unknown. Consequential impact additionally requires a mutating tool call and a semantic difference or failure. Accidental SDK-surface expansion requires an upgrade without exact capability/wire fixtures. Inspector exploitation requires running the tool against untrusted content or exposing its proxy; the repository's pinned 2.2.0 isolated safe-tool path substantially lowers that likelihood.

**Existing Controls**

- Exact official Go SDK and npm protocol-tool pins with checksums/integrity and source provenance.
- Static tools-only capabilities and exact checked-in manifest fixture across in-memory, stdio and HTTP paths.
- Typed schemas, output validation, structured content, text fallback, explicit annotations, deterministic names, and protocol/tool error separation.
- Stateless JSON Streamable HTTP, required header/origin tests, stdout-purity integration coverage, cancellation tests, and no deprecated HTTP+SSE path.
- Four blocking official conformance scenarios with full scenario classification, plus pinned Inspector checks limited to a fake device and safe read-only discovery tool.
- Documentation explicitly rejects inferring support from SDK availability and records protocol source provenance.

**Residual Risk**

The supportedVersions mismatch remains a concrete public-contract defect until both transports reject unclaimed revisions. Product-specific tests cover representative calls but do not prove every mandatory raw-wire field across success and failure responses. Alpha conformance churn can also create maintenance noise or blind spots. Annotations improve caller comprehension but are explicitly untrusted hints and cannot ensure a client obtains human approval or avoids unsafe retries.

**Compatibility Or Semver Effect**

Restricting discovery from five revisions to 2026-07-28 is technically a wire compatibility reduction for any client already relying on legacy negotiation, but it repairs an undocumented and untested promise before public release. Treat it as a deliberate pre-1.0 compatibility correction, note it in release notes, and do not silently add the old revisions later. Tool names, schemas, annotations, output schemas and error classification remain compatibility-sensitive even before 1.0; the manifest fixture appropriately makes changes review-visible.

**Privacy Effect**

Current discovery and tools/list metadata are static and contain no device identifiers, aliases, credentials, screenshots, typed text, or caller-specific authorization data, so cacheScope=public is defensible. If tools become caller- or device-specific, public caching must be changed to private. Inspector and conformance artifacts must continue excluding credentials and consequential device output.

**Operational Effect**

Modern-only negotiation reduces the matrix the maintainer must diagnose and avoids hidden state/session expectations. Retaining JSONResponse=true keeps HTTP operation simple; no SSE infrastructure is needed for current complete calls. A positive tools-list TTL would reduce repeated metadata traffic but is optional. The gate suite adds npm/network time and should remain a targeted release/CI claim rather than expanding with irrelevant fixture jobs.

**Maintainer Effect**

The recommended change removes four revision families from the mental and test matrix while preserving high-value fixtures. Recurring work is limited to reviewing each dated MCP/SDK release, advisories, conformance inventory changes and deliberate manifest diffs. Supporting all advertised legacy revisions would multiply lifecycle, transport, error and host-interoperability review for little one-maintainer value.

#### Decision

**Disposition**

change

**Recommendation**

Before public release, make the executable wire contract modern-only: server/discover must advertise only 2026-07-28 and both stdio and stateless HTTP must reject legacy and unknown revisions with the specified error behavior. Retain the official Go SDK v1.7.0, static tools-only capability, stateless JSON HTTP, typed structured results, text fallback, explicit consequence annotations, exact manifest parity, selected official gates and isolated Inspector smoke tests. Treat tasks, MRTR, deprecated roots/sampling/logging, sessions, HTTP+SSE and other SDK features as unsupported/non-goals unless a real product requirement emerges. Describe conformance precisely as a pinned applicable-scenario gate, not certification.

**Minimal Practical Change**

Add an explicit supported-revision restriction at each SDK transport boundary, preferably through an upstream SDK option or a very thin reviewed ProtocolVersionSupporter adapter; update the manifest expectation to exactly ["2026-07-28"]; add raw-wire tests that legacy initialize and all four older revisions are rejected and that unknown modern revision returns -32022 with only the intended supported list. Correct documentation that currently says Inspector 'initializes' and explicitly state the four official scenarios that gate. Do not otherwise redesign the server.

**Optional Stronger Control**

If external adoption later creates demonstrated legacy demand, choose and publish a finite compatibility matrix, run revision-specific raw fixtures and representative hosts for each supported revision, and sunset revisions explicitly. For the static list, a reviewed positive ttlMs such as one hour or one day can improve cache behavior once representative clients demonstrate value. Upgrade the official conformance pin when a stable release or materially relevant scenario appears, preserving the inventory hash and full classification.

**Rejected Or Overengineered Alternatives**

- Do not implement or claim four legacy revisions merely because the SDK can translate them; that turns accidental availability into permanent test and review burden.
- Do not add deprecated roots, sampling, logging, protocol sessions, initialize, ping, or HTTP+SSE.
- Do not add tasks, MRTR/input-required, subscriptions, dynamic tool-list notifications, prompts, or resources without an actual product use case.
- Do not create conformance-only tools in the production manifest solely to make every official fixture scenario green; substitutions with exact product tests are more faithful.
- Do not call selected conformance scenarios certification, and do not use Inspector UI execution against untrusted public servers or give it broad workstation/runner authority.
- Do not fork the specification or maintain a large SDK fork; prefer an upstream configuration seam or minimal boundary adapter.

**Rationale**

The dated specification permits a server to support only 2026-07-28, and this small product expressly chose that bounded contract. The implementation is otherwise strong and current, but server/discover is normative: advertising four old revisions is a product promise, not harmless SDK internals. Because the tools can cause physical mutations, ambiguous lifecycle/error/cancellation compatibility deserves correction before callers exist. Narrowing the wire surface has high reliability and maintainer value, while adding legacy implementations or optional MCP features has disproportionate recurring cost. Exact capability/manifest tests and focused official gates preserve useful assurance without governance theater.

**Migration Or Rollout Considerations**

Make the restriction before the first public release. Run exact manifest and raw-wire tests over in-memory, stdio and HTTP, then run the pinned official gates and Inspector smoke. Record the compatibility correction in release notes because a private/internal legacy client could have relied on the wire despite contrary docs. If such a client exists, either migrate it to 2026-07-28 or make an explicit, time-bounded support decision; do not silently leave all legacy revisions enabled. Preserve rollback by keeping the change localized to transport/version selection.

**Priority**

P0 before public release because discovery currently makes a false normative compatibility promise on safety-relevant tools; P2 for positive caching and wider host interoperability.

**Implementation Effort**

S if an SDK restriction seam is available; M if both transports require a small adapter or upstream change.

**Ongoing Maintenance Burden**

low after modern-only restriction: review dated spec/SDK/advisory changes and deliberate fixture diffs; supporting legacy revisions would be medium to high.

#### Verification

**Acceptance Evidence**

- Raw server/discover over stdio and HTTP returns supportedVersions exactly ["2026-07-28"], tools-only capabilities, resultType=complete and valid cache hints.
- Requests declaring each of 2025-11-25, 2025-06-18, 2025-03-26, 2024-11-05 and a fabricated revision fail with specified unsupported-version behavior and do not invoke any handler.
- No initialize/initialized, Mcp-Session-Id, GET/SSE, ping, deprecated capability or extension appears in raw traffic or manifest.
- Representative tools/list and tools/call success, operational error and malformed request carry required 2026 result/error shapes on both transports; output schemas validate.
- Pinned applicable conformance scenarios and Inspector safe smoke pass from a clean release candidate, with logs recording exact tool versions and no secrets.
- README, product contract, protocol sources/gates and executable fixture agree on the same support statement.

**Proposed Tests Or Checks**

- Table-test all supported and unsupported revision values at the raw JSON-RPC layer over stdio and HTTP, including body/header disagreement.
- Assert mandatory per-request _meta keys, HTTP MCP-Protocol-Version, Mcp-Method and Mcp-Name matching, Content-Type/Accept handling, and -32020/-32022 errors.
- Assert every complete response has wire resultType=complete even where the SDK hides the discriminator from application types.
- Retain exact deterministic tools/list snapshot and test ttlMs/cacheScope, no cursor for the single page, only tools capability, and listChanged=false.
- Validate every successful structuredContent value against its outputSchema and retain text fallback; assert operational failures use isError and unknown tool/malformed params use protocol errors.
- Run cancellation and stdout-purity tests over both transports; verify cancellation cannot turn a delivered mutation into an automatic retry.
- Run the four pinned official gates and the pinned Inspector read-only smoke in an isolated temp directory with fake device configuration.

**Negative Or Abuse Cases**

- Legacy initialize request, legacy revision in _meta/header, missing revision, unknown revision, contradictory header/body revision, missing/mismatched method or tool-name header, wrong Content-Type, malformed JSON and batched input.
- Unknown tool, wrong argument type, extra/boundary arguments, handler operational failure, output-schema violation, unknown resultType, missing resultType and cancellation before/after downstream dispatch.
- Attempted capability use for prompts, resources, logging, roots, sampling, tasks, subscriptions or session endpoints when not advertised.
- Tools/list cache leakage check if aliases or authorization-specific tools are ever introduced; public cache results must remain caller-independent.
- Malicious tool description/result rendered by Inspector, exposed Inspector proxy, poisoned npm package/lock mismatch, and an upstream scenario inventory change not classified by the pin validator.

**Evidence Needed Before Claiming Support**

Claim support only for MCP 2026-07-28 after exact raw-wire revision rejection, required-field/header, capability, tool schema/result/error, cancellation and both-transport tests pass. Claim 'official conformance scenarios pass' only with the exact four scenario names, suite alpha version, command result and documented N/A inventory; do not claim full certification. Claim compatibility with a named host only after testing that host/version. Never claim legacy revision, deprecated primitive, task/MRTR extension, SSE/session, or physical-device behavior from SDK availability, a fake fixture, Inspector, or cross-build alone.

**Revisit Trigger**

Revisit on every new dated MCP revision; Go SDK or official conformance release/advisory; any change to server capabilities, transport, tool list, cache scope, result type or authentication-dependent visibility; first external compatibility report; evidence that a client needs legacy support, SSE or input-required calls; or an Inspector/conformance security advisory. Review at least quarterly while the 2026 ecosystem is rapidly changing, then at each dependency/release cycle once stable.

### 6. MCP tool-surface design, consequence communication, and compatibility burden

#### Identity And Scope

**Item Name**

MCP tool-surface design, consequence communication, and compatibility burden

**Research Question**

Does the exact repository tool surface communicate consequences accurately, bound and validate caller-controlled data, return stable typed results and errors, and keep only tools whose safety and compatibility value justifies their permanent review burden before a public 1.0 release?

**Scope**

The 18 statically advertised MCP tools, their names, titles, descriptions, JSON Schemas, annotations, result and error contracts, transport-independent compatibility fixtures, deprecation policy, and pre-1.0 evolution. This includes caller comprehension, metadata/schema attack surface, and the distinction between protocol errors and tool execution errors. It excludes device-protocol correctness, deployment authorization, and physical qualification except where those facts determine an annotation or public claim.

**Repository Surfaces**

- internal/mcpserver/server.go: Server.New, addInputTool, addMutationTool, addSanitizedInputMutationTool, mutationErrorResult, and result construction
- internal/mcpserver/controls.go: all tool registrations, input/result types, schemas, descriptions, and annotations
- internal/mcpserver/server_test.go, controls_test.go, manifest_contract_test.go, transport_contract_test.go, and testdata/tool-manifest.json
- internal/mcpserver/discovery_manifest.go and internal/mcpserver/testdata/server-discover-manifest.json
- docs/product-contract.md, docs/threat-model.md, docs/protocol-sources.md, README.md, go.mod, and repository tag v0.1.0

**Applicability Stage**

current_private, before_public_release, after_external_adoption

#### Sources And Authority

**Authoritative Sources**

- **Title:** jetkvm-mcp exact baseline source and tests | **Publisher:** BenDManning/jetkvm-mcp | **Url:** https://github.com/BenDManning/jetkvm-mcp/commit/6e52f0027b13f928b768de0feeab4847ef9ca53e | **Publication Or Update Date:** 2026-08-15 | **Version Or Revision:** commit 6e52f0027b13f928b768de0feeab4847ef9ca53e, tree 34e8b4451d76821950c23d7c06958d021700f3a7 | **Access Date:** 2026-08-15 | **Supported Claims:** Executable registrations, schemas, annotations, typed outputs, error behavior, manifest fixtures, deprecation, and documented product contract.
- **Title:** MCP Specification 2026-07-28: Tools | **Publisher:** Model Context Protocol | **Url:** https://modelcontextprotocol.io/specification/2026-07-28/server/tools | **Publication Or Update Date:** 2026-07-28 | **Version Or Revision:** 2026-07-28 | **Access Date:** 2026-08-15 | **Supported Claims:** Tool discovery and calls, annotations as untrusted hints, structured content and text fallback, output-schema conformance, error categories, and tool security requirements.
- **Title:** MCP Specification 2026-07-28: Schema Reference | **Publisher:** Model Context Protocol | **Url:** https://modelcontextprotocol.io/specification/2026-07-28/schema | **Publication Or Update Date:** 2026-07-28 | **Version Or Revision:** 2026-07-28 | **Access Date:** 2026-08-15 | **Supported Claims:** Normative Tool, ToolAnnotations, CallToolResult, inputSchema, outputSchema, title, description, and metadata shapes.
- **Title:** MCP 2026-07-28 Final Release | **Publisher:** Model Context Protocol | **Url:** https://blog.modelcontextprotocol.io/posts/2026-07-28/ | **Publication Or Update Date:** 2026-07-28 | **Version Or Revision:** 2026-07-28 final release | **Access Date:** 2026-08-15 | **Supported Claims:** Deterministic, cacheable list responses and the final revision lifecycle.
- **Title:** Tool Annotations | **Publisher:** Model Context Protocol | **Url:** https://blog.modelcontextprotocol.io/posts/2026-03-16-tool-annotations/ | **Publication Or Update Date:** 2026-03-16 | **Version Or Revision:** 2026 guidance | **Access Date:** 2026-08-15 | **Supported Claims:** Meaning, pessimistic defaults, client uses, and non-enforcement of read-only, destructive, idempotent, and open-world hints.
- **Title:** MCP 2026-07-28 Release Candidate | **Publisher:** Model Context Protocol | **Url:** https://blog.modelcontextprotocol.io/posts/2026-07-28-release-candidate/ | **Publication Or Update Date:** 2026-05-21 | **Version Or Revision:** 2026-07-28 release candidate guidance retained by final release | **Access Date:** 2026-08-15 | **Supported Claims:** JSON Schema 2020-12 support, prohibition on automatic external-reference dereferencing, and bounding schema depth and validation time.
- **Title:** Semantic Versioning 2.0.0 | **Publisher:** Semantic Versioning | **Url:** https://semver.org/spec/v2.0.0.html | **Publication Or Update Date:** 2013-06-19 | **Version Or Revision:** 2.0.0 | **Access Date:** 2026-08-15 | **Supported Claims:** Public API declaration, initial-development semantics, deprecation, and incompatible-change versioning.

**Source Class**

- **Repository Baseline:** repository_evidence
- **Mcp Tools And Schema:** normative_specification
- **Mcp Release And Annotation Articles:** official_documentation
- **Semantic Versioning:** normative_specification

**Normative Status**

- MCP: tools with outputSchema MUST return conforming structuredContent; the repository validates this before sending.
- MCP: annotations are hints and clients MUST treat them as untrusted unless the server is trusted; they are not authorization or confirmation controls.
- MCP: servers MUST validate tool inputs; the repository combines JSON Schema and handler validation.
- MCP: structured results SHOULD also include serialized JSON TextContent for backward compatibility; capture_screen currently returns image plus structured metadata but no JSON text metadata.
- MCP official guidance: idempotentHint communicates safe repeated invocation and can influence retry behavior; destructiveHint communicates whether a mutation may destructively update its environment.
- SemVer 2.0.0: 0.y.z APIs may be unstable; the repository's product contract voluntarily imposes a stricter major-version rule for breaking tool changes.

**Freshness And Supersession**

The MCP 2026-07-28 final specification and its final-release material were current on 2026-08-15 and supersede earlier MCP revisions for normative behavior. The May release-candidate article is used only for security rationale carried into the final model, not to override final schema text. SemVer 2.0.0 is older but remains the current canonical version. Repository evidence is from the exact stipulated tree.

**Source Disagreements**

- SemVer permits instability during 0.y.z, while docs/product-contract.md promises major-version treatment for breaking tool changes. Resolve in favor of the repository's stronger published contract; it is a deliberate compatibility promise, not a standards conflict.
- The repository describes wake, DC power state, and unmount operations as never blindly retryable after unknown outcome while advertising idempotentHint=true. Official annotation guidance allows clients to use that hint for retry decisions. Resolve conservatively: wake tools are not idempotent in the action sense, and all five hints should be false unless qualified evidence and the retry contract are deliberately reconciled.
- Keyboard and mouse descriptions acknowledge potentially destructive host effects while destructiveHint=false. Resolve using the MCP definition's may-be-destructive standard: set both true even though ordinary input is often harmless.

#### Repository Evidence

**Exact Baseline Commit**

The stipulated baseline is 6e52f0027b13f928b768de0feeab4847ef9ca53e (tree 34e8b4451d76821950c23d7c06958d021700f3a7). Live HEAD during this workstream was 71445bc6bf2325e6c683e362393605089c336b63, but it has the identical tree and an empty diff against the stipulated commit; evidence was therefore not mixed across source revisions. Pre-existing untracked .agents/, jetkvm_mcp_2026_public_release_baseline/, and skills-lock.json were present.

**Current Repository Evidence**

- internal/mcpserver/controls.go registers exactly 18 tools. Seventeen are one-purpose tools; jetkvm_virtual_media is an explicitly deprecated multiplexed compatibility tool with status, mount, unmount, and upload operations.
- internal/mcpserver/testdata/tool-manifest.json snapshots every tool's name, title, description, input schema, output schema, and annotations, plus representative success, tool error, and protocol error behavior.
- manifest_contract_test.go and transport_contract_test.go exercise the contract in memory, over stdio, and over stateless Streamable HTTP; successful structured output is checked against outputSchema.
- Every input object sets additionalProperties=false. Capture dimensions, typed text, media paths/URLs, mouse coordinates, and wheel deltas are bounded; device and target aliases, keyboard key, and modifier array cardinality remain incompletely bounded.
- Tool descriptions explain consequential power, HID, media, and capture effects and explicitly warn that unknown mutation outcomes must not be blindly retried.
- keyboard and mouse have destructiveHint=false despite descriptions saying they may execute commands, alter data, or activate destructive UI.
- turn_host_dc_power_on, turn_host_dc_power_off, wake_host_usb, wake_host_lan, and unmount_virtual_media have idempotentHint=true despite the universal unknown-outcome no-retry contract; wake operations repeat a packet or HID action on every invocation.
- mount_virtual_media_url alone has openWorldHint=true because it makes the appliance fetch a caller-selected URL; configured device and wake targets remain closed-world.
- addInputTool returns operational and business failures as CallToolResult with IsError=true, while unknown tools remain JSON-RPC protocol errors. mutationErrorResult supplies version, code, message, outcome, and retryable fields.
- addInputTool normally adds serialized JSON TextContent beside structuredContent, but capture_screen supplies ImageContent and therefore does not receive the JSON text compatibility fallback.
- addSanitizedInputMutationTool protects media URL/path validation from value reflection. keyboard uses the ordinary schema-validation path, whose detailed validation error may echo rejected typed text to the trusted caller transcript.
- discovery_manifest.go advertises a static tools-only surface with public cache scope and ttlMs=0; list ordering and the manifest fixture are deterministic.
- docs/product-contract.md documents all 18 tools, typed results/errors, deprecation, and stronger-than-default pre-1.0 compatibility rules. README.md and docs/threat-model.md agree that annotations communicate consequences but do not enforce authority.

**Implementation And Documentation Agreement**

Implementation, manifest fixtures, README, threat model, and product contract substantially agree on the static 18-tool surface, one-purpose replacements, structured results, tool-versus-protocol errors, and unknown-outcome behavior. The main conflicts are semantic rather than undocumented behavior: five true idempotent hints can invite behavior the descriptions forbid, and false destructive hints understate keyboard/mouse consequences. Documentation acknowledges schema errors may echo rejected values to the same trusted caller, but that is an avoidable privacy weakness for typed text. Capture's lack of JSON text metadata falls short of an MCP SHOULD, not the repository's declared executable contract.

**Current State**

partial: the static, typed, fixture-locked surface and consequence-rich descriptions are unusually strong, but two annotation classes materially miscommunicate safety; some caller-controlled identifiers are unbounded; typed-text validation may reflect private input; one deprecated multiplexed tool remains; and capture omits the recommended structured-result text fallback.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- An MCP client uses idempotentHint=true to retry wake_host_usb after a timeout; the first HID wake was delivered but its acknowledgement was lost, so a second physical input is emitted contrary to the server's unknown-outcome warning.
- A client suppresses confirmation because keyboard or mouse is marked non-destructive; injected or mistaken arguments type a command or click a destructive control on the attached host.
- Rejected keyboard text containing a password or secret is incorporated into a detailed schema-validation error and retained in the client transcript or diagnostics.
- An oversized device/target alias or key string consumes avoidable parsing, validation, error, or logging resources before handler rejection; HTTP request limits reduce but do not define the tool contract.
- The deprecated virtual_media tool's single destructive/open-world annotation cannot accurately represent its read-only status branch versus local-file, URL-fetch, upload, and unmount branches, so a client receives misleading whole-tool metadata.
- A provider regression returns the wrong action/status string; permissive output fields still validate and silently expand the public result vocabulary.
- A legacy client consumes only TextContent and receives an image from capture_screen without the structured metadata's JSON fallback.

**Affected Assets And Trust Boundaries**

Affected assets are host integrity and availability, typed secrets, captured screen confidentiality, JetKVM media state, physical power state, caller compatibility, and maintainer review capacity. Boundaries are MCP client to server arguments/metadata, server to configured JetKVM, JetKVM to attached host through HID/power/media, and server results back into an agent transcript. The server trusts an authenticated/authorized caller with full configured-device authority; annotations still shape client decisions but are not an enforcement boundary.

**Plausible Impact**

Incorrect retry or consequence hints can cause duplicate physical input, unintended commands, power/media changes, or lost caller confirmation. Validation reflection can expose sensitive typed text inside otherwise trusted but retained transcripts. Unbounded strings and weak output vocabularies mainly create robustness and compatibility drift rather than an independent high-severity exploit under existing request limits. Retaining the multiplexed legacy tool permanently increases review, fixture, documentation, and compatibility cost and preserves ambiguous metadata.

**Existing Controls**

- Static, deterministic tool registration with no runtime list mutation or third-party metadata ingestion.
- Strict object schemas, shallow local anyOf branches, startup schema resolution, and no external JSON Schema references.
- Rich descriptions and explicit destructive/idempotent/open-world annotations on every tool.
- Typed success results, declared output schemas, pre-send output validation, and stable redacted mutation-error envelopes.
- Manifest contract tests across in-memory, stdio, and stateless HTTP transports.
- One-purpose power and media tools with the deprecated combined tool clearly labeled and documented.
- Descriptions explicitly prohibit blind retries after unknown mutation outcome.
- No prompts, resources, dynamic tools, icons, custom metadata, or raw JetKVM RPC tool enlarges the public MCP surface.

**Residual Risk**

The static fixture prevents silent metadata drift but faithfully locks in misleading hint values until corrected. Descriptions do not reliably compensate when clients consume annotations mechanically. A trusted caller can always intentionally operate the host, so the server cannot eliminate excessive-agent-authority risk; it can make consequences accurate and errors non-leaking. Hardware semantics of state-setting operations remain unqualified.

**Compatibility Or Semver Effect**

Changing annotations and tightening previously accepted input lengths alters observable API behavior; under the repository's product contract these should ship deliberately with the already-required v1.0.0 boundary and regenerated compatibility fixture. Removing jetkvm_virtual_media is breaking and should occur at v1 only after migration notes and a check for actual users. Tightening output schemas without changing emitted values is contract clarification but should still be fixture-reviewed. Adding JSON TextContent to capture is additive, though content ordering must remain stable. Retaining the remaining 17 tools makes them the intended long-lived v1 compatibility surface.

**Privacy Effect**

Value-free validation errors for keyboard prevent rejected typed text, including credentials, from being copied into agent transcripts or diagnostics. The change does not add collection or a new log. The capture result remains sensitive regardless of adding a metadata text part; only nonsensitive metadata, never image bytes, should be serialized there.

**Operational Effect**

The proposed changes do not add services, state, approval infrastructure, or runtime control planes. They improve client confirmation/retry cues and reconciliation behavior. Contract snapshots make releases slightly more deliberate. A positive static list TTL could reduce repeated discovery but is optional and not an availability control.

**Maintainer Effect**

Removing the deprecated multiplexed tool eliminates a handler, conditional schema, tests, documentation, and permanent compatibility branch. Correcting annotations and bounds is low recurring cost because the manifest fixture detects drift. Keeping keyboard and mouse as bounded multiplexed operation families avoids a tool explosion. Every new public tool after v1 should require a distinct consequence/input/result need and representative compatibility cases.

#### Decision

**Disposition**

simplify

**Recommendation**

- Retain the static tools-only architecture, typed results/errors, one-purpose power/media tools, shallow local schemas, rich descriptions, and manifest contract gate.
- Before public v1, set keyboard and mouse destructiveHint=true. Set wake_host_usb and wake_host_lan idempotentHint=false; conservatively set all five currently true mutation idempotent hints false unless physical/firmware qualification proves repeat safety and the documented no-retry policy is consciously narrowed.
- Use value-free stable invalid_input errors for keyboard and preferably all tools; add explicit maximums for device/target identifiers, key length, and modifier count/uniqueness.
- Remove deprecated jetkvm_virtual_media at the v1 boundary if live usage evidence does not justify one explicit sunset release. Keep the other 17 tools: each exposes a distinct safe-discovery, observation, input, power, wake, or media contract.
- As P2 cleanup, constrain fixed output vocabularies, add nonsensitive JSON metadata TextContent to capture_screen, and consider a small positive TTL for the immutable public tool list.

**Minimal Practical Change**

Update the annotation table and fixture so keyboard/mouse are destructive and wake operations are non-idempotent; decide the remaining three idempotent hints against qualified repeat semantics. Route keyboard schema failures through a value-free invalid_input path. Add defensible string/array bounds. Make the v1 release decision for the deprecated tool and document its replacement mapping. No architectural abstraction is required.

**Optional Stronger Control**

After external adoption, maintain machine-readable compatibility fixtures for the latest released major and run a small client-interoperability matrix that verifies descriptions, confirmations, content parts, and error presentation. Require a short consequence review for any new tool or annotation change. This becomes worthwhile only when real downstream clients exist; it is not a precondition for the first public release.

**Rejected Or Overengineered Alternatives**

- Do not combine the seven power/wake operations: their consequences and repeat semantics differ and deserve separate descriptions and annotations.
- Do not split each keyboard key, mouse action, or media mode into its own tool: the bounded conditional schemas group operations with the same authority and would otherwise create permanent tool-list noise.
- Do not add a policy engine, server-side human approval workflow, custom consequence taxonomy, or per-tool scopes; the product grants a trusted caller full configured-device authority and confirmation belongs at the MCP client/deployment boundary.
- Do not expose raw JetKVM RPC, dynamic device-generated tools, icons, prompts, resources, or arbitrary metadata merely because the SDK permits them.
- Do not retain the deprecated combined tool indefinitely without demonstrated callers; backward compatibility is not free for a one-maintainer project.

**Rationale**

The repository already has a comprehensible and strongly tested public contract, so replacement frameworks would add more risk than value. The concrete gaps are narrow but consequential: the metadata clients are invited to use contradicts the descriptions in exactly the lost-acknowledgement and host-mutation cases the product treats as safety-critical. Conservative annotations, non-reflecting validation, complete bounds, and a clean v1 surface directly address those failures. Removing one known legacy multiplexor reduces burden while retaining the 17 tools whose distinct consequences or results justify their names.

**Dependencies Or Prerequisites**

- Maintainer decision on whether v1 removes jetkvm_virtual_media immediately or after one documented sunset release.
- Live release/download or issue evidence before assuming no legacy users.
- Representative JetKVM firmware/hardware tests before asserting state-setting calls are safely repeatable after lost acknowledgement.
- A documented maximum alias length aligned between configuration and MCP schemas.
- Coordination with the already-required v1 contract update and release notes.

**Migration Or Rollout Considerations**

Publish a v1 migration table mapping each legacy virtual_media operation to get_virtual_media_status, mount_virtual_media_url, mount_virtual_media_file, upload_virtual_media_file, or unmount_virtual_media. Land annotation/bound changes with an intentional manifest update and explain that false idempotence means callers must reconcile state after unknown outcomes. If real legacy use is found, retain the deprecated tool for one explicit, time-bounded release rather than inventing a permanent compatibility layer. Add capture TextContent without removing ImageContent or structuredContent.

**Priority**

P1: consequence-hint correctness, typed-text error privacy, bounded inputs, and the v1 legacy-tool decision should be complete before public release; output-schema/fallback/cache refinements are P2.

**Implementation Effort**

S for annotation, validation, bounds, and fixture changes; M if legacy removal, migration documentation, and interoperability checks are included.

**Ongoing Maintenance Burden**

low: review the manifest diff whenever a tool contract changes and reconsider the surface only at releases or when downstream evidence appears.

**Confidence**

high for repository behavior and specification interpretation; medium for removal timing and repeat safety because usage and physical qualification evidence are missing.

#### Verification

**Acceptance Evidence**

- The generated manifest contains the approved 17-tool v1 surface or an explicitly time-bounded deprecated exception, with an exact migration map.
- keyboard and mouse are destructive; wake tools are non-idempotent; any remaining true idempotent hint has documented device evidence and reconciled retry wording.
- Invalid keyboard arguments containing a unique secret sentinel return a stable error that does not contain the sentinel on any supported transport.
- All caller-controlled string and array fields have reviewed maxima and negative boundary tests.
- Every success result validates against outputSchema; capture includes ImageContent, structured metadata, and a compatible nonsensitive JSON TextContent part if that P2 change is accepted.
- Focused and transport contract tests pass and the manifest diff is explicitly reviewed as a public API change.

**Proposed Tests Or Checks**

- Table-test all 17 or 18 advertised tools for exact readOnly, destructive, idempotent, and openWorld annotations against a reviewed consequence matrix.
- Run manifest contract tests in memory, stdio, and stateless HTTP and classify every snapshot change as breaking, additive, or corrective.
- Inject oversized device, target, key, modifier, text, path, URL, coordinate, and unknown-property inputs at boundary minus one, boundary, and boundary plus one.
- Use a sensitive sentinel in invalid keyboard text and assert it is absent from result content, server stderr, and captured transport messages.
- Validate representative success and every stable error code against schemas; mutate fixed action/status values in a fake provider to ensure tightened output schemas reject them.
- Exercise capture with a legacy TextContent-only consumer and a modern structured/image consumer.
- Run go test ./internal/mcpserver; it passed at the studied tree on 2026-08-15.

**Negative Or Abuse Cases**

- Lost acknowledgement followed by attempted automatic retry of each power, wake, upload, mount, unmount, keyboard, and mouse mutation.
- Keyboard text containing credentials, control characters, non-ASCII data, maximum-size data, and invalid conditional-operation fields.
- Mouse operations with fields from the wrong anyOf branch, extreme coordinates/deltas, and extra properties.
- Media URL versus local-file tools presented to a client to verify only the URL fetch is open-world.
- Deprecated combined-tool status operation misleadingly inheriting destructive metadata, demonstrating why branch-specific consequences cannot be represented.
- Malformed structured provider output, unknown tool name, invalid arguments, known operational failure, and unknown mutation outcome to preserve protocol/tool error separation.
- Schemas containing recursive or external references if future changes introduce them; startup must reject or bound them rather than dereference network content.

**Evidence Needed Before Claiming Support**

Claim only schema and transport conformance from the current fixtures. Do not claim that true idempotence makes retry safe without lost-acknowledgement tests on supported JetKVM firmware and attached-host effects. Do not claim legacy removal is harmless without checking published releases, issues, downloads where available, and maintainer knowledge. Do not claim broad MCP-client compatibility until representative clients consume image plus structured/text results and present consequence hints correctly.

**Revisit Trigger**

Revisit on any MCP specification or Go SDK change to Tool, ToolAnnotations, JSON Schema, list caching, or CallToolResult; before adding, renaming, or removing a tool; at v1 release; after the first external caller or compatibility complaint; after any duplicate-mutation or secret-reflection incident; when qualified firmware changes repeat semantics; and at least annually while public.

### 7. MCP transports, authentication, authorization, and deployment boundaries

#### Identity And Scope

**Item Name**

MCP transports, authentication, authorization, and deployment boundaries

**Research Question**

At exact initial commit 6e52f0027b13f928b768de0feeab4847ef9ca53e, does jetkvm-mcp expose stdio and MCP 2026-07-28 stateless Streamable HTTP with a security boundary proportionate to one administrative principal and the server's physical authority, and which controls belong in the binary, at a TLS reverse proxy/network deployment, or remain explicit non-goals?

**Scope**

This workstream examines stdio ownership and stream purity; the opt-in HTTP listener; MCP 2026-07-28 stateless POST semantics and cancellation; static bearer parsing, comparison, storage and rotation; bind, Host, Origin, DNS-rebinding and CORS behavior; health exposure; TLS termination and reverse-proxy requirements; request/header/body/time limits; global/per-device admission and external rate limiting; session isolation; OAuth applicability, audience/issuer/scope validation and token passthrough; confused-deputy paths; and secure deployment guidance. It distinguishes the MCP client/process boundary, client-to-HTTP endpoint, proxy-to-backend, server-to-JetKVM, and appliance-to-media-origin boundaries. It does not redesign tool-level consequence communication, JetKVM SSRF controls, release supply chain, or agent prompt-injection defenses except where they cross transport authority.

**Repository Surfaces**

- cmd/jetkvm-mcp/main.go: run, parseArgs, serveHTTP, loopbackHost, shutdown and stream selection
- internal/mcpserver/transport.go: NewHTTPHandler, postOnly, trustedHostAndOrigin, loopbackHTTPHost and requireBearer
- internal/httporigin/origin.go and its unit/fuzz tests
- internal/config/config.go: fileHTTP, HTTPBearerToken, HTTPAllowedOrigins, environment resolution and exact-origin normalization
- internal/mcpserver/transport_test.go, origin_test.go, cancellation_test.go and manifest_contract_test.go
- cmd/jetkvm-mcp/stdio_integration_test.go and main_test.go
- internal/jetkvm/admission_test.go, manager.go and provider/network deadline surfaces
- README.md HTTP Server, configuration, operation-limit and raw-RPC sections
- config.example.yaml HTTP bearer/origin and admission-limit examples
- docs/product-contract.md, docs/threat-model.md, docs/adr/0001-mcp-transports-and-authentication.md and docs/adr/0007-same-origin-browser-http.md
- docs/protocol-sources.md and testdata/mcp-gates/pins.json including dns-rebinding and HTTP-header conformance gates
- github.com/modelcontextprotocol/go-sdk v1.7.0 mcp/streamable.go as the exact dependency implementation used by the pinned tree

**Applicability Stage**

current_private, before_public_release, after_external_adoption

#### Sources And Authority

**Authoritative Sources**

- **Title:** jetkvm-mcp exact studied tree | **Publisher:** BenDManning/jetkvm-mcp | **Url:** https://github.com/BenDManning/jetkvm-mcp/commit/6e52f0027b13f928b768de0feeab4847ef9ca53e | **Publication Or Update Date:** 2026-08-15 | **Version Or Revision:** 6e52f0027b13f928b768de0feeab4847ef9ca53e; identical tracked trees 34e8b4451d76821950c23d7c06958d021700f3a7 and later 71445bc6bf2325e6c683e362393605089c336b63 | **Access Date:** 2026-08-15 | **Supported Claims:** Exact transport code, configuration, tests, threat model, deployment contract and dependency usage.
- **Title:** Streamable HTTP | **Publisher:** Model Context Protocol project | **Url:** https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http | **Publication Or Update Date:** 2026-07-28 | **Version Or Revision:** MCP revision 2026-07-28 | **Access Date:** 2026-08-15 | **Supported Claims:** POST-only per-message transport, removal of protocol sessions and GET stream, mandatory Origin validation, localhost binding guidance, authentication guidance, per-request cancellation and required metadata/header validation.
- **Title:** stdio transport | **Publisher:** Model Context Protocol project | **Url:** https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio | **Publication Or Update Date:** 2026-07-28 | **Version Or Revision:** MCP revision 2026-07-28 | **Access Date:** 2026-08-15 | **Supported Claims:** Newline-delimited protocol traffic on stdin/stdout, logging on stderr, stdout purity, cancellation and shutdown behavior.
- **Title:** Authorization | **Publisher:** Model Context Protocol project | **Url:** https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization | **Publication Or Update Date:** 2026-07-28 | **Version Or Revision:** MCP revision 2026-07-28 | **Access Date:** 2026-08-15 | **Supported Claims:** Authorization is optional; stdio should use environment credentials; HTTP implementations choosing MCP authorization should conform to protected-resource discovery, audience validation, token isolation, errors and scope behavior.
- **Title:** Security Best Practices | **Publisher:** Model Context Protocol project | **Url:** https://modelcontextprotocol.io/docs/2026-07-28/tutorials/security/security_best_practices | **Publication Or Update Date:** 2026-07-28 | **Version Or Revision:** Guidance accompanying MCP revision 2026-07-28 | **Access Date:** 2026-08-15 | **Supported Claims:** Local server restriction, token-passthrough prohibition, audience separation, confused-deputy preconditions, proxy/stdio privilege boundaries and least-privilege deployment guidance.
- **Title:** HTTP Semantics | **Publisher:** IETF | **Url:** https://www.rfc-editor.org/rfc/rfc9110.html | **Publication Or Update Date:** 2022-06 | **Version Or Revision:** RFC 9110 | **Access Date:** 2026-08-15 | **Supported Claims:** HTTP authentication schemes are case-insensitive tokens; credentials are conveyed in Authorization; origin is scheme, host and port.
- **Title:** The OAuth 2.0 Authorization Framework: Bearer Token Usage | **Publisher:** IETF | **Url:** https://www.rfc-editor.org/rfc/rfc6750.html | **Publication Or Update Date:** 2012-10; updated by RFC 8996 and RFC 9700 | **Version Or Revision:** RFC 6750 | **Access Date:** 2026-08-15 | **Supported Claims:** Bearer possession grants authority; tokens require TLS-equivalent confidentiality, must not be placed in URLs, and need protection between TLS front end and backend.
- **Title:** Best Current Practice for OAuth 2.0 Security | **Publisher:** IETF | **Url:** https://www.rfc-editor.org/info/rfc9700/ | **Publication Or Update Date:** 2025-01 | **Version Or Revision:** RFC 9700 / BCP 240 | **Access Date:** 2026-08-15 | **Supported Claims:** OAuth deployments should restrict token privileges and audience, prevent replay, protect reverse-proxy security headers and secure the proxy-to-backend link.
- **Title:** Package net/http | **Publisher:** The Go Project | **Url:** https://pkg.go.dev/net/http@go1.26.5 | **Publication Or Update Date:** 2026-07 Go 1.26 documentation | **Version Or Revision:** Go 1.26 net/http Server | **Access Date:** 2026-08-15 | **Supported Claims:** ReadHeaderTimeout does not bound body reads when ReadTimeout is zero; IdleTimeout covers waiting for the next request; MaxHeaderBytes bounds parsed header bytes; WriteTimeout has whole-handler consequences.
- **Title:** Go SDK Streamable HTTP implementation | **Publisher:** Model Context Protocol project | **Url:** https://github.com/modelcontextprotocol/go-sdk/blob/v1.7.0/mcp/streamable.go | **Publication Or Update Date:** 2026-07-28 release line | **Version Or Revision:** go-sdk v1.7.0 | **Access Date:** 2026-08-15 | **Supported Claims:** Stateless mode creates a temporary session per request, ignores MCP session IDs, propagates cancellation, and applies DefaultMaxRequestBodyBytes of 4 MiB when the option is zero.

**Source Class**

- The exact repository and pinned SDK source are repository_evidence.
- MCP transport and authorization pages are normative_specification; MCP Security Best Practices is official_documentation with normative requirements where it quotes the specification.
- RFC 9110 and RFC 6750 are normative_specification; RFC 9700 is a Best Current Practice.
- Go net/http documentation is official_documentation.

**Normative Status**

- MCP Streamable HTTP servers MUST validate present Origin and return 403 when invalid; local servers SHOULD bind localhost and SHOULD authenticate connections.
- MCP authorization is OPTIONAL. If the product chooses the MCP OAuth authorization feature for HTTP, it SHOULD conform to that feature; its token validation, audience and non-passthrough rules then contain MUST requirements. The current pre-shared deployment secret is explicitly not claimed as MCP OAuth.
- The stdio stdout/stderr framing rules are MCP MUST/MAY requirements and are implemented.
- RFC 9110 makes auth-scheme matching case-insensitive, exposing an interoperability defect in the current exact `scheme == "Bearer"` parser.
- TLS, proxy-link protection and OAuth audience controls are normative for OAuth bearer deployments; for this non-OAuth static secret they remain directly applicable security guidance because possession still grants all authority.
- The choice to keep a single principal, reject embedded policy/OAuth machinery and place rate limiting at deployment is a repository-specific judgment.

**Source Disagreements**

There is no conflict in declining OAuth: the MCP authorization feature is optional, and stdio is specifically advised not to use it. The apparent tension is that Streamable HTTP says servers should authenticate while the product permits unauthenticated loopback HTTP; localhost-only reachability plus exact Host/Origin protection is a documented single-host deployment choice, while non-loopback startup requires a secret. The resolution is to retain this narrow mode and not advertise it as multi-user authorization. RFC 6750 requires TLS for bearer access, whereas the binary allows a token on a directly bound plain-HTTP protected interface; resolve this by making TLS or equivalent protected transport a deployment requirement and continuing to recommend loopback behind TLS. The threat model says no request-body cap exists at the wrapper, while the pinned SDK actually defaults to 4 MiB; resolve by documenting the dependency control accurately and setting it explicitly so a security limit is not inherited silently.

#### Repository Evidence

**Exact Baseline Commit**

Exact initial studied commit: 6e52f0027b13f928b768de0feeab4847ef9ca53e. Its tracked tree is identical to 34e8b4451d76821950c23d7c06958d021700f3a7 and later merged 71445bc6bf2325e6c683e362393605089c336b63. It differs from mission-expected 176ec421f9ee6c801517180e1ad0ec9c84570e8e only by removal of bespoke CI policy machinery. Current checkout was 71445bc with pre-existing untracked .agents/, jetkvm_mcp_2026_public_release_baseline/, and skills-lock.json; those are not product evidence.

**Current Repository Evidence**

- Stdio is the default and opens no HTTP listener. server.Run uses MCP IOTransport over supplied stdin/stdout, all lifecycle and telemetry output goes to stderr, and subprocess tests enforce malformed-input recovery, clean EOF and stdout protocol purity. Process/stream ownership is therefore the only principal boundary.
- HTTP exists only with `--http HOST:PORT`. `serveHTTP` rejects a non-loopback bind when the resolved static bearer is empty; the documented direct default is 127.0.0.1:8080. `localhost`, IPv4 loopback and IPv6 loopback are accepted as local bind hosts.
- NewHTTPHandler configures SDK v1.7.0 Stateless=true, JSONResponse=true and PropagateRequestCancellation=true. GET and DELETE receive 405, no Mcp-Session-Id is minted, and each request receives a temporary SDK session. HTTP cancellation reaches the tool handler and device-work admission is shared across requests.
- trustedHostAndOrigin runs before bearer and method/MCP handling. Malformed Host is 403. Loopback Hosts are intrinsically admitted; every public Host must match the authority of an exact configured HTTP(S) origin. Wildcards, credentials, paths, queries, fragments, invalid ports and duplicates are rejected during strict configuration loading.
- A missing Origin is accepted for native clients after Host admission. A present Origin must be exactly one nonempty parseable HTTP(S) origin whose effective host equals request Host; public scheme/authority must be configured and loopback scheme must reflect direct request TLS. Invalid, duplicate, opaque null and foreign values are 403 before the bearer challenge. Extensive unit/fuzz tests and official dns-rebinding/header gates cover this matrix.
- The endpoint emits no CORS grant headers. Same-origin OPTIONS is authenticated if a token exists and then receives 405; foreign preflight is 403. Separately hosted browser clients are explicitly unsupported. `allowed_origins` is therefore Host/origin admission, not ambient browser authorization.
- requireBearer hashes the configured and provided values with SHA-256 and compares the fixed-size digests with subtle.ConstantTimeCompare. The token comes only from a named nonempty environment variable and is not logged or passed to JetKVM. Any holder receives every configured device and tool; annotations do not authorize.
- The bearer parser accepts only exactly `Bearer` followed by one ASCII space. RFC 9110 defines auth-scheme as case-insensitive, so `bearer token` is incorrectly rejected. It also uses Header.Get rather than explicitly rejecting multiple Authorization field lines, leaving ambiguous proxy-normalization behavior untested. There is no token strength validation, expiry, overlap rotation, revocation, issuer, audience, identity, scope or per-tool/device ACL.
- The server deliberately does not implement MCP OAuth, protected-resource metadata, client registration, user consent or scopes. Its pre-shared secret is deployment authentication, not an OAuth access token claim. It never forwards that secret or an MCP Authorization value to JetKVM, eliminating token-passthrough and third-party OAuth confused-deputy paths.
- The Go HTTP server sets ReadHeaderTimeout=10s, IdleTimeout=60s and MaxHeaderBytes=1 MiB, then uses a five-second graceful shutdown. It has no ReadTimeout or per-request body-read deadline. The pinned SDK supplies a 4 MiB MaxBytesReader default, but the product does not set that security-relevant value explicitly. WriteTimeout is intentionally absent, avoiding blind cancellation of long/ambiguous mutations but leaving response duration governed by operation contexts and deployment.
- Manager admission defaults to 16 total operations, four per device, eight sessions, two captures and two decoders; exhaustion is immediate busy/not_sent and mutation work is per-device serialized. These protect expensive downstream work but are not request-rate, per-principal or connection limits. `/healthz` is unauthenticated and bypasses Host/Origin middleware, returning only `ok`; this is a small intentional liveness exposure, not device authority.
- TLS is not implemented in-process. README and ADRs recommend loopback behind a TLS reverse proxy, require preserving the external Host, and identify routing, bearer rotation, rate limiting, process limits and proxy hardening as deployment work. The application ignores forwarded-host/proto identity rather than trusting spoofable headers.
- README, product contract, threat model and ADRs consistently state one bearer/stdio owner has full authority, native Origin omission is supported, same-origin browser only, no OAuth/CORS/policy engine, TLS at deployment, and no automatic retry of unknown physical mutations.

**Implementation And Documentation Agreement**

Code, tests, README, product contract, threat model and ADRs agree on the principal architecture and non-goals. Exact Host/Origin behavior, middleware order, loopback default, non-loopback token gate, stateless/no-SSE behavior and CORS rejection are executable and tested. Documentation correctly says the static secret is not MCP OAuth and is never a per-user authorization layer. Two corrections are needed: threat-model wording should acknowledge the pinned SDK's default 4 MiB body cap while noting it is not explicit at the wrapper, and HTTP Bearer parsing should follow RFC 9110 case-insensitive scheme matching. The non-loopback startup gate, duplicate Authorization handling, slow body behavior and proxy deployment are not directly tested end to end.

**Current State**

substantially_satisfied: the selected one-principal boundary is clear, narrow, well tested and aligned with MCP 2026-07-28. Before public release, fix standards-compliant bearer parsing, explicitly own the request-body cap/read deadline, test the non-loopback startup gate, and make proxy/TLS/rate-limit/token-strength requirements operationally concrete. OAuth, scopes and policy machinery remain unjustified.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- A malicious website rebinds a hostname to loopback and POSTs to /mcp. Without validation it could exercise power/HID; current Host and present-Origin checks reject it before authentication and dispatch.
- An operator binds 0.0.0.0 with no token. Startup refuses. If the operator supplies a weak/reused token over cleartext, any observer or log reader can replay it for every device/tool because there is no expiry or scope.
- A reverse proxy rewrites Host to the backend name. Legitimate public requests fail 403; an operator tempted to wildcard admission cannot because wildcards are rejected. The safe fix is preserving exact external Host, not weakening application checks.
- A slow client sends a 4 MiB-or-smaller body indefinitely. ReadHeaderTimeout has expired after headers and IdleTimeout does not apply to an active request; without ReadTimeout/proxy enforcement the connection can consume resources before MCP admission.
- A conforming HTTP client sends `authorization: bearer <secret>`. Header names work case-insensitively in Go, but the current scheme-value comparison rejects it, causing avoidable interoperability failure.
- Multiple Authorization field lines are normalized differently by proxy and backend. The current Header.Get behavior is not an auth bypass shown by evidence, but ambiguity at a credential boundary warrants fail-closed rejection.
- A native client omits Origin and sends an admitted Host plus stolen bearer. This is intentionally authorized: Origin is browser/DNS-rebinding evidence, not client identity. TLS, token protection and routing are decisive.
- A caller retries an HTTP request after connection loss. Stateless transport does not make a possibly delivered power/HID/media mutation safe to retry; result outcome classification and independent reconciliation remain necessary.
- An OAuth token intended for another service is sent to this server. The current static comparator rejects it unless it coincidentally equals the configured secret, and no token is passed downstream. Adding generic JWT acceptance without audience/issuer validation would create the prohibited confused-deputy path.
- A compromised stdio MCP host already owns the child streams/environment and can invoke every tool. Adding server-side OAuth inside stdio would not restore trust; client process isolation and least privilege own this boundary.

**Affected Assets And Trust Boundaries**

Assets include JetKVM administrator credentials/sessions, screen data, HID and physical power authority, host availability/data, media, the shared MCP bearer, configuration environment, stdout protocol integrity, process/network capacity and deployment metadata. Boundaries are MCP host to child stdio; native/browser client to HTTP; public TLS proxy to loopback backend; bearer holder to all configured tools/devices; server to JetKVM; appliance to media origins; and MCP caller cancellation to possibly delivered physical effects.

**Plausible Impact**

Transport compromise grants screen observation and consequential HID, reset, power, wake and media operations, potentially causing credential disclosure, host takeover, interruption or data loss. Slow-client/resource abuse can deny the one process. Proxy/Host mistakes can either break availability or, if operators weaken controls, expose full authority. OAuth/token passthrough mistakes could broaden token replay across services. Standards-incompatible bearer parsing primarily causes availability/interoperability failure.

**Existing Controls**

- Stdio default, stdout purity, stderr diagnostics and process-lifecycle tests.
- Explicit HTTP opt-in, loopback default and non-loopback startup bearer requirement.
- Stateless JSON Streamable HTTP, POST-only endpoint, no legacy SSE/session IDs and propagated cancellation.
- Exact fail-closed Host/Origin parser, wildcard rejection, same-origin browser-only policy, no CORS headers and extensive unit/fuzz/conformance gates.
- SHA-256 plus constant-time static secret comparison, environment-only loading, no logging and no downstream forwarding.
- SDK 4 MiB body cap, HTTP header/read-header/idle bounds, device-work admission and bounded operation/network/decoder contexts.
- TLS/proxy/network/process requirements and single-principal/no-OAuth limitations documented in maintained threat model and ADRs.

**Residual Risk**

One secret is a replayable, non-expiring all-authority credential; TLS, entropy, storage, rotation and ingress limits are external. Loopback trust depends on local process/user isolation. Native clients legitimately omit Origin. Active request body reads have no binary-owned deadline, rate limiting is absent, and health is public on the listener. Statelessness removes session confusion but not duplicate physical effects. No per-user audit or attribution exists. These are acceptable only for the stated one-maintainer/single-principal deployment and must trigger redesign if multiple principals appear.

**Compatibility Or Semver Effect**

Case-insensitive Bearer acceptance is backward-compatible. Rejecting duplicate Authorization is a compatible security hardening because ambiguous credentials are not a meaningful supported input. Explicitly retaining the SDK's 4 MiB body cap is behavior-preserving; lowering it or adding a read deadline must exceed all documented valid schemas and be classified for existing clients. Adding OAuth, scopes, CORS, sessions or per-device ACLs would create substantial configuration/wire/deployment surfaces and at least a minor addition, possibly a major change to authentication expectations. Removing a supported transport is major under the product contract.

**Privacy Effect**

Transport secrecy is essential because the bearer, screenshots, typed text and device state cross HTTP. TLS and Authorization-log suppression belong at the proxy. The application currently emits no request/payload log and rejects Origin before exposing a bearer challenge. OAuth would add identity, consent, metadata and audit data that the product does not currently collect. Rate-limit logs should remain metadata-only if later added.

**Operational Effect**

The recommended small changes add deterministic auth/read-boundary tests and a concrete proxy checklist without creating another service. Operators must generate and rotate a strong secret, preserve Host, protect proxy-to-backend reachability, set connection/rate/resource limits and avoid automatic mutation retries. An application ReadTimeout must be chosen to bound only body ingestion, not indiscriminately cancel long physical operations.

**Maintainer Effect**

The current boundary is maintainable because it uses standard net/http middleware, one optional secret and no identity database. Correcting parser semantics and owning limits is low recurring cost. Embedded OAuth, token validation keys, scopes, consent, multi-tenancy, CORS credentials or a policy engine would impose high protocol/security/operations maintenance and are not justified for one principal. A documented proxy example has modest update burden but should avoid claiming support for many proxy vendors.

#### Decision

**Disposition**

change

**Recommendation**

Retain stdio as the preferred local path; retain stateless JSON Streamable HTTP, exact Host/Origin admission, same-origin browser-only behavior, non-loopback secret gate, deployment TLS and the explicit absence of OAuth/scopes/policy. Before public release, make Bearer auth-scheme matching case-insensitive and reject ambiguous duplicate Authorization fields; explicitly configure the SDK request-body cap; add a bounded body-read timeout or require and verify it at the ingress proxy; test the non-loopback bind gate; and provide one concise secure reverse-proxy deployment checklist. Describe the static value as a pre-shared administrative secret, not an OAuth access token, and state it must be high entropy, protected by TLS/equivalent transport, omitted from logs and rotated by restart. Keep rate limiting at the ingress unless observed unauthenticated parsing pressure justifies an in-process limiter.

**Minimal Practical Change**

Change requireBearer to compare the scheme with strings.EqualFold, accept one syntactically valid Authorization credential only, and fail closed on multiple field lines; add table tests. Set StreamableHTTPOptions.MaxRequestBodyBytes explicitly to the reviewed bound already supplied by SDK v1.7.0, add a product-level oversize test, and set a conservative HTTP ReadTimeout sufficient for the maximum body or document a mandatory proxy body-read timeout if application timing would be unsafe. Add focused serveHTTP tests proving non-loopback-without-secret refusal before listen and loopback behavior. Expand README with a 128-bit-or-stronger generated secret example and proxy requirements: TLS, preserved Host, stripped untrusted forwarded headers, backend isolation, rate/connection/body timeout, and Authorization redaction.

**Optional Stronger Control**

After multiple principals or externally operated shared deployments exist, place a mature OAuth-aware gateway/authorization server in front and make jetkvm-mcp validate gateway identity or adopt MCP OAuth as a separately designed resource server using maintained libraries, protected-resource metadata, issuer/audience/expiry/signature validation, narrow step-up scopes and no token passthrough. At that point add per-principal quotas and privacy-safe audit events. Prefer the external gateway first; embed OAuth only if direct MCP client interoperability is a demonstrated requirement.

**Rejected Or Overengineered Alternatives**

- Reject embedded OAuth, DCR/CIMD, consent UI, user database and JWKS lifecycle before there are multiple principals or a direct-client requirement.
- Reject per-tool/device policy engines and AI gateways; they create a second product and cannot make a compromised all-authority stdio host trustworthy.
- Reject wildcard/reflected CORS and separately hosted browser support without a browser identity and credential model.
- Reject stateful Streamable HTTP, legacy HTTP+SSE, durable sessions and distributed session stores; MCP 2026-07-28 and the static tool surface need none.
- Reject trusting X-Forwarded-Host/Proto from arbitrary clients; preserve external Host and isolate the backend instead.
- Reject application WriteTimeout as a blunt global operation deadline because cancellation of a possibly delivered physical mutation can create unknown outcomes; use operation-specific deadlines and ingress body/read limits.
- Reject in-process distributed rate limiting or per-IP identity before a real exposed deployment exists; use the TLS ingress/network boundary.

**Rationale**

Every MCP tool has administrator-level device or host consequences, so reachability and credential handling are the primary security boundary. The repository already implements the MCP-required Origin protection and preferred localhost deployment, while its stateless/no-session shape reduces state isolation and cleanup risk. The static secret is proportional for one principal and avoids every current OAuth proxy/passthrough precondition. The remaining issues are concrete and small: RFC 9110 interoperability, ambiguous credential fields, silently inherited SDK body policy, missing active-read deadline and incomplete deployment verification. Solving those yields more value than importing identity machinery whose security obligations would exceed the product requirement.

**Dependencies Or Prerequisites**

- Choose and document the explicit MCP request body cap above every valid current schema payload.
- Measure an adequate body-read deadline for supported native/proxied clients without applying a global mutation completion timeout.
- Select one generic reverse-proxy baseline or vendor-neutral checklist and define protected backend reachability.
- Maintainer decision on whether direct non-loopback cleartext is merely technically possible or explicitly unsupported without equivalent protected networking.
- Any future OAuth work requires an accepted multi-principal/direct-client objective, authorization server choice, scope model and privacy/audit decision.

**Migration Or Rollout Considerations**

Ship parser acceptance first because it only broadens standards-conforming input. Reject duplicate Authorization with a clear 401 that never echoes credentials. Explicitly set the existing 4 MiB cap before considering a lower value; inventory all tool schemas before narrowing. Stage body-read deadlines against slow legitimate clients and proxies, and never translate a transport timeout into safe mutation retry. Publish the proxy checklist before recommending remote exposure. Existing single-secret deployments rotate by updating the environment and restarting; document brief downtime and no dual-token window rather than adding a token database.

**Priority**

P1: small standards and denial-of-service boundary corrections before public release; OAuth redesign is deferred.

**Implementation Effort**

S for parser, explicit limits, focused tests and deployment documentation.

**Ongoing Maintenance Burden**

low for the recommendation; medium to high only if OAuth, per-principal policy or multi-proxy support is later adopted.

**Confidence**

high for code/spec findings; medium for deployment adequacy because no real proxy/TLS/rate-limit environment was inspected.

#### Verification

**Acceptance Evidence**

- Unit tests accept Bearer auth scheme in representative casing, reject wrong schemes, missing/empty tokens and duplicate Authorization fields, and never echo the configured/provided secret.
- Server tests prove non-loopback bind without secret fails before listening, loopback works without secret, and public Host/Origin plus bearer ordering remains unchanged.
- A product-owned body cap test rejects cap+1 and accepts the largest valid tool request; a slow-body test proves bounded connection lifetime at either application or documented ingress.
- Existing origin fuzz/tests and pinned dns-rebinding/http-header conformance gates pass.
- A sanitized proxy smoke demonstrates TLS, exact preserved Host, same-origin behavior, missing/invalid bearer rejection, no CORS grant, cancellation propagation and no Authorization logging.
- README/threat model distinguish pre-shared authentication, MCP OAuth non-support, application controls, deployment controls and non-goals.

**Proposed Tests Or Checks**

- Table-test `Bearer`, `bearer`, `BEARER`, wrong scheme, no space, extra whitespace, empty token, duplicate lines and comma-combined ambiguity.
- Exercise `serveHTTP` with an injectable listener or pre-listen validation helper for 0.0.0.0, public IP, localhost, 127.0.0.1 and ::1.
- Send oversized, chunked and deliberately slow bodies; verify 413/400-style bounded failure, connection cleanup and no device dispatch.
- Retain Host/Origin matrices for default ports, IPv6 canonicalization, malformed authority, missing/duplicate/null Origin, public native no-Origin and reverse-proxy Host.
- Smoke an actual TLS proxy with direct backend inaccessible externally, exact allowed origin, Authorization log redaction, connection/rate limits and controlled shutdown/cancellation.
- Review every SDK update for MaxRequestBodyBytes, cancellation, localhost protection, stateless/session debug flags and header validation changes.

**Negative Or Abuse Cases**

- DNS rebinding to localhost; forged Host; missing, duplicate, whitespace-only, opaque null, foreign, cross-scheme and wildcard Origin; unbracketed IPv6 and default-port confusion.
- Missing, lowercase, duplicate, oversized, weak, stale, leaked and logged bearer; token in query; proxy-to-backend cleartext observation; header smuggling/normalization disagreement.
- Slowloris headers and bodies, chunked oversize bodies, many unauthenticated connections, repeated cheap discovery and admitted expensive captures/uploads, canceled clients and shutdown with active work.
- Mcp-Session-Id and legacy GET/DELETE/SSE attempts; required metadata header/body mismatch; client disconnect followed by unsafe mutation retry.
- Cross-origin credentialed browser preflight, reflected/wildcard CORS, proxy rewriting Host, spoofed forwarded headers and publicly reachable health endpoint.
- Foreign-audience JWT or downstream token presented as MCP credential, token passthrough attempt, OAuth metadata discovery injection and one principal impersonating another after future scope expansion.

**Evidence Needed Before Claiming Support**

Direct local stdio support needs stdout/stderr, cancellation and EOF evidence under a trusted launching principal. Remote HTTP support needs exact MCP revision conformance, bind/Host/Origin/bearer/body/deadline tests and a named TLS/proxy/network baseline. Same-origin browser support needs actual browser/proxy interoperability without CORS; cross-origin is unsupported. OAuth support would require protected-resource metadata, authorization-server discovery, secure client registration choices, issuer/audience/signature/expiry/scope validation, correct 401/403 challenges, TLS and explicit no-passthrough tests. Multi-user or per-device authorization requires principal identity, policy semantics, audit/privacy and quota evidence. Stateless build evidence alone does not prove a deployment is securely reachable.

**Revisit Trigger**

Revisit on an MCP transport/authorization or Go SDK revision; any Host/Origin/auth bypass; public remote deployment; token disclosure; slow-client/resource incident; proxy product change; demand for cross-origin browser access, sessions, server-initiated features, multiple principals or differentiated tool/device rights; or annually before a supported release. OAuth becomes relevant only at the explicit multi-principal/direct-client trigger.

### 8. Agentic-AI and MCP-specific threat landscape

#### Identity And Scope

**Item Name**

Agentic-AI and MCP-specific threat landscape

**Research Question**

For jetkvm-mcp's exact bounded architecture, which agentic-AI and MCP attacks have concrete paths to screen, device, host, media, credential, or availability harm; which controls can this server prevent or reduce; and which decisions must remain with the MCP client or deployment rather than becoming an in-product AI policy system?

**Scope**

This item covers direct and indirect prompt injection, tool-description poisoning, tool shadowing and name collision, metadata mutation/rug pulls, malicious metadata and tool results, cross-server confusion and toxic flows, excessive agency, confused-deputy behavior, schema manipulation and complexity abuse, unsafe retries, secret exfiltration, malicious MCP clients, compromised agents, and multi-server composition. It maps those threats through stdio/HTTP caller boundaries, static tool discovery, JetKVM/host-derived results, screenshots, HID/power/wake/media effects, and privacy-safe telemetry. It does not assess model training, client-specific approval UI, general HTTP hardening, firmware exploitability, or contribution/supply-chain controls except where they form a direct MCP poisoning path.

**Repository Surfaces**

- docs/threat-model.md trusted-principal model, boundaries B1-B8, assets, consequence map, privacy flows, residual risks, and deliberate exclusions
- docs/product-contract.md supported capability and consequence boundaries
- README.md deployment model, full tool table, annotation warning, privacy warning, unknown-outcome guidance, and physical qualification limits
- internal/mcpserver/server.go and controls.go static 18-tool registration, names, descriptions, annotations, schemas, bounded inputs, and typed outputs
- internal/mcpserver/errors.go operational error taxonomy and mutation outcome handling
- internal/mcpserver/testdata/tool-manifest.json and manifest_contract_test.go exact metadata integrity gate
- internal/mcpserver/discovery_metadata_test.go, server_test.go, error_result_test.go, typed_results_test.go, telemetry_test.go, output_validation_test.go, cancellation_test.go, and origin_test.go
- internal/jetkvm/manager.go typed status projection and constant warning labels, session.go delivery classification, admission.go capacity limits, and associated tests
- internal/telemetry privacy-minimized event schema and recorder tests
- internal/config strict configuration, media roots/origins, named Wake-on-LAN targets, bearer configuration, and tests
- cmd/jetkvm-mcp-mutation-checklist and docs/mutation-validation.md supervised physical-mutation evidence procedure
- testdata/mcp-gates/pins.json absence of prompts/resources/sampling/roots/tasks and exact product capability checks

**Applicability Stage**

current_private, before_public_release, after_external_adoption

#### Sources And Authority

**Source Class**

- Repository commit, code, tests and maintained threat model: repository_evidence.
- MCP tools specification: normative_specification.
- MCP security best practices: official_documentation; its MUST language applies to described authorization/proxy designs, not automatically to this non-OAuth product.
- NIST sources: security_framework and official_documentation.
- OWASP sources: security_framework.
- AgentDojo: peer_reviewed_research.
- MCP Security community taxonomy and Invariant Labs demonstration: secondary_commentary and maintainer_case_study, used for attack patterns rather than product requirements.

**Normative Status**

- MCP tool schemas and output conformance are protocol MUST requirements; annotation fields are optional and untrusted client hints, never authorization.
- MCP's prohibitions on token passthrough apply when OAuth tokens are accepted or proxied. jetkvm-mcp uses one opaque bearer and distinct JetKVM credentials, so the vulnerable OAuth proxy pattern is not implemented.
- NIST and OWASP recommendations are guidance. They support least privilege, action mediation and adversarial evaluation but do not require an in-server model, classifier, policy engine, or approval UI.
- Peer-reviewed and case-study attack results are observations about agents/clients; they do not prove every model or deployment is exploitable.

**Source Disagreements**

- Some community MCP guidance recommends universal tool signing, runtime scanning, behavioral monitoring and sandboxing. For this repository, tool metadata is compiled from reviewed source and frozen by an exact fixture; signing a tool definition separately from the release would not authenticate the running binary. Resolution: secure/review the release and manifest, and make the client pin/review server identity; do not build a new registry or scanner.
- Prompt-injection guidance often proposes classifiers or model instructions, while NIST and empirical agent research do not establish reliable semantic detection. Resolution: treat injection as possible and enforce deterministic authority/data boundaries outside the model; this server should not claim to recognize malicious text or pixels.
- OWASP least-privilege guidance favors narrower authority, while the repository explicitly defines one fully trusted JetKVM administrator. Resolution: keep that honest bounded product model, recommend isolated deployments and client-side approval for high-impact tools, and do not imply that a bearer, schema, or annotation proves human intent.
- MCP confused-deputy guidance focuses on OAuth proxies, static client IDs, consent cookies and token passthrough. Those preconditions are absent here. The applicable confused-deputy path is instead a valid client using this server's device authority for an injected goal; authentication alone cannot distinguish it.

#### Repository Evidence

**Exact Baseline Commit**

The assigned initial commit is 6e52f0027b13f928b768de0feeab4847ef9ca53e with tree 34e8b4451d76821950c23d7c06958d021700f3a7. During research local HEAD was later 71445bc6bf2325e6c683e362393605089c336b63, whose tree is exactly identical. Pre-existing untracked .agents/, skills-lock.json, and jetkvm_mcp_2026_public_release_baseline/ were present. Conclusions are tied to the identical tree, not silently mixed revisions.

**Current Repository Evidence**

- docs/threat-model.md explicitly says the server converts calls from one trusted administrative principal, enforces authentication rather than human intent, has no per-tool ACL/consent/approval/audit identity, and grants a compromised or prompt-injected client that principal's full authority.
- The server contains no LLM, prompt interpreter, MCP prompts/resources/sampling/roots/elicitation/tasks, dynamic tool registration, memory, autonomous loop, or server-to-client request surface. It receives a tool name and structured arguments and dispatches deterministically.
- The 18 tools are statically compiled. The exact manifest snapshots names, titles, safety-focused descriptions, input/output schemas and annotations across in-memory, stdio and HTTP. Any reviewed metadata/schema change fails the fixture until explicitly accepted.
- Names use a jetkvm_ prefix and one-purpose power/media tools make consequences legible. The deprecated multiplexed jetkvm_virtual_media tool remains a larger comprehension/annotation risk and is excluded from physical mutation qualification.
- Tool descriptions explicitly state visible-secret, HID-command, UI-click, data-loss, appliance-fetch, storage-retention, physical-effect and unknown-outcome consequences. Annotations distinguish read-only/destructive/idempotent/open-world hints, and README states annotations are not authorization.
- Schemas reject unknown properties and bound operations. Keyboard text is US-ASCII and at most 4096 bytes; capture dimensions are bounded; device/target select configured identifiers; Wake-on-LAN cannot accept arbitrary addresses; file paths stay below configured roots; URL mount requires an exact configured origin.
- A prompt-injected client can still call keyboard, mouse, force-off, reset, power, wake, upload or mount using any configured device. A valid stdio owner or bearer holder is not asked to prove human approval or task provenance.
- jetkvm_capture_screen returns an image that can contain visible host secrets or attacker-supplied instructions. The server does not OCR or interpret it; a vision-capable client may treat its content as instructions and then invoke this or another server.
- Status projection discards unknown firmware fields and media URLs/paths, uses fixed warning labels for probe failures, and tool errors replace raw firmware detail with a stable code/outcome/retryable taxonomy. Firmware-derived version/extension/state strings and the screen remain untrusted client-visible data.
- Mutations classify failures after possible dispatch as outcome=unknown and retryable=false. Tests cover before-send not_sent, during/after-send unknown, cancellation, timeout, error wording and no automatic retry. The server cannot stop a malicious or defective client from issuing a new call.
- Telemetry records operation class, transport, stage, code, outcome, duration and correlation ID while excluding arguments, screenshots, URLs, paths, credentials, raw errors and device identifiers. It helps detect volumes/failures but cannot attribute intent to an end user or reconstruct exact actions.
- HTTP has one opaque bearer principal and no OAuth, token passthrough, scopes or multi-tenancy. JetKVM passwords/cookies are server-held and are never returned or derived from the MCP bearer.
- The mutation checklist requires an expendable target, independent observation, bounded window, postcondition and stop-without-retry on uncertainty, but it is a qualification procedure, not runtime authorization.

**Implementation And Documentation Agreement**

Code and maintained documentation strongly agree on the actual authority and residual agent risk: a trusted caller has full configured-device control; annotations communicate but do not enforce intent; outputs can be private; unknown mutations must not be retried; optional agent-control planes are absent by design. Tests verify the static metadata and privacy/error claims. The remaining presentation gap is that this unusually important trusted-principal statement lives mainly in the threat model and is less prominent than setup instructions; a public user could connect the server to an autonomous or multi-server client without reading the security model. No documentation can establish that a particular client honors annotations, preserves server provenance, asks for approval, isolates untrusted results, or avoids retries.

**Current State**

substantially_satisfied: the server has a clear, honest, bounded principal model; a static reviewable surface; strong input/result minimization; accurate consequence metadata; non-retryable ambiguous outcomes; and no unnecessary agent features. It cannot prevent client-side prompt injection or excessive autonomy, and it should make the resulting deployment obligations more prominent before public release. Runtime per-user policy, semantic injection scanning, and multi-server governance are intentionally not part of this product.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- **Threat:** Direct prompt injection and compromised agent | **Repository Attack Path:** A user, document, issue, or prior conversation causes the MCP client model to call jetkvm_force_host_power_off, reset, HID, or media tools. The server receives a valid structured call from the authenticated principal and cannot distinguish injected intent from genuine intent. | **Server Handling:** Reduces impact through configured-device selection, schemas, input bounds, capacity limits, consequence descriptions and privacy-safe errors; does not and cannot prevent semantic hijacking under its stated trust model. | **Primary Owner:** MCP client and deployment: trustworthy instruction/data separation, high-impact approval, task-scoped tool enablement, identity/provenance display and least privilege.
- **Threat:** Indirect injection in a captured screen | **Repository Attack Path:** An attached host, webpage, login banner or compromised device displays text instructing a vision-capable agent to reveal secrets or invoke mutations. jetkvm_capture_screen faithfully returns the PNG, and the client may feed it back to its model. | **Server Handling:** Documents that images can contain secrets, bounds/decodes one frame and does not persist it. It cannot reliably classify the semantic content of pixels and should not add an LLM scanner. | **Primary Owner:** Client: treat screen content as untrusted data, do not derive privileged actions from it without trusted user intent and approval, and keep screenshot data out of other servers unless explicitly authorized.
- **Threat:** Tool-description poisoning | **Repository Attack Path:** A malicious contribution/dependency/release changes a description to tell the model to read or transmit secrets or prefer a destructive tool. | **Server Handling:** Static compiled metadata and exact manifest fixture make source-reviewed changes visible; descriptions are consequence-focused. A compromised binary can still lie. | **Primary Owner:** Repository/release integrity and client server-review/pinning. The runtime server cannot self-attest honest metadata.
- **Threat:** Tool shadowing, impersonation and cross-server name confusion | **Repository Attack Path:** A second malicious MCP server exposes the same jetkvm_* name or a similar safer-looking name and poisons selection, or falsely claims to be this server. Conversely, its metadata could instruct the agent to call the real server's mutation tool. | **Server Handling:** The jetkvm_ prefix and stable manifest reduce accidental ambiguity but do not create globally unique authenticated names or control another server. | **Primary Owner:** Client: bind calls and approvals to server identity plus tool name, qualify collisions, show provenance, and require re-approval when a server/manifest changes.
- **Threat:** Rug pull or metadata mutation | **Repository Attack Path:** A new binary/release changes tool behavior, schema, annotations or descriptions after a user originally approved the server. | **Server Handling:** No runtime mutation or dynamic registration exists; changes require a new process artifact and manifest fixture update. This narrows the path to release/update compromise. | **Primary Owner:** Release integrity plus client update review. Pin/review artifact and manifest changes; do not infer continuity from a tool name.
- **Threat:** Malicious metadata and tool-result poisoning | **Repository Attack Path:** A compromised JetKVM controls screen pixels and can influence firmware-derived version, extension or state strings. A client may interpret those values as instructions, make an unsafe follow-on call, or relay them to another server. | **Server Handling:** Typed projections discard unknown raw fields, redact media sources, replace raw errors and use constant warning labels; image and a few bounded protocol values necessarily remain untrusted observations. It detects malformed structures, not malicious semantics. | **Primary Owner:** Client: label tool outputs by trust source, keep data separate from instructions, and authorize downstream actions independently.
- **Threat:** Excessive agency | **Repository Attack Path:** An autonomous client is given the full server and can observe secrets, type commands, click UI, power/reset hosts and mount media without an intervening human. One bearer covers all configured tools/devices. | **Server Handling:** Makes each consequence explicit and keeps the product surface bounded, but deliberately implements no per-tool ACL, scope, consent dialog or human identity. | **Primary Owner:** Deployment/client: connect only a trusted administrative principal; enable high-impact calls only for intended workflows; use client approval and separate narrowly configured instances/security boundaries when needed.
- **Threat:** Confused deputy | **Repository Attack Path:** A prompt-injected but authenticated client uses the server's JetKVM password/session authority to act on a configured appliance. This is authority confusion even though the classic OAuth proxy/code-theft preconditions are absent. | **Server Handling:** Authenticates the calling boundary and uses separate downstream credentials; does not pass through bearer tokens or implement third-party OAuth. It cannot validate the client's semantic purpose. | **Primary Owner:** Client/deployment for intent and approval; server for retaining strict configured targets and never accepting arbitrary downstream credentials/tokens.
- **Threat:** Schema manipulation and complexity abuse | **Repository Attack Path:** A malicious server release publishes misleading schemas, or a malicious client sends extra fields, invalid unions, oversized text/paths/images or parser-complex input to alter dispatch or exhaust resources. | **Server Handling:** Schemas are local/static/snapshotted; typed SDK validation rejects unknown/invalid arguments; handlers add semantic checks and explicit length/range/enum bounds; admission limits concurrent work. No remote party can mutate a schema at runtime. | **Primary Owner:** Server/source review for schema correctness and resource bounds; client must still treat schemas as untrusted metadata before selecting a newly added server.
- **Threat:** Unsafe retry and cascading physical effects | **Repository Attack Path:** A timeout or lost response occurs after HID, reset, power, wake or media dispatch; an agent retries and duplicates a key/click/reset/upload or changes a now-different host state. | **Server Handling:** Classifies possible delivery as outcome=unknown/retryable=false, uses not_sent only before dispatch, marks idempotence conservatively, documents inspection, and never automatically retries mutations. | **Primary Owner:** Client must honor the result, stop automatic retry, inspect through an independent read-only/physical path and obtain new intent. The stateless server cannot recognize a semantically duplicate new request without adding risky idempotency state.
- **Threat:** Secret exfiltration and toxic multi-server flow | **Repository Attack Path:** The server returns a screenshot/status/device alias; a compromised client sends it using another MCP server. Or the client asks local-file media tools to upload a guessed file under the configured media root, or mounts a secret-bearing URL at an allowed origin. | **Server Handling:** Screenshots are explicit private output; status/media state is typed/redacted; errors/telemetry omit raw data; file access is root-confined and only reaches configured JetKVM; URL origins are administrator-allowlisted. It cannot control data after returning it to the trusted caller. | **Primary Owner:** Client for information-flow isolation and cross-server egress; deployment for a non-secret media root, trusted origins, device/network segmentation and not placing secrets in URLs.
- **Threat:** Malicious MCP client and denial/abuse | **Repository Attack Path:** A bearer holder or stdio owner repeatedly invokes bounded but expensive capture/session operations or high-impact mutations, ignores descriptions and lies about client metadata. | **Server Handling:** Authentication, Host/Origin checks, strict schemas, timeouts and global/per-device/session/capture/decoder admission reduce resource abuse. There is no per-principal rate quota or audit identity because there is only one principal. | **Primary Owner:** Deployment must protect the bearer/stdio channel and not share it across mutually distrusting users; reverse-proxy rate controls are appropriate if exposed beyond one trusted client.

**Affected Assets And Trust Boundaries**

Assets include host screen secrets and state, typed credentials/commands, configured device aliases, JetKVM passwords/cookies, HTTP bearer, local media content, allowed media origins, attached-host availability/data integrity, physical power and USB control, appliance storage, and maintainer/release trust. Relevant boundaries are MCP client to stdio process, bearer holder/proxy to HTTP, static discovery metadata to the model, server outputs to client context/memory, compromised host/appliance pixels and typed state into the server, server-held credentials to configured JetKVM, media root/origin to appliance/host, and one MCP server's results to another server through a composing client.

**Plausible Impact**

A successful path can expose visible secrets, type or execute privileged commands, activate a destructive UI control, reset or power off a host, boot/wake equipment, corrupt or lose host data, upload/persist local media, induce an appliance network fetch, create cross-server data exfiltration, or exhaust availability. False or duplicated acknowledgement handling can compound a physical effect. A metadata/release compromise can redirect many clients. Because one caller principal has full configured authority, authentication compromise and prompt-injected agency have nearly the same runtime blast radius.

**Existing Controls**

- One explicitly documented administrative principal; no false per-user or intent claim.
- Static tools-only MCP surface with no prompts, resources, sampling, roots, elicitation, task extension, memory, dynamic registration or raw RPC tool.
- Exact cross-transport manifest fixture and discovery metadata tests, making names/descriptions/schemas/annotations review-visible.
- One-purpose consequence-oriented tools, jetkvm_ naming, bounded schemas, configured target selection and deny-by-default media capabilities.
- Typed/redacted outputs, constant public error messages, no request logger, privacy-minimized telemetry and no credential return.
- Explicit destructive/read-only/idempotent/open-world hints plus prose consequences and an explicit statement that hints are not authorization.
- Mutation delivery taxonomy with non-retryable unknown outcomes and focused before/during/after-send and cancellation tests.
- Admission limits, deadlines and no automatic queuing/retry; physical validation procedure requires independent observation and stop rules.
- Opaque MCP bearer is not forwarded; distinct device credentials are used downstream, avoiding token-passthrough design.

**Residual Risk**

A valid client call is authoritative even if produced by prompt injection, hallucination, poisoned memory, malicious metadata or another server. The server cannot tell whether a screenshot contains an instruction or whether a human approved a force-off. Tool results can still carry adversarial pixels and limited firmware strings, and returned private data can be exfiltrated by the client. A malicious caller can ignore retryable=false and submit a new mutation. Tool metadata integrity ultimately depends on source/release integrity and client review. These risks are acceptable only under the declared trusted-administrator deployment boundary, not for an arbitrary autonomous agent.

**Compatibility Or Semver Effect**

Strengthening warnings, client/deployment guidance and threat tests is non-breaking. Changing tool names, descriptions, annotations or schemas affects model selection and the checked compatibility fixture even before 1.0. Removing the deprecated multiplexed virtual-media tool would be a public compatibility change but reduces permanent ambiguity; follow the repository's deprecation policy. Adding server-side confirmation tokens, dynamic per-client policy or per-user scopes would materially change the stateless contract and should not be smuggled in as a security patch.

**Privacy Effect**

The largest agent-specific privacy path is not server logging but authorized return of a private screenshot or status followed by client memory, model-provider retention, or cross-server exfiltration. Existing redaction and telemetry minimization are strong and should remain. Client guidance should explicitly treat screen/result content as untrusted private data, disable retention where appropriate, and avoid forwarding it. Adding prompt-scanning logs or full request auditing would create a new sensitive dataset and is rejected.

**Operational Effect**

The safe operating posture is simple: protect stdio/bearer as full administrator authority; configure only intended devices, media roots, origins and wake targets; keep the server isolated from mutually distrusting users; require client-side confirmation for consequential calls; and stop on unknown mutation outcome. Reverse-proxy rate limiting may be useful for remote HTTP but is not an agent-intent control. No second control plane is required.

**Maintainer Effect**

Preserving a static tool manifest and honest trust model keeps review tractable. The recurring work is reviewing every tool/description/schema/result change as security-sensitive, checking upstream MCP/agent advisories, and maintaining a concise client/deployment threat table. An embedded model scanner, policy language, tool registry, multi-user scope system or behavioral SIEM would impose high ongoing tuning and incident burden without reliably solving client-side injection.

#### Decision

**Disposition**

retain

**Recommendation**

Retain the current one-principal, static tools-only architecture, schema and target bounds, privacy-minimized results, consequence-rich metadata, non-retryable unknown outcomes, and explicit non-goals. Before public release, elevate the existing threat-model conclusion into concise operator-facing guidance: any stdio owner or bearer holder, including a prompt-injected or compromised agent, has full configured JetKVM authority; screen/tool results are untrusted private data; annotations are hints; clients must bind approvals to server identity plus exact tool, require fresh human approval for destructive/high-impact calls, isolate cross-server data flows, and never automatically retry unknown mutations. Do not claim the server prevents prompt injection. Prefer deployment separation and configuration minimization over an in-product policy engine.

**Minimal Practical Change**

Add one compact, test-linked 'Agent and multi-server safety' section to public security/deployment documentation and release review criteria, derived from the maintained threat model. It should enumerate the concrete screen-to-mutation, result-to-other-server, malicious-client, shadowing and unsafe-retry paths; identify client versus server ownership; state that the bearer is full administrator authority; recommend server-qualified tool provenance and high-impact client approval; and prohibit claims that annotations, authentication or injection detection prove intent. Retain exact manifest review as the metadata-rug-pull gate. This is research guidance only; no repository file is modified here.

**Optional Stronger Control**

After real external adoption, publish tested client profiles that document whether a named/versioned host shows server identity, honors annotations, supports per-tool approval, isolates results and suppresses retries. If operators demonstrate a need to expose this server to less-trusted automation, consider a separate deliberately read-only or explicitly allowlisted deployment mode as a product decision, with static discovery that omits unavailable tools. Add coarse proxy rate limits for shared remote HTTP. These controls are justified by demonstrated deployments, not by framework scoring.

**Rejected Or Overengineered Alternatives**

- Reject an in-server LLM, prompt-injection classifier, regex scrubber or screenshot OCR filter: it cannot reliably distinguish instructions from data and would process/possibly retain highly sensitive content.
- Reject a bespoke OAuth/scopes/policy engine, per-user consent database or AI gateway for the current one-maintainer/one-principal product; deployment/client controls own user intent.
- Reject signing individual tool descriptions or creating a tool registry. Authenticate the released artifact and review its exact manifest instead.
- Reject sensitive full request/result audit logs, prompt transcripts, screenshots or typed-text retention; they create a high-value exfiltration store.
- Reject automatic replay, mutation idempotency guesses or server-side deduplication state for physical calls; an explicit repeated request may occur after world state changes.
- Reject adding sampling, prompts, resources, memory, tasks or a multi-agent approval framework as security machinery; each expands the injection and compatibility surface.
- Reject a blanket claim that all autonomous use is safe or unsafe. Safety depends on client authority, provenance, approvals, untrusted-data handling and deployment isolation.

**Rationale**

The repository is not an AI agent; it is a deterministic privileged tool server. Its strongest leverage is to minimize and accurately describe authority, validate structured inputs, constrain targets, minimize outputs, preserve ambiguous-outcome semantics and avoid optional agent surfaces. Empirical research shows that model-side injection defenses remain incomplete, while official MCP guidance assigns local-server consent/sandboxing and cross-tool authority to the client/deployment boundary. The code already implements most server-owned controls and honestly documents residual risk. Making the boundary prominent yields high value at low maintenance cost; pretending to solve semantic intent inside the server would be both misleading and disproportionate.

**Migration Or Rollout Considerations**

No runtime migration is required for the retained model. Put the full-administrator and untrusted-screen warnings before copy-paste client configuration, not only in the threat model. When publishing, give operators a short preflight: trusted client, no unreviewed co-server composition, protected bearer/stdio, intended devices only, safe media root/origins, approval for mutation classes, private-result retention understood, and retry disabled on unknown. If the deprecated multiplexed media tool is later removed, announce and fixture-test the compatibility change separately. Do not imply that documentation shifts responsibility away from server-owned validation and privacy controls.

**Priority**

P1 before public release for prominent agent/multi-server boundary guidance and release-review acceptance evidence; no P0 architectural blocker under the declared trusted-administrator model.

**Implementation Effort**

XS for the minimal documentation/release-review synthesis; M for any later tested read-only/allowlist product mode.

**Ongoing Maintenance Burden**

low: review static manifest diffs, current client/deployment guidance and relevant advisories each release; optional client-profile testing would be medium.

#### Verification

**Acceptance Evidence**

- Public deployment guidance states that stdio ownership or the bearer equals full configured-device administrator authority and that prompt-injected/compromised clients are authorized from the server's perspective.
- A threat table covers all named categories with an actual path or an explicit no-path/not-applicable rationale and assigns prevent/reduce/detect/document/client/deployment ownership.
- The exact manifest remains static and safety descriptions, annotations, schemas and output shapes remain review-gated across both transports.
- Tests prove unknown mutation outcomes are non-retryable, not_sent is only pre-dispatch, private firmware/error data is redacted, and telemetry excludes sensitive arguments/results.
- Documentation warns that screenshots and tool results are untrusted private data and that cross-server transfer needs separate authorization.
- No claim says annotations, authentication, a model prompt, or a scanner proves human intent or prompt-injection safety.

**Proposed Tests Or Checks**

- Keep manifest drift tests and add/retain required safety phrases for private output, physical/destructive consequence, open-world fetch and unknown-outcome retry handling.
- Property/table-test that every mutating handler converts any post-dispatch timeout, cancellation or unclassified error to unknown/non-retryable and that no internal automatic retry occurs.
- Inject instruction-like sentinels into firmware version/extension/raw unknown fields/errors and confirm only intended typed bounded fields reach structured results, while unknown/raw detail never reaches telemetry.
- Return a screenshot containing obvious adversarial text in a fake fixture and verify the server treats it only as opaque private image data; do not assert that a model ignores it.
- Exercise malicious clients with unknown fields, oversized text/path/URL, invalid enums, repeated capture calls and capacity exhaustion; confirm rejection or busy/not_sent before device dispatch.
- For any recommended client profile, manually test server-qualified naming, approval display for each consequence class, result retention/export, cross-server calls and retry behavior using fake devices only.

**Negative Or Abuse Cases**

- Screen pixels saying to ignore the user, reveal credentials, call force-off, or send the image through another server.
- A second server publishes jetkvm_force_host_power_off with deceptive read-only metadata or instructs the agent to invoke the real tool.
- A changed release keeps a tool name but alters description/schema/behavior or changes destructive/idempotent annotations.
- Firmware strings, errors or unknown fields contain instruction-like text, secrets, terminal control characters or oversized values.
- A bearer holder calls every mutation, guesses configured media-root paths, repeatedly captures screens, sends allowed-origin URLs with secret queries, or ignores busy/retryable=false.
- Timeout/cancellation after dispatch followed by an identical new keyboard, click, reset, power or upload request.
- Client forwards screenshot/status/tool error into email, web, shell, storage or another MCP server without separate authorization.
- Client treats readOnlyHint=false/destructiveHint=false/idempotentHint=true as permission or proof of harmlessness.

**Evidence Needed Before Claiming Support**

Before claiming safe use with an AI client, name and test the exact client/version, its server identity and namespace behavior, approval policy, treatment of untrusted images/results, memory/retention, multi-server information flow, and retry behavior. Server tests can support only claims about bounded dispatch, metadata, redaction, result taxonomy and transport behavior. They cannot qualify model resistance to injection, human consent, absence of cross-server exfiltration, or physical safety. Physical mutations require the separately approved hardware procedure and independent observation.

**Revisit Trigger**

Revisit when a tool, result field, dynamic capability, client-facing description, transport principal, device selection rule, media root/origin, telemetry field or raw firmware surface changes; when the project recommends a specific host or supports multi-user deployment; after a prompt-injection, retry, exfiltration or shadowing incident; on material MCP/OWASP/NIST guidance or advisory; or at least quarterly while agent security evidence is rapidly evolving, then each release once stable.

### 9. JetKVM, WebRTC, FFmpeg, HID, power, wake, upload, and virtual-media boundaries

#### Identity And Scope

**Item Name**

JetKVM, WebRTC, FFmpeg, HID, power, wake, upload, and virtual-media boundaries

**Research Question**

At the exact repository tree, are hostile JetKVM protocol data, WebRTC negotiation/media, FFmpeg decoding, local media, appliance-side URL fetches, uploads, HID, wake, and power actions bounded and represented honestly enough for public release, and what evidence is still required before claiming firmware or hardware compatibility?

**Scope**

The configured JetKVM HTTP login/cookie, WebSocket signaling, Pion ICE/SDP/DTLS/SRTP/data-channel/RTP path, H.264 assembly and FFmpeg child, confined local media and upload lifecycle, appliance-side URL fetching, HID translation/release, wake and power RPCs, and physical qualification boundary. Deployment transport authorization and general dependency/release policy are considered only where they control these paths.

**Repository Surfaces**

- internal/jetkvm/auth.go, provider.go, signaling.go, rpc_session.go, rpc_codec.go, video.go, video_receiver.go, capture.go, decoder_ffmpeg.go, hid.go, virtual_media.go, manager.go, errors.go
- internal/jetkvm/*_test.go including signaling/RPC/video/virtual-media fuzz tests, provider video integration, decoder real/synthetic tests, upload and admission tests
- internal/config/config.go and tests; internal/mcpserver/controls.go and error-result tests
- docs/protocol-sources.md, docs/compatibility/**, docs/product-contract.md, docs/threat-model.md, docs/adr/0002, 0003, 0004, and 0006, README.md
- go.mod/go.sum, Dockerfile, Makefile, .github/workflows/ci.yml, and .goreleaser.yaml

**Applicability Stage**

current_private, before_public_release, after_external_adoption

#### Sources And Authority

**Authoritative Sources**

- **Title:** jetkvm-mcp exact baseline source, tests, protocol ledger, and ADRs | **Publisher:** BenDManning/jetkvm-mcp | **Url:** https://github.com/BenDManning/jetkvm-mcp/commit/6e52f0027b13f928b768de0feeab4847ef9ca53e | **Publication Or Update Date:** 2026-08-15 | **Version Or Revision:** commit 6e52f0027b13f928b768de0feeab4847ef9ca53e; tree 34e8b4451d76821950c23d7c06958d021700f3a7 | **Access Date:** 2026-08-15 | **Supported Claims:** Executable protocol behavior, bounds, cleanup, tests, declared claims, and recorded upstream pin.
- **Title:** JetKVM KVM source repository | **Publisher:** JetKVM | **Url:** https://github.com/jetkvm/kvm | **Publication Or Update Date:** continuously updated; inspected repository pin recorded locally | **Version Or Revision:** locally recorded inspected commit b3c29a44d9e2862b8ff7530830781803ce27b060; drift comparison to fe77acd5f00300a4ab9acd5da57d7bb0916351d9 on 2026-08-14 | **Access Date:** 2026-08-15 | **Supported Claims:** Observed login, signaling, RPC method names/parameters, USB, wake, video, virtual-media state and resumable upload behavior.
- **Title:** Pion WebRTC releases | **Publisher:** Pion | **Url:** https://github.com/pion/webrtc/releases/tag/v4.2.18 | **Publication Or Update Date:** 2026-07-27 | **Version Or Revision:** v4.2.18 | **Access Date:** 2026-08-15 | **Supported Claims:** Current pinned Pion release and refreshed ICE/SCTP/interceptor dependencies; earlier v4.2.10 introduced RemoteIPFilter and v4.2.9 fixed an ICEGatherer deadlock.
- **Title:** Failed DTLS certificate verification does not stop data channel communication | **Publisher:** Pion / GitHub Security Advisory | **Url:** https://github.com/pion/webrtc/security/advisories/GHSA-74xm-qj29-cq8p | **Publication Or Update Date:** 2021-03-18 | **Version Or Revision:** GHSA-74xm-qj29-cq8p; fixed in v3.0.15 | **Access Date:** 2026-08-15 | **Supported Claims:** Known historical Pion certificate-verification flaw does not affect pinned v4.2.18.
- **Title:** FFmpeg Security | **Publisher:** FFmpeg Project | **Url:** https://ffmpeg.org/security.html | **Publication Or Update Date:** continuously updated | **Version Or Revision:** security tables current through FFmpeg 8.1.2 and 2026 CVEs | **Access Date:** 2026-08-15 | **Supported Claims:** FFmpeg remains a security-sensitive native-code decoder with version-specific fixes, including 2026 fixes; executable presence alone is not qualification.
- **Title:** FFmpeg Download | **Publisher:** FFmpeg Project | **Url:** https://ffmpeg.org/download.html | **Publication Or Update Date:** continuously updated | **Version Or Revision:** current download/support guidance | **Access Date:** 2026-08-15 | **Supported Claims:** Release branches receive selected backports while development receives fixes sooner; operators must track the actual supplied build.
- **Title:** Traversal-resistant file APIs | **Publisher:** The Go Project | **Url:** https://go.dev/blog/osroot | **Publication Or Update Date:** 2025-03-12 | **Version Or Revision:** Go 1.24 os.Root introduction | **Access Date:** 2026-08-15 | **Supported Claims:** os.Root prevents path and symlink escape on supported native platforms, with documented mount-point/platform limitations.
- **Title:** GO-2026-4970: os.Root improperly follows final symlink ending in slash | **Publisher:** Go Vulnerability Database | **Url:** https://pkg.go.dev/vuln/GO-2026-4970 | **Publication Or Update Date:** 2026-07-07 | **Version Or Revision:** CVE-2026-39822; fixed in Go 1.25.12 and 1.26.5 | **Access Date:** 2026-08-15 | **Supported Claims:** Minimum Go 1.25 without a patch floor can compile an os.Root confinement path with a known escape defect.
- **Title:** Server Side Request Forgery Prevention Cheat Sheet | **Publisher:** OWASP Foundation | **Url:** https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html | **Publication Or Update Date:** continuously updated | **Version Or Revision:** accessed 2026-08-15 | **Access Date:** 2026-08-15 | **Supported Claims:** Exact destination allowlisting is preferable, but DNS, redirects, and address changes require enforcement at the component performing the fetch.

**Source Class**

- **Repository Baseline:** repository_evidence
- **Jetkvm Source:** official_documentation
- **Pion Release:** official_documentation
- **Pion Ghsa And Go Vulnerability:** official_advisory
- **Ffmpeg Security And Download:** official_documentation
- **Go Osroot:** official_documentation
- **Owasp Ssrf:** security_framework

**Normative Status**

- Repository code and tests are observation of implemented behavior; JetKVM upstream source is observation of an undocumented browser-to-device protocol, not a vendor compatibility promise.
- The Go vulnerability version ranges are official advisory facts; builds using os.Root should use Go 1.25.12 or Go 1.26.5 and later.
- FFmpeg security tables and Pion advisories are official project evidence, not a guarantee that unlisted versions are vulnerability-free.
- OWASP destination allowlisting and egress restriction are guidance; because JetKVM firmware performs URL fetches, network enforcement belongs at the appliance boundary.

**Freshness And Supersession**

Pion v4.2.18 was the latest release and the repository's exact pin on 2026-08-15. It includes fixes and dependency updates after v4.2.9-v4.2.17. GHSA-74xm-qj29-cq8p is historical and fixed long before v4. FFmpeg's live security table lists 2026 fixes, superseding any assumption that distro package presence is enough. GO-2026-4970 was published after the repository's broad Go 1.25 minimum claim; Go 1.25.12 and 1.26.5 supersede affected patch versions. JetKVM protocol evidence is explicitly pinned and already known to have drifted on every reviewed upstream surface, so it must not be described as current vendor support without re-review and hardware tests.

**Source Disagreements**

- README/product-contract say Go 1.25 or newer, while GO-2026-4970 makes pre-1.25.12 unsafe for this exact os.Root confinement use. Resolve by keeping language minimum 1.25 but requiring patch floor 1.25.12 (or 1.26.5+) for builds/releases.
- Official OWASP guidance favors controlling redirect/DNS/address behavior, but this process cannot observe firmware-side fetch behavior. Resolve honestly with exact-origin preauthorization plus appliance egress controls, not application-side DNS checks that the firmware can bypass.
- The product supports conventional local JetKVM operation based on observed source, while upstream provides no stable API contract and recorded drift exists. Resolve by claiming protocol evidence and tested fixtures only, not firmware-version compatibility.

#### Repository Evidence

**Exact Baseline Commit**

The stipulated source revision is 6e52f0027b13f928b768de0feeab4847ef9ca53e, tree 34e8b4451d76821950c23d7c06958d021700f3a7. Live HEAD was 71445bc6bf2325e6c683e362393605089c336b63 with the identical tree and empty source diff. Pre-existing untracked .agents/, jetkvm_mcp_2026_public_release_baseline/, and skills-lock.json were not treated as product evidence.

**Current Repository Evidence**

- provider.go creates a fresh cookie jar, HTTP client, Pion PeerConnection, RPC data channel, signaling socket, and optional video receiver per operation; redirects are rejected, proxy inheritance is disabled, TLS minimum is 1.2 unless explicit insecure_skip_verify, and teardown cancels state, closes RPC/signaling/peer/idle HTTP connections, and waits for pump goroutines up to the caller context.
- signaling.go limits WebSocket frames and decoded SDP to 64 KiB, admits answer only once, queues at most 32 pre-answer ICE candidates, accepts only non-empty candidates, and uses a recvonly H.264 profile with four fixed packetization-mode=1 codecs. It enables local loopback candidates and has no remote ICE IP filter.
- rpc_session.go serializes sends, limits pending calls to 64, applies per-request timeouts, removes cancelled waiters, and labels post-send timeout/close as unknown outcome.
- video.go/video_receiver.go bound H.264 assembly, RTP packet/state growth, one capture waiter, fresh-frame watermark, codec/clock/packetization checks, and PLI behavior; parser fuzz corpora exist.
- decoder_ffmpeg.go resolves FFmpeg once to an absolute regular executable path, never invokes a shell, uses fixed arguments and scrubbed LANG/LC_ALL/PATH, pipe-only protocol whitelist, 4 MiB H.264 input, 32 MiB PNG output, 16 KiB discarded stderr, allocation/pixel/thread/frame/probe limits, a 15-second context, WaitDelay, and full PNG decode/dimension validation. It is hardening, not an OS sandbox.
- Dockerfile installs Debian bookworm ffmpeg without recording the package version; native releases merely require ffmpeg on PATH. No FFmpeg implementation/version matrix or provenance is qualified.
- virtual_media.go uses os.OpenRoot, lexical traversal checks, regular non-empty file requirement, 64 GiB maximum, basename-only appliance names, per-device serialization, before/during/after SHA-256 checks, identity/size/modtime reopens, offset-zero uploads, strict upload IDs, bounded authenticated HTTP bodies/responses, and two-second detached best-effort cleanup. It does not cross bind-mount boundaries and cannot prove firmware deletion.
- parseAllowedMediaURL is deny-by-default and matches exact normalized HTTP(S) scheme/host/effective port; it rejects credentials, opacity, missing host, non-HTTP(S), wildcards in configuration, and dispatch before opening a device session. It deliberately cannot inspect firmware DNS, redirects, or final address.
- hid.go maps bounded US-ASCII/named keys and bounded mouse operations to fixed RPC methods, sends explicit release reports, and attempts a detached two-second release after partial failure. A lost reply remains unknown and a best-effort release is not host-state proof.
- Power and wake methods are fixed, selected from configured devices/targets, serialize per device, and return conservative unknown outcomes after possible dispatch. Wake-on-LAN callers cannot supply arbitrary MAC or broadcast destinations.
- docs/protocol-sources.md pins JetKVM b3c29a44... and records 2026-08-14 drift to fe77acd5... on every reviewed surface. Product contract and validator explicitly distinguish fake/protocol/build checks from physical qualification.
- go.mod pins github.com/pion/webrtc/v4 v4.2.18. Docker build uses Go 1.26.5, the first fixed 1.26 release for GO-2026-4970, but documented minimum Go 1.25 lacks required patch 1.25.12.

**Implementation And Documentation Agreement**

Code, tests, ADRs, README, product contract, and threat model agree closely on fresh-session ownership, hostile-input bounds, FFmpeg non-sandbox status, upload integrity/cleanup, URL-fetch residual risk, and no physical qualification. Two claims need correction or sharper release enforcement: the bare Go 1.25 minimum ignores a directly applicable os.Root advisory, and container FFmpeg availability is sometimes easier to read as support than the contract's accurate statement that no version is qualified. The threat model correctly documents absent ICE candidate policy and firmware-side URL enforcement.

**Current State**

substantially_satisfied: repository-level boundary design is strong and unusually explicit, with good bounds, cleanup, error ambiguity, path confinement, upload integrity, and fake/fuzz evidence. Public-release gaps are patch-level toolchain enforcement, FFmpeg version/provenance policy, adjudication of recorded JetKVM drift, and a qualified decision on remote ICE candidate reach. Physical compatibility remains unknown by design.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- A compromised or malicious configured appliance returns attacker-crafted SDP/ICE candidates; Pion parses them and may send connectivity checks to addresses reachable from the server, because no RemoteIPFilter restricts remote candidates.
- A hostile H.264 access unit exploits a vulnerable system FFmpeg decoder. Fixed pipes and limits constrain I/O and duration but the child shares the server identity and readable filesystem; the Docker package is not version-qualified.
- A build made with Go 1.25.0-1.25.11 opens a trailing-slash final symlink through os.Root under CVE-2026-39822, potentially escaping the configured media root before the regular-file check.
- A local file is replaced during upload; without the implemented pre/upload/post hashes and identity checks, stale or attacker-selected bytes could be persisted or mounted. Current checks detect common replacement, but privileged bind-mount changes remain outside os.Root.
- Firmware reports a stale resumable offset with no prefix digest. Blind resumption would splice old appliance bytes with a new local file; current code deletes the partial and demands offset zero.
- A URL matches the configured origin but firmware follows a redirect or resolves/re-resolves DNS to an unintended private service. This process cannot see the final destination; exact origin alone is not complete SSRF prevention.
- A keyboard press or mouse-down RPC reaches the host but its release/reply is lost. The server attempts release but cannot prove host HID state; retry may duplicate commands or clicks.
- Power, reset, wake, mount, or upload is delivered but acknowledgement is lost. Reporting failure as not-sent or automatically retrying could duplicate physical/network effects; the current unknown/non-retryable classification avoids that claim.
- Upstream JetKVM changes method shapes or session behavior after the inspected commit. Fake fixtures continue passing while real firmware becomes incompatible.
- Two local media paths with the same basename address the same appliance object; cleanup after an ambiguous operation can delete a pre-existing object with that basename.

**Affected Assets And Trust Boundaries**

Assets include JetKVM credentials/cookies, server network reach, attached-host display and integrity, physical power/availability, firmware storage, local media bytes, URL-origin services, process filesystem access, and compatibility claims. Boundaries are operator configuration to process; process HTTP/WebSocket/WebRTC to configured appliance; untrusted SDP/ICE/RTP/RPC back into Go/Pion; H.264 into native FFmpeg; media root into authenticated upload; appliance into media-origin network; and appliance HID/power outputs into the physical host.

**Plausible Impact**

Impacts range from process compromise through a decoder flaw, local file disclosure through a confinement regression, server-side network probing through candidates, and appliance-side SSRF, to arbitrary console input, data loss, reboot/power interruption, stale or malicious boot media, persistent firmware storage, and false compatibility assurance. Most high-impact paths require an authorized caller, compromised configured appliance, writable media root, vulnerable decoder/toolchain, or permissive appliance network.

**Existing Controls**

- Fresh single-operation sessions, timeouts, cancellation, pending-call bounds, transport cleanup, direct HTTP without proxy/redirect, and conservative outcome taxonomy.
- Bounded signaling, single answer, candidate queue, codec allowlist, recvonly video, bounded RTP/H.264 assembly, parser fuzzing, and Pion v4.2.18.
- Fixed non-shell FFmpeg invocation, scrubbed environment, pipe-only protocols, input/output/stderr/allocation/pixel/thread/frame/time limits, and complete PNG validation.
- os.Root confinement, regular-file and 64 GiB checks, content/identity verification, no unauthenticated resume, authenticated bounded upload, per-device serialization, and best-effort cleanup.
- Exact deny-by-default media origins and explicit documentation that network egress is the final firmware-fetch control.
- Fixed HID/power/wake methods, configured-only WOL targets, release reports, and unknown/non-retryable post-dispatch failures.
- Protocol-source pin, drift manifest, ADRs, fake-provider tests, fuzz smoke, race tests, container non-root identity, and explicit non-qualification language.

**Residual Risk**

Pion and FFmpeg share the process account's network/filesystem failure domain; FFmpeg is not sandboxed and remote ICE destinations are unrestricted. The configured device can lie about state and acknowledgements. Exact URL origins cannot govern firmware redirects/DNS/routing. Cleanup cannot prove remote deletion or undo physical mutations. os.Root does not block privileged bind mounts and affected Go patch versions are unsafe. Fake tests cannot establish firmware or hardware support.

**Compatibility Or Semver Effect**

Raising the build patch floor does not change the MCP API but changes supported build environments and should be documented as a security floor. Adding a restrictive ICE policy can break multi-homed, IPv6, NAT, or unusual firmware candidates and requires hardware evidence before defaulting. Tightening FFmpeg versions affects native/container support claims. JetKVM method changes may require a major release only when they alter public tool inputs/results/consequences; a provider-only compatibility fix can remain patch/minor if behavior is preserved.

**Privacy Effect**

Screen frames, typed text, media bytes/paths/URLs, session cookies, SDP/candidates, and device state remain transient private data. Current application code avoids temp screenshots and raw error bodies, but FFmpeg receives screen bytes and can access files readable by the service identity. URL query/fragment values are forwarded to firmware and may leak through origin or appliance logs; documentation should continue to prohibit secrets there. Qualification artifacts must be sanitized and should not retain raw frames, SDP, or media.

**Operational Effect**

Fresh sessions and triple-read upload verification trade latency/I/O for simple cleanup and isolation; retain until measured objectives justify change. Requiring patched Go and recording FFmpeg versions adds release checks but no runtime service. Appliance egress rules and process limits remain deployment tasks. Candidate filtering would require a qualification matrix and troubleshooting guidance, so monitoring/experimentation should precede enforcement.

**Maintainer Effect**

The current architecture is locally comprehensible and avoids pools, daemons, and resume state. Recurring work should be limited to upstream JetKVM surface review, Pion/FFmpeg/Go advisory monitoring, one small firmware qualification matrix, and preservation of boundary tests. A decoder sidecar, generalized SSRF engine, or device abstraction framework would impose disproportionate expertise and lifecycle cost.

#### Decision

**Disposition**

change

**Recommendation**

- Retain fresh in-process Pion sessions, fixed profile/codecs, bounded parsers, FFmpeg-per-capture, os.Root media confinement, offset-zero upload integrity, exact-origin URL authorization, fixed HID/power/WOL methods, and non-retryable unknown outcomes.
- Before public release, enforce and document Go 1.25.12 as the minimum 1.25 patch (or 1.26.5+) because this repository calls vulnerable os.Root.Open. Record the exact FFmpeg package/version in container verification and state a maintained/patched FFmpeg requirement for native installs; do not claim a broad compatible version range yet.
- Review recorded JetKVM upstream drift and run a named firmware/hardware qualification plan before claiming device compatibility. Publish the tested versions and label all other versions unqualified, not unsupported.
- Experiment with Pion RemoteIPFilter or equivalent candidate admission against representative devices. If normal devices only require addresses tied to the configured endpoint/network policy, add the narrowest compatible filter; otherwise document the residual and rely on dedicated-process network egress controls.
- Keep URL origin matching as an application authorization gate but require appliance network segmentation/egress allowlisting for SSRF control. Do not pretend process-side DNS resolution controls firmware resolution or redirects.
- Treat FFmpeg sandboxing as an optional stronger deployment control triggered by threat/adoption, not a mandatory second service for the initial one-maintainer release.

**Minimal Practical Change**

Set a tested Go patch floor of 1.25.12/1.26.5, add a release/container check that records FFmpeg version and rejects a known-vulnerable floor chosen by the maintainer, review the existing JetKVM drift ledger, and execute a minimal named-device qualification covering status, capture, HID release, power/wake acknowledgement ambiguity, URL media, offset-zero upload/mount, cancellation, and cleanup. Run a focused candidate capture/filter experiment and record the decision.

**Optional Stronger Control**

For higher-risk deployments, run the server/FFmpeg under an OS sandbox or container with read-only filesystem, no-new-privileges/seccomp, tight CPU/memory/PID limits, read-only media mount, and egress limited to configured appliances/origins; consider an FFmpeg helper only if an accepted isolation requirement defines IPC and lifecycle. Add Pion RemoteIPFilter when device evidence proves a safe policy. These become justified after outside adoption, exposure to untrusted appliances/media, or an applicable incident.

**Rejected Or Overengineered Alternatives**

- Do not pool WebRTC sessions; cross-call cookies, pending RPCs, media, reconnect, and cancellation state are not justified by measured latency.
- Do not resume firmware uploads from byte count without a cryptographically bound prefix or immutable content identifier.
- Do not add a general URL policy engine, application DNS pinning, redirect simulator, or IP taxonomy as proof of firmware-fetch safety; enforce egress where the appliance fetch occurs.
- Do not replace FFmpeg with an unqualified pure-Go decoder, keep a decoder daemon, or mandate a sidecar without corpus, performance, and isolation acceptance evidence.
- Do not expose arbitrary HID/RPC/WOL destinations or derive firmware support from upstream source, fake fixtures, cross-builds, or one successful connection.
- Do not promise that cleanup reverses a possibly delivered mutation or deletes firmware artifacts after transport loss.

**Rationale**

The repository has already implemented the controls most directly tied to real failure paths: bounded untrusted inputs, one-operation ownership, fixed child invocation, traversal-resistant opens, content-bound uploads, exact origins, explicit releases, and ambiguity-preserving errors. The remaining immediate corrections are evidence/floor problems, not an architectural rewrite. GO-2026-4970 directly affects the media boundary; FFmpeg's active 2026 advisory stream makes unversioned availability insufficient; and recorded JetKVM drift precludes broad compatibility claims. ICE filtering is potentially valuable but can break legitimate device networking, so hardware evidence must precede a default policy.

**Dependencies Or Prerequisites**

- Maintainer selection of supported Go patch lines and an FFmpeg vulnerability/version floor compatible with Debian and native packaging.
- Access to at least one representative JetKVM, firmware version inventory, disposable attached host, isolated network, and harmless media fixtures for qualification.
- Review of docs/compatibility drift from b3c29a44... to a chosen current upstream commit.
- Network capture/test environment for candidate policy and firmware URL redirect/DNS behavior.
- Coordination with release/container and vulnerability-management workstreams for recurring advisory checks.

**Migration Or Rollout Considerations**

Raise build floors in documentation, CI, and release metadata together; do not silently reject otherwise supported source users. Record container FFmpeg version per image digest and provide a verification command. Introduce any ICE filter behind qualification with explicit allowed topology and a rollback path. Publish device qualification as a dated matrix naming firmware/hardware and operation evidence, while retaining fake tests as protocol regression gates. Existing exact-origin configurations need no migration; deployment guidance must emphasize egress enforcement.

**Priority**

P1 before public release for patched Go floor, FFmpeg version evidence, and honest device qualification; P2 for ICE candidate-policy experiment and stronger decoder isolation guidance.

**Implementation Effort**

M: code/documentation checks are S, but meaningful device, candidate, media, power, and cancellation qualification is M and requires hardware access.

**Ongoing Maintenance Burden**

medium: quarterly/advisory-triggered Go/Pion/FFmpeg review, release-time version recording, and firmware qualification on meaningful upstream changes; no continuous governance system is needed.

**Confidence**

high for repository controls and the Go/FFmpeg/Pion source facts; medium for physical, firmware, and candidate-policy conclusions because those require unavailable hardware/network evidence.

#### Verification

**Acceptance Evidence**

- Build/release checks use Go 1.25.12 or 1.26.5+ and a negative test demonstrates the trailing-slash final-symlink case cannot escape the media root.
- Every released container records its FFmpeg version/build source, uses a version not affected by then-known applicable advisories, and passes real decode plus malformed/bounded input tests.
- A dated qualification report names JetKVM hardware and firmware and separates each tested operation, cancellation/timeout, unknown outcome, cleanup observation, and soak count.
- Protocol drift from the inspected JetKVM source pin is adjudicated surface by surface; fixture updates cite upstream evidence and are not treated as device proof.
- A captured candidate corpus and test show either an enforced compatible RemoteIPFilter or an explicit residual-risk decision plus deployment egress rule.
- URL-fetch tests document exact-origin rejection before dispatch and physical/network tests document firmware redirect/DNS behavior without claiming application enforcement.
- go test ./internal/jetkvm passes; it passed on 2026-08-15 at the studied tree.

**Proposed Tests Or Checks**

- Add the exact CVE-2026-39822 trailing-slash final-symlink case and retain traversal, intermediate/final symlink, replacement, non-regular, size, basename-collision, and bind-mount limitation reviews.
- Run connect/cancel/close loops under race and leak observation with signaling failure before/after answer, 33+ queued candidates, malformed SDP, invalid codec/track, stalled RTP, and pending RPC saturation.
- Fuzz and corpus-test SDP envelopes, ICE candidates, RPC responses, RTP/H.264 assembly, and firmware media state while enforcing memory/time ceilings.
- Decode valid and malformed H.264 with the exact release FFmpeg build; assert fixed argv/environment, no file/network protocols, one frame, pixel/allocation/thread limits, stdout/stderr bounds, timeout kill, and no temporary images.
- On disposable hardware, interrupt upload at negotiation/body/response/verification/mount phases; inspect .incomplete/completed artifacts and mount state, and never infer deletion from a lost response.
- Test allowed-origin DNS changes and same-origin redirects in an isolated appliance network, confirming egress controls—not this process—block unintended destinations.
- Exercise every HID press/release and mouse-down/release cancellation point, WOL configured-target restriction, and power/reset operations only under an explicit physical qualification plan.

**Negative Or Abuse Cases**

- Oversized/duplicate answers, malformed base64/SDP, unsupported codecs, binary signaling frames, candidate flood, loopback/link-local/multicast/public candidate destinations, stalled ICE, and peer failure during close.
- RTP sequence gaps, FU-A/STAP-A abuse, oversized access units, missing keyframe, decompression bombs, malformed PNG, FFmpeg hang/crash/output flood, and hostile stderr.
- Absolute/traversal/reserved paths, final/intermediate symlinks, trailing slash, bind mounts, FIFO/socket/device, empty/over-64-GiB file, in-place mutation, atomic replacement, same basename, and cancellation during all three reads.
- Invalid upload IDs, nonzero resume offsets, short/long body, HTTP error with private body, connection loss after dispatch, cleanup timeout, stale partial, failed deletion, and reboot persistence.
- URL credentials, opaque/non-HTTP schemes, wildcard/mismatched/default/non-default ports, IDN/case variants, redirects, rebinding, private/link-local/metadata targets, and query secrets.
- Lost acknowledgement after key-down, mouse-down, power/reset, WOL, mount, unmount, or upload; confirm error remains unknown/non-retryable and no automatic retry occurs.

**Evidence Needed Before Claiming Support**

Do not claim a JetKVM firmware/hardware version until the named version passes physical qualification for each claimed operation and a soak/cancellation sample. Do not claim FFmpeg compatibility from executable discovery; name the tested upstream/distro builds and architectures. Do not claim URL SSRF prevention beyond exact-origin authorization without appliance-side redirect/DNS tests and enforced egress. Do not claim process isolation for FFmpeg or Pion. Cross-compilation, fake WebRTC peers, fuzzing, and upstream source observation prove only their specific build/protocol properties.

**Revisit Trigger**

Any JetKVM firmware/source drift on reviewed surfaces; new Pion/ICE/SCTP/DTLS/SRTP, FFmpeg/H.264/PNG, Go os.Root, or coder/websocket advisory; change to FFmpeg/container base; support for remote/cloud/TURN or additional codecs; URL-fetch incident; media-root escape; duplicate physical mutation; device compatibility report; accepted performance objective; or annual public-release review.

### 10. Privacy, logging, diagnostics, telemetry, and incident evidence

#### Identity And Scope

**Item Name**

Privacy, logging, diagnostics, telemetry, and incident evidence

**Research Question**

Does the studied jetkvm-mcp tree minimize sensitive data while retaining enough bounded, trustworthy evidence to diagnose failures and reconstruct consequential device operations, and what small changes are required before public release without creating a sensitive request log or a second operational control plane?

**Scope**

Covers credentials, cookies, bearer tokens, typed text and HID input, screenshots, device and host state, identifiers, URLs, paths, media, RPC bodies, errors, stderr, MCP results, telemetry, validation and qualification reports, CI artifacts, retention, cardinality, and incident evidence across the MCP-client/process, process/JetKVM, process/FFmpeg, process/filesystem, process/supervisor, and repository/GitHub boundaries. It excludes a general SIEM, per-user audit authorization, legal-retention advice, and physical-device testing.

**Repository Surfaces**

Studied docs/telemetry.md; docs/threat-model.md; docs/ci-quality.md; docs/mutation-validation.md; docs/device-compatibility.md; internal/telemetry/recorder.go and recorder_test.go; internal/mcpserver/server.go, errors.go, server_test.go, tools_*.go and tests; internal/jetkvm/auth.go, client.go, rpc.go, media.go, decoder_ffmpeg.go and tests; internal/config/config.go and tests; cmd/jetkvm-mcp/main.go and tests; cmd/jetkvm-mcp-validate/main.go and tests; cmd/jetkvm-mcp-mutation-checklist/main.go and tests; .github/workflows/ci.yml; scripts/check_mcp_protocol.sh and tests; tests/mcp-conformance/**; config.example.yaml; README.md.

**Applicability Stage**

current_private, before_public_release

#### Sources And Authority

**Authoritative Sources**

- **Title:** Studied jetkvm-mcp repository tree | **Publisher:** BenDManning/jetkvm-mcp | **Url:** https://github.com/BenDManning/jetkvm-mcp/tree/6e52f0027b13f928b768de0feeab4847ef9ca53e | **Publication Or Update Date:** 2026-08-15 commit snapshot | **Version Or Revision:** 6e52f0027b13f928b768de0feeab4847ef9ca53e; identical tracked tree at 34e8b4451d76821950c23d7c06958d021700f3a7 and later 71445bc6bf2325e6c683e362393605089c336b63 | **Access Date:** 2026-08-15 | **Supported Claims:** Executable data handling, emitted fields, queue behavior, sanitizer tests, documentation claims, and CI artifact contents.
- **Title:** Logging Cheat Sheet | **Publisher:** OWASP Foundation | **Url:** https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html | **Publication Or Update Date:** continuously maintained; page accessed 2026-08-15 | **Version Or Revision:** current web revision on access date | **Access Date:** 2026-08-15 | **Supported Claims:** Log when/where/who/what and interaction identifiers; exclude or mask passwords, access tokens, session identifiers, keys and sensitive personal data; protect and dispose of logs; logging failures must not cause application denial of service.
- **Title:** NIST Privacy Framework: A Tool for Improving Privacy through Enterprise Risk Management, Version 1.0 | **Publisher:** National Institute of Standards and Technology | **Url:** https://www.nist.gov/privacy-framework/privacy-framework | **Publication Or Update Date:** 2020-01-16 | **Version Or Revision:** 1.0 | **Access Date:** 2026-08-15 | **Supported Claims:** Data minimization, disassociability, and documented review of logging, retention, and disposal are privacy-risk controls.
- **Title:** Signals | **Publisher:** OpenTelemetry | **Url:** https://opentelemetry.io/docs/concepts/signals/ | **Publication Or Update Date:** 2026-03-10 | **Version Or Revision:** current documentation | **Access Date:** 2026-08-15 | **Supported Claims:** Logs, metrics, and traces are distinct observability signals; adopting all three is not a requirement.
- **Title:** Metrics | **Publisher:** OpenTelemetry | **Url:** https://opentelemetry.io/docs/concepts/signals/metrics/ | **Publication Or Update Date:** current page accessed 2026-08-15 | **Version Or Revision:** current documentation | **Access Date:** 2026-08-15 | **Supported Claims:** Metric cardinality is the number of unique attribute combinations and high-cardinality values such as raw URLs or user identifiers increase cost and memory pressure.
- **Title:** Configuring the retention period for GitHub Actions artifacts and logs in your organization | **Publisher:** GitHub | **Url:** https://docs.github.com/en/organizations/managing-organization-settings/configuring-the-retention-period-for-github-actions-artifacts-and-logs-in-your-organization | **Publication Or Update Date:** current page accessed 2026-08-15 | **Version Or Revision:** GitHub Actions documentation | **Access Date:** 2026-08-15 | **Supported Claims:** The default retention is 90 days; configurable ranges differ for public and private repositories.
- **Title:** Storing and sharing data from a workflow | **Publisher:** GitHub | **Url:** https://docs.github.com/en/actions/tutorials/store-and-share-data | **Publication Or Update Date:** current page accessed 2026-08-15 | **Version Or Revision:** GitHub Actions documentation | **Access Date:** 2026-08-15 | **Supported Claims:** upload-artifact supports an explicit retention-days setting within repository or organization limits.
- **Title:** MCP stdio transport specification | **Publisher:** Model Context Protocol | **Url:** https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio | **Publication Or Update Date:** 2026-07-28 | **Version Or Revision:** 2026-07-28 | **Access Date:** 2026-08-15 | **Supported Claims:** A stdio server must reserve stdout for protocol messages and may write UTF-8 logging to stderr.

**Source Class**

Repository snapshot: repository_evidence. MCP transport: normative_specification. GitHub and OpenTelemetry pages: official_documentation. OWASP Logging Cheat Sheet and NIST Privacy Framework: security_framework. No secondary commentary is relied upon.

**Normative Status**

MCP stdout reservation is MUST and stderr logging is MAY. Repository code is observation of actual behavior. OWASP and NIST propositions are guidance, not product or protocol requirements. OpenTelemetry cardinality and signal descriptions and GitHub retention behavior are official-documentation observations. The proposed retention periods and incident-record design are repository-specific judgments, not external mandates.

**Freshness And Supersession**

The repository evidence is pinned to the 2026-08-15 studied tree. MCP evidence uses the requested 2026-07-28 revision. OpenTelemetry material was current in 2026. OWASP is a maintained page and NIST Privacy Framework 1.0 is older canonical guidance; neither overrides repository-specific proportionality. GitHub feature details were checked on 2026-08-15 and can change, so they require release-time revalidation.

**Source Disagreements**

OWASP's generic recommendation to capture when, where, who, and what is intentionally narrowed here: this one-principal server should capture time, process/version, operation class, stage, outcome and correlation, but not device alias, client identity, arguments, URLs, paths, typed input, screenshots, or raw results. Those identifiers would improve attribution while materially expanding collection and linkability. The resolution is minimal pseudonymous operational evidence plus deployment-owned access logs when source attribution is genuinely needed. Likewise, OWASP's desire for reliable security logging is balanced against availability: logging must remain non-blocking, and loss must be disclosed rather than making a physical mutation fail after delivery.

#### Repository Evidence

**Exact Baseline Commit**

The exact initial studied commit is 6e52f0027b13f928b768de0feeab4847ef9ca53e, not the mission's expected 176ec421f9ee6c801517180e1ad0ec9c84570e8e. Its tracked tree is identical to 34e8b4451d76821950c23d7c06958d021700f3a7 and the later current commit 71445bc6bf2325e6c683e362393605089c336b63. Pre-existing untracked .agents/, skills-lock.json, and jetkvm_mcp_2026_public_release_baseline/ research artifacts were outside the studied product tree and were not treated as product evidence.

**Current Repository Evidence**

docs/telemetry.md defines jetkvm.operation.v1 as eight closed fields: schema, correlation_id, transport, operation, stage, duration_ms, code, and outcome, with explicit prohibitions on arguments, results, aliases, addresses, URLs, paths, typed text, images, firmware, raw RPC, credentials, raw errors, HTTP bodies, and subprocess details. internal/telemetry/recorder.go implements 96-bit random correlation IDs with a process-local fallback, closed-enum validation, JSON encoding, separate bounded 256-entry event and terminal queues, one writer, non-blocking drops, and bounded close; recorder_test.go exercises schema, privacy sentinels, concurrency, slow/broken writers, terminal reservation, and bounded goroutine/memory behavior. internal/mcpserver/server.go instruments known tools/call operations and stages; errors.go returns stable sanitized typed errors and discards raw causes. docs/threat-model.md and matching tests document that credentials/cookies stay in process, typed HID input is transient, screenshots return only to the caller and are not written to disk, paths and device identifiers are omitted from evidence, and debug RPC is a deliberate local raw-result escape hatch. internal/config rejects inline secrets and unsafe device URL components. cmd/jetkvm-mcp-validate emits only check/result metadata and bounded image dimensions/size, discards child stderr, fully decodes then clears PNG bytes; its tests use privacy sentinels. The mutation checklist emits no target identity or captured content. scripts/check_mcp_protocol.sh uses a private temporary directory and uploads only sanitized pins, summaries, and scenario/status artifacts; tests enforce this boundary. .github/workflows/ci.yml uploads coverage and sanitized protocol artifacts without explicit retention-days. Telemetry has no wall-clock timestamp, process-instance/version record, or loss counter, and application code does not own stderr retention. The compatibility ledger intentionally omits serials, aliases and addresses but also lacks retained exact firmware/source evidence for its historical run.

**Implementation And Documentation Agreement**

Implementation and tests substantially match the documented minimization contract: telemetry uses closed enums, typed results/errors redact raw device data, stdout remains protocol-only, validators and protocol artifacts are sanitized, and no request-history store exists. Documentation candidly states that telemetry can be lost and that client, proxy, shell-history, core-dump, appliance, and stderr retention are deployment responsibilities. The important shortfall is not a contradiction but an evidence gap: the documented telemetry schema cannot independently place an event in wall-clock time or prove completeness, and there is no live evidence that deployments apply bounded retention. The raw debug_rpc tool agrees with documentation but remains an intentional privacy exception whose output is controlled by the MCP caller, not telemetry.

**Current State**

substantially_satisfied: sensitive-data minimization and bounded non-blocking telemetry are unusually strong and test-backed for a small server. Before public release, incident usefulness and retention are only partial because events lack wall-clock/process context, silent loss cannot be quantified, and CI/runtime retention is implicit.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- A user redirects stderr to a file without supervisor timestamps; an unknown-outcome power or HID operation can be correlated inside the process but cannot be placed reliably on an incident timeline because jetkvm.operation.v1 has no wall-clock time.
- A slow, full, or broken stderr sink fills both bounded queues; an operation still executes, but telemetry is silently dropped. An investigator could incorrectly treat absence of an event as proof that no mutation occurred.
- CI retains coverage or sanitized protocol artifacts for the implicit platform default longer than the maintainer expects; future changes could accidentally add sensitive content to an artifact whose retention was never consciously chosen.
- A maintainer adds aliases, paths, raw URLs, RPC parameters, typed text, screenshot bytes, firmware responses, bearer values, or raw errors to make diagnosis easier; logs and CI artifacts then become a durable secret and private-data store.
- A trusted caller invokes debug_rpc with sensitive parameters or obtains a sensitive raw result; the MCP response, client logs, terminal transcript or shell history retains it even though server telemetry does not.
- A failure is investigated using only coarse telemetry; without exact binary version, device model/firmware, client-visible structured result and physical observation, the maintainer overclaims compatibility or cannot distinguish not_sent from unknown physical outcome.
- An observability exporter turns correlation IDs, device identifiers, URLs or paths into metric labels, causing unbounded cardinality and increased privacy exposure.

**Affected Assets And Trust Boundaries**

Assets include JetKVM passwords and session cookies, MCP HTTP bearer tokens, typed text and key presses, screenshots, host power/video/USB state, device model and firmware, media contents and mount URLs, local paths, RPC parameters/results, and evidence about consequential physical mutations. Boundaries are MCP client to server, server to JetKVM HTTP/WebSocket/WebRTC, server to FFmpeg/subprocess, server to local filesystem, process to stderr supervisor, workflow to GitHub artifact storage, and maintainer to incident evidence. Raw debug RPC deliberately crosses server-to-client without typed redaction.

**Plausible Impact**

Sensitive logging could expose credentials, typed secrets, screen contents, internal URLs, media, host state, or device identity long after execution. Insufficient evidence can delay incident containment, obscure an ambiguous physical mutation, produce false assurance from missing events, or cause unsupported device claims. Unbounded or blocking observability could impair availability during a consequential operation.

**Likelihood Or Preconditions**

Payload exposure requires an authorized or compromised MCP caller, a future logging regression, unsafe client/proxy/supervisor configuration, artifact expansion, or deliberate debug_rpc use. Silent telemetry loss requires sink backpressure/failure or sustained event volume; tests prove this is possible by design, although normal low-volume single-process operation lowers likelihood. Timeline ambiguity occurs whenever stderr is retained without external timestamps. CI over-retention is plausible because no workflow-level duration is set. Physical impact still requires access to a configured device and operation authorization.

**Existing Controls**

Strict configuration rejects inline secret fields and unsafe config diagnostics; secrets are environment-referenced and response bodies are bounded. Typed tools emit bounded structured results and stable sanitized errors. Telemetry is closed-enum, payload-free JSONL on stderr, uses opaque correlation, caps durations, serializes atomically, never blocks operations, and has privacy/slow-writer/concurrency tests. Stdout is reserved for MCP. Screenshot validation is memory-only, bounded, fully decodes, and clears its buffer. Validator, checklist and protocol-gate artifacts are deliberately sanitized. CI uses fake fixtures rather than production credentials or devices. Documentation names privacy boundaries and external retention responsibilities.

**Residual Risk**

Telemetry is intentionally best-effort and cannot be an authoritative audit ledger. It lacks event time, process/version context, source/client attribution, device attribution and a loss indicator. Adding all of those would conflict with minimization; the accepted model should therefore provide modest timeline/completeness metadata and explicitly require corroboration. Sensitive data can still exist in process memory, core dumps, MCP client records, proxy logs, shell history, JetKVM firmware and debug_rpc results. Those are outside this server's payload-free telemetry control.

**Compatibility Or Semver Effect**

Adding fields to the versioned stderr JSON schema can affect strict log consumers even though it is not the MCP tool API. Prefer a documented jetkvm.operation.v2 or a separately versioned lifecycle record rather than silently changing v1. Explicit artifact retention and an incident runbook have no MCP SemVer effect. Removing or restricting debug_rpc would be a public tool compatibility decision and is not recommended in this workstream.

**Privacy Effect**

The recommendation preserves zero payload logging and avoids stable device or user identifiers. A timestamp, random process-instance ID, public binary version, and aggregate dropped-event count add limited operational metadata and cross-event linkability. Explicit short retention reduces exposure. Any future pseudonymous device tag still creates linkability and should require demonstrated incident need.

**Operational Effect**

Operators gain a usable timeline, version context, and an explicit warning when evidence was lost while mutations remain non-blocking. They must configure stderr access, rotation and deletion and preserve narrowly scoped evidence during an incident. CI artifact expiry becomes predictable. There is no collector, daemon, database, exporter, or new service to operate.

**Maintainer Effect**

The minimal design adds a small schema/test/documentation obligation, one retention setting per uploaded artifact, and a concise incident checklist. Closed enums keep review cheap. The maintainer must review telemetry fields for sensitivity whenever tools or results change and periodically confirm retention. Avoiding OpenTelemetry infrastructure, per-user audit, and device identifiers prevents substantial recurring operational and privacy burden.

#### Decision

**Disposition**

change

**Recommendation**

Retain the payload-free, closed-enum, bounded, non-blocking stderr design and all sanitizer tests. Before public release, make incident evidence explicitly time-bounded and honest: introduce a versioned safe lifecycle/operation schema containing UTC event time, a random per-process instance identifier and public server version; report aggregate dropped-event/writer-failure evidence at bounded shutdown when possible; state that absence of telemetry never proves absence of execution; document a short operator retention and incident-preservation procedure; and set explicit retention-days for sanitized CI artifacts. Keep aliases, addresses, URLs, paths, typed input, screenshots, media, credentials, RPC bodies/results and raw errors out of logs and artifacts. Use closed operation/stage/code/outcome dimensions only if metrics are ever derived.

**Minimal Practical Change**

Add observed_at in UTC RFC3339Nano plus process_instance_id and server_version in a versioned lifecycle/start record, and a shutdown record with bounded dropped_event_count and writer_failure_observed when the sink remains writable. Preserve operation events and non-blocking semantics. Update docs/telemetry.md with the completeness warning and an operator example recommending access-controlled stderr rotation and a suggested 14-day routine retention, subject to deployment/legal needs. Set retention-days: 30 on the existing sanitized coverage and protocol uploads. Add a one-page incident evidence checklist that captures exact binary digest/version, UTC window, telemetry/loss status, client-visible structured outcome, model/firmware, physical observation and sanitized proxy/supervisor context, while forbidding unnecessary payload capture.

**Optional Stronger Control**

After a real incident or external multi-device adoption demonstrates an attribution gap, consider an opt-in, per-process-salted device pseudonym in logs and deployment-owned authenticated access logs. If operational scale later requires central aggregation, export the existing low-cardinality events to an operator-owned log backend with field allowlisting, access control, deletion and cost limits. Do not enable this by default.

**Rejected Or Overengineered Alternatives**

Reject a built-in request/response logger, screenshot or typed-input audit trail, raw RPC/error capture, device-address logging, durable audit database, embedded dashboard, OpenTelemetry collector stack, universal traces/metrics, per-user identity system, SIEM requirement, immutable/WORM log, and blocking/lossless logging. They collect disproportionately sensitive data, introduce another control plane, or can impair operations. Reject correlation IDs, URLs, paths, aliases or user IDs as metric labels. Reject a universal legally framed retention period; deployment context determines obligations.

**Rationale**

The code already prevents the highest-risk privacy failure—durable capture of credentials and device payloads—and its tests enforce that claim. The actual gap is narrower: a consequence-aware server needs enough metadata to align an unknown physical outcome with an exact process and time, and it must disclose when its best-effort sink lost evidence. OWASP supports event time, interaction correlation, secret exclusion, retention and non-disruptive logging; NIST supports minimization and lifecycle control; OpenTelemetry supports guarding cardinality. A few safe fields, a loss summary, explicit retention and a checklist yield most of the incident value without converting a one-maintainer MCP server into an observability platform.

**Dependencies Or Prerequisites**

Choose whether to version operation records as jetkvm.operation.v2 or add a distinct lifecycle schema; expose the existing build version safely to telemetry; define precise counter behavior for event-queue, terminal-queue and writer failures; decide the documented suggested routine retention; verify GitHub's current accepted retention range and live repository/organization setting before workflow change; and coordinate terminology with the threat model, CI-quality and mutation-validation documents.

**Migration Or Rollout Considerations**

Treat telemetry as an operator-facing versioned interface: document v1/v2 coexistence or a clean pre-1.0 cutover, update fixtures before enabling emission, and keep unknown-schema consumers tolerant. Roll out CI retention independently because it is non-runtime. Make new operational metadata opt-out only if a documented threat model requires it; never make logging failure change a tool outcome. Existing retained logs and artifacts need not be rewritten. Announce that v1 absence remains ambiguous and that old events lack wall-clock context.

**Priority**

P1: complete before public release because the changes are small and materially improve incident truthfulness and data lifecycle without expanding payload collection.

**Implementation Effort**

S: a versioned safe lifecycle record/counters, focused tests, documentation, and workflow retention settings; no backend or schema migration.

**Ongoing Maintenance Burden**

low: review the allowlist with tool changes, confirm retention periodically, and update the incident checklist/version field at release time.

**Confidence**

high for repository behavior and the proportional recommendation; medium for deployment and live GitHub retention because those settings were unavailable.

#### Verification

**Acceptance Evidence**

A reviewed telemetry schema and fixture show only approved fields; emitted timestamps parse as UTC RFC3339Nano; process identifiers change across runs and contain no host/device identity; server version matches the built artifact; forced queue saturation and writer failure produce a bounded loss indication when the sink recovers or closes, while tool results and latency remain unaffected; documentation explicitly says logs are incomplete and payload-free; CI workflow uploads declare retention-days and actual run artifacts expire accordingly; an incident tabletop can correlate a simulated unknown-outcome operation with exact version, time window and loss status without collecting arguments or results.

**Proposed Tests Or Checks**

Extend telemetry unit tests for valid/invalid timestamps, process-ID uniqueness, version bounds, counter saturation, concurrent close, failed/blocked writer, and no operation blocking. Re-run privacy-sentinel tests with passwords, bearer values, cookies, aliases, URLs with query secrets, paths, typed text, PNG bytes, firmware text, RPC params/results, newlines and control characters across stderr, MCP errors, validator/checklist output and CI summaries. Parse every JSONL record against the closed schema. Inspect a completed Actions run for declared retention and sanitized artifact contents. Conduct a no-hardware tabletop using fake unknown-outcome and cancellation paths.

**Negative Or Abuse Cases**

Sink permanently blocks or fails; both queues fill; shutdown deadline expires; clock moves backward or is unavailable; version string contains attacker-controlled text; correlation fallback activates; concurrent operations interleave; malformed aliases/URLs/paths/newlines attempt log injection; debug_rpc sends tokens or large private results; schema validation errors echo client values only to the same caller; CI failure tries to upload raw child stdout/stderr; high-cardinality values are proposed as metrics; an investigator assumes no event means no mutation.

**Evidence Needed Before Claiming Support**

Do not claim complete, tamper-evident, lossless, attributable, or audit-grade logging without durable authenticated storage, source identity, completeness controls and adversarial operational evidence; this recommendation does not seek that claim. Do not claim privacy of client, proxy, shell, core-dump, firmware or appliance storage without inspecting those systems. Do not claim device compatibility from sanitized fixture or validator artifacts; require exact public model/firmware/server version, authorized hardware procedure, result, limitations and physical postcondition without serial/address/alias or captured content.

**Revisit Trigger**

Revisit after any privacy/security incident; introduction of multi-user or multi-tenant auth; remote default deployment; a new tool carrying sensitive payloads; changes to debug_rpc, screenshot/media handling, proxy/client logging or CI artifacts; adoption of metrics/tracing; sustained evidence loss; outside users requesting support; GitHub retention/plan changes; MCP telemetry guidance changes; or at least annually during threat-model review.

### 11. AI-assisted development and untrusted contribution security

#### Identity And Scope

**Item Name**

AI-assisted development and untrusted contribution security

**Research Question**

What minimal, evidence-based controls should jetkvm-mcp use to accept AI-assisted work and untrusted public contributions without letting generated plausibility, malicious instructions, new dependencies, shallow tests, excessive automation authority, or report volume overwhelm one maintainer or compromise a safety-relevant release?

**Scope**

This item covers coding-agent and generated-patch errors; automated dependency and contribution pull requests; fabricated APIs and packages, slopsquatting, hidden behavior and unfamiliar dependencies; shallow or self-fulfilling tests; malicious instructions in repository, issue, PR, review and artifact content; local/cloud agent secret, GitHub, runner, device and release authority; copied-code license/provenance uncertainty; control-bypass verification; review overload, low-evidence vulnerability reports and spam; and preserving accountable human judgment over shipment. It excludes general branch/ruleset design, detailed Actions/action-pin analysis, release provenance, and the project's own runtime agent threat model except where those boundaries constrain development automation.

**Repository Surfaces**

- AGENTS.md commands, credential/hardware boundaries, canonical document routing, GitHub-write authority and delivery rules
- CONTEXT.md bounded vocabulary and code/tests-authoritative statement
- docs/agents/issue-tracker.md complete issue/comment reads, GitHub mutations, leases, fail-closed checks and authority rules
- docs/agents/triage-labels.md live-state and maintainer-approval rules
- docs/ci-quality.md and Makefile quality gates, evidence boundaries and local parity
- .github/workflows/ci.yml pull_request trigger, contents:read permission, hosted runners, jobs and artifacts
- .github/dependabot.yml monthly grouped updates and one-open-PR limits
- go.mod, go.sum and testdata/mcp-gates/package-lock.json dependency identities and integrity evidence
- docs/product-contract.md, docs/threat-model.md, docs/protocol-sources.md, docs/adr/, README.md and exact tool manifest as review contracts
- Repository negative/no-dispatch, privacy-sentinel, race, fuzz, cancellation, malformed-input, protocol, cross-build and container tests
- Absence of CONTRIBUTING.md, SECURITY.md, CODEOWNERS, issue/PR templates, AI contribution policy and automated merge workflow at the studied tree
- Live GitHub repository metadata and recent owner-authored issue/PR history read 2026-08-15

**Applicability Stage**

current_private, before_public_release, after_external_adoption

#### Sources And Authority

**Source Class**

- Repository tree, test execution, commit/PR history and live GitHub metadata: repository_evidence.
- GitHub Actions and Copilot documentation: official_documentation and vendor responsible-use guidance; it is not neutral evidence of product efficacy.
- OpenSSF/CNCF guide and working-group issue: security_framework and evolving community guidance; issue 178 is explicitly unfinished.
- NIST SSDF: security_framework.
- USENIX package-hallucination study: peer_reviewed_research.
- curl policy: maintainer_case_study, useful for proportional intake controls rather than a rule this smaller project must copy.

**Normative Status**

- Repository AGENTS.md authority and CI configuration are executable/local policy for contributors and automation.
- GitHub workflow permission/trigger behavior is official platform behavior; recommendations are guidance unless enforced in repository settings.
- NIST SSDF and OpenSSF recommendations are risk-based guidance, not legal or protocol requirements.
- GitHub Copilot limitations are vendor observations and warnings, not proof that other agents behave identically.
- The USENIX study is an empirical observation about sampled models/languages; it supports verification of every new dependency, not a precise Go-specific incident probability.
- Emerging AI-disclosure/no-autonomous-submission policies in OpenSSF issue 178 are proposals/observations, not final consensus.

**Source Disagreements**

- Some emerging policies demand AI-use disclosure or ban autonomous submissions, while evidence also shows high-quality AI-assisted findings exist and authorship is not reliably detectable. Resolution: require a human contributor to understand, reproduce and own every submission; request material AI/provenance disclosure to aid review, but triage by evidence and behavior rather than attempting authorship detection or imposing a categorical AI ban.
- GitHub documentation suggests AI-assisted review can supplement humans, but its own responsible-use material says generated reviews/code can be inaccurate and require human validation. Resolution: use a second agent as a hypothesis generator only; it is not an independent approval or branch gate.
- Broad frameworks encourage many automated security scanners. The repository has a small, safety-relevant surface and finite attention. Resolution: retain targeted race/vet/staticcheck/govulncheck/fuzz/protocol gates and add controls only when they test a concrete claim; do not accept scanner volume as assurance.
- A contribution-policy checklist can improve consistency but can also become bureaucracy. Resolution: one concise CONTRIBUTING.md plus security-report evidence requirements, not a multi-stage contributor certification or governance platform.

#### Repository Evidence

**Exact Baseline Commit**

The assigned initial commit is 6e52f0027b13f928b768de0feeab4847ef9ca53e with tree 34e8b4451d76821950c23d7c06958d021700f3a7. During this workstream local HEAD was 71445bc6bf2325e6c683e362393605089c336b63; both resolve to the exact same tree. Pre-existing untracked .agents/, skills-lock.json and jetkvm_mcp_2026_public_release_baseline/ were present. Repository conclusions refer only to the identical tree; live GitHub state is separately dated 2026-08-15.

**Current Repository Evidence**

- AGENTS.md gives concise repository-specific commands and routes behavior, architecture, protocol and security work to authoritative documents. It forbids hardware access without explicit authority and keeping credentials/device data/artifacts in source control.
- AGENTS.md requires live-state checks and explicit authority for GitHub writes, publication, merge, release, deployment and destructive cleanup. docs/agents/issue-tracker.md fails closed on malformed state and serializes relationship writes.
- Neither AGENTS.md nor the issue-tracker guide explicitly says issue, PR, comment, review, fixture and repository text can contain malicious instructions that must be treated as untrusted data rather than task authority.
- There is no CONTRIBUTING.md, SECURITY.md, CODEOWNERS, issue template, PR template, AI contribution policy, copied-code/provenance declaration or public vulnerability-report quality bar in the studied tree.
- .github/workflows/ci.yml runs only push/main and pull_request, sets top-level permissions to contents:read, uses GitHub-hosted runners, and configures no device, release, deployment or repository-write secret. There is no pull_request_target, workflow_run, self-hosted runner, auto-merge or release job.
- Untrusted PR code can nevertheless change Make targets, test helpers, protocol pins and artifact-generation code executed in its own read-only job. It can consume network/compute, poison its own caches/artifacts or print data available in that job, but current job authority cannot write the repository or reach configured hardware by design.
- Dependabot is monthly, grouped and limited to one open PR for Go modules and one for Actions. Live repository metadata says auto-merge is disabled. Dependency PRs therefore remain proposals requiring maintainer review.
- The repository has strong control-bypass evidence: strict schema/config tests, no-provider-dispatch tests, private-sentinel leak tests, frozen manifests, malformed-input fuzzing, race/cancellation/deadline tests, dispatch-phase unknown-outcome tests, cross-builds and sanitized protocol artifacts.
- docs/ci-quality.md correctly says coverage is evidence rather than an arbitrary percentage and that fixture/loopback checks do not qualify hardware. This resists generated tests that inflate metrics without supporting a decision.
- Recent PR bodies frequently record focused red-before-green failures, exact candidate commits, scoped evidence limits, full gates, private-coordinate scans and independent exact-tree review claims. This is valuable review provenance, although independent human identity is not established.
- Recent history also shows fast correction/removal of disproportionate machinery: the read-only performance baseline from PR 70 was removed by PR 73 and a custom CI policy introduced in PR 68 was removed by PR 74. This is evidence that generated-looking completeness can create maintenance cost even when tests pass.
- Live repository metadata on 2026-08-15 showed a private repository and recent PRs authored by the OWNER. Several PRs merged within minutes, including small cleanups; elapsed time alone does not establish inadequate review but provides no independent-review assurance.
- go.mod uses fully qualified module paths and go.sum; protocol npm dependencies have a lock/integrity pin. These reduce accidental substitution but do not prove a new package name, maintainer, source or license is trustworthy.
- No workflow automatically turns issue/comment text into a privileged patch, merge, release or deployment. Local agent tooling documented under .agents/ exists as untracked workspace state and is not part of the studied release tree.

**Current State**

partial: the repository already has an excellent executable verification core and safely unprivileged PR CI, but it lacks a concise public contribution contract, an explicit untrusted-instruction rule for agents, a dependency/provenance review requirement, and a reproducibility bar for security reports. These are low-cost gaps before public release; a complex AI governance system is neither present nor warranted.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- **Scenario:** Poisoned issue, PR, comment, repository file or test fixture instructs a coding agent to reveal environment values, run a download, alter unrelated policy, contact hardware, approve/merge itself or close the ticket. | **Preconditions:** An agent reads the untrusted content and has shell, network, secret or GitHub-write authority beyond the maintainer's actual task. | **Current Controls And Gap:** Explicit task authority, no-hardware rules and GitHub write checks reduce scope; PR CI is read-only. Missing explicit instruction/data separation leaves local/cloud agents dependent on platform behavior.
- **Scenario:** An AI patch invents a plausible Go module/API or selects a freshly registered malicious repository at the hallucinated path; go mod download succeeds and executes code during builds/tests or ships it. | **Preconditions:** A contributor/agent adds the dependency and the maintainer reviews only compilation/tests or package existence. | **Current Controls And Gap:** go.sum and module verification authenticate bytes, Dependabot manages known updates, and the graph is small. No policy requires origin, history, maintainer, license, necessity and source review for a new module.
- **Scenario:** Generated code calls a real SDK API with fabricated semantics, ignores cancellation/unknown outcomes, leaks private result fields, or subtly broadens URL/file/device authority while compiling successfully. | **Preconditions:** Review focuses on syntax/green CI rather than product contract, threat model and negative paths. | **Current Controls And Gap:** Canonical docs and deep semantic tests are strong. Human must still trace the full path and independently derive abuse cases; an AI self-review is not proof.
- **Scenario:** A patch supplies shallow tests that mirror its implementation, mocks away the trust boundary, deletes/relaxes a failing assertion, regenerates the manifest, or raises coverage while never proving no dispatch/leak/retry. | **Preconditions:** Maintainer accepts passing gates without inspecting test sensitivity or first seeing the intended failure. | **Current Controls And Gap:** Many existing PRs record semantic RED cases and the repo rejects arbitrary coverage thresholds. A contribution policy should require test purpose, failure evidence where practical, and control-bypass negatives for sensitive changes.
- **Scenario:** A malicious PR changes CI/Make/protocol harness code so its own job prints available metadata, consumes resources, poisons caches/artifacts, disguises failures, or produces a deceptive report that a maintainer later trusts. | **Preconditions:** Untrusted code runs automatically; later privileged automation consumes its artifacts or a reviewer trusts generated output without inspecting workflow changes. | **Current Controls And Gap:** Hosted PR jobs have contents:read and no secrets/write/hardware. No privileged follow-on exists. Review must treat PR artifacts/caches as untrusted and scrutinize workflow/harness diffs.
- **Scenario:** A coding agent with a broad PAT, repository admin, release credential, signing identity, SSH keys, production config or device password is prompt-injected or simply wrong and pushes, merges, tags, releases, changes settings or touches hardware. | **Preconditions:** The agent receives standing authority not required for patch construction/testing. | **Current Controls And Gap:** Repository instructions require authorization but local agent credentials were not inspected. Deterministic least privilege and fresh human authorization are needed outside the repository.
- **Scenario:** Generated/copied code closely matches incompatible or unattributed public code, or a new dependency's license conflicts with MIT distribution. | **Preconditions:** Contributor assumes model output is original or code-reference filtering is complete. | **Current Controls And Gap:** Repository has an MIT license and dependency inventory evidence elsewhere, but no contributor provenance statement. Vendor matching is partial and cannot replace source/license review.
- **Scenario:** After publication, automated or low-effort AI PRs and vulnerability reports flood a sole maintainer with plausible prose, inflated severity and non-reproducible claims; genuine reports are delayed and burnout increases. | **Preconditions:** Low submission cost, no evidence template/quality bar, publicity or bounty-like incentives. | **Current Controls And Gap:** No external history yet. GitHub is canonical and triage fails closed, but no public intake policy says reports need exact revision, attack path, minimal reproduction and reporter understanding.
- **Scenario:** Multiple agents generate and approve the same patch, creating an appearance of independent consensus while sharing the same misunderstood premise, fabricated source or shallow test strategy. | **Preconditions:** Maintainer treats agent count or confident agreement as an approval gate. | **Current Controls And Gap:** PRs sometimes record independent review, but identity/method is unknown. Human shipment judgment and evidence diversity must remain explicit.

**Affected Assets And Trust Boundaries**

Assets are source and test integrity, Go/npm dependency graph, GitHub issues/PRs/main/tags/releases/settings, CI cache/artifacts and minutes, maintainer credentials and attention, license compliance, private device configuration/secrets, release/signing authority, and the physical hosts controlled by shipped code. Boundaries include untrusted contributor/issue text to human or coding agent; repository files to an instruction-following agent; PR code to hosted runner; third-party registry/module source to build; CI artifact to reviewer or later job; local agent to shell/environment/GitHub; and generated patch/review claims to the sole human release decision.

**Plausible Impact**

Failures can introduce a backdoor, secret leak, supply-chain compromise, unsafe retry, physical power/HID/media behavior, privacy regression, dependency/license obligation, false compatibility claim, CI/repository takeover if privilege later expands, corrupted release, or maintainer denial of service. Less dramatic AI-generated abstraction and test machinery can still make a small project incomprehensible and raise recurring maintenance cost. False security reports consume the same scarce review capacity needed for real vulnerabilities.

**Existing Controls**

- Concise canonical domain/product/threat/protocol/ADR documents and code/tests-authoritative rule.
- Explicit no-hardware, no-credential, scoped external-write and live-state instructions for agents.
- Single pull_request CI workflow with contents:read, hosted ephemeral runners, no secrets, no self-hosted runner, no privileged trigger and no auto-merge/release/deployment.
- Monthly grouped Dependabot with one-open-PR limits and manual merge boundary.
- Exact dependency locks/checksums plus module verification.
- Race, vet, staticcheck, govulncheck, fuzz, protocol, malformed-input, cancellation, privacy-sentinel, manifest, cross-build and container evidence.
- Strong product-specific negative tests proving no dispatch, redaction, bounds, cleanup and non-retryable ambiguous mutations.
- Issue/PR tracker as one canonical system with fail-closed mutation rules; no autonomous issue-to-release pipeline.
- Recent practice of exact-candidate checks, scoped PR evidence and removal of disproportional machinery.

**Residual Risk**

A sole maintainer remains the only shipment authority and can be persuaded by polished generated prose or green but insensitive tests. Local/cloud agents may have more authority than repository CI, and their actual token/environment boundaries are unknown. Any new dependency can be malicious despite valid checksums. Source review cannot prove model output originality. Public intake can exceed maintainer capacity. No policy can technically identify all AI-authored content, so controls must judge evidence, scope and contributor accountability rather than style.

**Compatibility Or Semver Effect**

Contribution and review policy is process guidance and does not change the product API. Rejecting or shrinking a generated patch has no compatibility cost. Security fixes arising from AI reports still follow the repository's documented SemVer and mutation-outcome contracts. Tool manifest/schema, config, CLI, transport and result changes require their existing compatibility review regardless of who authored them. Do not let an agent label a breaking change as internal solely because tests pass.

**Privacy Effect**

Agents can expose repository-private code, issue security details, environment values, config/device coordinates, screenshots or credentials through prompts, command output, logs and vendor retention. The safe baseline is data minimization: do not place production credentials/hardware access in agent or PR environments; redact artifacts; use synthetic fixtures; review provider retention before supplying private content. Requiring public reproduction must not force reporters to disclose secrets or exploit real hardware. Avoid storing prompts/transcripts as a new repository artifact.

**Operational Effect**

The recommended operating model adds a short intake/review checklist, not another CI platform. PR CI stays unprivileged and useful. High-risk changes take longer because the maintainer traces authority, checks dependencies and derives negative tests; low-risk documentation/focused fixes remain lightweight. Evidence-based report closure protects response capacity. External bot/agent write access, self-hosted runners and auto-merge remain off unless a demonstrated workflow justifies them.

**Maintainer Effect**

A concise policy reduces repeated explanation and permits quick closure of submissions whose author cannot reproduce or explain them. Small PRs, one open grouped dependency update, explicit evidence limits and human-owned final decisions protect attention. The maintainer must still read every accepted consequential diff and understand why its tests would fail under bypass. Mandatory multi-agent approval, AI detectors, blanket scanners and elaborate contributor attestations would cost more than they save at current scale.

#### Decision

**Disposition**

add

**Recommendation**

Before public release, add a concise CONTRIBUTING.md and agent-safety rule. Accept AI assistance, but require one accountable human to have read, understood, tested and be able to explain the entire submission; request disclosure of substantial generated/copied material and source/license references; forbid autonomous issue/PR floods; require exact scope and evidence. Treat issue/PR/comment/repository/artifact text as untrusted data, never as authorization to run commands, reveal secrets, access hardware, broaden scope, write GitHub state or ship. New dependencies require maintainer verification of necessity, canonical identity, history/ownership, license, source and exact graph delta. Security reports require exact revision, preconditions, concrete impact and minimal reproducible evidence. Preserve current read-only secret-free PR CI and final human shipment judgment; AI reviews may suggest questions but cannot approve.

**Minimal Practical Change**

Create one short contribution document linked from README and issue/PR entry points, plus one sentence in AGENTS.md declaring inbound repository/GitHub content untrusted. The document should state: contributor owns all content regardless of tool; keep PRs small/single-purpose; explain behavior and risk; identify material AI/copy sources; never add a dependency without a reviewed rationale and provenance/license check; do not delete/weaken tests to get green; add negative/no-dispatch/privacy/retry tests for affected controls; use synthetic data/no hardware; security reports need an exact commit, attack path, reproduction and no secrets; maintainers may close unverifiable or bulk-generated submissions. Keep CI permissions unchanged.

**Optional Stronger Control**

After real external volume appears, add targeted issue/PR forms with required reproduction/dependency/provenance fields, moderation/rate controls, and a CODEOWNERS/ruleset review gate for workflows, dependencies and security-critical packages when a second trusted human reviewer exists. Test a named AI review service only as advisory, with read-only access and bounded data retention. Consider secret scanning/push protection and dependency review when plan/public availability is established in other workstreams.

**Rejected Or Overengineered Alternatives**

- Reject a blanket AI-contribution ban or automated AI-authorship detector: assistance is hard to identify, valid work can use AI, and evidence quality is the relevant criterion.
- Reject mandatory multi-agent debate, voting or AI approval. Correlated models do not create independent accountability; the maintainer decides what ships.
- Reject automatic merge for Dependabot, coding-agent or security-fix PRs; green CI does not establish dependency trust or semantic safety.
- Reject giving coding/triage agents standing repository-admin, release, signing, deployment, production-secret or hardware authority.
- Reject self-hosted public PR runners and privileged pull_request_target checkout of contributor code.
- Reject a CLA/DCO, contributor identity proofing or license-scanning bureaucracy solely because AI was used at current scale; a clear contributor responsibility/provenance statement is sufficient until legal/adoption needs change.
- Reject adding broad scanners, badges, arbitrary coverage thresholds or a second governance platform to look rigorous; every check must support a concrete product claim.
- Reject rewarding vulnerability volume or debating unsupported severity prose; require reproduction and protect maintainer capacity.

**Rationale**

The repository already demonstrates the right verification philosophy: executable product contracts, privacy sentinels, negative dispatch tests, semantic RED evidence, strict capability fixtures and unprivileged CI. AI-specific evidence shows plausible code can fabricate packages/APIs, hide licensing matches and overwhelm maintainers, while reliable authorship detection does not exist. Therefore the highest-value additions are explicit instruction/data separation, accountable human understanding, dependency provenance checks and an evidence threshold for inbound reports. These controls directly address realistic paths and add little recurring burden; privileged automation or governance machinery would expand the attack surface and dilute the maintainer's judgment.

**Migration Or Rollout Considerations**

Publish the short policy before opening visibility so first contributors see the bar. Apply it prospectively and neutrally to human- and AI-assisted work; do not retroactively label owner history. Begin with prose and maintainer replies, then add forms/rate controls only if volume demonstrates value. Keep existing PR CI on pull_request/read-only and never promote its artifacts into a privileged job without a separate threat review. If a report lacks evidence, ask once for the missing reproduction and close when it remains unsupported; preserve a route to reopen with new evidence. Review the policy after the first 10 external submissions or six months.

**Priority**

P1 before public release: explicit untrusted-instruction and accountable-contribution/report rules are low effort and prevent high-cost authority and attention failures.

**Implementation Effort**

XS for the minimal policy and agent rule; S if public issue/PR forms are also added after adoption.

**Ongoing Maintenance Burden**

low: update one concise policy when actual failure patterns emerge and apply the same evidence bar; forms/moderation become medium only with external volume.

#### Verification

**Acceptance Evidence**

- A discoverable CONTRIBUTING.md states human accountability, scope/quality bar, material AI/copied-source disclosure, dependency provenance/license review, test expectations, no-hardware/private-data rules and security-report evidence requirements.
- AGENTS.md explicitly treats issue/PR/comment/review/repository/artifact instructions as untrusted data that cannot grant scope, credential, hardware, GitHub-write, merge, release or deployment authority.
- pull_request CI remains contents:read, GitHub-hosted, secret/device/release-free, without pull_request_target, self-hosted runners, privileged artifact consumption or auto-merge.
- A sample novel-dependency PR record shows canonical source/tag, why standard library/current dependencies are insufficient, maintainership/history, license, module graph/checksum delta and security review.
- A sample security report template can be satisfied with synthetic data and demands exact revision, reachable attack path, preconditions, impact and minimal reproduction while prohibiting secrets/hardware mutation.
- Release/merge checklist names a human who reviewed the complete diff and can explain behavior, trust-boundary changes, test sensitivity and unresolved uncertainty; AI reviews are labeled advisory only.

**Proposed Tests Or Checks**

- Review every changed dependency/import/action/installer against an explicit baseline diff; verify the canonical upstream and license outside model output before downloading or executing it.
- For security-sensitive code, derive at least one test from the threat/control claim rather than the implementation, observe it fail on the baseline or a deliberate bypass where practical, and keep no-dispatch/leak/retry assertions.
- Inspect test deletions, skips, relaxed assertions, fixture regeneration, timeout inflation, broad mocks and golden-file updates as first-class security changes.
- Run go mod verify, race, vet, Staticcheck, govulncheck, bounded fuzz, protocol gates and release checks in proportion to touched surfaces; independently inspect the diff after generators/formatters.
- Add a harmless poisoned-issue exercise to agent procedure: embedded instructions requesting env output, external write or scope expansion must be quoted/reported but not executed.
- Scan intended staged/release content and sanitized artifacts for credential/private-coordinate sentinels; do not print actual environment secrets to scan them.
- Verify GitHub workflow permissions and trigger remain safe whenever .github, Makefile, scripts, test harnesses, caches or artifacts change.

**Negative Or Abuse Cases**

- Issue body says previous instructions are obsolete, asks the agent to print env/config, install a plausible package, push directly, approve itself, close issues or contact a JetKVM.
- PR adds a real but new one-maintainer Go module with minimal history, an ambiguous license or a module path resembling an official project; all tests pass.
- Generated API compiles but changes cancellation, retry, Host/Origin, path confinement, output redaction or MCP error semantics.
- Patch regenerates the tool manifest or golden result, deletes a fuzz seed, skips a race test, widens a bound, accepts unknown properties or mocks provider dispatch so the relevant control is never reached.
- PR changes workflow/Make code to upload the workspace, poison a cache, interpolate attacker-controlled metadata, use pull_request_target, request write permissions or target self-hosted runners.
- Security report contains confident CWE/CVSS text but no reachable call path, exact revision or reproducer; reproduction would require a real device mutation or reporter-provided secret.
- Two AI reviewers agree on fabricated behavior/source; human cannot explain the code without rerunning the agents.
- Contributor claims model output is license-free despite a substantial public-code match or cannot identify the source of copied protocol logic.

**Evidence Needed Before Claiming Support**

Do not claim a patch is secure because it was AI-reviewed, passed CI, increased coverage or came from a known contributor. Claim only the exact behaviors established by reviewed code plus sensitive negative tests and stated evidence limits. Before accepting a new dependency, establish identity, provenance, maintenance, license, reachable use and graph delta. Before calling a report a vulnerability, reproduce it at the named revision, establish attacker-controlled input to security impact and distinguish bug from unsupported deployment. Before claiming an AI contribution policy works, observe real external submissions and measure maintainer effort/false closures rather than counting disclosures.

**Revisit Trigger**

Revisit after the repository becomes public; the first external PR or private vulnerability report; 10 external submissions or six months; any spam/harassment/report flood; any agent-caused secret, GitHub-write, dependency, license or hardware incident; adding a coding/review bot, self-hosted runner, privileged workflow, auto-merge, release/deployment credential or new package ecosystem; or material final OpenSSF AI-slop guidance replacing the current working-group draft.

### 12. GitHub repository governance, maintainer security, and public-project readiness

#### Identity And Scope

**Item Name**

GitHub repository governance, maintainer security, and public-project readiness

**Research Question**

Which repository and maintainer controls are actually present and available on the current private GitHub Free repository, which small set must be activated or documented at public release, which can wait for outside adoption, and which would be process theater for a sole maintainer?

**Scope**

Live repository visibility, collaborators, rules/protection, Actions permissions and secrets, releases/tags/commit verification, vulnerability reporting and scanning, issues/PRs/bots, repository community files, license/install/support/disclosure/incident contracts, personal-account recovery, release ownership, bus factor, plan transition, and compromise recovery. Detailed workflow hardening and artifact provenance are referenced but owned by their dedicated workstreams.

**Repository Surfaces**

- Live GitHub REST state for BenDManning/jetkvm-mcp on 2026-08-15: repository metadata, rulesets, main protection, collaborators, Actions permissions/secrets/environments/runs, Dependabot/secret/code-scanning alerts, releases, tags, commit verification, issues, and pull requests
- .github/workflows/ci.yml and .github/dependabot.yml
- README.md, LICENSE, AGENTS.md, docs/agents/issue-tracker.md, docs/agents/triage-labels.md, docs/product-contract.md, docs/threat-model.md, docs/ci-quality.md
- Absence checks for SECURITY.md, CONTRIBUTING.md, CODE_OF_CONDUCT.md, SUPPORT.md, GOVERNANCE.md, CODEOWNERS, issue forms, and pull-request template
- Local and GitHub verification state of exact baseline/current commit and annotated tag v0.1.0

**Applicability Stage**

current_private, before_public_release, after_external_adoption

#### Sources And Authority

**Authoritative Sources**

- **Title:** jetkvm-mcp exact repository tree and live GitHub state | **Publisher:** BenDManning/jetkvm-mcp and GitHub REST API | **Url:** https://github.com/BenDManning/jetkvm-mcp | **Publication Or Update Date:** live state inspected 2026-08-15 | **Version Or Revision:** source commit 6e52f0027b13f928b768de0feeab4847ef9ca53e; identical tree 34e8b4451d76821950c23d7c06958d021700f3a7; live HEAD 71445bc6bf2325e6c683e362393605089c336b63 | **Access Date:** 2026-08-15 | **Supported Claims:** Exact files, visibility, plan-gated API responses, collaborators, workflow permissions, secrets, releases, tag/commit signatures, PR/issue history, and scanning state.
- **Title:** Available rules for rulesets | **Publisher:** GitHub Docs | **Url:** https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets | **Publication Or Update Date:** continuously updated | **Version Or Revision:** GitHub.com as of 2026-08-15 | **Access Date:** 2026-08-15 | **Supported Claims:** Public repositories receive rulesets on GitHub Free; private repositories require Pro/Team/Enterprise; available branch/tag rules and bypass behavior.
- **Title:** About protected branches | **Publisher:** GitHub Docs | **Url:** https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches | **Publication Or Update Date:** continuously updated | **Version Or Revision:** GitHub.com as of 2026-08-15 | **Access Date:** 2026-08-15 | **Supported Claims:** Protection availability by visibility/plan and supported requirements such as status checks, reviews, deletion, force-push, and administrator enforcement.
- **Title:** Configuring private vulnerability reporting for a repository | **Publisher:** GitHub Docs | **Url:** https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configuring-private-vulnerability-reporting-for-a-repository | **Publication Or Update Date:** continuously updated | **Version Or Revision:** GitHub.com public-repository feature | **Access Date:** 2026-08-15 | **Supported Claims:** Public repository administrators can enable private vulnerability reports through Security Advisories.
- **Title:** About secret scanning alerts | **Publisher:** GitHub Docs | **Url:** https://docs.github.com/en/code-security/concepts/secret-security/about-alerts | **Publication Or Update Date:** continuously updated | **Version Or Revision:** GitHub.com as of 2026-08-15 | **Access Date:** 2026-08-15 | **Supported Claims:** Secret scanning runs automatically for public repositories at no charge and alerts repositories/providers for detected supported secrets.
- **Title:** About two-factor authentication | **Publisher:** GitHub Docs | **Url:** https://docs.github.com/en/authentication/securing-your-account-with-two-factor-authentication-2fa/about-two-factor-authentication | **Publication Or Update Date:** continuously updated | **Version Or Revision:** GitHub.com account security as of 2026-08-15 | **Access Date:** 2026-08-15 | **Supported Claims:** 2FA, passkeys/security keys, and permanent account-loss risk without recovery methods.
- **Title:** Configuring two-factor authentication recovery methods | **Publisher:** GitHub Docs | **Url:** https://docs.github.com/en/authentication/securing-your-account-with-two-factor-authentication-2fa/configuring-two-factor-authentication-recovery-methods | **Publication Or Update Date:** continuously updated | **Version Or Revision:** GitHub.com as of 2026-08-15 | **Access Date:** 2026-08-15 | **Supported Claims:** GitHub recommends multiple authentication methods and secure storage of one-time recovery codes.
- **Title:** Security hardening for GitHub Actions | **Publisher:** GitHub Docs | **Url:** https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions | **Publication Or Update Date:** continuously updated | **Version Or Revision:** GitHub Actions as of 2026-08-15 | **Access Date:** 2026-08-15 | **Supported Claims:** Least-privilege workflow tokens, untrusted contributions, third-party action pinning, secrets, and OIDC/workload identity guidance.
- **Title:** Open Source Security Foundation OSPS Baseline | **Publisher:** OpenSSF | **Url:** https://baseline.openssf.org/ | **Publication Or Update Date:** current edition accessed 2026-08-15 | **Version Or Revision:** current baseline | **Access Date:** 2026-08-15 | **Supported Claims:** Signals for access control, vulnerability disclosure, change review, release integrity, and project documentation; not automatically mandatory for this project.

**Source Class**

- **Repository And Live Github:** repository_evidence
- **Github Feature And Account Docs:** official_documentation
- **Openssf Osps:** security_framework

**Normative Status**

- GitHub feature/plan statements are official platform facts; they do not require the project to enable every available control.
- GitHub account security, Actions hardening, and recovery statements are guidance applied proportionally to a sole maintainer with release authority.
- Repository product/support promises become the project's own public contract once published.
- OpenSSF controls are framework guidance and signals, not product objectives or proof of security.

**Freshness And Supersession**

Live API responses were captured on 2026-08-15 and supersede repository documentation for current settings. GitHub's current ruleset model is preferred over creating overlapping legacy branch-protection rules after visibility changes. Plan/feature availability is time-sensitive and must be rechecked immediately before public conversion. The v0.1.0 release/tag evidence predates the current tree and remains relevant as historical public-distribution evidence.

**Source Disagreements**

- Repository issue #5 was closed as a public-readiness decision, but the live repository remains private and community/security files are absent. Resolve from live state: public transition is not complete.
- The repository has a strong PR/CI practice, but current GitHub Free private-plan APIs return 403 for rulesets and branch protection; successful PR runs do not prove required gates. Resolve by treating current practice as voluntary until an active public ruleset is read back.
- Signed-commit rules can strengthen provenance signals, but mandatory signatures do not replace review/CI and can exclude bots or contributors. Resolve by requiring verified release tags/attestations and protected tags first; do not mandate all commits for a one-maintainer project absent a concrete need.

#### Repository Evidence

**Exact Baseline Commit**

The required source baseline is 6e52f0027b13f928b768de0feeab4847ef9ca53e, tree 34e8b4451d76821950c23d7c06958d021700f3a7. Live HEAD during inspection was 71445bc6bf2325e6c683e362393605089c336b63 with the identical tree. Pre-existing untracked .agents/, jetkvm_mcp_2026_public_release_baseline/, and skills-lock.json were recorded and not treated as published repository content.

**Current Repository Evidence**

- Live repository: personal-account-owned BenDManning/jetkvm-mcp, private, GitHub Free feature behavior, default branch main, MIT license detected, Issues enabled, Discussions disabled, Wiki disabled, Projects enabled, forking allowed, and one administrator/collaborator BenDManning. Bus factor and release authority are one.
- Rulesets and main branch-protection APIs both returned HTTP 403: upgrade to GitHub Pro or make the repository public. Therefore no enforceable current branch/tag rule was demonstrated. Admin/bypass/review/check/deletion/force-push settings are unavailable now, not silently assumed.
- Actions is enabled; all actions are allowed; immutable SHA pinning is not required; default workflow token permission is read and Actions cannot approve PRs. ci.yml explicitly grants only contents:read and uses no repository secrets. Live API returned zero repository Actions secrets and zero environments.
- CI runs on pull_request and main push; latest studied PR and main runs succeeded. There are four jobs whose displayed checks are test, Minimum Go 1.25, MCP protocol gates, and container. No rule currently requires them.
- Dependabot is configured monthly for Go modules and Actions, grouped with one open PR limit each. Twenty-two PRs were all merged; authors were BenDManning and dependabot[bot]. No open PRs existed. The Dependabot alerts endpoint returned no open alerts, but enablement metadata was not independently exposed.
- Secret scanning endpoint states it is disabled; code scanning endpoint states it is not enabled. GitHub documentation says public repositories receive secret scanning for free. CodeQL is not automatically required because the existing Go analyzers/govulncheck cover concrete claims and another scanner adds maintenance.
- Private vulnerability reporting endpoint returned 404 while private. No SECURITY.md or documented private disclosure route exists. Open issue #29 already tracks contributor, support, and private vulnerability-reporting contracts.
- SECURITY.md, CONTRIBUTING.md, CODE_OF_CONDUCT.md, SUPPORT.md, GOVERNANCE.md, CODEOWNERS, issue forms/config, and PR template are absent. AGENTS.md is agent/operator workflow guidance, not a public contributor contract.
- LICENSE is standard MIT with 2026 copyright; README gives source, binary, and local-container install paths, checksum instructions, FFmpeg prerequisite, supported OS/architectures, and explicitly says GHCR is not yet published. It does not define supported release lines/EOL or a security-report route.
- One release v0.1.0 exists and is mutable (immutable=false). Its annotated tag object and target commit are unsigned/unverified. Current HEAD is GitHub-verified because GitHub web-flow signed the merged commit; local GPG verification could not validate GitHub's key, but the GitHub API reports valid verification.
- Repository permits merge commits, squash, and rebase and does not delete merged branches automatically. Recent history contains GitHub-verified merges as well as at least one unsigned commit. web_commit_signoff_required is false.
- Issues and PRs are the documented canonical tracker. There are 52 issues, six open including roadmap/release/security work; 22 merged PRs and no open PR. Current contributors are effectively the maintainer plus Dependabot, so review-count/CODEOWNERS enforcement would not add independent human judgment.

**Implementation And Documentation Agreement**

AGENTS.md and issue-tracker docs agree with live use of Issues/PRs and the active roadmap. README/LICENSE agree with current installation and licensing, though the public repository link is inaccessible to outsiders while private. Documentation does not falsely claim branch protection, private disclosure, SBOM/signing, or published container availability. The major gap is omission rather than contradiction: no public security, contribution, support/EOL, incident, or compromise-recovery contract exists, and v0.1.0 tag/release integrity is weaker than a trustworthy public release baseline.

**Current State**

partial: source documentation, MIT licensing, PR-based change history, CI, least-privilege workflow token, Dependabot, roadmap discipline, and safe install caveats are strong. Current private Free plan has no enforceable branch/tag rules; there is one admin; account recovery is unknown; scanning/disclosure/community contracts are absent; and the sole release tag is unsigned/mutable.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- A stolen sole-maintainer session/PAT force-pushes or deletes main or a release tag, modifies workflows, and publishes a malicious binary; no current branch/tag rule blocks it and no second owner can recover the project.
- A mistaken direct push reaches main without the full CI matrix because checks are practiced but not required.
- A reporter discloses a device-control vulnerability publicly because the repository has no SECURITY.md/private report button or supported-version guidance.
- An attacker replaces or recreates unsigned v0.1.0 tag/release assets; checksums hosted in the same mutable release do not independently establish publisher identity.
- A malicious or compromised third-party Action becomes reachable because repository policy allows all actions and does not require immutable SHAs; a future release workflow with write permissions would increase impact.
- A generated/spam issue or dependency PR consumes the sole maintainer's review budget, inducing shallow review or unsafe merging. Current grouped monthly one-PR limits mitigate bot volume but no public intake templates exist.
- The maintainer loses all 2FA recovery methods or becomes unavailable; a personal repository with one admin has no demonstrated continuity or release-revocation path.
- Users install @latest or an obsolete binary without a support/EOL policy and cannot tell which release receives security fixes.

**Affected Assets And Trust Boundaries**

Assets are source/main history, tags/releases/checksums, future container packages, Actions workflow identity and token, maintainer credentials/signing keys, vulnerability reports, issue/PR review attention, project name/reputation, and downstream operators. Boundaries are maintainer account to GitHub; contributors/forks/bots to PR workflows; Actions and third-party actions to GITHUB_TOKEN/secrets/OIDC; release publisher to consumers; reporters to maintainer; and the sole maintainer to project continuity.

**Plausible Impact**

Repository takeover can ship code with full JetKVM/host authority, erase audit history, redirect installation, suppress advisories, or compromise downstream credentials and machines. Missing disclosure/support policy increases public exposure and delayed remediation. Missing gates makes accidental untested changes plausible. Process-heavy alternatives mainly impose review delay and maintainer burnout without adding independent review.

**Existing Controls**

- One clearly identified admin, no repository secrets/environments, read-only default and explicit Actions token, no Actions PR approvals, and current CI without privileged release behavior.
- PR-based history with successful PR and push CI, comprehensive test matrix, canonical Issues roadmap, and durable issue/PR decisions.
- Monthly grouped Dependabot with one-PR caps limits bot noise.
- MIT license, explicit install methods/checksum instruction, platform scope, configuration/deployment warnings, and no false GHCR claim.
- Current HEAD GitHub-verified merge commit and clear upstream/protocol/source provenance documentation.
- No unnecessary webhooks, repository secrets, deployment environments, stale bot, Discussions, Wiki, or governance platform demonstrated.

**Residual Risk**

The controls are voluntary because no branch/tag enforcement exists. Sole-account compromise/loss remains the dominant governance risk. Unsigned mutable v0.1.0 is not a trustworthy release precedent. Public vulnerability intake and supported-version communication do not exist. Future release automation can silently enlarge Actions authority unless workload identity and protected release rules are designed first.

**Compatibility Or Semver Effect**

Governance controls do not change MCP behavior. SUPPORT/SECURITY must name supported release lines consistently with SemVer/product-contract decisions. Protecting tags means mistakes require a new tag rather than retagging; this is desirable but needs release discipline. Restricting merges to PR plus required checks changes maintainer workflow, not downstream API. Do not rewrite or retroactively sign v0.1.0; document that future releases use the new assurance baseline.

**Privacy Effect**

Private vulnerability reports may contain exploits, device data, screenshots, credentials, or network details and need minimal access/retention. Issue forms should warn users not to post secrets or device data. Public CI artifacts/logs must stay sanitized. Account recovery codes, PATs, signing keys, and incident notes must remain outside the repository.

**Operational Effect**

A small ruleset makes CI and PR history actual gates and prevents destructive branch/tag operations. SECURITY/SUPPORT/CONTRIBUTING reduce triage ambiguity. Private vulnerability reporting centralizes confidential intake without a second system. Account recovery and compromise drills are occasional manual work. Required independent reviews, rotations, councils, or complex issue automation would block a solo maintainer.

**Maintainer Effect**

Highest-value recurring work is monthly dependency review, prompt CI/ruleset exceptions review, quarterly account/access/advisory audit, release-time tag/artifact verification, and annual support/security review. CODEOWNERS and one-approval requirements provide no independent reviewer with one maintainer. Templates should be short and route evidence, not create administrative forms for their own sake.

#### Decision

**Disposition**

add

**Recommendation**

- Before public conversion, secure the maintainer account with at least two phishing-resistant/passkey or security-key methods plus offline recovery codes; inventory/revoke stale PATs, SSH/deploy keys, sessions, OAuth Apps, and signing/release credentials. Record only completion, never secrets, in a private recovery checklist.
- Add concise SECURITY.md, CONTRIBUTING.md, and SUPPORT.md. SECURITY should identify supported versions, private-report route, expected acknowledgement/remediation coordination, sensitive-data rules, and no public exploit request. SUPPORT should state best-effort support, no SLA, release/EOL policy, and issue versus security routing. CONTRIBUTING should require scoped PRs, tests, provenance/license for dependencies/generated code, and no secrets/device data.
- At the public transition, enable private vulnerability reporting and verify the button/notifications; accept free public secret scanning. Do not add a separate disclosure mailbox or ticket system unless GitHub intake proves insufficient.
- Create one active main ruleset on public GitHub Free: require pull requests with zero required approvals for the sole maintainer, require the four uniquely named CI checks, require conversation resolution, block force pushes/deletion, and apply rules to administrators. Allow an emergency bypass only if it is narrow, logged, and followed by an issue/post-incident review. Read back and test the rule before announcing public readiness.
- Create a release-tag ruleset preventing update/deletion of release tags. Require future release tags/checksums/artifacts to be cryptographically verified under the supply-chain ladder; do not rewrite v0.1.0.
- Keep Actions token default read-only and no secrets for PR CI. For future publishing use GitHub OIDC/workload identity or narrowly scoped short-lived credentials, protected release workflow/environment, and no pull_request_target execution of fork code.
- Retain Issues/PRs and Dependabot as the only work/intake systems. Add minimal issue forms only when the repository becomes public: bug, compatibility evidence, and feature request, each warning against secrets; provide blank security-issue links to private reporting.
- Defer CODEOWNERS, required independent approval, organization transfer, formal governance, stale bot, Discussions, contributor license agreement, and multiple release roles until a second sustained maintainer or real contributor volume exists.

**Minimal Practical Change**

Complete a private account/recovery/access audit; add three short public contract files; flip public; immediately enable private vulnerability reporting and secret scanning; install and test one main ruleset and one release-tag ruleset; protect future publishing with least-privilege workload identity; and document a one-page compromise response covering revoke/rotate, disable workflows/releases, preserve evidence, restore from known-good commit, republish new versions, and notify users. No committee or platform migration is needed.

**Optional Stronger Control**

After a second trusted maintainer emerges, transfer to a small organization with two owners, enforced 2FA, separated release/security roles, CODEOWNERS for high-risk workflows/release files, and one independent approval for those paths. Consider required signed commits or code scanning only if contributor volume, threat evidence, or a consumer requirement justifies their false-positive and onboarding cost.

**Rejected Or Overengineered Alternatives**

- Reject one required approving review now: a sole maintainer cannot independently approve their own PR, so it blocks releases or creates ceremonial reviewers.
- Reject CODEOWNERS now: it names the only owner and adds no new judgment unless paired with a real second maintainer.
- Reject a governance board, DCO/CLA bot, multi-agent approval system, external ticket platform, support portal, security inbox, or policy engine for the initial public project.
- Reject mandatory signing for every commit as a substitute for CI/review; prioritize protected verified release tags/artifacts and account security.
- Reject stale automation before issue volume demonstrates a problem; it can close valid low-volume hardware reports and adds bot authority/noise.
- Reject paid GitHub Pro as an immediate requirement solely for private rules: equivalent public rules become available on Free at the intended transition.
- Reject enabling every GitHub scanner or Scorecard badge merely to raise a score; each gate must support a concrete repository claim.

**Rationale**

This product has high runtime authority but a very small project and one maintainer. The dominant governance failures are sole-account compromise/loss, unenforced main/release integrity, confidential disclosure absence, and ambiguous support/release ownership. GitHub Free removes the current protection limitation as soon as the repository is public, so paying or building parallel machinery is unnecessary. A zero-approval PR gate preserves reviewable diffs and required CI without pretending there is an independent reviewer. Account recovery, protected tags, concise contracts, and a tested compromise plan deliver more real resilience than badges, committees, or broad automation.

**Dependencies Or Prerequisites**

- Maintainer-only verification of personal account authentication/recovery/access inventory and custody of release/signing credentials.
- Decision on supported release lines/EOL and security-response expectations for SECURITY.md/SUPPORT.md.
- Final stable unique CI check names and successful main run before ruleset activation.
- Public visibility or GitHub Pro; public is the intended no-cost route.
- Supply-chain workstream decision for signed tags/checksums, attestations, GHCR identity, rollback, and OIDC.
- A known-good offline source/release record and contact channel for compromise recovery.

**Migration Or Rollout Considerations**

Draft community/security contracts and account recovery before visibility changes. Immediately after conversion, verify public contents for leaked secrets, enable/report private vulnerability intake, enable free scanning, install rulesets, then test an ordinary PR, failed check, direct push, force push, tag update, and admin path. Keep v0.1.0 as historical unsigned evidence and establish the stronger baseline at the next release. If rules block emergency recovery, use only the documented logged bypass, preserve evidence, and restore enforcement afterward. Do not announce installation from GHCR until the package is actually published and verified.

**Priority**

P0 for maintainer account recovery/security before public ownership exposure; P1 before public release for SECURITY/SUPPORT/CONTRIBUTING, private reporting, main/tag rules, and compromise response.

**Implementation Effort**

S for documents and public rules; M including account audit, release identity, transition testing, and compromise drill.

**Ongoing Maintenance Burden**

low: quarterly access/recovery and settings audit, release-time verification, monthly dependency triage, and annual/support-triggered policy review.

**Confidence**

high for repository/live plan state and proportional recommendations; medium for personal-account and future-public settings because those are unobservable until the maintainer verifies or changes them.

#### Verification

**Acceptance Evidence**

- A private maintainer checklist records two strong authentication methods, tested offline recovery, and reviewed sessions/PATs/SSH/deploy/OAuth/App grants without exposing secret material.
- SECURITY.md, SUPPORT.md, and CONTRIBUTING.md are present, mutually consistent, linked from README/community profile, and tested by a maintainer walkthrough.
- Live public API/UI proves private vulnerability reporting and secret scanning enabled and a harmless private test report reaches the maintainer.
- Live ruleset JSON proves main requires PR, four exact CI checks, conversation resolution, no force push/deletion, and admin enforcement; direct/admin bypass tests behave as documented.
- Live tag rules reject update/deletion and the next release has a verified tag plus the supply-chain evidence selected by workstream 14.
- Repository Actions remains read-only by default; PR CI has no secrets; release permissions/identity are explicitly scoped and testable.
- A tabletop compromise exercise produces timestamps, revocation/rotation steps, known-good restoration, superseding release/advisory communication, and owner/recovery actions.

**Proposed Tests Or Checks**

- Before and after visibility change, export repository metadata, collaborators, Actions permissions, secrets/environments names, scanning/private-report state, rulesets, tag protection, releases, and merge settings to a private sanitized audit record.
- Open a test PR: verify all four checks are required, a failing check blocks merge, conversation resolution blocks merge, maintainer can merge after success without a fake approval, and PR fork code receives no secrets/write token.
- Attempt a disposable direct push, force push, main deletion, existing release-tag update/deletion, and administrator bypass without changing durable history; confirm denial or logged emergency behavior.
- Submit a harmless private vulnerability report and verify notifications, permissions, redaction, closure, and advisory workflow.
- Scan full Git history/release assets for secrets before public conversion and rotate any discovered value before exposure.
- Verify install instructions against a canonical signed release and checksum; reject @latest as reproducible guidance where an exact version is needed.
- Quarterly, review sole-admin/collaborators, Apps/PATs/keys/sessions, bot permissions, rules, release packages, supported versions, open security reports, and emergency-bypass events.

**Negative Or Abuse Cases**

- Compromised maintainer browser/PAT, stolen signing key, lost 2FA device/recovery codes, locked personal account, and maintainer unavailability.
- Direct push with skipped CI, malicious workflow change, fork PR attempting secret/token exfiltration, dependency bot compromise, and action-tag retargeting.
- Force-pushed main, deleted/recreated tag, replaced release asset/checksum, malicious GHCR image, and compromised OIDC subject/ref conditions.
- Public vulnerability report containing exploit/secrets, coordinated-disclosure disagreement, report spam, and accidental issue attachment of screenshots/configuration.
- Spam/AI-generated PR flood, fabricated dependency justification, stale bot closing valid hardware issues, and reviewer fatigue.
- Unsupported/EOL release security report, abandoned PR, emergency hotfix when CI is unavailable, and recovery from a known malicious release.

**Evidence Needed Before Claiming Support**

Do not claim protected development until live public rules are read back and abuse-tested. Do not claim private disclosure until the button and notification route work. Do not claim signed or immutable releases from v0.1.0; require next-release evidence. Do not claim maintainer-account resilience without a private recovery test. Do not claim community governance or response SLA with one maintainer; state best effort. Do not advertise GHCR or a supported release line until artifacts and EOL policy exist.

**Revisit Trigger**

Immediately before/after public conversion; any GitHub plan/feature change; addition/removal of collaborator, bot, secret, environment, release workflow, registry, or bypass; first external contributor/security report/spam wave; second sustained maintainer; account/release compromise; missed support expectation; annually; and whenever GitHub changes rulesets, vulnerability reporting, Actions identity, or scanning availability.

### 13. GitHub Actions and CI/CD security, reliability, and reproducibility

#### Identity And Scope

**Item Name**

GitHub Actions and CI/CD security, reliability, and reproducibility

**Research Question**

Does the exact jetkvm-mcp CI workflow provide trustworthy, proportionate evidence for its product and release claims while safely executing untrusted pull-request code, and which repository and workflow controls must change before public release?

**Scope**

Covers GitHub Actions triggers, token and secret authority, external actions and transitive code, pull-request trust, expressions, caches, artifacts and logs, hosted runners, network installation, tool and runner versions, timeouts, duplicate work, concurrency, job-to-claim traceability, branch gates, and bypass testing. Release publication credentials, signing and attestations are considered only as boundaries and are primarily workstream 14.

**Repository Surfaces**

Exact .github/workflows/ci.yml, .github/dependabot.yml, Makefile, docs/ci-quality.md, go.mod, go.sum, Dockerfile, testdata/mcp-gates/pins.json, testdata/mcp-gates/npm/package.json, package-lock.json, internal/protocolgate/**, cmd/jetkvm-mcp-protocol-gates/**, scripts/run-fuzz-targets.py, artifact sanitizer tests, and live repository Actions permissions, workflow-token permissions, secrets, variables, environments, cache usage, artifacts, recent runs, visibility, rulesets and main-branch protection APIs.

**Applicability Stage**

current_private, before_public_release

#### Sources And Authority

**Authoritative Sources**

- **Title:** Studied jetkvm-mcp repository tree and CI workflow | **Publisher:** BenDManning/jetkvm-mcp | **Url:** https://github.com/BenDManning/jetkvm-mcp/tree/6e52f0027b13f928b768de0feeab4847ef9ca53e | **Publication Or Update Date:** 2026-08-15 snapshot | **Version Or Revision:** 6e52f0027b13f928b768de0feeab4847ef9ca53e; identical tracked tree 34e8b4451d76821950c23d7c06958d021700f3a7 and later 71445bc6bf2325e6c683e362393605089c336b63 | **Access Date:** 2026-08-15 | **Supported Claims:** Actual triggers, actions, permissions, jobs, timeouts, caches, artifacts, tool pins and local gates.
- **Title:** Secure use reference | **Publisher:** GitHub | **Url:** https://docs.github.com/en/actions/reference/security/secure-use | **Publication Or Update Date:** continuously maintained; accessed 2026-08-15 | **Version Or Revision:** current GitHub Actions documentation | **Access Date:** 2026-08-15 | **Supported Claims:** Full-length commit SHA is the only immutable action reference; minimize token permissions; untrusted input and self-hosted public-runner risks.
- **Title:** REST API endpoints for GitHub Actions permissions | **Publisher:** GitHub | **Url:** https://docs.github.com/en/rest/actions/permissions?apiVersion=2026-03-10 | **Publication Or Update Date:** API version 2026-03-10 | **Version Or Revision:** 2026-03-10 | **Access Date:** 2026-08-15 | **Supported Claims:** Repositories can select allowed actions, require full-length SHA pins, set read-only default workflow permissions, and prevent workflow PR approval.
- **Title:** Script injections | **Publisher:** GitHub | **Url:** https://docs.github.com/en/actions/concepts/security/script-injections | **Publication Or Update Date:** continuously maintained; accessed 2026-08-15 | **Version Or Revision:** current GitHub Actions documentation | **Access Date:** 2026-08-15 | **Supported Claims:** Many github context fields and branch names are untrusted and direct expression interpolation into shell can execute attacker input.
- **Title:** Dependency caching reference | **Publisher:** GitHub | **Url:** https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching | **Publication Or Update Date:** continuously maintained; accessed 2026-08-15 | **Version Or Revision:** current GitHub Actions documentation | **Access Date:** 2026-08-15 | **Supported Claims:** Caches are not signed, may be read by fork PRs, and can enable code execution if trusted; pull_request caches are merge-ref scoped and low-trust default-branch cache writes are restricted.
- **Title:** Compromised runners | **Publisher:** GitHub | **Url:** https://docs.github.com/en/actions/concepts/security/compromised-runners | **Publication Or Update Date:** continuously maintained; accessed 2026-08-15 | **Version Or Revision:** current GitHub Actions documentation | **Access Date:** 2026-08-15 | **Supported Claims:** GitHub-hosted runners do not scan downloaded code; malicious code can steal GITHUB_TOKEN and referenced secrets; fork pull_request runs receive read-only access and no secrets.
- **Title:** CVE-2025-30066 / GHSA-mrrh-fwg8-r2c3 | **Publisher:** GitHub Advisory Database | **Url:** https://github.com/advisories/ghsa-mrrh-fwg8-r2c3 | **Publication Or Update Date:** published 2025-03-15; updated 2025-10-22 | **Version Or Revision:** reviewed advisory | **Access Date:** 2026-08-15 | **Supported Claims:** The tj-actions/changed-files compromise moved many version tags to malicious code and exposed runner secrets, demonstrating the concrete risk of mutable action tags.
- **Title:** Use GITHUB_TOKEN for authentication in workflows | **Publisher:** GitHub | **Url:** https://docs.github.com/en/actions/how-tos/writing-workflows/choosing-what-your-workflow-does/controlling-permissions-for-github_token | **Publication Or Update Date:** continuously maintained; accessed 2026-08-15 | **Version Or Revision:** current GitHub Actions documentation | **Access Date:** 2026-08-15 | **Supported Claims:** Actions can access github.token even if not explicitly passed and workflows should grant only minimum permissions.
- **Title:** Managing GitHub Actions settings for a repository | **Publisher:** GitHub | **Url:** https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository | **Publication Or Update Date:** continuously maintained; accessed 2026-08-15 | **Version Or Revision:** current GitHub documentation | **Access Date:** 2026-08-15 | **Supported Claims:** Repositories can allow selected owners/actions and require full SHA pins; fork PR workflows use read-only tokens without secrets under the documented setting.
- **Title:** Configuring retention for GitHub Actions artifacts and logs | **Publisher:** GitHub | **Url:** https://docs.github.com/en/organizations/managing-organization-settings/configuring-the-retention-period-for-github-actions-artifacts-and-logs-in-your-organization | **Publication Or Update Date:** continuously maintained; accessed 2026-08-15 | **Version Or Revision:** current GitHub documentation | **Access Date:** 2026-08-15 | **Supported Claims:** Default retention is 90 days and public/private repositories have different configurable ranges.

**Source Class**

Repository tree and live API responses are repository_evidence. GitHub security, API, caching and retention pages are official_documentation. GHSA-mrrh-fwg8-r2c3 is an official_advisory. No secondary source is required.

**Normative Status**

Repository workflow behavior and live settings are observations. GitHub's workflow syntax and permission mechanics are platform requirements; full-SHA pinning, least privilege, hosted-runner isolation and cache cautions are official guidance. The advisory is observation of a concrete incident. Proposed job composition, retention duration, concurrency policy and branch-gate selection are repository-specific judgments.

**Freshness And Supersession**

The source tree and live APIs were inspected on 2026-08-15. GitHub REST evidence used the 2026-03-10 API where documented. Recent official documentation supersedes older security-hardening page organization but retains the full-SHA, least-privilege and untrusted-input guidance. The 2025 tj-actions incident remains directly relevant. Action SHAs and hosted-runner images are intentionally time-varying and must be resolved and reviewed at implementation time.

**Source Disagreements**

GitHub permits tags and recommends full SHAs; this repository currently uses trusted publishers but mutable major tags. Because CI result integrity matters even without deployment secrets and the platform now offers SHA enforcement, resolve this in favor of full SHA pins plus reviewable tag comments. GitHub caching is supported and low-trust writes are scoped, while caches remain unsigned and untrusted; retain only if measured latency value justifies it, never as evidence. Reproducibility conflicts with ubuntu-latest and network installs, but fully hermetic CI is disproportionate; pin controllable identities and describe hosted images/network as environmental inputs rather than claiming byte reproducibility.

#### Repository Evidence

**Exact Baseline Commit**

The exact initial studied commit is 6e52f0027b13f928b768de0feeab4847ef9ca53e, diverging from the mission's expected 176ec421f9ee6c801517180e1ad0ec9c84570e8e. Its tracked tree is identical to 34e8b4451d76821950c23d7c06958d021700f3a7 and later 71445bc6bf2325e6c683e362393605089c336b63. Current HEAD during live inspection was 71445bc6bf2325e6c683e362393605089c336b63. Pre-existing untracked .agents/, skills-lock.json and research outputs were excluded from product evidence.

**Current Repository Evidence**

The single workflow triggers on pull_request and pushes to main, declares workflow-wide contents: read, uses no pull_request_target, workflow_run, issue event, deployment event, secrets, id-token or write permission, and has four GitHub-hosted ubuntu-latest jobs with 15-30 minute timeouts. test supports formatting/tidy/checksum/race/vet/staticcheck/govulncheck/fuzz/coverage/cross-build claims but make verify repeats ordinary tests and vet already run; minimum-go independently verifies Go 1.25 with GOTOOLCHAIN=local; protocol-gates uses Go 1.26.6, Node 22.22.0, an integrity-locked npm tree installed by npm ci --ignore-scripts, reviewed protocol pins, loopback fixtures and sanitized artifacts; container uses Buildx/QEMU for amd64/arm64 binary and image construction. All seven external uses references are mutable major tags: actions/checkout@v7 four times, actions/setup-go@v7 three times, actions/setup-node@v4, actions/upload-artifact@v4 twice, docker/setup-qemu-action@v4 and docker/setup-buildx-action@v4. checkout does not explicitly disable persisted credentials. No event/body/title/ref value is interpolated into a run script; runner.temp is the only expression used in shell/path context. setup-go caching is enabled in three jobs. The workflow has no concurrency cancellation and no explicit artifact retention-days. Dependabot groups Go and GitHub Actions monthly with one open PR each. Go and Node versions, Staticcheck v0.7.0, govulncheck v1.7.0, npm lock integrity, and container base digests are pinned; ubuntu-latest, action tags, apt repository contents and downloaded tool graphs are not fully immutable. Live API: Actions enabled; allowed_actions=all; sha_pinning_required=false; default_workflow_permissions=read; can_approve_pull_request_reviews=false; zero Actions secrets, variables and environments; no self-hosted Actions access; 39 active caches totaling 4,808,383,288 bytes; 38 artifacts; sampled artifacts expire after 90 days; recent pinned-tree PR and main runs succeeded in about 6-8 minutes. Ruleset and branch-protection endpoints returned a plan/publication 403.

**Implementation And Documentation Agreement**

docs/ci-quality.md accurately describes a single read-only, credential-free, fake/loopback CI authority and its four lanes. Code confirms timeouts, exact Go/Node/tool versions, npm integrity locking, sanitizer boundaries and no release/deploy authority. Its phrase pinned analyzers is accurate at the module-version/go.sum level, but CI actions themselves are not immutable. Its release lane description says make ci-quality although the workflow invokes individual steps and make verify, causing repeated test/vet work. No documentation claims full reproducibility, and the mutable runner/network inputs mean such a claim would be unsupported.

**Current State**

partial: job authority and product-claim coverage are substantially sound, but action identity and live policy are an avoidable supply-chain gap, branch gates are unavailable in the current private plan, and efficiency/evidence lifecycle controls are incomplete.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- A compromised action publisher moves a v4 or v7 tag to malicious code, as occurred with tj-actions/changed-files; the next PR or main run executes it. It can falsify checks, read the checked-out private source and read-only token, poison artifacts, and contact the network.
- A public contributor changes .github/workflows/ci.yml or Makefile so jobs retain expected names but skip real gates. Status checks turn green unless the maintainer reviews the gate-definition change; required checks alone do not prove the submitted workflow was trustworthy.
- A future maintainer adds pull_request_target plus checkout of PR code or adds secrets/write/id-token to the existing test workflow, turning deliberate arbitrary PR-code execution into a repository or credential compromise.
- An unsigned restored cache contains malicious executable material. GitHub's merge-ref/default-branch restrictions reduce cross-trigger writes, and Go content checks reduce risk, but the workflow must not treat caches as trusted evidence or store secrets in them.
- The default 90-day artifact/log retention preserves future accidentally sensitive evidence longer than intended; if-no-files-found: warn can also let a run finish without the artifact someone later assumes exists.
- Rapid pushes launch duplicate 6-8 minute workflows because no concurrency group cancels superseded revisions, increasing queue time, network exposure and maintainer noise.
- ubuntu-latest or live registries change beneath an unchanged commit; CI fails or changes behavior, and a passing run is incorrectly described as reproducible rather than evidence from a particular hosted environment.
- A compromised Go/npm/apt upstream or DNS/network path serves malicious content; checksums and npm integrity constrain module/package bytes, while apt package versions and action downloads remain environmental dependencies.
- A public self-hosted runner is later added to save cost; arbitrary PR code establishes persistence and steals credentials from subsequent privileged jobs.

**Affected Assets And Trust Boundaries**

Assets are repository source and CI truth, GITHUB_TOKEN, private source before publication, future release credentials, caches, workflow artifacts/logs, dependency and analyzer inputs, container outputs, branch integrity and maintainer attention. Boundaries are untrusted PR to reviewed repository, workflow YAML to GitHub controller, external action publisher to runner, runner to public registries, cache/artifact service to jobs/maintainer, and CI status to branch-merge decision.

**Plausible Impact**

A malicious or mutable CI dependency can falsify required checks, expose source/token data, prepare a compromised release, or mislead review. A privileged-trigger mistake can modify the repository or leak durable credentials. Reliability failures waste maintainer time or block merges. Mutable environmental inputs weaken forensic/verifiability claims, while excessive duplicate jobs consume quota and obscure signal.

**Likelihood Or Preconditions**

Mutable-tag compromise requires upstream action compromise or tag movement; the 2025 advisory demonstrates feasibility, though selected current publishers are prominent. PR gate bypass requires a malicious contribution plus insufficient maintainer review. Credential impact is presently limited because permissions are read-only and live repo secrets/environments are empty; it rises sharply when release authority is added. Cache poisoning requires a writable compatible cache scope or compromised trusted run. Runner drift and duplicate execution are routine rather than hypothetical.

**Existing Controls**

One small workflow; pull_request rather than pull_request_target; explicit contents: read and live read-only token default; workflow PR approval disabled; no secrets, variables, environments, OIDC or write jobs; GitHub-hosted ephemeral runners; per-job timeouts; exact Go/Node/analyzer versions; go.sum/module verification; npm ci with committed integrity lock and --ignore-scripts; container base digest pins; bounded fuzz; sanitized artifacts; loopback fixtures and no hardware; Dependabot Action updates; hosted-runner cache-scope protections; successful recent runs for the studied tree.

**Residual Risk**

Every external action can change under the referenced tag and live policy permits any action without SHA enforcement. checkout credentials remain persisted by default. PR code necessarily executes with network access and can alter gates. Hosted images, registries and apt packages are not hermetic. Caches and artifacts are service-controlled, unsigned evidence. Branch protection cannot currently be confirmed/enabled under the private plan. No CI design can replace maintainer review of changed workflow, Makefile, lockfile and fixture/gate definitions.

**Compatibility Or Semver Effect**

SHA pins, allowlists, persist-credentials:false, concurrency, artifact retention and branch settings do not alter the MCP API. Splitting duplicate Make targets changes developer commands only if existing names are removed; retain compatibility aliases. Pin updates may intentionally change CI behavior and need normal review. Public branch rules can change maintainer merge workflow and should be tested before declaring them mandatory.

**Privacy Effect**

No product/device secrets are present in current CI, and artifacts are sanitized. Short explicit artifact retention reduces exposure. Public PRs and external actions can still read repository contents and logs. Do not add device credentials, screenshots, typed input, raw protocol output, personal tokens or production endpoints to this workflow or its caches/artifacts.

**Operational Effect**

Immutable actions and enforcement make runs slightly more deliberate but more stable. PR concurrency reduces redundant work. Removing duplicated test/vet invocations should reduce duration without reducing claims. Disabling or narrowing caches increases downloads but reduces 4.8 GB churn. Branch gates become available after publication and turn the four job outcomes into merge conditions. Network outages remain a known CI dependency.

**Maintainer Effect**

Dependabot can propose SHA updates, but the maintainer must review upstream release notes and resolved commits rather than blindly merge a grouped action update. An allowlist, four required checks and a small workflow are comprehensible. Monthly review of cache/artifact use and occasional pin updates are low burden. Self-hosted runners, egress firewalls, duplicated security scanners and complex reusable-workflow policy would add disproportionate maintenance.

#### Decision

**Disposition**

change

**Recommendation**

Before public release, replace every external action tag with a reviewed full-length commit SHA and retain a human-readable version comment; set checkout persist-credentials:false; change live Actions policy from all actions/no enforcement to selected trusted publishers or exact actions plus required full-SHA pinning; retain read-only permissions, pull_request, hosted runners, zero secrets and no OIDC in CI. On public conversion, create a ruleset requiring the four meaningful job results on main and blocking force-push/deletion while preserving sole-maintainer emergency recovery. Add PR/main concurrency cancellation, explicit short artifact retention, and remove only demonstrably duplicate test/vet execution while preserving cross-build evidence. Treat cache contents and job outputs as untrusted and review any change to workflows, Makefile, lockfiles, gate pins or artifact selection as security-sensitive.

**Minimal Practical Change**

Pin actions/checkout, setup-go, setup-node, upload-artifact, docker/setup-qemu-action and setup-buildx-action to full 40-character commits; configure sha_pinning_required=true and selected actions for actions/* and docker/* or exact required repositories; add persist-credentials:false to checkout; add a safe concurrency group based on workflow plus PR number/ref with cancel-in-progress:true; set retention-days:30 on the two sanitized artifacts; use if-no-files-found:error when the producing step succeeded; split a cross-build-only target so the test job does not rerun make test and make vet inside make verify; and document the exact required job names and claim each supports.

**Optional Stronger Control**

If external adoption or a release workflow adds valuable credentials, separate release from untrusted CI, use protected environments and short-lived OIDC with audience-scoped cloud policy, require artifact attestations, and consider a reviewed egress-monitoring runner action pinned by SHA. If cache performance is negligible, disable setup-go caches entirely; otherwise keep current GitHub scoping and periodically prune stale PR caches. A static workflow analyzer such as zizmor is optional only if it catches a demonstrated recurring review failure and is itself pinned and maintained.

**Rejected Or Overengineered Alternatives**

Reject self-hosted runners for public PRs, pull_request_target execution of contributor code, long-lived PATs/cloud keys, secrets in the test workflow, an in-repository CI orchestration framework, enterprise required-workflow machinery, blanket network isolation that prevents the declared dependency gates, multiple duplicate SAST products, arbitrary coverage thresholds, full hermetic/reproducible-runner claims, and CI jobs without a product claim. Do not add a cache-cleanup workflow with write permission merely to manage 4.8 GB; first disable/reduce caching or prune manually when needed.

**Rationale**

The current design already contains the blast radius: PR code runs only on ephemeral hosted runners with read-only repository authority, no secrets and bounded time. Its most consequential preventable gap is identity: seven action references rely on movable tags while live settings allow every action and do not enforce SHA pins. GitHub identifies a full SHA as the immutable reference, and the tj-actions compromise shows tags can be retroactively redirected. SHA enforcement, an allowlist and non-persisted checkout credentials materially protect CI truth for little upkeep. Concurrency, explicit retention and deduplication then improve reliability and cost without adding gates for their own sake.

**Dependencies Or Prerequisites**

Resolve each desired action release to a publisher-controlled full SHA and review its release/source; confirm selected-action pattern syntax and SHA enforcement on the account/repository; publish the repository before relying on free rulesets/branch protection; select stable required check names; preserve local Make aliases while splitting duplication; decide whether 30-day evidence retention meets maintainer needs; and coordinate any future release credentials with the separate supply-chain workstream.

**Migration Or Rollout Considerations**

Pin and test actions in one reviewable change before enabling SHA enforcement so CI is not unexpectedly blocked. Then enable selected-action policy and run a deliberate negative PR using a tag/unallowed action to confirm rejection. Add public ruleset requirements only after all four checks have successful default-branch runs with stable names and retain a documented emergency bypass. Introduce concurrency after checking required-check behavior for canceled superseded commits. Measure job duration before/after deduplication and cache changes; roll back cache removal if network time is disproportionate.

**Priority**

P0 before public release for immutable action pins and live SHA/allowlist enforcement; P1 for credentials persistence, public branch gates, concurrency, retention and deduplication.

**Implementation Effort**

S: workflow/settings/documentation changes and focused negative tests; no new service or credentials.

**Ongoing Maintenance Burden**

low: review monthly Dependabot pin updates, verify required checks/settings after workflow changes, and periodically inspect cache/artifact use.

**Confidence**

high for workflow and exposed live Actions settings; medium for branch controls and public-fork behavior because the current private plan blocked those inspections.

#### Verification

**Acceptance Evidence**

All uses entries contain reviewed 40-character SHAs with version comments; repository API reports allowed_actions selected and sha_pinning_required true; workflow permissions remain read and PR approval false; checkout credentials are not persisted; repository has no CI secrets/environments/OIDC permission; a disallowed/tag-pinned action is rejected; four documented jobs pass on the exact default-branch commit; public main rules require those checks and prevent force-push/deletion; superseded PR runs cancel without allowing an ungated merge; artifacts show the chosen expiry and contain only sanctioned files; job timing confirms duplicate work removal did not remove cross-build coverage.

**Proposed Tests Or Checks**

Add a workflow-lint/control test that parses YAML and fails on non-SHA uses, pull_request_target, self-hosted labels, permissions beyond contents:read, secrets/id-token, missing timeouts, unsafe event-context interpolation, missing artifact retention, or unrecognized actions. Exercise a fork PR with malicious title, branch name, filename, Makefile, workflow change, cache contents and artifact text. Verify fork gets no secrets/write token and cannot populate default-branch cache scope. Force analyzer/npm/network failures and confirm fail-closed gates plus sanitized artifacts. Inspect API settings at release time and periodically compare required checks to workflow jobs.

**Negative Or Abuse Cases**

Mutable tag moved after review; selected action outside allowlist; short SHA; compromised allowed action at the pinned commit; contributor renames or replaces a required job with echo success; PR edits Makefile, npm lock or protocol pins; github.event title/ref contains shell syntax; checkout token read from git config; fork tries to read secrets, request OIDC, write contents or poison default cache; dependency emits ANSI/log commands; artifact path includes unintended files or symlinks; upload missing after producer success; registry outage; job hangs; two pushes race; runner image update breaks FFmpeg/Docker; canceled check is incorrectly accepted by ruleset.

**Evidence Needed Before Claiming Support**

A green workflow proves only the defined tests on the recorded GitHub-hosted environment and exact commit; it does not prove runtime platform/device compatibility, reproducible bytes, uncompromised upstream registries, physical qualification or release provenance. Claim a protected branch only after live public ruleset inspection and bypass test. Claim immutable CI dependencies only when every action is full-SHA pinned and enforcement is live; Go/npm locks and container digests cover different inputs and do not make ubuntu/apt/network hermetic.

**Revisit Trigger**

Any new workflow, action, secret, environment, OIDC permission, release/publish/deploy job, self-hosted runner, pull_request_target/workflow_run trigger, cache path, artifact, registry installer or contributor automation; an upstream action/advisory or tag compromise; GitHub plan/policy change; public conversion; required-check rename; repeated flaky/slow runs; cache growth or poisoning evidence; annually during release-baseline review.

### 14. Releases, containers, SBOMs, signing, provenance, and software supply chain

#### Identity And Scope

**Item Name**

Releases, containers, SBOMs, signing, provenance, and software supply chain

**Research Question**

What is the smallest release and container assurance baseline that makes jetkvm-mcp artifacts attributable, inventoryable, verifiable, and recoverable after compromise, while clearly separating integrity, provenance, dependency inventory, reproducibility, runtime hardening, and publication availability?

**Scope**

Covers the repository's GoReleaser archives, checksums, GitHub release process, source archives, Go build determinism, SBOMs, Cosign/Sigstore signatures, GitHub artifact attestations, SLSA provenance, containers and FFmpeg packages, consumer verification, publication, rollback, and compromise response. It does not treat an SBOM as a vulnerability verdict, provenance as a safety claim, a checksum as publisher authentication, or a locally buildable container as a published image. It excludes deployment authentication and general CI hardening except where they are part of the release trust boundary.

**Repository Surfaces**

Studied .goreleaser.yaml; Dockerfile; .dockerignore; go.mod and go.sum; Makefile; README.md release, go-install, FFmpeg, and container instructions; docs/product-contract.md; docs/threat-model.md; docs/ci-quality.md; .github/workflows/ci.yml; git tag v0.1.0; and the live GitHub v0.1.0 release, release assets, tag object, immutable-release setting, and artifact-attestation lookup.

**Applicability Stage**

current_private, before_public_release, after_external_adoption

#### Sources And Authority

**Source Class**

Repository tree, live GitHub release, tag, assets, and settings are repository_evidence. GoReleaser, Sigstore, GitHub, and Go pages are official_documentation. SLSA and CISA VEX are security_framework sources. No secondary source was needed for a decision.

**Normative Status**

Repository configuration and release API observations describe facts, not requirements. SLSA conformance statements are normative within a claimed SLSA level; this project currently makes no such claim. GitHub, GoReleaser, Sigstore, Go reproducibility, and CISA material are guidance. The recommended release controls are repository risk decisions, not externally imposed MUSTs.

**Source Disagreements**

GoReleaser documentation says signing only checksums is usually sufficient, while GitHub/SLSA emphasize per-subject provenance and artifact attestations. These solve different questions: a signed checksum authenticates one digest manifest; provenance binds each subject to a builder and invocation. Resolution: retain checksums, sign their manifest for portable publisher authentication, and add per-artifact GitHub provenance after the repository is public. SLSA promotes stronger hosted-build levels, but for this one-maintainer project plain GitHub-hosted provenance plus immutable publication is the proportional baseline; pursuing a level label is deferred until a consumer requires it. Exact Debian package pinning improves byte repeatability but can prevent timely security updates or later rebuilding from moving mirrors; resolution: record packages in an image SBOM and publish image digests, using a snapshot repository only if container reproducibility becomes a supported claim.

#### Repository Evidence

**Exact Baseline Commit**

The assigned exact initial studied commit is 6e52f0027b13f928b768de0feeab4847ef9ca53e. Current HEAD during this item was 71445bc6bf2325e6c683e362393605089c336b63; both resolve to identical tree 34e8b4451d76821950c23d7c06958d021700f3a7, so executable evidence was not mixed across trees. Expected baseline 176ec421f9ee6c801517180e1ad0ec9c84570e8e differs. Pre-existing untracked .agents/, skills-lock.json, and jetkvm_mcp_2026_public_release_baseline/ were present; only this research result was created.

**Current Repository Evidence**

Strong foundations: .goreleaser.yaml v2 builds linux/darwin amd64/arm64 with CGO_ENABLED=0, -trimpath, stripped symbols, version injection, deterministic names, and SHA-256 checksums.txt. Dockerfile pins both Go builder and Debian runtime images by digest, builds a static Go binary, uses a multi-stage build, and runs the final image as fixed non-root UID/GID 10001. CI and Makefile structurally build all four native targets and multi-platform container definitions. Gaps: there is no release workflow, GoReleaser/Syft/Cosign version pin, sboms/signs/docker/provenance section, OCI labels, registry publication, consumer identity-verification command, rebuild comparison, or compromise playbook. Dockerfile installs unversioned ca-certificates and ffmpeg from the moving Debian repository during build; those packages are outside go.sum and a native-binary SBOM. README recommends go install ...@latest and checksum verification, but @latest is time-varying and the unsigned checksum file does not authenticate the publisher. Its local docker build example leaves VERSION=dev. Live v0.1.0 contains checksums.txt and four archives, reports immutable=false, has an unsigned annotated tag, no uploaded SBOM/signature/provenance/source-bundle/container, and no artifact attestation for the sampled Linux archive digest. Repository immutable releases are disabled. Release notes identify tag commit f1be6653e494ba618c40dab9dd12cda34bd0bfab and state that a former private-registry image was not migrated.

**Implementation And Documentation Agreement**

Documentation accurately admits checksums without demonstrated SBOM, signature, provenance, complete CI pinning, or container registry publication. GoReleaser implements the four documented archive targets. README's checksum statement is true for corruption detection but could be misread as origin authentication unless qualified. README's source-build container claim agrees with Dockerfile and CI; there is no published-image support. The build-from-source example does not pass VERSION, so it conflicts with any expectation that the resulting container self-identifies as its source version.

**Current State**

partial: archive generation, checksums, pinned container base digests, cross-build coverage, and non-root runtime are useful controls, but the public release chain has no automated trusted builder, authenticated digest manifest, artifact-specific dependency inventory, build provenance, immutable publication, reproducibility evidence, or practiced compromise response.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- An attacker who compromises the maintainer account, mutable release, or ad-hoc workstation replaces an archive and its unsigned checksums.txt together; checksum verification succeeds but origin is false, and the binary gains JetKVM credentials and physical-control authority.
- An unrecorded or compromised local GoReleaser/Syft/Cosign executable changes build output; there is no builder identity or invocation evidence for consumers to distinguish it.
- A release contains a vulnerable Go module or Debian FFmpeg/library package, but no artifact-specific SBOM maps the advisory to shipped bytes; go.sum alone cannot inventory runtime OS packages.
- A consumer installs @latest or downloads a mutable tag at a later time and gets different code than a reviewed exact version; a container tag can similarly move unless its digest is pinned.
- A moving Debian mirror changes FFmpeg and dependencies under the same Dockerfile/base digest, producing a different and possibly newly vulnerable image; no image digest, SBOM, or rebuild record identifies it.
- The maintainer attempts to repair a compromised release by overwriting assets or reusing a tag, erasing forensic identity and making different consumers hold different bytes.
- An SBOM or provenance file is uploaded but consumers only check for its presence; a subject-digest mismatch, unexpected workflow identity, or altered parameters is accepted.
- A source-built container reports version dev, frustrating incident inventory and rollback even though the image may otherwise be correctly built.

**Affected Assets And Trust Boundaries**

Crosses source/tag to GitHub-hosted builder, dependency networks and package repositories, build tools, GitHub release storage, optional OCI registry, and consumer verification/runtime. Assets include release signing identity, GitHub OIDC identity, tags, archives, checksums, SBOMs, attestations, image manifests, Go modules, FFmpeg/OS libraries, JetKVM credentials available to the installed server, screenshots/media, and the server's physical HID/power/reset authority.

**Plausible Impact**

A substituted or vulnerable artifact can exfiltrate device credentials or observed screen/media data and issue keyboard, mouse, power, reset, wake, upload, or virtual-media operations. Less severe failures include inability to identify affected deployments, unreliable rollback, false security claims, broken macOS installation, divergent builds, and high-effort incident reconstruction.

**Likelihood Or Preconditions**

Release substitution requires maintainer/GitHub/workflow compromise, mutable-asset authority, DNS/dependency compromise, or an untrusted local build. Dependency inventory failures require an advisory affecting an actually shipped module or OS package. These are lower-frequency than normal code defects but high-impact because release consumers grant the binary device-control authority. Time-varying apt output and @latest selection are expected, not hypothetical. Container-registry risks are dormant while no image is published.

**Existing Controls**

SHA-256 checksums, exact module sums, CGO-disabled trimpath builds, version ldflags, four-target cross-build checks, digest-pinned base images, multi-stage container construction, non-root runtime, minimal apt install, CI contents:read by default, documented absence of stronger assurances, and exact v0.1.0 tag/commit identification reduce accidental corruption and ambiguity. GitHub exposes host-computed asset digests, but these are not independent publisher authentication.

**Residual Risk**

Checksums stored beside mutable artifacts are not an authenticity boundary. Pinned base-image digests do not pin the apt transaction. No evidence binds v0.1.0 bytes to a controlled builder, enumerates the shipped dependencies, or permits third-party rebuild comparison. Consumers cannot distinguish reviewed output from a malicious but consistently checksummed replacement. A public project would make these weaknesses more consequential through broader distribution.

**Compatibility Or Semver Effect**

Adding metadata, signatures, SBOMs, attestations, and immutable releases does not change MCP behavior or SemVer. Consumers and automation should move from @latest/mutable tags to exact versions and digests. Never reuse v0.1.0 or another published tag; a compromised artifact requires a new version. An explicit container publication promise would create a new supported distribution surface and patch cadence, so defer it until deliberately accepted.

**Privacy Effect**

SBOMs and provenance should disclose component names, versions, source repository, commit, builder identity, and build parameters, but must not include repository secrets, local paths, OIDC tokens, JetKVM addresses, credentials, media paths, or workflow environment dumps. Public build logs and diagnostic artifacts need the same redaction. Release signing uses public identity evidence rather than a long-lived private key.

**Operational Effect**

A single automated release path makes releases repeatable and produces an incident ledger of commit, workflow run, digests, SBOMs, signature bundle, and attestations. Verification adds a small consumer step. Immutable releases prohibit in-place repair, intentionally forcing a clear new version. Publishing a Debian/FFmpeg container would add regular base/package rebuild and vulnerability-response work; source-build-only containers avoid that publication burden.

**Maintainer Effect**

Initial automation is medium effort; once pinned, checksums, SBOM, keyless signing, and provenance are low recurring work. The maintainer must periodically update pinned release tools/actions and base digests, inspect failed signing/attestation, and respond to actionable SBOM advisories. Maintaining long-lived signing keys, dual SBOM formats, VEX for every CVE, custom FFmpeg builds, or SLSA-level paperwork would create disproportionate recurring work.

#### Decision

**Disposition**

add

**Recommendation**

Retain GoReleaser's narrow four-archive matrix, SHA-256 checksums, pure-Go trimpath build, pinned container base digests, and non-root runtime. Before public release, create one trusted, tag-driven GitHub-hosted release path with pinned Go patch, GoReleaser, Syft, Cosign, and immutable action commits; build once; generate checksums and one artifact-specific SPDX JSON SBOM per native deliverable; keyless-sign checksums.txt with a Cosign v3 bundle; record the exact workflow/run and version; upload assets to a draft; and publish only after verification with immutable releases enabled. Once public, add GitHub build-provenance attestations for each uploaded artifact and document identity-constrained verification. Keep the container as a source-build definition unless the maintainer deliberately accepts GHCR publication and FFmpeg patch duties. Do not retrofit or mutate v0.1.0; label it historical/private and use a new version for the assured pipeline.

**Minimal Practical Change**

Automate the next release from an exact protected tag on a GitHub-hosted runner; pin every executable release tool and action; emit archives, checksums.txt, SPDX JSON SBOMs, and a keyless Cosign bundle; verify all subject digests and the certificate repository/workflow/ref identity before moving a draft to published; enable immutable releases for future releases; and replace production examples using @latest with an exact version plus sha256sum and cosign verify-blob commands. Store no long-lived signing key. Record tag, commit, workflow URL, Go/GoReleaser/Syft/Cosign versions, and artifact hashes in release notes or attached provenance.

**Optional Stronger Control**

After public release, generate GitHub artifact attestations for archives, checksum manifest, SBOMs, and any image manifest digest, then document gh attestation verify with the expected repository. If external consumers demand stronger provenance, adopt GitHub's reusable SLSA Build L3 generator and a policy that verifies builder identity/build type/parameters. Demonstrate raw Go-binary reproducibility with clean independent rebuilds before claiming it; add deterministic archive comparison only if consumers need byte-identical archives. If publishing GHCR, produce a linux/amd64+arm64 manifest, OCI source/revision/version/license labels, image SBOM including Debian and FFmpeg packages, signature/attestation on the manifest digest, and digest-pinned deployment instructions.

**Rejected Or Overengineered Alternatives**

Reject long-lived GPG/Cosign private keys for a one-maintainer project when keyless OIDC is available; dual SPDX and CycloneDX SBOMs without a consumer requirement; boilerplate VEX for every release; claiming SLSA conformance merely for a badge; rebuilding FFmpeg from source; forcing scratch/distroless despite required FFmpeg/certificates; exact apt version pins on moving mirrors as a substitute for an image digest/SBOM; a private transparency service; paid GitHub Enterprise Cloud solely to attest a still-private pre-release; notarization as a public-release blocker absent macOS demand; and publishing GHCR merely because the Dockerfile exists. Also reject curl-pipe-shell installers and overwriting/reusing compromised release tags.

**Rationale**

This server's released bytes inherit authority over credentials, screens, host input, and power, so publisher authentication and incident traceability have high value. The existing pure-Go build and GoReleaser layout make SBOM/signature/provenance automation shallow rather than architectural. Signed checksums address portable origin authentication; per-artifact attestations address build origin; SBOMs address inventory; clean rebuilds address reproducibility; non-root/digest-pinned bases address runtime hardening; registry publication addresses availability. Keeping these claims separate prevents checkbox assurance. Public GitHub makes attestations feasible without a paid private-repository feature, while source-build-only containers avoid a second release product until users justify it.

**Dependencies Or Prerequisites**

Maintainer decision on trusted tag/release trigger and whether the next release should publish only native archives; public visibility before relying on no-cost GitHub artifact attestations; immutable-releases setting enabled before the next assured release; exact release tool/action pins; OIDC and attestations permissions scoped only to the release job; Syft and Cosign v3 availability; a documented expected certificate identity and issuer; and an explicit decision whether macOS code signing/notarization or GHCR availability is a supported product commitment.

**Migration Or Rollout Considerations**

Leave v0.1.0 unchanged because changing assets would destroy its historical identity. Test the workflow against a non-production tag or draft without publishing; validate schemas/signatures/attestations and archive execution; then publish a new SemVer version. Enable immutability before publication, attach every asset while draft, and never repair by replacement. On compromise, stop publication, rotate GitHub credentials/secrets, preserve affected digests and logs, publish an advisory identifying versions/digests, issue a fixed new version, move only mutable convenience tags to the fixed image, and use a later Go module retract directive where appropriate. Consumers should pin versions/image digests and retain verification evidence.

**Priority**

P0 before public release for authenticated, automated, immutable native releases and consumer verification; P1 for public-repository build attestations and artifact SBOMs if not included in P0; P2/defer for GHCR, reproducibility claims, notarization, and VEX until demand or evidence exists.

**Implementation Effort**

M for one release workflow, SBOM/signature/provenance outputs, settings, and documentation; L if GHCR multi-architecture publication and macOS notarization are included, which is not recommended now.

**Ongoing Maintenance Burden**

low for automated native checksums, SBOMs, keyless signatures, attestations, and immutable releases; medium if a Debian/FFmpeg container is published because it needs regular rebuild/advisory triage; high for notarization, custom FFmpeg, long-lived keys, exhaustive VEX, or formal reproducibility support.

**Confidence**

high for repository configuration, absence/presence of live release assets/settings, feature boundaries, and staged recommendation; medium for v0.1.0 payload details that could not be downloaded independently.

#### Verification

**Acceptance Evidence**

A future release created only by the pinned trusted workflow from an exact tag; immutable setting enabled; release API reports immutable; four archives plus checksums.txt, SPDX JSON SBOMs, and Cosign bundle are present; sha256sum succeeds; Cosign verification succeeds only with the expected GitHub OIDC issuer and exact repository/workflow/ref identity; SBOM subjects match archive/binary digests and enumerate Go dependencies; after public visibility, gh attestation verify succeeds for every subject and reports the expected repository/workflow; embedded --version matches the tag; release record contains commit and workflow URL. If a container remains unpublished, README says so. If published later, manifest platforms, OCI labels, SBOM including exact FFmpeg/Debian packages, non-root UID, signature/attestation, and digest-pinned pull all verify.

**Proposed Tests Or Checks**

- Run GoReleaser in snapshot/check mode with exact pinned version, inspect archives, execute each native binary where the platform permits, and compare embedded version to tag.
- Recompute every checksum from downloaded assets; alter one byte and require verification failure.
- Verify the Cosign bundle with expected issuer and certificate identity; repeat with wrong repository, workflow, ref, subject, and bundle and require failure.
- Validate SPDX JSON and ensure its subject digest matches the corresponding shipped artifact; compare listed Go modules with go version -m and flag omissions.
- After public release, run gh attestation verify for each digest and reject unexpected builder/repository/build parameters.
- Perform two clean raw-binary builds using the exact toolchain and module graph and compare hashes before making any reproducibility claim.
- For a future image, inspect linux/amd64 and linux/arm64 manifest entries, labels, user 10001:10001, version, installed FFmpeg version, image SBOM, signature, attestation, and digest pull.

**Negative Or Abuse Cases**

Replacement of archive and unsigned checksums together; signature bundle copied from another repository/ref; valid signature over the wrong checksum file; SBOM with wrong subject digest or missing FFmpeg; provenance from an unexpected workflow or user-controlled builder; release from a lightweight/retargeted tag; partial draft upload; retry after publication; expired/unavailable transparency-log services; malicious archive paths; tag reuse attempt; container tag drift; base digest unchanged while apt packages change; build secret leaked into SBOM/provenance/logs; VERSION left dev; compromised release requiring rollback; and a CVE for an SBOM component whose VEX status cannot be justified.

**Evidence Needed Before Claiming Support**

Do not claim signed releases until identity-constrained verification works from a clean consumer. Do not claim SLSA level until the exact v1.2 requirements and verifier policy are met. Do not claim reproducibility until independent clean rebuild hashes match for the claimed object and platforms. Do not claim an SBOM covers the container unless it inventories the final image's Debian/FFmpeg packages as well as the Go binary. Do not claim a published container until a registry manifest, supported tags/digests, update policy, and consumer verification exist. Do not claim macOS trust/notarization without Gatekeeper testing on supported macOS versions. A VEX statement requires component/product/version-specific exploitability analysis and justification.

**Revisit Trigger**

Before the first public release; any release-workflow, GoReleaser, Syft, Cosign, GitHub attestation/immutable-release, SLSA, Go toolchain, Debian base, or FFmpeg security change; decision to publish GHCR; first meaningful external consumer or platform request; an artifact/component advisory; maintainer/GitHub compromise; failed verification; repository plan/visibility change; or six months after adoption of the release pipeline.

### 15. Framework mapping and pragmatic one-maintainer operating model

#### Identity And Scope

**Item Name**

Framework mapping and pragmatic one-maintainer operating model

**Research Question**

Which current framework controls materially reduce a concrete security, reliability, release, or maintainer risk in this exact repository, which are already satisfied, and what minimal steady-state operating model can one maintainer sustain without turning framework scores or enterprise process into product objectives?

**Scope**

Applicable controls and signals from OpenSSF OSPS Baseline v2026.02.19, OpenSSF Scorecard, SLSA v1.2 Source and Build tracks, NIST SSDF 1.1, CISA Secure by Design, OWASP Agentic Applications 2026 and GenAI/LLM 2025-2026 material, MCP Security community work, and AAIF governance/projects. The scope maps these to all 15 repository workstreams and produces staged decisions, top changes, recurring work, and rejection criteria. It does not claim formal compliance, certification, procurement readiness, or an exhaustive control catalog.

**Repository Surfaces**

- Exact source/test/configuration/documentation tree at 6e52f0027b13f928b768de0feeab4847ef9ca53e and identical tree 34e8b4451d76821950c23d7c06958d021700f3a7
- go.mod, go.sum, Makefile, Dockerfile, .goreleaser.yaml, .github/workflows/ci.yml, .github/dependabot.yml, LICENSE, README.md, AGENTS.md, CONTEXT.md
- cmd/**, internal/**, tests/fuzz/fixtures/manifests, docs/product-contract.md, docs/threat-model.md, docs/protocol-sources.md, docs/ci-quality.md, docs/telemetry.md, docs/mutation-validation.md, docs/adr/**, docs/compatibility/**
- Live GitHub repository, rules/protection plan responses, collaborators, Actions permissions/secrets, releases/tags, issues/PRs, scanning and vulnerability-reporting state inspected 2026-08-15
- Structured evidence and decisions from the other repository-specific research workstreams in jetkvm_mcp_2026_public_release_baseline/results

**Applicability Stage**

current_private, before_public_release, after_external_adoption, future_only

#### Sources And Authority

**Authoritative Sources**

- **Title:** jetkvm-mcp exact baseline, live GitHub state, and repository-specific workstream evidence | **Publisher:** BenDManning/jetkvm-mcp | **Url:** https://github.com/BenDManning/jetkvm-mcp/commit/6e52f0027b13f928b768de0feeab4847ef9ca53e | **Publication Or Update Date:** 2026-08-15 | **Version Or Revision:** commit 6e52f0027b13f928b768de0feeab4847ef9ca53e; tree 34e8b4451d76821950c23d7c06958d021700f3a7 | **Access Date:** 2026-08-15 | **Supported Claims:** Actual product authority, implemented controls, missing evidence, live plan constraints, workstream priorities, and maintainability.
- **Title:** Open Source Project Security Baseline | **Publisher:** OpenSSF Security Baseline SIG | **Url:** https://baseline.openssf.org/versions/2026-02-19 | **Publication Or Update Date:** 2026-02-19 | **Version Or Revision:** v2026.02.19, current on 2026-08-15 | **Access Date:** 2026-08-15 | **Supported Claims:** Versioned maturity-oriented MUST controls for access, build/release, documentation, governance/legal, quality, and vulnerability management; controls should be focused, realistic, actionable, and meaningful.
- **Title:** OpenSSF Scorecard check documentation | **Publisher:** OpenSSF | **Url:** https://github.com/ossf/scorecard/blob/main/docs/checks.md | **Publication Or Update Date:** continuously updated | **Version Or Revision:** main inspected 2026-08-15 | **Access Date:** 2026-08-15 | **Supported Claims:** Signals for branch protection, CI, dangerous workflows, update tools, fuzzing, license, pinning, security policy, SBOM, signed releases, token permissions, and vulnerabilities; automated detection and scores have stated limitations.
- **Title:** SLSA specification | **Publisher:** OpenSSF SLSA | **Url:** https://slsa.dev/spec/v1.2/ | **Publication Or Update Date:** 2025-11-24 | **Version Or Revision:** v1.2 Approved | **Access Date:** 2026-08-15 | **Supported Claims:** Separate Source and Build tracks, incremental levels, source protection/history/provenance, build provenance and hosted/hardened builders, verification, and explicit claims.
- **Title:** Secure Software Development Framework Version 1.1 | **Publisher:** NIST | **Url:** https://doi.org/10.6028/NIST.SP.800-218 | **Publication Or Update Date:** 2022-02-03; updated 2022-11-29 | **Version Or Revision:** NIST SP 800-218, SSDF 1.1 | **Access Date:** 2026-08-15 | **Supported Claims:** Prepare the organization, protect software, produce well-secured software, and respond to vulnerabilities; practices integrate into an existing lifecycle rather than dictate one.
- **Title:** Shifting the Balance of Cybersecurity Risk: Principles and Approaches for Security-by-Design and -Default | **Publisher:** CISA, NSA, FBI, and international partners | **Url:** https://www.cisa.gov/resources-tools/resources/secure-by-design | **Publication Or Update Date:** 2023-04-13; revised 2023-10-25 | **Version Or Revision:** Secure by Design joint guidance | **Access Date:** 2026-08-15 | **Supported Claims:** Take ownership of customer security outcomes, embrace transparency/accountability, and make secure defaults a product responsibility while documenting deployment duties honestly.
- **Title:** OWASP Top 10 for Agentic Applications for 2026 | **Publisher:** OWASP GenAI Security Project | **Url:** https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/ | **Publication Or Update Date:** 2025-12-09 | **Version Or Revision:** 2026 edition | **Access Date:** 2026-08-15 | **Supported Claims:** Peer-reviewed threat starting point for goal hijack, tool misuse, identity/privilege abuse, supply chain, unexpected execution, memory/context poisoning, cascading failure, and human-agent trust.
- **Title:** OWASP Top 10 for LLM Applications | **Publisher:** OWASP GenAI Security Project | **Url:** https://genai.owasp.org/llm-top-10/ | **Publication Or Update Date:** 2025 edition; 2026 resources published 2026-08-03 | **Version Or Revision:** 2025/2026 project material | **Access Date:** 2026-08-15 | **Supported Claims:** Prompt injection, insecure output handling, excessive agency, sensitive disclosure, supply-chain, and misinformation risks, applied only where an MCP client/agent can reach this server.
- **Title:** Model Context Protocol Security community project | **Publisher:** Model Context Protocol Security / Cloud Security Alliance community | **Url:** https://modelcontextprotocol-security.io/ | **Publication Or Update Date:** continuously updated | **Version Or Revision:** community material accessed 2026-08-15 | **Access Date:** 2026-08-15 | **Supported Claims:** Non-normative MCP threat discovery and terminology for tool poisoning, cross-server confusion, authorization, and supply-chain monitoring.
- **Title:** Linux Foundation Announces the Agentic AI Foundation | **Publisher:** Agentic AI Foundation / Linux Foundation | **Url:** https://aaif.io/news/linux-foundation-announces-formation-of-aaif | **Publication Or Update Date:** 2025-12-09 | **Version Or Revision:** AAIF formation and initial hosted projects | **Access Date:** 2026-08-15 | **Supported Claims:** MCP and AGENTS.md have a neutral upstream governance home; AAIF projects and working groups are ecosystem signals, not required components of this server.
- **Title:** AAIF First Quarter Success Story: Open Governance | **Publisher:** Agentic AI Foundation | **Url:** https://aaif.io/blog/aaifs-first-quarter-success-story-new-members-technical-wins-and-open-governance/ | **Publication Or Update Date:** 2026-02-24; updated 2026-03-12 | **Version Or Revision:** AAIF governance/project update | **Access Date:** 2026-08-15 | **Supported Claims:** Current ecosystem work on identity/trust, security/privacy, observability, and MCP extensions should be monitored without importing enterprise controls by default.

**Source Class**

- **Repository And Workstream Evidence:** repository_evidence
- **Osps Scorecard Slsa Nist Cisa Owasp:** security_framework
- **Mcp Security Community:** secondary_commentary
- **Aaif:** official_documentation

**Normative Status**

- OSPS v2026.02.19 uses MUST language within its declared maturity/control scope, but this report does not assert an OSPS conformance level without a control-by-control versioned assessment.
- SLSA v1.2 requirements are normative only when claiming a Source or Build level. This repository has no Source VSA or build provenance, so it should make no level claim.
- NIST SSDF and CISA guidance are risk-management recommendations, not certifications or product-specific mandates.
- Scorecard outputs are observations/signals; its own documentation says detection is imperfect and independent review requirements may be infeasible for projects without enough participants.
- OWASP and MCP Security are threat frameworks/guidance. Official MCP specification/security guidance, repository behavior, and actual attack paths outrank them for protocol decisions.
- AAIF governance establishes upstream stewardship and monitoring channels; it does not require this repository to adopt AAIF-hosted gateways, extensions, working groups, or governance.

**Freshness And Supersession**

OSPS v2026.02.19 is the current version and supersedes v2025 editions for new assessments. SLSA v1.2 is Approved and supersedes v1.1/v1.0; its new Source track means older single-number SLSA descriptions are obsolete. NIST SSDF 1.1 remains current canonical SSDF despite its 2022 date. OWASP Agentic 2026 and newer GenAI material supersede generic 2023 lists for agentic threats. AAIF was formed in late 2025 and its 2026 updates are ecosystem context. All framework conclusions are bounded by repository/live evidence dated 2026-08-15.

**Source Disagreements**

- Scorecard rewards independent review and contributor diversity, while its documentation acknowledges review can be infeasible for projects without participants. Resolve by requiring PRs and CI with zero mandatory approvals now, then add one genuine review only after a second trusted maintainer exists; do not recruit ceremonial reviewers for points.
- CISA says manufacturers should take ownership rather than shift burden to users, while this server necessarily depends on MCP-client confirmation, reverse-proxy TLS, appliance egress, and host safety. Resolve by making safe server defaults and accurate consequences product-owned, and explicitly documenting only controls that are impossible at the server boundary as deployment/client duties.
- SLSA encourages level claims and provenance; reproducibility and two-party source review are separate properties. Resolve by first delivering signed hosted-build provenance and verification instructions, without claiming reproducibility or two-party review that has not occurred.
- OWASP/AAIF ecosystem material often assumes multi-agent, multi-user, gateway, memory, model, or enterprise identity systems. The repository has none. Resolve by mapping the caller-compromise/tool-authority paths and rejecting in-product gateway/policy/memory controls as non-applicable until architecture changes.
- OSPS control breadth can suggest a compliance backlog, while its own principles reject ineffective maintainer burden. Resolve each control through the repository's failure mode, acceptance evidence, and recurring cost rather than chasing completeness.

#### Repository Evidence

**Exact Baseline Commit**

The exact studied source is commit 6e52f0027b13f928b768de0feeab4847ef9ca53e, tree 34e8b4451d76821950c23d7c06958d021700f3a7. Live HEAD later became 71445bc6bf2325e6c683e362393605089c336b63 with the identical tree; an empty tree diff prevents revision mixing. Pre-existing untracked research/.agents/skills-lock files were recorded separately.

**Current Repository Evidence**

- Already strong / OSPS-SSDF Produce Well-Secured Software: strict bounded configuration; static typed MCP tools; explicit consequences and unknown outcomes; fresh session/resource ownership; bounded WebRTC/RPC/H.264/FFmpeg; os.Root media confinement; deny-by-default URL origins; privacy-safe typed results/errors; threat model, product contract, ADRs, protocol provenance, and extensive abuse tests.
- Already strong / Scorecard signals: MIT license, no checked-in release binaries, CI on PR/main, race/vet/staticcheck/govulncheck/fuzz/coverage/protocol/container checks, top-level contents:read workflow permissions, monthly grouped Dependabot, Go module checksums, and no current privileged PR workflow/secrets.
- Already strong / CISA ownership/transparency: loopback default, non-loopback bearer gate, exact Host/Origin admission, no OAuth/token passthrough, no automatic mutation retry, redacted media/status results, clear unqualified hardware/performance claims, and documented deployment residuals.
- Missing / source governance: current private GitHub Free repo has no enforceable main/tag rules, one admin, no SECURITY/CONTRIBUTING/SUPPORT, no private vulnerability reporting, and an unsigned mutable v0.1.0 tag/release. Account recovery is unknown.
- Missing / SLSA Build and release assurance: checksums exist, but no demonstrated SBOM, artifact signature, signed provenance, verification workflow, reproducible-build claim, complete immutable Action pinning, or qualified FFmpeg/package inventory. Therefore current Build level is unclaimed and strict SLSA evidence is absent.
- Missing / vulnerability/build floors: repository results identify just-released Go security patch misalignment, incomplete tracked analyzer/tool dependency graph, no exact release-binary govulncheck/license evidence, and inconsistent minimum/primary/container toolchains.
- Missing / MCP contract: repository research found the SDK advertises legacy protocol revisions beyond the product's claimed/tested semantics, inaccurate destructive/idempotent hints, incomplete identifier bounds, and small error/privacy/structured-content gaps.
- Missing / lifecycle/deployment: active Streamable HTTP work is not fully tied to shutdown cancellation; bearer parsing/request-body/read-timeout and operational proxy/TLS/rate-limit/token guidance need small corrections.
- Missing / evidence: real FFmpeg and packaged/container smoke must be unambiguous; JetKVM model/firmware mutation qualification, native target runtime, soak, and representative performance remain intentionally unclaimed.
- Agentic/OWASP mapping: no model, memory, planner, prompt ingestion, sampling, roots, resources, elicitation, or inter-server routing runs inside the server. Applicable threats are compromised/prompt-injected caller using full authority, malicious tool metadata/change, unsafe retry, secret-bearing inputs/results, supply-chain compromise, and multi-server tool confusion at the client. Static fixtures, bounded schemas, one principal, consequence descriptions, and non-retry outcomes reduce but cannot eliminate those paths.
- One-maintainer proportionality: repository packages are internal and conventional; one product context and no public Go API; no need for microservices, session pools, policy engine, multi-tenancy, OAuth, agent gateway, governance board, CODEOWNERS, required independent approval, or benchmark program without a decision objective.

**Implementation And Documentation Agreement**

Implementation and maintained docs agree strongly on product authority, safe defaults, privacy exclusions, non-retry ambiguity, fixture-versus-qualification boundaries, and intentionally excluded enterprise/agent features. Important contradictions are concrete and repairable: advertised legacy MCP revisions exceed the declared contract; tool annotations conflict with described physical consequences; the documented Go minimum is below current applicable security patch floors; current GitHub practice is not enforced; and release checksums can be mistaken for stronger assurance unless the release ladder is explicit. Framework mapping does not override code evidence.

**Current State**

partial: the engineering/security core is substantially stronger than a typical small pre-1.0 project and maps well to SSDF secure-development, CISA secure-default/transparency, and many OSPS/Scorecard controls. Public source governance, vulnerability intake, patched build floors, exact protocol claims, release identity/inventory/provenance, and a few boundary corrections remain. No framework level or compliance claim is justified yet.

#### Risk And Analysis

**Concrete Threat Or Failure Scenarios**

- A compromised sole maintainer or mutable CI dependency changes main/release artifacts; absent source/tag protection, signing, and provenance, consumers cannot reliably bind binaries to reviewed source.
- A public reporter cannot disclose privately or determine supported versions; a consequential MCP/device vulnerability is exposed publicly or remains untriaged.
- An old vulnerable Go/FFmpeg/toolchain builds an otherwise reviewed release; source-only analysis misses standard-library or binary-reachable flaws.
- A client selects an advertised legacy MCP revision whose safety-relevant semantics were never tested, creating a false compatibility promise.
- A prompt-injected/compromised agent uses the trusted principal's HID/power/media authority. Server-side prompt filters cannot determine intent; misleading annotations or automatic retry amplify the effect.
- Shutdown or lost acknowledgements interrupt consequential work; forced close prevents a typed outcome or cleanup, and retry duplicates host/physical mutations.
- Framework-score chasing adds scanners, badges, required reviewers, gateways, policies, or scheduled jobs that produce alerts no sole maintainer can review, displacing actual dependency, issue, and release work.
- Documentation claims physical compatibility, performance, reproducibility, SLSA level, or security compliance from fixtures/cross-builds rather than the required evidence.

**Affected Assets And Trust Boundaries**

Assets are reviewed source/history, maintainer/release authority, Actions identities, dependencies/toolchains, binaries/containers/checksums/SBOM/provenance, JetKVM credentials and sessions, local media, screenshots/typed text/device state, attached-host integrity and physical availability, private vulnerability reports, and maintainer attention. Boundaries are contributors/agents/bots to source review; GitHub/Actions/build inputs to artifacts; MCP client/agent to server; server to configured appliance/FFmpeg/files; appliance to host/media origin; and maintainer to users/reporters.

**Plausible Impact**

A control failure can ship a malicious or vulnerable release with full KVM authority, cause host commands/data loss/power interruption, disclose secrets/screens/media, create SSRF/network reach, lose audit history, or leave users unable to verify/remediate. A disproportionate operating model can cause maintainer burnout, noisy unreviewed automation, delayed security patches, and abandonment—the opposite of the frameworks' intended outcome.

**Existing Controls**

- Risk-directed design and tests, strict static public surface, strong result/error/privacy model, bounded resources and cleanup, conservative mutation outcomes, and explicit trust boundaries.
- Comprehensive CI, module integrity, dependency automation, least-privilege current workflow token, analyzer/fuzz/protocol/container evidence, and release checksums.
- Product contract, threat model, protocol provenance/drift ledger, CI quality matrix, telemetry policy, mutation validation, ADRs, and canonical GitHub issue roadmap.
- MIT licensing, supported platform/install documentation, loopback/default deployment safety, deny-by-default media URL origins, and non-root container.
- Deliberate exclusions of raw RPC from MCP, OAuth/token passthrough, multi-tenancy, dynamic tools, sampling/resources/prompts, policy/gateway control plane, session pooling, and unqualified compatibility/performance claims.

**Residual Risk**

The sole maintainer/source host/build publisher remains a concentration point; checksums alone do not resist publisher compromise. The trusted MCP principal can intentionally or through prompt injection exercise all device authority. FFmpeg/Pion/firmware/physical behavior cannot be fully contained or proven by repository tests. Deployment still owns TLS termination, network segmentation/egress, process limits, client confirmation, and secret retention. These residuals must be explicit rather than hidden behind framework badges.

**Compatibility Or Semver Effect**

Correcting legacy protocol advertisement, annotations, bounds, deprecated tool surface, and typed results may require the already-planned v1 boundary under the repository's stronger compatibility contract. Build/release/governance controls do not change MCP semantics but can narrow supported build/runtime/release lines. SLSA/OSPS claims must begin only from revisions/releases where evidence is continuously enforced; do not retroactively claim them for v0.1.0.

**Privacy Effect**

Framework implementation should preserve data minimization: no new request log, model transcript, prompt scanner, centralized telemetry plane, or public qualification artifacts containing screens, typed text, paths, URLs, tokens, firmware payloads, candidates, or device identifiers. SECURITY/incident evidence needs private access and retention. SBOM/provenance should contain build/dependency identity, not runtime/customer data.

**Operational Effect**

The recommended model adds a few release and quarterly gates, not a service. It makes CI/rules enforceable, adds private disclosure, produces verifiable release evidence, and maintains an explicit cadence. Hardware qualification and dependency/protocol reviews remain event-driven. Optional stronger controls are triggered by adoption, incidents, or architecture change instead of running continuously without an owner.

**Maintainer Effect**

Recurring burden is deliberately capped: monthly grouped dependency/issue review; release-time contract, artifact, vulnerability, license, SBOM/signature/provenance and smoke verification; quarterly account/GitHub/upstream/advisory audit; annual threat/support/retention/framework review. The maintainer reviews evidence and decisions, not framework scores. New automation must remove more recurring work than it creates and have a named response path.

#### Decision

**Disposition**

simplify

**Recommendation**

- Rank 1 — P0/XS/high value/low burden: align Go minimum, primary CI, and container to fixed current security patch releases; track analyzer/tool dependencies; scan the exact release binary and generate license evidence.
- Rank 2 — P0/S/high value/low burden: stop advertising untested legacy MCP revisions; correct destructive/idempotent annotations, remaining input bounds, and value-free sensitive validation errors at the v1 contract boundary.
- Rank 3 — P0-P1/S/high value/low burden: secure/test sole-maintainer account recovery, add SECURITY/SUPPORT/CONTRIBUTING, enable private reporting/free scanning, and activate public main/tag rules requiring PR plus the concrete CI checks without fake approval requirements.
- Rank 4 — P1/M/high value/low-medium burden: automate one release path that emits checksums, SPDX or CycloneDX SBOMs, keyless signatures and hosted-build provenance/attestations, pins trusted Actions immutably, uses least-privilege OIDC, and documents verification/rollback/compromise response. Claim only the achieved SLSA properties/level.
- Rank 5 — P1/S/high reliability/low burden: propagate shutdown cancellation through active Streamable HTTP calls and preserve bounded cleanup/unknown outcome; add focused shutdown tests.
- Rank 6 — P1/S/medium-high value/low burden: fix bearer grammar and explicitly own request body/read/time/admission limits; publish concrete loopback/proxy/TLS/token/network/process deployment profiles without adding OAuth or a policy engine.
- Rank 7 — P1/S-M/medium-high value/low burden: make real FFmpeg and packaged/container smoke non-skipping and capture one clean release-candidate evidence bundle; name qualified JetKVM/firmware only after approved hardware tests and keep other compatibility/performance unclaimed.
- Rank 8 — P1/S/medium value/low burden: add privacy-safe lifecycle/incident context and drop counters plus explicit CI/runtime retention, without payload logs or a telemetry backend.
- Rank 9 — P1/XS/medium value/low burden: make trusted-principal, prompt-injected-client, multi-server confusion, no-automatic-retry, and untrusted-contribution/instruction boundaries prominent in deployment/contribution/release review.
- After those changes, publish a scoped OSPS v2026.02.19 crosswalk and optional Scorecard result as consumer-facing evidence, not as a gate or target score.

**Minimal Practical Change**

Complete the nine ranked changes using existing packages, CI, GitHub, and release definitions; create one short versioned control crosswalk linking each applicable framework signal to repository evidence, owner/cadence, and acceptance check. Use GitHub Issues for remaining work. Do not add a GRC platform, scheduled framework workflow, or new runtime component.

**Optional Stronger Control**

After sustained external use or a second maintainer, add one independent review for high-risk source/release paths, organization recovery with two owners, reproducible-build comparison, stronger decoder/process/network sandboxing, periodic hardware soak, and client interoperability profiles. A deployment may add an external gateway/policy/identity layer for multi-user enterprise requirements, but keep it outside this server unless a new product context is explicitly accepted.

**Rejected Or Overengineered Alternatives**

- Reject aggregate Scorecard/OSPS/SLSA targets as release objectives; accept only controls tied to a threat, claim, and response owner.
- Reject claiming SLSA Source/Build levels without required VSAs/provenance and continuous enforcement; reject calling checksums provenance or SBOMs signatures.
- Reject CODEOWNERS, one/two mandatory reviewers, governance board, CLA/DCO bot, security committee, multiple release roles, or organization transfer until real people exist to perform them.
- Reject CodeQL/Sonar/extra scanners, long fuzz/soak schedules, benchmark dashboards, or duplicate CI merely for framework detection when existing gates answer the product decision.
- Reject MCP OAuth, multi-tenancy, per-tool scopes, AI gateway, agentgateway, policy engine, prompt-injection detector, semantic firewall, memory service, dynamic tool registry, or audit control plane inside the declared one-principal server.
- Reject pooled WebRTC sessions, FFmpeg daemon/sidecar, generalized provider framework, Kubernetes manifests/operators, service mesh, and enterprise observability stack without measured need.
- Reject broad hardware/firmware/platform/performance/reproducibility claims from fake fixtures, conformance, cross-builds, one run, or framework completion.

**Rationale**

OSPS, Scorecard, SLSA, SSDF, and CISA converge on protected changes, controlled dependencies/builds, secure defaults, vulnerability response, release integrity, and transparent evidence. The repository already implements most product-level secure-development practices; its most consequential omissions are source/release enforcement and a handful of false or incomplete public claims. OWASP/MCP-specific frameworks confirm that a compromised agent can misuse this server's authority, but the server has no model or memory to secure and cannot infer human intent. The proportional answer is accurate consequences, bounded schemas/results, no unsafe retry, least-privilege deployment, and client-side confirmation—not importing an enterprise agent stack.

**Dependencies Or Prerequisites**

- Completion and reconciliation of all 15 workstream result files and maintainer decisions on v1 scope, legacy compatibility, supported release lines, hardware qualification, and public timing.
- Public GitHub visibility or paid plan for enforceable rules; sole-maintainer recovery verification; stable unique CI check names.
- A selected canonical hosted release workflow, OIDC/signing/attestation mechanism, SBOM generator/format, license inventory, registry target, and consumer verification procedure.
- Patched Go/FFmpeg/tool versions and exact binary/container smoke evidence.
- Official MCP/SDK mechanism or adapter to restrict advertised protocol revisions.
- Authorized disposable JetKVM/host/network fixtures before any physical mutation or compatibility claim.

**Migration Or Rollout Considerations**

Sequence changes so evidence gates the claims: patch toolchains first; fix v1 protocol/tool contract; establish account/disclosure/source/tag rules; then publish through the new signed/SBOM/provenance workflow. Preserve v0.1.0 as historical unsigned evidence and do not rewrite tags. Release notes must call out breaking contract corrections and exact supported/qualified surfaces. Introduce stronger reviews or runtime controls only after their trigger, with rollback and a named maintainer response. Crosswalk status should use satisfied/partial/gap/not-applicable and cite evidence, never a marketing compliance claim.

**Priority**

P1 overall operating model, containing two immediate P0 prerequisites: patched build floors and truthful MCP protocol/release claims before public release.

**Implementation Effort**

M for the minimal public-release operating baseline across existing workstreams; XL only for explicitly rejected enterprise/high-assurance extensions.

**Ongoing Maintenance Burden**

low to medium: approximately monthly dependency/issue triage, per-release assurance, quarterly access/upstream review, and annual framework/threat/support review; hardware work remains release/advisory triggered.

#### Verification

**Acceptance Evidence**

- All 15 workstreams have a disposition, repository failure mode, acceptance test, priority, effort, and recurring burden; the executive action list contains only the highest combined-value changes.
- A versioned OSPS v2026.02.19 crosswalk links each applicable control to exact repository/live evidence and labels honest gaps/not-applicable controls; no unsupported conformance statement appears.
- A post-public Scorecard run is retained as a signal and each actionable low score maps to an issue or documented proportional rejection; no aggregate target is used.
- Live source/tag rules, security reporting, account recovery audit, and CI checks prove protected development and response readiness.
- The next release includes verified checksums, SBOM, signature, provenance/attestation, exact source/tag, binary vulnerability/license evidence, smoke results, rollback and consumer verification; any SLSA claim is independently checked against v1.2 requirements.
- Protocol discovery, annotations, schemas, shutdown, deployment, privacy, and contribution tests prove the concrete P0/P1 corrections.
- A dated cadence checklist shows monthly, release, quarterly, and annual tasks completed or consciously skipped with reason.

**Proposed Tests Or Checks**

- Before public release, review every recommendation against four questions: exact failure path, repository evidence, smallest effective change, and who will act on failure. Reject it if any answer is missing.
- Run the existing full validation plus focused protocol discovery, annotation/error privacy, HTTP shutdown/admission/bearer, real FFmpeg, offline archive/container, manifest/provenance/signature/SBOM, and negative verification tests.
- Read back GitHub rules, permissions, secrets/environments, scanning/private-report state, tag/release verification, and maintainer bypass behavior after public conversion.
- Verify a release from a clean consumer environment using only published instructions; compare digest, signature, provenance subject/source, SBOM identity, version output, and basic offline behavior.
- Tabletop source/build compromise, leaked credential, malicious dependency/Action, bad release, public vulnerability report, ambiguous physical mutation, and maintainer lockout; capture the minimum recovery evidence.
- Quarterly diff official MCP/SDK, JetKVM reviewed surfaces, Go/Pion/FFmpeg advisories, OSPS current version, SLSA approved version, and GitHub feature state; open work only for applicable changes.
- After adoption, measure contributor/security-report/hardware/latency/incident volume before enabling deferred controls.

**Negative Or Abuse Cases**

- Framework score recommends a bot/scanner/reviewer the maintainer cannot service; verify the recommendation is rejected rather than installed.
- Unsigned/mutable tag, mismatched provenance subject, SBOM for wrong artifact, compromised Action tag, release from unprotected ref, and checksum served by compromised publisher.
- Prompt-injected client calls keyboard/power/media, malicious multi-server metadata shadows a tool, lost acknowledgement invites retry, and private tool results poison later agent context.
- Public security report leaks a screenshot/config/token, unsupported-version report has no route, and incident evidence lacks safe timing/drop context.
- CI passes fixtures/cross-builds while real FFmpeg/container/device fails; ensure claims remain narrow.
- Sole maintainer unavailable, locked out, or compromised; rules/bypass and recovery must preserve both integrity and recoverability.
- New OAuth, stateful sessions, dynamic tools, gateway, model/memory, second principal, or persistent telemetry silently invalidates the not-applicable mapping.

**Evidence Needed Before Claiming Support**

Do not claim OSPS compliance without versioned full assessment; do not claim a Scorecard-derived security level; do not claim SLSA without valid v1.2 evidence and verification; do not claim SSDF/CISA certification because none exists here. Product claims require exact MCP wire tests, artifact verification, named runtime/tool versions, and named physical qualification where applicable. Agentic safety claims must distinguish what this server prevents/reduces/documents from what the client/deployment owns.

**Revisit Trigger**

Any public release or visibility change; new external maintainer/contributor/user; security or physical incident; new runtime principal/transport/tool/model/memory/gateway; official MCP/SDK revision/advisory; JetKVM firmware drift; Go/Pion/FFmpeg/GitHub supply-chain advisory; OSPS/SLSA approved revision; CISA/NIST/OWASP material change; consumer compliance request; or annual baseline review.

## Complete source registry

- [We Have a Package for You! A Comprehensive Analysis of Package Hallucinations by Code Generating LLMs](https://www.usenix.org/conference/usenixsecurity25/presentation/spracklen) — 34th USENIX Security Symposium; updated/published 2025; USENIX Security 2025 peer-reviewed paper; 576,000 samples across 16 models; accessed 2026-08-15.
- [AAIF First Quarter Success Story: Open Governance](https://aaif.io/blog/aaifs-first-quarter-success-story-new-members-technical-wins-and-open-governance/) — Agentic AI Foundation; updated/published 2026-02-24; updated 2026-03-12; AAIF governance/project update; accessed 2026-08-15.
- [Linux Foundation Announces the Agentic AI Foundation](https://aaif.io/news/linux-foundation-announces-formation-of-aaif) — Agentic AI Foundation / Linux Foundation; updated/published 2025-12-09; AAIF formation and initial hosted projects; accessed 2026-08-15.
- [Making retries safe with idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/) — Amazon Web Services, Amazon Builders' Library; updated/published 2021-01-15 announcement; living article accessed 2026-08-15; Malcolm Featonby article; accessed 2026-08-15.
- [Exact jetkvm-mcp repository tree associated with 6e52f0027b13f928b768de0feeab4847ef9ca53e](https://github.com/BenDManning/jetkvm-mcp/commit/6e52f0027b13f928b768de0feeab4847ef9ca53e) — BenDManning/jetkvm-mcp; updated/published 2026-08-15; commit 6e52f0027b13f928b768de0feeab4847ef9ca53e, tree 34e8b4451d76821950c23d7c06958d021700f3a7; accessed 2026-08-15.
- [Exact jetkvm-mcp repository tree at 6e52f0027b13f928b768de0feeab4847ef9ca53e](https://github.com/BenDManning/jetkvm-mcp/commit/6e52f0027b13f928b768de0feeab4847ef9ca53e) — BenDManning/jetkvm-mcp; updated/published 2026-08-15; 6e52f0027b13f928b768de0feeab4847ef9ca53e; accessed 2026-08-15.
- [Repository tree and release configuration at tree 34e8b4451d76821950c23d7c06958d021700f3a7](repository-local) — BenDManning/jetkvm-mcp; updated/published 2026-08-15 studied state; commits 6e52f0027b13f928b768de0feeab4847ef9ca53e, 71445bc6bf2325e6c683e362393605089c336b63; identical tree; accessed 2026-08-15.
- [Studied jetkvm-mcp repository tree](https://github.com/BenDManning/jetkvm-mcp/tree/6e52f0027b13f928b768de0feeab4847ef9ca53e) — BenDManning/jetkvm-mcp; updated/published 2026-08-15 commit snapshot; 6e52f0027b13f928b768de0feeab4847ef9ca53e; identical tracked tree at 34e8b4451d76821950c23d7c06958d021700f3a7 and later 71445bc6bf2325e6c683e362393605089c336b63; accessed 2026-08-15.
- [Studied jetkvm-mcp repository tree and CI workflow](https://github.com/BenDManning/jetkvm-mcp/tree/6e52f0027b13f928b768de0feeab4847ef9ca53e) — BenDManning/jetkvm-mcp; updated/published 2026-08-15 snapshot; 6e52f0027b13f928b768de0feeab4847ef9ca53e; identical tracked tree 34e8b4451d76821950c23d7c06958d021700f3a7 and later 71445bc6bf2325e6c683e362393605089c336b63; accessed 2026-08-15.
- [jetkvm-mcp exact baseline source and tests](https://github.com/BenDManning/jetkvm-mcp/commit/6e52f0027b13f928b768de0feeab4847ef9ca53e) — BenDManning/jetkvm-mcp; updated/published 2026-08-15; commit 6e52f0027b13f928b768de0feeab4847ef9ca53e, tree 34e8b4451d76821950c23d7c06958d021700f3a7; accessed 2026-08-15.
- [jetkvm-mcp exact baseline source, tests, protocol ledger, and ADRs](https://github.com/BenDManning/jetkvm-mcp/commit/6e52f0027b13f928b768de0feeab4847ef9ca53e) — BenDManning/jetkvm-mcp; updated/published 2026-08-15; commit 6e52f0027b13f928b768de0feeab4847ef9ca53e; tree 34e8b4451d76821950c23d7c06958d021700f3a7; accessed 2026-08-15.
- [jetkvm-mcp exact baseline, live GitHub state, and repository-specific workstream evidence](https://github.com/BenDManning/jetkvm-mcp/commit/6e52f0027b13f928b768de0feeab4847ef9ca53e) — BenDManning/jetkvm-mcp; updated/published 2026-08-15; commit 6e52f0027b13f928b768de0feeab4847ef9ca53e; tree 34e8b4451d76821950c23d7c06958d021700f3a7; accessed 2026-08-15.
- [jetkvm-mcp exact studied tree](https://github.com/BenDManning/jetkvm-mcp/commit/6e52f0027b13f928b768de0feeab4847ef9ca53e) — BenDManning/jetkvm-mcp; updated/published 2026-08-15 commit date context; exact timestamp not required for tree identity; 6e52f0027b13f928b768de0feeab4847ef9ca53e; accessed 2026-08-15.
- [jetkvm-mcp exact repository tree and live GitHub state](https://github.com/BenDManning/jetkvm-mcp) — BenDManning/jetkvm-mcp and GitHub REST API; updated/published live state inspected 2026-08-15; source commit 6e52f0027b13f928b768de0feeab4847ef9ca53e; identical tree 34e8b4451d76821950c23d7c06958d021700f3a7; live HEAD 71445bc6bf2325e6c683e362393605089c336b63; accessed 2026-08-15.
- [Minimum Requirements for Vulnerability Exploitability eXchange](https://www.cisa.gov/resources-tools/resources/minimum-requirements-vulnerability-exploitability-exchange-vex) — CISA; updated/published 2023-04-21; CISA VEX minimum requirements; accessed 2026-08-15.
- [Shifting the Balance of Cybersecurity Risk: Principles and Approaches for Security-by-Design and -Default](https://www.cisa.gov/resources-tools/resources/secure-by-design) — CISA, NSA, FBI, and international partners; updated/published 2023-04-13; revised 2023-10-25; Secure by Design joint guidance; accessed 2026-08-15.
- [Multi-platform builds](https://docs.docker.com/build/building/multi-platform/) — Docker; updated/published 2026-04 official documentation snapshot reported by the source index; Docker Buildx multi-platform documentation accessed 2026-08-15; accessed 2026-08-15.
- [Staticcheck repository and release guidance](https://github.com/dominikh/go-tools) — Dominik Honnef / Staticcheck maintainers; updated/published 2026-02-13; Staticcheck 2026.1, module tag v0.7.0; accessed 2026-08-15.
- [FFmpeg Download](https://ffmpeg.org/download.html) — FFmpeg Project; updated/published continuously updated; current download/support guidance; accessed 2026-08-15.
- [FFmpeg Security](https://ffmpeg.org/security.html) — FFmpeg Project; updated/published continuously updated; security tables current through FFmpeg 8.1.2 and 2026 CVEs; accessed 2026-08-15.
- [Compromised runners](https://docs.github.com/en/actions/concepts/security/compromised-runners) — GitHub; updated/published continuously maintained; accessed 2026-08-15; current GitHub Actions documentation; accessed 2026-08-15.
- [Configuring retention for GitHub Actions artifacts and logs](https://docs.github.com/en/organizations/managing-organization-settings/configuring-the-retention-period-for-github-actions-artifacts-and-logs-in-your-organization) — GitHub; updated/published continuously maintained; accessed 2026-08-15; current GitHub documentation; accessed 2026-08-15.
- [Configuring the retention period for GitHub Actions artifacts and logs in your organization](https://docs.github.com/en/organizations/managing-organization-settings/configuring-the-retention-period-for-github-actions-artifacts-and-logs-in-your-organization) — GitHub; updated/published current page accessed 2026-08-15; GitHub Actions documentation; accessed 2026-08-15.
- [Dependabot version updates and dependabot.yml reference](https://docs.github.com/en/code-security/concepts/supply-chain-security/dependabot-version-updates) — GitHub; GitHub.com documentation current 2026-08-15; accessed 2026-08-15.
- [Dependency caching reference](https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching) — GitHub; updated/published continuously maintained; accessed 2026-08-15; current GitHub Actions documentation; accessed 2026-08-15.
- [Events that trigger workflows](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows) — GitHub; GitHub.com documentation current 2026-08-15; accessed 2026-08-15.
- [GitHub Copilot agents application card](https://docs.github.com/en/copilot/responsible-use/agents) — GitHub; Responsible-use documentation current 2026-08-15; accessed 2026-08-15.
- [Managing GitHub Actions settings for a repository](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository) — GitHub; updated/published continuously maintained; accessed 2026-08-15; current GitHub documentation; accessed 2026-08-15.
- [REST API endpoints for GitHub Actions permissions](https://docs.github.com/en/rest/actions/permissions?apiVersion=2026-03-10) — GitHub; updated/published API version 2026-03-10; 2026-03-10; accessed 2026-08-15.
- [REST API endpoints for software bill of materials](https://docs.github.com/en/rest/dependency-graph/sboms?apiVersion=2026-03-10) — GitHub; updated/published 2026-03-10 API version; REST API 2026-03-10, SPDX 2.3 export; accessed 2026-08-15.
- [Review AI-generated code](https://docs.github.com/en/enterprise-cloud@latest/copilot/tutorials/review-ai-generated-code) — GitHub; updated/published 2026; GitHub Enterprise Cloud documentation current 2026-08-15; accessed 2026-08-15.
- [Script injections](https://docs.github.com/en/actions/concepts/security/script-injections) — GitHub; updated/published continuously maintained; accessed 2026-08-15; current GitHub Actions documentation; accessed 2026-08-15.
- [Secure use reference](https://docs.github.com/en/actions/reference/security/secure-use) — GitHub; updated/published continuously maintained; accessed 2026-08-15; current GitHub Actions documentation; accessed 2026-08-15.
- [Secure use reference for GitHub Actions](https://docs.github.com/en/actions/reference/security/secure-use) — GitHub; GitHub.com documentation current 2026-08-15; accessed 2026-08-15.
- [Storing and sharing data from a workflow](https://docs.github.com/en/actions/tutorials/store-and-share-data) — GitHub; updated/published current page accessed 2026-08-15; GitHub Actions documentation; accessed 2026-08-15.
- [Supply chain security and Dependabot feature availability](https://docs.github.com/en/code-security/concepts/supply-chain-security/supply-chain-security) — GitHub; updated/published 2026; GitHub.com documentation current 2026-08-15; accessed 2026-08-15.
- [Use GITHUB_TOKEN for authentication in workflows](https://docs.github.com/en/actions/how-tos/writing-workflows/choosing-what-your-workflow-does/controlling-permissions-for-github_token) — GitHub; updated/published continuously maintained; accessed 2026-08-15; current GitHub Actions documentation; accessed 2026-08-15.
- [Release v0.1.0 and Git data APIs](https://github.com/BenDManning/jetkvm-mcp/releases/tag/v0.1.0) — GitHub / BenDManning; updated/published 2026-08-10; v0.1.0, tag commit f1be6653e494ba618c40dab9dd12cda34bd0bfab; accessed 2026-08-15.
- [CVE-2025-30066 / GHSA-mrrh-fwg8-r2c3](https://github.com/advisories/ghsa-mrrh-fwg8-r2c3) — GitHub Advisory Database; updated/published published 2025-03-15; updated 2025-10-22; reviewed advisory; accessed 2026-08-15.
- [MCP Inspector proxy server lacks authentication between the Inspector client and proxy](https://github.com/advisories/GHSA-7f8r-222p-6f5g) — GitHub Advisory Database / MCP Inspector maintainers; updated/published 2025-06-13; updated 2025-07-09; GHSA-7f8r-222p-6f5g / CVE-2025-49596; fixed in 0.14.1; accessed 2026-08-15.
- [About protected branches](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches) — GitHub Docs; updated/published continuously updated; GitHub.com as of 2026-08-15; accessed 2026-08-15.
- [About secret scanning alerts](https://docs.github.com/en/code-security/concepts/secret-security/about-alerts) — GitHub Docs; updated/published continuously updated; GitHub.com as of 2026-08-15; accessed 2026-08-15.
- [About two-factor authentication](https://docs.github.com/en/authentication/securing-your-account-with-two-factor-authentication-2fa/about-two-factor-authentication) — GitHub Docs; updated/published continuously updated; GitHub.com account security as of 2026-08-15; accessed 2026-08-15.
- [Artifact attestations; feature availability](https://docs.github.com/en/code-security/getting-started/github-security-features) — GitHub Docs; GitHub artifact attestations; accessed 2026-08-15.
- [Available rules for rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets) — GitHub Docs; updated/published continuously updated; GitHub.com as of 2026-08-15; accessed 2026-08-15.
- [Configuring private vulnerability reporting for a repository](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configuring-private-vulnerability-reporting-for-a-repository) — GitHub Docs; updated/published continuously updated; GitHub.com public-repository feature; accessed 2026-08-15.
- [Configuring two-factor authentication recovery methods](https://docs.github.com/en/authentication/securing-your-account-with-two-factor-authentication-2fa/configuring-two-factor-authentication-recovery-methods) — GitHub Docs; updated/published continuously updated; GitHub.com as of 2026-08-15; accessed 2026-08-15.
- [Immutable releases](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases) — GitHub Docs; GitHub immutable releases; accessed 2026-08-15.
- [Increase the security rating of artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/increase-security-rating) — GitHub Docs; SLSA Build L3 reusable workflow guidance; accessed 2026-08-15.
- [Security hardening for GitHub Actions](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions) — GitHub Docs; updated/published continuously updated; GitHub Actions as of 2026-08-15; accessed 2026-08-15.
- [Verify release integrity](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/verify-release-integrity) — GitHub Docs; gh release verify and verify-asset; accessed 2026-08-15.
- [What to do when you receive a vulnerability report: a maintainer's guide](https://github.blog/security/vulnerability-research/a-maintainers-guide-to-vulnerability-disclosure-github-tools-to-make-it-simple/) — GitHub Security; GitHub maintainer guidance current on access date; accessed 2026-08-15.
- [GO-2026-4970: os.Root improperly follows final symlink ending in slash](https://pkg.go.dev/vuln/GO-2026-4970) — Go Vulnerability Database; updated/published 2026-07-07; CVE-2026-39822; fixed in Go 1.25.12 and 1.26.5; accessed 2026-08-15.
- [Checksum files](https://goreleaser.com/customization/package/checksum/) — GoReleaser; GoReleaser configuration v2; accessed 2026-08-15.
- [Signing artifacts](https://goreleaser.com/customization/sign/sign/) — GoReleaser; updated/published 2026-04-15; GoReleaser v2; Cosign v3 bundle examples; accessed 2026-08-15.
- [Software Bill of Materials](https://goreleaser.com/customization/sbom/) — GoReleaser; updated/published 2026-04-15; GoReleaser v2 documentation; Syft-backed pipe; accessed 2026-08-15.
- [Best Current Practice for OAuth 2.0 Security](https://www.rfc-editor.org/info/rfc9700/) — IETF; updated/published 2025-01; RFC 9700 / BCP 240; accessed 2026-08-15.
- [HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.html) — IETF; updated/published 2022-06; RFC 9110; accessed 2026-08-15.
- [The OAuth 2.0 Authorization Framework: Bearer Token Usage](https://www.rfc-editor.org/rfc/rfc6750.html) — IETF; updated/published 2012-10; updated by RFC 8996 and RFC 9700; RFC 6750; accessed 2026-08-15.
- [RFC 9110: HTTP Semantics, section 9.2.2 Idempotent Methods](https://www.rfc-editor.org/rfc/rfc9110.html#section-9.2.2) — IETF / RFC Editor; updated/published 2022-06; RFC 9110; accessed 2026-08-15.
- [MCP Security Notification: Tool Poisoning Attacks](https://invariantlabs.ai/blog/mcp-security-notification-tool-poisoning-attacks) — Invariant Labs; Technical maintainer/security case study; accessed 2026-08-15.
- [JetKVM KVM source repository](https://github.com/jetkvm/kvm) — JetKVM; updated/published continuously updated; inspected repository pin recorded locally; locally recorded inspected commit b3c29a44d9e2862b8ff7530830781803ce27b060; drift comparison to fe77acd5f00300a4ab9acd5da57d7bb0916351d9 on 2026-08-14; accessed 2026-08-15.
- [Potential Command Execution in MCP Inspector via XSS When Connecting to an Untrusted MCP Server](https://github.com/advisories/GHSA-g9hg-qhmf-q45m) — MCP Inspector maintainers; updated/published 2025-09-06; GHSA-g9hg-qhmf-q45m; accessed 2026-08-15.
- [2026-07-28 Specification Changelog](https://modelcontextprotocol.io/specification/2026-07-28/changelog) — Model Context Protocol; updated/published 2026-07-28; Changes from 2025-11-25 to 2026-07-28; accessed 2026-08-15.
- [Caching](https://modelcontextprotocol.io/specification/2026-07-28/server/utilities/caching) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28; accessed 2026-08-15.
- [Deprecated Features](https://modelcontextprotocol.io/specification/2026-07-28/deprecated) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28 deprecation registry; accessed 2026-08-15.
- [MCP 2026-07-28 Final Release](https://blog.modelcontextprotocol.io/posts/2026-07-28/) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28 final release; accessed 2026-08-15.
- [MCP 2026-07-28 Release Candidate](https://blog.modelcontextprotocol.io/posts/2026-07-28-release-candidate/) — Model Context Protocol; updated/published 2026-05-21; 2026-07-28 release candidate guidance retained by final release; accessed 2026-08-15.
- [MCP Specification 2026-07-28: Schema Reference](https://modelcontextprotocol.io/specification/2026-07-28/schema) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28; accessed 2026-08-15.
- [MCP Specification 2026-07-28: Tools](https://modelcontextprotocol.io/specification/2026-07-28/server/tools) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28; accessed 2026-08-15.
- [MCP stdio transport specification](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28; accessed 2026-08-15.
- [Official MCP Conformance Suite](https://github.com/modelcontextprotocol/conformance) — Model Context Protocol; repository pin 0.2.0-alpha.11 at c321dd32035556e6769d3724a8ee97d87c3faaac; accessed 2026-08-15.
- [Protocol Versioning](https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28; accessed 2026-08-15.
- [Security Best Practices for MCP](https://modelcontextprotocol.io/docs/2026-07-28/tutorials/security/security_best_practices) — Model Context Protocol; Guidance aligned to MCP 2026-07-28; accessed 2026-08-15.
- [Server Discovery](https://modelcontextprotocol.io/specification/2026-07-28/server/discover) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28; accessed 2026-08-15.
- [Streamable HTTP Transport](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28; accessed 2026-08-15.
- [Tool Annotations](https://blog.modelcontextprotocol.io/posts/2026-03-16-tool-annotations/) — Model Context Protocol; updated/published 2026-03-16; 2026 guidance; accessed 2026-08-15.
- [Tools](https://modelcontextprotocol.io/specification/2026-07-28/server/tools) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28; accessed 2026-08-15.
- [stdio Transport](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio) — Model Context Protocol; updated/published 2026-07-28; 2026-07-28; accessed 2026-08-15.
- [Model Context Protocol Specification 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28) — Model Context Protocol / Linux Foundation Agentic AI Foundation; updated/published 2026-07-28; 2026-07-28; accessed 2026-08-15.
- [modelcontextprotocol/go-sdk v1.7.0 release](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.7.0) — Model Context Protocol Go SDK maintainers; v1.7.0; 2026 support based on spec commit f817239f4d6b1efff2c4dfc2f7af85c985d73076; accessed 2026-08-15.
- [Model Context Protocol Security community project](https://modelcontextprotocol-security.io/) — Model Context Protocol Security / Cloud Security Alliance community; updated/published continuously updated; community material accessed 2026-08-15; accessed 2026-08-15.
- [Tool Poisoning and Metadata Attacks](https://modelcontextprotocol-security.io/ttps/tool-poisoning/) — Model Context Protocol Security community project; Community threat taxonomy current on access date; accessed 2026-08-15.
- [The 2026-07-28 Specification](https://blog.modelcontextprotocol.io/posts/2026-07-28/) — Model Context Protocol maintainers; updated/published 2026-07-28; 2026-07-28 release announcement; accessed 2026-08-15.
- [Authorization](https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization) — Model Context Protocol project; updated/published 2026-07-28; MCP revision 2026-07-28; accessed 2026-08-15.
- [Go SDK Streamable HTTP implementation](https://github.com/modelcontextprotocol/go-sdk/blob/v1.7.0/mcp/streamable.go) — Model Context Protocol project; updated/published 2026-07-28 release line; go-sdk v1.7.0; accessed 2026-08-15.
- [MCP Conformance Test Framework](https://github.com/modelcontextprotocol/conformance) — Model Context Protocol project; updated/published Repository state accessed 2026-08-15; Repository pins @modelcontextprotocol/conformance 0.2.0-alpha.11 at c321dd32035556e6769d3724a8ee97d87c3faaac for MCP 2026-07-28; accessed 2026-08-15.
- [Security Best Practices](https://modelcontextprotocol.io/docs/2026-07-28/tutorials/security/security_best_practices) — Model Context Protocol project; updated/published 2026-07-28; Guidance accompanying MCP revision 2026-07-28; accessed 2026-08-15.
- [Streamable HTTP](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http) — Model Context Protocol project; updated/published 2026-07-28; MCP revision 2026-07-28; accessed 2026-08-15.
- [stdio transport](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio) — Model Context Protocol project; updated/published 2026-07-28; MCP revision 2026-07-28; accessed 2026-08-15.
- [Adversarial Machine Learning: A Taxonomy and Terminology of Attacks and Mitigations](https://nvlpubs.nist.gov/nistpubs/ai/NIST.AI.100-2e2025.pdf) — NIST; updated/published 2025-03; NIST AI 100-2e2025; accessed 2026-08-15.
- [Lessons Learned from the Consortium: Tool Use in Agent Systems](https://www.nist.gov/news-events/news/2025/08/lessons-learned-consortium-tool-use-agent-systems) — NIST; updated/published 2025-08; NIST AI Safety Institute Consortium lessons; accessed 2026-08-15.
- [Secure Software Development Framework (SSDF) Version 1.1](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-218.pdf) — NIST; updated/published 2022-02; official copy accessed 2026-08-15; NIST SP 800-218 v1.1; accessed 2026-08-15.
- [Secure Software Development Framework Version 1.1](https://doi.org/10.6028/NIST.SP.800-218) — NIST; updated/published 2022-02-03; updated 2022-11-29; NIST SP 800-218, SSDF 1.1; accessed 2026-08-15.
- [Technical Blog: Strengthening AI Agent Hijacking Evaluations](https://www.nist.gov/news-events/news/2025/01/technical-blog-strengthening-ai-agent-hijacking-evaluations) — NIST Center for AI Standards and Innovation; updated/published 2025-01-17; CAISI agent-hijacking evaluation guidance; accessed 2026-08-15.
- [NIST Privacy Framework: A Tool for Improving Privacy through Enterprise Risk Management, Version 1.0](https://www.nist.gov/privacy-framework/privacy-framework) — National Institute of Standards and Technology; updated/published 2020-01-16; 1.0; accessed 2026-08-15.
- [AgentDojo: A Dynamic Environment to Evaluate Prompt Injection Attacks and Defenses for LLM Agents](https://papers.neurips.cc/paper_files/paper/2024/file/97091a5177d8dc64b1da8bf3e1f6fb54-Paper-Datasets_and_Benchmarks_Track.pdf) — NeurIPS 2024 Datasets and Benchmarks Track / ETH Zurich; updated/published 2024; NeurIPS 2024 paper; 97 tasks and 629 security cases; accessed 2026-08-15.
- [Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html) — OWASP Foundation; updated/published continuously maintained; page accessed 2026-08-15; current web revision on access date; accessed 2026-08-15.
- [Server Side Request Forgery Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html) — OWASP Foundation; updated/published continuously updated; accessed 2026-08-15; accessed 2026-08-15.
- [LLM06:2025 Excessive Agency](https://genai.owasp.org/llmrisk/llm062025-excessive-agency/) — OWASP GenAI Security Project; updated/published 2025; OWASP Top 10 for LLM Applications 2025, LLM06; accessed 2026-08-15.
- [OWASP Top 10 for Agentic Applications for 2026](https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/) — OWASP GenAI Security Project; updated/published 2025-12-09; 2026 edition; accessed 2026-08-15.
- [OWASP Top 10 for LLM Applications](https://genai.owasp.org/llm-top-10/) — OWASP GenAI Security Project; updated/published 2025 edition; 2026 resources published 2026-08-03; 2025/2026 project material; accessed 2026-08-15.
- [OWASP Top 10 for Agentic Applications 2026](https://genai.owasp.org/2025/12/09/owasp-top-10-for-agentic-applications-the-benchmark-for-agentic-security-in-the-age-of-autonomous-ai/) — OWASP GenAI Security Project, Agentic Security Initiative; updated/published 2025-12-09; 2026 edition; Agentic Top 10 2026; accessed 2026-08-15.
- [Open Source Security Foundation OSPS Baseline](https://baseline.openssf.org/) — OpenSSF; updated/published current edition accessed 2026-08-15; current baseline; accessed 2026-08-15.
- [OpenSSF Scorecard check documentation](https://github.com/ossf/scorecard/blob/main/docs/checks.md) — OpenSSF; updated/published continuously updated; main inspected 2026-08-15; accessed 2026-08-15.
- [Open Source Project Security Baseline](https://baseline.openssf.org/versions/2026-02-19.html) — OpenSSF OSPS Baseline SIG; updated/published 2026-02-19; v2026.02.19; accessed 2026-08-15.
- [Open Source Project Security Baseline](https://baseline.openssf.org/versions/2026-02-19) — OpenSSF Security Baseline SIG; updated/published 2026-02-19; v2026.02.19, current on 2026-08-15; accessed 2026-08-15.
- [AI-SLOP: Develop best current practices for Open Source maintainers](https://github.com/ossf/wg-vulnerability-disclosures/issues/178) — OpenSSF Vulnerability Disclosures Working Group; updated/published 2026-02-04; open work item on access date; Issue 178, emerging guidance rather than final policy; accessed 2026-08-15.
- [Securing Open Source in the Age of AI: A Practical Guide](https://openssf.org/resources/securing-open-source-in-the-age-of-ai-a-practical-guide/) — OpenSSF and CNCF; updated/published 2026; Practical guide current 2026-08-15; accessed 2026-08-15.
- [Metrics](https://opentelemetry.io/docs/concepts/signals/metrics/) — OpenTelemetry; updated/published current page accessed 2026-08-15; current documentation; accessed 2026-08-15.
- [Signals](https://opentelemetry.io/docs/concepts/signals/) — OpenTelemetry; updated/published 2026-03-10; current documentation; accessed 2026-08-15.
- [Pion WebRTC releases](https://github.com/pion/webrtc/releases/tag/v4.2.18) — Pion; updated/published 2026-07-27; v4.2.18; accessed 2026-08-15.
- [Failed DTLS certificate verification does not stop data channel communication](https://github.com/pion/webrtc/security/advisories/GHSA-74xm-qj29-cq8p) — Pion / GitHub Security Advisory; updated/published 2021-03-18; GHSA-74xm-qj29-cq8p; fixed in v3.0.15; accessed 2026-08-15.
- [Package github.com/pion/webrtc/v4](https://pkg.go.dev/github.com/pion/webrtc/v4#PeerConnection) — Pion WebRTC maintainers; updated/published Living module documentation accessed 2026-08-15; Repository dependency v4.2.18; documentation distinguishes Close and GracefulClose; accessed 2026-08-15.
- [SLSA specification](https://slsa.dev/spec/v1.2/) — SLSA; updated/published 2025-10-17; v1.2; accessed 2026-08-15.
- [Verifying artifacts](https://slsa.dev/spec/v1.2/verifying-artifacts) — SLSA; updated/published 2025-10-17; SLSA v1.2; accessed 2026-08-15.
- [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html) — Semantic Versioning; updated/published 2013-06-19; 2.0.0; accessed 2026-08-15.
- [Signing blobs with Cosign](https://docs.sigstore.dev/cosign/signing/signing_with_blobs/) — Sigstore; Cosign v3 workflow; accessed 2026-08-15.
- [Perfectly Reproducible, Verified Go Toolchains](https://go.dev/blog/rebuild) — The Go Blog / Go team; updated/published 2023-08-28; canonical Go reproducible-build guidance; accessed 2026-08-15.
- [Coverage profiling support for integration tests](https://go.dev/doc/build-cover) — The Go Project; Coverage integration support introduced in Go 1.20 and current for Go 1.26.6; accessed 2026-08-15.
- [Data Race Detector](https://go.dev/doc/articles/race_detector) — The Go Project; Official race-detector guidance as accessed 2026-08-15; accessed 2026-08-15.
- [Go Concurrency Patterns: Context](https://go.dev/blog/context) — The Go Project; updated/published 2014-07-29; Canonical first-party context guidance; accessed 2026-08-15.
- [Go Doc Comments](https://go.dev/doc/comment) — The Go Project; Official documentation as accessed 2026-08-15; discusses gofmt behavior introduced in Go 1.19; accessed 2026-08-15.
- [Go Fuzzing](https://go.dev/doc/security/fuzz/) — The Go Project; Native Go fuzzing guidance current for Go 1.26.6; accessed 2026-08-15.
- [Go Modules Reference](https://go.dev/ref/mod) — The Go Project; Current through Go 1.26; accessed 2026-08-15.
- [Go Toolchains](https://go.dev/doc/toolchain) — The Go Project; Go 1.21+ toolchain-selection semantics; accessed 2026-08-15.
- [Go Vulnerability Management](https://go.dev/doc/security/vuln/) — The Go Project; Go vulnerability database and govulncheck current service; accessed 2026-08-15.
- [Go Wiki: Go Code Review Comments](https://go.dev/wiki/CodeReviewComments#interfaces) — The Go Project; Interfaces section as accessed 2026-08-15; accessed 2026-08-15.
- [Organizing a Go module](https://go.dev/doc/modules/layout) — The Go Project; Living official Go module-layout guidance accessed with Go 1.26 current; accessed 2026-08-15.
- [Package context](https://pkg.go.dev/context) — The Go Project; updated/published Living standard-library documentation; rendered for the current Go release on access date; Go standard library as accessed 2026-08-15; includes Cause and WithoutCancel; accessed 2026-08-15.
- [Package names](https://go.dev/blog/package-names) — The Go Project; updated/published 2015-02-04; Canonical Go package-design article; accessed 2026-08-15.
- [Package net/http](https://pkg.go.dev/net/http@go1.26.5) — The Go Project; updated/published 2026-07 Go 1.26 documentation; Go 1.26 net/http Server; accessed 2026-08-15.
- [Package net/http: Server.Shutdown, Server.Close, and Server fields](https://pkg.go.dev/net/http#Server.Shutdown) — The Go Project; updated/published Living standard-library documentation; rendered for the current Go release on access date; Go standard library as accessed 2026-08-15; accessed 2026-08-15.
- [Package os/exec](https://pkg.go.dev/os/exec) — The Go Project; updated/published Living standard-library documentation; rendered for the current Go release on access date; Go standard library as accessed 2026-08-15; accessed 2026-08-15.
- [Package testing](https://pkg.go.dev/testing) — The Go Project; updated/published 2026-07-30 package documentation for Go 1.26.6; Go 1.26.6 testing package; accessed 2026-08-15.
- [Profile-guided optimization](https://go.dev/doc/pgo) — The Go Project; Official PGO guidance current for Go 1.26.6; accessed 2026-08-15.
- [Release History and Release Policy](https://go.dev/doc/devel/release) — The Go Project; updated/published 2026-08-13; Through Go 1.26.6 and Go 1.25.13; accessed 2026-08-15.
- [Traversal-resistant file APIs](https://go.dev/blog/osroot) — The Go Project; updated/published 2025-03-12; Go 1.24 os.Root introduction; accessed 2026-08-15.
- [govulncheck command documentation](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) — The Go Project; v1.7.0; accessed 2026-08-15.
- [GO-2026-5932: golang.org/x/crypto/openpgp is unmaintained and unsafe by design](https://pkg.go.dev/vuln/GO-2026-5932) — The Go Vulnerability Database; updated/published 2026; GO-2026-5932; accessed 2026-08-15.
- [Vulnerability Disclosure Policy](https://curl.se/dev/vuln-disclosure.html) — curl project; curl disclosure policy current on access date; accessed 2026-08-15.
