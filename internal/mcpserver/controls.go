package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	CaptureScreenToolName = "jetkvm_capture_screen"
	KeyboardToolName      = "jetkvm_keyboard"
	MouseToolName         = "jetkvm_mouse"
	VirtualMediaToolName  = "jetkvm_virtual_media"
)

type CaptureRequest struct {
	MaxWidth  int `json:"maxWidth,omitempty"`
	MaxHeight int `json:"maxHeight,omitempty"`
}

type CaptureResult struct {
	Device     string    `json:"device"`
	CapturedAt time.Time `json:"capturedAt"`
	MIMEType   string    `json:"mimeType"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	PNG        []byte    `json:"-"`
}

type CaptureOutput struct {
	Device     string    `json:"device"`
	CapturedAt time.Time `json:"capturedAt"`
	MIMEType   string    `json:"mimeType"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	SizeBytes  int       `json:"sizeBytes"`
}

type KeyboardOperation string

const (
	KeyboardTypeText KeyboardOperation = "type_text"
	KeyboardPressKey KeyboardOperation = "press_key"
)

type KeyboardRequest struct {
	Operation KeyboardOperation `json:"operation"`
	Text      string            `json:"text,omitempty"`
	Key       string            `json:"key,omitempty"`
	Modifiers []string          `json:"modifiers,omitempty"`
}

type KeyboardResult struct {
	Device    string            `json:"device"`
	Operation KeyboardOperation `json:"operation"`
	Status    string            `json:"status"`
}

type MouseOperation string

const (
	MouseMoveAbsolute MouseOperation = "move_absolute"
	MouseMoveRelative MouseOperation = "move_relative"
	MouseClick        MouseOperation = "click"
	MouseScroll       MouseOperation = "scroll"
)

type MouseRequest struct {
	Operation MouseOperation `json:"operation"`
	X         *int           `json:"x,omitempty"`
	Y         *int           `json:"y,omitempty"`
	DX        *int           `json:"dx,omitempty"`
	DY        *int           `json:"dy,omitempty"`
	Button    string         `json:"button,omitempty"`
	WheelX    int            `json:"wheelX,omitempty"`
	WheelY    int            `json:"wheelY,omitempty"`
}

type MouseResult struct {
	Device    string         `json:"device"`
	Operation MouseOperation `json:"operation"`
	Status    string         `json:"status"`
}

type VirtualMediaOperation string

const (
	VirtualMediaStatus    VirtualMediaOperation = "status"
	VirtualMediaMountURL  VirtualMediaOperation = "mount_url"
	VirtualMediaMountFile VirtualMediaOperation = "mount_file"
	VirtualMediaUnmount   VirtualMediaOperation = "unmount"
	VirtualMediaUpload    VirtualMediaOperation = "upload"
)

type VirtualMediaRequest struct {
	Operation VirtualMediaOperation `json:"operation"`
	Source    string                `json:"source,omitempty"`
	Mode      string                `json:"mode,omitempty"`
}

type VirtualMediaResult struct {
	Device    string                `json:"device"`
	Operation VirtualMediaOperation `json:"operation"`
	Mounted   bool                  `json:"mounted"`
	Source    string                `json:"source,omitempty"`
	Mode      string                `json:"mode,omitempty"`
	Status    string                `json:"status"`
}

type captureInput struct {
	Device    string `json:"device" jsonschema:"JetKVM device name from the server configuration"`
	MaxWidth  int    `json:"max_width,omitempty" jsonschema:"maximum PNG width; omit to use the configured default"`
	MaxHeight int    `json:"max_height,omitempty" jsonschema:"maximum PNG height; omit to use the configured default"`
}

type keyboardInput struct {
	Device    string            `json:"device" jsonschema:"JetKVM device name from the server configuration"`
	Operation KeyboardOperation `json:"operation" jsonschema:"typed keyboard operation"`
	Text      string            `json:"text,omitempty" jsonschema:"text to type for type_text"`
	Key       string            `json:"key,omitempty" jsonschema:"named key or one printable character for press_key"`
	Modifiers []string          `json:"modifiers,omitempty" jsonschema:"optional ctrl, alt, shift, meta modifiers"`
}

type mouseInput struct {
	Device    string         `json:"device" jsonschema:"JetKVM device name from the server configuration"`
	Operation MouseOperation `json:"operation" jsonschema:"typed mouse operation"`
	X         *int           `json:"x,omitempty" jsonschema:"absolute x coordinate from 0 through 32767"`
	Y         *int           `json:"y,omitempty" jsonschema:"absolute y coordinate from 0 through 32767"`
	DX        *int           `json:"dx,omitempty" jsonschema:"relative horizontal movement"`
	DY        *int           `json:"dy,omitempty" jsonschema:"relative vertical movement"`
	Button    string         `json:"button,omitempty" jsonschema:"left, middle, or right button for click"`
	WheelX    int            `json:"wheel_x,omitempty" jsonschema:"horizontal wheel movement"`
	WheelY    int            `json:"wheel_y,omitempty" jsonschema:"vertical wheel movement"`
}

type virtualMediaInput struct {
	Device    string                `json:"device" jsonschema:"JetKVM device name from the server configuration"`
	Operation VirtualMediaOperation `json:"operation" jsonschema:"typed virtual-media operation"`
	Source    string                `json:"source,omitempty" jsonschema:"URL or configured-media-directory path required by the selected operation"`
	Mode      string                `json:"mode,omitempty" jsonschema:"read_only or read_write; defaults to read_only"`
}

func addControlTools(server *mcp.Server, device Device) {
	addReadTool(server, &mcp.Tool{
		Name:        CaptureScreenToolName,
		Description: "Capture one fresh PNG from the host display attached to a configured JetKVM.",
		InputSchema: captureSchema(),
		Annotations: annotations(true, false, true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input captureInput) (*mcp.CallToolResult, CaptureOutput, error) {
		if err := validDevice(input.Device); err != nil {
			return nil, CaptureOutput{}, invalidInput(err)
		}
		if input.MaxWidth < 0 || input.MaxHeight < 0 {
			return nil, CaptureOutput{}, invalidInput(errors.New("capture dimensions must be positive"))
		}
		capture, err := device.CaptureScreen(ctx, input.Device, CaptureRequest{MaxWidth: input.MaxWidth, MaxHeight: input.MaxHeight})
		if err != nil {
			return nil, CaptureOutput{}, err
		}
		if capture.MIMEType != "image/png" || len(capture.PNG) == 0 || capture.Width <= 0 || capture.Height <= 0 {
			return nil, CaptureOutput{}, errors.New("device returned an invalid screen capture")
		}
		output := CaptureOutput{
			Device: capture.Device, CapturedAt: capture.CapturedAt, MIMEType: capture.MIMEType,
			Width: capture.Width, Height: capture.Height, SizeBytes: len(capture.PNG),
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.ImageContent{MIMEType: capture.MIMEType, Data: append([]byte(nil), capture.PNG...)}}}, output, nil
	})

	addMutationTool(server, &mcp.Tool{
		Name:        KeyboardToolName,
		Description: "Type bounded text or press one named key through JetKVM USB HID.",
		InputSchema: operationSchema[keyboardInput]([]string{string(KeyboardTypeText), string(KeyboardPressKey)}),
		Annotations: annotations(false, false, false),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input keyboardInput) (*mcp.CallToolResult, KeyboardResult, error) {
		request := KeyboardRequest{Operation: input.Operation, Text: input.Text, Key: input.Key, Modifiers: input.Modifiers}
		if err := validateKeyboardInput(input.Device, request); err != nil {
			return nil, KeyboardResult{}, invalidInput(err)
		}
		result, err := device.Keyboard(ctx, input.Device, request)
		return nil, result, err
	})

	addMutationTool(server, &mcp.Tool{
		Name:        MouseToolName,
		Description: "Move, click, or scroll the host pointer through JetKVM USB HID.",
		InputSchema: mouseSchema(),
		Annotations: annotations(false, false, false),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input mouseInput) (*mcp.CallToolResult, MouseResult, error) {
		request := MouseRequest{
			Operation: input.Operation, X: input.X, Y: input.Y, DX: input.DX, DY: input.DY,
			Button: input.Button, WheelX: input.WheelX, WheelY: input.WheelY,
		}
		if err := validateMouseInput(input.Device, request); err != nil {
			return nil, MouseResult{}, invalidInput(err)
		}
		result, err := device.Mouse(ctx, input.Device, request)
		return nil, result, err
	})

	addConditionalMutationTool(server, &mcp.Tool{
		Name:        VirtualMediaToolName,
		Description: "Inspect, mount, unmount, or upload JetKVM virtual media using a URL or configured media-directory path.",
		InputSchema: operationSchema[virtualMediaInput]([]string{string(VirtualMediaStatus), string(VirtualMediaMountURL), string(VirtualMediaMountFile), string(VirtualMediaUnmount), string(VirtualMediaUpload)}),
		Annotations: annotations(false, true, false),
	}, func(input virtualMediaInput) bool {
		return input.Operation != VirtualMediaStatus
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input virtualMediaInput) (*mcp.CallToolResult, VirtualMediaResult, error) {
		request := VirtualMediaRequest{Operation: input.Operation, Source: input.Source, Mode: input.Mode}
		if err := validateVirtualMediaInput(input.Device, request); err != nil {
			return nil, VirtualMediaResult{}, invalidInput(err)
		}
		result, err := device.VirtualMedia(ctx, input.Device, request)
		return nil, result, err
	})
}

func captureSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[captureInput](nil)
	if err != nil {
		panic(fmt.Sprintf("capture input schema: %v", err))
	}
	setIntegerRange(schema.Properties["max_width"], 1, 3840)
	setIntegerRange(schema.Properties["max_height"], 1, 2160)
	return schema
}

func mouseSchema() *jsonschema.Schema {
	schema := operationSchema[mouseInput]([]string{string(MouseMoveAbsolute), string(MouseMoveRelative), string(MouseClick), string(MouseScroll)})
	for _, name := range []string{"dx", "dy", "wheel_x", "wheel_y"} {
		setIntegerRange(schema.Properties[name], -128, 127)
	}
	return schema
}

func operationSchema[T any](values []string) *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("tool input schema: %v", err))
	}
	property := schema.Properties["operation"]
	property.Type = "string"
	property.Types = nil
	property.Enum = make([]any, len(values))
	for index, value := range values {
		property.Enum[index] = value
	}
	return schema
}

func setIntegerRange(schema *jsonschema.Schema, minimum, maximum int) {
	min := float64(minimum)
	max := float64(maximum)
	schema.Minimum = &min
	schema.Maximum = &max
}

func validateKeyboardInput(device string, request KeyboardRequest) error {
	if err := validDevice(device); err != nil {
		return err
	}
	switch request.Operation {
	case KeyboardTypeText:
		if request.Text == "" || len(request.Text) > 4096 || request.Key != "" || len(request.Modifiers) != 0 {
			return errors.New("type_text requires text only, up to 4096 bytes")
		}
	case KeyboardPressKey:
		if strings.TrimSpace(request.Key) == "" || request.Text != "" {
			return errors.New("press_key requires key and no text")
		}
		for _, modifier := range request.Modifiers {
			switch modifier {
			case "ctrl", "alt", "shift", "meta":
			default:
				return errors.New("unsupported keyboard modifier")
			}
		}
	default:
		return errors.New("unsupported keyboard operation")
	}
	return nil
}

func validateMouseInput(device string, request MouseRequest) error {
	if err := validDevice(device); err != nil {
		return err
	}
	switch request.Operation {
	case MouseMoveAbsolute:
		if !coordinatesInRange(request.X, request.Y, 0, 32767) {
			return errors.New("move_absolute requires x and y from 0 through 32767")
		}
	case MouseMoveRelative:
		if !coordinatesInRange(request.DX, request.DY, -128, 127) {
			return errors.New("move_relative requires dx and dy from -128 through 127")
		}
	case MouseClick:
		if request.Button != "left" && request.Button != "middle" && request.Button != "right" {
			return errors.New("click requires a supported button")
		}
	case MouseScroll:
		if request.WheelX < -128 || request.WheelX > 127 || request.WheelY < -128 || request.WheelY > 127 || request.WheelX == 0 && request.WheelY == 0 {
			return errors.New("scroll requires wheel_x or wheel_y from -128 through 127")
		}
	default:
		return errors.New("unsupported mouse operation")
	}
	return nil
}

func coordinatesInRange(x, y *int, minimum, maximum int) bool {
	return x != nil && y != nil && *x >= minimum && *x <= maximum && *y >= minimum && *y <= maximum
}

func validateVirtualMediaInput(device string, request VirtualMediaRequest) error {
	if err := validDevice(device); err != nil {
		return err
	}
	if request.Mode != "" && request.Mode != "read_only" && request.Mode != "read_write" {
		return errors.New("mode must be read_only or read_write")
	}
	source := strings.TrimSpace(request.Source)
	switch request.Operation {
	case VirtualMediaStatus, VirtualMediaUnmount:
		if source != "" {
			return errors.New("operation does not accept source")
		}
	case VirtualMediaMountURL, VirtualMediaMountFile, VirtualMediaUpload:
		if source == "" {
			return errors.New("operation requires source")
		}
	default:
		return errors.New("unsupported virtual media operation")
	}
	return nil
}
