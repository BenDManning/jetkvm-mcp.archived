package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type recordingDevice struct{}

func (*recordingDevice) Status(context.Context, string) (Status, error) {
	return Status{}, nil
}

func (*recordingDevice) Power(context.Context, string, PowerAction, string) (PowerResult, error) {
	return PowerResult{}, nil
}

func (*recordingDevice) CaptureScreen(context.Context, string, CaptureRequest) (CaptureResult, error) {
	return CaptureResult{}, nil
}

func (*recordingDevice) Keyboard(context.Context, string, KeyboardRequest) (KeyboardResult, error) {
	return KeyboardResult{}, nil
}

func (*recordingDevice) Mouse(context.Context, string, MouseRequest) (MouseResult, error) {
	return MouseResult{}, nil
}

func (*recordingDevice) VirtualMedia(context.Context, string, VirtualMediaRequest) (VirtualMediaResult, error) {
	return VirtualMediaResult{}, nil
}

func TestServerPublishesExplicitPowerTools(t *testing.T) {
	ctx := context.Background()
	server := New(&recordingDevice{}, "test")
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "contract-test", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	listed, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(listed.Tools), 12; got != want {
		t.Fatalf("tool count = %d, want %d", got, want)
	}

	tools := make(map[string]*mcp.Tool, len(listed.Tools))
	for _, tool := range listed.Tools {
		tools[tool.Name] = tool
	}
	if tools["jetkvm_power"] != nil {
		t.Fatal("combined jetkvm_power tool must not be published")
	}
	for _, forbidden := range []string{"jetkvm_rpc", "jetkvm_raw_rpc", "jetkvm_debug_rpc"} {
		if tools[forbidden] != nil {
			t.Fatalf("local-only RPC leaked into tools/list as %q", forbidden)
		}
	}

	assertTool(t, tools, "jetkvm_get_status", true, false, true, []string{"device"})
	assertTool(t, tools, "jetkvm_press_host_power_button", false, true, false, []string{"device"})
	assertTool(t, tools, "jetkvm_force_host_power_off", false, true, false, []string{"device"})
	assertTool(t, tools, "jetkvm_press_host_reset_button", false, true, false, []string{"device"})
	assertTool(t, tools, "jetkvm_turn_host_dc_power_on", false, false, true, []string{"device"})
	assertTool(t, tools, "jetkvm_turn_host_dc_power_off", false, true, true, []string{"device"})
	assertTool(t, tools, "jetkvm_wake_host_usb", false, false, true, []string{"device"})
	assertTool(t, tools, "jetkvm_wake_host_lan", false, false, true, []string{"device", "target"})
	assertTool(t, tools, "jetkvm_capture_screen", true, false, true, []string{"device"})
	assertTool(t, tools, "jetkvm_keyboard", false, false, false, []string{"device", "operation"})
	assertTool(t, tools, "jetkvm_mouse", false, false, false, []string{"device", "operation"})
	assertTool(t, tools, "jetkvm_virtual_media", false, true, false, []string{"device", "operation"})

	assertStringEnum(t, tools["jetkvm_keyboard"], "operation", []string{"type_text", "press_key"})
	assertStringEnum(t, tools["jetkvm_mouse"], "operation", []string{"move_absolute", "move_relative", "click", "scroll"})
	assertStringEnum(t, tools["jetkvm_virtual_media"], "operation", []string{"status", "mount_url", "mount_file", "unmount", "upload"})

	for name, tool := range tools {
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
			Required   []string                   `json:"required"`
		}
		data, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("marshal %s schema: %v", name, err)
		}
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatalf("decode %s schema: %v", name, err)
		}
		if _, exists := schema.Properties["action"]; exists {
			t.Fatalf("%s exposes a combined action selector", name)
		}
	}
}

func assertTool(t *testing.T, tools map[string]*mcp.Tool, name string, readOnly, destructive, idempotent bool, required []string) {
	t.Helper()
	tool := tools[name]
	if tool == nil {
		t.Fatalf("missing tool %q", name)
	}
	if tool.Annotations == nil || tool.Annotations.ReadOnlyHint != readOnly || tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint != destructive || tool.Annotations.IdempotentHint != idempotent {
		t.Fatalf("%s annotations = %#v", name, tool.Annotations)
	}
	if tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
		t.Fatalf("%s must declare a closed-world device interaction", name)
	}
	data, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if !sameStrings(schema.Required, required) {
		t.Fatalf("%s required fields = %v, want %v", name, schema.Required, required)
	}
}

func assertStringEnum(t *testing.T, tool *mcp.Tool, property string, want []string) {
	t.Helper()
	if tool == nil {
		t.Fatal("missing tool")
	}
	data, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if !sameStrings(schema.Properties[property].Enum, want) {
		t.Fatalf("%s %s enum = %v, want %v", tool.Name, property, schema.Properties[property].Enum, want)
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	counts := make(map[string]int, len(got))
	for _, value := range got {
		counts[value]++
	}
	for _, value := range want {
		counts[value]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}
