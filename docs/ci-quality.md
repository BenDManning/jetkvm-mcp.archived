# CI quality and toolchain matrix

GitHub Actions workflow `.github/workflows/ci.yml` is the single canonical CI
authority for pull requests and `main`. Every CI gate has the same local Make
target; no mirror or second workflow is authoritative.

## Accepted public v1 target

Public v1 uses four stable required checks with non-overlapping claims:

1. **Go quality**, on Go 1.26.6: format, tidy, module integrity, one
   race-enabled full test run that also emits coverage, vet, Staticcheck,
   reachable-source `govulncheck`, one-second fuzz-target smoke, and Linux
   native cross-builds.
2. **Minimum Go**, on Go 1.25.13 with `GOTOOLCHAIN=local`: one ordinary full
   test/build pass, without repeating formatting, analyzers, fuzzing, or
   coverage.
3. **MCP protocol**: build the server once, then run the pinned applicable
   official scenarios and isolated Inspector smoke over stdio and HTTP. This is
   an applicable-scenario gate, not conformance certification.
4. **Container**: build Linux amd64 and arm64 once and smoke each final image for
   its product version, FFmpeg availability, non-root identity, and offline
   configuration validation.

Coverage is diagnostic evidence without a percentage gate. CI does not repeat
plain tests, vet, protocol builds, real-FFmpeg integration, or cross-builds when
an earlier execution already establishes the same claim. Long scheduled
fuzzing requires a concrete failure or adoption-driven decision.

Behavior and evidence gates retain the complete MCP manifest and transport
parity, pinned protocol scenarios and upstream integrity, bounded fuzz inventory
and privacy-safe corpora, one real-FFmpeg integration run, and compatibility
ledger validation of schema, provenance, uniqueness, and claim limits. Ledger
tests do not freeze the number of historical entries or exact commit IDs. Tests
do not enforce ADR counts, document headings, index wording, threat-model prose,
or other administrative document shape; those documents receive ordinary
review.

Public conversion is an owner-controlled sequence. After visibility changes
and before further changes are accepted, the repository must enable and read
back a `main` ruleset requiring pull requests, the four stable checks above,
conversation resolution, blocked force-push and deletion, zero approving
reviews, and administrator enforcement. Release tags matching `v*` must be
protected from update and deletion. A narrow logged owner bypass is reserved
for emergencies and requires a follow-up GitHub issue and review. The project
does not require signed commits, CODEOWNERS, or a ceremonial solo approval.

Every public workflow pins each Action to a reviewed full commit SHA and is
covered by an exact allowlist. Workflow permissions default to `contents: read`;
write, package, and OIDC permissions exist only on the release job that requires
them. Pull-request CI uses `pull_request`, GitHub-hosted runners, no secrets or
release credentials, and checkout with credential persistence disabled;
`pull_request_target` is prohibited. Concurrency cancels superseded PR runs but
never `main`, tag, or release runs. Sanitized coverage and protocol evidence is
retained for 30 days. Release jobs do not consume caches produced by untrusted
pull requests.

## Toolchain lanes

| Lane | Toolchain | Command | Purpose |
|---|---:|---|---|
| Minimum Go | Go 1.25.13 with `GOTOOLCHAIN=local` | `GOTOOLCHAIN=go1.25.13 make ci-minimum` | One ordinary full test and server build against the module's declared minimum |
| Go quality | Go 1.26.6 | `GOTOOLCHAIN=go1.26.6 make ci-quality` | Format, tidy, module integrity, one race/coverage suite, vet, pinned analyzers, reachable vulnerabilities, fuzz smoke, and Linux native cross-builds |
| MCP protocol | Go 1.26.6 / Node 22.22.0 | `GOTOOLCHAIN=go1.26.6 make protocol-gates` | One server build followed by pinned conformance and Inspector scenarios from `testdata/mcp-gates/pins.json` |
| Container | Docker Buildx | `make container-verify` | One final amd64 image and one final arm64 image, each with runtime smokes |

`GOTOOLCHAIN=local` in CI prevents the minimum lane from silently downloading a
newer compiler. A local machine may use `GOTOOLCHAIN=go1.25.13` to select the
same compiler before running `make ci-minimum`.

## Local gate commands

- `make format`: non-mutating `gofmt -l` check that fails when Go files require
  formatting.
- `make tidy`: non-mutating `go mod tidy -diff` with workspace mode disabled.
- `make module-verify`: module checksum verification.
- `make test`, `make race`, `make vet`: repository tests, race detector, and vet.
  The race target owns the opt-in real FFmpeg encode/decode integration; the
  ordinary test target skips it so minimum-Go does not repeat that evidence.
- `make race-coverage`: the single race-enabled full suite plus diagnostic
  coverage in `COVERAGE_DIR`.
- `make staticcheck`: the Staticcheck version tracked by the root Go module.
- `make govulncheck`: the govulncheck version tracked by the root Go module.
- `make release-tool-versions`: build and report the GoReleaser, Cosign, and
  Syft versions tracked by the isolated `tools` Go module.
- `make release-snapshot`: run the tracked GoReleaser with Go 1.26.6 in
  non-publishing snapshot mode.
- `make fuzz-smoke`: exact bounded fuzz inventory and one-second target runs.
- `make coverage`: atomic coverage profile and `go tool cover -func` summary in
  `COVERAGE_DIR` (default `/tmp/jetkvm-mcp-coverage`).
- `make cross-build-linux`: native Linux amd64 and arm64 server cross-builds.
- `make ci-minimum`: one ordinary full suite and server build, with the caller
  responsible for selecting Go 1.25.13 and `GOTOOLCHAIN=local` in CI.
- `make ci-quality`: the complete non-overlapping current-Go quality lane.
- `make verify`: tests, vet, and supported release cross-builds.
- `make protocol-gates`: build and run the pinned MCP conformance/Inspector
  harness. `MCP_GATE_SERVER` and `MCP_GATE_DIR` may redirect temporary outputs.
- `make container-verify`: build each final platform image once and smoke its
  injected version, FFmpeg executable, UID/GID, and offline configuration
  validation.

## Coverage and evidence

CI uploads the race suite's `coverage.out` and `coverage.txt` as the
`go-coverage` artifact. Coverage is review evidence, not an arbitrary pass
percentage. A future threshold requires repository history and a separately
reviewed policy change. MCP gate artifacts remain a separate sanitized
artifact. Both artifacts retain for 30 days.

## Security boundary

PR jobs use only `contents: read`; no device, release, deployment, package,
firmware, or repository-write credential is configured. Gates use fixtures and
loopback transports. They do not contact JetKVM hardware, publish releases,
push mirrors, deploy services, or build public badges.

Dependency and executable-build-input review is defined in
[`dependency-policy.md`](dependency-policy.md).
