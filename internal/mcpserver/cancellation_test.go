package mcpserver

import (
	"context"
	"net/http/httptest"
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
