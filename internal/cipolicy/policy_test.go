package cipolicy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCanonicalWorkflowMatchesLocalQualityCommands(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	workflow := readPolicyFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))
	makefile := readPolicyFile(t, filepath.Join(root, "Makefile"))
	workflows := workflowNames(t, filepath.Join(root, ".github", "workflows"))
	for _, problem := range policyErrors(workflow, makefile, workflows) {
		t.Error(problem)
	}
}

func TestPolicyRejectsAcceptanceCriticalMutations(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	workflow := readPolicyFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))
	makefile := readPolicyFile(t, filepath.Join(root, "Makefile"))
	containerStart := strings.Index(workflow, "\n  container:\n")
	if containerStart < 0 {
		t.Fatal("fixture workflow has no container lane")
	}
	if problems := policyErrors(workflow, makefile, []string{"ci.yml"}); len(problems) != 0 {
		t.Fatalf("baseline policy invalid: %v", problems)
	}
	for _, test := range []struct {
		name      string
		workflow  string
		makefile  string
		workflows []string
	}{
		{name: "duplicate yaml workflow", workflow: workflow, makefile: makefile, workflows: []string{"ci.yml", "duplicate.yaml"}},
		{name: "write permission", workflow: strings.Replace(workflow, "permissions:\n  contents: read", "permissions:\n  contents: read\n  actions: write", 1), makefile: makefile, workflows: []string{"ci.yml"}},
		{name: "workflow write-all", workflow: strings.Replace(workflow, "permissions:\n  contents: read", "permissions: write-all", 1), makefile: makefile, workflows: []string{"ci.yml"}},
		{name: "job write-all", workflow: strings.Replace(workflow, "  test:\n", "  test:\n    permissions: write-all\n", 1), makefile: makefile, workflows: []string{"ci.yml"}},
		{name: "job mapped write", workflow: strings.Replace(workflow, "  test:\n", "  test:\n    permissions:\n      contents: write\n", 1), makefile: makefile, workflows: []string{"ci.yml"}},
		{name: "missing container lane", workflow: workflow[:containerStart], makefile: makefile, workflows: []string{"ci.yml"}},
		{name: "missing local container target", workflow: workflow, makefile: strings.Replace(makefile, "\ncontainer-verify:", "\nremoved-container-verify:", 1), workflows: []string{"ci.yml"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if problems := policyErrors(test.workflow, test.makefile, test.workflows); len(problems) == 0 {
				t.Fatal("forbidden policy mutation passed")
			}
		})
	}
}

func TestWritePermissionDetectionScopesAndSyntax(t *testing.T) {
	for _, workflow := range []string{
		"permissions: write-all\n",
		"permissions:\n  contents: write\n",
		"jobs:\n  test:\n    permissions: write-all\n",
		"jobs:\n  test:\n    permissions: {contents: write}\n",
	} {
		if problems := permissionErrors(workflow); len(problems) == 0 {
			t.Fatalf("write permission accepted:\n%s", workflow)
		}
	}
	for _, workflow := range []string{
		"permissions: read-all\n",
		"permissions:\n  contents: read\n",
		"jobs:\n  test:\n    permissions: {}\n",
	} {
		if problems := permissionErrors(workflow); len(problems) != 0 {
			t.Fatalf("read-only permissions rejected: %v\n%s", problems, workflow)
		}
	}
}

func policyErrors(workflow, makefile string, workflows []string) []string {
	var problems []string
	if len(workflows) != 1 || workflows[0] != "ci.yml" {
		problems = append(problems, fmt.Sprintf("canonical workflows=%v", workflows))
	}
	for _, target := range []string{"format", "tidy", "module-verify", "race", "vet", "staticcheck", "govulncheck", "fuzz-smoke", "coverage", "verify", "protocol-gates", "container-verify", "ci-minimum"} {
		if !strings.Contains(makefile, "\n"+target+":") {
			problems = append(problems, fmt.Sprintf("Makefile missing %s target", target))
		}
		if !strings.Contains(workflow, "make "+target) {
			problems = append(problems, fmt.Sprintf("workflow missing make %s", target))
		}
	}
	for _, required := range []string{"push:\n    branches:\n      - main", "pull_request:", "go-version: '1.25.0'", "go-version: '1.26.6'", "GOTOOLCHAIN: local", "name: go-coverage", "permissions:\n  contents: read"} {
		if !strings.Contains(workflow, required) {
			problems = append(problems, fmt.Sprintf("workflow missing %q", required))
		}
	}
	for _, pin := range []string{"STATICCHECK_VERSION := v0.7.0", "GOVULNCHECK_VERSION := v1.7.0"} {
		if !strings.Contains(makefile, pin) {
			problems = append(problems, fmt.Sprintf("Makefile missing %q", pin))
		}
	}
	lower := strings.ToLower(workflow + makefile)
	for _, forbidden := range []string{"${{ secrets.", "codecov", "coveralls", "coverage-threshold", "--fail-under"} {
		if strings.Contains(lower, forbidden) {
			problems = append(problems, fmt.Sprintf("CI contains forbidden policy %q", forbidden))
		}
	}
	problems = append(problems, permissionErrors(workflow)...)
	return problems
}

func permissionErrors(workflow string) []string {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(workflow), &document); err != nil {
		return []string{"workflow YAML is invalid"}
	}
	root := documentRoot(&document)
	if root == nil {
		return []string{"workflow YAML has no mapping document"}
	}
	var problems []string
	if grantsWrite(mappingValue(root, "permissions")) {
		problems = append(problems, "workflow grants a write permission")
	}
	jobs := mappingValue(root, "jobs")
	if jobs == nil || jobs.Kind != yaml.MappingNode {
		return problems
	}
	for index := 0; index+1 < len(jobs.Content); index += 2 {
		jobName, job := jobs.Content[index], jobs.Content[index+1]
		if grantsWrite(mappingValue(job, "permissions")) {
			problems = append(problems, fmt.Sprintf("job %q grants a write permission", jobName.Value))
		}
	}
	return problems
}

func documentRoot(document *yaml.Node) *yaml.Node {
	if document == nil || document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		return nil
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	return root
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
}

func grantsWrite(permissions *yaml.Node) bool {
	if permissions == nil {
		return false
	}
	if permissions.Kind == yaml.AliasNode {
		return grantsWrite(permissions.Alias)
	}
	switch permissions.Kind {
	case yaml.ScalarNode:
		return strings.EqualFold(strings.TrimSpace(permissions.Value), "write-all")
	case yaml.MappingNode:
		for index := 1; index < len(permissions.Content); index += 2 {
			value := permissions.Content[index]
			if value.Kind == yaml.ScalarNode && strings.EqualFold(strings.TrimSpace(value.Value), "write") {
				return true
			}
		}
	}
	return false
}

func workflowNames(t *testing.T, directory string) []string {
	t.Helper()
	var names []string
	for _, pattern := range []string{"*.yml", "*.yaml"} {
		paths, err := filepath.Glob(filepath.Join(directory, pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			names = append(names, filepath.Base(path))
		}
	}
	return names
}

func readPolicyFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
