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

const (
	ListDevicesToolName          = "jetkvm_list_devices"
	GetStatusToolName            = "jetkvm_get_status"
	PressHostPowerButtonToolName = "jetkvm_press_host_power_button"
	ForceHostPowerOffToolName    = "jetkvm_force_host_power_off"
	PressHostResetButtonToolName = "jetkvm_press_host_reset_button"
	TurnHostDCPowerOnToolName    = "jetkvm_turn_host_dc_power_on"
	TurnHostDCPowerOffToolName   = "jetkvm_turn_host_dc_power_off"
	WakeHostUSBToolName          = "jetkvm_wake_host_usb"
	WakeHostLANToolName          = "jetkvm_wake_host_lan"
)

// PowerAction identifies one concrete JetKVM host-power operation.
type PowerAction string

const (
	PowerActionPressHostPowerButton PowerAction = "press_host_power_button"
	PowerActionForceHostPowerOff    PowerAction = "force_host_power_off"
	PowerActionPressHostResetButton PowerAction = "press_host_reset_button"
	PowerActionTurnHostDCPowerOn    PowerAction = "turn_host_dc_power_on"
	PowerActionTurnHostDCPowerOff   PowerAction = "turn_host_dc_power_off"
	PowerActionWakeHostUSB          PowerAction = "wake_host_usb"
	PowerActionWakeHostLAN          PowerAction = "wake_host_lan"
)

// Status is the device state returned by the status tool. Optional fields remain
// absent when a particular firmware does not report them.
type Status struct {
	Device          string             `json:"device" jsonschema:"configured device identifier"`
	Connected       bool               `json:"connected" jsonschema:"whether the appliance status read connected"`
	Application     string             `json:"applicationVersion,omitempty" jsonschema:"private appliance application version when reported"`
	System          string             `json:"systemVersion,omitempty" jsonschema:"private appliance system version when reported"`
	Extension       string             `json:"activeExtension,omitempty" jsonschema:"private active appliance extension when reported"`
	ATXPowerOn      *bool              `json:"atxPowerOn,omitempty" jsonschema:"private observed ATX host power state when reported"`
	DCPowerOn       *bool              `json:"dcPowerOn,omitempty" jsonschema:"private observed DC output state when reported"`
	DCVoltage       float64            `json:"dcVoltage,omitempty" jsonschema:"private observed DC voltage when reported"`
	VideoReady      *bool              `json:"videoReady,omitempty" jsonschema:"private observed host video availability when reported"`
	VideoWidth      int                `json:"videoWidth,omitempty" jsonschema:"private observed host video width when reported"`
	VideoHeight     int                `json:"videoHeight,omitempty" jsonschema:"private observed host video height when reported"`
	VideoFPS        int                `json:"videoFPS,omitempty" jsonschema:"private observed host video frame rate when reported"`
	VirtualMedia    *VirtualMediaState `json:"virtualMedia,omitempty" jsonschema:"redacted observed virtual-media state when reported"`
	USBState        string             `json:"usbState,omitempty" jsonschema:"private observed USB state when reported"`
	USBWakeAttached *bool              `json:"usbWakeAttached,omitempty" jsonschema:"private observed USB wake attachment state when reported"`
	Warnings        []string           `json:"warnings,omitempty" jsonschema:"private appliance warnings when reported"`
}

// PowerResult describes the submitted host-power operation without exposing
// firmware-private RPC details.
type PowerResult struct {
	Device string      `json:"device" jsonschema:"configured device identifier"`
	Action PowerAction `json:"action" jsonschema:"submitted power or wake action"`
	Target string      `json:"target,omitempty" jsonschema:"configured Wake-on-LAN target identifier when applicable"`
	Status string      `json:"status" jsonschema:"completed means the appliance RPC returned, not independent physical-state proof"`
}

type DeviceCapabilities struct {
	MountVirtualMediaURL   bool `json:"mountVirtualMediaURL" jsonschema:"whether required URL-mount configuration exists"`
	MountVirtualMediaFile  bool `json:"mountVirtualMediaFile" jsonschema:"whether a configured local media directory exists"`
	UploadVirtualMediaFile bool `json:"uploadVirtualMediaFile" jsonschema:"whether a configured local media directory exists"`
	WakeHostLAN            bool `json:"wakeHostLAN" jsonschema:"whether configured Wake-on-LAN targets exist"`
}

type ConfiguredDevice struct {
	Device       string             `json:"device" jsonschema:"configured device alias"`
	Capabilities DeviceCapabilities `json:"capabilities" jsonschema:"configuration-derived availability flags, not firmware qualification"`
}

type DeviceList struct {
	Devices []ConfiguredDevice `json:"devices" jsonschema:"configured device aliases in deterministic order"`
}

// Device is the device-layer boundary used by MCP handlers.
type Device interface {
	ListDevices(ctx context.Context) (DeviceList, error)
	Status(ctx context.Context, device string) (Status, error)
	Power(ctx context.Context, device string, action PowerAction, target string) (PowerResult, error)
	CaptureScreen(ctx context.Context, device string, request CaptureRequest) (CaptureResult, error)
	Keyboard(ctx context.Context, device string, request KeyboardRequest) (KeyboardResult, error)
	Mouse(ctx context.Context, device string, request MouseRequest) (MouseResult, error)
	VirtualMedia(ctx context.Context, device string, request VirtualMediaRequest) (VirtualMediaResult, error)
}

type deviceInput struct {
	Device string `json:"device" jsonschema:"configured JetKVM device name"`
}

type listDevicesInput struct{}

type wakeLANInput struct {
	Device string `json:"device" jsonschema:"configured JetKVM device name"`
	Target string `json:"target" jsonschema:"configured Wake-on-LAN target name; it does not accept arbitrary network destinations"`
}

// New builds a JetKVM MCP server using only the official Go SDK.
func New(device Device, version string) *mcp.Server {
	return newServer(device, version, CaptureScreenTimeout)
}

func newServer(device Device, version string, captureTimeout time.Duration) *mcp.Server {
	// The manifest is static, so do not advertise tool-list change notifications.
	server := mcp.NewServer(&mcp.Implementation{Name: "jetkvm-mcp", Version: version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
	})
	server.AddReceivingMiddleware(captureDeadlineMiddleware(captureTimeout))

	addReadTool(server, &mcp.Tool{
		Name:         ListDevicesToolName,
		Title:        "List configured JetKVM devices",
		Description:  "List configured JetKVM aliases and configuration-derived availability flags without contacting a device. Results contain private deployment identifiers but omit URLs, credentials, origins, media paths, and Wake-on-LAN details. This read has no unknown mutation outcome; follow a failure's retryable flag before retrying.",
		OutputSchema: deviceListOutputSchema(),
		Annotations:  annotations(true, false, true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ listDevicesInput) (*mcp.CallToolResult, DeviceList, error) {
		devices, err := device.ListDevices(ctx)
		return nil, devices, err
	})

	addReadTool(server, &mcp.Tool{
		Name:         GetStatusToolName,
		Title:        "Get JetKVM status",
		Description:  "Read private current appliance and attached-host status from a configured JetKVM without changing it. Status can expose power, video, USB, version, warning, and redacted media state. This read has no unknown mutation outcome; follow a failure's retryable flag before retrying.",
		OutputSchema: statusOutputSchema(),
		Annotations:  annotations(true, false, true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input deviceInput) (*mcp.CallToolResult, Status, error) {
		if err := validDevice(input.Device); err != nil {
			return nil, Status{}, invalidInput(err)
		}
		status, err := device.Status(ctx, input.Device)
		return nil, status, err
	})

	registerPowerTool(server, device, PressHostPowerButtonToolName,
		"Press host power button", "Briefly press the physical ATX power button of a configured attached host; host firmware or OS behavior may change power state. If a mutation reports outcome unknown, do not blindly retry; inspect status first.", PowerActionPressHostPowerButton, true, false)
	registerPowerTool(server, device, ForceHostPowerOffToolName,
		"Force host power off", "Hold the physical ATX power button of a configured attached host to force it off; this can interrupt work and cause data loss. If a mutation reports outcome unknown, do not blindly retry; inspect status first.", PowerActionForceHostPowerOff, true, false)
	registerPowerTool(server, device, PressHostResetButtonToolName,
		"Press host reset button", "Briefly press the physical reset button of a configured attached host; this can interrupt work or corrupt data. If a mutation reports outcome unknown, do not blindly retry; inspect status first.", PowerActionPressHostResetButton, true, false)
	registerPowerTool(server, device, TurnHostDCPowerOnToolName,
		"Turn host DC power on", "Enable the physical JetKVM-controlled DC output for a configured attached host, which may boot equipment. The request is intended to converge, but if its outcome is unknown do not blindly retry; inspect status first.", PowerActionTurnHostDCPowerOn, false, true)
	registerPowerTool(server, device, TurnHostDCPowerOffToolName,
		"Turn host DC power off", "Disable the physical JetKVM-controlled DC output for a configured attached host, which can interrupt work and cause data loss. The request is intended to converge, but if its outcome is unknown do not blindly retry; inspect status first.", PowerActionTurnHostDCPowerOff, true, true)
	registerPowerTool(server, device, WakeHostUSBToolName,
		"Wake host over USB", "Send a USB HID wake action to a configured attached host, which may resume or boot it. The request is intended to converge, but if its outcome is unknown do not blindly retry; inspect status first.", PowerActionWakeHostUSB, false, true)

	addMutationTool(server, &mcp.Tool{
		Name:        WakeHostLANToolName,
		Title:       "Wake host over LAN",
		Description: "Make the configured JetKVM send a Wake-on-LAN network magic packet to a named configured target; callers cannot supply an arbitrary MAC address. The request is intended to converge, but if its outcome is unknown do not blindly retry; inspect status first.",
		Annotations: annotations(false, false, true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input wakeLANInput) (*mcp.CallToolResult, PowerResult, error) {
		if err := validDevice(input.Device); err != nil {
			return nil, PowerResult{}, invalidInput(err)
		}
		if strings.TrimSpace(input.Target) == "" {
			return nil, PowerResult{}, invalidInput(errors.New("target is required"))
		}
		result, err := device.Power(ctx, input.Device, PowerActionWakeHostLAN, input.Target)
		return nil, result, err
	})

	addControlTools(server, device)

	return server
}

func registerPowerTool(server *mcp.Server, device Device, name, title, description string, action PowerAction, destructive, idempotent bool) {
	addMutationTool(server, &mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		Annotations: annotations(false, destructive, idempotent),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input deviceInput) (*mcp.CallToolResult, PowerResult, error) {
		if err := validDevice(input.Device); err != nil {
			return nil, PowerResult{}, invalidInput(err)
		}
		result, err := device.Power(ctx, input.Device, action, "")
		return nil, result, err
	})
}

func addReadTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(server, tool, withToolFailure(func(In) bool { return false }, handler))
}

func addMutationTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(server, tool, withToolFailure(func(In) bool { return true }, handler))
}

// addSanitizedInputMutationTool retains the public JSON Schema while handling
// its validation locally. The SDK's validation errors include rejected values,
// which is unsafe for media URLs and local paths that may contain private data.
func addSanitizedInputMutationTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	addSanitizedInputTool(server, tool, func(In) bool { return true }, handler)
}

func addSanitizedConditionalMutationTool[In, Out any](server *mcp.Server, tool *mcp.Tool, mutation func(In) bool, handler mcp.ToolHandlerFor[In, Out]) {
	addSanitizedInputTool(server, tool, mutation, handler)
}

func addSanitizedInputTool[In, Out any](server *mcp.Server, tool *mcp.Tool, mutation func(In) bool, handler mcp.ToolHandlerFor[In, Out]) {
	inputSchema, ok := tool.InputSchema.(*jsonschema.Schema)
	if !ok || inputSchema == nil {
		panic(fmt.Sprintf("tool %q requires a JSON Schema input", tool.Name))
	}
	inputResolved, err := inputSchema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		panic(fmt.Sprintf("tool %q input schema: %v", tool.Name, err))
	}
	outputSchema, ok := tool.OutputSchema.(*jsonschema.Schema)
	if !ok || outputSchema == nil {
		panic(fmt.Sprintf("tool %q requires a JSON Schema output", tool.Name))
	}
	outputResolved, err := outputSchema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		panic(fmt.Sprintf("tool %q output schema: %v", tool.Name, err))
	}
	wrapped := withToolFailure(mutation, handler)

	server.AddTool(tool, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input, err := decodeSanitizedToolInput[In](request.Params.Arguments, inputResolved)
		if err != nil {
			result := new(mcp.CallToolResult)
			result.SetError(toolFailure(invalidInput(errors.New("arguments do not match the tool schema")), true))
			return result, nil
		}
		result, output, err := wrapped(ctx, request, input)
		if err != nil {
			if result == nil {
				result = new(mcp.CallToolResult)
			}
			result.SetError(err)
			return result, nil
		}
		if result == nil {
			result = new(mcp.CallToolResult)
		}
		outputJSON, err := json.Marshal(output)
		if err != nil {
			return nil, fmt.Errorf("marshaling tool output: %w", err)
		}
		var outputValue any
		if err := json.Unmarshal(outputJSON, &outputValue); err != nil {
			return nil, fmt.Errorf("unmarshaling tool output: %w", err)
		}
		if err := outputResolved.Validate(outputValue); err != nil {
			return nil, fmt.Errorf("validating tool output: %w", err)
		}
		result.StructuredContent = json.RawMessage(outputJSON)
		if result.Content == nil {
			result.Content = []mcp.Content{&mcp.TextContent{Text: string(outputJSON)}}
		}
		return result, nil
	})
}

func decodeSanitizedToolInput[In any](arguments json.RawMessage, schema *jsonschema.Resolved) (In, error) {
	var input In
	if len(arguments) == 0 {
		arguments = json.RawMessage(`{}`)
	}
	value := make(map[string]any)
	if err := json.Unmarshal(arguments, &value); err != nil {
		return input, err
	}
	var instance any = value
	if err := schema.ApplyDefaults(&instance); err != nil {
		return input, err
	}
	if err := schema.Validate(instance); err != nil {
		return input, err
	}
	normalized, err := json.Marshal(instance)
	if err != nil {
		return input, err
	}
	if err := json.Unmarshal(normalized, &input); err != nil {
		return input, err
	}
	return input, nil
}

func withToolFailure[In, Out any](mutation func(In) bool, handler mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, request *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		result, output, err := handler(ctx, request, input)
		if err != nil {
			return result, output, toolFailure(err, mutation(input))
		}
		return result, output, nil
	}
}

func annotations(readOnly, destructive, idempotent bool) *mcp.ToolAnnotations {
	return annotationsWithOpenWorld(readOnly, destructive, idempotent, false)
}

func annotationsWithOpenWorld(readOnly, destructive, idempotent, openWorld bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    readOnly,
		DestructiveHint: &destructive,
		IdempotentHint:  idempotent,
		OpenWorldHint:   &openWorld,
	}
}

func deviceListOutputSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[DeviceList](nil)
	if err != nil {
		panic(fmt.Sprintf("device list output schema: %v", err))
	}
	return schema
}

func validDevice(device string) error {
	if strings.TrimSpace(device) == "" {
		return errors.New("device is required")
	}
	return nil
}
