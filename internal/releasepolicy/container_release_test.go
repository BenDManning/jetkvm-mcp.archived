package releasepolicy_test

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestContainerReleaseSnapshotBuildsOneVerifiedMultiPlatformSubject(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	command := exec.Command(
		"make", "-n", "container-release-snapshot",
		"CONTAINER_VERSION=v1.0.0",
		"CONTAINER_REVISION=0123456789abcdef0123456789abcdef01234567",
		"CONTAINER_CREATED=2026-08-17T00:00:00Z",
		"CONTAINER_RELEASE_DIR=/tmp/container-release",
	)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n container-release-snapshot: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"docker buildx build --platform linux/amd64,linux/arm64",
		"--build-arg VERSION=v1.0.0",
		"--build-arg SOURCE=https://github.com/BenDManning/jetkvm-mcp",
		"--build-arg REVISION=0123456789abcdef0123456789abcdef01234567",
		"--build-arg CREATED=2026-08-17T00:00:00Z",
		"--provenance=false",
		"--output type=oci,dest=/tmp/container-release/jetkvm-mcp.oci.tar",
		"scripts/verify-container-release.sh /tmp/container-release/jetkvm-mcp.oci.tar /tmp/container-release",
		"go -C tools tool syft oci-archive:/tmp/container-release/jetkvm-mcp.oci.tar --platform linux/amd64 --output spdx-json=/tmp/container-release/linux-amd64.spdx.json",
		"go -C tools tool syft oci-archive:/tmp/container-release/jetkvm-mcp.oci.tar --platform linux/arm64 --output spdx-json=/tmp/container-release/linux-arm64.spdx.json",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("container release snapshot does not contain %q\n%s", expected, text)
		}
	}
	for _, forbidden := range []string{"--push", "cache-from", "cache-to", "docker login", "latest"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("container release snapshot unexpectedly contains %q\n%s", forbidden, text)
		}
	}
}

func TestHostedContainerRehearsalVerifiesDigestEvidenceWithoutPublishing(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "container-release.yml"))
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
	if _, ok := workflow.On["workflow_dispatch"]; !ok {
		t.Error("container rehearsal is missing workflow_dispatch")
	}
	if _, ok := workflow.On["workflow_call"]; !ok {
		t.Error("container rehearsal is missing workflow_call")
	}
	if len(workflow.On) != 2 {
		t.Errorf("container rehearsal triggers = %#v", workflow.On)
	}
	if !reflect.DeepEqual(workflow.Permissions, map[string]string{"contents": "read"}) {
		t.Errorf("default permissions = %#v", workflow.Permissions)
	}
	job, ok := workflow.Jobs["container-rehearsal"]
	if !ok {
		t.Fatal("container-rehearsal job is missing")
	}
	if !reflect.DeepEqual(job.Permissions, map[string]string{
		"attestations": "write",
		"contents":     "read",
		"id-token":     "write",
	}) {
		t.Errorf("container rehearsal permissions = %#v", job.Permissions)
	}

	for _, expected := range []string{
		"persist-credentials: false",
		"cache: false",
		"RELEASE_IDENTITY_REF: ${{ inputs.release_ref || github.ref }}",
		"[[ ${REUSABLE_RELEASE_REF} =~ ^refs/tags/v",
		"[[ ${GITHUB_REF} == refs/heads/main ]]",
		"make container-release-snapshot",
		"cosign sign-blob --yes",
		"cosign verify-blob",
		"--certificate-identity https://github.com/${GITHUB_REPOSITORY}/.github/workflows/container-release.yml@${RELEASE_IDENTITY_REF}",
		"subject-path: dist/container/image-manifest.json",
		"subject-path: dist/container/image-manifest-linux-amd64.json",
		"subject-path: dist/container/image-manifest-linux-arm64.json",
		"sbom-path: dist/container/linux-amd64.spdx.json",
		"sbom-path: dist/container/linux-arm64.spdx.json",
		"gh attestation verify dist/container/image-manifest.json",
		"--predicate-type https://spdx.dev/Document/v2.3",
		"manifest-digests.json",
		"publication-plan.json",
		"retention-days: 30",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("container rehearsal does not contain %q", expected)
		}
	}
	for _, forbidden := range []string{
		"packages: write", "contents: write", "pull_request", "pull_request_target", "secrets.", "docker/login-action", "--push", "cache-from", "cache-to", "ghcr.io/bendmanning/jetkvm-mcp:latest",
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("container rehearsal unexpectedly contains %q", forbidden)
		}
	}
	pinned := regexp.MustCompile(`^[[:space:]]+(?:- )?uses: [^@[:space:]]+@[0-9a-f]{40} # v[^[:space:]]+$`)
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "uses:") && !pinned.MatchString(line) {
			t.Errorf("Action is not pinned to a full SHA with a version comment: %s", line)
		}
	}
}

func TestReadmeExplainsContainerRehearsalVerificationWithoutPublicationClaims(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, expected := range []string{
		"manifest-digests.json",
		"sha256sum image-manifest.json",
		`[.manifests[].platform | (.os + "/" + .architecture)] | sort == ["linux/amd64", "linux/arm64"]`,
		"cosign verify-blob",
		"--certificate-github-workflow-repository \"$repo\"",
		"--certificate-github-workflow-ref \"$release_ref\"",
		"--certificate-github-workflow-sha \"$release_commit\"",
		`any(.packages[]; .name == "ca-certificates")`,
		`any(.packages[]; .name == "ffmpeg")`,
		"linux-amd64.spdx.json",
		"linux-arm64.spdx.json",
		"image-manifest-linux-amd64.json",
		"image-manifest-linux-arm64.json",
		"gh attestation verify image-manifest.json",
		"--predicate-type https://spdx.dev/Document/v2.3",
		"does not publish an image",
		"does not claim reproducibility or a SLSA level",
		".version_tag_immutable == true",
		`.latest_after == "complete_stable_publication"`,
		".published == false",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("README container rehearsal guidance does not contain %q", expected)
		}
	}
}

func TestContainerReleaseVerifierRecordsExactManifestSubjects(t *testing.T) {
	releaseDir := t.TempDir()
	archivePath, wantIndexDigest, wantPlatformDigests := writeOCIReleaseFixture(t, releaseDir)
	command := exec.Command("bash", filepath.Join("..", "..", "scripts", "verify-container-release.sh"), archivePath, releaseDir)
	command.Env = append(os.Environ(),
		"CONTAINER_EXPECTED_VERSION=v1.0.0",
		"CONTAINER_EXPECTED_SOURCE=https://github.com/BenDManning/jetkvm-mcp",
		"CONTAINER_EXPECTED_REVISION=0123456789abcdef0123456789abcdef01234567",
		"CONTAINER_EXPECTED_CREATED=2026-08-17T00:00:00Z",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("verify container release: %v\n%s", err, output)
	}

	data, err := os.ReadFile(filepath.Join(releaseDir, "manifest-digests.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Image     string            `json:"image"`
		Manifest  string            `json:"manifest_digest"`
		Platforms map[string]string `json:"platform_digests"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Image != "ghcr.io/bendmanning/jetkvm-mcp" {
		t.Errorf("image = %q", got.Image)
	}
	if got.Manifest != wantIndexDigest {
		t.Errorf("manifest digest = %q, want %q", got.Manifest, wantIndexDigest)
	}
	if !reflect.DeepEqual(got.Platforms, wantPlatformDigests) {
		t.Errorf("platform digests = %#v, want %#v", got.Platforms, wantPlatformDigests)
	}
	manifest, err := os.ReadFile(filepath.Join(releaseDir, "image-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if digestBytes(manifest) != wantIndexDigest {
		t.Errorf("recorded manifest bytes do not match %s", wantIndexDigest)
	}
	for platform, digest := range wantPlatformDigests {
		path := filepath.Join(releaseDir, "image-manifest-"+strings.ReplaceAll(platform, "/", "-")+".json")
		manifest, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if digestBytes(manifest) != digest {
			t.Errorf("recorded %s manifest bytes do not match %s", platform, digest)
		}
	}
	planData, err := os.ReadFile(filepath.Join(releaseDir, "publication-plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan struct {
		VersionTag          string `json:"version_tag"`
		VersionTagImmutable bool   `json:"version_tag_immutable"`
		LatestTag           string `json:"latest_tag"`
		LatestAfter         string `json:"latest_after"`
		Published           bool   `json:"published"`
	}
	if err := json.Unmarshal(planData, &plan); err != nil {
		t.Fatal(err)
	}
	if plan.VersionTag != "ghcr.io/bendmanning/jetkvm-mcp:v1.0.0" || !plan.VersionTagImmutable {
		t.Errorf("version publication plan = %#v", plan)
	}
	if plan.LatestTag != "ghcr.io/bendmanning/jetkvm-mcp:latest" || plan.LatestAfter != "complete_stable_publication" || plan.Published {
		t.Errorf("latest publication plan = %#v", plan)
	}
}

func TestContainerReleaseVerifierRejectsInvalidSemanticPrereleaseVersions(t *testing.T) {
	for _, version := range []string{"v1.0.0-a..b", "v1.0.0-a.", "v1.0.0-01"} {
		t.Run(version, func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "empty.oci.tar")
			if err := os.WriteFile(archive, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("bash", filepath.Join("..", "..", "scripts", "verify-container-release.sh"), archive, t.TempDir())
			command.Env = append(os.Environ(),
				"CONTAINER_EXPECTED_VERSION="+version,
				"CONTAINER_EXPECTED_SOURCE=https://github.com/BenDManning/jetkvm-mcp",
				"CONTAINER_EXPECTED_REVISION=0123456789abcdef0123456789abcdef01234567",
				"CONTAINER_EXPECTED_CREATED=2026-08-17T00:00:00Z",
			)
			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("invalid version %q was accepted", version)
			}
			if !strings.Contains(string(output), "not an exact semantic version") {
				t.Fatalf("invalid version %q failed for the wrong reason:\n%s", version, output)
			}
		})
	}
}

func writeOCIReleaseFixture(t *testing.T, releaseDir string) (string, string, map[string]string) {
	t.Helper()
	blobs := make(map[string][]byte)
	addBlob := func(value any) (string, int64) {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		digest := digestBytes(data)
		blobs[strings.TrimPrefix(digest, "sha256:")] = data
		return digest, int64(len(data))
	}
	addRawBlob := func(data []byte) (string, int64) {
		digest := digestBytes(data)
		blobs[strings.TrimPrefix(digest, "sha256:")] = data
		return digest, int64(len(data))
	}

	labels := map[string]string{
		"org.opencontainers.image.source":   "https://github.com/BenDManning/jetkvm-mcp",
		"org.opencontainers.image.revision": "0123456789abcdef0123456789abcdef01234567",
		"org.opencontainers.image.version":  "v1.0.0",
		"org.opencontainers.image.licenses": "MIT",
		"org.opencontainers.image.created":  "2026-08-17T00:00:00Z",
	}
	platformDigests := make(map[string]string)
	descriptors := make([]map[string]any, 0, 2)
	for _, architecture := range []string{"amd64", "arm64"} {
		configDigest, configSize := addBlob(map[string]any{
			"architecture": architecture,
			"os":           "linux",
			"config": map[string]any{
				"User":       "10001:10001",
				"Entrypoint": []string{"/usr/local/bin/jetkvm-mcp"},
				"Labels":     labels,
			},
		})
		layerDigest, layerSize := addRawBlob([]byte("fixture-layer-" + architecture))
		manifestDigest, manifestSize := addBlob(map[string]any{
			"schemaVersion": 2,
			"mediaType":     "application/vnd.oci.image.manifest.v1+json",
			"config": map[string]any{
				"mediaType": "application/vnd.oci.image.config.v1+json",
				"digest":    configDigest,
				"size":      configSize,
			},
			"layers": []map[string]any{{
				"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
				"digest":    layerDigest,
				"size":      layerSize,
			}},
		})
		platform := "linux/" + architecture
		platformDigests[platform] = manifestDigest
		descriptors = append(descriptors, map[string]any{
			"mediaType": "application/vnd.oci.image.manifest.v1+json",
			"digest":    manifestDigest,
			"size":      manifestSize,
			"platform": map[string]string{
				"os":           "linux",
				"architecture": architecture,
			},
		})
	}
	indexDigest, indexSize := addBlob(map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.index.v1+json",
		"manifests":     descriptors,
	})
	outerIndex, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"manifests": []map[string]any{{
			"mediaType": "application/vnd.oci.image.index.v1+json",
			"digest":    indexDigest,
			"size":      indexSize,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	sbom := map[string]any{
		"spdxVersion": "SPDX-2.3",
		"packages": []map[string]string{
			{"name": "ca-certificates", "versionInfo": "20250419"},
			{"name": "ffmpeg", "versionInfo": "7:7.1.1-1"},
		},
	}
	sbomData, err := json.Marshal(sbom)
	if err != nil {
		t.Fatal(err)
	}
	for _, architecture := range []string{"amd64", "arm64"} {
		if err := os.WriteFile(filepath.Join(releaseDir, "linux-"+architecture+".spdx.json"), sbomData, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	archivePath := filepath.Join(releaseDir, "jetkvm-mcp.oci.tar")
	archive, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := tar.NewWriter(archive)
	writeEntry := func(name string, data []byte) {
		t.Helper()
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	writeEntry("oci-layout", []byte(`{"imageLayoutVersion":"1.0.0"}`))
	writeEntry("index.json", outerIndex)
	for digest, data := range blobs {
		writeEntry(filepath.Join("blobs", "sha256", digest), data)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath, indexDigest, platformDigests
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(digest[:]))
}
