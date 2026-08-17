package cipolicy_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type workflow struct {
	Permissions map[string]string `yaml:"permissions"`
	Concurrency struct {
		Group            string `yaml:"group"`
		CancelInProgress string `yaml:"cancel-in-progress"`
	} `yaml:"concurrency"`
	Jobs map[string]workflowJob `yaml:"jobs"`
}

type workflowJob struct {
	Name  string            `yaml:"name"`
	Env   map[string]string `yaml:"env"`
	Steps []workflowStep    `yaml:"steps"`
}

type workflowStep struct {
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}

func TestWorkflowExposesFourStableLeastPrivilegeChecks(t *testing.T) {
	config, raw := loadWorkflow(t)

	wantJobs := map[string]string{
		"quality":      "Go quality",
		"minimum-go":   "Minimum Go",
		"mcp-protocol": "MCP protocol",
		"container":    "Container",
	}
	if len(config.Jobs) != len(wantJobs) {
		t.Fatalf("workflow has %d jobs, want %d", len(config.Jobs), len(wantJobs))
	}
	for id, name := range wantJobs {
		job, ok := config.Jobs[id]
		if !ok {
			t.Errorf("workflow is missing stable job %q", id)
			continue
		}
		if job.Name != name {
			t.Errorf("job %q name = %q, want %q", id, job.Name, name)
		}
		assertCheckoutDoesNotPersistCredentials(t, id, job)
	}

	if len(config.Permissions) != 1 || config.Permissions["contents"] != "read" {
		t.Errorf("workflow permissions = %#v, want contents: read only", config.Permissions)
	}
	if config.Concurrency.Group != "${{ github.workflow }}-${{ github.event.pull_request.number || github.run_id }}" {
		t.Errorf("unexpected concurrency group %q", config.Concurrency.Group)
	}
	if config.Concurrency.CancelInProgress != "${{ github.event_name == 'pull_request' }}" {
		t.Errorf("unexpected cancel-in-progress policy %q", config.Concurrency.CancelInProgress)
	}

	for _, forbidden := range []string{"pull_request_target", "secrets."} {
		if strings.Contains(raw, forbidden) {
			t.Errorf("workflow contains forbidden %q", forbidden)
		}
	}
	assertActionsArePinned(t, raw)
}

func TestWorkflowRunsEachEvidenceLaneOnceAndRetainsSanitizedArtifacts(t *testing.T) {
	config, _ := loadWorkflow(t)

	assertRunContains(t, config.Jobs["quality"], "make ci-quality COVERAGE_DIR=\"${RUNNER_TEMP}/coverage\"")
	assertRunContains(t, config.Jobs["minimum-go"], "make ci-minimum")
	if got := config.Jobs["minimum-go"].Env["GOTOOLCHAIN"]; got != "local" {
		t.Errorf("minimum Go GOTOOLCHAIN = %q, want local", got)
	}
	assertRunContains(t, config.Jobs["mcp-protocol"], "make protocol-gates")
	assertRunNotContains(t, config.Jobs["mcp-protocol"], "go build")
	assertRunContains(t, config.Jobs["container"], "make container-verify")

	assertArtifact(t, config.Jobs["quality"], "go-coverage", "${{ runner.temp }}/coverage")
	assertArtifact(t, config.Jobs["mcp-protocol"], "mcp-protocol-gates", "${{ runner.temp }}/mcp-gate-artifacts")
}

func loadWorkflow(t *testing.T) (workflow, string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var config workflow
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	return config, string(data)
}

func assertCheckoutDoesNotPersistCredentials(t *testing.T, jobID string, job workflowJob) {
	t.Helper()
	for _, step := range job.Steps {
		if strings.HasPrefix(step.Uses, "actions/checkout@") {
			if value, ok := step.With["persist-credentials"].(bool); !ok || value {
				t.Errorf("job %q checkout persist-credentials = %#v, want false", jobID, step.With["persist-credentials"])
			}
			return
		}
	}
	t.Errorf("job %q does not check out the repository", jobID)
}

func assertActionsArePinned(t *testing.T, workflow string) {
	t.Helper()
	pinned := regexp.MustCompile(`^[[:space:]]+(?:- )?uses: [^@[:space:]]+@[0-9a-f]{40} # v[^[:space:]]+$`)
	for _, line := range strings.Split(workflow, "\n") {
		if strings.Contains(line, "uses:") && !pinned.MatchString(line) {
			t.Errorf("Action is not pinned to a full SHA with a version comment: %s", line)
		}
	}
}

func assertRunContains(t *testing.T, job workflowJob, expected string) {
	t.Helper()
	for _, step := range job.Steps {
		if strings.Contains(step.Run, expected) {
			return
		}
	}
	t.Errorf("job %q does not run %q", job.Name, expected)
}

func assertRunNotContains(t *testing.T, job workflowJob, forbidden string) {
	t.Helper()
	for _, step := range job.Steps {
		if strings.Contains(step.Run, forbidden) {
			t.Errorf("job %q unexpectedly runs %q", job.Name, forbidden)
		}
	}
}

func assertArtifact(t *testing.T, job workflowJob, name, path string) {
	t.Helper()
	for _, step := range job.Steps {
		if !strings.HasPrefix(step.Uses, "actions/upload-artifact@") || step.With["name"] != name {
			continue
		}
		if step.With["path"] != path {
			t.Errorf("artifact %q path = %#v, want %q", name, step.With["path"], path)
		}
		if step.With["retention-days"] != 30 {
			t.Errorf("artifact %q retention-days = %#v, want 30", name, step.With["retention-days"])
		}
		return
	}
	t.Errorf("job %q does not upload artifact %q", job.Name, name)
}
