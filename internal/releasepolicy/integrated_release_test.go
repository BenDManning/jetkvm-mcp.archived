package releasepolicy_test

import (
	"crypto/sha256"
	"encoding/hex"
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

func TestIntegratedReleaseWorkflowHasOneLeastPrivilegePublicationPath(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	action, err := os.ReadFile(filepath.Join(root, ".github", "actions", "finalize-release", "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var actionDocument map[string]any
	if err := yaml.Unmarshal(action, &actionDocument); err != nil {
		t.Fatalf("parse finalize-release action: %v", err)
	}
	text := string(data) + "\n" + string(action)
	var workflow struct {
		On          map[string]any    `yaml:"on"`
		Permissions map[string]string `yaml:"permissions"`
		Jobs        map[string]struct {
			Permissions map[string]string `yaml:"permissions"`
			If          string            `yaml:"if"`
			Needs       any               `yaml:"needs"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	for _, trigger := range []string{"pull_request", "push", "workflow_dispatch"} {
		if _, ok := workflow.On[trigger]; !ok {
			t.Errorf("integrated release workflow is missing %s", trigger)
		}
	}
	if !reflect.DeepEqual(workflow.Permissions, map[string]string{"contents": "read"}) {
		t.Errorf("default workflow permissions = %#v", workflow.Permissions)
	}
	for _, name := range []string{"stage-native", "stage-container"} {
		job, ok := workflow.Jobs[name]
		if !ok {
			t.Errorf("%s job is missing", name)
			continue
		}
		if len(job.Permissions) > 0 && !reflect.DeepEqual(job.Permissions, map[string]string{"contents": "read"}) {
			t.Errorf("%s permissions = %#v", name, job.Permissions)
		}
	}
	prepare := workflow.Jobs["prepare"]
	if !reflect.DeepEqual(prepare.Permissions, map[string]string{"contents": "read"}) {
		t.Errorf("prepare permissions = %#v", prepare.Permissions)
	}
	if prepare.Needs != nil {
		t.Errorf("prepare inherits the push-only admission dependency: %#v", prepare.Needs)
	}
	publishAdmission, ok := workflow.Jobs["verify-publish-commit"]
	if !ok {
		t.Fatal("verify-publish-commit job is missing")
	}
	if publishAdmission.If != "github.event_name == 'push'" {
		t.Errorf("verify-publish-commit condition = %q", publishAdmission.If)
	}
	if !reflect.DeepEqual(publishAdmission.Permissions, map[string]string{"checks": "read", "contents": "read"}) {
		t.Errorf("verify-publish-commit permissions = %#v", publishAdmission.Permissions)
	}
	rehearse, ok := workflow.Jobs["rehearse"]
	if !ok {
		t.Fatal("rehearse job is missing")
	}
	if !reflect.DeepEqual(rehearse.Permissions, map[string]string{
		"attestations": "write",
		"contents":     "read",
		"id-token":     "write",
	}) {
		t.Errorf("rehearse permissions = %#v", rehearse.Permissions)
	}
	publish, ok := workflow.Jobs["publish"]
	if !ok {
		t.Fatal("publish job is missing")
	}
	if !reflect.DeepEqual(publish.Permissions, map[string]string{
		"attestations": "write",
		"contents":     "write",
		"id-token":     "write",
		"packages":     "write",
	}) {
		t.Errorf("publish permissions = %#v", publish.Permissions)
	}
	if !reflect.DeepEqual(publish.Needs, []any{"prepare", "verify-publish-commit", "stage-native", "stage-container"}) {
		t.Errorf("publish dependencies = %#v", publish.Needs)
	}

	for _, expected := range []string{
		"tags:", "- 'v*'", "needs.prepare.outputs.release_mode == 'rehearse'", "needs.prepare.outputs.release_mode == 'publish'",
		"git cat-file -t \"${GITHUB_REF}\"", "GITHUB_REF_PROTECTED", "GITHUB_ACTOR", "GITHUB_RUN_ATTEMPT", "git merge-base --is-ancestor", "check-runs",
		"make release-subjects", "make container-release-snapshot",
		"persist-credentials: false", "cache: false",
		"BUILDX_VERSION: v0.36.1", "BUILDX_SHA256: 48af8a397ebd60178778bf63611dbcebe5f5e7a9be90eb9147b24b9587455778",
		"moby/buildkit:v0.32.2@sha256:28a898719c18a33f4e8000685287fa36fd0dd9560c6440227d3a732d79bb41d8",
		"actions/upload-artifact@", "actions/download-artifact@",
		"cosign sign-blob --yes",
		"subject-checksums:", "subject-path:", "sbom-path:",
		"scripts/publish-release.sh",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("integrated workflow does not contain %q", expected)
		}
	}
	for _, forbidden := range []string{
		"pull_request_target", "secrets.", "docker/login-action", "--push", "cache-from", "cache-to",
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("integrated workflow unexpectedly contains %q", forbidden)
		}
	}
	pinned := regexp.MustCompile(`^[[:space:]]+(?:- )?uses: [^@[:space:]]+@[0-9a-f]{40} # v[^[:space:]]+$`)
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "uses:") && !strings.Contains(line, "uses: ./.github/actions/") && !pinned.MatchString(line) {
			t.Errorf("Action is not pinned to a full SHA with a version comment: %s", line)
		}
	}
}

func TestDependencyPolicyRecordsORASLicense(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "dependency-policy.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "ORAS is licensed under Apache License 2.0") {
		t.Error("dependency policy does not identify the ORAS license")
	}
}

func TestReleasePreparationRendersCompleteMutationFreePublicationInputs(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	temporary := t.TempDir()
	nativeDir := filepath.Join(temporary, "native")
	containerDir := filepath.Join(temporary, "container")
	outputDir := filepath.Join(temporary, "output")
	for _, directory := range []string{nativeDir, containerDir, outputDir} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeReleasePreparationFixture(t, nativeDir, containerDir)
	ledger := filepath.Join(temporary, "ledger.json")
	writeTestFile(t, ledger, `{"entries":[{"id":"rehearsal-fixture","evidenceClass":"release_rehearsal","result":"pass","authorizationReference":"not-applicable","approvedUTCWindow":"not-applicable","observedOn":"2026-08-19","jetkvm":{"model":"not-observed","firmwareVersion":"not-observed"},"server":{"sourceRef":"0123456789abcdef","version":"v0.0.0-rehearsal.0123456789ab"},"runtime":{"os":"not-observed","architecture":"not-observed"},"ffmpeg":{"identity":"not-observed"},"mcp":{"transport":"not-observed","client":"not-observed"},"attachedHost":{"fixture":"not-observed","os":"not-observed","architecture":"not-observed"},"checks":["publication preparation"],"limitations":["Synthetic non-publishing fixture; no physical qualification."]}]}`)
	tagNotes := filepath.Join(temporary, "tag-notes.json")
	writeTestFile(t, tagNotes, `{"summary":"Integrated release rehearsal.","compatibilityAndMigration":[],"securityRelevantFixes":[],"knownLimitations":["Non-publishing rehearsal."],"supersededVersions":[],"retractedVersions":[]}`)

	command := exec.Command("bash", "-c", `source "$1"; prepare_release_materials "$2" "$3" "$4" "$5" "$6" release_rehearsal`, "bash", filepath.Join(root, "scripts", "release-publication-materials.sh"), nativeDir, containerDir, ledger, tagNotes, outputDir)
	command.Env = append(os.Environ(),
		"GITHUB_REPOSITORY=BenDManning/jetkvm-mcp",
		"GITHUB_SHA=0123456789abcdef",
		"RELEASE_REF=refs/heads/main",
		"RELEASE_TAG=v0.0.0-rehearsal.0123456789ab",
		"RELEASE_WORKFLOW=BenDManning/jetkvm-mcp/.github/workflows/release.yml",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("prepare release materials: %v\n%s", err, output)
	}

	record, err := os.ReadFile(filepath.Join(containerDir, "release-record.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"tag": "v0.0.0-rehearsal.0123456789ab"`, `"commit": "0123456789abcdef"`, `"physical_qualification": "rehearsal-fixture"`, `"published": false`} {
		if !strings.Contains(string(record), expected) {
			t.Errorf("release record does not contain %s\n%s", expected, record)
		}
	}
	notes, err := os.ReadFile(filepath.Join(outputDir, "release-notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Integrated release rehearsal.", "rehearsal-fixture", "## Artifact digests", "## Verification"} {
		if !strings.Contains(string(notes), expected) {
			t.Errorf("release notes do not contain %q", expected)
		}
	}
	assets, err := os.ReadFile(filepath.Join(outputDir, "release-assets.txt"))
	if err != nil {
		t.Fatal(err)
	}
	assetLines := strings.Fields(string(assets))
	if len(assetLines) != 18 {
		t.Errorf("release asset count = %d, want 18\n%s", len(assetLines), assets)
	}
	for _, expected := range []string{"checksums.txt", "linux-amd64.spdx.json", "release-record.json", "sbom-linux-arm64.sigstore.json"} {
		if !strings.Contains(string(assets), expected) {
			t.Errorf("release assets do not contain %q", expected)
		}
	}
}

func TestReleasePreparationRejectsMismatchedQualification(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	temporary := t.TempDir()
	nativeDir := filepath.Join(temporary, "native")
	containerDir := filepath.Join(temporary, "container")
	outputDir := filepath.Join(temporary, "output")
	for _, directory := range []string{nativeDir, containerDir, outputDir} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeReleasePreparationFixture(t, nativeDir, containerDir)
	ledger := filepath.Join(temporary, "ledger.json")
	writeTestFile(t, ledger, `{"entries":[{"id":"wrong-candidate","evidenceClass":"release_rehearsal","result":"pass","server":{"sourceRef":"different-commit","version":"v0.0.0-rehearsal.0123456789ab"}}]}`)
	tagNotes := filepath.Join(temporary, "tag-notes.json")
	writeTestFile(t, tagNotes, `{"summary":"Integrated release rehearsal.","compatibilityAndMigration":[],"securityRelevantFixes":[],"knownLimitations":[],"supersededVersions":[],"retractedVersions":[]}`)

	command := exec.Command("bash", "-c", `source "$1"; prepare_release_materials "$2" "$3" "$4" "$5" "$6" release_rehearsal`, "bash", filepath.Join(root, "scripts", "release-publication-materials.sh"), nativeDir, containerDir, ledger, tagNotes, outputDir)
	command.Env = append(os.Environ(),
		"GITHUB_REPOSITORY=BenDManning/jetkvm-mcp",
		"GITHUB_SHA=0123456789abcdef",
		"RELEASE_REF=refs/heads/main",
		"RELEASE_TAG=v0.0.0-rehearsal.0123456789ab",
		"RELEASE_WORKFLOW=BenDManning/jetkvm-mcp/.github/workflows/release.yml",
	)
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("mismatched qualification unexpectedly passed\n%s", output)
	}
	if !strings.Contains(string(output), "exactly one matching passing qualification is required") {
		t.Fatalf("unexpected mismatch error:\n%s", output)
	}
}

func writeReleasePreparationFixture(t *testing.T, nativeDir, containerDir string) {
	t.Helper()
	for _, name := range []string{
		"jetkvm-mcp_linux_amd64.tar.gz", "jetkvm-mcp_linux_arm64.tar.gz",
		"jetkvm-mcp_linux_amd64.tar.gz.spdx.json", "jetkvm-mcp_linux_arm64.tar.gz.spdx.json",
		"checksums.txt", "checksums.txt.sigstore.json", "provenance.sigstore.json",
	} {
		writeTestFile(t, filepath.Join(nativeDir, name), "fixture\n")
	}
	manifest := "{\"schemaVersion\":2}\n"
	digest := sha256.Sum256([]byte(manifest))
	writeTestFile(t, filepath.Join(containerDir, "image-manifest.json"), manifest)
	writeTestFile(t, filepath.Join(containerDir, "manifest-digests.json"), fmt.Sprintf(`{"manifest_digest":"sha256:%s"}`, hex.EncodeToString(digest[:])))
	for _, name := range []string{
		"image-manifest-linux-amd64.json", "image-manifest-linux-arm64.json",
		"linux-amd64.spdx.json", "linux-arm64.spdx.json", "image-manifest.sigstore.json",
		"provenance.sigstore.json", "sbom-linux-amd64.sigstore.json", "sbom-linux-arm64.sigstore.json",
	} {
		writeTestFile(t, filepath.Join(containerDir, name), "fixture\n")
	}
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPublicationCoordinatorConsumesVersionBeforePublishingAndMovesLatestLast(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "publish-release.sh"))
	if err != nil {
		t.Fatal(err)
	}
	materials, err := os.ReadFile(filepath.Join("..", "..", "scripts", "release-publication-materials.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data) + "\n" + string(materials)
	ordered := []string{
		`gh api "repos/${GITHUB_REPOSITORY}/releases/tags/${release_tag}"`,
		`oras manifest fetch "$image:$release_tag"`,
		`gh release create "$release_tag" --draft`,
		`oras cp --from-oci-layout`,
		`gh release edit "$release_tag" --draft=false`,
		`oras tag "$image@$manifest_digest" latest`,
	}
	previous := -1
	for _, marker := range ordered {
		index := strings.Index(text, marker)
		if index < 0 {
			t.Errorf("publication coordinator does not contain %q", marker)
			continue
		}
		if index <= previous {
			t.Errorf("publication coordinator orders %q incorrectly", marker)
		}
		previous = index
	}
	for _, expected := range []string{
		"RELEASE_MODE", "rehearse", "publish", "refs/tags/", "GITHUB_SHA",
		"isImmutable", "manifest-digests.json", "publication-plan.json",
		"checksums.txt.sigstore.json", "provenance.sigstore.json",
		"RELEASE_WORKFLOW", "--source-ref", "--source-digest",
		"physical_qualification", "securityRelevantFixes", "knownLimitations",
		"supersededVersions", "retractedVersions",
		"## Artifact digests", `done < "$native_dir/checksums.txt"`,
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("publication coordinator does not contain %q", expected)
		}
	}
	latest := strings.Index(text, `oras tag "$image@$manifest_digest" latest`)
	if latest >= 0 {
		tail := text[latest+len(`oras tag "$image@$manifest_digest" latest`):]
		if strings.Contains(tail, "oras resolve") || strings.Contains(tail, "gh release") {
			t.Error("publication coordinator performs a fallible remote operation after moving latest")
		}
	}
}

func TestPublicationStateMachineRehearsesEveryFailureBoundaryWithoutMutation(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, path := range []string{
		filepath.Join(root, "scripts", "publish-release.sh"),
		filepath.Join(root, "scripts", "verify-release-publication-state.sh"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "source \"$script_dir/release-publication-state.sh\"") {
			t.Errorf("%s does not use the shared publication state machine", path)
		}
	}
	command := exec.Command("bash", filepath.Join(root, "scripts", "verify-release-publication-state.sh"))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("publication state rehearsal: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "verified mutation-free publication state machine and failure outcomes" {
		t.Fatalf("unexpected publication state rehearsal output: %s", output)
	}
}

func TestPublishedConsumerCommandsConstrainIntegratedWorkflowIdentity(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, ".github/workflows/native-release.yml") || strings.Contains(text, ".github/workflows/container-release.yml") {
		t.Error("README still names a split release workflow")
	}
	for _, expected := range []string{
		".github/workflows/release.yml", "refs/tags/vX.Y.Z", "--cert-identity",
		"--source-ref", "--source-digest", "ghcr.io/bendmanning/jetkvm-mcp@sha256:",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("README release verification does not contain %q", expected)
		}
	}
}
