#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=release-publication-state.sh
source "$script_dir/release-publication-state.sh"

failure_phase=none
failure_status=0
events=()

run_backend_phase() {
  local phase=$1
  events+=("$phase")
  if [[ $failure_phase == "$phase" ]]; then
    return "$failure_status"
  fi
}

release_backend_preflight() { run_backend_phase preflight; }
release_backend_consume_version() { run_backend_phase consume_version; }
release_backend_publish_exact_image() { run_backend_phase publish_exact_image; }
release_backend_publish_release() { run_backend_phase publish_release; }
release_backend_move_latest() { run_backend_phase move_latest; }

run_scenario() {
  local expected_status=$1
  local expected_events=$2
  failure_phase=$3
  failure_status=$4
  events=()
  local actual_status=0
  coordinate_release_publication || actual_status=$?
  [[ $actual_status == "$expected_status" ]]
  [[ ${events[*]} == "$expected_events" ]]
}

run_scenario 0 "preflight consume_version publish_exact_image publish_release move_latest" none 0
run_scenario 1 "preflight" preflight 1
run_scenario 1 "preflight consume_version" consume_version 1
run_scenario 1 "preflight consume_version publish_exact_image" publish_exact_image 1
run_scenario 1 "preflight consume_version publish_exact_image publish_release" publish_release 1
run_scenario 75 "preflight consume_version publish_exact_image publish_release" publish_release 75
run_scenario 1 "preflight consume_version publish_exact_image publish_release move_latest" move_latest 1
run_scenario 75 "preflight consume_version publish_exact_image publish_release move_latest" move_latest 75

echo "verified mutation-free publication state machine and failure outcomes"
