//go:build linux || darwin

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPrepareCommandKillsDescendantProcessGroupOnCancellation(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	command := exec.CommandContext(ctx, "sh", "-c", `sleep 30 & echo $! > "$PID_FILE"; wait`)
	command.Env = append(os.Environ(), "PID_FILE="+pidPath)
	prepareCommand(command)
	started := time.Now()
	if err := command.Run(); err == nil {
		t.Fatal("cancelled process group returned no error")
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("process-group cancellation took %s", elapsed)
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		err = syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("descendant process %d survived cancellation", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRunningServerStopKillsDescendantThatIgnoresInterrupt(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "child.pid")
	command := exec.CommandContext(context.Background(), "sh", "-c", `trap 'exit 0' INT; (trap '' INT; sleep 30) & echo $! > "$PID_FILE"; wait`)
	command.Env = append(os.Environ(), "PID_FILE="+pidPath)
	prepareCommand(command)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	server := &runningServer{command: command, done: make(chan error, 1)}
	go func() { server.done <- command.Wait() }()
	var pid int
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidPath)
		if err == nil {
			pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("descendant PID was not recorded")
	}
	if err := server.stop(); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("descendant process %d survived normal stop", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
