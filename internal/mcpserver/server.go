package mcpserver

import (
	"context"
	"errors"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
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
	Device          string   `json:"device"`
	Connected       bool     `json:"connected"`
	Application     string   `json:"applicationVersion,omitempty"`
	System          string   `json:"systemVersion,omitempty"`
	Extension       string   `json:"activeExtension,omitempty"`
	ATXPowerOn      *bool    `json:"atxPowerOn,omitempty"`
	DCPowerOn       *bool    `json:"dcPowerOn,omitempty"`
	DCVoltage       float64  `json:"dcVoltage,omitempty"`
	VideoReady      *bool    `json:"videoReady,omitempty"`
	VideoWidth      int      `json:"videoWidth,omitempty"`
	VideoHeight     int      `json:"videoHeight,omitempty"`
	VideoFPS        int      `json:"videoFPS,omitempty"`
	VirtualMedia    string   `json:"virtualMedia,omitempty"`
	USBState        string   `json:"usbState,omitempty"`
	USBWakeAttached *bool    `json:"usbWakeAttached,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

// PowerResult describes the submitted host-power operation without exposing
// firmware-private RPC details.
type PowerResult struct {
	Device string      `json:"device"`
	Action PowerAction `json:"action"`
	Target string      `json:"target,omitempty"`
	Status string      `json:"status"`
}

// Device is the device-layer boundary used by MCP handlers.
type Device interface {
	Status(ctx context.Context, device string) (Status, error)
	Power(ctx context.Context, device string, action PowerAction, target string) (PowerResult, error)
	CaptureScreen(ctx context.Context, device string, request CaptureRequest) (CaptureResult, error)
	Keyboard(ctx context.Context, device string, request KeyboardRequest) (KeyboardResult, error)
	Mouse(ctx context.Context, device string, request MouseRequest) (MouseResult, error)
	VirtualMedia(ctx context.Context, device string, request VirtualMediaRequest) (VirtualMediaResult, error)
}

type deviceInput struct {
	Device string `json:"device" jsonschema:"JetKVM device name from the server configuration"`
}

type wakeLANInput struct {
	Device string `json:"device" jsonschema:"JetKVM device name from the server configuration"`
	Target string `json:"target" jsonschema:"Configured Wake-on-LAN target name"`
}

// New builds a JetKVM MCP server using only the official Go SDK.
func New(device Device, version string) *mcp.Server {
	// The manifest is static, so do not advertise tool-list change notifications.
	server := mcp.NewServer(&mcp.Implementation{Name: "jetkvm-mcp", Version: version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
	})

	addReadTool(server, &mcp.Tool{
		Name:        GetStatusToolName,
		Description: "Read the current status of a configured JetKVM and its attached host.",
		Annotations: annotations(true, false, true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input deviceInput) (*mcp.CallToolResult, Status, error) {
		if err := validDevice(input.Device); err != nil {
			return nil, Status{}, invalidInput(err)
		}
		status, err := device.Status(ctx, input.Device)
		return nil, status, err
	})

	registerPowerTool(server, device, PressHostPowerButtonToolName,
		"Briefly press the attached host's ATX power button.", PowerActionPressHostPowerButton, true, false)
	registerPowerTool(server, device, ForceHostPowerOffToolName,
		"Hold the attached host's ATX power button to force it off.", PowerActionForceHostPowerOff, true, false)
	registerPowerTool(server, device, PressHostResetButtonToolName,
		"Briefly press the attached host's reset button.", PowerActionPressHostResetButton, true, false)
	registerPowerTool(server, device, TurnHostDCPowerOnToolName,
		"Turn on the JetKVM-controlled DC output for the attached host.", PowerActionTurnHostDCPowerOn, false, true)
	registerPowerTool(server, device, TurnHostDCPowerOffToolName,
		"Turn off the JetKVM-controlled DC output for the attached host.", PowerActionTurnHostDCPowerOff, true, true)
	registerPowerTool(server, device, WakeHostUSBToolName,
		"Wake the attached host through JetKVM USB HID.", PowerActionWakeHostUSB, false, true)

	addMutationTool(server, &mcp.Tool{
		Name:        WakeHostLANToolName,
		Description: "Send Wake-on-LAN to a named target configured for this JetKVM.",
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

func registerPowerTool(server *mcp.Server, device Device, name, description string, action PowerAction, destructive, idempotent bool) {
	addMutationTool(server, &mcp.Tool{
		Name:        name,
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

func addConditionalMutationTool[In, Out any](server *mcp.Server, tool *mcp.Tool, mutation func(In) bool, handler mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(server, tool, withToolFailure(mutation, handler))
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

func validDevice(device string) error {
	if strings.TrimSpace(device) == "" {
		return errors.New("device is required")
	}
	return nil
}
