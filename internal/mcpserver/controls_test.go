package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestKeyboardAndMouseSchemasEnforceOperationSpecificArguments(t *testing.T) {
	tests := []struct {
		name   string
		schema *jsonschema.Schema
		input  map[string]any
		valid  bool
	}{
		{name: "keyboard type text", schema: keyboardSchema(), input: map[string]any{"device": "lab", "operation": "type_text", "text": "hello"}, valid: true},
		{name: "keyboard type text missing text", schema: keyboardSchema(), input: map[string]any{"device": "lab", "operation": "type_text"}},
		{name: "keyboard type text rejects key", schema: keyboardSchema(), input: map[string]any{"device": "lab", "operation": "type_text", "text": "hello", "key": "enter"}},
		{name: "keyboard type text rejects modifier", schema: keyboardSchema(), input: map[string]any{"device": "lab", "operation": "type_text", "text": "hello", "modifiers": []any{"ctrl"}}},
		{name: "keyboard type text rejects extra", schema: keyboardSchema(), input: map[string]any{"device": "lab", "operation": "type_text", "text": "hello", "extra": true}},
		{name: "keyboard type text rejects oversized text", schema: keyboardSchema(), input: map[string]any{"device": "lab", "operation": "type_text", "text": string(make([]byte, 4097))}},
		{name: "keyboard press key", schema: keyboardSchema(), input: map[string]any{"device": "lab", "operation": "press_key", "key": "enter", "modifiers": []any{"ctrl"}}, valid: true},
		{name: "keyboard press key missing key", schema: keyboardSchema(), input: map[string]any{"device": "lab", "operation": "press_key"}},
		{name: "keyboard press key rejects text", schema: keyboardSchema(), input: map[string]any{"device": "lab", "operation": "press_key", "key": "enter", "text": "hello"}},
		{name: "keyboard press key rejects invalid modifier", schema: keyboardSchema(), input: map[string]any{"device": "lab", "operation": "press_key", "key": "enter", "modifiers": []any{"caps_lock"}}},
		{name: "mouse move absolute", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "move_absolute", "x": 0, "y": 32767}, valid: true},
		{name: "mouse move absolute missing y", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "move_absolute", "x": 1}},
		{name: "mouse move absolute rejects relative coordinate", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "move_absolute", "x": 1, "y": 2, "dx": 1}},
		{name: "mouse move absolute rejects range", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "move_absolute", "x": 32768, "y": 2}},
		{name: "mouse move relative", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "move_relative", "dx": -128, "dy": 127}, valid: true},
		{name: "mouse move relative missing dy", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "move_relative", "dx": 1}},
		{name: "mouse move relative rejects absolute coordinate", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "move_relative", "dx": 1, "dy": 2, "x": 1}},
		{name: "mouse move relative rejects range", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "move_relative", "dx": -129, "dy": 1}},
		{name: "mouse click", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "click", "button": "left"}, valid: true},
		{name: "mouse click missing button", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "click"}},
		{name: "mouse click rejects coordinate", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "click", "button": "left", "x": 1}},
		{name: "mouse click rejects enum", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "click", "button": "primary"}},
		{name: "mouse scroll wheel x", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "scroll", "wheel_x": 1}, valid: true},
		{name: "mouse scroll wheel y", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "scroll", "wheel_y": -1}, valid: true},
		{name: "mouse scroll missing wheels", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "scroll"}},
		{name: "mouse scroll rejects zero deltas", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "scroll", "wheel_x": 0, "wheel_y": 0}},
		{name: "mouse scroll rejects coordinate", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "scroll", "wheel_x": 1, "x": 1}},
		{name: "mouse scroll rejects range", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "scroll", "wheel_y": 128}},
		{name: "mouse rejects extra", schema: mouseSchema(), input: map[string]any{"device": "lab", "operation": "click", "button": "left", "extra": true}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved, err := test.schema.Resolve(nil)
			if err != nil {
				t.Fatal(err)
			}
			err = resolved.Validate(test.input)
			if test.valid && err != nil {
				t.Fatalf("Validate(%v) = %v", test.input, err)
			}
			if !test.valid && err == nil {
				t.Fatalf("Validate(%v) succeeded", test.input)
			}
		})
	}
}

func TestMouseValidationUsesFirmwareInt8Ranges(t *testing.T) {
	zero, tooHigh, tooLow := 0, 128, -129
	for _, request := range []MouseRequest{
		{Operation: MouseMoveRelative, DX: &tooHigh, DY: &zero},
		{Operation: MouseMoveRelative, DX: &zero, DY: &tooLow},
		{Operation: MouseScroll, WheelX: 128},
		{Operation: MouseScroll, WheelY: -129},
	} {
		if err := validateMouseInput("lab", request); err == nil {
			t.Fatalf("validateMouseInput(%+v) error = nil", request)
		}
	}
}

type controlValidationRecordingDevice struct {
	recordingDevice
	mu            sync.Mutex
	keyboardCalls int
	mouseCalls    int
}

func (device *controlValidationRecordingDevice) Keyboard(context.Context, string, KeyboardRequest) (KeyboardResult, error) {
	device.mu.Lock()
	defer device.mu.Unlock()
	device.keyboardCalls++
	return KeyboardResult{}, nil
}

func (device *controlValidationRecordingDevice) Mouse(context.Context, string, MouseRequest) (MouseResult, error) {
	device.mu.Lock()
	defer device.mu.Unlock()
	device.mouseCalls++
	return MouseResult{}, nil
}

func TestMalformedControlCallsAreSchemaRejectedBeforeProviderDispatch(t *testing.T) {
	device := &controlValidationRecordingDevice{}
	clientSession, cleanup := connectVirtualMediaTestClient(t, device)
	defer cleanup()

	for _, test := range []struct {
		name      string
		arguments map[string]any
	}{
		{name: KeyboardToolName, arguments: map[string]any{"device": "lab", "operation": "type_text"}},
		{name: KeyboardToolName, arguments: map[string]any{"device": "lab", "operation": "press_key", "key": "enter", "text": "private-argument"}},
		{name: MouseToolName, arguments: map[string]any{"device": "lab", "operation": "move_absolute", "x": 32768, "y": 1}},
		{name: MouseToolName, arguments: map[string]any{"device": "lab", "operation": "click", "button": "left", "wheel_x": 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: test.name, Arguments: test.arguments})
			if err != nil || result == nil || !result.IsError {
				t.Fatalf("CallTool = %#v, %v", result, err)
			}
			content, err := json.Marshal(result.Content)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), "validating \\\"arguments\\\"") || strings.Contains(string(content), "private-argument") {
				t.Fatalf("schema error content = %s", content)
			}
		})
	}

	device.mu.Lock()
	defer device.mu.Unlock()
	if device.keyboardCalls != 0 || device.mouseCalls != 0 {
		t.Fatalf("malformed calls reached provider: keyboard=%d mouse=%d", device.keyboardCalls, device.mouseCalls)
	}
}
