package mcpserver

import (
	"context"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type takeoverSessionRecordingDevice struct {
	recordingDevice
	device string
	calls  int
}

func (device *takeoverSessionRecordingDevice) TakeOverSession(_ context.Context, name string) (SessionTakeoverResult, error) {
	device.device = name
	device.calls++
	return SessionTakeoverResult{Device: name, Status: SessionStatusAuthoritative}, nil
}

func TestTakeOverSessionToolPublishesAndRoutesExactContract(t *testing.T) {
	device := new(takeoverSessionRecordingDevice)
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
	tool := findTool(t, listed.Tools, TakeOverSessionToolName)
	if tool.Annotations == nil || tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint || tool.Annotations.IdempotentHint || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
		t.Fatalf("annotations = %#v", tool.Annotations)
	}
	assertClosedObjectSchema(t, "input", tool.InputSchema, []string{"device"}, map[string][]string{})
	assertClosedObjectSchema(t, "output", tool.OutputSchema, []string{"device", "status"}, map[string][]string{"status": {string(SessionStatusAuthoritative)}})

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: TakeOverSessionToolName, Arguments: map[string]any{"device": "lab"},
	})
	if err != nil || result == nil || result.IsError {
		t.Fatalf("CallTool = %#v, %v", result, err)
	}
	want := map[string]any{"device": "lab", "status": string(SessionStatusAuthoritative)}
	if !reflect.DeepEqual(result.StructuredContent, want) || device.device != "lab" || device.calls != 1 {
		t.Fatalf("structured content = %#v device=%q calls=%d", result.StructuredContent, device.device, device.calls)
	}
}

func TestTakeOverSessionToolRejectsExtraInputBeforeDispatch(t *testing.T) {
	device := new(takeoverSessionRecordingDevice)
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
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: TakeOverSessionToolName, Arguments: map[string]any{"device": "lab", "force": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	failure := decodeToolError(t, result)
	if failure.Code != "invalid_input" || failure.Outcome != "not_sent" || failure.Retryable || device.calls != 0 {
		t.Fatalf("failure = %+v calls = %d", failure, device.calls)
	}
}
