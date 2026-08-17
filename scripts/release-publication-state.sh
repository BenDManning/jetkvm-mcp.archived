#!/usr/bin/env bash

# coordinate_release_publication is the single ordering seam for both the
# mutation-free rehearsal backend and the GitHub/GHCR production backend.
# Backend functions return 0 for confirmed success, 1 for definite failure, or
# 75 when an irreversible remote mutation requires owner reconciliation.
coordinate_release_publication() {
  local phase status
  for phase in preflight consume_version publish_exact_image publish_release move_latest; do
    if "release_backend_${phase}"; then
      continue
    else
      status=$?
      return "$status"
    fi
  done
}
