package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestParseArgsRequiresExplicitInputs(t *testing.T) {
	got, err := parseArgs([]string{"--binary", "/opt/jetkvm-mcp", "--config", "/run/config.yaml", "--device", "lab"})
	if err != nil {
		t.Fatal(err)
	}
	if got.binary != "/opt/jetkvm-mcp" || got.config != "/run/config.yaml" || got.device != "lab" {
		t.Fatalf("options = %#v", got)
	}
	for _, args := range [][]string{
		{"--config", "config.yaml", "--device", "lab"},
		{"--binary", "jetkvm-mcp", "--device", "lab"},
		{"--binary", "jetkvm-mcp", "--config", "config.yaml"},
		{"--binary", "jetkvm-mcp", "--config", "config.yaml", "--device", "lab", "extra"},
	} {
		if _, err := parseArgs(args); err == nil {
			t.Fatalf("parseArgs(%q) succeeded", args)
		}
	}
}

func TestValidateReadOnlyTools(t *testing.T) {
	no := false
	tools := []*mcp.Tool{
		{Name: deviceListTool, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: &no, IdempotentHint: true, OpenWorldHint: &no}},
		{Name: statusTool, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: &no, IdempotentHint: true, OpenWorldHint: &no}},
		{Name: captureTool, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: &no, IdempotentHint: true, OpenWorldHint: &no}},
		{Name: mediaStatusTool, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: &no, IdempotentHint: true, OpenWorldHint: &no}},
	}
	if err := validateReadOnlyTools(tools); err != nil {
		t.Fatal(err)
	}
	if err := validateReadOnlyTools(tools[1:]); err == nil {
		t.Fatal("accepted a tool list without configured-device discovery")
	}
	tools[3].Annotations.ReadOnlyHint = false
	if err := validateReadOnlyTools(tools); err == nil {
		t.Fatal("accepted a mutating virtual-media status tool")
	}
}

func TestValidateConfiguredDeviceDiscovery(t *testing.T) {
	valid := map[string]any{"devices": []any{
		map[string]any{"device": "alpha", "capabilities": map[string]any{
			"mountVirtualMediaURL": false, "mountVirtualMediaFile": false, "uploadVirtualMediaFile": false, "wakeHostLAN": false,
		}},
		map[string]any{"device": "lab", "capabilities": map[string]any{
			"mountVirtualMediaURL": true, "mountVirtualMediaFile": true, "uploadVirtualMediaFile": true, "wakeHostLAN": true,
		}},
	}}
	if err := validateConfiguredDevices(valid, "lab"); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []map[string]any{
		{"devices": []any{}},
		{"devices": []any{map[string]any{"device": "other", "capabilities": valid["devices"].([]any)[0].(map[string]any)["capabilities"]}}},
		{"devices": []any{map[string]any{"device": "lab", "url": "https://private.invalid", "capabilities": valid["devices"].([]any)[0].(map[string]any)["capabilities"]}}},
		{"devices": []any{map[string]any{"device": "lab", "capabilities": map[string]any{
			"mountVirtualMediaURL": true, "mountVirtualMediaFile": true, "uploadVirtualMediaFile": true, "wakeHostLAN": true, "mediaDirectory": "/private",
		}}}},
		{"devices": []any{valid["devices"].([]any)[1], valid["devices"].([]any)[0]}},
	} {
		if err := validateConfiguredDevices(invalid, "lab"); err == nil {
			t.Fatalf("accepted discovery result %#v", invalid)
		}
	}
}

type configuredDeviceCaller struct {
	params *mcp.CallToolParams
	result *mcp.CallToolResult
	err    error
}

func (caller *configuredDeviceCaller) CallTool(_ context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	caller.params = params
	return caller.result, caller.err
}

func TestValidateConfiguredDeviceCallUsesEmptyInput(t *testing.T) {
	caller := &configuredDeviceCaller{result: &mcp.CallToolResult{
		StructuredContent: map[string]any{"devices": []any{
			map[string]any{"device": "lab", "capabilities": map[string]any{
				"mountVirtualMediaURL": false, "mountVirtualMediaFile": false, "uploadVirtualMediaFile": false, "wakeHostLAN": false,
			}},
		}},
	}}
	if err := validateConfiguredDeviceCall(context.Background(), caller, "lab"); err != nil {
		t.Fatal(err)
	}
	arguments, ok := caller.params.Arguments.(map[string]any)
	if caller.params == nil || caller.params.Name != deviceListTool || !ok || len(arguments) != 0 {
		t.Fatalf("call params = %#v", caller.params)
	}
	caller.result.IsError = true
	if err := validateConfiguredDeviceCall(context.Background(), caller, "lab"); err == nil {
		t.Fatal("accepted a tool-error discovery result")
	}
}

type discoveryRejectingSession struct {
	calls []string
}

func (*discoveryRejectingSession) ListTools(context.Context, *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	no := false
	annotations := func() *mcp.ToolAnnotations {
		return &mcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: &no, IdempotentHint: true, OpenWorldHint: &no}
	}
	return &mcp.ListToolsResult{Tools: []*mcp.Tool{
		{Name: deviceListTool, Annotations: annotations()},
		{Name: statusTool, Annotations: annotations()},
		{Name: captureTool, Annotations: annotations()},
		{Name: mediaStatusTool, Annotations: annotations()},
	}}, nil
}

func (session *discoveryRejectingSession) CallTool(_ context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	session.calls = append(session.calls, params.Name)
	return &mcp.CallToolResult{StructuredContent: map[string]any{"devices": []any{
		map[string]any{"device": "other", "capabilities": map[string]any{
			"mountVirtualMediaURL": false, "mountVirtualMediaFile": false, "uploadVirtualMediaFile": false, "wakeHostLAN": false,
		}},
	}}}, nil
}

func TestValidateSessionRejectsMissingAliasBeforeDeviceCalls(t *testing.T) {
	session := new(discoveryRejectingSession)
	report := validateSession(context.Background(), session, "lab")
	if report.Result != "fail" || report.Failed != "devices" || !reflect.DeepEqual(report.Checks, []string{"tools_list"}) {
		t.Fatalf("report = %#v", report)
	}
	if !reflect.DeepEqual(session.calls, []string{deviceListTool}) {
		t.Fatalf("tool calls = %#v", session.calls)
	}
}

func TestValidateStatusStructure(t *testing.T) {
	valid := map[string]any{"device": "lab", "connected": true, "videoWidth": float64(1920), "warnings": []any{"signal settling"}, "virtualMedia": map[string]any{"mounted": true, "sourceType": "http", "mode": "read_only"}}
	if err := validateStatus(valid, "lab"); err != nil {
		t.Fatal(err)
	}
	for _, value := range []map[string]any{
		{"connected": true},
		{"device": "other", "connected": true},
		{"device": "lab", "connected": false},
		{"device": "lab", "connected": "yes"},
		{"device": "lab", "connected": true, "videoWidth": "1920"},
		{"device": "lab", "connected": true, "warnings": []any{"ok", float64(1)}},
		{"device": "lab", "connected": true, "virtualMedia": `{"source":"private"}`},
		{"device": "lab", "connected": true, "virtualMedia": map[string]any{"mounted": true, "source": "private.iso", "sourceType": "storage", "mode": "read_only"}},
		{"device": "lab", "connected": true, "virtualMedia": map[string]any{"mounted": true, "sourceType": "future", "mode": "read_only"}},
	} {
		if err := validateStatus(value, "lab"); err == nil {
			t.Fatalf("accepted status %#v", value)
		}
	}
}

func TestDecodeCaptureFullyDecodesPNG(t *testing.T) {
	var encoded bytes.Buffer
	source := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	source.Set(1, 2, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	decodedBytes := append([]byte(nil), encoded.Bytes()...)
	metadata, err := decodeCapture(&mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.ImageContent{MIMEType: "image/png", Data: decodedBytes}},
		StructuredContent: map[string]any{"device": "lab", "capturedAt": "2026-08-10T03:00:00Z", "mimeType": "image/png", "width": float64(2), "height": float64(3), "sizeBytes": float64(encoded.Len())},
	}, "lab")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(metadata, captureMetadata{Width: 2, Height: 3, SizeBytes: encoded.Len()}) {
		t.Fatalf("metadata = %#v", metadata)
	}
	if !bytes.Equal(decodedBytes, make([]byte, len(decodedBytes))) {
		t.Fatal("decoded PNG buffer was not cleared")
	}

	truncated := encoded.Bytes()[:encoded.Len()-4]
	if _, err := decodeCapture(&mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.ImageContent{MIMEType: "image/png", Data: truncated}},
		StructuredContent: map[string]any{"device": "lab", "capturedAt": "2026-08-10T03:00:00Z", "mimeType": "image/png", "width": float64(2), "height": float64(3), "sizeBytes": float64(len(truncated))},
	}, "lab"); err == nil {
		t.Fatal("accepted a truncated PNG")
	}
}

func TestReportContainsOnlySanitizedMetadata(t *testing.T) {
	report := validationReport{Result: "pass", Checks: []string{"tools_list", "status", "capture"}, Capture: &captureMetadata{Width: 2, Height: 3, SizeBytes: 80}}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"lab", "/run/config.yaml", "/opt/jetkvm-mcp", "pixels", "connected"} {
		if bytes.Contains(data, []byte(private)) {
			t.Fatalf("report contains private/runtime value %q: %s", private, data)
		}
	}
}

func TestValidationFailureReportExcludesPrivateInputSentinel(t *testing.T) {
	const sentinel = "JETKVM-PRIVATE-SENTINEL-7eea7c9f"
	report := runValidation(context.Background(), options{
		binary: "/missing/" + sentinel,
		config: "/private/" + sentinel + ".yaml",
		device: sentinel,
	})
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if report.Result != "fail" || report.Failed != "connect" {
		t.Fatalf("report = %#v", report)
	}
	if bytes.Contains(data, []byte(sentinel)) {
		t.Fatalf("validation report exposed private input: %s", data)
	}
}
