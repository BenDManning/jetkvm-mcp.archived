# Mutation validation checklist

This procedure prepares a future supervised validation of JetKVM mutation tools. It is dry-run-first and fail-closed. It does not grant permission to operate hardware.

## No execution authority

The repository command only validates a plan:

```sh
go run ./cmd/jetkvm-mcp-mutation-checklist --plan testdata/mutation-validation-plan.json
```

A passing report always contains `"execution_authorized":false`. The command has no MCP client, device transport, URL fetch, media reader, or mutation implementation. A live run requires a separate approval naming an expendable target and time window. Never infer that approval from this plan, a passing dry run, repository access, or earlier read-only evidence.

The deprecated combined `jetkvm_virtual_media` tool is deliberately excluded. Future validation uses only the one-purpose tools below so each consequence is explicit.

## Designated expendable target

Before any future live step, an operator who is not relying on the MCP configuration alone must:

1. identify the expendable, non-production host and JetKVM through an independent physical or console observation;
2. compare that observation with the privately supplied run identity without recording the identity in this plan, logs, screenshots, or repository artifacts;
3. mark the plan only after the target is visibly confirmed;
4. stop if the target changes, is ambiguous, has production workload, contains valuable unsaved state, or cannot be recovered locally.

The plan stores only booleans. It rejects aliases, URLs, addresses, serials, paths, tokens, typed content, or other free-form target data.

## Observer and emergency stop

A future approved window requires one operator executing the step and one observer watching the physical host state. Recovery media, local console access, and the intended power-restoration path must be ready before the first mutation. The observer calls stop on identity drift, unexpected host state, timeout, ambiguous result, loss of observation, or any consequence outside the row below. Stop means no later step and no retry.

## Outcome unknown: stop and never retry

For every mutation, `outcome=unknown` means the request may have reached the device. Do not repeat it. Observe the host and JetKVM through an independent read-only path, preserve only sanitized outcome evidence, and end the window. Timeout or cancellation does not make a mutation safe to retry. A new attempt requires a new approved plan after state is re-established.

## Per-operation checklist

Each step requires an independently observable postcondition, a bounded timeout, and a prepared recovery action. Use only synthetic, non-secret HID input and expendable media. Each HID step must name one concrete `hid_operation`: `type_text` or `press_key` for `jetkvm_keyboard`, and `move_absolute`, `move_relative`, `click`, or `scroll` for `jetkvm_mouse`.

| Tool | Consequence | Preconditions | Observable postcondition | Timeout and recovery |
|---|---|---|---|---|
| `jetkvm_keyboard` / `type_text` | Sends an inert text sequence to the focused host application. | Open a disposable text field; verify no shell, password, or privileged prompt has focus; select a fixed inert ASCII test sequence. | Observer sees exactly the inert sequence once. | 30 s; on mismatch or uncertainty, stop without resending and clear the disposable field locally. |
| `jetkvm_keyboard` / `press_key` | Sends one named or printable key, optionally with modifiers, to the focused host application. | Open a disposable text field; select one inert key combination that cannot submit, close, switch, or invoke a privileged action. | Observer sees exactly the planned key effect once. | 30 s; on mismatch or uncertainty, stop without resending and recover focus locally. |
| `jetkvm_mouse` / `move_absolute` | Moves the host pointer to one absolute coordinate. | Open an inert disposable surface; confirm the destination has no destructive or activating control. | Observer sees one move to the planned inert location. | 30 s; stop on unexpected focus, hover effect, or position and recover locally. |
| `jetkvm_mouse` / `move_relative` | Moves the host pointer by one relative offset. | Open an inert disposable surface with enough margin for the planned offset. | Observer sees one move by the planned offset within the inert region. | 30 s; stop on unexpected focus, hover effect, or position and recover locally. |
| `jetkvm_mouse` / `click` | Clicks one mouse button and may activate host UI. | Place the pointer on an inert disposable target with no destructive control; select one button. | Observer sees exactly one planned button action on the inert target. | 30 s; stop on unexpected focus or activation and recover locally. |
| `jetkvm_mouse` / `scroll` | Sends one horizontal or vertical wheel movement and may change the visible host context. | Open an inert disposable scroll surface; confirm no focused control interprets wheel input as a value change. | Observer sees exactly the planned scroll once on the inert surface. | 30 s; stop on unexpected context or value change and recover locally. |
| `jetkvm_press_host_power_button` | Simulates a short physical power-button press and may suspend or shut down the host. | Expendable host is idle; observer confirms expected configured power-button behavior. | Observer sees the one expected power-state transition. | 60 s; do not press again; use the prepared local recovery path. |
| `jetkvm_press_host_reset_button` | Immediately resets the host and may lose state. | Expendable host has no valuable state and reset recovery is ready. | Observer sees one reset and subsequent expected boot indicator. | 60 s; do not repeat if boot state is unclear; recover locally. |
| `jetkvm_force_host_power_off` | Holds the ATX button and can cause data loss. | Expendable host is idle and forced-off recovery is accepted. | Observer sees power become off once. | 60 s; never resend on timeout or unknown outcome; inspect and recover locally. |
| `jetkvm_turn_host_dc_power_on` | Applies host DC power. | Observer confirms host is off and safe to energize. | Observer sees one transition to powered-on/booting. | 60 s; do not repeat if state is ambiguous. |
| `jetkvm_turn_host_dc_power_off` | Removes host DC power and can cause data loss. | Expendable host is idle and safe for immediate power removal. | Observer sees one transition to powered off. | 60 s; do not repeat; restore only through the separately planned recovery step. |
| `jetkvm_wake_host_lan` | Sends a network wake request to the configured target. | Observer confirms the expendable host is asleep/off and the configured wake target was independently verified. | Observer sees at most one expected wake/boot transition. | 120 s; no resend on uncertainty; inspect configuration and host state offline. |
| `jetkvm_wake_host_usb` | Requests wake through the configured USB path. | Observer confirms the expendable host is in the intended wakeable state. | Observer sees at most one expected wake transition. | 120 s; no resend on uncertainty; recover locally. |
| `jetkvm_upload_virtual_media_file` | Copies a file to JetKVM storage and may consume storage. | Use a small synthetic non-secret image; precompute its digest and reserve cleanup capacity. | Sanitized status confirms a storage-backed item; independently compare the approved synthetic source digest where the approved procedure permits. | 300 s; on mismatch/unknown outcome stop, do not upload again, and perform separately approved cleanup only after state inspection. |
| `jetkvm_mount_virtual_media_url` | Makes the host consume remote media through an allowed origin. | Use a synthetic non-secret image at an approved isolated origin; confirm read-only mode unless write behavior is explicitly approved. | Sanitized media status reports mounted HTTP media with the expected mode; host sees only the synthetic medium. | 120 s; stop on redirect/origin/state uncertainty and use the planned unmount recovery once state is known. |
| `jetkvm_mount_virtual_media_file` | Uploads a confined local file and then mounts it to the host. | Use a small synthetic non-secret image, precompute its digest, confirm the intended mode, and reserve cleanup capacity. | Sanitized media status reports mounted storage media with the expected mode; independently compare the approved synthetic source digest where the approved procedure permits. | 120 s; on integrity mismatch or unknown outcome stop without re-uploading, then use the planned unmount/cleanup recovery only after state is known. |
| `jetkvm_unmount_virtual_media` | Detaches currently mounted media and may interrupt host access. | Observer confirms the expendable host is not writing or booting from the medium. | Sanitized media status reports unmounted and host no longer sees the medium. | 60 s; do not repeat on unknown outcome; inspect status and host state. |

## Run order and stopping rules

1. Validate the exact plan offline and retain only the sanitized report.
2. Obtain a separate approval for one named private target and one window.
3. Reconfirm target, controls, observer, emergency stop, and recovery immediately before the window.
4. Execute at most one checklist row at a time. Confirm its postcondition before considering the next row.
5. Stop on timeout, cancellation, unknown outcome, target drift, observer loss, cleanup failure, integrity mismatch, or unexpected state.
6. Never convert a read-only status result, fixture result, source review, or dry-run report into mutation authority.

## Sanitized evidence

Retain only plan schema version, operation class, timestamps rounded as required by the approved procedure, stable result code, stable consequence outcome, whether the independent postcondition was observed, and whether cleanup completed. Do not retain device aliases, addresses, URLs, paths, filenames, typed input, screenshots, image bytes, tokens, firmware values, raw RPC/child output, raw errors, private target identity, or operator credentials.
