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
	if prepare := workflow.Jobs["prepare"].Permissions; !reflect.DeepEqual(prepare, map[string]string{"checks": "read", "contents": "read"}) {
		t.Errorf("prepare permissions = %#v", prepare)
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

func TestPublicationCoordinatorConsumesVersionBeforePublishingAndMovesLatestLast(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "publish-release.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
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
