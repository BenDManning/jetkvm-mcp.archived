# Security

## Supported versions

Only the latest stable JetKVM MCP release receives security fixes. Development
builds, prereleases, and older releases are unsupported. There is no remediation
SLA or blanket backport promise.

## Reporting a vulnerability

Use [GitHub private vulnerability reporting](https://github.com/BenDManning/jetkvm-mcp/security/advisories/new).
Do not open a public issue for a suspected vulnerability, exploit, credential,
device address, device data, screenshot, configuration, or unsanitized log.

Include the affected version or commit, the smallest reproducible description,
the security impact, and sanitized evidence. Do not access hardware, accounts,
or data you do not own or have explicit permission to test.

The maintainer aims to acknowledge a report within seven days on a best-effort
basis. Validation, remediation, release, and disclosure timing depend on impact
and available evidence. The maintainer will coordinate disclosure and publish an
advisory when users need to identify affected versions or take action.

JetKVM firmware, MCP clients, models, attached hosts, proxies, and deployment
infrastructure have separate security boundaries. See
[`docs/threat-model.md`](docs/threat-model.md) for the server's owned controls
and residual responsibilities.

## Publicity gate

Before public-readiness is announced, the owner privately verifies two
phishing-resistant GitHub authentication methods and offline recovery codes;
reviews sessions, PATs, SSH and deploy keys, OAuth and GitHub App grants, and
release credentials; and tests private vulnerability-report notifications. The
public repository must have secret scanning and push protection where
available, Dependabot alerts and security updates, immutable releases, the
reviewed Actions allowlist and full-SHA policy, and verified main and tag
rulesets. Only completion or a blocker is recorded publicly; recovery details
and credentials never enter the repository.

Additional scanners require evidence of a gap not answered by vet, Staticcheck,
`govulncheck`, secret scanning, and the existing protocol and privacy gates.
