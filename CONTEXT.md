# Domain context

JetKVM MCP is one bounded context: a Go MCP server that operates configured JetKVM devices and their attached hosts.

## Vocabulary

- **Configured JetKVM** — a JetKVM appliance admitted through a named configuration entry.
- **Attached host** — the computer controlled or observed through a configured JetKVM.
- **Device alias** — the configuration name exposed to MCP callers instead of private connection details.
- **Device owner** — the per-device module that owns managed-session lifecycle, scheduling, observations, and firmware adaptation behind the Manager.
- **Managed device session** — the server-owned authoritative WebRTC operator connection for one configured JetKVM.
- **Managed-session generation** — one uninterrupted managed-session lifetime and the fence against stale work within the process.
- **Owner snapshot** — the immutable private ownership, transition, health, observation, and capability view published to the Manager.
- **Capability profile** — the versioned firmware-specific scheduling and compatibility rules selected for a configured JetKVM.
- **External authenticated session** — another authenticated JetKVM operator session whose actor type is not identified by the protocol.
- **Idle lease** — the bounded unused period before automatic managed-session release.
- **Released device** — a configured JetKVM explicitly yielded from ordinary server operation for the current process lifetime.
- **Server takeover** — explicit acquisition of authoritative operator ownership, which may displace an external authenticated session.
- **Recognized session takeover** — the sticky ownership state after JetKVM reports that another authenticated session became authoritative.
- **Ownership uncertainty** — the sticky ownership state after an unexplained managed-session loss.
- **MCP tool** — a supported MCP operation; raw JetKVM RPC is a separate local diagnostic surface.
- **Read-only operation** — an operation intended to observe state without changing the appliance or attached host.
- **Mutation** — an operation that may change appliance, storage, HID, power, network, or attached-host state.
- **Unknown outcome** — a mutation result meaning the request may have been sent without a conclusive response. It is never retryable and ends the current approved mutation window; state must be re-established through an independent read-only path before any separately approved future attempt.
- **Virtual media** — media fetched or uploaded for appliance storage or mounting on the attached host.
- **Physical qualification** — retained evidence that a named product build, JetKVM model and firmware, runtime, and explicitly listed operations were exercised on actual hardware. It qualifies only that recorded combination and those checks.

## Canonical documents

- Product scope, compatibility, and support: `docs/product-contract.md`
- Architecture decisions: `docs/adr/README.md` and the relevant ADRs
- Setup and operation: `README.md`
- Security and privacy boundaries: `docs/threat-model.md`
- Protocol provenance: `docs/protocol-sources.md`

Code and tests remain authoritative for executable behavior.
