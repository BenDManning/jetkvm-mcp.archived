# 0007-same-origin-browser-http: Keep browser Streamable HTTP same-origin

Status: accepted

Date: 2026-08-13

## Context

Streamable HTTP must validate incoming `Origin` values to resist DNS rebinding,
but origin admission alone does not make an endpoint usable through browser
CORS. Cross-origin browser support would also require an explicit preflight and
response-header policy and decisions about browser credentials, authentication,
and exposed headers. The product has no browser application, OAuth principal,
or credentialed cross-origin mode.

The supported deployment boundary already consists of native MCP clients and a
single trusted administrative principal using stdio, loopback HTTP, or stateless
HTTP behind a TLS reverse proxy. Public `Host` and browser `Origin` authorities
are explicit deployment policy under
[ADR 0001](0001-mcp-transports-and-authentication.md).

## Decision

Support native Streamable HTTP clients and same-origin browser deployments.
Native clients may omit `Origin`, subject to the existing Host-admission and
authentication rules. A browser request must use the MCP endpoint's external
origin: its `Origin` scheme and authority must match the admitted public endpoint
or the exact loopback scheme and authority.

Treat `http.allowed_origins` as a list of exact public endpoint origins. Entries
establish admitted public `Host` authorities and valid browser origins; they are
not CORS grants. Reject wildcard entries. Validate every present Origin before
bearer authentication, method handling, or MCP dispatch. Invalid, foreign, and
opaque `null` origins receive HTTP 403.

Do not emit `Access-Control-Allow-Origin`,
`Access-Control-Allow-Credentials`, `Access-Control-Allow-Methods`, or
`Access-Control-Allow-Headers`. Same-origin `OPTIONS` requests are not CORS
preflights needed by the supported mode; after any configured bearer is
satisfied, they receive the endpoint's normal HTTP 405 response. Foreign or
invalid preflights are rejected with 403 before authentication or method
handling.

TLS remains a reverse-proxy or deployment boundary. A non-loopback listener
still requires the configured bearer. A loopback listener behind a proxy relies
on protected proxy reachability and any proxy authentication the deployment
selects; this decision adds no OAuth or browser identity semantics.

## Rationale

Same-origin browser operation needs no CORS response policy and preserves the
existing single-principal model. Explicit Host and Origin admission implements
the MCP transport's DNS-rebinding requirement without turning a routing setting
into an ambient browser authorization grant. Rejecting cross-origin requests
also avoids wildcard-plus-credentials and reflected-origin configurations that
would be unsafe for a server whose bearer grants every tool and device.

## Rejected alternatives

- **Rejected:** General cross-origin browser support, because it requires a new
  CORS, credential, and browser-client contract that no current product surface
  needs.
- **Rejected:** Wildcard or reflected `Access-Control-Allow-Origin`, because the
  endpoint carries administrative authority and the product has no safe public
  browser principal.
- **Rejected:** Requiring `Origin` from native MCP clients, because non-browser
  clients do not reliably send it and Host plus bearer admission already owns
  that path.

## Consequences

- A browser UI hosted at a different origin cannot call this MCP endpoint
  directly. Operators must serve it at the same external origin or use a native
  MCP client.
- Reverse proxies must preserve the external `Host`; admitted public origins
  must be configured exactly, including scheme and non-default port.
- `allowed_origins` does not cause CORS headers to be emitted and must not be
  described as enabling arbitrary browser origins.
- Origin rejection remains ahead of authentication and MCP dispatch, limiting
  DNS-rebinding requests before any bearer or protocol behavior is considered.
- No browser smoke test is required for direct cross-origin access because that
  mode is unsupported; deterministic HTTP handler tests own the contract.

## Evidence

- HTTP middleware and ordering: [`transport.go`](../../internal/mcpserver/transport.go)
- Origin and OPTIONS matrix: [`origin_test.go`](../../internal/mcpserver/origin_test.go)
- Configuration parsing: [`config.go`](../../internal/config/config.go)
- Configuration validation tests: [`config_test.go`](../../internal/config/config_test.go)
- Product compatibility boundary: [`product-contract.md`](../product-contract.md)
- Deployment and residual-risk boundary: [`threat-model.md`](../threat-model.md)

## Revisit trigger

Revisit when any of the following is true:

- an accepted product requirement introduces a separately hosted browser client;
- a supported MCP client demonstrably requires cross-origin browser access;
- an accepted authentication design adds browser principals and credential
  semantics suitable for cross-origin use; or
- the supported MCP transport revision materially changes mandatory Origin or
  browser handling.
