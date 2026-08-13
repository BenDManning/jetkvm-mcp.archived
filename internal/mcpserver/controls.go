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
	CaptureScreenToolName          = "jetkvm_capture_screen"
	KeyboardToolName               = "jetkvm_keyboard"
	MouseToolName                  = "jetkvm_mouse"
	VirtualMediaToolName           = "jetkvm_virtual_media"
	GetVirtualMediaStatusToolName  = "jetkvm_get_virtual_media_status"
	MountVirtualMediaURLToolName   = "jetkvm_mount_virtual_media_url"
	MountVirtualMediaFileToolName  = "jetkvm_mount_virtual_media_file"
	UnmountVirtualMediaToolName    = "jetkvm_unmount_virtual_media"
	UploadVirtualMediaFileToolName = "jetkvm_upload_virtual_media_file"
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

type VirtualMediaSourceType string

const (
	VirtualMediaSourceHTTP    VirtualMediaSourceType = "http"
	VirtualMediaSourceStorage VirtualMediaSourceType = "storage"
)

// VirtualMediaState is the privacy-safe public projection of firmware media
// state. It deliberately identifies only the source class, never a URL, path,
// filename, or unknown firmware field.
type VirtualMediaState struct {
	Mounted    bool                   `json:"mounted"`
	SourceType VirtualMediaSourceType `json:"sourceType,omitempty"`
	Mode       string                 `json:"mode,omitempty"`
}

type VirtualMediaResult struct {
	Device     string                 `json:"device"`
	Operation  VirtualMediaOperation  `json:"operation"`
	Mounted    bool                   `json:"mounted"`
	SourceType VirtualMediaSourceType `json:"sourceType,omitempty"`
	Mode       string                 `json:"mode,omitempty"`
	Status     string                 `json:"status"`
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

type virtualMediaStatusInput struct {
	Device string `json:"device" jsonschema:"JetKVM device name from the server configuration"`
}

type virtualMediaURLInput struct {
	Device string `json:"device" jsonschema:"JetKVM device name from the server configuration"`
	URL    string `json:"url" jsonschema:"HTTP(S) media URL fetched by the configured JetKVM"`
	Mode   string `json:"mode,omitempty" jsonschema:"read_only or read_write; defaults to read_only"`
}

type virtualMediaFileInput struct {
	Device string `json:"device" jsonschema:"JetKVM device name from the server configuration"`
	Path   string `json:"path" jsonschema:"relative path below the device's configured media directory"`
	Mode   string `json:"mode,omitempty" jsonschema:"read_only or read_write; defaults to read_only"`
}

type virtualMediaUploadInput struct {
	Device string `json:"device" jsonschema:"JetKVM device name from the server configuration"`
	Path   string `json:"path" jsonschema:"relative path below the device's configured media directory"`
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
		InputSchema: keyboardSchema(),
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

	addReadTool(server, &mcp.Tool{
		Name:         GetVirtualMediaStatusToolName,
		Description:  "Read the current virtual-media mount state of a configured JetKVM without changing it. Requires only a configured device. Returns the firmware-reported mount state from this read; it does not prove that a later mutation is safe.",
		OutputSchema: virtualMediaResultSchema(),
		Annotations:  annotations(true, false, true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input virtualMediaStatusInput) (*mcp.CallToolResult, VirtualMediaResult, error) {
		if err := validDevice(input.Device); err != nil {
			return nil, VirtualMediaResult{}, invalidInput(err)
		}
		result, err := device.VirtualMedia(ctx, input.Device, VirtualMediaRequest{Operation: VirtualMediaStatus})
		return nil, result, err
	})

	addSanitizedInputMutationTool(server, &mcp.Tool{
		Name:         MountVirtualMediaURLToolName,
		Description:  "Ask the configured JetKVM appliance to fetch an HTTP(S) URL whose scheme, host, and effective port match a configured exact origin, then replace its current virtual-media mount. URL mounting is unavailable without an allowed origin. Success acknowledges the firmware RPC but does not independently verify the resulting mount; inspect status before deciding whether another mutation is safe.",
		InputSchema:  virtualMediaURLSchema(),
		OutputSchema: virtualMediaResultSchema(),
		Annotations:  annotationsWithOpenWorld(false, true, false, true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input virtualMediaURLInput) (*mcp.CallToolResult, VirtualMediaResult, error) {
		request := VirtualMediaRequest{Operation: VirtualMediaMountURL, Source: input.URL, Mode: input.Mode}
		if err := validateVirtualMediaInput(input.Device, request); err != nil {
			return nil, VirtualMediaResult{}, invalidInput(err)
		}
		result, err := device.VirtualMedia(ctx, input.Device, request)
		return nil, result, err
	})

	addSanitizedInputMutationTool(server, &mcp.Tool{
		Name:         MountVirtualMediaFileToolName,
		Description:  "Upload one confined local media file and replace the configured JetKVM's current mount. Requires a configured media directory and a non-empty relative path confined beneath it. Success means upload and mount RPCs completed, not that an external observer verified the mount; inspect status before another mutation.",
		InputSchema:  virtualMediaFileSchema(),
		OutputSchema: virtualMediaResultSchema(),
		Annotations:  annotations(false, true, false),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input virtualMediaFileInput) (*mcp.CallToolResult, VirtualMediaResult, error) {
		request := VirtualMediaRequest{Operation: VirtualMediaMountFile, Source: input.Path, Mode: input.Mode}
		if err := validateVirtualMediaInput(input.Device, request); err != nil {
			return nil, VirtualMediaResult{}, invalidInput(err)
		}
		result, err := device.VirtualMedia(ctx, input.Device, request)
		return nil, result, err
	})

	addMutationTool(server, &mcp.Tool{
		Name:         UnmountVirtualMediaToolName,
		Description:  "Unmount the configured JetKVM's current virtual media. Requires only a configured device; the request is valid even when no media is mounted. Success acknowledges the firmware RPC but does not independently verify that media is absent; inspect status before another mutation.",
		OutputSchema: virtualMediaResultSchema(),
		Annotations:  annotations(false, true, true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input virtualMediaStatusInput) (*mcp.CallToolResult, VirtualMediaResult, error) {
		if err := validDevice(input.Device); err != nil {
			return nil, VirtualMediaResult{}, invalidInput(err)
		}
		result, err := device.VirtualMedia(ctx, input.Device, VirtualMediaRequest{Operation: VirtualMediaUnmount})
		return nil, result, err
	})

	addSanitizedInputMutationTool(server, &mcp.Tool{
		Name:         UploadVirtualMediaFileToolName,
		Description:  "Upload one confined local media file to the configured JetKVM without mounting it. Requires a configured media directory and a non-empty relative path confined beneath it. Success means the upload completed without a mount request; the appliance retains the stored file outside this process.",
		InputSchema:  virtualMediaUploadSchema(),
		OutputSchema: virtualMediaResultSchema(),
		Annotations:  annotations(false, true, false),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input virtualMediaUploadInput) (*mcp.CallToolResult, VirtualMediaResult, error) {
		request := VirtualMediaRequest{Operation: VirtualMediaUpload, Source: input.Path}
		if err := validateVirtualMediaInput(input.Device, request); err != nil {
			return nil, VirtualMediaResult{}, invalidInput(err)
		}
		result, err := device.VirtualMedia(ctx, input.Device, request)
		return nil, result, err
	})

	addSanitizedConditionalMutationTool(server, &mcp.Tool{
		Name:         VirtualMediaToolName,
		Description:  "Deprecated compatibility tool for virtual-media status, mount, unmount, and upload operations; use the one-purpose jetkvm_*_virtual_media* tools instead.",
		InputSchema:  operationSchema[virtualMediaInput]([]string{string(VirtualMediaStatus), string(VirtualMediaMountURL), string(VirtualMediaMountFile), string(VirtualMediaUnmount), string(VirtualMediaUpload)}),
		OutputSchema: virtualMediaResultSchema(),
		Annotations:  annotations(false, true, false),
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

func statusOutputSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[Status](nil)
	if err != nil {
		panic(fmt.Sprintf("status output schema: %v", err))
	}
	media := schema.Properties["virtualMedia"]
	media.Type = "object"
	media.Types = nil
	setStringEnum(media.Properties["sourceType"], []string{string(VirtualMediaSourceHTTP), string(VirtualMediaSourceStorage)})
	setStringEnum(media.Properties["mode"], []string{"read_only", "read_write"})
	return schema
}

func virtualMediaResultSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[VirtualMediaResult](nil)
	if err != nil {
		panic(fmt.Sprintf("virtual media output schema: %v", err))
	}
	setStringEnum(schema.Properties["operation"], []string{
		string(VirtualMediaStatus), string(VirtualMediaMountURL), string(VirtualMediaMountFile),
		string(VirtualMediaUnmount), string(VirtualMediaUpload),
	})
	setStringEnum(schema.Properties["sourceType"], []string{string(VirtualMediaSourceHTTP), string(VirtualMediaSourceStorage)})
	setStringEnum(schema.Properties["mode"], []string{"read_only", "read_write"})
	setStringEnum(schema.Properties["status"], []string{"observed", "completed"})
	return schema
}

func virtualMediaURLSchema() *jsonschema.Schema {
	schema := virtualMediaSchema[virtualMediaURLInput]("url")
	setStringEnum(schema.Properties["mode"], []string{"read_only", "read_write"})
	return schema
}

func virtualMediaFileSchema() *jsonschema.Schema {
	schema := virtualMediaSchema[virtualMediaFileInput]("path")
	setStringEnum(schema.Properties["mode"], []string{"read_only", "read_write"})
	return schema
}

func virtualMediaUploadSchema() *jsonschema.Schema {
	return virtualMediaSchema[virtualMediaUploadInput]("path")
}

func virtualMediaSchema[T any](sourceProperty string) *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("virtual media input schema: %v", err))
	}
	minimum, maximum := 1, 4096
	schema.Properties[sourceProperty].MinLength = &minimum
	schema.Properties[sourceProperty].MaxLength = &maximum
	return schema
}

func mouseSchema() *jsonschema.Schema {
	schema := operationSchema[mouseInput]([]string{string(MouseMoveAbsolute), string(MouseMoveRelative), string(MouseClick), string(MouseScroll)})
	setIntegerRange(schema.Properties["x"], 0, 32767)
	setIntegerRange(schema.Properties["y"], 0, 32767)
	for _, name := range []string{"dx", "dy", "wheel_x", "wheel_y"} {
		setIntegerRange(schema.Properties[name], -128, 127)
	}
	setStringEnum(schema.Properties["button"], []string{"left", "middle", "right"})
	schema.AnyOf = []*jsonschema.Schema{
		operationCase(string(MouseMoveAbsolute), []string{"x", "y"}, []string{"dx", "dy", "button", "wheel_x", "wheel_y"}),
		operationCase(string(MouseMoveRelative), []string{"dx", "dy"}, []string{"x", "y", "button", "wheel_x", "wheel_y"}),
		operationCase(string(MouseClick), []string{"button"}, []string{"x", "y", "dx", "dy", "wheel_x", "wheel_y"}),
		operationCase(string(MouseScroll), nil, []string{"x", "y", "dx", "dy", "button"}),
	}
	schema.AnyOf[3].AnyOf = []*jsonschema.Schema{
		requiredNonZeroProperty("wheel_x"),
		requiredNonZeroProperty("wheel_y"),
	}
	return schema
}

func requiredNonZeroProperty(name string) *jsonschema.Schema {
	zero := any(0)
	return &jsonschema.Schema{
		Properties: map[string]*jsonschema.Schema{name: {Not: &jsonschema.Schema{Const: &zero}}},
		Required:   []string{name},
	}
}

func keyboardSchema() *jsonschema.Schema {
	schema := operationSchema[keyboardInput]([]string{string(KeyboardTypeText), string(KeyboardPressKey)})
	minimum, maximum := 1, 4096
	schema.Properties["text"].MinLength = &minimum
	schema.Properties["text"].MaxLength = &maximum
	schema.Properties["key"].MinLength = &minimum
	setStringEnum(schema.Properties["modifiers"].Items, []string{"ctrl", "alt", "shift", "meta"})
	schema.AnyOf = []*jsonschema.Schema{
		operationCase(string(KeyboardTypeText), []string{"text"}, []string{"key", "modifiers"}),
		operationCase(string(KeyboardPressKey), []string{"key"}, []string{"text"}),
	}
	return schema
}

func operationCase(operation string, required, forbidden []string) *jsonschema.Schema {
	operationValue := any(operation)
	branch := &jsonschema.Schema{
		Properties: map[string]*jsonschema.Schema{
			"operation": {Const: &operationValue},
		},
		Required: append([]string{"device", "operation"}, required...),
	}
	if len(forbidden) == 0 {
		return branch
	}
	branch.Not = &jsonschema.Schema{AnyOf: make([]*jsonschema.Schema, 0, len(forbidden))}
	for _, property := range forbidden {
		branch.Not.AnyOf = append(branch.Not.AnyOf, &jsonschema.Schema{Required: []string{property}})
	}
	return branch
}

func operationSchema[T any](values []string) *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("tool input schema: %v", err))
	}
	setStringEnum(schema.Properties["operation"], values)
	return schema
}

func setStringEnum(property *jsonschema.Schema, values []string) {
	property.Type = "string"
	property.Types = nil
	property.Enum = make([]any, len(values))
	for index, value := range values {
		property.Enum[index] = value
	}
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
