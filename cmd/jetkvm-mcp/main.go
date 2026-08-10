package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/config"
	"github.com/BenDManning/jetkvm-mcp/internal/jetkvm"
	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

type commandKind uint8

const (
	commandServe commandKind = iota
	commandDebugRPC
	commandVersion
)

type commandOptions struct {
	kind        commandKind
	configPath  string
	httpAddress string
	debugDevice string
	debugMethod string
	debugParams json.RawMessage
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, os.LookupEnv); err != nil {
		fmt.Fprintf(os.Stderr, "jetkvm-mcp: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, lookup config.LookupEnvironment) error {
	options, err := parseArgs(args)
	if err != nil {
		return err
	}
	if options.kind == commandVersion {
		_, err := fmt.Fprintf(stdout, "jetkvm-mcp %s\n", version)
		return err
	}
	loaded, err := config.Load(options.configPath, lookup)
	if err != nil {
		return err
	}
	provider := jetkvm.NewWebRTCProvider(jetkvm.WebRTCProviderOptions{})
	if options.kind == commandDebugRPC {
		manager, err := jetkvm.NewManager(loaded.Devices, provider)
		if err != nil {
			return err
		}
		result, err := manager.DebugRPC(ctx, options.debugDevice, options.debugMethod, options.debugParams)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(struct {
			Result json.RawMessage `json:"result"`
		}{Result: result})
	}

	decoder, err := jetkvm.NewFFmpegDecoder()
	if err != nil {
		return err
	}
	manager, err := jetkvm.NewManager(loaded.Devices, provider, jetkvm.WithDecoder(decoder))
	if err != nil {
		return err
	}
	server := mcpserver.New(manager, version)
	if options.httpAddress == "" {
		fmt.Fprintln(stderr, "jetkvm-mcp: serving MCP over stdio")
		return server.Run(ctx, &mcp.IOTransport{Reader: io.NopCloser(stdin), Writer: nopWriteCloser{stdout}})
	}
	return serveHTTP(ctx, server, options.httpAddress, loaded.HTTPBearerToken, loaded.HTTPAllowedOrigins, stderr)
}

func parseArgs(args []string) (commandOptions, error) {
	if len(args) >= 1 && args[0] == "debug" {
		if len(args) < 2 || args[1] != "rpc" {
			return commandOptions{}, errors.New("usage: jetkvm-mcp debug rpc --config FILE --device NAME --method METHOD [--params JSON]")
		}
		flags := flag.NewFlagSet("debug rpc", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		configPath := flags.String("config", "", "configuration file")
		device := flags.String("device", "", "configured device name")
		method := flags.String("method", "", "JetKVM RPC method")
		params := flags.String("params", "{}", "JSON object parameters")
		if err := flags.Parse(args[2:]); err != nil || flags.NArg() != 0 || *configPath == "" || *device == "" || *method == "" {
			return commandOptions{}, errors.New("usage: jetkvm-mcp debug rpc --config FILE --device NAME --method METHOD [--params JSON]")
		}
		return commandOptions{kind: commandDebugRPC, configPath: *configPath, debugDevice: *device, debugMethod: *method, debugParams: json.RawMessage(*params)}, nil
	}

	flags := flag.NewFlagSet("jetkvm-mcp", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "configuration file")
	httpAddress := flags.String("http", "", "Streamable HTTP listen address")
	showVersion := flags.Bool("version", false, "print version")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return commandOptions{}, errors.New("usage: jetkvm-mcp --config FILE [--http 127.0.0.1:8080]")
	}
	if *showVersion {
		return commandOptions{kind: commandVersion}, nil
	}
	if *configPath == "" {
		return commandOptions{}, errors.New("--config is required")
	}
	if *httpAddress != "" {
		if _, _, err := net.SplitHostPort(*httpAddress); err != nil {
			return commandOptions{}, errors.New("--http must be a host:port address")
		}
	}
	return commandOptions{kind: commandServe, configPath: *configPath, httpAddress: *httpAddress}, nil
}

func serveHTTP(ctx context.Context, mcpServer *mcp.Server, address, bearerToken string, allowedOrigins []string, stderr io.Writer) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return errors.New("invalid HTTP listen address")
	}
	if !loopbackHost(host) && bearerToken == "" {
		return errors.New("a bearer token is required for a non-loopback HTTP address")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Handler:           mcpserver.NewHTTPHandler(mcpServer, bearerToken, allowedOrigins...),
		ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20,
	}
	fmt.Fprintf(stderr, "jetkvm-mcp: serving MCP Streamable HTTP on %s%s\n", listener.Addr(), mcpserver.MCPPath)
	done := make(chan error, 1)
	go func() { done <- httpServer.Serve(listener) }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		<-done
		return nil
	}
}

func loopbackHost(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
