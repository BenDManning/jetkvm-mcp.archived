package mcpserver

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type expectedADR struct {
	file  string
	title string
}

var architectureDecisions = []expectedADR{
	{file: "0001-mcp-transports-and-authentication.md", title: "MCP transports and authentication boundary"},
	{file: "0002-fresh-in-process-webrtc-sessions.md", title: "Fresh in-process WebRTC sessions"},
	{file: "0003-ffmpeg-screenshot-decoding.md", title: "FFmpeg subprocess screenshot decoding"},
	{file: "0004-virtual-media-integrity-and-cleanup.md", title: "Virtual-media integrity and cleanup"},
	{file: "0005-local-only-raw-rpc.md", title: "Keep raw RPC local and outside MCP"},
	{file: "0006-virtual-media-url-origin-boundary.md", title: "Deny virtual-media URL mounts outside configured origins"},
	{file: "0007-same-origin-browser-http.md", title: "Keep browser Streamable HTTP same-origin"},
}

func TestArchitectureDecisionIndexAndStatuses(t *testing.T) {
	index := readRepositoryDocument(t, filepath.Join("docs", "adr", "README.md"))
	template := readRepositoryDocument(t, filepath.Join("docs", "adr", "template.md"))
	allowedStatuses := []string{"proposed", "accepted", "superseded", "deferred"}
	for _, status := range allowedStatuses {
		if !strings.Contains(index, "`"+status+"`") || !strings.Contains(template, status) {
			t.Errorf("ADR status %q is not defined by both the index and template", status)
		}
	}
	entries, err := os.ReadDir(filepath.Join(repositoryRoot(t), "docs", "adr"))
	if err != nil {
		t.Fatal(err)
	}
	recordPattern := regexp.MustCompile(`^[0-9]{4}-.*\.md$`)
	var recordFiles []string
	for _, entry := range entries {
		if recordPattern.MatchString(entry.Name()) {
			recordFiles = append(recordFiles, entry.Name())
		}
	}
	if len(recordFiles) != len(architectureDecisions) {
		t.Fatalf("ADR files = %v, want exactly %d indexed records", recordFiles, len(architectureDecisions))
	}

	for _, expected := range architectureDecisions {
		number := strings.SplitN(expected.file, "-", 2)[0]
		row := "| [" + number + "](" + expected.file + ") | " + expected.title + " | accepted |"
		if !strings.Contains(index, row) {
			t.Errorf("ADR index does not list accepted record %q", expected.file)
		}
		document := readRepositoryDocument(t, filepath.Join("docs", "adr", expected.file))
		if !strings.Contains(document, "# "+strings.TrimSuffix(expected.file, ".md")+": "+expected.title) {
			t.Errorf("ADR %q has an unexpected title", expected.file)
		}
		if !strings.Contains(document, "Status: accepted") {
			t.Errorf("ADR %q is not accepted", expected.file)
		}
		statusPattern := regexp.MustCompile(`(?m)^Status: ([a-z]+)$`)
		matches := statusPattern.FindAllStringSubmatch(document, -1)
		if len(matches) != 1 || !containsString(allowedStatuses, matches[0][1]) {
			t.Errorf("ADR %q status declarations = %v", expected.file, matches)
		}
	}
}

func TestArchitectureDecisionLogIsOwnedAndDiscoverable(t *testing.T) {
	productContract := readRepositoryDocument(t, filepath.Join("docs", "product-contract.md"))
	if !strings.Contains(productContract, "[architecture decision index](adr/README.md)") {
		t.Fatal("product contract does not assign architectural rationale to the ADR index")
	}
	repositoryGuide := readRepositoryDocument(t, "AGENTS.md")
	if !strings.Contains(repositoryGuide, "[docs/adr/README.md](docs/adr/README.md)") {
		t.Fatal("repository guide does not direct architecture changes through the ADR index")
	}
}

func TestArchitectureDecisionsContainRationaleEvidenceAndRevisitTriggers(t *testing.T) {
	for _, expected := range architectureDecisions {
		documentPath := filepath.Join("docs", "adr", expected.file)
		document := readRepositoryDocument(t, documentPath)
		for _, heading := range []string{
			"## Context", "## Decision", "## Rationale", "## Rejected alternatives", "## Consequences", "## Evidence", "## Revisit trigger",
		} {
			if !strings.Contains(document, heading) {
				t.Errorf("ADR %q is missing %q", expected.file, heading)
			}
		}
		if !strings.Contains(document, "**Rejected:**") {
			t.Errorf("ADR %q does not name a rejected alternative", expected.file)
		}
		if !strings.Contains(document, "Revisit when any of the following is true:") {
			t.Errorf("ADR %q does not define measurable revisit conditions", expected.file)
		}

		links := markdownRelativeLinks(document)
		var codeEvidence, testEvidence, contractEvidence bool
		for _, target := range links {
			resolved := filepath.Clean(filepath.Join(filepath.Dir(documentPath), target))
			if _, err := os.Stat(filepath.Join(repositoryRoot(t), resolved)); err != nil {
				t.Errorf("ADR %q link %q does not resolve: %v", expected.file, target, err)
				continue
			}
			codeEvidence = codeEvidence || strings.HasSuffix(target, ".go") && !strings.HasSuffix(target, "_test.go")
			testEvidence = testEvidence || strings.HasSuffix(target, "_test.go")
			contractEvidence = contractEvidence || strings.HasSuffix(target, "product-contract.md") || strings.HasSuffix(target, "threat-model.md")
		}
		if !codeEvidence || !testEvidence || !contractEvidence {
			t.Errorf("ADR %q evidence links code=%v tests=%v contract=%v", expected.file, codeEvidence, testEvidence, contractEvidence)
		}
	}
}

func readRepositoryDocument(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), path))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func markdownRelativeLinks(document string) []string {
	pattern := regexp.MustCompile(`\[[^]]+\]\(([^)]+)\)`)
	var links []string
	for _, match := range pattern.FindAllStringSubmatch(document, -1) {
		target := match[1]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "#") {
			continue
		}
		target, _, _ = strings.Cut(target, "#")
		if target != "" {
			links = append(links, target)
		}
	}
	return links
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
