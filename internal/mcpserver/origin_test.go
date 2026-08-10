package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPHandlerRejectsForeignOrigins(t *testing.T) {
	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), ""))
	defer httpServer.Close()
	discover := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`
	request, err := http.NewRequest(http.MethodPost, httpServer.URL+MCPPath, strings.NewReader(discover))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "server/discover")
	request.Header.Set("Origin", "https://foreign.example.invalid")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign Origin status = %d, want 403", response.StatusCode)
	}

	request, err = http.NewRequest(http.MethodPost, httpServer.URL+MCPPath, strings.NewReader(discover))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "server/discover")
	request.Header.Set("Origin", httpServer.URL)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("same Origin status = %d, want 200", response.StatusCode)
	}
}

func TestHTTPHandlerRejectsLoopbackSchemeMismatch(t *testing.T) {
	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), ""))
	defer httpServer.Close()
	origin := "https://" + strings.TrimPrefix(httpServer.URL, "http://")
	if status := discoverRequestStatus(t, httpServer.URL, "", origin); status != http.StatusForbidden {
		t.Fatalf("cross-scheme loopback Origin status = %d, want 403", status)
	}
}

func TestHTTPHandlerRejectsMalformedLoopbackHostsWithoutOrigin(t *testing.T) {
	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), ""))
	defer httpServer.Close()
	for _, host := range []string{"localhost:", "127.0.0.1:", "[::1]:", "::1"} {
		t.Run(host, func(t *testing.T) {
			if status := discoverRequestStatus(t, httpServer.URL, host, ""); status != http.StatusForbidden {
				t.Fatalf("malformed loopback Host %q status = %d, want 403", host, status)
			}
		})
	}
}

func TestHTTPHandlerRejectsUnbracketedIPv6HostAndOrigin(t *testing.T) {
	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), ""))
	defer httpServer.Close()
	if status := discoverRequestStatus(t, httpServer.URL, "::1", "http://::1"); status != http.StatusForbidden {
		t.Fatalf("unbracketed IPv6 Host/Origin status = %d, want 403", status)
	}
}

func TestHTTPHandlerAllowsHTTPSLoopbackSameOrigin(t *testing.T) {
	httpServer := httptest.NewTLSServer(NewHTTPHandler(New(&recordingDevice{}, "test"), ""))
	defer httpServer.Close()
	request, err := http.NewRequest(http.MethodPost, httpServer.URL+MCPPath, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", httpServer.URL)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "server/discover")
	response, err := httpServer.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("HTTPS loopback same-Origin status = %d, want 200", response.StatusCode)
	}
}

func TestHTTPHandlerRejectsConfiguredPublicOriginMismatches(t *testing.T) {
	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), "", "https://mcp.example.invalid:8443"))
	defer httpServer.Close()
	for _, test := range []struct {
		name   string
		origin string
	}{
		{name: "scheme", origin: "http://mcp.example.invalid:8443"},
		{name: "port", origin: "https://mcp.example.invalid:9443"},
		{name: "foreign", origin: "https://foreign.example.invalid:8443"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if status := discoverRequestStatus(t, httpServer.URL, "mcp.example.invalid:8443", test.origin); status != http.StatusForbidden {
				t.Fatalf("configured public %s mismatch status = %d, want 403", test.name, status)
			}
		})
	}
}

func TestHTTPHandlerRejectsMalformedConfiguredOrigins(t *testing.T) {
	for _, test := range []struct {
		name       string
		configured string
		host       string
		origin     string
	}{
		{name: "empty query", configured: "https://mcp.example.invalid?", host: "mcp.example.invalid", origin: "https://mcp.example.invalid"},
		{name: "empty fragment", configured: "https://mcp.example.invalid#", host: "mcp.example.invalid", origin: "https://mcp.example.invalid"},
		{name: "invalid port", configured: "https://mcp.example.invalid:99999", host: "mcp.example.invalid:99999", origin: "https://mcp.example.invalid:99999"},
		{name: "empty port", configured: "https://mcp.example.invalid:", host: "mcp.example.invalid:", origin: "https://mcp.example.invalid:"},
	} {
		t.Run(test.name, func(t *testing.T) {
			httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), "", test.configured))
			defer httpServer.Close()
			if status := discoverRequestStatus(t, httpServer.URL, test.host, test.origin); status != http.StatusForbidden {
				t.Fatalf("malformed configured Origin status = %d, want 403", status)
			}
		})
	}
}

func TestHTTPHandlerAllowsSameOriginReverseProxyHost(t *testing.T) {
	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), "", "https://mcp.example.invalid"))
	defer httpServer.Close()
	request, err := http.NewRequest(http.MethodPost, httpServer.URL+MCPPath, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "mcp.example.invalid"
	request.Header.Set("Origin", "https://mcp.example.invalid")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "server/discover")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("reverse-proxy Host status = %d, want 200", response.StatusCode)
	}
}

func TestHTTPHandlerRejectsUnconfiguredMatchingPublicHostAndOrigin(t *testing.T) {
	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), ""))
	defer httpServer.Close()
	request, err := http.NewRequest(http.MethodPost, httpServer.URL+MCPPath, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "attacker.example.invalid"
	request.Header.Set("Origin", "https://attacker.example.invalid")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "server/discover")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("unconfigured public Host/Origin status = %d, want 403", response.StatusCode)
	}
}

func TestHTTPHandlerAllowsNoOriginForConfiguredPublicHost(t *testing.T) {
	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), "", "https://mcp.example.invalid"))
	defer httpServer.Close()
	request, err := http.NewRequest(http.MethodPost, httpServer.URL+MCPPath, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "mcp.example.invalid"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "server/discover")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("configured public Host without Origin status = %d, want 200", response.StatusCode)
	}
}

func TestHTTPHandlerRejectsNoOriginForUnconfiguredPublicHost(t *testing.T) {
	httpServer := httptest.NewServer(NewHTTPHandler(New(&recordingDevice{}, "test"), ""))
	defer httpServer.Close()
	request, err := http.NewRequest(http.MethodPost, httpServer.URL+MCPPath, strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "attacker.example.invalid"
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("unconfigured public Host without Origin status = %d, want 403", response.StatusCode)
	}
}

func discoverRequestStatus(t *testing.T, endpoint, host, origin string) int {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, endpoint+MCPPath, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if host != "" {
		request.Host = host
	}
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "server/discover")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	return response.StatusCode
}
