package mcpserver

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerPublishesConsequenceCorrectVirtualMediaTools(t *testing.T) {
	clientSession, cleanup := connectVirtualMediaTestClient(t, &recordingDevice{})
	defer cleanup()

	listed, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	tools := make(map[string]*mcp.Tool, len(listed.Tools))
	for _, tool := range listed.Tools {
		tools[tool.Name] = tool
	}

	assertVirtualMediaTool(t, tools, GetVirtualMediaStatusToolName, true, false, true, false, []string{"device"})
	assertVirtualMediaTool(t, tools, MountVirtualMediaURLToolName, false, true, false, true, []string{"device", "url"})
	assertVirtualMediaTool(t, tools, MountVirtualMediaFileToolName, false, true, false, false, []string{"device", "path"})
	assertVirtualMediaTool(t, tools, UnmountVirtualMediaToolName, false, true, true, false, []string{"device"})
	assertVirtualMediaTool(t, tools, UploadVirtualMediaFileToolName, false, true, false, false, []string{"device", "path"})
	assertInputProperties(t, tools[GetVirtualMediaStatusToolName], []string{"device"})
	assertInputProperties(t, tools[MountVirtualMediaURLToolName], []string{"device", "mode", "url"})
	assertInputProperties(t, tools[MountVirtualMediaFileToolName], []string{"device", "mode", "path"})
	assertInputProperties(t, tools[UnmountVirtualMediaToolName], []string{"device"})
	assertInputProperties(t, tools[UploadVirtualMediaFileToolName], []string{"device", "path"})
	assertStringEnum(t, tools[MountVirtualMediaURLToolName], "mode", []string{"read_only", "read_write"})
	assertStringEnum(t, tools[MountVirtualMediaFileToolName], "mode", []string{"read_only", "read_write"})
	assertStringBounds(t, tools[MountVirtualMediaURLToolName], "url", 1, 4096)
	assertStringBounds(t, tools[MountVirtualMediaFileToolName], "path", 1, 4096)
	assertStringBounds(t, tools[UploadVirtualMediaFileToolName], "path", 1, 4096)
	if description := strings.ToLower(tools[MountVirtualMediaURLToolName].Description); !strings.Contains(description, "configured exact origin") || !strings.Contains(description, "appliance") {
		t.Fatalf("%s description does not state its configured origin and fetching actor: %q", MountVirtualMediaURLToolName, description)
	}

	legacy := tools[VirtualMediaToolName]
	if legacy == nil || !strings.Contains(strings.ToLower(legacy.Description), "deprecated") {
		t.Fatalf("legacy %s must remain discoverable with a deprecation notice", VirtualMediaToolName)
	}
	assertStringEnum(t, legacy, "operation", []string{"status", "mount_url", "mount_file", "unmount", "upload"})
}

type virtualMediaCall struct {
	device  string
	request VirtualMediaRequest
}

type virtualMediaRecordingDevice struct {
	recordingDevice
	mu    sync.Mutex
	calls []virtualMediaCall
	err   error
}

func (device *virtualMediaRecordingDevice) VirtualMedia(_ context.Context, name string, request VirtualMediaRequest) (VirtualMediaResult, error) {
	device.mu.Lock()
	defer device.mu.Unlock()
	device.calls = append(device.calls, virtualMediaCall{device: name, request: request})
	return VirtualMediaResult{Device: name, Operation: request.Operation, Status: "completed"}, device.err
}

func (device *virtualMediaRecordingDevice) recorded() []virtualMediaCall {
	device.mu.Lock()
	defer device.mu.Unlock()
	return append([]virtualMediaCall(nil), device.calls...)
}

func TestConsequenceCorrectVirtualMediaToolDispatch(t *testing.T) {
	device := &virtualMediaRecordingDevice{}
	clientSession, cleanup := connectVirtualMediaTestClient(t, device)
	defer cleanup()

	tests := []struct {
		name      string
		arguments map[string]any
		want      VirtualMediaRequest
	}{
		{name: GetVirtualMediaStatusToolName, arguments: map[string]any{"device": "lab"}, want: VirtualMediaRequest{Operation: VirtualMediaStatus}},
		{name: MountVirtualMediaURLToolName, arguments: map[string]any{"device": "lab", "url": "https://example.invalid/media.iso", "mode": "read_write"}, want: VirtualMediaRequest{Operation: VirtualMediaMountURL, Source: "https://example.invalid/media.iso", Mode: "read_write"}},
		{name: MountVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": "boot/media.iso", "mode": "read_only"}, want: VirtualMediaRequest{Operation: VirtualMediaMountFile, Source: "boot/media.iso", Mode: "read_only"}},
		{name: UnmountVirtualMediaToolName, arguments: map[string]any{"device": "lab"}, want: VirtualMediaRequest{Operation: VirtualMediaUnmount}},
		{name: UploadVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": "boot/media.iso"}, want: VirtualMediaRequest{Operation: VirtualMediaUpload, Source: "boot/media.iso"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := len(device.recorded())
			result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: test.name, Arguments: test.arguments})
			if err != nil || result.IsError {
				t.Fatalf("CallTool = %#v, %v", result, err)
			}
			calls := device.recorded()
			if len(calls) != before+1 || calls[before].device != "lab" || calls[before].request != test.want {
				t.Fatalf("calls = %#v, want %#v", calls, test.want)
			}
		})
	}
}

func TestConsequenceCorrectVirtualMediaToolValidationSkipsProvider(t *testing.T) {
	device := &virtualMediaRecordingDevice{}
	clientSession, cleanup := connectVirtualMediaTestClient(t, device)
	defer cleanup()

	for _, test := range []struct {
		name      string
		arguments map[string]any
	}{
		{name: GetVirtualMediaStatusToolName, arguments: map[string]any{"device": "", "path": "not-accepted.iso"}},
		{name: MountVirtualMediaURLToolName, arguments: map[string]any{"device": "lab", "url": ""}},
		{name: MountVirtualMediaURLToolName, arguments: map[string]any{"device": "lab", "url": strings.Repeat("x", 4097)}},
		{name: MountVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": "", "mode": "invalid"}},
		{name: MountVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": strings.Repeat("x", 4097)}},
		{name: UnmountVirtualMediaToolName, arguments: map[string]any{"device": "lab", "url": "https://example.invalid/media.iso"}},
		{name: UploadVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": ""}},
		{name: UploadVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": strings.Repeat("x", 4097)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: test.name, Arguments: test.arguments})
			if err == nil && (result == nil || !result.IsError) {
				t.Fatalf("CallTool accepted malformed arguments: %#v", result)
			}
		})
	}
	if calls := device.recorded(); len(calls) != 0 {
		t.Fatalf("malformed calls reached provider: %#v", calls)
	}
}

func TestConsequenceCorrectVirtualMediaToolErrors(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		provider  error
		outcome   string
		retryable bool
	}{
		{name: GetVirtualMediaStatusToolName, arguments: map[string]any{"device": "lab"}, provider: classifiedFixtureError{error: context.DeadlineExceeded, code: "timeout", outcome: "failed", retryable: true}, outcome: "failed", retryable: true},
		{name: MountVirtualMediaURLToolName, arguments: map[string]any{"device": "lab", "url": "https://example.invalid/media.iso"}, provider: context.DeadlineExceeded, outcome: "unknown"},
		{name: MountVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": "media.iso"}, provider: context.DeadlineExceeded, outcome: "unknown"},
		{name: UnmountVirtualMediaToolName, arguments: map[string]any{"device": "lab"}, provider: context.DeadlineExceeded, outcome: "unknown"},
		{name: UploadVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": "media.iso"}, provider: context.DeadlineExceeded, outcome: "unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			device := &virtualMediaRecordingDevice{err: test.provider}
			clientSession, cleanup := connectVirtualMediaTestClient(t, device)
			defer cleanup()
			result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: test.name, Arguments: test.arguments})
			if err != nil {
				t.Fatal(err)
			}
			failure := decodeToolError(t, result)
			if failure.Code != "timeout" || failure.Outcome != test.outcome || failure.Retryable != test.retryable {
				t.Fatalf("failure = %+v", failure)
			}
		})
	}
}

type cancelingVirtualMediaDevice struct {
	recordingDevice
	started  chan struct{}
	canceled chan struct{}
}

func (device *cancelingVirtualMediaDevice) VirtualMedia(ctx context.Context, _ string, _ VirtualMediaRequest) (VirtualMediaResult, error) {
	close(device.started)
	<-ctx.Done()
	close(device.canceled)
	return VirtualMediaResult{}, ctx.Err()
}

func TestConsequenceCorrectVirtualMediaToolCancellation(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
	}{
		{name: GetVirtualMediaStatusToolName, arguments: map[string]any{"device": "lab"}},
		{name: MountVirtualMediaURLToolName, arguments: map[string]any{"device": "lab", "url": "https://example.invalid/media.iso"}},
		{name: MountVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": "media.iso"}},
		{name: UnmountVirtualMediaToolName, arguments: map[string]any{"device": "lab"}},
		{name: UploadVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": "media.iso"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			device := &cancelingVirtualMediaDevice{started: make(chan struct{}), canceled: make(chan struct{})}
			clientSession, cleanup := connectVirtualMediaTestClient(t, device)
			defer cleanup()
			ctx, cancel := context.WithCancel(context.Background())
			callDone := make(chan struct{})
			go func() {
				defer close(callDone)
				_, _ = clientSession.CallTool(ctx, &mcp.CallToolParams{Name: test.name, Arguments: test.arguments})
			}()
			select {
			case <-device.started:
			case <-time.After(time.Second):
				cancel()
				t.Fatal("provider did not start")
			}
			cancel()
			select {
			case <-device.canceled:
			case <-time.After(time.Second):
				t.Fatal("cancellation did not reach provider")
			}
			select {
			case <-callDone:
			case <-time.After(time.Second):
				t.Fatal("canceled MCP call did not stop")
			}
		})
	}
}

func connectVirtualMediaTestClient(t *testing.T, device Device) (*mcp.ClientSession, func()) {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(device, "test").Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatal(err)
	}
	return clientSession, func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}
}

func assertVirtualMediaTool(t *testing.T, tools map[string]*mcp.Tool, name string, readOnly, destructive, idempotent, openWorld bool, required []string) {
	t.Helper()
	tool := tools[name]
	if tool == nil {
		t.Fatalf("missing tool %q", name)
	}
	annotations := tool.Annotations
	if annotations == nil || annotations.ReadOnlyHint != readOnly || annotations.DestructiveHint == nil || *annotations.DestructiveHint != destructive || annotations.IdempotentHint != idempotent || annotations.OpenWorldHint == nil || *annotations.OpenWorldHint != openWorld {
		t.Fatalf("%s annotations = %#v", name, annotations)
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

func assertInputProperties(t *testing.T, tool *mcp.Tool, want []string) {
	t.Helper()
	if tool == nil {
		t.Fatal("missing tool")
	}
	data, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		AdditionalProperties any                        `json:"additionalProperties"`
		Properties           map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		got = append(got, name)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !sameStrings(got, want) {
		t.Fatalf("%s properties = %v, want %v", tool.Name, got, want)
	}
	if value, ok := schema.AdditionalProperties.(bool); !ok || value {
		t.Fatalf("%s additionalProperties = %#v, want false", tool.Name, schema.AdditionalProperties)
	}
}

func assertStringBounds(t *testing.T, tool *mcp.Tool, property string, minimum, maximum int) {
	t.Helper()
	data, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]struct {
			MinLength *int `json:"minLength"`
			MaxLength *int `json:"maxLength"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	got := schema.Properties[property]
	if got.MinLength == nil || *got.MinLength != minimum || got.MaxLength == nil || *got.MaxLength != maximum {
		t.Fatalf("%s %s bounds = %#v", tool.Name, property, got)
	}
}
