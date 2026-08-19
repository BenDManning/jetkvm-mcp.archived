#!/usr/bin/env bash

# prepare_release_materials validates candidate metadata and renders every
# mutation-free input consumed by the GitHub/GHCR publication adapter.
prepare_release_materials() (
  set -euo pipefail
  if [[ $# -ne 6 ]]; then
    echo "usage: prepare_release_materials NATIVE_DIR CONTAINER_DIR QUALIFICATION_LEDGER TAG_NOTES OUTPUT_DIR EVIDENCE_CLASS" >&2
    return 2
  fi

  local native_dir=$1
  local container_dir=$2
  local qualification_ledger=$3
  local tag_notes_source=$4
  local output_dir=$5
  local evidence_class=$6
  : "${RELEASE_REF:?RELEASE_REF is required}"
  : "${RELEASE_TAG:?RELEASE_TAG is required}"
  : "${RELEASE_WORKFLOW:?RELEASE_WORKFLOW is required}"
  : "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
  : "${GITHUB_SHA:?GITHUB_SHA is required}"

  mkdir -p "$output_dir"
  local qualification="$output_dir/physical-qualification.json"
  jq -ce \
    --arg commit "$GITHUB_SHA" \
    --arg version "$RELEASE_TAG" \
    --arg evidence_class "$evidence_class" '
    [.entries[] |
      select(
        .evidenceClass == $evidence_class and
        .result == "pass" and
        .server.sourceRef == $commit and
        .server.version == $version
      )] |
    if length != 1 then
      error("exactly one matching passing qualification is required for this release")
    else
      .[0]
    end |
    select(
      (.id | type == "string" and length > 0) and
      (.authorizationReference | type == "string" and length > 0) and
      (.approvedUTCWindow | type == "string" and length > 0) and
      (.observedOn | type == "string" and length > 0) and
      (.jetkvm.model | type == "string" and length > 0) and
      (.jetkvm.firmwareVersion | type == "string" and length > 0) and
      (.runtime.os | type == "string" and length > 0) and
      (.runtime.architecture | type == "string" and length > 0) and
      (.ffmpeg.identity | type == "string" and length > 0) and
      (.mcp.transport | type == "string" and length > 0) and
      (.mcp.client | type == "string" and length > 0) and
      (.attachedHost.fixture | type == "string" and length > 0) and
      (.attachedHost.os | type == "string" and length > 0) and
      (.attachedHost.architecture | type == "string" and length > 0) and
      (.checks | type == "array" and length > 0 and all(.[]; type == "string" and length > 0)) and
      (.limitations | type == "array" and length > 0 and all(.[]; type == "string" and length > 0))
    )
  ' "$qualification_ledger" > "$qualification"
  local qualification_id
  qualification_id=$(jq -er '.id' "$qualification")

  local tag_notes="$output_dir/tag-release-notes.json"
  jq -ce '
    select(
      (.summary | type == "string" and length > 0) and
      ([
        .compatibilityAndMigration,
        .securityRelevantFixes,
        .knownLimitations,
        .supersededVersions,
        .retractedVersions
      ] | all(.[]; type == "array" and all(.[]; type == "string" and length > 0)))
    )
  ' "$tag_notes_source" > "$tag_notes"

  local manifest_digest
  manifest_digest=$(jq -er '.manifest_digest | select(test("^sha256:[0-9a-f]{64}$"))' "$container_dir/manifest-digests.json")
  local actual_manifest
  actual_manifest=sha256:$(sha256sum "$container_dir/image-manifest.json" | awk '{print $1}')
  [[ $actual_manifest == "$manifest_digest" ]] || {
    echo "container manifest digest does not match its staged record" >&2
    return 1
  }

  local image=ghcr.io/bendmanning/jetkvm-mcp
  local published=false
  if [[ $evidence_class == physical_qualification ]]; then
    published=true
  fi
  jq -n \
    --arg tag "$RELEASE_TAG" \
    --arg ref "$RELEASE_REF" \
    --arg commit "$GITHUB_SHA" \
    --arg workflow "$RELEASE_WORKFLOW" \
    --arg image "$image@$manifest_digest" \
    --arg qualification "$qualification_id" \
    --argjson published "$published" \
    '{
      tag: $tag,
      ref: $ref,
      commit: $commit,
      workflow: $workflow,
      container: $image,
      physical_qualification: $qualification,
      immutable: true,
      published: $published,
      latest_after: "complete_stable_publication"
    }' > "$container_dir/release-record.json"

  local notes="$output_dir/release-notes.md"
  {
    jq -r '
      def section($heading; $items):
        "## " + $heading + "\n\n" +
        (if ($items | length) == 0 then "- None." else ($items | map("- " + .) | join("\n")) end) + "\n";
      "## Changes\n\n" + .summary + "\n\n" +
      section("Compatibility and migration"; .compatibilityAndMigration) + "\n" +
      section("Security-relevant fixes"; .securityRelevantFixes) + "\n" +
      section("Known limitations"; .knownLimitations) + "\n" +
      section("Superseded versions"; .supersededVersions) + "\n" +
      section("Retracted versions"; .retractedVersions)
    ' "$tag_notes"
    jq -r '
      "\n## Physical qualification\n\n" +
      "- Evidence: `" + .id + "`\n" +
      "- Authorization: `" + .authorizationReference + "` during `" + .approvedUTCWindow + "`\n" +
      "- Qualification date: `" + .observedOn + "`\n" +
      "- JetKVM: `" + .jetkvm.model + "`, firmware `" + .jetkvm.firmwareVersion + "`\n" +
      "- Server: `" + .server.version + "` at `" + .server.sourceRef + "`\n" +
      "- Runtime: `" + .runtime.os + "/" + .runtime.architecture + "`\n" +
      "- FFmpeg: `" + .ffmpeg.identity + "`\n" +
      "- MCP: `" + .mcp.transport + "` with `" + .mcp.client + "`\n" +
      "- Attached-host fixture: `" + .attachedHost.fixture + "`, `" + .attachedHost.os + "/" + .attachedHost.architecture + "`\n" +
      "- Completed checks: " + (.checks | join(", ")) + "\n" +
      "- Qualification limits:\n" + (.limitations | map("  - " + .) | join("\n"))
    ' "$qualification"
    printf '\n## Artifact digests\n\n'
    while read -r digest subject; do
      printf -- '- `%s`: `%s`\n' "$subject" "sha256:$digest"
    done < "$native_dir/checksums.txt"
    printf -- '- `%s`: `%s`\n\n' "$image@$manifest_digest" "$manifest_digest"
    cat <<EOF
## Supported release surface

- MCP revision: \`2026-07-28\`
- Source minimum: Go 1.25.13; release toolchain: Go 1.26.6
- Native: Linux amd64 and arm64
- Container: Linux amd64 and arm64, with FFmpeg, running as UID/GID 10001
- Hardware compatibility: only the exact combination in \`$qualification_id\`; no broader model or firmware claim

## Verification

Release identity:

- Protected tag: \`$RELEASE_REF\`
- Commit: \`$GITHUB_SHA\`
- Workflow: \`$RELEASE_WORKFLOW\`
- Container: \`$image@$manifest_digest\`

Verify downloaded native subjects with \`checksums.txt\`, then constrain Sigstore verification to:

\`\`\`sh
repo=$GITHUB_REPOSITORY
workflow=$RELEASE_WORKFLOW
release_ref=$RELEASE_REF
release_commit=$GITHUB_SHA
sha256sum --check --strict checksums.txt
cosign verify-blob --bundle checksums.txt.sigstore.json --certificate-identity "https://github.com/\${workflow}@\${release_ref}" --certificate-oidc-issuer https://token.actions.githubusercontent.com --certificate-github-workflow-repository "\$repo" --certificate-github-workflow-ref "\$release_ref" --certificate-github-workflow-sha "\$release_commit" checksums.txt
gh attestation verify SUBJECT --repo "\$repo" --bundle provenance.sigstore.json --cert-identity "https://github.com/\${workflow}@\${release_ref}" --source-ref "\$release_ref" --source-digest "\$release_commit" --deny-self-hosted-runners
\`\`\`

Pull the immutable container and verify its downloaded manifest evidence with:

\`\`\`sh
docker pull $image@$manifest_digest
cosign verify-blob --bundle image-manifest.sigstore.json --certificate-identity "https://github.com/\${workflow}@\${release_ref}" --certificate-oidc-issuer https://token.actions.githubusercontent.com --certificate-github-workflow-repository "\$repo" --certificate-github-workflow-ref "\$release_ref" --certificate-github-workflow-sha "\$release_commit" image-manifest.json
gh attestation verify image-manifest.json --repo "\$repo" --bundle provenance.sigstore.json --cert-identity "https://github.com/\${workflow}@\${release_ref}" --source-ref "\$release_ref" --source-digest "\$release_commit" --deny-self-hosted-runners
\`\`\`
EOF
  } > "$notes"

  local assets=(
    "$native_dir"/*.tar.gz
    "$native_dir"/*.spdx.json
    "$native_dir/checksums.txt"
    "$native_dir/checksums.txt.sigstore.json"
    "$native_dir/provenance.sigstore.json"
    "$container_dir/image-manifest.json"
    "$container_dir/image-manifest-linux-amd64.json"
    "$container_dir/image-manifest-linux-arm64.json"
    "$container_dir/linux-amd64.spdx.json"
    "$container_dir/linux-arm64.spdx.json"
    "$container_dir/manifest-digests.json"
    "$container_dir/release-record.json"
    "$container_dir/image-manifest.sigstore.json"
    "$container_dir/provenance.sigstore.json"
    "$container_dir/sbom-linux-amd64.sigstore.json"
    "$container_dir/sbom-linux-arm64.sigstore.json"
  )
  local asset
  for asset in "${assets[@]}"; do
    [[ -f $asset ]] || {
      echo "missing release asset: $asset" >&2
      return 1
    }
  done
  printf '%s\n' "${assets[@]}" > "$output_dir/release-assets.txt"
)
