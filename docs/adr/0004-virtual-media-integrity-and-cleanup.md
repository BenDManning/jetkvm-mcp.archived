# 0004-virtual-media-integrity-and-cleanup: Virtual-media integrity and cleanup

Status: accepted

Date: 2026-08-12

## Context

Local virtual-media operations read an operator-configured file, upload it to
JetKVM storage, and may mount it. Firmware reports a resumable byte offset but
provides no digest that binds an existing prefix to the current local file. The
local path can also be replaced while an upload is running, and lost replies can
leave partial or completed appliance artifacts in an uncertain state.

## Decision

Serialize virtual-media operations per configured device. Confine local sources
beneath the configured media root with `os.OpenRoot`, require a non-empty regular
file of at most 64 GiB, and reduce the appliance filename to the local basename.

Before every upload, delete a matching `.incomplete` appliance artifact and
require a new offset-zero upload. Hash the confined file before upload, hash the
exact bytes consumed by the upload, then reopen and re-hash the configured path
and verify file identity, size, and modification time before reporting success
or mounting. If negotiation, upload, verification, or mounting is ambiguous,
use independent short cleanup contexts to release a valid pending upload and
best-effort delete both partial and completed artifacts. Never resume solely
from the firmware byte count.

## Rationale

Without a firmware prefix digest, an offset is insufficient evidence that old
remote bytes match the selected local file. Fresh upload plus pre/during/post
content checks makes the completed object traceable to one stable confined
source. Per-device serialization prevents two calls from racing over appliance
filenames and media state. Conservative cleanup avoids mounting or reporting a
known-ambiguous object even though firmware deletion cannot be proven.

## Rejected alternatives

- **Rejected:** resume from `alreadyUploadedBytes`, because the existing remote
  prefix cannot be authenticated against the current local file.
- **Rejected:** rely only on path, size, or modification time, because content
  can change while retaining those values or the path can be atomically
  replaced.
- **Rejected:** leave an uploaded object after a failed verification or mount,
  because later operations could mistake ambiguous data for approved media.

## Consequences

- Every local upload reads the source for a pre-hash, the upload itself, and a
  post-upload verification, increasing local I/O and latency.
- Interrupted uploads restart from zero rather than saving bandwidth.
- Cleanup is best effort; a transport failure means the appliance may still
  retain data despite attempted deletion.
- Failure cleanup may delete an appliance object with the same basename, so the
  configured media root and filenames are an administrative boundary.
- URL mounting is a separate firmware-side fetch and does not receive these
  local-file integrity guarantees.

## Evidence

- Integrity, confinement, serialization, and cleanup: [`virtual_media.go`](../../internal/jetkvm/virtual_media.go)
- Upload transport bounds: [`provider.go`](../../internal/jetkvm/provider.go)
- Replacement, resume, cleanup, and confinement tests: [`virtual_media_test.go`](../../internal/jetkvm/virtual_media_test.go)
- Authenticated upload test: [`upload_test.go`](../../internal/jetkvm/upload_test.go)
- Media behavior contract: [`product-contract.md`](../product-contract.md)
- Media threat and residual-risk model: [`threat-model.md`](../threat-model.md)

## Revisit trigger

Revisit when any of the following is true:

- supported firmware returns a cryptographically bound prefix or whole-object
  digest plus documented atomic commit/abort semantics, and tests verify those
  values before resuming;
- an appliance exposes immutable content-addressed object identifiers and mount
  operations can select the verified identifier instead of a mutable filename;
- retained measurements on the largest declared media workload show the extra
  hash passes violate an accepted throughput objective; or
- firmware storage or deletion semantics change such that current cleanup calls
  can no longer identify both partial and completed artifacts deterministically.
