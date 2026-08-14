package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/protocolgate"
)

type options struct {
	pinsPath    string
	serverPath  string
	artifactDir string
}

type sourceSummary struct {
	Package string `json:"package"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

type checkSummary struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type gateSummary struct {
	SchemaVersion    int            `json:"schemaVersion"`
	Conformance      sourceSummary  `json:"conformance"`
	Inspector        sourceSummary  `json:"inspector"`
	SpecVersion      string         `json:"specVersion"`
	ExpectedFailures []string       `json:"expectedFailures"`
	Checks           []checkSummary `json:"checks"`
	Outcome          string         `json:"outcome"`
}

type conformanceArtifact struct {
	SchemaVersion        int      `json:"schemaVersion"`
	Scenario             string   `json:"scenario"`
	Status               string   `json:"status"`
	AllowedSkippedChecks []string `json:"allowedSkippedChecks"`
}

type runningServer struct {
	command  *exec.Cmd
	done     chan error
	endpoint string
	stdout   bytes.Buffer
	stderr   bytes.Buffer
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "jetkvm-mcp-protocol-gates: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) (runErr error) {
	options, err := parseOptions(args)
	if err != nil {
		return err
	}
	pins, err := protocolgate.LoadPins(options.pinsPath)
	if err != nil {
		return errors.New("load reviewed MCP gate pins")
	}
	if info, err := os.Stat(options.serverPath); err != nil || info.IsDir() {
		return errors.New("--server-binary must name a built server executable")
	}
	if err := os.MkdirAll(options.artifactDir, 0o755); err != nil {
		return errors.New("create MCP gate artifact directory")
	}
	pinsData, err := os.ReadFile(options.pinsPath)
	if err != nil {
		return errors.New("read reviewed MCP gate pins")
	}
	if err := os.WriteFile(filepath.Join(options.artifactDir, "pins.json"), pinsData, 0o644); err != nil {
		return errors.New("write MCP gate pin artifact")
	}

	summary := gateSummary{
		SchemaVersion:    1,
		Conformance:      sourceSummary{Package: pins.Conformance.Package, Version: pins.Conformance.Version, Commit: pins.Conformance.Commit},
		Inspector:        sourceSummary{Package: pins.Inspector.Package, Version: pins.Inspector.Version, Commit: pins.Inspector.Commit},
		SpecVersion:      pins.Conformance.SpecVersion,
		ExpectedFailures: append([]string{}, pins.Conformance.ExpectedFailures...),
		Outcome:          "failed",
	}
	defer func() {
		if err := writeSummary(options.artifactDir, summary); err != nil && runErr == nil {
			runErr = err
		}
	}()

	temporaryDir, err := os.MkdirTemp("", "jetkvm-mcp-protocol-gates-")
	if err != nil {
		return errors.New("create MCP gate temporary directory")
	}
	defer os.RemoveAll(temporaryDir)
	lockDirectory := filepath.Join(filepath.Dir(options.pinsPath), "npm")
	if err := protocolgate.VerifyNPMLock(filepath.Join(lockDirectory, "package-lock.json"), pins); err != nil {
		return recordFailure(&summary, "tooling/npm-lock", err)
	}
	toolingDirectory := filepath.Join(temporaryDir, "tooling")
	if err := installLockedTooling(ctx, lockDirectory, toolingDirectory); err != nil {
		return recordFailure(&summary, "tooling/npm-ci", err)
	}
	recordPass(&summary, "tooling/npm-lock")
	recordPass(&summary, "tooling/npm-ci")
	recordPass(&summary, "conformance/source-pin")
	recordPass(&summary, "inspector/source-pin")
	conformanceBinary := filepath.Join(toolingDirectory, "node_modules", ".bin", "conformance")
	inspectorBinary := filepath.Join(toolingDirectory, "node_modules", ".bin", "mcp-inspector")
	inventoryOutput, err := runBinary(ctx, conformanceBinary, nil, nil, "list", "--server")
	if err != nil {
		return recordFailure(&summary, "conformance/inventory", err)
	}
	inventory, err := protocolgate.ParseOfficialServerScenarioList(string(inventoryOutput))
	if err == nil {
		err = pins.ValidateScenarioInventory(inventory)
	}
	if err == nil {
		var digest string
		digest, err = protocolgate.OfficialServerScenarioInventoryDigest(string(inventoryOutput))
		if err == nil && digest != pins.Conformance.ScenarioInventorySHA256 {
			err = errors.New("official MCP scenario applicability inventory differs from the reviewed pin")
		}
	}
	if err != nil {
		return recordFailure(&summary, "conformance/inventory", err)
	}
	recordPass(&summary, "conformance/inventory")

	configPath := filepath.Join(temporaryDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("devices:\n  fixture:\n    url: http://jetkvm.example.invalid\n"), 0o600); err != nil {
		return errors.New("write MCP gate fixture configuration")
	}

	server, err := startHTTPServer(ctx, options.serverPath, configPath)
	if err != nil {
		return recordFailure(&summary, "server/http-start", err)
	}
	serverStopped := false
	defer func() {
		if !serverStopped {
			_ = server.stop()
		}
	}()
	recordPass(&summary, "server/http-start")

	conformanceArtifacts := filepath.Join(options.artifactDir, "conformance")
	if err := os.MkdirAll(conformanceArtifacts, 0o755); err != nil {
		return errors.New("create conformance artifact directory")
	}
	privateConformanceOutput := filepath.Join(temporaryDir, "conformance-output")
	if err := os.MkdirAll(privateConformanceOutput, 0o700); err != nil {
		return errors.New("create private conformance output directory")
	}
	for _, scenarioName := range pins.GatedScenarios() {
		id := "conformance/" + scenarioName
		output, err := runConformance(ctx, conformanceBinary, stdout,
			"server", "--url", server.endpoint, "--scenario", scenarioName,
			"--spec-version", pins.Conformance.SpecVersion, "--output-dir", privateConformanceOutput)
		var scenario protocolgate.ScenarioClassification
		if err == nil {
			var ok bool
			scenario, ok = pins.Scenario(scenarioName)
			if !ok {
				err = errors.New("gated scenario is absent from reviewed pins")
			} else {
				err = protocolgate.ValidateConformanceScenarioResult(string(output), scenario.AllowedSkippedChecks)
			}
		}
		if err != nil {
			return recordFailure(&summary, id, err)
		}
		if err := writeConformanceArtifact(conformanceArtifacts, scenarioName, scenario.AllowedSkippedChecks); err != nil {
			return recordFailure(&summary, id, err)
		}
		recordPass(&summary, id)
	}

	inspectorEnvironment := append(os.Environ(),
		"MCP_AUTO_OPEN_ENABLED=false",
		"MCP_STORAGE_DIR="+filepath.Join(temporaryDir, "inspector-storage"),
		"NO_PROXY=127.0.0.1,localhost",
		"no_proxy=127.0.0.1,localhost",
	)
	inspectorResults := map[string]map[string][]byte{"stdio": {}, "http": {}}
	for _, transport := range pins.Inspector.Transports {
		for _, method := range pins.Inspector.Methods {
			id := "inspector/" + transport + "/" + method
			output, err := runInspector(ctx, inspectorBinary, inspectorEnvironment, transport, method, "", options.serverPath, configPath, server.endpoint)
			if err == nil {
				err = protocolgate.ValidateInspectorResult(method, output, pins.Inspector.FixtureSafeTools[0])
			}
			if err == nil {
				inspectorResults[transport][method], err = protocolgate.CanonicalInspectorResult(output)
			}
			if err != nil {
				return recordFailure(&summary, id, err)
			}
			recordPass(&summary, id)
		}
		for _, tool := range pins.Inspector.FixtureSafeTools {
			method := "tools/call:" + tool
			id := "inspector/" + transport + "/" + method
			output, err := runInspector(ctx, inspectorBinary, inspectorEnvironment, transport, "tools/call", tool, options.serverPath, configPath, server.endpoint)
			if err == nil {
				err = protocolgate.ValidateInspectorResult(method, output, tool)
			}
			if err == nil {
				inspectorResults[transport][method], err = protocolgate.CanonicalInspectorResult(output)
			}
			if err != nil {
				return recordFailure(&summary, id, err)
			}
			recordPass(&summary, id)
		}
	}
	for method, stdioResult := range inspectorResults["stdio"] {
		httpResult, ok := inspectorResults["http"][method]
		id := "inspector/parity/" + method
		if !ok || !bytes.Equal(stdioResult, httpResult) {
			return recordFailure(&summary, id, errors.New("Inspector stdio and HTTP results differ"))
		}
		recordPass(&summary, id)
	}

	if err := server.stop(); err != nil {
		return recordFailure(&summary, "server/http-stop", err)
	}
	recordPass(&summary, "server/http-stop")
	serverStopped = true
	if server.stdout.Len() != 0 {
		return recordFailure(&summary, "server/http-stdout-purity", errors.New("HTTP server wrote protocol-external data to stdout"))
	}
	recordPass(&summary, "server/http-stdout-purity")
	summary.Outcome = "passed"
	_, _ = fmt.Fprintln(stderr, "MCP protocol gates passed")
	return nil
}

func parseOptions(args []string) (options, error) {
	flags := flag.NewFlagSet("jetkvm-mcp-protocol-gates", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var result options
	flags.StringVar(&result.pinsPath, "pins", "testdata/mcp-gates/pins.json", "reviewed MCP gate pins")
	flags.StringVar(&result.serverPath, "server-binary", "", "built jetkvm-mcp binary")
	flags.StringVar(&result.artifactDir, "artifacts", "", "artifact output directory")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || result.serverPath == "" || result.artifactDir == "" {
		return options{}, errors.New("usage: jetkvm-mcp-protocol-gates --server-binary FILE --artifacts DIR [--pins FILE]")
	}
	serverPath, err := filepath.Abs(result.serverPath)
	if err != nil {
		return options{}, errors.New("resolve server binary")
	}
	result.serverPath = serverPath
	return result, nil
}

func installLockedTooling(ctx context.Context, sourceDirectory, targetDirectory string) error {
	if err := os.MkdirAll(targetDirectory, 0o755); err != nil {
		return errors.New("create locked MCP tooling directory")
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		data, err := os.ReadFile(filepath.Join(sourceDirectory, name))
		if err != nil {
			return errors.New("read locked MCP tooling manifest")
		}
		if err := os.WriteFile(filepath.Join(targetDirectory, name), data, 0o600); err != nil {
			return errors.New("write locked MCP tooling manifest")
		}
	}
	_, err := runBinaryRejectStderr(ctx, "npm", nil, nil, "ci", "--ignore-scripts", "--no-audit", "--no-fund", "--loglevel=error", "--prefix", targetDirectory)
	return err
}

func runBinary(ctx context.Context, binary string, environment []string, mirror io.Writer, args ...string) ([]byte, error) {
	return runBinaryWithStderrPolicy(ctx, binary, environment, mirror, false, args...)
}

func runBinaryRejectStderr(ctx context.Context, binary string, environment []string, mirror io.Writer, args ...string) ([]byte, error) {
	return runBinaryWithStderrPolicy(ctx, binary, environment, mirror, true, args...)
}

func runBinaryWithStderrPolicy(ctx context.Context, binary string, environment []string, mirror io.Writer, rejectStderr bool, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, binary, args...)
	prepareCommand(command)
	if environment != nil {
		command.Env = environment
	}
	var output, commandError bytes.Buffer
	if mirror != nil {
		command.Stdout = io.MultiWriter(&output, mirror)
	} else {
		command.Stdout = &output
	}
	command.Stderr = &commandError
	if err := command.Run(); err != nil {
		return nil, errors.New("pinned MCP command failed")
	}
	if rejectStderr && commandError.Len() != 0 {
		return nil, errors.New("pinned MCP command emitted unexpected stderr")
	}
	return output.Bytes(), nil
}

func writeConformanceArtifact(directory, scenario string, allowedSkippedChecks []string) error {
	artifact := conformanceArtifact{
		SchemaVersion:        1,
		Scenario:             scenario,
		Status:               "passed",
		AllowedSkippedChecks: append([]string{}, allowedSkippedChecks...),
	}
	encoded, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return errors.New("encode sanitized conformance artifact")
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(directory, scenario+".json"), encoded, 0o644); err != nil {
		return errors.New("write sanitized conformance artifact")
	}
	return nil
}

func runConformance(ctx context.Context, binary string, mirror io.Writer, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, binary, args...)
	prepareCommand(command)
	var output, commandError bytes.Buffer
	if mirror != nil {
		command.Stdout = io.MultiWriter(&output, mirror)
		command.Stderr = io.MultiWriter(&commandError, mirror)
	} else {
		command.Stdout = &output
		command.Stderr = &commandError
	}
	if err := command.Run(); err != nil {
		return nil, errors.New("pinned MCP conformance command failed")
	}
	combined := append(append([]byte(nil), output.Bytes()...), commandError.Bytes()...)
	return combined, nil
}

func runInspector(ctx context.Context, inspectorBinary string, environment []string, transport, method, tool, serverPath, configPath, endpoint string) ([]byte, error) {
	args := []string{"--cli"}
	if transport == "http" {
		args = append(args, "--server-url", endpoint, "--transport", "http", "--method", method)
	} else if transport == "stdio" {
		args = append(args, serverPath, "--config", configPath, "--", "--method", method)
	} else {
		return nil, errors.New("unreviewed Inspector transport")
	}
	if tool != "" {
		args = append(args, "--tool-name", tool, "--tool-args-json", "{}")
	}
	args = append(args, "--format", "json", "--connect-timeout", "5000")
	return runBinary(ctx, inspectorBinary, environment, nil, args...)
}

func startHTTPServer(ctx context.Context, serverPath, configPath string) (*runningServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, errors.New("reserve loopback MCP gate address")
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return nil, errors.New("release loopback MCP gate address")
	}
	server := &runningServer{done: make(chan error, 1), endpoint: "http://" + address + "/mcp"}
	server.command = exec.CommandContext(ctx, serverPath, "--config", configPath, "--http", address)
	prepareCommand(server.command)
	server.command.Stdout = &server.stdout
	server.command.Stderr = &server.stderr
	if err := server.command.Start(); err != nil {
		return nil, errors.New("start fixture HTTP server")
	}
	go func() { server.done <- server.command.Wait() }()

	healthURL := "http://" + address + "/healthz"
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = killCommand(server.command)
			<-server.done
			return nil, errors.New("fixture HTTP server start cancelled")
		case <-deadline.C:
			_ = killCommand(server.command)
			<-server.done
			return nil, errors.New("fixture HTTP server did not become ready")
		case <-ticker.C:
			response, requestErr := client.Get(healthURL)
			if requestErr == nil {
				_ = response.Body.Close()
				if response.StatusCode == http.StatusOK {
					return server, nil
				}
			}
		case <-server.done:
			return nil, errors.New("fixture HTTP server exited before readiness")
		}
	}
}

func (server *runningServer) stop() error {
	if server.command.Process == nil {
		return nil
	}
	if err := interruptCommand(server.command); err != nil {
		return errors.New("stop fixture HTTP server")
	}
	select {
	case err := <-server.done:
		if killErr := killCommand(server.command); killErr != nil {
			return errors.New("kill fixture HTTP server descendants")
		}
		if err != nil {
			return errors.New("fixture HTTP server stopped with an error")
		}
		return nil
	case <-time.After(5 * time.Second):
		_ = killCommand(server.command)
		<-server.done
		return errors.New("fixture HTTP server did not stop promptly")
	}
}

func recordPass(summary *gateSummary, id string) {
	summary.Checks = append(summary.Checks, checkSummary{ID: id, Status: "passed"})
}

func recordFailure(summary *gateSummary, id string, err error) error {
	summary.Checks = append(summary.Checks, checkSummary{ID: id, Status: "failed"})
	return err
}

func writeSummary(directory string, summary gateSummary) error {
	encoded, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return errors.New("encode MCP gate summary")
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(directory, "summary.json"), encoded, 0o644); err != nil {
		return errors.New("write MCP gate summary")
	}
	return nil
}
