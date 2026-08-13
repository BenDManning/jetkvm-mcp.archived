package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type typedResultDevice struct {
	recordingDevice
	failure    error
	mediaCalls int
}

func (*typedResultDevice) Status(context.Context, string) (Status, error) {
	return Status{
		Device: "lab", Connected: true,
		VirtualMedia: &VirtualMediaState{Mounted: true, SourceType: VirtualMediaSourceHTTP, Mode: "read_only"},
	}, nil
}

func (device *typedResultDevice) VirtualMedia(_ context.Context, _ string, request VirtualMediaRequest) (VirtualMediaResult, error) {
	device.mediaCalls++
	if device.failure != nil {
		return VirtualMediaResult{}, device.failure
	}
	return VirtualMediaResult{
		Device: "lab", Operation: request.Operation, Mounted: true,
		SourceType: VirtualMediaSourceHTTP, Mode: "read_only", Status: "observed",
	}, nil
}

func TestTypedMediaSchemaErrorsDoNotEchoPrivateSources(t *testing.T) {
	const (
		userinfo = "private-user:private-password@"
		query    = "private-query-token"
		fragment = "private-fragment-token"
		pathPart = "private-local-path"
	)
	overlongURL := "https://" + userinfo + "media.invalid/" + strings.Repeat("x", 4096) + ".iso?token=" + query + "#" + fragment
	overlongPath := strings.Repeat(pathPart+"/", 300) + "private.iso"

	for _, test := range []struct {
		name      string
		tool      string
		arguments map[string]any
	}{
		{name: "mount URL", tool: MountVirtualMediaURLToolName, arguments: map[string]any{"device": "lab", "url": overlongURL}},
		{name: "mount file", tool: MountVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": overlongPath}},
		{name: "upload file", tool: UploadVirtualMediaFileToolName, arguments: map[string]any{"device": "lab", "path": overlongPath}},
		{name: "legacy malformed source", tool: VirtualMediaToolName, arguments: map[string]any{
			"device": "lab", "operation": "mount_url", "source": map[string]any{"private": query},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			device := &typedResultDevice{}
			session, cleanup := connectVirtualMediaTestClient(t, device)
			defer cleanup()

			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: test.tool, Arguments: test.arguments})
			if err != nil || result == nil || !result.IsError {
				t.Fatalf("CallTool(%s) = %#v, %v", test.tool, result, err)
			}
			encoded, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{userinfo, query, fragment, pathPart, "media.invalid"} {
				if bytes.Contains(encoded, []byte(forbidden)) {
					t.Fatalf("%s schema error leaked private source component %q: %s", test.tool, forbidden, encoded)
				}
			}
			if device.mediaCalls != 0 {
				t.Fatalf("%s dispatched rejected input to device", test.tool)
			}
		})
	}
}

func TestTypedStatusAndMediaResultsExcludeRawSources(t *testing.T) {
	session, cleanup := connectVirtualMediaTestClient(t, &typedResultDevice{})
	defer cleanup()
	for _, call := range []mcp.CallToolParams{
		{Name: GetStatusToolName, Arguments: map[string]any{"device": "lab"}},
		{Name: GetVirtualMediaStatusToolName, Arguments: map[string]any{"device": "lab"}},
	} {
		result, err := session.CallTool(context.Background(), &call)
		if err != nil || result == nil || result.IsError {
			t.Fatalf("CallTool(%s) = %#v, %v", call.Name, result, err)
		}
		encoded, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range [][]byte{[]byte(`"source"`), []byte(`"url"`), []byte(`"filename"`)} {
			if bytes.Contains(encoded, forbidden) {
				t.Fatalf("%s result exposes raw media source field %s: %s", call.Name, forbidden, encoded)
			}
		}
		if !bytes.Contains(encoded, []byte(`"sourceType":"http"`)) {
			t.Fatalf("%s result lacks typed source class: %s", call.Name, encoded)
		}
	}
}

func TestTypedMediaErrorsExcludeRawSources(t *testing.T) {
	const sentinel = "JETKVM-PRIVATE-ERROR-SENTINEL-74b1d8"
	device := &typedResultDevice{failure: errors.New("firmware rejected https://media.invalid/" + sentinel + ".iso?token=" + sentinel)}
	session, cleanup := connectVirtualMediaTestClient(t, device)
	defer cleanup()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      MountVirtualMediaURLToolName,
		Arguments: map[string]any{"device": "lab", "url": "https://media.invalid/" + sentinel + ".iso?token=" + sentinel},
	})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("CallTool = %#v, %v", result, err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range [][]byte{[]byte(sentinel), []byte("media.invalid"), []byte("token=")} {
		if bytes.Contains(encoded, forbidden) {
			t.Fatalf("tool error leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestTypedMediaOutputSchemasExcludeRawSources(t *testing.T) {
	session, cleanup := connectVirtualMediaTestClient(t, &typedResultDevice{})
	defer cleanup()
	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range listed.Tools {
		if tool.Name != GetStatusToolName && tool.Name != GetVirtualMediaStatusToolName &&
			tool.Name != MountVirtualMediaURLToolName && tool.Name != MountVirtualMediaFileToolName &&
			tool.Name != UnmountVirtualMediaToolName && tool.Name != UploadVirtualMediaFileToolName &&
			tool.Name != VirtualMediaToolName {
			continue
		}
		encoded, err := json.Marshal(tool.OutputSchema)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(encoded, []byte(`"source"`)) {
			t.Fatalf("%s output schema exposes raw source: %s", tool.Name, encoded)
		}
	}
}
