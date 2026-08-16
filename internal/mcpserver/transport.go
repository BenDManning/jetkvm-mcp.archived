package mcpserver

import (
	"crypto/sha256"
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	"github.com/BenDManning/jetkvm-mcp/internal/httporigin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	DefaultHTTPAddress = "127.0.0.1:8080"
	MCPPath            = "/mcp"
	HealthPath         = "/healthz"
)

// NewHTTPHandler exposes the server over stateless MCP Streamable HTTP. An
// empty bearer token deliberately means no application-level authentication.
func NewHTTPHandler(server *Server, bearerToken string, allowedOrigins ...string) http.Handler {
	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server.sdk
	}, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
		DisableLocalhostProtection:   true, // trustedHostAndOrigin permits only loopback or explicitly configured public Hosts.
	})

	mcpHandler := postOnly(requireSupportedHTTPProtocol(streamable))
	if bearerToken != "" {
		mcpHandler = requireBearer(mcpHandler, bearerToken)
	}
	mcpHandler = trustedHostAndOrigin(mcpHandler, allowedOrigins)

	mux := http.NewServeMux()
	mux.Handle(MCPPath, mcpHandler)
	mux.HandleFunc(HealthPath, func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.Header().Set("Allow", http.MethodGet)
			http.Error(response, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ok\n"))
	})
	return mux
}

func postOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			response.Header().Set("Allow", http.MethodPost)
			http.Error(response, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func trustedHostAndOrigin(next http.Handler, configured []string) http.Handler {
	allowedOrigins := make(map[string]struct{}, len(configured))
	configuredOrigins := make([]httporigin.Origin, 0, len(configured))
	for _, value := range configured {
		origin, err := httporigin.ParseEffective(value)
		if err != nil || strings.Contains(origin.Host, "*") {
			continue
		}
		if _, duplicate := allowedOrigins[origin.Value]; duplicate {
			continue
		}
		allowedOrigins[origin.Value] = struct{}{}
		configuredOrigins = append(configuredOrigins, origin)
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestHost, err := httporigin.ParseAuthority(request.Host)
		if err != nil {
			http.Error(response, "forbidden host", http.StatusForbidden)
			return
		}
		loopback := loopbackHTTPHost(requestHost)
		hostAllowed := loopback
		if !hostAllowed {
			for _, configuredOrigin := range configuredOrigins {
				requestOrigin, parseErr := httporigin.ParseEffective(configuredOrigin.Scheme + "://" + requestHost)
				if parseErr == nil && requestOrigin.Host == configuredOrigin.Host {
					hostAllowed = true
					break
				}
			}
		}
		if !hostAllowed {
			http.Error(response, "forbidden host", http.StatusForbidden)
			return
		}
		originHeaders := request.Header.Values("Origin")
		if len(originHeaders) == 0 {
			next.ServeHTTP(response, request)
			return
		}
		if len(originHeaders) != 1 {
			http.Error(response, "forbidden origin", http.StatusForbidden)
			return
		}
		origin := strings.TrimSpace(originHeaders[0])
		parsedOrigin, err := httporigin.ParseEffective(origin)
		requestOrigin, requestOriginErr := httporigin.ParseEffective(parsedOrigin.Scheme + "://" + requestHost)
		if err != nil || requestOriginErr != nil || parsedOrigin.Host != requestOrigin.Host {
			http.Error(response, "forbidden origin", http.StatusForbidden)
			return
		}
		if loopback {
			requestScheme := "http"
			if request.TLS != nil {
				requestScheme = "https"
			}
			if parsedOrigin.Scheme != requestScheme {
				http.Error(response, "forbidden origin", http.StatusForbidden)
				return
			}
		} else {
			if _, allowed := allowedOrigins[parsedOrigin.Value]; !allowed {
				http.Error(response, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(response, request)
	})
}

func loopbackHTTPHost(hostPort string) bool {
	host := hostPort
	if parsedHost, _, err := net.SplitHostPort(hostPort); err == nil {
		host = parsedHost
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

func requireBearer(next http.Handler, token string) http.Handler {
	want := sha256.Sum256([]byte(token))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		authorization := request.Header.Get("Authorization")
		provided := ""
		if scheme, value, found := strings.Cut(authorization, " "); found && scheme == "Bearer" {
			provided = value
		}
		got := sha256.Sum256([]byte(provided))
		if subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
			response.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(response, request)
	})
}
