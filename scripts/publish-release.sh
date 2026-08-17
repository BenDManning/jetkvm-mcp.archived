#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=release-publication-state.sh
source "$script_dir/release-publication-state.sh"

if [[ $# -ne 2 ]]; then
  echo "usage: scripts/publish-release.sh NATIVE_DIR CONTAINER_DIR" >&2
  exit 2
fi

native_dir=$1
container_dir=$2
: "${RELEASE_MODE:?RELEASE_MODE is required}"
: "${RELEASE_REF:?RELEASE_REF is required}"
: "${RELEASE_TAG:?RELEASE_TAG is required}"
: "${RELEASE_TRIGGER:?RELEASE_TRIGGER is required}"
: "${RELEASE_WORKFLOW:?RELEASE_WORKFLOW is required}"
: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
: "${GITHUB_SHA:?GITHUB_SHA is required}"
: "${GH_TOKEN:?GH_TOKEN is required}"

[[ $RELEASE_MODE == rehearse || $RELEASE_MODE == publish ]] || {
  echo "RELEASE_MODE must be rehearse or publish" >&2
  exit 1
}

identity="https://github.com/${RELEASE_WORKFLOW}@${RELEASE_REF}"
cosign_path=$(GOTOOLCHAIN=go1.26.6 go -C tools tool -n cosign)
PATH="$(dirname "$cosign_path"):${PATH}"

(cd "$native_dir" && sha256sum --check --strict checksums.txt)
manifest_digest=$(jq -er '.manifest_digest | select(test("^sha256:[0-9a-f]{64}$"))' "$container_dir/manifest-digests.json")
actual_manifest=sha256:$(sha256sum "$container_dir/image-manifest.json" | awk '{print $1}')
[[ $actual_manifest == "$manifest_digest" ]] || {
  echo "container manifest digest does not match its staged record" >&2
  exit 1
}
jq -e --arg version "ghcr.io/bendmanning/jetkvm-mcp:${RELEASE_TAG}" '
  .version_tag == $version and
  .version_tag_immutable == true and
  .latest_tag == "ghcr.io/bendmanning/jetkvm-mcp:latest" and
  .latest_after == "complete_stable_publication" and
  .published == false
' "$container_dir/publication-plan.json" >/dev/null

cosign verify-blob \
  --bundle "$native_dir/checksums.txt.sigstore.json" \
  --certificate-identity "$identity" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-github-workflow-repository "$GITHUB_REPOSITORY" \
  --certificate-github-workflow-ref "$RELEASE_REF" \
  --certificate-github-workflow-sha "$GITHUB_SHA" \
  --certificate-github-workflow-trigger "$RELEASE_TRIGGER" \
  "$native_dir/checksums.txt"

cosign verify-blob \
  --bundle "$container_dir/image-manifest.sigstore.json" \
  --certificate-identity "$identity" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-github-workflow-repository "$GITHUB_REPOSITORY" \
  --certificate-github-workflow-ref "$RELEASE_REF" \
  --certificate-github-workflow-sha "$GITHUB_SHA" \
  --certificate-github-workflow-trigger "$RELEASE_TRIGGER" \
  "$container_dir/image-manifest.json"

while read -r _ subject; do
  gh attestation verify "$native_dir/$subject" \
    --repo "$GITHUB_REPOSITORY" \
    --bundle "$native_dir/provenance.sigstore.json" \
    --cert-identity "$identity" \
    --source-ref "$RELEASE_REF" \
    --source-digest "$GITHUB_SHA" \
    --deny-self-hosted-runners
done < "$native_dir/checksums.txt"

gh attestation verify "$container_dir/image-manifest.json" \
  --repo "$GITHUB_REPOSITORY" \
  --bundle "$container_dir/provenance.sigstore.json" \
  --cert-identity "$identity" \
  --source-ref "$RELEASE_REF" \
  --source-digest "$GITHUB_SHA" \
  --deny-self-hosted-runners
for architecture in amd64 arm64; do
  gh attestation verify "$container_dir/image-manifest-linux-${architecture}.json" \
    --repo "$GITHUB_REPOSITORY" \
    --bundle "$container_dir/sbom-linux-${architecture}.sigstore.json" \
    --predicate-type https://spdx.dev/Document/v2.3 \
    --cert-identity "$identity" \
    --source-ref "$RELEASE_REF" \
    --source-digest "$GITHUB_SHA" \
    --deny-self-hosted-runners
done

scripts/verify-release-publication-state.sh

if [[ $RELEASE_MODE == rehearse ]]; then
  [[ $RELEASE_REF == refs/heads/main && $RELEASE_TRIGGER == workflow_dispatch ]]
  echo "verified complete integrated release rehearsal for $GITHUB_SHA"
  exit 0
fi

release_tag=$RELEASE_TAG
[[ $RELEASE_REF == "refs/tags/$release_tag" ]]
[[ $RELEASE_TRIGGER == push ]]
[[ ${GITHUB_REF_PROTECTED:-false} == true ]]
[[ ${GITHUB_ACTOR:-} == BenDManning ]]
[[ $release_tag =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]
[[ $(git cat-file -t "$RELEASE_REF") == tag ]]
[[ $(git rev-parse "$RELEASE_REF^{}") == "$GITHUB_SHA" ]]
: "${ORAS_PATH:?ORAS_PATH is required for publication}"

oras() {
  "$ORAS_PATH" "$@"
}

image=ghcr.io/bendmanning/jetkvm-mcp

temporary_dir=$(mktemp -d)
cleanup() {
  rm -r -- "$temporary_dir"
}
trap cleanup EXIT

qualification="$temporary_dir/physical-qualification.json"
jq -ce --arg commit "$GITHUB_SHA" --arg version "$release_tag" '
  [.entries[] |
    select(
      .evidenceClass == "physical_qualification" and
      .result == "pass" and
      .server.sourceRef == $commit and
      .server.version == $version
    )] |
  if length != 1 then
    error("exactly one passing physical qualification is required for this release")
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
' docs/compatibility/jetkvm-ledger.json > "$qualification"
qualification_id=$(jq -er '.id' "$qualification")

annotation=$(git for-each-ref --format='%(contents)' "$RELEASE_REF")
tag_notes="$temporary_dir/tag-release-notes.json"
printf '%s' "$annotation" | jq -e '
  (.summary | type == "string" and length > 0) and
  ([
    .compatibilityAndMigration,
    .securityRelevantFixes,
    .knownLimitations,
    .supersededVersions,
    .retractedVersions
  ] | all(.[]; type == "array" and all(.[]; type == "string" and length > 0)))
' >/dev/null
printf '%s' "$annotation" | jq -c . > "$tag_notes"

jq -n \
  --arg tag "$release_tag" \
  --arg ref "$RELEASE_REF" \
  --arg commit "$GITHUB_SHA" \
  --arg workflow "$RELEASE_WORKFLOW" \
  --arg image "$image@$manifest_digest" \
  --arg qualification "$qualification_id" \
  '{
    tag: $tag,
    ref: $ref,
    commit: $commit,
    workflow: $workflow,
    container: $image,
    physical_qualification: $qualification,
    immutable: true,
    published: true,
    latest_after: "complete_stable_publication"
  }' > "$container_dir/release-record.json"

notes="$temporary_dir/release-notes.md"
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
- Hardware compatibility: only the exact combination in `$qualification_id`; no broader model or firmware claim

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

assets=(
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
for asset in "${assets[@]}"; do
  [[ -f $asset ]] || { echo "missing release asset: $asset" >&2; exit 1; }
done

record_reconciliation() {
  local phase=$1
  local reason=$2
  mkdir -p release
  jq -n \
    --arg phase "$phase" \
    --arg reason "$reason" \
    --arg tag "$release_tag" \
    --arg commit "$GITHUB_SHA" \
    --arg image "$image@$manifest_digest" \
    '{
      status: "owner_reconciliation_required",
      phase: $phase,
      reason: $reason,
      tag: $tag,
      commit: $commit,
      image: $image,
      latest_must_not_be_retried_automatically: true
    }' > release/reconciliation.json || true
  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    {
      echo '## Owner reconciliation required'
      echo
      echo "The protected release reached an unknown outcome during \`$phase\` (\`$reason\`)."
      echo 'Do not rerun the workflow or automatically retry `latest`; inspect the immutable release and registry state independently.'
    } >> "$GITHUB_STEP_SUMMARY" || true
  fi
  echo "::error title=Owner reconciliation required::Unknown $phase outcome; do not retry automatically" >&2
  return 75
}

release_backend_preflight() {
  printf '%s' "$GH_TOKEN" | oras login ghcr.io --username "$GITHUB_ACTOR" --password-stdin || return 1
  gh api "repos/${GITHUB_REPOSITORY}/immutable-releases" \
    --jq '.enabled == true' | grep -Fx true >/dev/null || return 1

  local release_lookup registry_lookup
  if release_lookup=$(gh api "repos/${GITHUB_REPOSITORY}/releases/tags/${release_tag}" 2>&1); then
    echo "release version is already consumed: $release_tag" >&2
    return 1
  fi
  if ! grep -F 'HTTP 404' <<<"$release_lookup" >/dev/null; then
    echo "unable to confirm release-version availability" >&2
    return 1
  fi

  if registry_lookup=$(oras manifest fetch "$image:$release_tag" 2>&1); then
    echo "container version is already consumed: $image:$release_tag" >&2
    return 1
  fi
  if ! grep -Eqi 'not found|manifest unknown|NAME_UNKNOWN' <<<"$registry_lookup"; then
    echo "unable to confirm container-version availability" >&2
    return 1
  fi
}

release_backend_consume_version() {
  # The protected tag and single run attempt already consume the version. The
  # draft makes any later publication failure visibly incomplete.
  gh release create "$release_tag" --draft \
    --repo "$GITHUB_REPOSITORY" \
    --verify-tag \
    --latest=false \
    --title "$release_tag" \
    --notes-file "$notes" \
    "${assets[@]}"
}

release_backend_publish_exact_image() {
  oras cp --from-oci-layout \
    "$container_dir/jetkvm-mcp.oci.tar@$manifest_digest" \
    "$image:$release_tag" || return 1
  local remote_digest
  remote_digest=$(oras resolve "$image:$release_tag") || return 1
  [[ $remote_digest == "$manifest_digest" ]] || {
    echo "published container digest does not match its staged digest" >&2
    return 1
  }
}

release_backend_publish_release() {
  local publish_error release_state
  if publish_error=$(gh release edit "$release_tag" --draft=false \
    --repo "$GITHUB_REPOSITORY" \
    --verify-tag \
    --latest=false 2>&1); then
    return 0
  fi
  if release_state=$(gh release view "$release_tag" \
    --repo "$GITHUB_REPOSITORY" \
    --json isDraft,isImmutable 2>/dev/null); then
    if jq -e '.isDraft == true' <<<"$release_state" >/dev/null; then
      echo "release remains a draft after a definite publication failure" >&2
      return 1
    fi
    if jq -e '.isDraft == false and .isImmutable == true' <<<"$release_state" >/dev/null; then
      return 0
    fi
    record_reconciliation publish_release unconfirmed_immutable_state
    return $?
  fi
  record_reconciliation publish_release remote_response_unknown
}

release_backend_move_latest() {
  local latest_error
  echo "immutable release is confirmed; moving latest as the terminal remote operation"
  if latest_error=$(oras tag "$image@$manifest_digest" latest 2>&1); then
    return 0
  fi
  if grep -Eqi 'unauthorized|denied|forbidden|not found|manifest unknown|invalid|bad request' <<<"$latest_error"; then
    echo "latest was not moved because the registry rejected the request" >&2
    return 1
  fi
  record_reconciliation move_latest remote_response_unknown
}

publication_status=0
coordinate_release_publication || publication_status=$?
exit "$publication_status"
