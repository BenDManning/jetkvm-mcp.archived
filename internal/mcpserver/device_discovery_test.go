package mcpserver

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type deviceDiscoveryRecordingDevice struct {
	recordingDevice
	calls int
}

func (device *deviceDiscoveryRecordingDevice) ListDevices(context.Context) (DeviceList, error) {
	device.calls++
	return DeviceList{Devices: []ConfiguredDevice{
		{Device: "alpha", Capabilities: DeviceCapabilities{}},
		{Device: "zeta", Capabilities: DeviceCapabilities{
			MountVirtualMediaURL:   true,
			MountVirtualMediaFile:  true,
			UploadVirtualMediaFile: true,
			WakeHostLAN:            true,
		}},
	}}, nil
}

func TestServerPublishesSafeConfiguredDeviceDiscovery(t *testing.T) {
	device := new(deviceDiscoveryRecordingDevice)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(device, "test").Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	listed, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var tool *mcp.Tool
	for _, candidate := range listed.Tools {
		if candidate.Name == "jetkvm_list_devices" {
			tool = candidate
			break
		}
	}
	if tool == nil {
		t.Fatal("missing jetkvm_list_devices")
	}
	if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint || !tool.Annotations.IdempotentHint || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
		t.Fatalf("annotations = %#v", tool.Annotations)
	}
	inputJSON, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var inputSchema struct {
		Properties           map[string]json.RawMessage `json:"properties"`
		Required             []string                   `json:"required"`
		AdditionalProperties any                        `json:"additionalProperties"`
	}
	if err := json.Unmarshal(inputJSON, &inputSchema); err != nil {
		t.Fatal(err)
	}
	if len(inputSchema.Properties) != 0 || len(inputSchema.Required) != 0 || inputSchema.AdditionalProperties != false {
		t.Fatalf("input schema = %s", inputJSON)
	}

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: tool.Name, Arguments: map[string]any{}})
	if err != nil || result == nil || result.IsError {
		t.Fatalf("CallTool = %#v, %v", result, err)
	}
	want := map[string]any{"devices": []any{
		map[string]any{"device": "alpha", "capabilities": map[string]any{
			"mountVirtualMediaURL": false, "mountVirtualMediaFile": false, "uploadVirtualMediaFile": false, "wakeHostLAN": false,
		}},
		map[string]any{"device": "zeta", "capabilities": map[string]any{
			"mountVirtualMediaURL": true, "mountVirtualMediaFile": true, "uploadVirtualMediaFile": true, "wakeHostLAN": true,
		}},
	}}
	if !reflect.DeepEqual(result.StructuredContent, want) {
		t.Fatalf("structured content = %#v, want %#v", result.StructuredContent, want)
	}
	if device.calls != 1 {
		t.Fatalf("discovery calls = %d, want one", device.calls)
	}
}

func TestSafeConfiguredDeviceDiscoveryIsDocumented(t *testing.T) {
	readme := readRepositoryDocument(t, "README.md")
	productContract := readRepositoryDocument(t, "docs/product-contract.md")
	threatModel := readRepositoryDocument(t, "docs/threat-model.md")

	for name, document := range map[string]string{
		"README":           readme,
		"product contract": productContract,
		"threat model":     threatModel,
	} {
		for _, required := range []string{
			"`jetkvm_list_devices`",
			"configuration-derived",
			"does not open a device session",
		} {
			if !strings.Contains(document, required) {
				t.Errorf("%s does not document %q", name, required)
			}
		}
	}
	if !strings.Contains(productContract, "18 current tools") {
		t.Error("product contract does not classify the additive 18-tool surface")
	}
	for _, private := range []string{"URLs", "credentials", "allowed origins", "media directories", "Wake-on-LAN targets"} {
		if !strings.Contains(threatModel, private) {
			t.Errorf("threat model does not exclude private discovery field %q", private)
		}
	}
}
