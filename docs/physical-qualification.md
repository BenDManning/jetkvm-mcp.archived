# Physical qualification runbook

This runbook qualifies one exact JetKVM MCP release candidate and physical
fixture. It is a record template, not execution authority. Complete it only in
an owner-authorized window that names the fixture, operator, observer, and
start/end time. CI, fakes, source review, builds, and earlier unattributed runs
do not qualify hardware.

## Qualification record

Fill every field before the first device operation. Use stable product
identifiers, not secrets or network identifiers.

| Field | Exact value |
|---|---|
| Authorization reference and approved UTC window | |
| Operator and observer | |
| Qualification date (UTC) | |
| JetKVM model and firmware version | |
| Server commit and reported product version | |
| Server runtime OS and architecture | |
| FFmpeg version/build identity | |
| MCP transport and client name/version | |
| Disposable attached host make/model, OS, and architecture | |
| Synthetic local-media fixture size and SHA-256 | |
| Synthetic URL-media fixture origin class, size, and SHA-256 | |
| Recovery method and local console observer | |

The retained record must not contain device aliases, hostnames, serial numbers,
addresses, URLs, local paths, credentials, bearer tokens, typed text,
screenshots, image bytes, raw status/RPC results, or free-form errors. Keep the
configuration, selected alias and Wake-on-LAN target, media locations, and
credentials only in the private run sheet for the approved window.

## Preconditions

Before starting, the operator and observer confirm all of the following:

- the recorded commit is the exact release-candidate source, `--version`
  matches the record, and the configured MCP transport/client are ready;
- the JetKVM and attached host match the record by independent physical or
  local-console observation; the host is non-production, expendable, idle, and
  contains no valuable unsaved state;
- local console access, host power restoration, boot recovery, and an emergency
  stop are immediately available;
- the host has an inert text field and pointer surface where no key, click,
  scroll, or focus change can submit data, execute a command, change a setting,
  or activate a destructive control;
- the local and URL media are synthetic and non-secret, their digests are
  recorded, the URL origin is isolated and allowed, and the host will not write
  to or boot from either medium;
- the approved fixture has known safe starting states for power-button, reset,
  DC power, USB wake, and Wake-on-LAN checks; and
- telemetry and client output retention are disabled or constrained to the
  sanitized evidence fields below.

Any failed precondition is a failed qualification with no device operation.

## Stop rule

Execute one row at a time and record its result before continuing. Stop the
window immediately on an unexpected effect, timeout, cancellation, loss of
observation, identity drift, integrity mismatch, cleanup failure, or any
mutation result with `outcome: unknown`.

An `unknown` outcome means the request may have been sent. Never repeat that
mutation, never continue to another mutation, and do not attempt mutating
cleanup in the same window. Re-establish state only through independent
read-only observation. Any later attempt requires a separately authorized new
window after the fixture state is known.

## Checks

For each row retain: UTC start/end time, operation name, `pass` or `fail`, the
stable result code and outcome when present, whether the stated observation was
seen, and whether cleanup completed. Use bounded client deadlines. Do not copy
arguments, returned content, screenshots, raw output, or errors into evidence.

| Order | Operation | Preconditions and expected independent observation | Cleanup |
|---:|---|---|---|
| 1 | `jetkvm_list_devices` | No device session is expected. The selected configured alias appears once with the expected configuration-derived capabilities. | None. |
| 2 | `jetkvm_get_status` | The observer compares power, video, and USB state with the physical host; the structured result is internally consistent and contains no unredacted media source. | None. |
| 3 | `jetkvm_get_virtual_media_status` | Start unmounted. The structured result reports unmounted without a URL, path, or filename. | None. |
| 4 | `jetkvm_capture_screen` | Display a non-secret fixture. A fresh PNG decodes, its dimensions are credible, and it depicts that fixture without being retained. | Clear the decoded image buffer and close the viewer/client surface. |
| 5 | local `debug rpc`: `ping`, `getLocalVersion`, `getActiveExtension` | Invoke only the three source-reviewed methods without `--unsafe-acknowledge-risk`. Each returns one JSON result; version/extension observations agree with the recorded fixture. | Discard raw stdout; retain only per-method pass/fail. |
| 6 | `jetkvm_keyboard` / `type_text` | Focus the inert text field and send one fixed non-secret ASCII marker. It appears exactly once. | Clear the field locally and confirm no key remains pressed. |
| 7 | `jetkvm_keyboard` / `press_key` | Send one inert printable key without a submitting or system modifier. Its effect appears exactly once. | Clear the field locally and confirm neutral keyboard state. |
| 8 | `jetkvm_mouse` / `move_absolute` | On the inert surface, the pointer moves once to the planned safe coordinate. | Return the pointer locally to a neutral area. |
| 9 | `jetkvm_mouse` / `move_relative` | The pointer moves once by the planned offset and remains within the inert surface. | Return the pointer locally to a neutral area. |
| 10 | `jetkvm_mouse` / `click` | Click an inert target once; the target shows exactly one harmless activation. | Release/neutralize mouse buttons and reset the inert target locally. |
| 11 | `jetkvm_mouse` / `scroll` | The inert surface moves once by the planned horizontal or vertical amount without changing a value. | Return the surface and pointer to neutral state; confirm no button remains pressed. |
| 12 | `jetkvm_upload_virtual_media_file` | With no mount active, upload the recorded local fixture. Success reports storage-backed, unmounted completion; the source remains unchanged with the recorded digest. | Record the storage artifact for end-of-window removal through the approved local appliance cleanup path. |
| 13 | `jetkvm_mount_virtual_media_file` | Mount the same recorded local fixture read-only. Media status reports storage/read-only and the host sees the expected synthetic medium. | `jetkvm_unmount_virtual_media`; confirm status unmounted and host removal. |
| 14 | `jetkvm_mount_virtual_media_url` | Mount the recorded URL fixture read-only. Media status reports HTTP/read-only and the host sees only the expected synthetic medium. | `jetkvm_unmount_virtual_media`; confirm status unmounted and host removal. |
| 15 | `jetkvm_unmount_virtual_media` | Call once more while already unmounted. It completes and status remains unmounted. | Remove uploaded synthetic storage artifacts through the approved local appliance path and confirm no test media remains. |
| 16 | `jetkvm_press_host_power_button` | The fixture is in the recorded safe state for its configured short-press behavior. Exactly one expected suspend, shutdown, or wake transition occurs. | Restore the planned baseline locally; do not use another tool call as an implicit retry. |
| 17 | `jetkvm_wake_host_usb` | Put the host in the recorded USB-wakeable state. Exactly one expected wake transition occurs. | Restore the planned baseline locally. |
| 18 | `jetkvm_wake_host_lan` | Put the independently verified configured target in its recorded wakeable state. At most one expected boot/wake transition occurs. | Restore the planned baseline locally; do not resend on uncertainty. |
| 19 | `jetkvm_press_host_reset_button` | The idle expendable host has no valuable state. Exactly one reset and expected boot sequence occurs. | Observe a complete return to the planned baseline. |
| 20 | `jetkvm_force_host_power_off` | The idle expendable host is safe for forced loss. Exactly one transition to off occurs. | Restore power locally and observe a complete return to baseline. |
| 21 | `jetkvm_turn_host_dc_power_off` | The idle expendable host is safe for immediate DC removal. Exactly one transition to DC-off occurs. | Continue only after independent confirmation of off. |
| 22 | `jetkvm_turn_host_dc_power_on` | The observer has independently confirmed DC-off. Exactly one transition to powered/booting occurs. | Observe a complete return to the final planned state. |

The HID rows also qualify the implementation's neutral-release cleanup only
when the observer confirms no stuck key or button after each operation. The
media rows pass only when every mount is observed, every unmount is observed,
the source digests remain unchanged, and all synthetic appliance storage is
removed. Best-effort cleanup without an observed final state is a failure.

## Managed-session supplement

This supplement is mandatory for a release candidate implementing
[ADR 0008](adr/0008-managed-per-device-webrtc-ownership.md). It is not authority
to access hardware and does not alter the stop rule above.

Add the exact capability-profile revision and the approved browser/client build
to the qualification record. The attached disposable host must display a
changing, non-secret visual marker such as a fixture timestamp or counter. The
named observer records the marker and browser session-switch prompt state at
each checkpoint. Retain only capture time, dimensions, byte count, and a
sanitized marker observation; never retain screenshot or video bytes.

Run these bounded scenarios in the approved order, restoring and independently
observing the fixture baseline between scenarios:

1. Perform 100 sequential mixed status, media-status, capture, HID, and
   owner-approved controlled power/media operations. Exactly one server
   generation becomes active, each mutation reaches its expected observable
   completion, captures follow the changing marker, and the observer records
   zero browser session-switch prompts.
2. Perform 20 concurrent status-plus-capture pairs. Each status is complete,
   each capture observes a fresh post-admission marker, no frame is shared, and
   the observer records zero prompts.
3. During one bounded synthetic virtual-media upload, perform approved reads and
   captures. Verify upload integrity and cleanup, distinct captures, one
   generation, and no connection destabilization.
4. Perform three cycles of `jetkvm_release_session`, external browser
   connection, browser release, and `jetkvm_take_over_session`. Each server
   release completes local cleanup before success, the browser connects without
   a prompt after release, and takeover becomes authoritative without an
   approval handshake.
5. Leave the managed generation with no generation-bound work for the configured
   60-second idle interval. After idle cleanup completes, connect the browser
   and verify no switch prompt. Then explicitly take over again.
6. Force an external browser takeover during an in-flight read. Verify the
   server observes recognized takeover, reports `session_taken_over`, sends no
   later RPC on the displaced generation, does not reconnect automatically, and
   resumes only after explicit takeover.
7. Cause separately approved short and long terminal transport interruptions
   without a takeover notification. Both must report `ownership_uncertain`,
   fence ordinary work, avoid automatic reconnect, and require explicit
   takeover. Do not accept a same-generation recovery grace.
8. Send SIGTERM during an active read/capture and, in a separate restored
   scenario, during one owner-approved disposable mutation. Verify bounded
   cancellation, parallel generation closure, pump joining, mutation phase
   classification, and no mutation replay.
9. After explicit release, normal idle expiry, and clean server shutdown,
   connect the browser at a prescribed checkpoint and verify no switch prompt.

The read-only, non-interactive `jetkvm-mcp-validate` interface may be extended
only for repeated read/capture evidence that fits its existing inputs and
sanitized report. Browser handoff, mutation, network interruption, and shutdown
remain fixture-runbook steps. An automated helper may orchestrate bounded MCP
call sequences but cannot supply physical observation or authorization.
The repository's source-run `jetkvm-mcp-fixture-runner` accepts sequential
batches of bounded MCP calls for this purpose. Keep its plan private. A
consequential plan also requires its explicit owner-authorization
acknowledgement flag, which records operator intent but is not authorization.
Run browser checkpoints and record observations outside the helper.

Retain the exact build, model, firmware, runtime, FFmpeg identity, capability
profile, prompt observations, sanitized marker observations, operation
outcomes, session telemetry, and timestamps. A named observer's browser record
is required; telemetry corroborates generation reuse but never substitutes for
that record.

If RPC/frame acquisition or upload overlap destabilizes the connection, select
session-wide serialization in the capability profile and rerun this complete
supplement. If terminal `Disconnected` behavior does not match the specified
fence, revise the capability behavior and rerun. If sequential reuse itself
causes a switch prompt, reject the resident-session design. No failed
combination is qualified with warnings.

Forced mutation ambiguity is established with fixture-level dispatch-phase
injection proving `unknown` and no retry. Deliberately interrupting a physical
mutation requires separate owner authorization and a disposable recovery plan;
it is not implied by this supplement.

## Pass and retained evidence

Qualification passes only when every row passes in one approved window, no
`unknown` mutation outcome occurs, every expected postcondition is independently
observed, and all cleanup is confirmed. A candidate implementing ADR 0008 must
also pass every managed-session supplement scenario. Retain one sanitized record
containing:

- every exact identity from the qualification table except the excluded
  private/network identifiers;
- the authorization reference, window, operator, observer, and UTC date;
- every operation/sub-operation above with its bounded timestamps, result,
  stable code/outcome, observed-postcondition flag, and cleanup flag; and
- an explicit statement that CI, fakes, builds, source review, and historical
  runs were not used as physical qualification.

A pass qualifies only the recorded server commit/version, model, firmware,
runtime, FFmpeg, transport/client, attached-host fixture, and completed checks.
It does not qualify a model family, firmware range, other runtime, other
transport, future release, performance, soak behavior, or indefinite support.
Any missing identity, row, observation, or cleanup makes the record non-
qualifying.
