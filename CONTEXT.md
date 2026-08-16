# Domain context

JetKVM MCP is one bounded context: a Go MCP server that operates configured JetKVM devices and their attached hosts.

## Vocabulary

- **Configured JetKVM** — a JetKVM appliance admitted through a named configuration entry.
- **Attached host** — the computer controlled or observed through a configured JetKVM.
- **Device alias** — the configuration name exposed to MCP callers instead of private connection details.
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
