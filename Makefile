SHELL := /bin/sh
BINARY := bin/jetkvm-mcp

.PHONY: build test race vet verify fuzz-smoke fuzz update-tool-manifest container

build:
	go build -trimpath -o $(BINARY) ./cmd/jetkvm-mcp

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

verify: test vet
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /tmp/jetkvm-mcp-linux-amd64 ./cmd/jetkvm-mcp
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o /tmp/jetkvm-mcp-linux-arm64 ./cmd/jetkvm-mcp
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -o /tmp/jetkvm-mcp-darwin-amd64 ./cmd/jetkvm-mcp
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o /tmp/jetkvm-mcp-darwin-arm64 ./cmd/jetkvm-mcp

fuzz-smoke:
	go test ./internal/fuzzpolicy -run '^TestFuzzTargetInventoryAndCorpusPolicy$$' -count=1
	python3 scripts/run-fuzz-targets.py --fuzztime 1s

fuzz:
	go test ./internal/fuzzpolicy -run '^TestFuzzTargetInventoryAndCorpusPolicy$$' -count=1
	python3 scripts/run-fuzz-targets.py --fuzztime 30s

update-tool-manifest:
	JETKVM_UPDATE_TOOL_MANIFEST=1 go test ./internal/mcpserver -run '^TestToolManifestFixtureUpdate$$' -count=1

container:
	docker build -t jetkvm-mcp:dev .
