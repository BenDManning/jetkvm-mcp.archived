package jetkvm

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
)

const hidReleaseTimeout = 2 * time.Second

type keyReport struct {
	modifier byte
	usage    int
}

var namedKeyboardUsages = map[string]int{
	"enter": 0x28, "escape": 0x29, "esc": 0x29, "backspace": 0x2a,
	"tab": 0x2b, "space": 0x2c, "insert": 0x49, "home": 0x4a,
	"pageup": 0x4b, "page_up": 0x4b, "delete": 0x4c, "del": 0x4c,
	"end": 0x4d, "pagedown": 0x4e, "page_down": 0x4e,
	"arrowright": 0x4f, "right": 0x4f, "arrowleft": 0x50, "left": 0x50,
	"arrowdown": 0x51, "down": 0x51, "arrowup": 0x52, "up": 0x52,
	"f1": 0x3a, "f2": 0x3b, "f3": 0x3c, "f4": 0x3d,
	"f5": 0x3e, "f6": 0x3f, "f7": 0x40, "f8": 0x41,
	"f9": 0x42, "f10": 0x43, "f11": 0x44, "f12": 0x45,
}

var punctuationUsages = map[byte]keyReport{
	'-': {usage: 0x2d}, '_': {modifier: 0x02, usage: 0x2d},
	'=': {usage: 0x2e}, '+': {modifier: 0x02, usage: 0x2e},
	'[': {usage: 0x2f}, '{': {modifier: 0x02, usage: 0x2f},
	']': {usage: 0x30}, '}': {modifier: 0x02, usage: 0x30},
	'\\': {usage: 0x31}, '|': {modifier: 0x02, usage: 0x31},
	';': {usage: 0x33}, ':': {modifier: 0x02, usage: 0x33},
	'\'': {usage: 0x34}, '"': {modifier: 0x02, usage: 0x34},
	'`': {usage: 0x35}, '~': {modifier: 0x02, usage: 0x35},
	',': {usage: 0x36}, '<': {modifier: 0x02, usage: 0x36},
	'.': {usage: 0x37}, '>': {modifier: 0x02, usage: 0x37},
	'/': {usage: 0x38}, '?': {modifier: 0x02, usage: 0x38},
}

func (manager *Manager) Keyboard(ctx context.Context, name string, request mcpserver.KeyboardRequest) (mcpserver.KeyboardResult, error) {
	device, err := manager.device(name)
	if err != nil {
		return mcpserver.KeyboardResult{}, err
	}
	reports, err := keyboardReports(request)
	if err != nil {
		return mcpserver.KeyboardResult{}, classifyOperationError(err, ToolOutcomeNotSent)
	}
	err = manager.withOperation(ctx, device, true, false, func() error {
		return manager.withSession(ctx, device, func(operationCtx context.Context, session Session) error {
			pressed := false
			defer func() {
				if pressed {
					bestEffortKeyboardRelease(session)
				}
			}()
			for index, report := range reports {
				pressed = true
				if err := session.Call(operationCtx, "keyboardReport", map[string]any{"modifier": report.modifier, "keys": []int{report.usage}}, nil); err != nil {
					return mutationSequenceError(err, index > 0)
				}
				if err := releaseKeyboard(operationCtx, session); err != nil {
					return mutationSequenceError(err, true)
				}
				pressed = false
			}
			return nil
		})
	})
	if err != nil {
		return mcpserver.KeyboardResult{}, err
	}
	return mcpserver.KeyboardResult{Device: device.Name, Operation: request.Operation, Status: mcpserver.ResultStatusCompleted}, nil
}

func (manager *Manager) Mouse(ctx context.Context, name string, request mcpserver.MouseRequest) (mcpserver.MouseResult, error) {
	device, err := manager.device(name)
	if err != nil {
		return mcpserver.MouseResult{}, err
	}
	calls, err := mouseCalls(request)
	if err != nil {
		return mcpserver.MouseResult{}, classifyOperationError(err, ToolOutcomeNotSent)
	}
	err = manager.withOperation(ctx, device, true, false, func() error {
		return manager.withSession(ctx, device, func(operationCtx context.Context, session Session) error {
			pressed := false
			defer func() {
				if pressed {
					bestEffortMouseRelease(session)
				}
			}()
			for index, call := range calls {
				if request.Operation == mcpserver.MouseClick && index == 0 {
					pressed = true
				}
				if err := session.Call(operationCtx, call.method, call.params, nil); err != nil {
					return mutationSequenceError(err, index > 0)
				}
				if request.Operation == mcpserver.MouseClick && index == len(calls)-1 {
					pressed = false
				}
			}
			return nil
		})
	})
	if err != nil {
		return mcpserver.MouseResult{}, err
	}
	return mcpserver.MouseResult{Device: device.Name, Operation: request.Operation, Status: mcpserver.ResultStatusCompleted}, nil
}

func mutationSequenceError(err error, priorDispatch bool) error {
	if err == nil || !priorDispatch {
		return err
	}
	return classifyOperationError(err, ToolOutcomeUnknown)
}

func keyboardReports(request mcpserver.KeyboardRequest) ([]keyReport, error) {
	switch request.Operation {
	case mcpserver.KeyboardTypeText:
		if request.Text == "" || !utf8.ValidString(request.Text) {
			return nil, fmt.Errorf("%w: text", ErrUnsupportedInput)
		}
		reports := make([]keyReport, 0, len(request.Text))
		for _, value := range []byte(request.Text) {
			report, ok := keyboardReportForByte(value)
			if !ok {
				return nil, fmt.Errorf("%w: text requires US-ASCII", ErrUnsupportedInput)
			}
			reports = append(reports, report)
		}
		return reports, nil
	case mcpserver.KeyboardPressKey:
		report, ok := keyboardReportForKey(request.Key, request.Modifiers)
		if !ok {
			return nil, fmt.Errorf("%w: key or modifier", ErrUnsupportedInput)
		}
		return []keyReport{report}, nil
	default:
		return nil, fmt.Errorf("%w: keyboard operation", ErrUnsupportedInput)
	}
}

func keyboardReportForByte(value byte) (keyReport, bool) {
	switch {
	case value >= 'a' && value <= 'z':
		return keyReport{usage: int(0x04 + value - 'a')}, true
	case value >= 'A' && value <= 'Z':
		return keyReport{modifier: 0x02, usage: int(0x04 + value - 'A')}, true
	case value >= '1' && value <= '9':
		return keyReport{usage: int(0x1e + value - '1')}, true
	case value == '0':
		return keyReport{usage: 0x27}, true
	case value == '\n' || value == '\r':
		return keyReport{usage: 0x28}, true
	case value == '\t':
		return keyReport{usage: 0x2b}, true
	case value == ' ':
		return keyReport{usage: 0x2c}, true
	case strings.ContainsRune("!@#$%^&*()", rune(value)):
		index := strings.IndexByte("!@#$%^&*()", value)
		usage := 0x1e + index
		if index == 9 {
			usage = 0x27
		}
		return keyReport{modifier: 0x02, usage: usage}, true
	default:
		report, ok := punctuationUsages[value]
		return report, ok
	}
}

func keyboardReportForKey(key string, modifiers []string) (keyReport, bool) {
	trimmed := strings.TrimSpace(key)
	var report keyReport
	var ok bool
	if len(trimmed) == 1 {
		report, ok = keyboardReportForByte(trimmed[0])
	} else {
		report.usage, ok = namedKeyboardUsages[strings.ToLower(trimmed)]
	}
	if !ok {
		return keyReport{}, false
	}
	for _, modifier := range modifiers {
		switch strings.ToLower(strings.TrimSpace(modifier)) {
		case "ctrl", "control":
			report.modifier |= 0x01
		case "shift":
			report.modifier |= 0x02
		case "alt":
			report.modifier |= 0x04
		case "meta", "super", "command":
			report.modifier |= 0x08
		default:
			return keyReport{}, false
		}
	}
	return report, true
}

func releaseKeyboard(ctx context.Context, session Session) error {
	return session.Call(ctx, "keyboardReport", map[string]any{"modifier": byte(0), "keys": []int{}}, nil)
}

func bestEffortKeyboardRelease(session Session) {
	ctx, cancel := context.WithTimeout(context.Background(), hidReleaseTimeout)
	defer cancel()
	_ = releaseKeyboard(ctx, session)
}

func bestEffortMouseRelease(session Session) {
	ctx, cancel := context.WithTimeout(context.Background(), hidReleaseTimeout)
	defer cancel()
	_ = session.Call(ctx, "relMouseReport", map[string]any{"dx": int8(0), "dy": int8(0), "buttons": uint8(0)}, nil)
}

type deviceCall struct {
	method string
	params any
}

func mouseCalls(request mcpserver.MouseRequest) ([]deviceCall, error) {
	switch request.Operation {
	case mcpserver.MouseMoveAbsolute:
		if request.X == nil || request.Y == nil || *request.X < 0 || *request.X > 32767 || *request.Y < 0 || *request.Y > 32767 {
			return nil, fmt.Errorf("%w: absolute coordinates", ErrUnsupportedInput)
		}
		return []deviceCall{{"absMouseReport", map[string]any{"x": *request.X, "y": *request.Y, "buttons": uint8(0)}}}, nil
	case mcpserver.MouseMoveRelative:
		if request.DX == nil || request.DY == nil || *request.DX < -128 || *request.DX > 127 || *request.DY < -128 || *request.DY > 127 {
			return nil, fmt.Errorf("%w: relative coordinates", ErrUnsupportedInput)
		}
		return []deviceCall{{"relMouseReport", map[string]any{"dx": int8(*request.DX), "dy": int8(*request.DY), "buttons": uint8(0)}}}, nil
	case mcpserver.MouseClick:
		buttons := map[string]uint8{"left": 1, "right": 2, "middle": 4}
		button, ok := buttons[strings.ToLower(strings.TrimSpace(request.Button))]
		if !ok {
			return nil, fmt.Errorf("%w: mouse button", ErrUnsupportedInput)
		}
		return []deviceCall{
			{"relMouseReport", map[string]any{"dx": int8(0), "dy": int8(0), "buttons": button}},
			{"relMouseReport", map[string]any{"dx": int8(0), "dy": int8(0), "buttons": uint8(0)}},
		}, nil
	case mcpserver.MouseScroll:
		if request.WheelX < -128 || request.WheelX > 127 || request.WheelY < -128 || request.WheelY > 127 || request.WheelX == 0 && request.WheelY == 0 {
			return nil, fmt.Errorf("%w: wheel movement", ErrUnsupportedInput)
		}
		return []deviceCall{{"wheelReport", map[string]any{"wheelY": int8(request.WheelY), "wheelX": int8(request.WheelX)}}}, nil
	default:
		return nil, fmt.Errorf("%w: mouse operation", ErrUnsupportedInput)
	}
}
