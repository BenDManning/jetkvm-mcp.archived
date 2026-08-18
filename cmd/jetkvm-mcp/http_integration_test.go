package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/BenDManning/jetkvm-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServeHTTPStopsSlowHeadersAfterFiveSeconds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	address, done := startHTTPRuntime(t, ctx, mcpserver.New(nil, "test"))
	defer stopHTTPRuntime(t, cancel, done)

	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	started := time.Now()
	if _, err := io.WriteString(connection, "POST /mcp HTTP/1.1\r\nHost: "+address+"\r\n"); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(started.Add(7 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(connection); err != nil {
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			t.Fatal("server did not stop the incomplete header within seven seconds")
		}
		t.Fatal(err)
	}
	elapsed := time.Since(started)
	if elapsed < 4*time.Second || elapsed > 7*time.Second {
		t.Fatalf("slow header stopped after %s, want approximately five seconds", elapsed)
	}
}

func TestServeHTTPRequiresBearerForNonLoopbackListener(t *testing.T) {
	err := serveHTTP(context.Background(), mcpserver.New(nil, "test"), "0.0.0.0:0", "", nil, io.Discard)
	if err == nil || err.Error() != "a bearer token is required for a non-loopback HTTP address" {
		t.Fatalf("error = %v", err)
	}
}

func TestServeHTTPStopsSlowBodiesAfterFifteenSeconds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	address, done := startHTTPRuntime(t, ctx, mcpserver.New(nil, "test"))
	defer stopHTTPRuntime(t, cancel, done)

	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := io.WriteString(connection, incompleteMCPRequestHeaders(address)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(4 * time.Second)
	bodyStarted := time.Now()
	if _, err := io.WriteString(connection, "\r\n{"); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(bodyStarted.Add(17 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(connection); err != nil {
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			t.Fatal("server did not stop the incomplete body within seventeen seconds")
		}
		t.Fatal(err)
	}
	elapsed := time.Since(bodyStarted)
	if elapsed < 14*time.Second || elapsed > 17*time.Second {
		t.Fatalf("slow body stopped after %s after four seconds spent on headers, want approximately fifteen seconds", elapsed)
	}
}

func TestServeHTTPClosesRejectedRequestsWithUnreadBodies(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	address, done := startHTTPRuntime(t, ctx, mcpserver.New(nil, "test"))
	defer stopHTTPRuntime(t, cancel, done)

	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	started := time.Now()
	if _, err := io.WriteString(connection, incompleteMCPRequestHeaders("untrusted.example.invalid")+"\r\n{"); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(started.Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	response, err := io.ReadAll(connection)
	if err != nil {
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			t.Fatal("server tried to drain an unread rejected body without a deadline")
		}
		t.Fatal(err)
	}
	if !strings.Contains(string(response), "403 Forbidden") {
		t.Fatalf("response = %q, want a forbidden rejection", response)
	}
}

func TestServeHTTPClosesIdleConnectionsAfterSixtySeconds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	address, done := startHTTPRuntime(t, ctx, mcpserver.New(nil, "test"))
	defer stopHTTPRuntime(t, cancel, done)

	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := io.WriteString(connection, "GET /healthz HTTP/1.1\r\nHost: "+address+"\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(connection)
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || string(body) != "ok\n" {
		t.Fatalf("health response status=%d body=%q", response.StatusCode, body)
	}

	idleStarted := time.Now()
	if err := connection.SetReadDeadline(idleStarted.Add(63 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.ReadByte(); err != nil {
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			t.Fatal("server left the connection idle for more than sixty-three seconds")
		}
		if !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}
	} else {
		t.Fatal("idle connection received unexpected data")
	}
	elapsed := time.Since(idleStarted)
	if elapsed < 58*time.Second || elapsed > 63*time.Second {
		t.Fatalf("idle connection closed after %s, want approximately sixty seconds", elapsed)
	}
}

type rootCancellationDevice struct {
	started     chan struct{}
	canceled    chan struct{}
	release     chan struct{}
	releaseOnce sync.Once
}

func (device *rootCancellationDevice) Release() {
	device.releaseOnce.Do(func() { close(device.release) })
}

func (*rootCancellationDevice) ListDevices(context.Context) (mcpserver.DeviceList, error) {
	return mcpserver.DeviceList{Devices: []mcpserver.ConfiguredDevice{{Device: "fixture"}}}, nil
}

func (*rootCancellationDevice) ReleaseSession(context.Context, string) (mcpserver.SessionReleaseResult, error) {
	return mcpserver.SessionReleaseResult{}, nil
}

func (device *rootCancellationDevice) Status(ctx context.Context, _ string) (mcpserver.Status, error) {
	close(device.started)
	select {
	case <-ctx.Done():
		close(device.canceled)
		return mcpserver.Status{}, ctx.Err()
	case <-device.release:
		return mcpserver.Status{}, nil
	}
}

func (*rootCancellationDevice) Power(context.Context, string, mcpserver.PowerAction, string) (mcpserver.PowerResult, error) {
	return mcpserver.PowerResult{}, nil
}

func (*rootCancellationDevice) CaptureScreen(context.Context, string, mcpserver.CaptureRequest) (mcpserver.CaptureResult, error) {
	return mcpserver.CaptureResult{}, nil
}

func (*rootCancellationDevice) Keyboard(context.Context, string, mcpserver.KeyboardRequest) (mcpserver.KeyboardResult, error) {
	return mcpserver.KeyboardResult{}, nil
}

func (*rootCancellationDevice) Mouse(context.Context, string, mcpserver.MouseRequest) (mcpserver.MouseResult, error) {
	return mcpserver.MouseResult{}, nil
}

func (*rootCancellationDevice) VirtualMedia(context.Context, string, mcpserver.VirtualMediaRequest) (mcpserver.VirtualMediaResult, error) {
	return mcpserver.VirtualMediaResult{}, nil
}

func TestServeHTTPProcessContextCancelsActiveRequests(t *testing.T) {
	device := &rootCancellationDevice{started: make(chan struct{}), canceled: make(chan struct{}), release: make(chan struct{})}
	defer device.Release()
	ctx, cancel := context.WithCancel(context.Background())
	address, done := startHTTPRuntime(t, ctx, mcpserver.New(device, "test"))
	defer stopHTTPRuntime(t, cancel, done)

	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "http-root-cancellation-test", Version: "test"}, nil).Connect(
		context.Background(),
		&mcp.StreamableClientTransport{Endpoint: "http://" + address + mcpserver.MCPPath},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	callDone := make(chan struct{})
	go func() {
		defer close(callDone)
		_, _ = clientSession.CallTool(context.Background(), &mcp.CallToolParams{
			Name: mcpserver.GetStatusToolName, Arguments: map[string]any{"device": "fixture"},
		})
	}()
	select {
	case <-device.started:
	case <-time.After(time.Second):
		t.Fatal("tool handler did not start")
	}

	cancel()
	select {
	case <-device.canceled:
	case <-time.After(time.Second):
		device.Release()
		t.Fatal("process context cancellation did not reach the active HTTP request")
	}
	select {
	case <-callDone:
	case <-time.After(time.Second):
		t.Fatal("canceled HTTP tool call did not stop")
	}
}

func TestServeHTTPForceClosesAfterFiveSecondDrain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	address, done := startHTTPRuntime(t, ctx, mcpserver.New(nil, "test"))
	stopped := false
	defer func() {
		if !stopped {
			stopHTTPRuntime(t, cancel, done)
		}
	}()

	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	writeIncompleteMCPRequest(t, connection, address)
	time.Sleep(50 * time.Millisecond)

	started := time.Now()
	cancel()
	select {
	case err := <-done:
		stopped = true
		if err != nil {
			t.Fatalf("forced shutdown error = %v", err)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("HTTP runtime did not force-close after the drain budget")
	}
	elapsed := time.Since(started)
	if elapsed < 4*time.Second || elapsed > 7*time.Second {
		t.Fatalf("forced shutdown completed after %s, want approximately five seconds", elapsed)
	}
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(connection); err != nil {
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			t.Fatal("forced shutdown left the active connection open")
		}
		t.Fatal(err)
	}
}

func TestHTTPProcessHealthSignalsAndStdoutContract(t *testing.T) {
	installFixtureFFmpeg(t)
	binary := filepath.Join(t.TempDir(), "jetkvm-mcp")
	build := exec.Command("go", "build", "-trimpath", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build server binary: %v: %s", err, output)
	}
	configPath := writeStdioFixtureConfig(t)

	for _, test := range []struct {
		name   string
		signal os.Signal
	}{
		{name: "SIGINT", signal: os.Interrupt},
		{name: "SIGTERM", signal: syscall.SIGTERM},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command(binary, "--config", configPath, "--http", "127.0.0.1:0")
			var stdout strings.Builder
			command.Stdout = &stdout
			stderr, err := command.StderrPipe()
			if err != nil {
				t.Fatal(err)
			}
			addressResult := make(chan string, 1)
			stderrResult := make(chan string, 1)
			go func() {
				scanner := bufio.NewScanner(stderr)
				var output strings.Builder
				for scanner.Scan() {
					line := scanner.Text()
					output.WriteString(line)
					output.WriteByte('\n')
					const prefix = "jetkvm-mcp: serving MCP Streamable HTTP on "
					if strings.HasPrefix(line, prefix) && strings.HasSuffix(line, mcpserver.MCPPath) {
						select {
						case addressResult <- strings.TrimSuffix(strings.TrimPrefix(line, prefix), mcpserver.MCPPath):
						default:
						}
					}
				}
				stderrResult <- output.String()
			}()
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if command.ProcessState == nil {
					_ = command.Process.Kill()
					_ = command.Wait()
				}
			}()
			var address string
			select {
			case address = <-addressResult:
			case <-time.After(5 * time.Second):
				_ = command.Process.Kill()
				_ = command.Wait()
				t.Fatal("HTTP process did not start")
			}

			for _, check := range []struct {
				path       string
				wantStatus int
				wantBody   string
			}{
				{path: mcpserver.HealthPath, wantStatus: http.StatusOK, wantBody: "ok\n"},
				{path: "/readyz", wantStatus: http.StatusNotFound, wantBody: "404 page not found\n"},
			} {
				response, err := http.Get("http://" + address + check.path)
				if err != nil {
					t.Fatal(err)
				}
				body, readErr := io.ReadAll(response.Body)
				response.Body.Close()
				if readErr != nil {
					t.Fatal(readErr)
				}
				if response.StatusCode != check.wantStatus || string(body) != check.wantBody {
					t.Fatalf("GET %s: status=%d body=%q", check.path, response.StatusCode, body)
				}
			}

			if err := command.Process.Signal(test.signal); err != nil {
				t.Fatal(err)
			}
			wait := make(chan error, 1)
			go func() { wait <- command.Wait() }()
			select {
			case err := <-wait:
				if err != nil {
					t.Fatalf("process exit after %s: %v", test.name, err)
				}
			case <-time.After(7 * time.Second):
				_ = command.Process.Kill()
				<-wait
				t.Fatalf("process did not exit after %s", test.name)
			}
			processStderr := <-stderrResult
			if stdout.Len() != 0 {
				t.Fatalf("HTTP process polluted stdout: %q", stdout.String())
			}
			if !strings.Contains(processStderr, "serving MCP Streamable HTTP") {
				t.Fatalf("stderr = %q", processStderr)
			}
		})
	}
}

func startHTTPRuntime(t *testing.T, ctx context.Context, server *mcpserver.Server) (string, <-chan error) {
	t.Helper()
	stderrReader, stderrWriter := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- serveHTTP(ctx, server, "127.0.0.1:0", "", nil, stderrWriter)
		_ = stderrWriter.Close()
	}()
	line, err := bufio.NewReader(stderrReader).ReadString('\n')
	_ = stderrReader.Close()
	if err != nil {
		t.Fatalf("read serving notice: %v", err)
	}
	prefix := "jetkvm-mcp: serving MCP Streamable HTTP on "
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(strings.TrimSpace(line), mcpserver.MCPPath) {
		t.Fatalf("serving notice = %q", line)
	}
	return strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(line, prefix)), mcpserver.MCPPath), done
}

func stopHTTPRuntime(t *testing.T, cancel context.CancelFunc, done <-chan error) {
	t.Helper()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("HTTP runtime did not stop")
	}
}

func writeIncompleteMCPRequest(t *testing.T, connection net.Conn, address string) {
	t.Helper()
	if _, err := io.WriteString(connection, incompleteMCPRequestHeaders(address)+"\r\n{"); err != nil {
		t.Fatal(err)
	}
}

func incompleteMCPRequestHeaders(address string) string {
	return "POST /mcp HTTP/1.1\r\n" +
		"Host: " + address + "\r\n" +
		"Content-Type: application/json\r\n" +
		"Accept: application/json, text/event-stream\r\n" +
		"Mcp-Protocol-Version: 2026-07-28\r\n" +
		"Mcp-Method: server/discover\r\n" +
		"Content-Length: 128\r\n"
}
