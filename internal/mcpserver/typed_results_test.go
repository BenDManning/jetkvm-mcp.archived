package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type typedResultDevice struct {
	recordingDevice
	failure    error
	mediaCalls int
}

type invalidOutputDevice struct {
	recordingDevice
}

func (*invalidOutputDevice) Status(context.Context, string) (Status, error) {
	return Status{
		Device: "lab", Connected: true,
		VirtualMedia: &VirtualMediaState{Mounted: true, SourceType: VirtualMediaSourceType("PRIVATE-OUTPUT-SENTINEL")},
	}, nil
}

func (*invalidOutputDevice) Keyboard(context.Context, string, KeyboardRequest) (KeyboardResult, error) {
	return KeyboardResult{Device: "lab", Operation: KeyboardOperation("PRIVATE-OUTPUT-SENTINEL"), Status: "completed"}, nil
}

func (*invalidOutputDevice) VirtualMedia(_ context.Context, _ string, request VirtualMediaRequest) (VirtualMediaResult, error) {
	return VirtualMediaResult{
		Device: "lab", Operation: request.Operation, SourceType: VirtualMediaSourceHTTP,
		Mode: "read_only", Status: ResultStatusCompleted,
	}, nil
}

func (*typedResultDevice) Status(context.Context, string) (Status, error) {
	return Status{
		Device: "lab", Connected: true,
		VirtualMedia: &VirtualMediaState{Mounted: true, SourceType: VirtualMediaSourceHTTP, Mode: "read_only"},
	}, nil
}

func (device *typedResultDevice) VirtualMedia(_ context.Context, _ string, request VirtualMediaRequest) (VirtualMediaResult, error) {
	device.mediaCalls++
	if device.failure != nil {
		return VirtualMediaResult{}, device.failure
	}
	return VirtualMediaResult{
		Device: "lab", Operation: request.Operation, Mounted: true,
		SourceType: VirtualMediaSourceHTTP, Mode: "read_only", Status: "observed",
	}, nil
}

func TestTypedMediaSchemaErrorsDoNotEchoPrivateSources(t *testing.T) {
	const (
		userinfo = "private-user:private-password@"
		query    = "private-query-token"
		fragment = "private-fragment-token"
		pathPart = "private-local-path"
	)
	overlongURL := "https://" + userinfo + "media.invalid/" + strings.Repeat("x", 4096) + ".iso?token=" + query + "#" + fragment
	overlongPath := strings.Repeat(pathPart+"/", 300) + "private.iso"

	for _, test := range []struct {
		name      string
		tool      string
		arguments map[string]any
	}{
		{name: "mount URL", tool: MountVirtualMediaURLToolName, arguments: map[string]any{"device": "lab", "url": overlongURL}},
		{name: "mount file", tool: MountVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": overlongPath}},
		{name: "upload file", tool: UploadVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": overlongPath}},
	} {
		t.Run(test.name, func(t *testing.T) {
			device := &typedResultDevice{}
			session, cleanup := connectVirtualMediaTestClient(t, device)
			defer cleanup()

			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: test.tool, Arguments: test.arguments})
			if err != nil || result == nil || !result.IsError {
				t.Fatalf("CallTool(%s) = %#v, %v", test.tool, result, err)
			}
			encoded, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{userinfo, query, fragment, pathPart, "media.invalid"} {
				if bytes.Contains(encoded, []byte(forbidden)) {
					t.Fatalf("%s schema error leaked private source component %q: %s", test.tool, forbidden, encoded)
				}
			}
			if device.mediaCalls != 0 {
				t.Fatalf("%s dispatched rejected input to device", test.tool)
			}
		})
	}
}

func TestTypedStatusAndMediaResultsExcludeRawSources(t *testing.T) {
	session, cleanup := connectVirtualMediaTestClient(t, &typedResultDevice{})
	defer cleanup()
	for _, call := range []mcp.CallToolParams{
		{Name: GetStatusToolName, Arguments: map[string]any{"device": "lab"}},
		{Name: GetVirtualMediaStatusToolName, Arguments: map[string]any{"device": "lab"}},
	} {
		result, err := session.CallTool(context.Background(), &call)
		if err != nil || result == nil || result.IsError {
			t.Fatalf("CallTool(%s) = %#v, %v", call.Name, result, err)
		}
		encoded, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range [][]byte{[]byte(`"source"`), []byte(`"url"`), []byte(`"filename"`)} {
			if bytes.Contains(encoded, forbidden) {
				t.Fatalf("%s result exposes raw media source field %s: %s", call.Name, forbidden, encoded)
			}
		}
		if !bytes.Contains(encoded, []byte(`"sourceType":"http"`)) {
			t.Fatalf("%s result lacks typed source class: %s", call.Name, encoded)
		}
	}
}

func TestTypedMediaErrorsExcludeRawSources(t *testing.T) {
	const sentinel = "JETKVM-PRIVATE-ERROR-SENTINEL-74b1d8"
	device := &typedResultDevice{failure: errors.New("firmware rejected https://media.invalid/" + sentinel + ".iso?token=" + sentinel)}
	session, cleanup := connectVirtualMediaTestClient(t, device)
	defer cleanup()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      MountVirtualMediaURLToolName,
		Arguments: map[string]any{"device": "lab", "url": "https://media.invalid/" + sentinel + ".iso?token=" + sentinel},
	})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("CallTool = %#v, %v", result, err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range [][]byte{[]byte(sentinel), []byte("media.invalid"), []byte("token=")} {
		if bytes.Contains(encoded, forbidden) {
			t.Fatalf("tool error leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestTypedMediaOutputSchemasExcludeRawSources(t *testing.T) {
	session, cleanup := connectVirtualMediaTestClient(t, &typedResultDevice{})
	defer cleanup()
	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range listed.Tools {
		if tool.Name != GetStatusToolName && tool.Name != GetVirtualMediaStatusToolName &&
			tool.Name != MountVirtualMediaURLToolName && tool.Name != MountVirtualMediaFileToolName &&
			tool.Name != UnmountVirtualMediaToolName && tool.Name != UploadVirtualMediaFileToolName {
			continue
		}
		encoded, err := json.Marshal(tool.OutputSchema)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(encoded, []byte(`"source"`)) {
			t.Fatalf("%s output schema exposes raw source: %s", tool.Name, encoded)
		}
	}
}

func TestPublicOutputSchemasConstrainFixedVocabularies(t *testing.T) {
	session, cleanup := connectVirtualMediaTestClient(t, &contractDevice{})
	defer cleanup()
	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]map[string][]string{
		ReleaseSessionToolName:         {"status": {string(SessionStatusReleased)}},
		TakeOverSessionToolName:        {"status": {string(SessionStatusAuthoritative)}},
		PressHostPowerButtonToolName:   {"action": {string(PowerActionPressHostPowerButton)}, "status": {"completed"}},
		ForceHostPowerOffToolName:      {"action": {string(PowerActionForceHostPowerOff)}, "status": {"completed"}},
		PressHostResetButtonToolName:   {"action": {string(PowerActionPressHostResetButton)}, "status": {"completed"}},
		TurnHostDCPowerOnToolName:      {"action": {string(PowerActionTurnHostDCPowerOn)}, "status": {"completed"}},
		TurnHostDCPowerOffToolName:     {"action": {string(PowerActionTurnHostDCPowerOff)}, "status": {"completed"}},
		WakeHostUSBToolName:            {"action": {string(PowerActionWakeHostUSB)}, "status": {"completed"}},
		WakeHostLANToolName:            {"action": {string(PowerActionWakeHostLAN)}, "status": {"completed"}},
		KeyboardToolName:               {"operation": {string(KeyboardTypeText), string(KeyboardPressKey)}, "status": {"completed"}},
		MouseToolName:                  {"operation": {string(MouseMoveAbsolute), string(MouseMoveRelative), string(MouseClick), string(MouseScroll)}, "status": {"completed"}},
		GetVirtualMediaStatusToolName:  {"operation": {string(VirtualMediaStatus)}, "sourceType": {"http", "storage"}, "status": {"observed"}},
		MountVirtualMediaURLToolName:   {"operation": {string(VirtualMediaMountURL)}, "sourceType": {"http"}, "status": {"completed"}},
		MountVirtualMediaFileToolName:  {"operation": {string(VirtualMediaMountFile)}, "sourceType": {"storage"}, "status": {"completed"}},
		UnmountVirtualMediaToolName:    {"operation": {string(VirtualMediaUnmount)}, "status": {"completed"}},
		UploadVirtualMediaFileToolName: {"operation": {string(VirtualMediaUpload)}, "sourceType": {"storage"}, "mode": {"read_only"}, "status": {"completed"}},
		CaptureScreenToolName:          {"mimeType": {"image/png"}},
	}
	for name, properties := range expected {
		tool := findTool(t, listed.Tools, name)
		data, err := json.Marshal(tool.OutputSchema)
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			Properties map[string]struct {
				Enum    []string `json:"enum"`
				Minimum *float64 `json:"minimum"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatal(err)
		}
		for property, want := range properties {
			if !sameStrings(schema.Properties[property].Enum, want) {
				t.Fatalf("%s output %s enum = %v, want %v", name, property, schema.Properties[property].Enum, want)
			}
		}
		if name == CaptureScreenToolName {
			for _, property := range []string{"width", "height", "sizeBytes"} {
				minimum := schema.Properties[property].Minimum
				if minimum == nil || *minimum != 1 {
					t.Fatalf("%s output %s minimum = %v, want 1", name, property, minimum)
				}
			}
		}
	}

	statusTool := findTool(t, listed.Tools, GetStatusToolName)
	data, err := json.Marshal(statusTool.OutputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var statusSchema struct {
		Properties map[string]struct {
			Items struct {
				Enum []string `json:"enum"`
			} `json:"items"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &statusSchema); err != nil {
		t.Fatal(err)
	}
	warnings := []string{
		"version unavailable", "active extension unavailable", "virtual media unavailable", "video unavailable",
		"USB unavailable", "ATX state unavailable", "DC state unavailable",
	}
	if got := statusSchema.Properties["warnings"].Items.Enum; !sameStrings(got, warnings) {
		t.Fatalf("status warning enum = %v, want %v", got, warnings)
	}
}

func TestCaptureReturnsImageThenSafeJSONMetadata(t *testing.T) {
	session, cleanup := connectVirtualMediaTestClient(t, &contractDevice{})
	defer cleanup()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: CaptureScreenToolName, Arguments: map[string]any{"device": "fixture-device"},
	})
	if err != nil || result == nil || result.IsError {
		t.Fatalf("CallTool = %#v, %v", result, err)
	}
	if len(result.Content) != 2 {
		t.Fatalf("capture content = %#v, want image and JSON metadata", result.Content)
	}
	image, ok := result.Content[0].(*mcp.ImageContent)
	if !ok || image.MIMEType != "image/png" || len(image.Data) == 0 {
		t.Fatalf("first capture content = %#v, want PNG image", result.Content[0])
	}
	text, ok := result.Content[1].(*mcp.TextContent)
	if !ok {
		t.Fatalf("second capture content = %T, want JSON text", result.Content[1])
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(text.Text), &metadata); err != nil {
		t.Fatalf("capture metadata = %q: %v", text.Text, err)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok || !jsonEqual(metadata, structured) {
		t.Fatalf("capture text metadata = %#v, structured = %#v", metadata, result.StructuredContent)
	}
	if len(metadata) != 6 || metadata["device"] != "fixture-device" || metadata["mimeType"] != "image/png" {
		t.Fatalf("capture metadata contains unexpected fields: %#v", metadata)
	}
}

func TestInvalidProviderOutputsReturnSanitizedToolErrors(t *testing.T) {
	session, cleanup := connectVirtualMediaTestClient(t, &invalidOutputDevice{})
	defer cleanup()
	for _, call := range []mcp.CallToolParams{
		{Name: GetStatusToolName, Arguments: map[string]any{"device": "lab"}},
		{Name: KeyboardToolName, Arguments: map[string]any{"device": "lab", "operation": "press_key", "key": "enter"}},
		{Name: UnmountVirtualMediaToolName, Arguments: map[string]any{"device": "lab"}},
	} {
		result, err := session.CallTool(context.Background(), &call)
		if err != nil || result == nil || !result.IsError {
			t.Fatalf("CallTool(%s) = %#v, %v; want sanitized tool error", call.Name, result, err)
		}
		if result.StructuredContent != nil {
			t.Fatalf("CallTool(%s) error has structured content: %#v", call.Name, result.StructuredContent)
		}
		failure := decodeToolError(t, result)
		wantOutcome := "failed"
		if call.Name == KeyboardToolName || call.Name == UnmountVirtualMediaToolName {
			wantOutcome = "unknown"
		}
		if failure.Code != "operation_failed" || failure.Outcome != wantOutcome || failure.Retryable {
			t.Fatalf("CallTool(%s) failure = %+v", call.Name, failure)
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(encoded, []byte("PRIVATE-OUTPUT-SENTINEL")) {
			t.Fatalf("CallTool(%s) leaked provider output: %s", call.Name, encoded)
		}
	}
}
