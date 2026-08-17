package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
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

func TestHTTPHandlerRejectsPresentInvalidOriginsBeforeAuthentication(t *testing.T) {
	handler := NewHTTPHandler(New(&recordingDevice{}, "test"), "test-only-token", "https://mcp.example.invalid")
	for _, test := range []struct {
		name    string
		origins []string
	}{
		{name: "empty", origins: []string{""}},
		{name: "duplicate", origins: []string{"https://mcp.example.invalid", "https://foreign.example.invalid"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://backend.invalid"+MCPPath, strings.NewReader(`{}`))
			request.Host = "mcp.example.invalid"
			for _, origin := range test.origins {
				request.Header.Add("Origin", origin)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", response.Code)
			}
			if challenge := response.Header().Get("WWW-Authenticate"); challenge != "" {
				t.Fatalf("Origin rejection exposed bearer challenge %q", challenge)
			}
		})
	}
}

func TestHTTPHandlerRejectsInvalidOriginBeforeMCPHandling(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		called = true
		response.WriteHeader(http.StatusNoContent)
	})
	handler := trustedHostAndOrigin(next, []string{"https://mcp.example.invalid"})
	request := httptest.NewRequest(http.MethodPost, "http://backend.invalid"+MCPPath, nil)
	request.Host = "mcp.example.invalid"
	request.Header.Set("Origin", "null")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || called {
		t.Fatalf("status=%d downstream-called=%v; want 403 false", response.Code, called)
	}
}

func TestHTTPHandlerOriginBearerAndMethodOrdering(t *testing.T) {
	handler := NewHTTPHandler(New(&recordingDevice{}, "test"), "test-only-token", "https://mcp.example.invalid")
	tests := []struct {
		name          string
		origin        string
		authorization string
		wantStatus    int
		wantChallenge string
		wantAllow     string
	}{
		{name: "foreign Origin stops before bearer", origin: "https://browser.example.invalid", wantStatus: http.StatusForbidden},
		{name: "same Origin remains subject to bearer", origin: "https://mcp.example.invalid", wantStatus: http.StatusUnauthorized, wantChallenge: "Bearer"},
		{name: "authenticated same Origin reaches method handling", origin: "https://mcp.example.invalid", authorization: "Bearer test-only-token", wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodOptions, "http://backend.invalid"+MCPPath, nil)
			request.Host = "mcp.example.invalid"
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Access-Control-Request-Method", http.MethodPost)
			if test.authorization != "" {
				request.Header.Set("Authorization", test.authorization)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || response.Header().Get("WWW-Authenticate") != test.wantChallenge || response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf("status=%d challenge=%q Allow=%q; want %d %q %q", response.Code, response.Header().Get("WWW-Authenticate"), response.Header().Get("Allow"), test.wantStatus, test.wantChallenge, test.wantAllow)
			}
			for _, header := range []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Credentials"} {
				if value := response.Header().Get(header); value != "" {
					t.Fatalf("%s = %q, want absent", header, value)
				}
			}
		})
	}
}

func TestHTTPHandlerOriginAndOPTIONSContract(t *testing.T) {
	handler := NewHTTPHandler(New(&recordingDevice{}, "test"), "", "https://mcp.example.invalid")
	discover := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`
	tests := []struct {
		name       string
		method     string
		host       string
		origin     string
		wantStatus int
		wantAllow  string
	}{
		{name: "native public request without Origin", method: http.MethodPost, host: "mcp.example.invalid", wantStatus: http.StatusOK},
		{name: "public same-origin actual request", method: http.MethodPost, host: "mcp.example.invalid", origin: "https://mcp.example.invalid", wantStatus: http.StatusOK},
		{name: "foreign actual request", method: http.MethodPost, host: "mcp.example.invalid", origin: "https://browser.example.invalid", wantStatus: http.StatusForbidden},
		{name: "opaque null actual request", method: http.MethodPost, host: "mcp.example.invalid", origin: "null", wantStatus: http.StatusForbidden},
		{name: "malformed actual request", method: http.MethodPost, host: "mcp.example.invalid", origin: "https://", wantStatus: http.StatusForbidden},
		{name: "same-origin OPTIONS", method: http.MethodOptions, host: "mcp.example.invalid", origin: "https://mcp.example.invalid", wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "foreign preflight", method: http.MethodOptions, host: "mcp.example.invalid", origin: "https://browser.example.invalid", wantStatus: http.StatusForbidden},
		{name: "opaque null preflight", method: http.MethodOptions, host: "mcp.example.invalid", origin: "null", wantStatus: http.StatusForbidden},
		{name: "loopback IPv4 actual request", method: http.MethodPost, host: "127.0.0.1:8080", origin: "http://127.0.0.1:8080", wantStatus: http.StatusOK},
		{name: "loopback IPv6 actual request", method: http.MethodPost, host: "[::1]:8080", origin: "http://[::1]:8080", wantStatus: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "http://backend.invalid"+MCPPath, strings.NewReader(discover))
			request.Host = test.host
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if test.method == http.MethodOptions {
				request.Header.Set("Access-Control-Request-Method", http.MethodPost)
				request.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
			} else {
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Accept", "application/json, text/event-stream")
				request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
				request.Header.Set("Mcp-Method", "server/discover")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus || response.Header().Get("Allow") != test.wantAllow {
				t.Fatalf("status = %d, Allow = %q; want %d, %q", response.Code, response.Header().Get("Allow"), test.wantStatus, test.wantAllow)
			}
			for _, header := range []string{
				"Access-Control-Allow-Origin",
				"Access-Control-Allow-Credentials",
				"Access-Control-Allow-Methods",
				"Access-Control-Allow-Headers",
				"Access-Control-Expose-Headers",
				"Access-Control-Max-Age",
			} {
				if value := response.Header().Get(header); value != "" {
					t.Fatalf("%s = %q, want absent", header, value)
				}
			}
		})
	}
}

func TestSameOriginBrowserHTTPContractIsDocumented(t *testing.T) {
	readme := readRepositoryDocument(t, "README.md")
	configExample := readRepositoryDocument(t, "config.example.yaml")
	productContract := readRepositoryDocument(t, filepath.Join("docs", "product-contract.md"))

	for name, document := range map[string]string{
		"README":           readme,
		"config example":   configExample,
		"product contract": productContract,
	} {
		for _, required := range []string{"same-origin", "not a CORS", "wildcard"} {
			if !strings.Contains(document, required) {
				t.Errorf("%s does not document %q", name, required)
			}
		}
	}
	for _, required := range []string{
		"Native MCP clients may omit `Origin`",
		"separately hosted browser",
		"does not emit CORS response headers",
		"TLS reverse proxy",
		"bearer token",
		"does not implement MCP OAuth",
	} {
		if !strings.Contains(readme, required) {
			t.Errorf("README does not document %q", required)
		}
	}
	for _, required := range []string{"Host/origin admission", "OPTIONS", "HTTP 403", "HTTP 405"} {
		if !strings.Contains(productContract, required) {
			t.Errorf("product contract does not document %q", required)
		}
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

func TestHTTPHandlerMatchesEffectivePublicOrigins(t *testing.T) {
	for _, test := range []struct {
		name       string
		configured string
		host       string
		origin     string
	}{
		{name: "configured explicit HTTPS default request implicit", configured: "https://mcp.example.invalid:443", host: "mcp.example.invalid", origin: "https://mcp.example.invalid"},
		{name: "configured implicit HTTPS default request explicit", configured: "https://mcp.example.invalid", host: "mcp.example.invalid:443", origin: "https://mcp.example.invalid:443"},
		{name: "configured explicit HTTP default request implicit", configured: "http://mcp.example.invalid:80", host: "mcp.example.invalid", origin: "http://mcp.example.invalid"},
		{name: "configured implicit HTTP default request explicit", configured: "http://mcp.example.invalid", host: "mcp.example.invalid:80", origin: "http://mcp.example.invalid:80"},
		{name: "equivalent IPv6 spelling", configured: "https://[2001:0db8:0:0:0:0:0:1]:8443", host: "[2001:db8::1]:8443", origin: "https://[2001:db8::1]:8443"},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHTTPHandler(New(&recordingDevice{}, "test"), "", test.configured)
			request := httptest.NewRequest(http.MethodOptions, "http://backend.invalid"+MCPPath, nil)
			request.Host = test.host
			request.Header.Set("Origin", test.origin)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodPost {
				t.Fatalf("status=%d Allow=%q; want 405 POST", response.Code, response.Header().Get("Allow"))
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

func TestHTTPHandlerRejectsWildcardConfiguredOrigins(t *testing.T) {
	for _, configured := range []string{"https://*", "https://*.example.invalid", "https://mcp.*.invalid"} {
		t.Run(configured, func(t *testing.T) {
			handler := NewHTTPHandler(New(&recordingDevice{}, "test"), "", configured)
			request := httptest.NewRequest(http.MethodPost, "http://backend.invalid"+MCPPath, strings.NewReader(`{}`))
			request.Host = strings.TrimPrefix(configured, "https://")
			request.Header.Set("Origin", configured)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("wildcard configured Origin status = %d, want 403", response.Code)
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

func TestHTTPHandlerIgnoresForwardedHeaders(t *testing.T) {
	handler := NewHTTPHandler(New(&recordingDevice{}, "test"), "", "https://mcp.example.invalid")
	discover := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`
	for _, test := range []struct {
		name           string
		host           string
		origin         string
		forwardedHost  string
		forwardedFor   string
		forwardedProto string
		wantStatus     int
	}{
		{
			name: "forwarded admission cannot replace Host", host: "backend.invalid",
			forwardedHost: "mcp.example.invalid", forwardedProto: "https", forwardedFor: "127.0.0.1",
			wantStatus: http.StatusForbidden,
		},
		{
			name: "forwarded rejection cannot override Host and Origin", host: "mcp.example.invalid", origin: "https://mcp.example.invalid",
			forwardedHost: "foreign.example.invalid", forwardedProto: "http", forwardedFor: "203.0.113.10",
			wantStatus: http.StatusOK,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://backend.invalid"+MCPPath, strings.NewReader(discover))
			request.Host = test.host
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "application/json, text/event-stream")
			request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
			request.Header.Set("Mcp-Method", "server/discover")
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Forwarded", "for="+test.forwardedFor+";host="+test.forwardedHost+";proto="+test.forwardedProto)
			request.Header.Set("X-Forwarded-For", test.forwardedFor)
			request.Header.Set("X-Forwarded-Host", test.forwardedHost)
			request.Header.Set("X-Forwarded-Proto", test.forwardedProto)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", response.Code, test.wantStatus, response.Body.String())
			}
		})
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
