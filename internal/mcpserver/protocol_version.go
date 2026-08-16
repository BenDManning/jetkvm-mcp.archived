package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SupportedProtocolVersion is the complete JetKVM MCP v1 wire-version set.
const SupportedProtocolVersion = "2026-07-28"

const (
	maxMCPRequestBodyBytes    = int64(1 << 20)
	mcpRequestBodyReadTimeout = 15 * time.Second
)

// Server keeps SDK protocol compatibility from expanding the product's public
// compatibility surface. Feature registration remains owned by the SDK server;
// connections are admitted only for the one product-supported revision.
type Server struct {
	sdk *mcp.Server
}

// Connect creates one protocol-restricted SDK session.
func (server *Server) Connect(ctx context.Context, transport mcp.Transport, options *mcp.ServerSessionOptions) (*mcp.ServerSession, error) {
	return server.sdk.Connect(ctx, protocolVersionTransport{Transport: transport}, options)
}

// Run serves one protocol-restricted session until the peer closes it or ctx is canceled.
func (server *Server) Run(ctx context.Context, transport mcp.Transport) error {
	return server.sdk.Run(ctx, protocolVersionTransport{Transport: transport})
}

type protocolVersionTransport struct {
	mcp.Transport
}

func (transport protocolVersionTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	connection, err := transport.Transport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &protocolVersionConnection{Connection: connection}, nil
}

func (protocolVersionTransport) SupportsProtocolVersion(version string) bool {
	return version == SupportedProtocolVersion
}

type protocolVersionConnection struct {
	mcp.Connection
}

func (connection *protocolVersionConnection) Read(ctx context.Context) (jsonrpc.Message, error) {
	for {
		message, err := connection.Connection.Read(ctx)
		if err != nil {
			return nil, err
		}
		request, ok := message.(*jsonrpc.Request)
		if !ok {
			return message, nil
		}
		wireError := protocolVersionRequestError(request, false, "")
		if wireError == nil {
			return message, nil
		}
		if !request.ID.IsValid() {
			continue
		}
		if err := connection.Connection.Write(ctx, &jsonrpc.Response{ID: request.ID, Error: wireError}); err != nil {
			return nil, err
		}
	}
}

func protocolVersionRequestError(request *jsonrpc.Request, requireHTTPVersion bool, httpVersion string) *jsonrpc.Error {
	requested, hasRequested := requestProtocolVersion(request)
	if request.Method == "initialize" {
		if requested != SupportedProtocolVersion {
			return unsupportedProtocolVersion(requested)
		}
		return &jsonrpc.Error{
			Code:    jsonrpc.CodeMethodNotFound,
			Message: fmt.Sprintf("%q is not supported in protocol version %s", request.Method, SupportedProtocolVersion),
		}
	}
	if requireHTTPVersion && httpVersion != SupportedProtocolVersion {
		return unsupportedProtocolVersion(httpVersion)
	}
	if hasRequested && requested != SupportedProtocolVersion {
		return unsupportedProtocolVersion(requested)
	}
	if requireHTTPVersion && !hasRequested {
		return unsupportedProtocolVersion("")
	}
	return nil
}

func requestProtocolVersion(request *jsonrpc.Request) (string, bool) {
	var params struct {
		ProtocolVersion string         `json:"protocolVersion"`
		Meta            map[string]any `json:"_meta"`
	}
	if len(request.Params) == 0 || json.Unmarshal(request.Params, &params) != nil {
		return "", false
	}
	if request.Method == "initialize" {
		return params.ProtocolVersion, true
	}
	version, ok := params.Meta[mcp.MetaKeyProtocolVersion].(string)
	return version, ok
}

func unsupportedProtocolVersion(requested string) *jsonrpc.Error {
	data, _ := json.Marshal(mcp.UnsupportedProtocolVersionData{
		Supported: []string{SupportedProtocolVersion},
		Requested: requested,
	})
	return &jsonrpc.Error{
		Code:    mcp.CodeUnsupportedProtocolVersion,
		Message: "unsupported protocol version",
		Data:    data,
	}
}

func protocolVersionDiscoveryMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
		result, err := next(ctx, method, request)
		if err != nil || method != "server/discover" {
			return result, err
		}
		discovery, ok := result.(*mcp.DiscoverResult)
		if ok {
			discovery.SupportedVersions = []string{SupportedProtocolVersion}
		}
		return result, nil
	}
}

func requireSupportedHTTPProtocol(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := readMCPRequestBody(response, request)
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(response, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(response, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		request.Body = io.NopCloser(bytes.NewReader(body))
		httpVersion := request.Header.Get("Mcp-Protocol-Version")
		message, decodeErr := jsonrpc.DecodeMessage(body)
		call, ok := message.(*jsonrpc.Request)
		if decodeErr != nil || !ok {
			if httpVersion != SupportedProtocolVersion {
				http.Error(response, "unsupported protocol version", http.StatusBadRequest)
				return
			}
			next.ServeHTTP(response, request)
			return
		}
		wireError := protocolVersionRequestError(call, true, httpVersion)
		if wireError == nil {
			next.ServeHTTP(response, request)
			return
		}
		encoded, err := jsonrpc.EncodeMessage(&jsonrpc.Response{ID: call.ID, Error: wireError})
		if err != nil {
			http.Error(response, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusBadRequest)
		_, _ = response.Write(encoded)
	})
}

func readMCPRequestBody(response http.ResponseWriter, request *http.Request) ([]byte, error) {
	controller := http.NewResponseController(response)
	deadlineSet := controller.SetReadDeadline(time.Now().Add(mcpRequestBodyReadTimeout)) == nil
	if deadlineSet {
		defer controller.SetReadDeadline(time.Time{})
	}
	body := http.MaxBytesReader(response, request.Body, maxMCPRequestBodyBytes)
	defer body.Close()
	return io.ReadAll(body)
}
