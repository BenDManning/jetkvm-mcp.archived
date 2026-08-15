#!/usr/bin/env python3
"""Generate the decision-ready research report from validated workstream JSON."""

from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Any

import yaml


ROOT = Path(__file__).resolve().parent
RESULTS = ROOT / "results"
FIELDS = ROOT / "fields.yaml"
OUTLINE = ROOT / "outline.yaml"
REPORT = ROOT / "report.md"

# Required by the research-report skill for mixed flat/nested result compatibility.
CATEGORY_MAPPING = {
    "Basic Info": ["basic_info", "Basic Info"],
    "Technical Features": ["technical_features", "technical_characteristics", "Technical Features"],
    "Performance Metrics": ["performance_metrics", "performance", "Performance Metrics"],
    "Milestone Significance": ["milestone_significance", "milestones", "Milestone Significance"],
    "Business Info": ["business_info", "commercial_info", "Business Info"],
    "Competition & Ecosystem": ["competition_ecosystem", "competition", "Competition & Ecosystem"],
    "History": ["history", "History"],
    "Market Positioning": ["market_positioning", "market", "Market Positioning"],
}

SUMMARY_FIELDS = ("current_state", "priority", "disposition")
INTERNAL_FIELDS = {"_source_file", "uncertain"}


def slug(value: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", value.lower()).strip("-")


def short(value: Any, limit: int = 110) -> str:
    text = re.sub(r"\s+", " ", str(value)).strip()
    return text if len(text) <= limit else text[: limit - 1].rstrip() + "…"


def state_label(value: Any) -> str:
    text = str(value).strip().lower()
    for label in ("substantially_satisfied", "not_applicable", "satisfied", "partial", "gap", "unknown"):
        if text.startswith(label):
            return label
    return short(value, 32)


def priority_label(value: Any) -> str:
    match = re.search(r"\b(P[0-3]|no_action)\b", str(value))
    return match.group(1) if match else short(value, 24)


def uncertain_names(data: dict[str, Any]) -> set[str]:
    value = data.get("uncertain", [])
    return {str(v) for v in value} if isinstance(value, list) else set()


def is_uncertain(name: str, value: Any, data: dict[str, Any]) -> bool:
    if name in uncertain_names(data) or value is None or value == "":
        return True
    try:
        return "[uncertain]" in json.dumps(value, ensure_ascii=False)
    except TypeError:
        return "[uncertain]" in str(value)


def lookup(data: dict[str, Any], category: str, name: str) -> Any:
    if name in data:
        return data[name]
    candidates = [category, *CATEGORY_MAPPING.get(category, [])]
    for key in candidates:
        nested = data.get(key)
        if isinstance(nested, dict) and name in nested:
            return nested[name]
    stack: list[Any] = list(data.values())
    while stack:
        value = stack.pop()
        if isinstance(value, dict):
            if name in value:
                return value[name]
            stack.extend(value.values())
    return None


def inline(value: Any) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, (str, int, float)):
        return str(value).replace("\n", " ")
    return json.dumps(value, ensure_ascii=False, sort_keys=True)


def format_value(value: Any, depth: int = 0) -> list[str]:
    if isinstance(value, dict):
        lines: list[str] = []
        for key, nested in value.items():
            if isinstance(nested, (dict, list)):
                lines.append(f"- **{str(key).replace('_', ' ').title()}:**")
                lines.extend("  " + line for line in format_value(nested, depth + 1))
            else:
                lines.append(f"- **{str(key).replace('_', ' ').title()}:** {inline(nested)}")
        return lines
    if isinstance(value, list):
        if not value:
            return ["None."]
        if all(not isinstance(item, (dict, list)) for item in value):
            if len(value) <= 4 and sum(len(str(item)) for item in value) <= 240:
                return [", ".join(inline(item) for item in value)]
            return [f"- {inline(item)}" for item in value]
        lines = []
        for item in value:
            if isinstance(item, dict):
                parts = [f"**{str(k).replace('_', ' ').title()}:** {inline(v)}" for k, v in item.items()]
                lines.append("- " + " | ".join(parts))
            elif isinstance(item, list):
                lines.append("- " + ", ".join(inline(v) for v in item))
            else:
                lines.append(f"- {inline(item)}")
        return lines
    text = str(value).strip()
    if len(text) > 100:
        return [text]
    return [text]


def field_defs() -> tuple[list[dict[str, Any]], set[str]]:
    raw = yaml.safe_load(FIELDS.read_text(encoding="utf-8"))
    categories = raw["field_categories"]
    names = {field["name"] for category in categories for field in category["fields"]}
    return categories, names


def load_results() -> list[dict[str, Any]]:
    outline = yaml.safe_load(OUTLINE.read_text(encoding="utf-8"))
    order = {item["name"]: item["id"] for item in outline["items"]}
    records = []
    for path in RESULTS.glob("*.json"):
        data = json.loads(path.read_text(encoding="utf-8"))
        data["_source_file"] = path.name
        records.append(data)
    records.sort(key=lambda data: order.get(data.get("item_name", ""), 999))
    return records


def report_preamble(records: list[dict[str, Any]]) -> str:
    rows = []
    for i, data in enumerate(records, 1):
        rows.append(
            f"| {i} | [{data['item_name']}](#{slug(data['item_name'])}) | "
            f"{state_label(data.get('current_state', ''))} | {priority_label(data.get('priority', ''))} | "
            f"{short(data.get('disposition', ''), 18)} |"
        )
    matrix = "\n".join(rows)
    return f"""# Evidence-backed 2026 public-release baseline for `BenDManning/jetkvm-mcp`

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
{matrix}

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
"""


def generate() -> None:
    categories, defined_names = field_defs()
    records = load_results()
    lines = [report_preamble(records).rstrip()]
    for index, data in enumerate(records, 1):
        bits = [
            f"State: {state_label(data.get('current_state', ''))}",
            f"Priority: {priority_label(data.get('priority', ''))}",
            f"Decision: {short(data.get('disposition', ''), 18)}",
        ]
        suffix = " — " + " | ".join(bits) if bits else ""
        lines.append(f"{index}. [{data['item_name']}](#{slug(data['item_name'])}){suffix}")

    lines.extend(["", "## Detailed workstream evidence", ""])
    for index, data in enumerate(records, 1):
        lines.append(f"### {index}. {data['item_name']}")
        lines.append("")
        for category in categories:
            category_name = category["category"]
            rendered = []
            for field in category["fields"]:
                name = field["name"]
                if name == "uncertain":
                    continue
                value = lookup(data, category_name, name)
                if is_uncertain(name, value, data):
                    continue
                rendered.append((name, value))
            if not rendered:
                continue
            lines.append(f"#### {category_name.replace('_', ' ').title()}")
            lines.append("")
            for name, value in rendered:
                lines.append(f"**{name.replace('_', ' ').title()}**")
                lines.append("")
                lines.extend(format_value(value))
                lines.append("")

        extra: dict[str, Any] = {}
        nested_category_keys = {key for values in CATEGORY_MAPPING.values() for key in values}
        for name, value in data.items():
            if name in defined_names or name in INTERNAL_FIELDS or name in nested_category_keys:
                continue
            extra[name] = value
        if extra:
            lines.extend(["#### Other Info", ""])
            for name, value in extra.items():
                lines.extend([f"**{name.replace('_', ' ').title()}**", ""])
                lines.extend(format_value(value))
                lines.append("")

    sources: dict[tuple[str, str], dict[str, Any]] = {}
    for data in records:
        value = data.get("authoritative_sources", [])
        if not isinstance(value, list):
            continue
        for source in value:
            if not isinstance(source, dict):
                continue
            key = (str(source.get("title", "Untitled")), str(source.get("url", "")))
            sources.setdefault(key, source)
    lines.extend(["## Complete source registry", ""])
    for source in sorted(sources.values(), key=lambda s: (str(s.get("publisher", "")), str(s.get("title", "")))):
        title = str(source.get("title", "Untitled source"))
        url = str(source.get("url", ""))
        publisher = str(source.get("publisher", "Unknown publisher"))
        updated = str(source.get("publication_or_update_date", ""))
        version = str(source.get("version_or_revision", ""))
        accessed = str(source.get("access_date", "2026-08-15"))
        label = f"[{title}]({url})" if url else title
        metadata = [publisher]
        if updated and "[uncertain]" not in updated:
            metadata.append(f"updated/published {updated}")
        if version and "[uncertain]" not in version:
            metadata.append(version)
        metadata.append(f"accessed {accessed}")
        lines.append(f"- {label} — {'; '.join(metadata)}.")

    REPORT.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")


if __name__ == "__main__":
    generate()
