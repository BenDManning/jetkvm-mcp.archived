# 0006-virtual-media-url-origin-boundary: Deny virtual-media URL mounts outside configured origins

Status: accepted

Date: 2026-08-13

## Context

A virtual-media URL mount asks the configured JetKVM appliance, rather than this
process, to fetch caller-selected content. An unrestricted URL therefore lets an
MCP caller reach services visible from the appliance network. Application-side
host classification alone cannot reliably constrain firmware-side DNS
resolution, DNS rebinding, redirects, or routing, and the product does not have a
general grant engine.

## Decision

Each device may define `media_url_allowed_origins` as a list of exact HTTP(S)
origins. An origin consists only of scheme, host or IP literal, and effective
port. Omitted HTTP port 80 and HTTPS port 443 are equivalent to their explicit
forms; every other port remains explicit. Wildcards, user information, paths,
queries, fragments, malformed ports, and non-HTTP(S) schemes are invalid
configuration.

URL mounting is unavailable when the list is absent or empty. Before any device
session is opened, every mount URL must have an origin exactly equal to one
configured entry. Scheme, hostname, and explicit-port differences are decisive;
DNS hostnames and schemes compare case-insensitively, subject to the default-port
normalization above. URL paths, queries, and
fragments do not participate in the origin comparison and are forwarded only
after the origin passes. Loopback, private, link-local, or other IP literals are
not implicitly trusted or denied: they work only when their exact origin is
explicitly configured.

The appliance network must remain segmented and its media-fetch egress limited
to the intended origin. This process does not resolve DNS, follow or inspect
firmware redirects, pin addresses, or prove the final network destination.

## Rationale

Deny-by-default exact origins turn URL mounting from unrestricted appliance-side
network access into an explicit per-device administrative capability while
remaining deterministic and testable without network access. Treating all
address classes uniformly avoids pretending that application-side IP rules can
control later firmware DNS and redirect behavior. Keeping paths out of the
allowlist also avoids a per-file approval database while retaining a narrow
network boundary.

## Rejected alternatives

- **Rejected:** unrestricted URL mounting for the trusted MCP principal, because
  bearer or stdio authority would also grant unconstrained appliance-visible
  network reach.
- **Rejected:** wildcard hosts or suffix matching, because they enlarge the
  boundary implicitly and make DNS-controlled subdomains difficult to audit.
- **Rejected:** a private/local-address denylist, because DNS and redirects occur
  in firmware and can bypass an application-only classification.
- **Rejected:** DNS resolution, IP pinning, redirect inspection, a general grant
  engine, or per-file approval database in this process, because the appliance
  performs the fetch and exposes no verified enforcement hook for those controls.

## Consequences

- Existing configurations that omit `media_url_allowed_origins` can no longer
  mount URL media; local upload/mount and media status remain available.
- Every permitted scheme, hostname or IP literal, and effective port must be
  configured per device. An implicit default port and the same explicit port are
  the same origin; non-default ports remain distinct.
- Query strings and fragments are permitted after origin validation but may
  contain secrets and must not be used for credentials.
- Firmware-side DNS answers, redirects, and routing can still reach an unintended
  address. Network segmentation and egress controls are required rather than
  claimed as application enforcement.
- The change adds no dynamic policy service, wildcard grant, or per-file state.

## Evidence

- Configuration and runtime enforcement: [`config.go`](../../internal/config/config.go) and [`virtual_media.go`](../../internal/jetkvm/virtual_media.go)
- Pure config and fake-provider tests: [`config_test.go`](../../internal/config/config_test.go) and [`virtual_media_test.go`](../../internal/jetkvm/virtual_media_test.go)
- Product behavior contract: [`product-contract.md`](../product-contract.md)
- Threat, deployment, and residual-risk model: [`threat-model.md`](../threat-model.md)

## Revisit trigger

Revisit when any of the following is true:

- supported firmware exposes a verified way to disable redirects and bind a
  request to application-resolved destination addresses;
- a supported deployment cannot express its required media service as a bounded
  set of exact origins and presents a reviewed replacement boundary;
- retained evidence shows firmware rewrites scheme, authority, or port before
  fetching in a way that invalidates exact-origin validation; or
- the product adopts a separately accepted authorization or grant architecture
  that supersedes this per-device configuration model.
