package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type failingDevice struct {
	recordingDevice
}

func (*failingDevice) Status(context.Context, string) (Status, error) {
	return Status{}, errors.New("device is offline")
}

type timedOutMutationDevice struct {
	recordingDevice
}

func (*timedOutMutationDevice) Power(context.Context, string, PowerAction, string) (PowerResult, error) {
	return PowerResult{}, context.DeadlineExceeded
}

type connectionFailedMutationDevice struct {
	recordingDevice
}

func (*connectionFailedMutationDevice) Power(context.Context, string, PowerAction, string) (PowerResult, error) {
	return PowerResult{}, classifiedFixtureError{
		error: errors.New("private connection detail"), code: "device_unavailable", outcome: "not_sent",
	}
}

type timedOutMediaStatusDevice struct {
	recordingDevice
}

func (*timedOutMediaStatusDevice) VirtualMedia(context.Context, string, VirtualMediaRequest) (VirtualMediaResult, error) {
	return VirtualMediaResult{}, classifiedFixtureError{
		error: context.DeadlineExceeded, code: "timeout", outcome: "failed", retryable: true,
	}
}

func TestToolExecutionFailuresUseCallToolIsError(t *testing.T) {
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(&failingDevice{}, "test").Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      GetStatusToolName,
		Arguments: map[string]any{"device": "lab"},
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	if !result.IsError || len(result.Content) == 0 {
		t.Fatalf("CallTool result = %#v, want tool error content", result)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("error content = %T, want text", result.Content[0])
	}
	if bytes.Contains([]byte(text.Text), []byte("device is offline")) {
		t.Fatalf("tool error leaked raw cause: %q", text.Text)
	}
	var got struct {
		Version string `json:"version"`
		Code    string `json:"code"`
		Outcome string `json:"outcome"`
	}
	if err := json.Unmarshal([]byte(text.Text), &got); err != nil {
		t.Fatalf("error content is not stable JSON: %q: %v", text.Text, err)
	}
	if got.Version != "1" || got.Code != "operation_failed" || got.Outcome != "failed" {
		t.Fatalf("tool error = %+v, want sanitized operation failure", got)
	}
}

func TestMutationTimeoutHasStableUnknownOutcome(t *testing.T) {
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(&timedOutMutationDevice{}, "test").Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      PressHostPowerButtonToolName,
		Arguments: map[string]any{"device": "lab"},
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	if !result.IsError || len(result.Content) != 1 {
		t.Fatalf("CallTool result = %#v, want one tool-error content block", result)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("error content = %T, want text", result.Content[0])
	}
	var got struct {
		Version   string `json:"version"`
		Code      string `json:"code"`
		Outcome   string `json:"outcome"`
		Retryable bool   `json:"retryable"`
	}
	if err := json.Unmarshal([]byte(text.Text), &got); err != nil {
		t.Fatalf("error content is not stable JSON: %q: %v", text.Text, err)
	}
	if got.Version != "1" || got.Code != "timeout" || got.Outcome != "unknown" || got.Retryable {
		t.Fatalf("tool error = %+v, want timeout with non-retryable unknown outcome", got)
	}
}

func TestMutationUnknownOutcomeNeverReportsCompletedOrBlindRetry(t *testing.T) {
	failure := toolFailure(context.DeadlineExceeded, true).Error()
	for _, forbidden := range []string{"completed", "retry now", "retry safely", "safe to retry"} {
		if bytes.Contains(bytes.ToLower([]byte(failure)), []byte(forbidden)) {
			t.Fatalf("unknown mutation error recommends or reports %q: %s", forbidden, failure)
		}
	}
}

func TestMutationValidationFailureIsDefinitelyNotSent(t *testing.T) {
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(&recordingDevice{}, "test").Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      WakeHostLANToolName,
		Arguments: map[string]any{"device": "lab", "target": ""},
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	failure := decodeToolError(t, result)
	if failure.Code != "invalid_input" || failure.Outcome != "not_sent" || failure.Retryable {
		t.Fatalf("tool error = %+v, want non-retryable invalid_input that was not sent", failure)
	}
}

func TestMutationConnectionFailurePreservesDefinitelyNotSent(t *testing.T) {
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(&connectionFailedMutationDevice{}, "test").Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: PressHostPowerButtonToolName, Arguments: map[string]any{"device": "lab"},
	})
	if err != nil {
		t.Fatal(err)
	}
	failure := decodeToolError(t, result)
	if failure.Code != "device_unavailable" || failure.Outcome != "not_sent" || failure.Retryable {
		t.Fatalf("tool error = %+v, want non-retryable device_unavailable/not_sent", failure)
	}
}

type decodedToolError struct {
	Version   string `json:"version"`
	Code      string `json:"code"`
	Outcome   string `json:"outcome"`
	Retryable bool   `json:"retryable"`
}

func decodeToolError(t *testing.T, result *mcp.CallToolResult) decodedToolError {
	t.Helper()
	if result == nil || !result.IsError || len(result.Content) != 1 {
		t.Fatalf("CallTool result = %#v, want one tool-error content block", result)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("error content = %T, want text", result.Content[0])
	}
	var failure decodedToolError
	if err := json.Unmarshal([]byte(text.Text), &failure); err != nil {
		t.Fatalf("error content is not stable JSON: %q: %v", text.Text, err)
	}
	return failure
}

func TestStableErrorCodesRemainDistinct(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		code string
	}{
		{name: "device unavailable", err: classifiedFixtureError{error: errors.New("private device detail"), code: "device_unavailable", outcome: "failed", retryable: true}, code: "device_unavailable"},
		{name: "video unavailable", err: classifiedFixtureError{error: errors.New("private decoder detail"), code: "video_unavailable", outcome: "failed"}, code: "video_unavailable"},
		{name: "no signal", err: classifiedFixtureError{error: errors.New("private signal detail"), code: "no_signal", outcome: "failed", retryable: true}, code: "no_signal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			failure := toolFailure(test.err, false)
			var got decodedToolError
			if err := json.Unmarshal([]byte(failure.Error()), &got); err != nil {
				t.Fatal(err)
			}
			if got.Code != test.code || got.Outcome != "failed" {
				t.Fatalf("tool error = %+v", got)
			}
			if bytes.Contains([]byte(failure.Error()), []byte("private")) {
				t.Fatalf("tool error leaked private detail: %s", failure)
			}
		})
	}
}

func TestReadTimeoutHasFailedRetryableOutcome(t *testing.T) {
	failure := toolFailure(classifiedFixtureError{
		error: errors.New("private timeout detail"), code: "timeout", outcome: "unknown", retryable: false,
	}, false)
	var got decodedToolError
	if err := json.Unmarshal([]byte(failure.Error()), &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != "timeout" || got.Outcome != "failed" || !got.Retryable {
		t.Fatalf("tool error = %+v, want retryable failed read", got)
	}
}

func TestVirtualMediaStatusPreservesReadOnlyRetryability(t *testing.T) {
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(&timedOutMediaStatusDevice{}, "test").Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: VirtualMediaToolName, Arguments: map[string]any{"device": "lab", "operation": "status"},
	})
	if err != nil {
		t.Fatal(err)
	}
	failure := decodeToolError(t, result)
	if failure.Code != "timeout" || failure.Outcome != "failed" || !failure.Retryable {
		t.Fatalf("tool error = %+v, want retryable timeout/failed read", failure)
	}
}

func TestUnknownInternalClassificationFallsBackToStableTaxonomy(t *testing.T) {
	failure := toolFailure(classifiedFixtureError{
		error: errors.New("private future detail"), code: "future_private_code", outcome: "maybe",
	}, true)
	var got decodedToolError
	if err := json.Unmarshal([]byte(failure.Error()), &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != "operation_failed" || got.Outcome != "unknown" || got.Retryable {
		t.Fatalf("tool error = %+v, want conservative stable fallback", got)
	}
}

type classifiedFixtureError struct {
	error
	code      string
	outcome   string
	retryable bool
}

func (err classifiedFixtureError) ToolErrorCode() string    { return err.code }
func (err classifiedFixtureError) ToolErrorOutcome() string { return err.outcome }
func (err classifiedFixtureError) ToolErrorRetryable() bool { return err.retryable }
