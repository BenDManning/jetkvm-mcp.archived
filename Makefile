SHELL := /bin/sh
BINARY := bin/jetkvm-mcp
RELEASE_GO_VERSION := 1.26.6
COVERAGE_DIR ?= /tmp/jetkvm-mcp-coverage
MCP_GATE_DIR ?= /tmp/jetkvm-mcp-gates
MCP_GATE_SERVER ?= /tmp/jetkvm-mcp-gates-server
CONTAINER_AMD64_DIR ?= /tmp/jetkvm-mcp-container-amd64
CONTAINER_ARM64_DIR ?= /tmp/jetkvm-mcp-container-arm64

.PHONY: build format tidy tools-tidy module-verify tools-module-verify test race vet staticcheck govulncheck release-tool-versions release-snapshot coverage verify protocol-gates fuzz-smoke fuzz ci-minimum ci-quality update-tool-manifest container container-verify

build:
	go build -trimpath -o $(BINARY) ./cmd/jetkvm-mcp

format:
	test -z "$$(gofmt -l .)"

tidy:
	GOWORK=off go mod tidy -diff

tools-tidy:
	GOWORK=off GOTOOLCHAIN=go$(RELEASE_GO_VERSION) go -C tools mod tidy -diff

module-verify:
	go mod verify

tools-module-verify:
	GOTOOLCHAIN=go$(RELEASE_GO_VERSION) go -C tools mod verify

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

staticcheck:
	go tool staticcheck ./...

govulncheck:
	go tool govulncheck ./...

release-tool-versions:
	GOTOOLCHAIN=go$(RELEASE_GO_VERSION) go -C tools tool goreleaser --version
	GOTOOLCHAIN=go$(RELEASE_GO_VERSION) go -C tools tool cosign version
	GOTOOLCHAIN=go$(RELEASE_GO_VERSION) go -C tools tool syft --version

release-snapshot:
	GOTOOLCHAIN=go$(RELEASE_GO_VERSION) "$$(GOTOOLCHAIN=go$(RELEASE_GO_VERSION) go -C tools tool -n goreleaser)" release --snapshot --clean --skip=publish

coverage:
	mkdir -p $(COVERAGE_DIR)
	go test -covermode=atomic -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -func=$(COVERAGE_DIR)/coverage.out > $(COVERAGE_DIR)/coverage.txt

verify: test vet
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /tmp/jetkvm-mcp-linux-amd64 ./cmd/jetkvm-mcp
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o /tmp/jetkvm-mcp-linux-arm64 ./cmd/jetkvm-mcp
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -o /tmp/jetkvm-mcp-darwin-amd64 ./cmd/jetkvm-mcp
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o /tmp/jetkvm-mcp-darwin-arm64 ./cmd/jetkvm-mcp

protocol-gates:
	go build -trimpath -o $(MCP_GATE_SERVER) ./cmd/jetkvm-mcp
	go run ./cmd/jetkvm-mcp-protocol-gates --server-binary $(MCP_GATE_SERVER) --pins testdata/mcp-gates/pins.json --artifacts $(MCP_GATE_DIR)

fuzz-smoke:
	go test ./internal/fuzzpolicy -run '^TestFuzzTargetInventoryAndCorpusPolicy$$' -count=1
	python3 scripts/run-fuzz-targets.py --fuzztime 1s

fuzz:
	go test ./internal/fuzzpolicy -run '^TestFuzzTargetInventoryAndCorpusPolicy$$' -count=1
	python3 scripts/run-fuzz-targets.py --fuzztime 30s

ci-minimum: format tidy module-verify test vet fuzz-smoke

ci-quality: format tidy tools-tidy module-verify tools-module-verify race vet staticcheck govulncheck fuzz-smoke coverage verify

update-tool-manifest:
	JETKVM_UPDATE_TOOL_MANIFEST=1 go test ./internal/mcpserver -run '^TestToolManifestFixtureUpdate$$' -count=1

container:
	docker build -t jetkvm-mcp:dev .

container-verify:
	docker buildx build --platform linux/amd64 --target binary --output=type=local,dest=$(CONTAINER_AMD64_DIR) .
	docker buildx build --platform linux/arm64 --target binary --output=type=local,dest=$(CONTAINER_ARM64_DIR) .
	python3 -c 'import struct; assert struct.unpack("<H", open("$(CONTAINER_AMD64_DIR)/jetkvm-mcp", "rb").read(20)[18:20])[0] == 62'
	python3 -c 'import struct; assert struct.unpack("<H", open("$(CONTAINER_ARM64_DIR)/jetkvm-mcp", "rb").read(20)[18:20])[0] == 183'
	docker buildx build --platform linux/amd64,linux/arm64 --output=type=cacheonly .
