# JetKVM compatibility evidence

This directory keeps compact evidence about the observed JetKVM protocol boundary. It is not a device-qualification database and does not create a firmware support claim.

## Ledger

[`jetkvm-ledger.json`](jetkvm-ledger.json) records five distinct evidence classes:

- `source_review` identifies the exact upstream commit whose relevant implementation was reviewed;
- `source_drift` records a bounded comparison with another exact upstream commit and uses `review_required` when a reviewed surface differs;
- `read_only_hardware` records the retained date, server revision, environment class, check names, result, and limitations of a device run;
- `managed_session_hardware` records physical WebRTC ownership and takeover observations against an exact application firmware version; and
- `mutation_hardware` records a bounded negative observation from a standing-authorized mutation window. `observed_failure` is a compatibility limitation, not a positive qualification or a claim that the same failure affects other devices or firmware.

An entry contains an exact JetKVM source reference when source was inspected, an exact server source reference, the product version when one was actually exercised, a date, a coarse environment class, fixed check names, a result, and explicit limitations. `not_observed`, `not_attributed`, and `not_retained` are evidence gaps, not wildcard compatibility.

Credential values, session cookies, bearer tokens, and host-screen or media contents do not belong in this ledger. Device names, endpoints, firmware identity, and transport/event observations are not secret. A `pass` qualifies only the listed checks for the recorded combination. It never means that mocks or source inspection establish physical compatibility.

## Observed HTTP mount and unmount limitations

On 2026-08-19, the standing-authorized expendable fixture running JetKVM
application 0.5.8 / system 0.2.8 fetched byte ranges from the approved media
origin but returned an unknown outcome while mounting URL media. Explicit
unmount recovered the partial media state. Local upload and mount passed in the
same window. The exact JetKVM model and attached-host identity were not
retained, so this observation neither qualifies nor disqualifies application
0.5.8 generally.

Review of the official application 0.5.8 source found that its browser and this
client both call `mountWithHTTP` with `url` and `mode`. Firmware records HTTP
media state before starting NBD and does not roll that state back when later NBD
or USB attachment fails. The second retained range request can arise only after
NBD startup, when firmware attaches `/dev/nbd0` to USB mass storage and the
gadget reads its first block. Because attachment is the handler's last fallible
step, the returned RPC error localizes the failure to that attachment rather
than the HTTP response, requested mode, or RPC contract. The raw firmware error
was not retained, so the lower-level attachment cause remains unknown and no
evidence-backed client repair is available. Keep treating an unknown mount
result as non-retryable and re-establish state independently.

On 2026-08-20, a second standing-authorized run on application 0.5.8 / system
0.2.8 and an attached `MS-S1 MAX` host successfully mounted URL media. The host
read a 4096-byte prefix matching the synthetic source, after which unmount
returned `timeout` / `unknown`. Independent host observation found `/dev/sr0`
absent, but media-status calls then timed out while `ping` continued to pass.
An authorized appliance reboot restored unmounted status. Deleting the exact
synthetic upload also returned an unknown protocol result; a separate storage
listing established that it was absent and that unrelated files remained.
There was no read-only path to recover or inspect firmware's locked internal
media state. Before the separately authorized reboot, independent evidence
established only that the host had detached the medium and the appliance still
answered `ping`; the internal state remained unknown.

Five mount/unmount runs with no explicit host read passed. A bounded loop with
one verified host read reproduced the timeout and wedged status. In the 0.5.8
source, unmount holds the virtual-media state lock while calling the NBD client
disconnect, and media status requires the same lock. A blocked disconnect thus
explains the independently observed detach, missing acknowledgement, responsive
`ping`, wedged status, and reboot recovery. A longer client timeout, inferred
success, retry, or automatic appliance reboot would violate the existing
unknown-outcome contract rather than repair the firmware behavior.

## Observed HID acknowledgement limitation

On 2026-08-20, a standing-authorized qualification run on exact server commit
`947e65d542fe8c0e10e3fdbe33e191f5233238d8`, JetKVM application 0.5.8 /
system 0.2.8, and an attached `MS-S1 MAX` host observed an intermittent missing
mouse effect. A polling observer opened all JetKVM Linux input event devices and
reported ready before dispatch. The relative `+8/+8` mouse call then returned
completed without any host event during the bounded observation interval. The
window stopped immediately. The initial recovery observation confirmed no
pressed middle button and no mounted medium. After the separately authorized
exact-file cleanup, a second read-only host observation at 2026-08-20T03:26:18Z
confirmed zero pressed keys or buttons across all three JetKVM input devices,
with `/dev/sr0` unmounted and containing no medium. This later observation does
not retroactively move the keyboard check ahead of the earlier cleanup.

Other bounded attempts on that fixture produced exact host events for absolute
and relative movement, middle-button press and release, and ordinary plus
high-resolution vertical wheel movement. The negative observation therefore
does not establish permanent USB detach or a broader firmware incompatibility.
Application 0.5.8 source shows that its common HID wrapper returns success
without writing when its internal USB state is not ready and suppresses
temporary HID-unavailable write errors. The protocol exposes neither branch in
the acknowledgement and has no per-report delivery receipt. Existing typed USB
status cannot prove a later report reached the host.

The client consequently cannot safely turn the acknowledgement into a physical
success claim, retry the HID action without duplication risk, or manufacture an
`unknown` result from an otherwise successful RPC response. Physical
qualification must continue to require independent observation. This bounded
negative entry supplies no client workaround and makes no model- or
firmware-wide claim.

## Focused source drift

The reviewed source boundary is declared in [`jetkvm-upstream-surfaces.json`](jetkvm-upstream-surfaces.json). It covers authentication, signaling, RPC/version reads, video, HID, and virtual media. Given a separate local checkout of the official upstream repository, run:

```sh
go run ./cmd/jetkvm-mcp-upstream-drift --upstream /path/to/jetkvm/kvm --target origin/main
```

The command performs no clone, fetch, device request, or mutation. It verifies that the reviewed pin and target exist in the supplied Git repository, compares only the declared public upstream paths, and emits one JSON report. Exit `0` means no declared path changed, exit `1` means focused source review is required, and exit `2` means the comparison could not be trusted. Local checkout paths are not included in the report. Custom paths and revisions must be passed directly to the argument-safe Go CLI as shown above; they are deliberately not accepted through Make variables.

A source-only run on 2026-08-14 compared reviewed pin `b3c29a44d9e2862b8ff7530830781803ce27b060` with historical upstream commit `fe77acd5f00300a4ab9acd5da57d7bb0916351d9`. Every declared surface differed on at least one path, so the ledger records `review_required`. This does not prove incompatibility and does not move the reviewed pin.

## Evidence triggers

| Change or observation | Required response |
|---|---|
| A declared upstream path changes | Review the focused source diff before moving the pin; record the exact target and affected surfaces. |
| Authentication, signaling, RPC names/parameters, typed status/media fields, RTP/video behavior, or HID/media semantics change | Update the corresponding synthetic fixtures and focused tests, then perform source review. |
| A model, firmware version, relevant runtime, FFmpeg identity, MCP transport/client, or attached-host fixture changes for a proposed positive claim | Run the standing-authorized [physical qualification runbook](../physical-qualification.md) and retain its record tied to the exact identities required by the product contract. |
| The read-only validator's discovery, status, capture, or media-status behavior changes | Refresh validator tests and use the configured non-production physical target when device evidence is relevant. |
| A mutation path or consequence may have changed | Stop. Repeat the standing-authorized physical qualification runbook; source or read-only evidence is insufficient. |
| Evidence is missing, stale, unattributed, or no longer covers a declared behavior | Keep or add an explicit compatibility warning. Do not infer support, auto-upgrade, or treat fake-device success as authority. |

Moving the pin or adding a positive model/firmware claim is not part of the drift command and requires reviewed evidence. Hardware use follows the repository's standing fixture authorization and mutation stop rule.
