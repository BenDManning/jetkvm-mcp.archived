package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type expectedToolMetadata struct {
	title       string
	readOnly    bool
	destructive bool
	idempotent  bool
	openWorld   bool
	terms       []string
}

func TestSensitiveToolSchemasDescribeInputsAndResultsWithoutPrivateValues(t *testing.T) {
	tools := listedMetadataTools(t)
	for _, test := range []struct {
		tool        string
		inputField  string
		inputTerm   string
		outputField string
		outputTerm  string
	}{
		{CaptureScreenToolName, "max_width", "private png", "mimeType", "private png"},
		{KeyboardToolName, "text", "private us-ascii", "operation", "hid"},
		{MountVirtualMediaURLToolName, "url", "private http(s)", "sourceType", "redacted"},
		{MountVirtualMediaFileToolName, "path", "private relative", "sourceType", "redacted"},
		{UploadVirtualMediaFileToolName, "path", "private relative", "sourceType", "redacted"},
	} {
		t.Run(test.tool, func(t *testing.T) {
			tool := tools[test.tool]
			if tool == nil {
				t.Fatalf("missing tool %q", test.tool)
			}
			input := schemaDescriptions(t, tool.InputSchema)
			if !strings.Contains(strings.ToLower(input[test.inputField]), test.inputTerm) {
				t.Errorf("%s input %q description = %q, want %q", test.tool, test.inputField, input[test.inputField], test.inputTerm)
			}
			output := schemaDescriptions(t, tool.OutputSchema)
			if !strings.Contains(strings.ToLower(output[test.outputField]), test.outputTerm) {
				t.Errorf("%s output %q description = %q, want %q", test.tool, test.outputField, output[test.outputField], test.outputTerm)
			}
		})
	}
}

func schemaDescriptions(t *testing.T, schema any) map[string]string {
	t.Helper()
	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	descriptions := make(map[string]string, len(decoded.Properties))
	for name, property := range decoded.Properties {
		descriptions[name] = property.Description
	}
	return descriptions
}

func TestREADMEExplainsEveryPublicToolAndSafeRetryRules(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	document := string(readme)
	if !strings.Contains(document, "| Tool | Arguments and consequence | Retry |") {
		t.Fatal("README does not contain the public-tool argument/consequence/retry table")
	}
	for name := range map[string]expectedToolMetadata{
		ListDevicesToolName: {}, GetStatusToolName: {}, CaptureScreenToolName: {}, KeyboardToolName: {}, MouseToolName: {},
		GetVirtualMediaStatusToolName: {}, MountVirtualMediaURLToolName: {}, MountVirtualMediaFileToolName: {}, UnmountVirtualMediaToolName: {}, UploadVirtualMediaFileToolName: {}, VirtualMediaToolName: {},
		PressHostPowerButtonToolName: {}, ForceHostPowerOffToolName: {}, PressHostResetButtonToolName: {}, TurnHostDCPowerOnToolName: {}, TurnHostDCPowerOffToolName: {}, WakeHostUSBToolName: {}, WakeHostLANToolName: {},
	} {
		if !strings.Contains(document, "`"+name+"`") {
			t.Errorf("README does not cover %q", name)
		}
	}
	for _, requirement := range []string{
		"US-ASCII", "4096", "Do not blindly retry", "outcome: unknown", "jetkvm_list_devices", "jetkvm_get_status",
	} {
		if !strings.Contains(document, requirement) {
			t.Errorf("README does not state %q", requirement)
		}
	}
}

func TestEveryToolPublishesConsequenceCompleteMetadata(t *testing.T) {
	tools := listedMetadataTools(t)
	want := map[string]expectedToolMetadata{
		ListDevicesToolName:            {"List configured JetKVM devices", true, false, true, false, []string{"without contacting", "configured", "retry"}},
		GetStatusToolName:              {"Get JetKVM status", true, false, true, false, []string{"configured", "private", "retry"}},
		CaptureScreenToolName:          {"Capture host screen", true, false, true, false, []string{"configured", "private", "png", "retry"}},
		KeyboardToolName:               {"Send keyboard input", false, false, false, false, []string{"configured", "private", "hid", "unknown", "retry"}},
		MouseToolName:                  {"Send mouse input", false, false, false, false, []string{"configured", "hid", "unknown", "retry"}},
		GetVirtualMediaStatusToolName:  {"Get virtual-media status", true, false, true, false, []string{"configured", "redacted", "retry"}},
		MountVirtualMediaURLToolName:   {"Mount virtual media from URL", false, true, false, true, []string{"configured exact origin", "appliance", "unknown", "retry"}},
		MountVirtualMediaFileToolName:  {"Mount virtual media from file", false, true, false, false, []string{"configured media directory", "local", "unknown", "retry"}},
		UnmountVirtualMediaToolName:    {"Unmount virtual media", false, true, true, false, []string{"configured", "unknown", "retry"}},
		UploadVirtualMediaFileToolName: {"Upload virtual-media file", false, true, false, false, []string{"configured media directory", "storage", "unknown", "retry"}},
		VirtualMediaToolName:           {"Virtual media (deprecated)", false, true, false, false, []string{"deprecated", "configured", "unknown", "retry"}},
		PressHostPowerButtonToolName:   {"Press host power button", false, true, false, false, []string{"configured", "physical", "unknown", "retry"}},
		ForceHostPowerOffToolName:      {"Force host power off", false, true, false, false, []string{"configured", "data loss", "unknown", "retry"}},
		PressHostResetButtonToolName:   {"Press host reset button", false, true, false, false, []string{"configured", "data", "unknown", "retry"}},
		TurnHostDCPowerOnToolName:      {"Turn host DC power on", false, false, true, false, []string{"configured", "physical", "unknown", "retry"}},
		TurnHostDCPowerOffToolName:     {"Turn host DC power off", false, true, true, false, []string{"configured", "data loss", "unknown", "retry"}},
		WakeHostUSBToolName:            {"Wake host over USB", false, false, true, false, []string{"configured", "usb", "unknown", "retry"}},
		WakeHostLANToolName:            {"Wake host over LAN", false, false, true, false, []string{"configured", "network", "unknown", "retry"}},
	}
	if len(tools) != len(want) {
		t.Fatalf("published tools = %d, want %d", len(tools), len(want))
	}
	for name, expected := range want {
		tool := tools[name]
		if tool == nil {
			t.Errorf("missing tool %q", name)
			continue
		}
		if tool.Title != expected.title {
			t.Errorf("%s title = %q, want %q", name, tool.Title, expected.title)
		}
		if strings.TrimSpace(tool.Description) == "" {
			t.Errorf("%s has an empty description", name)
		}
		description := strings.ToLower(tool.Description)
		for _, term := range expected.terms {
			if !strings.Contains(description, term) {
				t.Errorf("%s description does not describe %q: %q", name, term, tool.Description)
			}
		}
		annotations := tool.Annotations
		if annotations == nil || annotations.ReadOnlyHint != expected.readOnly || annotations.DestructiveHint == nil || *annotations.DestructiveHint != expected.destructive || annotations.IdempotentHint != expected.idempotent || annotations.OpenWorldHint == nil || *annotations.OpenWorldHint != expected.openWorld {
			t.Errorf("%s annotations = %#v, want readOnly=%t destructive=%t idempotent=%t openWorld=%t", name, annotations, expected.readOnly, expected.destructive, expected.idempotent, expected.openWorld)
		}
	}
}

func listedMetadataTools(t *testing.T) map[string]*mcp.Tool {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(&recordingDevice{}, "test").Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	listed, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	tools := make(map[string]*mcp.Tool, len(listed.Tools))
	for _, tool := range listed.Tools {
		tools[tool.Name] = tool
	}
	return tools
}
