package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CaptureScreenTimeout bounds the complete MCP screenshot operation.
const CaptureScreenTimeout = 30 * time.Second

var errCaptureScreenDeadline = errors.New("capture screen deadline exceeded")

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
	Device     string    `json:"device" jsonschema:"configured device identifier"`
	CapturedAt time.Time `json:"capturedAt" jsonschema:"time the private PNG was captured"`
	MIMEType   string    `json:"mimeType" jsonschema:"private PNG image MIME type"`
	Width      int       `json:"width" jsonschema:"private PNG width in pixels"`
	Height     int       `json:"height" jsonschema:"private PNG height in pixels"`
	SizeBytes  int       `json:"sizeBytes" jsonschema:"private PNG byte count"`
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
	Device    string            `json:"device" jsonschema:"configured device identifier"`
	Operation KeyboardOperation `json:"operation" jsonschema:"submitted HID keyboard operation, not typed text"`
	Status    string            `json:"status" jsonschema:"completed means the HID RPC returned, not independent host-state proof"`
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
	Device    string         `json:"device" jsonschema:"configured device identifier"`
	Operation MouseOperation `json:"operation" jsonschema:"submitted HID mouse operation"`
	Status    string         `json:"status" jsonschema:"completed means the HID RPC returned, not independent host-state proof"`
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
	Mounted    bool                   `json:"mounted" jsonschema:"whether firmware reports a mounted medium"`
	SourceType VirtualMediaSourceType `json:"sourceType,omitempty" jsonschema:"redacted media source class; never a URL, path, or filename"`
	Mode       string                 `json:"mode,omitempty" jsonschema:"reported read_only or read_write mode when available"`
}

type VirtualMediaResult struct {
	Device     string                 `json:"device" jsonschema:"configured device identifier"`
	Operation  VirtualMediaOperation  `json:"operation" jsonschema:"submitted or observed virtual-media operation"`
	Mounted    bool                   `json:"mounted" jsonschema:"reported mount state"`
	SourceType VirtualMediaSourceType `json:"sourceType,omitempty" jsonschema:"redacted media source class; never a URL, path, or filename"`
	Mode       string                 `json:"mode,omitempty" jsonschema:"reported read_only or read_write mode when available"`
	Status     string                 `json:"status" jsonschema:"observed for status, completed for an acknowledged mutation; neither independently proves final device state"`
}

type captureInput struct {
	Device    string `json:"device" jsonschema:"configured JetKVM device name whose host display will be captured"`
	MaxWidth  int    `json:"max_width,omitempty" jsonschema:"maximum private PNG width from 1 through 3840; omit to use the configured default"`
	MaxHeight int    `json:"max_height,omitempty" jsonschema:"maximum private PNG height from 1 through 2160; omit to use the configured default"`
}

type keyboardInput struct {
	Device    string            `json:"device" jsonschema:"configured JetKVM device name whose attached host receives HID input"`
	Operation KeyboardOperation `json:"operation" jsonschema:"HID operation: type_text or press_key"`
	Text      string            `json:"text,omitempty" jsonschema:"private US-ASCII text for type_text, at most 4096 bytes; it is sent transiently and not logged"`
	Key       string            `json:"key,omitempty" jsonschema:"named key or one printable character for press_key; this host-control intent is private operational data"`
	Modifiers []string          `json:"modifiers,omitempty" jsonschema:"optional ctrl, alt, shift, meta modifiers for press_key; this host-control intent is private operational data"`
}

type mouseInput struct {
	Device    string         `json:"device" jsonschema:"configured JetKVM device name whose attached host receives HID input"`
	Operation MouseOperation `json:"operation" jsonschema:"HID operation: move_absolute, move_relative, click, or scroll"`
	X         *int           `json:"x,omitempty" jsonschema:"absolute x coordinate from 0 through 32767 for move_absolute"`
	Y         *int           `json:"y,omitempty" jsonschema:"absolute y coordinate from 0 through 32767 for move_absolute"`
	DX        *int           `json:"dx,omitempty" jsonschema:"relative horizontal movement for move_relative"`
	DY        *int           `json:"dy,omitempty" jsonschema:"relative vertical movement for move_relative"`
	Button    string         `json:"button,omitempty" jsonschema:"left, middle, or right button for click; can activate host UI actions"`
	WheelX    int            `json:"wheel_x,omitempty" jsonschema:"horizontal wheel movement for scroll"`
	WheelY    int            `json:"wheel_y,omitempty" jsonschema:"vertical wheel movement for scroll"`
}

type virtualMediaInput struct {
	Device    string                `json:"device" jsonschema:"configured JetKVM device name"`
	Operation VirtualMediaOperation `json:"operation" jsonschema:"deprecated compatibility operation: status, mount_url, mount_file, unmount, or upload"`
	Source    string                `json:"source,omitempty" jsonschema:"private URL or configured-media-directory relative path required by the selected operation; never returned in media state"`
	Mode      string                `json:"mode,omitempty" jsonschema:"read_only or read_write mount mode; defaults to read_only"`
}

type virtualMediaStatusInput struct {
	Device string `json:"device" jsonschema:"configured JetKVM device name"`
}

type virtualMediaURLInput struct {
	Device string `json:"device" jsonschema:"configured JetKVM device name with an allowed media URL origin"`
	URL    string `json:"url" jsonschema:"private HTTP(S) media URL; the appliance fetches it only when its origin matches configured exact scheme, host, and effective port"`
	Mode   string `json:"mode,omitempty" jsonschema:"read_only or read_write mount mode; defaults to read_only"`
}

type virtualMediaFileInput struct {
	Device string `json:"device" jsonschema:"configured JetKVM device name with a media directory"`
	Path   string `json:"path" jsonschema:"private relative local-media path confined below the device's configured media directory"`
	Mode   string `json:"mode,omitempty" jsonschema:"read_only or read_write mount mode; defaults to read_only"`
}

type virtualMediaUploadInput struct {
	Device string `json:"device" jsonschema:"configured JetKVM device name with a media directory"`
	Path   string `json:"path" jsonschema:"private relative local-media path confined below the device's configured media directory"`
}

func addControlTools(server *mcp.Server, device Device) {
	addReadTool(server, &mcp.Tool{
		Name:        CaptureScreenToolName,
		Title:       "Capture host screen",
		Description: "Capture one fresh private PNG from the host display attached to a configured JetKVM. The result can contain any visible host secret and is returned only to the MCP caller, not written to disk. This read has no unknown mutation outcome; follow a failure's retryable flag before retrying.",
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
		imageData := append([]byte(nil), capture.PNG...)
		if err := ctx.Err(); err != nil {
			return nil, CaptureOutput{}, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.ImageContent{MIMEType: capture.MIMEType, Data: imageData}}}, output, nil
	})

	addMutationTool(server, &mcp.Tool{
		Name:        KeyboardToolName,
		Title:       "Send keyboard input",
		Description: "Send private bounded US-ASCII text or one named key through USB HID to a configured attached host; it can enter credentials, execute commands, or alter host data. Input is transient and not logged. If a mutation reports outcome unknown, do not blindly retry; inspect host state first.",
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
		Title:       "Send mouse input",
		Description: "Move, click, or scroll a configured attached host's pointer through USB HID; clicks can activate destructive host UI actions. If a mutation reports outcome unknown, do not blindly retry; inspect host state first.",
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
		Title:        "Get virtual-media status",
		Description:  "Read the current virtual-media mount state of a configured JetKVM without changing it. The result is a redacted source class and mode, never a URL, path, filename, or raw firmware fields. This read has no unknown mutation outcome; follow a failure's retryable flag before retrying.",
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
		Title:        "Mount virtual media from URL",
		Description:  "Ask the configured JetKVM appliance to fetch a private HTTP(S) URL whose scheme, host, and effective port match a configured exact origin, then replace its current virtual-media mount. URL mounting is unavailable without an allowed origin. If a mutation reports outcome unknown, do not blindly retry; inspect status first.",
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
		Title:        "Mount virtual media from file",
		Description:  "Upload one private confined local media file and replace the configured JetKVM's current mount. Requires a configured media directory and a non-empty relative path beneath it. If a mutation reports outcome unknown, do not blindly retry; inspect status first.",
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
		Title:        "Unmount virtual media",
		Description:  "Unmount a configured JetKVM's current virtual media. The request is valid even when no media is mounted and is intended to converge. If a mutation reports outcome unknown, do not blindly retry; inspect status first.",
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
		Title:        "Upload virtual-media file",
		Description:  "Upload one private confined local media file to appliance storage on a configured JetKVM without mounting it. Requires a configured media directory and a non-empty relative path beneath it; appliance storage retention is outside this process. If a mutation reports outcome unknown, do not blindly retry; inspect status first.",
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
		Title:        "Virtual media (deprecated)",
		Description:  "Deprecated compatibility tool for configured virtual-media status, mount, unmount, and upload operations; use the one-purpose jetkvm_*_virtual_media* tools instead. Status is read-only, while mount and upload can alter appliance storage or network state. If a mutation reports outcome unknown, do not blindly retry; inspect status first.",
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

type captureDeadlineResult struct {
	mcp.ResultBase
	ctx    context.Context
	cancel context.CancelFunc
	result *mcp.CallToolResult
}

func (result *captureDeadlineResult) SetMeta(meta map[string]any) {
	result.ResultBase.SetMeta(meta)
	result.result.SetMeta(meta)
}

func (result *captureDeadlineResult) MarshalJSON() ([]byte, error) {
	if result.ctx.Err() != nil {
		return result.marshalContextFailure()
	}
	encoded, err := json.Marshal(result.result)
	if err != nil {
		result.cancel()
		return nil, err
	}
	if result.ctx.Err() == nil {
		result.cancel()
		return encoded, nil
	}

	return result.marshalContextFailure()
}

func (result *captureDeadlineResult) marshalContextFailure() ([]byte, error) {
	failure := new(mcp.CallToolResult)
	if errors.Is(context.Cause(result.ctx), errCaptureScreenDeadline) {
		failure.SetError(toolFailure(context.DeadlineExceeded, false))
	} else {
		failure.SetError(toolFailure(result.ctx.Err(), false))
	}
	result.cancel()
	return json.Marshal(failure)
}

func captureDeadlineMiddleware(timeout time.Duration) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
			params, ok := request.GetParams().(*mcp.CallToolParamsRaw)
			if method != "tools/call" || !ok || params.Name != CaptureScreenToolName {
				return next(ctx, method, request)
			}
			captureCtx, cancel := context.WithTimeoutCause(ctx, timeout, errCaptureScreenDeadline)
			result, err := next(captureCtx, method, request)
			if err != nil || result == nil {
				cancel()
				return result, err
			}
			callResult, ok := result.(*mcp.CallToolResult)
			if !ok || (captureCtx.Err() != nil && !errors.Is(context.Cause(captureCtx), errCaptureScreenDeadline)) {
				cancel()
				return result, nil
			}
			return &captureDeadlineResult{ctx: captureCtx, cancel: cancel, result: callResult}, nil
		}
	}
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
	// The HID text path supports tab, line breaks, and printable US-ASCII, so
	// code-point and byte limits are equivalent for every admitted value.
	schema.Properties["text"].Pattern = `^[	\n\r\x20-\x7E]+$`
	schema.Properties["key"].MinLength = &minimum
	// Every supported printable or named key contains a non-space ASCII byte.
	// This rejects both ASCII and Unicode whitespace-only values while retaining
	// handler-side trimming as defense in depth.
	schema.Properties["key"].Pattern = `[!-~]`
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
	schema.Type = "integer"
	schema.Types = nil
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
