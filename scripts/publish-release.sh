#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=release-publication-state.sh
source "$script_dir/release-publication-state.sh"
# shellcheck source=release-publication-materials.sh
source "$script_dir/release-publication-materials.sh"

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

release_tag=$RELEASE_TAG
image=ghcr.io/bendmanning/jetkvm-mcp
temporary_dir=$(mktemp -d)
cleanup() {
  rm -r -- "$temporary_dir"
}
trap cleanup EXIT

qualification_ledger=docs/compatibility/jetkvm-ledger.json
tag_notes_source="$temporary_dir/tag-release-notes-source.json"
evidence_class=physical_qualification
if [[ $RELEASE_MODE == rehearse ]]; then
  [[ $RELEASE_REF == refs/heads/main && $RELEASE_TRIGGER == workflow_dispatch ]]
  qualification_ledger="$temporary_dir/rehearsal-qualification-ledger.json"
  evidence_class=release_rehearsal
  jq -n \
    --arg commit "$GITHUB_SHA" \
    --arg version "$release_tag" \
    --arg observed_on "$(git show -s --format=%cs "$GITHUB_SHA")" \
    '{entries: [{
      id: ("release-rehearsal-" + ($commit[0:12])),
      evidenceClass: "release_rehearsal",
      result: "pass",
      authorizationReference: "not-applicable-nonpublishing-rehearsal",
      approvedUTCWindow: "not-applicable-nonpublishing-rehearsal",
      observedOn: $observed_on,
      jetkvm: {model: "not-observed", firmwareVersion: "not-observed"},
      server: {sourceRef: $commit, version: $version},
      runtime: {os: "not-observed", architecture: "not-observed"},
      ffmpeg: {identity: "not-observed"},
      mcp: {transport: "not-observed", client: "not-observed"},
      attachedHost: {fixture: "not-observed", os: "not-observed", architecture: "not-observed"},
      checks: ["mutation-free release material preparation"],
      limitations: ["Synthetic workflow fixture only; no hardware access or physical qualification occurred."]
    }]}' > "$qualification_ledger"
  jq -n '{
    summary: "Non-publishing integrated release rehearsal.",
    compatibilityAndMigration: [],
    securityRelevantFixes: [],
    knownLimitations: ["Synthetic release notes; no release was published."],
    supersededVersions: [],
    retractedVersions: []
  }' > "$tag_notes_source"
else
  [[ $RELEASE_REF == "refs/tags/$release_tag" ]]
  [[ $RELEASE_TRIGGER == push ]]
  [[ ${GITHUB_REF_PROTECTED:-false} == true ]]
  [[ ${GITHUB_ACTOR:-} == BenDManning ]]
  [[ $release_tag =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]
  [[ $(git cat-file -t "$RELEASE_REF") == tag ]]
  [[ $(git rev-parse "$RELEASE_REF^{}") == "$GITHUB_SHA" ]]
  git for-each-ref --format='%(contents)' "$RELEASE_REF" > "$tag_notes_source"
fi

prepare_release_materials \
  "$native_dir" \
  "$container_dir" \
  "$qualification_ledger" \
  "$tag_notes_source" \
  "$temporary_dir/materials" \
  "$evidence_class"

notes="$temporary_dir/materials/release-notes.md"
mapfile -t assets < "$temporary_dir/materials/release-assets.txt"

if [[ $RELEASE_MODE == rehearse ]]; then
  echo "verified complete integrated release rehearsal for $GITHUB_SHA"
  exit 0
fi

: "${ORAS_PATH:?ORAS_PATH is required for publication}"
oras() {
  "$ORAS_PATH" "$@"
}

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
