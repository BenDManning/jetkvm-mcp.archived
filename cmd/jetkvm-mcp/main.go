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
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/config"
	"github.com/BenDManning/jetkvm-mcp/internal/jetkvm"
	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
	"github.com/BenDManning/jetkvm-mcp/internal/telemetry"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version string

type commandKind uint8

const (
	commandServe commandKind = iota
	commandDebugRPC
	commandVersion
	commandHelp
	commandConfigValidate
	commandConfigHelp
	commandDebugHelp
)

const rootHelp = `Usage:
  jetkvm-mcp --config FILE [--http HOST:PORT]
  jetkvm-mcp --version
  jetkvm-mcp config validate --config FILE
  jetkvm-mcp debug rpc --config FILE --device NAME --method METHOD [--params JSON] [--unsafe-acknowledge-risk]

Unreviewed RPC methods require --unsafe-acknowledge-risk and may mutate hardware
or boot/storage state and may return sensitive raw firmware data.
`

const configHelp = `Usage:
  jetkvm-mcp config validate --config FILE
`

const debugHelp = `Usage:
  jetkvm-mcp debug rpc --config FILE --device NAME --method METHOD [--params JSON] [--unsafe-acknowledge-risk]

Safe by default: ping, getLocalVersion, getActiveExtension.
All other methods require --unsafe-acknowledge-risk and may mutate hardware or
boot/storage state and may return sensitive raw firmware data.
`

type commandOptions struct {
	kind                    commandKind
	configPath              string
	httpAddress             string
	debugDevice             string
	debugMethod             string
	debugParams             json.RawMessage
	debugUnsafeAcknowledged bool
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
		_, err := fmt.Fprintf(stdout, "jetkvm-mcp %s\n", reportedVersion())
		return err
	}
	if options.kind == commandHelp {
		_, err := io.WriteString(stdout, rootHelp)
		return err
	}
	if options.kind == commandConfigHelp {
		_, err := io.WriteString(stdout, configHelp)
		return err
	}
	if options.kind == commandDebugHelp {
		_, err := io.WriteString(stdout, debugHelp)
		return err
	}
	loaded, err := config.Load(options.configPath, lookup)
	if err != nil {
		return err
	}
	provider := jetkvm.NewWebRTCProvider(jetkvm.WebRTCProviderOptions{})
	if options.kind == commandConfigValidate {
		if _, err := jetkvm.NewManager(loaded.Devices, provider, jetkvm.WithLimits(loaded.Limits)); err != nil {
			return err
		}
		_, err := io.WriteString(stdout, "configuration valid\n")
		return err
	}
	if options.kind == commandDebugRPC {
		manager, err := jetkvm.NewManager(loaded.Devices, provider, jetkvm.WithLimits(loaded.Limits))
		if err != nil {
			return err
		}
		recorder := telemetry.New(stderr)
		operationCtx, span := recorder.Start(ctx, telemetry.TransportStdio, telemetry.OperationDebugRPC)
		result, err := manager.DebugRPC(operationCtx, options.debugDevice, options.debugMethod, options.debugParams, options.debugUnsafeAcknowledged)
		code, outcome := commandTelemetryResult(err)
		span.Record(telemetry.StageTool, code, outcome)
		finishTelemetry(recorder, telemetry.TransportStdio, err)
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
	manager, err := jetkvm.NewManager(loaded.Devices, provider, jetkvm.WithDecoder(decoder), jetkvm.WithLimits(loaded.Limits))
	if err != nil {
		return err
	}
	transport := telemetry.TransportStdio
	if options.httpAddress != "" {
		transport = telemetry.TransportHTTP
	}
	recorder := telemetry.New(stderr)
	server := mcpserver.NewWithTelemetry(manager, reportedVersion(), recorder, transport)
	var serveErr error
	if options.httpAddress == "" {
		fmt.Fprintln(stderr, "jetkvm-mcp: serving MCP over stdio")
		serveErr = server.Run(ctx, &mcp.IOTransport{Reader: io.NopCloser(stdin), Writer: nopWriteCloser{stdout}})
	} else {
		serveErr = serveHTTP(ctx, server, options.httpAddress, loaded.HTTPBearerToken, loaded.HTTPAllowedOrigins, stderr)
	}
	finishTelemetry(recorder, transport, serveErr)
	return serveErr
}

type commandClassifiedError interface {
	error
	ToolErrorCode() string
	ToolErrorOutcome() string
}

func commandTelemetryResult(err error) (string, string) {
	if err == nil {
		return telemetry.CodeSuccess, telemetry.OutcomeSucceeded
	}
	var classified commandClassifiedError
	if errors.As(err, &classified) {
		return classified.ToolErrorCode(), classified.ToolErrorOutcome()
	}
	if errors.Is(err, context.Canceled) {
		return "canceled", telemetry.OutcomeFailed
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", telemetry.OutcomeFailed
	}
	return "operation_failed", telemetry.OutcomeFailed
}

func finishTelemetry(recorder *telemetry.Recorder, transport string, runErr error) {
	_, span := recorder.Start(context.Background(), transport, telemetry.OperationLifecycle)
	code, outcome := telemetry.CodeSuccess, telemetry.OutcomeSucceeded
	if errors.Is(runErr, context.Canceled) {
		code, outcome = "canceled", telemetry.OutcomeFailed
	} else if errors.Is(runErr, context.DeadlineExceeded) {
		code, outcome = "timeout", telemetry.OutcomeFailed
	} else if runErr != nil && !errors.Is(runErr, io.EOF) {
		code, outcome = "operation_failed", telemetry.OutcomeFailed
	}
	span.Record(telemetry.StageShutdown, code, outcome)
	flushCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = recorder.Close(flushCtx)
}

func parseArgs(args []string) (commandOptions, error) {
	if len(args) >= 1 && args[0] == "config" {
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			return commandOptions{kind: commandConfigHelp}, nil
		}
		if len(args) < 2 || args[1] != "validate" {
			return commandOptions{}, errors.New("usage: jetkvm-mcp config validate --config FILE")
		}
		flags := flag.NewFlagSet("config validate", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		configPath := flags.String("config", "", "configuration file")
		if err := flags.Parse(args[2:]); errors.Is(err, flag.ErrHelp) {
			return commandOptions{kind: commandConfigHelp}, nil
		} else if err != nil {
			return commandOptions{}, actionableFlagError(err)
		}
		if flags.NArg() != 0 || *configPath == "" {
			return commandOptions{}, errors.New("usage: jetkvm-mcp config validate --config FILE")
		}
		return commandOptions{kind: commandConfigValidate, configPath: *configPath}, nil
	}
	if len(args) >= 1 && args[0] == "debug" {
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			return commandOptions{kind: commandDebugHelp}, nil
		}
		if len(args) < 2 || args[1] != "rpc" {
			return commandOptions{}, errors.New("usage: jetkvm-mcp debug rpc --config FILE --device NAME --method METHOD [--params JSON] [--unsafe-acknowledge-risk]")
		}
		flags := flag.NewFlagSet("debug rpc", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		configPath := flags.String("config", "", "configuration file")
		device := flags.String("device", "", "configured device name")
		method := flags.String("method", "", "JetKVM RPC method")
		params := flags.String("params", "{}", "JSON object parameters")
		unsafeAcknowledged := flags.Bool("unsafe-acknowledge-risk", false, "acknowledge mutation and sensitive-output risk for an unreviewed method")
		if err := flags.Parse(args[2:]); errors.Is(err, flag.ErrHelp) {
			return commandOptions{kind: commandDebugHelp}, nil
		} else if err != nil {
			return commandOptions{}, actionableFlagError(err)
		}
		if flags.NArg() != 0 || *configPath == "" || *device == "" || *method == "" {
			return commandOptions{}, errors.New("usage: jetkvm-mcp debug rpc --config FILE --device NAME --method METHOD [--params JSON] [--unsafe-acknowledge-risk]")
		}
		return commandOptions{kind: commandDebugRPC, configPath: *configPath, debugDevice: *device, debugMethod: *method, debugParams: json.RawMessage(*params), debugUnsafeAcknowledged: *unsafeAcknowledged}, nil
	}

	flags := flag.NewFlagSet("jetkvm-mcp", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "configuration file")
	httpAddress := flags.String("http", "", "Streamable HTTP listen address")
	showVersion := flags.Bool("version", false, "print version")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return commandOptions{kind: commandHelp}, nil
	} else if err != nil {
		return commandOptions{}, actionableFlagError(err)
	} else if flags.NArg() != 0 {
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

func serveHTTP(ctx context.Context, mcpServer *mcpserver.Server, address, bearerToken string, allowedOrigins []string, stderr io.Writer) error {
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
	handler := mcpserver.NewHTTPHandler(mcpServer, bearerToken, allowedOrigins...)
	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			// Server.ReadTimeout includes time spent reading headers. Start the
			// independent body-read budget only after header admission.
			controller := http.NewResponseController(response)
			if err := controller.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
				request.Close = true
				response.Header().Set("Connection", "close")
				http.Error(response, "request body deadline unavailable", http.StatusInternalServerError)
				return
			}
			body := &bodyReadDeadline{
				ReadCloser: request.Body,
				request:    request,
				response:   response,
				controller: controller,
				remaining:  request.ContentLength,
				complete:   request.Body == nil || request.Body == http.NoBody || request.ContentLength == 0,
			}
			defer func() {
				if body.complete {
					_ = controller.SetReadDeadline(time.Time{})
				} else {
					body.closeConnection()
				}
			}()
			request.Body = body
			handler.ServeHTTP(response, request)
		}),
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
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
		shutdownErr := httpServer.Shutdown(shutdownCtx)
		cancel()
		if shutdownErr != nil {
			closeErr := httpServer.Close()
			serveErr := <-done
			if closeErr != nil {
				return closeErr
			}
			if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				return serveErr
			}
			if !errors.Is(shutdownErr, context.DeadlineExceeded) {
				return shutdownErr
			}
			return nil
		}
		err := <-done
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// bodyReadDeadline prevents clearing a request's read deadline before either
// the declared body is consumed or the connection is made non-reusable.
type bodyReadDeadline struct {
	io.ReadCloser
	request    *http.Request
	response   http.ResponseWriter
	controller *http.ResponseController
	remaining  int64
	complete   bool
}

func (body *bodyReadDeadline) Read(buffer []byte) (int, error) {
	count, err := body.ReadCloser.Read(buffer)
	if body.remaining > 0 {
		body.remaining -= int64(count)
		body.complete = body.remaining <= 0
	}
	if errors.Is(err, io.EOF) {
		body.complete = true
	} else if err != nil {
		body.complete = false
		body.closeConnection()
	}
	return count, err
}

func (body *bodyReadDeadline) closeConnection() {
	body.request.Close = true
	body.response.Header().Set("Connection", "close")
	_ = body.controller.SetReadDeadline(time.Now())
}

func reportedVersion() string {
	info, ok := debug.ReadBuildInfo()
	return resolveVersion(version, info, ok)
}

func resolveVersion(explicit string, info *debug.BuildInfo, ok bool) string {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		return explicit
	}
	if !ok || info == nil {
		return "dev"
	}
	settings := make(map[string]string, len(info.Settings))
	for _, setting := range info.Settings {
		settings[setting.Key] = strings.TrimSpace(setting.Value)
	}
	revision := settings["vcs.revision"]
	if len(revision) >= 12 && isHex(revision) {
		value := "devel+" + revision[:12]
		if settings["vcs.modified"] == "true" {
			value += ".dirty"
		}
		return value
	}
	if moduleVersion := strings.TrimSpace(info.Main.Version); moduleVersion != "" && moduleVersion != "(devel)" {
		return moduleVersion
	}
	return "devel"
}

func actionableFlagError(err error) error {
	message := err.Error()
	if name, ok := safeFlagName(strings.TrimPrefix(message, "flag provided but not defined: -")); ok && strings.HasPrefix(message, "flag provided but not defined: -") {
		return fmt.Errorf("unknown flag --%s", name)
	}
	if name, ok := safeFlagName(strings.TrimPrefix(message, "flag needs an argument: -")); ok && strings.HasPrefix(message, "flag needs an argument: -") {
		return fmt.Errorf("flag --%s needs an argument", name)
	}
	for _, marker := range []string{" for flag -", " for -"} {
		if index := strings.LastIndex(message, marker); index >= 0 {
			if name, ok := safeFlagName(message[index+len(marker):]); ok {
				return fmt.Errorf("invalid value for flag --%s", name)
			}
		}
	}
	return errors.New("invalid command-line flag")
}

func safeFlagName(value string) (string, bool) {
	if index := strings.IndexByte(value, ':'); index >= 0 {
		value = value[:index]
	}
	name := strings.TrimSpace(value)
	for _, character := range name {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '-' {
			continue
		}
		return "", false
	}
	if name == "" {
		return "", false
	}
	return name, true
}

func isHex(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				if character < 'A' || character > 'F' {
					return false
				}
			}
		}
	}
	return value != ""
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
