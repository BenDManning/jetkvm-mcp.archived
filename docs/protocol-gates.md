# MCP protocol gates

The protocol gate exercises the server against pinned official MCP tooling without contacting a JetKVM appliance. It uses a generated fixture configuration whose URL is non-routable and limits tool calls to `jetkvm_list_devices`.

## Pinned sources and coverage

`testdata/mcp-gates/pins.json` records exact npm versions, source commits, and tarball integrities for:

- `@modelcontextprotocol/conformance` `0.2.0-alpha.11` at `c321dd32035556e6769d3724a8ee97d87c3faaac`;
- `@modelcontextprotocol/inspector` `2.2.0` at `672f9f41c548487a468b9e7007d2f9de14da5a69`.

The conformance matrix classifies every server scenario exposed by the pinned release. Applicable 2026-07-28 Streamable HTTP scenarios are gates. Every N/A scenario carries a reason and a repository or Inspector replacement. The runner fails if the official inventory and the checked classification differ. There is no expected-failure baseline, and each allowed skipped check is explicitly pinned.

Inspector checks initialization, `tools/list`, and `jetkvm_list_devices` over both stdio and Streamable HTTP. Repository integration tests separately lock malformed JSONL failure, fresh-process recovery, clean EOF, and protocol-only stdout.

## Local run

Node.js 22.22.0 or newer within the Node 22 release line and the Go version declared by `go.mod` are required. From the repository root:

```sh
tmp="$(mktemp -d)"
go build -trimpath -o "$tmp/jetkvm-mcp" ./cmd/jetkvm-mcp
go run ./cmd/jetkvm-mcp-protocol-gates \
  --server-binary "$tmp/jetkvm-mcp" \
  --pins testdata/mcp-gates/pins.json \
  --artifacts "$tmp/mcp-gate-artifacts"
```

`testdata/mcp-gates/npm/package-lock.json` locks the complete npm dependency graph. The runner verifies every non-root package has an exact version, registry artifact URL, and SHA-512 integrity, installs with `npm ci --ignore-scripts --loglevel=error` in a disposable directory, rejects successful commands that emit unexpected stderr, and executes only the resulting local binaries. The Inspector launcher does not provide a top-level `--version` operation; that unsupported flag is not used.

## Evidence and failure policy

The runner writes:

- `summary.json`, containing source pins and pass/fail check identifiers;
- a copy of `pins.json`;
- one sanitized per-scenario JSON record containing only the scenario name, pass status, and reviewed skipped-check identifiers.

The official runner writes raw output only below the disposable private directory. Uploaded artifacts intentionally exclude raw Conformance and Inspector output, generated fixture configuration, process output, local endpoints and filesystem paths, environment values, and credentials. CI uploads the sanitized directory even when a check fails.

Any gated command failure, missing or extra official scenario, malformed Inspector result, unexpected server stdout, unexplained skip, or non-empty expected-failure baseline fails the job. Known gaps remain explicit N/A classifications with bounded reasons and replacement evidence; they do not suppress applicable test failures.
