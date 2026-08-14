package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestWriteSummaryHasStableSanitizedArtifactSchema(t *testing.T) {
	summary := gateSummary{
		SchemaVersion: 1,
		Conformance: sourceSummary{
			Package: "@modelcontextprotocol/conformance",
			Version: "0.2.0-alpha.11",
			Commit:  "c321dd32035556e6769d3724a8ee97d87c3faaac",
		},
		Inspector: sourceSummary{
			Package: "@modelcontextprotocol/inspector",
			Version: "2.2.0",
			Commit:  "672f9f41c548487a468b9e7007d2f9de14da5a69",
		},
		SpecVersion:      "2026-07-28",
		ExpectedFailures: []string{},
		Checks:           []checkSummary{{ID: "conformance/tools-list", Status: "passed"}},
		Outcome:          "passed",
	}
	artifactDir := filepath.Join(t.TempDir(), "private-local-component")
	if err := os.MkdirAll(artifactDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeSummary(artifactDir, summary); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(artifactDir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(document))
	for key := range document {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	wantKeys := []string{"checks", "conformance", "expectedFailures", "inspector", "outcome", "schemaVersion", "specVersion"}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Fatalf("summary keys = %v", keys)
	}
	if !bytes.Equal(bytes.TrimSpace(document["expectedFailures"]), []byte("[]")) {
		t.Fatalf("expectedFailures = %s", document["expectedFailures"])
	}
	text := string(data)
	for _, forbidden := range []string{
		"private-local-component",
		"jetkvm.example.invalid",
		"MCP_INSPECTOR_API_TOKEN",
		"password",
		"credential",
		"image payload",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("summary contains forbidden marker %q", forbidden)
		}
	}
}

func TestRunBinaryRejectsSuccessfulStderr(t *testing.T) {
	if _, err := runBinaryRejectStderr(context.Background(), "sh", nil, nil, "-c", "printf warning >&2"); err == nil {
		t.Fatal("successful command stderr was accepted")
	}
	output, err := runBinary(context.Background(), "sh", nil, nil, "-c", "printf ok")
	if err != nil || string(output) != "ok" {
		t.Fatalf("clean command output=%q err=%v", output, err)
	}
}

func TestConformanceDiagnosticsWriterUsesStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	writer := conformanceDiagnosticsWriter(&stdout, &stderr)
	if _, err := io.WriteString(writer, "gate diagnostic"); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("conformance diagnostic polluted stdout: %q", stdout.String())
	}
	if got := stderr.String(); got != "gate diagnostic" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestWriteInspectorConfigPinsModernProtocolEra(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inspector.json")
	if err := writeInspectorConfig(path, "/tmp/jetkvm-mcp", "/tmp/config.yaml", "http://127.0.0.1:1234/mcp"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		MCPServers map[string]struct {
			Type        string   `json:"type"`
			Command     string   `json:"command"`
			Args        []string `json:"args"`
			URL         string   `json:"url"`
			ProtocolEra string   `json:"protocolEra"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	stdio := document.MCPServers["fixture-stdio"]
	http := document.MCPServers["fixture-http"]
	if stdio.Type != "stdio" || stdio.Command != "/tmp/jetkvm-mcp" || !reflect.DeepEqual(stdio.Args, []string{"--config", "/tmp/config.yaml"}) || stdio.ProtocolEra != "modern" {
		t.Fatalf("stdio Inspector config = %#v", stdio)
	}
	if http.Type != "streamable-http" || http.URL != "http://127.0.0.1:1234/mcp" || http.ProtocolEra != "modern" {
		t.Fatalf("HTTP Inspector config = %#v", http)
	}
}

func TestWriteConformanceArtifactOmitsRawOutputAndEndpoints(t *testing.T) {
	directory := t.TempDir()
	if err := writeConformanceArtifact(directory, "http-header-validation", []string{"reviewed-check"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(directory, "http-header-validation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(document))
	for key := range document {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if want := []string{"allowedSkippedChecks", "scenario", "schemaVersion", "status"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("artifact keys = %v, want %v", keys, want)
	}
	for _, forbidden := range []string{"127.0.0.1", "originHeader", "checks.json", "private"} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("artifact contains forbidden marker %q: %s", forbidden, data)
		}
	}
}
