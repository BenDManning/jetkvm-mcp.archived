package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type releaseSessionRecordingDevice struct {
	recordingDevice
	device string
	calls  int
	err    error
}

func (device *releaseSessionRecordingDevice) ReleaseSession(_ context.Context, name string) (SessionReleaseResult, error) {
	device.device = name
	device.calls++
	return SessionReleaseResult{Device: name, Status: SessionStatusReleased}, device.err
}

func TestReleaseSessionToolPublishesAndRoutesExactContract(t *testing.T) {
	device := new(releaseSessionRecordingDevice)
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
		if candidate.Name == ReleaseSessionToolName {
			tool = candidate
			break
		}
	}
	if tool == nil {
		t.Fatalf("missing %s", ReleaseSessionToolName)
	}
	if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint || !tool.Annotations.IdempotentHint || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
		t.Fatalf("annotations = %#v", tool.Annotations)
	}
	assertClosedObjectSchema(t, "input", tool.InputSchema, []string{"device"}, map[string][]string{})
	assertClosedObjectSchema(t, "output", tool.OutputSchema, []string{"device", "status"}, map[string][]string{"status": {string(SessionStatusReleased)}})

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: ReleaseSessionToolName, Arguments: map[string]any{"device": "lab"},
	})
	if err != nil || result == nil || result.IsError {
		t.Fatalf("CallTool = %#v, %v", result, err)
	}
	want := map[string]any{"device": "lab", "status": string(SessionStatusReleased)}
	if !reflect.DeepEqual(result.StructuredContent, want) {
		t.Fatalf("structured content = %#v, want %#v", result.StructuredContent, want)
	}
	if device.device != "lab" {
		t.Fatalf("release device = %q, want lab", device.device)
	}
}

func TestReleaseSessionToolRejectsExtraInputBeforeDispatch(t *testing.T) {
	device := new(releaseSessionRecordingDevice)
	clientSession, cleanup := connectReleaseSessionClient(t, device)
	defer cleanup()
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: ReleaseSessionToolName, Arguments: map[string]any{"device": "lab", "wait": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	failure := decodeToolError(t, result)
	if failure.Code != "invalid_input" || failure.Outcome != "not_sent" || failure.Retryable || device.calls != 0 {
		t.Fatalf("failure = %+v calls = %d", failure, device.calls)
	}
}

func TestReleaseSessionToolPreservesLifecycleErrors(t *testing.T) {
	for _, test := range []struct {
		name    string
		code    string
		outcome string
	}{
		{name: "busy", code: "busy", outcome: "not_sent"},
		{name: "released", code: "session_released", outcome: "not_sent"},
		{name: "uncertain", code: "ownership_uncertain", outcome: "failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			device := &releaseSessionRecordingDevice{err: classifiedFixtureError{
				error: errors.New("private lifecycle detail"), code: test.code, outcome: test.outcome,
			}}
			clientSession, cleanup := connectReleaseSessionClient(t, device)
			defer cleanup()
			result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
				Name: ReleaseSessionToolName, Arguments: map[string]any{"device": "lab"},
			})
			if err != nil {
				t.Fatal(err)
			}
			failure := decodeToolError(t, result)
			if failure.Code != test.code || failure.Outcome != test.outcome || failure.Retryable {
				t.Fatalf("failure = %+v", failure)
			}
		})
	}
}

func connectReleaseSessionClient(t *testing.T, device Device) (*mcp.ClientSession, func()) {
	t.Helper()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(device, "test").Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		serverSession.Close()
		t.Fatal(err)
	}
	return clientSession, func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}
}

func assertClosedObjectSchema(t *testing.T, label string, schema any, required []string, enums map[string][]string) {
	t.Helper()
	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
		Required             []string `json:"required"`
		AdditionalProperties any      `json:"additionalProperties"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.AdditionalProperties != false || !sameStrings(decoded.Required, required) || len(decoded.Properties) != len(required) {
		t.Fatalf("%s schema = %s", label, data)
	}
	for property, want := range enums {
		if !sameStrings(decoded.Properties[property].Enum, want) {
			t.Fatalf("%s %s enum = %v, want %v", label, property, decoded.Properties[property].Enum, want)
		}
	}
}
