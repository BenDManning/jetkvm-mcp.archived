package mcpserver

import (
	"context"
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
	go func() {
		defer close(callDone)
		_, _ = clientSession.CallTool(ctx, &mcp.CallToolParams{
			Name:      GetStatusToolName,
			Arguments: map[string]any{"device": "lab"},
		})
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
