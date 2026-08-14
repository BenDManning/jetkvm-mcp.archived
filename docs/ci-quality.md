# CI quality and toolchain matrix

GitHub Actions workflow `.github/workflows/ci.yml` is the single canonical CI
authority for pull requests and `main`. Every CI gate has the same local Make
target; no mirror or second workflow is authoritative.

## Toolchain lanes

| Lane | Toolchain | Command | Purpose |
|---|---:|---|---|
| Minimum | Go 1.25.0 with `GOTOOLCHAIN=local` | `GOTOOLCHAIN=go1.25.0 make ci-minimum` | Formatting, tidy, module checksums, tests, vet, and bounded fuzz smoke against the module's declared minimum |
| Release | Go 1.26.6 | `GOTOOLCHAIN=go1.26.6 make ci-quality` | Race, vet, pinned analyzers, vulnerability analysis, fuzz smoke, coverage evidence, and release targets |
| MCP protocol | Go 1.26.6 / Node 22.22.0 | `GOTOOLCHAIN=go1.26.6 make protocol-gates` | Pinned conformance and Inspector scenarios from `testdata/mcp-gates/pins.json` |
| Container | Docker Buildx | `make container-verify` | amd64/arm64 binary architecture and multi-platform image build |

`GOTOOLCHAIN=local` in CI prevents the minimum lane from silently downloading a
newer compiler. A local machine may use `GOTOOLCHAIN=go1.25.0` to select the
same compiler before running `make ci-minimum`.

## Local gate commands

- `make format`: non-mutating bounded `gofmt` comparison.
- `make tidy`: non-mutating `go mod tidy -diff` with workspace mode disabled.
- `make module-verify`: module checksum verification.
- `make test`, `make race`, `make vet`: repository tests, race detector, and vet.
- `make staticcheck`: Staticcheck `v0.7.0`.
- `make govulncheck`: govulncheck `v1.7.0`.
- `make fuzz-smoke`: exact bounded fuzz inventory and one-second target runs.
- `make coverage`: atomic coverage profile and `go tool cover -func` summary in
  `COVERAGE_DIR` (default `/tmp/jetkvm-mcp-coverage`).
- `make verify`: tests, vet, and supported release cross-builds.
- `make protocol-gates`: build and run the pinned MCP conformance/Inspector
  harness. `MCP_GATE_SERVER` and `MCP_GATE_DIR` may redirect temporary outputs.
- `make container-verify`: run the same Buildx amd64/arm64 binary architecture
  checks and multi-platform image build as the container CI lane.

`cmd/jetkvm-ci-check` owns format/tidy checks. Tidy rejects symlinked module
files, applies a two-minute deadline, and terminates its process group on Unix.
Its tests intentionally provide an unformatted Go file, a stale `go.mod`,
symlinked module files, and a timed-out descendant and require every gate to
fail safely.
`internal/cipolicy` verifies the workflow-to-Make mapping, both Go lanes,
analyzer pins, read-only workflow permissions, exactly one `.yml` or `.yaml`
workflow, container-lane parity, and absence of a coverage percentage policy.

## Coverage and evidence

CI uploads `coverage.out` and `coverage.txt` as the `go-coverage` artifact.
Coverage is review evidence, not an arbitrary pass percentage. A future
threshold requires repository history and a separately reviewed policy change.
MCP gate artifacts remain a separate sanitized artifact.

## Security boundary

PR jobs use only `contents: read`; no device, release, deployment, package,
firmware, or repository-write credential is configured. Gates use fixtures and
loopback transports. They do not contact JetKVM hardware, publish releases,
push mirrors, deploy services, or build public badges.
