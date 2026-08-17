#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: scripts/verify-native-release.sh DIST_DIR" >&2
  exit 2
fi

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
dist_dir=$1
metadata="${dist_dir}/metadata.json"
artifacts="${dist_dir}/artifacts.json"
checksums="${dist_dir}/checksums.txt"

for required in "$metadata" "$artifacts" "$checksums"; do
  [[ -f $required ]] || { echo "missing release metadata: $required" >&2; exit 1; }
done

version=$(jq -er '.version | select(type == "string" and length > 0)' "$metadata")
commit=$(jq -er '.commit | select(test("^[0-9a-f]{40}$"))' "$metadata")
expected_version=${RELEASE_EXPECTED_VERSION:-$version}
expected_commit=${RELEASE_EXPECTED_COMMIT:-$(git -C "$repo_root" rev-parse HEAD)}

[[ $version == "$expected_version" ]] || { echo "release version $version does not match $expected_version" >&2; exit 1; }
[[ $commit == "$expected_commit" ]] || { echo "release commit $commit does not match $expected_commit" >&2; exit 1; }
jq -e --arg version "$version" --arg commit "$commit" \
  '.project_name == "jetkvm-mcp" and .version == $version and .commit == $commit' \
  "$metadata" >/dev/null

expected_names=()
for arch in amd64 arm64; do
  archive="jetkvm-mcp_${version}_linux_${arch}.tar.gz"
  expected_names+=("$archive" "${archive}.spdx.json")
done

mapfile -t checksum_names < <(awk 'NF == 2 && $1 ~ /^[0-9a-f]{64}$/ { print $2 }' "$checksums" | sort)
mapfile -t sorted_expected_names < <(printf '%s\n' "${expected_names[@]}" | sort)
[[ ${#checksum_names[@]} -eq ${#sorted_expected_names[@]} ]] || {
  echo "checksum manifest does not cover exactly the release archives and SBOMs" >&2
  exit 1
}
for index in "${!sorted_expected_names[@]}"; do
  [[ ${checksum_names[$index]} == "${sorted_expected_names[$index]}" ]] || {
    echo "unexpected checksum subject ${checksum_names[$index]}" >&2
    exit 1
  }
done
(cd "$dist_dir" && sha256sum --check --strict checksums.txt)

mapfile -t archive_subjects < <(jq -r '.[] | select(.type == "Archive") | [.name, .goos, .goarch] | @tsv' "$artifacts" | sort)
mapfile -t sbom_subjects < <(jq -r '.[] | select(.type == "SBOM") | .name' "$artifacts" | sort)
[[ ${#archive_subjects[@]} -eq 2 && ${archive_subjects[0]} == *$'\tlinux\tamd64' && ${archive_subjects[1]} == *$'\tlinux\tarm64' ]] || {
  echo "artifact metadata does not contain exactly Linux amd64 and arm64 archives" >&2
  exit 1
}
[[ ${#sbom_subjects[@]} -eq 2 ]] || { echo "artifact metadata does not contain two archive SBOMs" >&2; exit 1; }

temporary_dir=$(mktemp -d)
trap 'rm -rf -- "$temporary_dir"' EXIT

for arch in amd64 arm64; do
  archive_name="jetkvm-mcp_${version}_linux_${arch}.tar.gz"
  archive_path="${dist_dir}/${archive_name}"
  sbom_path="${archive_path}.spdx.json"
  [[ -f $archive_path && -f $sbom_path ]] || { echo "missing subject for linux/$arch" >&2; exit 1; }

  mapfile -t members < <(tar -tzf "$archive_path" | sort)
  expected_members=(LICENSE THIRD_PARTY_NOTICES.md jetkvm-mcp)
  [[ ${members[*]} == "${expected_members[*]}" ]] || {
    echo "$archive_name has unexpected members: ${members[*]}" >&2
    exit 1
  }

  extract_dir="${temporary_dir}/${arch}"
  mkdir -p "$extract_dir"
  tar -xzf "$archive_path" -C "$extract_dir"
  [[ -x ${extract_dir}/jetkvm-mcp ]] || { echo "$archive_name executable is not executable" >&2; exit 1; }
  cmp "$repo_root/LICENSE" "${extract_dir}/LICENSE"
  cmp "$repo_root/THIRD_PARTY_NOTICES.md" "${extract_dir}/THIRD_PARTY_NOTICES.md"

  build_info=$(go version -m "${extract_dir}/jetkvm-mcp")
  grep -F $'\tbuild\tvcs.revision='"$commit" <<<"$build_info" >/dev/null
  grep -F $'\tbuild\tGOOS=linux' <<<"$build_info" >/dev/null
  grep -F $'\tbuild\tGOARCH='"$arch" <<<"$build_info" >/dev/null
  grep -F ': go1.26.6' <<<"$build_info" >/dev/null
  grep -aF "$version" "${extract_dir}/jetkvm-mcp" >/dev/null

  runner=()
  verify_executable=false
  if [[ $arch == amd64 && $(uname -m) == x86_64 ]] || [[ $arch == arm64 && $(uname -m) == aarch64 ]]; then
    verify_executable=true
  elif [[ $arch == arm64 && -n ${RELEASE_ARM64_RUNNER:-} ]]; then
    runner=("$RELEASE_ARM64_RUNNER")
    verify_executable=true
  elif [[ $arch == arm64 && ${RELEASE_VERIFY_ARM64:-0} == 1 ]]; then
    # Hosted rehearsal registers arm64 user-mode emulation through binfmt_misc.
    verify_executable=true
  fi
  if [[ $verify_executable == true ]]; then
    actual_version=$("${runner[@]}" "${extract_dir}/jetkvm-mcp" --version)
    [[ $actual_version == "jetkvm-mcp $version" ]] || {
      echo "$archive_name reports unexpected version: $actual_version" >&2
      exit 1
    }
  fi

  archive_digest=$(sha256sum "$archive_path" | awk '{print $1}')
  jq -e --arg name "$archive_name" --arg digest "$archive_digest" '
    [.relationships[] | select(.spdxElementId == "SPDXRef-DOCUMENT" and .relationshipType == "DESCRIBES") | .relatedSpdxElement] as $described |
    .spdxVersion == "SPDX-2.3" and
    .dataLicense == "CC0-1.0" and
    .name == $name and
    ($described | length) == 1 and
    any(.packages[]; .SPDXID == $described[0] and .name == $name and .versionInfo == ("sha256:" + $digest) and any(.checksums[]; .algorithm == "SHA256" and .checksumValue == $digest))
  ' "$sbom_path" >/dev/null

done

[[ ${sbom_subjects[0]} == "jetkvm-mcp_${version}_linux_amd64.tar.gz.spdx.json" ]]
[[ ${sbom_subjects[1]} == "jetkvm-mcp_${version}_linux_arm64.tar.gz.spdx.json" ]]

echo "verified native release subjects for $version at $commit"
