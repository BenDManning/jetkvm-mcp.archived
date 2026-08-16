package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type telemetryDevice struct {
	recordingDevice
	statusErr       error
	powerErr        error
	virtualMediaErr error
}

func (device *telemetryDevice) Status(context.Context, string) (Status, error) {
	return Status{}, device.statusErr
}

func (device *telemetryDevice) Power(context.Context, string, PowerAction, string) (PowerResult, error) {
	return PowerResult{}, device.powerErr
}

func (device *telemetryDevice) VirtualMedia(context.Context, string, VirtualMediaRequest) (VirtualMediaResult, error) {
	return VirtualMediaResult{}, device.virtualMediaErr
}

type telemetryClassifiedError struct {
	error
	code      string
	outcome   string
	retryable bool
}

func (err telemetryClassifiedError) ToolErrorCode() string    { return err.code }
func (err telemetryClassifiedError) ToolErrorOutcome() string { return err.outcome }
func (err telemetryClassifiedError) ToolErrorRetryable() bool { return err.retryable }

type privacyTelemetryDevice struct {
	recordingDevice
}

func (*privacyTelemetryDevice) Status(context.Context, string) (Status, error) {
	return Status{Device: "PRIVATE-device", Application: "PRIVATE-firmware", Warnings: []string{"PRIVATE-raw-config-token"}}, nil
}

func (*privacyTelemetryDevice) CaptureScreen(context.Context, string, CaptureRequest) (CaptureResult, error) {
	return CaptureResult{Device: "PRIVATE-device", CapturedAt: time.Unix(1, 0), MIMEType: "image/png", Width: 1, Height: 1, PNG: []byte("PRIVATE-image-bytes")}, nil
}

func TestToolTelemetryCorrelatesSuccessFailureTimeoutCancelAndBusy(t *testing.T) {
	const privateSentinel = "PRIVATE-device-url-token-path-firmware-rpc-child-output"
	tests := []struct {
		name      string
		tool      string
		arguments map[string]any
		failure   error
		operation string
		code      string
		outcome   string
	}{
		{name: "success", tool: ListDevicesToolName, operation: telemetry.OperationInventory, code: telemetry.CodeSuccess, outcome: telemetry.OutcomeSucceeded},
		{name: "timeout", tool: GetStatusToolName, arguments: map[string]any{"device": privateSentinel}, failure: context.DeadlineExceeded, operation: telemetry.OperationStatus, code: "timeout", outcome: telemetry.OutcomeFailed},
		{name: "canceled", tool: GetStatusToolName, arguments: map[string]any{"device": privateSentinel}, failure: context.Canceled, operation: telemetry.OperationStatus, code: "canceled", outcome: telemetry.OutcomeFailed},
		{name: "busy", tool: GetStatusToolName, arguments: map[string]any{"device": privateSentinel}, failure: telemetryClassifiedError{error: errors.New(privateSentinel), code: "busy", outcome: "not_sent"}, operation: telemetry.OperationStatus, code: "busy", outcome: telemetry.OutcomeNotSent},
		{name: "mutation unknown", tool: PressHostPowerButtonToolName, arguments: map[string]any{"device": privateSentinel}, failure: telemetryClassifiedError{error: errors.New(privateSentinel), code: "protocol_error", outcome: telemetry.OutcomeUnknown}, operation: telemetry.OperationPower, code: "protocol_error", outcome: telemetry.OutcomeUnknown},
		{name: "media read failure", tool: GetVirtualMediaStatusToolName, arguments: map[string]any{"device": privateSentinel}, failure: errors.New(privateSentinel), operation: telemetry.OperationStatus, code: "operation_failed", outcome: telemetry.OutcomeFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			recorder := telemetry.New(&stderr)
			device := &telemetryDevice{}
			if test.tool == PressHostPowerButtonToolName {
				device.powerErr = test.failure
			} else if test.tool == GetVirtualMediaStatusToolName {
				device.virtualMediaErr = test.failure
			} else {
				device.statusErr = test.failure
			}
			serverTransport, clientTransport := mcp.NewInMemoryTransports()
			serverSession, err := NewWithTelemetry(device, "test", recorder, telemetry.TransportStdio).Connect(context.Background(), serverTransport, nil)
			if err != nil {
				t.Fatal(err)
			}
			clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "telemetry-test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
			if err != nil {
				t.Fatal(err)
			}

			result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: test.tool, Arguments: test.arguments})
			if err != nil {
				t.Fatal(err)
			}
			if test.failure == nil && (result == nil || result.IsError) {
				t.Fatalf("successful result = %#v", result)
			}
			if test.failure != nil && (result == nil || !result.IsError) {
				t.Fatalf("failed result = %#v", result)
			}
			_ = clientSession.Close()
			_ = serverSession.Close()
			if err := recorder.Close(context.Background()); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(stderr.String(), privateSentinel) {
				t.Fatalf("telemetry retained private sentinel: %s", stderr.String())
			}
			var event struct {
				Schema        string `json:"schema"`
				CorrelationID string `json:"correlation_id"`
				Transport     string `json:"transport"`
				Operation     string `json:"operation"`
				Stage         string `json:"stage"`
				DurationMS    int64  `json:"duration_ms"`
				Code          string `json:"code"`
				Outcome       string `json:"outcome"`
			}
			decoder := json.NewDecoder(&stderr)
			if err := decoder.Decode(&event); err != nil {
				t.Fatal(err)
			}
			if event.Schema != "jetkvm.operation.v1" || event.CorrelationID == "" || event.Transport != telemetry.TransportStdio || event.Operation != test.operation || event.Stage != telemetry.StageTool || event.DurationMS < 0 || event.Code != test.code || event.Outcome != test.outcome {
				t.Fatalf("event = %#v", event)
			}
			if err := decoder.Decode(new(any)); err == nil {
				t.Fatal("telemetry emitted more than one tool event")
			}
		})
	}
}

func TestHTTPToolTelemetryUsesHTTPTransport(t *testing.T) {
	var stderr bytes.Buffer
	recorder := telemetry.New(&stderr)
	httpServer := httptest.NewServer(NewHTTPHandler(NewWithTelemetry(&telemetryDevice{}, "test", recorder, telemetry.TransportHTTP), ""))
	defer httpServer.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "http-telemetry-test", Version: "test"}, nil).Connect(
		context.Background(),
		&mcp.StreamableClientTransport{Endpoint: httpServer.URL + MCPPath},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: ListDevicesToolName})
	if err != nil || result == nil || result.IsError {
		t.Fatalf("CallTool = %#v, %v", result, err)
	}
	_ = clientSession.Close()
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	var event struct {
		Transport string `json:"transport"`
		Operation string `json:"operation"`
		Stage     string `json:"stage"`
	}
	if err := json.NewDecoder(&stderr).Decode(&event); err != nil {
		t.Fatal(err)
	}
	if event.Transport != telemetry.TransportHTTP || event.Operation != telemetry.OperationInventory || event.Stage != telemetry.StageTool {
		t.Fatalf("event = %#v", event)
	}
}

func TestToolTelemetryUsesFinalOutputSchemaFailure(t *testing.T) {
	var stderr bytes.Buffer
	recorder := telemetry.New(&stderr)
	outputSchema, err := jsonschema.For[PowerResult](nil)
	if err != nil {
		t.Fatal(err)
	}
	setStringEnum(outputSchema.Properties["status"], []string{"required-fixture-status"})
	inputSchema, err := jsonschema.For[deviceInput](nil)
	if err != nil {
		t.Fatal(err)
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "telemetry-schema-server", Version: "test"}, nil)
	addMutationTool(server, &mcp.Tool{
		Name: PressHostPowerButtonToolName, InputSchema: inputSchema, OutputSchema: outputSchema,
	}, func(context.Context, *mcp.CallToolRequest, deviceInput) (*mcp.CallToolResult, PowerResult, error) {
		return nil, PowerResult{}, nil
	})
	server.AddReceivingMiddleware(toolTelemetryMiddleware(recorder, telemetry.TransportStdio))
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "telemetry-schema-test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: PressHostPowerButtonToolName, Arguments: map[string]any{"device": "fixture"},
	})
	_ = clientSession.Close()
	_ = serverSession.Close()
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	var got struct {
		Stage   string `json:"stage"`
		Code    string `json:"code"`
		Outcome string `json:"outcome"`
	}
	if err := json.NewDecoder(&stderr).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Stage != telemetry.StageTool || got.Code != "operation_failed" || got.Outcome != telemetry.OutcomeUnknown {
		t.Fatalf("terminal event = %#v, want operation_failed/unknown", got)
	}
}

func TestToolTelemetryProhibitsSensitiveInputsOutputsAndErrors(t *testing.T) {
	const sentinel = "PRIVATE-SENTINEL-typed-image-url-path-token-config-firmware-rpc-child"
	var stderr bytes.Buffer
	recorder := telemetry.New(&stderr)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := NewWithTelemetry(&privacyTelemetryDevice{}, "test", recorder, telemetry.TransportStdio).Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "privacy-test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range []mcp.CallToolParams{
		{Name: GetStatusToolName, Arguments: map[string]any{"device": sentinel}},
		{Name: CaptureScreenToolName, Arguments: map[string]any{"device": sentinel}},
		{Name: KeyboardToolName, Arguments: map[string]any{"device": sentinel, "operation": "type_text", "text": sentinel}},
		{Name: MountVirtualMediaURLToolName, Arguments: map[string]any{"device": sentinel, "url": "https://private.invalid/" + sentinel + ".iso?token=" + sentinel}},
	} {
		_, _ = clientSession.CallTool(context.Background(), &call)
	}
	_ = clientSession.Close()
	_ = serverSession.Close()
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, prohibited := range []string{sentinel, "PRIVATE-device", "PRIVATE-firmware", "PRIVATE-raw-config-token", "PRIVATE-image-bytes", "https://private.invalid"} {
		if strings.Contains(stderr.String(), prohibited) {
			t.Fatalf("telemetry retained %q: %s", prohibited, stderr.String())
		}
	}
}
