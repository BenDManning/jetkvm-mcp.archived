package jetkvm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestManagerStatusProjectsTypedVirtualMediaWithoutRawFirmware(t *testing.T) {
	const sentinel = "JETKVM-PRIVATE-STATUS-SENTINEL-5d7c2c"
	session := &fakeSession{results: map[string]any{
		"ping": "pong",
		"getLocalVersion": map[string]any{
			"appVersion": "0.6.0", "systemVersion": "1.2.3",
			"unknownFirmwareField": sentinel,
		},
		"getActiveExtension": "",
		"getVirtualMediaState": map[string]any{
			"source": "HTTP", "mode": "CDROM",
			"url":                  "https://media.invalid/private/" + sentinel + ".iso?token=" + sentinel + "#" + sentinel,
			"unknownFirmwareField": sentinel,
		},
		"getVideoState": map[string]any{},
		"getUSBState":   "not attached",
	}}
	manager := testManager(t, session)
	status, err := manager.Status(context.Background(), "lab")
	if err != nil {
		t.Fatal(err)
	}
	if status.VirtualMedia == nil || !status.VirtualMedia.Mounted || status.VirtualMedia.SourceType != mcpserver.VirtualMediaSourceHTTP || status.VirtualMedia.Mode != "read_only" {
		t.Fatalf("virtual media = %#v", status.VirtualMedia)
	}
	assertJSONExcludesPrivateMedia(t, status, sentinel)
}

func TestManagerStatusRejectsUnknownFirmwareMediaValuesWithoutLeak(t *testing.T) {
	const sentinel = "JETKVM-PRIVATE-STATUS-VALUE-SENTINEL-49f1cb"
	session := &fakeSession{results: map[string]any{
		"ping":               "pong",
		"getLocalVersion":    map[string]any{},
		"getActiveExtension": "",
		"getVirtualMediaState": map[string]any{
			"source": sentinel, "mode": "CDROM", "filename": sentinel + ".iso",
			"url": "https://media.invalid/" + sentinel + ".iso?token=" + sentinel,
		},
		"getVideoState": map[string]any{},
		"getUSBState":   "not attached",
	}}
	status, err := testManager(t, session).Status(context.Background(), "lab")
	if err != nil {
		t.Fatal(err)
	}
	if status.VirtualMedia != nil || !slices.Contains(status.Warnings, "virtual media unavailable") {
		t.Fatalf("status = %#v, want fixed media warning and no media state", status)
	}
	assertJSONExcludesPrivateMedia(t, status, sentinel)
}

func TestManagerVirtualMediaResultsRedactURLPathAndFirmwareFields(t *testing.T) {
	const sentinel = "JETKVM-PRIVATE-MEDIA-SENTINEL-9f02b1"
	tests := []struct {
		name       string
		request    mcpserver.VirtualMediaRequest
		results    map[string]any
		wantSource mcpserver.VirtualMediaSourceType
	}{
		{
			name:    "HTTP status",
			request: mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaStatus},
			results: map[string]any{"getVirtualMediaState": map[string]any{
				"source": "HTTP", "mode": "CDROM",
				"url":                  "https://media.invalid/private/" + sentinel + ".iso?token=" + sentinel + "#" + sentinel,
				"unknownFirmwareField": sentinel,
			}},
			wantSource: mcpserver.VirtualMediaSourceHTTP,
		},
		{
			name:    "storage status",
			request: mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaStatus},
			results: map[string]any{"getVirtualMediaState": map[string]any{
				"source": "Storage", "mode": "Disk", "filename": sentinel + ".iso",
			}},
			wantSource: mcpserver.VirtualMediaSourceStorage,
		},
		{
			name:       "URL mount success",
			request:    mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaMountURL, Source: "https://media.invalid/private/" + sentinel + ".iso?token=" + sentinel + "#" + sentinel},
			results:    map[string]any{},
			wantSource: mcpserver.VirtualMediaSourceHTTP,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := testManager(t, &fakeSession{results: test.results})
			result, err := manager.VirtualMedia(context.Background(), "lab", test.request)
			if err != nil {
				t.Fatal(err)
			}
			if result.SourceType != test.wantSource {
				t.Fatalf("source type = %q, want %q", result.SourceType, test.wantSource)
			}
			assertJSONExcludesPrivateMedia(t, result, sentinel)
		})
	}
}

func TestManagerVirtualMediaFileSuccessRedactsConfiguredPath(t *testing.T) {
	const sentinel = "JETKVM-PRIVATE-PATH-SENTINEL-a5d08e"
	mediaDirectory := t.TempDir()
	path := sentinel + ".iso"
	if err := os.WriteFile(filepath.Join(mediaDirectory, path), []byte("fixture media"), 0o600); err != nil {
		t.Fatal(err)
	}
	session := &fakeSession{results: map[string]any{
		"listStorageFiles":       map[string]any{"files": []any{}},
		"startStorageFileUpload": map[string]any{"alreadyUploadedBytes": 0, "dataChannel": "upload_12345678-1234-1234-1234-123456789abc"},
	}}
	manager := mediaManager(t, session, mediaDirectory)
	result, err := manager.VirtualMedia(context.Background(), "lab", mcpserver.VirtualMediaRequest{Operation: mcpserver.VirtualMediaUpload, Source: path})
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceType != mcpserver.VirtualMediaSourceStorage {
		t.Fatalf("result = %#v", result)
	}
	assertJSONExcludesPrivateMedia(t, result, sentinel)
}

func TestMCPMediaResultsAndErrorsRedactPrivateSources(t *testing.T) {
	const sentinel = "JETKVM-PRIVATE-MCP-SENTINEL-8c661f"
	tests := []struct {
		name      string
		manager   func(*testing.T) *Manager
		call      mcp.CallToolParams
		wantError bool
	}{
		{
			name: "status ignores raw firmware fields",
			manager: func(t *testing.T) *Manager {
				return testManager(t, &fakeSession{results: map[string]any{
					"ping": "pong", "getLocalVersion": map[string]any{}, "getActiveExtension": "",
					"getVirtualMediaState": map[string]any{
						"source": "HTTP", "mode": "CDROM",
						"url":                  "https://media.invalid/private/" + sentinel + ".iso?token=" + sentinel + "#" + sentinel,
						"unknownFirmwareField": sentinel,
					},
					"getVideoState": map[string]any{}, "getUSBState": "not attached",
				}})
			},
			call: mcp.CallToolParams{Name: mcpserver.GetStatusToolName, Arguments: map[string]any{"device": "lab"}},
		},
		{
			name: "URL success",
			manager: func(t *testing.T) *Manager {
				return testManager(t, &fakeSession{results: map[string]any{}})
			},
			call: mcp.CallToolParams{Name: mcpserver.MountVirtualMediaURLToolName, Arguments: map[string]any{
				"device": "lab", "url": "https://media.invalid/private/" + sentinel + ".iso?token=" + sentinel + "#" + sentinel,
			}},
		},
		{
			name: "URL credential rejection",
			manager: func(t *testing.T) *Manager {
				return testManager(t, &fakeSession{results: map[string]any{}})
			},
			call: mcp.CallToolParams{Name: mcpserver.MountVirtualMediaURLToolName, Arguments: map[string]any{
				"device": "lab", "url": "https://" + sentinel + ":" + sentinel + "@media.invalid/private.iso",
			}},
			wantError: true,
		},
		{
			name: "missing local path",
			manager: func(t *testing.T) *Manager {
				return mediaManager(t, &fakeSession{results: map[string]any{}}, t.TempDir())
			},
			call: mcp.CallToolParams{Name: mcpserver.UploadVirtualMediaFileToolName, Arguments: map[string]any{
				"device": "lab", "path": sentinel + ".iso",
			}},
			wantError: true,
		},
		{
			name: "invalid firmware media state",
			manager: func(t *testing.T) *Manager {
				return testManager(t, &fakeSession{results: map[string]any{
					"getVirtualMediaState": map[string]any{
						"source": "HTTP", "mode": sentinel,
						"url": "https://media.invalid/" + sentinel + ".iso",
					},
				}})
			},
			call:      mcp.CallToolParams{Name: mcpserver.GetVirtualMediaStatusToolName, Arguments: map[string]any{"device": "lab"}},
			wantError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientTransport, serverTransport := mcp.NewInMemoryTransports()
			serverSession, err := mcpserver.New(test.manager(t), "test").Connect(context.Background(), serverTransport, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer serverSession.Close()
			clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer clientSession.Close()
			result, err := clientSession.CallTool(context.Background(), &test.call)
			if err != nil || result == nil || result.IsError != test.wantError {
				t.Fatalf("CallTool = %#v, %v, wantError=%v", result, err, test.wantError)
			}
			assertJSONExcludesPrivateMedia(t, result, sentinel)
		})
	}
}

func TestMCPVirtualMediaURLDefaultDenyIsAnUnsentToolError(t *testing.T) {
	const source = "https://unconfigured-media.example.invalid/private.iso?token=private#fragment"
	session := &fakeSession{results: map[string]any{}}
	provider := &mediaURLPolicyProvider{session: session}
	base, err := url.Parse("https://jetkvm.invalid")
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager([]DeviceConfig{{Name: "lab", BaseURL: *base}}, provider)
	if err != nil {
		t.Fatal(err)
	}

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := mcpserver.New(manager, "test").Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	for _, call := range []mcp.CallToolParams{
		{Name: mcpserver.MountVirtualMediaURLToolName, Arguments: map[string]any{"device": "lab", "url": source}},
	} {
		result, err := clientSession.CallTool(context.Background(), &call)
		if err != nil {
			t.Fatalf("CallTool(%s) returned protocol error: %v", call.Name, err)
		}
		if result == nil || !result.IsError || len(result.Content) != 1 {
			t.Fatalf("CallTool(%s) = %#v, want one tool-error content block", call.Name, result)
		}
		text, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("CallTool(%s) error content = %T, want text", call.Name, result.Content[0])
		}
		var failure struct {
			Code      string `json:"code"`
			Outcome   string `json:"outcome"`
			Retryable bool   `json:"retryable"`
		}
		if err := json.Unmarshal([]byte(text.Text), &failure); err != nil {
			t.Fatalf("CallTool(%s) error is not stable JSON: %q: %v", call.Name, text.Text, err)
		}
		if failure.Code != "invalid_input" || failure.Outcome != ToolOutcomeNotSent || failure.Retryable {
			t.Fatalf("CallTool(%s) failure = %+v, want non-retryable invalid_input/not_sent", call.Name, failure)
		}
		if bytes.Contains([]byte(text.Text), []byte(source)) || bytes.Contains([]byte(text.Text), []byte("unconfigured-media.example.invalid")) {
			t.Fatalf("CallTool(%s) error leaked rejected URL: %s", call.Name, text.Text)
		}
	}
	if calls := provider.calls.Load(); calls != 0 {
		t.Fatalf("provider calls = %d, want none", calls)
	}
	if len(session.calls) != 0 {
		t.Fatalf("device calls = %#v, want none", session.calls)
	}
}

func TestMCPMediaSentinelsStayRedactedOverStdioAndHTTP(t *testing.T) {
	const sentinel = "JETKVM-PRIVATE-TRANSPORT-SENTINEL-153bd4"
	tests := []struct {
		name    string
		connect func(*testing.T) (*mcp.ClientSession, *lockedBuffer, func())
	}{
		{name: "stdio", connect: connectTypedResultStdio},
		{name: "http", connect: connectTypedResultHTTP},
	}
	calls := []struct {
		params    mcp.CallToolParams
		wantError bool
	}{
		{params: mcp.CallToolParams{Name: mcpserver.GetStatusToolName, Arguments: map[string]any{"device": "lab"}}},
		{params: mcp.CallToolParams{Name: mcpserver.GetVirtualMediaStatusToolName, Arguments: map[string]any{"device": "lab"}}},
		{params: mcp.CallToolParams{Name: mcpserver.MountVirtualMediaURLToolName, Arguments: map[string]any{
			"device": "lab", "url": "https://media.invalid/private/" + sentinel + ".iso?token=" + sentinel + "#" + sentinel,
		}}},
		{params: mcp.CallToolParams{Name: mcpserver.MountVirtualMediaURLToolName, Arguments: map[string]any{
			"device": "lab", "url": "https://" + sentinel + ":" + sentinel + "@media.invalid/private.iso",
		}}, wantError: true},
		{params: mcp.CallToolParams{Name: mcpserver.UploadVirtualMediaFileToolName, Arguments: map[string]any{
			"device": "lab", "path": sentinel + ".iso",
		}}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session, stderr, cleanup := test.connect(t)
			defer cleanup()
			for index := range calls {
				call := calls[index]
				t.Run(call.params.Name, func(t *testing.T) {
					result, err := session.CallTool(context.Background(), &call.params)
					if err != nil || result == nil || result.IsError != call.wantError {
						t.Fatalf("CallTool(%s) = %#v, %v, wantError=%v", call.params.Name, result, err, call.wantError)
					}
					assertJSONExcludesPrivateMedia(t, result, sentinel)
					if stderr.Contains(sentinel) {
						t.Fatalf("stderr leaked private sentinel: %s", stderr.String())
					}
				})
			}
		})
	}
}

func TestTypedResultStdioHelperProcess(t *testing.T) {
	if os.Getenv("JETKVM_TYPED_RESULT_HELPER") != "1" {
		return
	}
	manager := typedResultManager(t, "JETKVM-PRIVATE-TRANSPORT-SENTINEL-153bd4")
	if err := mcpserver.New(manager, "test").Run(context.Background(), &mcp.StdioTransport{}); err != nil && !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
}

func connectTypedResultStdio(t *testing.T) (*mcp.ClientSession, *lockedBuffer, func()) {
	t.Helper()
	stderr := new(lockedBuffer)
	command := exec.Command(os.Args[0], "-test.run=^TestTypedResultStdioHelperProcess$")
	command.Env = append(os.Environ(), "JETKVM_TYPED_RESULT_HELPER=1")
	command.Stderr = stderr
	session, err := mcp.NewClient(&mcp.Implementation{Name: "typed-result-test", Version: "test"}, nil).Connect(
		context.Background(), &mcp.CommandTransport{Command: command, TerminateDuration: time.Second}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return session, stderr, func() { _ = session.Close() }
}

func connectTypedResultHTTP(t *testing.T) (*mcp.ClientSession, *lockedBuffer, func()) {
	t.Helper()
	stderr := new(lockedBuffer)
	manager := typedResultManager(t, "JETKVM-PRIVATE-TRANSPORT-SENTINEL-153bd4")
	server := httptest.NewServer(mcpserver.NewHTTPHandler(mcpserver.New(manager, "test"), ""))
	session, err := mcp.NewClient(&mcp.Implementation{Name: "typed-result-test", Version: "test"}, nil).Connect(
		context.Background(), &mcp.StreamableClientTransport{Endpoint: server.URL + mcpserver.MCPPath}, nil,
	)
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	return session, stderr, func() {
		_ = session.Close()
		server.Close()
	}
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (buffer *lockedBuffer) Write(value []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.Write(value)
}

func (buffer *lockedBuffer) Contains(value string) bool {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return bytes.Contains(buffer.buffer.Bytes(), []byte(value))
}

func (buffer *lockedBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buffer.String()
}

func typedResultManager(t *testing.T, sentinel string) *Manager {
	t.Helper()
	return testManager(t, &fakeSession{results: map[string]any{
		"ping": "pong", "getLocalVersion": map[string]any{"unknownFirmwareField": sentinel}, "getActiveExtension": "",
		"getVirtualMediaState": map[string]any{
			"source": "HTTP", "mode": "CDROM",
			"url":                  "https://media.invalid/private/" + sentinel + ".iso?token=" + sentinel + "#" + sentinel,
			"unknownFirmwareField": sentinel,
		},
		"getVideoState": map[string]any{}, "getUSBState": "not attached",
	}})
}

func assertJSONExcludesPrivateMedia(t *testing.T, value any, sentinels ...string) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range append(sentinels, "media.invalid", "unknownFirmwareField", `"url"`, `"filename"`, `"source"`) {
		if bytes.Contains(encoded, []byte(forbidden)) {
			t.Fatalf("typed result leaked %q: %s", forbidden, encoded)
		}
	}
}
