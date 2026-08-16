package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const protocolVersionStdioHelperEnvironment = "JETKVM_PROTOCOL_VERSION_STDIO_HELPER"

type rawProtocolResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		SupportedVersions []string `json:"supportedVersions"`
	} `json:"result"`
	Error *struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Supported []string `json:"supported"`
			Requested string   `json:"requested"`
		} `json:"data"`
	} `json:"error"`
}

func TestServerAcceptsOnlyV1ProtocolVersionAcrossTransports(t *testing.T) {
	const supportedVersion = "2026-07-28"
	requests := []struct {
		name      string
		version   string
		supported bool
	}{
		{name: "v1", version: supportedVersion, supported: true},
		{name: "2025_11_25", version: "2025-11-25"},
		{name: "2025_06_18", version: "2025-06-18"},
		{name: "2025_03_26", version: "2025-03-26"},
		{name: "2024_11_05", version: "2024-11-05"},
		{name: "unknown", version: "2099-01-01"},
	}
	transports := []struct {
		name string
		call func(*testing.T, string, []byte) (int, []byte)
	}{
		{name: "stdio", call: rawStdioProtocolCall},
		{name: "stateless_http", call: rawHTTPProtocolCall},
	}

	for _, transport := range transports {
		for _, request := range requests {
			t.Run(transport.name+"/"+request.name, func(t *testing.T) {
				method := "initialize"
				params := map[string]any{
					"protocolVersion": request.version,
					"capabilities":    map[string]any{},
					"clientInfo":      map[string]any{"name": "raw-protocol-test", "version": "test"},
				}
				if request.supported || request.name == "unknown" {
					method = "server/discover"
					params = map[string]any{"_meta": map[string]any{
						"io.modelcontextprotocol/protocolVersion":    request.version,
						"io.modelcontextprotocol/clientCapabilities": map[string]any{},
					}}
				}
				wire, err := json.Marshal(map[string]any{
					"jsonrpc": "2.0", "id": 1, "method": method, "params": params,
				})
				if err != nil {
					t.Fatal(err)
				}
				status, body := transport.call(t, request.version, wire)
				var response rawProtocolResponse
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("decode response %q: %v", body, err)
				}
				if response.JSONRPC != "2.0" || response.ID != 1 {
					t.Fatalf("response envelope = %#v", response)
				}
				if request.supported {
					if (status != 0 && status != http.StatusOK) || response.Error != nil || len(response.Result.SupportedVersions) != 1 || response.Result.SupportedVersions[0] != supportedVersion {
						t.Fatalf("supported response status=%d body=%s", status, body)
					}
					return
				}
				if (status != 0 && status != http.StatusBadRequest) || response.Error == nil || response.Error.Code != mcp.CodeUnsupportedProtocolVersion ||
					response.Error.Message != "unsupported protocol version" || response.Error.Data.Requested != request.version ||
					len(response.Error.Data.Supported) != 1 || response.Error.Data.Supported[0] != supportedVersion {
					t.Fatalf("unsupported response status=%d body=%s", status, body)
				}
			})
		}
	}
}

func rawStdioProtocolCall(t *testing.T, _ string, request []byte) (int, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProtocolVersionStdioHelperProcess$")
	command.Env = append(os.Environ(), protocolVersionStdioHelperEnvironment+"=1")
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := stdin.Write(append(request, '\n')); err != nil {
		t.Fatal(err)
	}
	response, err := bufio.NewReader(stdout).ReadBytes('\n')
	if err != nil {
		t.Fatalf("read response: %v stderr=%q", err, stderr.String())
	}
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("stdio helper: %v stderr=%q", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stdio diagnostics = %q", stderr.String())
	}
	return 0, response
}

func rawHTTPProtocolCall(t *testing.T, version string, body []byte) (int, []byte) {
	t.Helper()
	handler := NewHTTPHandler(New(&recordingDevice{}, "test"), "")
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1"+MCPPath, strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", version)
	if version >= "2026-07-28" {
		request.Header.Set("Mcp-Method", "server/discover")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code, response.Body.Bytes()
}

func TestHTTPProtocolGuardRejectsOversizedRequestBody(t *testing.T) {
	handler := NewHTTPHandler(New(&recordingDevice{}, "test"), "")
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1"+MCPPath, strings.NewReader(strings.Repeat(" ", (1<<20)+1)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", SupportedProtocolVersion)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body=%q", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}
}

func TestProtocolVersionStdioHelperProcess(t *testing.T) {
	if os.Getenv(protocolVersionStdioHelperEnvironment) != "1" {
		return
	}
	if err := New(&recordingDevice{}, "test").Run(context.Background(), &mcp.StdioTransport{}); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}
