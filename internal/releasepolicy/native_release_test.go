package releasepolicy_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

func TestExactReleaseSubjectsUseTheProtectedVersionWithoutPublishing(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	command := exec.Command("make", "-n", "release-subjects", "RELEASE_TAG=v1.2.3")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n release-subjects: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		`GORELEASER_CURRENT_TAG="v1.2.3"`,
		"release --clean --skip=announce,publish,validate",
		`RELEASE_EXPECTED_VERSION="${RELEASE_TAG#v}"`,
		`RELEASE_EXPECTED_COMMIT="${GITHUB_SHA:-$(git rev-parse HEAD)}"`,
		"scripts/verify-native-release.sh dist",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("release-subjects output does not contain %q\n%s", expected, text)
		}
	}
	for _, forbidden := range []string{"--snapshot", "gh release", "COSIGN_PASSWORD"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("release-subjects output unexpectedly contains %q\n%s", forbidden, text)
		}
	}
}
