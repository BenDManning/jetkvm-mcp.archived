package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
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
	return Status{Device: "PRIVATE-device", Application: "PRIVATE-firmware", Warnings: []StatusWarning{StatusWarningVideoUnavailable}}, nil
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
			recorder := telemetry.New(&stderr, "test")
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
			if event.Schema != "jetkvm.operation.v2" || event.CorrelationID == "" || event.Transport != telemetry.TransportStdio || event.Operation != test.operation || event.Stage != telemetry.StageTool || event.DurationMS < 0 || event.Code != test.code || event.Outcome != test.outcome {
				t.Fatalf("event = %#v", event)
			}
			var summary struct {
				Operation string `json:"operation"`
				Stage     string `json:"stage"`
				Code      string `json:"code"`
			}
			if err := decoder.Decode(&summary); err != nil || summary.Operation != telemetry.OperationLifecycle || summary.Stage != telemetry.StageShutdown || summary.Code != "telemetry_summary" {
				t.Fatalf("shutdown summary = %#v, error = %v", summary, err)
			}
		})
	}
}

func TestHTTPToolTelemetryUsesHTTPTransport(t *testing.T) {
	const sentinel = "PRIVATE-HTTP-device-url-token-path-firmware-rpc-child-output"
	var stderr bytes.Buffer
	recorder := telemetry.New(&stderr, "test")
	httpServer := httptest.NewServer(NewHTTPHandler(NewWithTelemetry(&privacyTelemetryDevice{}, "test", recorder, telemetry.TransportHTTP), ""))
	defer httpServer.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "http-telemetry-test", Version: "test"}, nil).Connect(
		context.Background(),
		&mcp.StreamableClientTransport{Endpoint: httpServer.URL + MCPPath},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{Name: GetStatusToolName, Arguments: map[string]any{"device": sentinel}})
	if err != nil || result == nil || result.IsError {
		t.Fatalf("CallTool = %#v, %v", result, err)
	}
	for _, call := range []mcp.CallToolParams{
		{Name: CaptureScreenToolName, Arguments: map[string]any{"device": sentinel}},
		{Name: KeyboardToolName, Arguments: map[string]any{"device": sentinel, "operation": "type_text", "text": sentinel}},
		{Name: MountVirtualMediaURLToolName, Arguments: map[string]any{"device": sentinel, "url": "https://private.invalid/" + sentinel + ".iso?token=" + sentinel}},
	} {
		_, _ = clientSession.CallTool(context.Background(), &call)
	}
	_ = clientSession.Close()
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&stderr)
	processID := ""
	toolSeen, summarySeen := false, false
	for {
		var event struct {
			Schema            string `json:"schema"`
			Time              string `json:"time"`
			ProcessInstanceID string `json:"process_instance_id"`
			ServerVersion     string `json:"server_version"`
			CorrelationID     string `json:"correlation_id"`
			Transport         string `json:"transport"`
			Operation         string `json:"operation"`
			Stage             string `json:"stage"`
			Code              string `json:"code"`
			DroppedEvents     uint64 `json:"dropped_events"`
			WriterFailed      bool   `json:"writer_failed"`
		}
		if err := decoder.Decode(&event); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		parsedTime, err := time.Parse(time.RFC3339Nano, event.Time)
		if err != nil || !strings.HasSuffix(event.Time, "Z") || time.Since(parsedTime) > time.Minute || event.Schema != "jetkvm.operation.v2" || event.ServerVersion != "test" || !strings.HasPrefix(event.ProcessInstanceID, "proc_") || !strings.HasPrefix(event.CorrelationID, "op_") || event.Transport != telemetry.TransportHTTP {
			t.Fatalf("event = %#v, time error = %v", event, err)
		}
		if processID == "" {
			processID = event.ProcessInstanceID
		} else if event.ProcessInstanceID != processID {
			t.Fatalf("process identity changed from %q to %q", processID, event.ProcessInstanceID)
		}
		toolSeen = toolSeen || event.Operation == telemetry.OperationStatus && event.Stage == telemetry.StageTool
		if event.Code == "telemetry_summary" {
			summarySeen = true
			if event.DroppedEvents != 0 || event.WriterFailed {
				t.Fatalf("unexpected HTTP telemetry loss: %#v", event)
			}
		}
	}
	if !toolSeen || !summarySeen {
		t.Fatalf("tool=%v summary=%v telemetry=%s", toolSeen, summarySeen, stderr.String())
	}
	for _, prohibited := range []string{sentinel, "PRIVATE-device", "PRIVATE-firmware", "PRIVATE-raw-config-token", "PRIVATE-image-bytes", "https://private.invalid"} {
		if strings.Contains(stderr.String(), prohibited) {
			t.Fatalf("HTTP telemetry retained %q: %s", prohibited, stderr.String())
		}
	}
}

type failingTelemetryWriter struct{}

func (failingTelemetryWriter) Write([]byte) (int, error) {
	return 0, errors.New("PRIVATE-telemetry-writer-error")
}

type slowTelemetryWriter struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	output  bytes.Buffer
}

func (writer *slowTelemetryWriter) Write(data []byte) (int, error) {
	writer.once.Do(func() {
		close(writer.started)
		<-writer.release
	})
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.output.Write(data)
}

func (writer *slowTelemetryWriter) bytes() []byte {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return bytes.Clone(writer.output.Bytes())
}

func TestMCPResultIgnoresFailingTelemetrySink(t *testing.T) {
	for _, transport := range []string{telemetry.TransportStdio, telemetry.TransportHTTP} {
		t.Run(transport, func(t *testing.T) {
			recorder := telemetry.New(failingTelemetryWriter{}, "test")
			client, cleanup := connectTelemetryClient(t, transport, recorder, &telemetryDevice{})
			defer cleanup()
			result, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: ListDevicesToolName})
			if err != nil || result == nil || result.IsError {
				t.Fatalf("CallTool = %#v, %v", result, err)
			}
			if err := recorder.Close(context.Background()); err != nil {
				t.Fatalf("Close propagated sink failure: %v", err)
			}
		})
	}
}

func TestMCPResultDoesNotWaitForSlowTelemetrySink(t *testing.T) {
	for _, transport := range []string{telemetry.TransportStdio, telemetry.TransportHTTP} {
		t.Run(transport, func(t *testing.T) {
			writer := &slowTelemetryWriter{started: make(chan struct{}), release: make(chan struct{})}
			recorder := telemetry.New(writer, "test")
			client, cleanup := connectTelemetryClient(t, transport, recorder, &telemetryDevice{})
			defer cleanup()
			callDone := make(chan error, 1)
			go func() {
				result, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: ListDevicesToolName})
				if err == nil && (result == nil || result.IsError) {
					err = errors.New("successful tool returned an error result")
				}
				callDone <- err
			}()
			select {
			case <-writer.started:
			case <-time.After(time.Second):
				close(writer.release)
				t.Fatal("telemetry sink was not invoked")
			}
			select {
			case err := <-callDone:
				if err != nil {
					close(writer.release)
					t.Fatal(err)
				}
			case <-time.After(100 * time.Millisecond):
				close(writer.release)
				<-callDone
				t.Fatal("slow telemetry sink delayed MCP result")
			}
			closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			started := time.Now()
			err := recorder.Close(closeCtx)
			cancel()
			close(writer.release)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("Close error = %v, want deadline exceeded", err)
			}
			if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
				t.Fatalf("Close took %v after its deadline", elapsed)
			}
		})
	}
}

func TestTransportPressureReportsLossAndKeepsJSONLines(t *testing.T) {
	const queueCapacity = 256
	for _, transport := range []string{telemetry.TransportStdio, telemetry.TransportHTTP} {
		t.Run(transport, func(t *testing.T) {
			writer := &slowTelemetryWriter{started: make(chan struct{}), release: make(chan struct{})}
			recorder := telemetry.New(writer, "test")
			client, cleanup := connectTelemetryClient(t, transport, recorder, &telemetryDevice{})
			defer cleanup()
			call := func() {
				result, err := client.CallTool(context.Background(), &mcp.CallToolParams{Name: ListDevicesToolName})
				if err != nil || result == nil || result.IsError {
					t.Fatalf("CallTool = %#v, %v", result, err)
				}
			}
			call()
			select {
			case <-writer.started:
			case <-time.After(time.Second):
				close(writer.release)
				t.Fatal("telemetry sink was not invoked")
			}
			for index := 0; index < 2*queueCapacity+8; index++ {
				call()
			}
			close(writer.release)
			if err := recorder.Close(context.Background()); err != nil {
				t.Fatal(err)
			}

			decoder := json.NewDecoder(bytes.NewReader(writer.bytes()))
			toolEvents := 0
			summarySeen := false
			var droppedEvents uint64
			writerFailed := false
			for {
				var event struct {
					Stage         string `json:"stage"`
					Code          string `json:"code"`
					DroppedEvents uint64 `json:"dropped_events"`
					WriterFailed  bool   `json:"writer_failed"`
				}
				if err := decoder.Decode(&event); errors.Is(err, io.EOF) {
					break
				} else if err != nil {
					t.Fatalf("decode concurrent JSON line: %v", err)
				}
				if event.Stage == telemetry.StageTool {
					toolEvents++
				}
				if event.Code == "telemetry_summary" {
					summarySeen = true
					droppedEvents = event.DroppedEvents
					writerFailed = event.WriterFailed
				}
			}
			if toolEvents < 1+2*queueCapacity || !summarySeen || droppedEvents == 0 || writerFailed {
				t.Fatalf("tool events=%d summary=%v dropped=%d writer_failed=%v", toolEvents, summarySeen, droppedEvents, writerFailed)
			}
		})
	}
}

func connectTelemetryClient(t *testing.T, transport string, recorder *telemetry.Recorder, device Device) (*mcp.ClientSession, func()) {
	t.Helper()
	server := NewWithTelemetry(device, "test", recorder, transport)
	if transport == telemetry.TransportHTTP {
		httpServer := httptest.NewServer(NewHTTPHandler(server, ""))
		client, err := mcp.NewClient(&mcp.Implementation{Name: "telemetry-sink-test", Version: "test"}, nil).Connect(
			context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL + MCPPath}, nil,
		)
		if err != nil {
			httpServer.Close()
			t.Fatal(err)
		}
		return client, func() {
			_ = client.Close()
			httpServer.Close()
		}
	}
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()
	serverTransport := &mcp.IOTransport{Reader: serverReader, Writer: serverWriter}
	clientTransport := &mcp.IOTransport{Reader: clientReader, Writer: clientWriter}
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	client, err := mcp.NewClient(&mcp.Implementation{Name: "telemetry-sink-test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatal(err)
	}
	return client, func() {
		_ = client.Close()
		_ = serverSession.Close()
		_ = clientWriter.Close()
		_ = clientReader.Close()
		_ = serverWriter.Close()
		_ = serverReader.Close()
	}
}

func TestToolTelemetryUsesFinalOutputSchemaFailure(t *testing.T) {
	var stderr bytes.Buffer
	recorder := telemetry.New(&stderr, "test")
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
	recorder := telemetry.New(&stderr, "test")
	clientSession, cleanup := connectTelemetryClient(t, telemetry.TransportStdio, recorder, &privacyTelemetryDevice{})
	defer cleanup()
	for _, call := range []mcp.CallToolParams{
		{Name: GetStatusToolName, Arguments: map[string]any{"device": sentinel}},
		{Name: CaptureScreenToolName, Arguments: map[string]any{"device": sentinel}},
		{Name: KeyboardToolName, Arguments: map[string]any{"device": sentinel, "operation": "type_text", "text": sentinel}},
		{Name: MountVirtualMediaURLToolName, Arguments: map[string]any{"device": sentinel, "url": "https://private.invalid/" + sentinel + ".iso?token=" + sentinel}},
	} {
		_, _ = clientSession.CallTool(context.Background(), &call)
	}
	if err := recorder.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, prohibited := range []string{sentinel, "PRIVATE-device", "PRIVATE-firmware", "PRIVATE-raw-config-token", "PRIVATE-image-bytes", "https://private.invalid"} {
		if strings.Contains(stderr.String(), prohibited) {
			t.Fatalf("telemetry retained %q: %s", prohibited, stderr.String())
		}
	}
}
