package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type cancellationDevice struct {
	recordingDevice
	started     chan struct{}
	canceled    chan struct{}
	release     chan struct{}
	releaseOnce sync.Once
}

type deadlineCaptureDevice struct{ recordingDevice }

type notSentCancellationError struct{}

type deadlineBlockingJSON struct{ ctx context.Context }

func (value deadlineBlockingJSON) MarshalJSON() ([]byte, error) {
	<-value.ctx.Done()
	return []byte(`{}`), nil
}

func (notSentCancellationError) Error() string            { return "capture canceled before dispatch" }
func (notSentCancellationError) ToolErrorCode() string    { return "canceled" }
func (notSentCancellationError) ToolErrorOutcome() string { return "not_sent" }
func (notSentCancellationError) ToolErrorRetryable() bool { return false }

func (*deadlineCaptureDevice) CaptureScreen(ctx context.Context, _ string, _ CaptureRequest) (CaptureResult, error) {
	<-ctx.Done()
	return CaptureResult{
		Device: "lab", CapturedAt: time.Now().UTC(), MIMEType: "image/png",
		Width: 1, Height: 1, PNG: []byte{1},
	}, nil
}

func (device *cancellationDevice) Release() {
	device.releaseOnce.Do(func() { close(device.release) })
}

func (device *cancellationDevice) Status(ctx context.Context, _ string) (Status, error) {
	close(device.started)
	select {
	case <-ctx.Done():
		close(device.canceled)
		return Status{}, ctx.Err()
	case <-device.release:
		return Status{}, nil
	}
}

func TestHTTPClientCancellationReachesToolHandler(t *testing.T) {
	device := &cancellationDevice{started: make(chan struct{}), canceled: make(chan struct{}), release: make(chan struct{})}
	defer device.Release()
	httpServer := httptest.NewServer(NewHTTPHandler(New(device, "test"), ""))
	defer httpServer.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(
		context.Background(),
		&mcp.StreamableClientTransport{Endpoint: httpServer.URL + MCPPath},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	ctx, cancel := context.WithCancel(context.Background())
	callDone := make(chan struct{})
	callResult := make(chan error, 1)
	go func() {
		defer close(callDone)
		_, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
			Name:      GetStatusToolName,
			Arguments: map[string]any{"device": "lab"},
		})
		callResult <- err
	}()
	select {
	case <-device.started:
	case <-time.After(time.Second):
		t.Fatal("tool handler did not start")
	}
	cancel()
	select {
	case <-device.canceled:
	case <-time.After(200 * time.Millisecond):
		device.Release()
		t.Fatal("HTTP cancellation did not reach tool handler")
	}
	select {
	case <-callDone:
	case <-time.After(time.Second):
		t.Fatal("client call did not stop")
	}
	if err := <-callResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("client call error = %v, want cancellation", err)
	}
}

func TestCaptureDeadlineCoversMCPResultConstruction(t *testing.T) {
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := newServer(&deadlineCaptureDevice{}, "test", 10*time.Millisecond).Connect(context.Background(), serverTransport, nil)
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
		Name: CaptureScreenToolName, Arguments: map[string]any{"device": "lab"},
	})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("CallTool = %#v, %v, want timeout tool result", result, err)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, `"code":"timeout"`) || !strings.Contains(text.Text, `"outcome":"failed"`) {
		t.Fatalf("tool result = %#v, want timeout/failed", result.Content)
	}
}

func TestCaptureDeadlineMiddlewarePreservesCallerNotSentOutcome(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	result := new(mcp.CallToolResult)
	result.SetError(toolFailure(notSentCancellationError{}, false))
	handler := captureDeadlineMiddleware(time.Second)(func(context.Context, string, mcp.Request) (mcp.Result, error) {
		cancel()
		return result, nil
	})
	request := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: CaptureScreenToolName}}

	got, err := handler(ctx, "tools/call", request)
	if err != nil {
		t.Fatal(err)
	}
	callResult, ok := got.(*mcp.CallToolResult)
	if !ok || len(callResult.Content) != 1 {
		t.Fatalf("result = %#v, want call-tool failure", got)
	}
	text, ok := callResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content = %#v, want text failure", callResult.Content)
	}
	var failure toolError
	if err := json.Unmarshal([]byte(text.Text), &failure); err != nil {
		t.Fatal(err)
	}
	if failure.Code != toolErrorCanceled || failure.Outcome != "not_sent" || failure.Retryable {
		t.Fatalf("failure = %#v, want canceled/not_sent/non-retryable", failure)
	}
}

func TestCaptureDeadlineCoversResponseSerialization(t *testing.T) {
	handler := captureDeadlineMiddleware(10 * time.Millisecond)(func(ctx context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
		return &mcp.CallToolResult{
			Content:           []mcp.Content{&mcp.TextContent{Text: "capture"}},
			StructuredContent: deadlineBlockingJSON{ctx: ctx},
		}, nil
	})
	request := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: CaptureScreenToolName}}

	result, err := handler(context.Background(), "tools/call", request)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		IsError bool `json:"isError"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if !wire.IsError || len(wire.Content) != 1 {
		t.Fatalf("result = %s, want timeout tool failure", encoded)
	}
	var failure toolError
	if err := json.Unmarshal([]byte(wire.Content[0].Text), &failure); err != nil {
		t.Fatal(err)
	}
	if failure.Code != toolErrorTimeout || failure.Outcome != toolOutcomeFailed {
		t.Fatalf("failure = %#v, want timeout/failed", failure)
	}
}
