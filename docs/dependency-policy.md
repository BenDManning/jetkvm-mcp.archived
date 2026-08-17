# Dependency and build-input policy

Status: maintained.

This policy covers Go modules and tools, GitHub Actions, container images,
release executables, and the deliberately pinned MCP npm protocol inputs. It
keeps routine maintenance bounded without treating automated update proposals
as trusted changes.

## Tracked inputs

- The root `go.mod` tracks application modules plus Staticcheck and
  `govulncheck` through native Go `tool` directives. `go.sum` covers the
  selected source graph used to build those analyzers.
- `tools/go.mod` isolates release executables from the shipped module. It tracks
  GoReleaser, Cosign, and Syft through native Go `tool` directives and builds
  them with Go 1.26.6. The two module files are the version authorities.
- Workflow Actions use reviewed full commit SHAs with release-version comments.
  The Buildx release binary has an exact version and repository-recorded SHA-256
  verified before installation. QEMU, BuildKit, container frontend, builder, and
  runtime images use reviewable tag-plus-digest references. A tag explains the
  selected line; the digest fixes fetched bytes.
- The tag-only publisher downloads ORAS from the canonical
  `oras-project/oras` GitHub Release at the exact version and SHA-256 recorded in
  the finalization action. That action is the ORAS version/checksum authority;
  updates review the upstream release, license, advisories, archive checksum,
  OCI-layout copy behavior, authentication, resolve, and tag semantics together.
- Debian `ca-certificates` and `ffmpeg` remain distribution-selected packages.
  Their exact resolved versions are release evidence recorded from the built
  image and its SBOM, not reproducible package pins. The repository makes no
  reproducible-build claim.
- `testdata/mcp-gates/npm/package-lock.json` and `pins.json` are exact protocol
  review inputs. There is intentionally no general npm Dependabot entry: an MCP
  conformance or Inspector change must update the package lock, integrity,
  upstream commit and scenario inventory together through protocol review.

## Update cadence and grouping

Dependabot runs monthly and may keep at most one version-update pull request
open for each of these review queues:

1. application Go modules and tracked Go tools across `/` and `/tools`;
2. GitHub Actions; and
3. Docker inputs.

Updates never auto-merge. Relevant security advisories are handled promptly
outside the monthly cadence and grouped into one security-review queue per
ecosystem. Docker discovery is only an update signal:
Dependabot does not discover every input in every multi-stage Dockerfile, so a
Docker review must inspect all explicit frontend, builder, and runtime pins.
That inspection includes workflow inputs such as the QEMU and BuildKit images,
which Dependabot cannot infer as Dockerfile dependencies.
Go-family changes similarly update `go.mod`, CI, the release snapshot path, and
the container builder as one reviewed toolchain decision.

## Required review

For every proposed input change, the reviewer records in the pull request:

- canonical source and ownership, the old and new immutable identities, and why
  the dependency or tool is needed;
- upstream release notes or changelog and material behavior, compatibility, or
  deprecation changes;
- license and notice changes across newly selected direct and transitive code;
- relevant published advisories and the repository's exposure; and
- the applicable gates below and their results.

An advisory that is present at module level but unreachable from shipped source
gets an explicit risk disposition. It is not an automatic failure and does not
justify a blind transitive upgrade. Unknown source, ownership, license, or
security impact blocks the update until resolved.

| Changed input | Minimum local or pull-request evidence |
|---|---|
| Application Go module or root Go tool | `make tidy module-verify race vet staticcheck govulncheck verify` |
| Release tool | `make tools-tidy tools-module-verify release-tool-versions release-snapshot`; for ORAS, inspect its version/checksum and exercise the integrated non-publishing state-machine rehearsal |
| GitHub Action or downloaded workflow build helper | Inspect the full-SHA or checksum diff. Require all unprivileged build/verification logic to pass from the pull request without write or release credentials. An accepted OIDC-only release identity stage is exercised by a non-publishing default-branch rehearsal after merge and before its ticket closes. |
| Docker frontend, builder, runtime, or distro package resolution | `make container-verify`; inspect both platform results and record resolved image/package identities. |
| MCP npm protocol input | `make protocol-gates` plus deliberate review of the upstream commit, integrity, MCP revision, and scenario inventory. |

Use the stronger union of rows when one update changes more than one input
class. A snapshot, container check, or protocol gate is a dry rehearsal: it must
not push an image, create or mutate a release, upload an asset, or use publishing
credentials.

## Dry update rehearsal and rollback

Before merging a new update mechanism or materially changing a tracked input,
exercise one representative update on a branch or in a temporary copy. Change
the normal manifest and checksum files, run the applicable row above, inspect
the generated diff and evidence, and confirm that no publication occurred.
Record the exact command and result in the pull request or owning issue rather
than adding a parallel status document.

If validation or review fails before merge, restore the previous manifest,
checksum, SHA, or digest in that same pull request. If a merged input fails,
revert it through an ordinary reviewed pull request and invalidate unpublished
candidates built from it. Published tags and artifacts are immutable: a shipped
input defect is corrected by a new product patch release, never by replacing or
retagging existing artifacts.
