package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/identifier"
	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
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

// ResultStatus is the fixed acknowledgement vocabulary used by successful
// operation results.
type ResultStatus string

const (
	ResultStatusCompleted ResultStatus = "completed"
	ResultStatusObserved  ResultStatus = "observed"
)

type StatusWarning string

const (
	StatusWarningVersionUnavailable         StatusWarning = "version unavailable"
	StatusWarningActiveExtensionUnavailable StatusWarning = "active extension unavailable"
	StatusWarningVirtualMediaUnavailable    StatusWarning = "virtual media unavailable"
	StatusWarningVideoUnavailable           StatusWarning = "video unavailable"
	StatusWarningUSBUnavailable             StatusWarning = "USB unavailable"
	StatusWarningATXUnavailable             StatusWarning = "ATX state unavailable"
	StatusWarningDCUnavailable              StatusWarning = "DC state unavailable"
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
	Warnings        []StatusWarning    `json:"warnings,omitempty" jsonschema:"private appliance warnings when reported"`
}

// PowerResult describes the submitted host-power operation without exposing
// firmware-private RPC details.
type PowerResult struct {
	Device string       `json:"device" jsonschema:"configured device identifier"`
	Action PowerAction  `json:"action" jsonschema:"submitted power or wake action"`
	Target string       `json:"target,omitempty" jsonschema:"configured Wake-on-LAN target identifier when applicable"`
	Status ResultStatus `json:"status" jsonschema:"completed means the appliance RPC returned, not independent physical-state proof"`
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
func New(device Device, version string) *Server {
	return newServerWithTelemetry(device, version, CaptureScreenTimeout, nil, "")
}

func newServer(device Device, version string, captureTimeout time.Duration) *Server {
	return newServerWithTelemetry(device, version, captureTimeout, nil, "")
}

// NewWithTelemetry builds a server whose operation events are written by recorder.
// Transport and operation values are selected from closed, privacy-safe enums.
func NewWithTelemetry(device Device, version string, recorder *telemetry.Recorder, transport string) *Server {
	return newServerWithTelemetry(device, version, CaptureScreenTimeout, recorder, transport)
}

func newServerWithTelemetry(device Device, version string, captureTimeout time.Duration, recorder *telemetry.Recorder, transport string) *Server {
	// The manifest is static, so do not advertise tool-list change notifications.
	server := mcp.NewServer(&mcp.Implementation{Name: "jetkvm-mcp", Version: version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
	})
	server.AddReceivingMiddleware(protocolVersionDiscoveryMiddleware)
	server.AddReceivingMiddleware(captureDeadlineMiddleware(captureTimeout))
	if recorder != nil {
		server.AddReceivingMiddleware(toolTelemetryMiddleware(recorder, transport))
	}

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
		"Turn host DC power on", "Enable the physical JetKVM-controlled DC output for a configured attached host, which may boot equipment. If the mutation's outcome is unknown, do not blindly retry; inspect status first.", PowerActionTurnHostDCPowerOn, false, false)
	registerPowerTool(server, device, TurnHostDCPowerOffToolName,
		"Turn host DC power off", "Disable the physical JetKVM-controlled DC output for a configured attached host, which can interrupt work and cause data loss. If the mutation's outcome is unknown, do not blindly retry; inspect status first.", PowerActionTurnHostDCPowerOff, true, false)
	registerPowerTool(server, device, WakeHostUSBToolName,
		"Wake host over USB", "Send a USB HID wake action to a configured attached host, which may resume or boot it. If the mutation's outcome is unknown, do not blindly retry; inspect status first.", PowerActionWakeHostUSB, false, false)

	addMutationTool(server, &mcp.Tool{
		Name:         WakeHostLANToolName,
		Title:        "Wake host over LAN",
		Description:  "Make the configured JetKVM send a Wake-on-LAN network magic packet to a named configured target; callers cannot supply an arbitrary MAC address. If the mutation's outcome is unknown, do not blindly retry; inspect status first.",
		OutputSchema: powerResultSchema(PowerActionWakeHostLAN),
		Annotations:  annotations(false, false, false),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input wakeLANInput) (*mcp.CallToolResult, PowerResult, error) {
		if err := validDevice(input.Device); err != nil {
			return nil, PowerResult{}, invalidInput(err)
		}
		if err := validIdentifier(input.Target, "target"); err != nil {
			return nil, PowerResult{}, invalidInput(err)
		}
		result, err := device.Power(ctx, input.Device, PowerActionWakeHostLAN, input.Target)
		return nil, result, err
	})

	addControlTools(server, device)

	return &Server{sdk: server}
}

func toolTelemetryMiddleware(recorder *telemetry.Recorder, transport string) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
			params, ok := request.GetParams().(*mcp.CallToolParamsRaw)
			if method != "tools/call" || !ok {
				return next(ctx, method, request)
			}
			operation, mutation, ok := telemetryToolClass(params.Name)
			if !ok {
				return next(ctx, method, request)
			}
			operationCtx, span := recorder.Start(ctx, transport, operation)
			state := &toolTelemetryState{span: span, mutation: mutation}
			operationCtx = context.WithValue(operationCtx, toolTelemetryKey{}, state)
			result, err := next(operationCtx, method, request)
			if err != nil {
				code, outcome := telemetryToolResult(nil, err, mutation)
				state.record(code, outcome)
			}
			return result, err
		}
	}
}

type toolTelemetryKey struct{}

type toolTelemetryState struct {
	span     *telemetry.Span
	mutation bool
	recorded atomic.Bool
}

func finishToolTelemetry(ctx context.Context, failure error, mutation bool) {
	state, _ := ctx.Value(toolTelemetryKey{}).(*toolTelemetryState)
	if state == nil {
		return
	}
	if failure == nil {
		state.record(telemetry.CodeSuccess, telemetry.OutcomeSucceeded)
		return
	}
	code, outcome := telemetryToolResult(nil, failure, mutation)
	state.record(code, outcome)
}

func finishToolTelemetryResult(ctx context.Context, result mcp.Result, failure error, mutation bool) {
	state, _ := ctx.Value(toolTelemetryKey{}).(*toolTelemetryState)
	if state == nil {
		return
	}
	code, outcome := telemetryToolResult(result, failure, mutation)
	state.record(code, outcome)
}

func (state *toolTelemetryState) record(code, outcome string) {
	if state != nil && state.recorded.CompareAndSwap(false, true) {
		state.span.Record(telemetry.StageTool, code, outcome)
	}
}

func telemetryToolClass(name string) (operation string, mutation, ok bool) {
	switch name {
	case ListDevicesToolName:
		return telemetry.OperationInventory, false, true
	case GetStatusToolName, GetVirtualMediaStatusToolName:
		return telemetry.OperationStatus, false, true
	case PressHostPowerButtonToolName, ForceHostPowerOffToolName, PressHostResetButtonToolName,
		TurnHostDCPowerOnToolName, TurnHostDCPowerOffToolName, WakeHostUSBToolName, WakeHostLANToolName:
		return telemetry.OperationPower, true, true
	case KeyboardToolName, MouseToolName:
		return telemetry.OperationHID, true, true
	case CaptureScreenToolName:
		return telemetry.OperationCapture, false, true
	case MountVirtualMediaURLToolName, MountVirtualMediaFileToolName, UnmountVirtualMediaToolName,
		UploadVirtualMediaFileToolName:
		return telemetry.OperationMedia, true, true
	default:
		return "", false, false
	}
}

func telemetryToolResult(result mcp.Result, err error, mutation bool) (string, string) {
	if err == nil {
		callResult, ok := result.(*mcp.CallToolResult)
		if ok && !callResult.IsError {
			return telemetry.CodeSuccess, telemetry.OutcomeSucceeded
		}
		if ok {
			err = callResult.GetError()
		}
	}
	if err == nil {
		if mutation {
			return string(toolErrorOperationFailed), telemetry.OutcomeUnknown
		}
		return string(toolErrorOperationFailed), telemetry.OutcomeFailed
	}
	var failure toolError
	if !errors.As(err, &failure) {
		failure = toolFailure(err, mutation).(toolError)
	}
	return string(failure.Code), string(failure.Outcome)
}

func registerPowerTool(server *mcp.Server, device Device, name, title, description string, action PowerAction, destructive, idempotent bool) {
	addMutationTool(server, &mcp.Tool{
		Name:         name,
		Title:        title,
		Description:  description,
		OutputSchema: powerResultSchema(action),
		Annotations:  annotations(false, destructive, idempotent),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input deviceInput) (*mcp.CallToolResult, PowerResult, error) {
		if err := validDevice(input.Device); err != nil {
			return nil, PowerResult{}, invalidInput(err)
		}
		result, err := device.Power(ctx, input.Device, action, "")
		return nil, result, err
	})
}

func addReadTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	addInputTool(server, tool, func(In) bool { return false }, handler)
}

func addMutationTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	addInputTool(server, tool, func(In) bool { return true }, handler)
}

func addInputTool[In, Out any](server *mcp.Server, tool *mcp.Tool, mutation func(In) bool, handler mcp.ToolHandlerFor[In, Out]) {
	inputSchema, ok := tool.InputSchema.(*jsonschema.Schema)
	if !ok || inputSchema == nil {
		var err error
		inputSchema, err = jsonschema.For[In](nil)
		if err != nil {
			panic(fmt.Sprintf("tool %q input schema: %v", tool.Name, err))
		}
		tool.InputSchema = inputSchema
	}
	boundIdentifierProperty(inputSchema, "device")
	boundIdentifierProperty(inputSchema, "target")
	inputResolved, err := inputSchema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		panic(fmt.Sprintf("tool %q input schema: %v", tool.Name, err))
	}
	outputSchema, ok := tool.OutputSchema.(*jsonschema.Schema)
	if !ok || outputSchema == nil {
		var err error
		outputSchema, err = jsonschema.For[Out](nil)
		if err != nil {
			panic(fmt.Sprintf("tool %q output schema: %v", tool.Name, err))
		}
		tool.OutputSchema = outputSchema
	}
	outputResolved, err := outputSchema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		panic(fmt.Sprintf("tool %q output schema: %v", tool.Name, err))
	}
	wrapped := withToolFailure(mutation, handler)

	server.AddTool(tool, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input, err := decodeSanitizedToolInput[In](request.Params.Arguments, inputResolved)
		if err != nil {
			var zero In
			mutated := mutation(zero)
			telemetryFailure := toolFailure(invalidInput(errors.New("arguments do not match the tool schema")), mutated)
			if tool.Name != CaptureScreenToolName {
				finishToolTelemetry(ctx, telemetryFailure, mutated)
			}
			result := new(mcp.CallToolResult)
			result.SetError(telemetryFailure)
			return result, nil
		}
		mutated := mutation(input)
		result, output, err := wrapped(ctx, request, input)
		if err != nil {
			if tool.Name != CaptureScreenToolName {
				finishToolTelemetry(ctx, err, mutated)
			}
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
			return sanitizedToolOutputFailure(ctx, mutated)
		}
		var outputValue any
		if err := json.Unmarshal(outputJSON, &outputValue); err != nil {
			return sanitizedToolOutputFailure(ctx, mutated)
		}
		if err := outputResolved.Validate(outputValue); err != nil {
			return sanitizedToolOutputFailure(ctx, mutated)
		}
		result.StructuredContent = json.RawMessage(outputJSON)
		if result.Content == nil {
			result.Content = []mcp.Content{&mcp.TextContent{Text: string(outputJSON)}}
		} else {
			result.Content = append(result.Content, &mcp.TextContent{Text: string(outputJSON)})
		}
		if tool.Name != CaptureScreenToolName {
			finishToolTelemetry(ctx, nil, mutated)
		}
		return result, nil
	})
}

func sanitizedToolOutputFailure(ctx context.Context, mutation bool) (*mcp.CallToolResult, error) {
	failure := toolFailure(errors.New("provider returned invalid tool output"), mutation)
	finishToolTelemetry(ctx, failure, mutation)
	result := new(mcp.CallToolResult)
	result.SetError(failure)
	return result, nil
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
	for _, name := range []string{"device", "target"} {
		if text, ok := value[name].(string); ok {
			normalized, _ := identifier.Normalize(text)
			value[name] = normalized
		}
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
		mutated := mutation(input)
		result, output, err := handler(ctx, request, input)
		if err != nil {
			return result, output, toolFailure(err, mutated)
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

func powerResultSchema(action PowerAction) *jsonschema.Schema {
	schema, err := jsonschema.For[PowerResult](nil)
	if err != nil {
		panic(fmt.Sprintf("power output schema: %v", err))
	}
	setStringEnum(schema.Properties["action"], []string{string(action)})
	setStringEnum(schema.Properties["status"], []string{string(ResultStatusCompleted)})
	return schema
}

func boundIdentifierProperty(schema *jsonschema.Schema, name string) {
	property := schema.Properties[name]
	if property == nil {
		return
	}
	minimum, maximum := 1, identifier.MaxCodePoints
	property.MinLength = &minimum
	property.MaxLength = &maximum
}

func validDevice(device string) error {
	return validIdentifier(device, "device")
}

func validIdentifier(value, name string) error {
	if _, ok := identifier.Normalize(value); !ok {
		return fmt.Errorf("%s must contain 1 through %d Unicode code points", name, identifier.MaxCodePoints)
	}
	return nil
}
