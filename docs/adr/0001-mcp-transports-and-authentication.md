# 0001-mcp-transports-and-authentication: MCP transports and authentication boundary

Status: accepted

Date: 2026-08-12

## Context

The server must work as a local MCP subprocess and may be exposed remotely by an
operator. Every current tool has JetKVM administrator consequences, so transport
reachability and authentication define the product's principal boundary. The
product is deliberately a small single-process integration rather than an
identity, consent, or policy service.

## Decision

Use MCP over stdio by default and expose stateless Streamable HTTP only when
`--http` is supplied. HTTP serves JSON responses at `/mcp`; it does not create
MCP sessions or expose deprecated HTTP+SSE.

Treat ownership of the stdio process and streams as the local administrative
principal. Permit unauthenticated direct HTTP only on loopback. Require one
configured bearer token for a non-loopback listener, compare it in constant
time, and require public `Host` and browser `Origin` authorities to match exact
configured origins. Delegate TLS termination and network access control to the
deployment. Do not add MCP OAuth, end-user identity, scopes, or per-tool/device
authorization to this product boundary.

## Rationale

Stdio gives MCP hosts the conventional least-exposed local path. Stateless
Streamable HTTP supports native and reverse-proxied remote clients without
retaining server-side session state. A single bearer matches the current
single-principal administration model, while bind, Host, and Origin checks
prevent accidental public exposure and DNS rebinding. Adding OAuth or policy
semantics before there is a multi-principal requirement would create a second
product rather than document the current integration.

## Rejected alternatives

- **Rejected:** HTTP-only operation, because it would remove the simpler local
  subprocess boundary and require a network listener for every use.
- **Rejected:** deprecated HTTP+SSE or stateful Streamable HTTP sessions, because
  current tools and JSON responses need no server-held session or event stream.
- **Rejected:** an embedded OAuth/capability service, because no current product
  requirement distinguishes users, scopes, tools, or devices.

## Consequences

- One stdio owner or bearer holder receives authority for every configured
  device and tool; annotations inform clients but do not authorize calls.
- Remote confidentiality, bearer rotation, routing, and proxy hardening remain
  deployment responsibilities.
- Native HTTP clients may omit `Origin`, but their public `Host` must still be
  configured and non-loopback binding still requires the bearer.
- Stateless operation makes retries simple at the transport layer, but callers
  must still avoid retrying uncertain physical mutations.
- Features that require server-initiated streams or durable MCP sessions are not
  available.

## Evidence

- Transport construction and authorization: [`transport.go`](../../internal/mcpserver/transport.go)
- CLI bind gate and HTTP server: [`main.go`](../../cmd/jetkvm-mcp/main.go)
- HTTP, bearer, and stateless tests: [`transport_test.go`](../../internal/mcpserver/transport_test.go)
- Host and Origin tests: [`origin_test.go`](../../internal/mcpserver/origin_test.go)
- Product boundary: [`product-contract.md`](../product-contract.md)
- Threat and deployment boundary: [`threat-model.md`](../threat-model.md)

## Revisit trigger

Revisit when any of the following is true:

- an accepted requirement introduces at least two principals with different
  device or tool permissions;
- a supported MCP client requires a server-held session or server-initiated
  transport feature that the current stateless JSON path cannot provide;
- the supported MCP revision removes or materially changes either selected
  transport or mandates a different remote authorization mechanism; or
- a reproducible security finding bypasses the bearer plus bind/Host/Origin
  boundary under the documented deployment baseline.
