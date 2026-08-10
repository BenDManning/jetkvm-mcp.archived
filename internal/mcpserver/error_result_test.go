package mcpserver

import (
	"context"
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
}
