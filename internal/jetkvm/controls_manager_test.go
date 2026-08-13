package jetkvm

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

func TestManagerKeyboardUsesNumericHIDReportsAndRelease(t *testing.T) {
	session := &fakeSession{results: map[string]any{}}
	manager := testManager(t, session)
	result, err := manager.Keyboard(context.Background(), "lab", mcpserver.KeyboardRequest{Operation: mcpserver.KeyboardTypeText, Text: "A!\n"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || result.Operation != mcpserver.KeyboardTypeText {
		t.Fatalf("result = %+v", result)
	}
	want := []recordedCall{
		{method: "keyboardReport", params: map[string]any{"modifier": byte(2), "keys": []int{4}}},
		{method: "keyboardReport", params: map[string]any{"modifier": byte(0), "keys": []int{}}},
		{method: "keyboardReport", params: map[string]any{"modifier": byte(2), "keys": []int{30}}},
		{method: "keyboardReport", params: map[string]any{"modifier": byte(0), "keys": []int{}}},
		{method: "keyboardReport", params: map[string]any{"modifier": byte(0), "keys": []int{40}}},
		{method: "keyboardReport", params: map[string]any{"modifier": byte(0), "keys": []int{}}},
	}
	if !reflect.DeepEqual(session.calls, want) {
		t.Fatalf("calls = %#v, want %#v", session.calls, want)
	}
}

func TestManagerKeyboardNamedKeyAndModifiers(t *testing.T) {
	session := &fakeSession{results: map[string]any{}}
	manager := testManager(t, session)
	_, err := manager.Keyboard(context.Background(), "lab", mcpserver.KeyboardRequest{
		Operation: mcpserver.KeyboardPressKey, Key: "delete", Modifiers: []string{"ctrl", "alt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []recordedCall{
		{method: "keyboardReport", params: map[string]any{"modifier": byte(5), "keys": []int{76}}},
		{method: "keyboardReport", params: map[string]any{"modifier": byte(0), "keys": []int{}}},
	}
	if !reflect.DeepEqual(session.calls, want) {
		t.Fatalf("calls = %#v, want %#v", session.calls, want)
	}
}

func TestManagerKeyboardReleasesPressedKeyAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	session := &fakeSession{results: map[string]any{}}
	session.callHook = func(callCtx context.Context, method string, _ any) error {
		if method != "keyboardReport" {
			return nil
		}
		switch len(session.calls) {
		case 1:
			cancel()
			return nil
		case 2:
			return callCtx.Err()
		case 3:
			if callCtx.Err() != nil {
				t.Fatalf("cleanup context is canceled: %v", callCtx.Err())
			}
		}
		return nil
	}
	manager := testManager(t, session)
	_, err := manager.Keyboard(ctx, "lab", mcpserver.KeyboardRequest{Operation: mcpserver.KeyboardPressKey, Key: "a"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if len(session.calls) != 3 {
		t.Fatalf("calls = %#v, want press, failed release, cleanup release", session.calls)
	}
	wantRelease := map[string]any{"modifier": byte(0), "keys": []int{}}
	if !reflect.DeepEqual(session.calls[2].params, wantRelease) {
		t.Fatalf("cleanup release = %#v, want %#v", session.calls[2].params, wantRelease)
	}
}

func TestManagerMouseMapsOfficialRPCMethods(t *testing.T) {
	x, y, dx, dy := 1234, 5678, -12, 9
	for _, test := range []struct {
		name    string
		request mcpserver.MouseRequest
		want    []recordedCall
	}{
		{name: "absolute", request: mcpserver.MouseRequest{Operation: mcpserver.MouseMoveAbsolute, X: &x, Y: &y}, want: []recordedCall{{method: "absMouseReport", params: map[string]any{"x": 1234, "y": 5678, "buttons": uint8(0)}}}},
		{name: "relative", request: mcpserver.MouseRequest{Operation: mcpserver.MouseMoveRelative, DX: &dx, DY: &dy}, want: []recordedCall{{method: "relMouseReport", params: map[string]any{"dx": int8(-12), "dy": int8(9), "buttons": uint8(0)}}}},
		{name: "click", request: mcpserver.MouseRequest{Operation: mcpserver.MouseClick, Button: "left"}, want: []recordedCall{
			{method: "relMouseReport", params: map[string]any{"dx": int8(0), "dy": int8(0), "buttons": uint8(1)}},
			{method: "relMouseReport", params: map[string]any{"dx": int8(0), "dy": int8(0), "buttons": uint8(0)}},
		}},
		{name: "scroll", request: mcpserver.MouseRequest{Operation: mcpserver.MouseScroll, WheelX: -2, WheelY: 3}, want: []recordedCall{{method: "wheelReport", params: map[string]any{"wheelY": int8(3), "wheelX": int8(-2)}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			session := &fakeSession{results: map[string]any{}}
			manager := testManager(t, session)
			result, err := manager.Mouse(context.Background(), "lab", test.request)
			if err != nil {
				t.Fatal(err)
			}
			if result.Status != "completed" || !reflect.DeepEqual(session.calls, test.want) {
				t.Fatalf("result=%+v calls=%#v want=%#v", result, session.calls, test.want)
			}
		})
	}
}

func TestManagerMouseReleasesPressedButtonAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	session := &fakeSession{results: map[string]any{}}
	session.callHook = func(callCtx context.Context, method string, _ any) error {
		if method != "relMouseReport" {
			return nil
		}
		switch len(session.calls) {
		case 1:
			cancel()
			return nil
		case 2:
			return callCtx.Err()
		case 3:
			if callCtx.Err() != nil {
				t.Fatalf("cleanup context is canceled: %v", callCtx.Err())
			}
		}
		return nil
	}
	manager := testManager(t, session)
	_, err := manager.Mouse(ctx, "lab", mcpserver.MouseRequest{Operation: mcpserver.MouseClick, Button: "left"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if len(session.calls) != 3 {
		t.Fatalf("calls = %#v, want press, failed release, cleanup release", session.calls)
	}
	wantRelease := map[string]any{"dx": int8(0), "dy": int8(0), "buttons": uint8(0)}
	if !reflect.DeepEqual(session.calls[2].params, wantRelease) {
		t.Fatalf("cleanup release = %#v, want %#v", session.calls[2].params, wantRelease)
	}
}

func TestManagerRejectsUnsupportedKeyboardAndMouseInput(t *testing.T) {
	session := &fakeSession{results: map[string]any{}}
	manager := testManager(t, session)
	if _, err := manager.Keyboard(context.Background(), "lab", mcpserver.KeyboardRequest{Operation: mcpserver.KeyboardTypeText, Text: "é"}); !errors.Is(err, ErrUnsupportedInput) {
		t.Fatalf("keyboard error = %v", err)
	}
	tooLarge, zero := 128, 0
	if _, err := manager.Mouse(context.Background(), "lab", mcpserver.MouseRequest{Operation: mcpserver.MouseMoveRelative, DX: &tooLarge, DY: &zero}); !errors.Is(err, ErrUnsupportedInput) {
		t.Fatalf("mouse error = %v", err)
	}
	if len(session.calls) != 0 {
		t.Fatalf("calls = %#v", session.calls)
	}
}

func TestManagerKeyboardLocalValidationIsDefinitelyNotSent(t *testing.T) {
	manager := testManager(t, &fakeSession{results: map[string]any{}})
	_, err := manager.Keyboard(context.Background(), "lab", mcpserver.KeyboardRequest{Operation: mcpserver.KeyboardTypeText, Text: "é"})
	assertToolOutcome(t, err, ToolOutcomeNotSent)
}

func TestManagerKeyboardPartialSequenceHasUnknownOutcome(t *testing.T) {
	session := &fakeSession{results: map[string]any{}}
	session.callHook = func(_ context.Context, method string, _ any) error {
		if method == "keyboardReport" && len(session.calls) == 3 {
			return classifyOperationError(context.Canceled, ToolOutcomeNotSent)
		}
		return nil
	}
	manager := testManager(t, session)
	_, err := manager.Keyboard(context.Background(), "lab", mcpserver.KeyboardRequest{Operation: mcpserver.KeyboardTypeText, Text: "ab"})
	assertToolOutcome(t, err, ToolOutcomeUnknown)
}
