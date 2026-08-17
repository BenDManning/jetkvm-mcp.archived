package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSchemaErrorsDoNotReflectArguments(t *testing.T) {
	const privateArgument = "JETKVM-PRIVATE-ARGUMENT-7eea7c9f"
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
		Name: KeyboardToolName,
		Arguments: map[string]any{
			"device":    "lab",
			"operation": privateArgument,
		},
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	content, err := json.Marshal(result.Content)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || bytes.Contains(content, []byte(privateArgument)) {
		t.Fatalf("schema rejection reflected the rejected argument: %#v", result)
	}
}
