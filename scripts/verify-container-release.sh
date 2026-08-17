#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: verify-container-release.sh OCI_ARCHIVE RELEASE_DIR" >&2
  exit 2
fi

archive=$1
release_dir=$2
[[ -f $archive ]] || { echo "missing OCI archive: $archive" >&2; exit 1; }
: "${CONTAINER_EXPECTED_VERSION:?CONTAINER_EXPECTED_VERSION is required}"
: "${CONTAINER_EXPECTED_SOURCE:?CONTAINER_EXPECTED_SOURCE is required}"
: "${CONTAINER_EXPECTED_REVISION:?CONTAINER_EXPECTED_REVISION is required}"
: "${CONTAINER_EXPECTED_CREATED:?CONTAINER_EXPECTED_CREATED is required}"
valid_semantic_version=false
if [[ $CONTAINER_EXPECTED_VERSION =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(.+))?$ ]]; then
  prerelease=${BASH_REMATCH[5]-}
  valid_semantic_version=true
  if [[ -n $prerelease ]]; then
    if [[ $prerelease == .* || $prerelease == *. || $prerelease == *..* ]]; then
      valid_semantic_version=false
    else
      IFS=. read -r -a prerelease_identifiers <<<"$prerelease"
      for identifier in "${prerelease_identifiers[@]}"; do
        if [[ ! $identifier =~ ^[0-9A-Za-z-]+$ ]] || [[ $identifier =~ ^0[0-9]+$ ]]; then
          valid_semantic_version=false
          break
        fi
      done
    fi
  fi
fi
[[ $valid_semantic_version == true ]] || {
  echo "container release version is not an exact semantic version: $CONTAINER_EXPECTED_VERSION" >&2
  exit 1
}

temporary_dir=$(mktemp -d)
cleanup() {
  rm -r -- "$temporary_dir"
}
trap cleanup EXIT

while IFS= read -r member; do
  case $member in
    /*|../*|*/../*) echo "unsafe OCI archive member: $member" >&2; exit 1 ;;
  esac
done < <(tar -tf "$archive")
tar -xf "$archive" -C "$temporary_dir"

layout=${temporary_dir}/oci-layout
outer_index=${temporary_dir}/index.json
[[ -f $layout && -f $outer_index ]] || { echo "OCI archive is missing its layout or index" >&2; exit 1; }
jq -e '.imageLayoutVersion == "1.0.0"' "$layout" >/dev/null
jq -e '
  .schemaVersion == 2 and
  (.manifests | length) == 1 and
  .manifests[0].mediaType == "application/vnd.oci.image.index.v1+json" and
  (.manifests[0].digest | test("^sha256:[0-9a-f]{64}$")) and
  (.manifests[0].size | type == "number")
' "$outer_index" >/dev/null

blob_path() {
  local digest=$1
  [[ $digest =~ ^sha256:([0-9a-f]{64})$ ]] || { echo "invalid OCI digest: $digest" >&2; exit 1; }
  printf '%s/blobs/sha256/%s' "$temporary_dir" "${BASH_REMATCH[1]}"
}

verify_descriptor() {
  local descriptor=$1
  local digest size blob actual_digest actual_size
  digest=$(jq -er '.digest' <<<"$descriptor")
  size=$(jq -er '.size' <<<"$descriptor")
  blob=$(blob_path "$digest")
  [[ -f $blob ]] || { echo "missing OCI blob: $digest" >&2; exit 1; }
  actual_digest=sha256:$(sha256sum "$blob" | awk '{print $1}')
  actual_size=$(stat -c %s "$blob")
  [[ $actual_digest == "$digest" && $actual_size == "$size" ]] || {
    echo "OCI descriptor does not match blob: $digest" >&2
    exit 1
  }
  printf '%s' "$blob"
}

index_descriptor=$(jq -c '.manifests[0]' "$outer_index")
image_index=$(verify_descriptor "$index_descriptor")
manifest_digest=$(jq -er '.digest' <<<"$index_descriptor")
jq -e '
  .schemaVersion == 2 and
  .mediaType == "application/vnd.oci.image.index.v1+json" and
  (.manifests | length) == 2 and
  ([.manifests[].platform | (.os + "/" + .architecture)] | sort) == ["linux/amd64", "linux/arm64"] and
  all(.manifests[]; .mediaType == "application/vnd.oci.image.manifest.v1+json")
' "$image_index" >/dev/null

declare -A platform_digests
for architecture in amd64 arm64; do
  descriptor=$(jq -cer --arg architecture "$architecture" '
    [.manifests[] | select(.platform.os == "linux" and .platform.architecture == $architecture)] |
    if length == 1 then .[0] else error("platform descriptor is not unique") end
  ' "$image_index")
  platform_digests[$architecture]=$(jq -er '.digest' <<<"$descriptor")
  manifest=$(verify_descriptor "$descriptor")
  cp "$manifest" "${release_dir}/image-manifest-linux-${architecture}.json"
  jq -e '
    .schemaVersion == 2 and
    .mediaType == "application/vnd.oci.image.manifest.v1+json" and
    .config.mediaType == "application/vnd.oci.image.config.v1+json" and
    (.layers | length) > 0 and
    all(.layers[]; (.digest | test("^sha256:[0-9a-f]{64}$")) and (.size | type == "number"))
  ' "$manifest" >/dev/null

  config_descriptor=$(jq -c '.config' "$manifest")
  config=$(verify_descriptor "$config_descriptor")
  jq -e \
    --arg architecture "$architecture" \
    --arg version "$CONTAINER_EXPECTED_VERSION" \
    --arg source "$CONTAINER_EXPECTED_SOURCE" \
    --arg revision "$CONTAINER_EXPECTED_REVISION" \
    --arg created "$CONTAINER_EXPECTED_CREATED" '
      .os == "linux" and
      .architecture == $architecture and
      .config.User == "10001:10001" and
      .config.Entrypoint == ["/usr/local/bin/jetkvm-mcp"] and
      .config.Labels["org.opencontainers.image.version"] == $version and
      .config.Labels["org.opencontainers.image.source"] == $source and
      .config.Labels["org.opencontainers.image.revision"] == $revision and
      .config.Labels["org.opencontainers.image.created"] == $created and
      .config.Labels["org.opencontainers.image.licenses"] == "MIT"
    ' "$config" >/dev/null

  while IFS= read -r layer_descriptor; do
    verify_descriptor "$layer_descriptor" >/dev/null
  done < <(jq -c '.layers[]' "$manifest")
  sbom=${release_dir}/linux-${architecture}.spdx.json
  [[ -f $sbom ]] || { echo "missing linux/${architecture} container SBOM" >&2; exit 1; }
  jq -e '
    (.spdxVersion | startswith("SPDX-2.")) and
    any(.packages[]; .name == "ca-certificates" and (.versionInfo | type == "string" and length > 0)) and
    any(.packages[]; .name == "ffmpeg" and (.versionInfo | type == "string" and length > 0))
  ' "$sbom" >/dev/null
done

cp "$image_index" "${release_dir}/image-manifest.json"
jq -n \
  --arg image "ghcr.io/bendmanning/jetkvm-mcp" \
  --arg manifest "$manifest_digest" \
  --arg amd64 "${platform_digests[amd64]}" \
  --arg arm64 "${platform_digests[arm64]}" \
  '{
    image: $image,
    manifest_digest: $manifest,
    platform_digests: {
      "linux/amd64": $amd64,
      "linux/arm64": $arm64
    }
  }' > "${release_dir}/manifest-digests.json"

jq -n \
  --arg version_tag "ghcr.io/bendmanning/jetkvm-mcp:${CONTAINER_EXPECTED_VERSION}" \
  --arg latest_tag "ghcr.io/bendmanning/jetkvm-mcp:latest" \
  '{
    version_tag: $version_tag,
    version_tag_immutable: true,
    latest_tag: $latest_tag,
    latest_after: "complete_stable_publication",
    published: false
  }' > "${release_dir}/publication-plan.json"

echo "verified container release subject ${manifest_digest}"
