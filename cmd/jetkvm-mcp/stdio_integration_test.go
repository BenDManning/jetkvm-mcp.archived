package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunStdioDiscoveryUsesStdoutAndLogsToStderr(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("devices:\n  smoke:\n    url: http://jetkvm.example.invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	var stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, []string{"--config", configPath}, stdinReader, stdoutWriter, &stderr, os.LookupEnv)
	}()

	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "server/discover",
		"params": map[string]any{"_meta": map[string]any{
			"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stdinWriter.Write(append(request, '\n')); err != nil {
		t.Fatal(err)
	}
	lineResult := make(chan []byte, 1)
	lineError := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(stdoutReader).ReadBytes('\n')
		if err != nil {
			lineError <- err
			return
		}
		lineResult <- line
	}()
	var line []byte
	select {
	case line = <-lineResult:
	case err := <-lineError:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var response struct {
		Result struct {
			SupportedVersions []string `json:"supportedVersions"`
		} `json:"result"`
	}
	if err := json.Unmarshal(line, &response); err != nil {
		t.Fatalf("stdout is not one JSON response: %q: %v", line, err)
	}
	found := false
	for _, value := range response.Result.SupportedVersions {
		found = found || value == "2026-07-28"
	}
	if !found || !strings.Contains(stderr.String(), "serving MCP over stdio") {
		t.Fatalf("versions=%v stderr=%q", response.Result.SupportedVersions, stderr.String())
	}
	_ = stdinWriter.Close()
	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("stdio runtime did not stop")
	}
}
