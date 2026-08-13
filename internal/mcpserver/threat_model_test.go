package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestThreatModelInputOutputFieldWalkThrough(t *testing.T) {
	document := readThreatModel(t)
	fixture, err := os.ReadFile(filepath.Join("testdata", "tool-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Tools []struct {
			Name        string `json:"name"`
			InputSchema struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"inputSchema"`
			OutputSchema struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"outputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(fixture, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Tools) == 0 {
		t.Fatal("tool manifest is empty")
	}

	for _, tool := range manifest.Tools {
		if !strings.Contains(document, "| `"+tool.Name+"` |") {
			t.Errorf("threat model has no consequence row for %q", tool.Name)
		}
		for _, field := range sortedSchemaFields(tool.InputSchema.Properties, tool.OutputSchema.Properties) {
			if !strings.Contains(document, "`"+field+"`") {
				t.Errorf("threat model does not classify manifest field %q", field)
			}
		}
	}

	for _, field := range []string{
		"--config", "--http", "--version", "--binary", "--device", "--method", "--params",
	} {
		if !strings.Contains(document, "`"+field+"`") {
			t.Errorf("threat model does not classify configuration or CLI field %q", field)
		}
	}
	for _, entryPoint := range []string{"| `debug rpc --method --params` |", "| `jetkvm-mcp-validate` |"} {
		if !strings.Contains(document, entryPoint) {
			t.Errorf("threat model has no consequence row containing %q", entryPoint)
		}
	}
}

func TestThreatModelThreatToControlTraceability(t *testing.T) {
	document := readThreatModel(t)
	for _, marker := range []string{
		"**Implemented control**",
		"**Deployment requirement**",
		"**Not implemented**",
	} {
		if !strings.Contains(document, marker) {
			t.Errorf("threat model is missing traceability marker %q", marker)
		}
	}
	lowerDocument := strings.ToLower(strings.Join(strings.Fields(document), " "))
	for _, marker := range []string{
		"SSRF",
		"DNS rebinding",
		"resource exhaustion",
		"disclosure",
		"build compromise",
		"uncertain mutation",
	} {
		if !strings.Contains(lowerDocument, strings.ToLower(marker)) {
			t.Errorf("threat model is missing traceability marker %q", marker)
		}
	}

	rowPattern := regexp.MustCompile(`(?m)^\| (T-[0-9]{2}) \|.*\[[^]]+\]\(\.\./[^)]+\).*$`)
	rows := rowPattern.FindAllStringSubmatch(document, -1)
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		if seen[row[1]] {
			t.Errorf("duplicate threat-register row %s", row[1])
		}
		seen[row[1]] = true
		for _, marker := range []string{"**Implemented control:**", "**Deployment requirement:**", "**Not implemented:**"} {
			if !strings.Contains(row[0], marker) {
				t.Errorf("threat-register row %s is missing %q", row[1], marker)
			}
		}
	}
	for number := 1; number <= 13; number++ {
		id := "T-" + twoDigits(number)
		if !seen[id] {
			t.Errorf("threat register has no code/test-backed row for %s", id)
		}
	}
}

func TestThreatModelAccountsForSchemaErrorsEchoingClientArguments(t *testing.T) {
	const privateArgument = "JETKVM-PRIVATE-ARGUMENT-7eea7c9f"
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(&recordingDevice{}, "test").Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: KeyboardToolName,
		Arguments: map[string]any{
			"device":    "lab",
			"operation": privateArgument,
		},
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	content, err := json.Marshal(result.Content)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !bytes.Contains(content, []byte(privateArgument)) {
		t.Fatalf("schema rejection no longer echoes the rejected argument: %#v", result)
	}
	if !strings.Contains(readThreatModel(t), "SDK input-schema failures can echo rejected client-supplied values") {
		t.Fatal("threat model does not classify client-supplied values echoed by SDK schema errors")
	}
}

func TestThreatModelRelativeEvidenceLinksExist(t *testing.T) {
	document := readThreatModel(t)
	linkPattern := regexp.MustCompile(`\[[^]]+\]\(([^)]+)\)`)
	for _, match := range linkPattern.FindAllStringSubmatch(document, -1) {
		target := match[1]
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "#") {
			continue
		}
		target, _, _ = strings.Cut(target, "#")
		if target == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join("../../docs", target)); err != nil {
			t.Errorf("threat model link %q does not resolve: %v", match[1], err)
		}
	}
}

func readThreatModel(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../docs/threat-model.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func sortedSchemaFields(schemas ...map[string]json.RawMessage) []string {
	unique := make(map[string]struct{})
	for _, schema := range schemas {
		for field := range schema {
			unique[field] = struct{}{}
		}
	}
	fields := make([]string, 0, len(unique))
	for field := range unique {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

func twoDigits(value int) string {
	return string([]byte{'0' + byte(value/10), '0' + byte(value%10)})
}
