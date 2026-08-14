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

func installFixtureFFmpeg(t *testing.T) {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "ffmpeg"), []byte("#!/bin/sh\nexit 127\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestRunStdioMalformedJSONLClosesCleanlyAndFreshProcessRecovers(t *testing.T) {
	installFixtureFFmpeg(t)
	binary := filepath.Join(t.TempDir(), "jetkvm-mcp")
	build := exec.Command("go", "build", "-trimpath", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build server binary: %v: %s", err, output)
	}
	configPath := writeStdioFixtureConfig(t)
	malformedContext, cancelMalformed := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelMalformed()
	malformed := exec.CommandContext(malformedContext, binary, "--config", configPath)
	malformed.Stdin = strings.NewReader("{not-json}\n")
	var malformedStdout, malformedStderr bytes.Buffer
	malformed.Stdout = &malformedStdout
	malformed.Stderr = &malformedStderr
	if err := malformed.Run(); err == nil || errors.Is(malformedContext.Err(), context.DeadlineExceeded) {
		t.Fatalf("malformed process error = %v context=%v", err, malformedContext.Err())
	}
	if malformedStdout.Len() != 0 {
		t.Fatalf("malformed input polluted stdout: %q", malformedStdout.String())
	}
	if !strings.Contains(malformedStderr.String(), "serving MCP over stdio") {
		t.Fatalf("malformed stderr = %q", malformedStderr.String())
	}

	discover, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "server/discover",
		"params": map[string]any{"_meta": map[string]any{
			"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	recoveryContext, cancelRecovery := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelRecovery()
	recovery := exec.CommandContext(recoveryContext, binary, "--config", configPath)
	var recoveredStderr bytes.Buffer
	recovery.Stderr = &recoveredStderr
	stdin, err := recovery.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := recovery.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := recovery.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := stdin.Write(append(discover, '\n')); err != nil {
		t.Fatal(err)
	}
	lineResult := make(chan []byte, 1)
	lineError := make(chan error, 1)
	go func() {
		line, err := bufio.NewReader(stdout).ReadBytes('\n')
		if err != nil {
			lineError <- err
			return
		}
		lineResult <- line
	}()
	var recoveredStdout []byte
	select {
	case recoveredStdout = <-lineResult:
	case err := <-lineError:
		t.Fatal(err)
	case <-recoveryContext.Done():
		t.Fatal(recoveryContext.Err())
	}
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	wait := make(chan error, 1)
	go func() { wait <- recovery.Wait() }()
	select {
	case err := <-wait:
		if err != nil && !strings.Contains(recoveredStderr.String(), "server is closing: EOF") {
			t.Fatalf("fresh process error = %v stderr=%q", err, recoveredStderr.String())
		}
	case <-recoveryContext.Done():
		t.Fatal(recoveryContext.Err())
	}
	var recovered struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			SupportedVersions []string `json:"supportedVersions"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recoveredStdout, &recovered); err != nil {
		t.Fatalf("recovery response polluted stdout: %q: %v", recoveredStdout, err)
	}
	if recovered.JSONRPC != "2.0" || recovered.ID != 2 || len(recovered.Result.SupportedVersions) == 0 {
		t.Fatalf("recovery response = %s", recoveredStdout)
	}
	if !strings.Contains(recoveredStderr.String(), "serving MCP over stdio") {
		t.Fatalf("recovery stderr = %q", recoveredStderr.String())
	}
}

func TestRunStdioCleanEOFStopsWithoutStdout(t *testing.T) {
	installFixtureFFmpeg(t)
	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, []string{"--config", writeStdioFixtureConfig(t)}, strings.NewReader(""), &stdout, &stderr, os.LookupEnv)
	}()
	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		t.Fatal("stdio runtime did not stop after clean EOF")
	}
	if !benignStdioEOF(err) {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("clean EOF polluted stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "serving MCP over stdio") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func writeStdioFixtureConfig(t *testing.T) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("devices:\n  smoke:\n    url: http://jetkvm.example.invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return configPath
}

func benignStdioEOF(err error) bool {
	return err == nil || errors.Is(err, io.EOF) || err.Error() == "server is closing: EOF"
}

func TestRunStdioDiscoveryUsesStdoutAndLogsToStderr(t *testing.T) {
	installFixtureFFmpeg(t)
	configPath := writeStdioFixtureConfig(t)
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
