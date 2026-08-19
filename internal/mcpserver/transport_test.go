package mcpserver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHTTPHandlerUsesStatelessStreamableHTTP(t *testing.T) {
	if DefaultHTTPAddress != "127.0.0.1:8080" {
		t.Fatalf("default HTTP address = %q", DefaultHTTPAddress)
	}

	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), ""))
	defer httpServer.Close()

	health, err := http.Get(httpServer.URL + HealthPath)
	if err != nil {
		t.Fatal(err)
	}
	health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", health.StatusCode)
	}

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		request, err := http.NewRequest(method, httpServer.URL+MCPPath, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusMethodNotAllowed || response.Header.Get("Allow") != http.MethodPost || response.Header.Get("Mcp-Session-Id") != "" {
			t.Fatalf("%s status=%d allow=%q session=%q", method, response.StatusCode, response.Header.Get("Allow"), response.Header.Get("Mcp-Session-Id"))
		}
	}

	discover := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`
	request, err := http.NewRequest(http.MethodPost, httpServer.URL+MCPPath, strings.NewReader(discover))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "server/discover")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("Mcp-Session-Id") != "" || !strings.Contains(string(body), `"2026-07-28"`) {
		t.Fatalf("discover status=%d session=%q body=%s", response.StatusCode, response.Header.Get("Mcp-Session-Id"), body)
	}
}

func TestHTTPHandlerOptionalBearerToken(t *testing.T) {
	const token = "test-only-token"
	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), token))
	defer httpServer.Close()

	request, err := http.NewRequest(http.MethodPost, httpServer.URL+MCPPath, strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized || response.Header.Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("unauthenticated status=%d challenge=%q", response.StatusCode, response.Header.Get("WWW-Authenticate"))
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "transport-test", Version: "test"}, nil)
	clientSession, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL + MCPPath,
		HTTPClient: &http.Client{Transport: bearerRoundTripper{
			token: token,
			base:  http.DefaultTransport,
		}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	listed, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	foundStatus := false
	for _, tool := range listed.Tools {
		foundStatus = foundStatus || tool.Name == GetStatusToolName
	}
	if !foundStatus {
		t.Fatal("authenticated tools/list omitted status tool")
	}
}

func TestHTTPHandlerRequiresOneSyntacticallyValidBearerCredential(t *testing.T) {
	discover := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`
	for _, test := range []struct {
		name          string
		token         string
		authorization []string
		wantStatus    int
	}{
		{name: "canonical", token: "test-only-token", authorization: []string{"Bearer test-only-token"}, wantStatus: http.StatusOK},
		{name: "case insensitive scheme", token: "test-only-token", authorization: []string{"bearer test-only-token"}, wantStatus: http.StatusOK},
		{name: "multiple scheme separators", token: "test-only-token", authorization: []string{"Bearer   test-only-token"}, wantStatus: http.StatusOK},
		{name: "token68 characters", token: "abc-._~+/==", authorization: []string{"Bearer abc-._~+/=="}, wantStatus: http.StatusOK},
		{name: "missing", token: "test-only-token", wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", token: "test-only-token", authorization: []string{"Basic test-only-token"}, wantStatus: http.StatusUnauthorized},
		{name: "missing credential", token: "test-only-token", authorization: []string{"Bearer"}, wantStatus: http.StatusUnauthorized},
		{name: "embedded whitespace", token: "test only token", authorization: []string{"Bearer test only token"}, wantStatus: http.StatusUnauthorized},
		{name: "padding without data", token: "=", authorization: []string{"Bearer ="}, wantStatus: http.StatusUnauthorized},
		{name: "padding before data", token: "test=token", authorization: []string{"Bearer test=token"}, wantStatus: http.StatusUnauthorized},
		{name: "comma joined credentials", token: "test-only-token", authorization: []string{"Bearer test-only-token, Bearer test-only-token"}, wantStatus: http.StatusUnauthorized},
		{name: "duplicate headers", token: "test-only-token", authorization: []string{"Bearer test-only-token", "Bearer test-only-token"}, wantStatus: http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHTTPHandler(New(&recordingDevice{}, "test"), test.token)
			request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1"+MCPPath, strings.NewReader(discover))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "application/json, text/event-stream")
			request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
			request.Header.Set("Mcp-Method", "server/discover")
			for _, value := range test.authorization {
				request.Header.Add("Authorization", value)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", response.Code, test.wantStatus, response.Body.String())
			}
			if response.Code == http.StatusUnauthorized && response.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("challenge = %q, want Bearer", response.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestHTTPHandlerCapsRequestBodiesAtOneMiB(t *testing.T) {
	discover := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`
	for _, test := range []struct {
		name          string
		size          int
		contentLength bool
		wantStatus    int
	}{
		{name: "one MiB with content length", size: 1 << 20, contentLength: true, wantStatus: http.StatusOK},
		{name: "over one MiB with content length", size: 1<<20 + 1, contentLength: true, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "one MiB chunked", size: 1 << 20, wantStatus: http.StatusOK},
		{name: "over one MiB chunked", size: 1<<20 + 1, wantStatus: http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := discover + strings.Repeat(" ", test.size-len(discover))
			request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1"+MCPPath, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "application/json, text/event-stream")
			request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
			request.Header.Set("Mcp-Method", "server/discover")
			if !test.contentLength {
				request.ContentLength = -1
			}
			response := httptest.NewRecorder()

			NewHTTPHandler(New(&recordingDevice{}, "test"), "").ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}

type bearerRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (transport bearerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+transport.token)
	return transport.base.RoundTrip(clone)
}
