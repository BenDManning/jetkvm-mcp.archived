JetKVM MCP is a conventional Go program that operates JetKVM devices through MCP over stdio or Streamable HTTP.

## Commands

- Focused test: `go test ./path/to/package -run '^TestName$'`
- Build: `make build`
- Full validation: `go mod verify && make race && make verify`
- Format changed Go files: `gofmt -w <changed .go files>`

## Working rules

- Keep logs and diagnostics on stderr; reserve stdio-server stdout for MCP protocol traffic.
- Automated tests do not qualify physical compatibility. Physical-device work requires explicit owner authorization for the exact execution window and consequence boundaries. Stop a mutation sequence on any `outcome: unknown` until state is independently established.
- Keep generated and private operational artifacts out of source control. Device aliases, endpoints, serial numbers, private paths, screenshots, and unsanitized qualification output are sensitive even when they are not credentials.
- Use [README.md](README.md) for setup and operation, [docs/product-contract.md](docs/product-contract.md) before behavior changes, [docs/adr/README.md](docs/adr/README.md) before architecture changes, and [docs/protocol-sources.md](docs/protocol-sources.md) before protocol work or selective reuse.

## Delivery

- Use GitHub issues and pull requests for task status and review; do not create parallel Markdown TODO lists.
- Run the checks relevant to every changed surface and commit only intended paths.
- Do not merge, release, deploy, change repository visibility, or operate physical hardware without the repository owner's explicit approval.
