package mcpserver

import "testing"

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
