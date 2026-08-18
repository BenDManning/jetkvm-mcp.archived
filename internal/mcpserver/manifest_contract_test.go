package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	updateToolManifestEnvironment = "JETKVM_UPDATE_TOOL_MANIFEST"
	toolManifestCount             = 18
)

var toolManifestFixturePath = filepath.Join("testdata", "tool-manifest.json")

type contractDevice struct{}

func (*contractDevice) ListDevices(context.Context) (DeviceList, error) {
	return DeviceList{Devices: []ConfiguredDevice{
		{Device: "fixture-device", Capabilities: DeviceCapabilities{
			MountVirtualMediaURL:   true,
			MountVirtualMediaFile:  true,
			UploadVirtualMediaFile: true,
			WakeHostLAN:            true,
		}},
	}}, nil
}

func (*contractDevice) Status(context.Context, string) (Status, error) {
	atxPowerOn, videoReady, usbWakeAttached := true, true, true
	return Status{
		Device: "fixture-device", Connected: true, Application: "fixture-app", System: "fixture-system",
		Extension: "atx-power", ATXPowerOn: &atxPowerOn, VideoReady: &videoReady,
		VideoWidth: 640, VideoHeight: 480, VideoFPS: 30,
		VirtualMedia: &VirtualMediaState{Mounted: false},
		USBState:     "attached", USBWakeAttached: &usbWakeAttached, Warnings: []StatusWarning{StatusWarningVideoUnavailable},
	}, nil
}

func (*contractDevice) ReleaseSession(_ context.Context, device string) (SessionReleaseResult, error) {
	return SessionReleaseResult{Device: device, Status: SessionStatusReleased}, nil
}

func (*contractDevice) Power(_ context.Context, _ string, action PowerAction, target string) (PowerResult, error) {
	return PowerResult{Device: "fixture-device", Action: action, Target: target, Status: "completed"}, nil
}

func (*contractDevice) CaptureScreen(context.Context, string, CaptureRequest) (CaptureResult, error) {
	return CaptureResult{
		Device: "fixture-device", CapturedAt: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		MIMEType: "image/png", Width: 1, Height: 1, PNG: contractPNG(),
	}, nil
}

func (*contractDevice) Keyboard(_ context.Context, _ string, request KeyboardRequest) (KeyboardResult, error) {
	return KeyboardResult{Device: "fixture-device", Operation: request.Operation, Status: "completed"}, nil
}

func (*contractDevice) Mouse(_ context.Context, _ string, request MouseRequest) (MouseResult, error) {
	return MouseResult{Device: "fixture-device", Operation: request.Operation, Status: "completed"}, nil
}

func (*contractDevice) VirtualMedia(_ context.Context, _ string, request VirtualMediaRequest) (VirtualMediaResult, error) {
	sourceType := VirtualMediaSourceType("")
	status := ResultStatusCompleted
	mounted := false
	if request.Operation == VirtualMediaMountURL {
		sourceType = VirtualMediaSourceHTTP
		mounted = true
	} else if request.Operation == VirtualMediaMountFile || request.Operation == VirtualMediaUpload {
		sourceType = VirtualMediaSourceStorage
		mounted = request.Operation == VirtualMediaMountFile
	} else if request.Operation == VirtualMediaStatus {
		status = ResultStatusObserved
	}
	return VirtualMediaResult{
		Device: "fixture-device", Operation: request.Operation, Mounted: mounted,
		SourceType: sourceType, Mode: request.Mode, Status: status,
	}, nil
}

type contractFailingDevice struct {
	contractDevice
}

func (*contractFailingDevice) Status(context.Context, string) (Status, error) {
	return Status{}, errors.New("fixture device unavailable")
}

type manifestSnapshot struct {
	ProtocolVersion    string                  `json:"protocolVersion"`
	ClientCapabilities map[string]any          `json:"clientCapabilities"`
	ServerInfo         *mcp.Implementation     `json:"serverInfo"`
	Instructions       string                  `json:"instructions"`
	InitializeMeta     map[string]any          `json:"initializeMeta"`
	Capabilities       *mcp.ServerCapabilities `json:"capabilities"`
	Discovery          discoveryContract       `json:"discovery"`
	Tools              []manifestTool          `json:"tools"`
	Results            map[string]any          `json:"results"`
}

type manifestTool struct {
	Meta         map[string]any       `json:"meta"`
	Name         string               `json:"name"`
	Title        string               `json:"title"`
	Description  string               `json:"description"`
	Icons        []mcp.Icon           `json:"icons"`
	InputSchema  any                  `json:"inputSchema"`
	OutputSchema any                  `json:"outputSchema"`
	Annotations  *mcp.ToolAnnotations `json:"annotations"`
}

type observedDiscovery struct {
	ClientCapabilities map[string]any
	Result             discoveryContract
}

type discoveryContract struct {
	Meta              map[string]any          `json:"meta"`
	CacheScope        string                  `json:"cacheScope"`
	Capabilities      *mcp.ServerCapabilities `json:"capabilities"`
	Instructions      string                  `json:"instructions"`
	ResultType        string                  `json:"resultType"`
	SupportedVersions []string                `json:"supportedVersions"`
	TTLMs             int                     `json:"ttlMs"`
}

func TestToolManifestContract(t *testing.T) {
	ctx := context.Background()
	want := readToolManifestFixture(t)
	paths := []struct {
		name         string
		connect      func(*testing.T) (*mcp.ClientSession, *observedDiscovery, func())
		connectError func(*testing.T) (*mcp.ClientSession, func())
	}{
		{name: "in_memory", connect: connectManifestInMemory, connectError: connectManifestFailingInMemory},
		{name: "stdio_subprocess", connect: connectManifestStdio, connectError: connectManifestFailingStdio},
		{name: "stateless_http", connect: connectManifestHTTP, connectError: connectManifestFailingHTTP},
	}

	var canonical []byte
	for _, path := range paths {
		t.Run(path.name, func(t *testing.T) {
			session, discovery, cleanup := path.connect(t)
			defer cleanup()
			got := captureToolManifest(t, ctx, session, discovery, path.connectError, toolManifestCount)
			if canonical == nil {
				canonical = got
			} else if !bytes.Equal(got, canonical) {
				t.Fatalf("manifest differs from first transport (-first +%s):\n%s", path.name, jsonDiff(canonical, got))
			}
			if !manifestMatchesFixture(got, want) {
				t.Fatalf("manifest fixture changed; classify the compatibility impact and run `make update-tool-manifest` (-fixture +runtime):\n%s", jsonDiff(want, got))
			}
		})
	}
}

func TestToolManifestFixtureUpdate(t *testing.T) {
	if os.Getenv(updateToolManifestEnvironment) != "1" {
		t.Skip("fixture update was not requested")
	}
	ctx := context.Background()
	session, discovery, cleanup := connectManifestInMemory(t)
	defer cleanup()
	manifest := captureToolManifest(t, ctx, session, discovery, connectManifestFailingInMemory, toolManifestCount)
	if err := os.WriteFile(toolManifestFixturePath, append(manifest, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestToolManifestFixtureRejectsUnreviewedMutation(t *testing.T) {
	fixture := readToolManifestFixture(t)
	session, discovery, cleanup := connectManifestMutatedInMemory(t)
	defer cleanup()
	mutated := captureToolManifest(t, context.Background(), session, discovery, connectManifestFailingInMemory, toolManifestCount)
	if manifestMatchesFixture(mutated, fixture) {
		t.Fatal("temporary runtime tool mutation did not fail the manifest gate")
	}
}

func TestManifestClientAdvertisesNoDeprecatedCapabilities(t *testing.T) {
	ctx := context.Background()
	observed := new(observedDiscovery)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(&contractDevice{}, "contract-test").Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession := connectManifestClientWithObservation(t, ctx, clientTransport, observed)
	defer clientSession.Close()

	initialized := serverSession.InitializeParams()
	if initialized == nil || initialized.Capabilities == nil {
		t.Fatal("server did not observe client discovery capabilities")
	}
	capabilities := initialized.Capabilities
	// Roots is an SDK compatibility mirror whose zero value re-marshals as
	// {"roots":{}}; RootsV2 is the authoritative presence marker on the wire.
	//lint:ignore SA1019 Deliberately assert deprecated capabilities stay absent throughout their compatibility window.
	if capabilities.RootsV2 != nil || capabilities.Sampling != nil || capabilities.Elicitation != nil ||
		len(capabilities.Experimental) != 0 || len(capabilities.Extensions) != 0 {
		t.Fatalf("unexpected client capabilities = %#v", capabilities)
	}
	if got := observed.ClientCapabilities; len(got) != 0 {
		t.Fatalf("client discovery capabilities = %#v, want empty object", got)
	}
	if observed.Result.CacheScope != "public" || observed.Result.TTLMs != 0 ||
		len(observed.Result.SupportedVersions) != 1 || observed.Result.SupportedVersions[0] != SupportedProtocolVersion {
		t.Fatalf("discovery contract = %#v", observed.Result)
	}
	serverCapabilities := clientSession.InitializeResult().Capabilities
	toolsOnly := map[string]any{"tools": map[string]any{}}
	if !jsonEqual(serverCapabilities, toolsOnly) || !jsonEqual(observed.Result.Capabilities, toolsOnly) {
		t.Fatalf("server capability leakage: initialized=%#v discovery=%#v", serverCapabilities, observed.Result.Capabilities)
	}
	fixture := readToolManifestFixture(t)
	var snapshot manifestSnapshot
	if err := json.Unmarshal(fixture, &snapshot); err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(snapshot.ClientCapabilities, observed.ClientCapabilities) || !jsonEqual(snapshot.Discovery, observed.Result) {
		t.Fatalf("fixture discovery metadata does not match observed wire values")
	}
}

func captureToolManifest(t *testing.T, ctx context.Context, session *mcp.ClientSession, discovery *observedDiscovery, connectError func(*testing.T) (*mcp.ClientSession, func()), expectedToolCount int) []byte {
	t.Helper()
	initialized := session.InitializeResult()
	if initialized == nil || initialized.Capabilities == nil || initialized.ServerInfo == nil {
		t.Fatal("server did not return complete discovery metadata")
	}
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if listed.NextCursor != "" {
		t.Fatalf("tools/list unexpectedly paginated with cursor %q", listed.NextCursor)
	}
	if got, want := len(listed.Tools), expectedToolCount; got != want {
		t.Fatalf("tool count = %d, want %d", got, want)
	}
	if !sort.SliceIsSorted(listed.Tools, func(i, j int) bool { return listed.Tools[i].Name < listed.Tools[j].Name }) {
		t.Fatal("tools/list is not sorted by stable tool name")
	}
	for _, tool := range listed.Tools {
		if tool.Description == "" || tool.InputSchema == nil || tool.OutputSchema == nil || tool.Annotations == nil {
			t.Fatalf("incomplete manifest entry for %q: title=%q description=%q input=%T output=%T annotations=%#v", tool.Name, tool.Title, tool.Description, tool.InputSchema, tool.OutputSchema, tool.Annotations)
		}
	}
	tools := make([]manifestTool, 0, len(listed.Tools))
	for _, tool := range listed.Tools {
		tools = append(tools, manifestTool{
			Meta: tool.GetMeta(), Name: tool.Name, Title: tool.Title, Description: tool.Description, Icons: tool.Icons,
			InputSchema: tool.InputSchema, OutputSchema: tool.OutputSchema, Annotations: tool.Annotations,
		})
	}

	results := map[string]any{}
	for name, arguments := range representativeCalls() {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
		if err != nil {
			t.Fatalf("call %s: %v", name, err)
		}
		if result.IsError {
			t.Fatalf("call %s returned tool error: %#v", name, result.Content)
		}
		tool := findTool(t, listed.Tools, name)
		validateStructuredResult(t, tool, result)
		results[name] = normalizeCallToolResult(t, result)
	}

	errorSession, cleanup := connectError(t)
	defer cleanup()
	failure, err := errorSession.CallTool(ctx, &mcp.CallToolParams{Name: GetStatusToolName, Arguments: map[string]any{"device": "fixture"}})
	if err != nil {
		t.Fatalf("operational failure became a protocol error: %v", err)
	}
	if !failure.IsError || failure.StructuredContent != nil {
		t.Fatalf("operational error envelope = %#v", failure)
	}
	if len(failure.Content) != 1 {
		t.Fatalf("operational error content = %#v, want one stable JSON block", failure.Content)
	}
	text, ok := failure.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("operational error content = %T, want text", failure.Content[0])
	}
	var stableError decodedToolError
	if err := json.Unmarshal([]byte(text.Text), &stableError); err != nil || stableError.Version != toolErrorVersion || stableError.Code == "" || stableError.Outcome == "" {
		t.Fatalf("operational error is not the versioned taxonomy: %q: %v", text.Text, err)
	}
	results["operational_error"] = normalizeCallToolResult(t, failure)

	_, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "jetkvm_missing_fixture", Arguments: map[string]any{}})
	var wireError *jsonrpc.Error
	if !errors.As(err, &wireError) || wireError.Code != jsonrpc.CodeInvalidParams {
		t.Fatalf("unknown tool error = %#v, want JSON-RPC invalid params", err)
	}
	results["protocol_error"] = map[string]any{"code": wireError.Code, "category": "invalid_params"}

	snapshot := manifestSnapshot{
		ProtocolVersion:    initialized.ProtocolVersion,
		ClientCapabilities: discovery.ClientCapabilities,
		ServerInfo:         initialized.ServerInfo,
		Instructions:       initialized.Instructions,
		InitializeMeta:     initialized.GetMeta(),
		Capabilities:       initialized.Capabilities,
		Discovery:          discovery.Result,
		Tools:              tools,
		Results:            results,
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func connectManifestInMemory(t *testing.T) (*mcp.ClientSession, *observedDiscovery, func()) {
	t.Helper()
	ctx := context.Background()
	observed := new(observedDiscovery)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(&contractDevice{}, "contract-test").Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession := connectManifestClientWithObservation(t, ctx, clientTransport, observed)
	return clientSession, observed, func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}
}

func connectManifestFailingInMemory(t *testing.T) (*mcp.ClientSession, func()) {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(&contractFailingDevice{}, "contract-test").Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession := connectManifestClient(t, ctx, clientTransport)
	return clientSession, func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}
}

func connectManifestMutatedInMemory(t *testing.T) (*mcp.ClientSession, *observedDiscovery, func()) {
	t.Helper()
	ctx := context.Background()
	observed := new(observedDiscovery)
	device := &contractDevice{}
	server := New(device, "contract-test")
	mcp.AddTool(server.sdk, &mcp.Tool{
		Name:        GetStatusToolName,
		Description: "Temporary unreviewed status description used to prove the manifest gate fails.",
		Annotations: annotations(true, false, true),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input deviceInput) (*mcp.CallToolResult, Status, error) {
		if err := validDevice(input.Device); err != nil {
			return nil, Status{}, err
		}
		status, err := device.Status(ctx, input.Device)
		return nil, status, err
	})
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	clientSession := connectManifestClientWithObservation(t, ctx, clientTransport, observed)
	return clientSession, observed, func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	}
}

func connectManifestStdio(t *testing.T) (*mcp.ClientSession, *observedDiscovery, func()) {
	t.Helper()
	ctx := context.Background()
	observed := new(observedDiscovery)
	command := exec.Command(os.Args[0], "-test.run=^TestToolManifestStdioHelperProcess$")
	command.Env = append(os.Environ(), "JETKVM_TOOL_MANIFEST_HELPER=1")
	transport := &mcp.CommandTransport{Command: command, TerminateDuration: time.Second}
	session := connectManifestClientWithObservation(t, ctx, transport, observed)
	return session, observed, func() { _ = session.Close() }
}

func connectManifestFailingStdio(t *testing.T) (*mcp.ClientSession, func()) {
	t.Helper()
	ctx := context.Background()
	command := exec.Command(os.Args[0], "-test.run=^TestToolManifestStdioHelperProcess$")
	command.Env = append(os.Environ(), "JETKVM_TOOL_MANIFEST_HELPER=error")
	transport := &mcp.CommandTransport{Command: command, TerminateDuration: time.Second}
	session := connectManifestClient(t, ctx, transport)
	return session, func() { _ = session.Close() }
}

func TestToolManifestStdioHelperProcess(t *testing.T) {
	mode := os.Getenv("JETKVM_TOOL_MANIFEST_HELPER")
	if mode == "" {
		return
	}
	var device Device = &contractDevice{}
	if mode == "error" {
		device = &contractFailingDevice{}
	}
	if err := New(device, "contract-test").Run(context.Background(), &mcp.StdioTransport{}); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func connectManifestHTTP(t *testing.T) (*mcp.ClientSession, *observedDiscovery, func()) {
	t.Helper()
	observed := new(observedDiscovery)
	httpServer := httptest.NewServer(NewHTTPHandler(New(&contractDevice{}, "contract-test"), ""))
	transport := manifestHTTPTransport(t, httpServer.URL+MCPPath)
	session := connectManifestClientWithObservation(t, context.Background(), transport, observed)
	return session, observed, func() {
		_ = session.Close()
		httpServer.Close()
	}
}

func connectManifestFailingHTTP(t *testing.T) (*mcp.ClientSession, func()) {
	t.Helper()
	httpServer := httptest.NewServer(NewHTTPHandler(New(&contractFailingDevice{}, "contract-test"), ""))
	transport := manifestHTTPTransport(t, httpServer.URL+MCPPath)
	session := connectManifestClient(t, context.Background(), transport)
	return session, func() {
		_ = session.Close()
		httpServer.Close()
	}
}

func manifestHTTPTransport(t *testing.T, endpoint string) *mcp.StreamableClientTransport {
	t.Helper()
	return &mcp.StreamableClientTransport{
		Endpoint: endpoint,
		HTTPClient: &http.Client{Transport: statelessContractRoundTripper{
			base: http.DefaultTransport,
		}},
	}
}

type statelessContractRoundTripper struct {
	base http.RoundTripper
}

func (transport statelessContractRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if sessionID := request.Header.Get("Mcp-Session-Id"); sessionID != "" {
		return nil, fmt.Errorf("stateless HTTP request unexpectedly used session %q", sessionID)
	}
	response, err := transport.base.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	if sessionID := response.Header.Get("Mcp-Session-Id"); sessionID != "" {
		response.Body.Close()
		return nil, fmt.Errorf("stateless HTTP response unexpectedly created session %q", sessionID)
	}
	return response, nil
}

func connectManifestClient(t *testing.T, ctx context.Context, transport mcp.Transport) *mcp.ClientSession {
	return connectManifestClientWithObservation(t, ctx, transport, nil)
}

func connectManifestClientWithObservation(t *testing.T, ctx context.Context, transport mcp.Transport, observed *observedDiscovery) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "manifest-contract-test", Version: "test"}, &mcp.ClientOptions{
		Capabilities: manifestClientCapabilities(),
	})
	if observed != nil {
		client.AddSendingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
			return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
				if method == "server/discover" {
					params, ok := request.GetParams().(*mcp.DiscoverParams)
					if !ok {
						t.Fatalf("server/discover params = %T", request.GetParams())
					}
					capabilities, ok := params.GetMeta()[mcp.MetaKeyClientCapabilities]
					if !ok {
						t.Fatal("server/discover omitted client capabilities")
					}
					data, err := json.Marshal(capabilities)
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(data, &observed.ClientCapabilities); err != nil {
						t.Fatal(err)
					}
				}
				result, err := next(ctx, method, request)
				if method == "server/discover" && err == nil {
					discovery, ok := result.(*mcp.DiscoverResult)
					if !ok {
						t.Fatalf("server/discover result = %T", result)
					}
					data, err := json.Marshal(discovery)
					if err != nil {
						t.Fatal(err)
					}
					var wire struct {
						Meta map[string]any `json:"_meta"`
						discoveryContract
					}
					if err := json.Unmarshal(data, &wire); err != nil {
						t.Fatal(err)
					}
					observed.Result = wire.discoveryContract
					observed.Result.Meta = wire.Meta
				}
				return result, err
			}
		})
	}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func manifestClientCapabilities() *mcp.ClientCapabilities {
	return &mcp.ClientCapabilities{}
}

func jsonEqual(got, want any) bool {
	gotJSON, err := json.Marshal(got)
	if err != nil {
		return false
	}
	wantJSON, err := json.Marshal(want)
	return err == nil && bytes.Equal(gotJSON, wantJSON)
}

func representativeCalls() map[string]map[string]any {
	return map[string]map[string]any{
		ListDevicesToolName:            {},
		GetStatusToolName:              {"device": "fixture"},
		ReleaseSessionToolName:         {"device": "fixture"},
		PressHostPowerButtonToolName:   {"device": "fixture"},
		ForceHostPowerOffToolName:      {"device": "fixture"},
		PressHostResetButtonToolName:   {"device": "fixture"},
		TurnHostDCPowerOnToolName:      {"device": "fixture"},
		TurnHostDCPowerOffToolName:     {"device": "fixture"},
		WakeHostUSBToolName:            {"device": "fixture"},
		WakeHostLANToolName:            {"device": "fixture", "target": "fixture-target"},
		CaptureScreenToolName:          {"device": "fixture", "max_width": 640, "max_height": 480},
		KeyboardToolName:               {"device": "fixture", "operation": string(KeyboardPressKey), "key": "enter"},
		MouseToolName:                  {"device": "fixture", "operation": string(MouseClick), "button": "left"},
		GetVirtualMediaStatusToolName:  {"device": "fixture"},
		MountVirtualMediaURLToolName:   {"device": "fixture", "url": "https://example.invalid/media.iso"},
		MountVirtualMediaFileToolName:  {"device": "fixture", "path": "fixture.iso"},
		UnmountVirtualMediaToolName:    {"device": "fixture"},
		UploadVirtualMediaFileToolName: {"device": "fixture", "path": "fixture.iso"},
	}
}

func findTool(t *testing.T, tools []*mcp.Tool, name string) *mcp.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("missing tool %q", name)
	return nil
}

func validateStructuredResult(t *testing.T, tool *mcp.Tool, result *mcp.CallToolResult) {
	t.Helper()
	data, err := json.Marshal(tool.OutputSchema)
	if err != nil {
		t.Fatalf("marshal %s output schema: %v", tool.Name, err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode %s output schema: %v", tool.Name, err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("resolve %s output schema: %v", tool.Name, err)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal %s structured result: %v", tool.Name, err)
	}
	var instance any
	if err := json.Unmarshal(structured, &instance); err != nil {
		t.Fatalf("decode %s structured result: %v", tool.Name, err)
	}
	if err := resolved.Validate(instance); err != nil {
		t.Fatalf("%s structured result violates advertised output schema: %v", tool.Name, err)
	}
}

func normalizeCallToolResult(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	content := make([]any, 0, len(result.Content))
	for _, block := range result.Content {
		switch value := block.(type) {
		case *mcp.TextContent:
			normalized := map[string]any{
				"type": "text", "meta": value.Meta, "annotations": value.Annotations,
			}
			var decoded any
			if json.Unmarshal([]byte(value.Text), &decoded) == nil {
				normalized["json"] = decoded
			} else {
				normalized["text"] = value.Text
			}
			content = append(content, normalized)
		case *mcp.ImageContent:
			content = append(content, map[string]any{
				"type": "image", "meta": value.Meta, "annotations": value.Annotations,
				"mimeType": value.MIMEType, "sizeBytes": len(value.Data),
			})
		default:
			data, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			var decoded any
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			content = append(content, decoded)
		}
	}
	return map[string]any{
		"meta":              result.Meta,
		"isError":           result.IsError,
		"needsInput":        result.NeedsInput(),
		"inputRequests":     result.InputRequests,
		"requestState":      result.RequestState,
		"structuredContent": result.StructuredContent,
		"content":           content,
	}
}

func readToolManifestFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(toolManifestFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(data)
}

func manifestMatchesFixture(got, want []byte) bool {
	return bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want))
}

func jsonDiff(want, got []byte) string {
	wantPath, err := os.CreateTemp("", "jetkvm-manifest-want-*.json")
	if err != nil {
		return fmt.Sprintf("want=%s\ngot=%s", want, got)
	}
	defer os.Remove(wantPath.Name())
	gotPath, err := os.CreateTemp("", "jetkvm-manifest-got-*.json")
	if err != nil {
		wantPath.Close()
		return fmt.Sprintf("want=%s\ngot=%s", want, got)
	}
	defer os.Remove(gotPath.Name())
	if _, err := wantPath.Write(want); err != nil {
		return err.Error()
	}
	if _, err := gotPath.Write(got); err != nil {
		return err.Error()
	}
	_ = wantPath.Close()
	_ = gotPath.Close()
	output, err := exec.Command("diff", "-u", wantPath.Name(), gotPath.Name()).CombinedOutput()
	if err == nil || len(output) != 0 {
		return strings.TrimSpace(string(output))
	}
	return fmt.Sprintf("want=%s\ngot=%s", want, got)
}

func contractPNG() []byte {
	imageFixture := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	imageFixture.Set(0, 0, color.NRGBA{R: 0x22, G: 0x44, B: 0x66, A: 0xff})
	var output bytes.Buffer
	if err := png.Encode(&output, imageFixture); err != nil {
		panic(err)
	}
	return output.Bytes()
}
