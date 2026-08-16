package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type boundedInputDevice struct {
	recordingDevice
	statusCalls   int
	powerCalls    int
	keyboardCalls int
}

func (device *boundedInputDevice) Status(_ context.Context, name string) (Status, error) {
	device.statusCalls++
	return Status{Device: name, Connected: true}, nil
}

func (device *boundedInputDevice) Power(_ context.Context, name string, action PowerAction, target string) (PowerResult, error) {
	device.powerCalls++
	return PowerResult{Device: name, Action: action, Target: target, Status: "completed"}, nil
}

func (device *boundedInputDevice) Keyboard(_ context.Context, name string, request KeyboardRequest) (KeyboardResult, error) {
	device.keyboardCalls++
	return KeyboardResult{Device: name, Operation: request.Operation, Status: "completed"}, nil
}

func TestPublicToolInputsEnforceUnicodeBoundsWithoutReflectingValues(t *testing.T) {
	device := new(boundedInputDevice)
	session, cleanup := connectVirtualMediaTestClient(t, device)
	defer cleanup()

	boundaryAlias := strings.Repeat("界", 128)
	boundaryTarget := strings.Repeat("機", 128)
	for _, call := range []mcp.CallToolParams{
		{Name: GetStatusToolName, Arguments: map[string]any{"device": boundaryAlias}},
		{Name: WakeHostLANToolName, Arguments: map[string]any{"device": boundaryAlias, "target": boundaryTarget}},
		{Name: KeyboardToolName, Arguments: map[string]any{
			"device": boundaryAlias, "operation": "press_key", "key": strings.Repeat("界", 31) + "k",
			"modifiers": []any{"ctrl", "alt", "shift", "meta"},
		}},
	} {
		result, err := session.CallTool(context.Background(), &call)
		if err != nil || result == nil || result.IsError {
			t.Fatalf("CallTool(%s) boundary input = %#v, %v", call.Name, result, err)
		}
	}
	if device.statusCalls != 1 || device.powerCalls != 1 || device.keyboardCalls != 1 {
		t.Fatalf("boundary dispatches = status %d, power %d, keyboard %d", device.statusCalls, device.powerCalls, device.keyboardCalls)
	}

	privateSentinel := "PRIVATE-BOUNDARY-SENTINEL"
	invalid := []mcp.CallToolParams{
		{Name: GetStatusToolName, Arguments: map[string]any{"device": ""}},
		{Name: GetStatusToolName, Arguments: map[string]any{"device": strings.Repeat(privateSentinel, 13)}},
		{Name: GetStatusToolName, Arguments: map[string]any{"device": "lab", "unknown": privateSentinel}},
		{Name: WakeHostLANToolName, Arguments: map[string]any{"device": "lab", "target": ""}},
		{Name: WakeHostLANToolName, Arguments: map[string]any{"device": "lab", "target": strings.Repeat(privateSentinel, 13)}},
		{Name: KeyboardToolName, Arguments: map[string]any{"device": "lab", "operation": "press_key", "key": strings.Repeat(privateSentinel, 3)}},
		{Name: KeyboardToolName, Arguments: map[string]any{"device": "lab", "operation": "press_key", "key": strings.Repeat("界", 32) + "k"}},
		{Name: KeyboardToolName, Arguments: map[string]any{"device": "lab", "operation": "press_key", "key": "enter", "modifiers": []any{"ctrl", "ctrl"}}},
		{Name: KeyboardToolName, Arguments: map[string]any{"device": "lab", "operation": "press_key", "key": "enter", "modifiers": []any{"ctrl", "alt", "shift", "meta", "caps_lock"}}},
	}
	for _, call := range invalid {
		result, err := session.CallTool(context.Background(), &call)
		if err != nil || result == nil || !result.IsError {
			t.Fatalf("CallTool(%s) invalid input = %#v, %v", call.Name, result, err)
		}
		failure := decodeToolError(t, result)
		if failure.Code != "invalid_input" || failure.Outcome != "not_sent" || failure.Retryable {
			t.Fatalf("CallTool(%s) failure = %+v", call.Name, failure)
		}
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), privateSentinel) {
			t.Fatalf("CallTool(%s) reflected a rejected value: %s", call.Name, encoded)
		}
	}
	if device.statusCalls != 1 || device.powerCalls != 1 || device.keyboardCalls != 1 {
		t.Fatalf("rejected inputs reached provider: status %d, power %d, keyboard %d", device.statusCalls, device.powerCalls, device.keyboardCalls)
	}
}
