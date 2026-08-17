package cipolicy_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCITargetsProduceNonOverlappingEvidence(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))

	quality := dryRun(t, root, "ci-quality", "COVERAGE_DIR=/tmp/ci-coverage")
	for _, required := range []string{
		"gofmt -l .",
		"go mod tidy -diff",
		"go mod verify",
		"JETKVM_TEST_REAL_FFMPEG=1 go test -race -covermode=atomic -coverprofile=/tmp/ci-coverage/coverage.out ./...",
		"go vet ./...",
		"go tool staticcheck ./...",
		"go tool govulncheck ./...",
		"--fuzztime 1s",
		"GOOS=linux GOARCH=amd64 go build",
		"GOOS=linux GOARCH=arm64 go build",
	} {
		if !strings.Contains(quality, required) {
			t.Errorf("ci-quality output does not contain %q\n%s", required, quality)
		}
	}
	for _, duplicate := range []string{
		"go test ./...",
		"GOOS=darwin",
	} {
		if strings.Contains(quality, duplicate) {
			t.Errorf("ci-quality output unexpectedly contains %q\n%s", duplicate, quality)
		}
	}

	minimum := dryRun(t, root, "ci-minimum")
	for _, required := range []string{"go test ./...", "go build -trimpath"} {
		if !strings.Contains(minimum, required) {
			t.Errorf("ci-minimum output does not contain %q\n%s", required, minimum)
		}
	}
	for _, duplicate := range []string{
		"gofmt", "tidy", "mod verify", "go vet", "staticcheck", "govulncheck", "fuzz", "-race", "-cover", "JETKVM_TEST_REAL_FFMPEG",
	} {
		if strings.Contains(minimum, duplicate) {
			t.Errorf("ci-minimum output unexpectedly contains %q\n%s", duplicate, minimum)
		}
	}

	protocol := dryRun(t, root, "protocol-gates", "MCP_GATE_SERVER=/tmp/ci-protocol-server")
	if got := strings.Count(protocol, "go build -trimpath"); got != 1 {
		t.Errorf("protocol-gates builds the server %d times, want 1\n%s", got, protocol)
	}
	if !strings.Contains(protocol, "--server-binary /tmp/ci-protocol-server") {
		t.Errorf("protocol-gates does not run the built server\n%s", protocol)
	}
}

func TestContainerTargetBuildsAndSmokesEachFinalImageOnce(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	output := dryRun(t, root, "container-verify", "CONTAINER_CREATED=1970-01-01T00:00:00Z", "CONTAINER_SBOM_DIR=/tmp/ci-container-sbom")

	for _, platform := range []string{"linux/amd64", "linux/arm64"} {
		build := "docker buildx build --platform " + platform
		if got := strings.Count(output, build); got != 1 {
			t.Errorf("container-verify builds %s %d times, want 1\n%s", platform, got, output)
		}
	}
	for _, required := range []string{
		"--build-arg VERSION=ci",
		"--build-arg SOURCE=https://github.com/BenDManning/jetkvm-mcp",
		"--build-arg REVISION=",
		"--build-arg CREATED=1970-01-01T00:00:00Z",
		"--load --tag jetkvm-mcp:ci-amd64",
		"--load --tag jetkvm-mcp:ci-arm64",
		"go -C tools tool syft docker:jetkvm-mcp:ci-amd64 --output spdx-json=/tmp/ci-container-sbom/amd64.spdx.json",
		"go -C tools tool syft docker:jetkvm-mcp:ci-arm64 --output spdx-json=/tmp/ci-container-sbom/arm64.spdx.json",
		"scripts/smoke-container.sh jetkvm-mcp:ci-amd64 linux/amd64 ci",
		"scripts/smoke-container.sh jetkvm-mcp:ci-arm64 linux/arm64 ci",
	} {
		if !strings.Contains(output, required) {
			t.Errorf("container-verify output does not contain %q\n%s", required, output)
		}
	}
	if strings.Contains(output, "--target binary") || strings.Contains(output, "type=cacheonly") {
		t.Errorf("container-verify builds intermediate or duplicate images\n%s", output)
	}
}

func dryRun(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	args := append([]string{"-n"}, arguments...)
	command := exec.Command("make", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}
