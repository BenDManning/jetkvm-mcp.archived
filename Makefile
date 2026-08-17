SHELL := /bin/sh
BINARY := bin/jetkvm-mcp
RELEASE_GO_VERSION := 1.26.6
COVERAGE_DIR ?= /tmp/jetkvm-mcp-coverage
MCP_GATE_DIR ?= /tmp/jetkvm-mcp-gates
MCP_GATE_SERVER ?= /tmp/jetkvm-mcp-gates-server
CONTAINER_VERSION ?= ci
CONTAINER_SOURCE ?= https://github.com/BenDManning/jetkvm-mcp
CONTAINER_REVISION ?= $(shell git rev-parse HEAD)
CONTAINER_CREATED ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
CONTAINER_CREATED := $(CONTAINER_CREATED)
CONTAINER_SBOM_DIR ?= /tmp/jetkvm-mcp-container-sbom
CONTAINER_AMD64_IMAGE ?= jetkvm-mcp:ci-amd64
CONTAINER_ARM64_IMAGE ?= jetkvm-mcp:ci-arm64

.PHONY: build format tidy tools-tidy module-verify tools-module-verify test race race-coverage vet staticcheck govulncheck release-tool-versions release-snapshot coverage cross-build-linux verify protocol-gates fuzz-smoke fuzz ci-minimum ci-quality update-tool-manifest container container-verify

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
	JETKVM_TEST_REAL_FFMPEG=1 go test -race ./...

race-coverage:
	mkdir -p $(COVERAGE_DIR)
	JETKVM_TEST_REAL_FFMPEG=1 go test -race -covermode=atomic -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -func=$(COVERAGE_DIR)/coverage.out > $(COVERAGE_DIR)/coverage.txt

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

cross-build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /tmp/jetkvm-mcp-linux-amd64 ./cmd/jetkvm-mcp
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o /tmp/jetkvm-mcp-linux-arm64 ./cmd/jetkvm-mcp

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

ci-minimum: test build

ci-quality: format tidy tools-tidy module-verify tools-module-verify race-coverage vet staticcheck govulncheck fuzz-smoke cross-build-linux

update-tool-manifest:
	JETKVM_UPDATE_TOOL_MANIFEST=1 go test ./internal/mcpserver -run '^TestToolManifestFixtureUpdate$$' -count=1

container:
	docker build --build-arg VERSION=dev --build-arg SOURCE=$(CONTAINER_SOURCE) --build-arg REVISION=$(CONTAINER_REVISION) --build-arg CREATED=$(CONTAINER_CREATED) -t jetkvm-mcp:dev .

define verify-container-image
docker buildx build --platform $(1) --build-arg VERSION=$(CONTAINER_VERSION) --build-arg SOURCE=$(CONTAINER_SOURCE) --build-arg REVISION=$(CONTAINER_REVISION) --build-arg CREATED=$(CONTAINER_CREATED) --load --tag $(2) .
GOTOOLCHAIN=go$(RELEASE_GO_VERSION) go -C tools tool syft docker:$(2) --output spdx-json=$(CONTAINER_SBOM_DIR)/$(3).spdx.json
scripts/smoke-container.sh $(2) $(1) $(CONTAINER_VERSION) $(CONTAINER_SOURCE) $(CONTAINER_REVISION) $(CONTAINER_CREATED) $(CONTAINER_SBOM_DIR)/$(3).spdx.json
endef

container-verify:
	mkdir -p $(CONTAINER_SBOM_DIR)
	$(call verify-container-image,linux/amd64,$(CONTAINER_AMD64_IMAGE),amd64)
	$(call verify-container-image,linux/arm64,$(CONTAINER_ARM64_IMAGE),arm64)
