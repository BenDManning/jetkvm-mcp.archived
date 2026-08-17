#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 7 ]]; then
  echo "usage: smoke-container.sh IMAGE PLATFORM VERSION SOURCE REVISION CREATED SPDX_SBOM" >&2
  exit 2
fi

image=$1
platform=$2
version=$3
source=$4
revision=$5
created=$6
sbom=$7
architecture=${platform#linux/}
repository_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
config_path=${repository_root}/testdata/container/config.yaml
temporary_directory=$(mktemp -d)
container_id=
filesystem_container_id=

cleanup() {
  if [[ -n ${container_id} ]]; then
    docker logs "${container_id}" >&2 || true
    docker rm --force "${container_id}" >/dev/null 2>&1 || true
  fi
  if [[ -n ${filesystem_container_id} ]]; then
    docker rm --force "${filesystem_container_id}" >/dev/null 2>&1 || true
  fi
  rm -r -- "${temporary_directory}"
}
trap cleanup EXIT

inspect() {
  docker image inspect --format "$1" "${image}"
}

label() {
  inspect "{{ index .Config.Labels \"$1\" }}"
}

[[ $(inspect '{{.Architecture}}') == "${architecture}" ]]
[[ $(inspect '{{.Config.User}}') == "10001:10001" ]]
[[ $(inspect '{{json .Config.Entrypoint}}') == '["/usr/local/bin/jetkvm-mcp"]' ]]
[[ $(label org.opencontainers.image.source) == "${source}" ]]
[[ $(label org.opencontainers.image.revision) == "${revision}" ]]
[[ $(label org.opencontainers.image.version) == "${version}" ]]
[[ $(label org.opencontainers.image.licenses) == "MIT" ]]
[[ $(label org.opencontainers.image.created) == "${created}" ]]

filesystem_container_id=$(docker create "${image}")
docker cp "${filesystem_container_id}:/usr/local/bin/jetkvm-mcp" "${temporary_directory}/jetkvm-mcp"
docker rm "${filesystem_container_id}" >/dev/null
filesystem_container_id=
binary_identity=$(file --brief "${temporary_directory}/jetkvm-mcp")
case ${architecture} in
  amd64) [[ ${binary_identity} == *"x86-64"* ]] ;;
  arm64) [[ ${binary_identity} == *"ARM aarch64"* ]] ;;
  *) echo "unsupported container architecture: ${architecture}" >&2; exit 1 ;;
esac
printf 'embedded binary: %s\n' "${binary_identity}"

if inspect '{{range .Config.Env}}{{println .}}{{end}}' | grep -Eiq '(^|_)(password|token|secret|credential)='; then
  echo "container image embeds a credential-shaped environment variable" >&2
  exit 1
fi

[[ $(docker run --rm "${image}" --version) == "jetkvm-mcp ${version}" ]]
[[ $(docker run --rm --entrypoint id "${image}" -u) == "10001" ]]
[[ $(docker run --rm --entrypoint id "${image}" -g) == "10001" ]]

docker run --rm --entrypoint test "${image}" ! -e /config.yaml
container_packages=$(docker run --rm --entrypoint cat "${image}" /usr/share/jetkvm-mcp/container-packages.txt)
ffmpeg_identity=$(docker run --rm --entrypoint sed "${image}" -n 1p /usr/share/jetkvm-mcp/ffmpeg-version.txt)
printf '%s\n%s\n' "${container_packages}" "${ffmpeg_identity}"

package_version() {
  awk -v package="$1" '$1 == package { print $2 }' <<<"${container_packages}"
}

sbom_package_version() {
  jq -er --arg package "$1" \
    '[.packages[] | select(.name == $package) | .versionInfo] | unique | if length == 1 then .[0] else error("package identity is not unique") end' \
    "${sbom}"
}

jq -e '.spdxVersion | startswith("SPDX-2.")' "${sbom}" >/dev/null
[[ -n $(package_version ca-certificates) ]]
[[ -n $(package_version ffmpeg) ]]
[[ $(sbom_package_version ca-certificates) == "$(package_version ca-certificates)" ]]
[[ $(sbom_package_version ffmpeg) == "$(package_version ffmpeg)" ]]

docker run --rm \
  --network none \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=64m,mode=1777 \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  -e JETKVM_LAB_PASSWORD=ci-fixture \
  -e JETKVM_MCP_HTTP_TOKEN=ci-offline-token \
  -v "${config_path}:/config.yaml:ro" \
  "${image}" config validate --config /config.yaml

docker run --rm \
  --network none \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=64m,mode=1777 \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --entrypoint sh \
  "${image}" -euc '
    ffmpeg -nostdin -hide_banner -loglevel error \
      -f lavfi -i color=c=blue:s=64x48:d=0.1 -frames:v 1 \
      -c:v libx264 -preset ultrafast -tune zerolatency -f h264 /tmp/frame.h264
    ffmpeg -nostdin -hide_banner -loglevel error \
      -f h264 -i /tmp/frame.h264 -frames:v 1 -pix_fmt rgb24 -c:v png /tmp/frame.png
    test "$(ffprobe -v error -select_streams v:0 -show_entries stream=codec_name -of default=noprint_wrappers=1:nokey=1 /tmp/frame.png)" = png
    test "$(ffprobe -v error -select_streams v:0 -show_entries stream=width,height -of csv=p=0 /tmp/frame.png)" = 64,48
  '

container_id=$(docker run --detach \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=64m,mode=1777 \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --publish 127.0.0.1::8080 \
  -e JETKVM_LAB_PASSWORD=ci-fixture \
  -e JETKVM_MCP_HTTP_TOKEN=ci-health-token \
  -v "${config_path}:/config.yaml:ro" \
  "${image}" --config /config.yaml --http 0.0.0.0:8080)
endpoint=$(docker port "${container_id}" 8080/tcp | head -n 1)

healthy=false
for _ in {1..30}; do
  if curl --silent --fail \
    --dump-header "${temporary_directory}/headers" \
    --output "${temporary_directory}/body" \
    "http://${endpoint}/healthz"; then
    healthy=true
    break
  fi
  sleep 1
done
if [[ ${healthy} != true ]]; then
  echo "container health endpoint did not become available" >&2
  exit 1
fi
tr -d '\r' < "${temporary_directory}/headers" | grep -Eiq '^content-type: text/plain; charset=utf-8$'
[[ $(<"${temporary_directory}/body") == "ok" ]]
