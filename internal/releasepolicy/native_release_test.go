package releasepolicy_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type goreleaserConfig struct {
	Builds []struct {
		ID     string   `yaml:"id"`
		Goos   []string `yaml:"goos"`
		Goarch []string `yaml:"goarch"`
	} `yaml:"builds"`
	Archives []struct {
		ID    string   `yaml:"id"`
		IDs   []string `yaml:"ids"`
		Files []string `yaml:"files"`
	} `yaml:"archives"`
	Checksum struct {
		Name      string `yaml:"name_template"`
		Algorithm string `yaml:"algorithm"`
	} `yaml:"checksum"`
	SBOMs []struct {
		ID        string   `yaml:"id"`
		Artifacts string   `yaml:"artifacts"`
		IDs       []string `yaml:"ids"`
		Documents []string `yaml:"documents"`
		Args      []string `yaml:"args"`
	} `yaml:"sboms"`
}

func TestThirdPartyNoticesMatchReleasePackageClosure(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	command := exec.Command("go", "run", "./scripts/generate-third-party-notices.go", "-check", "THIRD_PARTY_NOTICES.md", "./cmd/jetkvm-mcp")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("third-party notice check: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "" {
		t.Fatalf("third-party notice check wrote unexpected output:\n%s", output)
	}
}

func TestGoReleaserProducesOnlyConsumerVerifiableLinuxArchives(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	var config goreleaserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}

	if len(config.Builds) != 1 {
		t.Fatalf("build count = %d, want 1", len(config.Builds))
	}
	build := config.Builds[0]
	if !reflect.DeepEqual(build.Goos, []string{"linux"}) {
		t.Errorf("release operating systems = %v, want [linux]", build.Goos)
	}
	if !reflect.DeepEqual(build.Goarch, []string{"amd64", "arm64"}) {
		t.Errorf("release architectures = %v, want [amd64 arm64]", build.Goarch)
	}

	if len(config.Archives) != 1 {
		t.Fatalf("archive count = %d, want 1", len(config.Archives))
	}
	archive := config.Archives[0]
	if !reflect.DeepEqual(archive.IDs, []string{build.ID}) {
		t.Errorf("archive build IDs = %v, want [%s]", archive.IDs, build.ID)
	}
	if !reflect.DeepEqual(archive.Files, []string{"LICENSE", "THIRD_PARTY_NOTICES.md"}) {
		t.Errorf("archive notice files = %v", archive.Files)
	}

	if config.Checksum.Name != "checksums.txt" || config.Checksum.Algorithm != "sha256" {
		t.Errorf("checksum = %#v, want checksums.txt using sha256", config.Checksum)
	}

	if len(config.SBOMs) != 1 {
		t.Fatalf("SBOM configuration count = %d, want 1", len(config.SBOMs))
	}
	sbom := config.SBOMs[0]
	if sbom.Artifacts != "archive" || !reflect.DeepEqual(sbom.IDs, []string{archive.ID}) {
		t.Errorf("SBOM subjects = artifacts %q IDs %v, want archive [%s]", sbom.Artifacts, sbom.IDs, archive.ID)
	}
	if !reflect.DeepEqual(sbom.Documents, []string{"{{ .ArtifactName }}.spdx.json"}) {
		t.Errorf("SBOM documents = %v", sbom.Documents)
	}
	wantArgs := []string{"$artifact", "--output", "spdx-json=$document"}
	if !reflect.DeepEqual(sbom.Args, wantArgs) {
		t.Errorf("SBOM arguments = %v, want %v", sbom.Args, wantArgs)
	}
}

func TestReleaseSnapshotUsesAcceptedToolchainAndVerifiesItsSubjects(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	command := exec.Command("make", "-n", "release-snapshot")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n release-snapshot: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"GOTOOLCHAIN=go1.26.6 go env GOROOT",
		"go run ./scripts/generate-third-party-notices.go -check THIRD_PARTY_NOTICES.md ./cmd/jetkvm-mcp",
		"release --snapshot --clean --skip=publish",
		"scripts/verify-native-release.sh dist",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("release-snapshot output does not contain %q\n%s", expected, text)
		}
	}
	for _, forbidden := range []string{"--key", "COSIGN_PASSWORD", "goreleaser release --clean", "gh release", "GOOS=darwin"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("release-snapshot output unexpectedly contains %q\n%s", forbidden, text)
		}
	}
}

func TestHostedNativeRehearsalUsesKeylessConstrainedIdentityWithoutPublishing(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "native-release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	var workflow struct {
		On          map[string]any    `yaml:"on"`
		Permissions map[string]string `yaml:"permissions"`
		Jobs        map[string]struct {
			Permissions map[string]string `yaml:"permissions"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	_, hasDispatch := workflow.On["workflow_dispatch"]
	_, hasCall := workflow.On["workflow_call"]
	if len(workflow.On) != 2 || !hasDispatch || !hasCall {
		t.Errorf("workflow triggers = %#v, want workflow_dispatch and workflow_call", workflow.On)
	}
	if !reflect.DeepEqual(workflow.Permissions, map[string]string{"contents": "read"}) {
		t.Errorf("default workflow permissions = %#v, want contents: read", workflow.Permissions)
	}
	job, ok := workflow.Jobs["native-rehearsal"]
	if !ok || !reflect.DeepEqual(job.Permissions, map[string]string{
		"attestations": "write",
		"contents":     "read",
		"id-token":     "write",
	}) {
		t.Errorf("native rehearsal permissions = %#v", job.Permissions)
	}

	for _, expected := range []string{
		"workflow_dispatch:",
		"workflow_call:",
		"release_ref:",
		"RELEASE_IDENTITY_REF: ${{ inputs.release_ref || github.ref }}",
		"RELEASE_IDENTITY_TRIGGER: ${{ inputs.release_ref && 'push' || 'workflow_dispatch' }}",
		"REUSABLE_RELEASE_REF: ${{ inputs.release_ref }}",
		"[[ ${REUSABLE_RELEASE_REF} =~ ^refs/tags/v",
		"[[ ${GITHUB_REF} == refs/heads/main ]]",
		"[[ ${GITHUB_REF} == ${RELEASE_IDENTITY_REF} ]]",
		"contents: read",
		"id-token: write",
		"attestations: write",
		"make release-snapshot RELEASE_VERIFY_ARM64=1",
		"cosign sign-blob --yes --bundle dist/checksums.txt.sigstore.json dist/checksums.txt",
		"cosign verify-blob",
		"--certificate-identity https://github.com/${GITHUB_REPOSITORY}/.github/workflows/native-release.yml@${RELEASE_IDENTITY_REF}",
		"--certificate-github-workflow-sha ${GITHUB_SHA}",
		"--certificate-github-workflow-trigger ${RELEASE_IDENTITY_TRIGGER}",
		"uses: actions/attest@1e69f48acb82d1966a394da916b4c1698aa569d6 # v4.2.2",
		"subject-checksums: dist/checksums.txt",
		"gh attestation verify",
		"--cert-identity https://github.com/${GITHUB_REPOSITORY}/.github/workflows/native-release.yml@${RELEASE_IDENTITY_REF}",
		"--source-ref ${RELEASE_IDENTITY_REF}",
		"--source-digest ${GITHUB_SHA}",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("native release workflow does not contain %q", expected)
		}
	}
	for _, forbidden := range []string{
		"contents: write", "packages: write", "pull_request_target", "secrets.", "cosign.key", "--key", "--signer-workflow", "gh release create", "goreleaser release --clean", "--certificate-github-workflow-trigger workflow_dispatch", "push:", "tags:",
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("native release workflow unexpectedly contains %q", forbidden)
		}
	}
	pinned := regexp.MustCompile(`^[[:space:]]+(?:- )?uses: [^@[:space:]]+@[0-9a-f]{40} # v[^[:space:]]+$`)
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "uses:") && !pinned.MatchString(line) {
			t.Errorf("Action is not pinned to a full SHA with a version comment: %s", line)
		}
	}
}
