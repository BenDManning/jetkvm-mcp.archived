# 0005-local-only-raw-rpc: Keep raw RPC local and outside MCP

Status: accepted

Date: 2026-08-12

## Context

Typed MCP tools provide reviewed names, schemas, annotations, results, errors,
and physical consequences. Troubleshooting or protocol investigation can still
need a firmware method that has no typed tool. A generic remote raw-RPC tool
would bypass the reviewed manifest and could invoke undocumented mutations or
return private firmware data with no reliable annotation.

## Decision

Keep raw JetKVM RPC out of MCP discovery. Provide it only as the explicit local
CLI command `jetkvm-mcp debug rpc`, requiring a configured device, a syntactically
bounded method name, and one duplicate-free bounded JSON object for parameters.
Permit only the source-reviewed read-only methods `ping`, `getLocalVersion`, and
`getActiveExtension` by default. Every other method requires the explicit
per-invocation flag `--unsafe-acknowledge-risk`; configuration and environment
cannot persist that acknowledgement. Use a fresh data WebRTC session and return
the raw JSON result to the invoking terminal's stdout. The acknowledgement does
not classify an unreviewed method as safe. Do not expose a generic `jetkvm_rpc`,
`jetkvm_raw_rpc`, or `jetkvm_debug_rpc` tool.

## Rationale

The CLI boundary preserves a deliberate operator escape hatch without granting
remote MCP clients an unversioned capability that defeats the tool contract.
Syntactic validation and frame limits protect protocol parsing. The small
source-reviewed default set supports common observations without treating method
naming as authority. Local execution plus a deliberate one-shot acknowledgement
makes shell/process access—not an MCP annotation or persistent grant—the
explicit boundary for everything else. Specific stable operations can graduate
into typed MCP tools through ordinary compatibility and security review.

## Rejected alternatives

- **Rejected:** a generic raw-RPC MCP tool, because method consequences, schemas,
  idempotence, destructive behavior, and returned private data cannot be inferred
  safely from an arbitrary method string.
- **Rejected:** treating unknown methods as read-only diagnostics, because
  firmware method naming is not an authority for side effects.
- **Rejected:** a configuration or environment bypass, because it would turn a
  consequence acknowledgement into a persistent grant that callers could miss.
- **Rejected:** removing the escape hatch entirely, because bounded local
  protocol diagnosis remains useful before a method earns a typed product API.

## Consequences

- Anyone able to execute the binary with its configuration and environment can
  invoke undocumented read or mutation methods with the device credential only
  after supplying the unsafe acknowledgement on that invocation.
- Raw parameters may appear in process arguments or shell history, and raw
  results intentionally reach stdout; callers must not put secrets there or
  retain the streams unsafely.
- MCP clients cannot discover or invoke arbitrary firmware methods and continue
  to see only the static reviewed tools-only contract.
- The default allowlist is intentionally closed and source-maintained rather than
  configurable. The unsafe path has no semantic result redaction or automatic
  consequence detection; acknowledgement is not proof that a method is safe.

## Evidence

- Raw method validation and invocation: [`debug.go`](../../internal/jetkvm/debug.go)
- Local CLI routing and raw result stream: [`main.go`](../../cmd/jetkvm-mcp/main.go)
- Debug validation and data-session tests: [`debug_test.go`](../../internal/jetkvm/debug_test.go)
- MCP exclusion test: [`server_test.go`](../../internal/mcpserver/server_test.go)
- Product CLI and tool boundary: [`product-contract.md`](../product-contract.md)
- Raw-RPC consequence and privacy model: [`threat-model.md`](../threat-model.md)
- Inspected firmware revision and source ownership: [`protocol-sources.md`](../protocol-sources.md)

## Revisit trigger

Revisit when any of the following is true:

- a specific firmware method has typed input and output schemas, documented
  consequences and error semantics, correct tool annotations, implementation
  tests, threat-model coverage, and a reviewed manifest-fixture change; that
  method may then become a dedicated MCP tool without exposing generic RPC;
- the inspected firmware changes any default-safe handler or supplies another
  testable read-only method suitable for explicit source review; or
- retained support evidence shows the command has no remaining approved
  diagnostic use and an accepted compatibility change schedules its deprecation.

A request to expose arbitrary method strings remotely is a material product and
security boundary change, not a routine revisit of this ADR.
